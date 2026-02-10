package services

import (
	"errors"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// VideoFileService 视频文件服务
type VideoFileService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewVideoFileService 创建视频文件服务
func NewVideoFileService(db *gorm.DB, logger *zap.Logger) *VideoFileService {
	return &VideoFileService{
		db:     db,
		logger: logger,
	}
}

// ListFilesRequest 文件列表请求
type ListFilesRequest struct {
	Page                int     `form:"page"`
	PageSize            int     `form:"page_size" binding:"max=100"`
	Keyword             string  `form:"keyword"`
	ConferenceRecordID  *uint   `form:"conference_record_id"`
	Status              string  `form:"status"`
	Format              string  `form:"format"`
	StartDate           string  `form:"start_date"`
	EndDate             string  `form:"end_date"`
}

// ListFilesResponse 文件列表响应
type ListFilesResponse struct {
	Total        int64              `json:"total"`
	Items        []models.VideoFile `json:"items"`
	TotalSize    int64              `json:"total_size"`
	TotalSizeGB  float64            `json:"total_size_gb"`
}

// ListFiles 获取文件列表
func (s *VideoFileService) ListFiles(req *ListFilesRequest) (*ListFilesResponse, error) {
	var files []models.VideoFile
	var total int64

	query := s.db.Model(&models.VideoFile{}).Preload("ConferenceRecord")

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("file_name LIKE ? OR file_path LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 会议筛选
	if req.ConferenceRecordID != nil {
		query = query.Where("conference_record_id = ?", *req.ConferenceRecordID)
	}

	// 状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 格式筛选
	if req.Format != "" {
		query = query.Where("format = ?", req.Format)
	}

	// 日期范围筛选
	if req.StartDate != "" {
		if startTime, err := parseDate(req.StartDate); err == nil {
			query = query.Where("created_at >= ?", startTime)
		}
	}
	if req.EndDate != "" {
		if endTime, err := parseDate(req.EndDate); err == nil {
			endTime = endTime.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endTime)
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
		Find(&files).Error; err != nil {
		return nil, err
	}

	// 计算总大小
	var totalSize int64
	for _, file := range files {
		totalSize += file.FileSize
	}

	return &ListFilesResponse{
		Total:       total,
		Items:       files,
		TotalSize:   totalSize,
		TotalSizeGB: float64(totalSize) / (1024 * 1024 * 1024),
	}, nil
}

// GetFileByID 根据ID获取文件
func (s *VideoFileService) GetFileByID(id uint) (*models.VideoFile, error) {
	var file models.VideoFile
	if err := s.db.Preload("ConferenceRecord").First(&file, id).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

// DeleteFile 删除文件
func (s *VideoFileService) DeleteFile(id uint) error {
	var file models.VideoFile
	if err := s.db.First(&file, id).Error; err != nil {
		return errors.New("文件不存在")
	}

	// 检查文件状态
	if file.Status == models.FileStatusProcessing {
		return errors.New("文件正在处理中，无法删除")
	}

	// 删除物理文件
	if file.Exists() {
		if err := file.Delete(); err != nil {
			s.logger.Warn("Failed to delete physical file",
				zap.Uint("file_id", id),
				zap.Error(err),
			)
		}
	}

	// 删除数据库记录
	result := s.db.Delete(&models.VideoFile{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("文件不存在")
	}

	s.logger.Info("Video file deleted", zap.Uint("file_id", id))

	return nil
}

// GetFilesByConferenceID 根据会议ID获取文件列表
func (s *VideoFileService) GetFilesByConferenceID(conferenceID uint) ([]models.VideoFile, error) {
	var files []models.VideoFile
	if err := s.db.Where("conference_record_id = ?", conferenceID).
		Order("created_at ASC").
		Find(&files).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// UpdateFileStatus 更新文件状态
func (s *VideoFileService) UpdateFileStatus(id uint, status string) error {
	return s.db.Model(&models.VideoFile{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// GetFileStats 获取文件统计信息
func (s *VideoFileService) GetFileStats() (map[string]interface{}, error) {
	var total int64
	var totalSize int64

	if err := s.db.Model(&models.VideoFile{}).Count(&total).Error; err != nil {
		return nil, err
	}

	if err := s.db.Model(&models.VideoFile{}).
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&totalSize).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":        total,
		"total_size":   totalSize,
		"total_size_gb": float64(totalSize) / (1024 * 1024 * 1024),
	}, nil
}
