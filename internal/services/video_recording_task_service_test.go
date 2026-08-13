package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
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
	auditpkg "github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
)

// newTestDB 构造 SQLite 测试数据库并迁移本测试所需的表。
//
// DSN 形如 "file:<unique>?mode=memory&cache=shared":
//   - "file:" + 测试唯一名:在同一进程内每个测试拿到独立的 cache 命名空间,避免
//     -count=N 下跨 iteration 数据污染 (Bug B: 匿名 cache 全局共享,前次写入被
//     后续 iteration 读到,例如 TestMarkTaskEndedEarly_AuditSnapshot 期望
//     "huawei_state_empty" 实际拿到前次 "both_silence_and_stall")。
//   - "mode=memory": 使用 in-memory DB,无磁盘 I/O,速度等同 ":memory:"。
//   - "cache=shared": connection pool 内所有连接共享同一份 schema 与数据,
//     修复 Bug A — 原生 ":memory:" 在 GORM 默认连接池下,每个连接独立持有
//     in-memory DB,AutoMigrate 在 conn A 建的表在 conn B 不可见 →
//     "no such table: audit_logs"。验证: TestDiagnose_MemoryDB_ConnectionIsolation
//     显示 200 并发探测中 185 个 goroutine 落到新连接,sqlite_master 不含 audit_logs。
//
// 取唯一名的两种方式: (1) t.Name() 含子测试层级 + counter,天然唯一但长度不可控;
// (2) 短随机串 + counter。t.TempDir-based 方案(把 sqlite 写到 t.TempDir() 文件)
// 也可解决,但带来 auditSvc.processQueue goroutine 与 t.Cleanup TempDir RemoveAll
// 的 close-time race (Bug C: "The process cannot access the file")。当前方案
// 完全无磁盘 I/O,无 cleanup race。
//
// 注: 如果未来切到 modernc.org/sqlite (pure-Go),语法可能微调,届时再适配。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:test-%d-%s?mode=memory&cache=shared", time.Now().UnixNano(), strings.ReplaceAll(t.Name(), "/", "-"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	if err != nil {
		t.Fatalf("打开内存 sqlite 失败: %v", err)
	}
	if err := db.AutoMigrate(&models.VideoRecordingTask{}, &models.TaskInputConfig{}, &models.InputConfig{}, &models.AuditLog{}); err != nil {
		t.Fatalf("AutoMigrate 失败: %v", err)
	}
	return db
}

// newTestConfig 构造带 SmartEnd 默认值的 cfg,MaxExtendCount=4。
// Phase 25 撤回后 SmartEnd 仅保留 Enabled / ExtendStepMin / MaxExtendCount 3 字段。
func newTestConfig() *config.Config {
	return &config.Config{SmartEnd: config.SmartEndConfig{
		Enabled:        true,
		ExtendStepMin:  30,
		MaxExtendCount: 4,
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

// TestUpdateTaskExtension_Exists 验证反射可发现 UpdateTaskExtension 方法
// (Phase 25 撤回后唯一服务入口)。用 reflect 而非直接调用,目的:未来若有人
// 删除方法签名,reflect 直接拿到 ErrMethod 比跑测试快且意图明确。
func TestUpdateTaskExtension_Exists(t *testing.T) {
	// 方法是 *VideoRecordingTaskService pointer receiver,reflect.TypeOf(*T)(nil)
	// 不再 .Elem() 直接得 *T 才能看到方法。
	typ := reflect.TypeOf((*VideoRecordingTaskService)(nil))
	for _, name := range []string{"UpdateTaskExtension"} {
		m, ok := typ.MethodByName(name)
		require.True(t, ok, "VideoRecordingTaskService 必须有 %s 方法", name)
		// NumIn 包括 receiver,所以 UpdateTaskExtension(ctx, taskID, deltaMin, reason) = 5
		require.Equal(t, 5, m.Type.NumIn(),
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
		Name:           "extend-audit-test",
		StartTime:      base,
		EndTime:        base.Add(30 * time.Minute),
		Status:         models.VideoStatusRecording,
		PreJoinMinutes: 0,
		CreatedBy:      1,
		InputConfigID:  nil,
		ExtensionCount: 0,
	}
	require.NoError(t, db.Create(&task).Error)
	originalEnd := task.EndTime

	// 注: Phase 25 智能退出撤回后,UpdateTaskExtension 不再接收 ActivitySnapshot 参数。

	err := svc.UpdateTaskExtension(context.Background(), task.ID, 30, "active_meeting")
	require.NoError(t, err)

	// GORM 字段断言 (3 字段实际此处只更新 3)
	var got models.VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error)
	assert.Equal(t, 1, got.ExtensionCount, "ExtensionCount 应 +1")
	assert.Equal(t, "active_meeting", got.LastExtensionReason)
	assert.True(t, got.EndTime.Equal(originalEnd.Add(30*time.Minute)),
		"EndTime 应推进 30min,原 %v 期望 %v 实际 %v",
		originalEnd, originalEnd.Add(30*time.Minute), got.EndTime)

	// AUDIT-02:audit_logs 行存在,JSON 含 6 关键字段
	// FlushNow: 显式落库避免 1s ticker 未到点导致 waitForAuditLogs 拿空表
	// (CI 上 ticker 时序 + race detector 调度可能与本地不同)
	auditSvc.FlushNow(context.Background())
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
	// 注: Phase 25 撤回后,ActivitySnapshot 整类型删除,file_size_bytes /
	// file_growth_bps / silence_since / last_file_growth 字段都不再写入 audit log。
	// SmartEndSnapshot 仅剩 ExtensionCount + NewEndTime 2 字段。
	assert.EqualValues(t, 1, newDataMap["extension_count"], "extension_count 字段必须保留")
	assert.NotNil(t, newDataMap["new_end_time"], "new_end_time 字段必须保留")

	// OldData must preserve the pre-Updates end_time and extension_count values.
	var oldDataMap map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(auditLog.OldData), &oldDataMap))
	assert.EqualValues(t, 0, oldDataMap["extension_count"])
	oldEndText, ok := oldDataMap["end_time"].(string)
	require.True(t, ok, "OldData.end_time must be an RFC3339 string")
	oldEnd, err := time.Parse(time.RFC3339Nano, oldEndText)
	require.NoError(t, err)
	assert.True(t, oldEnd.Equal(originalEnd), "OldData.end_time 应保留更新前值")
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
		Name:           "max-extend-test",
		StartTime:      time.Now().UTC().Add(-time.Hour),
		EndTime:        time.Now().UTC().Add(30 * time.Minute),
		Status:         models.VideoStatusRecording,
		PreJoinMinutes: 0,
		CreatedBy:      1,
		ExtensionCount: 4, // 已达 MaxExtendCount
	}
	require.NoError(t, db.Create(&task).Error)

	err := svc.UpdateTaskExtension(context.Background(), task.ID, 30, "active_meeting")
	require.Error(t, err, "已经达到 MaxExtendCount 时延长应被拒绝")
	assert.True(t, errors.Is(err, apperrors.ErrRecordingSmartExtend),
		"返回错误必须链上 apperrors.ErrRecordingSmartExtend,实际 %v", err)

	// 字段未被改动
	var got models.VideoRecordingTask
	require.NoError(t, db.First(&got, task.ID).Error)
	assert.Equal(t, 4, got.ExtensionCount, "拒绝路径不应改 ExtensionCount")
	assert.Equal(t, "", got.LastExtensionReason, "拒绝路径不应改 LastExtensionReason")
}

// 注: Phase 25 智能退出撤回后,MarkTaskEndedEarly 整方法与其 3 个测试
// (H 信号 / A+B 双信号 / AuditSnapshot) 全部删除。提前结束流程改由
// monitorProcess(ffmpeg 退出) + IsProcessAlive(endtime 查 ProcessState) 接管。
// 详见 .planning/debug/huawei-auto-smart-end.md 撤回决策。

// ---------------------------------------------------------------------------
// Phase 25 Plan 04 — Nyquist 闭环测试（gold JSON + antipattern grep）
// ---------------------------------------------------------------------------

// resolveSchedulerSource 解析 internal/scheduler/video_scheduler.go 的绝对路径。
// 走 runtime.Caller(0) 拿本测试文件所在目录,再 ../scheduler/video_scheduler.go 拼出。
// 这避免了硬编码 ../../internal/... (cwd 变化时脆弱)。
func resolveSchedulerSource(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "runtime.Caller(0) 必须可用")
	thisDir := filepath.Dir(thisFile)
	candidate := filepath.Clean(filepath.Join(thisDir, "..", "scheduler", "video_scheduler.go"))
	if _, err := os.Stat(candidate); err != nil {
		t.Fatalf("解析 scheduler 源文件失败 %s: %v", candidate, err)
	}
	return candidate
}

// TestServiceEntrypoint_OnlyPath 验证 Phase 25 AUDIT-04 守门:scheduler
// 不得直 GORM 写 5 smart-end 字段(extension_count / ended_early /
// last_extension_reason / ended_early_reason / ended_by_huawei_api)。
// 通过反模式 grep 实现:scheduler 源文件不应包含
// `s.taskService.GetDB().Model(&models.VideoRecordingTask{}).Updates(` 或
// `s.taskService.GetDB().Model(&task).Updates(`(即使是局部变量也会被命中)。
//
// 注意:本测试是 service 包内的"side check",scheduler 包内另有
// TestScheduler_DoesNotDirectlyUpdateTask(plan 04 sole owner 双侧防御)。
func TestServiceEntrypoint_OnlyPath(t *testing.T) {
	src, err := os.ReadFile(resolveSchedulerSource(t))
	require.NoError(t, err)
	content := string(src)

	// antipattern 1: literal struct reference
	antipattern1 := "s.taskService.GetDB().Model(&models.VideoRecordingTask{}).Updates("
	assert.NotContains(t, content, antipattern1,
		"scheduler 不应直 GORM 写 VideoRecordingTask;应走 service.UpdateTaskExtension / MarkTaskEndedEarly")

	// antipattern 2: local-variable reference
	antipattern2 := "s.taskService.GetDB().Model(&task).Updates("
	assert.NotContains(t, content, antipattern2,
		"scheduler 不应直 GORM 写 task 局部变量;应走 service.UpdateTaskExtension / MarkTaskEndedEarly")
}

// TestMarkTaskEndedEarly_AuditSnapshot 撤回 (Phase 25 撤回后 MarkTaskEndedEarly 删除)。

// TestAuditSnapshot_ZeroTimeOmitsSilence 撤回 (Phase 25 撤回后,ActivitySnapshot
// 整类型删除,SilenceSince/LastFileGrowthAt/FileSizeBytes/FileGrowthBps 等字段
// 都不再写入 audit log NewData。TestUpdateTaskExtension_AuditSnapshot 仍保留
// 覆盖 audit snapshot 字段 (extension_count / new_end_time) 的存在性)。
