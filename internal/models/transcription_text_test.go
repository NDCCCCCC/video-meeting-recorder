package models

import (
	"reflect"
	"testing"
)

func TestTranscriptionTextModel(t *testing.T) {
	text := TranscriptionText{
		TranscriptionTaskID: 1,
		Text:                "测试文字",
		BeginTime:           1000,
		EndTime:             2000,
		SegmentIndex:        0,
	}

	textType := reflect.TypeOf(text)
	if textType.Name() != "TranscriptionText" {
		t.Errorf("Expected type name 'TranscriptionText', got '%s'", textType.Name())
	}

	fields := []string{
		"TranscriptionTaskID", "Text", "BeginTime", "EndTime", "SegmentIndex",
	}
	for _, field := range fields {
		if _, found := textType.FieldByName(field); !found {
			t.Errorf("Field %s does not exist in TranscriptionText", field)
		}
	}

	// Verify Base fields (ID, CreatedAt, UpdatedAt, DeletedAt)
	baseFields := []string{"ID", "CreatedAt", "UpdatedAt", "DeletedAt"}
	for _, field := range baseFields {
		if _, found := textType.FieldByName(field); !found {
			t.Errorf("Base field %s does not exist in TranscriptionText", field)
		}
	}
}

func TestTranscriptionTextTableName(t *testing.T) {
	text := TranscriptionText{}
	if text.TableName() != "transcription_texts" {
		t.Errorf("Expected table name 'transcription_texts', got '%s'", text.TableName())
	}
}

func TestTranscriptionTaskModeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"ModeLocal", TranscriptionModeLocal},
		{"ModeCloud", TranscriptionModeCloud},
		{"StageUploading", TranscriptionStageUploading},
		{"StageQueued", TranscriptionStageQueued},
		{"StageProcessing", TranscriptionStageProcessing},
		{"StageDownloading", TranscriptionStageDownloading},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value == "" {
				t.Errorf("Constant %s is empty", tt.name)
			}
		})
	}
}

func TestTranscriptionTaskModeFields(t *testing.T) {
	task := TranscriptionTask{
		VideoFileID: 1,
		CreatedBy:   1,
	}
	taskType := reflect.TypeOf(task)

	modeFields := []string{"Mode", "CloudTaskID", "OSSURL"}
	for _, field := range modeFields {
		if _, found := taskType.FieldByName(field); !found {
			t.Errorf("Field %s does not exist in TranscriptionTask", field)
		}
	}
}
