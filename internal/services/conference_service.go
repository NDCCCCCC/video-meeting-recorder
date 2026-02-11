package services

import (
	"errors"
	"time"

	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ConferenceRecordService 会议记录服务
type ConferenceRecordService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewConferenceRecordService 创建会议记录服务
func NewConferenceRecordService(db *gorm.DB, logger *zap.Logger) *ConferenceRecordService {
	return &ConferenceRecordService{
		db:     db,
		logger: logger,
	}
}

// ListConferencesRequest 会议列表请求
type ListConferencesRequest struct {
	Page             int                        `form:"page"`
	PageSize         int                        `form:"page_size" binding:"max=100"`
	Keyword          string                     `form:"keyword"`
	Status           models.ConferenceStatus  `form:"status"`
	ConferenceNumber string                     `form:"conference_number"`
	StartDate        string                     `form:"start_date"`
	EndDate          string                     `form:"end_date"`
}

// ListConferencesResponse 会议列表响应
type ListConferencesResponse struct {
	Total int64                    `json:"total"`
	Items []models.ConferenceRecord `json:"items"`
}

// CreateConferenceRequest 创建会议请求
type CreateConferenceRequest struct {
	ConferenceNumber string `json:"conference_number" binding:"required,max=50"`
	Title             string `json:"title" binding:"required,max=200"`
	StartTime         string `json:"start_time" binding:"required"` // RFC3339
	EndTime           string `json:"end_time" binding:"required"`   // RFC3339
	Description       string `json:"description" binding:"max=1000"`
	HuaweiConfigID   *uint  `json:"huawei_config_id"`
}

// UpdateConferenceRequest 更新会议请求
type UpdateConferenceRequest struct {
	Title       *string `json:"title" binding:"omitempty,max=200"`
	EndTime     *string `json:"end_time" binding:"omitempty"` // RFC3339
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Status       *models.ConferenceStatus `json:"status"`
}

// ListConferences 获取会议列表
func (s *ConferenceRecordService) ListConferences(req *ListConferencesRequest) (*ListConferencesResponse, error) {
	var conferences []models.ConferenceRecord
	var total int64

	query := s.db.Model(&models.ConferenceRecord{}).Preload("HuaweiConfig").Preload("VideoFiles")

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("title LIKE ? OR description LIKE ? OR conference_number LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 会议号筛选
	if req.ConferenceNumber != "" {
		query = query.Where("conference_number = ?", req.ConferenceNumber)
	}

	// 日期范围筛选
	if req.StartDate != "" {
		if startTime, err := parseDate(req.StartDate); err == nil {
			query = query.Where("start_time >= ?", startTime)
		}
	}
	if req.EndDate != "" {
		if endTime, err := parseDate(req.EndDate); err == nil {
			// 包含整天
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("start_time < ?", endTime)
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
		Order("start_time DESC").
		Find(&conferences).Error; err != nil {
		return nil, err
	}

	return &ListConferencesResponse{
		Total: total,
		Items: conferences,
	}, nil
}

// GetConferenceByID 根据ID获取会议
func (s *ConferenceRecordService) GetConferenceByID(id uint) (*models.ConferenceRecord, error) {
	var conference models.ConferenceRecord
	if err := s.db.Preload("HuaweiConfig").Preload("VideoFiles").Preload("VideoRecordingTask").First(&conference, id).Error; err != nil {
		return nil, err
	}
	return &conference, nil
}

// CreateConference 创建会议
func (s *ConferenceRecordService) CreateConference(req *CreateConferenceRequest) (*models.ConferenceRecord, error) {
	// 检查会议号是否已存在
	var existing models.ConferenceRecord
	if err := s.db.Where("conference_number = ?", req.ConferenceNumber).First(&existing).Error; err == nil {
		return nil, errors.New("会议号已存在")
	}

	// 解析时间
	startTime, err := parseTime(req.StartTime)
	if err != nil {
		return nil, errors.New("开始时间格式错误")
	}
	endTime, err := parseTime(req.EndTime)
	if err != nil {
		return nil, errors.New("结束时间格式错误")
	}

	// 验证时间
	if startTime.After(endTime) {
		return nil, errors.New("开始时间不能晚于结束时间")
	}

	// 验证华为配置
	if req.HuaweiConfigID != nil {
		var config models.HuaweiConfig
		if err := s.db.First(&config, *req.HuaweiConfigID).Error; err != nil {
			return nil, errors.New("华为配置不存在")
		}
	}

	conference := &models.ConferenceRecord{
		ConferenceNumber: req.ConferenceNumber,
		Title:             req.Title,
		StartTime:         startTime,
		EndTime:           &endTime,
		Status:            models.ConferenceStatusNotStarted,
		Description:       req.Description,
		HuaweiConfigID:   req.HuaweiConfigID,
	}

	if err := s.db.Create(conference).Error; err != nil {
		return nil, err
	}

	// 重新加载关联数据
	s.db.Preload("HuaweiConfig").First(conference, conference.ID)

	s.logger.Info("会议已创建",
		zap.Uint("conference_id", conference.ID),
		zap.String("conference_number", conference.ConferenceNumber),
	)

	return conference, nil
}

// UpdateConference 更新会议
func (s *ConferenceRecordService) UpdateConference(id uint, req *UpdateConferenceRequest) (*models.ConferenceRecord, error) {
	var conference models.ConferenceRecord
	if err := s.db.Preload("HuaweiConfig").First(&conference, id).Error; err != nil {
		return nil, errors.New("会议不存在")
	}

	updates := make(map[string]interface{})

	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		// 验证状态转换
		if !canTransitionStatus(conference.Status, *req.Status) {
			return nil, errors.New("无效的状态转换")
		}
		updates["status"] = *req.Status
	}
	if req.EndTime != nil {
		endTime, err := parseTime(*req.EndTime)
		if err != nil {
			return nil, errors.New("结束时间格式错误")
		}
		if conference.StartTime.After(endTime) {
			return nil, errors.New("结束时间不能早于开始时间")
		}
		updates["end_time"] = &endTime
	}

	if err := s.db.Model(&conference).Updates(updates).Error; err != nil {
		return nil, err
	}

	// 重新加载数据
	s.db.Preload("HuaweiConfig").Preload("VideoFiles").First(&conference, id)

	s.logger.Info("会议已更新", zap.Uint("conference_id", id))

	return &conference, nil
}

// DeleteConference 删除会议
func (s *ConferenceRecordService) DeleteConference(id uint) error {
	var conference models.ConferenceRecord
	if err := s.db.First(&conference, id).Error; err != nil {
		return errors.New("会议不存在")
	}

	// 检查是否有录制任务关联
	var taskCount int64
	if err := s.db.Model(&models.VideoRecordingTask{}).Where("conference_record_id = ?", id).Count(&taskCount).Error; err != nil {
		return err
	}
	if taskCount > 0 {
		return errors.New("会议有关联的录制任务，无法删除")
	}

	// 检查是否有视频文件
	var fileCount int64
	if err := s.db.Model(&models.VideoFile{}).Where("conference_record_id = ?", id).Count(&fileCount).Error; err != nil {
		return err
	}
	if fileCount > 0 {
		return errors.New("会议有关联的视频文件，无法删除")
	}

	result := s.db.Delete(&models.ConferenceRecord{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("会议不存在")
	}

	s.logger.Info("会议已删除", zap.Uint("conference_id", id))

	return nil
}

// BatchDeleteConferencesRequest 批量删除会议请求
type BatchDeleteConferencesRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

// BatchDeleteConferences 批量删除会议
func (s *ConferenceRecordService) BatchDeleteConferences(req *BatchDeleteConferencesRequest) ([]uint, []error) {
	var deletedIDs []uint
	var errors []error

	for _, id := range req.IDs {
		if err := s.DeleteConference(id); err != nil {
			s.logger.Warn("Failed to delete conference",
				zap.Uint("conference_id", id),
				zap.Error(err))
			errors = append(errors, err)
		} else {
			deletedIDs = append(deletedIDs, id)
		}
	}

	return deletedIDs, errors
}

// canTransitionStatus 检查状态转换是否合法
func canTransitionStatus(current, new models.ConferenceStatus) bool {
	validTransitions := map[models.ConferenceStatus][]models.ConferenceStatus{
		models.ConferenceStatusNotStarted: {
			models.ConferenceStatusInProgress,
			models.ConferenceStatusFailed,
		},
		models.ConferenceStatusInProgress: {
			models.ConferenceStatusCompleted,
			models.ConferenceStatusFailed,
		},
		models.ConferenceStatusCompleted:  {},
		models.ConferenceStatusFailed:     {models.ConferenceStatusNotStarted},
	}

	allowed, ok := validTransitions[current]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == new {
			return true
		}
	}
	return false
}

// GetConferencesByStatus 根据状态获取会议列表
func (s *ConferenceRecordService) GetConferencesByStatus(status models.ConferenceStatus) ([]models.ConferenceRecord, error) {
	var conferences []models.ConferenceRecord
	if err := s.db.Where("status = ?", status).Find(&conferences).Error; err != nil {
		return nil, err
	}
	return conferences, nil
}

// parseTime 解析RFC3339时间格式
func parseTime(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}

// parseDate 解析日期格式 (YYYY-MM-DD)
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
