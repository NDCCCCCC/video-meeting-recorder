package migrations

import (
	"fmt"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"gorm.io/gorm"
)

// Migration017_CreateAdminMigrationJobs PR-F: 创建 admin_migration_jobs 表。
// 用于异步管理类任务 (huawei_configs -> encrypted input_configs 迁移) 的进度跟踪。
// 由 admin_handler.SubmitAdminMigration 与 GetAdminMigrationStatus 共同使用。
type Migration017_CreateAdminMigrationJobs struct{}

func (m *Migration017_CreateAdminMigrationJobs) Name() string {
	return "017_create_admin_migration_jobs"
}

func (m *Migration017_CreateAdminMigrationJobs) Up(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='admin_migration_jobs'").Scan(&count)
	if count == 0 {
		err := db.Exec(`
			CREATE TABLE admin_migration_jobs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				status VARCHAR(20) DEFAULT 'pending',
				total INTEGER DEFAULT 0,
				migrated INTEGER DEFAULT 0,
				skipped INTEGER DEFAULT 0,
				requested_by INTEGER NOT NULL,
				started_at DATETIME,
				finished_at DATETIME,
				error_msg TEXT,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(requested_by) REFERENCES users(id)
			)
		`).Error
		if err != nil {
			return fmt.Errorf("failed to create admin_migration_jobs table: %w: %w", apperrors.ErrInternal, err)
		}
		db.Exec("CREATE INDEX IF NOT EXISTS idx_admin_migration_jobs_status ON admin_migration_jobs(status)")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_admin_migration_jobs_requested_by ON admin_migration_jobs(requested_by)")
	}
	return nil
}

func (m *Migration017_CreateAdminMigrationJobs) Down(db *gorm.DB) error {
	db.Exec("DROP INDEX IF EXISTS idx_admin_migration_jobs_status")
	db.Exec("DROP INDEX IF EXISTS idx_admin_migration_jobs_requested_by")
	db.Exec("DROP TABLE IF EXISTS admin_migration_jobs")
	return nil
}
