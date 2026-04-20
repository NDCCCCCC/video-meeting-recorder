# Phase 2: Local Transcription - Context

**Gathered:** 2026-04-17
**Status:** Ready for planning

<domain>
## Phase Boundary

System extracts video frames, detects slide changes using multi-dimensional similarity (SSIM + pHash + edge analysis), generates PPTX files locally, with real-time progress tracking. Users can trigger transcription from the file list, see staged progress in a modal, and download the resulting PPTX.

Requirements covered: LCL-01, LCL-02, LCL-03, LCL-04, TRAN-01 (local trigger), TRAN-04, TRAN-06.

</domain>

<decisions>
## Implementation Decisions

### Frame Extraction Strategy
- **D-01:** Dual-layer strategy: first extract frames at fixed interval, then apply similarity detection to deduplicate. Two-pass filtering for higher accuracy.
- **D-02:** Default sampling rate is 1 frame per 2 seconds. User can choose from preset options (1s / 2s / 5s) in the transcription trigger UI.
- **D-03:** Frames are extracted as JPEG (quality 95) to temporary directory. JPEG is 5-10x smaller than PNG with negligible impact on similarity detection accuracy. Temp files cleaned up after processing.
- **D-04:** For comparison, frames are downscaled to 720p resolution. For the final PPTX, original-resolution frames are re-extracted by FFmpeg at the detected keyframe timestamps.
- **D-05:** Temp directory cleaned up after transcription completes (both success and failure paths).

### Similarity Detection Algorithm
- **D-06:** Pure Go implementation using `golang.org/x/image` for image decoding and `github.com/corona10/goimagehash` for perceptual hashing. SSIM and edge detection implemented in Go. No FFmpeg filter dependency for comparison logic.
- **D-07:** OR logic: any single detection method registering a change causes the frame to be retained. Per ROADMAP: SSIM < 0.85 OR pHash difference > 10 OR edge change rate > 0.25.
- **D-08:** Fixed thresholds from ROADMAP: SSIM < 0.85, pHash diff > 10, edge change rate > 0.25. No user-adjustable parameters for detection thresholds.

### PPTX Generation
- **D-09:** Use `unidoc/unioffice` library for PPTX generation (user selected over Go-pptx for better documentation and community support).
- **D-10:** Slide layout: full-frame image with no margins, padding, or decorations. Image fills the entire slide area.
- **D-11:** Slide dimensions: 16:9 (standard widescreen). Modern standard, matches most displays and projectors.
- **D-12:** Each unique (deduplicated) frame becomes one slide page. No additional metadata (page numbers, timestamps) on slides.

### Progress & UX Flow
- **D-13:** "转录" button lives in the file list action column (same row as download/delete/split buttons). Applies to both full videos and split segments.
- **D-14:** After clicking "转录", a modal popup shows real-time progress with staged phases: "帧提取中..." → "画面检测中 (45/200)..." → "生成PPT...". User can close the modal and continue browsing; transcription continues in the background.
- **D-15:** On completion, modal displays "转录完成" with a "下载PPT" button and a "关闭" button. File list auto-refreshes to show the new PPT association.
- **D-16:** Polling interval is 5 seconds (slower than Phase 1's 2s — transcription tasks take minutes, 5s reduces server load while still feeling responsive).
- **D-17:** Progress data structure includes: current stage (extracting/detecting/generating), frames processed, total frames, percentage.

### Claude's Discretion
- Exact Go implementation of SSIM calculation
- Exact Go implementation of edge change rate detection
- TranscriptionTask database model design (fields, status enum)
- API endpoint paths and request/response structures
- Temporary directory naming and cleanup strategy
- goimagehash library usage for pHash (dhash vs phash implementation choice)
- How to handle concurrent transcription tasks (queue with workers like SplittingService)
- Error handling for individual frame processing failures

### Folded Todos
None.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Local transcription requirements
- `.planning/REQUIREMENTS.md` §Local Transcription — LCL-01 through LCL-04 acceptance criteria
- `.planning/REQUIREMENTS.md` §Transcription — TRAN-01 (local trigger), TRAN-04, TRAN-06 acceptance criteria
- `.planning/ROADMAP.md` §Phase 2 — Success criteria and phase boundary

### Phase 1 context (prior decisions and patterns)
- `.planning/phases/01-video-splitting/01-CONTEXT.md` — Worker pool pattern, statusMap pattern, file registration, FFmpeg invocation patterns, frontend polling pattern

### Existing code to reuse or extend
- `internal/services/splitting_service.go` — Worker pool with task queue, statusMap with sync.RWMutex (exact template for TranscriptionService)
- `internal/services/conversion_service.go` — FFmpeg invocation pattern with exec.CommandContext, retry with exponential backoff
- `internal/models/ppt_file.go` — PPTFile model with SourceVideoFileID foreign key, placeholder GenerateFromVideo method
- `internal/handlers/video_file_handler.go` — Handler pattern for reference
- `frontend/src/pages/split/index.tsx` — Polling pattern for progress tracking
- `frontend/src/pages/files/index.tsx` — File list action column (where "转录" button will be added)
- `frontend/src/api/split.ts` — API client module pattern

### Project constraints
- `.planning/PROJECT.md` — Tech stack (Go 1.24/Gin, React 19/Ant Design 6, SQLite/GORM, FFmpeg), key decisions, constraints
- `.planning/STATE.md` — Tech stack context, critical pitfalls

No external specs — requirements are fully captured in REQUIREMENTS.md and PROJECT.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **SplittingService** (`internal/services/splitting_service.go`): Task queue with worker pool, in-memory statusMap, cancellable per-task contexts. Copy this architecture for TranscriptionService.
- **ConversionService** (`internal/services/conversion_service.go`): FFmpeg invocation with `exec.CommandContext`, retry logic, worker pool. Template for FFmpeg frame extraction.
- **PPTFile model** (`internal/models/ppt_file.go`): Already exists with SourceVideoFileID, FilePath, PageCount. Needs extending with transcription metadata fields.
- **File list action column** (`frontend/src/pages/files/index.tsx`): Button rendering pattern with PermissionGuard and Tooltip — add "转录" button here.
- **Split page polling** (`frontend/src/pages/split/index.tsx`): setInterval + API polling pattern — template for transcription progress modal.
- **API client pattern** (`frontend/src/api/split.ts`): `apiRequest<T>()` wrapper with typed interfaces — create `transcription.ts` following this.

### Established Patterns
- **Worker pool services**: Struct-based service with task channel, N workers, status map, Start/Stop lifecycle
- **FFmpeg commands**: `exec.CommandContext()` with cancellable context, stderr capture, timeout
- **API handlers**: Gin handlers with `ShouldBindQuery`/`ShouldBindJSON`, unified `response.GinSuccess/GinError`
- **Frontend state**: Local useState for page state, API calls via `apiRequest<T>()`
- **Database**: GORM with soft delete, SQL migrations registered in migration registry
- **Frontend polling**: setInterval every 2-5 seconds, status check API, clear on complete/fail

### Integration Points
- **VideoFile model**: "转录" button triggers transcription for any VideoFile (full video or split segment)
- **PPTFile model**: Needs extension for TranscriptionTask relationship and status fields
- **File list page**: Add "转录" button to action column, refresh list on transcription completion
- **cmd/server/app.go**: Register TranscriptionService, TranscriptionHandler, and API routes
- **go.mod**: Add `unidoc/unioffice`, `golang.org/x/image`, `github.com/corona10/goimagehash`

</code_context>

<specifics>
## Specific Ideas

- No specific product reference — standard slide extraction from video, similar to PowerPoint's "Create from Photo Album" feature
- The dual-layer strategy (fixed sampling + similarity dedup) is the core technical innovation of this phase
- Progress modal should feel informative without being overwhelming — three clear stages with frame counts
- "转录" button should be visually distinct from other actions (perhaps colored differently)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 02-local-transcription*
*Context gathered: 2026-04-17*
