package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services/video_recording"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// 调度器常量
const (
	// TriggerTimeTolerance 触发时间容错窗口（任务在触发时间之前1分钟内仍可执行）
	TriggerTimeTolerance = 1 * time.Minute
)

// SchedulerInterface 调度器接口
type SchedulerInterface interface {
	Start() error
	Stop()
	AddTask(task *models.VideoRecordingTask) error
	RemoveTask(taskID uint) error
	SyncPendingTasks() error
	IsTaskScheduled(taskID uint) bool
	IsTaskExecuting(taskID uint) bool
	ExecuteTask(taskID uint) error
	CancelTaskExecution(taskID uint) error
}

// VideoSimpleScheduler 视频录制任务调度器
type VideoSimpleScheduler struct {
	cron          *cron.Cron
	taskService   TaskServiceInterface
	coordinator   RecorderCoordinatorInterface
	connector     *video_recording.HuaweiConferenceConnector
	taskEntries   map[uint]cron.EntryID
	entryTasks    map[cron.EntryID]uint
	executing     map[uint]bool
	cancelFuncs   map[uint]context.CancelFunc // 任务取消函数
	logger        *zap.Logger
	config        *config.Config
	mu            sync.RWMutex
	startTime     time.Time
}

// TaskServiceInterface 任务服务接口
type TaskServiceInterface interface {
	GetTask(id uint) (*models.VideoRecordingTask, error)
	GetPendingTasks() ([]*models.VideoRecordingTask, error)
	UpdateTaskStatus(id uint, status models.VideoRecordingTaskStatus, errorMsg string) error
	GetHuaweiConfig(id uint) (*models.HuaweiConfig, error)
}

// RecorderCoordinatorInterface 录制协调器接口
type RecorderCoordinatorInterface interface {
	StartRecording(task *models.VideoRecordingTask, huaweiConfig *models.HuaweiConfig) error
	StopRecording(taskID uint) error
	HealthCheck() error
}

// NewVideoSimpleScheduler 创建调度器
func NewVideoSimpleScheduler(
	taskService TaskServiceInterface,
	coordinator RecorderCoordinatorInterface,
	connector *video_recording.HuaweiConferenceConnector,
	logger *zap.Logger,
	cfg *config.Config,
) *VideoSimpleScheduler {
	// 创建Cron调度器（秒级精度）
	c := cron.New(cron.WithSeconds())

	scheduler := &VideoSimpleScheduler{
		cron:         c,
		taskService:  taskService,
		coordinator:   coordinator,
		connector:    connector,
		taskEntries:  make(map[uint]cron.EntryID),
		entryTasks:   make(map[cron.EntryID]uint),
		executing:    make(map[uint]bool),
		cancelFuncs:  make(map[uint]context.CancelFunc),
		logger:       logger,
		config:       cfg,
		startTime:    time.Now(),
	}

	return scheduler
}

// AddTask 添加任务到调度器
func (s *VideoSimpleScheduler) AddTask(task *models.VideoRecordingTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查任务是否已存在
	if _, exists := s.taskEntries[task.ID]; exists {
		return fmt.Errorf("任务已在调度器中: %d", task.ID)
	}

	// 计算触发时间
	triggerTime := s.calculateTriggerTime(task.StartTime, task.PreJoinMinutes)

	// 检查触发时间是否在未来
	if triggerTime.Before(time.Now().Add(-1 * TriggerTimeTolerance)) {
		return fmt.Errorf("触发时间已过期: %s", triggerTime.Format(time.RFC3339))
	}

	// 生成Cron表达式
	cronExpr := s.generateCronExpression(triggerTime)

	// 添加到Cron调度器
	taskID := task.ID
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeTask(taskID)
	})
	if err != nil {
		return fmt.Errorf("添加Cron任务失败: %w", err)
	}

	// 保存映射关系
	s.taskEntries[task.ID] = entryID
	s.entryTasks[entryID] = task.ID

	s.logger.Debug("任务已添加到调度器",
		zap.Uint("task_id", task.ID),
		zap.String("name", task.Name),
		zap.String("cron_expr", cronExpr),
		zap.Time("trigger_time", triggerTime),
		zap.Time("start_time", task.StartTime),
	)

	return nil
}

// RemoveTask 从调度器移除任务
func (s *VideoSimpleScheduler) RemoveTask(taskID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.taskEntries[taskID]
	if !exists {
		return fmt.Errorf("任务不在调度器中: %d", taskID)
	}

	// 从Cron调度器移除
	s.cron.Remove(entryID)

	// 删除映射关系
	delete(s.taskEntries, taskID)
	delete(s.entryTasks, entryID)

	s.logger.Debug("任务已从调度器移除",
		zap.Uint("task_id", taskID),
	)

	return nil
}

// executeTask 执行任务
func (s *VideoSimpleScheduler) executeTask(taskID uint) {
	s.logger.Info("开始执行任务",
		zap.Uint("task_id", taskID),
	)

	// 创建任务context用于取消
	ctx, cancel := context.WithCancel(context.Background())

	// 检查是否正在执行
	s.mu.Lock()
	if s.executing[taskID] {
		s.mu.Unlock()
		cancel()
		s.logger.Warn("任务正在执行中", zap.Uint("task_id", taskID))
		return
	}
	s.executing[taskID] = true
	s.cancelFuncs[taskID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.executing, taskID)
		delete(s.cancelFuncs, taskID)
		s.mu.Unlock()
	}()

	// 加载任务
	task, err := s.taskService.GetTask(taskID)
	if err != nil {
		s.logger.Error("加载任务失败",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		return
	}

	// 验证任务状态
	if task.Status != models.VideoStatusPending {
		s.logger.Warn("任务状态不正确",
			zap.Uint("task_id", taskID),
			zap.String("status", string(task.Status)),
		)
		return
	}

	// 检查任务是否过期
	if time.Now().After(task.EndTime) {
		s.logger.Warn("任务已过期",
			zap.Uint("task_id", taskID),
			zap.Time("end_time", task.EndTime),
		)
		s.updateTaskStatus(taskID, models.VideoStatusFailed, "任务已过期")
		return
	}

	// 更新状态为连接中
	if err := s.updateTaskStatus(taskID, models.VideoStatusConnecting, ""); err != nil {
		s.logger.Error("更新任务状态失败", zap.Error(err))
		return
	}

	// 连接华为会议
	if s.connector != nil {
		s.logger.Info("开始连接华为会议",
			zap.Uint("task_id", taskID),
			zap.String("conference_number", task.ConferenceNumber),
		)
		if err := s.connector.ConnectToConference(ctx, task); err != nil {
			s.logger.Error("连接华为会议失败",
				zap.Uint("task_id", taskID),
				zap.Error(err),
			)
			s.updateTaskStatus(taskID, models.VideoStatusFailed, err.Error())
			return
		}
		s.logger.Info("华为会议连接成功",
			zap.Uint("task_id", taskID),
		)
	}

	// 等待录制延迟
	if task.RecordDelayMinutes > 0 {
		s.logger.Info("等待录制延迟",
			zap.Uint("task_id", taskID),
			zap.Int("delay_minutes", task.RecordDelayMinutes),
		)
		time.Sleep(time.Duration(task.RecordDelayMinutes) * time.Minute)
	}

	// 加载华为配置
	huaweiConfig, err := s.taskService.GetHuaweiConfig(task.HuaweiConfigID)
	if err != nil {
		s.logger.Error("加载华为配置失败", zap.Error(err))
		s.updateTaskStatus(taskID, models.VideoStatusFailed, err.Error())
		return
	}

	// 更新状态为录制中
	if err := s.updateTaskStatus(taskID, models.VideoStatusRecording, ""); err != nil {
		s.logger.Error("更新任务状态失败", zap.Error(err))
		return
	}

	// 启动录制
	if err := s.coordinator.StartRecording(task, huaweiConfig); err != nil {
		s.logger.Error("启动录制失败",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		s.updateTaskStatus(taskID, models.VideoStatusFailed, err.Error())
		return
	}

	// 启动监控（传递context用于取消）
	go s.monitorTask(ctx, task)

	s.logger.Info("任务进入录制状态",
		zap.Uint("task_id", taskID),
		zap.Time("end_time", task.EndTime),
	)
}

// monitorTask 监控任务直到结束
func (s *VideoSimpleScheduler) monitorTask(ctx context.Context, task *models.VideoRecordingTask) {
	// 计算剩余时间
	remaining := time.Until(task.EndTime)
	if remaining < 0 {
		remaining = 0
	}

	s.logger.Info("开始监控任务",
		zap.Uint("task_id", task.ID),
		zap.Duration("remaining", remaining),
	)

	// 等待会议结束或取消
	timer := time.NewTimer(remaining)
	defer timer.Stop()

	select {
	case <-timer.C:
		// 正常结束
		s.completeTask(task.ID)
	case <-ctx.Done():
		// 任务被取消
		s.logger.Info("任务监控被取消",
			zap.Uint("task_id", task.ID),
		)
		return
	}
}

// completeTask 完成任务
func (s *VideoSimpleScheduler) completeTask(taskID uint) {
	s.logger.Info("开始完成任务",
		zap.Uint("task_id", taskID),
	)

	// 加载任务（用于断开华为会议连接）
	task, err := s.taskService.GetTask(taskID)
	if err != nil {
		s.logger.Error("加载任务失败", zap.Error(err))
	} else {
		// 断开华为会议连接
		if s.connector != nil {
			s.logger.Info("断开华为会议连接",
				zap.Uint("task_id", taskID),
				zap.String("conference_number", task.ConferenceNumber),
			)
			if err := s.connector.DisconnectFromConference(context.Background(), task); err != nil {
				s.logger.Error("断开华为会议连接失败", zap.Error(err))
				// 继续执行，不阻止任务完成
			} else {
				s.logger.Info("华为会议连接已断开",
					zap.Uint("task_id", taskID),
				)
			}
		}
	}

	// 停止录制
	if err := s.coordinator.StopRecording(taskID); err != nil {
		s.logger.Error("停止录制失败",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
	}

	// 更新任务状态
	if err := s.updateTaskStatus(taskID, models.VideoStatusCompleted, ""); err != nil {
		s.logger.Error("更新任务状态失败", zap.Error(err))
	}

	// 从调度器移除
	s.RemoveTask(taskID)

	s.logger.Info("任务已完成",
		zap.Uint("task_id", taskID),
	)
}

// calculateTriggerTime 计算触发时间
func (s *VideoSimpleScheduler) calculateTriggerTime(startTime time.Time, preJoinMinutes int) time.Time {
	return startTime.Add(-time.Duration(preJoinMinutes) * time.Minute)
}

// generateCronExpression 生成Cron表达式
func (s *VideoSimpleScheduler) generateCronExpression(triggerTime time.Time) string {
	// Cron 格式: 秒 分 时 日 月 周
	// 注意：任务执行后会通过RemoveTask移除，确保只执行一次
	return fmt.Sprintf("%d %d %d %d %d *",
		triggerTime.Second(),
		triggerTime.Minute(),
		triggerTime.Hour(),
		triggerTime.Day(),
		int(triggerTime.Month()),
	)
}

// updateTaskStatus 更新任务状态（带状态转换验证）
func (s *VideoSimpleScheduler) updateTaskStatus(taskID uint, status models.VideoRecordingTaskStatus, errorMsg string) error {
	// 重新加载任务以获取最新状态
	task, err := s.taskService.GetTask(taskID)
	if err != nil {
		return err
	}

	// 验证状态转换是否合法
	if !task.CanTransitionTo(status) {
		s.logger.Warn("非法的状态转换",
			zap.Uint("task_id", taskID),
			zap.String("from", string(task.Status)),
			zap.String("to", string(status)),
		)
		return fmt.Errorf("非法状态转换: %s -> %s", task.Status, status)
	}

	return s.taskService.UpdateTaskStatus(taskID, status, errorMsg)
}

// GetScheduledTasks 获取已调度的任务列表
func (s *VideoSimpleScheduler) GetScheduledTasks() []uint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]uint, 0, len(s.taskEntries))
	for taskID := range s.taskEntries {
		tasks = append(tasks, taskID)
	}
	return tasks
}

// GetExecutingTasks 获取正在执行的任务列表
func (s *VideoSimpleScheduler) GetExecutingTasks() []uint {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]uint, 0, len(s.executing))
	for taskID := range s.executing {
		tasks = append(tasks, taskID)
	}
	return tasks
}

// GetStats 获取调度器统计信息
func (s *VideoSimpleScheduler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"scheduled_tasks": len(s.taskEntries),
		"executing_tasks":  len(s.executing),
		"uptime":           time.Since(s.startTime).String(),
	}
}

// HealthCheck 健康检查
func (s *VideoSimpleScheduler) HealthCheck() error {
	if err := s.coordinator.HealthCheck(); err != nil {
		return fmt.Errorf("录制协调器异常: %w", err)
	}
	return nil
}

// Start 启动调度器
func (s *VideoSimpleScheduler) Start() error {
	s.logger.Info("启动调度器")

	// 启动Cron调度器
	s.cron.Start()

	// 从数据库加载待执行的任务
	tasks, err := s.taskService.GetPendingTasks()
	if err != nil {
		return fmt.Errorf("加载待执行任务失败: %w", err)
	}

	// 添加到调度器
	addedCount := 0
	for _, task := range tasks {
		triggerTime := s.calculateTriggerTime(task.StartTime, task.PreJoinMinutes)
		if triggerTime.After(time.Now().Add(-1 * TriggerTimeTolerance)) {
			if err := s.AddTask(task); err != nil {
				s.logger.Error("添加任务失败",
					zap.Uint("task_id", task.ID),
					zap.Error(err),
				)
			} else {
				addedCount++
			}
		} else {
			// 触发时间已过，标记为失败
			s.updateTaskStatus(task.ID, models.VideoStatusFailed, "触发时间已过期")
		}
	}

	s.logger.Info("调度器启动完成",
		zap.Int("total_tasks", len(tasks)),
		zap.Int("scheduled_tasks", addedCount),
	)

	return nil
}

// Stop 停止调度器
func (s *VideoSimpleScheduler) Stop() {
	s.logger.Info("停止调度器")

	// 停止Cron调度器
	s.cron.Stop()

	s.logger.Info("调度器已停止")
}

// IsTaskScheduled 检查任务是否已调度
func (s *VideoSimpleScheduler) IsTaskScheduled(taskID uint) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.taskEntries[taskID]
	return exists
}

// IsTaskExecuting 检查任务是否正在执行
func (s *VideoSimpleScheduler) IsTaskExecuting(taskID uint) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executing[taskID]
}

// SyncPendingTasks 同步待执行任务到调度器
// 从数据库重新加载所有待执行任务，并更新调度器中的任务
func (s *VideoSimpleScheduler) SyncPendingTasks() error {
	s.logger.Info("同步待执行任务")

	// 从数据库加载待执行的任务
	tasks, err := s.taskService.GetPendingTasks()
	if err != nil {
		return fmt.Errorf("加载待执行任务失败: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 收集数据库中的待执行任务ID
	dbTaskIDs := make(map[uint]bool)
	for _, task := range tasks {
		dbTaskIDs[task.ID] = true
	}

	// 移除不再待执行的任务
	for taskID := range s.taskEntries {
		if !dbTaskIDs[taskID] {
			entryID := s.taskEntries[taskID]
			s.cron.Remove(entryID)
			delete(s.taskEntries, taskID)
			delete(s.entryTasks, entryID)
			s.logger.Debug("移除已取消/完成的任务",
				zap.Uint("task_id", taskID),
			)
		}
	}

	// 添加新的待执行任务
	addedCount := 0
	for _, task := range tasks {
		// 检查是否已在调度器中
		if _, exists := s.taskEntries[task.ID]; exists {
			continue
		}

		// 计算触发时间
		triggerTime := s.calculateTriggerTime(task.StartTime, task.PreJoinMinutes)

		// 检查触发时间是否在未来（或在容错窗口内）
		if triggerTime.Before(time.Now().Add(-1 * TriggerTimeTolerance)) {
			s.logger.Warn("任务触发时间已过期，跳过",
				zap.Uint("task_id", task.ID),
				zap.Time("trigger_time", triggerTime),
			)
			// 标记为失败
			go s.updateTaskStatus(task.ID, models.VideoStatusFailed, "触发时间已过期")
			continue
		}

		// 生成Cron表达式并添加
		cronExpr := s.generateCronExpression(triggerTime)
		taskID := task.ID
		entryID, err := s.cron.AddFunc(cronExpr, func() {
			s.executeTask(taskID)
		})
		if err != nil {
			s.logger.Error("添加Cron任务失败",
				zap.Uint("task_id", task.ID),
				zap.Error(err),
			)
			continue
		}

		s.taskEntries[task.ID] = entryID
		s.entryTasks[entryID] = task.ID
		addedCount++

		s.logger.Info("新任务已添加到调度器",
			zap.Uint("task_id", task.ID),
			zap.String("name", task.Name),
			zap.Time("trigger_time", triggerTime),
		)
	}

	s.logger.Info("同步待执行任务完成",
		zap.Int("total_pending", len(tasks)),
		zap.Int("newly_added", addedCount),
		zap.Int("scheduled_total", len(s.taskEntries)),
	)

	return nil
}

// ExecuteTask 手动执行任务（供外部调用）
func (s *VideoSimpleScheduler) ExecuteTask(taskID uint) error {
	s.logger.Info("手动执行任务",
		zap.Uint("task_id", taskID),
	)

	// 检查任务是否正在执行
	s.mu.RLock()
	if s.executing[taskID] {
		s.mu.RUnlock()
		return fmt.Errorf("任务正在执行中: %d", taskID)
	}
	s.mu.RUnlock()

	// 加载任务
	task, err := s.taskService.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("加载任务失败: %w", err)
	}

	// 验证任务状态
	if task.Status != models.VideoStatusPending {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}

	// 检查任务是否过期
	if time.Now().After(task.EndTime) {
		return fmt.Errorf("任务已过期: %s", task.EndTime.Format(time.RFC3339))
	}

	// 直接执行任务
	go s.executeTask(taskID)

	return nil
}

// CancelTaskExecution 取消正在执行的任务
func (s *VideoSimpleScheduler) CancelTaskExecution(taskID uint) error {
	s.logger.Info("取消任务执行",
		zap.Uint("task_id", taskID),
	)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否有取消函数
	cancel, hasCancel := s.cancelFuncs[taskID]
	if hasCancel {
		cancel()
		s.logger.Info("任务执行已取消",
			zap.Uint("task_id", taskID),
		)
		return nil
	}

	// 如果没有取消函数，检查是否在执行中
	if s.executing[taskID] {
		return fmt.Errorf("任务正在执行中，但无法取消")
	}

	return fmt.Errorf("任务未在执行中")
}
