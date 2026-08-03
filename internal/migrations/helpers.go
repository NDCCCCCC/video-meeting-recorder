package migrations

import (
	"gorm.io/gorm"
)

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
