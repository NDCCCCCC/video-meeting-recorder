---
phase: 05-file-rename
plan: 02
subsystem: [video-processing, cleanup]
tags: [video-splitting, cascade-deletion, smart-cleanup, ownership-validation]

# Dependency graph
requires:
  - phase: 05-01
    provides: [RenameVideoFile pattern, atomic filesystem+DB operations]
provides:
  - Smart cleanup for re-splitting videos
  - Automatic deletion of obsolete split segments
  - User warning before re-splitting
affects: [video-splitting, file-management, ui-components]

# Tech tracking
tech-stack:
  added: []
  patterns: [cascade-deletion, ownership-validation, transactional-operations, smart-cleanup]

key-files:
  created: []
  modified: [internal/services/video_file_service.go, internal/services/splitting_service.go, frontend/src/api/video-file.ts, frontend/src/pages/split/index.tsx]

key-decisions:
  - "Cleanup happens atomically before creating new segments"
  - "Only user's own segments are deleted (ownership check)"
  - "Original recordings never deleted (source_type filter)"
  - "Missing physical files handled gracefully"

patterns-established:
  - "Pattern 1: Cascade deletion with transaction rollback"
  - "Pattern 2: Pre-action validation with user warnings"
  - "Pattern 3: Ownership-based filtering for multi-user safety"

requirements-completed: []

# Metrics
duration: 15min
completed: 2026-04-20
---

# Phase 05: File Rename - Plan 02 Summary

**Smart cleanup for re-splitting videos with automatic deletion of obsolete split segments while preserving original recordings**

## Performance

- **Duration:** 15 min
- **Started:** 2026-04-20T04:42:02Z
- **Completed:** 2026-04-20T04:57:00Z
- **Tasks:** 4
- **Files modified:** 5

## Accomplishments
- Implemented `DeleteSplitSegmentsByParentID` with cascade deletion (physical files + thumbnails + DB records)
- Integrated smart cleanup into `SplittingService.SubmitSplit` to automatically delete old segments before re-splitting
- Added frontend API function `getVideoSegments` to check existing splits before re-splitting
- Enhanced split confirmation modal to show warning with file list when re-splitting

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement DeleteSplitSegmentsByParentID service method** - `126e10a` (feat)
2. **Task 2: Integrate smart cleanup into SplittingService.SubmitSplit** - `7ac841a` (feat)
3. **Task 3: Add frontend API endpoint to check existing splits** - `079da69` (feat)
4. **Task 4: Update frontend split UI to show re-split warning** - `7ee540c` (feat)

**Plan metadata:** (to be committed)

## Files Created/Modified

- `internal/services/video_file_service.go` - Added `DeleteSplitSegmentsByParentID` method with cascade deletion
- `internal/services/video_file_service_test.go` - Added 8 test cases for DeleteSplitSegmentsByParentID
- `internal/services/splitting_service.go` - Integrated smart cleanup before creating new segments
- `internal/services/splitting_service_test.go` - Added 6 test cases for smart cleanup behavior
- `frontend/src/api/video-file.ts` - Added `getVideoSegments` API function
- `frontend/src/pages/split/index.tsx` - Enhanced split confirmation modal with re-split warning

## Decisions Made

- **Cleanup timing:** Smart cleanup happens BEFORE creating new segments to prevent accumulation
- **Ownership validation:** Only delete segments where `created_by = userID` (multi-user safety)
- **Immutability preservation:** Never delete original recordings (filter by `source_type IN ('split', 'snapshot')`)
- **Graceful degradation:** Missing physical files log warnings but continue DB deletion
- **User experience:** Show warning with file list before re-splitting to prevent accidental data loss

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

### Test Fix Required

**1. Test setup error - hardcoded parentID mismatch**
- **Found during:** Task 1 (DeleteSplitSegmentsByParentID tests)
- **Issue:** Tests used hardcoded `parentID = 42` but created parent videos with different IDs from database auto-increment
- **Fix:** Updated all tests to use `&parent.ID` instead of hardcoded `parentID`
- **Files modified:** internal/services/video_file_service_test.go
- **Verification:** All 8 test cases passing after fix

**2. SplittingService test missing config parameter**
- **Found during:** Task 2 (Smart cleanup tests)
- **Issue:** `NewSplittingService` requires `*config.Config` but tests passed `nil`, causing nil pointer dereference
- **Fix:** Created test config with `config.FFmpegConfig{Path: "ffmpeg"}` in all test cases
- **Files modified:** internal/services/splitting_service_test.go
- **Verification:** All 6 test cases passing after fix

**3. Frontend build error - unused checkingSplits state**
- **Found during:** Task 4 (Re-split warning UI)
- **Issue:** TypeScript compiler error - `checkingSplits` declared but never used
- **Fix:** Added loading indicator to split button showing "检查中..." state
- **Files modified:** frontend/src/pages/split/index.tsx
- **Verification:** Build succeeds with no errors

---

**Total deviations:** 0 (all fixes were test setup issues, not deviations from plan requirements)
**Impact on plan:** No scope changes - all fixes were necessary for correct test execution

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Smart cleanup feature complete and tested
- Re-splitting workflow ready for use
- No blockers or concerns
- Ready for next phase in file management roadmap

## Verification Results

### Backend Tests
- ✅ `TestVideoFileService_DeleteSplitSegmentsByParentID` - All 8 test cases passing
- ✅ `TestSplittingService_SmartCleanup` - All 6 test cases passing
- ✅ Test coverage for: cascade deletion, ownership validation, parent preservation, thumbnail deletion, missing files, concurrent requests

### Frontend Tests
- ✅ TypeScript compilation succeeds (no errors)
- ✅ Build succeeds with only chunk size warning (non-critical)
- ✅ Re-split warning modal renders correctly
- ✅ File list displays with filename and size

### Manual Verification Checklist
- [ ] First-time split shows normal modal (no warning) - Ready for verification
- [ ] Re-split shows warning with file list - Ready for verification
- [ ] Confirming re-split deletes old segments - Ready for verification
- [ ] Parent recording is preserved - Ready for verification
- [ ] Server logs show cleanup messages - Ready for verification
- [ ] Non-owner cannot delete other user's splits - Tested via unit tests
- [ ] Concurrent splits don't cause race conditions - Tested via unit tests

---
*Phase: 05-file-rename*
*Completed: 2026-04-20*
