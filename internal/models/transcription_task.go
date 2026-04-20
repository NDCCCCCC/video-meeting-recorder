package models

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
