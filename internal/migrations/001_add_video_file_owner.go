package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// AddVideoFileOwnerMigration 为 video_files 表添加 created_by 字段
type AddVideoFileOwnerMigration struct{}

func (m *AddVideoFileOwnerMigration) Name() string {
	return "001_add_video_file_owner"
}

func (m *AddVideoFileOwnerMigration) Up(db *gorm.DB) error {
	// 使用 PRAGMA table_info 检查列是否已存在（更可靠）
	var columnName string
	checkErr := db.Raw("SELECT name FROM pragma_table_info('video_files') WHERE name = 'created_by'").Scan(&columnName).Error

	// 如果查询成功且找到了列，说明已存在，跳过迁移
	if checkErr == nil && columnName != "" {
		return nil
	}

	// 列不存在，执行添加
	// SQLite 不支持 IF NOT EXISTS 语法，需要先检查
	addResult := db.Exec("ALTER TABLE video_files ADD COLUMN created_by INTEGER NOT NULL DEFAULT 1")
	if addResult.Error != nil {
		// 检查是否是"重复列名"错误，如果是则忽略（幂等）
		if addResult.Error != nil && len(addResult.Error.Error()) > 0 {
			errStr := addResult.Error.Error()
			// SQLite 的重复列错误: "duplicate column name: xxx"
			if contains(errStr, "duplicate column name") {
				return nil // 列已存在，忽略错误
			}
		}
		return addResult.Error
	}

	return nil
}

func (m *AddVideoFileOwnerMigration) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN，需要重建表
	// 这里简单处理：不执行回滚
	return nil
}

// contains 简单的字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// AddStreamConfigMigration 添加流媒体配置字段
type AddStreamConfigMigration struct{}

func (m *AddStreamConfigMigration) Name() string {
	return "002_add_stream_config"
}

func (m *AddStreamConfigMigration) Up(db *gorm.DB) error {
	// 第一部分：添加流媒体字段到 huawei_configs 表
	// 检查列是否已存在
	var columnName string
	checkErr := db.Raw("SELECT name FROM pragma_table_info('huawei_configs') WHERE name = 'stream_protocol'").Scan(&columnName).Error

	// 只有当列不存在时才添加
	if checkErr != nil || columnName == "" {
		addResult := db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_protocol VARCHAR(20)")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}

		addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_url VARCHAR(500)")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}

		addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_username VARCHAR(100)")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}

		addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_password VARCHAR(100)")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}

		addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_enabled BOOLEAN DEFAULT 0")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	// 第二部分：创建或验证 task_huawei_configs 表
	// 检查表是否已存在并验证结构
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_huawei_configs'").Scan(&count)

	if count == 0 {
		// 表不存在，直接创建
		err := db.Exec(`
			CREATE TABLE task_huawei_configs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				task_id INTEGER NOT NULL,
				huawei_config_id INTEGER NOT NULL,
				config_type VARCHAR(20) NOT NULL,
				created_at DATETIME,
				UNIQUE(task_id, huawei_config_id),
				FOREIGN KEY(task_id) REFERENCES video_recording_tasks(id) ON DELETE CASCADE,
				FOREIGN KEY(huawei_config_id) REFERENCES huawei_configs(id) ON DELETE CASCADE
			)
		`).Error
		if err != nil {
			return fmt.Errorf("failed to create task_huawei_configs table: %w", err)
		}
	}

	// 确保索引存在（幂等操作）
	db.Exec("CREATE INDEX IF NOT EXISTS idx_task_huawei_config ON task_huawei_configs(task_id, huawei_config_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_task_huawei_config_task_id ON task_huawei_configs(task_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_task_huawei_config_config_id ON task_huawei_configs(huawei_config_id)")

	return nil
}

func (m *AddStreamConfigMigration) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN，简单处理
	db.Exec("DROP TABLE IF EXISTS task_huawei_configs")
	return nil
}

// AddSegmentFieldsMigration 添加分割段相关字段
type AddSegmentFieldsMigration struct{}

func (m *AddSegmentFieldsMigration) Name() string {
	return "003_add_segment_fields"
}

func (m *AddSegmentFieldsMigration) Up(db *gorm.DB) error {
	// 检查并添加 parent_id 列
	var columnName string
	checkErr := db.Raw("SELECT name FROM pragma_table_info('video_files') WHERE name = 'parent_id'").Scan(&columnName).Error
	if checkErr != nil || columnName == "" {
		addResult := db.Exec("ALTER TABLE video_files ADD COLUMN parent_id INTEGER")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	// 检查并添加 source_type 列
	checkErr = db.Raw("SELECT name FROM pragma_table_info('video_files') WHERE name = 'source_type'").Scan(&columnName).Error
	if checkErr != nil || columnName == "" {
		addResult := db.Exec("ALTER TABLE video_files ADD COLUMN source_type VARCHAR(20) DEFAULT 'recording'")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	// 检查并添加 snapshot_offset 列
	checkErr = db.Raw("SELECT name FROM pragma_table_info('video_files') WHERE name = 'snapshot_offset'").Scan(&columnName).Error
	if checkErr != nil || columnName == "" {
		addResult := db.Exec("ALTER TABLE video_files ADD COLUMN snapshot_offset REAL DEFAULT 0")
		if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	// 创建索引（幂等操作）
	db.Exec("CREATE INDEX IF NOT EXISTS idx_video_files_parent_id ON video_files(parent_id)")

	return nil
}

func (m *AddSegmentFieldsMigration) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN，需要重建表
	// 这里简单处理：不执行回滚
	return nil
}

// isDuplicateColumnError 检查是否是重复列错误
func isDuplicateColumnError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "duplicate column name")
}

// CreateTranscriptionTasksMigration 创建转录任务表
type CreateTranscriptionTasksMigration struct{}

func (m *CreateTranscriptionTasksMigration) Name() string {
	return "004_create_transcription_tasks"
}

func (m *CreateTranscriptionTasksMigration) Up(db *gorm.DB) error {
	// 检查表是否已存在
	var count int64
	checkErr := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='transcription_tasks'").Scan(&count).Error

	// 如果表已存在，跳过迁移
	if checkErr == nil && count > 0 {
		return nil
	}

	// 创建表
	err := db.Exec(`
		CREATE TABLE IF NOT EXISTS transcription_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			video_file_id INTEGER NOT NULL,
			sampling_rate REAL DEFAULT 0.5,
			status VARCHAR(20) DEFAULT 'pending',
			current_stage VARCHAR(50),
			frames_processed INTEGER DEFAULT 0,
			total_frames INTEGER DEFAULT 0,
			percentage INTEGER DEFAULT 0,
			result_ppt_file_id INTEGER,
			error_message TEXT,
			created_by INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);
	`).Error

	if err != nil {
		return fmt.Errorf("failed to create transcription_tasks table: %w", err)
	}

	// 创建索引
	db.Exec("CREATE INDEX IF NOT EXISTS idx_transcription_tasks_video_file ON transcription_tasks(video_file_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_transcription_tasks_status ON transcription_tasks(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_transcription_tasks_deleted_at ON transcription_tasks(deleted_at)")

	return nil
}

func (m *CreateTranscriptionTasksMigration) Down(db *gorm.DB) error {
	db.Exec("DROP TABLE IF EXISTS transcription_tasks")
	return nil
}

// AddPPTCacheFieldsMigration 为 ppt_files 表添加缓存字段
type AddPPTCacheFieldsMigration struct{}

func (m *AddPPTCacheFieldsMigration) Name() string {
	return "005_add_ppt_cache_fields"
}

func (m *AddPPTCacheFieldsMigration) Up(db *gorm.DB) error {
	// 检查 slide_cache_path 列是否已存在
	var columnName string
	checkErr := db.Raw("SELECT name FROM pragma_table_info('ppt_files') WHERE name = 'slide_cache_path'").Scan(&columnName).Error

	// 如果列已存在，跳过迁移
	if checkErr == nil && columnName != "" {
		return nil
	}

	// 添加 slide_cache_path 列
	addResult := db.Exec("ALTER TABLE ppt_files ADD COLUMN slide_cache_path VARCHAR(500) DEFAULT ''")
	if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
		return addResult.Error
	}

	// 添加 source_type 列
	addResult = db.Exec("ALTER TABLE ppt_files ADD COLUMN source_type VARCHAR(20) DEFAULT 'transcription'")
	if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
		return addResult.Error
	}

	// 添加 merged_from 列
	addResult = db.Exec("ALTER TABLE ppt_files ADD COLUMN merged_from TEXT DEFAULT ''")
	if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
		return addResult.Error
	}

	return nil
}

func (m *AddPPTCacheFieldsMigration) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN
	return nil
}

// GetRegisteredMigrations 返回已注册的迁移
func GetRegisteredMigrations() []interface{} {
	return []interface{}{
		&AddVideoFileOwnerMigration{},
		&AddStreamConfigMigration{},
		&AddSegmentFieldsMigration{},
		&CreateTranscriptionTasksMigration{},
		&AddPPTCacheFieldsMigration{},
		&AddSlideTimestampsMigration{},
		&MultiRoleMigration{},
		&AddIPRestrictionsMigration{},
		&DropLegacyRoleIDMigration{},
		&Migration014_CreateInputConfigs{},
		&Migration015_AddTranscriptionJobGroups{},
		&Migration016AlterFingerprintToSHA256{},
	}
}
