package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestFrameCaptureService_CaptureFrame(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	logger := zap.NewNop()
	ffmpegPath := "ffmpeg"
	ffprobePath := "ffprobe"

	service := NewFrameCaptureService(ffmpegPath, ffprobePath, logger)

	// Create a test video path (this would need an actual video file for testing)
	// For now, we'll test with a mock scenario
	videoPath := "test_video.mp4"

	// Test that validation works for non-existent file
	ctx := context.Background()
	timestamp := 5.0
	outputPath := filepath.Join(os.TempDir(), "test_capture.jpg")

	err := service.CaptureFrame(ctx, videoPath, timestamp, outputPath)
	if err == nil {
		t.Error("Expected error for non-existent video file")
	}

	// In real testing, you would:
	// 1. Create or use a test video file
	// 2. Capture a frame at a known timestamp
	// 3. Verify the output file exists
	// 4. Verify the output file is a valid JPEG
}

func TestFrameCaptureService_ValidateTimestamp(t *testing.T) {
	logger := zap.NewNop()
	ffmpegPath := "ffmpeg"
	ffprobePath := "ffprobe"

	service := NewFrameCaptureService(ffmpegPath, ffprobePath, logger)

	tests := []struct {
		name      string
		timestamp float64
		wantErr   bool
	}{
		{
			name:      "valid timestamp",
			timestamp: 10.5,
			wantErr:   false,
		},
		{
			name:      "zero timestamp",
			timestamp: 0.0,
			wantErr:   false,
		},
		{
			name:      "negative timestamp",
			timestamp: -1.0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would need an actual video file to work properly
			// For now, we test the negative timestamp case
			if tt.timestamp < 0 {
				_, err := service.ValidateTimestamp(context.Background(), "dummy.mp4", tt.timestamp)
				if err == nil {
					t.Error("Expected error for negative timestamp")
				}
			}
		})
	}
}

func TestFrameCaptureService_CaptureFrameToBytes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	logger := zap.NewNop()
	ffmpegPath := "ffmpeg"
	ffprobePath := "ffprobe"

	service := NewFrameCaptureService(ffmpegPath, ffprobePath, logger)

	ctx := context.Background()
	videoPath := "test_video.mp4"
	timestamp := 5.0

	// Test with non-existent file
	data, mime, err := service.CaptureFrameToBytes(ctx, videoPath, timestamp)
	if err == nil {
		t.Error("Expected error for non-existent video file")
	}
	if data != nil {
		t.Error("Expected nil data for error case")
	}
	if mime != "" {
		t.Error("Expected empty MIME type for error case")
	}

	// In real testing, you would:
	// 1. Use a real video file
	// 2. Capture a frame to bytes
	// 3. Verify the byte array is not empty
	// 4. Verify the MIME type is "image/jpeg"
	// 5. Verify the bytes can be decoded as a JPEG
}

func TestFrameCaptureService_GetVideoDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	logger := zap.NewNop()
	ffprobePath := "ffprobe"

	service := NewFrameCaptureService("ffmpeg", ffprobePath, logger)

	// Test with non-existent file
	_, err := service.GetVideoDuration(context.Background(), "nonexistent.mp4")
	if err == nil {
		t.Error("Expected error for non-existent video file")
	}

	// In real testing, you would:
	// 1. Use a real video file with known duration
	// 2. Get the duration
	// 3. Verify it matches the expected duration
}

func TestFrameCaptureService_validatePath(t *testing.T) {
	logger := zap.NewNop()
	service := NewFrameCaptureService("ffmpeg", "ffprobe", logger)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid relative path",
			path:    "test/video.mp4",
			wantErr: false,
		},
		{
			name:    "valid absolute path",
			path:    "/home/user/video.mp4",
			wantErr: false,
		},
		{
			name:    "path with shell metacharacter - backtick",
			path:    "test/`rm -rf`/video.mp4",
			wantErr: true,
		},
		{
			name:    "path with shell metacharacter - dollar sign",
			path:    "test/$HOME/video.mp4",
			wantErr: true,
		},
		{
			name:    "path with shell metacharacter - semicolon",
			path:    "test/; echo/video.mp4",
			wantErr: true,
		},
		{
			name:    "path with newline",
			path:    "test/\n/video.mp4",
			wantErr: true,
		},
		{
			name:    "path accessing /etc",
			path:    "/etc/passwd",
			wantErr: true,
		},
		{
			name:    "path accessing /sys",
			path:    "/sys/kernel/video.mp4",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
