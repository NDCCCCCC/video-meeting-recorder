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
	Status          string  // pending, processing, completed, failed
	CurrentStage    string  // extracting, detecting, generating
	FramesProcessed int     // Number of frames processed
	TotalFrames     int     // Total frames to process
	Percentage      int     // Progress percentage (0-100)
	ErrorMessage    string  // Error message if failed
	ResultPPTFileID *uint   // ID of the generated PPT file
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
	s.wg.Wait()
}

// SubmitTranscription submits a transcription task
func (s *TranscriptionService) SubmitTranscription(videoFileID uint, samplingRate float64, createdBy uint) error {
	// Validate video file exists
	var videoFile models.VideoFile
	if err := s.db.First(&videoFile, videoFileID).Error; err != nil {
		return fmt.Errorf("视频文件不存在: ID=%d", videoFileID)
	}

	// Validate sampling rate (per D-02: 1s/2s/5s intervals = 1.0/0.5/0.2 fps)
	validRates := map[float64]bool{1.0: true, 0.5: true, 0.2: true}
	if !validRates[samplingRate] {
		return fmt.Errorf("无效的采样率: %.1f, 必须是 1.0, 0.5 或 0.2", samplingRate)
	}

	// Create transcription task in database
	task := &models.TranscriptionTask{
		VideoFileID:  videoFileID,
		SamplingRate: samplingRate,
		Status:       models.TranscriptionStatusPending,
		CurrentStage: "",
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
	}
	s.statusMu.Unlock()

	// Submit to queue
	select {
	case s.taskQueue <- task:
		s.logger.Info("转录任务已提交",
			zap.Uint("video_file_id", videoFileID),
			zap.Uint("task_id", task.ID),
			zap.Float64("sampling_rate", samplingRate))
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
			Status:           progress.Status,
			CurrentStage:     progress.CurrentStage,
			FramesProcessed:  progress.FramesProcessed,
			TotalFrames:      progress.TotalFrames,
			Percentage:       progress.Percentage,
			ErrorMessage:     progress.ErrorMessage,
			ResultPPTFileID:  progress.ResultPPTFileID,
		}
	}
	return nil
}

// worker processes transcription tasks from the queue
func (s *TranscriptionService) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case task := <-s.taskQueue:
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
		}
	}()

	// Create temp directory per D-05
	tempDir, err := s.frameExtractor.CreateTempDir(s.config.Storage.RecordingsPath, task.VideoFileID)
	if err != nil {
		s.logger.Error("创建临时目录失败",
			zap.Uint("video_file_id", task.VideoFileID),
			zap.Error(err))
		s.updateProgress(task.VideoFileID, "", 0, 0, 0, err.Error(), nil)
		s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
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
		return
	}

	s.logger.Info("帧提取完成",
		zap.Uint("video_file_id", task.VideoFileID),
		zap.Int("frame_count", len(frames)))

	// Update task with total frames
	s.updateTaskStatus(task.ID, models.TranscriptionStatusProcessing, "", len(frames), nil)

	// Stage 2: Similarity Detection per D-01, D-07
	s.updateProgress(task.VideoFileID, models.TranscriptionStageDetecting, 0, len(frames), 0, "", nil)

	uniqueFrames := make([]ExtractedFrame, 0, len(frames))
	uniqueFrames = append(uniqueFrames, frames[0]) // Keep first frame as reference

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

		// If changed, add to unique frames
		if result.Changed {
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
		zap.Int("unique_frames", len(uniqueFrames)))

	// Stage 3: PPTX Generation per D-04, D-09
	s.updateProgress(task.VideoFileID, models.TranscriptionStageGenerating, len(frames), len(frames), 90, "", nil)

	// Re-extract unique frames at original resolution per D-04
	highResFramePaths := make([]string, len(uniqueFrames))
	for i, frame := range uniqueFrames {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("highres_%04d.jpg", i))
		if err := s.frameExtractor.ExtractFrameAtTimestamp(ctx, videoFile.FilePath, frame.Timestamp, outputPath); err != nil {
			s.logger.Error("高分辨率帧提取失败",
				zap.Int("frame_index", i),
				zap.Float64("timestamp", frame.Timestamp),
				zap.Error(err))
			s.updateProgress(task.VideoFileID, "", 0, len(frames), 0, err.Error(), nil)
			s.updateTaskStatus(task.ID, models.TranscriptionStatusFailed, err.Error(), 0, nil)
			return
		}
		highResFramePaths[i] = outputPath
	}

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

	// Update task as completed
	s.updateProgress(task.VideoFileID, "", len(frames), len(frames), 100, "", &pptFile.ID)
	s.updateTaskStatus(task.ID, models.TranscriptionStatusCompleted, "", 0, &pptFile.ID)

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

	if progress, ok := s.statusMap[videoFileID]; ok {
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
			progress.Status = models.TranscriptionStatusFailed
			progress.ErrorMessage = errorMsg
		}
		if resultPPTFileID != nil {
			progress.Status = models.TranscriptionStatusCompleted
			progress.ResultPPTFileID = resultPPTFileID
		}
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
