package models

import "time"

// TranscriptionJobGroup 转录任务组模型
type TranscriptionJobGroup struct {
	ID             uint                `gorm:"primaryKey" json:"id"`
	UserID         uint                `gorm:"not null;index" json:"user_id"`
	User           *User               `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Status         string              `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	TotalCount     int                 `gorm:"default:0" json:"total_count"`
	CompletedCount int                 `gorm:"default:0" json:"completed_count"`
	FailedCount    int                 `gorm:"default:0" json:"failed_count"`
	Tasks          []TranscriptionTask `gorm:"foreignKey:JobGroupID" json:"tasks,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// 任务组状态常量
const (
	JobGroupStatusPending    = "pending"
	JobGroupStatusProcessing = "processing"
	JobGroupStatusCompleted  = "completed"
	JobGroupStatusFailed     = "failed"
)

// TableName 指定表名
func (TranscriptionJobGroup) TableName() string {
	return "transcription_job_groups"
}

// GetPercentage 获取完成百分比
func (g *TranscriptionJobGroup) GetPercentage() int {
	if g.TotalCount == 0 {
		return 0
	}
	return (g.CompletedCount * 100) / g.TotalCount
}

// IsCompleted 检查是否全部完成
func (g *TranscriptionJobGroup) IsCompleted() bool {
	return g.CompletedCount+g.FailedCount >= g.TotalCount
}

// UpdateStatus 根据任务完成情况更新状态
func (g *TranscriptionJobGroup) UpdateStatus() {
	if g.TotalCount == 0 {
		g.Status = JobGroupStatusPending
		return
	}

	if g.IsCompleted() {
		if g.FailedCount > 0 && g.CompletedCount == 0 {
			g.Status = JobGroupStatusFailed
		} else if g.FailedCount > 0 {
			g.Status = JobGroupStatusCompleted // 部分成功也算完成
		} else {
			g.Status = JobGroupStatusCompleted
		}
	} else if g.CompletedCount > 0 || g.FailedCount > 0 {
		g.Status = JobGroupStatusProcessing
	} else {
		g.Status = JobGroupStatusPending
	}
}
