package migrations

import (
	"fmt"

	"gorm.io/gorm"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
)

// Migration018_RenameFileMD5ToFingerprint renames the legacy column
// uploaded_files.file_md5 -> uploaded_files.file_fingerprint.
//
// 背景: SEC-006 已通过 migration 016 把字段长度拓宽到 VARCHAR(64) (SHA-256),
// 但遗留了误导性的列名 file_md5。本迁移把列名同步到命名约定 (FileFingerprint)。
//
// 幂等性: 应用启动时 GetRegisteredMigrations() 会按顺序对每个 migration 调用 Up()
// (无外部 migrations 表跟踪完成状态),因此 Up() 必须自幂等:
//  1. 表不存在 -> 直接返回 nil
//  2. file_md5 列已不存在 (已被本迁移或手工重命名) -> 直接返回 nil
//  3. file_fingerprint 列已存在 (下游 DDL 已先行变更) -> 不再 RENAME,避免破坏列宽
//  4. file_md5 存在且 file_fingerprint 缺失 -> 执行 SQLite RENAME COLUMN
//
// SQLite 3.25+ 支持 ALTER TABLE ... RENAME COLUMN (语法兼容 MySQL 8.0+)；
// 本项目实际只用 SQLite (go.mod 无 mysql 驱动),data/*.db 为 SQLite 3.53+。
type Migration018_RenameFileMD5ToFingerprint struct{}

func (m *Migration018_RenameFileMD5ToFingerprint) Name() string {
	return "018_rename_file_md5_to_fingerprint"
}

func (m *Migration018_RenameFileMD5ToFingerprint) Up(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.UploadedFile{}) {
		return nil
	}

	// 用 column 名（而非 Go 字段名 "FileMD5"）查询：字段在代码层已重命名为
	// FileFingerprint,LookupField 不会命中旧名,从而 pragma_table_info 走字面量比较。
	if !db.Migrator().HasColumn(&models.UploadedFile{}, "file_md5") {
		return nil
	}
	if db.Migrator().HasColumn(&models.UploadedFile{}, "file_fingerprint") {
		// 不变量冲突：旧列存在但新列也已存在。提示 DBA 介入，避免静默丢弃旧列。
		return fmt.Errorf(
			"uploaded_files 同时存在 file_md5 与 file_fingerprint：%w",
			apperrors.ErrInternal,
		)
	}

	// SQLite 3.25+ / MySQL 8.0+ 语法：保留原列属性 (VARCHAR(64), NULL, index)，
	// 不会触发表重建,优于 ALTER TABLE ... CHANGE COLUMN。
	if err := db.Exec(
		"ALTER TABLE uploaded_files RENAME COLUMN file_md5 TO file_fingerprint",
	).Error; err != nil {
		return fmt.Errorf("rename uploaded_files.file_md5 -> file_fingerprint: %w: %w",
			apperrors.ErrInternal, err)
	}
	return nil
}

func (m *Migration018_RenameFileMD5ToFingerprint) Down(db *gorm.DB) error {
	// 不可逆 DDL (列名回退需重建表,代价高且无业务收益),保留 no-op 与 016 一致。
	return nil
}
