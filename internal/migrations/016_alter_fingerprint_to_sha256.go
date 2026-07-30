package migrations

import (
	"fmt"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"gorm.io/gorm"
)

// Migration016AlterFingerprintToSHA256 widens the persisted file fingerprint.
type Migration016AlterFingerprintToSHA256 struct{}

func (m *Migration016AlterFingerprintToSHA256) Name() string {
	return "016_alter_fingerprint_to_sha256"
}

func (m *Migration016AlterFingerprintToSHA256) Up(db *gorm.DB) error {
	// SEC-006: fingerprint 升级 SHA-256，旧 MD5 指纹失效。
	// 实际 schema 沿用历史列名 uploaded_files.file_md5；等价 SQL 为：
	// ALTER TABLE uploaded_files ALTER COLUMN file_md5 TYPE VARCHAR(64) (fingerprint)
	if !db.Migrator().HasTable(&models.UploadedFile{}) {
		return nil
	}
	if err := db.Migrator().AlterColumn(&models.UploadedFile{}, "FileMD5"); err != nil {
		return fmt.Errorf("widen uploaded file fingerprint: %w", err)
	}
	return nil
}

func (m *Migration016AlterFingerprintToSHA256) Down(db *gorm.DB) error {
	return nil
}
