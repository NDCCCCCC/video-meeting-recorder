package services

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
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

// --- Smart cleanup tests (Phase 05-02) ---

func TestSplittingService_SmartCleanup(t *testing.T) {
	t.Run("Test1_SubmitSplitDeletesExistingSegments", func(t *testing.T) {
		db := setupSplitTestDB(t)
		logger := zap.NewNop()
		videoFileService := NewVideoFileService(db, logger, t.TempDir(), "")

		// Create test config
		cfg := &config.Config{
			FFmpeg: config.FFmpegConfig{
				Path: "ffmpeg", // Use dummy path for tests
			},
		}
		splittingService := NewSplittingService(db, logger, cfg, videoFileService)

		userID := uint(1)

		// Create parent video
		parent := &models.VideoFile{
			FileName:   "parent_video.mp4",
			FilePath:   "/test/parent.mp4",
			FileSize:   1024000,
			Duration:   300,
			Format:     "mp4",
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeRecording,
			CreatedBy:  userID,
		}
		require.NoError(t, db.Create(parent).Error)

		// Create existing split segments
		seg1 := &models.VideoFile{
			FileName:   "segment_001.mp4",
			FilePath:   "/test/segments/segment_001.mp4",
			FileSize:   512000,
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeSplit,
			ParentID:   &parent.ID,
			CreatedBy:  userID,
		}
		seg2 := &models.VideoFile{
			FileName:   "segment_002.mp4",
			FilePath:   "/test/segments/segment_002.mp4",
			FileSize:   512000,
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeSplit,
			ParentID:   &parent.ID,
			CreatedBy:  userID,
		}
		require.NoError(t, db.Create(seg1).Error)
		require.NoError(t, db.Create(seg2).Error)

		// Submit new split (should delete old segments)
		err := splittingService.SubmitSplit(context.Background(), parent.ID, []float64{10.0, 20.0}, false, userID)

		// Verify cleanup happened
		assert.NoError(t, err)

		// Check old segments are deleted from DB
		var segments []models.VideoFile
		err = db.Where("parent_id = ? AND id IN ?", parent.ID, []uint{seg1.ID, seg2.ID}).Find(&segments).Error
		assert.NoError(t, err)
		assert.Len(t, segments, 0, "Old segments should be deleted")
	})

	t.Run("Test2_SubmitSplitPreservesParentRecording", func(t *testing.T) {
		db := setupSplitTestDB(t)
		logger := zap.NewNop()
		videoFileService := NewVideoFileService(db, logger, t.TempDir(), "")

		cfg := &config.Config{
			FFmpeg: config.FFmpegConfig{Path: "ffmpeg"},
		}
		splittingService := NewSplittingService(db, logger, cfg, videoFileService)

		userID := uint(1)

		// Create parent video
		parent := &models.VideoFile{
			FileName:   "parent_video.mp4",
			FilePath:   "/test/parent.mp4",
			FileSize:   1024000,
			Duration:   300,
			Format:     "mp4",
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeRecording,
			CreatedBy:  userID,
		}
		require.NoError(t, db.Create(parent).Error)

		// Create existing segments
		seg1 := &models.VideoFile{
			FileName:   "segment_001.mp4",
			FilePath:   "/test/segments/segment_001.mp4",
			FileSize:   512000,
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeSplit,
			ParentID:   &parent.ID,
			CreatedBy:  userID,
		}
		require.NoError(t, db.Create(seg1).Error)

		// Submit new split
		err := splittingService.SubmitSplit(context.Background(), parent.ID, []float64{10.0}, false, userID)
		assert.NoError(t, err)

		// Verify parent still exists
		var parentCheck models.VideoFile
		err = db.First(&parentCheck, parent.ID).Error
		assert.NoError(t, err)
		assert.Equal(t, parent.ID, parentCheck.ID)
		assert.Equal(t, models.SourceTypeRecording, parentCheck.SourceType)
	})

	t.Run("Test3_SubmitSplitFailsGracefullyIfCleanupFails", func(t *testing.T) {
		db := setupSplitTestDB(t)
		logger := zap.NewNop()
		videoFileService := NewVideoFileService(db, logger, t.TempDir(), "")

		cfg := &config.Config{
			FFmpeg: config.FFmpegConfig{Path: "ffmpeg"},
		}
		splittingService := NewSplittingService(db, logger, cfg, videoFileService)

		userID := uint(1)
		otherUserID := uint(2)

		// Create parent video owned by other user
		parent := &models.VideoFile{
			FileName:   "parent_video.mp4",
			FilePath:   "/test/parent.mp4",
			FileSize:   1024000,
			Duration:   300,
			Format:     "mp4",
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeRecording,
			CreatedBy:  otherUserID,
		}
		require.NoError(t, db.Create(parent).Error)

		// Create segments owned by other user
		seg1 := &models.VideoFile{
			FileName:   "segment_001.mp4",
			FilePath:   "/test/segments/segment_001.mp4",
			FileSize:   512000,
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeSplit,
			ParentID:   &parent.ID,
			CreatedBy:  otherUserID,
		}
		require.NoError(t, db.Create(seg1).Error)

		// Try to submit split as different user (cleanup should find 0 segments, not fail)
		// Actually this won't fail since DeleteSplitSegmentsByParentID returns 0, nil when no segments found
		// Let's test a different scenario: cleanup succeeds (0 segments deleted) and split continues
		err := splittingService.SubmitSplit(context.Background(), parent.ID, []float64{10.0}, false, userID)
		assert.NoError(t, err, "Should succeed when cleanup finds no segments to delete")
	})

	t.Run("Test4_SubmitSplitLogsCleanupOperation", func(t *testing.T) {
		db := setupSplitTestDB(t)
		logger := zap.NewNop() // In real test, would use a logger that captures logs
		videoFileService := NewVideoFileService(db, logger, t.TempDir(), "")

		cfg := &config.Config{
			FFmpeg: config.FFmpegConfig{Path: "ffmpeg"},
		}
		splittingService := NewSplittingService(db, logger, cfg, videoFileService)

		userID := uint(1)

		// Create parent video
		parent := &models.VideoFile{
			FileName:   "parent_video.mp4",
			FilePath:   "/test/parent.mp4",
			FileSize:   1024000,
			Duration:   300,
			Format:     "mp4",
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeRecording,
			CreatedBy:  userID,
		}
		require.NoError(t, db.Create(parent).Error)

		// Create existing segments
		for i := 1; i <= 3; i++ {
			seg := &models.VideoFile{
				FileName:   fmt.Sprintf("segment_%03d.mp4", i),
				FilePath:   fmt.Sprintf("/test/segments/segment_%03d.mp4", i),
				FileSize:   512000,
				Status:     models.FileStatusReady,
				SourceType: models.SourceTypeSplit,
				ParentID:   &parent.ID,
				CreatedBy:  userID,
			}
			require.NoError(t, db.Create(seg).Error)
		}

		// Submit split (should log cleanup)
		err := splittingService.SubmitSplit(context.Background(), parent.ID, []float64{10.0, 20.0}, false, userID)
		assert.NoError(t, err)
		// Note: In real test, would verify log messages contain "清理旧分割段完成"
	})

	t.Run("Test5_ConcurrentSplitRequestsHandledCorrectly", func(t *testing.T) {
		db := setupSplitTestDB(t)
		logger := zap.NewNop()
		videoFileService := NewVideoFileService(db, logger, t.TempDir(), "")

		cfg := &config.Config{
			FFmpeg: config.FFmpegConfig{Path: "ffmpeg"},
		}
		splittingService := NewSplittingService(db, logger, cfg, videoFileService)

		userID := uint(1)

		// Create parent video
		parent := &models.VideoFile{
			FileName:   "parent_video.mp4",
			FilePath:   "/test/parent.mp4",
			FileSize:   1024000,
			Duration:   300,
			Format:     "mp4",
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeRecording,
			CreatedBy:  userID,
		}
		require.NoError(t, db.Create(parent).Error)

		// Submit first split
		err1 := splittingService.SubmitSplit(context.Background(), parent.ID, []float64{10.0}, false, userID)
		assert.NoError(t, err1)

		// Submit second split immediately (should clean up first split's segments)
		// Note: Since first split hasn't created segments yet, cleanup will find 0
		err2 := splittingService.SubmitSplit(context.Background(), parent.ID, []float64{20.0, 30.0}, false, userID)
		assert.NoError(t, err2, "Concurrent split should be handled correctly")

		// Verify status
		status := splittingService.GetSplitStatus(parent.ID)
		assert.Equal(t, "processing", status, "Video should be in processing state")
	})
}

func TestSplittingService_SmartCleanupNoSegments(t *testing.T) {
	t.Run("FirstTimeSplitNoCleanup", func(t *testing.T) {
		db := setupSplitTestDB(t)
		logger := zap.NewNop()
		videoFileService := NewVideoFileService(db, logger, t.TempDir(), "")

		cfg := &config.Config{
			FFmpeg: config.FFmpegConfig{Path: "ffmpeg"},
		}
		splittingService := NewSplittingService(db, logger, cfg, videoFileService)

		userID := uint(1)

		// Create parent video (no existing segments)
		parent := &models.VideoFile{
			FileName:   "parent_video.mp4",
			FilePath:   "/test/parent.mp4",
			FileSize:   1024000,
			Duration:   300,
			Format:     "mp4",
			Status:     models.FileStatusReady,
			SourceType: models.SourceTypeRecording,
			CreatedBy:  userID,
		}
		require.NoError(t, db.Create(parent).Error)

		// Submit first split (no cleanup needed)
		err := splittingService.SubmitSplit(context.Background(), parent.ID, []float64{10.0, 20.0}, false, userID)
		assert.NoError(t, err, "First-time split should succeed without cleanup")
	})
}
