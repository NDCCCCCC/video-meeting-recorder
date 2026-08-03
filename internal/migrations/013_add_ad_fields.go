package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// AddADFieldsMigration 添加AD相关字段到users表
type AddADFieldsMigration struct{}

func (m *AddADFieldsMigration) Name() string {
	return "013_add_ad_fields"
}

func (m *AddADFieldsMigration) Up(db *gorm.DB) error {
	// SEC-005: SQL 注入防护 — column/field 全部走硬编码白名单与 GORM Migrator。
	fields := []struct {
		column    string
		fieldName string
	}{
		{"ad_username", "ADUsername"},
		{"ad_dn", "ADDN"},
		{"ad_guid", "ADGUID"},
		{"ad_department", "ADDepartment"},
		{"ad_upn", "ADUPN"},
		{"last_ad_login", "LastADLogin"},
	}

	for _, field := range fields {
		if !db.Migrator().HasColumn(&models.User{}, field.column) {
			if err := db.Migrator().AddColumn(&models.User{}, field.fieldName); err != nil {
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
