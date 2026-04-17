package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PPTMergeService handles merging slides from multiple PPT files
type PPTMergeService struct {
	db                *gorm.DB
	logger            *zap.Logger
	config            *config.Config
	mergeScript       string
	slideCacheService *SlideCacheService
}

// pythonMergeResult represents the JSON output from the Python merge script
type pythonMergeResult struct {
	Success      bool   `json:"success"`
	SlidesMerged int    `json:"slides_merged"`
	OutputPath   string `json:"output_path"`
	Error        string `json:"error,omitempty"`
}

// NewPPTMergeService creates a new PPTMergeService instance
func NewPPTMergeService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, slideCacheService *SlideCacheService) *PPTMergeService {
	projectRoot := getProjectRoot()
	return &PPTMergeService{
		db:                db,
		logger:            logger,
		config:            cfg,
		mergeScript:       filepath.Join(projectRoot, "scripts", "merge_slides.py"),
		slideCacheService: slideCacheService,
	}
}

// MergeSlides merges selected slides from multiple PPT files into a new PPTX
func (s *PPTMergeService) MergeSlides(ctx context.Context, req *models.MergeRequest, userID uint) (*models.PPTFile, error) {
	// 1. Validate: Check slide count <= 200 (per D-17 merge limit)
	if len(req.Slides) > 200 {
		return nil, fmt.Errorf("超过200页幻灯片限制")
	}

	// 2. Validate ownership: Load VideoFile and verify user owns it
	var videoFile models.VideoFile
	if err := s.db.First(&videoFile, req.VideoFileID).Error; err != nil {
		return nil, fmt.Errorf("video file not found")
	}

	// Check ownership (admin or owner)
	if videoFile.CreatedBy != userID {
		// Note: isAdmin check should be done at handler level
		// This service layer check is a fallback
		return nil, fmt.Errorf("user does not own this video file")
	}

	// 3. Build slide spec: Group by source PPT and collect unique slide numbers
	sourcePptMap := make(map[uint][]int) // pptFileID -> slideNumbers
	for _, slide := range req.Slides {
		sourcePptMap[slide.PptFileID] = append(sourcePptMap[slide.PptFileID], slide.SlideNumber)
	}

	// 4. Validate each source PPT belongs to the same VideoFile
	var slideSpecs []map[string]interface{}
	sourcePptIDs := make([]uint, 0)

	for pptFileID := range sourcePptMap {
		var pptFile models.PPTFile
		if err := s.db.First(&pptFile, pptFileID).Error; err != nil {
			return nil, fmt.Errorf("source PPT file %d not found", pptFileID)
		}

		// Verify PPT belongs to same video
		if pptFile.SourceVideoFileID == nil || *pptFile.SourceVideoFileID != req.VideoFileID {
			return nil, fmt.Errorf("PPT file %d does not belong to video %d", pptFileID, req.VideoFileID)
		}

		// Deduplicate and sort slide numbers per source
		slideNumbers := sourcePptMap[pptFileID]
		uniqueSlides := uniqueInts(slideNumbers)

		slideSpecs = append(slideSpecs, map[string]interface{}{
			"pptx_path":    pptFile.FilePath,
			"slide_numbers": uniqueSlides,
		})
		sourcePptIDs = append(sourcePptIDs, pptFileID)
	}

	// 5. Execute merge: Call Python merge script
	// Generate output path: recordings/ppts/merged/merged_{timestamp}_{random}.pptx
	mergedDir := filepath.Join(s.config.Storage.RecordingsPath, "ppts", "merged")
	if err := os.MkdirAll(mergedDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create merged output directory: %w", err)
	}

	outputName := req.OutputName
	if outputName == "" {
		outputName = "合并PPT.pptx"
	}
	timestamp := time.Now().Unix()
	outputPath := filepath.Join(mergedDir, fmt.Sprintf("merged_%d_%s", timestamp, outputName))

	// Convert slide spec to JSON
	slideSpecJSON, err := json.Marshal(slideSpecs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal slide spec: %w", err)
	}

	// Execute Python script
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	cmdName := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		cmdName = "python"
	}

	args := []string{s.mergeScript, outputPath, string(slideSpecJSON)}
	cmd := exec.CommandContext(ctx, cmdName, args...)

	output, err := cmd.Output()
	if err != nil {
		s.logger.Error("Python merge script failed",
			zap.String("output", string(output)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to merge slides: %w (output: %s)", err, string(output))
	}

	// Parse JSON result
	var result pythonMergeResult
	if err := json.Unmarshal(output, &result); err != nil {
		s.logger.Error("Failed to parse Python merge output",
			zap.String("output", string(output)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse merge output: %w", err)
	}

	// Check for success
	if !result.Success {
		s.logger.Error("Python merge script reported failure",
			zap.String("error", result.Error))
		return nil, fmt.Errorf("slide merge failed: %s", result.Error)
	}

	// 6. Create PPTFile record
	// Convert source PPT IDs to JSON array for MergedFrom field
	mergedFromJSON, _ := json.Marshal(sourcePptIDs)

	// Get file size
	fileInfo, _ := os.Stat(outputPath)
	fileSize := fileInfo.Size()

	pptFile := &models.PPTFile{
		FileName:       outputName,
		FilePath:       outputPath,
		FileSize:       fileSize,
		PageCount:      result.SlidesMerged,
		Format:         "pptx",
		SourceType:     models.PPTSourceTypeMerge,
		MergedFrom:     string(mergedFromJSON),
		SourceVideoFileID: &req.VideoFileID,
	}

	if err := s.db.Create(pptFile).Error; err != nil {
		s.logger.Error("Failed to create merged PPT file record",
			zap.Uint("video_file_id", req.VideoFileID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create PPT file record: %w", err)
	}

	s.logger.Info("Slides merged successfully",
		zap.Uint("video_file_id", req.VideoFileID),
		zap.Uint("merged_ppt_file_id", pptFile.ID),
		zap.Int("slides_merged", result.SlidesMerged),
		zap.String("output_path", outputPath))

	return pptFile, nil
}

// uniqueInts returns unique integers from a slice
func uniqueInts(nums []int) []int {
	seen := make(map[int]bool)
	result := make([]int, 0)

	for _, num := range nums {
		if !seen[num] {
			seen[num] = true
			result = append(result, num)
		}
	}

	return result
}
