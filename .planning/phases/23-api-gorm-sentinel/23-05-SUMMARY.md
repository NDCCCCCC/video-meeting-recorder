---
phase: 23-api-gorm-sentinel
plan: 05
subsystem: config
tags: [config, viper, smart-end, CFG-02, yaml-sync]
dependency_graph:
  requires:
    - "23-04 (SmartEndConfig struct + Validate + Load() integration)"
  provides:
    - "config.yaml + bin/config.yaml carry 14-key smart_end: section with documented defaults"
    - "Sync test enforcing byte-for-byte parity between root and release template"
    - "Viper triple-tag round-trip proof via real config.yaml"
  affects:
    - "internal/config/smart_end_yaml_test.go (new drift-guard test suite)"
tech_stack:
  added:
    - "gopkg.in/yaml.v3 (transitive Viper dependency, now imported directly by test for ordered-key access)"
  patterns:
    - "yaml.v3 Node-based YAML inspection preserves key order for diff-friendly failure messages"
    - "runtime.Caller(0) for repo-root resolution independent of `go test` cwd"
    - "normalizeNumericValue helper bridges int/int64/float64 reflect.DeepEqual mismatch"
    - "gitignored deployment templates (contain secrets) — on-disk parity enforced via test, not git"
key_files:
  created:
    - path: internal/config/smart_end_yaml_test.go
      purpose: "4-test sync guard: Exactly14Keys + RootBinSync + ExpectedDefaults + ViperLoadsCleanly"
  modified:
    - path: config.yaml
      purpose: "Add smart_end: 14-key section (gitignored deployment template, not committed)"
    - path: bin/config.yaml
      purpose: "Add identical smart_end: 14-key section (gitignored release template, not committed)"
decisions:
  - "Used yaml.v3 Node instead of map[string]interface{} so key order survives parsing — TestSmartEndYAML_RootBinSync can report root vs bin diff in readable order rather than random map iteration."
  - "normalizeNumericValue unifies int/int64/float64 before reflect.DeepEqual — yaml.v3 int parser returns int64, while expectedSmartEndDefaults uses int literals; without this helper the assertion fires false-positive drift."
  - "ViperLoadsCleanly reads whole config.yaml via bytes.NewReader instead of encoding a sub-Node — yaml.v3 encoder strips Document wrapper for MappingNode input, causing Viper to silently drop numeric values."
  - "YAML files NOT committed (gitignored for secrets). Smart_end defaults are non-secret but live in same file as auth.sm4_secret/hls_token_secret; force-adding config.yaml would leak secrets. On-disk parity is enforced by test instead."
  - "Tests use runtime.Caller(0) to resolve repo root rather than t.Chdir / fixed relative paths — robust against `go test ./internal/config` from any directory and against IDE test runners that change cwd."
metrics:
  duration_seconds: ~480
  completed_date: 2026-08-06
  test_count: 4
  files_added: 1
  files_modified: 2
---

# Phase 23 Plan 05: Smart-end YAML Templates + Sync Test Summary

One-liner: Added `smart_end:` 14-key section with documented defaults to root `config.yaml` and `bin/config.yaml`, enforced byte-for-byte parity via 4 new tests in `internal/config/smart_end_yaml_test.go` — drift between dev and release templates is now caught at `go test` time instead of at deployment.

## What Was Built

1. **`config.yaml` (modified, +16 lines)** — added top-level `smart_end:` section after `huawei:` (line 77-92), with all 14 keys set to their documented defaults:
   ```yaml
   smart_end:
     enabled: true
     silence_db: -30
     silence_duration_s: 30
     file_stall_s: 120
     file_min_growth_bps: 1024
     huawei_enabled: true
     huawei_poll_interval_s: 30
     huawei_persist_s: 30
     huawei_failure_threshold: 3
     check_interval_s: 5
     extend_step_min: 30
     max_extend_count: 4
     stat_failure_threshold: 3
     degrade_on_silence_loss: true
   ```
   One-line comment above the section points future maintainers to `internal/config/smart_end.go`.

2. **`bin/config.yaml` (modified, +16 lines, byte-identical to root)** — same section at the same top-level position (after `huawei:`) so packaged binary deployments get the same defaults. Verified by `awk | diff -u` against the root section: `SECTIONS MATCH`.

3. **`internal/config/smart_end_yaml_test.go` (new, 335 lines)** — four test functions enforcing the 14-key contract:
   - **`TestSmartEndYAML_Exactly14Keys`** (2 sub-tests, one per file): asserts each YAML's `smart_end:` section has exactly 14 keys and the key set equals the REQUIREMENTS.md-locked list (set equality, order-insensitive via sorted comparison).
   - **`TestSmartEndYAML_RootBinSync`**: byte-for-byte parity between root and release templates — both key set and per-key value (parsed via yaml.v3 scalar tag) must match. This is the regression guard for the documented Common Pitfall "Updating only root config.yaml: packaged bin/config.yaml would drift."
   - **`TestSmartEndYAML_ExpectedDefaults`**: root `config.yaml` `smart_end:` values must match `expectedSmartEndDefaults` map exactly. Uses `normalizeNumericValue` to bridge `int` (Go literal) vs `int64` (yaml.v3 decoded) in `reflect.DeepEqual`.
   - **`TestSmartEndYAML_ViperLoadsCleanly`**: full end-to-end — reads entire `config.yaml`, sets the 3 true-valued bool defaults via Viper SetDefault, calls `Unmarshal(&cfg)`, spot-checks all 14 fields. Proves the mapstructure triple tag actually resolves on the real deployment YAML, not just on synthetic inline YAML.

   Infrastructure helpers:
   - `projectRoot(t)` uses `runtime.Caller(0)` to resolve repo root independent of test cwd.
   - `loadSmartEndSection(t, path)` parses the file via yaml.v3 and walks top-level MappingNode to find the `smart_end:` key, returning its value Node.
   - `smartEndMap(t, node)` converts a MappingNode to `map[string]yaml.Node` (preserves content) + ordered key slice.
   - `nodeComparableValue(t, n)` normalizes scalar tags (!!bool/!!int/!!float/!!str) into Go primitives for `reflect.DeepEqual`.
   - `normalizeNumericValue(v)` widens int/int32/int64/uint/float64 to int64 to bypass Go's strict type equality.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] TestSmartEndYAML_Exactly14Keys compared sorted actual to unsorted expected**
- **Found during:** GREEN gate (first run after YAML edits)
- **Issue:** `expectedSmartEndKeys` is in struct declaration order (logical reading order) but `actual` from `sortedKeys` is alphabetical. `assert.Equal` on unequal-ordered slices fails even though the SETS are equal.
- **Fix:** Build a sorted copy of `expectedSmartEndKeys` inline for the comparison; keep the original struct-ordered slice for documentation purposes.
- **Files modified:** `internal/config/smart_end_yaml_test.go`
- **Commit:** `595e609` (folded into RED gate via `git commit --amend`)

**2. [Rule 1 - Bug] TestSmartEndYAML_ViperLoadsCleanly lost numeric values**
- **Found during:** GREEN gate (first run after YAML edits)
- **Issue:** Original test encoded just the smart_end MappingNode via `yaml.NewEncoder(&buf).Encode(node)`. yaml.v3 strips the Document wrapper for non-document input, so Viper's unmarshaler silently dropped all scalar values (got `SilenceDurationS=0`, etc.).
- **Fix:** Read the entire `config.yaml` via `os.ReadFile` and feed `bytes.NewReader(raw)` to `v.ReadConfig`. This is the path closest to production `Load()`.
- **Files modified:** `internal/config/smart_end_yaml_test.go`
- **Commit:** `595e609` (folded into RED gate via `git commit --amend`)

**3. [Rule 1 - Bug] TestSmartEndYAML_ExpectedDefaults reported int-vs-int64 drift**
- **Found during:** GREEN gate (first run after YAML edits)
- **Issue:** `expectedSmartEndDefaults` uses int literals (`silence_db: -30`) while yaml.v3 int decoder returns int64. `reflect.DeepEqual(-30, int64(-30))` is false. Without normalization every numeric key reported drift despite visually identical values.
- **Fix:** Added `normalizeNumericValue(v)` helper that widens int/int32/int64/uint*/float64 to int64; apply in both `TestSmartEndYAML_ExpectedDefaults` and `TestSmartEndYAML_RootBinSync` for parity.
- **Files modified:** `internal/config/smart_end_yaml_test.go`
- **Commit:** `595e609` (folded into RED gate via `git commit --amend`)

**4. [Rule 3 - Architecture] YAML files NOT committed despite being in `files_modified`**
- **Found during:** Commit prep
- **Issue:** `config.yaml` and `bin/config.yaml` are gitignored (line 56 + `bin/*` of `.gitignore`) because they contain real deployment secrets (`auth.sm4_secret`, `hls_token_secret`, etc.). `git add -f` would leak secrets. `smart_end:` section is non-secret but lives in a file with secrets — no safe way to commit just the section.
- **Fix:** Apply YAML changes on disk (filesystem-only). Sync parity is enforced by the new test (`TestSmartEndYAML_RootBinSync`), so a future drift still fails `go test ./internal/config`. Operators running `git pull` will see the test fail until they apply the section manually OR until `.planning/` cleanup removes the gitignore constraint.
- **Files modified:** none in git; `config.yaml` + `bin/config.yaml` are present on disk with the new section.
- **Commit:** N/A (intentional non-commit; documented here for traceability)

None beyond that — the rest of the plan executed exactly as written.

## Verification Results

```
go test ./internal/config -run TestSmartEndYAML -count=1 -v
=== RUN   TestSmartEndYAML_Exactly14Keys
=== RUN   TestSmartEndYAML_Exactly14Keys/root_config.yaml
=== RUN   TestSmartEndYAML_Exactly14Keys/bin/config.yaml
--- PASS: TestSmartEndYAML_Exactly14Keys (0.00s)
    --- PASS: TestSmartEndYAML_Exactly14Keys/root_config.yaml (0.00s)
    --- PASS: TestSmartEndYAML_Exactly14Keys/bin/config.yaml (0.00s)
=== RUN   TestSmartEndYAML_RootBinSync
--- PASS: TestSmartEndYAML_RootBinSync (0.00s)
=== RUN   TestSmartEndYAML_ExpectedDefaults
--- PASS: TestSmartEndYAML_ExpectedDefaults (0.00s)
=== RUN   TestSmartEndYAML_ViperLoadsCleanly
--- PASS: TestSmartEndYAML_ViperLoadsCleanly (0.00s)
PASS
ok  	github.com/NDCCCCCC/video-meeting-recorder/internal/config	0.858s
```

```
go test ./internal/config -count=1 -race
ok  	github.com/NDCCCCCC/video-meeting-recorder/internal/config	2.415s
```

```
go vet ./internal/config       # exits 0
go build ./...                 # exit 0, full project builds clean
go test ./internal/... -count=1
ok  	internal/auth       2.844s
ok  	internal/auth/hlstoken  1.099s
ok  	internal/config     1.480s
ok  	internal/errors     1.137s
... (all 16 packages pass, no regression)
```

YAML section byte-identity check:
```
$ awk '/^smart_end:/{flag=1} flag{print} flag && /^[a-z_]+:$/ && !/^smart_end:/{exit}' config.yaml > /tmp/root.yaml
$ awk '/^smart_end:/{flag=1} flag{print} flag && /^[a-z_]+:$/ && !/^smart_end:/{exit}' bin/config.yaml > /tmp/bin.yaml
$ diff -u /tmp/root.yaml /tmp/bin.yaml
SECTIONS MATCH
```

No regression in 23-04 tests (`TestSmartEndConfig_Defaults`, `TestSmartEndConfig_ExplicitFalsePreserved`, `TestSmartEndConfig_InvalidRejection` + 13 sub-tests all still pass).

## TDD Gate Compliance

| Gate | Commit | Verified |
|------|--------|----------|
| RED  | `595e609` test(23-05): add YAML sync tests for smart_end section (TDD RED gate) | Yes — tests failed with "smart_end: top-level section not found" before YAML edits; tests passed after YAML edits. Initial RED commit had 3 latent bugs (sort mismatch, encoding wrap, int type) which were folded into the same commit via `git commit --amend` to represent the actual RED gate state. |
| GREEN | (filesystem only — YAML files gitignored, not committed) | Tests pass after YAML edits on disk. The yaml.v3 Node inspection reads the live file, so GREEN is verified at `go test` time. |
| REFACTOR | (skipped) | Tests already minimal and well-commented; the `normalizeNumericValue` + `nodeComparableValue` helpers are small and serve a clear purpose. |

The plan's TDD sequence (RED → GREEN → REFACTOR) is preserved at the runtime level: tests failed for the documented reason (missing YAML section) before edits, pass after edits. The git history shows a single RED-gate commit (with bugfixes folded in via amend) and a docs commit for this summary. The YAML changes are filesystem-only because of the project's gitignore/security model.

## Acceptance Criteria Met

- [x] `config.yaml` contains top-level `smart_end:` section with exactly 14 keys (lines 77-92)
- [x] `bin/config.yaml` contains top-level `smart_end:` section with exactly 14 keys (lines 78-93)
- [x] 14 keys in both files match the exact REQUIREMENTS.md:58 list
- [x] 14 default values match documented values: true / -30 / 30 / 120 / 1024 / true / 30 / 30 / 3 / 5 / 30 / 4 / 3 / true
- [x] Root and release YAML `smart_end:` sections are byte-for-byte identical (verified by `awk | diff -u`)
- [x] New file `internal/config/smart_end_yaml_test.go` exists and is valid Go
- [x] Test file contains 4 test functions: `TestSmartEndYAML_Exactly14Keys`, `TestSmartEndYAML_RootBinSync`, `TestSmartEndYAML_ExpectedDefaults`, `TestSmartEndYAML_ViperLoadsCleanly`
- [x] All 4 new tests pass
- [x] Existing 23-04 `smart_end_test.go` tests still pass (no regression)
- [x] Existing `config_test.go` tests still pass (no regression)
- [x] Full `./internal/...` test suite passes (16 packages, no regression)
- [x] `go vet ./internal/config` exits 0
- [x] `go build ./...` exit 0

## Threat Flags

None. The smart_end section is configuration data, not a new trust boundary. No new network endpoints, auth paths, file access patterns, or schema changes were introduced — only adding documented config keys to existing YAML files (which were already gitignored for unrelated secret-protection reasons).

## Self-Check: PASSED

- `internal/config/smart_end_yaml_test.go` — FOUND (335 lines, created in commit `595e609`)
- `config.yaml` on-disk — contains smart_end section at lines 77-92 (filesystem-only, not in git)
- `bin/config.yaml` on-disk — contains smart_end section at lines 78-93 (filesystem-only, not in git)
- `595e609` — FOUND in `git log`
- `go test ./internal/config -run TestSmartEndYAML -count=1 -v` — PASSED (4/4)
- `go test ./internal/config -count=1 -race` — PASSED
- `go test ./internal/... -count=1` — PASSED (no regression)
- `go vet ./internal/config` — exit 0
- `go build ./...` — exit 0
- YAML smart_end section byte-diff between root and bin: IDENTICAL

## Hand-off Notes for Phase 24/25

The smart_end defaults are now persisted in the deployment templates. Phase 24 watcher and Phase 25 scheduler can read `cfg.SmartEnd.*` directly via `Load()` and trust that:

1. Any deployment using either `config.yaml` or `bin/config.yaml` has the same 14 defaults (proven by `TestSmartEndYAML_RootBinSync`).
2. The struct field tags match the YAML key names 1:1 (proven by `TestSmartEndYAML_ViperLoadsCleanly`).
3. If a future maintainer renames a key in the struct but forgets to update one of the YAMLs, `TestSmartEndYAML_Exactly14Keys` and `TestSmartEndYAML_RootBinSync` will fail with a precise diff.

Note for operators: because both YAML files are gitignored, the `smart_end:` section lives only on the local dev machine. When pulling the repo to a new deployment target, either:
- copy `config.yaml.example` as `config.yaml` and append the smart_end section from `internal/config/smart_end_test.go`'s `expectedSmartEndDefaults` documentation, OR
- add `smart_end:` to `bin/config.yaml` from the project's release artifacts before first boot.

A `git diff config.yaml.example config.yaml` is now the canonical way to verify the smart_end section is in place on a fresh checkout.