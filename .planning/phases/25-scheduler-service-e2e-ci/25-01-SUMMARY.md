---
phase: 25-scheduler-service-e2e-ci
plan: 01
subsystem: services
tags: [gorm, audit-log, scheduler, recorder, ActivitySnapshot, smart-end]

# Dependency graph
requires:
  - phase: 23-audit-foundation
    provides: "5 GORM smart-end fields + 3 sentinel errors (ErrRecordingSmartExtend/SmartEarlyEnd/HuaWeiStateFetchFailed) + AuditLogService.RecordChange"
  - phase: 24-activity-watcher
    provides: "ActivityWatcher core fields + Snapshot() return value + fileTicker growthBps local computation"
provides:
  - "ActivitySnapshot.FileSizeBytes + FileGrowthBps fields populated by fileTicker"
  - "VideoRecordingTaskService.UpdateTaskExtension(taskID, deltaMin, reason, snap) as the single timer-driven extension entry point"
  - "VideoRecordingTaskService.MarkTaskEndedEarly(taskID, reason, byHuaWeiAPI, snap) as the single watcher-driven early-end entry point"
  - "SmartEndSnapshot JSON serialization struct (6+ fields) for AUDIT-02/03 audit log payloads"
  - "SetAuditService + SetConfig setters on VideoRecordingTaskService (Phase 25 AUDIT-04 dependency injection)"
affects: [25-02-scheduler-select, 25-03-observability, 25-04-e2e-nyquist]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Variadic Phase 19 D2 pattern preserved: NewVideoRecordingTaskService(db, logger, encryptor...) unchanged; SetAuditService / SetConfig setters for optional Phase 25 injection"
    - "ActivitySnapshot grows by 2 fileTicker-backed fields; unexposed mutex field lastFileGrowthBps keeps fileTicker semantics aligned with lastFileGrowthAt"
    - "auditSvc nil-guard pattern: service-layer audit log writes gracefully skip when audit infrastructure is missing (test path)"
    - "SmartEndSnapshot flat struct with *time.Time pointers for nil-pretty JSON (no 0001-01-01 in DB)"
    - "AuditLogService async tick-based flush (1s) — test pattern uses defer Stop + 2s polling instead of explicit Stop"

key-files:
  created: []
  modified:
    - internal/recorder/activity_watcher.go
    - internal/recorder/activity_watcher_test.go
    - internal/services/video_recording_task_service.go
    - internal/services/video_recording_task_service_test.go

key-decisions:
  - "ActivitySnapshot.FileSizeBytes is a direct mirror of lastFileSize (no separate field needed; fileTicker already updates lastFileSize every tick)"
  - "lastFileGrowthBps is a separately tracked cache only refreshed on the threshold-met branch (matches lastFileGrowthAt semantics — preserves last successful reading when growth stalls)"
  - "SmartEndSnapshot used both for UpdateTaskExtension and MarkTaskEndedEarly payloads; EndedByHuaWeiAPI / EndedEarlyReason omitempty removed so AUDIT-03 always carries the field even when false"
  - "Setter-based injection (SetAuditService / SetConfig) chosen over variadic extension to keep constructor readable as deps grow; Phase 19 D2 encryptor variadic remains untouched"
  - "auditSvc nil-guard: missing audit infra is a graceful degraded mode, not a hard error — production path always injects, test path can pass nil"

patterns-established:
  - "Pattern: Service-layer audit log writes must check `s.auditSvc != nil` before calling RecordChange; this is the single graceful-degradation point"
  - "Pattern: ActivitySnapshot field additions must be made inside the same `w.mu.Lock()` critical section in Snapshot() to avoid partial reads"
  - "Pattern: nilIfZero helper for time.Time → *time.Time conversion; audit log JSON null semantics for zero times"

requirements-completed: [AUDIT-02, AUDIT-03, AUDIT-04, EXTEND-01, EARLY-01, EARLY-02]

# Metrics
duration: 25min
completed: 2026-08-06
---

# Phase 25 Plan 01: ActivitySnapshot Extension + Service Entry Points Summary

**ActivitySnapshot 扩 FileSizeBytes/FileGrowthBps + VideoRecordingTaskService 引入 UpdateTaskExtension / MarkTaskEndedEarly 单入口 — 关闭 AUDIT-02/03/04 + EXTEND-01 + EARLY-01/02 契约面**

## Performance

- **Duration:** 25 min
- **Started:** 2026-08-06T09:15:31Z
- **Completed:** 2026-08-06T09:40:00Z
- **Tasks:** 2 / 2
- **Files modified:** 4

## Accomplishments

- **ActivitySnapshot exposes FileSizeBytes + FileGrowthBps**: 2 new public fields with json tags, populated by fileTicker (lastFileSize mirror + cached lastFileGrowthBps on threshold-met branch); 1 new unexposed mutex field `lastFileGrowthBps` added to ActivityWatcher
- **UpdateTaskExtension entry point**: single timer-driven extension path; reads cfg.SmartEnd.MaxExtendCount as guard (wrapped ErrRecordingSmartExtend); atomic 3-column GORM update; AUDIT-02 snapshot JSON audit log; nil-safe auditSvc guard
- **MarkTaskEndedEarly entry point**: single watcher-driven early-end path; 3-column GORM update (ended_early/ended_early_reason/ended_by_huawei_api); AUDIT-03 snapshot JSON audit log; status field untouched (orchestrator still drives completeTask)
- **5 new test functions** covering all 5 require scenarios: Exists (compile-time signature lock), AuditSnapshot (happy path + audit log row), MaxLimit (upper-bound error), HuaweiSignal (H path), BothSilenceAndStall (A+B path)
- **Constructor backwards-compatible**: NewVideoRecordingTaskService signature unchanged; SetAuditService / SetConfig setters for incremental Phase 25 injection

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend ActivitySnapshot with FileSizeBytes + FileGrowthBps** — `f8f8f6b` (feat)
2. **Task 2: Add UpdateTaskExtension + MarkTaskEndedEarly to VideoRecordingTaskService with audit and cfg injection** — `e527ecb` (feat)

## Files Created/Modified

- `internal/recorder/activity_watcher.go` — ActivitySnapshot struct +2 fields, fileTicker maintains lastFileGrowthBps, Snapshot() copies both inside same lock
- `internal/recorder/activity_watcher_test.go` — TestActivityWatcher_SnapshotExtension added (1 KiB + 16 KiB incremental scenario; uses 1s CheckIntervalS for fast deterministic completion)
- `internal/services/video_recording_task_service.go` — auditSvc + cfg fields, SetAuditService/SetConfig setters, SmartEndSnapshot struct, UpdateTaskExtension + MarkTaskEndedEarly methods
- `internal/services/video_recording_task_service_test.go` — newTestConfig + waitForAuditLogs helpers, 5 new tests covering all 5 require scenarios

## Decisions Made

- **newTestConfig + waitForAuditLogs helpers added** to test file: 1 config factory keeps SmartEnd values in one place; 2s polling for audit logs tolerates 1s tick + Go runtime scheduling
- **Empty OldData fields** in audit log: `task.EndTime` is mutated by `Updates()` so we can't reliably capture it post-write; OldData only contains `extension_count` (Pitfall 4 mitigation); end_time diff is restored by audit_ui off the NewData timestamp
- **`_ = s.auditSvc.RecordChange(...)`**: audit log write errors are intentionally swallowed (don't fail the GORM write path); matches existing service-layer patterns where audit is best-effort
- **Comments-in-Chinese** style preserved throughout (consistent with v1.1/v2.0 codebase convention)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed omitempty from SmartEndSnapshot.EndedByHuaWeiAPI / EndedEarlyReason**
- **Found during:** Task 2 (TestMarkTaskEndedEarly_BothSilenceAndStall)
- **Issue:** With omitempty, the field `ended_by_huawei_api: false` was omitted from JSON, causing audit log assertion to fail and breaking AUDIT-03 contract (the field must always be present per REQUIREMENTS.md)
- **Fix:** Removed `,omitempty` from both EndedByHuaWeiAPI and EndedEarlyReason in SmartEndSnapshot
- **Files modified:** internal/services/video_recording_task_service.go
- **Verification:** TestMarkTaskEndedEarly_BothSilenceAndStall passes; AUDIT-03 JSON unmarshal now sees `ended_by_huawei_api: false`
- **Committed in:** e527ecb (Task 2 commit)

**2. [Rule 3 - Blocking] Test pattern: defer auditSvc.Stop() vs explicit Stop() before assertions**
- **Found during:** Task 2 (TestMarkTaskEndedEarly_HuaweiSignal first try)
- **Issue:** Calling `auditSvc.Stop()` immediately after `RecordChange` cancels the lifecycle context, causing the pending async batch write to fail silently (no rows in DB)
- **Fix:** Used `defer auditSvc.Stop()` pattern (Stop runs at function exit, AFTER assertions + ticker flush); rely on AuditLogService's 1s ticker to flush naturally before `waitForAuditLogs` 2s polling deadline
- **Files modified:** internal/services/video_recording_task_service_test.go
- **Verification:** All 5 new tests pass; existing test pattern unchanged
- **Committed in:** e527ecb (Task 2 commit)

**3. [Rule 1 - Bug] TestActivityWatcher_SnapshotExtension used CheckIntervalS=5 (default) causing 3s timeout**
- **Found during:** Task 1 (initial test run)
- **Issue:** default newTestWatcher cfg has CheckIntervalS=5s; first tick takes 5s; test's 3s `require.Eventually` window was too short
- **Fix:** Built dedicated cfg with CheckIntervalS=1s in the test (1s tick + 1s eventual settling = 2s total, well within 3s windows)
- **Files modified:** internal/recorder/activity_watcher_test.go
- **Verification:** TestActivityWatcher_SnapshotExtension passes in 2.01s
- **Committed in:** f8f8f6b (Task 1 commit)

**4. [Rule 1 - Bug] Test incorrectly called fileTicker without manual wg.Add(1)**
- **Found during:** Task 1 (initial test run)
- **Issue:** `fileTicker` ends with `defer w.wg.Done()`; without pre-Add(1), the WaitGroup counter goes negative causing panic
- **Fix:** Added `w.wg.Add(1)` before `go w.fileTicker(ctx)` to match how `Start()` pre-Add's 4 for the 4 concurrent goroutines
- **Files modified:** internal/recorder/activity_watcher_test.go
- **Verification:** TestActivityWatcher_SnapshotExtension passes without panic
- **Committed in:** f8f8f6b (Task 1 commit)

**5. [Rule 1 - Bug] TestUpdateTaskExtension_Exists used .Elem() on pointer type**
- **Found during:** Task 2 (initial test run)
- **Issue:** `reflect.TypeOf((*VideoRecordingTaskService)(nil)).Elem()` returns the value type `VideoRecordingTaskService` (not pointer), which has 0 methods (all methods are pointer receiver); `MethodByName` returned false
- **Fix:** Use `reflect.TypeOf((*VideoRecordingTaskService)(nil))` directly (without `.Elem()`) to get `*VideoRecordingTaskService` type with all 25 methods
- **Files modified:** internal/services/video_recording_task_service_test.go
- **Verification:** TestUpdateTaskExtension_Exists passes
- **Committed in:** e527ecb (Task 2 commit)

---

**Total deviations:** 5 auto-fixed (5 bugs fixed inline)
**Impact on plan:** All 5 fixes are correctness adjustments (no scope creep). Plan output matches expected: 4 file modifications, 2 task commits, 5 tests + 1 audit test = 6/6 must_haves satisfied.

## Issues Encountered

- **Stop() + lifecycle cancellation race**: AuditLogService.Stop() cancels lifecycleCtx, which causes any pending flushBatch to fail silently. Documented test pattern with `defer Stop()` + rely on 1s ticker instead of explicit Stop. This is an audit library design quirk, not a Plan bug.
- **Pointer receiver reflect capture**: Discovered Go reflect.TypeOf((*T)(nil)) returns `*T` (not `*T` again — pointer to pointer). `.Elem()` returns `T` (value type), losing method visibility. Plan example code suggested `.Elem()`; this is a common Go gotcha.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **25-02 scheduler select 多信号驱动** can now consume the new service methods: `service.UpdateTaskExtension(ctx, taskID, cfg.SmartEnd.ExtendStepMin, "active_meeting", watcher.Snapshot())` from `monitorTask` timer.C branch, and `service.MarkTaskEndedEarly(...)` from `taskEndedCh` branch
- **25-03 observability** can wire `observability.RecordSmartExtend()` / `RecordSmartEarlyEnd()` into the two new methods
- **25-04 Nyquist E2E** can validate against the 5 test scenarios already covered; antipattern grep test (`TestServiceEntrypoint_OnlyPath`) stays sole-ownership of plan 04
- **cmd/server injection** pending: production path requires `videoTaskService.SetAuditService(auditSvc)` + `videoTaskService.SetConfig(cfg)` after both services are constructed (deferred to plan 04 integration or done by plan 02 if it touches scheduler init)
- **No blockers for plan 02**

---
*Phase: 25-scheduler-service-e2e-ci*
*Completed: 2026-08-06*

## Self-Check: PASSED
</content>
</invoke>