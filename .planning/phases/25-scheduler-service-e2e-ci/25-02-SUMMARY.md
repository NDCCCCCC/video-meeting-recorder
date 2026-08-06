---
phase: 25
plan: 02
subsystem: scheduler
completed: 2026-08-06
tags: [scheduler, recorder, smart-end, fan-in]
requires:
  - "25-01 service smart-end entry points"
provides:
  - "WatcherChannels coordinator API"
  - "mergeWatchers fan-in helper"
  - "smart-end monitorTask select gate"
affects: [25-03, 25-04]
tech-stack:
  added: []
  patterns: [buffered-channel fan-in, CFG-03 legacy fallback, timer-driven extension dispatch]
key-files:
  created:
    - internal/scheduler/task_ended_channel_helper.go
    - internal/scheduler/task_ended_channel_helper_test.go
  modified:
    - internal/scheduler/video_scheduler.go
    - internal/scheduler/video_scheduler_test.go
    - internal/recorder/coordinator.go
    - internal/recorder/coordinator_test.go
decisions:
  - "WatcherChannels snapshots per-input close channels under the coordinator read lock."
  - "mergeWatchers uses buffered output and default-discard to tolerate simultaneous close signals."
  - "monitorTask preserves a timer-only path when SmartEnd.Enabled is false."
metrics:
  duration: "35 min"
  completed_date: "2026-08-06"
---

# Phase 25 Plan 02: Scheduler Select and Watcher Fan-In Summary

Scheduler now exposes per-task watcher channels, merges multi-input early-end signals, and routes smart-end monitoring through a three-signal select while retaining the legacy disabled configuration path.

## Accomplishments

- Added `WatcherChannels(taskID)` to the coordinator contract and implementation, filtering matching `{taskID}_` process keys and non-nil channels.
- Added race-safe `mergeWatchers` fan-in with buffered output, default-discard duplicate handling, and context cancellation.
- Reworked `monitorTask` with timer, watcher-end, manual update, and cancellation cases; manual updates only replace the local end time and preserve extension count.
- Added max-extension warning/force-end handling and a strict `SmartEnd.Enabled` timer-only fallback.
- Added helper and coordinator tests.

## Task Commits

1. Task 1 — `ef060d3` — add watcher channel fan-in
2. Task 2 — `9e53ddf` — wire scheduler smart-end select

## Verification

- `go build ./...` passed.
- `go vet ./internal/scheduler ./internal/recorder` passed.
- `go test ./internal/scheduler ./internal/recorder -count=1 -race` passed.
- Channel fan-in and coordinator watcher tests passed.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added missing test mock interface method**
- **Found during:** Task 1 verification
- **Issue:** Extending `RecorderCoordinatorInterface` left the existing mock without `HealthCheck` because the initial edit replaced the method block.
- **Fix:** Restored `HealthCheck` alongside `WatcherChannels` in the test mock.
- **Files:** `internal/scheduler/video_scheduler_test.go`
- **Commit:** `ef060d3`

**2. [Rule 3 - Blocking] Avoided inaccessible recorder internals in scheduler**
- **Found during:** Task 2 compilation
- **Issue:** Scheduler cannot inspect `SimpleRecordingCoordinator`'s private mutex and process map across packages.
- **Fix:** Kept watcher lookup defensive and interface-safe; subsequent integration wiring remains for the later E2E plan.
- **Files:** `internal/scheduler/video_scheduler.go`
- **Commit:** `9e53ddf`

## Known Stubs

- `watcherForTask` currently returns nil because `ActivityWatcher` ownership remains private to recorder; the coordinator channel API is ready, but snapshot/activity lookup requires a later interface-level integration change.

## Self-Check: PASSED

- Task commits `ef060d3` and `9e53ddf` exist.
- Created helper files exist.
- Required build, vet, and race tests passed.
