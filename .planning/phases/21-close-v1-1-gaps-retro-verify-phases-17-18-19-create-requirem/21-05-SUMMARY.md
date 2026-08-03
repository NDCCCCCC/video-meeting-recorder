---
phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem
plan: 05
subsystem: handlers
tags: [fix, handlers, auth, sec-008, cr-01, code, handleerror, control-flow-equivalence]

# Dependency graph
requires:
  - phase: 20-handleerror-classify-convergence
    provides: HandleError(c, err) bool canonical pattern (pkg/response/response.go:173-180) + 8/8 handler family convergence + TestLogin_HandleError_ClassifyDrop 10 sub-tests regression net
provides:
  - "auth_handler.go Login error branch (line 57) normalized to canonical `HandleError(c, err); return`"
  - "v1.1-MILESTONE-AUDIT.md WARNING (latent CR-01 reintroduction risk at auth:57) closed"
  - "Login handler matches the 8/8 other handler family pattern from phase 20"
affects: [v1.1-MILESTONE-AUDIT gap closure, milestone-archive, future auth_handler tech_debt cleanup phase]

# Tech tracking
tech-stack:
  added: []  # no new libs/tools — pure control-flow normalization
  patterns:
    - "canonical HandleError pattern at handler error branches: `response.HandleError(c, err); return` (NOT `if response.HandleError(c, err) { return }; return`)"
    - "behavior-equivalence argument via CONTROL FLOW (both branches write-then-return), NOT via `HandleError always returns true` (which is factually wrong — unknown errors return false after writing 500)"
    - "key invariant check before claiming equivalence: confirm `c.Writer.Written()=false` at the call site (no prior HTTP write) so HandleError must reach its GinErrorWithStatus write branch"

key-files:
  created: []
  modified:
    - "internal/handlers/auth_handler.go (line 57-61 5-line pattern → line 57-58 2-line canonical form; line 60 兜底 comment deleted; line 53-56 Phase 20 mapping.go comment retained)"

key-decisions:
  - "行为等价性论据必须基于控制流, NOT 'HandleError always returns true' — pkg/response/response.go:179 实际返回 errors.IsKnownError(err), unknown error 时为 false; 但因原模式两条分支 (known→return / unknown→return) 都先写响应再 return, 与规范模式观察等价"
  - "关键不变式: 在 auth:57 调用点 c.Writer.Written()=false (ShouldBindJSON 失败已在 line 36-39 提前 GinError+return, Warn 日志 line 48-52 不写 HTTP), 故 HandleError 必走 GinErrorWithStatus 写响应分支, 不会 '未写响应就 return'"
  - "D-04.3 注释清理: 删 line 60 「兜底：unknown error（response.HandleError 已写 500）。」(fallback 分支已不存在); 保留 line 53-56 解释 mapping.go 行为的注释 (R-3 ErrADUserNotRegistered→403 / R-4 ErrADConfigError,ErrADUnreachable→503 语义)"
  - "D-04.4 显式不动项: auth_handler.go 其他 3 处 tech_debt (RefreshToken :93 / ChangePassword :182 / LogoutAll 的 raw GinError + err.Error() 泄漏) 不动 — 归后续 phase"
  - "代码改动单独提交 (4959e9c) 与 docs 分离 — 与用户 'debug 改动与 phase 工作分提交' 偏好一致; SUMMARY 走单独的 docs commit (git add -f 因 .planning/ 在 .gitignore)"

patterns-established:
  - "CR-01 防回归闭环: handler 错误分支一律 `response.HandleError(c, err); return`, 禁止 `if response.HandleError(c, err) { return }; <other write>; return` 模式 (后者为 CR-01 双写 bug 的 latent reintroduction vector)"
  - "回归网优先: 既有表驱动测试 (TestLogin_HandleError_ClassifyDrop 10 sub-tests) 直接调用 HandleError 本身而非 caller 控制流, 所以能 pin 写入契约独立于 caller if 形状"

requirements-completed: [P21-R5]

# Metrics
duration: 8 min
completed: 2026-08-03
---

# Phase 21 Plan 05: auth_handler.go:57 Canonical HandleError Pattern Summary

**Collapse Login handler's 5-line `if response.HandleError(c, err) { return }; // 兜底...; return` (line 57-61) into the 2-line canonical `response.HandleError(c, err); return` — closes v1.1-MILESTONE-AUDIT WARNING (latent CR-01 reintroduction risk) and brings Login in line with the 8/8 other handler families using the phase-20 canonical pattern; behavior-equivalence proven via control-flow (not the factually-wrong "always returns true" argument), 10 sub-test regression net green, single-file code commit 4959e9c separate from docs**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-08-03T01:50:51Z
- **Completed:** 2026-08-03T01:58:51Z (approx)
- **Tasks:** 1 (single-task plan)
- **Files modified:** 1 (`internal/handlers/auth_handler.go`)

## Accomplishments

- **`internal/handlers/auth_handler.go:57-61` collapsed to 57-58**: 5-line `if response.HandleError(c, err) { return }; // 兜底：unknown error（response.HandleError 已写 500）。; return` → 2-line `response.HandleError(c, err); return`. Single-file diff: 1 insertion, 4 deletions.
- **v1.1-MILESTONE-AUDIT.md WARNING closed**: §Cross-Phase Integration Findings flagged auth:57 as the only handler family (1/9) NOT using the canonical `HandleError(c, err); return` pattern — a latent CR-01 reintroduction vector if a future contributor appended a GinError after line 59. Now resolved.
- **Behavior-equivalence proven via control-flow (correct argument)**: per CONTEXT D-04.2 (corrected) + 21-RESEARCH §6 + Pitfall 1, the equivalence argument is CONTROL-FLOW based — both branches of the original pattern (known→return inside if / unknown→fall through to bare return) "write response then return", identical to `HandleError; return`. The early CONTEXT D-04.2 claim that "HandleError 始终非 false" is FACTUALLY WRONG (`pkg/response/response.go:179` returns `errors.IsKnownError(err)`, false for unknown errors) — the corrected argument does not rely on the return value.
- **Key invariant verified by reading the function**: at the auth:57 call site, `c.Writer.Written()=false` is guaranteed (ShouldBindJSON failure already returned at line 36-39 with a GinError write, and the `h.logger.Warn` at line 48-52 does not write HTTP), so HandleError always reaches its `GinErrorWithStatus` write branch — never silently returns false without writing.
- **Mapping.go comment retained per D-04.3**: line 53-56 `// Phase 20 (20-02): Login 错误统一走 response.HandleError；mapping.go 通过 errors.Is 链自动识别 sentinel → 对应 401/403/404/503/500 状态码。/ - ErrADUserNotRegistered → 403 (R-3 要求)。/ - ErrADConfigError / ErrADUnreachable → 503 (R-4: 500 → 503)。` left intact (documents the sentinel→HTTP-status mapping behavior — separate concern from the if-vs-canonical shape).
- **Tech_debt scope guard respected per D-04.4**: the other 3 tech_debt sites in the same file (RefreshToken :93 / ChangePassword :182 / LogoutAll raw `GinError + err.Error()` leaks) are NOT touched — verified post-commit `grep -c 'response.GinError' = 10` (RefreshToken / ChangePassword / LogoutAll / TestADConnection / ValidatePassword / GetCurrentUser / Login ShouldBindJSON / Login should-bind etc. all intact). These leaks are deferred to a follow-up phase.
- **Regression net green**: `TestLogin_HandleError_ClassifyDrop` 10 sub-tests (5 error classes × wrapped/unwrapped + R-3/R-4: ErrADUserNotRegistered→403/1003, wrapped→403, ErrADAccountNotFound→404/1004, ErrUserDisabled→403/1003, ErrADConfigError→503/1005, ErrADUnreachable→503/1005, ErrUnauthorized→401/1002, wrapped→401, BusinessError(InvalidInput)→400/1001, unknown ad-hoc→500/1005) — all PASS. The test invokes `response.HandleError(ctx, tt.err)` directly (line 127), NOT the Login handler's if-statement, so it pins the HandleError write contract independent of the caller's control-flow shape.

## Task Commits

Each task was committed atomically:

1. **Task 1: auth_handler.go:57 canonical HandleError pattern (1-line replace + 兜底 comment cleanup + regression net)** - `4959e9c` (fix)

**Plan metadata:** (this commit — `docs(21-05): complete auth:57 fix plan`)

_Note: code fix committed separately (per user "debug 改动与 phase 工作分提交" preference + CONTEXT D-06.1); SUMMARY/STATE/ROADMAP go in a separate docs commit._

## Files Created/Modified

- `internal/handlers/auth_handler.go` - line 57-61 (5 lines) collapsed to line 57-58 (2 lines). Single change: replaced `if response.HandleError(c, err) { return }` + `// 兜底：unknown error（response.HandleError 已写 500）。` + `return` with canonical `response.HandleError(c, err)` + `return`. The 4-line `// Phase 20 (20-02)...` mapping.go comment block above (line 53-56) is retained per D-04.3.

## Decisions Made

- **Equivalence argument source = 21-RESEARCH §6 (corrected), NOT CONTEXT D-04.2**: The plan's `<action>` `<verify>` block explicitly forbids the "HandleError always returns true" formulation (Pitfall 1). Commit body uses the control-flow argument: "both branches of the original write-then-return, identical to canonical". Verified post-commit: `git log -1 --format=%B | grep -qi 'always returns true'` returns no match.
- **Invariant check (call-site `Written()=false`) cited in commit body**: grounds the equivalence proof — without this invariant, the original pattern could in principle differ from canonical (if Written() were already true, HandleError would return false without writing, and the bare `return` would skip response writing). The commit message body explicitly states "ShouldBindJSON failure already returned at line 36-39 with GinError, and the Warn log at line 48-52 does not write HTTP" to defend this invariant.
- **No new tests added**: per plan `<action>` prohibitions, the existing 10-sub-test table already pins the HandleError write contract; Login handler's control-flow equivalence is guaranteed by code review (this commit's body + 21-RESEARCH §6), not by a new test — adding a control-flow test was considered out of scope.
- **Single-file scope**: verified post-commit `git show --stat HEAD` shows only `internal/handlers/auth_handler.go` (1 file, 1 insertion, 4 deletions); no `.planning/` / `internal/services/` / `internal/auth/` / `cmd/` leakage.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all 9 acceptance criteria passed on first attempt:

1. ✅ auth_handler.go line 57-61 (5 lines) replaced with line 57-58 (2 lines) `response.HandleError(c, err)` + `return`
2. ✅ line 60 「兜底」comment deleted
3. ✅ line 53-56 mapping.go behavior comment retained (Phase 20 (20-02) + R-3 要求 + R-4 all present)
4. ✅ `go build ./...` 0 errors
5. ✅ `go test -race ./internal/handlers/...` green (3.4s)
6. ✅ `go test -run TestLogin_HandleError_ClassifyDrop -v ./internal/handlers/...` shows 11 PASS lines (10 sub-tests + 1 parent test)
7. ✅ Other 3 tech_debt sites intact: `grep -c 'response.GinError' = 10` (well above the ≥3 floor)
8. ✅ Commit subject = `fix(handlers/SEC-008): auth_handler.go:57 canonical HandleError pattern`
9. ✅ Commit body contains `Behavior-equivalent` + `IsKnownError`, does NOT contain `always returns true`; single-file scope verified

## Verification Commands Run

```bash
# 1. go build
go build ./...                                # → BUILD OK

# 2. handlers race tests
go test -race ./internal/handlers/...         # → ok 3.443s

# 3. canonical pattern in Login range (line 40-65)
grep -n 'if response.HandleError(c, err)' internal/handlers/auth_handler.go \
  | awk -F: '$1 >= 40 && $1 <= 65 {exit 1}'   # → no match (PASS)
grep -n 'response.HandleError(c, err)' internal/handlers/auth_handler.go \
  | awk -F: '$1 >= 40 && $1 <= 65 {found=1} END {if (!found) exit 1}'
                                               # → line 57 (PASS)

# 4. 兜底 comment removed
! grep -q '兜底' internal/handlers/auth_handler.go   # → PASS

# 5. mapping.go comment retained
grep -q 'Phase 20 (20-02)' internal/handlers/auth_handler.go  # → PASS
grep -q 'R-3 要求'       internal/handlers/auth_handler.go    # → PASS
grep -q 'R-4'            internal/handlers/auth_handler.go    # → PASS

# 6. Other tech_debt intact (≥3 raw GinError sites)
grep -c 'response.GinError' internal/handlers/auth_handler.go # → 10 (≥3 PASS)

# 7. TestLogin_HandleError_ClassifyDrop sub-tests
go test -run 'TestLogin_HandleError_ClassifyDrop' -v ./internal/handlers/... \
  | grep -c -- '--- PASS'                      # → 11 (≥10 PASS)

# 8. Commit-message control-flow argument
git log -1 --format=%B | grep -q 'Behavior-equivalent'  # → PASS
git log -1 --format=%B | grep -q 'IsKnownError'         # → PASS
! git log -1 --format=%B | grep -qi 'always returns true' # → PASS (no false claim)

# 9. Single-file scope
git show --stat HEAD --format= \
  | grep -c 'internal/handlers/auth_handler.go'  # → 1 (exactly)
```

## User Setup Required

None - no external service configuration required. Pure Go code normalization; no new deps, env vars, or runtime config.

## Self-Check: PASSED

- [x] FOUND: `.planning/phases/21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem/21-05-SUMMARY.md` (created)
- [x] FOUND: `internal/handlers/auth_handler.go` (modified — line 57-58 canonical pattern in place)
- [x] FOUND: commit `4959e9c` (code fix — single-file scope, control-flow argument in body, no "always returns true" false claim)
- [x] FOUND: commit `2f11b39` (plan metadata — SUMMARY + STATE + ROADMAP)
- [x] All 9 acceptance_criteria from PLAN.md `<acceptance_criteria>` block PASS (verified in Issues Encountered section above)

## Next Phase Readiness

- **v1.1-MILESTONE-AUDIT.md auth:57 WARNING closed** — Login handler now matches the 8/8 other handler families using the canonical `HandleError(c, err); return` pattern; the latent CR-01 reintroduction vector is eliminated (future contributors appending a GinError at line 59 can no longer double-write).
- **Phase 21 plan 21-05 (this plan) done** — Wave 1 of phase 21 now 4/4 complete (21-01/02/03/05). Only 21-04 (REQUIREMENTS.md, Wave 2, depends_on [21-01,21-02,21-03]) remains.
- **Phase 20→21 contract intact**: the phase-20 HandleError convergence that established the canonical pattern across 8/9 handler families is now 9/9 — phase 21 closes the last holdout (Login was flagged as a deviation in 20-VERIFICATION.md).
- **Milestone re-audit ready**: with 21-01/02/03 (retro-verify VERIFICATION.md for phase 17/18/19) + 21-05 (this WARNING fix) done, the only remaining v1.1-MILESTONE-AUDIT gap is REQUIREMENTS.md (21-04). Once 21-04 lands, re-running `/gsd:audit-milestone v1.1` should move from `gaps_found` → `passed`.

---

*Phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem*
*Completed: 2026-08-03*
