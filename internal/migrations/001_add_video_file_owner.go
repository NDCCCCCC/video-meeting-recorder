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
	// 检查迁移是否已运行
	// 使用简单的方式检查：尝试查询列，如果失败说明列不存在
	var dummy int
	scanErr := db.Raw("SELECT created_by FROM video_files LIMIT 1").Scan(&dummy)

	// 如果查询成功，说明列已存在，跳过迁移
	if scanErr == nil {
		// 列已存在，幂等返回
		return nil
	}

	// 查询失败说明列不存在，执行添加
	// db.Exec 返回 *gorm.DB，需要检查其 Error 字段
	addResult := db.Exec("ALTER TABLE video_files ADD COLUMN created_by INTEGER NOT NULL DEFAULT 1")
	if addResult.Error != nil {
		return addResult.Error
	}

	// 为已存在的记录设置默认值
	_ = db.Exec("UPDATE video_files SET created_by = 1 WHERE created_by IS NULL")

	return nil
}

func (m *AddVideoFileOwnerMigration) Down(db *gorm.DB) error {
	dropResult := db.Exec("ALTER TABLE video_files DROP COLUMN created_by")
	if dropResult.Error != nil {
		return dropResult.Error
	}
	return nil
}

// GetRegisteredMigrations 返回已注册的迁移
func GetRegisteredMigrations() []interface{} {
	return []interface{}{
		&AddVideoFileOwnerMigration{},
	}
}
