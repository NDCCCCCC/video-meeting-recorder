package main

import (
	"context"
	"testing"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestTaskServiceAdapter_CancellationPropagation 验证 Phase 19 Wave 4
// (PERF-003/BUG-005) 的取消传播：pre-cancelled ctx 经 .WithContext(ctx)
// 传递到 GORM/database/sql 后，GetTask / UpdateTaskStatus / GetPendingTasks /
// UpdateRecordingPaths / GetInputConfig 均应返回错误（ctx.Err() = context.Canceled）。
//
// 这是计划要求的 3 个取消传播负载测试之一：优雅关停正确性依赖于这些
// DB 调用在 ctx 被取消时及时返回，而非挂起。
func TestTaskServiceAdapter_CancellationPropagation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := db.AutoMigrate(
		&models.VideoRecordingTask{},
		&models.InputConfig{},
		&models.TaskInputConfig{},
	); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	// 预置一行任务 + 一个 InputConfig（供 GetInputConfig/UpdateRecordingPaths 覆盖）
	seedTask := &models.VideoRecordingTask{
		Name:   "cancel-prop-test",
		Status: models.VideoStatusPending,
	}
	if err := db.Create(seedTask).Error; err != nil {
		t.Fatalf("预置任务失败: %v", err)
	}
	seedCfg := &models.InputConfig{
		Name:       "Test Config",
		ConfigType: models.ConfigTypeUSB,
	}
	if err := db.Create(seedCfg).Error; err != nil {
		t.Fatalf("预置输入配置失败: %v", err)
	}

	adapter := &taskServiceAdapter{db: db, logger: zap.NewNop()}

	// 预取消 ctx —— database/sql 在获取连接前即检查 ctx.Done() 并返回 ctx.Err()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("GetTask", func(t *testing.T) {
		_, err := adapter.GetTask(ctx, seedTask.ID)
		assert.Error(t, err, "pre-cancelled ctx 应使 GetTask 返回错误")
	})

	t.Run("GetPendingTasks", func(t *testing.T) {
		_, err := adapter.GetPendingTasks(ctx)
		assert.Error(t, err, "pre-cancelled ctx 应使 GetPendingTasks 返回错误")
	})

	t.Run("UpdateTaskStatus", func(t *testing.T) {
		err := adapter.UpdateTaskStatus(ctx, seedTask.ID, models.VideoStatusFailed, "should not apply")
		assert.Error(t, err, "pre-cancelled ctx 应使 UpdateTaskStatus 返回错误")
	})

	t.Run("UpdateRecordingPaths", func(t *testing.T) {
		err := adapter.UpdateRecordingPaths(ctx, seedTask.ID, "/tmp/a.mkv", "/tmp/a.m3u8")
		assert.Error(t, err, "pre-cancelled ctx 应使 UpdateRecordingPaths 返回错误")
	})

	t.Run("GetInputConfig", func(t *testing.T) {
		_, err := adapter.GetInputConfig(ctx, seedCfg.ID)
		assert.Error(t, err, "pre-cancelled ctx 应使 GetInputConfig 返回错误")
	})

	// 对照：正常 ctx 下 GetTask 应成功 —— 证明上述错误来自 ctx 取消，而非查询本身有误
	gotTask, err := adapter.GetTask(context.Background(), seedTask.ID)
	assert.NoError(t, err)
	assert.Equal(t, seedTask.ID, gotTask.ID)
	// 确认 UpdateTaskStatus 因 ctx 取消未实际写入（状态仍为 Pending）
	assert.Equal(t, models.VideoStatusPending, gotTask.Status, "pre-cancelled ctx 不应改变任务状态")
}
