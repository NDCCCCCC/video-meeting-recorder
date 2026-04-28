package migrations

import (
	"gorm.io/gorm"
)

// tableExists 检查表是否已存在
func tableExists(db *gorm.DB, tableName string) (bool, error) {
	var count int
	err := db.Raw(`
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type='table' AND name=?
	`, tableName).Scan(&count).Error
	return count > 0, err
}

// columnExists 检查列是否已存在
func columnExists(db *gorm.DB, tableName, columnName string) (bool, error) {
	var count int
	err := db.Raw(`
		SELECT COUNT(*)
		FROM pragma_table_info(?)
		WHERE name = ?
	`, tableName, columnName).Scan(&count).Error
	return count > 0, err
}
