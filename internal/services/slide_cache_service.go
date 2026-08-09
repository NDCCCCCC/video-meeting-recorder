package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/security"
)

const (
	slideFilenamePattern = `^slide_\d{3}\.jpg$`
	slideFilenamePrefix  = "slide_"
	slideFilenameExt     = ".jpg"
)

// SlideImageData represents slide image metadata for API responses
type SlideImageData struct {
	SlideNumber  int    `json:"slide_number"`
	ThumbnailURL string `json:"thumbnail_url"`
	FullsizeURL  string `json:"fullsize_url"`
}

// SlideCacheService manages slide image caching and retrieval
type SlideCacheService struct {
	db             *gorm.DB
	logger         *zap.Logger
	config         *config.Config
	slideExtractor *SlideExtractor
	cacheMutexes   sync.Map // map[uint]*sync.Mutex for per-PPT mutexes
}

// NewSlideCacheService creates a new SlideCacheService instance
func NewSlideCacheService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, extractor *SlideExtractor) *SlideCacheService {
	return &SlideCacheService{
		db:             db,
		logger:         logger,
		config:         cfg,
		slideExtractor: extractor,
	}
}

// GetOrExtractSlides retrieves slide images from cache or extracts them if not cached
func (s *SlideCacheService) GetOrExtractSlides(ctx context.Context, pptFileID uint) ([]SlideImageData, error) {
	// Load PPTFile from DB
	var pptFile models.PPTFile
	if err := s.db.WithContext(ctx).First(&pptFile, pptFileID).Error; err != nil {
		return nil, fmt.Errorf("PPT file not found: %w", err)
	}

	// Get or create mutex for this PPT (double-checked locking pattern)
	mutex, _ := s.cacheMutexes.LoadOrStore(pptFileID, &sync.Mutex{})
	mu := mutex.(*sync.Mutex)

	mu.Lock()
	defer mu.Unlock()

	// Check if cache exists and is valid (double-checked after acquiring lock)
	cacheDir := filepath.Join(s.config.Storage.RecordingsPath, "ppts", fmt.Sprintf("%d", pptFileID), "slides")
	thumbnailDir := filepath.Join(cacheDir, "thumbnails")

	// Check if thumbnail directory exists
	if _, err := os.Stat(thumbnailDir); err == nil {
		// Cache exists - read thumbnails and build slide data
		slides, err := s.readCachedSlides(thumbnailDir, pptFileID)
		if err == nil && len(slides) > 0 {
			return slides, nil
		}
		// If cache is invalid, fall through to extraction
	}

	// Cache miss - extract slides
	s.logger.Info("Slide cache miss, extracting slides",
		zap.Uint("ppt_file_id", pptFileID),
		zap.String("pptx_path", pptFile.FilePath))

	// Extract slides using SlideExtractor
	slideCount, err := s.slideExtractor.ExtractSlides(ctx, pptFile.FilePath, cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract slides: %w", err)
	}

	// Update PPTFile with cache path
	if err := s.db.WithContext(ctx).Model(&pptFile).Update("slide_cache_path", cacheDir).Error; err != nil {
		s.logger.Warn("Failed to update PPTFile cache path",
			zap.Uint("ppt_file_id", pptFileID),
			zap.Error(err))
	}

	// Read the newly extracted slides
	slides, err := s.readCachedSlides(thumbnailDir, pptFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to read extracted slides: %w", err)
	}

	s.logger.Info("Slides extracted successfully",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Int("slide_count", slideCount))

	return slides, nil
}

// readCachedSlides reads slide images from cache directory
func (s *SlideCacheService) readCachedSlides(thumbnailDir string, pptFileID uint) ([]SlideImageData, error) {
	// List all thumbnail files
	files, err := os.ReadDir(thumbnailDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read thumbnail directory: %w", err)
	}

	slides := make([]SlideImageData, 0, len(files))

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()

		// Validate filename format: slide_XXX.jpg
		if !s.isValidSlideFilename(filename) {
			continue
		}

		// Extract slide number from filename
		slideNum, err := s.extractSlideNumber(filename)
		if err != nil {
			continue
		}

		// Build URLs
		baseURL := fmt.Sprintf("/api/v1/ppts/%d/slides", pptFileID)
		slides = append(slides, SlideImageData{
			SlideNumber:  slideNum,
			ThumbnailURL: fmt.Sprintf("%s/thumbnails/%s", baseURL, filename),
			FullsizeURL:  fmt.Sprintf("%s/fullsize/%s", baseURL, filename),
		})
	}

	// Sort slides by slide number to ensure consistent ordering
	sort.Slice(slides, func(i, j int) bool {
		return slides[i].SlideNumber < slides[j].SlideNumber
	})

	return slides, nil
}

// isValidSlideFilename validates filename matches slide_\d{3}\.jpg pattern
func (s *SlideCacheService) isValidSlideFilename(filename string) bool {
	matched, _ := regexp.MatchString(slideFilenamePattern, filename)
	return matched
}

// extractSlideNumber extracts slide number from filename (e.g., "slide_001.jpg" -> 1)
func (s *SlideCacheService) extractSlideNumber(filename string) (int, error) {
	// Remove extension using constant
	base := filename[:len(filename)-len(slideFilenameExt)]

	// Split by underscore
	parts := strings.Split(base, "_")
	if len(parts) != 2 || parts[0] != slideFilenamePrefix[:len(slideFilenamePrefix)-1] {
		return 0, fmt.Errorf("invalid filename format")
	}

	// Parse number
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid slide number: %w", err)
	}

	return num, nil
}

// InvalidateCache removes cached slide images for a PPT file
func (s *SlideCacheService) InvalidateCache(ctx context.Context, pptFileID uint) error {
	// Load PPTFile to get cache path
	var pptFile models.PPTFile
	if err := s.db.WithContext(ctx).First(&pptFile, pptFileID).Error; err != nil {
		return fmt.Errorf("PPT file not found: %w", err)
	}

	// Remove cache directory if it exists
	if pptFile.SlideCachePath != "" {
		if err := os.RemoveAll(pptFile.SlideCachePath); err != nil {
			s.logger.Warn("Failed to remove cache directory",
				zap.String("path", pptFile.SlideCachePath),
				zap.Error(err))
		}
	}

	// Clear cache path in database
	if err := s.db.WithContext(ctx).Model(&pptFile).Update("slide_cache_path", "").Error; err != nil {
		return fmt.Errorf("failed to clear cache path: %w", err)
	}

	s.logger.Info("Slide cache invalidated",
		zap.Uint("ppt_file_id", pptFileID))

	return nil
}

// GetSlideImagePath returns the absolute path to a slide image file
// SECURITY: Validates path is within allowed directory to prevent traversal (T-03-01)
func (s *SlideCacheService) GetSlideImagePath(pptFileID uint, resolution, filename string) (string, error) {
	// Validate resolution
	if resolution != "thumbnails" && resolution != "fullsize" {
		return "", fmt.Errorf("invalid resolution: %s", resolution)
	}

	// filepath.Base 剥离任何目录成分，再过严格白名单正则。
	filename = filepath.Base(filename)
	// CodeQL 认可的路径注入 barrier：显式拒绝 ".."。下游 absolutePath 据此视为无污点。
	if strings.Contains(filename, "..") {
		return "", fmt.Errorf("invalid filename format")
	}
	// Validate filename format (strict whitelist)
	if !s.isValidSlideFilename(filename) {
		return "", fmt.Errorf("invalid filename format")
	}

	// Build absolute path（basePath 由 config + 固定段构成；filename 为唯一不可信输入）。
	// SECURITY: 包容校验 filename 不可逃出 basePath（路径穿越防护，per T-03-01）；
	// 使用 SafeJoin 返回值作为 sink，使 CodeQL 路径注入污点在 guard 处终止。
	basePath := filepath.Join(s.config.Storage.RecordingsPath, "ppts", fmt.Sprintf("%d", pptFileID), "slides", resolution)
	absolutePath, err := security.SafeJoin(basePath, filename)
	if err != nil {
		return "", fmt.Errorf("path traversal detected: %s", filename)
	}

	// Check if file exists
	if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
		return "", fmt.Errorf("slide image not found: %s", filename)
	}

	return absolutePath, nil
}
