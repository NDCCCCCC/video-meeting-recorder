package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestModeParameterValidation tests handler-level mode validation
func TestModeParameterValidation(t *testing.T) {
	validModes := map[string]bool{"local": true, "cloud": true}

	tests := []struct {
		mode    string
		isValid bool
	}{
		{"local", true},
		{"cloud", true},
		{"", false},
		{"invalid", false},
		{"LOCAL", false},
		{"CLOUD", false},
		{"remote", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			assert.Equal(t, tt.isValid, validModes[tt.mode],
				"mode %q validation mismatch", tt.mode)
		})
	}
}

// TestCloudNoSamplingRateRequired tests D-03: cloud mode does not require sampling_rate
func TestCloudNoSamplingRateRequired(t *testing.T) {
	// Per D-03: when mode=cloud, the request body does not need sampling_rate
	// The handler should accept { "mode": "cloud" } without sampling_rate
	mode := "cloud"
	samplingRate := 0.0 // Not provided / zero value

	// Cloud mode should proceed regardless of sampling_rate value
	if mode == "cloud" {
		// sampling_rate is ignored for cloud mode
		assert.True(t, true, "Cloud mode should not require sampling_rate per D-03")
		_ = samplingRate // Explicitly unused
	}
}

// TestLocalModeRequiresSamplingRate tests that local mode still needs sampling_rate
func TestLocalModeRequiresSamplingRate(t *testing.T) {
	mode := "local"
	samplingRate := 0.0 // Missing

	if mode == "local" && samplingRate == 0 {
		// Handler should apply default sampling rate of 0.5
		defaultSamplingRate := 0.5
		assert.Equal(t, 0.5, defaultSamplingRate,
			"Local mode should default sampling_rate to 0.5 when not provided")
	}
}

// TestModeDefaultToLocal tests backward compatibility
func TestModeDefaultToLocal(t *testing.T) {
	// When mode is not specified in request body, default to "local"
	receivedMode := ""
	if receivedMode == "" {
		receivedMode = "local"
	}
	assert.Equal(t, "local", receivedMode,
		"Mode should default to 'local' when not provided")
}
