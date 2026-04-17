package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Muprprpr/Go-pptx"
	"go.uber.org/zap"
)

// PPTXGenerator generates PowerPoint files from image frames
type PPTXGenerator struct {
	logger *zap.Logger
}

// NewPPTXGenerator creates a new PPTXGenerator instance
func NewPPTXGenerator(logger *zap.Logger) *PPTXGenerator {
	return &PPTXGenerator{
		logger: logger,
	}
}

const (
	// EMU (English Metric Units) per inch
	EMU_PER_INCH = 914400

	// 16:9 slide dimensions in inches
	SLIDE_WIDTH_INCH  = 10.0
	SLIDE_HEIGHT_INCH = 5.625

	// 16:9 slide dimensions in EMU
	// 10" * 914400 = 9,144,000 EMU
	// 5.625" * 914400 = 5,143,250 EMU
	SLIDE_WIDTH_EMU  = 10 * EMU_PER_INCH   // 9,144,000
	SLIDE_HEIGHT_EMU = 5.625 * EMU_PER_INCH // 5,143,250

	// Maximum number of slides to prevent OOM (T-02-06 mitigation)
	MAX_SLIDES = 500
)

// GeneratePPTX creates a PowerPoint file from a list of image frames
// Each image becomes a separate slide with full-frame 16:9 layout
func (g *PPTXGenerator) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
	// Validate inputs
	if len(framePaths) == 0 {
		return 0, fmt.Errorf("frame paths cannot be empty")
	}

	// Prevent OOM from extremely long videos (T-02-06 mitigation)
	if len(framePaths) > MAX_SLIDES {
		return 0, fmt.Errorf("number of frames (%d) exceeds maximum allowed (%d)", len(framePaths), MAX_SLIDES)
	}

	// Validate output directory exists
	outputDir := filepath.Dir(outputPath)
	if outputDir == "" {
		outputDir = "."
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Sanitize output path to prevent path traversal (T-02-05 mitigation)
	outputPath = filepath.Clean(outputPath)

	// Create presentation
	ppt := pptx.NewPresentation()

	// Set slide size to 16:9
	ppt.SetSlideSize(SLIDE_WIDTH_EMU, SLIDE_HEIGHT_EMU)

	pageCount := 0

	// Add each frame as a slide
	for _, framePath := range framePaths {
		// Sanitize file path
		framePath = filepath.Clean(framePath)

		// Check if file exists
		if _, err := os.Stat(framePath); os.IsNotExist(err) {
			g.logger.Warn("Image file does not exist, skipping",
				zap.String("path", framePath))
			continue
		}

		// Add image to presentation
		imgRef, err := ppt.AddImage(framePath)
		if err != nil {
			g.logger.Error("Failed to add image to presentation, skipping",
				zap.String("path", framePath),
				zap.Error(err))
			continue
		}

		// Create slide
		slide := ppt.AddSlide()

		// Add image to slide (full-frame, no margins per D-10)
		// Position: (0, 0) - top-left corner
		// Size: full slide size (16:9 per D-11)
		err = slide.AddImage(imgRef, 0, 0, SLIDE_WIDTH_EMU, SLIDE_HEIGHT_EMU)
		if err != nil {
			g.logger.Error("Failed to add image to slide, skipping",
				zap.String("path", framePath),
				zap.Error(err))
			continue
		}

		pageCount++
	}

	// Ensure at least one slide was created
	if pageCount == 0 {
		return 0, fmt.Errorf("no valid slides created from %d input frames", len(framePaths))
	}

	// Save presentation
	if err := ppt.SaveToFile(outputPath); err != nil {
		return 0, fmt.Errorf("failed to save PPTX: %w", err)
	}

	g.logger.Info("PPTX generated successfully",
		zap.String("output", outputPath),
		zap.Int("page_count", pageCount))

	return pageCount, nil
}

// ValidateImageFiles checks which image files exist and returns valid paths and any errors
func (g *PPTXGenerator) ValidateImageFiles(framePaths []string) ([]string, []error) {
	validPaths := []string{}
	errs := []error{}

	for _, path := range framePaths {
		// Sanitize path
		path = filepath.Clean(path)

		// Check if file exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("file does not exist: %s", path))
		} else {
			validPaths = append(validPaths, path)
		}
	}

	return validPaths, errs
}
