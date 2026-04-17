# Phase 1: Video Splitting - Context

**Gathered:** 2026-04-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can split videos at multiple time points, generate MP4 snapshots during recording, and all new MP4 files are automatically scanned into the file management system. All features are local with no external dependencies.

Requirements covered: SPLIT-01 through SPLIT-05, SNAP-01, SNAP-02, SCAN-01, SCAN-02, UI-01.

</domain>

<decisions>
## Implementation Decisions

### Split Marker Interaction
- **D-01:** Users add split markers by clicking on the video timeline OR by manually entering a timestamp in an input field — both methods are supported
- **D-02:** Markers can be repositioned by dragging along the timeline
- **D-03:** Precision is second-level only — no frame-level micro-adjustment or timeline zoom needed
- **D-04:** Markers display as vertical lines on the timeline with hover tooltip showing the timestamp; clicking a marker shows delete/edit actions

### Split Precision vs Speed
- **D-05:** Default split uses FFmpeg `-c copy` mode (fast, lossless, potential ±2s keyframe misalignment)
- **D-06:** If split results are imprecise, user can re-run with re-encode mode for frame-accurate cuts
- **D-07:** The UI should communicate that fast mode may have slight imprecision at split points

### Recording Snapshot
- **D-08:** "生成MP4快照" button appears inline on the active recording task row in the task list page
- **D-09:** After clicking, button text changes to "生成中..." and the system uses the existing notification system to alert on completion
- **D-10:** Snapshot MP4 file automatically appears in the file management list via service callback (same auto-scan mechanism as other MP4 generation)

### Segment Management & Auto Scan
- **D-11:** Split segments are stored as independent VideoFile records with a `parent_id` field linking back to the source video
- **D-12:** Segments appear in the existing file list with an additional column showing "来源" (source: 录制/快照/分割) and a link to the original video
- **D-13:** Auto-scan uses service callbacks — recording service, conversion service, and splitting service call VideoFileService directly when new MP4 files are produced (no file system watching, no polling)
- **D-14:** Segments can be independently renamed, deleted, downloaded, and triggered for transcription

### Claude's Discretion
- Exact timeline marker component implementation (custom component vs extending existing Slider)
- FFmpeg command construction for split and snapshot operations
- Snapshot extraction technique (copy partial MKV vs dual-output FFmpeg)
- Database model additions (parent_id field on VideoFile, new migration)
- API endpoint design for split operations
- Segment naming convention (e.g., "原视频名_段落1.mp4")
- File storage paths for split segments
- How to handle split during recording vs split of completed video
- Re-encode mode UI trigger (per-segment "re-split with precision" button or global option)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Video splitting requirements
- `.planning/REQUIREMENTS.md` §Video Splitting — SPLIT-01 through SPLIT-05, SNAP-01, SNAP-02, SCAN-01, SCAN-02 acceptance criteria
- `.planning/ROADMAP.md` §Phase 1 — Success criteria and phase boundary

### Existing FFmpeg integration
- `internal/recorder/coordinator.go` — FFmpeg recording with tee muxer (MKV+HLS), process management patterns
- `internal/services/conversion_service.go` — MKV-to-MP4 conversion worker pool, retry with exponential backoff
- `internal/services/video_file_service.go` — ScanFiles(), CreateFileFromTask(), ffprobe metadata extraction

### Existing file management
- `internal/models/video_file.go` — VideoFile model, fields, status enum
- `internal/models/video_recording_task.go` — Recording task model, status state machine
- `internal/handlers/video_file_handler.go` — File API endpoints pattern

### Existing frontend
- `frontend/src/components/VideoPlayerModal.tsx` — Current video player with seek slider (to be extended for markers)
- `frontend/src/pages/files/index.tsx` — File list page pattern (to add source column)
- `frontend/src/pages/tasks/index.tsx` — Task list page pattern (to add snapshot button)
- `frontend/src/api/apiClient.ts` — API client pattern

### Project constraints
- `.planning/PROJECT.md` — Tech stack, key decisions, constraints (all Go, FFmpeg, SQLite, React+Ant Design)
- `.planning/STATE.md` — Tech stack context, critical pitfalls (FFmpeg keyframe misalignment §3)

No external specs — requirements are fully captured in REQUIREMENTS.md and PROJECT.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **VideoPlayerModal** (`frontend/src/components/VideoPlayerModal.tsx`): Has video playback, seek slider, time display, custom controls. The seek slider (Ant Design Slider) needs to be extended into a marker-enabled timeline component.
- **ConversionService** (`internal/services/conversion_service.go`): Worker pool pattern with cancellable contexts and retry logic — same pattern should be used for split operations.
- **VideoFileService.ScanFiles()** (`internal/services/video_file_service.go`): Existing scan logic that walks recordings directory and creates VideoFile records with ffprobe metadata.
- **VideoFileService.CreateFileFromTask()**: Pattern for creating VideoFile records from recording tasks — similar to what snapshot and split will need.
- **Notification system**: Already integrated — snapshot completion can use existing notification infrastructure.

### Established Patterns
- **FFmpeg commands**: `exec.CommandContext()` with cancellable context, stderr capture, process stored in map
- **Service layer**: Struct-based services with injected dependencies (DB, logger, config)
- **API handlers**: Gin handlers with `ShouldBindQuery`/`ShouldBindJSON`, unified `response.GinSuccess/GinError`
- **Frontend state**: Local `useState` for page state, Zustand only for auth. API calls via `apiRequest<T>()`.
- **Database**: GORM with soft delete, custom SQL migrations registered in migration registry

### Integration Points
- **VideoFile model** needs `parent_id` field and `source_type` field (recording/snapshot/split)
- **Task list page** needs inline snapshot button for active recording tasks
- **File list page** needs source column and parent video link
- **Recording completion flow** needs callback to VideoFileService (for auto-scan)
- **Conversion completion flow** needs callback to VideoFileService (for auto-scan)
- **New split page** with extended video player + timeline markers + split confirmation

</code_context>

<specifics>
## Specific Ideas

- Markers should feel like video editing timeline markers — click to add, drag to move, visual and responsive
- Split page layout: video player (top) + timeline with markers (below player) + segment list (below timeline)
- Snapshot should feel seamless — one click, status updates, notification on done
- No specific product reference beyond standard video editing timeline patterns

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 01-video-splitting*
*Context gathered: 2026-04-17*
