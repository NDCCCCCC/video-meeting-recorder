package models

import (
	"reflect"
	"testing"
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
