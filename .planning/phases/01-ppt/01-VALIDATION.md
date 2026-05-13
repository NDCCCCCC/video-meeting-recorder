---
phase: 01
slug: ppt
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-05-12
---

# Phase 01 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | React Testing Library + Vitest (frontend), Go test (backend) |
| **Config file** | frontend/vitest.config.ts |
| **Quick run command** | `npm run test` (frontend), `go test ./... -short` (backend) |
| **Full suite command** | `npm run test:ci` (frontend), `go test ./... -v` (backend) |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `npm run test` (frontend only) or `go test ./internal/handlers -short` (backend only)
- **After every plan wave:** Run `npm run test:ci` (frontend) + `go test ./... -v` (backend)
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | D-01 (WebVTT) | T-01-01 | Parse only .vtt files, reject others | unit | `npm run test -- SubtitlePanel.test.tsx` | ❌ W0 | ⬜ pending |
| 01-01-02 | 01 | 1 | D-05 (独立区域) | — | Render in separate DOM node | unit | `npm run test -- SubtitlePanel.test.tsx` | ❌ W0 | ⬜ pending |
| 01-01-03 | 01 | 1 | D-10 (时间同步) | — | Sync within 250ms accuracy | unit | `npm run test -- useSubtitleSync.test.ts` | ❌ W0 | ⬜ pending |
| 01-02-01 | 02 | 1 | D-04 (API返回URL) | T-01-02 | Validate file path, prevent traversal | unit | `go test ./internal/handlers -run TestSubtitleCheck` | ❌ W0 | ⬜ pending |
| 01-02-02 | 02 | 1 | D-04 (下载流) | T-01-02 | Token auth required, no public access | unit | `go test ./internal/handlers -run TestSubtitleDownload` | ❌ W0 | ⬜ pending |
| 01-03-01 | 03 | 1 | D-06 (开关按钮) | — | Toggle renders/hides panel | unit | `npm run test -- SubtitleControls.test.tsx` | ❌ W0 | ⬜ pending |
| 01-03-02 | 03 | 1 | D-07 (字号调整) | — | Font size changes apply | unit | `npm run test -- SubtitleControls.test.tsx` | ❌ W0 | ⬜ pending |
| 01-03-03 | 03 | 1 | D-09 (样式设置) | — | Color/bg changes apply | unit | `npm run test -- SubtitleControls.test.tsx` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `frontend/src/components/__tests__/SubtitlePanel.test.tsx` — stubs for D-01, D-05
- [ ] `frontend/src/hooks/__tests__/useSubtitleSync.test.ts` — stubs for D-10
- [ ] `frontend/src/components/__tests__/SubtitleControls.test.tsx` — stubs for D-06, D-07, D-09
- [ ] `internal/handlers/video_file_handler_test.go` — stubs for subtitle endpoints
- [ ] `frontend/vitest.config.ts` — verify Vitest config exists

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| 视觉检查字幕显示效果 | D-05, UI-SPEC | Visual rendering requires human judgment | 1. Play video with subtitle 2. Verify text appears below video 3. Verify no overlay on video content |
| 字幕与视频同步准确性 | D-10 | Timing precision <250ms requires manual timing check | 1. Play video with subtitle 2. Observe subtitle change timing vs video audio |
| 键盘快捷键响应 | D-06 (Alt+C) | Interaction testing | 1. Play video 2. Press Alt+C 3. Verify subtitle toggles |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
