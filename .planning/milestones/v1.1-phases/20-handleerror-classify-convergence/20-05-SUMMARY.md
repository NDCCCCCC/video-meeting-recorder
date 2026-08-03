---
phase: 20-handleerror-classify-convergence
plan: 05
subsystem: error-handling-documentation
tags: [docs, generator, ci, go-generate, error-catalog]
dependency_graph:
  requires: [20-01, 20-02, 20-03]
  provides: [REQ-20c-generator, REQ-20c-doc-sync, REQ-20c-call-site-count]
  affects: [internal/errors, .github/workflows/ci.yml, docs/errors.md]
tech-stack:
  added:
    - "cmd/error-doc-gen — standalone Go binary (stdlib only)"
  patterns:
    - "Text-scan generator (regex) over Go source — no AST/reflection"
    - "CI gate via go generate + git diff --quiet"
key-files:
  created:
    - cmd/error-doc-gen/main.go
    - cmd/error-doc-gen/main_test.go
    - docs/errors.md
  modified:
    - internal/errors/errors.go
    - .github/workflows/ci.yml
decisions: []
metrics:
  duration: "TBD"
  completed_date: "2026-08-01"
  tasks_completed: 2
  files_created: 3
  files_modified: 2
---

# Phase 20 Plan 05: Error Documentation Generator + CI Sync Check Summary

Builds the `cmd/error-doc-gen` standalone Go binary that scans
`internal/errors/errors.go` + `internal/errors/mapping.go` and emits
`docs/errors.md` with sentinel / BusinessError tables + ad-hoc audit
footer, wired via `//go:generate` and enforced by a CI sync-check step
(per R-2 user-locked decision: **no Makefile**).

## What Was Built

### Task 1 — Generator binary + tests + first generated `docs/errors.md`

**Commit:** `d13a508` — `feat(20-05): error-doc-gen binary + generated docs/errors.md`

- `cmd/error-doc-gen/main.go` — single-file Go binary (stdlib only):
  - Parses `internal/errors/errors.go` regex-style for the `var (...)`
    sentinel block (`ErrXxx = errors.New("...")`) and the `const (...)`
    Code block (`CodeXxx = "..."`).
  - Parses `internal/errors/mapping.go` regex-style for the
    `MapToHTTPStatus` switch and the `mapBusinessError` switch. Handles
    multi-line `case` clauses (continuation `errors.Is(...)` lines
    without `case` prefix) and multi-Code cases
    (`case CodeAlreadyExists, CodeTaskInProgress:`). Captures the
    `default:` branch in `mapBusinessError` so unmatched Codes (e.g.
    `CodeForeignKeyConstraint`) get the correct fallback status
    (500).
  - Call-site counting via in-process `filepath.WalkDir` + per-line
    regex match (`\b<Name>\b` word boundary). No external `grep`
    dependency — portable to Windows without Git Bash / Cygwin.
  - Ad-hoc audit footer counts inline `err.Error()` and `errMsg :=`
    patterns in `internal/handlers/*.go` (excluding `_test.go` and
    `ShouldBindJSON`).
  - Deterministic output: sorted alphabetically by name, no
    timestamps, fixed column order. Running twice produces
    byte-identical files.

- `cmd/error-doc-gen/main_test.go` — 5 test cases (4 required by D-05.3
  + 1 bonus for audit footer):
  - `TestGenerate_SentinelTableComplete` — >= 42 sentinel rows with
    "Sentinel" kind column.
  - `TestGenerate_BusinessErrorTableComplete` — exactly 10
    `BusinessError(code=...)` rows.
  - `TestGenerate_MissingFilepath` — non-existent path returns error
    (no panic).
  - `TestGenerate_Deterministic` — two runs produce byte-identical
    output (`bytes.Equal`).
  - `TestGenerate_AuditFooter` — output contains "ad-hoc" audit
    footer.
  - All 5 PASS.

- `docs/errors.md` — first generated output (3,642 bytes, 84 lines):
  - 42 sentinel rows (sorted alphabetically) with HTTP status from
    `MapToHTTPStatus` and call-site counts from internal/.
  - 10 BusinessError rows (10 Code constants) with HTTP status from
    `mapBusinessError`.
  - Ad-hoc audit footer reporting 31 inline classify branches
    remaining (target: 0 — see Deviations for context).

### Task 2 — `//go:generate` directive + CI sync-check step (no Makefile per R-2)

**Commit:** `5df3b39` — `feat(20-05): wire //go:generate + CI sync-check step (no Makefile)`

- `internal/errors/errors.go` — package-level `//go:generate` directive
  added at the top, just below the package doc. Directive uses
  package-dir-relative paths (`../../cmd/error-doc-gen`,
  `errors.go`, `mapping.go`, `../../docs/errors.md`) since `go generate`
  sets the working directory to the package containing the directive.

- `cmd/error-doc-gen/main.go` — added `resolveRelative()` helper. When
  invoked via `go generate ./internal/errors/...`, the cwd is
  `internal/errors/`. The resolver walks up the directory tree looking
  for `go.mod` (module root), then re-anchors the relative paths from
  there. Makes `go generate` idempotent — running twice yields zero
  diff.

- `.github/workflows/ci.yml` — new `Verify errors doc sync` step
  inserted in the `backend` job **after `Build` and before `Test`**
  (per plan recommendation: fails fast before the slow test suite
  runs). The step runs:

  ```yaml
  - name: Verify errors doc sync
    run: |
      go generate ./internal/errors/...
      if ! git diff --quiet docs/errors.md; then
        echo "::error::docs/errors.md is out of sync with internal/errors — run 'go generate ./internal/errors/...' and commit the result"
        git diff docs/errors.md
        exit 1
      fi
  ```

  No Makefile target created — R-2 user-locked decision honored
  (project has no Make history; go:generate + CI step is the lightest
  integration).

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | exit 0 |
| `go vet ./...` | exit 0 |
| `go test ./cmd/error-doc-gen/... -count=1 -v` | 5 subtests PASS |
| `go test -race ./cmd/error-doc-gen/... ./internal/errors/... -count=1` | PASS |
| `go generate ./internal/errors/... && git diff --quiet docs/errors.md && echo SYNC_OK` | SYNC_OK |
| `grep -c '//go:generate go run' internal/errors/errors.go` | 1 |
| `grep -c 'go generate' .github/workflows/ci.yml` | 2 (step name + run body) |
| `grep -c 'git diff --quiet docs/errors.md' .github/workflows/ci.yml` | 1 |
| `grep -ic 'makefile' .github/workflows/ci.yml` | 0 (no Makefile reference) |
| `test ! -f Makefile && test ! -f makefile` | pass (NO_MAKEFILE_OK) |
| `go run ./cmd/error-doc-gen -output /tmp/x && diff -q docs/errors.md /tmp/x` | empty (deterministic) |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed case-clause parser that only matched `case` head lines**

- **Found during:** Task 1, first generator run.
- **Issue:** `MapToHTTPStatus` switch uses multi-line `case` clauses like
  ```
  case errors.Is(err, ErrNotFound),
      errors.Is(err, ErrTaskNotFound),
      ...:
      return http.StatusNotFound, ...
  ```
  The original regex only matched the `case errors.Is(err, ErrXxx)` head
  line, so continuation lines were silently dropped — causing 30+
  sentinels to show HTTP status 0.
- **Fix:** Loosened regex to `(?m)^\s*(?:case\s+)?errors\.Is\(err,\s*(Err\w+)\)`,
  so continuation lines without `case` prefix are also captured.
- **Files modified:** `cmd/error-doc-gen/main.go`.
- **Commit:** `d13a508` (caught during pre-commit verification).

**2. [Rule 1 - Bug] Fixed per-case status tracking that read `return` before `case`**

- **Found during:** Task 1, first generator run.
- **Issue:** Linear scan set `current = status` from `return http.StatusXxx`
  and then attributed `current` to the next `case`. But the `return`
  appears AFTER the `case` in source order — so `current` was always
  trailing by one case. Result: many cases showed the wrong status or 0.
- **Fix:** Replaced single `current` accumulator with a
  `caseBranch{sentinels, codes}` struct that queues names from each
  `case` line and flushes them when the next `return http.StatusXxx`
  line is seen.
- **Files modified:** `cmd/error-doc-gen/main.go`.
- **Commit:** `d13a508`.

**3. [Rule 1 - Bug] Fixed Code-case regex that only captured first name in `case A, B:`**

- **Found during:** Task 1, first BusinessError table check.
- **Issue:** `case CodeAlreadyExists, CodeTaskInProgress:` only captured
  `CodeAlreadyExists`, not `CodeTaskInProgress`. Result: 5 of 10
  BusinessError rows showed status 0.
- **Fix:** Changed regex to
  `(?m)^\s*case\s+((?:Code\w+\s*,\s*)*Code\w+)\s*:` and split on `,`
  when consuming the match.
- **Files modified:** `cmd/error-doc-gen/main.go`.
- **Commit:** `d13a508`.

**4. [Rule 1 - Bug] Fixed Code codes falling through to `default:` branch showing status 0**

- **Found during:** Task 1, after multi-code fix.
- **Issue:** `CodeForeignKeyConstraint` and other Codes not explicitly
  listed in `mapBusinessError` should fall to the `default:` branch
  (returning 500), but the generator was emitting 0 because the
  `default:` clause wasn't parsed.
- **Fix:** Added `default:` handling — captures the `return` status
  into a sentinel `__default__` key in the map; `Generate()` then
  applies it to any Code constant not explicitly listed and removes
  the sentinel.
- **Files modified:** `cmd/error-doc-gen/main.go`.
- **Commit:** `d13a508`.

**5. [Rule 1 - Bug] Fixed generator cwd mismatch with `go generate` invocation**

- **Found during:** Task 2, first `go generate ./internal/errors/...` test.
- **Issue:** `go generate` sets the working directory to the package
  directory (`internal/errors/`). The directive's relative paths
  (`internal/errors/errors.go`, etc.) resolved to
  `internal/errors/internal/errors/errors.go` — directory not found.
- **Fix:** Added `resolveRelative()` helper that, when given a relative
  path, tries it as-is from cwd first, then walks up to `go.mod` and
  re-anchors. Updated the directive to use package-dir-relative paths
  (`../../cmd/error-doc-gen`, etc.).
- **Files modified:** `cmd/error-doc-gen/main.go`, `internal/errors/errors.go`.
- **Commit:** `5df3b39`.

### Known cleanup (NOT a deviation, per plan §acceptance_criteria)

**Ad-hoc audit footer reports 31 inline `err.Error()` classify branches remaining**

- The plan's acceptance criteria explicitly noted: "target: 0; if
  non-zero, document remaining sites in commit body as known cleanup".
- After `20-02` + `20-03` landed, 31 inline branches remain in
  `internal/handlers/*.go` — Phase 20 planned but did not yet
  fully reach 0 (the cleanup was deferred or partially landed in
  other plans).
- These will be visible on every CI run via the docs/errors.md footer.
  This is the intended Phase 20 outcome per D-04.5 — turning the
  blind spot into a tracked signal that reviewers see on every PR.
- Spot-checks of `internal/handlers/` show the 31 matches are
  distributed across `err.Error()` calls and `errMsg :=` patterns,
  including some ShouldBindJSON error paths that are correctly
  pre-handler-validation and not true classify branches. The
  pattern can be tightened in a future phase if needed.

## Requirements Fulfilled

- **REQ-20c-generator**: ✅ `cmd/error-doc-gen/main.go` emits
  sentinel + BusinessError tables with HTTP status + call-site counts.
- **REQ-20c-doc-sync**: ✅ `//go:generate` directive + CI step
  enforcing `git diff --quiet docs/errors.md`.
- **REQ-20c-call-site-count**: ✅ Call-site column populated via
  in-process grep; values range from 4 (ErrDuplicateRecord) to 146
  (ErrInvalidInput) across the 42 sentinels.

## Locked Decisions Honored

- **D-04.1** (`//go:generate` at top of `internal/errors/errors.go`) ✅
- **D-04.2** (Generator location: `cmd/error-doc-gen/main.go`,
  pure text scan — no AST/reflection) ✅
- **D-04.3** (Table columns: Sentinel | Kind | HTTP Status |
  Call-site count) ✅
- **D-04.5** (Ad-hoc audit footer) ✅
- **D-05.3** (4-case verification — sentinel table / BusinessError
  table / missing filepath / diff stability; we have 5 including
  audit footer) ✅
- **R-2** (No Makefile — `go:generate` + CI step only) ✅

## Self-Check

PASSED. All created files exist, all commits land, all acceptance
criteria met.

- `cmd/error-doc-gen/main.go` — FOUND
- `cmd/error-doc-gen/main_test.go` — FOUND
- `docs/errors.md` — FOUND (3,642 bytes, 84 lines, 42 sentinel
  rows + 10 BusinessError rows + ad-hoc audit footer)
- `internal/errors/errors.go` — FOUND, contains `//go:generate`
  directive
- `.github/workflows/ci.yml` — FOUND, contains
  `Verify errors doc sync` step
- No `Makefile` / `makefile` — confirmed absent
- Commits `d13a508` and `5df3b39` — confirmed in git log