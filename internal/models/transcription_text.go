package models

// TranscriptionText 转录文字内容模型
type TranscriptionText struct {
	Base
	TranscriptionTaskID uint               `gorm:"not null;index" json:"transcription_task_id"`
	TranscriptionTask   *TranscriptionTask `gorm:"foreignKey:TranscriptionTaskID" json:"transcription_task,omitempty"`
	Text                string             `gorm:"type:text;not null" json:"text"` // 文字内容
	BeginTime           int                `gorm:"not null" json:"begin_time"`     // 开始时间（毫秒）
	EndTime             int                `gorm:"not null" json:"end_time"`       // 结束时间（毫秒）
	SegmentIndex        int                `gorm:"not null" json:"segment_index"`  // 段落序号
}

// TableName 指定表名
func (TranscriptionText) TableName() string {
	return "transcription_texts"
}
