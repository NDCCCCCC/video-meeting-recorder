---
phase: 02
slug: local-transcription
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-17
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (go test) |
| **Config file** | none — existing Go test infrastructure |
| **Quick run command** | `go test ./internal/services/... ./internal/handlers/... -count=1 -timeout 30s` |
| **Full suite command** | `go test ./... -count=1 -timeout 120s` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/... ./internal/handlers/... -count=1 -timeout 30s`
- **After every plan wave:** Run `go test ./... -count=1 -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01-01 | 01 | 1 | LCL-01 | — | N/A | unit | `go test ./internal/services/transcription_service_test.go` | ❌ W0 | ⬜ pending |
| 02-01-02 | 01 | 1 | LCL-01 | — | N/A | unit | `go test ./internal/services/frame_extractor_test.go` | ❌ W0 | ⬜ pending |
| 02-02-01 | 02 | 1 | LCL-02 | — | N/A | unit | `go test ./internal/services/similarity_detector_test.go` | ❌ W0 | ⬜ pending |
| 02-02-02 | 02 | 1 | LCL-02 | — | N/A | unit | `go test ./internal/services/similarity_detector_test.go` | ❌ W0 | ⬜ pending |
| 02-02-03 | 02 | 1 | LCL-02 | — | N/A | unit | `go test ./internal/services/similarity_detector_test.go` | ❌ W0 | ⬜ pending |
| 02-03-01 | 03 | 1 | LCL-03 | — | N/A | unit | `go test ./internal/services/pptx_generator_test.go` | ❌ W0 | ⬜ pending |
| 02-04-01 | 04 | 2 | TRAN-01, TRAN-04 | T-02-01 | Input validation on video file ID | unit | `go test ./internal/handlers/transcription_handler_test.go` | ❌ W0 | ⬜ pending |
| 02-04-02 | 04 | 2 | TRAN-06 | — | N/A | unit | `go test ./internal/handlers/transcription_handler_test.go` | ❌ W0 | ⬜ pending |
| 02-05-01 | 05 | 2 | LCL-04, TRAN-04 | — | N/A | integration | frontend build check | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/transcription_service_test.go` — stubs for LCL-01, TRAN-04
- [ ] `internal/services/frame_extractor_test.go` — stubs for LCL-01
- [ ] `internal/services/similarity_detector_test.go` — stubs for LCL-02
- [ ] `internal/services/pptx_generator_test.go` — stubs for LCL-03
- [ ] `internal/handlers/transcription_handler_test.go` — stubs for TRAN-01, TRAN-04, TRAN-06

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real-time progress modal updates correctly during transcription | LCL-04 | Requires browser interaction and timing verification | Start transcription, observe modal progress stages update in sequence |
| "转录" button triggers transcription for both full videos and split segments | TRAN-06 | Requires full-stack interaction with test video file | Click 转录 on full video and on split segment, verify both start transcription |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
