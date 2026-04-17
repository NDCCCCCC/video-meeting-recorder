package services

import (
	"testing"
)

// --- SnapshotService test stubs (Wave 0) ---

func TestSnapshotService_GenerateSnapshot(t *testing.T) {
	t.Skip("waiting for SnapshotService implementation")
	// SNAP-02: Generate MP4 snapshot from active recording
	// Setup: create test DB, create VideoRecordingTask with status=recording
	// Action: call GenerateSnapshot
	// Assert: returns VideoFile with source_type=snapshot, parent_id set
}

func TestSnapshotService_GenerateSnapshot_NotRecording(t *testing.T) {
	t.Skip("waiting for SnapshotService implementation")
	// SNAP-02: Cannot generate snapshot when task is not recording
	// Setup: create test DB, create task with status=completed
	// Action: call GenerateSnapshot
	// Assert: returns error "任务不在录制状态"
}

func TestSnapshotService_GenerateSnapshot_FileNotFound(t *testing.T) {
	t.Skip("waiting for SnapshotService implementation")
	// SNAP-02: Error when MKV file does not exist
	// Setup: create test DB, create task with status=recording but missing MKV path
	// Action: call GenerateSnapshot
	// Assert: returns error containing "录制文件不存在"
}
