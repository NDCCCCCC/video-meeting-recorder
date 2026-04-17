package models

// PPT source type constants
const (
	PPTSourceTypeTranscription = "transcription"
	PPTSourceTypeMerge         = "merge"
)

// PPTFile PPT文件模型
type PPTFile struct {
	Base
	FileName             string  `gorm:"type:varchar(200);not null" json:"file_name"`
	FilePath             string  `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize             int64   `gorm:"default:0" json:"file_size"`
	PageCount            int     `gorm:"default:0" json:"page_count"`
	Format               string  `gorm:"type:varchar(20)" json:"format"` // pptx
	SlideCachePath       string  `gorm:"type:varchar(500)" json:"slide_cache_path,omitempty"` // Path to extracted slide images cache directory
	SourceType           string  `gorm:"type:varchar(20);default:'transcription'" json:"source_type"` // "transcription" or "merge"
	MergedFrom           string  `gorm:"type:text" json:"merged_from,omitempty"` // JSON array of source PPT IDs, e.g. "[1,3,5]"
	// 移除 ConferenceRecord 关联
	// 直接关联 VideoFile，删除 VideoFile 时不会自动级联删除 PPTFile
	// 如需级联删除，请在外部手动处理或添加数据库级联约束
	SourceVideoFileID    *uint             `json:"source_video_file_id,omitempty"`
	SourceVideoFile      *VideoFile        `gorm:"foreignKey:SourceVideoFileID" json:"source_video_file,omitempty"`
	TranscriptionTaskID  *uint             `json:"transcription_task_id,omitempty"`
	TranscriptionTask    *TranscriptionTask `gorm:"foreignKey:TranscriptionTaskID" json:"transcription_task,omitempty"`
}

// GenerateFromVideo 从视频生成PPT（占位符实现）
func (p *PPTFile) GenerateFromVideo(videoFile *VideoFile) error {
	// TODO: 实现从视频提取帧并生成PPT的逻辑
	// 1. 使用ffmpeg提取关键帧
	// 2. 使用模板生成PPT
	// 3. 保存PPT文件
	return nil
}

// TableName 指定表名
func (PPTFile) TableName() string {
	return "ppt_files"
}
