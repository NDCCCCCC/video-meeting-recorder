package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SplitTask represents a pending split operation
type SplitTask struct {
	ID          uint
	VideoFileID uint
	Markers     []float64 // timestamps in seconds, sorted
	ReEncode    bool
	CreatedBy   uint
	CreatedAt   time.Time
}

type SplittingService struct {
	db               *gorm.DB
	logger           *zap.Logger
	config           *config.Config
	taskQueue        chan *SplitTask
	workers          int
	maxRetries       int
	cancelFuncs      map[uint]context.CancelFunc
	mu               sync.RWMutex
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	ffmpegPath       string
	videoFileService *VideoFileService
	// Track active splits: videoFileID -> status ("processing" / "completed" / "failed")
	statusMap map[uint]string
	statusMu  sync.RWMutex
}

func NewSplittingService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, videoFileService *VideoFileService) *SplittingService {
	ctx, cancel := context.WithCancel(context.Background())
	ffmpegPath := cfg.FFmpeg.Path
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &SplittingService{
		db:               db,
		logger:           logger,
		config:           cfg,
		taskQueue:        make(chan *SplitTask, 100),
		workers:          2,
		maxRetries:       3,
		cancelFuncs:      make(map[uint]context.CancelFunc),
		ctx:              ctx,
		cancel:           cancel,
		ffmpegPath:       ffmpegPath,
		videoFileService: videoFileService,
		statusMap:        make(map[uint]string),
	}
}

func (s *SplittingService) Start() error {
	for i := 0; i < s.workers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	s.logger.Info("分割服务启动", zap.Int("workers", s.workers))
	return nil
}

func (s *SplittingService) Stop() {
	s.cancel()
	s.wg.Wait()
}

// SubmitSplit submits a split task
func (s *SplittingService) SubmitSplit(videoFileID uint, markers []float64, reEncode bool, createdBy uint) error {
	task := &SplitTask{
		VideoFileID: videoFileID,
		Markers:     markers,
		ReEncode:    reEncode,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}
	s.statusMu.Lock()
	s.statusMap[videoFileID] = "processing"
	s.statusMu.Unlock()

	select {
	case s.taskQueue <- task:
		s.logger.Info("分割任务已提交", zap.Uint("video_file_id", videoFileID), zap.Int("markers", len(markers)))
		return nil
	default:
		s.statusMu.Lock()
		s.statusMap[videoFileID] = "failed"
		s.statusMu.Unlock()
		return fmt.Errorf("分割任务队列已满")
	}
}

// GetSplitStatus returns the current split status for a video file
func (s *SplittingService) GetSplitStatus(videoFileID uint) string {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	if status, ok := s.statusMap[videoFileID]; ok {
		return status
	}
	return ""
}

func (s *SplittingService) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case task := <-s.taskQueue:
			s.processSplit(task)
		case <-s.ctx.Done():
			return
		}
	}
}

func (s *SplittingService) processSplit(task *SplitTask) {
	// 1. Load source video file
	var sourceFile models.VideoFile
	if err := s.db.First(&sourceFile, task.VideoFileID).Error; err != nil {
		s.logger.Error("源视频文件不存在", zap.Uint("video_file_id", task.VideoFileID), zap.Error(err))
		s.statusMu.Lock()
		s.statusMap[task.VideoFileID] = "failed"
		s.statusMu.Unlock()
		return
	}

	// 2. Create output directory
	outputDir := filepath.Join(filepath.Dir(sourceFile.FilePath), "segments")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		s.logger.Error("创建输出目录失败", zap.Error(err))
		s.statusMu.Lock()
		s.statusMap[task.VideoFileID] = "failed"
		s.statusMu.Unlock()
		return
	}

	// 3. Build segment intervals from markers
	//    markers: [10, 30, 50] -> segments: [0-10, 10-30, 30-50, 50-end]
	type segment struct {
		start float64
		end   float64 // 0 means until end
	}
	var segments []segment
	sortedMarkers := make([]float64, len(task.Markers))
	copy(sortedMarkers, task.Markers)
	sort.Float64s(sortedMarkers)

	prev := 0.0
	for _, m := range sortedMarkers {
		segments = append(segments, segment{start: prev, end: m})
		prev = m
	}
	// Final segment from last marker to end
	segments = append(segments, segment{start: prev, end: 0})

	// 4. Execute FFmpeg for each segment
	parentID := uint(task.VideoFileID)
	sourceName := sourceFile.FileName
	ext := filepath.Ext(sourceName)
	baseName := sourceName[:len(sourceName)-len(ext)]

	var createdFiles []string
	for i, seg := range segments {
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s_segment_%03d.mp4", baseName, i+1))

		ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
		s.mu.Lock()
		s.cancelFuncs[task.VideoFileID] = cancel
		s.mu.Unlock()

		args := []string{"-y", "-i", sourceFile.FilePath}
		if seg.start > 0 {
			args = append(args, "-ss", fmt.Sprintf("%.3f", seg.start))
		}
		if seg.end > 0 {
			args = append(args, "-to", fmt.Sprintf("%.3f", seg.end))
		}
		if task.ReEncode {
			args = append(args, "-c:v", "libx264", "-c:a", "aac", "-b:a", "128k")
		} else {
			args = append(args, "-c", "copy", "-avoid_negative_ts", "1")
		}
		args = append(args, "-movflags", "+faststart", outputPath)

		cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf

		s.logger.Info("执行FFmpeg分割",
			zap.Int("segment", i+1),
			zap.Int("total", len(segments)),
			zap.String("output", outputPath),
		)

		if err := cmd.Run(); err != nil {
			cancel()
			s.logger.Error("FFmpeg分割失败",
				zap.Int("segment", i+1),
				zap.String("output", outputPath),
				zap.Error(err),
				zap.String("stderr", stderrBuf.String()),
			)
			continue
		}
		cancel()

		createdFiles = append(createdFiles, outputPath)
	}

	// 5. Register segment files via VideoFileService callback (D-13)
	for _, segPath := range createdFiles {
		segmentFile, err := s.videoFileService.CreateSegmentFile(segPath, &parentID, models.SourceTypeSplit, task.CreatedBy)
		if err != nil {
			s.logger.Error("注册分割段文件失败", zap.String("path", segPath), zap.Error(err))
			continue
		}
		s.logger.Info("分割段文件已注册", zap.Uint("segment_id", segmentFile.ID), zap.String("path", segPath))
	}

	// 6. Update status
	s.statusMu.Lock()
	if len(createdFiles) == len(segments) {
		s.statusMap[task.VideoFileID] = "completed"
	} else if len(createdFiles) > 0 {
		s.statusMap[task.VideoFileID] = "completed" // partial success still counts
	} else {
		s.statusMap[task.VideoFileID] = "failed"
	}
	s.statusMu.Unlock()
}
