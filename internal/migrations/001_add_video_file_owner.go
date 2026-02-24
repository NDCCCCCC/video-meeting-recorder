package migrations

import (
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
	// 使用 PRAGMA table_info 检查列是否已存在
	var columnName string
	checkErr := db.Raw("SELECT name FROM pragma_table_info('huawei_configs') WHERE name = 'stream_protocol'").Scan(&columnName).Error

	// 如果查询成功且找到了列，说明已存在，跳过迁移
	if checkErr == nil && columnName != "" {
		return nil
	}

	// 添加流媒体字段到 huawei_configs 表
	addResult := db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_protocol VARCHAR(20)")
	if addResult.Error != nil {
		if !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_url VARCHAR(500)")
	if addResult.Error != nil {
		if !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_username VARCHAR(100)")
	if addResult.Error != nil {
		if !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_password VARCHAR(100)")
	if addResult.Error != nil {
		if !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	addResult = db.Exec("ALTER TABLE huawei_configs ADD COLUMN stream_enabled BOOLEAN DEFAULT 0")
	if addResult.Error != nil {
		if !isDuplicateColumnError(addResult.Error) {
			return addResult.Error
		}
	}

	// 检查关联表是否已存在
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_huawei_configs'").Scan(&count)
	if count == 0 {
		// 创建任务配置关联表
		db.Exec(`
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
		`)
		db.Exec("CREATE INDEX IF NOT EXISTS idx_task_huawei_config ON task_huawei_configs(task_id, huawei_config_id)")
	}

	return nil
}

func (m *AddStreamConfigMigration) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN，简单处理
	db.Exec("DROP TABLE IF EXISTS task_huawei_configs")
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

// GetRegisteredMigrations 返回已注册的迁移
func GetRegisteredMigrations() []interface{} {
	return []interface{}{
		&AddVideoFileOwnerMigration{},
		&AddStreamConfigMigration{},
	}
}
