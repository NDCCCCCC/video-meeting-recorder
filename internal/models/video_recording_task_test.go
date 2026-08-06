package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err, "打开内存 SQLite 数据库")
	require.NoError(t, db.AutoMigrate(&VideoRecordingTask{}), "迁移 VideoRecordingTask")

	return db
}

func TestVideoRecordingTaskSmartEndFields_SchemaMigration(t *testing.T) {
	db := newTestDB(t)

	columns := []string{
		"extension_count",
		"last_extension_reason",
		"ended_early",
		"ended_early_reason",
		"ended_by_huawei_api",
	}
	for _, column := range columns {
		t.Run(column, func(t *testing.T) {
			assert.True(t, db.Migrator().HasColumn(&VideoRecordingTask{}, column),
				"video_recording_tasks 应包含 %s 列", column)
		})
	}
}

func TestVideoRecordingTaskSmartEndFields_Defaults(t *testing.T) {
	db := newTestDB(t)
	now := time.Now()
	task := VideoRecordingTask{
		Name:             "x",
		StartTime:        now.Add(-time.Hour),
		EndTime:          now.Add(time.Hour),
		ConferenceNumber: "TEST",
		Status:           VideoStatusPending,
		CreatedBy:        1,
	}

	require.NoError(t, db.Create(&task).Error, "创建使用默认智能收尾字段的任务")

	var got VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error, "重新读取任务")
	assert.Equal(t, 0, got.ExtensionCount)
	assert.Equal(t, "", got.LastExtensionReason)
	assert.False(t, got.EndedEarly)
	assert.Equal(t, "", got.EndedEarlyReason)
	assert.False(t, got.EndedByHuaWeAPI)
}

func TestVideoRecordingTaskSmartEndFields_RoundTrip(t *testing.T) {
	db := newTestDB(t)
	now := time.Now()
	task := VideoRecordingTask{
		Name:                "smart-end-round-trip",
		StartTime:           now.Add(-time.Hour),
		EndTime:             now.Add(time.Hour),
		ConferenceNumber:    "TEST",
		Status:              VideoStatusPending,
		CreatedBy:           1,
		ExtensionCount:      3,
		LastExtensionReason: "huawei_persist",
		EndedEarly:          true,
		EndedEarlyReason:    "both_silence_and_stall",
		EndedByHuaWeAPI:     true,
	}

	require.NoError(t, db.Create(&task).Error, "创建包含智能收尾审计值的任务")

	var got VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error, "重新读取任务")
	assert.Equal(t, task.ExtensionCount, got.ExtensionCount)
	assert.Equal(t, task.LastExtensionReason, got.LastExtensionReason)
	assert.Equal(t, task.EndedEarly, got.EndedEarly)
	assert.Equal(t, task.EndedEarlyReason, got.EndedEarlyReason)
	assert.Equal(t, task.EndedByHuaWeAPI, got.EndedByHuaWeAPI)
}
