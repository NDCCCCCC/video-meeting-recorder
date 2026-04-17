package services

import (
	"testing"

	"gorm.io/gorm"
)

// --- SplittingService test stubs (Wave 0) ---

func TestSplittingService_SubmitSplit(t *testing.T) {
	t.Skip("waiting for SplittingService implementation")
	// SPLIT-03: Submit split task with markers, verify worker processes it
	// Setup: create test DB, mock FFmpeg path, create source VideoFile
	// Action: call SubmitSplit with videoFileID and markers
	// Assert: status becomes "processing"
}

func TestSplittingService_ProcessSplit_SingleMarker(t *testing.T) {
	t.Skip("waiting for SplittingService implementation")
	// SPLIT-03: Single marker splits video into 2 segments
	// Setup: create test DB, create source VideoFile record, create temp video file
	// Action: submit split with markers [10.0]
	// Assert: 2 segment files created, both registered via CreateSegmentFile
}

func TestSplittingService_ProcessSplit_MultipleMarkers(t *testing.T) {
	t.Skip("waiting for SplittingService implementation")
	// SPLIT-03: Multiple markers split into N+1 segments
	// Setup: create test DB, create source VideoFile, markers [10.0, 30.0, 50.0]
	// Action: submit split with 3 markers
	// Assert: 4 segments created with correct parent_id and source_type=split
}

func TestSplittingService_GetSplitStatus(t *testing.T) {
	t.Skip("waiting for SplittingService implementation")
	// SPLIT-03: Status tracking for split operations
	// Setup: create service, submit a split task
	// Action: call GetSplitStatus
	// Assert: returns "processing" during work, "completed" after
}

func TestSplittingService_MarkerValidation(t *testing.T) {
	t.Skip("waiting for SplittingService implementation")
	// SPLIT-03: Markers must be validated (range, count, sorted)
	// Test: empty markers -> error, >20 markers -> error, unsorted -> auto-sorted
}

// setupSplitTestDB creates a test database for split service tests
func setupSplitTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// Reuse setupTestDB from video_file_service_test.go (same package)
	// Will be implemented when SplittingService is built
	return setupTestDB(t)
}
