package handlers

import (
	"testing"
)

// --- SplitHandler test stubs (Wave 0) ---

func TestSplitHandler_SubmitSplit(t *testing.T) {
	t.Skip("waiting for SplitHandler implementation")
	// SPLIT-04: POST /api/v1/videos/:id/split with markers
	// Setup: create test services, create gin test context with JSON body
	// Action: call SubmitSplit handler
	// Assert: 200 response with status=processing
}

func TestSplitHandler_SubmitSplit_InvalidMarkers(t *testing.T) {
	t.Skip("waiting for SplitHandler implementation")
	// SPLIT-04: Validation of marker input
	// Test: empty markers -> 400, >20 markers -> 400, non-numeric -> 400
}

func TestSplitHandler_SubmitSplit_UnauthorizedVideo(t *testing.T) {
	t.Skip("waiting for SplitHandler implementation")
	// SPLIT-04: User cannot split another user's video
	// Setup: create video owned by different user
	// Action: call SubmitSplit as non-owner non-admin
	// Assert: 403 or error response
}

func TestSplitHandler_GenerateSnapshot(t *testing.T) {
	t.Skip("waiting for SplitHandler implementation")
	// SNAP-02: POST /api/v1/tasks/:id/snapshot
	// Setup: create test services, create gin test context
	// Action: call GenerateSnapshot handler
	// Assert: 200 response with snapshot_file_id
}

func TestSplitHandler_GetSplitStatus(t *testing.T) {
	t.Skip("waiting for SplitHandler implementation")
	// SPLIT-04: GET /api/v1/videos/:id/split-status
	// Setup: submit a split, then check status
	// Action: call GetSplitStatus
	// Assert: returns current status
}

func TestSplitHandler_GetSegments(t *testing.T) {
	t.Skip("waiting for SplitHandler implementation")
	// SPLIT-04: GET /api/v1/videos/:id/segments
	// Setup: create test DB with segments linked to parent
	// Action: call GetSegments
	// Assert: returns array of segment VideoFile records
}
