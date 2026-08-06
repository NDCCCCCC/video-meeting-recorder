---
phase: 23-api-gorm-sentinel
plan: 01
subsystem: huawei
tags: [detect-01, detect-04, huawei-client, mailbox-state, tdd]
dependency_graph:
  requires: [23-03]
  provides:
    - HuaweiClient.GetConferenceState (stateless sample API)
    - MailboxState.State.{ConfState, JoinSum, ConfLeftTime} fields
    - ConferenceState struct with HasConferenceFields flag
    - parseMailboxState helper (sentinel-wrapped parse errors)
    - detectConferenceFields helper (JSON key presence detection)
  affects:
    - internal/huawei/client.go
    - internal/huawei/client_test.go
tech-stack:
  added: []
  patterns:
    - Stateless integration client (no timers/goroutines in shared client)
    - presence-aware JSON parsing via json.RawMessage
    - Sentinel-wrapped error returns on parse failure (never silent zero-value success)
key-files:
  created: []
  modified:
    - internal/huawei/client.go
    - internal/huawei/client_test.go
decisions:
  - "GetConferenceState is stateless — 30s persistence belongs to Phase 24 watcher"
  - "confState/joinSum presence tracked separately from zero values (old devices get HasConferenceFields=false)"
  - "parseMailboxState wraps ErrRecordingHuaWeiStateFetchFailed on empty/malformed payload (no silent fake empty meeting)"
  - "getMailboxRaw duplicates GetMailboxData HTTP plumbing instead of refactoring (preserves keep-alive goroutine test expectations)"
metrics:
  duration: "~5 minutes"
  tasks: 1
  files_changed: 2
  commits: 3
  completed_date: "2026-08-06"
---

# Phase 23 Plan 01: Huawei Client H-Signal Boundary Summary

## One-liner

Stateless `GetConferenceState` boundary exposes `confState`/`joinSum`/`confLeftTime` plus `HasConferenceFields` flag, with sentinel-wrapped parse errors replacing the silent zero-value fallback in `GetMailboxData`.

## What Was Built

`internal/huawei/client.go` was extended with the H-signal data path:

1. **`MailboxState.State` extended** with `ConfState` (string), `JoinSum` (int), `ConfLeftTime` (int). All three use `omitempty` so old TE40 firmware without these fields continues to parse cleanly into zero values; existing 9 fields remain unchanged.
2. **`parseMailboxState(data string) (*MailboxState, error)`** — package-private strict double-decode helper. Empty data, missing `state` field, wrong state type, or inner parse failure all return an error wrapping `apperrors.ErrRecordingHuaWeiStateFetchFailed` (Phase 23 AUDIT-05 sentinel). No silent zero-value success path.
3. **`detectConferenceFields(rawData string) bool`** — package-private presence-detection helper using `json.RawMessage` to distinguish JSON-absent keys from JSON-present-but-empty values. Returns true only when **both** `confState` AND `joinSum` keys are present.
4. **`ConferenceState` struct** — exported one-sample value type with 5 fields: `ConfState`, `JoinSum`, `ConfLeftTime`, `IsInConf` (fallback), `HasConferenceFields` (presence flag). No timers, no goroutines, no shared state.
5. **`getMailboxRaw(ctx context.Context) (string, error)`** — package-private sibling that intentionally duplicates `GetMailboxData`'s 8-line HTTP/session plumbing to avoid regressing existing keep-alive tests.
6. **`(*HuaweiClient).GetConferenceState(ctx) (*ConferenceState, error)`** — exported stateless method. Calls `getMailboxRaw` once, runs `parseMailboxState`, builds `ConferenceState`, computes `HasConferenceFields` from the raw data via `detectConferenceFields`. Returns wrapped sentinel on transport or parse failure.

`internal/huawei/client_test.go` was extended with 7 fixture-based subtests plus a top-level wrapper:

| Subtest | Verifies |
|---------|----------|
| `AllFieldsPresent` | full new-field payload parses correctly; detectConferenceFields=true |
| `EmptyMeeting` | explicit empty/zero new fields still classify as HasConferenceFields=true (not fallback) |
| `OldDeviceNoFields` | fixture without new fields parses with zero values; detectConferenceFields=false |
| `PartialFields` | only one of confState/joinSum present → detectConferenceFields=false (no partial application) |
| `EmptyData` | empty data string returns wrapped sentinel (not zero-value success) |
| `MalformedJSON` | three malformed variants all return wrapped sentinel |
| `GetConferenceState_FallbackFlag` | old-device fixture yields ConferenceState{HasConferenceFields=false, IsInConf=0, ConfState="", JoinSum=0} |

Top-level `TestGetConferenceState_FallbackFlag` mirrors the subtest for acceptance-criteria discoverability.

## TDD Gate Compliance

| Gate | Commit | Notes |
|------|--------|-------|
| RED (test) | `a463813` | Tests added; build failed at compile (undefined symbols) |
| GREEN (feat) | `9d4e7d8` | Production code added; all 7 subtests pass; existing `TestHuaweiSanitizeResponseBody` and `TestHuaweiClient_StopExitsKeepAliveGoroutine` unchanged and pass |
| REFACTOR | `cd60818` | Top-level `TestGetConferenceState_FallbackFlag` wrapper added; no production-code change |

All three gates are sequential in `git log` and follow the RED-before-GREEN-before-REFACTOR ordering.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Test fixture encoding mismatch**

- **Found during:** First GREEN test run (after writing production code)
- **Issue:** Initial `buildMailboxData` helper encoded the `state` field as a JSON-escaped string (`{"state":"{...}"}`) instead of an inline JSON object (`{"state":{...}}`). The real TE40 firmware returns the latter. All 4 success-path subtests failed with `mailbox state parse failed` because the inner `json.Unmarshal(wrapper.State, &mailboxState.State)` saw a string instead of an object.
- **Fix:** Changed helper to put `state` as a raw `map[string]interface{}` so `json.Marshal` produces an inline object. Tests passed after the one-line fix.
- **Files modified:** `internal/huawei/client_test.go`
- **Commit:** `9d4e7d8` (rolled into the GREEN commit)

**2. [Rule 1 - Action/acceptance contradiction] Top-level test alias**

- **Found during:** Final acceptance-criteria audit (after GREEN commit)
- **Issue:** Plan action section says "7 subtests inside one TestParseMailboxState function" but plan acceptance_criteria literally lists `TestGetConferenceState_FallbackFlag` as a top-level test function. Both interpretations could be defended; the strict reading favors the top-level function.
- **Fix:** Added a 30-line top-level `TestGetConferenceState_FallbackFlag` wrapper that exercises the same path as the subtest. Kept the subtest in `TestParseMailboxState` to satisfy the action section's table-driven preference.
- **Files modified:** `internal/huawei/client_test.go`
- **Commit:** `cd60818`

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| `client.go` contains `func (c *HuaweiClient) GetConferenceState(ctx context.Context)` | ✓ at line 861 |
| `client.go` contains package-private `parseMailboxState(data string)` | ✓ defined after MailboxState |
| `client.go` exports `ConferenceState` with ConfState/JoinSum/ConfLeftTime/IsInConf/HasConferenceFields | ✓ at line 278 |
| `MailboxState.State` extended with `confState`/`joinSum`/`confLeftTime` json tags | ✓ alongside existing IsInConf |
| `client_test.go` has `TestParseMailboxState` with 7 subtests (AllFieldsPresent, EmptyMeeting, OldDeviceNoFields, PartialFields, EmptyData, MalformedJSON) | ✓ all 7 subtests pass |
| `client_test.go` has top-level `TestGetConferenceState_FallbackFlag` | ✓ added in refactor commit |
| parseMailboxState wraps `ErrRecordingHuaWeiStateFetchFailed` | ✓ verified via `errors.Is` in EmptyData + MalformedJSON subtests |
| `TestHuaweiSanitizeResponseBody` and `TestHuaweiClient_StopExitsKeepAliveGoroutine` still pass | ✓ no regression (no source changes to existing functions) |
| `go test ./internal/huawei -count=1 -race` passes | ✓ all tests pass under race detector |
| `go vet ./internal/huawei` exits 0 | ✓ clean |
| `go build ./...` passes | ✓ no callers broken |

## Self-Check

- `internal/huawei/client.go` exists and contains all required symbols
- `internal/huawei/client_test.go` exists and contains 7 subtests + top-level wrapper
- Commits `a463813` (RED), `9d4e7d8` (GREEN), `cd60818` (REFACTOR) all reachable in `git log`

## Self-Check: PASSED

## Notes for Phase 24 Consumer

`ActivityWatcher` will call `client.GetConferenceState(ctx)` once per polling tick. The watcher must:

1. Use `HasConferenceFields` to choose between new H-criterion and old `IsInConf` fallback.
2. Track `emptySince = now()` per recording instance (NOT inside the shared client) when `HasConferenceFields && ConfState=="" && JoinSum==0`.
3. Reset on any active sample.
4. Increment failure counter (not emptySince) on `errors.Is(err, ErrRecordingHuaWeiStateFetchFailed)`.
5. Apply `huawei_persist_s` threshold after continuous empty samples reach it.

The shared client holds no per-recording state, so concurrent recording tasks cannot contaminate each other's `emptySince`.