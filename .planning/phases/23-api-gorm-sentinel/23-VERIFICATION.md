---
phase: 23-api-gorm-sentinel
verified: 2026-08-06T10:35:00Z
status: passed
score: 6/6 must-haves verified
overrides_applied: 0
overrides: []
gaps: []
gap_fix:
  - original_status: gaps_found
    fixed_in_commit: 971faf8
    fix_description: "docs/errors.md regenerated + committed. ErrRecordingHuaWeiStateFetchFailed call-site count updated from 5 → 20 (Plan 23-01 added 15 huawei call-sites after Plan 23-03 last regenerated). `git diff --exit-code -- docs/errors.md` now exits 0."
---

# Phase 23: API+GORM+Sentinel Verification Report

**Phase Goal:** 落地 H 信号数据通路与可观测基线 (DETECT-01/04, AUDIT-01/05, CFG-01/02)
**Verified:** 2026-08-06T10:35:00Z
**Status:** passed (gap fixed post-verification by orchestrator commit `971faf8`)

## Goal Achievement

### Observable Truths (per requirement)

| #   | REQ-ID   | Truth                                                                                                                  | Status        | Evidence                                                                                          |
| --- | -------- | ---------------------------------------------------------------------------------------------------------------------- | ------------- | ------------------------------------------------------------------------------------------------- |
| 1   | DETECT-01 | `HuaweiClient.GetConferenceState()` returns `ConferenceState` struct with `ConfState`/`JoinSum` and `HasConferenceFields` flag | ✓ VERIFIED    | `internal/huawei/client.go:861` exports `GetConferenceState(ctx) (*ConferenceState, error)`; struct at line 278 has 5 fields |
| 2   | DETECT-04 | HasConferenceFields distinguishes absent keys from empty values for old-device fallback                               | ✓ VERIFIED    | `internal/huawei/client.go:251` `detectConferenceFields` helper; uses `json.RawMessage` for presence detection; `HasConferenceFields` is set in build at line 877 |
| 3   | AUDIT-01  | VideoRecordingTask adds 5 GORM-tagged fields (ExtensionCount, LastExtensionReason, EndedEarly, EndedEarlyReason, EndedByHuaWeAPI) | ✓ VERIFIED    | `internal/models/video_recording_task.go:39-44` has the 5 fields with GORM tags; `EndedByHuaWeAPI` column tag locks `ended_by_huawei_api`; SQLite :memory: test asserts all 5 columns exist and round-trip works |
| 4   | AUDIT-05  | 3 sentinels defined, recognized by IsKnownError/FirstKnownSentinelName, map to HTTP 500, docs/errors.md regenerated     | ✓ VERIFIED    | Sentinels defined and wired (errors.go:101-105, mapping.go:90-95 + knownSentinels 206-208); tests pass; mapping_test.go covers all 3; docs/errors.md regenerated + committed in `971faf8` (`ErrRecordingHuaWeiStateFetchFailed` count updated from 5 → 20 reflecting 15 new huawei call-sites from Plan 23-01); CI sync-check `git diff --exit-code -- docs/errors.md` exits 0 |
| 5   | CFG-01    | SmartEndConfig struct with 14 typed fields + Validate                                                                  | ✓ VERIFIED    | `internal/config/smart_end.go:20-69` defines SmartEndConfig with exactly 14 fields, each with mapstructure/json/yaml triple tags; `Validate()` at line 147; `applySmartEndDefaults` at line 93; Config integration at `config.go:46` (field), 382-384 (SetDefault), 400 (Validate), 682 (applySmartEndDefaults); 17 tests pass |
| 6   | CFG-02    | Both `config.yaml` and `bin/config.yaml` carry same smart_end: section with 14 keys                                    | ✓ VERIFIED    | `config.yaml:77-92` and `bin/config.yaml:78-93` byte-for-byte identical; `internal/config/smart_end_yaml_test.go` has 4 tests (`TestSmartEndYAML_Exactly14Keys`, `_RootBinSync`, `_ExpectedDefaults`, `_ViperLoadsCleanly`) all pass |

**Score:** 6/6 requirements fully verified

### Required Artifacts

| Artifact                                                                                              | Expected                                                          | Status          | Details                                                                                       |
| ----------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- | --------------- | --------------------------------------------------------------------------------------------- |
| `internal/huawei/client.go`                                                                           | `ConferenceState` struct + `GetConferenceState` method + presence helper | ✓ VERIFIED      | All 5 fields; struct lines 278-286; method signature at line 861; presence helper at 251       |
| `internal/huawei/client_test.go`                                                                      | 7 fixture subtests + top-level `TestGetConferenceState_FallbackFlag`            | ✓ VERIFIED      | All 7 subtests pass; top-level alias at line 294                                              |
| `internal/models/video_recording_task.go`                                                              | 5 new GORM fields between ConversionRetryCount and CreatedBy       | ✓ VERIFIED      | Lines 40-44 with required tags; `EndedByHuaWeAPI` has `column:ended_by_huawei_api` tag       |
| `internal/models/video_recording_task_test.go` (NEW)                                                   | Schema migration + defaults + round-trip tests                    | ✓ VERIFIED      | 3 test functions; 5 subtests under SchemaMigration; all pass                                    |
| `internal/errors/errors.go`                                                                           | 3 new sentinel vars in phase-tagged block                          | ✓ VERIFIED      | Lines 101, 103, 105; under "Smart-end recording sentinels (Phase 23 AUDIT-05)" comment block    |
| `internal/errors/mapping.go`                                                                          | 500-case switch + knownSentinels entries                          | ✓ VERIFIED      | Lines 93-95 (switch); 206-208 (knownSentinels)                                                |
| `internal/errors/mapping_test.go`                                                                     | TestMapToHTTPStatus_Sentinels + TestFirstKnownSentinelName rows   | ✓ VERIFIED      | 3 rows each; all pass under `-race`                                                            |
| `docs/errors.md`                                                                                      | Auto-regenerated sentinel table with 3 new rows                   | ⚠️ PARTIAL      | Contains all 3 rows but `ErrRecordingHuaWeiStateFetchFailed` count stale (5 vs regenerated 20) |
| `internal/config/smart_end.go` (NEW)                                                                  | SmartEndConfig + 14 fields + applySmartEndDefaults + Validate      | ✓ VERIFIED      | File exists (185 lines); all 4 obligations met                                                  |
| `internal/config/smart_end_test.go` (NEW)                                                              | Defaults + ExplicitFalsePreserved + InvalidRejection (13 cases)   | ✓ VERIFIED      | 3 functions; 15 subtests total; all pass                                                        |
| `internal/config/smart_end_yaml_test.go` (NEW)                                                        | Exactly14Keys + RootBinSync + ExpectedDefaults + ViperLoadsCleanly | ✓ VERIFIED     | 4 test functions; all pass                                                                    |
| `config.yaml`                                                                                         | 14-key smart_end: section                                          | ✓ VERIFIED      | Lines 77-92; all 14 keys with documented defaults                                              |
| `bin/config.yaml`                                                                                     | 14-key smart_end: section (mirror root)                            | ✓ VERIFIED      | Lines 78-93; byte-for-byte identical to root via `TestSmartEndYAML_RootBinSync`               |

### Key Link Verification

| From                                                                                          | To                                                                                            | Via                                                              | Status    | Details                                                                                              |
| --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ---------------------------------------------------------------- | --------- | ---------------------------------------------------------------------------------------------------- |
| `internal/huawei/client.go` `parseMailboxState`                                               | `internal/errors`                                                                            | `fmt.Errorf("...: %w", apperrors.ErrRecordingHuaWeiStateFetchFailed)` | WIRED     | All 4 error paths in `parseMailboxState` (line 228/235/238/243) wrap sentinel; `GetConferenceState` (line 864) wraps transport errors |
| `internal/huawei/client.go` `detectConferenceFields`                                           | `parseMailboxState`                                                                          | `json.RawMessage` presence check                                 | WIRED     | Uses anonymous struct `State {ConfState RawMessage; JoinSum RawMessage}`; called from line 877      |
| `internal/models/video_recording_task.go`                                                     | `video_recording_tasks` GORM table                                                           | `gorm:"column:..."` tags (snake_case)                            | WIRED     | `EndedByHuaWeAPI` has explicit `column:ended_by_huawei_api` tag locking SQL name; `HasColumn` test asserts |
| `internal/errors/errors.go`                                                                    | `internal/errors/mapping.go`                                                                 | import + sentinel var references                                  | WIRED     | Both 500-case switch and knownSentinels include all 3 new sentinels                                  |
| `internal/errors/mapping.go`                                                                   | `docs/errors.md`                                                                             | `go generate ./internal/errors/...`                               | BROKEN    | Generator runs but regenerated docs is NOT committed (drift: 5 → 20 for ErrRecordingHuaWeiStateFetchFailed) |
| `internal/config/smart_end.go`                                                                 | `internal/config/config.go`                                                                  | `Config.SmartEnd SmartEndConfig` field + Load() integration       | WIRED     | Field at config.go:46; SetDefault at 382-384; Validate at 400; applySmartEndDefaults at 682           |
| `internal/config/config.go`                                                                    | `apperrors`                                                                                  | `fmt.Errorf("...: %w: %w", apperrors.ErrInternal, err)`          | WIRED     | Line 401 wraps Validate() error with apperrors.ErrInternal for fail-closed startup                   |
| `config.yaml`                                                                                  | `internal/config/smart_end.go` `SmartEndConfig`                                               | Viper mapstructure/triple tags + Unmarshal                        | WIRED     | `TestSmartEndYAML_ViperLoadsCleanly` reads full root config.yaml and unmarshals; all 14 fields populated correctly |
| `config.yaml`                                                                                  | `bin/config.yaml`                                                                            | on-disk parity (gitignored so no git enforcement)                | WIRED     | `TestSmartEndYAML_RootBinSync` enforces byte-for-byte equality; both files have identical smart_end: section |

### Data-Flow Trace (Level 4)

| Artifact                                                            | Data Variable             | Source                                                    | Produces Real Data | Status   |
| ------------------------------------------------------------------- | ------------------------- | --------------------------------------------------------- | ------------------ | -------- |
| `internal/huawei/client.go` `GetConferenceState`                     | raw mailbox JSON payload  | `c.getMailboxRaw(ctx)` HTTP call to WEB_GetMailboxDataAPI  | LIVE               | ✓ FLOWING (in production) — fixture-based in tests |
| `internal/huawei/client.go` `parseMailboxState`                     | `data string` → `*MailboxState` | caller passes JSON string                            | REAL (fixture)     | ✓ FLOWING — 7 fixtures cover all cases              |
| `internal/config/smart_end_yaml_test.go` `TestSmartEndYAML_ViperLoadsCleanly` | YAML → cfg.SmartEnd.*    | `bytes.NewReader(root config.yaml contents)` → Viper      | REAL               | ✓ FLOWING — assertions pass on live config           |
| `internal/models/video_recording_task_test.go` `_RoundTrip`         | `task model struct` → SQLite row → `task model` struct | `db.Create` then `db.First`                     | REAL               | ✓ FLOWING — round-trip preserves 5 fields            |

### Behavioral Spot-Checks

| Behavior                                                                    | Command                                                                                                              | Result                          | Status   |
| --------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- | ------------------------------- | -------- |
| All 7 Huawei mailbox-state subtests pass                                     | `go test ./internal/huawei -run 'TestParseMailboxState|TestGetConferenceState' -count=1 -v`                            | 8 subtests + 1 top-level PASS   | ✓ PASS   |
| 3 GORM schema tests pass (SchemaMigration + Defaults + RoundTrip)          | `go test ./internal/models -run 'TestVideoRecordingTaskSmartEndFields' -count=1 -v`                                  | 8 sub-tests PASS                | ✓ PASS   |
| 3 SmartEndConfig tests pass (Defaults + ExplicitFalsePreserved + 14 InvalidRejection subtests + 1 valid baseline) | `go test ./internal/config -run 'TestSmartEndConfig' -count=1 -v`                                  | 17 sub-tests PASS               | ✓ PASS   |
| 4 SmartEndYAML tests pass                                                    | `go test ./internal/config -run 'TestSmartEndYAML' -count=1 -v`                                                      | 4 tests pass (Exactly14Keys has 2 sub-tests) | ✓ PASS   |
| 3 sentinel recognition tests pass in errors package                         | `go test ./internal/errors -count=1 -v`                                                                              | All tests pass                  | ✓ PASS   |
| error-doc-gen tests pass                                                    | `go test ./cmd/error-doc-gen -count=1 -v`                                                                            | All 4 tests pass (slow ~60s; idempotency is what's at stake) | ✓ PASS   |
| Project-wide vet                                                            | `go vet ./...`                                                                                                       | exit 0                          | ✓ PASS   |
| Project-wide build                                                          | `go build ./...`                                                                                                     | exit 0                          | ✓ PASS   |
| CI sync-check (commutative idempotency of `go generate`)                   | `go generate ./internal/errors/... && git diff --exit-code -- docs/errors.md`                                        | **NON-EXIT-0**: docs regenerated changes `ErrRecordingHuaWeiStateFetchFailed` count from 5 → 20 | ✗ FAIL   |

### Probe Execution

| Probe | Command | Result | Status |
| ----- | ------- | ------ | ------ |
| (no `scripts/*/tests/probe-*.sh` in repo — probing not applicable for Go backend phase) | n/a | n/a | SKIP — Go backend phase; tested via `go test` with hard pass/fail |

### Requirements Coverage

| Requirement | Source Plan | Description                                                                                                            | Status        | Evidence                                                                                            |
| ----------- | ---------- | ---------------------------------------------------------------------------------------------------------------------- | ------------- | --------------------------------------------------------------------------------------------------- |
| DETECT-01   | 23-01      | HuaweiClient.GetConferenceState exposes confState/joinSum and presence flag                                            | ✓ SATISFIED   | client.go:278-286 struct; 861-879 method; 226-246 parseMailboxState; 7 fixture tests pass           |
| DETECT-04   | 23-01      | HasConferenceFields allows fallback to IsInConf==0 on old devices                                                      | ✓ SATISFIED   | detectConferenceFields (line 251) uses json.RawMessage presence detection; TestGetConferenceState_FallbackFlag verifies |
| AUDIT-01    | 23-02      | VideoRecordingTask adds 5 smart-end columns via AutoMigrate                                                            | ✓ SATISFIED   | video_recording_task.go:40-44; SQLite :memory: test SchemaMigration+Defaults+RoundTrip all pass      |
| AUDIT-05    | 23-03      | 3 sentinels defined, recognized as 500, docs/errors.md regenerated and CI sync-check passes                            | ⚠️ BLOCKED    | Sentinels + mapping + tests all OK; **docs/errors.md drifted** (5 vs 20) after Plan 23-01 added references to `ErrRecordingHuaWeiStateFetchFailed` |
| CFG-01      | 23-04      | SmartEndConfig struct + 14 typed fields + Validate                                                                     | ✓ SATISFIED   | smart_end.go: 14 fields + Validate + applySmartEndDefaults; Config integration at config.go:382,400,682 |
| CFG-02      | 23-05      | config.yaml + bin/config.yaml carry smart_end: section with 14 keys, sync test enforced                                 | ✓ SATISFIED   | Both YAMLs byte-identical at lines 77-92 / 78-93; 4 test functions pass                              |

**Coverage:** 5 fully covered + 1 blocked (docs drift)

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| (none scanned with grep patterns: TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER/console.log only) | — | — | — | — |
| docs/errors.md | 34 | Stale call-site count for ErrRecordingHuaWeiStateFetchFailed (5 vs regenerated 20) | 🛑 BLOCKER | CI sync-check gate at .github/workflows/ci.yml:54-60 would fail; this breaks the `AUDIT-05` truth that explicitly requires `git diff --exit-code -- docs/errors.md` to exit 0 |

### Human Verification Required

None. All 6 requirements are deterministically verifiable via grep, file inspection, and `go test`. No visual, real-time, or external-service behaviors to test.

### Gaps Summary

**1 gap blocking phase goal:**

**AUDIT-05 partial — docs/errors.md drift after Plan 23-01 added references**

The committed `docs/errors.md` shows `ErrRecordingHuaWeiStateFetchFailed | Sentinel | 500 | 5` but the actual call-site count in source code is 20 (5+ references in `errors.go`, `mapping.go`, `mapping_test.go` + 15 new references in `internal/huawei/client.go` and `client_test.go` introduced by Plan 23-01 between commits `a463813`/`9d4e7d8`).

Timeline of relevance:
- Commit `3400532` (Plan 23-03, 2026-08-06 09:53) regenerated docs/errors.md with count=5 (correct relative to source at that time)
- Commit `9d4e7d8` (Plan 23-01, 2026-08-06 10:02) added 14 references to `apperrors.ErrRecordingHuaWeiStateFetchFailed` in `internal/huawei/client.go` and `internal/huawei/client_test.go`
- Commit `cd60818` (Plan 23-01 refactor, 10:04) added 1 more reference in test
- No subsequent commit regenerated docs/errors.md to update the count from 5 → 20

Fix: Run `go generate ./internal/errors/...` and commit the regenerated docs/errors.md (line 34 will update from `5` to `20`). This is a one-line change to the file and will pass CI sync-check on the next push.

Verification commands to confirm fix:
```bash
cd "D:/CODE/ClaudeCode/record_V2"
go generate ./internal/errors/...
git diff docs/errors.md    # should show ONLY the ErrRecordingHuaWeiStateFetchFailed count change
go generate ./internal/errors/...   # second run
git diff --exit-code -- docs/errors.md    # should now exit 0
```

---

_Verified: 2026-08-06T10:30:00Z_
_Verifier: Claude (gsd-verifier)_
