package services

import (
	"errors"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/scheduler"
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
	Page         int                            `form:"page"`
	PageSize     int                            `form:"page_size" binding:"max=100"`
	Keyword      string                         `form:"keyword"`
	Status       models.VideoRecordingTaskStatus `form:"status"`
	CreatedBy    uint                           `form:"created_by"`
	StartDate    string                         `form:"start_date"`
	EndDate      string                         `form:"end_date"`
}

// ListTasksResponse 任务列表响应
type ListTasksResponse struct {
	Total int64                      `json:"total"`
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
	HuaweiConfigID     uint   `json:"huawei_config_id" binding:"required"`
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

	// 创建者筛选
	if req.CreatedBy > 0 {
		query = query.Where("created_by = ?", req.CreatedBy)
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
	if err := s.db.Preload("HuaweiConfig").Preload("Creator").Preload("ConferenceRecord").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// CreateTask 创建任务
func (s *VideoRecordingTaskService) CreateTask(req *CreateTaskRequest, createdBy uint) (*models.VideoRecordingTask, error) {
	// 解析时间
	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, errors.New("开始时间格式错误")
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, errors.New("结束时间格式错误")
	}

	// 验证华为配置存在
	var config models.HuaweiConfig
	if err := s.db.First(&config, req.HuaweiConfigID).Error; err != nil {
		return nil, errors.New("华为配置不存在")
	}

	// 创建任务
	task := &models.VideoRecordingTask{
		Name:               req.Name,
		Description:        req.Description,
		StartTime:          startTime,
		EndTime:            endTime,
		PreJoinMinutes:     req.PreJoinMinutes,
		RecordDelayMinutes: req.RecordDelayMinutes,
		ConferenceNumber:   req.ConferenceNumber,
		HuaweiConfigID:     req.HuaweiConfigID,
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

	// 重新加载关联数据
	s.db.Preload("HuaweiConfig").Preload("Creator").First(task, task.ID)

	s.logger.Info("Video recording task created",
		zap.Uint("task_id", task.ID),
		zap.String("name", task.Name),
		zap.Uint("created_by", createdBy),
	)

	// 同步任务到调度器
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

	// 只能更新待执行状态的任务
	if task.Status != models.VideoStatusPending {
		return nil, errors.New("只能更新待执行状态的任务")
	}

	// 检查权限
	if task.CreatedBy != userID {
		return nil, errors.New("无权限修改此任务")
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.StartTime != nil {
		startTime, err := time.Parse(time.RFC3339, *req.StartTime)
		if err != nil {
			return nil, errors.New("开始时间格式错误")
		}
		updates["start_time"] = startTime
	}
	if req.EndTime != nil {
		endTime, err := time.Parse(time.RFC3339, *req.EndTime)
		if err != nil {
			return nil, errors.New("结束时间格式错误")
		}
		updates["end_time"] = endTime
	}
	if req.PreJoinMinutes != nil {
		updates["pre_join_minutes"] = *req.PreJoinMinutes
	}
	if req.RecordDelayMinutes != nil {
		updates["record_delay_minutes"] = *req.RecordDelayMinutes
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

	s.logger.Info("Video recording task updated",
		zap.Uint("task_id", id),
		zap.Uint("updated_by", userID),
	)

	return &task, nil
}

// DeleteTask 删除任务
func (s *VideoRecordingTaskService) DeleteTask(id uint, userID uint) error {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, id).Error; err != nil {
		return errors.New("任务不存在")
	}

	// 只能删除待执行状态的任务
	if task.Status != models.VideoStatusPending {
		return errors.New("只能删除待执行状态的任务")
	}

	// 检查权限
	if task.CreatedBy != userID {
		return errors.New("无权限删除此任务")
	}

	result := s.db.Delete(&models.VideoRecordingTask{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("任务不存在")
	}

	s.logger.Info("Video recording task deleted",
		zap.Uint("task_id", id),
		zap.Uint("deleted_by", userID),
	)

	return nil
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

	// 更新状态为连接中
	task.Status = models.VideoStatusConnecting
	if err := s.db.Save(&task).Error; err != nil {
		return nil, err
	}

	// TODO: 触发任务执行逻辑
	// 这里应该调用 scheduler.executeTask(task.ID)

	s.logger.Info("Video recording task started manually",
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

	// 更新状态为已取消
	task.Status = models.VideoStatusCancelled
	if err := s.db.Save(&task).Error; err != nil {
		return nil, err
	}

	// TODO: 触发停止逻辑
	// 这里应该调用 scheduler.stopTask(task.ID)

	s.logger.Info("Video recording task stopped manually",
		zap.Uint("task_id", id),
		zap.Uint("stopped_by", userID),
	)

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

	// TODO: 从调度器移除
	// s.scheduler.RemoveTask(id)

	// 更新状态
	task.Status = models.VideoStatusCancelled
	if err := s.db.Save(&task).Error; err != nil {
		return err
	}

	s.logger.Info("Video recording task cancelled",
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

	// TODO: 重新调度
	// s.scheduler.ScheduleTask(task)

	s.logger.Info("Video recording task retried",
		zap.Uint("task_id", id),
		zap.Uint("retried_by", userID),
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
