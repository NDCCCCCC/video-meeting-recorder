---
phase: 20-handleerror-classify-convergence
verified: 2026-08-01T12:00:00Z
status: passed
score: 10/10 must-haves verified
overrides_applied: 0
overrides: []
gaps: []
gap_closure:
  - finding: "docs/errors.md stale (4 call-site count drifts: ErrNotFound 29→31, BusinessError(INTERNAL_ERROR) 64→66, BusinessError(INVALID_INPUT) 40→41, BusinessError(NOT_FOUND) 47→49)"
    resolved_by: "fix(20-VERIFICATION) regenerate docs/errors.md to sync with latest call-site counts (commit 7a72675)"
    verified: "go generate ./internal/errors/... && git diff --quiet docs/errors.md → SYNC_OK (idempotent); CI sync-check will pass"
deferred: []
human_verification: []
---

# Phase 20: HandleError Classify Convergence - Verification Report

**Phase Goal:** handler ad-hoc classify 全量清理（9 文件 27 处 + classifyAuthLoginError 删除）+ zap logger errors.Is 集成（response.SentinelField 接入 + 160+ zap.Error 站点升级）+ docs/errors.md 自动生成（cmd/error-doc-gen + //go:generate + CI 同步检查）

**Verified:** 2026-08-01T12:00:00Z
**Status:** passed (10/10 must-haves verified; 1 sync-check gap closed by commit 7a72675)
**Verifier:** Claude (gsd-verifier) — adversarial stance, goal-backward methodology

## Goal Achievement

### Observable Truths

| #   | Truth                                                                                                  | Status     | Evidence                                                                                                                                       |
| --- | ------------------------------------------------------------------------------------------------------ | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| 1   | All 9+ handler files converge to `if response.HandleError(c, err) { return }` pattern                   | VERIFIED   | 102 `response.HandleError(c, err)` sites across 12 handler files; `strings.Contains(errMsg,...)` scatter = 0; `classifyAuthLoginError` deleted |
| 2   | `classifyAuthLoginError` formal function deleted; Login routes through HandleError                      | VERIFIED   | `grep -c 'func classifyAuthLoginError' internal/handlers/auth_handler.go` = 0; `TestLogin_HandleError_ClassifyDrop` 10/10 sub-tests pass        |
| 3   | `response.SentinelField(err)` implements 4-state contract (sentinel/BusinessError/unknown/nil)         | VERIFIED   | `pkg/response/sentinel.go` exported; `TestSentinelField` 7/7 sub-tests pass (incl. priority-mirrors-IsKnownError-slice)                          |
| 4   | SentinelField priority mirrors `IsKnownError` slice order — first errors.Is hit wins                   | VERIFIED   | `internal/errors.FirstKnownSentinelName` iterates shared `knownSentinels` slice in original order; `IsKnownError` delegates to it                |
| 5   | `auth.ErrADUserNotRegistered` migrated to `internal/errors.ErrADUserNotRegistered` (R-3, 403 preserved) | VERIFIED   | `grep -c 'var ErrADUserNotRegistered' internal/auth/ad_auth.go` = 0; 403 mapping in `mapping.go:67`; `IsADUserNotRegistered` wrapper preserved     |
| 6   | Every `zap.Error(err)` call-site in scope upgraded to `zap.Error(err), response.SentinelField(err)`     | VERIFIED   | 12 handler files have SentinelField; 19 service/auth/scheduler/huawei files upgraded; middleware intentionally NOT touched (D-03.7 lock)       |
| 7   | `cmd/error-doc-gen` emits sentinel table (42 rows) + BusinessError table (10 rows) + ad-hoc audit      | VERIFIED   | Generator works; 6 tests pass (`TestGenerate_*`); `docs/errors.md` has 42 sentinel rows + 10 BusinessError rows + audit footer                   |
| 8   | `//go:generate go run ./cmd/error-doc-gen` directive at top of `internal/errors/errors.go`             | VERIFIED   | Line 3: `//go:generate go run ../../cmd/error-doc-gen -errors-file errors.go -mapping-file mapping.go -output ../../docs/errors.md -repo-root ../..` |
| 9   | `.github/workflows/ci.yml` contains step that runs `go generate` + `git diff --quiet docs/errors.md`   | VERIFIED   | Lines 44-51: "Verify errors doc sync" step with the required `go generate` + `git diff --quiet` pattern                                        |
| 10  | Re-running generator produces byte-identical output (deterministic)                                    | FAILED     | `TestGenerate_Deterministic` PASSES, BUT `go generate ./internal/errors/... && git diff --quiet docs/errors.md` returns non-zero — file is STALE |

**Score:** 9/10 truths verified (1 functional gap)

### Required Artifacts

| Artifact                                                                                                | Expected                                                | Status      | Details                                                                                                                                                  |
| ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pkg/response/sentinel.go`                                                                              | SentinelField public helper                             | VERIFIED    | Exported; imports zap + apperrors; returns `zap.Field` (R-6 future-proofing)                                                                            |
| `internal/errors/mapping.go`                                                                            | FirstKnownSentinelName + ErrADUserNotRegistered         | VERIFIED    | Single `knownSentinels` struct slice; `IsKnownError` delegates; 41 sentinels + ErrADUserNotRegistered                                                   |
| `internal/handlers/auth_handler.go`                                                                     | classifyAuthLoginError-free Login                       | VERIFIED    | Function + doc block deleted; Login routes through `response.HandleError`                                                                                 |
| `internal/handlers/*_handleerror_test.go` (11 files)                                                     | Table-driven 4-error-class tests                        | VERIFIED    | All 11 files exist; 44+ sub-tests pass under -race                                                                                                       |
| `internal/handlers/cr01_pattern_test.go`                                                                | CR-01 regression test                                   | VERIFIED    | 3 tests pass (positive control + negative control + Written() guard contract)                                                                            |
| `cmd/error-doc-gen/main.go`                                                                             | Generator binary                                        | VERIFIED    | Stdlib-only; resolveRelative() helper; captures default branch status (WR-01); source-context binding exclusion (WR-02)                             |
| `cmd/error-doc-gen/main_test.go`                                                                        | 6-case generator verification                           | VERIFIED    | All 6 tests pass (incl. TestGenerate_SentinelTableStatuses, TestGrepCount_ExcludesShouldBindJSONBlock, TestGrepCount_NestedBindingBlock)               |
| `docs/errors.md`                                                                                        | Auto-generated sentinel catalog + ad-hoc audit          | STALE       | Structure correct (42 sentinels + 10 BusinessError + footer); call-site counts drifted (ErrNotFound 29→31, INTERNAL_ERROR 64→66, etc.)                  |
| `internal/errors/errors.go`                                                                             | `//go:generate` directive + ErrADUserNotRegistered      | VERIFIED    | Line 3 directive; 42 sentinels including ErrADUserNotRegistered in 认证 group                                                                            |
| `.github/workflows/ci.yml`                                                                              | "Verify errors doc sync" step                           | VERIFIED    | Present at lines 44-51; fails fast before `Test` step                                                                                                    |
| `internal/middleware/error_mapper.go`                                                                   | UNCHANGED (D-03.7 lock)                                 | VERIFIED    | `grep -c 'response.SentinelField' internal/middleware/error_mapper.go` = 0; 2 zap.Error sites remain without SentinelField per locked user decision  |

### Key Link Verification

| From                                | To                                       | Via                                        | Status      | Details                                                                                                                                                  |
| ----------------------------------- | ---------------------------------------- | ------------------------------------------ | ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| handler files → pkg/response        | HandleError                              | `if response.HandleError(c, err) { return }` (post-CR-01 fix uses `HandleError + return`, NOT `if HandleError { return }`) | VERIFIED    | 102 sites use the FIXED pattern (no `if`-gated returns). Per-handler grep shows pattern is consistent across admin/file/ppt/video_file/etc.              |
| service files → pkg/response        | SentinelField                            | `zap.Error(err), response.SentinelField(err)` | VERIFIED    | 19 files upgraded; per-file grep matches plan estimates                                                                                                  |
| internal/errors/mapping.go → pkg/response | FirstKnownSentinelName (priority source) | shared `knownSentinels` struct slice       | VERIFIED    | Single source of truth for both `IsKnownError` and `FirstKnownSentinelName`                                                                              |
| cmd/error-doc-gen → internal/errors | sentinel parsing                         | regex over `var (...)` block               | VERIFIED    | Catches all 42 sentinels; multi-line case clauses captured via `(?m)^\s*(?:case\s+)?errors.Is(...)`                                                       |
| cmd/error-doc-gen → internal/errors | status mapping                           | regex over `MapToHTTPStatus` switch        | VERIFIED    | Captures `__default__` branch; multi-Code cases split on `,` (WR-01 fix verified by TestGenerate_SentinelTableStatuses)                                |
| .github/workflows/ci.yml → docs/errors.md | sync enforcement                        | `go generate && git diff --quiet`           | **GAP**     | Step is correctly wired; FAILS on current state because file is stale                                                                                    |

### Data-Flow Trace (Level 4)

| Artifact                                   | Data Variable             | Source                                                                 | Produces Real Data | Status      |
| ------------------------------------------ | ------------------------- | ---------------------------------------------------------------------- | ------------------ | ----------- |
| `pkg/response.SentinelField`               | sentinel_type field       | `internal/errors.FirstKnownSentinelName` (errors.Is chain)             | Yes                | VERIFIED    |
| `cmd/error-doc-gen.Generate`               | call-site count           | `filepath.WalkDir` + `regexp` over `internal/**/*.go`                  | Yes                | VERIFIED    |
| `cmd/error-doc-gen.Generate`               | ad-hoc audit count        | `grepCountInSource` with brace-depth tracking (WR-02 fix)              | Yes                | VERIFIED    |
| `docs/errors.md` (regenerated by `go generate`) | call-site count cells   | in-process grep                                                        | Yes                | VERIFIED (functionally); file currently shows stale values |

### Behavioral Spot-Checks

| Behavior                                                                                                       | Command                                                                                                       | Result                                                                | Status    |
| -------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- | --------- |
| SentinelField returns "ad-hoc" for unknown error                                                              | `go test -run "TestSentinelField/unknown_error" -v ./pkg/response/`                                            | PASS                                                                  | VERIFIED  |
| CR-01 fix produces exactly 1 JSON object for unknown errors                                                    | `go test -run "TestCR01_HandleErrorThenReturn_WritesOneObject" -v ./internal/handlers/`                         | PASS (4/4 sub-tests)                                                  | VERIFIED  |
| CR-01 buggy pattern produces 2 JSON objects (negative control)                                                  | `go test -run "TestCR01_PreFixPattern_ProducesTwoBodies" -v ./internal/handlers/`                               | PASS                                                                  | VERIFIED  |
| Generator output is deterministic                                                                              | `go test -run "TestGenerate_Deterministic" -v ./cmd/error-doc-gen/`                                            | PASS                                                                  | VERIFIED  |
| Login 503 semantics preserved (R-4)                                                                             | `go test -run "TestLogin_HandleError_ClassifyDrop/ErrADConfigError" -v ./internal/handlers/`                    | PASS (StatusServiceUnavailable)                                       | VERIFIED  |
| AD 403 semantics preserved (R-3)                                                                               | `go test -run "TestLogin_HandleError_ClassifyDrop/ErrADUserNotRegistered" -v ./internal/handlers/`              | PASS (StatusForbidden)                                                | VERIFIED  |
| All 12 handler-level handleerror tests pass                                                                    | `go test -run "TestPPTHandler_HandleError|TestVideoRecordingTaskHandler_HandleError|...|TestAPIKeyHandler_HandleError" -v ./internal/handlers/` | PASS (44+ sub-tests)                                                  | VERIFIED  |
| `go generate` then `git diff --quiet docs/errors.md` exits 0                                                    | `go generate ./internal/errors/... && git diff --quiet docs/errors.md && echo SYNC_OK`                        | FAIL — diff in 4 rows (ErrNotFound 29→31, INTERNAL_ERROR 64→66, INVALID_INPUT 40→41, NOT_FOUND 47→49) | **FAILED** |
| Full project race test (services/auth/scheduler/huawei/handlers/response/errors)                                | `go test -count=1 -race ./internal/services/... ./internal/auth/... ./internal/scheduler/... ./internal/huawei/... ./internal/handlers/... ./internal/errors/... ./pkg/response/...` | All packages PASS                                                     | VERIFIED  |
| No Makefile created (R-2 user-locked)                                                                          | `test ! -f Makefile && test ! -f makefile && echo NO_MAKEFILE_OK`                                              | NO_MAKEFILE_OK                                                        | VERIFIED  |

### Requirements Coverage

The ROADMAP.md states Phase 20 requirements as D-22:

| Requirement                                                                            | Source Plan      | Description                                                            | Status      | Evidence                                                                                       |
| -------------------------------------------------------------------------------------- | ---------------- | ---------------------------------------------------------------------- | ----------- | ---------------------------------------------------------------------------------------------- |
| D-22 (1) handler ad-hoc classify 全量清理 (9 文件 27 处 + classifyAuthLoginError 删除)    | 20-02, 20-03     | Replace inline `err.Error()` GinError / string-match with HandleError    | VERIFIED    | 102 HandleError sites; 0 string-match scatter; `classifyAuthLoginError` deleted                  |
| D-22 (2) zap logger errors.Is 集成 (response.SentinelField + 160+ zap.Error 站点升级)     | 20-01, 20-02, 20-03, 20-04 | `response.SentinelField` + upgrade ~160 service + handler sites | VERIFIED    | `pkg/response/sentinel.go` exported; 12 handler + 19 service files upgraded                      |
| D-22 (3) 自动生成 sentinel 文档 (cmd/error-doc-gen + //go:generate + CI 同步检查)       | 20-05            | Generator + directive + CI step                                         | PARTIAL     | Generator works + directive present + CI step present; BUT docs/errors.md is STALE (CI will fail) |
| D-22 (4) typed error kind 字段 (Sentinel vs BusinessError vs ad-hoc)                     | —                | Deferred to next phase per D-01.1                                       | DEFERRED    | Explicitly out-of-scope per CONTEXT.md D-01.1; SentinelField returns `zap.Field` (not string) to leave room for future typed-kind work (R-6) |

### Code Review Fixes Verification

All 5 review findings from `20-REVIEW.md` were addressed in commits `205462a` (CR-01), `fc0b656` (WR-03), `2d7ea72` (WR-01+WR-02), `8aac409` (WR-04):

| ID    | Severity | Status      | Evidence                                                                                                                                                  |
| ----- | -------- | ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| CR-01 | BLOCKER  | FIXED       | `internal/handlers/cr01_pattern_test.go` (3 tests pass); 8 handler families use `HandleError + return` pattern (no `if HandleError { return }` remnant)  |
| WR-01 | WARNING  | FIXED       | `cmd/error-doc-gen/main.go` captures `__default__` branch; `TestGenerate_SentinelTableStatuses` asserts every row has HTTP 100..599                          |
| WR-02 | WARNING  | FIXED       | `grepCountInSource` uses brace-depth tracking; `TestGrepCount_ExcludesShouldBindJSONBlock` + `TestGrepCount_NestedBindingBlock` pass                       |
| WR-03 | WARNING  | FIXED       | SentinelField added to remaining aliases (syncErr, stopErr, cloudErr, checkErr, rollbackErr, parseErr, firstErr)                                          |
| WR-04 | WARNING  | FIXED (caveat: full interface-based mocking deferred) | `cr01_pattern_test.go` provides pattern-test contract; full per-handler service-mock coverage deferred to a follow-up phase |

### Anti-Patterns Found

| File                                       | Line(s)      | Pattern                                      | Severity | Impact                                                                                                                                  |
| ------------------------------------------ | ------------ | -------------------------------------------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/handlers/auth_handler.go`        | 93, 182      | `response.GinError(c, ..., err.Error())`     | INFO     | Token-refresh specific responses (CodeInvalidToken/CodeInvalidPassword). Documented in 20-03 SUMMARY as "ownership/refresh-token paths, not service-error scatter" — explicitly out of 20-03 scope per plan. |
| `internal/handlers/ppt_handler.go`         | 400, 438, 582, 645, 725, 805, 877, 955    | `response.GinError(c, CodeForbidden, err.Error())` | INFO | Ownership-verification paths (`verifyPPTOwnership`). Same justification as above.                                              |
| `internal/handlers/*_handler.go` (multiple)| various      | `response.GinError(c, CodeInvalidRequest, "请求参数错误: "+err.Error())` for `c.ShouldBindJSON` failures | INFO | Canonical Gin pattern (per CONTEXT D-02.4 + 20-03 SUMMARY §1.3.5); explicitly preserved, NOT classify-scatter.                       |

No anti-patterns at BLOCKER severity. The 11 remaining `err.Error()` sites in non-ShouldBindJSON handlers are documented Phase 20 deferred items.

### Human Verification Required

None required — all behaviors are programmatically verified via tests and grep checks.

## Gaps Summary

**One functional gap found:** `docs/errors.md` call-site counts have drifted from the current code state. The CI sync-check step `.github/workflows/ci.yml` "Verify errors doc sync" will FAIL on the next CI run because `go generate ./internal/errors/...` will rewrite the file with updated counts and `git diff --quiet docs/errors.md` will return non-zero.

The generator itself works correctly (6 tests pass including `TestGenerate_Deterministic`), the `//go:generate` directive is in place, and the CI step is properly wired. The gap is purely a stale-file artifact: the last regeneration was in commit `2d7ea72` (WR-01/02 fix); since then, `8aac409` (CR-01 pattern test) added `cr01_pattern_test.go` which references `ErrNotFound` 2 additional times and `apperrors.NewBusinessError`/`apperrors.CodeInvalidInput` 1 additional time each, shifting the in-process grep counts.

**Fix:** Run `go generate ./internal/errors/...` locally to regenerate `docs/errors.md`, then commit the updated file. The Phase 20 implementation is otherwise complete; this is a one-line sync fix.

---

_Verified: 2026-08-01T12:00:00Z_
_Verifier: Claude (gsd-verifier)_