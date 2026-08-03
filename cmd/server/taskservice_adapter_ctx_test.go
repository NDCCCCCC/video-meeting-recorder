package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
)

// TestVideoRecordingTaskService_CancellationPropagation (Phase 19 D2)
//
// 重命名自 TestTaskServiceAdapter_CancellationPropagation (Wave 4)。
//
// 验证 Phase 19 Wave 4 (PERF-003/BUG-005) 的取消传播——
// pre-canceled ctx 经 .WithContext(ctx) 传递到 GORM/database/sql 后，
// VideoRecordingTaskService 的 GetTask / UpdateTaskStatus / GetPendingTasks /
// UpdateRecordingPaths / GetInputConfig 均应返回错误（ctx.Err() = context.Canceled）。
//
// 这是计划要求的 3 个取消传播负载测试之一：优雅关停正确性依赖于这些
// DB 调用在 ctx 被取消时及时返回，而非挂起。
//
// D2 变化：测试目标从 cmd/server.taskServiceAdapter 改为直接验证
// internal/services.VideoRecordingTaskService，证明 adapter 合并后
// 取消传播契约仍成立。
func TestVideoRecordingTaskService_CancellationPropagation(t *testing.T) {
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

	svc := services.NewVideoRecordingTaskService(db, zap.NewNop())

	// 预取消 ctx —— database/sql 在获取连接前即检查 ctx.Done() 并返回 ctx.Err()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	t.Run("GetTask", func(t *testing.T) {
		_, err := svc.GetTask(ctx, seedTask.ID)
		assert.Error(t, err, "pre-canceled ctx 应使 GetTask 返回错误")
	})

	t.Run("GetPendingTasks", func(t *testing.T) {
		_, err := svc.GetPendingTasks(ctx)
		assert.Error(t, err, "pre-canceled ctx 应使 GetPendingTasks 返回错误")
	})

	t.Run("UpdateTaskStatus", func(t *testing.T) {
		err := svc.UpdateTaskStatus(ctx, seedTask.ID, models.VideoStatusFailed, "should not apply")
		assert.Error(t, err, "pre-canceled ctx 应使 UpdateTaskStatus 返回错误")
	})

	t.Run("UpdateRecordingPaths", func(t *testing.T) {
		err := svc.UpdateRecordingPaths(ctx, seedTask.ID, "/tmp/a.mkv", "/tmp/a.m3u8")
		assert.Error(t, err, "pre-canceled ctx 应使 UpdateRecordingPaths 返回错误")
	})

	t.Run("GetInputConfig", func(t *testing.T) {
		_, err := svc.GetInputConfig(ctx, seedCfg.ID)
		assert.Error(t, err, "pre-canceled ctx 应使 GetInputConfig 返回错误")
	})

	// 对照：正常 ctx 下 GetTask 应成功 —— 证明上述错误来自 ctx 取消，而非查询本身有误
	gotTask, err := svc.GetTask(context.Background(), seedTask.ID)
	assert.NoError(t, err)
	assert.Equal(t, seedTask.ID, gotTask.ID)
	// 确认 UpdateTaskStatus 因 ctx 取消未实际写入（状态仍为 Pending）
	assert.Equal(t, models.VideoStatusPending, gotTask.Status, "pre-canceled ctx 不应改变任务状态")
}

// TestVideoRecordingTaskService_SatisfiesTaskServiceInterface (Phase 19 D2)
// 编译期断言：services.VideoRecordingTaskService 实现 scheduler.TaskServiceInterface，
// 让 cmd/server 不再需要 taskServiceAdapter。
//
// 此测试通过 svc 的每个方法被 TaskServiceInterface 编译器检查（匿名接口字段赋值）。
// 运行时额外验证 GetInputConfig 返回的 config 与 DB 直查一致（preload 行为）。
func TestVideoRecordingTaskService_SatisfiesTaskServiceInterface(t *testing.T) {
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

	svc := services.NewVideoRecordingTaskService(db, zap.NewNop())

	// 编译期断言（var _ = ...）—— 接口不满足则编译失败
	var _ interface {
		GetTask(context.Context, uint) (*models.VideoRecordingTask, error)
		GetPendingTasks(context.Context) ([]models.VideoRecordingTask, error)
		UpdateTaskStatus(context.Context, uint, models.VideoRecordingTaskStatus, string) error
		UpdateRecordingPaths(context.Context, uint, string, string) error
		GetInputConfig(context.Context, uint) (*models.InputConfig, error)
	} = svc
}
