package services

import (
	"testing"

	"github.com/cpic/record_v2/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestTimestampMapper_GetTimestampMap tests retrieving timestamp maps
func TestTimestampMapper_GetTimestampMap(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*TimestampMapper)
		videoFileID uint
		expectError bool
		validate    func(*testing.T, []models.SlideTimestamp, error)
	}{
		{
			name:        "Returns error for non-existent video",
			videoFileID: 999,
			expectError: true,
			validate: func(t *testing.T, ts []models.SlideTimestamp, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			},
		},
		{
			name: "Returns cached timestamp map on second call",
			setup: func(mapper *TimestampMapper) {
				// First call should populate cache
				mapper.GetTimestampMap(1)
			},
			videoFileID: 1,
			expectError: false,
			validate: func(t *testing.T, ts []models.SlideTimestamp, err error) {
				// Should return cached data (even if empty)
				assert.NoError(t, err)
			},
		},
		{
			name: "Returns sorted timestamps by slide number",
			setup: func(mapper *TimestampMapper) {
				// This would require setting up test data in database
				// For now, we test the sorting logic
			},
			videoFileID: 1,
			expectError: false,
			validate: func(t *testing.T, ts []models.SlideTimestamp, err error) {
				// Verify sorted order
				for i := 1; i < len(ts); i++ {
					assert.Greater(t, ts[i].SlideNumber, ts[i-1].SlideNumber)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test would require a mock database or test setup
			// For now, we'll create the test structure
			t.Skip("Requires database setup - will be implemented with mock")
		})
	}
}

// TestTimestampMapper_GetTimestampForSlide tests retrieving timestamp for specific slide
func TestTimestampMapper_GetTimestampForSlide(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*TimestampMapper)
		videoFileID uint
		slideNumber int
		expectError bool
		validate    func(*testing.T, float64, error)
	}{
		{
			name:        "Returns correct timestamp for existing slide",
			videoFileID: 1,
			slideNumber: 2,
			expectError: false,
			validate: func(t *testing.T, ts float64, err error) {
				assert.NoError(t, err)
				assert.GreaterOrEqual(t, ts, 0.0)
			},
		},
		{
			name:        "Returns error for non-existent slide",
			videoFileID: 1,
			slideNumber: 999,
			expectError: true,
			validate: func(t *testing.T, ts float64, err error) {
				assert.Error(t, err)
			},
		},
		{
			name:        "Interpolates timestamp for missing slide",
			videoFileID: 1,
			slideNumber: 5,
			expectError: false,
			validate: func(t *testing.T, ts float64, err error) {
				assert.NoError(t, err)
				// Interpolated value should be between neighboring slides
				assert.GreaterOrEqual(t, ts, 0.0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("Requires database setup - will be implemented with mock")
		})
	}
}

// TestTimestampMapper_BuildTimestampMapFromFrames tests building map from frame extraction
func TestTimestampMapper_BuildTimestampMapFromFrames(t *testing.T) {
	tests := []struct {
		name     string
		frames   []ExtractedFrame
		expected []models.SlideTimestamp
	}{
		{
			name: "Builds map with 1-based slide numbers",
			frames: []ExtractedFrame{
				{FilePath: "/path/frame1.jpg", Timestamp: 0.0, Index: 0},
				{FilePath: "/path/frame2.jpg", Timestamp: 15.5, Index: 1},
				{FilePath: "/path/frame3.jpg", Timestamp: 30.0, Index: 2},
			},
			expected: []models.SlideTimestamp{
				{SlideNumber: 1, Timestamp: 0.0},
				{SlideNumber: 2, Timestamp: 15.5},
				{SlideNumber: 3, Timestamp: 30.0},
			},
		},
		{
			name:     "Handles empty frame list",
			frames:   []ExtractedFrame{},
			expected: []models.SlideTimestamp{},
		},
		{
			name: "Handles single frame",
			frames: []ExtractedFrame{
				{FilePath: "/path/frame1.jpg", Timestamp: 0.0, Index: 0},
			},
			expected: []models.SlideTimestamp{
				{SlideNumber: 1, Timestamp: 0.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := &TimestampMapper{}
			result := mapper.BuildTimestampMapFromFrames(tt.frames)

			assert.Equal(t, len(tt.expected), len(result))

			for i, expected := range tt.expected {
				assert.Equal(t, expected.SlideNumber, result[i].SlideNumber)
				assert.Equal(t, expected.Timestamp, result[i].Timestamp)
			}
		})
	}
}

// TestTimestampMapper_InvalidateCache tests cache invalidation
func TestTimestampMapper_InvalidateCache(t *testing.T) {
	mapper := &TimestampMapper{
		logger: zap.NewNop(),
		cache:  newTimestampCache(),
	}

	// Populate cache
	mapper.cache.Set(uint(1), []models.SlideTimestamp{{SlideNumber: 1, Timestamp: 0.0}})

	// Invalidate
	mapper.InvalidateCache(uint(1))

	// Verify cache is cleared
	timestamps, found := mapper.cache.Get(uint(1))
	assert.False(t, found, "Cache should be cleared after invalidation")
	assert.Nil(t, timestamps, "Cached timestamps should be nil")
}

// TestTimestampMapper_InterpolateTimestamp tests interpolation logic
func TestTimestampMapper_InterpolateTimestamp(t *testing.T) {
	tests := []struct {
		name        string
		timestamps  []models.SlideTimestamp
		slideNumber int
		expected    float64
	}{
		{
			name: "Interpolates between two slides",
			timestamps: []models.SlideTimestamp{
				{SlideNumber: 1, Timestamp: 0.0},
				{SlideNumber: 3, Timestamp: 30.0},
			},
			slideNumber: 2,
			expected:    15.0, // (0 + 30) / 2
		},
		{
			name: "Returns first timestamp if slide before first",
			timestamps: []models.SlideTimestamp{
				{SlideNumber: 5, Timestamp: 50.0},
				{SlideNumber: 10, Timestamp: 100.0},
			},
			slideNumber: 1,
			expected:    50.0,
		},
		{
			name: "Returns last timestamp if slide after last",
			timestamps: []models.SlideTimestamp{
				{SlideNumber: 1, Timestamp: 10.0},
				{SlideNumber: 5, Timestamp: 50.0},
			},
			slideNumber: 10,
			expected:    50.0,
		},
		{
			name: "Handles single timestamp",
			timestamps: []models.SlideTimestamp{
				{SlideNumber: 1, Timestamp: 25.0},
			},
			slideNumber: 5,
			expected:    25.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapper := &TimestampMapper{}
			result := mapper.interpolateTimestamp(tt.timestamps, tt.slideNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}
