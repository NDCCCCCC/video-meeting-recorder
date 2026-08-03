package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/scheduler"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// VideoRecordingTaskService 视频录制任务服务
// Phase 19 Wave 5 (PERF-003/BUG-005)：全部 DB-touching 方法已改为 ctx context.Context
// 首参，并在每个 s.db. 链首加 .WithContext(ctx)，以支持优雅关停时的查询取消与 HTTP
// 超时级联。纯 getter/setter（SetScheduler / GetDB）与纯逻辑辅助（canDeleteTask）保留无参。
//
// Phase 19 D2：新增 encryptor 字段（Phase 18 SM4-GCM 凭据解密从 cmd/server
// taskServiceAdapter 整合到此服务），消除 scheduler 与服务之间的双层签名级联与
// 双源 truth。
type VideoRecordingTaskService struct {
	db        *gorm.DB
	logger    *zap.Logger
	scheduler scheduler.SchedulerInterface
	// encryptor Phase 18 SM4-GCM 凭据静态解密。GetInputConfig 末尾在 encryptor!=nil
	// 时解密 Password/StreamPassword；nil 时透传密文。
	encryptor *CredentialEncryptor
}

// NewVideoRecordingTaskService 创建视频录制任务服务
// Phase 19 D2: 增加可选 CredentialEncryptor 参数；传 nil 行为等同 Phase 18 之前（密文透传）。
func NewVideoRecordingTaskService(db *gorm.DB, logger *zap.Logger, encryptor ...*CredentialEncryptor) *VideoRecordingTaskService {
	s := &VideoRecordingTaskService{
		db:     db,
		logger: logger,
	}
	if len(encryptor) > 0 {
		s.encryptor = encryptor[0]
	}
	return s
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
	UserID         uint         `form:"-"` // 当前用户ID（不从query读取，由handler设置）
	IsAdmin        bool         `form:"-"` // 是否管理员（不从query读取，由handler设置）
	ApplyDataScope bool         `form:"-"` // 是否应用数据范围过滤
	User           *models.User `form:"-"` // User object with Roles preloaded for visibility control (D-11, D-12)
	RoleIDs        []uint       `form:"-"` // Role IDs from token claims for shared_viewer check (D-02)
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
	InputConfigIDs     []uint `json:"input_config_ids"` // 输入配置ID列表
}

// CreateTaskAutoRequest 自动创建任务请求（不需要传入华为配置）
type CreateTaskAutoRequest struct {
	Name               string `json:"name" binding:"required,max=200"`
	Description        string `json:"description"`
	StartTime          string `json:"start_time" binding:"required"` // RFC3339
	EndTime            string `json:"end_time" binding:"required"`   // RFC3339
	PreJoinMinutes     int    `json:"pre_join_minutes" binding:"min=0,max=60"`
	RecordDelayMinutes int    `json:"record_delay_minutes" binding:"min=0,max=60"`
	ConferenceNumber   string `json:"conference_number" binding:"required,max=50"`
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

// attachTaskExtras PERF-001/PR-E: 把原本靠 GORM 链式 Preload 触发的 N+1 (1+3×N 次查询)
// 改造为单次 IN-clause 批量加载 (N=1 时 4 次降到 1 次; N=100 时 301 次降到 4 次)。
// 保持原有 struct 字段填充,JSON 序列化结果与改造前一致。
//
// populates:
//   - tasks[i].InputConfig
//   - tasks[i].Creator
//   - tasks[i].TaskInputConfigs
//
// 调用方需保证 task.InputConfigID / task.CreatedBy / task.ID 已被 Find 加载。
func (s *VideoRecordingTaskService) attachTaskExtras(ctx context.Context, tasks []models.VideoRecordingTask) error {
	if len(tasks) == 0 {
		return nil
	}

	// 1. collect unique ids
	inputConfigIDs := make([]uint, 0, len(tasks))
	creatorIDs := make([]uint, 0, len(tasks))
	taskIDs := make([]uint, 0, len(tasks))
	seenIC := make(map[uint]struct{})
	seenCr := make(map[uint]struct{})
	for i := range tasks {
		t := &tasks[i]
		if t.InputConfigID != nil {
			if _, ok := seenIC[*t.InputConfigID]; !ok {
				seenIC[*t.InputConfigID] = struct{}{}
				inputConfigIDs = append(inputConfigIDs, *t.InputConfigID)
			}
		}
		if t.CreatedBy > 0 {
			if _, ok := seenCr[t.CreatedBy]; !ok {
				seenCr[t.CreatedBy] = struct{}{}
				creatorIDs = append(creatorIDs, t.CreatedBy)
			}
		}
		taskIDs = append(taskIDs, t.ID)
	}

	// 2. input configs (omit password fields explicitly)
	var configs []models.InputConfig
	if len(inputConfigIDs) > 0 {
		err := s.db.WithContext(ctx).
			Select("id, name, config_type, huawei_enabled, stream_url, usb_camera_device, is_locked, locked_by").
			Where("id IN ?", inputConfigIDs).
			Find(&configs).Error
		if err != nil {
			return err
		}
	}

	// 3. creators (只取展示字段)
	var creators []models.User
	if len(creatorIDs) > 0 {
		err := s.db.WithContext(ctx).
			Select("id, username, full_name").
			Where("id IN ?", creatorIDs).
			Find(&creators).Error
		if err != nil {
			return err
		}
	}

	// 4. many-side TaskInputConfigs
	var tics []models.TaskInputConfig
	if len(taskIDs) > 0 {
		err := s.db.WithContext(ctx).
			Select("id, task_id, input_config_id, config_type").
			Where("task_id IN ?", taskIDs).
			Find(&tics).Error
		if err != nil {
			return err
		}
	}

	// 5. stitch into maps
	icByID := make(map[uint]*models.InputConfig, len(configs))
	for i := range configs {
		icByID[configs[i].ID] = &configs[i]
	}
	creatorByID := make(map[uint]*models.User, len(creators))
	for i := range creators {
		creatorByID[creators[i].ID] = &creators[i]
	}
	ticsByTaskID := make(map[uint][]models.TaskInputConfig, len(taskIDs))
	for _, tic := range tics {
		ticsByTaskID[tic.TaskID] = append(ticsByTaskID[tic.TaskID], tic)
	}

	// 6. attach
	for i := range tasks {
		t := &tasks[i]
		if t.InputConfigID != nil {
			if c, ok := icByID[*t.InputConfigID]; ok {
				t.InputConfig = c
			}
		}
		if t.CreatedBy > 0 {
			if u, ok := creatorByID[t.CreatedBy]; ok {
				t.Creator = u
			}
		}
		if list, ok := ticsByTaskID[t.ID]; ok {
			t.TaskInputConfigs = list
		}
	}
	return nil
}

// ListTasks 获取任务列表
func (s *VideoRecordingTaskService) ListTasks(ctx context.Context, req *ListTasksRequest) (*ListTasksResponse, error) {
	var tasks []models.VideoRecordingTask
	var total int64

	// PERF-001/PR-E: 不再用 triple-Preload (1+3×N 查询);改为 Find 后批量填充。
	query := s.db.WithContext(ctx).Model(&models.VideoRecordingTask{})

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

	// DATA VISIBILITY: Shared viewers see all data (D-02, D-11, D-12)
	// Visibility check (data scope) happens before permission checks (operation authorization)
	// shared_viewer affects data scope, not operation permissions (D-01, D-03)
	if req.ApplyDataScope {
		// Check if user has shared_viewer role
		hasSharedViewer := false
		if req.User != nil {
			// Check from User object (preferred)
			hasSharedViewer = req.User.HasRole(models.RoleSharedViewer)
		} else if len(req.RoleIDs) > 0 {
			// Check from token claims RoleIDs (fallback)
			// shared_viewer role ID is 5
			for _, roleID := range req.RoleIDs {
				if roleID == 5 { // RoleSharedViewer ID
					hasSharedViewer = true
					break
				}
			}
		}

		// Apply created_by filter only for non-admin, non-shared-viewer users
		if !req.IsAdmin && !hasSharedViewer && req.UserID > 0 {
			// Non-shared-viewers only see their own data (D-10)
			query = query.Where("created_by = ?", req.UserID)
		}
		// shared_viewers and admins skip created_by filter to see all data (D-02)
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

	// PERF-001/PR-E: 把 3 个链式 Preload 折叠为单次 IN-clause 批量加载。
	if err := s.attachTaskExtras(ctx, tasks); err != nil {
		return nil, err
	}

	return &ListTasksResponse{
		Total: total,
		Items: tasks,
	}, nil
}

// GetTaskByID 根据ID获取任务
func (s *VideoRecordingTaskService) GetTaskByID(ctx context.Context, id uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).Preload("InputConfig").Preload("TaskInputConfigs").Preload("Creator").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// CreateTask 创建任务
func (s *VideoRecordingTaskService) CreateTask(ctx context.Context, req *CreateTaskRequest, createdBy uint) (*models.VideoRecordingTask, error) {
	// 定义北京时间时区（UTC+8）
	beijingLocation := time.FixedZone("CST", 8*3600)

	// 解析时间 - 输入时间是北京时间，转换为 UTC 存储
	startTime, err := time.ParseInLocation(time.RFC3339, req.StartTime, beijingLocation)
	if err != nil {
		s.logger.Error("开始时间解析失败",
			zap.String("start_time", req.StartTime),
			zap.Error(err),
			response.SentinelField(err),
		)
		return nil, apperrors.ErrInvalidInput
	}
	endTime, err := time.ParseInLocation(time.RFC3339, req.EndTime, beijingLocation)
	if err != nil {
		s.logger.Error("结束时间解析失败",
			zap.String("end_time", req.EndTime),
			zap.Error(err),
			response.SentinelField(err),
		)
		return nil, apperrors.ErrInvalidInput
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

	// 输入配置现在是可选的，允许创建 USB/纯流录制任务
	// 仅当有配置时才验证配置类型和存在性
	if len(req.InputConfigIDs) > 0 {
		// 验证配置存在（使用 input_configs 表）
		var firstConfig models.InputConfig
		if err := s.db.WithContext(ctx).First(&firstConfig, req.InputConfigIDs[0]).Error; err != nil {
			return nil, apperrors.ErrNotFound
		}
	}

	// 创建任务
	// 设置主配置ID（用于兼容 IsValid 检查）
	var primaryConfigID *uint
	if len(req.InputConfigIDs) > 0 {
		primaryConfigID = &req.InputConfigIDs[0]
	}

	task := &models.VideoRecordingTask{
		Name:               req.Name,
		Description:        req.Description,
		StartTime:          startTimeUTC,
		EndTime:            endTimeUTC,
		PreJoinMinutes:     req.PreJoinMinutes,
		RecordDelayMinutes: req.RecordDelayMinutes,
		ConferenceNumber:   req.ConferenceNumber,
		Status:             models.VideoStatusPending,
		CreatedBy:          createdBy,
		InputConfigID:      primaryConfigID,
	}

	// 验证任务数据
	if err := task.IsValid(); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}

	// 创建关联表记录
	for _, configID := range req.InputConfigIDs {
		var config models.InputConfig
		if err := s.db.WithContext(ctx).First(&config, configID).Error; err != nil {
			s.logger.Warn("加载华为配置失败，跳过关联",
				zap.Uint("config_id", configID),
				zap.Error(err),
				response.SentinelField(err),
			)
			continue
		}

		// 确定配置类型
		var configType string
		if config.HuaweiEnabled {
			// 华为控制启用，标记为huawei_auto
			configType = models.ConfigTypeHuaweiAuto
		} else if config.StreamEnabled && config.StreamURL != "" {
			// 流媒体配置
			configType = models.ConfigTypeStream
		} else if config.USBCameraDevice != "" || config.USBAudioDevice != "" {
			// USB配置
			configType = models.ConfigTypeUSB
		} else {
			// 既没有华为控制，也没有流媒体或USB，跳过
			s.logger.Warn("配置未指定任何输入源（华为/USB/流媒体），跳过",
				zap.Uint("config_id", configID),
				zap.String("config_name", config.Name),
			)
			continue
		}

		taskConfig := models.TaskInputConfig{
			TaskID:        task.ID,
			InputConfigID: configID,
			ConfigType:    configType,
		}
		s.db.WithContext(ctx).Create(&taskConfig)
	}

	// 重新加载关联数据
	s.db.WithContext(ctx).Preload("Creator").Preload("TaskInputConfigs").First(task, task.ID)

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
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("scheduler sync goroutine panicked",
						zap.Any("recover", r), zap.Stack("stack"))
				}
			}()
			if err := s.scheduler.SyncPendingTasks(); err != nil {
				s.logger.Error("同步任务到调度器失败",
					zap.Uint("task_id", task.ID),
					zap.Error(err),
					response.SentinelField(err),
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

// CreateTaskAuto 自动创建任务（固定华为配置ID为1）
func (s *VideoRecordingTaskService) CreateTaskAuto(ctx context.Context, req *CreateTaskAutoRequest, createdBy uint) (*models.VideoRecordingTask, error) {
	// 构造标准创建请求，固定华为配置ID为1
	standardReq := &CreateTaskRequest{
		Name:               req.Name,
		Description:        req.Description,
		StartTime:          req.StartTime,
		EndTime:            req.EndTime,
		PreJoinMinutes:     req.PreJoinMinutes,
		RecordDelayMinutes: req.RecordDelayMinutes,
		ConferenceNumber:   req.ConferenceNumber,
		InputConfigIDs:     []uint{}, // Empty for auto mode
	}
	return s.CreateTask(ctx, standardReq, createdBy)
}

// UpdateTask 更新任务
func (s *VideoRecordingTaskService) UpdateTask(ctx context.Context, id uint, req *UpdateTaskRequest, userID uint, hasSharedViewer bool) (*models.VideoRecordingTask, *models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, nil, apperrors.ErrTaskNotFound
	}

	// 检查权限 (shared_viewers 可以修改任何任务)
	if !hasSharedViewer && task.CreatedBy != userID {
		return nil, nil, apperrors.ErrForbidden
	}

	// 待执行状态：可以更新所有字段
	// 录制中状态：只能更新结束时间
	isRecording := task.Status == models.VideoStatusRecording

	if !isRecording && task.Status != models.VideoStatusPending {
		return nil, nil, apperrors.ErrInvalidInput
	}

	// Snapshot before mutation for audit OldData capture
	oldTask := task

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
				return nil, nil, apperrors.ErrInvalidInput
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
			return nil, nil, apperrors.ErrInvalidInput
		}

		// 验证结束时间必须在开始时间之后
		var newStartTime time.Time
		if req.StartTime != nil {
			beijingLocation := time.FixedZone("CST", 8*3600)
			parsedStartTime, parseErr := time.ParseInLocation(time.RFC3339, *req.StartTime, beijingLocation)
			if parseErr != nil {
				s.logger.Warn("开始时间解析失败", zap.Error(parseErr), response.SentinelField(parseErr))
				return nil, nil, apperrors.ErrInvalidInput
			}
			newStartTime = parsedStartTime
		} else {
			newStartTime = task.StartTime
		}

		if endTime.Before(newStartTime) {
			return nil, nil, apperrors.ErrInvalidInput
		}

		updates["end_time"] = endTime.UTC()

		// 如果是录制中的任务，需要通知调度器更新监控定时器
		if isRecording && s.scheduler != nil {
			if err := s.scheduler.UpdateTaskEndTime(id, endTime.UTC()); err != nil {
				s.logger.Warn("更新调度器任务结束时间失败",
					zap.Uint("task_id", id),
					zap.Error(err),
					response.SentinelField(err),
				)
				// 不阻止更新继续执行
			}
		}
	}

	if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
		return nil, nil, err
	}

	// 重新加载数据
	s.db.WithContext(ctx).Preload("InputConfig").Preload("TaskInputConfigs").Preload("Creator").First(&task, id)

	// 验证更新后的数据
	if err := task.IsValid(); err != nil {
		return nil, nil, err
	}

	s.logger.Info("录制任务已更新",
		zap.Uint("task_id", id),
		zap.String("status", string(task.Status)),
		zap.Uint("updated_by", userID),
	)

	return &oldTask, &task, nil
}

// DeleteTask 删除任务
func (s *VideoRecordingTaskService) DeleteTask(ctx context.Context, id, userID uint, isAdmin bool) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, apperrors.ErrTaskNotFound
	}

	// 只能删除非运行状态的任务（运行中的任务不能删除）
	if task.Status == models.VideoStatusRecording || task.Status == models.VideoStatusConnecting {
		return nil, apperrors.ErrTaskInProgress
	}

	// 检查权限（管理员可以删除任何任务）
	if !isAdmin && task.CreatedBy != userID {
		return nil, apperrors.ErrForbidden
	}

	// Snapshot before delete for audit OldData capture
	oldTask := task

	// 删除前先解锁所有关联的输入配置（防止锁遗留）
	// PERF-001: 改 Pluck + IN 批量更新，消除 N+1（原每配置一次 UPDATE）。
	var configIDs []uint
	if err := s.db.WithContext(ctx).Model(&models.TaskInputConfig{}).Where("task_id = ?", id).Pluck("input_config_id", &configIDs).Error; err == nil {
		if len(configIDs) > 0 {
			s.db.WithContext(ctx).Model(&models.InputConfig{}).Where("id IN ?", configIDs).Updates(map[string]interface{}{
				"is_locked": false,
				"locked_by": nil,
			})
		}
		s.logger.Info("删除任务时解锁终端",
			zap.Uint("task_id", id),
			zap.Int("unlocked_configs", len(configIDs)),
		)
	}

	result := s.db.WithContext(ctx).Delete(&models.VideoRecordingTask{}, id)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, apperrors.ErrTaskNotFound
	}

	s.logger.Info("录制任务已删除",
		zap.Uint("task_id", id),
		zap.Uint("deleted_by", userID),
		zap.String("task_status", string(task.Status)),
	)

	return &oldTask, nil
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
func (s *VideoRecordingTaskService) BatchDeleteTasks(ctx context.Context, ids []uint, userID uint, isAdmin bool) ([]models.VideoRecordingTask, *BatchDeleteTasksResult, error) {
	if len(ids) == 0 {
		return nil, nil, apperrors.ErrInvalidInput
	}

	// Snapshot requested tasks BEFORE any deletion or filtering — for audit OldData capture
	var oldTasks []models.VideoRecordingTask
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Limit(5000).Find(&oldTasks).Error; err != nil {
		return nil, nil, err
	}

	if len(oldTasks) == 0 {
		return nil, nil, apperrors.ErrTaskNotFound
	}

	result := &BatchDeleteTasksResult{
		DeletedIDs:  make([]uint, 0),
		FailedIDs:   make([]uint, 0),
		FailedTasks: make([]string, 0),
	}

	for _, task := range oldTasks {
		if canDelete, reason := s.canDeleteTask(task, userID, isAdmin); canDelete {
			result.DeletedIDs = append(result.DeletedIDs, task.ID)
		} else {
			result.FailedIDs = append(result.FailedIDs, task.ID)
			result.FailedTasks = append(result.FailedTasks, fmt.Sprintf("%s（%s）", task.Name, reason))
		}
	}

	if len(result.DeletedIDs) == 0 {
		return oldTasks, result, fmt.Errorf("没有可删除的任务: %w", apperrors.ErrInvalidInput)
	}

	// 删除前先解锁所有待删除任务的输入配置（防止锁遗留）
	// PERF-001: 单次 Pluck 跨所有待删除任务 + 单次批量 UPDATE，消除双层 N+1。
	var batchConfigIDs []uint
	if err := s.db.WithContext(ctx).Model(&models.TaskInputConfig{}).Where("task_id IN ?", result.DeletedIDs).Pluck("input_config_id", &batchConfigIDs).Error; err == nil {
		if len(batchConfigIDs) > 0 {
			s.db.WithContext(ctx).Model(&models.InputConfig{}).Where("id IN ?", batchConfigIDs).Updates(map[string]interface{}{
				"is_locked": false,
				"locked_by": nil,
			})
		}
	}
	s.logger.Info("批量删除任务时解锁终端",
		zap.Int("count", len(result.DeletedIDs)),
	)

	// 执行删除
	dbResult := s.db.WithContext(ctx).Delete(&models.VideoRecordingTask{}, result.DeletedIDs)
	if dbResult.Error != nil {
		return nil, nil, dbResult.Error
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

	return oldTasks, result, nil
}

// StartTask 手动启动任务
func (s *VideoRecordingTaskService) StartTask(ctx context.Context, id, userID uint) (*models.VideoRecordingTask, *models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, nil, apperrors.ErrTaskNotFound
	}

	// 检查权限
	if task.CreatedBy != userID {
		return nil, nil, apperrors.ErrForbidden
	}

	// 检查状态
	if task.Status != models.VideoStatusPending {
		return nil, nil, apperrors.ErrInvalidInput
	}

	// Snapshot pre-start state for audit OldData capture
	oldTask := task

	// 触发任务执行
	if s.scheduler != nil {
		if err := s.scheduler.ExecuteTask(id); err != nil {
			return nil, nil, fmt.Errorf("触发任务执行失败: %w", err)
		}
	} else {
		return nil, nil, fmt.Errorf("调度器未初始化: %w", apperrors.ErrInternal)
	}

	// Reload post-dispatch state for NewData (asynchronous scheduler may have mutated status).
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, nil, err
	}

	s.logger.Info("录制任务已手动启动",
		zap.Uint("task_id", id),
		zap.Uint("started_by", userID),
	)

	return &oldTask, &task, nil
}

// StopTask 手动停止任务
func (s *VideoRecordingTaskService) StopTask(ctx context.Context, id, userID uint, hasSharedViewer bool) (*models.VideoRecordingTask, *models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, nil, apperrors.ErrTaskNotFound
	}

	// 检查权限
	if !hasSharedViewer && task.CreatedBy != userID {
		return nil, nil, apperrors.ErrForbidden
	}

	// 检查状态
	if task.Status != models.VideoStatusRecording && task.Status != models.VideoStatusConnecting {
		return nil, nil, apperrors.ErrInvalidInput
	}

	// Snapshot pre-stop state for audit OldData capture (before CancelTaskExecution)
	oldTask := task

	// 记录原始状态，用于后续判断
	wasRecording := task.Status == models.VideoStatusRecording

	// 取消任务执行（这会停止录制并创建文件记录、提交转换任务）
	if s.scheduler != nil {
		if err := s.scheduler.CancelTaskExecution(id); err != nil {
			s.logger.Warn("取消任务执行失败", zap.Error(err), response.SentinelField(err))
			// 继续执行状态更新
		}
	}

	// 重新加载任务以获取最新状态（因为 CancelTaskExecution 可能创建了 MKV 文件）
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, nil, err
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
			zap.Uint("canceled_by", userID),
		)
	}

	if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
		return nil, nil, err
	}

	return &oldTask, &task, nil
}

// CancelTask 取消任务
func (s *VideoRecordingTaskService) CancelTask(ctx context.Context, id, userID uint, hasSharedViewer bool) error {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return apperrors.ErrTaskNotFound
	}

	// 检查权限
	if !hasSharedViewer && task.CreatedBy != userID {
		return apperrors.ErrForbidden
	}

	// 检查状态
	if task.Status != models.VideoStatusPending && task.Status != models.VideoStatusConnecting {
		return apperrors.ErrInvalidInput
	}

	// 从调度器移除
	if s.scheduler != nil {
		if err := s.scheduler.RemoveTask(id); err != nil {
			s.logger.Warn("从调度器移除任务失败", zap.Error(err), response.SentinelField(err))
		}
	}

	// 更新状态
	task.Status = models.VideoStatusCancelled
	if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
		return err
	}

	s.logger.Info("录制任务已取消",
		zap.Uint("task_id", id),
		zap.Uint("canceled_by", userID),
	)

	return nil
}

// RetryTask 重试失败任务
func (s *VideoRecordingTaskService) RetryTask(ctx context.Context, id, userID uint, hasSharedViewer bool) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&task, id).Error; err != nil {
		return nil, apperrors.ErrTaskNotFound
	}

	// 检查权限
	if !hasSharedViewer && task.CreatedBy != userID {
		return nil, apperrors.ErrForbidden
	}

	// 检查状态
	if task.Status != models.VideoStatusFailed {
		return nil, apperrors.ErrInvalidInput
	}

	// 重置状态
	task.Status = models.VideoStatusPending
	task.ErrorMsg = ""
	task.RecordingFile = ""
	task.RecordingDuration = 0

	// 重新计算触发时间 (当前时间 + 1分钟)
	// BUG-001 修复：必须先捕获原时长，再改 StartTime，否则 EndTime.Sub(StartTime) 读取的
	// 是已被改写的 StartTime，得到 oldEnd-newStart（负数或离谱值），静默损坏任务时长。
	newTriggerTime := time.Now().Add(1 * time.Minute)
	duration := task.EndTime.Sub(task.StartTime)
	task.StartTime = newTriggerTime.Add(time.Duration(task.PreJoinMinutes) * time.Minute)
	task.EndTime = task.StartTime.Add(duration)

	if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
		return nil, err
	}

	// 重新调度任务
	if s.scheduler != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Error("RetryTask scheduler sync goroutine panicked",
						zap.Any("recover", r), zap.Stack("stack"))
				}
			}()
			if err := s.scheduler.SyncPendingTasks(); err != nil {
				s.logger.Error("重新调度任务失败",
					zap.Uint("task_id", id),
					zap.Error(err),
					response.SentinelField(err),
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
func (s *VideoRecordingTaskService) GetTasksByStatus(ctx context.Context, status models.VideoRecordingTaskStatus) ([]models.VideoRecordingTask, error) {
	var tasks []models.VideoRecordingTask
	// PERF-002: 列表查询加 Limit 上限，防止数据增长后全表扫描。
	if err := s.db.WithContext(ctx).Where("status = ?", status).Limit(1000).Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetPendingTasks 获取待执行的任务
func (s *VideoRecordingTaskService) GetPendingTasks(ctx context.Context) ([]models.VideoRecordingTask, error) {
	return s.GetTasksByStatus(ctx, models.VideoStatusPending)
}

// UpdateTaskStatus 更新任务状态
func (s *VideoRecordingTaskService) UpdateTaskStatus(ctx context.Context, id uint, status models.VideoRecordingTaskStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	result := s.db.WithContext(ctx).Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrTaskNotFound
	}

	return nil
}

// UpdateRecordingInfo 更新录制信息
func (s *VideoRecordingTaskService) UpdateRecordingInfo(ctx context.Context, id uint, filePath string, duration int) error {
	updates := map[string]interface{}{
		"recording_file":     filePath,
		"recording_duration": duration,
	}

	result := s.db.WithContext(ctx).Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrTaskNotFound
	}

	return nil
}

// UpdateRecordingPaths 更新录制文件路径
func (s *VideoRecordingTaskService) UpdateRecordingPaths(ctx context.Context, id uint, mkvPath, hlsPath string) error {
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

	result := s.db.WithContext(ctx).Model(&models.VideoRecordingTask{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return apperrors.ErrTaskNotFound
	}
	return nil
}

// GetInputConfig 获取输入配置（供调度器使用）
// GetInputConfig 获取输入配置
//
// Phase 19 D2：与原 taskServiceAdapter.GetInputConfig 行为对齐——
// 当 encryptor!=nil 时解密 Password/StreamPassword 后返回明文（Phase 18 契约；
// scheduler / recorder 等调用方期望明文）。encryptor 为 nil 时保持原密文（向后兼容）。
//
// 解密失败 → 阻断调用方（不静默跳过），错误信息携带 ID 便于排查。
func (s *VideoRecordingTaskService) GetInputConfig(ctx context.Context, id uint) (*models.InputConfig, error) {
	var config models.InputConfig
	if err := s.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, err
	}
	if s.encryptor != nil {
		if config.Password != "" {
			pt, err := s.encryptor.Decrypt(config.Password)
			if err != nil {
				return nil, fmt.Errorf("VideoRecordingTaskService.GetInputConfig(id=%d) password 解密失败: %w", id, err)
			}
			config.Password = pt
		}
		if config.StreamPassword != "" {
			pt, err := s.encryptor.Decrypt(config.StreamPassword)
			if err != nil {
				return nil, fmt.Errorf("VideoRecordingTaskService.GetInputConfig(id=%d) stream_password 解密失败: %w", id, err)
			}
			config.StreamPassword = pt
		}
	}
	return &config, nil
}

// GetTask 是 scheduler.TaskServiceInterface 接口适配方法 —— 与
// VideoRecordingTaskService.GetTaskByID 行为等价但预加载 InputConfig/
// TaskInputConfigs（与原 taskServiceAdapter.GetTask 行为一致）。
//
// 调度器内部所有调用站点期望"任务+关联配置"在一行内拿到，因此本方法不是 GetTaskByID
// 的简单重命名，而是带 Preload 的语义化封装。该方法满足 scheduler.TaskServiceInterface，
// 让 cmd/server 不再需要 adapter。
func (s *VideoRecordingTaskService) GetTask(ctx context.Context, id uint) (*models.VideoRecordingTask, error) {
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).Preload("InputConfig").Preload("TaskInputConfigs").First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetDB 获取数据库连接（供调度器使用）
func (s *VideoRecordingTaskService) GetDB() *gorm.DB {
	return s.db
}

// ClearStuckTasks 清理卡住的任务
// 将 converting 状态超过指定时间的任务标记为失败，并释放终端锁
func (s *VideoRecordingTaskService) ClearStuckTasks(ctx context.Context, timeoutMinutes int) (*ClearStuckTasksResult, error) {
	if timeoutMinutes <= 0 {
		timeoutMinutes = 30 // 默认30分钟
	}

	timeout := time.Now().Add(-time.Duration(timeoutMinutes) * time.Minute)

	s.logger.Info("开始清理卡住的任务",
		zap.Int("timeout_minutes", timeoutMinutes),
		zap.Time("before_time", timeout),
	)

	// 查找所有 converting 状态且超过超时时间的任务
	var stuckTasks []models.VideoRecordingTask
	// PERF-002: 清理路径加 Limit 上限（bounded cleanup）。
	if err := s.db.WithContext(ctx).Where("status = ? AND updated_at < ?", models.VideoStatusConverting, timeout).Limit(5000).Find(&stuckTasks).Error; err != nil {
		return nil, fmt.Errorf("查询卡住任务失败: %w", err)
	}

	s.logger.Info("发现卡住的任务", zap.Int("count", len(stuckTasks)))

	result := &ClearStuckTasksResult{
		ClearedTaskIDs:    make([]uint, 0),
		UnlockedConfigIDs: make([]uint, 0),
	}

	for _, task := range stuckTasks {
		s.logger.Info("清理卡住的任务",
			zap.Uint("task_id", task.ID),
			zap.String("task_name", task.Name),
			zap.Time("updated_at", task.UpdatedAt),
		)

		// 更新任务状态为失败
		task.Status = models.VideoStatusFailed
		task.ErrorMsg = fmt.Sprintf("任务卡在转换中状态超过 %d 分钟，已自动清理", timeoutMinutes)
		task.ConversionStatus = models.ConversionStatusFailed
		task.ConversionErrorMsg = "任务被清理"

		if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
			s.logger.Error("更新任务状态失败",
				zap.Uint("task_id", task.ID),
				zap.Error(err),
				response.SentinelField(err),
			)
			continue
		}
		result.ClearedTaskIDs = append(result.ClearedTaskIDs, task.ID)

		// 释放所有关联的输入配置锁（PERF-001: Pluck + IN 批量更新，消除每配置一次 UPDATE）
		var configIDs []uint
		if err := s.db.WithContext(ctx).Model(&models.TaskInputConfig{}).Where("task_id = ?", task.ID).Pluck("input_config_id", &configIDs).Error; err == nil {
			if len(configIDs) > 0 {
				if err := s.db.WithContext(ctx).Model(&models.InputConfig{}).Where("id IN ?", configIDs).Updates(map[string]interface{}{
					"is_locked": false,
					"locked_by": nil,
					"locked_at": nil,
				}).Error; err == nil {
					result.UnlockedConfigIDs = append(result.UnlockedConfigIDs, configIDs...)
				}
			}
			s.logger.Info("已释放终端锁",
				zap.Uint("task_id", task.ID),
				zap.Int("unlocked_count", len(configIDs)),
			)
		}
	}

	result.TotalCleared = len(result.ClearedTaskIDs)
	result.TotalUnlocked = len(result.UnlockedConfigIDs)

	s.logger.Info("清理卡住任务完成",
		zap.Int("total_cleared", result.TotalCleared),
		zap.Int("total_unlocked", result.TotalUnlocked),
	)

	return result, nil
}

// ClearStuckTasksResult 清理卡住任务的结果
type ClearStuckTasksResult struct {
	ClearedTaskIDs    []uint `json:"cleared_task_ids"`    // 被清理的任务ID列表
	UnlockedConfigIDs []uint `json:"unlocked_config_ids"` // 被解锁的配置ID列表
	TotalCleared      int    `json:"total_cleared"`       // 总共清理的任务数
	TotalUnlocked     int    `json:"total_unlocked"`      // 总共解锁的配置数
}
