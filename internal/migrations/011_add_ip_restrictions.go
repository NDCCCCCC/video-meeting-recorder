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
	// Step 1: Add allowed_ips column to users table
	err := db.Exec("ALTER TABLE users ADD COLUMN allowed_ips TEXT").Error
	if err != nil {
		return fmt.Errorf("failed to add allowed_ips column to users: %w", err)
	}

	// Step 2: Add allowed_ips column to roles table
	err = db.Exec("ALTER TABLE roles ADD COLUMN allowed_ips TEXT").Error
	if err != nil {
		return fmt.Errorf("failed to add allowed_ips column to roles: %w", err)
	}

	log.Println("INFO: IP restrictions migration completed")
	return nil
}

func (m *AddIPRestrictionsMigration) Down(db *gorm.DB) error {
	// SQLite doesn't support DROP COLUMN, leave deprecated per multi-role pattern
	log.Println("WARN: Rolling back IP restrictions migration: columns will remain deprecated")
	return nil
}
