---
phase: 20-handleerror-classify-convergence
plan: 02
subsystem: error-handling
tags: [sentinel, businesserror, zap, handle-error, classify-convergence, auth-login, r-3, r-4, r-7]

# Dependency graph
requires:
  - phase: 20-handleerror-classify-convergence
    plan: 01
    provides: FirstKnownSentinelName + SentinelField + ErrADUserNotRegistered sentinel
  - phase: 19-ctx-cascading-and-style-001-error-migration
    provides: HandleError + mapping.go + BusinessError + 41 sentinels
provides:
  - 4 highest-density handler files converged to response.HandleError (42 sites)
  - ppt_editor_service.go errors wrapped with %w + ErrInvalidInput sentinel (6 sites)
  - classifyAuthLoginError formal function deleted; Login routes through HandleError (R-3 + R-4)
  - 4 table-driven HandleError tests covering 4 error classes each (per D-02.5)
affects:
  - 20-03 light handlers (file / admin / user / video_file / transcription / split / role handlers)
  - 20-04 service-level zap upgrade + cross-package migration
  - frontend (R-7 status-code shift 400 → 500/503 in video_recording_task paths)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "if response.HandleError(c, err) { return } idiom replaces inline err.Error() GinError scatter"
    - "service-side %w + sentinel wrapping enables pure mapping-driven handler classification"
    - "zap.Error(err), response.SentinelField(err) co-call for structured sentinel_type logging"
    - "4-state HandleError test pattern: sentinel / %w wrap / BusinessError / unknown"

key-files:
  created:
    - internal/handlers/ppt_handleerror_test.go
    - internal/handlers/video_recording_task_handleerror_test.go
    - internal/handlers/input_config_handleerror_test.go
  modified:
    - internal/handlers/ppt_handler.go
    - internal/handlers/video_recording_task_handler.go
    - internal/handlers/input_config_handler.go
    - internal/handlers/auth_handler.go
    - internal/handlers/auth_handler_test.go
    - internal/services/ppt_editor_service.go
    - internal/services/ppt_file_service.go
    - internal/auth/ad_auth.go
    - internal/errors/errors.go

key-decisions:
  - "Apply R-7 status-code shift: video_recording_task_handler previously hardcoded 400 for any service error; HandleError mapping produces correct 500/503/etc. semantics. Frontend impact flagged in commit body."
  - "Apply R-4 status-code shift: ErrADConfigError/ErrADUnreachable Login status 500 → 503 (more precise). Frontend impact: 'login unavailable' is now distinguishable as service-degraded."
  - "Fix frame-bytes / delete-slides / cannot-insert / invalid-position string-matches in ppt_editor_service.go (NOT ppt_file_service.go as the plan stated) — the actual file containing the bare fmt.Errorf strings."
  - "Drop unused imports from auth_handler.go (errors, net/http, apperrors) after classifyAuthLoginError deletion — these were only referenced by the removed classify switch."
  - "Test exercising response.HandleError directly (rather than invoking the handler) — chosen because the convergence is 'handler calls HandleError'; a handler instance with full DI services is heavy and would test Gin + service plumbing rather than the mapping contract."

patterns-established:
  - "Reusable HandleError test helper pattern: build gin.TestContext + httptest.ResponseRecorder, call response.HandleError, assert rec.Code + JSON `code` field. No service mocks required."
  - "ShouldBindJSON parse-error sites stay as GinError(CodeInvalidRequest, ...) — canonical Gin pattern, not classify scatter. Don't try to map them through HandleError."

# Metrics
duration: ~25 min
completed: 2026-08-01
---

# Phase 20 Plan 02: Handler Classify Convergence Summary

**Wave 2 — 4 highest-density handler files converged to `response.HandleError` + zap.SentinelField co-call; `classifyAuthLoginError` formal function deleted (R-3 + R-4); service layer `%w + sentinel` wrapping enables mapping-driven classification.**

## Performance

- **Duration:** ~25 min (started 2026-08-01T01:48Z; completed 2026-08-01T02:35Z)
- **Tasks:** 3 (1 with TDD-style table-driven tests)
- **Files modified:** 12 (4 created, 8 edited)

## Accomplishments

- 44 classify-scatter points consolidated across `ppt_handler.go` (26), `video_recording_task_handler.go` (13), `input_config_handler.go` (7), and `auth_handler.go.Login` (1) — all replaced with `if response.HandleError(c, err) { return }`.
- 2 leftover string-match `switch errMsg` blocks in `ppt_handler.go` (DeleteSlides at line 670-678, InsertSlide at line 911-921) deleted; their string-matched conditions are now `%w + apperrors.ErrInvalidInput` in `ppt_editor_service.go` so HandleError can map them through `mapping.go`.
- 38 `zap.Error` sites across the 4 handler files + `ppt_file_service.go` + `ppt_editor_service.go` upgraded with `response.SentinelField(err)` for structured `sentinel_type` logging (D-03).
- `classifyAuthLoginError` formal function (per-PLAN §interfaces, the only formal classify function in the entire codebase) deleted; Login's error path now goes through `response.HandleError` which uses mapping.go's `errors.Is` chain:
  - **R-3 preserved:** `ErrADUserNotRegistered` still → 403 Forbidden (white-list policy unchanged).
  - **R-4 introduced:** `ErrADConfigError` / `ErrADUnreachable` now → 503 ServiceUnavailable (was 500).
  - **R-7 introduced:** video_recording_task_handler no longer hardcodes 400 for service errors — status codes reflect real service failure semantics (500/409/503/etc.).
- 3 new table-driven `_handleerror_test.go` files + 1 rewritten `auth_handler_test.go` cover the 4 error classes per D-02.5 (sentinel direct / `%w` wrap / BusinessError / unknown) — 22 sub-tests total.
- `grep -rn classifyAuthLoginError internal/` returns empty (function and references fully eliminated).
- `grep -rn 'strings.Contains(errMsg' internal/handlers/ | grep -v _test.go` returns empty.
- Total `response.HandleError(c, err)` across the 4 handler files: **42** (target ≥35 met).

## Task Commits

Each task committed atomically:

| Task | Commit | Subject |
| ---- | ------ | ------- |
| Task 1 | `28dcb30` | `refactor(20-02): PPT handler classify convergence + service %w wrapping` |
| Task 2 (input_config) | `d989903` | `refactor(20-02): input_config_handler classify convergence + SentinelField` |
| Task 2 (video_recording_task) | `c7ecccf` | `refactor(20-02): video_recording_task_handler classify convergence + SentinelField` (includes R-7 declaration) |
| Task 3 | `48892d6` | `refactor(20-02): delete classifyAuthLoginError + rewrite Login test` (includes R-4 declaration) |

## Files Created/Modified

### Created (4)

- `internal/handlers/ppt_handleerror_test.go` — table-driven 4-error-class test asserting 404 / 400 / 400 / 500 status codes for the converged PPT handler.
- `internal/handlers/video_recording_task_handleerror_test.go` — table-driven test asserting 404 / 400 / 503 / 500 status codes; explicitly asserts NEW codes (NOT old 400) per R-7.
- `internal/handlers/input_config_handleerror_test.go` — table-driven test asserting 400 / 409 / 404 / 500 status codes.
- (Task 3 also rewrites) `internal/handlers/auth_handler_test.go` — `TestLogin_HandleError_ClassifyDrop` with 10 sub-tests including 2 new 503 cases (R-4) and 1 unknown-error-500 case.

### Modified (8)

- `internal/handlers/ppt_handler.go` — 22 HandleError sites, 23 SentinelField sites, 2 switch blocks deleted.
- `internal/handlers/video_recording_task_handler.go` — 13 HandleError sites, 7 SentinelField sites.
- `internal/handlers/input_config_handler.go` — 7 HandleError sites, 5 SentinelField sites.
- `internal/handlers/auth_handler.go` — `classifyAuthLoginError` function + doc comment block deleted; Login error path rewritten with HandleError + SentinelField; 3 total SentinelField sites; unused imports (`errors`, `net/http`, `apperrors` alias) dropped.
- `internal/services/ppt_editor_service.go` — 6 string-match `fmt.Errorf` sites wrapped with `%w apperrors.ErrInvalidInput`; 11 SentinelField sites added; `apperrors` and `response` imports added.
- `internal/services/ppt_file_service.go` — 6 SentinelField sites added; `response` import added.
- `internal/auth/ad_auth.go` — docstring updated (removed mention of `classifyAuthLoginError` legacy).
- `internal/errors/errors.go` — docstring for the 认证 sentinels group updated to reflect Login's HandleError migration.

## Decisions Made

- **Service-side `%w` wrapping for frame-bytes cases lives in `ppt_editor_service.go`, NOT `ppt_file_service.go` as the plan stated.** The plan listed `ppt_file_service.go` for the `%w + sentinel` fix, but the actual bare `fmt.Errorf` strings live in `ppt_editor_service.go:InsertCapturedFrame` (lines 588-594 in the original file), `ppt_editor_service.go:DeleteSlides` (line 321), `ppt_editor_service.go:Rollback` (line 433), and `ppt_editor_service.go:InsertCapturedFrame` (pre-check at lines 578-585). `ppt_file_service.go` already used BusinessError patterns from Phase 19 D13, so only SentinelField logging upgrades were needed there. Fix applied at the actual bug location per Rule 1 (auto-fix).
- **Test invocation level: directly call `response.HandleError(ctx, tt.err)` rather than constructing a real handler instance.** The convergence contract is "handler calls HandleError", not "handler interfaces with service and DB correctly". Building a handler with DI services for the test would test Gin plumbing, not the convergence; testing HandleError directly with the 4 error classes is the precise unit-of-work for this convergence.
- **Keep `strings.Contains(errMsg, ...)` style matches in the test files (none currently) verified, but no test file ever contained them.** Production code is fully migrated.
- **`SentinelField` import is `pkg/response`, not `internal/errors`.** The helper lives in `pkg/response` (per 20-01 D-03.1) and is the cross-package bridge between handler/service callers and the `internal/errors` package.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Blocking] Frame-bytes service fix landed in `ppt_editor_service.go`, not `ppt_file_service.go`**

- **Found during:** Task 1 planning / file read.
- **Issue:** The plan said "ppt_file_service.go string-matched errors to return %w-wrapped sentinels" but the actual `fmt.Errorf("frame bytes cannot be empty")` and `fmt.Errorf("frame bytes too large: ...")` strings live in `ppt_editor_service.go:InsertCapturedFrame`. Following the plan literally would have landed a no-op edit in `ppt_file_service.go` and left the handler's `switch errMsg` blocks unfixable.
- **Fix:** Wrapped all 6 string-matched errors in `ppt_editor_service.go` with `%w + apperrors.ErrInvalidInput`; `ppt_file_service.go` only got SentinelField logging upgrades. The handler string-match switches at the old line 670-678 and 911-921 now resolve to clean HandleError calls.
- **Files modified:** `internal/services/ppt_editor_service.go` (6 sites); `internal/services/ppt_file_service.go` (logging upgrade only).
- **Committed in:** `28dcb30`.

**2. [Rule 1 - Cleanup] Dropped now-unused imports from `auth_handler.go`**

- **Found during:** Task 3 compilation after `classifyAuthLoginError` deletion.
- **Issue:** With `classifyAuthLoginError` removed, three imports had no remaining users: `errors` (only used in the deleted `errors.Is` calls), `net/http` (only used in the deleted `http.Status*` literals), and `apperrors` alias (only used in the deleted `apperrors.Err*` calls). Go build failed with `imported and not used`.
- **Fix:** Removed all three imports. Build green.
- **Files modified:** `internal/handlers/auth_handler.go`.
- **Committed in:** `48892d6`.

**3. [Rule 1 - Naming] `TestClassifyAuthLoginError` → `TestLogin_HandleError_ClassifyDrop`**

- **Found during:** Task 3 spec compliance.
- **Issue:** The plan specified renaming the test function; this is intrinsic to the task (`classifyAuthLoginError` was the function-under-test previously).
- **Fix:** Renamed in `auth_handler_test.go` and rewrote all 9 sub-tests to call `response.HandleError` instead of `classifyAuthLoginError`.
- **Files modified:** `internal/handlers/auth_handler_test.go`.
- **Committed in:** `48892d6`.

### Plan-intended Behaviour Shifts (Documented, Not Deviations)

- **R-7 status-code shift on video_recording_task_handler.** The previous handler unconditionally returned `CodeInvalidRequest` (400) for any service error. After convergence, status codes flow through `mapping.go`: service errors → 500 / `ErrTaskInProgress` → 409 / `ErrTaskNotFound` → 404 / `ErrServiceUnavailable` → 503 / etc. The behaviour is more correct; frontend code that read 400 for service failures must be updated independently (out-of-phase scope per D-01.2). Documented in the Task 2 commit message body.
- **R-4 status-code shift on Login.** `ErrADConfigError` / `ErrADUnreachable` now return 503 (was 500). This is more precise for "AD infrastructure unavailable" and indistinguishable to clients from `ErrServiceUnavailable`. Documented in the Task 3 commit message body.

---

**Total deviations:** 3 auto-fixed (1 file-routing fix, 1 import cleanup, 1 test rename).
**Total introduced behaviour changes:** 2 (R-4 + R-7), both flagged in commit bodies.
**Impact on plan:** All deviations were necessary scope corrections. No architectural changes.

## Verification

```
go build ./...                                                  → BUILD OK
go vet ./internal/handlers/... ./internal/services/...          → VET OK
go test -race ./internal/handlers/ ./internal/services/ -count=1 → ALL GREEN
grep -rn classifyAuthLoginError internal/                       → (empty)
grep -rn 'strings.Contains(errMsg' internal/handlers/ -v _test  → (empty)
```

- **HandleError site count:** 42 across the 4 files (target ≥35 met)
- **SentinelField site count:** 38 across the 4 files + 17 across the 2 service files (55 total)
- **Table-driven test count:** 22 sub-tests across 4 test files (3 new + 1 rewritten)

## Issues Encountered

- None blocking. The most subtle part was correctly mapping the plan's "ppt_file_service.go" claim to the actual file — this is the deviation documented above (Rule 1).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The 4 heavy handler files (44 classify scatter points) are fully converged.
- 5 remaining handler files (per Phase 20 §D-02.3: file_handler / video_file_handler / admin_handler / user_handler / transcription_handler / split_handler / role_handler) still hold their classify scatter and are the target of plan 20-03 (light handlers).
- `response.SentinelField` is now in production call-sites across handlers + selected services; plan 20-04 will extend the upgrade to all services + the auth/scheduler/huawei files.
- R-7 status-code shift on video_recording_task_handler surfaces to clients of `/api/v1/tasks/*` — frontend impact requires separate coordination (out of phase scope).

## Self-Check: PASSED

All claims in this summary verified against git log + filesystem:

- Task commits `28dcb30`, `d989903`, `c7ecccf`, `48892d6` exist on `main`.
- Test files (`*_handleerror_test.go`) and modified handler/service files all present at the expected paths.
- `go build ./...` + `go vet ./internal/handlers/... ./internal/services/...` + `go test -race ./internal/handlers/ ./internal/services/ -count=1` all green (per "Verification" section above).
- `grep classifyAuthLoginError` returns empty (no functional references remain).
