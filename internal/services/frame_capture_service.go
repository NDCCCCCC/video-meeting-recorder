package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// FrameCaptureService handles capturing frames from videos at specific timestamps
type FrameCaptureService struct {
	ffprobePath string
	ffmpegPath  string
	logger      *zap.Logger
}

// NewFrameCaptureService creates a new FrameCaptureService instance
func NewFrameCaptureService(ffmpegPath, ffprobePath string, logger *zap.Logger) *FrameCaptureService {
	return &FrameCaptureService{
		ffprobePath: ffprobePath,
		ffmpegPath:  ffmpegPath,
		logger:      logger,
	}
}

// CaptureFrame captures a frame from video at specific timestamp and saves to file
// Uses FFmpeg with optimal flags for quality and speed
func (s *FrameCaptureService) CaptureFrame(ctx context.Context, videoPath string, timestamp float64, outputPath string) error {
	// Validate video path exists (WR-03: call validatePath)
	if err := s.validatePath(videoPath); err != nil {
		return fmt.Errorf("invalid video path: %w", err)
	}

	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file not found: %s", videoPath)
	}

	// Validate file extension (WR-01: add MIME type/extension validation)
	ext := strings.ToLower(filepath.Ext(videoPath))
	validExts := []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv"}
	isValidExt := false
	for _, valid := range validExts {
		if ext == valid {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		return fmt.Errorf("invalid video file extension: %s (supported: %v)", ext, validExts)
	}

	// Validate timestamp
	validatedTimestamp, err := s.ValidateTimestamp(videoPath, timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	s.logger.Debug("Capturing frame",
		zap.String("video_path", videoPath),
		zap.Float64("timestamp", validatedTimestamp),
		zap.String("output_path", outputPath),
	)

	// Build FFmpeg command for optimal frame capture
	// -ss before -i for fast seek (keyframe-aligned)
	// -vframes 1 to capture single frame
	// -q:v 2 for high quality JPEG (quality 95)
	args := []string{
		"-y",                                           // Overwrite output file
		"-ss", fmt.Sprintf("%.3f", validatedTimestamp), // Timestamp in seconds
		"-i", videoPath, // Input file
		"-vframes", "1", // Capture only one frame
		"-q:v", "2", // JPEG quality 2 (high quality, range 1-31)
		outputPath,
	}

	cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		s.logger.Error("FFmpeg frame capture failed",
			zap.Error(err),
			zap.String("stderr", stderr.String()),
		)
		return fmt.Errorf("frame capture failed: %w, stderr: %s", err, stderr.String())
	}

	// Verify output file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return fmt.Errorf("frame capture succeeded but output file not created")
	}

	s.logger.Info("Frame captured successfully",
		zap.Float64("timestamp", validatedTimestamp),
		zap.String("output_path", outputPath),
	)

	return nil
}

// CaptureFrameToBytes captures a frame and returns it as byte array for preview
// Useful for showing preview before insertion
func (s *FrameCaptureService) CaptureFrameToBytes(ctx context.Context, videoPath string, timestamp float64) ([]byte, string, error) {
	// Create temp file for captured frame
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fmt.Sprintf("capture_%d_%d.jpg", time.Now().UnixNano(), int(timestamp)))

	// Capture frame to temp file
	if err := s.CaptureFrame(ctx, videoPath, timestamp, tempFile); err != nil {
		return nil, "", err
	}

	// Read file bytes
	data, err := os.ReadFile(tempFile)
	if err != nil {
		// Clean up temp file on error
		os.Remove(tempFile)
		return nil, "", fmt.Errorf("failed to read captured frame: %w", err)
	}

	// Clean up temp file after reading
	os.Remove(tempFile)

	s.logger.Debug("Frame captured to bytes",
		zap.Float64("timestamp", timestamp),
		zap.Int("size", len(data)),
	)

	return data, "image/jpeg", nil
}

// ValidateTimestamp checks if timestamp is within video duration
// Returns clamped timestamp if out of bounds
func (s *FrameCaptureService) ValidateTimestamp(videoPath string, timestamp float64) (float64, error) {
	if timestamp < 0 {
		return 0, fmt.Errorf("timestamp cannot be negative: %.3f", timestamp)
	}

	// WR-04: Add context timeout for GetVideoDuration call
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	duration, err := s.GetVideoDuration(ctx, videoPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get video duration: %w", err)
	}

	// Clamp timestamp to valid range [0, duration]
	if timestamp > duration {
		s.logger.Warn("Timestamp exceeds video duration, clamping to end",
			zap.Float64("requested", timestamp),
			zap.Float64("duration", duration),
		)
		timestamp = duration
	}

	return timestamp, nil
}

// GetVideoDuration uses ffprobe to extract video duration in seconds
// WR-04: Added context parameter for timeout support
func (s *FrameCaptureService) GetVideoDuration(ctx context.Context, videoPath string) (float64, error) {
	// Validate video path exists (WR-03: call validatePath)
	if err := s.validatePath(videoPath); err != nil {
		return 0, fmt.Errorf("invalid video path: %w", err)
	}

	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("video file not found: %s", videoPath)
	}

	// Build ffprobe command to get duration
	// -select_streams v:0 - select only video stream
	// -show_entries stream=duration - show duration property
	// -of json - output in JSON format
	args := []string{
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=duration",
		"-of", "json",
		videoPath,
	}

	// WR-04: Use CommandContext to respect timeout
	cmd := exec.CommandContext(ctx, s.ffprobePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w, stderr: %s", err, stderr.String())
	}

	// Parse JSON output
	var result struct {
		Streams []struct {
			Duration string `json:"duration"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return 0, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	if len(result.Streams) == 0 {
		return 0, fmt.Errorf("no video stream found in file")
	}

	durationStr := result.Streams[0].Duration
	if durationStr == "" {
		return 0, fmt.Errorf("duration not found in video stream")
	}

	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return duration, nil
}

// validatePath validates that a path is safe and doesn't contain shell metacharacters
func (s *FrameCaptureService) validatePath(path string) error {
	// Check for shell metacharacters that could enable command injection
	dangerousChars := []string{"`", "$", ";", "&", "|", ">", "<", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return fmt.Errorf("path contains dangerous character: %s", char)
		}
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute path: %w", err)
	}

	// Verify path doesn't escape to sensitive system directories
	if strings.HasPrefix(absPath, "/etc") || strings.HasPrefix(absPath, "/sys") ||
		strings.HasPrefix(absPath, "/proc") || strings.HasPrefix(absPath, "/root") {
		return fmt.Errorf("access to system directory not allowed: %s", path)
	}

	return nil
}
