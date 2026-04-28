package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// AddADFieldsMigration 添加AD相关字段到users表
type AddADFieldsMigration struct{}

func (m *AddADFieldsMigration) Name() string {
	return "013_add_ad_fields"
}

func (m *AddADFieldsMigration) Up(db *gorm.DB) error {
	// Add AD fields (nullable for local users)
	fields := []struct {
		column string
		typ    string
	}{
		{"ad_username", "VARCHAR(100)"},
		{"ad_dn", "VARCHAR(255)"},
		{"ad_guid", "CHAR(36)"},
		{"ad_department", "VARCHAR(100)"},
		{"ad_upn", "VARCHAR(200)"},
		{"last_ad_login", "DATETIME"},
	}

	for _, field := range fields {
		exists, err := columnExists(db, "users", field.column)
		if err != nil {
			return fmt.Errorf("failed to check %s column in users: %w", field.column, err)
		}
		if !exists {
			if err := db.Exec("ALTER TABLE users ADD COLUMN " + field.column + " " + field.typ).Error; err != nil {
				return fmt.Errorf("failed to add %s column to users: %w", field.column, err)
			}
			log.Println("INFO: Added column " + field.column + " to users table")
		} else {
			log.Println("INFO: " + field.column + " column already exists in users table, skipping")
		}
	}

	// Create index on ad_guid for faster AD user lookups
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_ad_guid ON users(ad_guid)")

	log.Println("INFO: AD fields migration completed")
	return nil
}

func (m *AddADFieldsMigration) Down(db *gorm.DB) error {
	// SQLite doesn't support DROP COLUMN, leave deprecated per multi-role pattern
	log.Println("WARN: Rolling back AD fields migration: columns will remain deprecated")
	return nil
}
