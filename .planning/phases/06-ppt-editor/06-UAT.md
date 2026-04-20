---
status: testing
phase: 06-ppt-editor
source:
  - 06-01-SUMMARY.md
  - 06-02-SUMMARY.md
  - 06-03-SUMMARY.md
  - 06-04-SUMMARY.md
  - 06-05-SUMMARY.md
started: 2026-04-20T14:35:00Z
updated: 2026-04-20T14:35:00Z
---

## Current Test

number: 2
name: Detect Duplicate Slides
expected: |
  Open the PPT results page for a generated presentation. Click the "Detect Duplicates"
  button in the operations panel. The system scans all slides and displays groups of
  visually similar slides with similarity scores (SSIM, pHash distance, edge change rate).
  Each group shows side-by-side slide comparison with checkboxes for selection.
awaiting: user response

## Tests

### 1. Cold Start Smoke Test
expected: |
  Kill any running server/service. Clear ephemeral state (temp DBs, caches, lock files).
  Start the application from scratch. Server boots without errors, any seed/migration
  completes, and a primary query (health check, homepage load, or basic API call)
  returns live data.
result: pass
note: TypeScript compilation errors were fixed (removed unused imports, added optional chaining, fixed VideoPreviewPanel import)

### 2. Detect Duplicate Slides
expected: |
  Open the PPT results page for a generated presentation. Click the "Detect Duplicates"
  button in the operations panel. The system scans all slides and displays groups of
  visually similar slides with similarity scores (SSIM, pHash distance, edge change rate).
  Each group shows side-by-side slide comparison with checkboxes for selection.
result: pending

### 3. Delete Duplicate Slides
expected: |
  In the duplicate detection panel, select slides to delete (excluding the first slide
  in each group which is pre-selected as the keeper). Click "Delete Selected" with
  confirmation. The system creates a backup file, regenerates the PPT without the
  selected slides, and refreshes the slide list with updated page count.
result: pending

### 4. Rollback PPT Edits
expected: |
  After deleting slides, click the "Rollback" button in the operations panel with
  confirmation. The system restores the original PPT from the backup file, clears
  the backup path tracking, and refreshes the slide list to show the original
  slide count.
result: pending

### 5. Capture Frame from Video
expected: |
  Open the slide capture panel by clicking the "捕获幻灯片" (Capture Slides) button.
  The video player loads with the source video. Use playback controls to navigate
  to the desired frame. Click "捕获当前帧" (Capture Current Frame). The system
  displays a preview of the captured frame with the current timestamp.
result: pending

### 6. Insert Captured Frame as Slide
expected: |
  After capturing a frame, select an insert position (After Current, Before Current,
  At End, or Custom). Click "插入幻灯片" (Insert Slide). The system validates the
  position, saves the frame to the slide cache (fullsize + thumbnail), regenerates
  the PPT with the new slide, updates the page count, and closes the modal.
result: pending

### 7. View Video Preview Panel
expected: |
  On the PPT results page, a video preview panel is displayed below the PPT preview
  (by default). The panel shows an HTML5 video player with custom playback controls
  (play/pause, skip forward/backward, progress bar, fullscreen button, and current
  time display in MM:SS format).
result: pending

### 8. Sync Slide to Video (Forward Sync)
expected: |
  Click on any slide thumbnail in the PPT preview. The video player seeks to the
  corresponding timestamp within ±2 seconds accuracy. If auto-play is enabled, the
  video begins playing from that position.
result: pending

### 9. Sync Video to Slide (Reverse Sync)
expected: |
  Play the video from the video preview panel. As the video plays, the PPT slide
  display automatically updates to show the slide corresponding to the current
  video timestamp. Updates are debounced (approximately once per second) to avoid
  excessive UI changes.
result: pending

### 10. Jump to Current Slide Button
expected: |
  In the video preview panel header, click the "Jump to current slide" button.
  The video seeks to the timestamp of the currently displayed slide in the PPT
  preview, allowing manual re-synchronization after any manual video seeking.
result: pending

### 11. Toggle Video Panel Visibility
expected: |
  Click the "Show/Hide Video Preview" button in the operations panel. The video
  preview panel toggles between visible and hidden states. The panel visibility
  state persists during the session (default: visible).
result: pending

### 12. Handle Missing Timestamps Gracefully
expected: |
  Open a PPT results page for a transcription that has no timestamp data (older
  transcription or failed recording). The video preview panel displays a warning
  alert explaining that timestamp synchronization is not available, but the video
  player still functions and can be used for manual playback.
result: pending

### 13. Slide Capture Insert Position Validation
expected: |
  In the slide capture panel, attempt to insert a slide with an invalid position
  (e.g., 0 or a number greater than total slides + 1). The system shows a
  validation error and prevents the insertion. Valid positions are 1 to
  (total slides + 1).
result: pending

### 14. Backup File Created Before Edits
expected: |
  Before any slide deletion or insertion operation, verify that a backup file is
  created with a `.bak.{unix_timestamp}` suffix. The backup path is tracked in
  the PPTFile model and can be used for rollback operations.
result: pending

## Summary

total: 14
passed: 1
issues: 0
pending: 13
skipped: 0

## Gaps

none yet - all issues resolved
