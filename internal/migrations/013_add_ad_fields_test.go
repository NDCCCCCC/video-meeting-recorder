package migrations

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAddADFieldsMigrationIsIdempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE users (id integer primary key)").Error; err != nil {
		t.Fatal(err)
	}
	migration := &AddADFieldsMigration{}
	if err := migration.Up(db); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	if err := migration.Up(db); err != nil {
		t.Fatalf("second Up: %v", err)
	}
}
