package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type SnapshotService struct {
	db               *gorm.DB
	logger           *zap.Logger
	config           *config.Config
	ffmpegPath       string
	videoFileService *VideoFileService
}

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

// GenerateSnapshot generates an MP4 snapshot from an active recording task's MKV file.
// Per D-08/D-09: copies the partial MKV to temp, converts to MP4, registers via callback.
// Per D-15: Incremental — each snapshot starts from the end of the previous snapshot.
func (s *SnapshotService) GenerateSnapshot(taskID uint, createdBy uint) (*models.VideoFile, error) {
	// 1. Load recording task
	var task models.VideoRecordingTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("录制任务不存在: %w", err)
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
	if err := s.db.Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeSnapshot).
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

	// 5. Copy partial MKV to temp file (avoid locking issues with active recording)
	tempDir := filepath.Join(filepath.Dir(task.MKVFilePath), "snapshots")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建快照目录失败: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	tempMKV := filepath.Join(tempDir, fmt.Sprintf("snapshot_%s.mkv", timestamp))

	// Copy the file (read-only, won't interrupt recording)
	if err := copyFile(task.MKVFilePath, tempMKV); err != nil {
		return nil, fmt.Errorf("复制录制文件失败: %w", err)
	}
	defer os.Remove(tempMKV) // Clean up temp MKV after conversion

	// 6. Convert temp MKV to MP4 with incremental offset (D-15)
	outputMP4 := filepath.Join(tempDir, fmt.Sprintf("snapshot_%s.mp4", timestamp))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Build FFmpeg args with -ss for incremental seeking
	args := []string{"-y"}
	if seekOffset > 0 {
		args = append(args, "-ss", fmt.Sprintf("%.3f", seekOffset))
	}
	args = append(args,
		"-i", tempMKV,
		"-c", "copy",
		"-movflags", "+faststart",
		outputMP4,
	)

	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("FFmpeg快照转换失败: %w, output: %s", err, string(output))
	}

	// 7. Verify output file
	if _, err := os.Stat(outputMP4); err != nil {
		return nil, fmt.Errorf("快照文件未生成: %w", err)
	}

	// 8. Find the parent VideoFile for this task
	var parentFile models.VideoFile
	parentID := uint(0)
	if err := s.db.Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeRecording).First(&parentFile).Error; err == nil {
		parentID = parentFile.ID
	}

	// 9. Register snapshot file via VideoFileService callback (D-10, D-13)
	// Pass seekOffset as SnapshotOffset so it's stored on the VideoFile record
	snapshotFile, err := s.videoFileService.CreateSegmentFile(outputMP4, &parentID, models.SourceTypeSnapshot, createdBy, seekOffset)
	if err != nil {
		return nil, fmt.Errorf("注册快照文件失败: %w", err)
	}

	s.logger.Info("快照生成完成",
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
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Read from source in chunks (allows reading partial files being written to)
	buf := make([]byte, 32*1024)
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
