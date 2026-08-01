package migrations

import (
	"strings"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestMigration018_RenameFileMD5ToFingerprint_Up exercises the rename end-to-end.
//
// 1. Pre-state:  uploaded_files exists with legacy column file_md5 (mocked pre-016 schema).
// 2. Up():       performs RENAME COLUMN file_md5 -> file_fingerprint (preserves data + index).
// 3. Post-state: column file_md5 gone; column file_fingerprint present with old data + index.
func TestMigration018_RenameFileMD5ToFingerprint_Up(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Mock a pre-016 schema: file_md5 column with index, containing one row.
	// Include deleted_at because models.UploadedFile embeds Base (gorm.DeletedAt);
	// otherwise the GORM soft-delete scope adds "WHERE deleted_at IS NULL" and
	// db.First(&file) below fails with "no such column: deleted_at".
	if err := db.Exec(`CREATE TABLE uploaded_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_md5 VARCHAR(64),
		uploaded_by INTEGER NOT NULL DEFAULT 0,
		status VARCHAR(20) DEFAULT 'active',
		deleted_at DATETIME
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}
	if err := db.Exec(`CREATE INDEX idx_uploaded_files_file_md5 ON uploaded_files(file_md5)`).Error; err != nil {
		t.Fatalf("create index: %v", err)
	}
	if err := db.Exec(`INSERT INTO uploaded_files (file_md5, uploaded_by) VALUES ('abcdef0123456789', 7)`).Error; err != nil {
		t.Fatalf("insert row: %v", err)
	}

	// Pre-check: file_md5 present, file_fingerprint absent.
	if count := pragmaColumnCount(t, db, "uploaded_files", "file_md5"); count != 1 {
		t.Fatalf("pre-check: file_md5 column count = %d, want 1", count)
	}
	if count := pragmaColumnCount(t, db, "uploaded_files", "file_fingerprint"); count != 0 {
		t.Fatalf("pre-check: file_fingerprint column count = %d, want 0", count)
	}

	m := &Migration018_RenameFileMD5ToFingerprint{}
	if err := m.Up(db); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Post-check: file_md5 absent, file_fingerprint present (same column, just renamed).
	if count := pragmaColumnCount(t, db, "uploaded_files", "file_md5"); count != 0 {
		t.Errorf("post-check: file_md5 column count = %d, want 0", count)
	}
	if count := pragmaColumnCount(t, db, "uploaded_files", "file_fingerprint"); count != 1 {
		t.Errorf("post-check: file_fingerprint column count = %d, want 1", count)
	}

	// Verify the rename preserved existing data (rename is metadata-only on SQLite).
	var fp string
	if err := db.Raw("SELECT file_fingerprint FROM uploaded_files WHERE uploaded_by = 7").Scan(&fp).Error; err != nil {
		t.Fatalf("select after rename: %v", err)
	}
	if fp != "abcdef0123456789" {
		t.Errorf("preserved data: got %q, want %q", fp, "abcdef0123456789")
	}

	// Verify the GORM model can read the renamed column through the FileFingerprint field.
	var file models.UploadedFile
	if err := db.First(&file).Error; err != nil {
		t.Fatalf("gorm read: %v", err)
	}
	if file.FileFingerprint != "abcdef0123456789" {
		t.Errorf("FileFingerprint field: got %q, want %q", file.FileFingerprint, "abcdef0123456789")
	}

	// Verify the index survived the rename (SQLite renames the index with the column).
	var idxCount int
	if err := db.Raw(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND tbl_name='uploaded_files'`).Scan(&idxCount).Error; err != nil {
		t.Fatalf("index count: %v", err)
	}
	if idxCount < 1 {
		t.Errorf("expected at least 1 index on uploaded_files after rename, got %d", idxCount)
	}
}

// TestMigration018_RenameFileMD5ToFingerprint_Up_idempotent verifies re-running is a no-op.
func TestMigration018_RenameFileMD5ToFingerprint_Up_idempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	if err := db.Exec(`CREATE TABLE uploaded_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_md5 VARCHAR(64)
	)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	m := &Migration018_RenameFileMD5ToFingerprint{}

	// First run: rename happens.
	if err := m.Up(db); err != nil {
		t.Fatalf("first Up: %v", err)
	}

	// Second run: must be a no-op (no error, no further schema change).
	if err := m.Up(db); err != nil {
		t.Fatalf("second Up should be idempotent, got: %v", err)
	}

	if count := pragmaColumnCount(t, db, "uploaded_files", "file_md5"); count != 0 {
		t.Errorf("idempotency: file_md5 unexpectedly resurrected, count = %d", count)
	}
	if count := pragmaColumnCount(t, db, "uploaded_files", "file_fingerprint"); count != 1 {
		t.Errorf("idempotency: file_fingerprint column count = %d, want 1", count)
	}
}

// TestMigration018_RenameFileMD5ToFingerprint_NoTable verifies safety when table is absent.
func TestMigration018_RenameFileMD5ToFingerprint_NoTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	m := &Migration018_RenameFileMD5ToFingerprint{}
	if err := m.Up(db); err != nil {
		t.Errorf("Up on empty db should be no-op, got: %v", err)
	}
}

// pragmaColumnCount returns the number of columns on the given table whose lowercase name
// matches colName. Case-insensitive because SQLite identifiers are normally case-insensitive.
func pragmaColumnCount(t *testing.T, db *gorm.DB, table, colName string) int {
	t.Helper()
	var n int
	if err := db.Raw(
		"SELECT COUNT(*) FROM pragma_table_info(?) WHERE LOWER(name) = LOWER(?)",
		table, colName,
	).Scan(&n).Error; err != nil {
		t.Fatalf("pragma query: %v", err)
	}
	return n
}

// guard against unused-import lint when this file is the only new file in the package.
var _ = strings.HasPrefix
