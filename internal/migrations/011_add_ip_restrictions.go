package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// AddIPRestrictionsMigration 为用户和角色添加IP限制字段
type AddIPRestrictionsMigration struct{}

func (m *AddIPRestrictionsMigration) Name() string {
	return "011_add_ip_restrictions"
}

func (m *AddIPRestrictionsMigration) Up(db *gorm.DB) error {
	// Step 1: Add allowed_ips column to users table (if not exists)
	exists, err := columnExists(db, "users", "allowed_ips")
	if err != nil {
		return fmt.Errorf("failed to check allowed_ips column in users: %w", err)
	}
	if !exists {
		if err := db.Exec("ALTER TABLE users ADD COLUMN allowed_ips TEXT").Error; err != nil {
			return fmt.Errorf("failed to add allowed_ips column to users: %w", err)
		}
		log.Println("INFO: Added allowed_ips column to users table")
	} else {
		log.Println("INFO: allowed_ips column already exists in users table, skipping")
	}

	// Step 2: Add allowed_ips column to roles table (if not exists)
	exists, err = columnExists(db, "roles", "allowed_ips")
	if err != nil {
		return fmt.Errorf("failed to check allowed_ips column in roles: %w", err)
	}
	if !exists {
		if err := db.Exec("ALTER TABLE roles ADD COLUMN allowed_ips TEXT").Error; err != nil {
			return fmt.Errorf("failed to add allowed_ips column to roles: %w", err)
		}
		log.Println("INFO: Added allowed_ips column to roles table")
	} else {
		log.Println("INFO: allowed_ips column already exists in roles table, skipping")
	}

	log.Println("INFO: IP restrictions migration completed")
	return nil
}

func (m *AddIPRestrictionsMigration) Down(db *gorm.DB) error {
	// SQLite doesn't support DROP COLUMN, leave deprecated per multi-role pattern
	log.Println("WARN: Rolling back IP restrictions migration: columns will remain deprecated")
	return nil
}
