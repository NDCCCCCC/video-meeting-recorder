package services

import (
	"context"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DashboardService 仪表板服务
type DashboardService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewDashboardService 创建仪表板服务
func NewDashboardService(db *gorm.DB, logger *zap.Logger) *DashboardService {
	return &DashboardService{
		db:     db,
		logger: logger,
	}
}

// DashboardStatsResponse 仪表板统计响应
type DashboardStatsResponse struct {
	TaskStats   TaskStats   `json:"task_stats"`
	FileStats   FileStats   `json:"file_stats"`
	SystemStats SystemStats `json:"system_stats"`
}

// TaskStats 任务统计
type TaskStats struct {
	Total      int64   `json:"total"`
	InProgress int64   `json:"in_progress"`
	Success    int64   `json:"success"`
	Fail       int64   `json:"fail"`
	AvgTime    float64 `json:"avg_time"` // in seconds
}

// FileStats 文件统计
type FileStats struct {
	TotalVideos int64   `json:"total_videos"`
	StorageMB   float64 `json:"storage_mb"`
	Transcripts int64   `json:"transcripts"`
	Ppts        int64   `json:"ppts"`
}

// SystemStats 系统统计
type SystemStats struct {
	DiskUsagePercent   float64 `json:"disk_usage_percent"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	ErrorCount         int64   `json:"error_count"`
	ApiCalls           int64   `json:"api_calls"`
}

// GetDashboardStats 获取仪表板统计数据
func (s *DashboardService) GetDashboardStats(ctx context.Context) (*DashboardStatsResponse, error) {
	var stats DashboardStatsResponse

	// 获取任务统计
	taskStats, err := s.getTaskStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get task stats", zap.Error(err))
		return nil, err
	}
	stats.TaskStats = *taskStats

	// 获取文件统计
	fileStats, err := s.getFileStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get file stats", zap.Error(err))
		return nil, err
	}
	stats.FileStats = *fileStats

	// 获取系统统计
	systemStats, err := s.getSystemStats(ctx)
	if err != nil {
		s.logger.Error("Failed to get system stats", zap.Error(err))
		return nil, err
	}
	stats.SystemStats = *systemStats

	return &stats, nil
}

// getTaskStats 获取任务统计
func (s *DashboardService) getTaskStats(ctx context.Context) (*TaskStats, error) {
	var stats TaskStats

	// 总任务数
	if err := s.db.Model(&models.VideoRecordingTask{}).Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// 进行中的任务数（connecting, recording, converting）
	if err := s.db.Model(&models.VideoRecordingTask{}).
		Where("status IN ?", []models.VideoRecordingTaskStatus{
			models.VideoStatusConnecting,
			models.VideoStatusRecording,
			models.VideoStatusConverting,
		}).
		Count(&stats.InProgress).Error; err != nil {
		return nil, err
	}

	// 成功的任务数（completed）
	if err := s.db.Model(&models.VideoRecordingTask{}).
		Where("status = ?", models.VideoStatusCompleted).
		Count(&stats.Success).Error; err != nil {
		return nil, err
	}

	// 失败的任务数
	if err := s.db.Model(&models.VideoRecordingTask{}).
		Where("status = ?", models.VideoStatusFailed).
		Count(&stats.Fail).Error; err != nil {
		return nil, err
	}

	// 平均处理时间（仅计算已完成的任务）
	type Result struct {
		AvgTime float64
	}
	var result Result
	if err := s.db.Model(&models.VideoRecordingTask{}).
		Select("AVG(julianday(end_time) - julianday(start_time)) * 86400 as avg_time").
		Where("status = ? AND end_time IS NOT NULL AND start_time IS NOT NULL", models.VideoStatusCompleted).
		Scan(&result).Error; err != nil {
		// 如果查询失败，设置为0
		stats.AvgTime = 0
	} else {
		stats.AvgTime = result.AvgTime
	}

	return &stats, nil
}

// getFileStats 获取文件统计
func (s *DashboardService) getFileStats(ctx context.Context) (*FileStats, error) {
	var stats FileStats

	// 视频文件总数
	if err := s.db.Model(&models.VideoFile{}).Count(&stats.TotalVideos).Error; err != nil {
		return nil, err
	}

	// 存储空间使用（MB）
	type StorageResult struct {
		TotalBytes int64
	}
	var storageResult StorageResult
	if err := s.db.Model(&models.VideoFile{}).
		Select("COALESCE(SUM(file_size), 0) as total_bytes").
		Scan(&storageResult).Error; err != nil {
		return nil, err
	}
	stats.StorageMB = float64(storageResult.TotalBytes) / 1024 / 1024

	// 转录文件数（从 transcription_tasks 表统计）
	if err := s.db.Model(&models.TranscriptionTask{}).
		Where("status = ?", "completed").
		Count(&stats.Transcripts).Error; err != nil {
		// 如果表不存在或查询失败，设置为0
		stats.Transcripts = 0
	}

	// PPT文件数
	if err := s.db.Model(&models.PPTFile{}).Count(&stats.Ppts).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// getSystemStats 获取系统统计
func (s *DashboardService) getSystemStats(ctx context.Context) (*SystemStats, error) {
	var stats SystemStats

	// 错误数量（从审计日志统计最近24小时的错误）
	twentyFourHoursAgo := time.Now().UTC().Add(-24 * time.Hour)
	if err := s.db.Model(&models.AuditLog{}).
		Where("status = ? AND created_at >= ?", models.StatusFailure, twentyFourHoursAgo).
		Count(&stats.ErrorCount).Error; err != nil {
		return nil, err
	}

	// API调用数量（从审计日志统计最近24小时的query操作）
	if err := s.db.Model(&models.AuditLog{}).
		Where("action = ? AND created_at >= ?", models.ActionQuery, twentyFourHoursAgo).
		Count(&stats.ApiCalls).Error; err != nil {
		return nil, err
	}

	// 磁盘使用率（Windows 用 SystemDrive，其他平台用根 /）
	diskPath := "/"
	if runtime.GOOS == "windows" {
		diskPath = os.Getenv("SystemDrive") + string(os.PathSeparator)
	}
	if usage, err := disk.Usage(diskPath); err != nil {
		s.logger.Warn("获取磁盘使用率失败", zap.String("path", diskPath), zap.Error(err))
	} else if usage.Total > 0 {
		stats.DiskUsagePercent = roundPercent(usage.UsedPercent)
	}

	// 内存使用率
	if vm, err := mem.VirtualMemory(); err != nil {
		s.logger.Warn("获取内存使用率失败", zap.Error(err))
	} else {
		stats.MemoryUsagePercent = roundPercent(vm.UsedPercent)
	}

	return &stats, nil
}

// roundPercent 把百分比保留 1 位小数，避免前端展示过长。
func roundPercent(p float64) float64 {
	return math.Round(p*10) / 10
}
