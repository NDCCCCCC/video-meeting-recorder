package migrations

import (
	"log"

	"gorm.io/gorm"
)

// DropLegacyRoleIDMigration 删除遗留的 role_id 字段
// SQLite 3.35.0+ 支持 ALTER TABLE DROP COLUMN
type DropLegacyRoleIDMigration struct{}

func (m *DropLegacyRoleIDMigration) Name() string {
	return "012_drop_legacy_role_id"
}

func (m *DropLegacyRoleIDMigration) Up(db *gorm.DB) error {
	// 检查 role_id 列是否存在
	hasColumn, err := columnExists(db, "users", "role_id")
	if err != nil {
		return err
	}

	if !hasColumn {
		log.Println("INFO: No role_id column found in users table (already cleaned)")
		return nil
	}

	// 检查 SQLite 版本
	var version string
	err = db.Raw("SELECT sqlite_version()").Scan(&version).Error
	if err != nil {
		return err
	}

	log.Printf("INFO: SQLite version: %s", version)

	// SQLite 3.35.0+ 支持 DROP COLUMN
	// 使用 ALTER TABLE DROP COLUMN 直接删除列
	log.Println("INFO: Dropping role_id column using ALTER TABLE DROP COLUMN")
	err = db.Exec("ALTER TABLE users DROP COLUMN role_id").Error
	if err != nil {
		log.Printf("ERROR: Failed to drop column: %v", err)
		return err
	}

	log.Println("INFO: Successfully dropped role_id column from users table")
	return nil
}

func (m *DropLegacyRoleIDMigration) Down(db *gorm.DB) error {
	// 回滚：重新添加 role_id 列（作为 nullable）
	// 注意：不会恢复数据，只是恢复 schema
	log.Println("INFO: Rollback - adding role_id column (nullable)")
	err := db.Exec("ALTER TABLE users ADD COLUMN role_id INTEGER").Error
	if err != nil {
		return err
	}
	log.Println("INFO: Rollback complete")
	return nil
}
