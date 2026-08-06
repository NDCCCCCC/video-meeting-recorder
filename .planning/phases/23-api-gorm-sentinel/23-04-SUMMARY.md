---
phase: 23-api-gorm-sentinel
plan: 04
subsystem: config
tags: [config, viper, smart-end, CFG-01]
dependency_graph:
  requires: []
  provides:
    - "internal/config/smart_end.go exports SmartEndConfig (14 typed fields) with applySmartEndDefaults + Validate"
    - "Config.SmartEnd field with triple tag"
    - "Viper SetDefault hook for 3 true-valued bools (Pitfall 3 fix)"
  affects:
    - "internal/config/config.go Load() — bool-default registration + Validate fail-closed"
tech_stack:
  added: []
  patterns:
    - "Strong-typed config sub-struct (mapstructure/json/yaml triple tag)"
    - "Two-phase defaults: Viper SetDefault for bool (before Unmarshal) + applyDefaults zero-value fill for numerics (after Unmarshal)"
    - "Validate wraps apperrors.ErrInvalidInput so callers can errors.Is()"
key_files:
  created:
    - path: internal/config/smart_end.go
      purpose: "SmartEndConfig struct (14 fields) + applySmartEndDefaults + Validate"
    - path: internal/config/smart_end_test.go
      purpose: "Defaults / explicit-false preservation / table-driven invalid rejection tests"
  modified:
    - path: internal/config/config.go
      purpose: "SmartEnd field on Config, Load() Viper SetDefault hook, setDefaults() applySmartEndDefaults, Load() Validate fail-closed"
decisions:
  - "Split defaults into two phases: 3 true-valued bools via Viper SetDefault pre-Unmarshal (preserves operator's explicit YAML false); 11 numerics via applySmartEndDefaults post-Unmarshal. Avoids the bool-default-zero-value trap (RESEARCH.md Pitfall 3)."
  - "Validate() wraps apperrors.ErrInvalidInput (not a new sentinel) — boundaries/limit errors are caller-policy errors, not domain sentinels."
  - "Load() turns Validate() errors into apperrors.ErrInternal — fail-closed startup (avoids garbage thresholds in Phase 24 watcher)."
metrics:
  duration_seconds: ~600
  completed_date: 2026-08-06
  test_count: 17
  files_added: 2
  files_modified: 1
---

# Phase 23 Plan 04: SmartEndConfig Structure + Validation Summary

One-liner: SmartEndConfig struct with 14 typed fields, Viper defaults split into pre-Unmarshal bool SetDefault (Pitfall 3 fix) and post-Unmarshal numeric zero-value fill, plus Validate() wrapping apperrors.ErrInvalidInput — phase 24 watcher / phase 25 scheduler can now read cfg.SmartEnd.* directly.

## What Was Built

1. **`internal/config/smart_end.go` (new, 185 lines)** — `SmartEndConfig` struct with the 14 typed fields specified by `REQUIREMENTS.md:57` and `RESEARCH.md §CFG-01` table. Each field carries matching lowercase `mapstructure`/`json`/`yaml` triple tags so the same struct is usable from Viper, REST APIs, and template YAML. Companion helpers:
   - `applySmartEndDefaults(cfg *Config)`: zero-value replacement for 11 numeric fields; deliberately does NOT touch the 3 bool fields.
   - `(*SmartEndConfig).Validate() error`: range checks in declared order (silence_db → positive durations/counts → file_min_growth_bps ≥ 0 → max_extend_count > 0). Every error path wraps `apperrors.ErrInvalidInput`.

2. **`internal/config/smart_end_test.go` (new, 201 lines)** — three test functions:
   - `TestSmartEndConfig_Defaults` — exercises Viper → Unmarshal → applySmartEndDefaults with empty `smart_end` YAML and asserts all 14 expected defaults (Enabled=true, SilenceDB=-30, SilenceDurationS=30, FileStallS=120, FileMinGrowthBPS=1024, HuaweiEnabled=true, HuaweiPollIntervalS=30, HuaweiPersistS=30, HuaweiFailureThreshold=3, CheckIntervalS=5, ExtendStepMin=30, MaxExtendCount=4, StatFailureThreshold=3, DegradeOnSilenceLoss=true).
   - `TestSmartEndConfig_ExplicitFalsePreserved` — the **key** regression test for Pitfall 3: SetDefault(true) registers before ReadConfig(`enabled: false`/`huawei_enabled: false`/`degrade_on_silence_loss: false`), and the three bools stay false after Unmarshal + applySmartEndDefaults. Verifies CFG-03/04 rollback switches work.
   - `TestSmartEndConfig_InvalidRejection` — table-driven over 13 invalid-value mutations + 1 valid baseline; each case asserts `errors.Is(err, apperrors.ErrInvalidInput)`. Sub-tests cover silence_db too high/too low, silence_duration_s ≤ 0, file_stall_s ≤ 0, file_min_growth_bps < 0, all 8 other positive-check fields.

3. **`internal/config/config.go` (modified, +22 lines, 4 surgical edits)** — minimal changes:
   - L42-L46: add `SmartEnd SmartEndConfig` field to `Config` struct with `smart_end` triple tag.
   - L382-L384: in `Load()`, after the second `v = viper.New()` and BEFORE `bindSecretEnv(v)`, register `v.SetDefault("smart_end.enabled", true)`, `v.SetDefault("smart_end.huawei_enabled", true)`, `v.SetDefault("smart_end.degrade_on_silence_loss", true)`. Order matters: explicit YAML `false` must beat SetDefault.
   - L393-L397: in `Load()`, immediately after `setDefaults(&cfg)`, call `cfg.SmartEnd.Validate()`; on error return wrapped `apperrors.ErrInternal` (fail-closed startup).
   - L681-L682: at end of `setDefaults`, call `applySmartEndDefaults(cfg)` for numeric zero-value fill.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Stale package doc comment** (not flagged in plan but discovered during commit prep)
- **Found during:** Self-review before commit
- **Issue:** Initial draft of `smart_end.go` had no package-level doc explaining why defaults are split (the "Pitfall 3" reason).
- **Fix:** Added package-level `SmartEndConfig` struct doc explicitly stating why the bool fields skip `applySmartEndDefaults` and rely on Viper SetDefault instead. Serves future readers / Phase 25 implementers.
- **Files modified:** `internal/config/smart_end.go`
- **Commit:** `600a638` (rolled into GREEN commit)

None beyond that — the plan executed exactly as written.

## Verification Results

```
go test ./internal/config -run TestSmartEndConfig -count=1 -v
=== RUN   TestSmartEndConfig_Defaults
--- PASS: TestSmartEndConfig_Defaults (0.00s)
=== RUN   TestSmartEndConfig_ExplicitFalsePreserved
--- PASS: TestSmartEndConfig_ExplicitFalsePreserved (0.00s)
=== RUN   TestSmartEndConfig_InvalidRejection   (13 sub-tests + 1 valid)
--- PASS: TestSmartEndConfig_InvalidRejection (0.00s)
PASS
ok  	github.com/NDCCCCCC/video-meeting-recorder/internal/config	0.873s
```

```
go test ./internal/config -count=1 -race
ok  	github.com/NDCCCCCC/video-meeting-recorder/internal/config	2.503s
```

```
go vet ./internal/config    # exits 0 (no output)
go build ./...             # exit 0, full project builds clean
```

No regression in existing config tests (ValidateProductionSecrets, ValidateCredentialSM4Config_*, BindEnvCredentialSM4, CSRFEnabledEnvBinding, CSRFSafeOriginsEnvBinding all still pass).

## TDD Gate Compliance

| Gate | Commit | Verified |
|------|--------|----------|
| RED  | `43c04d4` test(23-04): add failing tests for SmartEndConfig | Yes — build failed until GREEN commit |
| GREEN | `600a638` feat(23-04): implement SmartEndConfig | Yes — all 17 test cases pass post-commit |
| REFACTOR | (skipped) | Implementation already minimal; no further cleanup warranted |

## Acceptance Criteria Met

- [x] `internal/config/smart_end.go` exists with `type SmartEndConfig` struct (14 fields)
- [x] Each field has matching lowercase `mapstructure`/`json`/`yaml` triple tags (e.g. `huawei_poll_interval_s`)
- [x] `SmartEndConfig` has `Validate() error` method
- [x] `internal/config/smart_end_test.go` exists with 3 test functions (Defaults / ExplicitFalsePreserved / InvalidRejection)
- [x] `Config.SmartEnd SmartEndConfig` field with triple tag `smart_end` is present
- [x] `Load()` registers `v.SetDefault` for the 3 true-valued bools BEFORE `v.Unmarshal`
- [x] `setDefaults()` calls `applySmartEndDefaults(cfg)`
- [x] `Load()` calls `cfg.SmartEnd.Validate()` and wraps non-nil with `apperrors.ErrInternal`
- [x] All 17 test cases pass; no regression in existing config tests

## Threat Flags

None. No new network endpoints, auth paths, file access patterns, or schema changes at trust boundaries were introduced — this plan adds a typed config struct and its defaults, all internal to config loading.

## Self-Check: PASSED

- `internal/config/smart_end.go` — FOUND (created in commit `600a638`)
- `internal/config/smart_end_test.go` — FOUND (created in commit `43c04d4`)
- `internal/config/config.go` edits — FOUND (lines 42-46 SmartEnd field, 382-384 SetDefault, 393-397 Validate call, 681-682 applySmartEndDefaults; all in commit `600a638`)
- Commits `43c04d4` and `600a638` — FOUND in `git log`
- `go test ./internal/config -race` exits 0 — PASSED
- `go vet ./internal/config` exits 0 — PASSED
- `go build ./...` exit 0 — PASSED

## Hand-off Notes for Phase 24/25

Phase 24 watcher should:
1. Read `cfg.SmartEnd.Enabled` as the master switch (if false, watcher goroutine does not start at all — CFG-03).
2. Read `cfg.SmartEnd.HuaweiEnabled` to decide whether to start the H-signal poller (CFG-04).
3. Use `cfg.SmartEnd.CheckIntervalS`, `cfg.SmartEnd.HuaweiPollIntervalS`, `cfg.SmartEnd.HuaweiPersistS`, etc. as native seconds — no unit-conversion logic needed in the watcher since defaults are already in seconds.

Phase 25 scheduler should:
1. Read `cfg.SmartEnd.ExtendStepMin` and `cfg.SmartEnd.MaxExtendCount` for the EXTEND-01 cap.
2. Read `cfg.SmartEnd.HuaweiFailureThreshold` and `cfg.SmartEnd.StatFailureThreshold` for WATCH-03 / WATCH-04 counters.
3. Use `cfg.SmartEnd.DegradeOnSilenceLoss` for WATCH-02 silent fallback toggle.

The Validate() fail-closed guarantee means a misconfigured deployment will not start — operators must set sane thresholds in `config.yaml` before the server boots.
