---
phase: 04-cloud-services
reviewed: 2026-04-18T19:30:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - internal/config/config.go
  - internal/services/oss_service.go
  - internal/services/oss_service_test.go
  - internal/services/transcription_service.go
  - internal/services/transcription_service_cloud_test.go
  - frontend/src/components/TextContentTab.tsx
  - frontend/src/pages/files/index.tsx
  - frontend/src/types/transcription.ts
  - frontend/src/types/__tests__/transcription.test.ts
findings:
  critical: 1
  warning: 4
  info: 4
  total: 9
status: issues_found
---

# Phase 4: Code Review Report

**Reviewed:** 2026-04-18T19:30:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Reviewed 9 source files from the cloud transcription feature (Phase 4) -- covering Go backend config, OSS service, transcription service with cloud pipeline, and the React/TypeScript frontend components and types.

One critical bug was found: `processCloudTranscription` does not abort after `pollTingwuStatus` returns following a failure. The polling function calls `handleCloudFailure` (which marks the task as failed) and then returns normally. The caller continues executing as if the task succeeded, overwriting the failed status, attempting to fetch results from a failed Tingwu task, and potentially marking a failed task as completed.

Four warnings were found: sensitive fields (OSS AccessKeySecret, Tingwu AppSecret, Huawei Password) on config structs carry `json` tags that would serialize them if the config is ever marshalled; TLS 1.0 is set as the default minimum version which is deprecated; the `statusMap` is keyed by `videoFileID` preventing concurrent transcription tasks for the same video; and two `console.error` calls remain in production frontend code.

## Critical Issues

### CR-01: processCloudTranscription continues executing after pollTingwuStatus handles a failure

**File:** `internal/services/transcription_service.go:584-628`
**Issue:** The `processCloudTranscription` method calls `s.pollTingwuStatus(ctx, task, cloudTaskID)` on line 584 and then unconditionally continues to line 587 and beyond. However, `pollTingwuStatus` can call `handleCloudFailure` internally (lines 640, 673-674, 684) when it encounters a failure (context cancelled, Tingwu reports Failed status, or max attempts exhausted). `handleCloudFailure` marks the task as failed in both the status map and the database. But `processCloudTranscription` does not check whether the polling succeeded or failed -- it proceeds to:

1. Set progress to "downloading" stage at 90% (line 587), overwriting the failed status
2. Call `s.tingwuClient.GetResult()` on a failed task ID (line 589), which will likely error
3. If `GetResult` somehow succeeds with stale data, it would save text content, mark the task completed at 100%, and fire the cleanup goroutine

This means a Tingwu failure that was correctly detected and handled inside polling gets its status silently overwritten by the downstream code. The task transitions from `failed` back to `completed` with incorrect state.

Additionally, for the initial-stage failures (OSS upload or Tingwu submit), `handleCloudFailure` is called with `isInitialStage=true` and triggers auto-fallback to local mode by calling `s.processTranscription(task)`. After that call returns, `processCloudTranscription` continues executing lines 564-628 (after the upload failure on line 560, the return on line 561 correctly exits; but for Tingwu submit failure on line 576, line 577 also correctly exits). The initial-stage paths are safe due to the immediate `return` statements. The critical issue is specifically with `pollTingwuStatus` where no return follows.

**Fix:**
Make `pollTingwuStatus` return a boolean indicating success, or check the task status after polling returns:

```go
// Option A: Return a boolean from pollTingwuStatus
func (s *TranscriptionService) pollTingwuStatus(...) bool {
    // ... existing logic ...
    // On failure paths: return false
    // On "Completed": return true
}

// Then in processCloudTranscription:
if !s.pollTingwuStatus(ctx, task, cloudTaskID) {
    return // Failure already handled by handleCloudFailure
}

// Option B: Check task status after polling
s.pollTingwuStatus(ctx, task, cloudTaskID)
s.statusMu.RLock()
progress := s.statusMap[task.VideoFileID]
failed := progress != nil && progress.Status == models.TranscriptionStatusFailed
s.statusMu.RUnlock()
if failed {
    return
}
```

## Warnings

### WR-01: Sensitive config fields have json tags that would serialize credentials

**File:** `internal/config/config.go:142-143`
**File:** `internal/config/config.go:96`
**File:** `internal/config/config.go:152`
**Issue:** The `OSSConfig.AccessKeySecret`, `HuaweiConfig.Password`, and `TingwuConfig.AppSecret` fields have `json:"access_key_secret"`, `json:"password"`, and `json:"app_secret"` tags respectively. If the `Config` struct is ever serialized to JSON (e.g., for a config dump endpoint, debug logging, or error response), these credentials will be exposed in plaintext. While no such serialization currently exists in the reviewed code, the tags make it a latent risk.

**Fix:** Add `json:"-"` to sensitive fields to prevent accidental serialization:
```go
type OSSConfig struct {
    // ...
    AccessKeySecret string `mapstructure:"access_key_secret" json:"-" yaml:"access_key_secret"`
    // ...
}
```
Apply the same pattern to `HuaweiConfig.Password` and `TingwuConfig.AppSecret`.

---

### WR-02: Default MinTLSVersion set to 1.0 which is deprecated

**File:** `internal/config/config.go:442-443`
**Issue:** The default value for `HuaweiConfig.MinTLSVersion` is set to `"1.0"`. TLS 1.0 and 1.1 have been formally deprecated by RFC 8996 and are disabled in most modern systems. Using TLS 1.0 exposes the connection to known vulnerabilities (BEAST, POODLE, etc.).

**Fix:**
```go
if cfg.Huawei.MinTLSVersion == "" {
    cfg.Huawei.MinTLSVersion = "1.2"
}
```

---

### WR-03: statusMap keyed by videoFileID prevents concurrent transcription tasks for same video

**File:** `internal/services/transcription_service.go:158-164`
**Issue:** The `statusMap` is keyed by `videoFileID` (a `map[uint]*TranscriptionProgress`). If two transcription tasks are submitted for the same video file (e.g., one local and one cloud), the second submission's progress entry will overwrite the first one's status at line 159. The first task will continue running but its progress updates will be lost or interleaved with the second task's updates. This also means `GetTranscriptionStatus` can only return progress for one task per video.

**Fix:** Key the status map by task ID instead of video file ID, or add a guard to prevent submitting a new task when one is already in progress for the same video:
```go
// Option: Check for existing active task before allowing new submission
s.statusMu.RLock()
_, exists := s.statusMap[videoFileID]
s.statusMu.RUnlock()
if exists {
    return fmt.Errorf("该视频已有正在进行的转录任务")
}
```

---

### WR-04: console.error calls in production frontend code

**File:** `frontend/src/pages/files/index.tsx:146`
**File:** `frontend/src/pages/files/index.tsx:166`
**Issue:** Two `console.error` calls remain in the file management page (`loadStats` and `checkHasPpt` functions). While not a bug, these can expose internal error details to anyone inspecting the browser console.

**Fix:** Consider using a centralized logging utility that can be disabled in production builds, or remove these calls.

## Info

### IN-01: Hardcoded default SM4Secret is insecure but intentional for initial setup

**File:** `internal/config/config.go:309-311`
**Issue:** The default `SM4Secret` is `"change-me-in-production"`. While this is standard for initial setup, the hardcoded value combined with the generated `config.yaml` (line 511) containing `"change-me-in-production-please-set-a-secure-random-key"` means any deployment that does not change this value is insecure. The default config file template at line 511 and the `setDefaults` function at line 310 use different default strings.

**Fix:** Use the same default string in both places and consider failing to start if the secret is unchanged in production mode.

---

### IN-02: Empty error message when result_ppt_file_id is nil for cloud transcription

**File:** `internal/services/transcription_service.go:622-623`
**Issue:** Cloud transcription completion calls `updateProgress` and `updateTaskStatus` with `nil` for `resultPPTFileID`. The `TranscriptionProgressModal` on the frontend checks `data.status === 'completed' && data.result_ppt_file_id` to determine completion, so cloud transcription will never trigger the normal completion path in the modal. The frontend needs a separate completion handling path for cloud mode (which produces text, not PPT).

**Fix:** The frontend `TranscriptionProgressModal` should check for cloud mode completion as `data.status === 'completed'` without requiring `result_ppt_file_id` when `data.mode === 'cloud'`.

---

### IN-03: Tingwu signature includes appKey in Authorization header

**File:** `internal/services/tingwu_client.go:225`
**Issue:** The Authorization header is set to `fmt.Sprintf("acs %s:%s", c.appKey, signature)`, which puts the appKey directly in the header. This is the correct Aliyun ROA specification format, but if HTTP request logging is enabled at a debug level, the appKey will appear in logs.

**Fix:** Ensure request logging redacts the Authorization header, or log at a level that does not include full headers.

---

### IN-04: Type-level test file uses void suppression pattern

**File:** `frontend/src/types/__tests__/transcription.test.ts:77-80`
**Issue:** The test file validates types using compile-time checks with `void` statements to suppress unused variable warnings. This is a valid pattern for type-level testing but the file is not executed by any test runner -- it only runs through `tsc --noEmit`. Consider adding a comment or build step that makes this explicit.

**Fix:** No action required -- the file header comment on line 4 already explains this is validated by `tsc --noEmit`.

---

_Reviewed: 2026-04-18T19:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
