---
phase: 8
slug: video-snapshot-player-enhancement
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-20
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify (backend), React Testing Library (frontend) |
| **Config file** | No specific config — uses standard Go test conventions |
| **Quick run command** | `go test ./internal/services/... -run TestSnapshot && npm test -- --testPathPattern=VideoPlayer` |
| **Full suite command** | `go test ./... && npm test` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/services/... -run TestSnapshot && npm test -- --testPathPattern=VideoPlayer`
- **After every plan wave:** Run `go test ./... && npm test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 08-01-01 | 01 | 1 | SNAPSHOT-01 | T-8-01 | Mutex prevents concurrent snapshots | unit | `go test ./internal/services/... -run TestGenerateSnapshot_Concurrent` | ❌ W0 | ⬜ pending |
| 08-01-02 | 01 | 1 | SNAPSHOT-02 | T-8-02 | Filenames sanitized, no path traversal | unit | `go test ./internal/services/... -run TestGenerateSnapshot_Naming` | ❌ W0 | ⬜ pending |
| 08-02-01 | 02 | 1 | PLAYER-02 | — | Keyboard shortcuts prevent default | unit | `npm test -- useKeyboardShortcuts.test` | ❌ W0 | ⬜ pending |
| 08-02-02 | 02 | 1 | PLAYER-03 | — | Playback rate within valid range | unit | `npm test -- VideoPlayerModal.speed` | ✅ exists | ⬜ pending |
| 08-03-01 | 03 | 2 | PLAYER-01 | — | Frame seeking respects video boundaries | integration | `npm test -- VideoPlayerModal.frame` | ❌ W0 | ⬜ pending |
| 08-03-02 | 03 | 2 | EDGE-02 | T-8-04 | Recording interruption handled gracefully | integration | `go test ./internal/services/... -run TestGenerateSnapshot_Interrupted` | ❌ W0 | ⬜ pending |
| 08-04-01 | 04 | 2 | EDGE-01 | T-8-01 | Time range validation prevents invalid snapshots | integration | `go test ./internal/services/... -run TestGenerateSnapshot_TimeRange` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/snapshot_service_test.go` — Test for concurrent snapshot mutex protection (T-8-01)
- [ ] `internal/services/snapshot_service_test.go` — Test for enhanced naming convention with sanitization (T-8-02)
- [ ] `internal/services/snapshot_service_test.go` — Test for time range validation (EDGE-01, EDGE-02)
- [ ] `frontend/src/hooks/__tests__/useKeyboardShortcuts.test.ts` — Test for keyboard shortcuts hook (PLAYER-02)
- [ ] `frontend/src/components/__tests__/VideoPlayerModal.test.tsx` — Test for frame navigation (PLAYER-01)
- [ ] Frontend test setup: Verify `@testing-library/react` and `@testing-library/user-event` are installed

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Keyboard shortcut visual feedback | PLAYER-02 | User experience requires visual confirmation | 1. Open video player modal 2. Press Space key 3. Verify toast message appears showing "播放" or "暂停" |
| Frame navigation browser compatibility | PLAYER-01 | Different browsers have varying API support | 1. Test in Chrome/Edge 2. Test in Firefox/Safari 3. Verify fallback behavior for unsupported browsers |
| Snapshot filename readability | SNAPSHOT-02 | Subjective visual assessment | 1. Generate snapshot 2. Verify filename includes task name and sequence 3. Confirm filename is recognizable in file explorer |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
