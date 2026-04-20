package services

import (
	"fmt"
	"sort"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TimestampMapper handles slide-to-video timestamp mapping
type TimestampMapper struct {
	db     *gorm.DB
	logger *zap.Logger
	cache  *timestampCache
}

// timestampCache provides thread-safe caching for timestamp maps
type timestampCache struct {
	maps map[uint][]models.SlideTimestamp
}

func newTimestampCache() *timestampCache {
	return &timestampCache{
		maps: make(map[uint][]models.SlideTimestamp),
	}
}

// Get retrieves cached timestamps for a video file
func (c *timestampCache) Get(videoFileID uint) ([]models.SlideTimestamp, bool) {
	timestamps, ok := c.maps[videoFileID]
	return timestamps, ok
}

// Set stores timestamps in cache
func (c *timestampCache) Set(videoFileID uint, timestamps []models.SlideTimestamp) {
	c.maps[videoFileID] = timestamps
}

// Delete removes timestamps from cache
func (c *timestampCache) Delete(videoFileID uint) {
	delete(c.maps, videoFileID)
}

// NewTimestampMapper creates a new timestamp mapper service
func NewTimestampMapper(db *gorm.DB, logger *zap.Logger) *TimestampMapper {
	return &TimestampMapper{
		db:     db,
		logger: logger,
		cache:  newTimestampCache(),
	}
}

// GetTimestampMap retrieves slide-to-timestamp mappings for a video file
// Returns sorted array of timestamps, caching results for performance
func (m *TimestampMapper) GetTimestampMap(videoFileID uint) ([]models.SlideTimestamp, error) {
	// Check cache first
	if cached, found := m.cache.Get(videoFileID); found {
		m.logger.Debug("Timestamp cache hit", zap.Uint("video_file_id", videoFileID))
		return cached, nil
	}

	m.logger.Debug("Timestamp cache miss", zap.Uint("video_file_id", videoFileID))

	// Query database for transcription task
	var task models.TranscriptionTask
	err := m.db.Where("video_file_id = ? AND status = ?", videoFileID, models.TranscriptionStatusCompleted).
		Order("created_at DESC").
		First(&task).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transcription task not found for video file %d", videoFileID)
		}
		return nil, fmt.Errorf("failed to query transcription task: %w", err)
	}

	// Parse slide timestamps from task
	timestamps, err := task.GetSlideTimestamps()
	if err != nil {
		return nil, fmt.Errorf("failed to parse slide timestamps: %w", err)
	}

	// Sort by slide number
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].SlideNumber < timestamps[j].SlideNumber
	})

	// Cache the result
	m.cache.Set(videoFileID, timestamps)

	return timestamps, nil
}

// GetTimestampForSlide retrieves timestamp for a specific slide number
// Uses binary search for O(log n) lookup, interpolates if not found
func (m *TimestampMapper) GetTimestampForSlide(videoFileID uint, slideNumber int) (float64, error) {
	// Get timestamp map
	timestamps, err := m.GetTimestampMap(videoFileID)
	if err != nil {
		return 0, err
	}

	// Binary search for slide number
	idx := sort.Search(len(timestamps), func(i int) bool {
		return timestamps[i].SlideNumber >= slideNumber
	})

	// Exact match found
	if idx < len(timestamps) && timestamps[idx].SlideNumber == slideNumber {
		return timestamps[idx].Timestamp, nil
	}

	// Not found, interpolate
	interpolated := m.interpolateTimestamp(timestamps, slideNumber)
	m.logger.Debug("Interpolated timestamp",
		zap.Uint("video_file_id", videoFileID),
		zap.Int("slide_number", slideNumber),
		zap.Float64("timestamp", interpolated))

	return interpolated, nil
}

// BuildTimestampMapFromFrames creates slide timestamp map from extracted frames
// Slide numbers are 1-based (frame[0] → slide 1)
func (m *TimestampMapper) BuildTimestampMapFromFrames(frames []ExtractedFrame) []models.SlideTimestamp {
	timestamps := make([]models.SlideTimestamp, 0, len(frames))

	for i, frame := range frames {
		timestamps = append(timestamps, models.SlideTimestamp{
			SlideNumber: i + 1, // 1-based slide numbers
			Timestamp:   frame.Timestamp,
		})
	}

	// Sort by slide number (should already be sorted, but ensures consistency)
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i].SlideNumber < timestamps[j].SlideNumber
	})

	return timestamps
}

// interpolateTimestamp estimates timestamp for missing slide numbers
// Uses linear interpolation between neighboring slides
func (m *TimestampMapper) interpolateTimestamp(timestamps []models.SlideTimestamp, slideNumber int) float64 {
	if len(timestamps) == 0 {
		return 0.0
	}

	// Find nearest slides before and after
	var prevSlide, nextSlide *models.SlideTimestamp

	for i := range timestamps {
		if timestamps[i].SlideNumber < slideNumber {
			prevSlide = &timestamps[i]
		}
		if timestamps[i].SlideNumber > slideNumber && nextSlide == nil {
			nextSlide = &timestamps[i]
			break
		}
	}

	// Edge cases
	if prevSlide == nil {
		// Slide is before first timestamp
		if len(timestamps) > 0 {
			return timestamps[0].Timestamp
		}
		return 0.0
	}

	if nextSlide == nil {
		// Slide is after last timestamp
		return prevSlide.Timestamp
	}

	// Linear interpolation: (prev_ts + next_ts) / 2
	interpolated := (prevSlide.Timestamp + nextSlide.Timestamp) / 2.0
	return interpolated
}

// InvalidateCache clears cached timestamp map for a specific video
// Should be called when new transcription completes or slides are edited
func (m *TimestampMapper) InvalidateCache(videoFileID uint) {
	m.cache.Delete(videoFileID)
	m.logger.Debug("Timestamp cache invalidated", zap.Uint("video_file_id", videoFileID))
}
