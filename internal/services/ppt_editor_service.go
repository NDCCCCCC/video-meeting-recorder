package services

import (
	"context"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/jpeg"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DuplicateGroup represents a group of duplicate slides
type DuplicateGroup struct {
	Slides       []int   `json:"slides"`
	Similarity   float64 `json:"similarity"`
	SSIMScore    float64 `json:"ssim_score"`
	PHashDist    int     `json:"phash_distance"`
	EdgeChange   float64 `json:"edge_change_rate"`
}

// DuplicatePair represents a single duplicate detection result
type DuplicatePair struct {
	Slide1       int     `json:"slide1"`
	Slide2       int     `json:"slide2"`
	Similarity   float64 `json:"similarity"`
	SSIMScore    float64 `json:"ssim_score"`
	PHashDist    int     `json:"phash_distance"`
	EdgeChange   float64 `json:"edge_change_rate"`
}

// PPTEditorService handles PPT editing operations
type PPTEditorService struct {
	db                 *gorm.DB
	logger             *zap.Logger
	config             *config.Config
	slideCache         *SlideCacheService
	similarityDetector *SimilarityDetector
	pptxGenerator      *PPTXGenerator
	timestampMapper    *TimestampMapper
}

// NewPPTEditorService creates a new PPTEditorService instance
func NewPPTEditorService(
	db *gorm.DB,
	logger *zap.Logger,
	cfg *config.Config,
	slideCache *SlideCacheService,
	similarityDetector *SimilarityDetector,
	pptxGenerator *PPTXGenerator,
	timestampMapper *TimestampMapper,
) *PPTEditorService {
	return &PPTEditorService{
		db:                 db,
		logger:             logger,
		config:             cfg,
		slideCache:         slideCache,
		similarityDetector: similarityDetector,
		pptxGenerator:      pptxGenerator,
		timestampMapper:    timestampMapper,
	}
}

// DetectDuplicateSlides detects duplicate slides using visual similarity
func (s *PPTEditorService) DetectDuplicateSlides(pptFileID uint) ([]DuplicateGroup, error) {
	s.logger.Info("Starting duplicate slide detection",
		zap.Uint("ppt_file_id", pptFileID))

	// Get all slides
	slides, err := s.slideCache.GetOrExtractSlides(pptFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slides: %w", err)
	}

	if len(slides) < 2 {
		return []DuplicateGroup{}, nil
	}

	// Load full-size images for comparison
	type slideImage struct {
		Number int
		Path   string
	}

	images := make([]slideImage, len(slides))
	for i, slide := range slides {
		// Get absolute path to full-size image
		resolution := "fullsize"
		filename := fmt.Sprintf("slide_%03d.jpg", slide.SlideNumber)
		path, err := s.slideCache.GetSlideImagePath(pptFileID, resolution, filename)
		if err != nil {
			s.logger.Warn("Failed to get slide image path",
				zap.Int("slide_number", slide.SlideNumber),
				zap.Error(err))
			continue
		}
		images[i] = slideImage{Number: slide.SlideNumber, Path: path}
	}

	// Compare all pairs
	duplicates := []DuplicatePair{}
	for i := 0; i < len(images); i++ {
		for j := i + 1; j < len(images); j++ {
			// Load images
			img1, err := s.loadImage(images[i].Path)
			if err != nil {
				s.logger.Warn("Failed to load slide image",
					zap.Int("slide_number", images[i].Number),
					zap.Error(err))
				continue
			}

			img2, err := s.loadImage(images[j].Path)
			if err != nil {
				s.logger.Warn("Failed to load slide image",
					zap.Int("slide_number", images[j].Number),
					zap.Error(err))
				continue
			}

			// Compare using similarity detector
			result, err := s.similarityDetector.IsFrameChanged(img1, img2)
			if err != nil {
				s.logger.Warn("Failed to compare slides",
					zap.Int("slide1", images[i].Number),
					zap.Int("slide2", images[j].Number),
					zap.Error(err))
				continue
			}

			// Check if slides are duplicates (NOT changed = similar)
			// Using strict thresholds for duplicates: SSIM > 0.95 AND pHash < 3
			if !result.Changed && result.SSIMScore > 0.95 && result.PHashDistance < 3 {
				duplicates = append(duplicates, DuplicatePair{
					Slide1:     images[i].Number,
					Slide2:     images[j].Number,
					Similarity: result.SSIMScore,
					SSIMScore:  result.SSIMScore,
					PHashDist:  result.PHashDistance,
					EdgeChange: result.EdgeChangeRate,
				})
			}
		}
	}

	// Group duplicates into clusters
	groups := s.groupDuplicates(duplicates)

	s.logger.Info("Duplicate detection completed",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Int("total_slides", len(slides)),
		zap.Int("duplicates_found", len(duplicates)),
		zap.Int("groups", len(groups)))

	return groups, nil
}

// groupDuplicates groups duplicate pairs into clusters using union-find
func (s *PPTEditorService) groupDuplicates(pairs []DuplicatePair) []DuplicateGroup {
	if len(pairs) == 0 {
		return []DuplicateGroup{}
	}

	// Build adjacency map
	adjacency := make(map[int][]int)
	for _, pair := range pairs {
		adjacency[pair.Slide1] = append(adjacency[pair.Slide1], pair.Slide2)
		adjacency[pair.Slide2] = append(adjacency[pair.Slide2], pair.Slide1)
	}

	// Find connected components (groups)
	visited := make(map[int]bool)
	groups := []DuplicateGroup{}

	for slide := range adjacency {
		if visited[slide] {
			continue
		}

		// BFS to find all slides in this group
		queue := []int{slide}
		groupSlides := []int{}
		avgSimilarity := 0.0
		avgSSIM := 0.0
		avgPHash := 0
		avgEdge := 0.0
		count := 0

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			if visited[current] {
				continue
			}
			visited[current] = true
			groupSlides = append(groupSlides, current)

			// Add neighbors
			for _, neighbor := range adjacency[current] {
				if !visited[neighbor] {
					queue = append(queue, neighbor)
				}
			}
		}

		// Calculate averages for this group
		for _, pair := range pairs {
			if contains(groupSlides, pair.Slide1) && contains(groupSlides, pair.Slide2) {
				avgSimilarity += pair.Similarity
				avgSSIM += pair.SSIMScore
				avgPHash += pair.PHashDist
				avgEdge += pair.EdgeChange
				count++
			}
		}

		if count > 0 {
			avgSimilarity /= float64(count)
			avgSSIM /= float64(count)
			avgPHash /= count
			avgEdge /= float64(count)
		}

		// Sort slides in group
		sort.Ints(groupSlides)

		groups = append(groups, DuplicateGroup{
			Slides:     groupSlides,
			Similarity: avgSimilarity,
			SSIMScore:  avgSSIM,
			PHashDist:  avgPHash,
			EdgeChange: avgEdge,
		})
	}

	return groups
}

// CreateBackup creates a backup of the PPT file
func (s *PPTEditorService) CreateBackup(pptFileID uint) error {
	var pptFile models.PPTFile
	if err := s.db.First(&pptFile, pptFileID).Error; err != nil {
		return fmt.Errorf("PPT file not found: %w", err)
	}

	// Check if backup already exists
	if pptFile.HasBackup() {
		return fmt.Errorf("backup already exists: %s", pptFile.BackupPath)
	}

	// Create backup path: original_path.bak.timestamp
	backupPath := fmt.Sprintf("%s.bak.%d", pptFile.FilePath, time.Now().Unix())

	// Copy file
	if err := s.copyFile(pptFile.FilePath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Update database
	if err := s.db.Model(&pptFile).Update("backup_path", backupPath).Error; err != nil {
		// Cleanup backup file if DB update fails
		os.Remove(backupPath)
		return fmt.Errorf("failed to update backup path: %w", err)
	}

	s.logger.Info("Backup created",
		zap.Uint("ppt_file_id", pptFileID),
		zap.String("backup_path", backupPath))

	return nil
}

// DeleteSlides deletes specified slides and regenerates PPT
func (s *PPTEditorService) DeleteSlides(pptFileID uint, slideNumbers []int) error {
	s.logger.Info("Deleting slides",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Ints("slide_numbers", slideNumbers))

	// Validate input
	if len(slideNumbers) == 0 {
		return fmt.Errorf("no slides specified for deletion")
	}

	// Load PPT file
	var pptFile models.PPTFile
	if err := s.db.First(&pptFile, pptFileID).Error; err != nil {
		return fmt.Errorf("PPT file not found: %w", err)
	}

	// Create backup if not exists
	if !pptFile.HasBackup() {
		if err := s.CreateBackup(pptFileID); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		// Reload to get backup path
		s.db.First(&pptFile, pptFileID)
	}

	// Validate slide numbers are within range
	for _, slideNum := range slideNumbers {
		if slideNum < 1 || slideNum > pptFile.PageCount {
			return fmt.Errorf("invalid slide number: %d (valid range: 1-%d)", slideNum, pptFile.PageCount)
		}
	}

	// Check if trying to delete all slides
	if len(slideNumbers) >= pptFile.PageCount {
		return fmt.Errorf("cannot delete all slides")
	}

	// Get all slides
	allSlides, err := s.slideCache.GetOrExtractSlides(pptFileID)
	if err != nil {
		return fmt.Errorf("failed to get slides: %w", err)
	}

	// Filter out deleted slides
	deletedSet := make(map[int]bool)
	for _, num := range slideNumbers {
		deletedSet[num] = true
	}

	remainingSlides := []string{}
	for _, slide := range allSlides {
		if !deletedSet[slide.SlideNumber] {
			// Build path to full-size image
			resolution := "fullsize"
			filename := fmt.Sprintf("slide_%03d.jpg", slide.SlideNumber)
			path, err := s.slideCache.GetSlideImagePath(pptFileID, resolution, filename)
			if err != nil {
				return fmt.Errorf("failed to get slide %d path: %w", slide.SlideNumber, err)
			}
			remainingSlides = append(remainingSlides, path)
		}
	}

	// Generate new PPTX
	outputPath := fmt.Sprintf("%s.new.pptx", pptFile.FilePath)
	slideCount, err := s.pptxGenerator.GeneratePPTX(context.Background(), remainingSlides, outputPath)
	if err != nil {
		return fmt.Errorf("failed to generate new PPTX: %w", err)
	}

	// Replace old PPTX with new one
	if err := os.Remove(pptFile.FilePath); err != nil {
		return fmt.Errorf("failed to remove old PPTX: %w", err)
	}

	if err := os.Rename(outputPath, pptFile.FilePath); err != nil {
		return fmt.Errorf("failed to rename new PPTX: %w", err)
	}

	// Invalidate slide cache
	if err := s.slideCache.InvalidateCache(pptFileID); err != nil {
		s.logger.Warn("Failed to invalidate slide cache", zap.Error(err))
	}

	// Update database record
	tx := s.db.Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Update page count
	if err := tx.Model(&pptFile).Update("page_count", slideCount).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update page count: %w", err)
	}

	// Reload to record deletion and edit history
	if err := tx.First(&pptFile, pptFileID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to reload PPT file: %w", err)
	}

	// Record deletion
	if err := pptFile.RecordDeletion(slideNumbers); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record deletion: %w", err)
	}

	// Record edit operation
	if err := pptFile.AddEditOperation("delete", slideNumbers); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record edit operation: %w", err)
	}

	// Save changes
	if err := tx.Save(&pptFile).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save PPT file: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Slides deleted successfully",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Int("deleted_count", len(slideNumbers)),
		zap.Int("new_page_count", slideCount))

	return nil
}

// Rollback restores PPT from backup
func (s *PPTEditorService) Rollback(pptFileID uint) error {
	var pptFile models.PPTFile
	if err := s.db.First(&pptFile, pptFileID).Error; err != nil {
		return fmt.Errorf("PPT file not found: %w", err)
	}

	if !pptFile.HasBackup() {
		return fmt.Errorf("no backup exists for rollback")
	}

	// Restore from backup
	if err := s.copyFile(pptFile.BackupPath, pptFile.FilePath); err != nil {
		return fmt.Errorf("failed to restore from backup: %w", err)
	}

	// Invalidate cache
	if err := s.slideCache.InvalidateCache(pptFileID); err != nil {
		s.logger.Warn("Failed to invalidate cache after rollback", zap.Error(err))
	}

	// Get original page count from backup
	originalCount, err := s.getSlideCountFromPPTX(pptFile.BackupPath)
	if err != nil {
		s.logger.Warn("Failed to get original slide count", zap.Error(err))
		originalCount = pptFile.PageCount
	}

	// Update database
	tx := s.db.Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	updates := map[string]interface{}{
		"page_count":    originalCount,
		"backup_path":   "",
		"deleted_slides": "[]",
	}

	if err := tx.Model(&pptFile).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update PPT file: %w", err)
	}

	// Record rollback operation
	if err := tx.First(&pptFile, pptFileID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to reload PPT file: %w", err)
	}

	if err := pptFile.AddEditOperation("rollback", []int{}); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record rollback: %w", err)
	}

	if err := tx.Save(&pptFile).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save PPT file: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Rollback completed",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Int("restored_page_count", originalCount))

	return nil
}

// Helper functions

func (s *PPTEditorService) loadImage(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

func (s *PPTEditorService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (s *PPTEditorService) getSlideCountFromPPTX(pptxPath string) (int, error) {
	// Placeholder - would parse PPTX to get slide count
	// For now, return a default value
	return 0, nil
}

func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// InsertCapturedFrame inserts a captured frame as a new slide into the PPT
// Creates backup before insertion, saves frame to cache, regenerates PPT
func (s *PPTEditorService) InsertCapturedFrame(pptFileID uint, frameBytes []byte, insertPosition int, timestamp float64) error {
	s.logger.Info("Inserting captured frame",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Int("insert_position", insertPosition),
		zap.Float64("timestamp", timestamp))

	// Load PPT file
	var pptFile models.PPTFile
	if err := s.db.First(&pptFile, pptFileID).Error; err != nil {
		return fmt.Errorf("PPT file not found: %w", err)
	}

	// Create backup if not exists
	if !pptFile.HasBackup() {
		if err := s.CreateBackup(pptFileID); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
		// Reload to get backup path
		s.db.First(&pptFile, pptFileID)
	}

		// WR-06: Check for overflow before validation
		if pptFile.PageCount == 2147483647 { // math.MaxInt32
			return fmt.Errorf("cannot insert slide: maximum page count reached")
		}

	// Validate insert position
	if insertPosition < 1 || insertPosition > pptFile.PageCount+1 {
		return fmt.Errorf("invalid insert position: %d (valid range: 1-%d)", insertPosition, pptFile.PageCount+1)
	}

	// Validate frame bytes
	if len(frameBytes) == 0 {
		return fmt.Errorf("frame bytes cannot be empty")
	}
	if len(frameBytes) > 10*1024*1024 { // 10MB limit (T-06-17 mitigation)
		return fmt.Errorf("frame bytes too large: %d bytes (max 10MB)", len(frameBytes))
	}

	// Get all existing slides
	allSlides, err := s.slideCache.GetOrExtractSlides(pptFileID)
	if err != nil {
		return fmt.Errorf("failed to get slides: %w", err)
	}

	// Determine new slide number (insertPosition becomes the new slide number)
	newSlideNumber := insertPosition

	// Save captured frame to slide cache
	framePath, err := s.SaveCapturedFrame(pptFileID, frameBytes, newSlideNumber)
	if err != nil {
		return fmt.Errorf("failed to save captured frame: %w", err)
	}

	// Build list of all slide paths including new slide
	allSlidePaths := make([]string, 0, len(allSlides)+1)
	inserted := false

	for _, slide := range allSlides {
		if !inserted && slide.SlideNumber >= insertPosition {
			// Insert new slide before current slide
			allSlidePaths = append(allSlidePaths, framePath)
			inserted = true
		}
		// Add existing slide path
		resolution := "fullsize"
		filename := fmt.Sprintf("slide_%03d.jpg", slide.SlideNumber)
		path, err := s.slideCache.GetSlideImagePath(pptFileID, resolution, filename)
		if err != nil {
			return fmt.Errorf("failed to get slide %d path: %w", slide.SlideNumber, err)
		}
		allSlidePaths = append(allSlidePaths, path)
	}

	// If inserting at the end, add new slide after all existing slides
	if !inserted {
		allSlidePaths = append(allSlidePaths, framePath)
	}

	// Generate new PPTX with inserted slide
	outputPath := fmt.Sprintf("%s.new.pptx", pptFile.FilePath)
	slideCount, err := s.pptxGenerator.GeneratePPTX(context.Background(), allSlidePaths, outputPath)
	if err != nil {
		// Clean up captured frame on failure
		os.Remove(framePath)
		thumbnailPath := strings.Replace(framePath, "/fullsize/", "/thumbnails/", 1)
		os.Remove(thumbnailPath)
		return fmt.Errorf("failed to generate new PPTX: %w", err)
	}

	// Replace old PPTX with new one
	if err := os.Remove(pptFile.FilePath); err != nil {
		return fmt.Errorf("failed to remove old PPTX: %w", err)
	}

	if err := os.Rename(outputPath, pptFile.FilePath); err != nil {
		return fmt.Errorf("failed to rename new PPTX: %w", err)
	}

	// Invalidate slide cache
	if err := s.slideCache.InvalidateCache(pptFileID); err != nil {
		s.logger.Warn("Failed to invalidate slide cache", zap.Error(err))
	}

	// Invalidate timestamp cache as well (CR-02 fix)
	if s.timestampMapper != nil && pptFile.SourceVideoFileID != nil {
		s.timestampMapper.InvalidateCache(*pptFile.SourceVideoFileID)
		s.logger.Debug("Timestamp cache invalidated after slide insertion",
			zap.Uint("ppt_file_id", pptFileID),
			zap.Uint("video_file_id", *pptFile.SourceVideoFileID))
	}

	// Update database record
	tx := s.db.Begin()
	if err := tx.Error; err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}

	// Update page count
	if err := tx.Model(&pptFile).Update("page_count", slideCount).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update page count: %w", err)
	}

	// Reload to record insertion
	if err := tx.First(&pptFile, pptFileID).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to reload PPT file: %w", err)
	}

	// Record insertion in edit history
	if err := pptFile.AddEditOperation("insert", []int{newSlideNumber}); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record insertion: %w", err)
	}

	// Save changes
	if err := tx.Save(&pptFile).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save PPT file: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.Info("Captured frame inserted successfully",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Int("insert_position", insertPosition),
		zap.Int("new_slide_number", newSlideNumber),
		zap.Int("new_page_count", slideCount))

	return nil
}

// SaveCapturedFrame saves captured frame bytes to slide cache directory
// Returns path to full-size image
func (s *PPTEditorService) SaveCapturedFrame(pptFileID uint, frameBytes []byte, slideNumber int) (string, error) {
	// Build cache directory paths
	cacheDir := filepath.Join(s.config.Storage.RecordingsPath, "ppts", fmt.Sprintf("%d", pptFileID), "slides")
	fullsizeDir := filepath.Join(cacheDir, "fullsize")
	thumbnailDir := filepath.Join(cacheDir, "thumbnails")

	// Create directories if they don't exist
	if err := os.MkdirAll(fullsizeDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create fullsize directory: %w", err)
	}
	if err := os.MkdirAll(thumbnailDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create thumbnail directory: %w", err)
	}

	// Generate filename
	filename := fmt.Sprintf("slide_%03d_captured.jpg", slideNumber)
	fullsizePath := filepath.Join(fullsizeDir, filename)
	thumbnailPath := filepath.Join(thumbnailDir, filename)

	// Save full-size image
	if err := os.WriteFile(fullsizePath, frameBytes, 0644); err != nil {
		return "", fmt.Errorf("failed to write full-size image: %w", err)
	}

	// Generate thumbnail
	if err := s.generateThumbnail(fullsizePath, thumbnailPath); err != nil {
		// WR-05: Return error for critical failures (disk full, permissions)
		if os.IsNotExist(err) || os.IsPermission(err) {
			return "", fmt.Errorf("failed to generate thumbnail: %w", err)
		}
		s.logger.Warn("Failed to generate thumbnail", zap.Error(err))
		// Continue for decode errors (non-critical)
	}

	s.logger.Info("Captured frame saved",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Int("slide_number", slideNumber),
		zap.String("fullsize_path", fullsizePath))

	return fullsizePath, nil
}

// generateThumbnail creates a thumbnail from full-size image
// Uses JPEG encoding with quality 85, size 200x112 (16:9 aspect ratio)
func (s *PPTEditorService) generateThumbnail(inputPath, outputPath string) error {
	// Open input image
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, err := jpeg.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode JPEG: %w", err)
	}

	// Calculate thumbnail dimensions maintaining aspect ratio
	thumbnailWidth := 200
	thumbnailHeight := 112

	// Create thumbnail image
	thumbnail := image.NewRGBA(image.Rect(0, 0, thumbnailWidth, thumbnailHeight))

	// Simple resize (nearest neighbor for speed)
	// For better quality, consider using a resizing library
	srcBounds := img.Bounds()

	for y := 0; y < thumbnailHeight; y++ {
		for x := 0; x < thumbnailWidth; x++ {
			// Map destination coordinates to source coordinates
			srcX := x * srcBounds.Dx() / thumbnailWidth
			srcY := y * srcBounds.Dy() / thumbnailHeight
			thumbnail.Set(x, y, img.At(srcX, srcY))
		}
	}

	// Save thumbnail
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer outFile.Close()

	if err := jpeg.Encode(outFile, thumbnail, &jpeg.Options{Quality: 85}); err != nil {
		return fmt.Errorf("failed to encode thumbnail: %w", err)
	}

	return nil
}

// ReorderSlides reorders slides in a PPT file according to the new slide order
// Returns the new slide order after successful reordering
func (s *PPTEditorService) ReorderSlides(pptFileID uint, newOrder []int) ([]int, error) {
	s.logger.Info("Reordering slides",
		zap.Uint("ppt_file_id", pptFileID),
		zap.Ints("new_order", newOrder))

	// Get PPT file
	var pptFile models.PPTFile
	if err := s.db.First(&pptFile, pptFileID).Error; err != nil {
		return nil, fmt.Errorf("PPT file not found: %w", err)
	}

	// Get current slides to verify all slide numbers exist
	slides, err := s.slideCache.GetOrExtractSlides(pptFileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slides: %w", err)
	}

	// Create a map of current slide numbers for quick lookup
	currentSlideMap := make(map[int]bool)
	for _, slide := range slides {
		currentSlideMap[slide.SlideNumber] = true
	}

	// Validate all slide numbers in new order exist
	for _, slideNum := range newOrder {
		if !currentSlideMap[slideNum] {
			return nil, fmt.Errorf("slide number %d does not exist in PPT", slideNum)
		}
	}

	// Check if order actually changed
	orderChanged := false
	for i, slideNum := range newOrder {
		if slideNum != i+1 {
			orderChanged = true
			break
		}
	}

	if !orderChanged {
		s.logger.Info("Slide order unchanged, skipping reordering")
		return newOrder, nil
	}

	// Get slide image directory - use slide cache directory structure
	// The correct path is: {recordingsPath}/ppts/{pptFileID}/slides/
	slideDir := filepath.Join(s.config.Storage.RecordingsPath, "ppts", fmt.Sprintf("%d", pptFileID), "slides")
	fullsizeDir := filepath.Join(slideDir, "fullsize")
	thumbnailDir := filepath.Join(slideDir, "thumbnails")

	// Verify the slide cache directory exists
	if _, err := os.Stat(fullsizeDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("slide cache directory does not exist: %s", fullsizeDir)
	}

	// Backup current slides before reordering
	backupDir := filepath.Join(slideDir, fmt.Sprintf("backup_before_reorder_%s", time.Now().Format("20060102_150405")))
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy current slides to backup
	for _, slide := range slides {
		srcFullsize := filepath.Join(fullsizeDir, fmt.Sprintf("slide_%03d.jpg", slide.SlideNumber))
		srcThumbnail := filepath.Join(thumbnailDir, fmt.Sprintf("slide_%03d.jpg", slide.SlideNumber))
		dstFullsize := filepath.Join(backupDir, fmt.Sprintf("slide_%03d_fullsize.jpg", slide.SlideNumber))
		dstThumbnail := filepath.Join(backupDir, fmt.Sprintf("slide_%03d_thumbnail.jpg", slide.SlideNumber))

		if err := copyFile(srcFullsize, dstFullsize); err != nil {
			s.logger.Warn("Failed to backup fullsize slide", zap.Error(err))
		}
		if err := copyFile(srcThumbnail, dstThumbnail); err != nil {
			s.logger.Warn("Failed to backup thumbnail slide", zap.Error(err))
		}
	}

	// Create temp directory for reordered slides
	tempDir := filepath.Join(slideDir, "temp_reorder")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Copy slides to new positions in temp directory
	for newPosition, oldSlideNum := range newOrder {
		newSlideNum := newPosition + 1

		srcFullsize := filepath.Join(fullsizeDir, fmt.Sprintf("slide_%03d.jpg", oldSlideNum))
		srcThumbnail := filepath.Join(thumbnailDir, fmt.Sprintf("slide_%03d.jpg", oldSlideNum))
		dstFullsize := filepath.Join(tempDir, fmt.Sprintf("slide_%03d.jpg", newSlideNum))
		dstThumbnail := filepath.Join(tempDir, fmt.Sprintf("thumb_%03d.jpg", newSlideNum))

		if err := copyFile(srcFullsize, dstFullsize); err != nil {
			return nil, fmt.Errorf("failed to copy slide %d: %w", oldSlideNum, err)
		}
		if err := copyFile(srcThumbnail, dstThumbnail); err != nil {
			return nil, fmt.Errorf("failed to copy thumbnail: %w", err)
		}
	}

	// Move reordered slides from temp to final directories
	for newPosition := range newOrder {
		newSlideNum := newPosition + 1

		srcFullsize := filepath.Join(tempDir, fmt.Sprintf("slide_%03d.jpg", newSlideNum))
		srcThumbnail := filepath.Join(tempDir, fmt.Sprintf("thumb_%03d.jpg", newSlideNum))
		dstFullsize := filepath.Join(fullsizeDir, fmt.Sprintf("slide_%03d.jpg", newSlideNum))
		dstThumbnail := filepath.Join(thumbnailDir, fmt.Sprintf("slide_%03d.jpg", newSlideNum))

		if err := os.Rename(srcFullsize, dstFullsize); err != nil {
			return nil, fmt.Errorf("failed to move fullsize slide: %w", err)
		}
		if err := os.Rename(srcThumbnail, dstThumbnail); err != nil {
			return nil, fmt.Errorf("failed to move thumbnail: %w", err)
		}
	}

	// Update backup path in PPT file
	oldBackupPath := pptFile.BackupPath
	pptFile.BackupPath = backupDir
	if err := s.db.Save(&pptFile).Error; err != nil {
		// Rollback backup path change
		pptFile.BackupPath = oldBackupPath
		s.db.Save(&pptFile)
		return nil, fmt.Errorf("failed to update backup path: %w", err)
	}

	// Clear cache to force re-extraction with new order
	s.slideCache.InvalidateCache(pptFileID)

	s.logger.Info("Successfully reordered slides",
		zap.Uint("ppt_file_id", pptFileID),
		zap.String("backup_dir", backupDir))

	return newOrder, nil
}

