package models

import (
	"encoding/json"
	"reflect"
	"testing"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/stretchr/testify/assert"
)

// TestTranscriptionTaskModel tests that the TranscriptionTask model is properly defined
func TestTranscriptionTaskModel(t *testing.T) {
	// Verify TranscriptionTask can be instantiated
	task := TranscriptionTask{
		VideoFileID: 1,
		CreatedBy:   1,
	}

	// Check struct type
	taskType := reflect.TypeOf(task)
	if taskType.Name() != "TranscriptionTask" {
		t.Errorf("Expected type name 'TranscriptionTask', got '%s'", taskType.Name())
	}

	// Verify key fields exist
	fields := []string{
		"VideoFileID", "SamplingRate", "Status", "CurrentStage",
		"FramesProcessed", "TotalFrames", "Percentage", "ResultPPTFileID",
		"ErrorMessage", "CreatedBy",
	}

	for _, field := range fields {
		if _, found := taskType.FieldByName(field); !found {
			t.Errorf("Field %s does not exist in TranscriptionTask", field)
		}
	}

	// Verify it embeds Base (has ID, CreatedAt, UpdatedAt, DeletedAt)
	baseFields := []string{"ID", "CreatedAt", "UpdatedAt", "DeletedAt"}
	for _, field := range baseFields {
		if _, found := taskType.FieldByName(field); !found {
			t.Errorf("Base field %s does not exist in TranscriptionTask", field)
		}
	}
}

// TestTranscriptionTaskConstants tests that all status constants are defined
func TestTranscriptionTaskConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"Pending", TranscriptionStatusPending},
		{"Processing", TranscriptionStatusProcessing},
		{"Completed", TranscriptionStatusCompleted},
		{"Failed", TranscriptionStatusFailed},
		{"StageExtracting", TranscriptionStageExtracting},
		{"StageDetecting", TranscriptionStageDetecting},
		{"StageGenerating", TranscriptionStageGenerating},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("Constant %s is empty", tt.name)
			}
		})
	}
}

// TestTranscriptionTaskTableName tests that the table name is correct
func TestTranscriptionTaskTableName(t *testing.T) {
	task := TranscriptionTask{}
	if task.TableName() != "transcription_tasks" {
		t.Errorf("Expected table name 'transcription_tasks', got '%s'", task.TableName())
	}
}

// TestTranscriptionTaskDefaults tests that default values are set correctly
func TestTranscriptionTaskDefaults(t *testing.T) {
	task := TranscriptionTask{
		VideoFileID: 1,
		CreatedBy:   1,
	}

	// Check default values
	if task.Status != "" {
		// Default should be set by database, not struct
		// But we can verify the constant exists
		if TranscriptionStatusPending == "" {
			t.Error("TranscriptionStatusPending constant is empty")
		}
	}

	if task.SamplingRate != 0 {
		t.Errorf("Expected default SamplingRate to be 0 (set by DB), got %f", task.SamplingRate)
	}
}

// TestTranscriptionTask_SlideTimestamps tests the slide timestamp functionality
func TestTranscriptionTask_SlideTimestamps(t *testing.T) {
	tests := []struct {
		name        string
		task        *TranscriptionTask
		setup       func(*TranscriptionTask)
		expectError bool
		validate    func(*testing.T, *TranscriptionTask)
	}{
		{
			name: "Parse valid slide timestamps JSON",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				timestamps := []SlideTimestamp{
					{SlideNumber: 1, Timestamp: 0.0},
					{SlideNumber: 2, Timestamp: 15.5},
					{SlideNumber: 3, Timestamp: 30.0},
				}
				data, _ := json.Marshal(timestamps)
				task.SlideTimestamps = string(data)
			},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				parsed, err := task.GetSlideTimestamps()
				assert.NoError(t, err)
				assert.Len(t, parsed, 3)
				assert.Equal(t, 1, parsed[0].SlideNumber)
				assert.Equal(t, 0.0, parsed[0].Timestamp)
				assert.Equal(t, 2, parsed[1].SlideNumber)
				assert.Equal(t, 15.5, parsed[1].Timestamp)
				assert.Equal(t, 3, parsed[2].SlideNumber)
				assert.Equal(t, 30.0, parsed[2].Timestamp)
			},
		},
		{
			name:        "Parse empty slide timestamps returns empty array",
			task:        &TranscriptionTask{SlideTimestamps: ""},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				parsed, err := task.GetSlideTimestamps()
				assert.NoError(t, err)
				assert.Len(t, parsed, 0)
			},
		},
		{
			name:        "Parse invalid JSON returns empty array gracefully",
			task:        &TranscriptionTask{SlideTimestamps: "invalid json"},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				parsed, err := task.GetSlideTimestamps()
				assert.NoError(t, err) // Should not error, just return empty
				assert.Len(t, parsed, 0)
			},
		},
		{
			name: "Set slide timestamps serializes to JSON",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				timestamps := []SlideTimestamp{
					{SlideNumber: 1, Timestamp: 0.0},
					{SlideNumber: 2, Timestamp: 10.5},
				}
				err := task.SetSlideTimestamps(timestamps)
				assert.NoError(t, err)
			},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				assert.NotEmpty(t, task.SlideTimestamps)
				var parsed []SlideTimestamp
				err := json.Unmarshal([]byte(task.SlideTimestamps), &parsed)
				assert.NoError(t, err)
				assert.Len(t, parsed, 2)
			},
		},
		{
			name: "GetTimestampForSlide returns correct timestamp",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				timestamps := []SlideTimestamp{
					{SlideNumber: 1, Timestamp: 0.0},
					{SlideNumber: 2, Timestamp: 15.5},
					{SlideNumber: 3, Timestamp: 30.0},
				}
				task.SetSlideTimestamps(timestamps)
			},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				ts, err := task.GetTimestampForSlide(2)
				assert.NoError(t, err)
				assert.Equal(t, 15.5, ts)
			},
		},
		{
			name: "GetTimestampForSlide returns error for non-existent slide",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				timestamps := []SlideTimestamp{
					{SlideNumber: 1, Timestamp: 0.0},
					{SlideNumber: 2, Timestamp: 15.5},
				}
				task.SetSlideTimestamps(timestamps)
			},
			expectError: true,
			validate: func(t *testing.T, task *TranscriptionTask) {
				_, err := task.GetTimestampForSlide(5)
				assert.Error(t, err)
				assert.True(t, apperrors.Is(err, apperrors.ErrNotFound))
			},
		},
		{
			name: "AddSlideTimestamp adds new timestamp",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				task.AddSlideTimestamp(1, 0.0)
				task.AddSlideTimestamp(2, 15.5)
			},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				ts1, err := task.GetTimestampForSlide(1)
				assert.NoError(t, err)
				assert.Equal(t, 0.0, ts1)

				ts2, err := task.GetTimestampForSlide(2)
				assert.NoError(t, err)
				assert.Equal(t, 15.5, ts2)
			},
		},
		{
			name: "AddSlideTimestamp updates existing timestamp",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				task.AddSlideTimestamp(1, 0.0)
				task.AddSlideTimestamp(1, 5.5) // Update slide 1
			},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				ts, err := task.GetTimestampForSlide(1)
				assert.NoError(t, err)
				assert.Equal(t, 5.5, ts) // Should be updated
			},
		},
		{
			name: "Validate slide number positive",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				timestamps := []SlideTimestamp{
					{SlideNumber: 1, Timestamp: 0.0},
					{SlideNumber: -1, Timestamp: 10.0}, // Invalid
				}
				task.SetSlideTimestamps(timestamps)
			},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				// Should have filtered out invalid slide numbers
				parsed, _ := task.GetSlideTimestamps()
				for _, ts := range parsed {
					assert.Greater(t, ts.SlideNumber, 0, "Slide number must be positive")
				}
			},
		},
		{
			name: "Validate timestamp non-negative",
			task: &TranscriptionTask{},
			setup: func(task *TranscriptionTask) {
				timestamps := []SlideTimestamp{
					{SlideNumber: 1, Timestamp: 0.0},
					{SlideNumber: 2, Timestamp: -5.0}, // Invalid
				}
				task.SetSlideTimestamps(timestamps)
			},
			expectError: false,
			validate: func(t *testing.T, task *TranscriptionTask) {
				// Should have filtered out invalid timestamps
				parsed, _ := task.GetSlideTimestamps()
				for _, ts := range parsed {
					assert.GreaterOrEqual(t, ts.Timestamp, 0.0, "Timestamp must be non-negative")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.task)
			}

			if tt.validate != nil {
				tt.validate(t, tt.task)
			}
		})
	}
}
