package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/scheduler"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VideoRecordingTaskService 视频录制任务服务
type VideoRecordingTaskService struct {
	db        *gorm.DB
	logger    *zap.Logger
	scheduler scheduler.SchedulerInterface
}

// NewVideoRecordingTaskService 创建视频录制任务服务
func NewVideoRecordingTaskService(db *gorm.DB, logger *zap.Logger) *VideoRecordingTaskService {
	return &VideoRecordingTaskService{
		db:     db,
		logger: logger,
	}
}

// SetScheduler 设置调度器
func (s *VideoRecordingTaskService) SetScheduler(scheduler scheduler.SchedulerInterface) {
	s.scheduler = scheduler
}

// ListTasksRequest 任务列表请求
type ListTasksRequest struct {
	Page      int                             `form:"page"`
	PageSize  int                             `form:"page_size" binding:"max=100"`
	Keyword   string                          `form:"keyword"`
	Status    models.VideoRecordingTaskStatus `form:"status"`
	CreatedBy uint                            `form:"created_by"`
	StartDate string                          `form:"start_date"`
	EndDate   string                          `form:"end_date"`
	// 数据范围过滤字段
	UserID         uint `form:"-"` // 当前用户ID（不从query读取，由handler设置）
	IsAdmin        bool `form:"-"` // 是否管理员（不从query读取，由handler设置）
	ApplyDataScope bool `form:"-"` // 是否应用数据范围过滤
}

// ListTasksResponse 任务列表响应
type ListTasksResponse struct {
	Total int64                       `json:"total"`
	Items []models.VideoRecordingTask `json:"items"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name               string `json:"name" binding:"required,max=200"`
	Description        string `json:"description"`
	StartTime          string `json:"start_time" binding:"required"` // RFC3339
	EndTime            string `json:"end_time" binding:"required"`   // RFC3339
	PreJoinMinutes     int    `json:"pre_join_minutes" binding:"min=0,max=60"`
	RecordDelayMinutes int    `json:"record_delay_minutes" binding:"min=0,max=60"`
	ConferenceNumber   string `json:"conference_number" binding:"required,max=50"`
	HuaweiConfigID     uint   `json:"huawei_config_id"`  // 单配置（向后兼容）
	HuaweiConfigIDs    []uint `json:"huawei_config_ids"` // 多配置支持
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Name               *string `json:"name" binding:"omitempty,max=200"`
	Description        *string `json:"description"`
	StartTime          *string `json:"start_time" binding:"omitempty"` // RFC3339
	EndTime            *string `json:"end_time" binding:"omitempty"`   // RFC3339
	PreJoinMinutes     *int    `json:"pre_join_minutes" binding:"omitempty,min=0,max=60"`
	RecordDelayMinutes *int    `json:"record_delay_minutes" binding:"omitempty,min=0,max=60"`
}

// ListTasks 获取任务列表
func (s *VideoRecordingTaskService) ListTasks(req *ListTasksRequest) (*ListTasksResponse, error) {
	var tasks []models.VideoRecordingTask
	var total int64

	query := s.db.Model(&models.VideoRecordingTask{}).Preload("HuaweiConfig").Preload("Creator")

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ? OR conference_number LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 创建者筛选（手动指定的创建者）
	if req.CreatedBy > 0 {
		query = query.Where("created_by = ?", req.CreatedBy)
	}

	// 数据范围过滤：非管理员只能看自己创建的任务
	if req.ApplyDataScope && !req.IsAdmin && req.UserID > 0 {
		query = query.Where("created_by = ?", req.UserID)
	}

	// 日期范围筛选
	if req.StartDate != "" {
		if startTime, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			query = query.Where("start_time >= ?", startTime)
		}
	}
	if req.EndDate != "" {
		if endTime, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			// 包含整天
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("end_time <= ?", endTime)
		}
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	return &ListTasksResponse{
		Total: total,
		Items: tasks,
	}, nil
}

// GetTaskByID 根据ID获取任务
func (s *VideoRecordingTaskService) GetTaskByID(id uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.Preload("HuaweiConfig").Preload("Creator").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// CreateTask 创建任务
func (s *VideoRecordingTaskService) CreateTask(req *CreateTaskRequest, createdBy uint) (*models.VideoRecordingTask, error) {
	// 定义北京时间时区（UTC+8）
	beijingLocation := time.FixedZone("CST", 8*3600)

	// 解析时间 - 输入时间是北京时间，转换为 UTC 存储
	startTime, err := time.ParseInLocation(time.RFC3339, req.StartTime, beijingLocation)
	if err != nil {
		s.logger.Error("开始时间解析失败",
			zap.String("start_time", req.StartTime),
			zap.Error(err),
		)
		return nil, errors.New("开始时间格式错误，请使用 RFC3339 格式（如：2026-02-11T15:18:01）")
	}
	endTime, err := time.ParseInLocation(time.RFC3339, req.EndTime, beijingLocation)
	if err != nil {
		s.logger.Error("结束时间解析失败",
			zap.String("end_time", req.EndTime),
			zap.Error(err),
		)
		return nil, errors.New("结束时间格式错误，请使用 RFC3339 格式（如：2026-02-11T15:20:01）")
	}

	// startTime 和 endTime 已经是 UTC 时间（因为 ParseInLocation 返回的是指定时区的时间，需要显式转换）
	startTimeUTC := startTime.UTC()
	endTimeUTC := endTime.UTC()

	s.logger.Info("解析任务时间",
		zap.String("input_start_time", req.StartTime),
		zap.String("input_end_time", req.EndTime),
		zap.String("parsed_start_beijing", startTime.Format(time.RFC3339)),
		zap.String("parsed_end_beijing", endTime.Format(time.RFC3339)),
		zap.String("start_time_utc", startTimeUTC.Format(time.RFC3339)),
		zap.String("end_time_utc", endTimeUTC.Format(time.RFC3339)),
	)

	// 确定使用的配置列表
	configIDs := req.HuaweiConfigIDs
	if len(configIDs) == 0 && req.HuaweiConfigID > 0 {
		configIDs = []uint{req.HuaweiConfigID}
	}
	if len(configIDs) == 0 {
		return nil, errors.New("必须指定华为配置")
	}

	// 验证配置类型
	if err := s.validateConfigTypes(configIDs); err != nil {
		return nil, err
	}

	// 验证华为配置存在
	var firstConfig models.HuaweiConfig
	if err := s.db.First(&firstConfig, configIDs[0]).Error; err != nil {
		return nil, errors.New("华为配置不存在")
	}

	// 创建任务（使用第一个配置ID作为主配置，保持向后兼容）
	task := &models.VideoRecordingTask{
		Name:               req.Name,
		Description:        req.Description,
		StartTime:          startTimeUTC,
		EndTime:            endTimeUTC,
		PreJoinMinutes:     req.PreJoinMinutes,
		RecordDelayMinutes: req.RecordDelayMinutes,
		ConferenceNumber:   req.ConferenceNumber,
		HuaweiConfigID:     &configIDs[0],
		Status:             models.VideoStatusPending,
		CreatedBy:          createdBy,
	}

	// 验证任务数据
	if err := task.IsValid(); err != nil {
		return nil, err
	}

	if err := s.db.Create(task).Error; err != nil {
		return nil, err
	}

	// 创建关联表记录
	for _, configID := range configIDs {
		var config models.HuaweiConfig
		if err := s.db.First(&config, configID).Error; err != nil {
			s.logger.Warn("加载华为配置失败，跳过关联",
				zap.Uint("config_id", configID),
				zap.Error(err),
			)
			continue
		}

		configType := "usb"
		if config.StreamEnabled && config.StreamURL != "" {
			configType = "stream"
		}

		taskConfig := models.TaskHuaweiConfig{
			TaskID:         task.ID,
			HuaweiConfigID: configID,
			ConfigType:     configType,
		}
		s.db.Create(&taskConfig)
	}

	// 重新加载关联数据
	s.db.Preload("HuaweiConfig").Preload("Creator").Preload("TaskHuaweiConfigs").First(task, task.ID)

	s.logger.Info("录制任务已创建",
		zap.Uint("task_id", task.ID),
		zap.String("name", task.Name),
		zap.Uint("created_by", createdBy),
		zap.Time("start_time", task.StartTime),
		zap.Time("end_time", task.EndTime),
	)

	// 同步任务到调度器（使用 goroutine 避免阻塞请求）
	if s.scheduler != nil {
		go func() {
			if err := s.scheduler.SyncPendingTasks(); err != nil {
				s.logger.Error("同步任务到调度器失败",
					zap.Uint("task_id", task.ID),
					zap.Error(err),
				)
			} else {
				s.logger.Info("任务已同步到调度器",
					zap.Uint("task_id", task.ID),
				)
			}
		}()
	}

	return task, nil
}

// UpdateTask 更新任务
func (s *VideoRecordingTaskService) UpdateTask(id uint, req *UpdateTaskRequest, userID uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, errors.New("任务不存在")
	}

	// 检查权限
	if task.CreatedBy != userID {
		return nil, errors.New("无权限修改此任务")
	}

	// 待执行状态：可以更新所有字段
	// 录制中状态：只能更新结束时间
	isRecording := task.Status == models.VideoStatusRecording

	if !isRecording && task.Status != models.VideoStatusPending {
		return nil, errors.New("只能更新待执行状态或录制中状态的任务")
	}

	// 更新字段
	updates := make(map[string]interface{})

	if !isRecording {
		// 非录制状态可以更新所有字段
		if req.Name != nil {
			updates["name"] = *req.Name
		}
		if req.Description != nil {
			updates["description"] = *req.Description
		}
		if req.StartTime != nil {
			// 按北京时间解析，转换为 UTC 存储
			beijingLocation := time.FixedZone("CST", 8*3600)
			startTime, err := time.ParseInLocation(time.RFC3339, *req.StartTime, beijingLocation)
			if err != nil {
				return nil, errors.New("开始时间格式错误")
			}
			updates["start_time"] = startTime.UTC()
		}
		if req.PreJoinMinutes != nil {
			updates["pre_join_minutes"] = *req.PreJoinMinutes
		}
		if req.RecordDelayMinutes != nil {
			updates["record_delay_minutes"] = *req.RecordDelayMinutes
		}
	}

	// 录制中和待执行状态都可以更新结束时间
	if req.EndTime != nil {
		// 按北京时间解析，转换为 UTC 存储
		beijingLocation := time.FixedZone("CST", 8*3600)
		endTime, err := time.ParseInLocation(time.RFC3339, *req.EndTime, beijingLocation)
		if err != nil {
			return nil, errors.New("结束时间格式错误")
		}

		// 验证结束时间必须在开始时间之后
		var newStartTime time.Time
		if req.StartTime != nil {
			beijingLocation := time.FixedZone("CST", 8*3600)
			newStartTime, _ = time.ParseInLocation(time.RFC3339, *req.StartTime, beijingLocation)
		} else {
			newStartTime = task.StartTime
		}

		if endTime.Before(newStartTime) {
			return nil, errors.New("结束时间不能早于开始时间")
		}

		updates["end_time"] = endTime.UTC()

		// 如果是录制中的任务，需要通知调度器更新监控定时器
		if isRecording && s.scheduler != nil {
			if err := s.scheduler.UpdateTaskEndTime(id, endTime.UTC()); err != nil {
				s.logger.Warn("更新调度器任务结束时间失败",
					zap.Uint("task_id", id),
					zap.Error(err),
				)
				// 不阻止更新继续执行
			}
		}
	}

	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新加载数据
	s.db.Preload("HuaweiConfig").Preload("Creator").First(&task, id)

	// 验证更新后的数据
	if err := task.IsValid(); err != nil {
		return nil, err
	}

	s.logger.Info("录制任务已更新",
		zap.Uint("task_id", id),
		zap.String("status", string(task.Status)),
		zap.Uint("updated_by", userID),
	)

	return &task, nil
}

// DeleteTask 删除任务
func (s *VideoRecordingTaskService) DeleteTask(id uint, userID uint, isAdmin bool) error {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, id).Error; err != nil {
		return errors.New("任务不存在")
	}

	// 只能删除非运行状态的任务（运行中的任务不能删除）
	if task.Status == models.VideoStatusRecording || task.Status == models.VideoStatusConnecting {
		return errors.New("运行中的任务无法删除，请先停止任务")
	}

	// 检查权限（管理员可以删除任何任务）
	if !isAdmin && task.CreatedBy != userID {
		return errors.New("无权限删除此任务")
	}

	// 删除前先解锁终端（防止锁遗留）
	// 如果任务有华为配置ID，尝试解锁该配置
	if task.HuaweiConfigID != nil && *task.HuaweiConfigID > 0 {
		updates := map[string]interface{}{
			"is_locked": false,
			"locked_by": nil,
		}
		s.db.Model(&models.HuaweiConfig{}).Where("id = ?", *task.HuaweiConfigID).Updates(updates)
		s.logger.Info("删除任务时解锁终端",
			zap.Uint("task_id", id),
			zap.Uint("huawei_config_id", *task.HuaweiConfigID),
		)
	}

	result := s.db.Delete(&models.VideoRecordingTask{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("任务不存在")
	}

	s.logger.Info("录制任务已删除",
		zap.Uint("task_id", id),
		zap.Uint("deleted_by", userID),
		zap.String("task_status", string(task.Status)),
	)

	return nil
}

// BatchDeleteTasksRequest 批量删除任务请求
type BatchDeleteTasksRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchDeleteTasksResult 批量删除任务结果
type BatchDeleteTasksResult struct {
	DeletedIDs   []uint   `json:"deleted_ids"`   // 成功删除的任务 ID
	FailedIDs    []uint   `json:"failed_ids"`    // 删除失败的任务 ID
	FailedTasks  []string `json:"failed_tasks"`  // 删除失败的任务名称及原因
	TotalDeleted int      `json:"total_deleted"` // 成功删除的总数
	TotalFailed  int      `json:"total_failed"`  // 删除失败的总数
}

// canDeleteTask 检查任务是否可删除
func (s *VideoRecordingTaskService) canDeleteTask(task models.VideoRecordingTask, userID uint, isAdmin bool) (bool, string) {
	// 检查权限（管理员可以删除任何任务）
	if !isAdmin && task.CreatedBy != userID {
		return false, "无权限"
	}
	// 转换为小写进行比较，忽略大小写差异
	status := strings.ToLower(string(task.Status))
	recording := strings.ToLower(string(models.VideoStatusRecording))
	connecting := strings.ToLower(string(models.VideoStatusConnecting))

	// 添加详细日志
	s.logger.Info("删除权限检查",
		zap.Uint("task_id", task.ID),
		zap.String("status", string(task.Status)),
		zap.String("status_lower", status),
		zap.Bool("can_delete", status != recording && status != connecting),
	)

	if status == recording || status == connecting {
		return false, "运行中"
	}
	return true, ""
}

// BatchDeleteTasks 批量删除任务
func (s *VideoRecordingTaskService) BatchDeleteTasks(ids []uint, userID uint, isAdmin bool) (*BatchDeleteTasksResult, error) {
	if len(ids) == 0 {
		return nil, errors.New("任务ID列表不能为空")
	}

	var tasks []models.VideoRecordingTask
	if err := s.db.Where("id IN ?", ids).Find(&tasks).Error; err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, errors.New("任务不存在")
	}

	result := &BatchDeleteTasksResult{
		DeletedIDs:  make([]uint, 0),
		FailedIDs:   make([]uint, 0),
		FailedTasks: make([]string, 0),
	}

	for _, task := range tasks {
		if canDelete, reason := s.canDeleteTask(task, userID, isAdmin); canDelete {
			result.DeletedIDs = append(result.DeletedIDs, task.ID)
		} else {
			result.FailedIDs = append(result.FailedIDs, task.ID)
			result.FailedTasks = append(result.FailedTasks, fmt.Sprintf("%s（%s）", task.Name, reason))
		}
	}

	if len(result.DeletedIDs) == 0 {
		return result, errors.New("没有可删除的任务")
	}

	// 删除前先解锁所有待删除任务的终端（防止锁遗留）
	for _, task := range tasks {
		if task.HuaweiConfigID != nil && *task.HuaweiConfigID > 0 {
			updates := map[string]interface{}{
				"is_locked": false,
				"locked_by": nil,
			}
			s.db.Model(&models.HuaweiConfig{}).Where("id = ?", *task.HuaweiConfigID).Updates(updates)
		}
	}
	s.logger.Info("批量删除任务时解锁终端",
		zap.Int("count", len(result.DeletedIDs)),
	)

	// 执行删除
	dbResult := s.db.Delete(&models.VideoRecordingTask{}, result.DeletedIDs)
	if dbResult.Error != nil {
		return nil, dbResult.Error
	}

	result.TotalDeleted = int(dbResult.RowsAffected)
	result.TotalFailed = len(result.FailedIDs)

	s.logger.Info("批量删除录制任务",
		zap.Int("deleted", result.TotalDeleted),
		zap.Int("failed", result.TotalFailed),
		zap.Uint("deleted_by", userID),
	)

	if result.TotalFailed > 0 {
		s.logger.Warn("部分任务无法删除", zap.Strings("tasks", result.FailedTasks))
	}

	return result, nil
}

// StartTask 手动启动任务
func (s *VideoRecordingTaskService) StartTask(id uint, userID uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, errors.New("任务不存在")
	}

	// 检查权限
	if task.CreatedBy != userID {
		return nil, errors.New("无权限操作此任务")
	}

	// 检查状态
	if task.Status != models.VideoStatusPending {
		return nil, errors.New("只能启动待执行状态的任务")
	}

	// 触发任务执行
	if s.scheduler != nil {
		if err := s.scheduler.ExecuteTask(id); err != nil {
			return nil, fmt.Errorf("触发任务执行失败: %w", err)
		}
	} else {
		return nil, errors.New("调度器未初始化")
	}

	s.logger.Info("录制任务已手动启动",
		zap.Uint("task_id", id),
		zap.Uint("started_by", userID),
	)

	return &task, nil
}

// StopTask 手动停止任务
func (s *VideoRecordingTaskService) StopTask(id uint, userID uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, errors.New("任务不存在")
	}

	// 检查权限
	if task.CreatedBy != userID {
		return nil, errors.New("无权限操作此任务")
	}

	// 检查状态
	if task.Status != models.VideoStatusRecording && task.Status != models.VideoStatusConnecting {
		return nil, errors.New("只能停止录制中或连接中的任务")
	}

	// 记录原始状态，用于后续判断
	wasRecording := task.Status == models.VideoStatusRecording

	// 取消任务执行（这会停止录制并创建文件记录、提交转换任务）
	if s.scheduler != nil {
		if err := s.scheduler.CancelTaskExecution(id); err != nil {
			s.logger.Warn("取消任务执行失败", zap.Error(err))
			// 继续执行状态更新
		}
	}

	// 重新加载任务以获取最新状态（因为 CancelTaskExecution 可能创建了 MKV 文件）
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, err
	}

	// 根据是否有录制文件来决定最终状态
	// 如果已经有 MKV 文件，说明已经开始了录制，应该进入转换状态
	// 如果没有 MKV 文件且原状态不是 recording，说明还没开始录制，状态为已取消
	if task.MKVFilePath != "" || wasRecording {
		// 已经开始录制，进入转换状态
		task.Status = models.VideoStatusConverting
		s.logger.Info("录制任务已手动停止，进入转换状态",
			zap.Uint("task_id", id),
			zap.Uint("stopped_by", userID),
			zap.String("mkv_file", task.MKVFilePath),
		)
	} else {
		// 还没开始录制，直接取消
		task.Status = models.VideoStatusCancelled
		s.logger.Info("录制任务已取消（未开始录制）",
			zap.Uint("task_id", id),
			zap.Uint("cancelled_by", userID),
		)
	}

	if err := s.db.Save(&task).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

// CancelTask 取消任务
func (s *VideoRecordingTaskService) CancelTask(id uint, userID uint) error {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, id).Error; err != nil {
		return errors.New("任务不存在")
	}

	// 检查权限
	if task.CreatedBy != userID {
		return errors.New("无权限操作此任务")
	}

	// 检查状态
	if task.Status != models.VideoStatusPending && task.Status != models.VideoStatusConnecting {
		return errors.New("只能取消待执行或连接中的任务")
	}

	// 从调度器移除
	if s.scheduler != nil {
		if err := s.scheduler.RemoveTask(id); err != nil {
			s.logger.Warn("从调度器移除任务失败", zap.Error(err))
		}
	}

	// 更新状态
	task.Status = models.VideoStatusCancelled
	if err := s.db.Save(&task).Error; err != nil {
		return err
	}

	s.logger.Info("录制任务已取消",
		zap.Uint("task_id", id),
		zap.Uint("cancelled_by", userID),
	)

	return nil
}

// RetryTask 重试失败任务
func (s *VideoRecordingTaskService) RetryTask(id uint, userID uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, id).Error; err != nil {
		return nil, errors.New("任务不存在")
	}

	// 检查权限
	if task.CreatedBy != userID {
		return nil, errors.New("无权限操作此任务")
	}

	// 检查状态
	if task.Status != models.VideoStatusFailed {
		return nil, errors.New("只能重试失败的任务")
	}

	// 重置状态
	task.Status = models.VideoStatusPending
	task.ErrorMsg = ""
	task.RecordingFile = ""
	task.RecordingDuration = 0

	// 重新计算触发时间 (当前时间 + 1分钟)
	newTriggerTime := time.Now().Add(1 * time.Minute)
	task.StartTime = newTriggerTime.Add(time.Duration(task.PreJoinMinutes) * time.Minute)
	task.EndTime = task.StartTime.Add(task.EndTime.Sub(task.StartTime))

	if err := s.db.Save(&task).Error; err != nil {
		return nil, err
	}

	// 重新调度任务
	if s.scheduler != nil {
		go func() {
			if err := s.scheduler.SyncPendingTasks(); err != nil {
				s.logger.Error("重新调度任务失败",
					zap.Uint("task_id", id),
					zap.Error(err),
				)
			}
		}()
	}

	s.logger.Info("录制任务已重试",
		zap.Uint("task_id", id),
		zap.Uint("retried_by", userID),
		zap.Time("new_start_time", task.StartTime),
	)

	return &task, nil
}

// GetTasksByStatus 根据状态获取任务列表
func (s *VideoRecordingTaskService) GetTasksByStatus(status models.VideoRecordingTaskStatus) ([]models.VideoRecordingTask, error) {
	var tasks []models.VideoRecordingTask
	if err := s.db.Where("status = ?", status).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetPendingTasks 获取待执行的任务
func (s *VideoRecordingTaskService) GetPendingTasks() ([]models.VideoRecordingTask, error) {
	return s.GetTasksByStatus(models.VideoStatusPending)
}

// UpdateTaskStatus 更新任务状态
func (s *VideoRecordingTaskService) UpdateTaskStatus(id uint, status models.VideoRecordingTaskStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	result := s.db.Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("任务不存在")
	}

	return nil
}

// UpdateRecordingInfo 更新录制信息
func (s *VideoRecordingTaskService) UpdateRecordingInfo(id uint, filePath string, duration int) error {
	updates := map[string]interface{}{
		"recording_file":     filePath,
		"recording_duration": duration,
	}

	result := s.db.Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("任务不存在")
	}

	return nil
}

// UpdateRecordingPaths 更新录制文件路径
func (s *VideoRecordingTaskService) UpdateRecordingPaths(id uint, mkvPath, hlsPath string) error {
	s.logger.Debug("更新录制文件路径",
		zap.Uint("task_id", id),
		zap.String("mkv_path", mkvPath),
		zap.String("hls_path", hlsPath),
	)

	updates := map[string]interface{}{
		"recording_file":   mkvPath,
		"mkv_file_path":    mkvPath,
		"hls_preview_path": hlsPath,
	}

	result := s.db.Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("任务不存在")
	}
	return nil
}

// validateConfigTypes 验证配置类型（不能多选同类型）
func (s *VideoRecordingTaskService) validateConfigTypes(configIDs []uint) error {
	if len(configIDs) == 0 {
		return errors.New("至少需要选择一个华为配置")
	}

	var configs []models.HuaweiConfig
	if err := s.db.Where("id IN ?", configIDs).Find(&configs).Error; err != nil {
		return err
	}

	usbCount := 0
	streamCount := 0

	for _, config := range configs {
		// 判断配置类型：有USB设备配置或启用了流媒体
		hasUSB := config.USBCameraDevice != "" || config.USBAudioDevice != ""
		hasStream := config.StreamEnabled && config.StreamURL != ""

		if hasUSB {
			usbCount++
		}
		if hasStream {
			streamCount++
		}

		// 既不是USB也不是流媒体的配置
		if !hasUSB && !hasStream {
			return fmt.Errorf("配置 %s 未配置USB设备或流媒体", config.Name)
		}
	}

	if usbCount > 1 {
		return errors.New("最多只能选择1个USB配置")
	}
	if streamCount > 1 {
		return errors.New("最多只能选择1个流媒体配置")
	}

	return nil
}

// GetDB 获取数据库连接（供调度器使用）
func (s *VideoRecordingTaskService) GetDB() *gorm.DB {
	return s.db
}
