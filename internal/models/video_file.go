package models

import (
	"fmt"
	"os"
	"time"
)

// VideoFile 视频文件模型
type VideoFile struct {
	Base
	FileName       string              `gorm:"type:varchar(200);not null" json:"file_name"`
	FilePath       string              `gorm:"type:varchar(500);not null;uniqueIndex:idx_file_path" json:"file_path"`
	FileSize       int64               `gorm:"default:0" json:"file_size"`         // 字节
	Duration       int                 `gorm:"default:0" json:"duration"`          // 秒
	Format         string              `gorm:"type:varchar(20)" json:"format"`     // mp4, mkv
	Resolution     string              `gorm:"type:varchar(20)" json:"resolution"` // 1920x1080
	Bitrate        int                 `json:"bitrate"`                            // kbps
	Codec          string              `gorm:"type:varchar(50)" json:"codec"`
	TaskID         *uint               `gorm:"index" json:"task_id,omitempty"`                                                     // 关联的录制任务ID
	Task           *VideoRecordingTask `gorm:"foreignKey:TaskID" json:"task,omitempty"`                                            // 关联的录制任务
	ParentID       *uint               `gorm:"index" json:"parent_id,omitempty"`                                                   // 父视频ID（用于分割段、快照）
	SourceType     string              `gorm:"type:varchar(20);default:'recording'" json:"source_type"`                           // recording, snapshot, split
	SnapshotOffset float64             `gorm:"default:0" json:"snapshot_offset"`                                                  // 增量快照偏移量（D-15）
	Parent         *VideoFile          `gorm:"foreignKey:ParentID" json:"parent,omitempty"`                                       // 父视频
	CreatedBy      uint                `gorm:"not null" json:"created_by"`                                                         // 创建者ID
	Creator        *User               `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`                                      // 创建者
	Status         string              `gorm:"type:varchar(20);default:'ready';index:idx_status_created,priority:1" json:"status"` // ready, processing, error
	ThumbnailPath  *string             `json:"thumbnail_path,omitempty"`
	RecordedAt     *time.Time          `gorm:"index" json:"recorded_at,omitempty"`
}

// 文件状态常量
const (
	FileStatusReady      = "ready"
	FileStatusProcessing = "processing"
	FileStatusError      = "error"
	FileStatusDeleting   = "deleting"
)

// 视频来源类型常量
const (
	SourceTypeRecording = "recording"
	SourceTypeSnapshot  = "snapshot"
	SourceTypeSplit     = "split"
)

// GetSizeMB 获取文件大小（MB）
func (v *VideoFile) GetSizeMB() float64 {
	return float64(v.FileSize) / (1024 * 1024)
}

// GetDurationString 获取时长字符串
func (v *VideoFile) GetDurationString() string {
	duration := time.Duration(v.Duration) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

// Exists 检查文件是否存在
func (v *VideoFile) Exists() bool {
	_, err := os.Stat(v.FilePath)
	return err == nil
}

// Delete 删除文件
func (v *VideoFile) Delete() error {
	if err := os.Remove(v.FilePath); err != nil {
		return err
	}
	// 删除缩略图
	if v.ThumbnailPath != nil {
		os.Remove(*v.ThumbnailPath)
	}
	return nil
}

// GetMetadata 获取视频元数据（占位符）
func (v *VideoFile) GetMetadata() (*VideoMetadata, error) {
	// TODO: 使用ffprobe获取视频元数据
	return &VideoMetadata{
		Format:     v.Format,
		Duration:   v.Duration,
		Resolution: v.Resolution,
		Bitrate:    v.Bitrate,
		Codec:      v.Codec,
	}, nil
}

// VideoMetadata 视频元数据
type VideoMetadata struct {
	Format     string `json:"format"`
	Duration   int    `json:"duration"`
	Resolution string `json:"resolution"`
	Bitrate    int    `json:"bitrate"`
	Codec      string `json:"codec"`
}

// TableName 指定表名
func (VideoFile) TableName() string {
	return "video_files"
}
