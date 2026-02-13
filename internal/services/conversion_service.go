package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ConversionService 转换服务接口
type ConversionService interface {
	// SubmitConversion 提交转换任务
	SubmitConversion(taskID uint) error

	// GetConversionStatus 获取转换状态
	GetConversionStatus(taskID uint) (models.ConversionStatus, error)

	// RetryConversion 重试失败任务
	RetryConversion(taskID uint) error

	// Start 启动服务
	Start() error

	// Stop 停止服务
	Stop()
}

// ConversionTask 转换任务
type ConversionTask struct {
	TaskID uint
	Status models.ConversionStatus
}

// FFmpegConversionService FFmpeg转换服务实现
type FFmpegConversionService struct {
	db               *gorm.DB
	logger           *zap.Logger
	config           *config.Config
	taskQueue        chan uint
	workers          int
	maxRetries       int
	cancelFuncs      map[uint]context.CancelFunc
	mu               sync.RWMutex
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	ffmpegPath       string
	videoFileService *VideoFileService // 视频文件服务
}

// NewFFmpegConversionService 创建转换服务
func NewFFmpegConversionService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, videoFileService *VideoFileService) ConversionService {
	return &FFmpegConversionService{
		db:               db,
		logger:           logger,
		config:           cfg,
		taskQueue:        make(chan uint, 100), // 缓冲队列
		workers:          3,                     // 默认3个worker
		maxRetries:       3,                     // 最大重试次数
		cancelFuncs:      make(map[uint]context.CancelFunc),
		ffmpegPath:       cfg.FFmpeg.Path,
		videoFileService: videoFileService,
	}
}

// Start 启动转换服务
func (s *FFmpegConversionService) Start() error {
	s.logger.Info("正在启动FFmpeg转换服务",
		zap.Int("workers", s.workers),
		zap.Int("max_retries", s.maxRetries),
	)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	// 启动worker
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	// 加载待转换的任务
	if err := s.loadPendingTasks(); err != nil {
		s.logger.Error("加载待转换任务失败", zap.Error(err))
	}

	s.logger.Info("FFmpeg转换服务启动成功")
	return nil
}

// Stop 停止转换服务
func (s *FFmpegConversionService) Stop() {
	s.logger.Info("正在停止FFmpeg转换服务")

	s.cancel()

	// 关闭队列
	close(s.taskQueue)

	// 等待所有worker完成
	s.wg.Wait()

	// 取消所有正在进行的转换
	s.mu.Lock()
	for taskID, cancel := range s.cancelFuncs {
		cancel()
		s.logger.Debug("已取消转换任务", zap.Uint("task_id", taskID))
	}
	s.cancelFuncs = make(map[uint]context.CancelFunc)
	s.mu.Unlock()

	s.logger.Info("FFmpeg转换服务已停止")
}

// SubmitConversion 提交转换任务
func (s *FFmpegConversionService) SubmitConversion(taskID uint) error {
	// 加载任务
	var task models.VideoRecordingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 检查MKV文件是否存在
	if task.MKVFilePath == "" {
		return fmt.Errorf("任务没有MKV文件")
	}
	if _, err := os.Stat(task.MKVFilePath); os.IsNotExist(err) {
		return fmt.Errorf("MKV文件不存在: %s", task.MKVFilePath)
	}

	// 检查当前状态
	if task.ConversionStatus == models.ConversionStatusProcessing {
		return fmt.Errorf("任务正在转换中")
	}
	if task.ConversionStatus == models.ConversionStatusCompleted {
		return fmt.Errorf("任务已完成转换")
	}

	// 更新状态为pending
	updates := map[string]interface{}{
		"conversion_status":       models.ConversionStatusPending,
		"conversion_error_msg":     "",
		"conversion_retry_count":   0,
	}
	if err := s.db.Model(&task).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 提交到队列
	select {
	case s.taskQueue <- taskID:
		s.logger.Info("转换任务已提交",
			zap.Uint("task_id", taskID),
			zap.String("mkv_file", task.MKVFilePath),
		)
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("服务已停止")
	default:
		return fmt.Errorf("队列已满")
	}
}

// GetConversionStatus 获取转换状态
func (s *FFmpegConversionService) GetConversionStatus(taskID uint) (models.ConversionStatus, error) {
	var task models.VideoRecordingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return "", err
	}
	return task.ConversionStatus, nil
}

// RetryConversion 重试失败任务
func (s *FFmpegConversionService) RetryConversion(taskID uint) error {
	// 重置重试计数并重新提交
	updates := map[string]interface{}{
		"conversion_status":     models.ConversionStatusPending,
		"conversion_error_msg":  "",
		"conversion_retry_count": 0,
	}
	if err := s.db.Model(&models.VideoRecordingTask{}).Where("id = ?", taskID).Updates(updates).Error; err != nil {
		return err
	}

	// 提交到队列
	select {
	case s.taskQueue <- taskID:
		return nil
	case <-s.ctx.Done():
		return fmt.Errorf("服务已停止")
	default:
		return fmt.Errorf("队列已满")
	}
}

// worker 处理转换任务的worker
func (s *FFmpegConversionService) worker(id int) {
	defer s.wg.Done()

	s.logger.Debug("转换worker已启动", zap.Int("worker_id", id))

	for {
		select {
		case taskID, ok := <-s.taskQueue:
			if !ok {
				s.logger.Debug("转换worker正在停止", zap.Int("worker_id", id))
				return
			}
			s.processTask(taskID)

		case <-s.ctx.Done():
			s.logger.Debug("转换worker停止中", zap.Int("worker_id", id))
			return
		}
	}
}

// processTask 处理单个转换任务
func (s *FFmpegConversionService) processTask(taskID uint) {
	s.logger.Info("正在处理转换任务", zap.Uint("task_id", taskID))

	// 加载任务
	var task models.VideoRecordingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		s.logger.Error("加载任务失败", zap.Uint("task_id", taskID), zap.Error(err))
		return
	}

	// 创建可取消的context
	ctx, cancel := context.WithCancel(s.ctx)
	s.mu.Lock()
	s.cancelFuncs[taskID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.cancelFuncs, taskID)
		s.mu.Unlock()
		cancel()
	}()

	// 更新状态为processing
	now := time.Now()
	updates := map[string]interface{}{
		"conversion_status":      models.ConversionStatusProcessing,
		"conversion_started_at":  &now,
		"conversion_error_msg":    "",
	}
	s.db.Model(&task).Updates(updates)

	// 执行转换
	outputPath, err := s.convertMKVToMP4(ctx, &task)

	if err != nil {
		s.handleConversionError(&task, err)
		return
	}

	// 转换成功
	now = time.Now()
	s.db.Model(&task).Updates(map[string]interface{}{
		"conversion_status":       models.ConversionStatusCompleted,
		"conversion_completed_at": &now,
		"conversion_error_msg":     "",
		"mp4_file_path":            outputPath,
	})

	// 创建 MP4 文件记录
	if s.videoFileService != nil {
		mp4 := "mp4"
		if _, err := s.videoFileService.CreateFileFromTask(&task, &mp4); err != nil {
			s.logger.Error("创建MP4文件记录失败",
				zap.Uint("task_id", taskID),
				zap.Error(err),
			)
		}
	}

	s.logger.Info("转换完成",
		zap.Uint("task_id", taskID),
		zap.String("mp4_file", outputPath),
	)
}

// convertMKVToMP4 执行MKV到MP4的转换
func (s *FFmpegConversionService) convertMKVToMP4(ctx context.Context, task *models.VideoRecordingTask) (string, error) {
	if task.MKVFilePath == "" {
		return "", fmt.Errorf("MKV文件路径为空")
	}

	// 生成输出路径
	inputPath := task.MKVFilePath
	outputPath := s.generateMP4Path(inputPath)

	// 确保输出目录存在
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 构建FFmpeg命令
	args := []string{
		"-y", // 覆盖输出文件
		"-i", inputPath,
		"-c:v", "copy", // 视频直接复制（不重新编码）
		"-c:a", "aac",  // 音频转AAC
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	}

	s.logger.Debug("FFmpeg转换命令",
		zap.Uint("task_id", task.ID),
		zap.String("ffmpeg", s.ffmpegPath),
		zap.Any("args", args),
	)

	// 执行转换，捕获输出用于错误诊断
	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	s.logger.Info("正在执行FFmpeg转换",
		zap.Uint("task_id", task.ID),
		zap.String("input", inputPath),
		zap.String("output", outputPath),
	)

	if err := cmd.Run(); err != nil {
		// 记录详细的错误信息
		s.logger.Error("FFmpeg转换失败",
			zap.Uint("task_id", task.ID),
			zap.String("input", inputPath),
			zap.String("output", outputPath),
			zap.Error(err),
			zap.String("stderr", stderr.String()),
			zap.String("stdout", stdout.String()),
		)
		return "", fmt.Errorf("FFmpeg转换失败: %w, stderr: %s", err, stderr.String())
	}

	// 验证输出文件
	if info, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("输出文件验证失败: %w", err)
	} else if info.Size() == 0 {
		return "", fmt.Errorf("输出文件为空")
	}

	return outputPath, nil
}

// generateMP4Path 生成MP4输出路径
func (s *FFmpegConversionService) generateMP4Path(mkvPath string) string {
	// 将.mkv替换为.mp4
	return mkvPath[:len(mkvPath)-4] + ".mp4"
}

// handleConversionError 处理转换错误
func (s *FFmpegConversionService) handleConversionError(task *models.VideoRecordingTask, err error) {
	task.ConversionRetryCount++

	// 检查是否超过最大重试次数
	if task.ConversionRetryCount >= s.maxRetries {
		// 标记为失败
		s.db.Model(task).Updates(map[string]interface{}{
			"conversion_status":  models.ConversionStatusFailed,
			"conversion_error_msg": err.Error(),
		})
		s.logger.Error("转换失败，已达最大重试次数",
			zap.Uint("task_id", task.ID),
			zap.Int("retry_count", task.ConversionRetryCount),
			zap.Error(err),
		)
		return
	}

	// 计算退避时间
	backoffDuration := s.calculateBackoff(task.ConversionRetryCount)

	s.logger.Warn("转换失败，安排重试",
		zap.Uint("task_id", task.ID),
		zap.Int("retry_count", task.ConversionRetryCount),
		zap.Duration("backoff", backoffDuration),
		zap.Error(err),
	)

	// 保存错误信息
	s.db.Model(task).Updates(map[string]interface{}{
		"conversion_error_msg": err.Error(),
	})

	// 安排重试
	go func() {
		select {
		case <-time.After(backoffDuration):
			select {
			case s.taskQueue <- task.ID:
				s.logger.Info("重试已安排",
					zap.Uint("task_id", task.ID),
					zap.Int("attempt", task.ConversionRetryCount),
				)
			case <-s.ctx.Done():
				return
			}
		case <-s.ctx.Done():
			return
		}
	}()
}

// calculateBackoff 计算退避时间（指数退避）
func (s *FFmpegConversionService) calculateBackoff(retryCount int) time.Duration {
	// 1分钟, 5分钟, 30分钟
	backoffs := []time.Duration{
		1 * time.Minute,
		5 * time.Minute,
		30 * time.Minute,
	}

	idx := retryCount - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(backoffs) {
		idx = len(backoffs) - 1
	}

	return backoffs[idx]
}

// loadPendingTasks 加载待转换的任务
func (s *FFmpegConversionService) loadPendingTasks() error {
	var tasks []models.VideoRecordingTask

	// 查找所有有MKV文件但未完成转换的任务
	if err := s.db.Where("mkv_file_path != ? AND (conversion_status = ? OR conversion_status = ? OR conversion_status = '' OR conversion_status IS NULL)",
		"",
		models.ConversionStatusPending,
		models.ConversionStatusFailed,
	).Find(&tasks).Error; err != nil {
		return err
	}

	s.logger.Info("发现待转换任务", zap.Int("count", len(tasks)))

	// 提交到队列
	for _, task := range tasks {
		select {
		case s.taskQueue <- task.ID:
		default:
			s.logger.Warn("任务队列已满，部分任务未加载")
			return nil
		}
	}

	return nil
}
