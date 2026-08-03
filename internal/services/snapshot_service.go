package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// snapshotCopyBufPool 复用 snapshot 服务 partial-MKV copy 的 32KB chunk buffer（PERF-007）。
var snapshotCopyBufPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

// SnapshotService 录制快照服务：为指定录制任务生成 partial-MKV 快照片段，
// 用于预览/对外发布。每个 taskID 持有独立 sync.Mutex 防止对同一任务的并发快照。
type SnapshotService struct {
	db               *gorm.DB
	logger           *zap.Logger
	config           *config.Config
	ffmpegPath       string
	videoFileService *VideoFileService
	snapshotMutexes  sync.Map // map[uint]*sync.Mutex - one mutex per task
}

// NewSnapshotService 创建录制快照服务。ffmpegPath 优先取 cfg.FFmpeg.Path，
// 否则使用 PATH 中的 ffmpeg。
func NewSnapshotService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, videoFileService *VideoFileService) *SnapshotService {
	ffmpegPath := cfg.FFmpeg.Path
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &SnapshotService{
		db:               db,
		logger:           logger,
		config:           cfg,
		ffmpegPath:       ffmpegPath,
		videoFileService: videoFileService,
	}
}

// getMutex returns (or creates) a mutex for the specified task ID
func (s *SnapshotService) getMutex(taskID uint) *sync.Mutex {
	mutex, _ := s.snapshotMutexes.LoadOrStore(taskID, &sync.Mutex{})
	return mutex.(*sync.Mutex)
}

// generateSnapshotFilename creates a sanitized filename with task name and sequence
func (s *SnapshotService) generateSnapshotFilename(task models.VideoRecordingTask, sequence int) string {
	// Sanitize task name for filename use (replace invalid chars with underscore)
	sanitizedName := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, task.Name)

	// Limit name length to avoid excessively long filenames (max 30 chars)
	if len(sanitizedName) > 30 {
		sanitizedName = sanitizedName[:30]
	}

	// Format: {sanitized_name}_snapshot_{seq:03d}_{timestamp}.mp4
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("%s_snapshot_%03d_%s.mp4", sanitizedName, sequence, timestamp)
}

// GenerateSnapshot generates an MP4 snapshot from an active recording task's MKV file.
// Per D-08/D-09: copies the partial MKV to temp, converts to MP4, registers via callback.
// Per D-15: Incremental — each snapshot starts from the end of the previous snapshot.
func (s *SnapshotService) GenerateSnapshot(ctx context.Context, taskID, createdBy uint, hasSharedViewer bool) (*models.VideoFile, error) {
	// Acquire mutex for this task to prevent concurrent snapshots
	mutex := s.getMutex(taskID)
	mutex.Lock()
	defer mutex.Unlock()

	// 1. Load recording task
	var task models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("录制任务不存在: %w", err)
	}

	// 1.5. Verify permission (shared_viewers can snapshot any task)
	if !hasSharedViewer && task.CreatedBy != createdBy {
		return nil, fmt.Errorf("无权限访问此录制任务")
	}

	// 2. Verify task is recording
	if task.Status != models.VideoStatusRecording {
		return nil, fmt.Errorf("任务不在录制状态，无法生成快照")
	}

	// 3. Verify MKV file exists
	if task.MKVFilePath == "" {
		return nil, fmt.Errorf("录制文件路径为空")
	}
	if _, err := os.Stat(task.MKVFilePath); err != nil {
		return nil, fmt.Errorf("录制文件不存在: %w", err)
	}

	// 4. D-15: Find the last snapshot for this task to determine incremental offset
	var lastSnapshot models.VideoFile
	seekOffset := 0.0
	if err := s.db.WithContext(ctx).Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeSnapshot).
		Order("created_at DESC").First(&lastSnapshot).Error; err == nil {
		// Last snapshot found — calculate its end offset (snapshot_offset + duration)
		if lastSnapshot.SnapshotOffset > 0 || lastSnapshot.Duration > 0 {
			seekOffset = lastSnapshot.SnapshotOffset + float64(lastSnapshot.Duration)
		}
		s.logger.Info("增量快照: 从上次结束点开始",
			zap.Uint("task_id", taskID),
			zap.Float64("last_offset", lastSnapshot.SnapshotOffset),
			zap.Int("last_duration", lastSnapshot.Duration),
			zap.Float64("seek_offset", seekOffset),
		)
	}

	// 4.5. Calculate current recording duration
	// For active recordings, calculate duration from StartTime to now
	// For completed recordings, use the stored RecordingDuration
	var recordingDuration float64
	if task.Status == models.VideoStatusRecording {
		// Calculate current duration based on elapsed time since recording started
		recordingDuration = time.Since(task.StartTime).Seconds()
		s.logger.Info("录制进行中，计算当前时长",
			zap.Uint("task_id", taskID),
			zap.Float64("duration", recordingDuration),
		)
	} else {
		// Use stored duration for completed/failed recordings
		recordingDuration = float64(task.RecordingDuration)
	}

	// Validate recording duration is positive
	if recordingDuration <= 0 {
		return nil, fmt.Errorf("录制时长无效: %.0f秒，无法生成快照", recordingDuration)
	}

	// Validate seek offset doesn't exceed recording duration
	if seekOffset >= recordingDuration {
		s.logger.Warn("快照偏移量超过录制时长",
			zap.Uint("task_id", taskID),
			zap.Float64("seek_offset", seekOffset),
			zap.Float64("recording_duration", recordingDuration),
		)
		return nil, fmt.Errorf("快照偏移量 %.2f 秒超过或等于录制时长 %.2f 秒", seekOffset, recordingDuration)
	}

	// Validate minimum snapshot duration (at least 1 second)
	if recordingDuration-seekOffset < 1.0 {
		s.logger.Warn("快照时长不足",
			zap.Uint("task_id", taskID),
			zap.Float64("remaining_duration", recordingDuration-seekOffset),
		)
		return nil, fmt.Errorf("快照时长不足 1 秒（剩余时长: %.2f 秒）", recordingDuration-seekOffset)
	}

	// Log snapshot parameters for debugging
	s.logger.Info("快照参数验证通过",
		zap.Uint("task_id", taskID),
		zap.Float64("recording_duration", recordingDuration),
		zap.Float64("seek_offset", seekOffset),
		zap.Float64("snapshot_duration", recordingDuration-seekOffset),
	)

	// 5. Copy partial MKV to temp file (avoid locking issues with active recording)
	tempDir := filepath.Join(filepath.Dir(task.MKVFilePath), "snapshots")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建快照目录失败: %w", err)
	}

	// Count existing snapshots for this task to determine sequence number
	var snapshotCount int64
	s.db.WithContext(ctx).Model(&models.VideoFile{}).
		Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeSnapshot).
		Count(&snapshotCount)
	sequence := int(snapshotCount) + 1

	// Generate filename with task context
	filename := s.generateSnapshotFilename(task, sequence)

	s.logger.Info("生成快照文件名",
		zap.Uint("task_id", taskID),
		zap.String("filename", filename),
		zap.Int("sequence", sequence),
	)

	timestamp := time.Now().Format("20060102_150405")
	tempMKV := filepath.Join(tempDir, fmt.Sprintf("snapshot_%s.mkv", timestamp))

	// Copy the file (read-only, won't interrupt recording)
	if err := copyFile(task.MKVFilePath, tempMKV); err != nil {
		return nil, fmt.Errorf("复制录制文件失败: %w", err)
	}
	defer func() { _ = os.Remove(tempMKV) }() // Clean up temp MKV after conversion

	// 6. Convert temp MKV to MP4 with incremental offset (D-15)
	outputMP4 := filepath.Join(tempDir, filename)

	// Re-validate task status before starting FFmpeg to catch race conditions
	var currentTask models.VideoRecordingTask
	if err := s.db.WithContext(ctx).First(&currentTask, taskID).Error; err == nil {
		if currentTask.Status != models.VideoStatusRecording {
			s.logger.Warn("录制任务状态变更，无法生成快照",
				zap.Uint("task_id", taskID),
				zap.String("status", string(currentTask.Status)),
			)
			return nil, fmt.Errorf("录制任务已停止或失败，无法生成快照")
		}
	}

	// PERF-003/BUG-005: derive FFmpeg ctx from request ctx so request cancellation
	// also interrupts the long-running transcode (cascade).
	ffmpegCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	// recordingDuration is already validated above (line 86-88)
	// CRITICAL: Use validated recordingDuration to capture video up to NOW, not file end

	// Build FFmpeg args with -ss for incremental seeking
	args := []string{"-y"}
	if seekOffset > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", seekOffset))
	}
	// CRITICAL FIX: Add -to parameter to limit output to current recording time
	// This ensures snapshot captures only video up to the moment button was clicked
	args = append(args,
		"-i", tempMKV,
		"-to", fmt.Sprintf("%.3f", recordingDuration),
		"-c", "copy",
		"-movflags", "+faststart",
		outputMP4,
	)

	cmd := exec.CommandContext(ffmpegCtx, s.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.logger.Error("FFmpeg快照转换失败",
			zap.Uint("task_id", taskID),
			zap.Error(err),
			zap.String("output", string(output)),
		)
		return nil, fmt.Errorf("FFmpeg快照转换失败: %w, output: %s", err, string(output))
	}

	// 7. Verify output file exists and has content
	if info, err := os.Stat(outputMP4); err != nil {
		return nil, fmt.Errorf("快照文件生成失败: %w", err)
	} else if info.Size() == 0 {
		return nil, fmt.Errorf("快照文件为空，可能录制已中断")
	}

	// 8. Find the parent VideoFile for this task
	var parentFile models.VideoFile
	var parentID *uint // Use pointer to allow nil when no parent exists
	if err := s.db.WithContext(ctx).Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeRecording).First(&parentFile).Error; err == nil {
		parentID = &parentFile.ID
	} else {
		s.logger.Warn("快照未找到父录制文件，将创建无父级的快照记录",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
	}

	// 9. Register snapshot file via VideoFileService callback (D-10, D-13)
	// Pass seekOffset as SnapshotOffset so it's stored on the VideoFile record
	snapshotFile, err := s.videoFileService.CreateSegmentFile(ctx, outputMP4, parentID, models.SourceTypeSnapshot, createdBy, seekOffset)
	if err != nil {
		return nil, fmt.Errorf("注册快照文件失败: %w", err)
	}

	s.logger.Info("快照生成完成 (互斥锁已释放)",
		zap.Uint("task_id", taskID),
		zap.Uint("snapshot_file_id", snapshotFile.ID),
		zap.Float64("offset", seekOffset),
		zap.String("path", outputMP4),
	)

	return snapshotFile, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = sourceFile.Close() }()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = destFile.Close() }()

	// Read from source in chunks (allows reading partial files being written to)
	bufPtr := snapshotCopyBufPool.Get().(*[]byte)
	defer snapshotCopyBufPool.Put(bufPtr)
	buf := *bufPtr
	for {
		n, err := sourceFile.Read(buf)
		if n > 0 {
			if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			break // EOF or error
		}
	}
	return nil
}
