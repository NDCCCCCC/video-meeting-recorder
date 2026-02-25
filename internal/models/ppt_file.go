package models

// PPTFile PPT文件模型
type PPTFile struct {
	Base
	FileName  string `gorm:"type:varchar(200);not null" json:"file_name"`
	FilePath  string `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize  int64  `gorm:"default:0" json:"file_size"`
	PageCount int    `gorm:"default:0" json:"page_count"`
	Format    string `gorm:"type:varchar(20)" json:"format"` // pptx
	// 移除 ConferenceRecord 关联
	// 直接关联 VideoFile，删除 VideoFile 时不会自动级联删除 PPTFile
	// 如需级联删除，请在外部手动处理或添加数据库级联约束
	SourceVideoFileID *uint      `json:"source_video_file_id,omitempty"`
	SourceVideoFile   *VideoFile `gorm:"foreignKey:SourceVideoFileID" json:"source_video_file,omitempty"`
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
