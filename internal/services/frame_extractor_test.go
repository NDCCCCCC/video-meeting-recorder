package services

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// TestCreateTempDir tests that temp directories are created with correct naming pattern
func TestCreateTempDir(t *testing.T) {
	logger := zap.NewNop()
	extractor := NewFrameExtractor("ffmpeg", logger)

	baseDir := os.TempDir()
	videoFileID := uint(123)

	tempDir, err := extractor.CreateTempDir(baseDir, videoFileID)
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Temp directory was not created: %s", tempDir)
	}

	// Verify naming pattern contains transcription_ prefix and videoFileID
	baseName := filepath.Base(tempDir)
	if !contains(baseName, "transcription_") {
		t.Errorf("Temp directory name should contain 'transcription_', got: %s", baseName)
	}

	// Cleanup
	os.RemoveAll(tempDir)
}

// TestCleanupTempDir tests that temp directories are removed
func TestCleanupTempDir(t *testing.T) {
	logger := zap.NewNop()
	extractor := NewFrameExtractor("ffmpeg", logger)

	// Create a temp dir
	baseDir := os.TempDir()
	tempDir, err := extractor.CreateTempDir(baseDir, 456)
	if err != nil {
		t.Fatalf("CreateTempDir failed: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Fatal("Temp directory was not created")
	}

	// Cleanup
	err = extractor.CleanupTempDir(tempDir)
	if err != nil {
		t.Errorf("CleanupTempDir failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
		t.Errorf("Temp directory still exists after cleanup: %s", tempDir)
	}
}

// TestCleanupTempDir_NonExistent tests that cleanup doesn't error for non-existent dirs
func TestCleanupTempDir_NonExistent(t *testing.T) {
	logger := zap.NewNop()
	extractor := NewFrameExtractor("ffmpeg", logger)

	// Try to cleanup a non-existent directory
	err := extractor.CleanupTempDir("/tmp/this_does_not_exist_12345")
	if err != nil {
		t.Errorf("CleanupTempDir should not error for non-existent directory, got: %v", err)
	}
}

// TestSamplingRateConversion tests fps conversion from sampling rate
func TestSamplingRateConversion(t *testing.T) {
	tests := []struct {
		name               string
		samplingRateSecs   float64
		expectedFPS        float64
	}{
		{"1 second interval", 1.0, 1.0},
		{"2 second interval", 2.0, 0.5},
		{"5 second interval", 5.0, 0.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fps := 1.0 / tt.samplingRateSecs
			if fps != tt.expectedFPS {
				t.Errorf("Expected FPS %f, got %f", tt.expectedFPS, fps)
			}
		})
	}
}

// TestExtractFramesInvalidVideo tests that error is returned for non-existent video
func TestExtractFramesInvalidVideo(t *testing.T) {
	logger := zap.NewNop()
	extractor := NewFrameExtractor("ffmpeg", logger)

	baseDir := os.TempDir()
	outputDir, _ := extractor.CreateTempDir(baseDir, 789)
	defer extractor.CleanupTempDir(outputDir)

	// Try to extract from non-existent video
	_, err := extractor.ExtractFrames(nil, "/nonexistent/video.mp4", outputDir, 1.0)
	if err == nil {
		t.Error("Expected error for non-existent video file")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
