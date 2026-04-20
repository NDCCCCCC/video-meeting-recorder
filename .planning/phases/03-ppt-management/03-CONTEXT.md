# Phase 3: PPT Management - Context

**Gathered:** 2026-04-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can preview PPT slides in browser with a main view + sidebar thumbnail layout, manage multiple transcription results for the same video via gallery-style switching, and merge selected slides from different PPT results into a new final PPTX file.

Requirements covered: PPT-01, PPT-02, PPT-03, PPT-04, PPT-05, PPT-06, UI-03.

</domain>

<decisions>
## Implementation Decisions

### PPT Preview (PPT-03)
- **D-01:** Server-side slide image extraction via Python-pptx (already integrated in PPTXGenerator). Each slide is converted to JPEG image served via API.
- **D-02:** Preview layout: main view (large slide) + sidebar thumbnail strip. Similar to PowerPoint's thumbnail sidebar.
- **D-03:** Thumbnails generated on-demand and cached. First preview shows a brief loading/progress indicator while images are extracted.
- **D-04:** Dual resolution strategy: thumbnails at 200x112px (fast loading), main view images at 1920x1080px (high clarity).
- **D-05:** Full-screen presentation mode supported — hides sidebar and navigation, slides fill entire screen.
- **D-06:** Page indicator ("第 3/25 页") displayed below main view with click-to-jump input for direct page navigation.
- **D-07:** Per-slide actions: single page download + copy image to clipboard.
- **D-08:** Slide images extracted as JPEG quality 90%. Balance of file size and visual clarity for PPT screenshots.
- **D-09:** API design: GET /api/v1/ppts/:id/slides returns list of slide image URLs (thumbnail + full-size pairs).

### Multi-result Display (PPT-04, PPT-05)
- **D-10:** Gallery-style switching for multiple PPT results of the same video. Horizontal thumbnail strip at bottom, current result displayed prominently.
- **D-11:** "重新转录" button lives inside the result page action panel. Reuses transcription trigger logic with TranscriptionProgressModal.
- **D-12:** Default selection: newest transcription result first. User can switch to any historical result via gallery strip.

### Slide Merge (PPT-06)
- **D-13:** Merge triggered from result page — "合并幻灯片" button enters merge mode inline (no page navigation).
- **D-14:** Slide selection: click-to-select on thumbnails (highlight on select, click again to deselect). Selected slides appear in a bottom bar with drag-to-reorder support.
- **D-15:** Merge result generates a new PPTX file saved on server, associated with the original video. Does not modify source PPTs.
- **D-16:** Merged PPT appears in the result gallery alongside transcription results, associated with original video.
- **D-17:** Merge limit: 200 slides maximum. UI shows selected count and limit indicator.
- **D-18:** Merge progress: simple loading spinner + completion toast. No detailed progress needed (merge is typically fast).

### Result Page Layout (UI-03)
- **D-19:** Left-right split layout: left side = PPT preview area (main view + sidebar thumbnails), right side = info/action panel.
- **D-20:** Navigation entry: "预览PPT" button in file list action column jumps to result detail page.
- **D-21:** Right panel contains three sections: basic info (video name, transcription time, sampling rate, page count, file size), action buttons (download, re-transcribe, merge, delete), multi-result gallery switcher (horizontal strip showing all results with time + page count).
- **D-22:** Result page URL pattern: /results/:videoFileId (shows all PPT results for that video).

### Claude's Discretion
- Exact Python-pptx slide extraction implementation (image rendering approach)
- Slide image caching strategy (file system vs database blob, cache invalidation)
- Merge PPTX generation approach (re-extract original frames vs combine existing images)
- PPTFile model extensions needed for merge results (source_type field, merged_from IDs)
- API endpoint paths and request/response structures for slide images and merge operations
- Thumbnail strip component implementation details
- Drag-to-reorder library choice for merge selection bar
- Merge mode UI state management (entering/exiting merge mode, selection state)
- Error handling for slide extraction failures
- How to handle deleted source PPTs in merge results

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### PPT management requirements
- `.planning/REQUIREMENTS.md` §PPT Management — PPT-01 through PPT-06 acceptance criteria
- `.planning/REQUIREMENTS.md` §UI Layout — UI-03 acceptance criteria
- `.planning/ROADMAP.md` §Phase 3 — Success criteria and phase boundary

### Phase 2 context (prior decisions and patterns)
- `.planning/phases/02-local-transcription/02-CONTEXT.md` — PPTFile model, TranscriptionTask model, PPTXGenerator, transcription progress polling pattern, file list "转录" button placement

### Phase 1 context (prior decisions)
- `.planning/phases/01-video-splitting/01-CONTEXT.md` — File list source column, action button patterns, auto-refresh

### Existing code to reuse or extend
- `internal/models/ppt_file.go` — PPTFile model with SourceVideoFileID, TranscriptionTaskID foreign keys
- `internal/models/transcription_task.go` — TranscriptionTask model with ResultPPTFileID, status tracking
- `internal/services/pptx_generator.go` — Python-based PPTX generation (Python-pptx already available for slide extraction)
- `internal/handlers/transcription_handler.go` — Transcription API pattern (submit, status)
- `frontend/src/components/TranscriptionProgressModal.tsx` — Reuse for "重新转录" trigger
- `frontend/src/pages/files/index.tsx` — File list action column (add "预览PPT" button)
- `frontend/src/api/transcription.ts` — API client pattern
- `frontend/src/types/transcription.ts` — Transcription types

### Project constraints
- `.planning/PROJECT.md` — Tech stack (Go 1.24/Gin, React 19/Ant Design 6, SQLite/GORM, FFmpeg)
- `.planning/STATE.md` — Tech stack context, critical pitfalls

No external specs — requirements are fully captured in REQUIREMENTS.md and PROJECT.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **PPTXGenerator** (`internal/services/pptx_generator.go`): Already integrates Python-pptx via Python script. Can be extended or a new SlideExtractor service created to extract slide images using the same Python-pptx library.
- **PPTFile model** (`internal/models/ppt_file.go`): Has SourceVideoFileID, TranscriptionTaskID, FilePath, PageCount. Needs extension: add slide_cache_path field, possibly source_type to distinguish merge results.
- **TranscriptionTask model** (`internal/models/transcription_task.go`): Has ResultPPTFileID linking task to PPT result. The one-to-many relationship (one video → many tasks → many PPTs) already exists.
- **TranscriptionHandler** (`internal/handlers/transcription_handler.go`): Handler pattern for transcription APIs. Same pattern for new PPT-related endpoints.
- **TranscriptionProgressModal** (`frontend/src/components/TranscriptionProgressModal.tsx`): Reusable for "重新转录" trigger from result page.
- **File list action column** (`frontend/src/pages/files/index.tsx`): Add "预览PPT" button here, next to existing download/delete/split buttons.

### Established Patterns
- **API handlers**: Gin handlers with `ShouldBindQuery`/`ShouldBindJSON`, unified `response.GinSuccess/GinError`
- **Frontend state**: Local useState for page state, API calls via `apiRequest<T>()`
- **Frontend polling**: setInterval every 2-5 seconds for progress tracking
- **Python scripts**: Python tools invoked via exec.CommandContext from Go services
- **File storage**: Files stored in recordings directory, paths relative to project root

### Integration Points
- **PPTFile model**: Needs slide_cache_path field and source_type (transcription/merge)
- **TranscriptionTask model**: Already has ResultPPTFileID — query tasks by VideoFileID to get all PPT results
- **File list page**: Add "预览PPT" button, conditionally shown when PPT exists for a video
- **New result page**: /results/:videoFileId — new route and page component
- **cmd/server/app.go**: Register new PPT handler and routes
- **Python script**: New or extended script for slide image extraction

</code_context>

<specifics>
## Specific Ideas

- Preview should feel like PowerPoint's reading view — main slide with sidebar thumbnails
- Gallery switcher at bottom of right panel should show compact cards with transcription timestamp + page count
- Merge mode is an inline state change: thumbnails become selectable, bottom bar appears with selected items in order
- "预览PPT" button should be visually distinct (perhaps primary color) to stand out in the action column

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 03-ppt-management*
*Context gathered: 2026-04-17*
