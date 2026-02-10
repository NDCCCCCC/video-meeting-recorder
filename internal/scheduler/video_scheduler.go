package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// VideoSimpleScheduler 视频录制任务调度器
type VideoSimpleScheduler struct {
	cron         *cron.Cron
	taskService  TaskServiceInterface
	coordinator  RecorderCoordinatorInterface
	taskEntries  map[uint]cron.EntryID
	entryTasks   map[cron.EntryID]uint
	executing    map[uint]bool
	logger       *zap.Logger
	config       *config.Config
	mu           sync.RWMutex
	startTime    time.Time
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
	logger *zap.Logger,
	cfg *config.Config,
) *VideoSimpleScheduler {
	// 创建Cron调度器（秒级精度）
	c := cron.New(cron.WithSeconds())

	scheduler := &VideoSimpleScheduler{
		cron:         c,
		taskService:  taskService,
		coordinator:   coordinator,
		taskEntries:  make(map[uint]cron.EntryID),
		entryTasks:   make(map[cron.EntryID]uint),
		executing:    make(map[uint]bool),
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
	if triggerTime.Before(time.Now().Add(-1 * time.Minute)) {
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

	s.logger.Info("任务已添加到调度器",
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

	s.logger.Info("任务已从调度器移除",
		zap.Uint("task_id", taskID),
	)

	return nil
}

// executeTask 执行任务
func (s *VideoSimpleScheduler) executeTask(taskID uint) {
	s.logger.Info("开始执行任务",
		zap.Uint("task_id", taskID),
	)

	// 检查是否正在执行
	s.mu.Lock()
	if s.executing[taskID] {
		s.mu.Unlock()
		s.logger.Warn("任务正在执行中", zap.Uint("task_id", taskID))
		return
	}
	s.executing[taskID] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.executing, taskID)
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

	// 启动监控
	go s.monitorTask(task)

	s.logger.Info("任务进入录制状态",
		zap.Uint("task_id", taskID),
		zap.Time("end_time", task.EndTime),
	)
}

// monitorTask 监控任务直到结束
func (s *VideoSimpleScheduler) monitorTask(task *models.VideoRecordingTask) {
	// 计算剩余时间
	remaining := time.Until(task.EndTime)
	if remaining < 0 {
		remaining = 0
	}

	s.logger.Info("开始监控任务",
		zap.Uint("task_id", task.ID),
		zap.Duration("remaining", remaining),
	)

	// 等待会议结束
	timer := time.NewTimer(remaining)
	<-timer.C

	// 完成任务
	s.completeTask(task.ID)
}

// completeTask 完成任务
func (s *VideoSimpleScheduler) completeTask(taskID uint) {
	s.logger.Info("开始完成任务",
		zap.Uint("task_id", taskID),
	)

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
	return fmt.Sprintf("%d %d %d %d %d *",
		triggerTime.Second(),
		triggerTime.Minute(),
		triggerTime.Hour(),
		triggerTime.Day(),
		int(triggerTime.Month()),
	)
}

// updateTaskStatus 更新任务状态
func (s *VideoSimpleScheduler) updateTaskStatus(taskID uint, status models.VideoRecordingTaskStatus, errorMsg string) error {
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

	// 从数据库加载待执行的任务
	tasks, err := s.taskService.GetPendingTasks()
	if err != nil {
		return fmt.Errorf("加载待执行任务失败: %w", err)
	}

	// 添加到调度器
	addedCount := 0
	for _, task := range tasks {
		triggerTime := s.calculateTriggerTime(task.StartTime, task.PreJoinMinutes)
		if triggerTime.After(time.Now().Add(-1 * time.Minute)) {
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
