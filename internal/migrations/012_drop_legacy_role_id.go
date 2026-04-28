package migrations

import (
	"log"

	"gorm.io/gorm"
)

// DropLegacyRoleIDMigration 清理多角色迁移后遗留的 role_id 字段约束
type DropLegacyRoleIDMigration struct{}

func (m *DropLegacyRoleIDMigration) Name() string {
	return "012_drop_legacy_role_id"
}

func (m *DropLegacyRoleIDMigration) Up(db *gorm.DB) error {
	// 检查 users 表是否还有 role_id 列
	hasColumn, err := columnExists(db, "users", "role_id")
	if err != nil {
		return err
	}

	// 如果没有 role_id 列，说明已经迁移过了
	if !hasColumn {
		log.Println("INFO: No legacy role_id column found in users table (already cleaned)")
		return nil
	}

	// 有 role_id 列 - 需要移除 NOT NULL 约束和 FOREIGN KEY 约束
	// SQLite 不支持直接修改约束，需要重建表
	log.Println("INFO: Rebuilding users table to remove role_id constraints")

	// Step 1: 禁用外键约束（SQLite 需要）
	if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		return err
	}
	log.Println("INFO: Disabled foreign key constraints for migration")

	// Step 2: 获取当前表结构
	var currentSQL string
	err = db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&currentSQL).Error
	if err != nil {
		// 恢复外键约束
		db.Exec("PRAGMA foreign_keys = ON")
		return err
	}
	log.Printf("INFO: Current users table DDL: %s", currentSQL)

	// Step 3: 创建新表（role_id 改为 nullable，移除外键约束）
	createNewTable := `
		CREATE TABLE users_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			username VARCHAR(50) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			email VARCHAR(100),
			full_name VARCHAR(100),
			role_id INTEGER,
			is_active NUMERIC DEFAULT 1,
			last_login_at DATETIME,
			allowed_ips TEXT,
			ad_username VARCHAR(100),
			addn VARCHAR(255),
			ad_guid CHAR(36),
			ad_department VARCHAR(100),
			adupn VARCHAR(200),
			last_ad_login DATETIME
		)
	`
	if err := db.Exec(createNewTable).Error; err != nil {
		db.Exec("PRAGMA foreign_keys = ON")
		return err
	}
	log.Println("INFO: Created users_new table")

	// Step 4: 复制数据（除了 role_id，因为多角色数据已在 users_roles 表中）
	copyData := `
		INSERT INTO users_new (
			id, created_at, updated_at, deleted_at, username, password_hash,
			email, full_name, is_active, last_login_at, allowed_ips,
			ad_username, addn, ad_guid, ad_department, adupn, last_ad_login
		)
		SELECT
			id, created_at, updated_at, deleted_at, username, password_hash,
			email, full_name, is_active, last_login_at, allowed_ips,
			ad_username, addn, ad_guid, ad_department, adupn, last_ad_login
		FROM users
	`
	if err := db.Exec(copyData).Error; err != nil {
		// 回滚：删除新表并恢复外键约束
		db.Exec("DROP TABLE IF EXISTS users_new")
		db.Exec("PRAGMA foreign_keys = ON")
		return err
	}
	log.Println("INFO: Copied data from users to users_new")

	// Step 5: 删除旧表并重命名新表
	if err := db.Exec("DROP TABLE users").Error; err != nil {
		// 回滚：删除新表并恢复外键约束
		db.Exec("DROP TABLE IF EXISTS users_new")
		db.Exec("PRAGMA foreign_keys = ON")
		return err
	}
	if err := db.Exec("ALTER TABLE users_new RENAME TO users").Error; err != nil {
		db.Exec("PRAGMA foreign_keys = ON")
		return err
	}
	log.Println("INFO: Replaced users table with new schema (role_id now nullable, no foreign key)")

	// Step 6: 重建索引
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at)").Error; err != nil {
		log.Printf("WARN: Failed to create index: %v", err)
	}
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username)").Error; err != nil {
		log.Printf("WARN: Failed to create index: %v", err)
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_users_ad_guid ON users(ad_guid)").Error; err != nil {
		log.Printf("WARN: Failed to create index: %v", err)
	}
	log.Println("INFO: Recreated indexes")

	// Step 7: 重新启用外键约束
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		log.Printf("WARN: Failed to re-enable foreign keys: %v", err)
	} else {
		log.Println("INFO: Re-enabled foreign key constraints")
	}

	return nil
}

func (m *DropLegacyRoleIDMigration) Down(db *gorm.DB) error {
	// 回滚：无需操作
	log.Println("INFO: Rollback not needed - migration skipped column removal")
	return nil
}
