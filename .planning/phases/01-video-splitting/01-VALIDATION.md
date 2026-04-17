---
phase: 01
slug: video-splitting
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-17
---

# Phase 01 -- Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (go test) + testify assertions (existing) |
| **Config file** | go.mod (existing) |
| **Quick run command** | `go test ./... -short` |
| **Full suite command** | `go test ./... -v` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./... -short`
- **After every plan wave:** Run `go test ./... -v`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-00-01 | 00 | 0 | SPLIT-03 | T-01-00-01 | N/A | stub | `go test ./internal/services/... -run TestSplitting -v` | W0 creates | pending |
| 01-00-02 | 00 | 0 | SNAP-02 | T-01-00-01 | N/A | stub | `go test ./internal/services/... -run TestSnapshot -v` | W0 creates | pending |
| 01-00-01b | 00 | 0 | SCAN-01 | T-01-00-02 | N/A | stub | `go test ./internal/services/... -run TestCreateSegmentFile -v` | W0 extends | pending |
| 01-01-01 | 01 | 1 | SPLIT-04 | T-01-01-01 | N/A | unit | `go build ./internal/...` | Y | pending |
| 01-01-02 | 01 | 1 | SCAN-01 | T-01-01-02 | N/A | unit | `go build ./internal/...` | Y | pending |
| 01-02-01 | 02 | 1 | SPLIT-03 | T-01-02-01 | N/A | integration | `go test ./internal/services/... -run TestSplitService -v` | Y (W0 stub) | pending |
| 01-02-02 | 02 | 1 | SCAN-01 | T-01-02-02 | N/A | integration | `go test ./internal/handlers/... -run TestSplitHandler -v` | Y (W0 stub) | pending |
| 01-03-01 | 03 | 2 | SPLIT-01 | -- | N/A | unit | `npx vitest run TimelineWithMarkers` | N/A (frontend) | pending |
| 01-03-02 | 03 | 2 | SPLIT-02 | -- | N/A | unit | `npx vitest run VideoPlayerModal` | N/A (frontend) | pending |
| 01-03-03 | 03 | 2 | UI-01 | -- | N/A | E2E | Manual-only (visual) | N/A | pending |
| 01-04-01 | 04 | 2 | SNAP-01 | -- | N/A | E2E | Manual-only (UI visibility) | N/A | pending |
| 01-04-02 | 04 | 2 | SCAN-02 | -- | N/A | E2E | Manual-only (UI refresh) | N/A | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [x] `internal/services/splitting_service_test.go` -- stubs for SPLIT-03 (5 tests)
- [x] `internal/services/snapshot_service_test.go` -- stubs for SNAP-02 (3 tests)
- [x] `internal/handlers/split_handler_test.go` -- stubs for SPLIT-04 (6 tests)
- [x] `internal/services/video_file_service_test.go` -- extends for SCAN-01 auto-scan callbacks (6 tests)
- [x] Test framework setup: Go testing + testify already in go.mod
- [x] Integration test helpers: Reuse setupTestDB pattern from existing video_file_service_test.go

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| User can mark split points on timeline | SPLIT-01 | Requires browser video playback interaction | Open split page, play video, click timeline to add markers, verify markers appear |
| User can preview and locate split points | SPLIT-02 | Requires video playback and seek | Open split page, use seek controls, verify time display matches seek position |
| Snapshot button appears on active recording | SNAP-01 | Requires active recording state | Start recording, verify "生成MP4快照" button appears on task row |
| File list updates in real-time | SCAN-02 | Requires live UI refresh observation | Generate MP4 file, verify file list updates within 5 seconds |
| Split page layout renders correctly | UI-01 | Visual layout verification | Navigate to split page, verify player + timeline + segment list layout |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter
- [x] `wave_0_complete: true` set in frontmatter

**Approval:** pending
