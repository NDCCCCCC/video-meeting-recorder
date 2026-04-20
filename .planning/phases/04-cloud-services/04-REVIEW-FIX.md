---
phase: 04-cloud-services
fixed_at: 2026-04-18T20:00:00Z
review_path: .planning/phases/04-cloud-services/04-REVIEW.md
iteration: 2
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 04: Code Review Fix Report (Iteration 2)

**Fixed at:** 2026-04-18T20:00:00Z
**Source review:** .planning/phases/04-cloud-services/04-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 5 (1 Critical + 4 Warning)
- Fixed: 5
- Skipped: 0

## Fixed Issues

### CR-01: processCloudTranscription continues executing after pollTingwuStatus handles a failure

**Files modified:** `internal/services/transcription_service.go`
**Commit:** 03b63f9
**Classification:** fixed: requires human verification
**Applied fix:** Changed `pollTingwuStatus` to return `bool` (true on Completed, false on failure). Updated all return paths: context cancelled returns false, Completed returns true, Failed returns false, exhausted attempts returns false. Updated the caller in `processCloudTranscription` to check the return value and abort early with `if !s.pollTingwuStatus(ctx, task, cloudTaskID) { return }`.

### WR-01: Sensitive config fields have json tags that would serialize credentials

**Files modified:** `internal/config/config.go`
**Commit:** 95d9106
**Applied fix:** Changed `json:"access_key_secret"` to `json:"-"` for `OSSConfig.AccessKeySecret`, `json:"password"` to `json:"-"` for `HuaweiConfig.Password`, and `json:"app_secret"` to `json:"-"` for `TingwuConfig.AppSecret`.

### WR-02: Default MinTLSVersion set to 1.0 which is deprecated

**Files modified:** `internal/config/config.go`
**Commit:** a9c8e8b
**Applied fix:** Changed default `MinTLSVersion` from `"1.0"` to `"1.2"` in both the `setDefaults` function and the `createDefaultConfigFile` template.

### WR-03: statusMap keyed by videoFileID prevents concurrent transcription tasks for same video

**Files modified:** `internal/services/transcription_service.go`
**Commit:** f909099
**Classification:** fixed: requires human verification
**Applied fix:** Added a guard at the top of `SubmitTranscriptionWithMode` that checks `statusMap` for an existing entry with `Status == TranscriptionStatusProcessing` for the same `videoFileID`. Returns an error "该视频已有正在进行的转录任务" if a task is already in progress.

### WR-04: console.error calls in production frontend code

**Files modified:** `frontend/src/pages/files/index.tsx`
**Commit:** 145f632
**Applied fix:** Replaced two `console.error` calls in `loadStats` and `checkHasPpt` catch blocks with silent comments, as both failures are already handled gracefully (stats are non-critical, PPT check returns false).

## Verification

- Go build (`go build ./internal/...`): Passes for all modified packages
- Go build (`go build ./internal/services/`): Passes after CR-01, WR-03 fixes
- Go build (`go build ./internal/config/`): Passes after WR-01, WR-02 fixes
- TypeScript check (`npx tsc --noEmit`): Pre-existing errors only in node_modules, no errors in modified file

---

_Fixed: 2026-04-18T20:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
