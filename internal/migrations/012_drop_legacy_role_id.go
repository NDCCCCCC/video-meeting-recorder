package migrations

import (
	"log"
	"regexp"

	"gorm.io/gorm"
)

// DropLegacyRoleIDMigration 清理多角色迁移后遗留的 role_id 字段约束
type DropLegacyRoleIDMigration struct{}

func (m *DropLegacyRoleIDMigration) Name() string {
	return "012_drop_legacy_role_id"
}

func (m *DropLegacyRoleIDMigration) Up(db *gorm.DB) error {
	// 检查 users 表是否还有 role_id 列
	var columns []struct {
		Name string
	}
	checkErr := db.Raw("SELECT name FROM pragma_table_info('users') WHERE name = 'role_id'").Scan(&columns).Error
	if checkErr != nil {
		return checkErr
	}

	// 如果没有 role_id 列，跳过
	if len(columns) == 0 {
		log.Println("INFO: No legacy role_id column found in users table (already cleaned)")
		return nil
	}

	// SQLite 不支持 ALTER TABLE DROP COLUMN，需要重建表
	// 但由于风险较高，我们只将该列设置为可空并更新现有数据为 NULL

	// Step 1: 检查 role_id 是否有 NOT NULL 约束
	var nullableInfo string
	db.Raw("SELECT \"notnull\" FROM pragma_table_info('users') WHERE name = 'role_id'").Scan(&nullableInfo)

	if nullableInfo == "1" {
		// 有 NOT NULL 约束，需要先重建表来移除约束
		log.Println("INFO: Dropping NOT NULL constraint from users.role_id (SQLite requires table recreation)")

		// 使用 SQLite 模式来重建表（危险操作，谨慎处理）
		// 由于 ALTER TABLE DROP COLUMN 不支持，我们创建一个没有 role_id 的新表

		// 获取创建表的 SQL
		var createSQL string
		db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&createSQL)

		// 移除 role_id 列定义
		re := regexp.MustCompile(`,\s*role_id\s+INTEGER[^,]*`)
		newCreateSQL := re.ReplaceAllString(createSQL, "")

		// 创建临时表
		tempTable := "users_new"
		if err := db.Exec("DROP TABLE IF EXISTS " + tempTable).Error; err != nil {
			return err
		}

		// 使用修改后的 DDL 创建新表
		createSQL = regexp.MustCompile(`CREATE TABLE users`).ReplaceAllString(newCreateSQL, "CREATE TABLE "+tempTable)
		if err := db.Exec(createSQL).Error; err != nil {
			return err
		}

		// 复制数据（排除 role_id 列）
		columnsToCopy := []string{
			"id", "created_at", "updated_at", "username", "password_hash",
			"email", "full_name", "is_active", "last_login_at", "allowed_ips",
		}
		columnsStr := ""
		for i, col := range columnsToCopy {
			if i > 0 {
				columnsStr += ", "
			}
			columnsStr += col
		}

		if err := db.Exec("INSERT INTO " + tempTable + " (" + columnsStr + ") SELECT " + columnsStr + " FROM users").Error; err != nil {
			return err
		}

		// 删除旧表
		if err := db.Exec("DROP TABLE users").Error; err != nil {
			return err
		}

		// 重命名新表
		if err := db.Exec("ALTER TABLE " + tempTable + " RENAME TO users").Error; err != nil {
			return err
		}

		// 重建索引
		db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username)")
		db.Exec("CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)")

		log.Println("INFO: Legacy role_id column dropped from users table")
	} else {
		// 没有 NOT NULL 约束，直接清空数据
		log.Println("INFO: Clearing legacy role_id data (column exists but nullable)")
		db.Exec("UPDATE users SET role_id = NULL WHERE role_id IS NOT NULL")
	}

	return nil
}

func (m *DropLegacyRoleIDMigration) Down(db *gorm.DB) error {
	// 回滚：重新添加 role_id 列（但注意会丢失数据）
	log.Println("WARN: Rolling back: re-adding role_id column (data will be lost)")
	return db.Exec("ALTER TABLE users ADD COLUMN role_id INTEGER").Error
}
