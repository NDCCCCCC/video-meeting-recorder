---
phase: 06-06
slug: ppt-preview-ui-improvements
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-20
---

# Phase 06-06 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | None (manual testing only) |
| **Config file** | none — project lacks test framework |
| **Quick run command** | `npm run dev` (start dev server for manual testing) |
| **Full suite command** | Manual visual inspection + click testing |
| **Estimated runtime** | ~5-10 minutes per feature |

---

## Sampling Rate

- **After every task commit:** Manual smoke test of modified component
- **After every plan wave:** Full manual testing of all features in wave
- **Before `/gsd-verify-work`:** All features visually inspected and tested
- **Max feedback latency:** Immediate (manual testing has no queue time)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Secure Behavior | Test Type | Manual Test Instructions | Status |
|---------|------|------|-------------|-----------------|-----------|-------------------------|--------|
| 06-06-01 | 01 | 1 | REQ-06-06-01 | Speed control updates video.playbackRate | manual | Click each speed option (0.5x-2x), verify playback rate changes | ⬜ pending |
| 06-06-02 | 02 | 1 | REQ-06-06-02 | 16:9 aspect ratio maintained on resize | manual | Resize browser window, verify previews stay 16:9 | ⬜ pending |
| 06-06-03 | 03 | 2 | REQ-06-06-03 | Operations bar below preview, no scroll | manual | Scroll page, verify operations bar always visible | ⬜ pending |
| 06-06-04 | 04 | 2 | REQ-06-06-04 | Direct capture inserts slide at position+1 | manual | Click capture, verify slide appears at correct position | ⬜ pending |
| 06-06-05 | 05 | 2 | REQ-06-06-05 | Thumbnails scroll vertically with lazy load | manual | Open DevTools Network, scroll 100+ slides, verify lazy loading | ⬜ pending |

*Status: ⬜ pending · ✅ pass · ❌ fail · ⚠️ needs-review*

---

## Wave 0 Requirements

**Existing infrastructure gap:** Project lacks automated test framework (no vitest/jest config).

**Wave 0 scope for this phase:**
- [ ] `frontend/src/components/__tests__/PlaybackSpeedControl.test.tsx` — stubs for speed control (optional - deferred)
- [ ] Test framework setup: Vitest or Jest configuration in `vite.config.ts` (optional - deferred)
- [ ] Test script in `package.json`: `"test": "vitest"` (optional - deferred)

**Note:** Since this is a UI enhancement phase with no backend changes, and project lacks test infrastructure, manual testing is the primary verification method. Automated tests are marked as optional/deferred to Phase 07+.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Video playback speed control (0.5x-2x) | REQ-06-06-01 | Requires visual verification of playback rate | 1. Load a PPT result with video preview<br>2. Click each speed option<br>3. Verify audio and video play at selected speed |
| Side-by-side 16:9 layout responsiveness | REQ-06-06-02 | Visual inspection of aspect ratio | 1. Open PPT preview page<br>2. Resize browser from 1920px to 768px<br>3. Verify both previews maintain 16:9 ratio |
| Operations bar positioning | REQ-06-06-03 | Visual layout verification | 1. Scroll PPT preview page<br>2. Verify operations bar stays below preview<br>3. Check no horizontal scroll on operations bar |
| Direct slide capture insertion | REQ-06-06-04 | API call verification | 1. Open video at 00:01:30<br>2. Click "捕获幻灯片"<br>3. Verify new slide appears at position+1<br>4. Verify slide timestamp is ~00:01:30 |
| Thumbnail lazy loading | REQ-06-06-05 | Network inspection required | 1. Open DevTools Network tab<br>2. Filter by "Img"<br>3. Scroll through 100+ thumbnails<br>4. Verify images load on-demand (not all at once) |

---

## Validation Sign-Off

- [ ] All tasks have manual verification instructions
- [ ] Wave 0 optional test stubs documented
- [ ] Manual test checklist covers all 5 requirements
- [ ] Security considerations addressed (no XSS/CSRF vectors introduced)

**Approval:** pending

---

*VALIDATION created: 2026-04-20*
*Note: Manual testing focus due to lack of automated test infrastructure in project*
