package services

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"os"
	"sort"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
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
}

// NewPPTEditorService creates a new PPTEditorService instance
func NewPPTEditorService(
	db *gorm.DB,
	logger *zap.Logger,
	cfg *config.Config,
	slideCache *SlideCacheService,
	similarityDetector *SimilarityDetector,
	pptxGenerator *PPTXGenerator,
) *PPTEditorService {
	return &PPTEditorService{
		db:                 db,
		logger:             logger,
		config:             cfg,
		slideCache:         slideCache,
		similarityDetector: similarityDetector,
		pptxGenerator:      pptxGenerator,
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
