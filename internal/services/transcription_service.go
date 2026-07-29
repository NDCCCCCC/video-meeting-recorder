package services

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TranscriptionProgress represents the progress of a transcription task per D-17
type TranscriptionProgress struct {
	Status          string // pending, processing, completed, failed
	CurrentStage    string // extracting, detecting, generating
	FramesProcessed int    // Number of frames processed
	TotalFrames     int    // Total frames to process
	Percentage      int    // Progress percentage (0-100)
	ErrorMessage    string // Error message if failed
	ResultPPTFileID *uint  // ID of the generated PPT file
	Mode            string // local, cloud
}

// TranscriptionService handles video transcription with worker pool
type TranscriptionService struct {
	db                 *gorm.DB
	logger             *zap.Logger
	config             *config.Config
	taskQueue          chan *models.TranscriptionTask
	workers            int
	cancelFuncs        map[uint]context.CancelFunc
	mu                 sync.RWMutex
	wg                 sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
	ffmpegPath         string
	frameExtractor     *FrameExtractor
	similarityDetector *SimilarityDetector
	pptxGenerator      *PPTXGenerator
	videoFileService   *VideoFileService
	ossService         *OSSService
	tingwuClient       *TingwuClient
	statusMap          map[uint]*TranscriptionProgress
	statusMu           sync.RWMutex
}

// NewTranscriptionService creates a new transcription service
func NewTranscriptionService(
	db *gorm.DB,
	logger *zap.Logger,
	cfg *config.Config,
	frameExtractor *FrameExtractor,
	similarityDetector *SimilarityDetector,
	pptxGenerator *PPTXGenerator,
	videoFileService *VideoFileService,
	ossService *OSSService,
	tingwuClient *TingwuClient,
) *TranscriptionService {
	ctx, cancel := context.WithCancel(context.Background())
	ffmpegPath := cfg.FFmpeg.Path
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	return &TranscriptionService{
		db:                 db,
		logger:             logger,
		config:             cfg,
		taskQueue:          make(chan *models.TranscriptionTask, 100),
		workers:            1, // Conservative for memory-intensive transcription
		cancelFuncs:        make(map[uint]context.CancelFunc),
		ctx:                ctx,
		cancel:             cancel,
		ffmpegPath:         ffmpegPath,
		frameExtractor:     frameExtractor,
		similarityDetector: similarityDetector,
		pptxGenerator:      pptxGenerator,
		videoFileService:   videoFileService,
		ossService:         ossService,
		tingwuClient:       tingwuClient,
		statusMap:          make(map[uint]*TranscriptionProgress),
	}
}

// Start starts the worker pool
func (s *TranscriptionService) Start() error {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	s.logger.Info("转录服务启动", zap.Int("workers", s.workers))
	return nil
}

// Stop stops the worker pool
func (s *TranscriptionService) Stop() {
	s.cancel()
	close(s.taskQueue) // Signal workers to exit by closing channel
	s.wg.Wait()
}

// SubmitTranscription submits a transcription task (backward compatible - defaults to local mode)
func (s *TranscriptionService) SubmitTranscription(videoFileID uint, samplingRate float64, createdBy uint) error {
	return s.SubmitTranscriptionWithMode(videoFileID, samplingRate, models.TranscriptionModeLocal, createdBy)
}

// SubmitTranscriptionWithMode submits a transcription task with specified mode (local or cloud)
func (s *TranscriptionService) SubmitTranscriptionWithMode(videoFileID uint, samplingRate float64, mode string, createdBy uint) error {
	// Validate mode
	validModes := map[string]bool{models.TranscriptionModeLocal: true, models.TranscriptionModeCloud: true}
	if !validModes[mode] {
		return fmt.Errorf("无效的转录模式: %s, 必须是 local 或 cloud", mode)
	}

	// For cloud mode, verify cloud services are available
	if mode == models.TranscriptionModeCloud {
		if s.ossService == nil || !s.ossService.IsEnabled() {
			return fmt.Errorf("云端转录不可用: OSS服务未配置")
		}
		if s.ossService.IsStub() {
			return fmt.Errorf("云端转录暂不可用: OSS服务尚未完全集成")
		}
		if s.tingwuClient == nil || !s.tingwuClient.IsEnabled() {
			return fmt.Errorf("云端转录不可用: Tingwu服务未配置")
		}
	}

	// Per D-03: cloud mode skips sampling rate validation entirely
	// Only validate sampling rate for local mode
	// 支持更高精度的采样率选项 (0.05s=20fps, 0.1s=10fps, 0.2s=5fps, 0.5s=2fps, 1.0s=1fps)
	if mode == models.TranscriptionModeLocal {
		validRates := map[float64]bool{1.0: true, 0.5: true, 0.2: true, 0.1: true, 0.05: true}
		if !validRates[samplingRate] {
			return fmt.Errorf("无效的采样率: %.2f (支持: 0.05, 0.1, 0.2, 0.5, 1.0)", samplingRate)
		}
	}

	// Prevent duplicate active tasks for the same video file
	s.statusMu.RLock()
	existing, exists := s.statusMap[videoFileID]
	s.statusMu.RUnlock()
	if exists && existing.Status == models.TranscriptionStatusProcessing {
		return fmt.Errorf("该视频已有正在进行的转录任务")
	}

	// Create task with mode
	task := &models.TranscriptionTask{
		VideoFileID:  videoFileID,
		SamplingRate: samplingRate,
		Status:       models.TranscriptionStatusPending,
		Mode:         mode,
		CreatedBy:    createdBy,
	}
	if err := s.db.Create(task).Error; err != nil {
		return fmt.Errorf("创建转录任务失败: %w", err)
	}

	// Initialize status map
	s.statusMu.Lock()
	s.statusMap[videoFileID] = &TranscriptionProgress{
		Status:       models.TranscriptionStatusProcessing,
		CurrentStage: "",
		Mode:         mode,
	}
	s.statusMu.Unlock()

	// Submit to queue
	select {
	case s.taskQueue <- task:
		s.logger.Info("转录任务已提交",
			zap.Uint("video_file_id", videoFileID),
			zap.String("mode", mode),
			zap.Uint("task_id", task.ID))
		return nil
	default:
		s.statusMu.Lock()
		s.statusMap[videoFileID] = &TranscriptionProgress{
			Status:       models.TranscriptionStatusFailed,
			ErrorMessage: "任务队列已满",
		}
		s.statusMu.Unlock()
		return fmt.Errorf("转录任务队列已满")
	}
}

// GetTranscriptionStatus returns the current transcription progress
func (s *TranscriptionService) GetTranscriptionStatus(videoFileID uint) *TranscriptionProgress {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()

	if progress, ok := s.statusMap[videoFileID]; ok {
		// Return a copy to avoid concurrent access issues
		return &TranscriptionProgress{
			Status:          progress.Status,
			CurrentStage:    progress.CurrentStage,
			FramesProcessed: progress.FramesProcessed,
			TotalFrames:     progress.TotalFrames,
			Percentage:      progress.Percentage,
			ErrorMessage:    progress.ErrorMessage,
			ResultPPTFileID: progress.ResultPPTFileID,
			Mode:            progress.Mode,
		}
	}
	return nil
}

// worker processes transcription tasks from the queue
func (s *TranscriptionService) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case task, ok := <-s.taskQueue:
			if !ok {
				// Channel closed, exit worker gracefully
				s.logger.Info("Worker exiting", zap.Int("worker_id", id))
				return
			}
			s.processTranscription(task)
		case <-s.ctx.Done():
			return
		}
	}
}

// processTranscription processes a single transcription task
func (s *TranscriptionService) processTranscription(task *models.TranscriptionTask) {
	// Defer recover to prevent panics from crashing the worker
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("转录任务panic",
				zap.Uint("task_id", task.ID),
				zap.Uint("video_file_id", task.VideoFileID),
				zap.Any("panic", r))
			s.updateProgress(task.VideoFileID, "", 0, 0, 0, "panic: "+fmt.Sprint(r), nil)
			s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, "panic: "+fmt.Sprint(r), 0, nil)
			// 更新任务组进度
			if task.JobGroupID != nil {
				s.updateJobGroupProgress(*task.JobGroupID)
			}
		}
	}()

	// Reload task to check mode
	if err := s.db.Preload("VideoFile").First(task, task.ID).Error; err != nil {
		s.logger.Error("Failed to load task", zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	// Route to appropriate pipeline based on mode
	if task.Mode == models.TranscriptionModeCloud {
		s.processCloudTranscription(task)
		return
	}

	// Create temp directory per D-05
	tempDir, err := s.frameExtractor.CreateTempDir(s.config.Storage.RecordingsPath, task.VideoFileID)
	if err != nil {
		s.logger.Error("创建临时目录失败",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}
	// Cleanup temp directory per D-05
	defer s.frameExtractor.CleanupTempDir(tempDir)

	// Create cancellable context with timeout
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
	defer cancel()

	// Load video file
	var videoFile models.VideoFile
	if err := s.db.First(&videoFile, task.VideoFileID).Error; err != nil {
		s.logger.Error("加载视频文件失败",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	// Stage 1: Frame Extraction per D-01
	s.updateProgress(task.VideoFileID, models.TranscriptionStageExtracting, 0, 0, 0, "", nil)
	s.updateTaskStatus(task.ID, models.TranscriptionStatusProcessing, "", 0, nil)

	samplingRateSeconds := 1.0 / task.SamplingRate
	frames, err := s.frameExtractor.ExtractFrames(ctx, videoFile.FilePath, tempDir, samplingRateSeconds)
	if err != nil {
		s.logger.Error("帧提取失败",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	s.logger.Info("帧提取完成",
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Int("frame_count", len(frames)))

	// Guard against empty frame extraction (corrupted video, very short clip, etc.)
	if len(frames) == 0 {
		s.logger.Error("帧提取返回空结果",
			zap.Uint("video_file_id", task.VideoFileID))
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, "帧提取返回空结果", nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, "帧提取返回空结果", 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	// Update task with total frames
	s.updateTaskStatus(task.ID, models.TranscriptionStatusProcessing, "", len(frames), nil)

	// Stage 2: Similarity Detection per D-01, D-07
	s.updateProgress(task.VideoFileID, models.TranscriptionStageDetecting, 0, len(frames), 0, "", nil)

	uniqueFrames := make([]ExtractedFrame, 0, len(frames))
	blackFrameCount := 0 // Track black frames for debugging

	// For decoding images
	decodeImage := func(filePath string) (image.Image, error) {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		img, err := jpeg.Decode(file)
		if err != nil {
			return nil, err
		}
		return img, nil
	}

	// Load first frame as previous image
	prevImg, err := decodeImage(frames[0].FilePath)
	if err != nil {
		s.logger.Error("解码首帧失败",
			zap.String("file_path", frames[0].FilePath),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, len(frames), 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		return
	}

	// Check if first frame is black - if so, don't add it to unique frames
	firstFrameResult, err := s.similarityDetector.IsFrameChanged(prevImg, prevImg)
	if err == nil && firstFrameResult.IsBlackFrame {
		blackFrameCount++
		s.logger.Info("首帧为黑色帧，跳过", zap.Int("black_frame_count", blackFrameCount))
	} else {
		uniqueFrames = append(uniqueFrames, frames[0])
	}

	// Process each subsequent frame
	for i := 1; i < len(frames); i++ {
		// Decode current frame
		currImg, err := decodeImage(frames[i].FilePath)
		if err != nil {
			s.logger.Warn("解码帧失败，跳过",
				zap.String("file_path", frames[i].FilePath),
				zap.Error(err))
			continue
		}

		// Check if frame changed using similarity detector
		result, err := s.similarityDetector.IsFrameChanged(prevImg, currImg)
		if err != nil {
			s.logger.Warn("相似度检测失败，跳过",
				zap.Int("frame_index", i),
				zap.Error(err))
			prevImg = currImg
			continue
		}

		// Track black frames
		if result.IsBlackFrame {
			blackFrameCount++
		}

		// If changed AND not a black frame, add to unique frames
		if result.Changed && !result.IsBlackFrame {
			uniqueFrames = append(uniqueFrames, frames[i])
		}

		// Update previous image
		prevImg = currImg

		// Update progress
		percentage := (i + 1) * 100 / len(frames)
		s.updateProgress(task.VideoFileID, models.TranscriptionStageDetecting, i+1, len(frames), percentage, "", nil)
	}

	s.logger.Info("相似度检测完成",
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Int("total_frames", len(frames)),
		zap.Int("unique_frames", len(uniqueFrames)),
		zap.Int("black_frames", blackFrameCount))

	// Record slide timestamps from extracted frames (after similarity detection)
	slideTimestamps := make([]models.SlideTimestamp, 0, len(uniqueFrames))
	for i, frame := range uniqueFrames {
		slideTimestamps = append(slideTimestamps, models.SlideTimestamp{
			SlideNumber: i + 1, // 1-based slide numbers
			Timestamp:   frame.Timestamp,
		})
	}

	// Store timestamps in TranscriptionTask
	if err := task.SetSlideTimestamps(slideTimestamps); err != nil {
		s.logger.Error("设置时间戳失败",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Error(err))
		// Don't fail the task for timestamp recording errors
	} else {
		// Update task in database
		if err := s.db.Model(task).Update("slide_timestamps", task.SlideTimestamps).Error; err != nil {
			s.logger.Warn("保存时间戳到数据库失败",
				zap.Uint("video_file_id", task.VideoFileID),
				zap.Error(err))
		} else {
			s.logger.Info("时间戳已记录",
				zap.Uint("video_file_id", task.VideoFileID),
				zap.Int("timestamp_count", len(slideTimestamps)),
				zap.Float64("first_timestamp", slideTimestamps[0].Timestamp),
				zap.Float64("last_timestamp", slideTimestamps[len(slideTimestamps)-1].Timestamp))
		}
	}

	// Check if video is entirely black or has too few unique frames
	if len(uniqueFrames) == 0 {
		s.logger.Warn("视频无有效内容（全黑或无变化）",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Int("total_frames", len(frames)),
			zap.Int("black_frames", blackFrameCount))
		s.updateProgress(task.VideoFileID, "", len(frames), len(frames), 100, "", nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, "视频无有效内容，无法生成PPT", 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	// Stage 3: PPTX Generation per D-04, D-09
	s.updateProgress(task.VideoFileID, models.TranscriptionStageGenerating, len(frames), len(frames), 90, "", nil)

	// Re-extract unique frames at original resolution per D-04
	highResFramePaths := make([]string, 0, len(uniqueFrames))
	for i, frame := range uniqueFrames {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("highres_%04d.jpg", i))
		if err := s.frameExtractor.ExtractFrameAtTimestamp(ctx, videoFile.FilePath, frame.Timestamp, outputPath); err != nil {
			s.logger.Error("高分辨率帧提取失败",
				zap.Int("frame_index", i),
				zap.Float64("timestamp", frame.Timestamp),
				zap.Error(err))
			s.updateProgress(task.VideoFileID, "", 0, len(frames), 0, err.Error(), nil)
			s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
			// 更新任务组进度
			if task.JobGroupID != nil {
				s.updateJobGroupProgress(*task.JobGroupID)
			}
			return
		}
		// Verify the file was actually created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			s.logger.Error("高分辨率帧文件未创建",
				zap.Int("frame_index", i),
				zap.String("output_path", outputPath))
			continue
		}
		highResFramePaths = append(highResFramePaths, outputPath)
	}

	// Verify we have at least some valid frames
	if len(highResFramePaths) == 0 {
		s.logger.Error("所有高分辨率帧提取失败，无法生成PPT",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Int("unique_frames", len(uniqueFrames)))
		s.updateProgress(task.VideoFileID, "", 0, len(frames), 0, "所有帧提取失败", nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, "所有帧提取失败", 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	s.logger.Info("高分辨率帧提取完成",
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Int("unique_frames", len(uniqueFrames)),
		zap.Int("highres_frames_created", len(highResFramePaths)))

	// Generate PPTX
	timestamp := time.Now().Unix()
	pptxOutputPath := filepath.Join(filepath.Dir(videoFile.FilePath), fmt.Sprintf("transcription_%d_%d.pptx", task.VideoFileID, timestamp))
	pageCount, err := s.pptxGenerator.GeneratePPTX(ctx, highResFramePaths, pptxOutputPath)
	if err != nil {
		s.logger.Error("PPTX生成失败",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, len(frames), 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	s.logger.Info("PPTX生成成功",
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Int("page_count", pageCount),
		zap.String("output_path", pptxOutputPath))

	// Get file info
	fileInfo, err := os.Stat(pptxOutputPath)
	if err != nil {
		s.logger.Error("获取PPTX文件信息失败",
			zap.String("path", pptxOutputPath),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, len(frames), 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		// 更新任务组进度
		if task.JobGroupID != nil {
			s.updateJobGroupProgress(*task.JobGroupID)
		}
		return
	}

	// Create PPTFile record in database
	pptFile := &models.PPTFile{
		FileName:            fmt.Sprintf("%s_转录.pptx", strings.TrimSuffix(videoFile.FileName, filepath.Ext(videoFile.FileName))),
		FilePath:            pptxOutputPath,
		FileSize:            fileInfo.Size(),
		PageCount:           pageCount,
		Format:              "pptx",
		SourceVideoFileID:   &task.VideoFileID,
		TranscriptionTaskID: &task.ID,
	}
	if err := s.db.Create(pptFile).Error; err != nil {
		s.logger.Error("创建PPT文件记录失败",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, len(frames), 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		return
	}

	// 更新任务组进度
	if task.JobGroupID != nil {
		s.updateJobGroupProgress(*task.JobGroupID)
	}
	// Update task as completed
	s.updateProgress(task.VideoFileID, "", len(frames), len(frames), 100, "", &pptFile.ID)
	s.updateTaskStatus(task.ID, models.TranscriptionStatusCompleted, "", 0, &pptFile.ID)

	// 更新任务组进度
	if task.JobGroupID != nil {
		s.updateJobGroupProgress(*task.JobGroupID)
	}
	s.logger.Info("转录任务完成",
		zap.Uint("task_id", task.ID),
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Uint("ppt_file_id", pptFile.ID),
		zap.Int("pages_generated", pageCount))
}

// updateProgress updates the progress map for a video file
func (s *TranscriptionService) updateProgress(
	videoFileID uint,
	stage string,
	processed int,
	total int,
	percentage int,
	errorMsg string,
	resultPPTFileID *uint,
) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	// Get or create progress entry atomically
	progress, ok := s.statusMap[videoFileID]
	if !ok {
		progress = &TranscriptionProgress{
			Status: models.TranscriptionStatusProcessing,
		}
		s.statusMap[videoFileID] = progress
	}

	// Now update all fields atomically while holding lock
	if stage != "" {
		progress.CurrentStage = stage
	}
	if processed > 0 {
		progress.FramesProcessed = processed
	}
	if total > 0 {
		progress.TotalFrames = total
	}
	if percentage > 0 {
		progress.Percentage = percentage
	}
	if errorMsg != "" {
		// If this is a cloud fallback message, don't set status to failed
		// The local pipeline will override this with normal progress
		if !strings.HasPrefix(errorMsg, "cloud_fallback:") {
			progress.Status = models.TranscriptionStatusFailed
		}
		progress.ErrorMessage = strings.TrimPrefix(errorMsg, "cloud_fallback:")
	}
	if resultPPTFileID != nil {
		progress.Status = models.TranscriptionStatusCompleted
		progress.ResultPPTFileID = resultPPTFileID
	}
}

// updateTaskStatus updates the task status in database
func (s *TranscriptionService) updateTaskStatus(
	taskID uint,
	status string,
	errorMsg string,
	totalFrames int,
	resultPPTFileID *uint,
) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	if totalFrames > 0 {
		updates["total_frames"] = totalFrames
	}
	if resultPPTFileID != nil {
		updates["result_ppt_file_id"] = resultPPTFileID
		updates["percentage"] = 100
	}

	result := s.db.Model(&models.TranscriptionTask{}).Where("id = ?", taskID).Updates(updates)
	if result.Error != nil {
		s.logger.Error("更新转录任务状态失败",
			zap.Uint("task_id", taskID),
			zap.Error(result.Error))
		return result.Error
	}
	return nil
}

// processCloudTranscription handles cloud transcription pipeline (OSS upload, Tingwu submit, polling, result retrieval)
func (s *TranscriptionService) processCloudTranscription(task *models.TranscriptionTask) {
	// Load video file
	var videoFile models.VideoFile
	if err := s.db.First(&videoFile, task.VideoFileID).Error; err != nil {
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
	defer cancel()

	// Stage 1: Upload to OSS (0-20%)
	s.updateProgress(task.VideoFileID, models.TranscriptionStageUploading, 0, 0, 5, "", nil)
	s.updateTaskStatus(task.ID, models.TranscriptionStatusProcessing, "", 0, nil)

	objectKey := fmt.Sprintf("transcriptions/%d/%d.mp4", task.VideoFileID, task.ID)
	ossURL, err := s.ossService.UploadFile(ctx, videoFile.FilePath, objectKey)
	if err != nil {
		s.handleCloudFailure(task, err, true) // Auto-fallback per D-07
		return
	}

	// Save OSS URL
	s.db.Model(task).Updates(map[string]interface{}{
		"oss_url": ossURL,
	})

	s.updateProgress(task.VideoFileID, models.TranscriptionStageUploading, 0, 0, 20, "", nil)

	// Stage 2: Submit to Tingwu (20%)
	s.updateProgress(task.VideoFileID, models.TranscriptionStageQueued, 0, 0, 25, "", nil)

	cloudTaskID, err := s.tingwuClient.SubmitTask(ctx, ossURL)
	if err != nil {
		s.handleCloudFailure(task, err, true) // Auto-fallback per D-07
		return
	}

	// Save CloudTaskID
	s.db.Model(task).Update("cloud_task_id", cloudTaskID)

	// Stage 3: Poll Tingwu status with exponential backoff (25-90%)
	if !s.pollTingwuStatus(ctx, task, cloudTaskID) {
		return // Failure already handled by handleCloudFailure
	}

	// Stage 4: Download results (90-100%)
	s.updateProgress(task.VideoFileID, models.TranscriptionStageDownloading, 0, 0, 90, "", nil)

	result, err := s.tingwuClient.GetResult(ctx, cloudTaskID)
	if err != nil {
		s.handleCloudFailure(task, err, false) // No auto-fallback per D-07
		return
	}

	// Save text content to TranscriptionText table
	if err := s.saveTextContent(task, result); err != nil {
		s.logger.Error("保存文字内容失败", zap.Error(err))
		// Don't fail the task for text save errors -- text is optional
	}

	// Trigger OSS lifecycle cleanup (24h) per OSS-02
	go func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ruleID := fmt.Sprintf("expire-transcription-%d", task.ID)
		if err := s.ossService.SetLifecycleRule(cleanupCtx, ruleID,
			fmt.Sprintf("transcriptions/%d/", task.VideoFileID), 1); err != nil {
			s.logger.Warn("设置OSS清理规则失败", zap.Error(err))
			// Fallback: attempt immediate deletion of this specific file
			if delErr := s.ossService.DeleteFile(cleanupCtx, objectKey); delErr != nil {
				s.logger.Error("OSS文件删除也失败，文件可能成为孤儿文件",
					zap.String("object_key", objectKey),
					zap.Error(delErr))
			} else {
				s.logger.Info("OSS生命周期规则失败，已直接删除文件",
					zap.String("object_key", objectKey))
			}
		}
	}()

	// Mark completed (cloud transcription does NOT generate PPT -- text is the output)
	s.updateProgress(task.VideoFileID, "", 0, 0, 100, "", nil)
	s.updateTaskStatus(task.ID, models.TranscriptionStatusCompleted, "", 0, nil)

	s.logger.Info("云端转录完成",
		zap.Uint("task_id", task.ID),
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Int("segments", len(result.Segments)))
}

// pollTingwuStatus polls Tingwu task status with exponential backoff (TRAN-04)
// Returns true if Tingwu reported Completed, false if polling failed (failure already handled by handleCloudFailure).
func (s *TranscriptionService) pollTingwuStatus(ctx context.Context, task *models.TranscriptionTask, cloudTaskID string) bool {
	delay := 2 * time.Second
	maxDelay := 60 * time.Second // TRAN-04: max delay for exponential backoff
	maxAttempts := 120

	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			s.handleCloudFailure(task, ctx.Err(), false)
			return false
		case <-time.After(delay):
		}

		status, err := s.tingwuClient.GetStatus(ctx, cloudTaskID)
		if err != nil {
			s.logger.Warn("查询Tingwu状态失败，重试",
				zap.String("cloud_task_id", cloudTaskID),
				zap.Error(err))
			// Exponential backoff with jitter: TRAN-04
			delay = time.Duration(float64(delay) * 1.5)
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		switch status.Status {
		case "Queued":
			s.updateProgress(task.VideoFileID, models.TranscriptionStageQueued, 0, 0, 30, "", nil)
			delay = 5 * time.Second
		case "Processing":
			// Map Tingwu progress (0-100) to our percentage (30-90)
			percentage := 30 + int(status.Progress*0.6)
			if percentage > 89 {
				percentage = 89
			}
			s.updateProgress(task.VideoFileID, models.TranscriptionStageProcessing, 0, 0, percentage, "", nil)
			delay = 10 * time.Second
		case "Completed":
			return true // Done!
		case "Failed":
			s.handleCloudFailure(task,
				fmt.Errorf("Tingwu处理失败: %s", status.ErrorMessage), false)
			return false
		default:
			s.logger.Warn("未知Tingwu状态",
				zap.String("status", status.Status))
			delay = 10 * time.Second
		}
	}

	// Exhausted attempts
	s.handleCloudFailure(task, fmt.Errorf("轮询超时: 超过最大尝试次数"), false)
	return false
}

// handleCloudFailure handles cloud transcription failures with auto-fallback logic (per D-07, D-08)
func (s *TranscriptionService) handleCloudFailure(task *models.TranscriptionTask, cloudErr error, isInitialStage bool) {
	s.logger.Error("云端转录失败",
		zap.Uint("task_id", task.ID),
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Bool("initial_stage", isInitialStage),
		zap.Error(cloudErr))

	if isInitialStage {
		// Auto-fallback to local transcription per D-07
		s.logger.Info("自动切换到本地转录",
			zap.Uint("video_file_id", task.VideoFileID))

		// Update task mode atomically
		s.db.Model(task).Updates(map[string]interface{}{
			"mode":          models.TranscriptionModeLocal,
			"cloud_task_id": "",
			"oss_url":       "",
		})
		task.Mode = models.TranscriptionModeLocal
		task.CloudTaskID = ""
		task.OSSURL = ""

		// Reset progress for local mode
		s.statusMu.Lock()
		s.statusMap[task.VideoFileID] = &TranscriptionProgress{
			Status:       models.TranscriptionStatusProcessing,
			CurrentStage: "",
			Mode:         models.TranscriptionModeLocal,
			ErrorMessage: "cloud_fallback:" + cloudErr.Error(), // Prefix so frontend can detect fallback
		}
		s.statusMu.Unlock()

		// Start local transcription
		s.processTranscription(task)
	} else {
		// Mid-processing failure: mark as failed, no auto-fallback per D-07
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, cloudErr.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, cloudErr.Error(), 0, nil)
	}
}

// saveTextContent saves Tingwu transcription result to TranscriptionText table
func (s *TranscriptionService) saveTextContent(task *models.TranscriptionTask, result *TingwuTaskResult) error {
	if result == nil || len(result.Segments) == 0 {
		s.logger.Info("没有文字内容需要保存")
		return nil
	}

	// Delete any existing text content for this task (idempotent)
	s.db.Where("transcription_task_id = ?", task.ID).Delete(&models.TranscriptionText{})

	// Create text segment records
	texts := make([]models.TranscriptionText, 0, len(result.Segments))
	for i, segment := range result.Segments {
		texts = append(texts, models.TranscriptionText{
			TranscriptionTaskID: task.ID,
			Text:                segment.Text,
			BeginTime:           segment.BeginTime,
			EndTime:             segment.EndTime,
			SegmentIndex:        i,
		})
	}

	if err := s.db.Create(&texts).Error; err != nil {
		return fmt.Errorf("保存文字内容失败: %w", err)
	}

	s.logger.Info("文字内容已保存",
		zap.Uint("task_id", task.ID),
		zap.Int("segments", len(texts)))

	return nil
}

// StartOSSCleanupScheduler starts a periodic cleanup of orphaned OSS files
func (s *TranscriptionService) StartOSSCleanupScheduler() {
	if s.ossService == nil || !s.ossService.IsEnabled() {
		return
	}

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.cleanupOrphanedOSSFiles()
			}
		}
	}()

	s.logger.Info("OSS定期清理调度器已启动")
}

// cleanupOrphanedOSSFiles deletes OSS files for completed/failed tasks older than 24 hours
func (s *TranscriptionService) cleanupOrphanedOSSFiles() {
	cutoff := time.Now().Add(-24 * time.Hour)

	var tasks []models.TranscriptionTask
	if err := s.db.Where(
		"mode = ? AND status IN ? AND oss_url != '' AND updated_at < ?",
		models.TranscriptionModeCloud,
		[]string{models.TranscriptionStatusCompleted, models.TranscriptionStatusFailed},
		cutoff,
	).Find(&tasks).Error; err != nil {
		s.logger.Error("查询待清理OSS文件失败", zap.Error(err))
		return
	}

	for _, task := range tasks {
		if task.OSSURL == "" {
			continue
		}

		objectKey := fmt.Sprintf("transcriptions/%d/%d.mp4", task.VideoFileID, task.ID)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := s.ossService.DeleteFile(ctx, objectKey); err != nil {
			s.logger.Warn("清理OSS文件失败",
				zap.String("object_key", objectKey),
				zap.Error(err))
		} else {
			// Clear the OSSURL to avoid re-processing
			s.db.Model(&task).Update("oss_url", "")
			s.logger.Info("已清理过期OSS文件",
				zap.Uint("task_id", task.ID),
				zap.String("object_key", objectKey))
		}
		cancel()
	}
}

// GetDB returns the database instance for handlers to use in queries
func (s *TranscriptionService) GetDB() *gorm.DB {
	return s.db
}

// GetActiveTasks returns all active transcription tasks (pending or processing)
func (s *TranscriptionService) GetActiveTasks() ([]models.TranscriptionTask, error) {
	var tasks []models.TranscriptionTask
	err := s.db.Where("status IN ?", []string{models.TranscriptionStatusPending, models.TranscriptionStatusProcessing}).
		Order("created_at DESC").
		Find(&tasks).Error
	return tasks, err
}

// BatchTranscriptionRequest 批量转录请求
type BatchTranscriptionRequest struct {
	VideoFileIDs []uint  `json:"video_file_ids"`
	SamplingRate float64 `json:"sampling_rate"`
	Mode         string  `json:"mode"`
	UserID       uint    `json:"-"`
}

// BatchTranscriptionResult 批量转录结果
type BatchTranscriptionResult struct {
	JobGroupID     uint     `json:"job_group_id"`
	TotalCount     int      `json:"total_count"`
	SubmittedCount int      `json:"submitted_count"`
	FailedCount    int      `json:"failed_count"`
	Errors         []string `json:"errors"`
}

// SubmitBatchTranscription 批量提交转录任务
func (s *TranscriptionService) SubmitBatchTranscription(req *BatchTranscriptionRequest) (*BatchTranscriptionResult, error) {
	// 验证转录模式
	if req.Mode != models.TranscriptionModeLocal && req.Mode != models.TranscriptionModeCloud {
		return nil, fmt.Errorf("无效的转录模式: %s", req.Mode)
	}

	// 对于本地转录，验证采样率
	if req.Mode == models.TranscriptionModeLocal {
		validRates := map[float64]bool{1.0: true, 0.5: true, 0.2: true, 0.1: true, 0.05: true}
		if req.SamplingRate == 0 {
			req.SamplingRate = 0.5 // 默认值
		}
		if !validRates[req.SamplingRate] {
			return nil, fmt.Errorf("无效的采样率: %.2f (支持: 0.05, 0.1, 0.2, 0.5, 1.0)", req.SamplingRate)
		}
	}

	// 创建任务组
	jobGroup := &models.TranscriptionJobGroup{
		UserID:     req.UserID,
		Status:     models.JobGroupStatusPending,
		TotalCount: len(req.VideoFileIDs),
	}
	if err := s.db.Create(jobGroup).Error; err != nil {
		return nil, fmt.Errorf("创建任务组失败: %w", err)
	}

	// 结果统计
	result := &BatchTranscriptionResult{
		JobGroupID: jobGroup.ID,
		TotalCount: len(req.VideoFileIDs),
		Errors:     make([]string, 0),
	}

	// 顺序创建任务
	for i, videoFileID := range req.VideoFileIDs {
		// 创建转录任务
		task := &models.TranscriptionTask{
			VideoFileID:  videoFileID,
			JobGroupID:   &jobGroup.ID,
			SamplingRate: req.SamplingRate,
			Mode:         req.Mode,
			Status:       models.TranscriptionStatusPending,
			CreatedBy:    req.UserID,
		}

		if err := s.db.Create(task).Error; err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("文件 %d 创建任务失败: %v", videoFileID, err))
			s.logger.Warn("批量转录创建任务失败",
				zap.Uint("video_file_id", videoFileID),
				zap.Error(err),
			)
			continue
		}

		// 提交到队列
		select {
		case s.taskQueue <- task:
			result.SubmittedCount++
			s.logger.Info("批量转录任务已提交",
				zap.Uint("job_group_id", jobGroup.ID),
				zap.Uint("task_id", task.ID),
				zap.Int("index", i+1),
				zap.Int("total", len(req.VideoFileIDs)),
			)
		default:
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Sprintf("文件 %d 队列已满", videoFileID))
			s.logger.Warn("批量转录队列已满",
				zap.Uint("video_file_id", videoFileID),
			)
		}
	}

	// 更新任务组状态
	if result.SubmittedCount > 0 {
		jobGroup.Status = models.JobGroupStatusProcessing
		jobGroup.CompletedCount = result.SubmittedCount
		jobGroup.FailedCount = result.FailedCount
		jobGroup.UpdateStatus()
		s.db.Save(jobGroup)
	}

	return result, nil
}

// GetJobGroupStatus 获取批量转录任务组状态
func (s *TranscriptionService) GetJobGroupStatus(jobGroupID uint, userID uint, isAdmin bool) (*models.TranscriptionJobGroup, error) {
	var jobGroup models.TranscriptionJobGroup
	err := s.db.Where("id = ?", jobGroupID).
		Preload("Tasks").
		First(&jobGroup).Error
	if err != nil {
		return nil, err
	}

	// 验证权限
	if !isAdmin && jobGroup.UserID != userID {
		return nil, fmt.Errorf("无权访问此任务组")
	}

	// 重新计算进度
	var completedCount, failedCount int
	for _, task := range jobGroup.Tasks {
		if task.Status == models.TranscriptionStatusCompleted {
			completedCount++
		} else if task.Status == models.TranscriptionStatusFailed {
			failedCount++
		}
	}
	jobGroup.CompletedCount = completedCount
	jobGroup.FailedCount = failedCount
	jobGroup.UpdateStatus()
	s.db.Save(&jobGroup)

	return &jobGroup, nil
}

// updateJobGroupProgress 更新任务组进度
func (s *TranscriptionService) updateJobGroupProgress(jobGroupID uint) {
	var jobGroup models.TranscriptionJobGroup
	if err := s.db.Preload("Tasks").First(&jobGroup, jobGroupID).Error; err != nil {
		s.logger.Error("更新任务组进度失败", zap.Uint("job_group_id", jobGroupID), zap.Error(err))
		return
	}

	var completedCount, failedCount int
	for _, task := range jobGroup.Tasks {
		if task.Status == models.TranscriptionStatusCompleted {
			completedCount++
		} else if task.Status == models.TranscriptionStatusFailed {
			failedCount++
		}
	}

	jobGroup.CompletedCount = completedCount
	jobGroup.FailedCount = failedCount
	jobGroup.UpdateStatus()
	s.db.Save(&jobGroup)
}
