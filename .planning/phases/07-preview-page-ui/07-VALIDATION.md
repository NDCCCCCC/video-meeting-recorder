---
phase: 07
slug: preview-page-ui
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-20
---

# Phase 07 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 2.x (existing) |
| **Config file** | frontend/vitest.config.ts |
| **Quick run command** | `npm test -- --run frontend/src/components/__tests__/` |
| **Full suite command** | `npm test -- --run` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `npm test -- --run frontend/src/components/__tests__/` (component tests only)
- **After every plan wave:** Run `npm test -- --run` (full suite including linting)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | UI-07-01 | — | No XSS from user-controlled content | component | `npm test -- ThumbnailSidebar` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | UI-07-02 | — | No unsafe CSS injection | component | `npm test -- VideoPlayer` | ❌ W0 | ⬜ pending |
| 07-02-01 | 02 | 1 | UI-07-03 | — | Input validation for time values | component | `npm test -- ProgressBar` | ❌ W0 | ⬜ pending |
| 07-02-02 | 02 | 1 | UI-07-04 | — | No XSS from transcription content | component | `npm test -- TranscriptionDropdown` | ❌ W0 | ⬜ pending |
| 07-03-01 | 03 | 1 | UI-07-05, UI-07-06 | — | No XSS from user-controlled content | component | `npm test -- BasicInfoPanel` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/ThumbnailSidebar.test.tsx` — stubs for UI-07-01
- [ ] `frontend/src/components/__tests__/VideoPlayer.test.tsx` — stubs for UI-07-02
- [ ] `frontend/src/components/__tests__/ProgressBar.test.tsx` — stubs for UI-07-03
- [ ] `frontend/src/components/__tests__/TranscriptionDropdown.test.tsx` — stubs for UI-07-04
- [ ] `frontend/src/components/__tests__/BasicInfoPanel.test.tsx` — stubs for UI-07-05, UI-07-06

*Existing infrastructure covers this phase (Vitest 2.x already configured).*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual layout correctness | UI-07-01 | CSS aspect ratio matching is visual | Open PPT preview page, verify thumbnails match preview height, no overflow |
| Video player cropping | UI-07-02 | Visual verification of black bar removal | Play video with non-16:9 content, confirm no black bars |
| Dropdown positioning | UI-07-04 | Visual placement at top-right | View PPT with multiple results, verify dropdown position |

*All functional behaviors have automated verification; visual items require manual confirmation.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
