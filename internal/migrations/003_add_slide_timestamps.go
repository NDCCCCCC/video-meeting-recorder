package migrations

import (
	"fmt"

	"github.com/cpic/record_v2/internal/models"
	"gorm.io/gorm"
)

// AddSlideTimestampsMigration 添加 slide_timestamps 字段到 transcription_tasks 表
type AddSlideTimestampsMigration struct{}

func (m *AddSlideTimestampsMigration) Name() string {
	return "006_add_slide_timestamps"
}

func (m *AddSlideTimestampsMigration) Up(db *gorm.DB) error {
	// 检查 slide_timestamps 列是否已存在
	var columnName string
	checkErr := db.Raw("SELECT name FROM pragma_table_info('transcription_tasks') WHERE name = 'slide_timestamps'").Scan(&columnName).Error

	// 如果列已存在，跳过迁移
	if checkErr == nil && columnName != "" {
		return nil
	}

	// 添加 slide_timestamps 列
	addResult := db.Exec("ALTER TABLE transcription_tasks ADD COLUMN slide_timestamps TEXT DEFAULT ''")
	if addResult.Error != nil && !isDuplicateColumnError(addResult.Error) {
		return fmt.Errorf("failed to add slide_timestamps column: %w", addResult.Error)
	}

	return nil
}

func (m *AddSlideTimestampsMigration) Down(db *gorm.DB) error {
	// SQLite 不支持 DROP COLUMN
	return nil
}
