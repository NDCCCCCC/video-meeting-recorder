---
phase: 25
slug: scheduler-service-e2e-ci
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-06
---

# Phase 25 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1 |
| **Config file** | none (standard `go test`) |
| **Quick run command** | `go test ./internal/scheduler ./internal/services ./internal/recorder ./internal/observability -run 'TestSchedulerMonitorTask|TestUpdateTaskExtension|TestMarkTaskEndedEarly|TestSmartEndMetrics|TestActivityWatcher_SnapshotExtension|TestRecorderCoordinator_WatcherChannels' -count=1 -v` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | quick ~10s; full ~120s |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/scheduler ./internal/services -count=1` (package-local fast feedback, < 5s)
- **After every plan wave:** Run quick run command above (scheduler + service + recorder + observability subtests)
- **Before `/gsd:verify-work`:** Full suite must be green: `go test -race ./...`
- **Max feedback latency:** 10 seconds (quick command)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 25-01-01 | 01 | 1 | AUDIT-02 (snapshot fields) | — | ActivitySnapshot exposes FileSizeBytes + FileGrowthBps nullable | unit | `go test ./internal/recorder -run 'TestActivityWatcher_SnapshotExtension' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-01-02 | 01 | 1 | AUDIT-04 (service layer) | — | `UpdateTaskExtension` exists in service | unit | `go test ./internal/services -run 'TestUpdateTaskExtension_Exists' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-01-03 | 01 | 1 | AUDIT-04 (service layer) | — | `MarkTaskEndedEarly` exists in service | unit | `go test ./internal/services -run 'TestMarkTaskEndedEarly_Exists' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-01 | 02 | 1 | SCHED-01 | — | monitorTask select 三路 | unit | `go test ./internal/scheduler -run 'TestMonitorTask_TripleSelect' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-02 | 02 | 1 | SCHED-02 | — | EndTime 到点 watcher.IsActive() → extend | unit | `go test ./internal/scheduler -run 'TestMonitorTask_OnTimerActive_Extends' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-03 | 02 | 1 | SCHED-03 | — | taskEndedCh close 后 EndTime.C 不再生效 | unit | `go test ./internal/scheduler -run 'TestMonitorTask_TaskEnded_PreemptsTimer' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-04 | 02 | 1 | SCHED-04 | — | 手动 updateChan 不重置 ExtensionCount | unit | `go test ./internal/scheduler -run 'TestMonitorTask_ManualUpdateDoesNotResetCount' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-05 | 02 | 1 | EXTEND-01 | — | ExtensionCount 上限 4，4×30min=2h | unit | `go test ./internal/services -run 'TestUpdateTaskExtension_MaxLimit' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-06 | 02 | 1 | EXTEND-02 | — | 上限达后 → max_extend_reached, status=completed | unit | `go test ./internal/scheduler -run 'TestMonitorTask_MaxExtendReached' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-07 | 02 | 1 | EARLY-01 | — | H 信号 → smart_early_end + EndedByHuaWeAPI=true | unit | `go test ./internal/services -run 'TestMarkTaskEndedEarly_HuaweiSignal' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-08 | 02 | 1 | EARLY-02 | — | A+B → smart_early_end + EndedByHuaWeAPI=false | unit | `go test ./internal/services -run 'TestMarkTaskEndedEarly_BothSilenceAndStall' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-09 | 02 | 1 | EARLY-03 | — | 提前结束优先 timer (复用 SCHED-03) | unit | (see 25-02-03) | ❌ Wave 0 | ⬜ pending |
| 25-02-10 | 02 | 1 | EARLY-04 | — | 多 input 任一 watcher 触发整体结束 | unit | `go test ./internal/scheduler -run 'TestMonitorTask_MultiInput_AnyEndsAll' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-11 | 02 | 1 | AUDIT-02 | — | 延时 audit log 含 6 字段 snapshot | unit | `go test ./internal/services -run 'TestUpdateTaskExtension_AuditSnapshot' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-12 | 02 | 1 | AUDIT-03 | — | 提前结束 audit log 含 5 字段 | unit | `go test ./internal/services -run 'TestMarkTaskEndedEarly_AuditSnapshot' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-13 | 02 | 1 | AUDIT-04 | — | service 封装 + 统一入口 | unit | `go test ./internal/services -run 'TestServiceEntrypoint_OnlyPath' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-14 | 02 | 1 | CFG-03 | — | `enabled=false` 退回 timer-only | unit | `go test ./internal/scheduler -run 'TestMonitorTask_SmartEndDisabled' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-02-15 | 02 | 1 | CFG-04 | — | `huawei_enabled=false` 跳过 huaweiPoller (Phase 24 已覆盖) | unit | `go test ./internal/recorder -run 'TestActivityWatcher_HuaweiDisabled' -count=1` | ✅ Phase 24 | ⬜ pending |
| 25-03-01 | 03 | 1 | OBS-01 | — | `smart_extend` 日志字段 | unit | `go test ./internal/services -run 'TestUpdateTaskExtension_LogFields' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-03-02 | 03 | 1 | OBS-02 | — | `smart_early_end` 日志字段 | unit | `go test ./internal/services -run 'TestMarkTaskEndedEarly_LogFields' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-03-03 | 03 | 1 | OBS-03 | — | `max_extend_reached` 日志 | unit | `go test ./internal/scheduler -run 'TestMonitorTask_LogMaxExtendReached' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-03-04 | 03 | 1 | OBS-04 | — | `activity_watcher_degraded` 日志 (Phase 24 已覆盖) | unit | `go test ./internal/recorder -run 'TestActivityWatcher_DegradedLog' -count=1` | ✅ Phase 24 | ⬜ pending |
| 25-03-05 | 03 | 1 | OBS-05 | — | `smart_extend_total` atomic +1 | unit | `go test ./internal/observability -run 'TestSmartEndMetrics_RecordExtend' -count=1` | ❌ Wave 0 | ⬜ pending |
| 25-04-01 | 04 | 2 | All E2E | — | Full phase integration | E2E | `go test -race ./...` | ❌ Wave 0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

### Plan 01 — ActivitySnapshot 扩展 + service 层骨架

- [ ] `internal/recorder/activity_watcher.go` — extend `ActivitySnapshot` struct with `FileSizeBytes int64` + `FileGrowthBps int64` nullable
- [ ] `internal/recorder/activity_watcher.go` — `fileTicker` stores `lastFileGrowthBps` into snapshot
- [ ] `internal/recorder/activity_watcher_test.go` — `TestActivityWatcher_SnapshotExtension` covers new fields populated
- [ ] `internal/services/video_recording_task_service.go` — `UpdateTaskExtension(task, deltaMin, reason) error` skeleton
- [ ] `internal/services/video_recording_task_service.go` — `MarkTaskEndedEarly(task, reason, byHuaWeiAPI bool) error` skeleton
- [ ] `internal/services/video_recording_task_service.go` — inject `auditLogSvc *audit.Service` + `cfg *config.SmartEndConfig` dependencies

### Plan 02 — scheduler select 三路 + multi-input merge + CFG-03 双守门

- [ ] `internal/recorder/coordinator.go` — `RecorderCoordinatorInterface` gains `WatcherChannels(taskID uint) []<-chan struct{}`
- [ ] `internal/scheduler/task_ended_channel_helper.go` — `mergeWatchers` fan-in goroutine (one per task)
- [ ] `internal/scheduler/video_scheduler.go:541-606` — `monitorTask` rewrite to select on timer.C + taskEndedCh + updateChan
- [ ] `internal/scheduler/video_scheduler.go` — CFG-03 gate: if `!cfg.SmartEnd.Enabled` → call legacy `monitorTaskEndTimeOnly` path
- [ ] `internal/scheduler/video_scheduler_test.go` — `TestMonitorTask_TripleSelect` + 6 sibling scenarios

### Plan 03 — observability (5 类日志 + atomic counters，禁止 prom 库)

- [ ] `internal/observability/smart_end_metrics.go` — 3 × `atomic.Int64` (extend / early_end / watcher_degraded)
- [ ] `internal/observability/smart_end_metrics.go` — `RecordExtend / RecordEarlyEnd / RecordWatcherDegraded` exposed functions
- [ ] `internal/services/video_recording_task_service.go` — `UpdateTaskExtension` emits OBS-01 `INFO smart_extend` log + `RecordExtend()`
- [ ] `internal/services/video_recording_task_service.go` — `MarkTaskEndedEarly` emits OBS-02 `INFO smart_early_end` log + `RecordEarlyEnd()`
- [ ] `internal/scheduler/video_scheduler.go` — emits OBS-03 `WARN max_extend_reached` log when ExtensionCount hits MaxExtendCount
- [ ] `internal/observability/smart_end_metrics_test.go` — `TestSmartEndMetrics_RecordExtend` + 2 siblings

### Plan 04 — Nyquist 全覆盖 E2E + audit snapshot golden JSON

- [ ] `internal/services/video_recording_task_service_test.go` — `TestUpdateTaskExtension_AuditSnapshot` golden JSON assert (6 fields)
- [ ] `internal/services/video_recording_task_service_test.go` — `TestMarkTaskEndedEarly_AuditSnapshot` golden JSON assert (5 fields)
- [ ] `internal/services/video_recording_task_service_test.go` — `TestServiceEntrypoint_OnlyPath` verifies scheduler never calls db.Update directly (count=0)
- [ ] `internal/scheduler/video_scheduler_test.go` — 7+ E2E subtest: TripleSelect / OnTimerActive_Extends / TaskEnded_PreemptsTimer / ManualUpdateDoesNotResetCount / MaxExtendReached / MultiInput_AnyEndsAll / SmartEndDisabled
- [ ] `internal/recorder/coordinator_test.go` — `TestRecorderCoordinator_WatcherChannels` for new interface method

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| (none) | — | All behaviors covered by unit tests | — |

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s (quick command)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending (set after execution)
