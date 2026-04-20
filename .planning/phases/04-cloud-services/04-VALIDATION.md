---
phase: 4
slug: cloud-services
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-17
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (go test) + Vitest (frontend) |
| **Config file** | None — existing infrastructure |
| **Quick run command** | `go test ./internal/... -short -count=1` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && npx vitest run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -short -count=1`
- **After every plan wave:** Run `go test ./... -count=1 && cd frontend && npx vitest run`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | OSS-01 | T-04-01 | OSS credentials not logged | unit | `go test ./internal/services/... -run TestOSS` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | OSS-02 | T-04-02 | Presigned URLs expire within 1h | unit | `go test ./internal/services/... -run TestOSS` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | TRAN-01 | T-04-03 | HMAC signature not leaked | unit | `go test ./internal/services/... -run TestTingwu` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | TRAN-02 | — | Mode parameter validated | unit | `go test ./internal/handlers/... -run TestTranscription` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 2 | TRAN-03 | — | Fallback triggers on submit failure | unit | `go test ./internal/services/... -run TestFallback` | ❌ W0 | ⬜ pending |
| 04-04-01 | 04 | 2 | TRAN-05 | — | Timestamps parsed correctly | unit | `go test ./internal/services/... -run TestTextContent` | ❌ W0 | ⬜ pending |
| 04-05-01 | 05 | 2 | UI-02 | — | Dropdown renders both modes | component | `cd frontend && npx vitest run` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/oss_service_test.go` — stubs for OSS-01, OSS-02
- [ ] `internal/services/tingwu_client_test.go` — stubs for TRAN-01 cloud, HMAC signing
- [ ] `internal/services/transcription_service_cloud_test.go` — stubs for cloud pipeline, fallback
- [ ] `internal/handlers/transcription_handler_cloud_test.go` — stubs for mode parameter
- [ ] `frontend/src/components/__tests__/TranscriptionDropdown.test.tsx` — stubs for UI-02
- [ ] `frontend/src/pages/results/__tests__/TextContent.test.tsx` — stubs for TRAN-05

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| OSS upload to real bucket | OSS-01 | Requires live Aliyun credentials | Configure .env, upload test file, verify presigned URL works |
| Tingwu API submission | TRAN-01 | Requires live Tingwu APP_KEY | Submit test transcription, verify task ID returned |
| Cloud status polling | TRAN-04 | Requires live Tingwu processing | Submit task, poll status until complete |
| Timestamp click-to-jump | TRAN-05 | Requires video player interaction | Click timestamp in text content, verify video seeks |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
