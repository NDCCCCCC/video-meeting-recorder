package models

import (
	"encoding/json"
	"fmt"
	"time"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// PPT source type constants
const (
	PPTSourceTypeTranscription = "transcription"
	PPTSourceTypeMerge         = "merge"
)

// EditOperation represents a single edit operation in the edit history
type EditOperation struct {
	Operation string `json:"operation"` // "delete", "rollback", etc.
	Slides    []int  `json:"slides"`    // Affected slide numbers
	Timestamp string `json:"timestamp"` // ISO 8601 timestamp
	UserID    uint   `json:"user_id,omitempty"`
}

// PPTFile PPT文件模型
type PPTFile struct {
	Base
	FileName       string `gorm:"type:varchar(200);not null" json:"file_name"`
	FilePath       string `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize       int64  `gorm:"default:0" json:"file_size"`
	PageCount      int    `gorm:"default:0" json:"page_count"`
	Format         string `gorm:"type:varchar(20)" json:"format"`                              // pptx
	SlideCachePath string `gorm:"type:varchar(500)" json:"slide_cache_path,omitempty"`         // Path to extracted slide images cache directory
	SourceType     string `gorm:"type:varchar(20);default:'transcription'" json:"source_type"` // "transcription" or "merge"
	MergedFrom     string `gorm:"type:text" json:"merged_from,omitempty"`                      // JSON array of source PPT IDs, e.g. "[1,3,5]"
	BackupPath     string `gorm:"type:varchar(500)" json:"backup_path,omitempty"`              // Path to backup PPTX file before editing
	DeletedSlides  string `gorm:"type:text" json:"deleted_slides,omitempty"`                   // JSON array of deleted slide numbers, e.g. "[1,5,10]"
	EditHistory    string `gorm:"type:text" json:"edit_history,omitempty"`                     // JSON array of edit operations
	// 移除 ConferenceRecord 关联
	// 直接关联 VideoFile，删除 VideoFile 时不会自动级联删除 PPTFile
	// 如需级联删除，请在外部手动处理或添加数据库级联约束
	SourceVideoFileID   *uint              `json:"source_video_file_id,omitempty"`
	SourceVideoFile     *VideoFile         `gorm:"foreignKey:SourceVideoFileID" json:"source_video_file,omitempty"`
	TranscriptionTaskID *uint              `json:"transcription_task_id,omitempty"`
	TranscriptionTask   *TranscriptionTask `gorm:"foreignKey:TranscriptionTaskID" json:"transcription_task,omitempty"`
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

// HasBackup checks if a backup exists for this PPT file
func (p *PPTFile) HasBackup() bool {
	return p.BackupPath != ""
}

// GetDeletedSlides parses the DeletedSlides JSON field
func (p *PPTFile) GetDeletedSlides() ([]int, error) {
	if p.DeletedSlides == "" {
		return []int{}, nil
	}

	var slides []int
	err := json.Unmarshal([]byte(p.DeletedSlides), &slides)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deleted slides: %w: %w", apperrors.ErrInternal, err)
	}
	return slides, nil
}

// RecordDeletion records slide deletion by appending to DeletedSlides JSON
func (p *PPTFile) RecordDeletion(slides []int) error {
	// Get existing deleted slides
	existing, err := p.GetDeletedSlides()
	if err != nil {
		return err
	}

	// Merge with new slides (avoid duplicates)
	merged := make(map[int]bool)
	for _, s := range existing {
		merged[s] = true
	}
	for _, s := range slides {
		merged[s] = true
	}

	// Convert back to array
	result := make([]int, 0, len(merged))
	for s := range merged {
		result = append(result, s)
	}

	// Sort and marshal
	sortAndMarshal := func(slides []int) (string, error) {
		// Simple bubble sort for small arrays
		for i := 0; i < len(slides)-1; i++ {
			for j := 0; j < len(slides)-i-1; j++ {
				if slides[j] > slides[j+1] {
					slides[j], slides[j+1] = slides[j+1], slides[j]
				}
			}
		}
		data, err := json.Marshal(slides)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}

	p.DeletedSlides, err = sortAndMarshal(result)
	return err
}

// AddEditOperation adds an operation to the edit history
func (p *PPTFile) AddEditOperation(operation string, slides []int) error {
	// Parse existing history
	var history []EditOperation
	if p.EditHistory != "" && p.EditHistory != "[]" {
		if err := json.Unmarshal([]byte(p.EditHistory), &history); err != nil {
			return fmt.Errorf("failed to parse edit history: %w: %w", apperrors.ErrInternal, err)
		}
	}

	// Create new operation
	newOp := EditOperation{
		Operation: operation,
		Slides:    slides,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Append and marshal
	history = append(history, newOp)
	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal edit history: %w: %w", apperrors.ErrInternal, err)
	}

	p.EditHistory = string(data)
	return nil
}
