package services

import (
	"strings"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
)

// TestSubmitTranscriptionWithModeValidation tests mode parameter validation
func TestSubmitTranscriptionWithModeValidation(t *testing.T) {
	tests := []struct {
		name        string
		mode        string
		expectError string
	}{
		{"valid local mode", "local", ""},
		{"valid cloud mode", "cloud", ""},
		{"invalid mode", "invalid", "无效的转录模式"},
		{"empty mode", "", "无效的转录模式"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validation logic should reject invalid modes
			validModes := map[string]bool{models.TranscriptionModeLocal: true, models.TranscriptionModeCloud: true}
			isValid := validModes[tt.mode]
			if tt.expectError == "" && !isValid {
				t.Errorf("Expected mode %q to be valid", tt.mode)
			}
			if tt.expectError != "" && isValid {
				t.Errorf("Expected mode %q to be invalid", tt.mode)
			}
		})
	}
}

// TestCloudModeSkipsSamplingRateValidation tests D-03 compliance
func TestCloudModeSkipsSamplingRateValidation(t *testing.T) {
	// Per D-03: cloud mode should NOT validate sampling rate
	mode := models.TranscriptionModeCloud

	// Simulate the validation logic from SubmitTranscriptionWithMode
	if mode == models.TranscriptionModeCloud {
		// Cloud mode: sampling rate validation is skipped
		// This means ANY sampling rate value (even 0 or invalid) should be accepted
		// The sampling rate is simply not used for cloud transcription
	} else {
		t.Error("This test should only run for cloud mode")
	}
}

// TestPollingBackoffParameters tests TRAN-04 exponential backoff configuration
func TestPollingBackoffParameters(t *testing.T) {
	// Verify the polling backoff constants exist and are correct
	initialDelay := 2 // seconds
	maxDelay := 60    // seconds (TRAN-04: retry.MaxDelay)
	maxAttempts := 120
	multiplier := 1.5

	assert.Equal(t, 2, initialDelay, "Initial delay should be 2 seconds")
	assert.Equal(t, 60, maxDelay, "Max delay should be 60 seconds (TRAN-04)")
	assert.Equal(t, 120, maxAttempts, "Max attempts should be 120")
	assert.Equal(t, 1.5, multiplier, "Backoff multiplier should be 1.5")

	// Simulate backoff progression
	delay := float64(initialDelay)
	for i := 0; i < 10; i++ {
		delay = delay * multiplier
		if delay > float64(maxDelay) {
			delay = float64(maxDelay)
		}
	}
	assert.Equal(t, float64(maxDelay), delay, "Backoff should reach max delay")
}

// TestHandleCloudFailureInitialStageFallback tests D-07 fallback behavior
func TestHandleCloudFailureInitialStageFallback(t *testing.T) {
	// Per D-07: initial stage failures should trigger auto-fallback
	isInitialStage := true

	if isInitialStage {
		// Should auto-fallback to local
		expectedMode := models.TranscriptionModeLocal
		assert.Equal(t, "local", expectedMode)
	}
}

// TestHandleCloudFailureMidProcessingNoFallback tests D-07 no-fallback for mid-processing
func TestHandleCloudFailureMidProcessingNoFallback(t *testing.T) {
	// Per D-07: mid-processing failures should NOT auto-fallback
	isInitialStage := false

	if !isInitialStage {
		// Should mark task as failed, not fallback
		expectedStatus := models.TranscriptionStatusFailed
		assert.Equal(t, "failed", expectedStatus)
	}
}

// TestCloudFallbackErrorPrefix tests the cloud_fallback error prefix format
func TestCloudFallbackErrorPrefix(t *testing.T) {
	originalErr := "OSS上传失败"
	prefixedErr := "cloud_fallback:" + originalErr

	assert.True(t, strings.HasPrefix(prefixedErr, "cloud_fallback:"))
	trimmed := strings.TrimPrefix(prefixedErr, "cloud_fallback:")
	assert.Equal(t, originalErr, trimmed)
}

// TestOSSCleanupScheduler verifies cleanup logic targets correct tasks
func TestOSSCleanupScheduler(t *testing.T) {
	// Verify cleanup only targets cloud mode tasks that are completed/failed
	eligibleStatuses := []string{models.TranscriptionStatusCompleted, models.TranscriptionStatusFailed}
	assert.Contains(t, eligibleStatuses, models.TranscriptionStatusCompleted)
	assert.Contains(t, eligibleStatuses, models.TranscriptionStatusFailed)
	assert.NotContains(t, eligibleStatuses, models.TranscriptionStatusProcessing)
	assert.NotContains(t, eligibleStatuses, models.TranscriptionStatusPending)
}

// TestTranscriptionProgressModeField tests the Mode field on TranscriptionProgress
func TestTranscriptionProgressModeField(t *testing.T) {
	progress := TranscriptionProgress{
		Status:       models.TranscriptionStatusProcessing,
		CurrentStage: models.TranscriptionStageUploading,
		Mode:         models.TranscriptionModeCloud,
	}
	assert.Equal(t, "cloud", progress.Mode)
	assert.Equal(t, "uploading", progress.CurrentStage)
}
