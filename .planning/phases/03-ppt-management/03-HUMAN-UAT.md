---
status: deferred
phase: 03-ppt-management
source: [03-VERIFICATION.md]
started: 2026-04-18T12:00:00Z
updated: 2026-04-18T12:00:00Z
completed: 2026-05-06T14:24:00Z
note: Tests were never executed. Verification (03-VERIFICATION.md) shows status: human_needed with 7/7 must_haves verified. Acknowledged at milestone close - these are UI interaction tests that require manual runtime testing.
---

## Current Test

[awaiting human testing]

## Tests

### 1. PPT Preview Rendering
expected: Click "预览PPT" in file list, navigate to result page with left-right split layout (70% preview + 30% info panel), sidebar thumbnails visible
result: [pending]

### 2. Slide Navigation (Keyboard + Click)
expected: Click thumbnails to navigate; ArrowLeft/ArrowRight keys work; page number input jumps to specified slide
result: [pending]

### 3. Fullscreen Mode
expected: Click "全屏演示" hides sidebar, slide fills container; Escape exits fullscreen mode
result: [pending]

### 4. Gallery Strip Multi-Result Switching
expected: Click gallery strip cards to switch between multiple PPT results; active card highlights
result: [pending]

### 5. Merge Mode (Select + Drag + Confirm)
expected: Click "合并幻灯片" enables selection; drag-to-reorder in bottom bar; "确认合并" completes merge with toast
result: [pending]

### 6. Re-transcribe Flow
expected: Click "重新转录" dropdown, select mode, TranscriptionProgressModal shows progress; on completion new PPT appears
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
