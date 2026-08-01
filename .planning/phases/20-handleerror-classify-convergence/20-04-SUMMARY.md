---
phase: 20-handleerror-classify-convergence
plan: 04
subsystem: error-handling
tags: [logger, sentinel, structured-logging, REQ-20b-upgrade]
dependency_graph:
  requires: [20-01, 20-02]
  provides: [service-layer-sentinel-coverage, scheduler-sentinel-coverage, auth-sentinel-coverage, huawei-sentinel-coverage]
  affects: [internal/services, internal/auth, internal/scheduler, internal/huawei]
tech-stack:
  added: []
  patterns: [zap.Error + response.SentinelField sidecar pattern]
key-files:
  created: []
  modified:
    - internal/services/video_file_service.go
    - internal/services/transcription_service.go
    - internal/services/video_recording_task_service.go
    - internal/services/usb_device_scanner.go
    - internal/services/dashboard_service.go
    - internal/services/conversion_service.go
    - internal/services/splitting_service.go
    - internal/services/config_service.go
    - internal/services/frame_extractor.go
    - internal/services/input_config_service.go
    - internal/services/ppt_merge_service.go
    - internal/services/python_deps.go
    - internal/services/storage/file_service.go
    - internal/auth/ad_auth.go
    - internal/auth/local_auth.go
    - internal/auth/sm4_token.go
    - internal/scheduler/video_scheduler.go
    - internal/huawei/manager.go
    - internal/huawei/client.go
decisions:
  - "Treated ppt_editor_service.go (11 sites) as pre-existing — already had response.SentinelField applied in 20-02 — and made no changes per plan's atomic-per-file discipline"
  - "Used multi-call Edit pattern (surrounding context for uniqueness) instead of replace_all where pattern was non-unique, to preserve exactly the planned scope"
  - "Respect D-03.7 boundary — internal/middleware/error_mapper.go NOT touched (its 2 sites deferred to next phase per locked user decision)"
metrics:
  duration: "~16m"
  completed: 2026-08-01
  sites-upgraded: 149
  commits: 16
  files-modified: 19
  tests-pass: true
---

# Phase 20 Plan 04: Service-Layer zap.Error SentinelField Upgrade — Summary

Zero-intrusion upgrade of 149 `zap.Error(err)` call-sites across 19 service/auth/scheduler/huawei files to also emit `response.SentinelField(err)`. Combined with ppt_editor_service.go (11 sites already done in 20-02), this completes REQ-20b by extending structured sentinel_type logging from the handler layer into the service layer where most error origination happens.

## What Was Built

Mechanical upgrade of `zap.Error(err)` → `zap.Error(err), response.SentinelField(err)` across 19 files. Each file gained a `pkg/response` import (where missing). No behavior changes — purely structured log enrichment that adds a `sentinel_type` field to existing error logs.

**Per-package counts:**

| Package / File | Sites |
|---|---|
| **Task 1a — top service files:** | |
| internal/services/video_file_service.go | 22 |
| internal/services/transcription_service.go | 20 |
| internal/services/video_recording_task_service.go | 9 |
| **Task 1b — remaining service files:** | |
| internal/services/conversion_service.go | 7 |
| internal/services/usb_device_scanner.go | 6 |
| internal/services/dashboard_service.go | 5 |
| internal/services/splitting_service.go | 5 |
| internal/services/frame_extractor.go | 4 |
| internal/services/config_service.go | 3 |
| internal/services/input_config_service.go | 3 |
| internal/services/ppt_merge_service.go | 3 |
| internal/services/python_deps.go | 3 |
| internal/services/storage/file_service.go | 3 |
| **Task 2 — auth/scheduler/huawei:** | |
| internal/auth/ad_auth.go | 11 |
| internal/auth/local_auth.go | 6 |
| internal/auth/sm4_token.go | 4 |
| internal/scheduler/video_scheduler.go | 28 |
| internal/huawei/manager.go | 5 |
| internal/huawei/client.go | 2 |
| **Subtotal (this plan)** | **149** |
| ppt_editor_service.go (pre-existing from 20-02) | 11 |
| **REQ-20b-upgrade TOTAL (service portion)** | **160** |

## Deviations from Plan

### Auto-fixed Issues

None — plan executed as written. No bugs, no missing functionality, no blocking issues encountered.

### Plan-scope Observations

**Pre-existing work in ppt_editor_service.go (11 sites)** — This file was already 100% upgraded with `response.SentinelField(err)` at all 11 zap.Error sites (verified via `grep -c` before this plan started). The plan listed it in Task 1a scope, but no edit was required because the work was already done (likely during 20-02 alongside ppt_file_service.go). The plan's atomic-per-file discipline was respected: I made no commit touching ppt_editor_service.go. Documented here as a plan-scope discrepancy, not a deviation.

**Files OUTSIDE plan scope with remaining bare `zap.Error(err)` sites (intentionally NOT touched per plan boundaries):**

| File | Sites | Reason NOT touched |
|---|---|---|
| internal/services/apikey_service.go | 1 | Not in 20-04 files_modified list |
| internal/services/audit/audit_log_service.go | 1 | Not in 20-04 files_modified list |
| internal/services/frame_capture_service.go | 1 | Not in 20-04 files_modified list |
| internal/services/pptx_generator.go | 2 | Not in 20-04 files_modified list |
| internal/services/similarity_detector.go | 1 | Not in 20-04 files_modified list |
| internal/services/slide_cache_service.go | 2 | Not in 20-04 files_modified list |
| internal/services/slide_extractor.go | 2 | Not in 20-04 files_modified list |
| internal/services/snapshot_service.go | 2 | Not in 20-04 files_modified list |
| internal/services/video_recording/huawei_conference_connector.go | 2 | Not in 20-04 files_modified list |
| internal/auth/hlstoken/hls_token.go | 1 | Not in 20-04 files_modified list (subdirectory) |
| internal/auth/service.go | 1 | Not in 20-04 files_modified list |

These 17 sites remain without SentinelField. They were not within plan scope and require either a future phase or a scope-extension decision. Deferred to next phase per the user's locked D-03.7 pattern of incremental scope.

### D-03.7 Compliance

`internal/middleware/error_mapper.go` was NOT touched. `git diff --stat c838abc..HEAD -- internal/middleware/` returns empty. The 2 middleware zap.Error sites remain without SentinelField this phase, per the locked user decision: "不动 error_mapper.go，Phase 19 已带开，本阶段不扩展".

## Auth Gates

None — no authentication gates triggered during execution. All work was mechanical file edits with build/test verification.

## Verification Results

| Check | Result |
|---|---|
| `go build ./...` | PASS (no errors) |
| `go vet ./...` | PASS (no warnings) |
| `go test -race ./internal/services/ ./internal/auth/ ./internal/scheduler/ ./internal/huawei/ -count=1` | PASS (all 4 packages green) |
| D-03.7 middleware boundary | PASS (error_mapper.go untouched) |
| Per-file SentinelField counts | Match plan estimates exactly |

## Atomic Commits (16 total)

```
db7d60f feat(20-logger): add response.SentinelField to huawei package zap.Error call-sites
648db4a feat(20-logger): add response.SentinelField to video_scheduler.go zap.Error call-sites
abf05da feat(20-logger): add response.SentinelField to auth package zap.Error call-sites
381b5a2 feat(20-logger): add response.SentinelField to storage/file_service.go zap.Error call-sites
59380ba feat(20-logger): add response.SentinelField to python_deps.go zap.Error call-sites
066a7e3 feat(20-logger): add response.SentinelField to ppt_merge_service.go zap.Error call-sites
8d96db0 feat(20-logger): add response.SentinelField to input_config_service.go zap.Error call-sites
32d9a85 feat(20-logger): add response.SentinelField to config_service.go zap.Error call-sites
f520308 feat(20-logger): add response.SentinelField to frame_extractor.go zap.Error call-sites
cb14ac8 feat(20-logger): add response.SentinelField to splitting_service.go zap.Error call-sites
26b3d93 feat(20-logger): add response.SentinelField to dashboard_service.go zap.Error call-sites
8953fa6 feat(20-logger): add response.SentinelField to usb_device_scanner.go zap.Error call-sites
d25cb2c feat(20-logger): add response.SentinelField to conversion_service.go zap.Error call-sites
458ccae feat(20-logger): add response.SentinelField to video_recording_task_service.go zap.Error call-sites
eeb8ecf feat(20-logger): add response.SentinelField to transcription_service.go zap.Error call-sites
6234103 feat(20-logger): add response.SentinelField to video_file_service.go zap.Error call-sites
```

## Success Criteria Status

- [x] 149 service/auth/scheduler/huawei zap.Error sites enriched with SentinelField (target ~160 incl. ppt_editor_service.go pre-existing)
- [x] Zero behavior change across all touched files (pure log enrichment)
- [x] D-03.7 boundary on error_mapper.go respected (middleware untouched, 2 sites deferred)
- [x] All existing tests pass with `-race`
- [x] REQ-20b-upgrade (service portion) complete; combined with 20-02/20-03 handler portion, full REQ-20b delivered

## Known Stubs

None — all upgrades are wired with the actual `response.SentinelField(err)` helper.

## Threat Flags

None introduced — no new security surface; plan only adds structured log tags without altering err.Error() content or call patterns. The threat model from the plan (T-20-01-log-leak, T-20-03-adhoc-visibility, T-20-middleware-boundary, T-20-SC-supply-chain) remains as documented.

## Self-Check: PASSED

- All 19 modified files exist and contain `response.SentinelField` references matching plan counts
- All 16 commit hashes verified in git log
- Middleware directory has zero diff from base c838abc (D-03.7 confirmed)
- `go build ./...`, `go vet ./...`, and `go test -race` all exit 0

---

*Phase 20-04 complete: 2026-08-01*