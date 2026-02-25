package scheduler

import (
	"context"
	"fmt"
	"strings"
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
	UpdateTaskEndTime(taskID uint, newEndTime time.Time) error
}

// VideoSimpleScheduler 视频录制任务调度器
type VideoSimpleScheduler struct {
	cron              *cron.Cron
	taskService       TaskServiceInterface
	coordinator       RecorderCoordinatorInterface
	connector         *video_recording.HuaweiConferenceConnector
	conversionService ConversionServiceInterface // 转换服务
	videoFileService  VideoFileServiceInterface  // 视频文件服务
	taskEntries       map[uint]cron.EntryID
	entryTasks        map[cron.EntryID]uint
	executing         map[uint]bool
	cancelFuncs       map[uint]context.CancelFunc // 任务取消函数
	taskEndTimes      map[uint]time.Time          // 任务结束时间（用于动态更新）
	taskUpdateChans   map[uint]chan time.Time     // 任务结束时间更新通道
	logger            *zap.Logger
	config            *config.Config
	mu                sync.RWMutex
	startTime         time.Time
}

// ConversionServiceInterface 转换服务接口
type ConversionServiceInterface interface {
	SubmitConversion(taskID uint) error
}

// VideoFileServiceInterface 视频文件服务接口
type VideoFileServiceInterface interface {
	CreateFileFromTask(task *models.VideoRecordingTask, format *string) (*models.VideoFile, error)
}

// TaskServiceInterface 任务服务接口
type TaskServiceInterface interface {
	GetTask(id uint) (*models.VideoRecordingTask, error)
	GetPendingTasks() ([]*models.VideoRecordingTask, error)
	UpdateTaskStatus(id uint, status models.VideoRecordingTaskStatus, errorMsg string) error
	UpdateRecordingPaths(id uint, mkvPath, hlsPath string) error
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
	conversionService ConversionServiceInterface,
	videoFileService VideoFileServiceInterface,
	logger *zap.Logger,
	cfg *config.Config,
) *VideoSimpleScheduler {
	// 创建Cron调度器（秒级精度，使用UTC时区）
	// 重要：所有任务时间都是UTC，cron也必须使用UTC时区，否则触发时间会错误
	c := cron.New(cron.WithSeconds(), cron.WithLocation(time.UTC))

	scheduler := &VideoSimpleScheduler{
		cron:              c,
		taskService:       taskService,
		coordinator:       coordinator,
		connector:         connector,
		conversionService: conversionService,
		videoFileService:  videoFileService,
		taskEntries:       make(map[uint]cron.EntryID),
		entryTasks:        make(map[cron.EntryID]uint),
		executing:         make(map[uint]bool),
		cancelFuncs:       make(map[uint]context.CancelFunc),
		taskEndTimes:      make(map[uint]time.Time),
		taskUpdateChans:   make(map[uint]chan time.Time),
		logger:            logger,
		config:            cfg,
		startTime:         time.Now(),
	}

	return scheduler
}

// AddTask 添加任务到调度器
func (s *VideoSimpleScheduler) AddTask(task *models.VideoRecordingTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 记录详细的时间信息用于调试
	s.logTaskTimeInfo(task)

	// 检查任务是否已存在
	if _, exists := s.taskEntries[task.ID]; exists {
		return fmt.Errorf("任务已在调度器中: %d", task.ID)
	}

	// 确保任务时间在 UTC 时区
	startTimeUTC := task.StartTime.UTC()
	endTimeUTC := task.EndTime.UTC()

	// 计算触发时间
	triggerTime := s.calculateTriggerTime(startTimeUTC, task.PreJoinMinutes)
	now := time.Now().UTC()

	s.logger.Info("任务触发时间计算",
		zap.Uint("task_id", task.ID),
		zap.String("start_time", startTimeUTC.Format(time.RFC3339)),
		zap.Int("pre_join_minutes", task.PreJoinMinutes),
		zap.String("trigger_time", triggerTime.Format(time.RFC3339)),
		zap.String("current_time", now.Format(time.RFC3339)),
		zap.Int64("seconds_until_trigger", int64(triggerTime.Sub(now).Seconds())),
		zap.String("current_time_zone", now.Location().String()),
		zap.String("trigger_time_zone", triggerTime.Location().String()),
	)

	// 检查任务是否已过期（超过结束时间）
	if now.After(endTimeUTC) {
		s.logger.Warn("任务已过期，拒绝添加",
			zap.Uint("task_id", task.ID),
			zap.String("name", task.Name),
			zap.Time("end_time", endTimeUTC),
			zap.Time("current_time", now),
		)
		return fmt.Errorf("任务已过期: 结束时间 %s", endTimeUTC.Format(time.RFC3339))
	}

	// 如果当前时间已超过触发时间，立即执行任务
	if now.After(triggerTime) {
		s.logger.Info("任务触发时间已过，立即执行",
			zap.Uint("task_id", task.ID),
			zap.String("name", task.Name),
			zap.Time("trigger_time", triggerTime),
			zap.Time("end_time", endTimeUTC),
			zap.Duration("overdue_by", now.Sub(triggerTime)),
		)
		// 在 goroutine 中执行，避免阻塞
		go s.executeTask(task.ID)
		// 不添加到 taskEntries，因为不是通过 cron 调度的
		return nil
	}

	// 生成Cron表达式
	cronExpr := s.generateCronExpression(triggerTime)
	s.logger.Info("生成Cron表达式",
		zap.Uint("task_id", task.ID),
		zap.String("cron_expr", cronExpr),
		zap.Time("trigger_time", triggerTime),
	)

	// 添加到Cron调度器
	taskID := task.ID
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.logger.Info("Cron触发执行任务",
			zap.Uint("task_id", taskID),
			zap.String("cron_expr", cronExpr),
		)
		s.executeTask(taskID)
	})
	if err != nil {
		s.logger.Error("添加Cron任务失败",
			zap.Uint("task_id", task.ID),
			zap.String("cron_expr", cronExpr),
			zap.Error(err),
		)
		return fmt.Errorf("添加Cron任务失败: %w", err)
	}

	// 保存映射关系
	s.taskEntries[task.ID] = entryID
	s.entryTasks[entryID] = task.ID

	s.logger.Info("任务已成功添加到调度器",
		zap.Uint("task_id", task.ID),
		zap.String("name", task.Name),
		zap.Int("entry_id", int(entryID)),
		zap.String("cron_expr", cronExpr),
		zap.Time("trigger_time", triggerTime),
		zap.Time("start_time", startTimeUTC),
		zap.Int("seconds_until_trigger", int(triggerTime.Sub(now))),
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
	if time.Now().UTC().After(task.EndTime) {
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

		// 清理：断开华为会议连接，解锁终端
		if s.connector != nil {
			s.logger.Info("录制启动失败，正在清理华为会议连接",
				zap.Uint("task_id", taskID),
			)
			// 使用带超时的 context，避免清理操作无限等待
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if cleanupErr := s.connector.DisconnectFromConference(cleanupCtx, task); cleanupErr != nil {
				s.logger.Error("清理华为会议连接失败",
					zap.Uint("task_id", taskID),
					zap.Error(cleanupErr),
				)
			}
		}
		return
	}

	// 更新数据库中的文件路径信息
	// coordinator.StartRecording 会修改 task 对象，设置 MKVFilePath 和 HLSPreviewPath
	s.logger.Info("准备更新录制文件路径",
		zap.Uint("task_id", taskID),
		zap.String("mkv_path", task.MKVFilePath),
		zap.String("hls_path", task.HLSPreviewPath),
	)
	if err := s.taskService.UpdateRecordingPaths(taskID, task.MKVFilePath, task.HLSPreviewPath); err != nil {
		s.logger.Warn("更新录制文件路径失败",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		// 不中断录制，只记录警告
	} else {
		s.logger.Info("录制文件路径已更新",
			zap.Uint("task_id", taskID),
		)
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
	s.mu.Lock()
	// 创建更新通道
	updateChan := make(chan time.Time, 1)
	s.taskUpdateChans[task.ID] = updateChan
	s.taskEndTimes[task.ID] = task.EndTime
	s.mu.Unlock()

	// 清理函数
	defer func() {
		s.mu.Lock()
		delete(s.taskUpdateChans, task.ID)
		delete(s.taskEndTimes, task.ID)
		s.mu.Unlock()
	}()

	endTime := task.EndTime

	for {
		// 计算剩余时间
		remaining := time.Until(endTime)
		if remaining < 0 {
			remaining = 0
		}

		s.logger.Info("监控任务中",
			zap.Uint("task_id", task.ID),
			zap.Duration("remaining", remaining),
			zap.Time("end_time", endTime),
		)

		// 创建定时器
		timer := time.NewTimer(remaining)

		select {
		case <-timer.C:
			timer.Stop()
			// 正常结束
			s.completeTask(task.ID)
			return

		case <-ctx.Done():
			timer.Stop()
			// 任务被取消
			s.logger.Info("任务监控被取消",
				zap.Uint("task_id", task.ID),
			)
			return

		case newEndTime, ok := <-updateChan:
			timer.Stop()
			if !ok {
				// 通道关闭，退出
				return
			}
			// 更新结束时间，继续监控
			endTime = newEndTime
			s.logger.Info("任务结束时间已更新",
				zap.Uint("task_id", task.ID),
				zap.Time("old_end_time", task.EndTime),
				zap.Time("new_end_time", newEndTime),
			)
		}
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

	// 更新任务状态为转换中（而不是直接完成）
	if err := s.updateTaskStatus(taskID, models.VideoStatusConverting, ""); err != nil {
		s.logger.Error("更新任务状态失败", zap.Error(err))
	}

	// 从调度器移除
	s.RemoveTask(taskID)

	// 创建视频文件记录（如果 videoFileService 可用）
	if s.videoFileService != nil && task != nil {
		// 为 MKV 文件创建记录
		if task.MKVFilePath != "" {
			mkv := "mkv"
			if _, err := s.videoFileService.CreateFileFromTask(task, &mkv); err != nil {
				s.logger.Error("创建MKV文件记录失败",
					zap.Uint("task_id", taskID),
					zap.Error(err),
				)
			}
		}
		// 如果 MP4 已存在，也为它创建记录
		if task.MP4FilePath != "" {
			mp4 := "mp4"
			if _, err := s.videoFileService.CreateFileFromTask(task, &mp4); err != nil {
				s.logger.Error("创建MP4文件记录失败",
					zap.Uint("task_id", taskID),
					zap.Error(err),
				)
			}
		}
	}

	// 提交转换任务（如果有转换服务）
	if s.conversionService != nil && task != nil && task.MKVFilePath != "" {
		s.logger.Info("提交MKV到MP4转换任务",
			zap.Uint("task_id", taskID),
			zap.String("mkv_file", task.MKVFilePath),
		)
		if err := s.conversionService.SubmitConversion(taskID); err != nil {
			s.logger.Error("提交转换任务失败",
				zap.Uint("task_id", taskID),
				zap.Error(err),
			)
			// 转换提交失败，将任务状态改为失败
			s.updateTaskStatus(taskID, models.VideoStatusFailed, "提交转换任务失败")
		}
	}

	s.logger.Info("录制完成，进入转换状态",
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
		"executing_tasks": len(s.executing),
		"uptime":          time.Since(s.startTime).String(),
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

	// 首先清理可能遗留的终端锁（服务异常退出导致）
	s.cleanupStaleTerminalLocks()

	// 启动Cron调度器
	s.cron.Start()

	// 从数据库加载待执行的任务
	tasks, err := s.taskService.GetPendingTasks()
	if err != nil {
		return fmt.Errorf("加载待执行任务失败: %w", err)
	}

	// 添加到调度器
	addedCount := 0
	immediateCount := 0
	expiredCount := 0
	for _, task := range tasks {
		if err := s.AddTask(task); err != nil {
			// 检查是否是过期错误
			if strings.Contains(err.Error(), "任务已过期") {
				expiredCount++
				s.logger.Info("跳过已过期任务",
					zap.Uint("task_id", task.ID),
					zap.String("name", task.Name),
					zap.Error(err),
				)
			} else {
				s.logger.Error("添加任务失败",
					zap.Uint("task_id", task.ID),
					zap.Error(err),
				)
			}
		} else {
			// 检查任务是否被添加到调度器（立即执行的任务不在 taskEntries 中）
			s.mu.Lock()
			_, isScheduled := s.taskEntries[task.ID]
			s.mu.Unlock()

			if isScheduled {
				addedCount++
			} else {
				immediateCount++
			}
		}
	}

	s.logger.Info("调度器启动完成",
		zap.Int("total_tasks", len(tasks)),
		zap.Int("scheduled_tasks", addedCount),
		zap.Int("immediate_executed", immediateCount),
		zap.Int("expired_skipped", expiredCount),
	)

	// 启动后立即同步一次，确保所有 pending 任务被正确处理
	s.logger.Info("启动后同步待执行任务")
	if syncErr := s.SyncPendingTasks(); syncErr != nil {
		s.logger.Error("启动后同步任务失败", zap.Error(syncErr))
	}

	return nil
}

// Stop 停止调度器
func (s *VideoSimpleScheduler) Stop() {
	s.logger.Info("停止调度器")

	// 停止Cron调度器
	s.cron.Stop()

	s.logger.Info("调度器已停止")
}

// SetConversionService 设置转换服务
func (s *VideoSimpleScheduler) SetConversionService(conversionService ConversionServiceInterface) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversionService = conversionService
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
	s.logger.Info("开始同步待执行任务")

	// 从数据库加载待执行的任务
	tasks, err := s.taskService.GetPendingTasks()
	if err != nil {
		s.logger.Error("加载待执行任务失败", zap.Error(err))
		return fmt.Errorf("加载待执行任务失败: %w", err)
	}

	s.logger.Info("从数据库加载待执行任务",
		zap.Int("total_pending", len(tasks)),
	)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 收集数据库中的待执行任务ID
	dbTaskIDs := make(map[uint]bool)
	for _, task := range tasks {
		dbTaskIDs[task.ID] = true
	}

	// 移除不再待执行的任务，并检查已调度任务是否需要处理
	removedCount := 0
	recheckCount := 0
	for taskID := range s.taskEntries {
		entryID := s.taskEntries[taskID]
		if !dbTaskIDs[taskID] {
			s.cron.Remove(entryID)
			delete(s.taskEntries, taskID)
			delete(s.entryTasks, entryID)
			removedCount++
			s.logger.Info("移除已取消/完成的任务",
				zap.Uint("task_id", taskID),
			)
		} else {
			// 任务仍在待执行列表中，需要重新处理（因为Cron调度器在重启后会丢失任务）
			// 加载任务详情
			task, err := s.taskService.GetTask(taskID)
			if err == nil {
				// 重新计算触发时间
				triggerTime := s.calculateTriggerTime(task.StartTime.UTC(), task.PreJoinMinutes)
				now := time.Now().UTC()

				s.logger.Info("重新检查已调度任务",
					zap.Uint("task_id", taskID),
					zap.String("name", task.Name),
					zap.String("status", string(task.Status)),
					zap.Time("now", now),
					zap.Time("trigger_time", triggerTime),
					zap.Time("end_time", task.EndTime.UTC()),
					zap.Bool("past_trigger", now.After(triggerTime)),
					zap.Bool("past_end", now.After(task.EndTime.UTC())),
				)

				// 先从Cron移除旧的任务（重启后Cron内部状态已清空）
				s.cron.Remove(entryID)
				delete(s.taskEntries, taskID)
				delete(s.entryTasks, entryID)

				// 如果任务已过期（超过结束时间），标记失败
				if now.After(task.EndTime.UTC()) {
					recheckCount++
					s.logger.Info("移除已过期任务",
						zap.Uint("task_id", taskID),
						zap.Time("end_time", task.EndTime.UTC()),
					)
					// 异步更新任务状态
					go s.updateTaskStatus(taskID, models.VideoStatusFailed, "任务已过期")
				} else if now.After(triggerTime) {
					// 触发时间已过，立即执行
					recheckCount++
					s.logger.Info("立即执行已过期任务",
						zap.Uint("task_id", taskID),
						zap.Time("trigger_time", triggerTime),
					)
					go s.executeTask(taskID)
				} else {
					// 重新添加到Cron调度器
					recheckCount++
					cronExpr := s.generateCronExpression(triggerTime)
					newEntryID, err := s.cron.AddFunc(cronExpr, func() {
						s.logger.Info("Cron触发执行任务",
							zap.Uint("task_id", taskID),
							zap.String("cron_expr", cronExpr),
						)
						s.executeTask(taskID)
					})
					if err != nil {
						s.logger.Error("重新添加Cron任务失败",
							zap.Uint("task_id", taskID),
							zap.Error(err),
						)
					} else {
						s.taskEntries[task.ID] = newEntryID
						s.entryTasks[newEntryID] = task.ID
						s.logger.Info("已重新添加到调度器",
							zap.Uint("task_id", taskID),
							zap.String("name", task.Name),
							zap.Int("new_entry_id", int(newEntryID)),
							zap.Time("trigger_time", triggerTime),
							zap.Int("seconds_until_trigger", int(triggerTime.Sub(now).Seconds())),
						)
					}
				}
			} else {
				s.logger.Error("加载已调度任务详情失败",
					zap.Uint("task_id", taskID),
					zap.Error(err),
				)
				// 出错时也要移除，避免僵尸条目
				s.cron.Remove(entryID)
				delete(s.taskEntries, taskID)
				delete(s.entryTasks, entryID)
			}
		}
	}

	now := time.Now().UTC()

	// 添加新的待执行任务
	addedCount := 0
	immediateCount := 0
	expiredCount := 0
	skippedCount := 0

	for _, task := range tasks {
		// 检查是否已在调度器中
		if _, exists := s.taskEntries[task.ID]; exists {
			skippedCount++
			continue
		}

		// 确保任务时间在 UTC 时区
		startTimeUTC := task.StartTime.UTC()
		endTimeUTC := task.EndTime.UTC()
		triggerTime := s.calculateTriggerTime(startTimeUTC, task.PreJoinMinutes)

		s.logger.Info("处理待执行任务",
			zap.Uint("task_id", task.ID),
			zap.String("name", task.Name),
			zap.Time("start_time", startTimeUTC),
			zap.Time("end_time", endTimeUTC),
			zap.Time("trigger_time", triggerTime),
			zap.Int("pre_join_minutes", task.PreJoinMinutes),
		)

		// 检查任务是否已过期（超过结束时间）
		if now.After(endTimeUTC) {
			expiredCount++
			s.logger.Warn("跳过已过期任务",
				zap.Uint("task_id", task.ID),
				zap.String("name", task.Name),
				zap.Time("end_time", endTimeUTC),
				zap.Time("current_time", now),
			)
			// 异步更新任务状态
			go s.updateTaskStatus(task.ID, models.VideoStatusFailed, "任务已过期")
			continue
		}

		// 如果当前时间已超过触发时间，立即执行任务
		if now.After(triggerTime) {
			immediateCount++
			s.logger.Info("任务触发时间已过，立即执行",
				zap.Uint("task_id", task.ID),
				zap.String("name", task.Name),
				zap.Time("trigger_time", triggerTime),
				zap.Time("current_time", now),
				zap.Duration("overdue_by", now.Sub(triggerTime)),
			)
			// 在 goroutine 中执行，避免阻塞
			go s.executeTask(task.ID)
			// 不添加到 taskEntries，因为不是通过 cron 调度的
			continue
		}

		// 生成Cron表达式并添加
		cronExpr := s.generateCronExpression(triggerTime)
		taskID := task.ID

		s.logger.Info("准备添加Cron任务",
			zap.Uint("task_id", taskID),
			zap.String("name", task.Name),
			zap.String("cron_expr", cronExpr),
			zap.Int("seconds_until_trigger", int(triggerTime.Sub(now)/time.Second)),
		)

		entryID, err := s.cron.AddFunc(cronExpr, func() {
			s.logger.Info("Cron触发执行任务",
				zap.Uint("task_id", taskID),
				zap.String("cron_expr", cronExpr),
			)
			s.executeTask(taskID)
		})
		if err != nil {
			s.logger.Error("添加Cron任务失败",
				zap.Uint("task_id", task.ID),
				zap.String("cron_expr", cronExpr),
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
			zap.Int("entry_id", int(entryID)),
			zap.Time("trigger_time", triggerTime),
			zap.Int("seconds_until_trigger", int(triggerTime.Sub(now)/time.Second)),
		)
	}

	s.logger.Info("同步待执行任务完成",
		zap.Int("total_pending", len(tasks)),
		zap.Int("newly_added", addedCount),
		zap.Int("immediate_executed", immediateCount),
		zap.Int("expired_skipped", expiredCount),
		zap.Int("already_scheduled", skippedCount),
		zap.Int("removed", removedCount),
		zap.Int("rechecked_scheduled", recheckCount),
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
	if time.Now().UTC().After(task.EndTime) {
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

	// 检查是否有取消函数
	cancel, hasCancel := s.cancelFuncs[taskID]
	if hasCancel {
		// 先停止录制，确保 ffmpeg 正常退出并完成 MKV 文件写入
		if stopErr := s.coordinator.StopRecording(taskID); stopErr != nil {
			s.logger.Warn("停止录制失败",
				zap.Uint("task_id", taskID),
				zap.Error(stopErr),
			)
		}

		// 取消监控任务
		delete(s.cancelFuncs, taskID)
		delete(s.executing, taskID)
		cancel()
		s.logger.Info("任务执行已取消",
			zap.Uint("task_id", taskID),
		)

		// 从调度器移除任务
		s.RemoveTask(taskID)
	}

	// 无论是否有 cancelFunc，都需要尝试清理资源
	// 断开华为会议连接，解锁终端，创建文件记录，提交转换任务
	// 注意：这里在锁外获取任务信息，避免在持有锁时进行数据库查询
	if s.connector != nil {
		s.mu.Unlock()
		s.releaseHuaweiDevice(taskID)
		return nil
	}

	s.mu.Unlock()

	// 如果没有 connector，至少尝试停止录制
	if !hasCancel {
		if stopErr := s.coordinator.StopRecording(taskID); stopErr != nil {
			s.logger.Warn("停止录制失败",
				zap.Uint("task_id", taskID),
				zap.Error(stopErr),
			)
		}
		s.RemoveTask(taskID)
	}

	return nil
}

// UpdateTaskEndTime 更新录制任务的结束时间
func (s *VideoSimpleScheduler) UpdateTaskEndTime(taskID uint, newEndTime time.Time) error {
	s.logger.Info("更新任务结束时间",
		zap.Uint("task_id", taskID),
		zap.Time("new_end_time", newEndTime),
	)

	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查任务是否正在执行
	if !s.executing[taskID] {
		return fmt.Errorf("任务未在执行中")
	}

	// 检查是否有更新通道
	updateChan, hasChan := s.taskUpdateChans[taskID]
	if !hasChan {
		return fmt.Errorf("任务监控通道不存在")
	}

	// 更新存储的结束时间
	s.taskEndTimes[taskID] = newEndTime

	// 通过通道发送新的结束时间（非阻塞）
	select {
	case updateChan <- newEndTime:
		s.logger.Info("任务结束时间更新已发送",
			zap.Uint("task_id", taskID),
			zap.Time("new_end_time", newEndTime),
		)
	default:
		return fmt.Errorf("更新通道繁忙，请稍后重试")
	}

	return nil
}

// releaseHuaweiDevice 释放华为设备（在取消任务时调用）
// 此函数会尝试完整地断开会议连接并解锁终端，即使获取任务信息失败也会尝试解锁
func (s *VideoSimpleScheduler) releaseHuaweiDevice(taskID uint) {
	// 加载任务信息用于断开连接和提交转换
	task, err := s.taskService.GetTask(taskID)

	if err == nil && task != nil {
		// 正常情况：成功获取任务信息，完整地断开连接
		s.logger.Info("取消任务时断开华为会议连接",
			zap.Uint("task_id", taskID),
		)
		ctx, cancelCtx := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelCtx()
		if disconnectErr := s.connector.DisconnectFromConference(ctx, task); disconnectErr != nil {
			s.logger.Warn("断开华为会议连接失败",
				zap.Uint("task_id", taskID),
				zap.Error(disconnectErr),
			)
		} else {
			s.logger.Info("华为会议连接已断开",
				zap.Uint("task_id", taskID),
			)
		}

		// 创建视频文件记录（如果 videoFileService 可用）
		// 这样手动停止的任务生成的文件也会在文件管理页面列出
		if s.videoFileService != nil {
			// 为 MKV 文件创建记录
			if task.MKVFilePath != "" {
				mkv := "mkv"
				if _, err := s.videoFileService.CreateFileFromTask(task, &mkv); err != nil {
					s.logger.Warn("创建MKV文件记录失败",
						zap.Uint("task_id", taskID),
						zap.Error(err),
					)
				} else {
					s.logger.Info("已取消任务的MKV文件记录已创建",
						zap.Uint("task_id", taskID),
					)
				}
			}
			// 如果 MP4 已存在，也为它创建记录
			if task.MP4FilePath != "" {
				mp4 := "mp4"
				if _, err := s.videoFileService.CreateFileFromTask(task, &mp4); err != nil {
					s.logger.Warn("创建MP4文件记录失败",
						zap.Uint("task_id", taskID),
						zap.Error(err),
					)
				} else {
					s.logger.Info("已取消任务的MP4文件记录已创建",
						zap.Uint("task_id", taskID),
					)
				}
			}
		}

		// 如果有 MKV 文件，提交转换任务
		if s.conversionService != nil && task.MKVFilePath != "" {
			s.logger.Info("提交已取消任务的转换任务",
				zap.Uint("task_id", taskID),
				zap.String("mkv_file", task.MKVFilePath),
			)
			if convertErr := s.conversionService.SubmitConversion(taskID); convertErr != nil {
				s.logger.Warn("提交转换任务失败",
					zap.Uint("task_id", taskID),
					zap.Error(convertErr),
				)
			}
		}
	} else {
		// 降级处理：获取任务信息失败，至少尝试解锁终端
		s.logger.Warn("获取任务信息失败，尝试强制解锁终端",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
		if unlockErr := s.connector.UnlockTerminalByTaskID(taskID); unlockErr != nil {
			s.logger.Error("强制解锁终端失败",
				zap.Uint("task_id", taskID),
				zap.Error(unlockErr),
			)
		} else {
			s.logger.Info("强制解锁终端成功",
				zap.Uint("task_id", taskID),
			)
		}
	}
}

// cleanupStaleTerminalLocks 清理过期的终端锁
// 服务异常退出可能导致终端锁没有释放，启动时检查并清理
func (s *VideoSimpleScheduler) cleanupStaleTerminalLocks() {
	s.logger.Info("开始清理过期的终端锁")

	if s.connector == nil {
		s.logger.Warn("华为连接器未初始化，跳过终端锁清理")
		return
	}

	if err := s.connector.ClearStaleTerminalLocks(); err != nil {
		s.logger.Error("清理过期终端锁失败", zap.Error(err))
	}
}

// logTaskTimeInfo 记录任务时间信息（用于调试时区问题）
func (s *VideoSimpleScheduler) logTaskTimeInfo(task *models.VideoRecordingTask) {
	now := time.Now().UTC()
	s.logger.Info("任务时间详情",
		zap.Uint("task_id", task.ID),
		zap.String("task_name", task.Name),
		zap.String("start_time", task.StartTime.Format(time.RFC3339)),
		zap.String("start_time_zone", task.StartTime.Location().String()),
		zap.String("end_time", task.EndTime.Format(time.RFC3339)),
		zap.String("end_time_zone", task.EndTime.Location().String()),
		zap.String("current_time", now.Format(time.RFC3339)),
		zap.String("current_time_zone", now.Location().String()),
		zap.Int64("current_unix", now.Unix()),
		zap.Int64("start_unix", task.StartTime.Unix()),
		zap.Int64("end_unix", task.EndTime.Unix()),
		zap.Bool("is_expired", now.After(task.EndTime)),
	)
}
