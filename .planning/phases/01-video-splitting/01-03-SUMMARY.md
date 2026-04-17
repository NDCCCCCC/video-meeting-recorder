---
phase: 01-video-splitting
plan: 03
subsystem: frontend
tags: [ui, split, timeline, markers]
wave: 2
dependency_graph:
  requires: ["01-01", "01-02"]
  provides: ["ui-01"]
  affects: []
tech_stack:
  added: []
  patterns:
    - Ant Design Slider with custom marks for timeline markers
    - React hooks for marker state management
    - Polling-based progress tracking for split operations
    - Manual timestamp parsing (MM:SS or seconds)
key_files:
  created:
    - path: frontend/src/api/split.ts
      provides: API client for video split operations
    - path: frontend/src/components/TimelineWithMarkers.tsx
      provides: Interactive timeline with click-to-add markers
    - path: frontend/src/pages/split/index.tsx
      provides: Video split page with player and segment preview
  modified:
    - path: frontend/src/router/index.tsx
      provides: Route registration for /split/:id
    - path: frontend/src/utils/permissions.ts
      provides: Permission constants for split and snapshot
decisions: []
metrics:
  duration_seconds: 96
  completed_date: "2026-04-17T04:13:50Z"
  tasks_completed: 2
  files_created: 3
  files_modified: 2
---

# Phase 01 Plan 03: Split Page Implementation Summary

Build the video split page with an interactive timeline that lets users add, reposition, and remove split markers. The page integrates with the backend split API to execute FFmpeg splits. Per D-01 through D-04, markers are added by clicking the timeline or entering timestamps manually, can be dragged to reposition, show as vertical lines with tooltips, and use second-level precision only.

## One-Liner

Video split page with Ant Design Slider-based interactive timeline, manual timestamp input, segment preview table, and polling-based split progress tracking.

## Deviations from Plan

### Auto-fixed Issues

None - plan executed exactly as written.

## Known Stubs

None - all features are fully implemented.

## Threat Flags

None - no new security-relevant surface introduced beyond plan expectations.

## Implementation Details

### Task 1: Split API Client and TimelineWithMarkers Component

**Files Created:**
- `frontend/src/api/split.ts` - API client with submitSplit, getSplitStatus, getSegments
- `frontend/src/components/TimelineWithMarkers.tsx` - Interactive timeline component

**Key Features:**
- Ant Design Slider with custom marks for marker positions
- Click-to-add markers on timeline track
- Manual timestamp input (MM:SS or seconds)
- Marker list as closable tags
- Validation: 2s minimum spacing, max 20 markers, within duration
- Tooltip showing MM:SS format on hover
- formatTime helper for consistent time display

**Permission Constants Added:**
- `FILE_SPLIT: 'files:split'` - Permission for video split operations
- `RECORDING_SNAPSHOT: 'recording:snapshot'` - Permission for recording snapshots

### Task 2: Split Page and Route Registration

**Files Created:**
- `frontend/src/pages/split/index.tsx` - Video split page

**Key Features:**
- HTML5 video player with custom controls (play/pause, skip ±10s, seek)
- TimelineWithMarkers integration for marker management
- Segment preview table computed from markers array
- Split execution with confirmation modal
- 2-second polling interval for split progress
- Warning alert about ±2s fast split imprecision (per D-07)
- Loading states and error handling
- Navigation back to /files on completion or cancel

**Route Registration:**
- Added `split/:id` route to frontend/src/router/index.tsx

**UI Copy (matches UI-SPEC):**
- Primary CTA: "确认分割"
- Empty state heading: "暂无分割标记"
- Empty state body: "点击视频时间线添加分割点，或输入时间精确定位。添加标记后点击"确认分割"开始处理。"
- Progress: "正在分割中..."
- Success: "分割完成！已生成 {count} 个视频段落"
- Warning: "快速分割模式可能有±2秒误差"

## Technical Patterns

### State Management
- `useState` for markers array (sorted timestamps)
- `useState` for video playback state (currentTime, isPlaying, duration)
- `useMemo` for segment previews computed from markers
- `useCallback` for event handlers to prevent re-renders

### Validation Logic
- Marker spacing: minimum 2 seconds between markers
- Marker count: maximum 20 markers per video
- Range validation: markers must be within video duration
- Minimum markers: at least 2 markers required for split

### Progress Polling
- 2-second interval polling of `/api/v1/videos/:id/split-status`
- Polling stops on 'completed' or 'failed' status
- Error handling with user-friendly messages

### Time Formatting
- `formatTime(seconds)` converts seconds to MM:SS format
- `parseTimeInput(input)` handles both "MM:SS" and pure seconds input

## Integration Points

### Backend API
- `POST /api/v1/videos/:id/split` - Submit split task with markers array
- `GET /api/v1/videos/:id/split-status` - Poll split progress
- `GET /api/v1/videos/:id/segments` - Get generated segments
- `GET /api/v1/files/:id` - Get video file metadata
- `GET /api/v1/files/:id/download?token=xxx` - Video stream URL

### Frontend Components
- Uses existing `apiRequest` from apiClient.ts for authenticated requests
- Extends VideoPlayerModal slider patterns for timeline
- Follows Ant Design patterns from tasks/index.tsx and files/index.tsx

## Self-Check: PASSED

**Created Files:**
- ✓ frontend/src/api/split.ts (65 lines)
- ✓ frontend/src/components/TimelineWithMarkers.tsx (187 lines)
- ✓ frontend/src/pages/split/index.tsx (457 lines)

**Modified Files:**
- ✓ frontend/src/router/index.tsx (added split/:id route)
- ✓ frontend/src/utils/permissions.ts (added FILE_SPLIT, RECORDING_SNAPSHOT)

**Commits:**
- ✓ 6a441e0: feat(01-03): create split API client, TimelineWithMarkers component, and permissions
- ✓ f12e1f8: feat(01-03): create split page with interactive timeline and register route

**Verification:**
- ✓ submitSplit function exists in split.ts
- ✓ TimelineWithMarkersProps interface exists
- ✓ FILE_SPLIT permission constant exists
- ✓ split/:id route registered
- ✓ "确认分割" appears 4 times in split page
- ✓ "快速分割模式" appears 2 times in split page

---
*Plan completed: 2026-04-17T04:13:50Z*
*Execution time: 96 seconds*
*Tasks: 2/2*
*Commits: 2*
