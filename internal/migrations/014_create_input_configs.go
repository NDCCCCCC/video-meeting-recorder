package migrations

import (
	"fmt"

	"gorm.io/gorm"
)

// Migration014_CreateInputConfigs 创建 input_configs 和 task_input_configs 表
type Migration014_CreateInputConfigs struct{}

func (m *Migration014_CreateInputConfigs) Name() string {
	return "014_create_input_configs"
}

func (m *Migration014_CreateInputConfigs) Up(db *gorm.DB) error {
	// 第一部分：创建 input_configs 表
	var count int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='input_configs'").Scan(&count)
	if count == 0 {
		// 创建 input_configs 表
		err := db.Exec(`
			CREATE TABLE input_configs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				created_at DATETIME,
				updated_at DATETIME,
				deleted_at DATETIME,
				name VARCHAR(100) NOT NULL,
				description TEXT,
				config_type VARCHAR(20) NOT NULL,
				huawei_enabled BOOLEAN DEFAULT 0,
				server VARCHAR(100),
				port INTEGER DEFAULT 80,
				username VARCHAR(50),
				password VARCHAR(100),
				terminal_number VARCHAR(50),
				conference_number VARCHAR(50),
				camera_backend VARCHAR(20) DEFAULT 'dshow',
				usb_camera_name VARCHAR(100),
				usb_camera_device VARCHAR(100),
				camera_binding_status VARCHAR(20) DEFAULT 'unbound',
				audio_backend VARCHAR(20) DEFAULT 'dshow',
				usb_audio_name VARCHAR(100),
				usb_audio_device VARCHAR(100),
				audio_binding_status VARCHAR(20) DEFAULT 'unbound',
				output_format VARCHAR(20) DEFAULT 'mp4',
				stream_protocol VARCHAR(20),
				stream_url VARCHAR(500),
				stream_username VARCHAR(100),
				stream_password VARCHAR(100),
				stream_enabled BOOLEAN DEFAULT 0,
				is_active BOOLEAN DEFAULT 1,
				is_locked BOOLEAN DEFAULT 0,
				locked_by INTEGER,
				locked_at DATETIME
			)
		`).Error
		if err != nil {
			return fmt.Errorf("failed to create input_configs table: %w", err)
		}

		// 创建索引
		db.Exec("CREATE INDEX IF NOT EXISTS idx_input_configs_config_type ON input_configs(config_type)")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_input_configs_deleted_at ON input_configs(deleted_at)")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_input_configs_is_active ON input_configs(is_active)")
	}

	// 第二部分：创建 task_input_configs 关联表
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_input_configs'").Scan(&count)
	if count == 0 {
		err := db.Exec(`
			CREATE TABLE task_input_configs (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				task_id INTEGER NOT NULL,
				input_config_id INTEGER NOT NULL,
				config_type VARCHAR(20) NOT NULL,
				created_at DATETIME,
				UNIQUE(task_id, input_config_id),
				FOREIGN KEY(task_id) REFERENCES video_recording_tasks(id) ON DELETE CASCADE,
				FOREIGN KEY(input_config_id) REFERENCES input_configs(id) ON DELETE CASCADE
			)
		`).Error
		if err != nil {
			return fmt.Errorf("failed to create task_input_configs table: %w", err)
		}

		// 创建索引
		db.Exec("CREATE INDEX IF NOT EXISTS idx_task_input_config ON task_input_configs(task_id, input_config_id)")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_task_input_configs_task_id ON task_input_configs(task_id)")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_task_input_configs_config_id ON task_input_configs(input_config_id)")
	}

	return nil
}

func (m *Migration014_CreateInputConfigs) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN，所以删除表
	db.Exec("DROP TABLE IF EXISTS task_input_configs")
	db.Exec("DROP TABLE IF EXISTS input_configs")
	return nil
}
