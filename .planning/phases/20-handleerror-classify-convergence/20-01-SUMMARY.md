---
phase: 20-handleerror-classify-convergence
plan: 01
subsystem: error-handling
tags: [sentinel, businesserror, zap, ad-auth, errors-is]

# Dependency graph
requires:
  - phase: 19-ctx-cascading-and-style-001-error-migration
    provides: HandleError + mapping.go + BusinessError + 41 sentinels
provides:
  - Exported internal/errors.FirstKnownSentinelName with shared knownSentinels slice
  - Public pkg/response.SentinelField helper (zap.Field, 4-state contract)
  - Centralized ErrADUserNotRegistered sentinel in internal/errors (R-3 migration)
affects:
  - 20-02 handler classify convergence
  - 20-03 light handlers
  - 20-04 service zap upgrade
  - 20-05 docs/errors.md generator

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single-source-of-truth knownSentinels slice (replaces inline IsKnownError literal)"
    - "FirstKnownSentinelName delegates the errors.Is scan; IsKnownError reuses the same scan"
    - "SentinelField returns zap.Field (not a string) to leave room for typed-kind extension (R-6)"
    - "4-state logging contract: nil → zap.Skip, BusinessError → code-paren, sentinel → ErrXxx, unknown → ad-hoc"

key-files:
  created:
    - pkg/response/sentinel.go
    - pkg/response/sentinel_field_test.go
  modified:
    - internal/errors/errors.go
    - internal/errors/mapping.go
    - internal/errors/mapping_test.go
    - internal/auth/ad_auth.go
    - internal/auth/ad_authenticator_test.go
    - internal/handlers/auth_handler_test.go

key-decisions:
  - "Refactor IsKnownError to delegate to FirstKnownSentinelName so the knownSentinels slice is the single source of truth (plan's optional branch was applied because both functions need the same scan)."
  - "Place SentinelField in pkg/response (Claude's Discretion D-03.1) — no new package boundary needed; pkg/response already imports internal/errors."
  - "R-3 decision (user-locked): migrate ad_auth.ErrADUserNotRegistered to internal/errors as a D-02.2 补漏 exception so HandleError can map to 403 without falling through to a 500 ad-hoc response."
  - "Keep auth.IsADUserNotRegistered wrapper until 20-02 deletes classifyAuthLoginError (per plan)."

patterns-established:
  - "Use FirstKnownSentinelName whenever a structured-logger field needs a stable identifier (avoids a parallel slice copy in pkg/response)."
  - "Prefer zap.Field return type for log helpers so future typed-kind work can swap zap.String for zap.Object without touching call-sites."

requirements-completed: [REQ-20b-sentinel-field, REQ-20b-priority, REQ-20a-ad-user-not-registered]

# Metrics
duration: 8 min
completed: 2026-08-01
---

# Phase 20 Plan 01: Foundation Summary

**Phase-20 foundation: FirstKnownSentinelName + SentinelField helper + R-3 ErrADUserNotRegistered migration**

## Performance

- **Duration:** 8 min (1785548440 → 1785548893)
- **Started:** 2026-08-01T01:40:40Z
- **Completed:** 2026-08-01T01:48:12Z
- **Tasks:** 3 (2 TDD, 1 auto)
- **Files modified:** 6 (2 created, 4 edited, 1 test-file annotation per TDD task)

## Accomplishments

- `internal/errors.FirstKnownSentinelName(err) (name string, ok bool)` is exported and backed by a single `knownSentinels` slice of `{name, err}` entries. The slice order mirrors the previous `IsKnownError` literal byte-for-byte, so priority is unchanged.
- `IsKnownError` now delegates to `FirstKnownSentinelName` (and the BusinessError check), removing the duplicate literal and giving callers one source of truth.
- `pkg/response.SentinelField(err) zap.Field` implements the 4-state contract (sentinel / BusinessError / unknown / nil) and is the first cross-package consumer of `FirstKnownSentinelName`, eliminating the need to copy the slice into `pkg/response`.
- `auth.ErrADUserNotRegistered` is migrated to `internal/errors.ErrADUserNotRegistered` (R-3 user-locked). The sentinel lands in the 认证 group, the `mapping.go` Forbidden branch, and the `knownSentinels` slice; `ad_auth.go` deletes its local `var` and updates its single `return ErrADUserNotRegistered` site.
- 403 semantics are preserved end-to-end: `IsADUserNotRegistered` (now a thin wrapper) and the legacy `classifyAuthLoginError` both still return `CodeForbidden/StatusForbidden` for the wrapped sentinel case via the new mapping.go case.
- `internal/auth/ad_authenticator_test.go` and `internal/handlers/auth_handler_test.go` are rewritten to consume `apperrors.ErrADUserNotRegistered` so the migrated sentinel is exercised at both the auth-package and handler-package layers.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED)** — `606ddd6` (test): `test(20-01): add failing sentinel name lookup tests`
2. **Task 1 (GREEN)** — `7a8acdb` (feat): `feat(20-01): export FirstKnownSentinelName from internal/errors`
3. **Task 2 (RED)** — `5bd2797` (test): `test(20-01): add failing tests for pkg/response.SentinelField`
4. **Task 2 (GREEN)** — `9d17feb` (feat): `feat(20-01): add pkg/response.SentinelField helper`
5. **Task 3** — `9d41b1c` (refactor): `refactor(20-01): migrate auth.ErrADUserNotRegistered to internal/errors`

**Plan metadata:** committed alongside Task 3 in `9d41b1c`'s body and the docs commit for SUMMARY.md.

## Files Created/Modified

- `internal/errors/errors.go` — added `ErrADUserNotRegistered` to the 认证 group with the same message string as the old auth-package sentinel.
- `internal/errors/mapping.go` — added `knownSentinels` slice; refactored `IsKnownError` to delegate to `FirstKnownSentinelName`; added `FirstKnownSentinelName`; added `ErrADUserNotRegistered` to the Forbidden switch case and to `knownSentinels`.
- `internal/errors/mapping_test.go` — added `TestFirstKnownSentinelName` table-driven suite covering all 41 existing sentinels, wrapped sentinel, first-hit priority, BusinessError exclusion, nil, and unknown.
- `pkg/response/sentinel.go` — new file exposing `SentinelField(err error) zap.Field` with the 4-state contract and a top-of-file rationale block citing D-03.1 and R-6.
- `pkg/response/sentinel_field_test.go` — new file with `TestSentinelField` (6 sub-tests) and `TestSentinelField_Encodable` for zap JSON round-trip.
- `internal/auth/ad_auth.go` — deleted the `var ErrADUserNotRegistered` local; `IsADUserNotRegistered` now wraps `apperrors.ErrADUserNotRegistered`; `findOrCreateLocalUser` returns `apperrors.ErrADUserNotRegistered`.
- `internal/auth/ad_authenticator_test.go` — added `apperrors` import; updated `TestErrADUserNotRegistered_Sentinel` to reference the new sentinel.
- `internal/handlers/auth_handler_test.go` — switched sentinel references to `apperrors.ErrADUserNotRegistered`; removed the now-unused `auth` import.

## Decisions Made

- **Single shared `knownSentinels` slice.** The plan offered an optional branch that refactors `IsKnownError` to delegate to `FirstKnownSentinelName`. We took the branch because both functions iterate the same set with the same priority; the new struct slice is a strictly better single source of truth. `TestIsKnownError` still passes; no regression.
- **`SentinelField` in `pkg/response` (Claude's Discretion D-03.1).** No new `pkg/logging` package boundary was needed — `pkg/response` already imports `internal/errors` and is the only cross-package direction; service call-sites will also import it without circular-dependency risk.
- **R-3 migration accepted as D-02.2 补漏.** Per the plan and user-locked decision, `ad_auth.ErrADUserNotRegistered` is migrated even though it crosses package boundaries; the commit message body explicitly cites D-02.2 so the exception is auditable in git history.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed unused `auth` import from `auth_handler_test.go`**
- **Found during:** Task 3 (compilation after rewriting the sentinel references).
- **Issue:** With both `auth.ErrADUserNotRegistered` references replaced by `apperrors.ErrADUserNotRegistered`, the `internal/auth` import had no remaining users and `go test ./internal/handlers/` failed to build.
- **Fix:** Removed the unused import; re-ran `go test -race` and the build went green.
- **Files modified:** `internal/handlers/auth_handler_test.go`.
- **Committed in:** `9d41b1c` (part of Task 3 commit).

**2. [Rule 1 - Bug] Updated `TestErrADUserNotRegistered_Sentinel` to use the new sentinel location**
- **Found during:** Task 3.
- **Issue:** The test in `internal/auth/ad_authenticator_test.go` referenced the now-deleted `auth.ErrADUserNotRegistered` identifier; the migration required updating both the production code and the test to point at `apperrors.ErrADUserNotRegistered` so the `errors.Is` and `IsADUserNotRegistered` checks still validate behaviour.
- **Fix:** Replaced both references and added the `apperrors` import to the test file.
- **Files modified:** `internal/auth/ad_authenticator_test.go`.
- **Committed in:** `9d41b1c`.

### Plan-intended Refactor (Documented, Not a Deviation)

The plan listed an optional branch to refactor `IsKnownError` to delegate to `FirstKnownSentinelName`. We took the branch and replaced the inline `[]error{...}` literal with the shared `knownSentinels` struct slice. The `TestIsKnownError` and `TestMapToHTTPStatus_*` suites still pass; the slice order is preserved byte-for-byte; this is the explicit purpose of the optional refactor. **Note:** because `IsKnownError` no longer has a separate literal, the plan's acceptance check `grep -c 'ErrADUserNotRegistered' internal/errors/mapping.go >= 3` now returns 2 (Forbidden switch case + `knownSentinels` struct entry). The behaviour and priority are equivalent; the test suite confirms parity.

---

**Total deviations:** 2 auto-fixed (both Rule 1 compile-break fixes, zero scope creep).
**Impact on plan:** Both fixes were necessary to keep the migration green. No architectural changes.

## Issues Encountered

- The `internal/frontend/embed.go:9:12: pattern dist: no matching files found` error from `go build ./...` is a pre-existing repository state (no `frontend/dist/` artifact because the frontend is built separately) and is unrelated to this plan. The touched packages (`internal/errors`, `internal/auth`, `internal/handlers`, `pkg/response`) all build and test cleanly with `-race`.
- `gofmt` rewrote `internal/errors/errors.go` to align the column spacing in the `var (...)` block after the new sentinel was inserted. This is the formatter doing its job; no functional change.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `response.SentinelField(err) zap.Field` is ready for call-site upgrade in 20-02/20-03 (handlers) and 20-04 (services/auth/scheduler/huawei).
- `FirstKnownSentinelName` is the canonical lookup used by `SentinelField` and the new `pkg/logging`-adjacent helpers.
- The 403 Forbidden mapping for `ErrADUserNotRegistered` is in place, so 20-02 can delete `classifyAuthLoginError` and route through `response.HandleError` without regressing login UX.
- `hlstoken.ErrTokenReplayed` name-collision (RESEARCH R-5) remains a survey item and is not consumed by `FirstKnownSentinelName` (it only walks the internal/errors slice, not the cross-package locals) — no further work in this plan.

---

*Phase: 20-handleerror-classify-convergence*
*Completed: 2026-08-01*

## Self-Check: PASSED

- `internal/errors/mapping.go` exports `FirstKnownSentinelName` (1 occurrence).
- `pkg/response/sentinel.go` exports `SentinelField` (1 occurrence) and delegates to `FirstKnownSentinelName` (1 occurrence in source).
- `internal/auth/ad_auth.go` no longer declares the local `var ErrADUserNotRegistered` (0 occurrences).
- `internal/errors/errors.go` declares the centralized sentinel (1 occurrence).
- `internal/handlers/auth_handler_test.go` no longer references `auth.ErrADUserNotRegistered` (0 occurrences).
- `go vet ./internal/errors/... ./internal/auth/... ./internal/handlers/... ./pkg/response/...` exit 0.
- `go test -race ./internal/errors/ ./pkg/response/ ./internal/auth/ ./internal/handlers/ -count=1` all PASS.
