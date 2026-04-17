---
phase: 03-ppt-management
plan: 01
subsystem: api
tags: [python-pptx, golang, gin, gorm, slide-extraction, ppt-merge, dual-resolution-caching]

# Dependency graph
requires:
  - phase: 02-local-transcription
    provides: [PPTFile model, TranscriptionTask model, PPTXGenerator service]
provides:
  - PPTFile model extensions (SlideCachePath, SourceType, MergedFrom)
  - Python scripts for slide extraction and merge (extract_slides.py, merge_slides.py)
  - SlideExtractor service for Python script execution
  - SlideCacheService for dual-resolution on-demand caching
  - PPTMergeService for slide merge with 200-slide limit
  - PPTFileService for CRUD operations
  - PPThandler with 6 API endpoints (slides, merge, download, delete)
affects: [03-ppt-management-02]

# Tech tracking
tech-stack:
  added: [python-pptx for slide extraction, PIL for image processing]
  patterns: [Python JSON IPC pattern, path traversal prevention, on-demand caching, ownership validation]

key-files:
  created:
    - internal/models/slide_merge.go
    - internal/services/slide_extractor.go
    - internal/services/slide_cache_service.go
    - internal/services/ppt_merge_service.go
    - internal/services/ppt_file_service.go
    - internal/handlers/ppt_handler.go
    - scripts/extract_slides.py
    - scripts/merge_slides.py
  modified:
    - internal/models/ppt_file.go (added SlideCachePath, SourceType, MergedFrom)
    - internal/migrations/001_add_video_file_owner.go (added 005_add_ppt_cache_fields)
    - cmd/server/app.go (wired PPT services and routes)

key-decisions:
  - "Embedded-image-only extraction for MVP (python-pptx cannot render slides to images)"
  - "Dual-resolution caching (200x112 thumbnails, 1920x1080 full-size) for fast preview"
  - "200-slide merge limit enforced server-side (per D-17) to prevent DoS"
  - "Path traversal prevention via strict filename validation and prefix checks (T-03-01)"
  - "Ownership validation on all endpoints using middleware.GetUserID pattern"

patterns-established:
  - "Python JSON IPC: stdout JSON output, stderr errors, exit codes 0/1"
  - "On-demand caching: check cache miss → extract → serve → update DB"
  - "Strict filename whitelist: slide_\\d{3}\\.jpg pattern only"
  - "Service composition: SlideCacheService uses SlideExtractor, PPTMergeService uses SlideCacheService"

requirements-completed: [PPT-01, PPT-02, PPT-03, PPT-04, PPT-05, PPT-06]

# Metrics
duration: 4min
completed: 2026-04-17
---

# Phase 03 Plan 01: PPT Management Backend Summary

**Backend foundation for PPT preview, multi-result management, and slide merge with dual-resolution caching, Python-pptx integration, and secure API endpoints**

## Performance

- **Duration:** 4 min (236 seconds)
- **Started:** 2026-04-17T10:00:59Z
- **Completed:** 2026-04-17T10:05:15Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments

- Extended PPTFile model with SlideCachePath, SourceType, and MergedFrom fields for cache tracking and merge result support
- Created Python scripts for dual-resolution slide extraction (thumbnails + full-size) and slide merge from multiple PPTX files
- Built Go services following established patterns: SlideExtractor (Python execution), SlideCacheService (on-demand caching), PPTMergeService (merge logic), PPTFileService (CRUD)
- Implemented PPThandler with 6 API endpoints including ownership validation, path traversal prevention, and 200-slide merge limit
- Wired all services into app.go with proper route registration under /api/v1/ppts and /api/v1/videos/:id/ppts

## Task Commits

Each task was committed atomically:

1. **Task 1: PPTFile model extension + migration + Python scripts + Go services** - `c262349` (feat)
2. **Task 2: PPThandler API endpoints + app.go wiring** - `f8d2223` (feat)

**Plan metadata:** (docs commit pending)

## Files Created/Modified

- `internal/models/ppt_file.go` - Added SlideCachePath, SourceType, MergedFrom fields + PPTSourceType constants
- `internal/models/slide_merge.go` - MergeRequest and MergeSlideItem structs for merge API
- `internal/migrations/001_add_video_file_owner.go` - Added 005_add_ppt_cache_fields migration
- `internal/services/slide_extractor.go` - Python script execution for slide image extraction
- `internal/services/slide_cache_service.go` - Dual-resolution on-demand caching with path traversal prevention
- `internal/services/ppt_merge_service.go` - Slide merge logic with 200-slide limit validation
- `internal/services/ppt_file_service.go` - CRUD operations with GetPptsByVideoFile (newest first)
- `internal/handlers/ppt_handler.go` - 6 API endpoints: GetSlides, ServeSlideImage, GetPptsByVideo, MergeSlides, DownloadPPT, DeletePPT
- `scripts/extract_slides.py` - Extract embedded images as dual-resolution JPEGs (MVP: no text/shapes rendering)
- `scripts/merge_slides.py` - Merge selected slides from multiple PPTX into one
- `cmd/server/app.go` - Wired PPT services, added PPT to Handlers struct, registered routes

## Decisions Made

- **Embedded-image-only extraction for MVP**: python-pptx cannot render slides to images (no layout engine), so extract only embedded images. Text/shapes not shown. Acceptable for MVP per RESEARCH.md recommendation.
- **Dual-resolution caching**: 200x112 thumbnails (fast loading), 1920x1080 full-size (high clarity). Cache per PPT in recordings/ppts/{id}/slides/
- **200-slide merge limit**: Enforced server-side in PPTMergeService per D-17 to prevent DoS
- **Path traversal prevention**: Strict filename validation (slide_XXX.jpg only) + resolved path prefix check per T-03-01
- **Ownership validation on all endpoints**: Reused middleware.GetUserID + GetIsAdmin pattern from TranscriptionHandler

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all services and handlers compiled successfully, no runtime issues.

## Known Stubs

None - all functionality implemented as specified. No hardcoded empty values or placeholders that flow to UI.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: path_traversal | internal/services/slide_cache_service.go | Strict filename whitelist + prefix check implemented (T-03-01) |
| threat_flag: dos | internal/services/ppt_merge_service.go | 200-slide limit enforced (T-03-02) |
| threat_flag: spoofing | internal/handlers/ppt_handler.go | Ownership validation on all endpoints (T-03-03, T-03-04) |
| threat_flag: info_disclosure | internal/handlers/ppt_handler.go | Generic error messages, no paths exposed (T-03-05) |
| threat_flag: tampering | internal/services/slide_extractor.go | Path validation from PPTXGenerator reused (T-03-06) |

## Next Phase Readiness

- Backend API endpoints ready for frontend consumption (Plan 02)
- All threat mitigations implemented per threat model
- No external service configuration required (Python scripts use installed python-pptx)
- Frontend can now implement PPT preview, multi-result gallery, and merge UI

## Self-Check: PASSED

- ✓ SUMMARY.md created at .planning/phases/03-ppt-management/03-01-SUMMARY.md
- ✓ Task 1 commit exists: c262349 (PPTFile model extension, Python scripts, Go services)
- ✓ Task 2 commit exists: f8d2223 (PPThandler and app.go wiring)
- ✓ All acceptance criteria met (models, services, handlers, routes, build passes)

---
*Phase: 03-ppt-management*
*Completed: 2026-04-17*
