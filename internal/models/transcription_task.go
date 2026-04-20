package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// SlideTimestamp represents a slide-to-timestamp mapping
type SlideTimestamp struct {
	SlideNumber int     `json:"slide_number"`
	Timestamp   float64 `json:"timestamp"` // Video timestamp in seconds
}

// TranscriptionTask 转录任务模型
type TranscriptionTask struct {
	Base
	VideoFileID     uint       `gorm:"not null;index" json:"video_file_id"`
	VideoFile       *VideoFile `gorm:"foreignKey:VideoFileID" json:"video_file,omitempty"`
	SamplingRate    float64    `gorm:"default:0.5" json:"sampling_rate"`                       // 帧采样率 (fps): 0.5=1帧/2s, 1=1帧/1s, 0.2=1帧/5s
	Status          string     `gorm:"type:varchar(20);default:'pending';index" json:"status"` // pending, processing, completed, failed
	CurrentStage    string     `gorm:"type:varchar(50)" json:"current_stage"`                  // extracting, detecting, generating
	FramesProcessed int        `gorm:"default:0" json:"frames_processed"`                      // 已处理帧数
	TotalFrames     int        `gorm:"default:0" json:"total_frames"`                          // 总帧数
	Percentage      int        `gorm:"default:0" json:"percentage"`                            // 进度百分比
	ResultPPTFileID *uint      `json:"result_ppt_file_id,omitempty"`
	ResultPPTFile   *PPTFile   `gorm:"foreignKey:ResultPPTFileID" json:"result_ppt_file,omitempty"`
	ErrorMessage    string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedBy       uint       `gorm:"not null" json:"created_by"`
	Creator         *User      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Mode            string     `gorm:"type:varchar(20);default:'local'" json:"mode"`     // local, cloud
	CloudTaskID     string     `gorm:"type:varchar(100)" json:"cloud_task_id,omitempty"` // Tingwu task ID
	OSSURL          string     `gorm:"type:varchar(500)" json:"oss_url,omitempty"`       // OSS presigned URL
	SlideTimestamps string     `gorm:"type:text" json:"slide_timestamps,omitempty"`     // JSON: [{"slide_number":1,"timestamp":0.0},...]
}

// 转录状态常量
const (
	TranscriptionStatusPending    = "pending"
	TranscriptionStatusProcessing = "processing"
	TranscriptionStatusCompleted  = "completed"
	TranscriptionStatusFailed     = "failed"
)

// 转录阶段常量
const (
	TranscriptionStageExtracting = "extracting"
	TranscriptionStageDetecting  = "detecting"
	TranscriptionStageGenerating = "generating"
)

// 转录模式常量
const (
	TranscriptionModeLocal = "local"
	TranscriptionModeCloud = "cloud"
)

// 云端转录阶段常量
const (
	TranscriptionStageUploading   = "uploading"
	TranscriptionStageQueued      = "queued"
	TranscriptionStageProcessing  = "cloud_processing" // avoid collision with existing "processing" status
	TranscriptionStageDownloading = "downloading"
)

// TableName 指定表名
func (TranscriptionTask) TableName() string {
	return "transcription_tasks"
}

// GetSlideTimestamps parses SlideTimestamps JSON field and returns slice of SlideTimestamp
// Returns empty array if JSON is empty or invalid (graceful degradation)
func (t *TranscriptionTask) GetSlideTimestamps() ([]SlideTimestamp, error) {
	if t.SlideTimestamps == "" {
		return []SlideTimestamp{}, nil
	}

	var timestamps []SlideTimestamp
	if err := json.Unmarshal([]byte(t.SlideTimestamps), &timestamps); err != nil {
		// Return empty array on parse error instead of failing
		return []SlideTimestamp{}, nil
	}

	// Filter out invalid entries (negative slide numbers or timestamps)
	validTimestamps := make([]SlideTimestamp, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.SlideNumber > 0 && ts.Timestamp >= 0 {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	return validTimestamps, nil
}

// SetSlideTimestamps serializes slide timestamps to JSON and stores in SlideTimestamps field
func (t *TranscriptionTask) SetSlideTimestamps(timestamps []SlideTimestamp) error {
	// Filter out invalid entries
	validTimestamps := make([]SlideTimestamp, 0, len(timestamps))
	for _, ts := range timestamps {
		if ts.SlideNumber > 0 && ts.Timestamp >= 0 {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	data, err := json.Marshal(validTimestamps)
	if err != nil {
		return fmt.Errorf("failed to marshal slide timestamps: %w", err)
	}

	t.SlideTimestamps = string(data)
	return nil
}

// GetTimestampForSlide returns timestamp for specific slide number
// Returns error if slide number not found
func (t *TranscriptionTask) GetTimestampForSlide(slideNumber int) (float64, error) {
	timestamps, err := t.GetSlideTimestamps()
	if err != nil {
		return 0, err
	}

	for _, ts := range timestamps {
		if ts.SlideNumber == slideNumber {
			return ts.Timestamp, nil
		}
	}

	return 0, errors.New("slide not found")
}

// AddSlideTimestamp adds or updates a timestamp for a specific slide number
func (t *TranscriptionTask) AddSlideTimestamp(slideNumber int, timestamp float64) error {
	// Validate inputs
	if slideNumber <= 0 {
		return fmt.Errorf("slide number must be positive, got %d", slideNumber)
	}
	if timestamp < 0 {
		return fmt.Errorf("timestamp must be non-negative, got %f", timestamp)
	}

	timestamps, err := t.GetSlideTimestamps()
	if err != nil {
		return err
	}

	// Check if slide number already exists, update if so
	found := false
	for i, ts := range timestamps {
		if ts.SlideNumber == slideNumber {
			timestamps[i].Timestamp = timestamp
			found = true
			break
		}
	}

	// If not found, add new entry
	if !found {
		timestamps = append(timestamps, SlideTimestamp{
			SlideNumber: slideNumber,
			Timestamp:   timestamp,
		})
		// Sort by slide number for consistency
		sort.Slice(timestamps, func(i, j int) bool {
			return timestamps[i].SlideNumber < timestamps[j].SlideNumber
		})
	}

	return t.SetSlideTimestamps(timestamps)
}
