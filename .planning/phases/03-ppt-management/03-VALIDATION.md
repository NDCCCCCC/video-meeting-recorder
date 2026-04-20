---
phase: 3
slug: ppt-management
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-17
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + React Testing Library / Vitest |
| **Config file** | go.test (inline) + vitest.config.ts |
| **Quick run command** | `go test ./internal/... -count=1 -short` |
| **Full suite command** | `go test ./... -count=1 && cd frontend && npx vitest run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -count=1 -short`
- **After every plan wave:** Run `go test ./... -count=1 && cd frontend && npx vitest run`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | PPT-03 | — | N/A | unit | `go test ./internal/services/slide_extractor_test.go` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | PPT-03 | T-03-01 | Path traversal prevention on slide image API | unit | `go test ./internal/handlers/ppt_handler_test.go` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 1 | UI-03 | — | N/A | component | `cd frontend && npx vitest run src/pages/results` | ❌ W0 | ⬜ pending |
| 03-03-01 | 03 | 2 | PPT-04, PPT-05 | — | N/A | unit | `go test ./internal/handlers/ppt_handler_test.go` | ❌ W0 | ⬜ pending |
| 03-04-01 | 04 | 2 | PPT-06 | T-03-02 | Merge limit enforced server-side | unit | `go test ./internal/services/merge_service_test.go` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/services/slide_extractor_test.go` — stubs for PPT-03
- [ ] `internal/handlers/ppt_handler_test.go` — stubs for PPT-01 through PPT-06
- [ ] `internal/services/merge_service_test.go` — stubs for PPT-06
- [ ] `frontend/src/pages/results/__tests__/` — component test stubs for UI-03

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| PPT preview layout matches PowerPoint reading view (main + sidebar) | UI-03 | Visual layout validation | Open result page, verify main view + thumbnail sidebar arrangement |
| Merge drag-to-reorder interaction | PPT-06 | Drag interaction is complex to automate | Select slides from multiple results, drag to reorder, verify order in merge preview |
| Full-screen presentation mode | PPT-03 | Browser fullscreen API behavior | Click fullscreen button, verify sidebar hidden and slides fill viewport |
| Multi-result gallery switching | PPT-04 | Visual state transition | Switch between multiple PPT results, verify correct PPT loads each time |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
