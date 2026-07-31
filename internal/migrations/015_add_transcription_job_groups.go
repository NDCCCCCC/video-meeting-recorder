package migrations

import (
	"fmt"

	"gorm.io/gorm"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// Migration015_AddTranscriptionJobGroups 添加转录任务组表
type Migration015_AddTranscriptionJobGroups struct{}

func (m *Migration015_AddTranscriptionJobGroups) Name() string {
	return "015_add_transcription_job_groups"
}

func (m *Migration015_AddTranscriptionJobGroups) Up(db *gorm.DB) error {
	// 创建 transcription_job_groups 表
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='transcription_job_groups'").Scan(&count)
	if count == 0 {
		err := db.Exec(`
			CREATE TABLE transcription_job_groups (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				user_id INTEGER NOT NULL,
				status VARCHAR(20) DEFAULT 'pending',
				total_count INTEGER DEFAULT 0,
				completed_count INTEGER DEFAULT 0,
				failed_count INTEGER DEFAULT 0,
				created_at DATETIME,
				updated_at DATETIME,
				FOREIGN KEY(user_id) REFERENCES users(id)
			)
		`).Error
		if err != nil {
			return fmt.Errorf("failed to create transcription_job_groups table: %w: %w", apperrors.ErrInternal, err)
		}

		// 创建索引
		db.Exec("CREATE INDEX IF NOT EXISTS idx_transcription_job_groups_user_id ON transcription_job_groups(user_id)")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_transcription_job_groups_status ON transcription_job_groups(status)")
	}

	// 为 transcription_tasks 表添加 job_group_id 字段
	var columnExists bool
	db.Raw("SELECT COUNT(*) > 0 FROM pragma_table_info('transcription_tasks') WHERE name='job_group_id'").Scan(&columnExists)
	if !columnExists {
		if err := db.Exec("ALTER TABLE transcription_tasks ADD COLUMN job_group_id INTEGER").Error; err != nil {
			return fmt.Errorf("failed to add job_group_id column: %w: %w", apperrors.ErrInternal, err)
		}

		// 创建外键索引（SQLite 的外键约束需要手动创建）
		db.Exec("CREATE INDEX IF NOT EXISTS idx_transcription_tasks_job_group_id ON transcription_tasks(job_group_id)")
	}

	return nil
}

func (m *Migration015_AddTranscriptionJobGroups) Down(db *gorm.DB) error {
	db.Exec("DROP INDEX IF EXISTS idx_transcription_tasks_job_group_id")
	// SQLite 不支持 DROP COLUMN，跳过
	db.Exec("DROP TABLE IF EXISTS transcription_job_groups")
	return nil
}
