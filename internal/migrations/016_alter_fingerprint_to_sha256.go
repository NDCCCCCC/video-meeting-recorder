package migrations

import (
	"fmt"

	"gorm.io/gorm"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// Migration016AlterFingerprintToSHA256 widens the persisted file fingerprint.
type Migration016AlterFingerprintToSHA256 struct{}

func (m *Migration016AlterFingerprintToSHA256) Name() string {
	return "016_alter_fingerprint_to_sha256"
}

func (m *Migration016AlterFingerprintToSHA256) Up(db *gorm.DB) error {
	// SEC-006: fingerprint 升级 SHA-256，旧 MD5 指纹失效。
	// 实际 schema 列名经 018 迁移后为 uploaded_files.file_fingerprint；
	// 本迁移仅负责 Varchar 列宽（VARCHAR(32) -> VARCHAR(64)），与列名无关。
	if !db.Migrator().HasTable(&models.UploadedFile{}) {
		return nil
	}
	// 幂等性：仅在列仍以旧名 file_md5 / 新名 file_fingerprint 存在时执行 widen。
	// 改造期间通过 018 完成重命名后，本行需使用新字段名 "FileFingerprint"。
	if err := db.Migrator().AlterColumn(&models.UploadedFile{}, "FileFingerprint"); err != nil {
		return fmt.Errorf("widen uploaded file fingerprint: %w: %w", apperrors.ErrInternal, err)
	}
	return nil
}

func (m *Migration016AlterFingerprintToSHA256) Down(db *gorm.DB) error {
	return nil
}
