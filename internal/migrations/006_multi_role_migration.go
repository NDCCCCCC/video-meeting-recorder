package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// MultiRoleMigration 创建 users_roles 关联表并迁移现有单角色数据
type MultiRoleMigration struct{}

func (m *MultiRoleMigration) Name() string {
	return "006_multi_role_migration"
}

func (m *MultiRoleMigration) Up(db *gorm.DB) error {
	// Step 1: Check if users_roles table already exists (idempotent migration)
	var count int64
	checkErr := db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users_roles'").Scan(&count).Error
	if checkErr != nil {
		return fmt.Errorf("failed to check users_roles table existence: %w", checkErr)
	}

	// If exists, return nil (idempotent)
	if count > 0 {
		return nil
	}

	// Step 2: Create users_roles junction table
	err := db.Exec(`
		CREATE TABLE users_roles (
			user_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			PRIMARY KEY (user_id, role_id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY(role_id) REFERENCES roles(id) ON DELETE CASCADE
		)
	`).Error
	if err != nil {
		return fmt.Errorf("failed to create users_roles table: %w", err)
	}

	// Step 3: Migrate existing single-role data (D-08)
	// Copy users.role_id → users_roles for users with valid roles
	migrationResult := db.Exec(`
		INSERT INTO users_roles (user_id, role_id, created_at, updated_at)
		SELECT id as user_id, role_id, datetime('now'), datetime('now')
		FROM users
		WHERE role_id IS NOT NULL AND role_id > 0
	`)
	if migrationResult.Error != nil {
		return fmt.Errorf("failed to migrate user roles: %w", migrationResult.Error)
	}

	// Step 4: Verify migration (D-18)
	// Count migrated roles vs total users to detect data loss
	var migratedCount int64
	db.Raw("SELECT COUNT(*) FROM users_roles").Scan(&migratedCount)

	var userCount int64
	var usersWithRoles int64
	db.Model(&UserStruct{}).Count(&userCount)
	db.Raw("SELECT COUNT(*) FROM users WHERE role_id IS NOT NULL AND role_id > 0").Scan(&usersWithRoles)

	// Log warning if mismatch (some users might have no role)
	if migratedCount < usersWithRoles {
		log.Printf("WARN: Multi-role migration: some users may have lost roles (users_with_roles=%d, migrated_roles=%d, total_users=%d)",
			usersWithRoles, migratedCount, userCount)
	}

	log.Printf("INFO: Multi-role migration completed (users_migrated=%d, total_users=%d)",
		migratedCount, userCount)

	// Step 5: Create indexes for performance
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_roles_user_id ON users_roles(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_roles_role_id ON users_roles(role_id)")

	// Step 6: Deprecate role_id column (set to NULL)
	// SQLite doesn't support DROP COLUMN, leaving deprecated per D-16
	db.Exec("UPDATE users SET role_id = NULL")

	return nil
}

func (m *MultiRoleMigration) Down(db *gorm.DB) error {
	// Rollback: Restore first role from users_roles back to users.role_id (lossy for multi-role)
	log.Printf("WARN: Rolling back multi-role migration: users with multiple roles will lose additional roles")

	restoreResult := db.Exec(`
		UPDATE users
		SET role_id = (
			SELECT role_id FROM users_roles
			WHERE user_id = users.id
			LIMIT 1
		)
	`)
	if restoreResult.Error != nil {
		return fmt.Errorf("failed to restore single role: %w", restoreResult.Error)
	}

	// Drop users_roles table
	db.Exec("DROP TABLE IF EXISTS users_roles")

	return nil
}

// UserStruct is a minimal struct for counting users during migration
type UserStruct struct {
	ID uint
}

func (UserStruct) TableName() string {
	return "users"
}
