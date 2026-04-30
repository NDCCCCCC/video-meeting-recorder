package services

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 常量定义
const (
	DefaultFormat     = "mkv"
	FFProbeTimeout    = 30 * time.Second
	DefaultBitrate    = 0
	DefaultCodec      = "h264"
	DefaultResolution = "1920x1080"
)

// VideoFileService 视频文件服务
type VideoFileService struct {
	db             *gorm.DB
	logger         *zap.Logger
	recordingsPath string
	hlsPath        string // HLS 预览文件存储路径
	ffprobePath    string // ffprobe 可执行文件路径
}

// NewVideoFileService 创建视频文件服务
func NewVideoFileService(db *gorm.DB, logger *zap.Logger, recordingsPath string, ffprobePath string) *VideoFileService {
	if ffprobePath == "" {
		ffprobePath = "./bin/ffprobe" // 默认使用项目内置的 ffprobe
	}
	return &VideoFileService{
		db:             db,
		logger:         logger,
		recordingsPath: recordingsPath,
		hlsPath:        "", // 需要通过 SetHLSPath 设置
		ffprobePath:    ffprobePath,
	}
}

// SetHLSPath 设置 HLS 路径
func (s *VideoFileService) SetHLSPath(hlsPath string) {
	s.hlsPath = hlsPath
}

// ListFilesRequest 文件列表请求
type ListFilesRequest struct {
	Page           int          `form:"page"`
	PageSize       int          `form:"page_size" binding:"max=100"`
	Keyword        string       `form:"keyword"`
	TaskID         *uint        `form:"task_id"`
	Status         string       `form:"status"`
	Format         string       `form:"format"`
	SourceType     string       `form:"source_type"`
	StartDate      string       `form:"start_date"`
	EndDate        string       `form:"end_date"`
	UserID         uint         `form:"-"`
	IsAdmin        bool         `form:"-"`
	ApplyDataScope bool         `form:"-"`
	User           *models.User `form:"-"` // User object with Roles preloaded for visibility control (D-11, D-12)
	RoleIDs        []uint       `form:"-"` // Role IDs from token claims for shared_viewer check (D-02)
}

// ListFilesResponse 文件列表响应
type ListFilesResponse struct {
	Total       int64              `json:"total"`
	Items       []models.VideoFile `json:"items"`
	TotalSize   int64              `json:"total_size"`
	TotalSizeGB float64            `json:"total_size_gb"`
}

// ScanResult 扫描结果
type ScanResult struct {
	Scanned int      `json:"scanned"`
	Created int      `json:"created"`
	Updated int      `json:"updated"` // 更新的文件数量
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
}

// 内部视频元数据
type videoMetadata struct {
	Format     string
	Duration   float64
	Resolution string
	Bitrate    int
	Codec      string
}

// ListFiles 获取文件列表
func (s *VideoFileService) ListFiles(req *ListFilesRequest) (*ListFilesResponse, error) {
	query := s.db.Model(&models.VideoFile{}).Preload("Task")

	// 应用筛选条件
	s.applyFilters(query, req)

	// 统计总数和总大小
	stats, err := s.getStats(query)
	if err != nil {
		return nil, err
	}

	// 分页查询
	var files []models.VideoFile
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		return nil, err
	}

	// 批量检查 PPT 状态（优化方案：使用 SQL EXISTS）
	if len(files) > 0 {
		videoIDs := make([]uint, len(files))
		for i, f := range files {
			videoIDs[i] = f.ID
		}

		// 使用 EXISTS 查询批量获取 PPT 状态
		var pptResults []struct {
			VideoFileID uint `gorm:"column:video_file_id"`
		}

		s.db.Table("ppt_files").
			Select("DISTINCT source_video_file_id as video_file_id").
			Where("source_video_file_id IN ?", videoIDs).
			Where("source_video_file_id IS NOT NULL").
			Scan(&pptResults)

		// 构建 PPT 状态映射
		hasPptMap := make(map[uint]bool)
		for _, r := range pptResults {
			hasPptMap[r.VideoFileID] = true
		}

		// 填充 HasPpt 字段
		for i := range files {
			files[i].HasPpt = hasPptMap[files[i].ID]
		}
	}

	return &ListFilesResponse{
		Total:       stats.Count,
		Items:       files,
		TotalSize:   stats.Size,
		TotalSizeGB: float64(stats.Size) / (1024 * 1024 * 1024),
	}, nil
}

// statsResult 统计结果
type statsResult struct {
	Count int64 `gorm:"column:count"`
	Size  int64 `gorm:"column:total_size"`
}

// getStats 获取统计数据
func (s *VideoFileService) getStats(query *gorm.DB) (*statsResult, error) {
	var stats statsResult
	statsQuery := query.Session(&gorm.Session{}).
		Select("COUNT(*) as count, COALESCE(SUM(file_size), 0) as total_size")
	if err := statsQuery.Scan(&stats).Error; err != nil {
		return nil, err
	}
	return &stats, nil
}

// applyFilters 应用筛选条件
func (s *VideoFileService) applyFilters(query *gorm.DB, req *ListFilesRequest) {
	if req.Keyword != "" {
		query = query.Where("file_name LIKE ? OR file_path LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	if req.TaskID != nil {
		query = query.Where("task_id = ?", *req.TaskID)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.Format != "" {
		query = query.Where("format = ?", req.Format)
	}

	if req.SourceType != "" {
		query = query.Where("source_type = ?", req.SourceType)
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
}

// parseDate 解析日期字符串
func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, dateStr)
}

// GetFileByID 根据ID获取文件
func (s *VideoFileService) GetFileByID(id uint) (*models.VideoFile, error) {
	var file models.VideoFile
	if err := s.db.Preload("Task").Where("id = ?", id).First(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

// DeleteFile 删除文件（同时删除整个任务目录和任务记录）
func (s *VideoFileService) DeleteFile(id uint) error {
	var file models.VideoFile
	if err := s.db.First(&file, id).Error; err != nil {
		return errors.New("文件不存在")
	}

	if file.Status == models.FileStatusProcessing {
		return errors.New("文件正在处理中，无法删除")
	}

	// 获取任务ID
	taskID := file.TaskID
	if taskID == nil {
		return errors.New("文件没有关联任务，无法删除")
	}

	// 使用事务删除数据库记录
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 删除任务的所有视频文件记录
		if err := tx.Where("task_id = ?", *taskID).Delete(&models.VideoFile{}).Error; err != nil {
			return err
		}
		// 删除任务记录
		if err := tx.Delete(&models.VideoRecordingTask{}, *taskID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		s.logger.Error("删除数据库记录失败", zap.Uint("file_id", id), zap.Error(err))
		return err
	}

	// 数据库删除成功后，删除物理目录
	taskDirName := fmt.Sprintf("task_%d", *taskID)

	// 删除 recordings 目录
	recordingsDir := filepath.Join(s.recordingsPath, taskDirName)
	if err := os.RemoveAll(recordingsDir); err != nil {
		s.logger.Warn("删除 recordings 目录失败",
			zap.String("dir", recordingsDir),
			zap.Error(err),
		)
	}

	// 删除 HLS 目录
	if s.hlsPath != "" {
		hlsDir := filepath.Join(s.hlsPath, taskDirName)
		if err := os.RemoveAll(hlsDir); err != nil {
			s.logger.Warn("删除 HLS 目录失败",
				zap.String("dir", hlsDir),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("任务已完全删除",
		zap.Uint("file_id", id),
		zap.Uint("task_id", *taskID),
	)

	return nil
}

// BatchDeleteFilesRequest 批量删除文件请求
type BatchDeleteFilesRequest struct {
	IDs []uint `json:"ids"`
}

// BatchDeleteFilesResult 批量删除文件结果
type BatchDeleteFilesResult struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

// BatchDeleteFiles 批量删除文件（按任务分组删除，避免重复删除）
// 返回的 success/failed 计数是删除的文件数，而非任务数
func (s *VideoFileService) BatchDeleteFiles(ids []uint) (*BatchDeleteFilesResult, error) {
	result := &BatchDeleteFilesResult{}

	if len(ids) == 0 {
		return result, nil
	}

	// 查询所有要删除的文件
	var files []models.VideoFile
	if err := s.db.Where("id IN ?", ids).Find(&files).Error; err != nil {
		return result, err
	}

	// 按文件状态和任务关联分类
	var filesWithTask []models.VideoFile // 有关联任务的文件
	var orphanFiles []models.VideoFile   // 孤立文件（TaskID 为 nil）
	var processingFileIDs []uint         // 处理中的文件 ID

	for _, file := range files {
		if file.Status == models.FileStatusProcessing {
			processingFileIDs = append(processingFileIDs, file.ID)
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("文件 %d 正在处理中，无法删除", file.ID))
		} else if file.TaskID == nil {
			orphanFiles = append(orphanFiles, file)
		} else {
			filesWithTask = append(filesWithTask, file)
		}
	}

	// 记录处理中的文件
	if len(processingFileIDs) > 0 {
		s.logger.Info("批量删除跳过处理中的文件",
			zap.Int("count", len(processingFileIDs)),
			zap.Uint("ids", processingFileIDs[0]),
		)
	}

	// 处理孤立文件
	for _, file := range orphanFiles {
		if err := s.deleteOrphanFile(&file); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("删除孤立文件 %d 失败: %v", file.ID, err))
		} else {
			result.Success++
		}
	}

	// 按任务 ID 分组（一个任务可能有两个文件：mp4 和 mkv）
	taskIDToFileCount := make(map[uint]int) // 任务 ID -> 文件数量
	taskIDs := make(map[uint]bool)
	for _, file := range filesWithTask {
		if file.TaskID != nil {
			taskIDs[*file.TaskID] = true
			taskIDToFileCount[*file.TaskID]++
		}
	}

	// 按任务删除
	for taskID := range taskIDs {
		fileCount := taskIDToFileCount[taskID]
		if err := s.deleteByTaskID(taskID); err != nil {
			result.Failed += fileCount
			result.Errors = append(result.Errors, fmt.Sprintf("删除任务 %d（含 %d 个文件）失败: %v", taskID, fileCount, err))
		} else {
			result.Success += fileCount
		}
	}

	s.logger.Info("批量删除文件完成",
		zap.Int("requested", len(ids)),
		zap.Int("success", result.Success),
		zap.Int("failed", result.Failed),
		zap.Int("processing_skipped", len(processingFileIDs)),
		zap.Int("orphan_deleted", len(orphanFiles)),
	)

	return result, nil
}

// deleteOrphanFile 删除孤立文件（没有关联任务的文件）
func (s *VideoFileService) deleteOrphanFile(file *models.VideoFile) error {
	// 使用事务删除数据库记录
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&models.VideoFile{}, file.ID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		s.logger.Error("删除孤立文件数据库记录失败", zap.Uint("file_id", file.ID), zap.Error(err))
		return err
	}

	// 数据库删除成功后，删除物理文件
	if err := os.Remove(file.FilePath); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("删除孤立文件物理文件失败",
			zap.Uint("file_id", file.ID),
			zap.String("file_path", file.FilePath),
			zap.Error(err),
		)
	}

	s.logger.Info("孤立文件已删除", zap.Uint("file_id", file.ID), zap.String("file_path", file.FilePath))

	return nil
}

// deleteByTaskID 按任务 ID 删除（删除整个任务目录）
func (s *VideoFileService) deleteByTaskID(taskID uint) error {
	// 使用事务删除数据库记录
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 删除任务的所有视频文件记录
		if err := tx.Where("task_id = ?", taskID).Delete(&models.VideoFile{}).Error; err != nil {
			return err
		}
		// 删除任务记录
		if err := tx.Delete(&models.VideoRecordingTask{}, taskID).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		s.logger.Error("删除数据库记录失败", zap.Uint("task_id", taskID), zap.Error(err))
		return err
	}

	// 数据库删除成功后，删除物理目录
	taskDirName := fmt.Sprintf("task_%d", taskID)

	// 删除 recordings 目录
	recordingsDir := filepath.Join(s.recordingsPath, taskDirName)
	if err := os.RemoveAll(recordingsDir); err != nil {
		s.logger.Warn("删除 recordings 目录失败",
			zap.String("dir", recordingsDir),
			zap.Error(err),
		)
	}

	// 删除 HLS 目录
	if s.hlsPath != "" {
		hlsDir := filepath.Join(s.hlsPath, taskDirName)
		if err := os.RemoveAll(hlsDir); err != nil {
			s.logger.Warn("删除 HLS 目录失败",
				zap.String("dir", hlsDir),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("任务已完全删除", zap.Uint("task_id", taskID))

	return nil
}

// findCounterpartFile 查找对应格式的文件
func (s *VideoFileService) findCounterpartFile(file *models.VideoFile) *models.VideoFile {
	if file.TaskID == nil {
		return nil
	}

	var counterpart models.VideoFile
	targetFormat := "mp4"
	if file.Format == "mp4" {
		targetFormat = "mkv"
	}

	if err := s.db.Where("task_id = ? AND format = ? AND id != ?",
		*file.TaskID, targetFormat, file.ID).First(&counterpart).Error; err != nil {
		return nil
	}

	return &counterpart
}

// deletePhysicalFile 删除物理文件
func (s *VideoFileService) deletePhysicalFile(file *models.VideoFile) error {
	if !file.Exists() {
		return nil
	}

	if err := file.Delete(); err != nil {
		return err
	}

	s.logger.Info("已删除物理文件",
		zap.Uint("file_id", file.ID),
		zap.String("file_path", file.FilePath),
		zap.String("format", file.Format),
	)

	return nil
}

// deleteCounterpartFile 删除对应格式的文件
func (s *VideoFileService) deleteCounterpartFile(file *models.VideoFile) {
	if file.Status == models.FileStatusProcessing {
		s.logger.Warn("对应格式文件正在处理中，跳过删除",
			zap.Uint("counterpart_id", file.ID),
			zap.String("counterpart_format", file.Format),
		)
		return
	}

	if err := s.deletePhysicalFile(file); err != nil {
		s.logger.Warn("删除对应格式物理文件失败",
			zap.Uint("counterpart_id", file.ID),
			zap.String("counterpart_path", file.FilePath),
			zap.Error(err),
		)
	}

	if err := s.db.Delete(&models.VideoFile{}, file.ID).Error; err != nil {
		s.logger.Warn("删除对应格式数据库记录失败",
			zap.Uint("counterpart_id", file.ID),
			zap.Error(err),
		)
	} else {
		s.logger.Info("已删除对应格式文件",
			zap.Uint("counterpart_id", file.ID),
			zap.String("counterpart_format", file.Format),
		)
	}
}

// GetFilesByTaskID 根据任务ID获取文件列表
func (s *VideoFileService) GetFilesByTaskID(taskID uint) ([]models.VideoFile, error) {
	var files []models.VideoFile
	if err := s.db.Where("task_id = ?", taskID).
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

// GetFileStats 获取文件统计信息（可按格式筛选）
func (s *VideoFileService) GetFileStats(format string) (map[string]interface{}, error) {
	query := s.db.Model(&models.VideoFile{})

	// 按格式筛选（默认只统计 mp4，忽略 mkv）
	// 如果 format 为空，则默认只统计 mp4
	if format == "" {
		format = "mp4"
	}
	query = query.Where("format = ?", format)

	var total int64
	var totalSize int64

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if err := query.
		Select("COALESCE(SUM(file_size), 0)").
		Scan(&totalSize).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":         total,
		"total_size":    totalSize,
		"total_size_gb": float64(totalSize) / (1024 * 1024 * 1024),
	}, nil
}

// extractVideoMetadata 使用 ffprobe 提取视频元数据
func (s *VideoFileService) extractVideoMetadata(filePath string) (*videoMetadata, error) {
	metadata := s.getDefaultMetadata(filePath)

	// 从文件扩展名推断格式
	metadata.Format = s.inferFormat(filePath)

	// 检查 ffprobe 是否存在（支持无扩展名和 .exe 扩展名）
	ffprobeExists := false
	actualFFprobePath := s.ffprobePath
	ffprobePaths := []string{s.ffprobePath, s.ffprobePath + ".exe"}
	for _, path := range ffprobePaths {
		if _, err := os.Stat(path); err == nil {
			ffprobeExists = true
			actualFFprobePath = path
			break
		}
	}

	if !ffprobeExists {
		s.logger.Warn("ffprobe 不存在，使用默认元数据",
			zap.String("ffprobe", s.ffprobePath),
			zap.String("searched_paths", strings.Join(ffprobePaths, ", ")),
			zap.String("file", filePath),
		)
		return metadata, nil
	}

	detailedMetadata, err := s.probeVideoMetadata(actualFFprobePath, filePath)
	if err != nil {
		s.logger.Warn("ffprobe 执行失败，使用默认元数据",
			zap.String("file", filePath),
			zap.Error(err),
		)
		return metadata, nil
	}

	// 合并详细元数据
	metadata.Duration = detailedMetadata.Duration
	metadata.Bitrate = detailedMetadata.Bitrate
	metadata.Codec = detailedMetadata.Codec
	if detailedMetadata.Resolution != "" {
		metadata.Resolution = detailedMetadata.Resolution
	}

	return metadata, nil
}

// getDefaultMetadata 获取默认元数据
func (s *VideoFileService) getDefaultMetadata(filePath string) *videoMetadata {
	return &videoMetadata{
		Format:     s.inferFormat(filePath),
		Duration:   0,
		Resolution: DefaultResolution,
		Bitrate:    DefaultBitrate,
		Codec:      DefaultCodec,
	}
}

// inferFormat 从文件路径推断格式
func (s *VideoFileService) inferFormat(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".mp4":
		return "mp4"
	case ".mkv":
		return "mkv"
	default:
		return DefaultFormat
	}
}

// probeVideoMetadata 使用 ffprobe 获取详细元数据
func (s *VideoFileService) probeVideoMetadata(ffprobePath, filePath string) (*videoMetadata, error) {
	ctx, cancel := context.WithTimeout(context.Background(), FFProbeTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return s.parseFFProbeOutput(output)
}

// parseFFProbeOutput 解析 ffprobe 输出
func (s *VideoFileService) parseFFProbeOutput(output []byte) (*videoMetadata, error) {
	var result struct {
		Format struct {
			Duration   string `json:"duration"`
			BitRate    string `json:"bit_rate"`
			FormatName string `json:"format_name"`
		} `json:"format"`
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	metadata := &videoMetadata{}

	if result.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			metadata.Duration = duration
		}
	}

	if result.Format.BitRate != "" {
		if bitrate, err := strconv.Atoi(result.Format.BitRate); err == nil {
			metadata.Bitrate = bitrate / 1000
		}
	}

	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			if stream.CodecName != "" {
				metadata.Codec = stream.CodecName
			}
			if stream.Width > 0 && stream.Height > 0 {
				metadata.Resolution = fmt.Sprintf("%dx%d", stream.Width, stream.Height)
			}
			break
		}
	}

	return metadata, nil
}

// CreateFileFromTask 从录制任务创建文件记录
func (s *VideoFileService) CreateFileFromTask(task *models.VideoRecordingTask, format *string) (*models.VideoFile, error) {
	if task == nil {
		return nil, errors.New("创建文件记录失败：任务对象为 nil")
	}

	formatStr := "mp4"
	if format != nil && *format != "" {
		formatStr = *format
	}

	filePath, err := s.getTaskFilePath(task, formatStr)
	if err != nil {
		return nil, err
	}

	// 检查文件是否已存在
	if existingFile, err := s.findExistingFile(filePath); err == nil {
		return existingFile, nil
	}

	// 创建新文件记录
	return s.createNewFile(filePath, &task.ID, nil, formatStr, task.CreatedBy)
}

// getTaskFilePath 获取任务文件路径
func (s *VideoFileService) getTaskFilePath(task *models.VideoRecordingTask, format string) (string, error) {
	var filePath string
	switch strings.ToLower(format) {
	case "mkv":
		filePath = task.MKVFilePath
	case "mp4":
		filePath = task.MP4FilePath
	default:
		return "", fmt.Errorf("不支持的格式: %s", format)
	}

	if filePath == "" {
		return "", fmt.Errorf("%s 文件路径为空", format)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("文件不存在: %s", filePath)
	}

	return filePath, nil
}

// findExistingFile 查找已存在的文件
func (s *VideoFileService) findExistingFile(filePath string) (*models.VideoFile, error) {
	var existingFile models.VideoFile
	if err := s.db.Where("file_path = ?", filePath).First(&existingFile).Error; err != nil {
		return nil, err
	}

	s.logger.Info("文件记录已存在",
		zap.Uint("file_id", existingFile.ID),
		zap.String("file_path", filePath),
	)

	return &existingFile, nil
}

// createNewFile 创建新的文件记录
func (s *VideoFileService) createNewFile(filePath string, taskID *uint, recordedAt *time.Time, format string, createdBy uint) (*models.VideoFile, error) {
	// 提取元数据
	metadata, err := s.extractVideoMetadata(filePath)
	if err != nil {
		s.logger.Warn("提取视频元数据失败",
			zap.String("file", filePath),
			zap.Error(err),
		)
		metadata = &videoMetadata{
			Format:     format,
			Duration:   0,
			Resolution: "",
			Bitrate:    0,
			Codec:      "",
		}
	}

	// 获取文件信息
	fileInfo, _ := os.Stat(filePath)
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	// 调试日志：记录 taskID 值
	if taskID != nil {
		s.logger.Debug("准备创建文件记录",
			zap.Uint("task_id", *taskID),
			zap.String("file_path", filePath),
		)
	}

	// 创建文件记录
	videoFile := &models.VideoFile{
		FileName:   filepath.Base(filePath),
		FilePath:   filePath,
		FileSize:   fileSize,
		Duration:   int(metadata.Duration),
		Format:     metadata.Format,
		Resolution: metadata.Resolution,
		Bitrate:    metadata.Bitrate,
		Codec:      metadata.Codec,
		Status:     models.FileStatusReady,
		TaskID:     taskID,
		CreatedBy:  createdBy,
		RecordedAt: recordedAt,
	}

	if err := s.createWithDuplicateCheck(videoFile); err != nil {
		// 如果是外键约束失败，尝试验证任务是否存在
		if strings.Contains(err.Error(), "FOREIGN KEY") && taskID != nil {
			var taskExists models.VideoRecordingTask
			if checkErr := s.db.Select("id").First(&taskExists, *taskID).Error; checkErr != nil {
				s.logger.Error("外键约束失败：任务不存在",
					zap.Uint("task_id", *taskID),
					zap.Error(checkErr),
				)
			} else {
				s.logger.Error("外键约束失败：任务存在但仍创建失败",
					zap.Uint("task_id", *taskID),
					zap.Error(err),
				)
			}
		}
		return nil, err
	}

	s.logger.Info("已创建文件记录",
		zap.Uint("file_id", videoFile.ID),
		zap.Uint("task_id", videoFile.ID),
		zap.String("file_name", videoFile.FileName),
		zap.String("format", format),
	)

	return videoFile, nil
}

// createWithDuplicateCheck 创建记录并处理重复
func (s *VideoFileService) createWithDuplicateCheck(videoFile *models.VideoFile) error {
	if err := s.db.Create(videoFile).Error; err != nil {
		if s.isDuplicateError(err) {
			// 并发情况下，重新查询现有记录
			var existingFile models.VideoFile
			if err := s.db.Where("file_path = ?", videoFile.FilePath).First(&existingFile).Error; err == nil {
				*videoFile = existingFile
				return nil
			}
		}
		return fmt.Errorf("创建文件记录失败: %w", err)
	}
	return nil
}

// isDuplicateError 检查是否为重复错误
func (s *VideoFileService) isDuplicateError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "UNIQUE") || strings.Contains(errStr, "duplicate")
}

// CreateFile 从文件路径创建文件记录（通用方法）
func (s *VideoFileService) CreateFile(filePath string, taskID *uint, recordedAt *time.Time) (*models.VideoFile, error) {
	if filePath == "" {
		return nil, errors.New("创建文件记录失败：文件路径为空")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 检查是否已存在
	if existingFile, err := s.findExistingFile(filePath); err == nil {
		return existingFile, nil
	}

	format := s.inferFormat(filePath)

	// 获取 createdBy：如果有 taskID，从任务中获取
	var createdBy uint = 1 // 默认系统用户
	if taskID != nil {
		var task models.VideoRecordingTask
		if err := s.db.Select("created_by").First(&task, *taskID).Error; err == nil {
			createdBy = task.CreatedBy
		}
	}

	return s.createNewFile(filePath, taskID, recordedAt, format, createdBy)
}

// CreateSegmentFile 创建分割段文件记录
func (s *VideoFileService) CreateSegmentFile(segmentPath string, parentVideoID *uint, sourceType string, createdBy uint, snapshotOffset ...float64) (*models.VideoFile, error) {
	// 1. 检查文件是否已存在（重复检查）
	existing, err := s.findExistingFile(segmentPath)
	if err == nil && existing != nil {
		s.logger.Warn("文件记录已存在", zap.String("path", segmentPath), zap.Uint("existing_id", existing.ID))
		return existing, nil
	}

	// 2. 获取文件信息
	fileInfo, err := os.Stat(segmentPath)
	if err != nil {
		return nil, fmt.Errorf("文件不存在: %w", err)
	}

	// 3. 提取元数据（使用现有方法）
	metadata, err := s.extractVideoMetadata(segmentPath)
	if err != nil {
		s.logger.Warn("提取视频元数据失败，使用默认值", zap.Error(err))
		metadata = s.getDefaultMetadata(segmentPath)
	}

	// 4. 生成文件名
	fileName := filepath.Base(segmentPath)

	// 5. 推断格式
	format := s.inferFormat(segmentPath)

	// 6. 确定快照偏移量
	offset := 0.0
	if len(snapshotOffset) > 0 {
		offset = snapshotOffset[0]
	}

	// 7. 创建 VideoFile 记录
	videoFile := &models.VideoFile{
		FileName:       fileName,
		FilePath:       segmentPath,
		FileSize:       fileInfo.Size(),
		Duration:       int(metadata.Duration),
		Format:         format,
		Resolution:     metadata.Resolution,
		Bitrate:        metadata.Bitrate,
		Codec:          metadata.Codec,
		ParentID:       parentVideoID,
		SourceType:     sourceType,
		SnapshotOffset: offset,
		CreatedBy:      createdBy,
		Status:         models.FileStatusReady,
		RecordedAt:     nil,
	}

	// 8. 从父视频继承 RecordedAt
	if parentVideoID != nil {
		var parent models.VideoFile
		if err := s.db.First(&parent, *parentVideoID).Error; err == nil {
			videoFile.RecordedAt = parent.RecordedAt
		}
	}

	// 9. 保存到数据库
	if err := s.createWithDuplicateCheck(videoFile); err != nil {
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	return videoFile, nil
}

// GetSegmentsByParentID 根据父视频ID获取所有分割段
func (s *VideoFileService) GetSegmentsByParentID(parentID uint) ([]models.VideoFile, error) {
	var segments []models.VideoFile
	err := s.db.Where("parent_id = ?", parentID).Order("id ASC").Find(&segments).Error
	return segments, err
}

// fileInfoWithPath 包含文件路径和推断的任务ID
type fileInfoWithPath struct {
	filePath string
	taskID   *uint
}

// ScanFiles 扫描录制目录并导入未入库的文件
func (s *VideoFileService) ScanFiles() (*ScanResult, error) {
	result := &ScanResult{}

	files, err := s.findVideoFiles(s.recordingsPath)
	if err != nil {
		return result, fmt.Errorf("查找视频文件失败: %w", err)
	}

	result.Scanned = len(files)

	if len(files) == 0 {
		s.logger.Info("扫描目录为空", zap.String("directory", s.recordingsPath))
		return result, nil
	}

	s.logger.Info("开始扫描录制文件",
		zap.Int("total_files", len(files)),
		zap.String("directory", s.recordingsPath),
	)

	// 批量查询已存在的文件
	existingMap, err := s.getExistingFilesMap()
	if err != nil {
		return result, fmt.Errorf("批量查询现有文件失败: %w", err)
	}

	// 处理文件
	s.processFiles(files, existingMap, result)

	s.logger.Info("文件扫描完成",
		zap.Int("scanned", result.Scanned),
		zap.Int("created", result.Created),
		zap.Int("updated", result.Updated),
		zap.Int("skipped", result.Skipped),
		zap.Int("errors", len(result.Errors)),
	)

	return result, nil
}

// getExistingFilesMap 获取已存在文件的映射
func (s *VideoFileService) getExistingFilesMap() (map[string]bool, error) {
	var existingFiles []models.VideoFile
	if err := s.db.Select("file_path").Find(&existingFiles).Error; err != nil {
		return nil, err
	}

	existingMap := make(map[string]bool, len(existingFiles))
	for _, f := range existingFiles {
		existingMap[f.FilePath] = true
	}

	return existingMap, nil
}

// processFiles 处理扫描的文件
func (s *VideoFileService) processFiles(files []fileInfoWithPath, existingMap map[string]bool, result *ScanResult) {
	for _, file := range files {
		if existingMap[file.filePath] {
			s.handleExistingFile(file, result)
			continue
		}

		if _, err := s.CreateFile(file.filePath, file.taskID, nil); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("创建记录 %s 失败: %v", file.filePath, err))
			continue
		}

		result.Created++
	}
}

// handleExistingFile 处理已存在的文件
func (s *VideoFileService) handleExistingFile(file fileInfoWithPath, result *ScanResult) {
	var existingFile models.VideoFile
	if err := s.db.Where("file_path = ?", file.filePath).First(&existingFile).Error; err != nil {
		result.Skipped++
		return
	}

	// 检查文件是否关联了任务
	var taskID uint
	if existingFile.TaskID != nil {
		taskID = *existingFile.TaskID
	} else if file.taskID != nil {
		taskID = *file.taskID
	}

	// 如果有任务ID，检查转换状态
	if taskID > 0 {
		var task models.VideoRecordingTask
		if err := s.db.Select("id, "+
			"conversion_status, "+
			"status").First(&task, taskID).Error; err == nil {

			// 如果任务还在转换中或转换失败，跳过元数据更新
			// 因为此时文件可能不完整或数据不准确
			if task.ConversionStatus == models.ConversionStatusProcessing ||
				task.ConversionStatus == models.ConversionStatusPending {
				result.Skipped++
				s.logger.Debug("跳过转换中的文件",
					zap.Uint("file_id", existingFile.ID),
					zap.Uint("task_id", taskID),
					zap.String("conversion_status", string(task.ConversionStatus)),
				)
				return
			}

			// 如果任务状态是转换中（converting），也跳过
			if task.Status == models.VideoStatusConverting {
				result.Skipped++
				s.logger.Debug("跳过转换中的文件（任务状态）",
					zap.Uint("file_id", existingFile.ID),
					zap.Uint("task_id", taskID),
					zap.String("task_status", string(task.Status)),
				)
				return
			}
		}
	}

	// 如果转换已完成，更新文件的元数据（文件大小、码率、时长等）
	updated := false

	// 如果 task_id 为空且可以推断出任务ID，则更新
	if existingFile.TaskID == nil && file.taskID != nil {
		if s.validateTaskID(*file.taskID) {
			s.db.Model(&existingFile).Update("task_id", file.taskID)
			s.logger.Info("更新文件记录的task_id",
				zap.Uint("file_id", existingFile.ID),
				zap.Uint("task_id", *file.taskID),
			)
			updated = true
		}
	}

	// 重新提取文件元数据并更新
	// 使用新的元数据更新文件大小、码率、时长等信息
	metadata, err := s.extractVideoMetadata(file.filePath)
	if err != nil {
		s.logger.Warn("提取视频元数据失败",
			zap.String("file", file.filePath),
			zap.Error(err),
		)
	} else {
		// 获取文件大小
		fileInfo, _ := os.Stat(file.filePath)
		fileSize := int64(0)
		if fileInfo != nil {
			fileSize = fileInfo.Size()
		}

		// 检查是否需要更新（只在数据有变化时更新）
		needsUpdate := false
		updates := make(map[string]interface{})

		if existingFile.FileSize != fileSize {
			updates["file_size"] = fileSize
			needsUpdate = true
		}

		if existingFile.Duration != int(metadata.Duration) {
			updates["duration"] = int(metadata.Duration)
			needsUpdate = true
		}

		if existingFile.Bitrate != metadata.Bitrate {
			updates["bitrate"] = metadata.Bitrate
			needsUpdate = true
		}

		if existingFile.Resolution != metadata.Resolution && metadata.Resolution != "" {
			updates["resolution"] = metadata.Resolution
			needsUpdate = true
		}

		if existingFile.Codec != metadata.Codec && metadata.Codec != "" {
			updates["codec"] = metadata.Codec
			needsUpdate = true
		}

		// 如果需要更新，执行更新
		if needsUpdate {
			if err := s.db.Model(&existingFile).Updates(updates).Error; err != nil {
				s.logger.Error("更新文件元数据失败",
					zap.Uint("file_id", existingFile.ID),
					zap.Error(err),
				)
			} else {
				s.logger.Info("更新文件元数据",
					zap.Uint("file_id", existingFile.ID),
					zap.String("file_path", file.filePath),
					zap.Int64("file_size", fileSize),
					zap.Int("duration", int(metadata.Duration)),
					zap.Int("bitrate", metadata.Bitrate),
				)
				updated = true
			}
		}
	}

	if updated {
		result.Updated++
	} else {
		result.Skipped++
	}
}

// validateTaskID 验证任务ID是否存在
func (s *VideoFileService) validateTaskID(taskID uint) bool {
	var taskExists models.VideoRecordingTask
	return s.db.Select("id").First(&taskExists, taskID).Error == nil
}

// findVideoFiles 查找目录下所有视频文件
func (s *VideoFileService) findVideoFiles(dir string) ([]fileInfoWithPath, error) {
	var files []fileInfoWithPath

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		ext := strings.ToLower(filepath.Ext(path))
		// 只扫描 MP4 文件，忽略 MKV（MKV 是中间格式，MP4 是最终格式）
		if ext == ".mp4" {
			files = append(files, fileInfoWithPath{
				filePath: path,
				taskID:   s.extractTaskIDFromPath(path),
			})
		}

		return nil
	})

	return files, err
}

// RenameVideoFile renames a video file with atomic database and filesystem update
// Parameters:
//   - id: video file ID
//   - newName: new filename without extension (extension will be preserved)
//   - userID: user ID requesting the rename (for ownership validation)
func (s *VideoFileService) RenameVideoFile(id uint, newName string, userID uint, hasSharedViewer bool) error {
	// Validation: load video file
	var videoFile models.VideoFile
	if err := s.db.First(&videoFile, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("视频文件不存在")
		}
		return fmt.Errorf("查询视频文件失败: %w", err)
	}

	// Validation: check ownership (admin or shared_viewer can rename any file)
	if !hasSharedViewer && videoFile.CreatedBy != userID {
		return fmt.Errorf("无权重命名此文件")
	}

	// Validation: check immutability (original recordings cannot be renamed)
	if videoFile.SourceType == models.SourceTypeRecording && videoFile.ParentID == nil {
		return fmt.Errorf("不能重命名原始录制文件")
	}

	// Validation: sanitize new name
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("新文件名不能为空")
	}
	if len(newName) > 200 {
		return fmt.Errorf("新文件名过长（最大200字符）")
	}
	// Reject path separators to prevent path traversal attacks
	if strings.ContainsAny(newName, "/\\") {
		return fmt.Errorf("文件名不能包含路径分隔符")
	}

	// Preserve file extension
	newName = strings.TrimSuffix(newName, filepath.Ext(newName))
	if newName == "" {
		return fmt.Errorf("新文件名不能为空")
	}
	ext := filepath.Ext(videoFile.FilePath)
	if ext == "" {
		ext = ".mp4" // Default extension if none exists
	}
	newFileName := newName + ext

	// Validation: check for duplicate filename in same directory
	dir := filepath.Dir(videoFile.FilePath)
	newFilePath := filepath.Join(dir, newFileName)

	// Check if another file with the same path already exists
	var existingFile models.VideoFile
	err := s.db.Where("file_path = ? AND id != ?", newFilePath, id).First(&existingFile).Error
	if err == nil {
		return fmt.Errorf("目标文件名已存在")
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查文件名重复失败: %w", err)
	}

	// Atomic rename: database transaction + filesystem rename
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Rename physical file
		if err := os.Rename(videoFile.FilePath, newFilePath); err != nil {
			s.logger.Warn("重命名物理文件失败",
				zap.Uint("file_id", id),
				zap.String("old_path", videoFile.FilePath),
				zap.String("new_path", newFilePath),
				zap.Error(err),
			)
			return fmt.Errorf("重命名物理文件失败: %w", err)
		}

		// Step 2: Update database record
		if err := tx.Model(&videoFile).Updates(map[string]interface{}{
			"file_name": newFileName,
			"file_path": newFilePath,
		}).Error; err != nil {
			// Rollback: try to revert physical file rename
			s.logger.Error("更新数据库失败，尝试回滚文件重命名",
				zap.Uint("file_id", id),
				zap.Error(err),
			)
			if rollbackErr := os.Rename(newFilePath, videoFile.FilePath); rollbackErr != nil {
				s.logger.Error("回滚文件重命名失败",
					zap.String("from", newFilePath),
					zap.String("to", videoFile.FilePath),
					zap.Error(rollbackErr),
				)
			}
			return fmt.Errorf("更新数据库记录失败: %w", err)
		}

		s.logger.Info("视频文件重命名成功",
			zap.Uint("file_id", id),
			zap.String("old_name", videoFile.FileName),
			zap.String("new_name", newFileName),
		)

		return nil
	})

	return err
}

// extractTaskIDFromPath 从路径提取任务ID
func (s *VideoFileService) extractTaskIDFromPath(path string) *uint {
	dir := filepath.Dir(path)
	if matches := filepath.Base(dir); strings.HasPrefix(matches, "task_") {
		idStr := strings.TrimPrefix(matches, "task_")
		if id, err := strconv.Atoi(idStr); err == nil {
			taskIDUint := uint(id)
			return &taskIDUint
		}
	}
	return nil
}

// DeleteSplitSegmentsByParentID deletes all split segments for a parent video
// Parameters:
//   - parentID: parent video file ID
//   - userID: user ID requesting the deletion (for ownership validation)
// Returns:
//   - count: number of segments deleted
//   - error: error if deletion fails
//
// This method performs cascade deletion:
// 1. Queries all VideoFile records where parent_id = parentID
// 2. Filters by ownership (created_by = userID)
// 3. Filters by source_type IN ('split', 'snapshot') - never deletes original recordings
// 4. Deletes physical files and thumbnails
// 5. Deletes database records atomically
func (s *VideoFileService) DeleteSplitSegmentsByParentID(parentID uint, userID uint) (int, error) {
	// 1. Query segments with ownership and source type validation
	var segments []models.VideoFile
	err := s.db.Where("parent_id = ? AND created_by = ? AND source_type IN ?", parentID, userID, []string{"split", "snapshot"}).Find(&segments).Error
	if err != nil {
		return 0, fmt.Errorf("failed to query segments: %w", err)
	}

	// Return early if no segments found
	if len(segments) == 0 {
		return 0, nil
	}

	s.logger.Info("开始删除分割段",
		zap.Uint("parent_id", parentID),
		zap.Uint("user_id", userID),
		zap.Int("segment_count", len(segments)),
	)

	// 2. Delete in transaction
	deletedCount := 0
	err = s.db.Transaction(func(tx *gorm.DB) error {
		for _, segment := range segments {
			// Delete physical file
			if segment.FilePath != "" {
				if err := os.Remove(segment.FilePath); err != nil {
					if !os.IsNotExist(err) {
						s.logger.Warn("删除分割段物理文件失败",
							zap.Uint("segment_id", segment.ID),
							zap.String("path", segment.FilePath),
							zap.Error(err),
						)
					}
					// Continue with DB deletion even if physical file is missing
				}
			}

			// Delete thumbnail if exists
			if segment.ThumbnailPath != nil && *segment.ThumbnailPath != "" {
				if err := os.Remove(*segment.ThumbnailPath); err != nil {
					if !os.IsNotExist(err) {
						s.logger.Warn("删除缩略图失败",
							zap.Uint("segment_id", segment.ID),
							zap.String("thumbnail_path", *segment.ThumbnailPath),
							zap.Error(err),
						)
					}
				}
			}

			// Delete database record
			if err := tx.Delete(&segment).Error; err != nil {
				return fmt.Errorf("failed to delete segment record %d: %w", segment.ID, err)
			}

			deletedCount++
			s.logger.Debug("已删除分割段",
				zap.Uint("segment_id", segment.ID),
				zap.String("file_name", segment.FileName),
			)
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to delete segments: %w", err)
	}

	s.logger.Info("成功删除分割段",
		zap.Uint("parent_id", parentID),
		zap.Uint("user_id", userID),
		zap.Int("deleted_count", deletedCount),
	)

	return deletedCount, nil
}


// BatchDownloadFilesRequest 批量下载请求
type BatchDownloadFilesRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1,dive,min=1"`
}

// BatchDownloadFilesResponse 批量下载响应
type BatchDownloadFilesResponse struct {
	Reader      io.ReadCloser
	Filename    string
	ContentType string
	FileCount   int
	TotalSize   int64
}

// BatchDownloadFiles 批量下载文件（打包为ZIP）
func (s *VideoFileService) BatchDownloadFiles(ids []uint, userID uint, isAdmin bool) (*BatchDownloadFilesResponse, error) {
	// 查询所有文件
	var files []models.VideoFile
	if err := s.db.Where("id IN ?", ids).Find(&files).Error; err != nil {
		return nil, fmt.Errorf("查询文件失败: %w", err)
	}

	// 验证所有权并检查文件存在性
	var validFiles []models.VideoFile
	for _, file := range files {
		// 权限检查：管理员或文件所有者
		if !isAdmin && file.CreatedBy != userID {
			s.logger.Warn("跳过无权限文件",
				zap.Uint("file_id", file.ID),
				zap.Uint("file_owner", file.CreatedBy),
				zap.Uint("user_id", userID),
			)
			continue
		}

		// 检查物理文件是否存在
		if file.Exists() {
			validFiles = append(validFiles, file)
		} else {
			s.logger.Warn("文件不存在，跳过",
				zap.Uint("file_id", file.ID),
				zap.String("file_path", file.FilePath),
			)
		}
	}

	if len(validFiles) == 0 {
		return nil, errors.New("没有有效的文件可下载")
	}

	// 创建流式 ZIP
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		zipWriter := zip.NewWriter(pw)
		defer zipWriter.Close()

		for _, file := range validFiles {
			folder := s.getFileFolder(file)
			if err := s.addFileToZip(zipWriter, &file, folder); err != nil {
				s.logger.Warn("添加文件到ZIP失败",
					zap.Uint("file_id", file.ID),
					zap.Error(err),
				)
			}
		}
	}()

	// 生成文件名
	filename := fmt.Sprintf("files_batch_%s.zip", time.Now().Format("20060102_150405"))

	return &BatchDownloadFilesResponse{
		Reader:      pr,
		Filename:    filename,
		ContentType: "application/zip",
		FileCount:   len(validFiles),
		TotalSize:   0, // 流式响应无法预先知道大小
	}, nil
}

// getFileFolder 根据文件类型返回ZIP内文件夹
func (s *VideoFileService) getFileFolder(file models.VideoFile) string {
	ext := strings.ToLower(filepath.Ext(file.FilePath))
	switch ext {
	case ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm":
		return "video"
	case ".pptx":
		return "ppt"
	default:
		return "other"
	}
}

// addFileToZip 添加文件到ZIP
func (s *VideoFileService) addFileToZip(zipWriter *zip.Writer, file *models.VideoFile, folder string) error {
	// 打开文件
	f, err := os.Open(file.FilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// 创建ZIP内文件路径
	zipPath := filepath.Join(folder, filepath.Base(file.FilePath))

	// 创建ZIP文件头
	header := &zip.FileHeader{
		Name:   zipPath,
		Method: zip.Deflate, // 使用压缩
	}
	header.SetModTime(time.Now())

	// 创建ZIP写入器
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// 复制文件内容
	_, err = io.Copy(writer, f)
	return err
}
