package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// parseDate 解析日期字符串
func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, dateStr)
}

// VideoFileService 视频文件服务
type VideoFileService struct {
	db             *gorm.DB
	logger         *zap.Logger
	recordingsPath string // 录制文件存储路径（可配置）
}

// NewVideoFileService 创建视频文件服务
func NewVideoFileService(db *gorm.DB, logger *zap.Logger, recordingsPath string) *VideoFileService {
	return &VideoFileService{
		db:             db,
		logger:         logger,
		recordingsPath: recordingsPath,
	}
}

// ListFilesRequest 文件列表请求
type ListFilesRequest struct {
	Page       int    `form:"page"`
	PageSize    int    `form:"page_size" binding:"max=100"`
	Keyword     string  `form:"keyword"`
	TaskID      *uint   `form:"task_id"` // 任务ID筛选
	Status      string  `form:"status"`
	Format      string  `form:"format"`
	StartDate   string  `form:"start_date"`
	EndDate     string  `form:"end_date"`
	// 数据范围过滤字段
	UserID       uint   `form:"-"` // 当前用户ID（不从query读取，由handler设置）
	IsAdmin      bool   `form:"-"` // 是否管理员（不从query读取，由handler设置）
	ApplyDataScope bool   `form:"-"` // 是否应用数据范围过滤
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

	query := s.db.Model(&models.VideoFile{}).Preload("Task")

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("file_name LIKE ? OR file_path LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 任务筛选
	if req.TaskID != nil {
		query = query.Where("task_id = ?", *req.TaskID)
	}

	// 状态筛选
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 格式筛选
	if req.Format != "" {
		query = query.Where("format = ?", req.Format)
	}

	// 数据范围过滤：非管理员只能看自己创建的文件
	if req.ApplyDataScope && !req.IsAdmin && req.UserID > 0 {
		query = query.Where("created_by = ?", req.UserID)
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

	// 统计总数和总大小（使用 SQL 聚合优化性能）
	type StatsResult struct {
		Count int64  `gorm:"column:count"`
		Size  int64  `gorm:"column:total_size"`
	}

	var stats StatsResult
	// 必须clone query，否则Select会影响原始query对象
	statsQuery := query.Session(&gorm.Session{}).Select("COUNT(*) as count, COALESCE(SUM(file_size), 0) as total_size")
	if err := statsQuery.Scan(&stats).Error; err != nil {
		return nil, err
	}

	total = stats.Count
	totalSize := stats.Size

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Offset(offset).
		Limit(req.PageSize).
		Order("created_at DESC").
		Find(&files).Error; err != nil {
		return nil, err
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
	if err := s.db.Preload("Task").First(&file, id).Error; err != nil {
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
			s.logger.Warn("删除物理文件失败",
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

	s.logger.Info("视频文件已删除", zap.Uint("file_id", id))

	return nil
}

// GetFilesByTaskID 根据任务ID获取文件列表
// 直接通过任务ID查询视频文件（移除中间表关联）
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

// ScanResult 扫描结果
type ScanResult struct {
	Scanned  int      `json:"scanned"`  // 扫描的文件数
	Created  int      `json:"created"`  // 创建的记录数
	Skipped  int      `json:"skipped"`  // 跳过的文件数
	Errors   []string `json:"errors"`   // 错误列表
}

// internalVideoMetadata 内部视频元数据（用于ffprobe解析）
type internalVideoMetadata struct {
	Format     string
	Duration   float64
	Resolution string
	Bitrate    int
	Codec      string
}

// extractVideoMetadata 使用 ffprobe 提取视频元数据
func (s *VideoFileService) extractVideoMetadata(filePath string) (*internalVideoMetadata, error) {
	// 尝试使用 ffprobe
	metadata := &internalVideoMetadata{
		Format: "mkv", // 默认格式
	}

	// 先检查文件是否存在以获取基本信息
	if _, err := os.Stat(filePath); err == nil {
		// 从文件扩展名推断格式
		ext := strings.ToLower(filepath.Ext(filePath))
		if ext == ".mp4" {
			metadata.Format = "mp4"
		} else if ext == ".mkv" {
			metadata.Format = "mkv"
		}
		metadata.Bitrate = 0 // 默认值
		metadata.Codec = "h264"
		metadata.Resolution = "1920x1080" // 默认分辨率
		metadata.Duration = 0
	}

	// 尝试使用 ffprobe 提取详细元数据
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		s.logger.Warn("ffprobe 不可用，使用基础元数据",
			zap.String("file", filePath),
			zap.Error(err),
		)
		return metadata, nil
	}

	// 创建带超时的 context（30秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 执行 ffprobe 命令
	cmd := exec.CommandContext(ctx, ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		s.logger.Warn("ffprobe 执行失败，使用基础元数据",
			zap.String("file", filePath),
			zap.Error(err),
		)
		return metadata, nil
	}

	// 解析 JSON 输出
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
		s.logger.Warn("解析 ffprobe 输出失败，使用基础元数据",
			zap.String("file", filePath),
			zap.Error(err),
		)
		return metadata, nil
	}

	// 提取时长（秒）
	if result.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			metadata.Duration = duration
		}
	}

	// 提取比特率
	if result.Format.BitRate != "" {
		if bitrate, err := strconv.Atoi(result.Format.BitRate); err == nil {
			// 转换为 kbps
			metadata.Bitrate = bitrate / 1000
		}
	}

	// 查找视频流
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			// 提取编码器
			if stream.CodecName != "" {
				metadata.Codec = stream.CodecName
			}
			// 提取分辨率
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
		return nil, fmt.Errorf("创建文件记录失败：任务对象为 nil")
	}

	// 处理 format 参数：如果为 nil 或空字符串，使用默认值 "mp4"
	formatStr := "mp4"
	if format != nil && *format != "" {
		formatStr = *format
	}

	// 根据格式确定文件路径
	var filePath string
	switch strings.ToLower(formatStr) {
	case "mkv":
		filePath = task.MKVFilePath
	case "mp4":
		filePath = task.MP4FilePath
	default:
		return nil, fmt.Errorf("不支持的格式: %s", formatStr)
	}

	if filePath == "" {
		return nil, fmt.Errorf("%s 文件路径为空", formatStr)
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 检查是否已存在相同路径的记录
	var existingFile models.VideoFile
	if err := s.db.Where("file_path = ?", filePath).First(&existingFile).Error; err == nil {
		s.logger.Info("文件记录已存在（幂等性）",
			zap.Uint("task_id", task.ID),
			zap.String("file_path", filePath),
			zap.Uint("existing_id", existingFile.ID),
		)
		return &existingFile, nil
	}

	// 提取视频元数据
	metadata, err := s.extractVideoMetadata(filePath)
	if err != nil {
		s.logger.Warn("提取视频元数据失败",
			zap.String("file", filePath),
			zap.Error(err),
		)
		// 使用默认元数据继续
		metadata = &internalVideoMetadata{
			Format:     formatStr,
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
	}

	// 关联录制任务（直接关联，移除中间的 ConferenceRecord）
	videoFile.TaskID = &task.ID

	// 设置录制时间
	if !task.StartTime.IsZero() {
		videoFile.RecordedAt = &task.StartTime
	}

	// 创建数据库记录（处理并发插入）
	if err := s.db.Create(videoFile).Error; err != nil {
		// 检查是否是唯一约束冲突（并发创建相同文件）
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			// 并发情况下，另一个请求可能已经创建了记录
			// 重新查询并返回现有记录
			s.logger.Debug("并发创建冲突，重新查询现有记录",
				zap.Uint("task_id", task.ID),
				zap.String("file_path", filePath),
			)
			var existingFile models.VideoFile
			if err := s.db.Where("file_path = ?", filePath).First(&existingFile).Error; err == nil {
				return &existingFile, nil
			}
		}
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	s.logger.Info("已创建文件记录",
		zap.Uint("file_id", videoFile.ID),
		zap.Uint("task_id", task.ID),
		zap.String("file_name", videoFile.FileName),
		zap.String("format", formatStr),
	)

	return videoFile, nil
}

// CreateFile 从文件路径创建文件记录（通用方法）
func (s *VideoFileService) CreateFile(filePath string, taskID *uint, recordedAt *time.Time) (*models.VideoFile, error) {
	if filePath == "" {
		return nil, fmt.Errorf("创建文件记录失败：文件路径为空")
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("文件不存在: %s", filePath)
	}

	// 检查是否已存在相同路径的记录
	var existingFile models.VideoFile
	if err := s.db.Where("file_path = ?", filePath).First(&existingFile).Error; err == nil {
		return &existingFile, nil
	}

	// 提取视频元数据
	metadata, err := s.extractVideoMetadata(filePath)
	if err != nil {
		s.logger.Warn("提取视频元数据失败",
			zap.String("file", filePath),
			zap.Error(err),
		)
		metadata = &internalVideoMetadata{
			Format:     "mkv",
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

	// 创建文件记录
	videoFile := &models.VideoFile{
		FileName:           filepath.Base(filePath),
		FilePath:           filePath,
		FileSize:           fileSize,
		Duration:           int(metadata.Duration),
		Format:             metadata.Format,
		Resolution:         metadata.Resolution,
		Bitrate:            metadata.Bitrate,
		Codec:              metadata.Codec,
		TaskID:            taskID,  // 直接关联任务ID
		CreatedBy:          1,       // 设置默认创建者ID为1（admin用户）
		RecordedAt:         recordedAt,
		Status:             models.FileStatusReady,
	}

	// 创建数据库记录（处理并发插入）
	if err := s.db.Create(videoFile).Error; err != nil {
		// 检查是否是唯一约束冲突（并发创建相同文件）
		if strings.Contains(err.Error(), "UNIQUE") || strings.Contains(err.Error(), "duplicate") {
			// 并发情况下，另一个请求可能已经创建了记录
			// 重新查询并返回现有记录
			s.logger.Debug("并发创建冲突，重新查询现有记录",
				zap.String("file_path", filePath),
			)
			var existingFile models.VideoFile
			if err := s.db.Where("file_path = ?", filePath).First(&existingFile).Error; err == nil {
				return &existingFile, nil
			}
		}
		return nil, fmt.Errorf("创建文件记录失败: %w", err)
	}

	s.logger.Info("已创建文件记录",
		zap.Uint("file_id", videoFile.ID),
		zap.String("file_path", filePath),
	)

	return videoFile, nil
}

// ScanFiles 扫描录制目录并导入未入库的文件
func (s *VideoFileService) ScanFiles() (*ScanResult, error) {
	result := &ScanResult{}

	// 使用配置的录制路径（避免硬编码）
	recordingsDir := s.recordingsPath

	// 检查目录是否存在
	if _, err := os.Stat(recordingsDir); os.IsNotExist(err) {
		return result, fmt.Errorf("录制目录不存在: %s", recordingsDir)
	}

	// 查找所有 .mkv 和 .mp4 文件（包含推断的任务ID）
	files, err := s.findVideoFiles(recordingsDir)
	if err != nil {
		return result, fmt.Errorf("查找视频文件失败: %w", err)
	}

	result.Scanned = len(files)

	if len(files) == 0 {
		s.logger.Info("扫描目录为空", zap.String("directory", recordingsDir))
		return result, nil
	}

	s.logger.Info("开始扫描录制文件",
		zap.Int("total_files", len(files)),
		zap.String("directory", recordingsDir),
	)

	// 批量查询已存在的文件路径（性能优化）
	var existingFiles []models.VideoFile
	if err := s.db.Select("file_path").Find(&existingFiles).Error; err != nil {
		return result, fmt.Errorf("批量查询现有文件失败: %w", err)
	}

	// 构建快速查找 map
	existingMap := make(map[string]bool, len(existingFiles))
	for _, f := range existingFiles {
		existingMap[f.FilePath] = true
	}

	s.logger.Debug("批量查询完成",
		zap.Int("existing_count", len(existingFiles)),
		zap.Int("scan_count", len(files)),
	)

	// 处理文件
	for _, file := range files {
		if existingMap[file.filePath] {
			// 文件已存在，检查是否需要更新 task_id
			var existingFile models.VideoFile
			if err := s.db.Where("file_path = ?", file.filePath).First(&existingFile).Error; err == nil {
				// 如果 task_id 为空且可以推断出任务ID，则更新
				if existingFile.TaskID == nil && file.taskID != nil {
					// 验证推断的 task_id 是否存在（数据完整性）
					var taskExists models.VideoRecordingTask
					if err := s.db.Select("id").First(&taskExists, *file.taskID).Error; err == nil {
						updateResult := s.db.Model(&existingFile).Update("task_id", file.taskID)
						if updateResult.Error != nil {
							s.logger.Warn("更新文件记录的task_id失败",
								zap.Uint("file_id", existingFile.ID),
								zap.Uint("task_id", *file.taskID),
								zap.Error(updateResult.Error),
							)
							result.Skipped++
						} else {
							s.logger.Info("更新文件记录的task_id",
								zap.Uint("file_id", existingFile.ID),
								zap.Uint("task_id", *file.taskID),
							)
							result.Skipped++
						}
					} else {
						s.logger.Warn("推断的task_id不存在，跳过更新",
							zap.Uint("inferred_task_id", *file.taskID),
							zap.String("file_path", file.filePath),
						)
						result.Skipped++
					}
				} else {
					result.Skipped++
				}
			} else {
				result.Skipped++
			}
			continue
		}

		// 创建文件记录，使用从路径推断的任务ID
		_, err := s.CreateFile(file.filePath, file.taskID, nil)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("创建记录 %s 失败: %v", file.filePath, err))
			continue
		}

		result.Created++
	}

	s.logger.Info("文件扫描完成",
		zap.Int("scanned", result.Scanned),
		zap.Int("created", result.Created),
		zap.Int("skipped", result.Skipped),
		zap.Int("errors", len(result.Errors)),
	)

	return result, nil
}

// fileInfoWithPath 包含文件路径和推断的任务ID
type fileInfoWithPath struct {
	filePath string
	taskID   *uint
}

// findVideoFiles 查找目录下所有视频文件，并从路径推断任务ID
func (s *VideoFileService) findVideoFiles(dir string) ([]fileInfoWithPath, error) {
	var files []fileInfoWithPath

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".mkv" || ext == ".mp4" {
			// 从文件路径推断任务ID：data/recordings/task_123/video.mkv -> 123
			var taskID *uint
			dir := filepath.Dir(path)
			if matches := filepath.Base(dir); strings.HasPrefix(matches, "task_") {
				idStr := strings.TrimPrefix(matches, "task_")
				if id, err := strconv.Atoi(idStr); err == nil {
					taskIDUint := uint(id)
					taskID = &taskIDUint
				}
			}
			files = append(files, fileInfoWithPath{
				filePath: path,
				taskID:   taskID,
			})
		}

		return nil
	})

	return files, err
}
