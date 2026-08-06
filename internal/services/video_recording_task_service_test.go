package services

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/recorder"
	auditpkg "github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
)

// newTestDB 构造 SQLite 内存数据库并迁移本测试所需的三张表。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := db.AutoMigrate(&models.VideoRecordingTask{}, &models.TaskInputConfig{}, &models.InputConfig{}, &models.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// newTestConfig 构造带 SmartEnd 默认值的 cfg,MaxExtendCount=4。
func newTestConfig() *config.Config {
	return &config.Config{SmartEnd: config.SmartEndConfig{
		Enabled:                true,
		HuaweiEnabled:          false,
		DegradeOnSilenceLoss:   false,
		SilenceDB:              -30,
		SilenceDurationS:       30,
		FileStallS:             120,
		FileMinGrowthBPS:       1024,
		HuaweiPollIntervalS:    30,
		HuaweiPersistS:         30,
		HuaweiFailureThreshold: 3,
		CheckIntervalS:         1,
		ExtendStepMin:          30,
		MaxExtendCount:         4,
		StatFailureThreshold:   3,
	}}
}

// waitForAuditFlush 等待 AuditLogService 异步队列 flush。Stop() 关闭 stopCh
// 触发 processQueue 把 batch 写入 DB;但 Stop 本身非阻塞 — processQueue 在
// 命中 stopCh 之前可能还在 select 中处理 asyncQueue 或 ticker.C 路径。
// 2 秒 polling 上限 + Sleep 20ms 间隔,覆盖 Go runtime 调度延迟。
func waitForAuditLogs(t *testing.T, db *gorm.DB, expected int) []models.AuditLog {
	t.Helper()
	var logs []models.AuditLog
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := db.Find(&logs).Error; err != nil {
			t.Fatalf("查询 audit_logs 失败: %v", err)
		}
		if len(logs) >= expected {
			return logs
		}
		time.Sleep(20 * time.Millisecond)
	}
	return logs
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
	cancel() // pre-canceled: 任何后续 DB 操作都应立即失败

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

// ---------------------------------------------------------------------------
// Phase 25 AUDIT-04 + EXTEND-01/02 + EARLY-01/02 — UpdateTaskExtension / MarkTaskEndedEarly
// ---------------------------------------------------------------------------

// TestUpdateTaskExtension_Exists 验证反射可发现 UpdateTaskExtension 与
// MarkTaskEndedEarly 两个方法 (Phase 25 AUDIT-04 单一入口契约)。用 reflect
// 而非直接调用,目的:未来若有人删除方法签名,reflect 直接拿到 ErrMethod
// 比跑测试快且意图明确。
func TestUpdateTaskExtension_Exists(t *testing.T) {
	// 方法是 *VideoRecordingTaskService pointer receiver,reflect.TypeOf(*T)(nil)
	// 不再 .Elem() 直接得 *T 才能看到方法。
	typ := reflect.TypeOf((*VideoRecordingTaskService)(nil))
	for _, name := range []string{"UpdateTaskExtension", "MarkTaskEndedEarly"} {
		m, ok := typ.MethodByName(name)
		require.True(t, ok, "VideoRecordingTaskService 必须有 %s 方法", name)
		// NumIn 包括 receiver,所以 UpdateTaskExtension(ctx, taskID, deltaMin, reason, snap) = 6
		require.Equal(t, 6, m.Type.NumIn(),
			"%s 签名参数数量变化,请同步更新 Phase 25-01 文档", name)
		require.Equal(t, 1, m.Type.NumOut(),
			"%s 必须返回 error", name)
	}
}

// TestUpdateTaskExtension_AuditSnapshot 验证 Phase 25 AUDIT-02:
// UpdateTaskExtension 写 3 GORM 字段 (end_time / extension_count / last_extension_reason) +
// 同步 audit_logs 1 行 (snapshot JSON 含 6 字段)。
//
// 路径:auditSvc 非 nil (用真实 NewAuditLogService),Stop() 强制 flush,
// 查 audit_logs 行验证 ResourceID + NewData JSON 字段覆盖。
func TestUpdateTaskExtension_AuditSnapshot(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig()
	svc := NewVideoRecordingTaskService(db, zap.NewNop())
	svc.SetConfig(cfg)
	auditSvc := auditpkg.NewAuditLogService(db, zap.NewNop())
	defer auditSvc.Stop()
	svc.SetAuditService(auditSvc)

	base := time.Now().UTC().Add(-1 * time.Hour)
	task := models.VideoRecordingTask{
		Name:          "extend-audit-test",
		StartTime:     base,
		EndTime:       base.Add(30 * time.Minute),
		Status:        models.VideoStatusRecording,
		PreJoinMinutes: 0,
		CreatedBy:     1,
		InputConfigID: nil,
		ExtensionCount: 0,
	}
	require.NoError(t, db.Create(&task).Error)
	originalEnd := task.EndTime

	fixedTime := time.Now().UTC().Add(-10 * time.Second)
	snap := recorder.ActivitySnapshot{
		FileSizeBytes:    1024,
		FileGrowthBps:    2048,
		LastFileGrowthAt: fixedTime,
		SilenceSince:     fixedTime,
	}

	err := svc.UpdateTaskExtension(context.Background(), task.ID, 30, "active_meeting", snap)
	require.NoError(t, err)

	// GORM 字段断言 (5 字段实际此处只更新 3 — EndedEarly* 不在延长路径)
	var got models.VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error)
	assert.Equal(t, 1, got.ExtensionCount, "ExtensionCount 应 +1")
	assert.Equal(t, "active_meeting", got.LastExtensionReason)
	assert.True(t, got.EndTime.Equal(originalEnd.Add(30*time.Minute)),
		"EndTime 应推进 30min,原 %v 期望 %v 实际 %v",
		originalEnd, originalEnd.Add(30*time.Minute), got.EndTime)
	assert.False(t, got.EndedEarly, "延长路径不应改 EndedEarly")

	// AUDIT-02:audit_logs 行存在,JSON 含 6 关键字段
	logs := waitForAuditLogs(t, db, 1)
	require.GreaterOrEqual(t, len(logs), 1, "audit_logs 应至少 1 行")
	auditLog := logs[0]
	assert.Equal(t, models.ActionUpdate, auditLog.Action)
	assert.Equal(t, models.ModuleTask, auditLog.Module)
	require.NotNil(t, auditLog.ResourceID)
	assert.Equal(t, task.ID, *auditLog.ResourceID)
	assert.Equal(t, "video_recording_task", auditLog.Resource)

	var newDataMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(auditLog.NewData), &newDataMap))
	assert.EqualValues(t, 1024, newDataMap["file_size_bytes"], "file_size_bytes 字段必须保留")
	assert.EqualValues(t, 2048, newDataMap["file_growth_bps"], "file_growth_bps 字段必须保留")
	assert.EqualValues(t, 1, newDataMap["extension_count"], "extension_count 字段必须保留")
	assert.NotNil(t, newDataMap["new_end_time"], "new_end_time 字段必须保留")
	// silence_since / last_file_growth 用 *time.Time → JSON null / string 均可,只要 key 存在
	_, hasSilence := newDataMap["silence_since"]
	_, hasLastGrowth := newDataMap["last_file_growth"]
	assert.True(t, hasSilence, "silence_since key 必须存在 JSON")
	assert.True(t, hasLastGrowth, "last_file_growth key 必须存在 JSON")
}

// TestUpdateTaskExtension_MaxLimit 验证 Phase 25 EXTEND-01/02:ExtensionCount
// == MaxExtendCount 时,UpdateTaskExtension 拒绝延长并返回 wrapped
// apperrors.ErrRecordingSmartExtend (errors.Is 可识别)。scheduler 据此
// 走 EXTEND-02 max_extend_reached 强制收尾路径。
func TestUpdateTaskExtension_MaxLimit(t *testing.T) {
	db := newTestDB(t)
	cfg := newTestConfig() // MaxExtendCount=4
	svc := NewVideoRecordingTaskService(db, zap.NewNop())
	svc.SetConfig(cfg)

	task := models.VideoRecordingTask{
		Name:          "max-extend-test",
		StartTime:     time.Now().UTC().Add(-time.Hour),
		EndTime:       time.Now().UTC().Add(30 * time.Minute),
		Status:        models.VideoStatusRecording,
		PreJoinMinutes: 0,
		CreatedBy:     1,
		ExtensionCount: 4, // 已达 MaxExtendCount
	}
	require.NoError(t, db.Create(&task).Error)

	err := svc.UpdateTaskExtension(context.Background(), task.ID, 30, "active_meeting", recorder.ActivitySnapshot{})
	require.Error(t, err, "已经达到 MaxExtendCount 时延长应被拒绝")
	assert.True(t, errors.Is(err, apperrors.ErrRecordingSmartExtend),
		"返回错误必须链上 apperrors.ErrRecordingSmartExtend,实际 %v", err)

	// 字段未被改动
	var got models.VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error)
	assert.Equal(t, 4, got.ExtensionCount, "拒绝路径不应改 ExtensionCount")
	assert.Equal(t, "", got.LastExtensionReason, "拒绝路径不应改 LastExtensionReason")
}

// TestMarkTaskEndedEarly_HuaweiSignal 验证 Phase 25 EARLY-01:H 信号触发
// 时,MarkTaskEndedEarly 写 3 GORM 字段 (ended_early / ended_early_reason /
// ended_by_huawei_api) + audit log 1 行。
func TestMarkTaskEndedEarly_HuaweiSignal(t *testing.T) {
	db := newTestDB(t)
	svc := NewVideoRecordingTaskService(db, zap.NewNop())
	auditSvc := auditpkg.NewAuditLogService(db, zap.NewNop())
	defer auditSvc.Stop()
	svc.SetAuditService(auditSvc)

	task := models.VideoRecordingTask{
		Name:          "huawei-early-end",
		StartTime:     time.Now().UTC().Add(-time.Hour),
		EndTime:       time.Now().UTC().Add(30 * time.Minute),
		Status:        models.VideoStatusRecording,
		PreJoinMinutes: 0,
		CreatedBy:     1,
	}
	require.NoError(t, db.Create(&task).Error)

	fixedTime := time.Now().UTC().Add(-30 * time.Second)
	snap := recorder.ActivitySnapshot{
		FileSizeBytes:    4096,
		FileGrowthBps:    0, // 华为主信号,文件停滞不意味停滞
		LastFileGrowthAt: fixedTime,
		HuaWeiEmptySince: fixedTime,
	}

	err := svc.MarkTaskEndedEarly(context.Background(), task.ID, "huawei_state_empty", true, snap)
	require.NoError(t, err)

	// GORM 字段断言
	var got models.VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error)
	assert.True(t, got.EndedEarly, "EndedEarly 应 true")
	assert.Equal(t, "huawei_state_empty", got.EndedEarlyReason)
	assert.True(t, got.EndedByHuaWeAPI, "EndedByHuaWeAPI 应 true (H 信号)")

	// audit log 断言:processQueue 每 1s ticker flush,waitForAuditLogs 上限 2s 让其落入 DB
	logs := waitForAuditLogs(t, db, 1)
	require.GreaterOrEqual(t, len(logs), 1)
	auditLog := logs[0]
	require.NotNil(t, auditLog.ResourceID)
	assert.Equal(t, task.ID, *auditLog.ResourceID)

	var newDataMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(auditLog.NewData), &newDataMap))
	assert.Equal(t, true, newDataMap["ended_by_huawei_api"])
	assert.Equal(t, "huawei_state_empty", newDataMap["ended_early_reason"])
	assert.EqualValues(t, 4096, newDataMap["file_size_bytes"])
}

// TestMarkTaskEndedEarly_BothSilenceAndStall 验证 Phase 25 EARLY-02:A+B 双
// AND 触发时,MarkTaskEndedEarly 写 ended_early=true + ended_by_huawei_api=false。
func TestMarkTaskEndedEarly_BothSilenceAndStall(t *testing.T) {
	db := newTestDB(t)
	svc := NewVideoRecordingTaskService(db, zap.NewNop())
	auditSvc := auditpkg.NewAuditLogService(db, zap.NewNop())
	defer auditSvc.Stop()
	svc.SetAuditService(auditSvc)

	task := models.VideoRecordingTask{
		Name:          "ab-early-end",
		StartTime:     time.Now().UTC().Add(-time.Hour),
		EndTime:       time.Now().UTC().Add(30 * time.Minute),
		Status:        models.VideoStatusRecording,
		PreJoinMinutes: 0,
		CreatedBy:     1,
	}
	require.NoError(t, db.Create(&task).Error)

	fixedTime := time.Now().UTC().Add(-120 * time.Second)
	snap := recorder.ActivitySnapshot{
		FileSizeBytes:    8192,
		FileGrowthBps:    0, // 文件停滞
		LastFileGrowthAt: fixedTime,
		SilenceSince:     fixedTime,
	}

	err := svc.MarkTaskEndedEarly(context.Background(), task.ID, "both_silence_and_stall", false, snap)
	require.NoError(t, err)

	// GORM 字段断言
	var got models.VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error)
	assert.True(t, got.EndedEarly)
	assert.Equal(t, "both_silence_and_stall", got.EndedEarlyReason)
	assert.False(t, got.EndedByHuaWeAPI, "A+B 路径 EndedByHuaWeAPI 应 false")

	// audit log 断言
	logs := waitForAuditLogs(t, db, 1)
	require.GreaterOrEqual(t, len(logs), 1)
	auditLog := logs[0]
	var newDataMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(auditLog.NewData), &newDataMap))
	assert.Equal(t, false, newDataMap["ended_by_huawei_api"],
		"A+B 路径 audit NewData.ended_by_huawei_api 应 false")
	assert.Equal(t, "both_silence_and_stall", newDataMap["ended_early_reason"])
}
