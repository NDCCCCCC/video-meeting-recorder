package models

import "time"

// AdminMigrationJob PR-F: 异步 admin 任务(将 huawei_configs 迁移加密到 input_configs)。
// 模板来自 transcription_job_group.go — 同样用 status + 计数推进 + StartedAt/FinishedAt 时间戳。
// 表名: admin_migration_jobs。
type AdminMigrationJob struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Status      string     `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	Total       int        `gorm:"default:0" json:"total"`
	Migrated    int        `gorm:"default:0" json:"migrated"`
	Skipped     int        `gorm:"default:0" json:"skipped"`
	RequestedBy uint       `gorm:"not null;index" json:"requested_by"`
	StartedAt   time.Time  `json:"started_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// 表名
func (AdminMigrationJob) TableName() string {
	return "admin_migration_jobs"
}

// 状态常量 — 与 transcription_job_group 保持语义一致。
const (
	AdminMigrationStatusPending   = "pending"
	AdminMigrationStatusRunning   = "running"
	AdminMigrationStatusCompleted = "completed"
	AdminMigrationStatusFailed    = "failed"
)

// GetPercentage 用于 UI 进度条展示。
func (j *AdminMigrationJob) GetPercentage() int {
	if j.Total == 0 {
		return 0
	}
	return ((j.Migrated + j.Skipped) * 100) / j.Total
}

// IsTerminal 判断是否已结束 (无需再等)。
func (j *AdminMigrationJob) IsTerminal() bool {
	return j.Status == AdminMigrationStatusCompleted || j.Status == AdminMigrationStatusFailed
}
