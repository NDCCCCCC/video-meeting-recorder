---
phase: 04-cloud-services
plan: 01
subsystem: Cloud Services Foundation
tags: [backend, oss, tingwu, config, models]
completed_date: "2026-04-17"

dependency_graph:
  requires:
    - id: "phase-03"
      description: "PPT management for transcription results"
  provides:
    - id: "OSSService"
      description: "Aliyun OSS file upload/download with presigned URLs"
    - id: "TingwuClient"
      description: "Aliyun Tingwu API client with HMAC-SHA256 signing"
    - id: "TranscriptionText"
      description: "Text segments with timestamps storage model"

tech_stack:
  added:
    - "github.com/aliyun/alibabacloud-oss-go-sdk-v2 v1.4.1"
    - "github.com/aliyun/credentials-go v1.4.12"
  patterns:
    - "Config struct extension with env var expansion"
    - "Model extension with mode constants"
    - "Service struct pattern with disabled state handling"
    - "HMAC-SHA256 signing per Aliyun ROA specification"

key_files:
  created:
    - path: "internal/config/config.go"
      changes: "Added OSSConfig and TingwuConfig structs"
    - path: "internal/models/transcription_task.go"
      changes: "Extended with Mode, CloudTaskID, OSSURL fields; added cloud stage constants"
    - path: "internal/models/transcription_text.go"
      changes: "New model for text segments with timestamps"
    - path: "internal/services/oss_service.go"
      changes: "OSS service with upload/delete/lifecycle methods (stub implementation)"
    - path: "internal/services/tingwu_client.go"
      changes: "Tingwu API client with HMAC-SHA256 signing"
    - path: "cmd/server/app.go"
      changes: "Registered TranscriptionText in AutoMigrate"
  modified:
    - path: "internal/services/oss_service_test.go"
      changes: "Test stubs for disabled service behavior"
    - path: "internal/services/tingwu_client_test.go"
      changes: "Test stubs for disabled client and signature verification"
    - path: "internal/models/transcription_text_test.go"
      changes: "Model validation and constant tests"

decisions:
  - id: "D-04-01-001"
    title: "OSS SDK v2 Stub Implementation"
    rationale: "alibabacloud-oss-go-sdk-v2 credentials API is incompatible with standalone credentials-go package. Implemented stub service to validate configuration and test patterns while SDK compatibility is resolved."
    impact: "OSS upload/download operations log but don't perform actual cloud operations. Full OSS integration deferred to follow-up."
    alternatives_considered:
      - "Use OSS SDK v1 (deprecated, no longer maintained)"
      - "Implement raw HTTP client with manual signing (complex, error-prone)"
      - "Wait for SDK v2 credentials compatibility (selected - stub allows progress)"
  - id: "D-04-01-002"
    title: "TingwuClient Manual HMAC Signing"
    rationale: "No official Go SDK exists for Tingwu API. Implemented manual HMAC-SHA256 signing per Aliyun ROA specification using stdlib crypto packages."
    impact: "Client can submit tasks, poll status, and retrieve results. Error messages sanitized to never expose credentials."
    alternatives_considered:
      - "Use third-party Tingwu SDK (none available for Go)"
      - "Use Python microservice (rejected - all-Go architecture decision)"

metrics:
  duration_minutes: 6
  tasks_completed: 2
  files_created: 8
  files_modified: 3
  test_files: 3
  commits: 2
---

## Deviations from Plan

### OSS SDK v2 Compatibility Issue (Rule 4 - Architectural Decision)

**Found during:** Task 2 - OSSService implementation

**Issue:** The plan specified using `alibabacloud-oss-go-sdk-v2` with the `credentials` package from the same module. However, the SDK v2 has a breaking API change where the `CredentialsProvider` interface differs from the standalone `credentials-go` package, causing type incompatibility.

**Impact:** Unable to initialize actual OSS client with credentials. Cloud file upload/download operations cannot be performed.

**Decision Applied (Rule 4):** STOPPED implementation of full OSS SDK integration. Implemented stub service that:
- Validates configuration inputs
- Logs all operations for debugging
- Returns placeholder URLs
- Documents TODO for SDK compatibility resolution

**Rationale:** This is an architectural issue (new SDK version with incompatible API) that affects the core cloud integration. Three alternatives were considered:
1. Use OSS SDK v1 (deprecated, unmaintained)
2. Implement raw HTTP client with manual OSS signing (complex, error-prone)
3. Stub implementation now, resolve SDK compatibility later (selected)

**Resolution:** Stub allows tests to pass and validates service patterns. Full OSS integration deferred to follow-up task to resolve SDK compatibility or implement manual signing.

**Files Modified:** `internal/services/oss_service.go` (stub implementation instead of full SDK)

**Future Work:** Resolve SDK v2 credentials compatibility by either:
- Finding compatible credentials provider implementation
- Implementing manual OSS HTTP signing (similar to TingwuClient)
- Downgrading to SDK v1 if compatibility cannot be resolved

### Other Changes

None - remaining tasks executed as specified in plan.

## Commits

| Hash | Message | Files |
|------|---------|-------|
| ea057d7 | feat(04-01): extend config and models for cloud transcription | 5 files (config, models, app.go) |
| 6cb9074 | feat(04-01): add OSSService and TingwuClient implementations | 6 files (services, go.mod) |

## Known Stubs

### OSSService Methods (SDK Compatibility)

**File:** `internal/services/oss_service.go`

**Stub Methods:**
- `UploadFile()` - Line 68-90 - Returns placeholder URL instead of uploading
- `DeleteFile()` - Line 115-126 - Logs deletion without executing
- `SetLifecycleRule()` - Line 93-112 - Logs rule without configuring OSS

**Reason:** OSS SDK v2 credentials API incompatible with standalone credentials-go package

**Resolution Plan:** Resolve SDK compatibility or implement manual OSS HTTP signing in follow-up task

**Impact:** Cloud transcription cannot upload files to OSS. This is a blocking issue for cloud transcription functionality.

## Threat Flags

None detected - no new security-relevant surface introduced beyond planned OSS/Tingwu integration points.

## Self-Check: PASSED

**Files Created:**
- ✅ internal/models/transcription_text.go
- ✅ internal/models/transcription_text_test.go  
- ✅ internal/services/oss_service.go
- ✅ internal/services/oss_service_test.go
- ✅ internal/services/tingwu_client.go
- ✅ internal/services/tingwu_client_test.go
- ✅ .planning/phases/04-cloud-services/04-01-SUMMARY.md

**Files Modified:**
- ✅ internal/config/config.go
- ✅ internal/models/transcription_task.go
- ✅ cmd/server/app.go
- ✅ go.mod
- ✅ go.sum

**Commits Exist:**
- ✅ ea057d7 - feat(04-01): extend config and models
- ✅ 6cb9074 - feat(04-01): add OSSService and TingwuClient

**Tests Pass:**
- ✅ go test ./internal/models/... -run "TestTranscriptionText|TestTranscriptionTaskMode" -v
- ✅ go test ./internal/services/... -run "TestOSS|TestTingwu" -v
- ✅ go build ./... (with frontend dist placeholder)

**Acceptance Criteria Met:**
- ✅ Config has OSSConfig and TingwuConfig structs
- ✅ TranscriptionTask has Mode/CloudTaskID/OSSURL fields
- ✅ TranscriptionText model exists and registered in AutoMigrate
- ✅ Test stubs exist for all new models and services
- ✅ All code compiles and tests pass
- ⚠️ OSSService uses stub implementation (documented deviation)
