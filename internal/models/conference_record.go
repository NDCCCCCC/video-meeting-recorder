package models

import "time"

// ConferenceRecord 会议记录模型
type ConferenceRecord struct {
	Base
	ConferenceNumber string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"conference_number"`
	Title             string    `gorm:"type:varchar(200)" json:"title"`
	StartTime         time.Time `gorm:"index" json:"start_time"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	Status            ConferenceStatus `gorm:"type:varchar(20);index" json:"status"`
	Attendees         int       `gorm:"default:0" json:"attendees"`
	Description       string    `gorm:"type:text" json:"description"`
	HuaweiConfigID    *uint     `json:"huawei_config_id,omitempty"`
	HuaweiConfig      *HuaweiConfig `gorm:"foreignKey:HuaweiConfigID" json:"huawei_config,omitempty"`

	VideoFiles         []VideoFile         `gorm:"foreignKey:ConferenceRecordID" json:"video_files,omitempty"`
	VideoRecordingTask *VideoRecordingTask `gorm:"foreignKey:ConferenceRecordID" json:"video_recording_task,omitempty"`
}

// ConferenceStatus 会议状态
type ConferenceStatus string

const (
	ConferenceStatusNotStarted = "not_started"
	ConferenceStatusInProgress = "in_progress"
	ConferenceStatusCompleted  = "completed"
	ConferenceStatusFailed     = "failed"
)

// GetDuration 获取会议时长
func (c *ConferenceRecord) GetDuration() time.Duration {
	if c.EndTime == nil {
		return 0
	}
	return c.EndTime.Sub(c.StartTime)
}

// IsInProgress 检查会议是否进行中
func (c *ConferenceRecord) IsInProgress() bool {
	return c.Status == ConferenceStatusInProgress
}

// HasRecordings 检查是否有录制
func (c *ConferenceRecord) HasRecordings() bool {
	return len(c.VideoFiles) > 0
}

// GetTotalRecordingSize 获取录制文件总大小
func (c *ConferenceRecord) GetTotalRecordingSize() int64 {
	var total int64
	for _, file := range c.VideoFiles {
		total += file.FileSize
	}
	return total
}

// TableName 指定表名
func (ConferenceRecord) TableName() string {
	return "conference_records"
}
