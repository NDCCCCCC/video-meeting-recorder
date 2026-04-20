---
status: partial
phase: 01-video-splitting
source: 01-00-SUMMARY.md, 01-01-SUMMARY.md, 01-02-SUMMARY.md, 01-03-SUMMARY.md, 01-04-SUMMARY.md
started: 2026-04-18T00:35:00Z
updated: 2026-04-18T00:55:00Z
---

## Current Test

[testing paused — 11 items skipped, 0 resolved]

## Tests

### 1. Cold Start Smoke Test
expected: Kill any running server. Start the application from scratch. Server boots without errors, database migration (including 003_add_segment_fields) completes, and the health check endpoint (/health) returns a 200 response with {"status":"healthy"}.
result: skipped
reason: 用户跳过

### 2. Split Page Navigation
expected: From the file list page, click a video file or navigate to /split/:id for an MP4 file. The split page loads with a video player showing the video, an interactive timeline below the player, and an empty markers section with the prompt "暂无分割标记".
result: skipped
reason: 用户跳过

### 3. Add Split Markers via Timeline Click
expected: On the split page, click on the timeline track. A marker appears as a vertical line with a tooltip showing the timestamp in MM:SS format. The marker also appears as a closable tag in the markers section below.
result: skipped
reason: 用户跳过

### 4. Manual Timestamp Input
expected: In the timestamp input field, type a time value (e.g., "01:30" or "90"). Click the add button. A new marker appears at that position on the timeline with correct timestamp.
result: skipped
reason: 用户跳过

### 5. Remove Split Markers
expected: Click the X button on a marker tag. The marker is removed from both the tag list and the timeline. If only 2 markers remain, the segment table still updates correctly.
result: skipped
reason: 用户跳过

### 6. Segment Preview Table
expected: Add 3 or more markers to the timeline. A segment preview table appears showing the segments with start/end times and durations, computed from the sorted marker positions.
result: skipped
reason: 用户跳过

### 7. Execute Video Split
expected: With at least 2 markers placed, click "确认分割". A confirmation dialog appears. Confirm the split. A progress indicator shows "正在分割中..." with polling updates. When complete, a success message "分割完成！已生成 N 个视频段落" appears.
result: skipped
reason: 用户跳过

### 8. Source Column in File List
expected: Navigate to the file list page (/files). A "来源" column is visible showing color-coded tags for each file: blue "录制" for recordings, green "快照" for snapshots, orange "分割" for split segments.
result: skipped
reason: 用户跳过

### 9. Split Action on File List
expected: In the file list, MP4 recording files show a "分割" action button or menu item. Clicking it navigates to /split/:id for that file. Non-MP4 files or segment/snapshot files should not show this button.
result: skipped
reason: 用户跳过

### 10. Snapshot Button on Task List
expected: Navigate to the task list page (/tasks). Active recording tasks (status: recording) show a "生成MP4快照" button. Clicking it triggers snapshot generation with a loading state showing "生成中...". When done, the snapshot file appears in the file list.
result: skipped
reason: 用户跳过

### 11. Auto-Refresh File List
expected: While on the file list page, start a new recording or split operation in another tab. The file list automatically updates within ~5 seconds to show the new file, without manual page refresh.
result: skipped
reason: 用户跳过

## Summary

total: 11
passed: 0
issues: 0
pending: 0
skipped: 11
blocked: 0

## Gaps

[none]
