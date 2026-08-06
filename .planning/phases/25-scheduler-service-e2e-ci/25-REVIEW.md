---
phase: 25-scheduler-service-e2e-ci
reviewed: 2026-08-06T00:00:00Z
depth: quick
files_reviewed: 12
files_reviewed_list:
  - internal/observability/smart_end_metrics.go
  - internal/observability/smart_end_metrics_test.go
  - internal/recorder/activity_watcher.go
  - internal/recorder/activity_watcher_test.go
  - internal/recorder/coordinator.go
  - internal/recorder/coordinator_test.go
  - internal/scheduler/task_ended_channel_helper.go
  - internal/scheduler/task_ended_channel_helper_test.go
  - internal/scheduler/video_scheduler.go
  - internal/scheduler/video_scheduler_test.go
  - internal/services/video_recording_task_service.go
  - internal/services/video_recording_task_service_test.go
findings:
  critical: 2
  warning: 3
  info: 0
  total: 5
status: issues_found
---

# Phase 25: Code Review Report

**Reviewed:** 2026-08-06
**Depth:** quick
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Reviewed Phase 25 scheduler-service-e2e-ci changes at quick depth (pattern-matching plus focused read-through of changed code paths). All targeted security and resource patterns (hardcoded creds, eval/exec, SQL injection, dangling context cancels, AUDIT-04 direct `db.Update` bypasses) are clean. **Two Critical correctness bugs break the core Phase 25 feature**: (1) the scheduler's taskEndedCh preemption path (`SCHED-03`/`EARLY-03`) reads from a dead channel because the coordinator wires a separate never-closed channel instead of the watcher's live `EndedCh()`; (2) `restartRecording` closes the watcher's stderr logFile without rebinding it, leaving the `silenceScanner` goroutine blocked on `ctx.Done()` for the remainder of the recording (and silently disabling silence detection / file-tick growth detection on the new mkv). Both bugs defeat the Nyquist/E2E test suite because the scheduler tests inject channels directly via `mockCoordinatorWithChannels` and never exercise the real `WatcherChannels` plumbing.

The AUDIT-04 single-entry contract holds: scheduler does not directly write `VideoRecordingTask` smart-end fields (`extension_count` / `ended_early` / `last_extension_reason` / `ended_early_reason` / `ended_by_huawei_api`) — writes go through `UpdateTaskExtension` / `MarkTaskEndedEarly` via interface assertion in `handleEndTimeReached` / `handleTaskEnded`. The antipattern grep tests (`TestServiceEntrypoint_OnlyPath`, `TestScheduler_DoesNotDirectlyUpdateTask`) correctly find zero matches.

## Critical Issues

### CR-01: scheduler's taskEndedCh path reads from a dead channel — SCHED-03 / EARLY-03 broken in production

**File:** `D:\CODE\ClaudeCode\record_V2\internal\recorder\coordinator.go:201,1063` (read site), `internal/recorder/coordinator.go:199-217` (set site)
**Issue:** `StartRecordingWithConfig` allocates a fresh buffered channel and stores it on `rec.taskEndedCh` (line 200-201). This channel is **never closed and never written to** anywhere in the codebase (verified via Grep — the only references are the assignment, the nil-check, and the `WatcherChannels` return). The live close-once channel lives inside `ActivityWatcher.taskEndedCh` (created in `NewActivityWatcher`, closed by `closeEnded` on signal/degradation paths), exposed via `ActivityWatcher.EndedCh()`. But `WatcherChannels(taskID)` returns the dead `proc.taskEndedCh` instead of `proc.ActivityWatcher.EndedCh()`. Result: when the scheduler does

```go
taskEndedCh := mergeWatchers(ctx, s.coordinator.WatcherChannels(task.ID), s.logger)
```

the merged channel **never fires** — silence/stall/huawei/early-end signals from the watcher cannot preempt the `EndTime` deadline in production. The whole SCHED-03 / EARLY-03 / EARLY-04 fan-in feature is a no-op against the real coordinator. Tests pass only because `TestMonitorTask_TaskEnded_PreemptsTimer` and `TestMonitorTask_MultiInput_AnyEndsAll` inject channels via `mockCoordinatorWithChannels`, bypassing `WatcherChannels` entirely. `TestRecorderCoordinator_WatcherChannels` only asserts slice length, never verifies channels actually close.

**Fix:** Make `proc.taskEndedCh` an alias for the watcher's live channel. Either (a) make `ActivityWatcher.taskEndedCh` exported/exposed via a setter and assign it in `coordinator.go`:

```go
rec.ActivityWatcher = NewActivityWatcher(...)
rec.taskEndedCh = rec.ActivityWatcher.EndedCh() // bridge to live channel
```

or (b) delete the dead `rec.taskEndedCh` field entirely and change `WatcherChannels` to return `proc.ActivityWatcher.EndedCh()` for each non-nil watcher. Option (a) keeps the existing field shape (no API change for `RecorderCoordinatorInterface` consumers) and matches the comment at line 80-83 ("Phase 24 仅构造不消费, Phase 25 接线"). Add an end-to-end test that exercises the real `SimpleRecordingCoordinator.WatcherChannels` round-trip (e.g., trigger a watcher degradation path and assert the returned channel closes within a deadline).

### CR-02: restartRecording leaves silenceScanner / fileTicker stuck on the OLD logFile/mkvPath — auto-end disabled for the rest of the recording after first reconnect

**File:** `D:\CODE\ClaudeCode\record_V2\internal\recorder\coordinator.go:418-421,419` (close site), `internal/recorder/activity_watcher.go:262-317` (silenceScanner), `internal/recorder/activity_watcher.go:336-394` (fileTicker)
**Issue:** On reconnect (`restartRecording` path), `process.logFile` is closed (line 418-421) and a new ffmpeg process is started with a *new* mkvPath / logFile (line 393-448). But the existing `ActivityWatcher` instance is reused unchanged:
- `silenceScanner` holds a `bufio.NewScanner(w.logFile)` (line 269) bound to the OLD (now closed) `logFile`. After `process.logFile.Close()`, `scanner.Scan()` returns false; the for-loop exits; the goroutine blocks on `<-ctx.Done()` (line 316) for the rest of the recording. **Silencedetect A-path stops producing events** — `Snapshot().SilenceSince` is only cleared (by `OnReconnect` at line 425-427), never set again. The A+B close condition (`!silSince.IsZero()`) at `activity_watcher.go:517-522` is unreachable for the remainder of the recording.
- `fileTicker` reads `w.filePath` (line 354), which was set to the FIRST `mkvPath` in `StartRecordingWithConfig`. After reconnect, the new mkvPath has a new timestamp (line 249: `time.Now().Format(...)`), so `w.filePath` is stale. `lastFileGrowthAt` / `lastFileSize` / `statConsecFailures` are updated against an orphaned file the new ffmpeg process never writes to. **B-path stalls are invisible**, and the `statConsecFailures` counter can erroneously accumulate against the old file, potentially triggering a spurious `closeEnded("file_stat_failed")` if the old file is deleted/moved.
- `decisionTicker` (line 488-526) keeps running and reads `silSince` / `lastGrowth` — but both are now stale, so it correctly emits no further close signals. Effectively the watcher is dead-on-arrival for any subsequent H/A/B convergence.

The leak is bounded (only one stale silenceScanner per `ActivityWatcher` lifetime, released at `Stop()`), but it produces silent functional regression: only the H-path (huawei state) can close the recording after a reconnect. There is no test that exercises `restartRecording` end-to-end against a real watcher (only `TestAttemptReconnectReturnsImmediatelyWhenContextCanceled` covers attemptReconnect timing).

**Fix:** In `restartRecording`, tear down and recreate the watcher alongside the new ffmpeg process:

```go
// stop old watcher before closing its logFile
if process.ActivityWatcher != nil {
    process.ActivityWatcher.Stop()
}
// ... close logFile, build new mkvPath/logFile, start ffmpeg ...
process.ActivityWatcher = NewActivityWatcher(c.config, c.huaweiClient(), mkvPath, logFile, c.logger)
process.OnReconnect = process.ActivityWatcher.OnReconnect
process.taskEndedCh = process.ActivityWatcher.EndedCh() // also fixes CR-01
process.ActivityWatcher.Start()
```

Add an integration test that performs one reconnect and asserts that `watcher.Snapshot().LastFileGrowthAt` advances against the new mkv file (or at minimum that `silenceScanner` does not block).

## Warnings

### WR-01: dead assignment in huaweiPoller failure path

**File:** `D:\CODE\ClaudeCode\record_V2\internal\recorder\activity_watcher.go:440,454`
**Issue:** `empty := w.huaweiEmptySince` (line 440) is assigned inside the lock and then immediately discarded via `_ = empty` (line 454) with a comment claiming "失败时保留 huaweiEmptySince (不重置) — 失败不代表'会议恢复'". The assignment is a no-op — nothing mutates `w.huaweiEmptySince` in this branch (the unlock at line 441 doesn't write, only reads). Either the original intent was to capture a value for some downstream consumer (and the consumer was removed), or the assignment is leftover from an earlier draft. Code smell; harmless at runtime but misleading.

**Fix:** Remove lines 440 and 454. If the intent was to document the read order, replace the comment with a plain English line above the `if consec >= ...` check.

### WR-02: cancelFuncs map populated but never read — unbounded memory growth

**File:** `D:\CODE\ClaudeCode\record_V2\internal\recorder\coordinator.go:26,120,193,452`
**Issue:** `SimpleRecordingCoordinator.cancelFuncs` map is written at line 193 (initial start) and line 452 (after reconnect), but there is **no read site anywhere in the package** (verified via Grep — zero match for `cancelFuncs[` outside the assignment sites). `StopRecording` does not delete from it. As tasks are started and reconnected, the map grows without bound for the lifetime of the coordinator. The actual cancellation is performed via `process.CancelFunc()` (line 630, 644), which already holds a per-process reference — so the map is genuinely redundant. Either dead code or forgotten cleanup.

**Fix:** Either remove the `cancelFuncs` field entirely (rely on `process.CancelFunc`) or document why both exist. If kept, add `delete(c.cancelFuncs, key)` in `StopRecording` and in the `restartRecording` failure path.

### WR-03: snapshot ExtensionCount / NewEndTime OldData drift in audit log (Pitfall 4 partially mitigated)

**File:** `D:\CODE\ClaudeCode\record_V2\internal\services\video_recording_task_service.go:1255-1274`
**Issue:** `UpdateTaskExtension` builds a local `oldMap` (line 1255-1258) that captures `task.EndTime` *after* `task.EndTime = newEnd` has been applied at line 1230 — so `oldMap["end_time"]` is the NEW value, not the OLD value. The code then discards `oldMap` via `_ = oldMap` and writes only `{"extension_count": extensionOld}` as OldData. The comment at line 1261-1263 acknowledges this is a deliberate "Pitfall 4" mitigation ("end_time 旧值由 audit_ui 按 DiffData 自动呈现"). The behavior is documented and consistent with `TestMarkTaskEndedEarly_AuditSnapshot` (which asserts only `ended_early: false` in OldData), but it does mean that a downstream audit-log consumer that does not compute DiffData will see the same value in both OldData and NewData for end_time — a quiet data-integrity gap. Same applies to `MarkTaskEndedEarly` (line 1339-1343): OldData only contains `{"ended_early": false}`, omitting `ended_early_reason` / `ended_by_huawei_api` old values.

**Fix:** Capture the original task snapshot *before* `Updates()` mutates the in-memory struct. Re-design:
```go
oldTask := task  // snapshot at line 1204, BEFORE Updates
// ... Updates(...) ...
oldMap := map[string]interface{}{
    "end_time":        oldTask.EndTime,
    "extension_count": oldTask.ExtensionCount,
}
```
Apply the same pattern to `MarkTaskEndedEarly`. Document the resulting OldData schema so audit_ui can drop the DiffData fallback.

---

_Reviewed: 2026-08-06T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: quick_