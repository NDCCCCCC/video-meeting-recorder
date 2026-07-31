package services

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// newTestDB 构造 SQLite 内存数据库并迁移本测试所需的三张表。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := db.AutoMigrate(&models.VideoRecordingTask{}, &models.TaskInputConfig{}, &models.InputConfig{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// TestRetryTask_PreservesDuration 验证 BUG-001 修复：RetryTask 重算后 EndTime-StartTime
// 必须等于原始时长（此前因先改 StartTime 再读 duration 导致 EndTime 被静默损坏）。
func TestRetryTask_PreservesDuration(t *testing.T) {
	cases := []struct {
		name           string
		duration       time.Duration
		preJoinMinutes int
	}{
		{"原时长 30 分钟", 30 * time.Minute, 0},
		{"原时长 5 分钟 + 提前 5 分钟", 5 * time.Minute, 5},
		{"原时长 60 分钟", 60 * time.Minute, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			svc := NewVideoRecordingTaskService(db, zap.NewNop())

			base := time.Now().Add(-2 * time.Hour)
			task := models.VideoRecordingTask{
				Name:           "retry-test",
				StartTime:      base,
				EndTime:        base.Add(tc.duration),
				Status:         models.VideoStatusFailed,
				PreJoinMinutes: tc.preJoinMinutes,
				CreatedBy:      1,
			}
			if err := db.Create(&task).Error; err != nil {
				t.Fatalf("创建任务失败: %v", err)
			}

			retried, err := svc.RetryTask(context.Background(), task.ID, task.CreatedBy, false)
			assert.NoError(t, err)
			assert.NotNil(t, retried)

			gotDuration := retried.EndTime.Sub(retried.StartTime)
			assert.Equal(t, tc.duration, gotDuration,
				"重试后任务时长应保持原值 %v，实际 %v", tc.duration, gotDuration)
			assert.True(t, retried.StartTime.After(base), "新 StartTime 应基于 time.Now() 向后推移")
		})
	}
}

// queryCounter 包装 GORM logger，统计对 input_configs 表的 UPDATE 次数（用于证明 N+1 已消除）。
type queryCounter struct {
	gormlogger.Interface
	mu                 sync.Mutex
	inputConfigUpdates int
}

func (q *queryCounter) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, _ := fc()
	q.mu.Lock()
	defer q.mu.Unlock()
	s := strings.TrimSpace(sql)
	if strings.HasPrefix(s, "UPDATE") && strings.Contains(s, "input_configs") {
		q.inputConfigUpdates++
	}
}

// TestDeleteTask_NoNPlusOne 验证 PERF-001 修复：DeleteTask 解锁输入配置改为 Pluck+IN 批量更新，
// 对 input_configs 表只产生 1 次 UPDATE（而非每个配置 1 次），且所有配置均被正确解锁。
func TestDeleteTask_NoNPlusOne(t *testing.T) {
	counter := &queryCounter{Interface: gormlogger.Default.LogMode(gormlogger.Silent)}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: counter})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := db.AutoMigrate(&models.VideoRecordingTask{}, &models.TaskInputConfig{}, &models.InputConfig{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}

	svc := NewVideoRecordingTaskService(db, zap.NewNop())

	task := models.VideoRecordingTask{
		Name:      "delete-test",
		StartTime: time.Now().Add(1 * time.Hour),
		EndTime:   time.Now().Add(2 * time.Hour),
		Status:    models.VideoStatusPending,
		CreatedBy: 1,
	}
	if err := db.Create(&task).Error; err != nil {
		t.Fatalf("创建任务失败: %v", err)
	}

	// 5 个被锁定的 InputConfig + 关联 TaskInputConfig
	for i := 0; i < 5; i++ {
		lockedBy := task.ID
		cfg := models.InputConfig{
			Name:       "cfg",
			ConfigType: "usb",
			IsLocked:   true,
			LockedBy:   &lockedBy,
		}
		if err := db.Create(&cfg).Error; err != nil {
			t.Fatalf("创建 InputConfig 失败: %v", err)
		}
		tc := models.TaskInputConfig{
			TaskID:        task.ID,
			InputConfigID: cfg.ID,
			ConfigType:    "usb",
		}
		if err := db.Create(&tc).Error; err != nil {
			t.Fatalf("创建 TaskInputConfig 失败: %v", err)
		}
	}

	_, err = svc.DeleteTask(context.Background(), task.ID, task.CreatedBy, false)
	assert.NoError(t, err)

	// 行为断言：所有 InputConfig 都已解锁
	var configs []models.InputConfig
	db.Find(&configs)
	for _, c := range configs {
		assert.False(t, c.IsLocked, "InputConfig %d 应被解锁", c.ID)
	}

	// 性能断言：对 input_configs 仅 1 次 UPDATE（批量），证明 N+1 已消除
	assert.Equal(t, 1, counter.inputConfigUpdates,
		"Pluck+IN 批量更新应只产生 1 次 input_configs UPDATE，实际 %d", counter.inputConfigUpdates)
}

// TestVideoRecordingTaskService_CancellationPropagation 验证 Phase 19 Wave 5 ctx 级联：
// 传入已取消的 ctx 时，DB 查询必须返回 context.Canceled（或其包装），证明取消信号能从
// 调用方经服务方法传播到底层 GORM 调用——这是优雅关停正确性的核心保证。
func TestVideoRecordingTaskService_CancellationPropagation(t *testing.T) {
	db := newTestDB(t)
	svc := NewVideoRecordingTaskService(db, zap.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled: 任何后续 DB 操作都应立即失败

	t.Run("ListTasks", func(t *testing.T) {
		_, err := svc.ListTasks(ctx, &ListTasksRequest{Page: 1, PageSize: 10})
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("GetTaskByID", func(t *testing.T) {
		_, err := svc.GetTaskByID(ctx, 1)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("GetTasksByStatus", func(t *testing.T) {
		_, err := svc.GetTasksByStatus(ctx, models.VideoStatusPending)
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("UpdateTaskStatus", func(t *testing.T) {
		err := svc.UpdateTaskStatus(ctx, 1, models.VideoStatusFailed, "test")
		assert.ErrorIs(t, err, context.Canceled)
	})
}
