---
phase: 05-file-rename
plan: 01
subsystem: file-management
tags: [rename, video-files, ppt-files, atomic-operations, ownership-validation]

# Dependency graph
requires:
  - phase: 01-video-splitting
    provides: [VideoFile model with source_type and parent_id fields]
  - phase: 03-ppt-management
    provides: [PPTFile model with SourceVideoFileID relation]
provides:
  - [RenameVideoFile and RenamePPTFile service methods with atomic DB+filesystem operations]
  - [Rename API endpoints POST /api/v1/videos/:id/rename and POST /api/v1/ppts/:id/rename]
  - [Frontend rename UI modal and API client functions]
  - [FILE_EDIT permission constant for access control]
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: [atomic-rename-transaction, ownership-validation-via-source-video, extension-preservation, path-traversal-prevention]

key-files:
  created: []
  modified:
    - internal/services/video_file_service.go - RenameVideoFile method
    - internal/services/ppt_file_service.go - RenamePPTFile method
    - internal/handlers/video_file_handler.go - RenameFile handler
    - internal/handlers/ppt_handler.go - RenamePPT handler
    - internal/handlers/utils.go - trimString and containsPathSeparator helpers
    - cmd/server/app.go - registered rename routes
    - frontend/src/api/video-file.ts - renameVideoFile function
    - frontend/src/api/ppt.ts - renamePptFile function
    - frontend/src/pages/files/index.tsx - rename modal and actions
    - frontend/src/utils/permissions.ts - FILE_EDIT permission

key-decisions:
  - "Used os.Rename for atomic filesystem operations with transaction rollback on failure"
  - "Original recordings (source_type='recording' && parent_id=NULL) are immutable per D-13"
  - "File extension preservation enforced at service layer, not client input"
  - "Ownership validation via CreatedBy field for both VideoFile and PPTFile (via SourceVideoFile)"
  - "Path separator rejection (/\\) prevents directory traversal attacks"

patterns-established:
  - "Atomic rename pattern: Start transaction → rename physical file → update DB → commit/rollback"
  - "Ownership validation pattern: Load file → check CreatedBy == userID → return forbidden if mismatch"
  - "Extension preservation pattern: Extract ext from current path → append to newName → validate"
  - "Helper functions pattern: trimString and containsPathSeparator in utils.go for reuse"

requirements-completed: []

# Metrics
duration: 45min
completed: 2026-04-20
---

# Phase 05: File Rename Summary

**Atomic file rename for split videos and PPTs with ownership validation, extension preservation, and rollback on failure**

## Performance

- **Duration:** 45 minutes
- **Started:** 2026-04-20T12:33:00Z
- **Completed:** 2026-04-20T13:18:00Z
- **Tasks:** 6
- **Files modified:** 10

## Accomplishments

- Implemented RenameVideoFile service method with atomic DB+filesystem operations (6 tests passing)
- Implemented RenamePPTFile service method with slide cache directory rename (6 tests passing)
- Created rename API endpoints with auth middleware and request validation
- Registered rename routes in cmd/server/app.go with MultiAuth protection
- Added frontend rename API client functions (renameVideoFile, renamePptFile)
- Implemented rename UI modal in file management page with extension preservation display

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement RenameVideoFile service method** - `ecdb774` (test)
2. **Task 2: Implement RenamePPTFile service method** - `e0d7bf3` (feat)
3. **Task 3-4: Implement rename API endpoints and register routes** - `14cf937` (feat)
4. **Task 5: Implement frontend rename API client functions** - `5801eae` (feat)
5. **Task 6: Implement rename UI modal and actions** - `e411b31` (feat)

**Plan metadata:** N/A (no separate docs commit)

## Files Created/Modified

- `internal/services/video_file_service.go` - RenameVideoFile method with ownership validation, immutability guard, extension preservation
- `internal/services/ppt_file_service.go` - RenamePPTFile method with slide cache path update
- `internal/handlers/video_file_handler.go` - RenameFile handler with request validation and error mapping
- `internal/handlers/ppt_handler.go` - RenamePPT handler with ownership verification
- `internal/handlers/utils.go` - trimString and containsPathSeparator helper functions
- `cmd/server/app.go` - Registered POST /api/v1/videos/:id/rename and POST /api/v1/ppts/:id/rename routes
- `frontend/src/api/video-file.ts` - renameVideoFile function and RenameRequest interface
- `frontend/src/api/ppt.ts` - renamePptFile function and PPTFile interface
- `frontend/src/pages/files/index.tsx` - Rename modal, state, handlers, and button in actions
- `frontend/src/utils/permissions.ts` - Added FILE_EDIT permission constant

## Decisions Made

1. **Used os.Rename for atomic filesystem operations** - Ensures atomicity at filesystem level; if os.Rename fails, DB transaction is rolled back
2. **Original recordings are immutable** - Enforced at service layer by checking source_type='recording' && parent_id=NULL
3. **Extension preservation enforced at service layer** - Client cannot change file extension; extracted from current path and appended to new name
4. **Ownership validation via CreatedBy field** - For VideoFile: check CreatedBy; for PPTFile: check SourceVideoFile.CreatedBy
5. **Path separator rejection prevents directory traversal** - Rejects / and \\ characters in new_name input

## Deviations from Plan

None - plan executed exactly as written

## Issues Encountered

1. **CGO_ENABLED=0 causing SQLite test failures** - Fixed by running `go clean -testcache` and `CGO_ENABLED=1 go test`
2. **Missing trimString and containsPathSeparator helper functions** - Added to utils.go using strings.TrimSpace and strings.ContainsAny
3. **Missing gorm import in video_file_handler.go** - Added "gorm.io/gorm" import for ErrRecordNotFound comparison

All issues were resolved during implementation without blocking progress.

## User Setup Required

None - no external service configuration required

## Next Phase Readiness

- File rename feature complete and ready for testing
- All rename endpoints protected with MultiAuth middleware (SM4 Token + API Key)
- Frontend UI includes permission guard (FILE_EDIT) and hides rename for original recordings
- Atomic operations ensure data consistency on failure
- No blockers or concerns for next phase

---
*Phase: 05-file-rename*
*Completed: 2026-04-20*
