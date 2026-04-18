---
phase: 04-cloud-services
plan: 05
subsystem: cloud-storage
tags: [oss, aliyun, sdk-v2, file-upload, presigned-url]
dependency_graph:
  requires: []
  provides: [OSS-01, OSS-02]
  affects: [cloud-transcription-pipeline]
tech_stack:
  added:
    - "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
    - "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
  patterns:
    - Static credentials provider from OSS SDK's own credentials package
    - Builder pattern for OSS config (NewConfig().WithEndpoint().WithCredentialsProvider())
    - Presigned URL generation with TTL
    - Lifecycle rule management via PutBucketLifecycle API
key_files:
  created: []
  modified:
    - internal/services/oss_service.go
    - internal/services/oss_service_test.go
decisions: []
metrics:
  duration_seconds: 120
  completed_date: 2026-04-18
---

# Phase 04 Plan 05: Real OSS SDK v2 Integration Summary

Replaced OSSService stub implementation with real Aliyun OSS SDK v2 integration, unblocking the cloud transcription pipeline.

## One-Liner

Real OSS SDK v2 integration using alibabacloud-oss-go-sdk-v2 with StaticCredentialsProvider, enabling file upload, presigned URL generation, lifecycle rules, and object deletion.

## Changes Made

### Task 1: Replace OSSService stub with real OSS SDK v2 implementation
**Commit:** `dd525ab`

Replaced the entire OSSService implementation with real OSS SDK v2 operations:

1. **Fixed credentials compatibility issue** - The previous stub was caused by using the standalone `github.com/aliyun/credentials-go` package which has an incompatible interface. The fix uses `oss/credentials.NewStaticCredentialsProvider` from the OSS SDK v2's own credentials package.

2. **Real client initialization** - NewOSSService now creates a real `oss.Client` using:
   ```go
   provider := credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.AccessKeySecret)
   ossCfg := oss.NewConfig().WithEndpoint(cfg.Endpoint).WithCredentialsProvider(provider)
   client := oss.NewClient(ossCfg)
   ```

3. **UploadFile implementation** - Opens local file, uploads via `PutObject`, generates presigned URL via `Presign()` with GetObjectRequest and configurable TTL.

4. **SetLifecycleRule implementation** - Calls `PutBucketLifecycle` with real lifecycle rule configuration (ID, Prefix, Status, Expiration/Days).

5. **DeleteFile implementation** - Calls `DeleteObject` to remove objects from OSS.

6. **IsStub() implementation** - Returns `s.client == nil` (false when properly initialized with valid credentials).

### Task 2: Update OSS service tests for real SDK integration
**Commit:** `d050a58`

Updated test file to verify the new implementation:

1. **TestOSSServiceIsStubWhenDisabled** - Verifies IsStub() returns true when service is disabled.

2. **TestOSSServiceEnabledWithValidConfig** - Verifies IsStub() returns false when service is enabled with valid credentials.

3. **TestOSSServiceEnabledRejectsEmptyCredentials** - Validates that empty AccessKeyID or AccessKeySecret causes initialization to fail.

4. **TestOSSServiceUploadFileValidatesInputs** - Tests input validation for UploadFile (empty localPath/objectKey).

5. All existing tests continue to pass.

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written.

## Verification Results

All verification criteria passed:

1. ✅ `go build ./internal/services/...` - zero errors
2. ✅ `go test ./internal/services/... -run TestOSSService -v` - all 10 tests pass
3. ✅ `grep -c "oss.NewClient" internal/services/oss_service.go` - returns 1 (real SDK client is created)
4. ✅ `grep -c "存根" internal/services/oss_service.go` - returns 0 (no stub language remains)
5. ✅ `grep "IsStub" internal/services/oss_service.go` - returns `return s.client == nil` (not hardcoded true)

## Known Stubs

None - all stub methods replaced with real OSS SDK v2 implementations.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: credential_exposure | internal/services/oss_service.go | OSS credentials (AccessKeyID/Secret) used for authentication - stored in config, never logged |
| threat_flag: presigned_url_abuse | internal/services/oss_service.go | Presigned URLs expose temporary OSS access - 24h TTL by default |
| threat_flag: file_orphaning | internal/services/oss_service.go | OSS file orphaning risk - lifecycle rules configured, manual deletion available |

All threat mitigations from the plan's threat_model are implemented:
- **T-04-01 (Disclosure)**: Credentials stored in .env with env var expansion, never logged
- **T-04-02 (Spoofing)**: 24h TTL on presigned URLs
- **T-04-03 (Denial of Service)**: Lifecycle rules auto-delete after configurable days
- **T-04-04 (Tampering)**: OSS SDK v2 handles checksum validation internally

## Self-Check: PASSED

### Created Files
- N/A (no files created, only modified)

### Modified Files
- ✅ internal/services/oss_service.go - EXISTS
- ✅ internal/services/oss_service_test.go - EXISTS

### Commits
- ✅ dd525ab - feat(04-05): replace OSS stub with real SDK v2 implementation
- ✅ d050a58 - test(04-05): add tests for real OSS SDK integration

## Success Criteria

1. ✅ OSSService.UploadFile opens a local file, uploads to OSS via PutObject, returns a presigned URL from Presign()
2. ✅ OSSService.SetLifecycleRule calls PutBucketLifecycle with real lifecycle rule
3. ✅ OSSService.DeleteFile calls DeleteObject
4. ✅ OSSService.IsStub() returns false when client is initialized with valid config
5. ✅ All existing unit tests pass (disabled service tests, config validation tests)
6. ✅ No "TODO" or "stub" comments remain in oss_service.go related to SDK compatibility
7. ✅ Cloud transcription pipeline is unblocked -- processCloudTranscription will now receive real presigned URLs

## User Setup Required

Before using cloud transcription, users must configure these environment variables in `.env` or `config.yaml`:

```bash
ALIYUN_OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
ALIYUN_OSS_BUCKET=your-bucket-name
ALIYUN_OSS_ACCESS_KEY_ID=your-access-key-id
ALIYUN_OSS_ACCESS_KEY_SECRET=your-access-key-secret
```

Source: Aliyun OSS Console and RAM Console. See plan frontmatter for detailed setup instructions.
