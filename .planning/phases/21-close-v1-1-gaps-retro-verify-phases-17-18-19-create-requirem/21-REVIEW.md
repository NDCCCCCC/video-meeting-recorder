---
phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem
reviewed: 2026-08-03T00:00:00Z
depth: standard
files_reviewed: 1
files_reviewed_list:
  - internal/handlers/auth_handler.go
findings:
  critical: 0
  warning: 0
  info: 0
  total: 0
status: clean
---

# Phase 21: Code Review Report

**Reviewed:** 2026-08-03T00:00:00Z
**Depth:** standard
**Files Reviewed:** 1
**Status:** clean

## Summary

Reviewed the single source change in Phase 21: a 1-line control-flow refactor at
`internal/handlers/auth_handler.go:57` that collapses the conditional
`if response.HandleError(c, err) { return }; // 兜底...; return` pattern into the
canonical unconditional `response.HandleError(c, err); return` form.

**The change is behavior-preserving and correct.** No critical issues, warnings, or
info findings against the reviewed file. Independent verification of the four
claimed invariants (HandleError return semantics, call-site write-state, status
mapping preservation, isolation) all hold. Build, `go vet`, and the full
`TestLogin_HandleError_ClassifyDrop` regression net (10 subtests) pass.

### Verification details

**1. HandleError return semantics (`pkg/response/response.go:173-180`)**

```go
func HandleError(c *gin.Context, err error) bool {
    if err == nil || c.Writer.Written() {
        return false
    }
    httpStatus, respCode, message := errors.MapToHTTPStatus(err)
    GinErrorWithStatus(c, httpStatus, respCode, message)
    return errors.IsKnownError(err)
}
```

Confirmed: when `err != nil` and `!c.Writer.Written()`, HandleError **always**
writes the HTTP response via `GinErrorWithStatus` — for both known and unknown
errors. The `bool` return only signals known-vs-unknown classification, NOT
whether a response was written. Therefore the old pattern's two branches:

- known error → HandleError writes response, returns true → `if true { return }`
- unknown error → HandleError writes 500, returns false → falls through to `return`

both did "write response then return", which is exactly what the new unconditional
`HandleError(c, err); return` does. Observably equivalent (same bytes on the wire,
same exit).

**2. `c.Writer.Written()=false` invariant at line 57**

Traced the full `Login` function path above line 57:

| Line | Statement | Writes HTTP? |
|------|-----------|--------------|
| 36-39 | `c.ShouldBindJSON(&req)` failure → `GinError` + `return` | Early-exit; if we pass line 39, writer is unwritten |
| 42 | `c.ClientIP()` | No — reads request headers |
| 43 | `c.GetHeader("User-Agent")` | No — reads header |
| 46 | `h.authService.Login(c.Request.Context(), ...)` | No — receives `context.Context`, not `*gin.Context`; no writer access |
| 48-52 | `h.logger.Warn(...)` | No — zap logger, not HTTP |

Invariant holds: at line 57 the writer is guaranteed unwritten, so HandleError
will execute its write branch exactly once. No double-write, no missed response.

**3. HTTP status mapping unchanged for all 5 tested error classes**

Cross-referenced `internal/errors/mapping.go` for each class in the regression net:

| Error | mapping.go line | HTTP | respCode |
|-------|-----------------|------|----------|
| ErrADUserNotRegistered | 67 (Forbidden case) | 403 | 1003 |
| ErrADAccountNotFound | 50 (NotFound case) | 404 | 1004 |
| ErrUserDisabled | 66 (Forbidden case) | 403 | 1003 |
| ErrADConfigError | 86 (ServiceUnavailable case) | 503 | 1005 |
| ErrADUnreachable | 87 (ServiceUnavailable case) | 503 | 1005 |
| ErrUnauthorized (and wrapped) | 55 (Unauthorized case) | 401 | 1002 |
| BusinessError(CodeInvalidInput) | `mapBusinessError` line 112 | 400 | 1001 |
| unknown ad-hoc | default line 95 | 500 | 1005 |

The refactor touches only control-flow wrapping, not the mapping. All 10
subtests of `TestLogin_HandleError_ClassifyDrop` pass (verified by `go test -run`).

**4. Change isolation**

The diff is confined to `Login` lines 53-58. No other function in the file
references or depends on the internal structure of this error block. The retained
4-line `// Phase 20 (20-02)` comment accurately describes the unchanged mapping
behavior (it documents Phase 20's mapping decision, which Phase 21 preserves).

### Out-of-scope (explicitly deferred tech_debt — not new findings)

Per phase scope, the following 3 sites in `auth_handler.go` still use the legacy
`GinError(c, code, err.Error())` pattern instead of `response.HandleError` and
are explicitly deferred out of Phase 21:

- `RefreshToken` line 90 (`CodeInvalidToken` — flat 401, no sentinel mapping)
- `ChangePassword` line 179 (`CodeInvalidPassword` — flat response)
- `LogoutAll` line 142 (`CodeInternalError` — flat 500)

These are pre-existing tech_debt items, not regressions introduced by Phase 21.

### Observation (not a finding — test file out of review scope)

The regression test `internal/handlers/auth_handler_test.go` lines 124-126
contains a comment referencing the OLD calling pattern
(`if response.HandleError(c, err) { return }`), which is now stale after this
refactor. This is not flagged as a finding because (a) the test file is not in
the review scope, and (b) the stale comment does not affect test reliability —
the test correctly exercises `HandleError`'s contract directly. Future cleanup
of that comment would be a minor doc improvement.

---

_Reviewed: 2026-08-03T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
