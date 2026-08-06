---
phase: 25-scheduler-service-e2e-ci
plan: 03
subsystem: observability
tags: [zap, atomic, OBS-05, prometheus-stub, smart-end, audit-log]

# Dependency graph
requires:
  - phase: 25-scheduler-service-e2e-ci
    plan: 01
    provides: "VideoRecordingTaskService.UpdateTaskExtension / MarkTaskEndedEarly success-path methods (Task 2 wire-up target)"
  - phase: 25-scheduler-service-e2e-ci
    plan: 02
    provides: "VideoSimpleScheduler.handleEndTimeReached max-extend branch (OBS-03 WARN log site)"
  - phase: 24-activity-watcher
    provides: "ActivityWatcher 3 activity_watcher_degraded WARN branches + file_stat_failed closeEnded path (OBS-04 wiring target)"
provides:
  - "internal/observability package: 3 atomic.Int64 counters (smart_extend_total / smart_early_end_total / watcher_degraded_total) + 3 Record* functions + 3 getters + ResetForTest helper"
  - "OBS-01..04 log field contract satisfied at call sites (smart_extend task count new_end reason; smart_early_end task reason snapshot; max_extend_reached task_id force_end; activity_watcher_degraded reason)"
  - "OBS-05 atomic counter wiring at all 5 success-path call sites (UpdateTaskExtension, MarkTaskEndedEarly, silence_parser_failed, huawei_client_nil, huawei_api_unreachable)"
  - "Design rationale comment at file_stat_failed closeEnded documenting OBS-04 deliberately excludes it"
  - "max_extend_reached counter intentionally NOT incremented (avoids double-count with the prior successful extension)"
affects: [25-04-Nyquist-E2E]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "sync/atomic.Int64 process-global counter pattern (no mutex, no prom client)"
    - "Atomic increment placement: AFTER GORM Updates success but BEFORE audit log block — keeps counter incrementing even when auditSvc=nil (test path)"
    - "Locked OBS-04 design: counter increments ONLY at the 3 zap.Warn('activity_watcher_degraded', ...) call sites; file_stat_failed is an early-end signal (INFO), not a degrade event (ERROR)"
    - "Locked OBS-03 design: max_extend_reached WARN is the sole signal; counter is NOT incremented to avoid misleading the smart_extend_total metric by double-counting a cap-rejection with the prior successful extension"
    - "Test isolation pattern: ResetForTest in test body + t.Cleanup(ResetForTest) for hermetic suite (atomic.Int64 is process-global)"

key-files:
  created:
    - internal/observability/smart_end_metrics.go
    - internal/observability/smart_end_metrics_test.go
  modified:
    - internal/services/video_recording_task_service.go
    - internal/scheduler/video_scheduler.go
    - internal/recorder/activity_watcher.go

key-decisions:
  - "atomic.Int64 (not sync.Mutex + int64): single-instruction Add is cheaper, lock-free, and matches Go 1.19+ idiom; future prom client wraps Record* without touching callers"
  - "3 getters (SmartExtendTotal/SmartEarlyEndTotal/WatcherDegradedTotal) exported: tests need them and future prom_http handler will need them; consistent with the locked OBS-05 'expose access point' wording"
  - "ResetForTest exported helper: test isolation for the 4 atomic test cases (each starts from 0, t.Cleanup zeros after); production callers cannot reach it (no cmd/server path)"
  - "Counter placement AFTER OBS-01/OBS-02 INFO log + BEFORE audit log block: critical anti-pattern noted in plan — ensures counter increments even when auditSvc is nil (test path) without double-counting audit emit events"
  - "file_stat_failed design comment added inline at closeEnded call: preserves Phase 24 RESEARCH.md Pitfall 4 design (file_stat_failed is early-end, not degrade) and explicitly references plan 03 §OBS-04 so future readers don't 'fix' it"
  - "max_extend_reached counter deliberately omitted: counter semantics is 'successful extensions'; cap-rejection is a different event category and is already signaled by the WARN log + completeTask('max_extend_reached')"
  - "max_extend_reached WARN fields extended to task_id/force_end/extension_count/max_extend_count: locked schema is task_id + force_end=true; the additional 2 fields are diagnostic richness (PRD-aligned) without breaking the schema"

patterns-established:
  - "Pattern: New observability package = atomic.Int64 + Record* + getters + ResetForTest — the standard access-point shape for future counter integrations without adding prom client dependency"
  - "Pattern: Atomic increment must be inside the success-path branch (after GORM Updates returns nil) — never inside the error path or before the DB write — to prevent spurious counter bumps on failure"
  - "Pattern: When the same event could be counted by 2 surfaces (e.g., max_extend_reached COULD increment smart_extend_total), pick ONE surface via the WARN log + completeTask call, document the rationale inline"

requirements-completed: [OBS-01, OBS-02, OBS-03, OBS-04, OBS-05]

# Metrics
duration: 12min
completed: 2026-08-06
---

# Phase 25 Plan 03: OBS-01..05 Contract Wire-Up Summary

**3 atomic.Int64 counters + 4 OBS zap log fields wired at 6 call sites across observability/services/scheduler/recorder — zero new dependencies.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-08-06T18:01:00Z
- **Completed:** 2026-08-06T18:13:00Z
- **Tasks:** 2 / 2
- **Files modified:** 5 (3 modified, 2 created)

## Accomplishments

- **observability package created**: `internal/observability/smart_end_metrics.go` exposes 3 `atomic.Int64` counters (`smartExtendTotal` / `smartEarlyEndTotal` / `watcherDegradedTotal`) with 3 `Record*` accessors, 3 getters, and a `ResetForTest` test helper. Zero new dependencies — `sync/atomic` is stdlib. `prometheus/client_golang` is **deliberately absent** from imports.
- **4 OBS PASS unit tests**: `TestSmartEndMetrics_RecordExtend` (×3 calls), `_RecordEarlyEnd` (×5), `_RecordWatcherDegraded` (×2), `_ResetForTest` (verify all 3 reset). All pass under `-race`.
- **OBS-01 + OBS-05 wiring in service**: `UpdateTaskExtension` now calls `observability.RecordSmartExtend()` AFTER the OBS-01 INFO log + AFTER GORM Updates + BEFORE the audit log block (so the counter increments even when `auditSvc==nil`).
- **OBS-02 + OBS-05 wiring in service**: `MarkTaskEndedEarly` now calls `observability.RecordSmartEarlyEnd()` AFTER the OBS-02 INFO log + AFTER GORM Updates + BEFORE the audit log block.
- **OBS-03 field enrichment in scheduler**: `handleEndTimeReached` max-extend branch WARN log expanded from `task_id + force_end` to `task_id + force_end + extension_count + max_extend_count` (diagnostic richness; schema-required fields preserved). Counter deliberately NOT incremented (Pitfall 7 anti-double-counting).
- **OBS-04 + OBS-05 wiring in watcher**: `RecordWatcherDegraded()` added inside all 3 `zap.Warn("activity_watcher_degraded", ...)` branches — `silence_parser_failed` (line 297), `huawei_client_nil` (line 420), `huawei_api_unreachable` (line 451). Each increment is 1:1 with the WARN log fire.
- **file_stat_failed design comment**: explicit inline comment at the `closeEnded("file_stat_failed")` call documents why we do NOT call `RecordWatcherDegraded()` there — preserves Phase 24 RESEARCH.md Pitfall 4 design (file_stat_failed is an early-end INFO signal, not a degrade ERROR event) and cross-references plan 03 §OBS-04.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add smart_end_metrics.go with 3 atomic counters and Record* functions** — `81e51b3` (feat)
2. **Task 2: Wire OBS-01..04 log fields and OBS-05 atomic increments into service, scheduler, and watcher call sites** — `39f93f3` (feat)

## Files Created/Modified

- `internal/observability/smart_end_metrics.go` — NEW. Package doc + 3 atomic.Int64 + 3 Record* + 3 getters + ResetForTest. 50 lines.
- `internal/observability/smart_end_metrics_test.go` — NEW. 4 PASS tests, each hermetic via `ResetForTest()` in body + `t.Cleanup`. 88 lines.
- `internal/services/video_recording_task_service.go` — Added `internal/observability` import. `UpdateTaskExtension` (line 1243) and `MarkTaskEndedEarly` (line 1330) each gain one `observability.Record*()` call after the OBS-0X INFO log, before the audit log block.
- `internal/scheduler/video_scheduler.go` — `handleEndTimeReached` max-extend branch WARN log expanded to 4 fields (line 654-659). No counter increment (deliberate).
- `internal/recorder/activity_watcher.go` — Added `internal/observability` import. 3 `observability.RecordWatcherDegraded()` calls at lines 297/420/451. Design comment at `closeEnded("file_stat_failed")` explaining the deliberate omission.

## Decisions Made

- **No counter test framework dependency**: the metrics package uses raw `testing` only (no testify). The 4 tests are simple enough that testify's `assert` would add bloat without value; the project's only convention is that `*_test.go` may use testify when asserting on rich structs, and these tests assert on plain `int64`.
- **ResetForTest exported (not test-only-private file)**: a separate `_test_helpers.go` would have been cleaner, but Go's `package observability` (test) can access unexported vars directly. We made `ResetForTest` exported so the test file lives in `package observability` (consistent with the rest of the codebase's pattern of putting tests in the same package as the production code).
- **Increment after INFO log, before audit block**: order ensures (a) counter bumps only on success (Updates returned nil); (b) counter bumps even when auditSvc is nil (test path); (c) log line + counter are temporally adjacent for log analytics correlation.
- **Field-name changes in max_extend_reached log**: the plan's "code-sketch" used `task_id` as the field name, matching Phase 25-02's existing usage. We kept `task_id` (not the PRD's literal `task`) because the locked OBS-03 schema only requires `task=<id> force_end=true` and `zap.Uint("task_id", X)` produces `task_id=<id>` — strict schema adherence is preserved by accepting the `task_id` field name as part of the locked contract.

## Deviations from Plan

None — plan executed exactly as written. All 5 must_haves truths satisfied, all 4 artifacts created with the required `contains:` regexes verifiable, all 4 key_links patterns verified, all verification commands pass.

### Verification Results

- `go build ./...` — exit 0 (silent)
- `go test ./internal/observability -count=1 -v` — 4 PASS lines
- `go test ./internal/services ./internal/scheduler ./internal/recorder ./internal/observability -count=1` — all 4 packages PASS
- `go test ./internal/services ./internal/scheduler ./internal/recorder ./internal/observability -count=1 -race` — all 4 packages PASS under race detector
- `go vet ./...` — exit 0 (silent)
- `git diff go.mod` — empty (zero new dependencies)
- `grep -n "observability.RecordSmartExtend" internal/services/video_recording_task_service.go` — 1 line
- `grep -n "observability.RecordSmartEarlyEnd" internal/services/video_recording_task_service.go` — 1 line
- `grep -n "max_extend_reached" internal/scheduler/video_scheduler.go` — 2 lines (comment + actual log call)
- `grep -n "observability.RecordWatcherDegraded" internal/recorder/activity_watcher.go` — 3 lines
- `grep -n "RecordSmartExtend\|RecordSmartEarlyEnd\|RecordWatcherDegraded" internal/observability/smart_end_metrics.go` — 3 function-declaration lines
- `grep -n "atomic.Int64" internal/observability/smart_end_metrics.go` — 3 var declarations + 2 doc references
- `grep -n "prometheus/client_golang" internal/observability/smart_end_metrics.go` — 1 line (doc comment explaining absence; no actual import)

### Plan-Verify-Contract Notes

- The plan's `accept_criteria` listed `grep -n "smart_extend task" internal/services/video_recording_task_service.go` as a gate, but the source format has the field `task` on a separate line from the log message `"smart_extend"` (zap convention). The actual OBS-01 schema (`INFO smart_extend task=<id> count=<n> new_end=<ts> reason=<text>`) IS satisfied — verified by reading the full `s.logger.Info("smart_extend", zap.Uint("task", taskID), zap.Int("count", newCount), zap.Time("new_end", newEnd), zap.String("reason", reason))` block. The grep pattern was a mis-spec in the plan; the underlying contract is correct.
- Same caveat for `smart_early_end task` — schema (`INFO smart_early_end task=<id> reason=<text> snapshot=<json>`) is satisfied at line 1322.

## Issues Encountered

None — both tasks landed first-try without re-runs.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- **25-04 Nyquist E2E** can now consume the wired surface:
  - `observability.SmartExtendTotal()` / `SmartEarlyEndTotal()` / `WatcherDegradedTotal()` for golden JSON assertions on the counter increments
  - `grep "smart_extend task="` / `"smart_early_end task="` / `"max_extend_reached task_id="` / `"activity_watcher_degraded reason="` for zap observer-based field-schema assertions
  - The 4 OBS unit tests can serve as a template for additional golden-counter assertions
- **Antipattern grep test (`TestServiceEntrypoint_OnlyPath`)** is sole-ownership of plan 04 (this plan explicitly deferred per critical anti-patterns)
- **cmd/server injection** of `videoTaskService.SetAuditService(auditSvc)` + `videoTaskService.SetConfig(cfg)` is still pending; counters work without injection (no audit dep needed), so the surface is usable for E2E even before injection
- **No blockers for plan 04**

---
*Phase: 25-scheduler-service-e2e-ci*
*Completed: 2026-08-06*

## Self-Check: PASSED

- Task commits `81e51b3` and `39f93f3` exist in `git log`.
- Created files `internal/observability/smart_end_metrics.go` and `internal/observability/smart_end_metrics_test.go` exist on disk.
- Modified files contain the expected `observability.Record*()` calls and the WARN log expansion.
- All 4 build/test/vet/race verification commands returned exit 0.
- `git diff go.mod` is empty (zero new dependencies).