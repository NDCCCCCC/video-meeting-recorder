// Package observability 暴露 Phase 25 v2.0 OBS-05 接入点:
// 3 个 atomic.Int64 计数器 (smart_extend_total / smart_early_end_total /
// watcher_degraded_total) + 3 个 Record* 函数 + 3 个 getter + 1 个 ResetForTest
// 测试辅助。本文件刻意不导入 github.com/prometheus/client_golang:
// 项目当前无 prometheus 集成 (per PROJECT.md v2.0 milestone "可观测" 节),
// 接入点仅暴露 atomic,后续 prom 接入只改包内 Record* 实现即可,scheduler /
// service / recorder 调用方不变。
//
// 调用约定:Record* 必须在 success-path 调;失败路径不调以避免计数误差。
// 进程级可见 (无 per-task 维度);测试场景用 ResetForTest 清零后单独校验。
package observability

import "sync/atomic"

// atomic.Int64 计数器 — 进程级全局,所有 goroutine 共享。
var (
	// smartExtendTotal OBS-01: service.UpdateTaskExtension 成功延时一次 +1。
	smartExtendTotal atomic.Int64
	// smartEarlyEndTotal OBS-02: service.MarkTaskEndedEarly 成功提前结束一次 +1。
	smartEarlyEndTotal atomic.Int64
	// watcherDegradedTotal OBS-04: ActivityWatcher 三个降级分支
	// (silence_parser_failed / huawei_client_nil / huawei_api_unreachable)
	// 任一触发即 +1;file_stat_failed 是早期结束信号 (INFO),不计入本计数器。
	watcherDegradedTotal atomic.Int64
)

// RecordSmartExtend OBS-01 智能延时计数器 +1。
// 必须在 UpdateTaskExtension GORM Updates 成功之后调用;失败路径不调。
func RecordSmartExtend() { smartExtendTotal.Add(1) }

// RecordSmartEarlyEnd OBS-02 智能提前结束计数器 +1。
// 必须在 MarkTaskEndedEarly GORM Updates 成功之后调用;失败路径不调。
func RecordSmartEarlyEnd() { smartEarlyEndTotal.Add(1) }

// RecordWatcherDegraded OBS-04 watcher 降级事件计数器 +1。
// 在三个已存在的 zap.Warn("activity_watcher_degraded", ...) 分支内立即调用;
// 不要在 file_stat_failed 路径 (closeEnded("file_stat_failed")) 调用,
// 那个是早期结束信号 (INFO),不是降级事件 (ERROR)。
func RecordWatcherDegraded() { watcherDegradedTotal.Add(1) }

// SmartExtendTotal 返回当前累计延时次数 (测试 + 未来 prom handler 使用)。
func SmartExtendTotal() int64 { return smartExtendTotal.Load() }

// SmartEarlyEndTotal 返回当前累计提前结束次数。
func SmartEarlyEndTotal() int64 { return smartEarlyEndTotal.Load() }

// WatcherDegradedTotal 返回当前累计 watcher 降级次数。
func WatcherDegradedTotal() int64 { return watcherDegradedTotal.Load() }

// ResetForTest 清零所有 3 个计数器。仅供 *_test.go 使用;
// 与生产流量并发调用不安全 (reset 与 Add 竞争)。测试间用 t.Cleanup 复位以避免
// 共享全局状态的污染。
func ResetForTest() {
	smartExtendTotal.Store(0)
	smartEarlyEndTotal.Store(0)
	watcherDegradedTotal.Store(0)
}