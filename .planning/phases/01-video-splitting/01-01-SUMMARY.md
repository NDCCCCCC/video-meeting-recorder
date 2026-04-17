---
phase: 01-video-splitting
plan: 01
subsystem: database
tags: [gorm, sqlite, migration, video-file-model, parent-child, segments]

# Dependency graph
requires:
  - phase: 00
    provides: base VideoFile model, migration infrastructure
provides:
  - Extended VideoFile model with parent_id, source_type, snapshot_offset fields
  - Migration 003_add_segment_fields for database schema changes
  - CreateSegmentFile service method for creating segment/snapshot records
  - GetSegmentsByParentID service method for retrieving child segments
  - Frontend VideoFile type with parent_id, source_type, snapshot_offset fields
affects: [01-02-split-handlers, 01-03-snapshot-service, 01-04-auto-scan]

# Tech tracking
tech-stack:
  added: []
  patterns: [parent-child video file relationships, idempotent migrations with pragma_table_info, segment file metadata extraction]

key-files:
  created: []
  modified: [internal/models/video_file.go, internal/migrations/001_add_video_file_owner.go, internal/services/video_file_service.go, frontend/src/types/video-file.ts]

key-decisions:
  - "source_type uses VARCHAR(20) with no CHECK constraint — validation at application layer"
  - "parent_id is nullable — NULL means original recording (not a segment/snapshot)"
  - "snapshot_offset defaults to 0 — stores seek offset for incremental snapshots (D-15)"

patterns-established:
  - "Migration idempotence pattern: check pragma_table_info before ALTER TABLE ADD COLUMN"
  - "Segment file creation: extract metadata via ffprobe, inherit RecordedAt from parent, set source_type"
  - "Parent-child query pattern: WHERE parent_id = ? ORDER BY id ASC for retrieving segments"

requirements-completed: [SPLIT-04, SCAN-01]

# Metrics
duration: 2min
completed: 2026-04-17T03:58:04Z
---

# Phase 01: Video Splitting - Plan 01 Summary

**VideoFile model extended with parent_id, source_type, snapshot_offset fields supporting split segments and incremental snapshots with idempotent migration 003**

## Performance

- **Duration:** 2 min
- **Started:** 2026-04-17T03:56:20Z
- **Completed:** 2026-04-17T03:58:04Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments

- Extended VideoFile model with ParentID, SourceType, SnapshotOffset, and Parent association fields
- Created migration 003_add_segment_fields with idempotent column additions using pragma_table_info pattern
- Added source type constants (recording, snapshot, split) to models package
- Implemented CreateSegmentFile service method with metadata extraction and parent RecordedAt inheritance
- Added source_type filter to ListFilesRequest and applyFilters method
- Implemented GetSegmentsByParentID method for retrieving child segments
- Updated frontend VideoFile type with parent_id, source_type, snapshot_offset, and parent fields
- Added source_type filter to VideoFileListParams interface

## Task Commits

Each task was committed atomically:

1. **Task 1: Extend VideoFile model and add migration for parent_id / source_type** - `1435609` (feat)
2. **Task 2: Add CreateSegmentFile method and source_type filter to VideoFileService** - `e222ad9` (feat)

**Plan metadata:** (not yet committed - orchestrator will commit)

## Files Created/Modified

- `internal/models/video_file.go` - Added ParentID, SourceType, SnapshotOffset, Parent fields and source type constants
- `internal/migrations/001_add_video_file_owner.go` - Added AddSegmentFieldsMigration (003) with idempotent column additions
- `internal/services/video_file_service.go` - Added CreateSegmentFile and GetSegmentsByParentID methods, source_type filter
- `frontend/src/types/video-file.ts` - Added parent_id, source_type, snapshot_offset, parent fields and source_type filter

## Decisions Made

- **source_type validation at application layer** — No CHECK constraint in SQLite, validation enforced by handler layer against allowed constants (recording/snapshot/split)
- **parent_id nullable for original recordings** — NULL indicates an original recording, non-NULL links to parent video for segments/snapshots
- **snapshot_offset defaults to 0** — Stores the seek offset for incremental snapshots (D-15), used to track position in parent video for incremental extraction

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - all code compiled without errors on first attempt.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- VideoFile model supports parent-child relationships required for split segments
- CreateSegmentFile method ready for handler integration (Plan 02)
- GetSegmentsByParentID method ready for segment listing (Plan 02)
- Migration 003 ready for deployment (idempotent, safe to re-run)
- Frontend types support new fields for UI display (Plan 05)

**No blockers or concerns.** All deliverables complete and verified.

---
*Phase: 01-video-splitting*
*Plan: 01*
*Completed: 2026-04-17*
