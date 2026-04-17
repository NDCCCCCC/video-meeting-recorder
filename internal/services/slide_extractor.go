package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// SlideExtractor extracts slide images from PPTX files using Python script
type SlideExtractor struct {
	logger       *zap.Logger
	pythonScript string
}

// NewSlideExtractor creates a new SlideExtractor instance
func NewSlideExtractor(logger *zap.Logger) *SlideExtractor {
	projectRoot := getProjectRoot()
	return &SlideExtractor{
		logger:       logger,
		pythonScript: filepath.Join(projectRoot, "scripts", "extract_slides.py"),
	}
}

// pythonSlideResult represents the JSON output from the Python script
type pythonSlideResult struct {
	Success    bool                    `json:"success"`
	SlideCount int                     `json:"slide_count"`
	Slides     []pythonSlideImageData  `json:"slides,omitempty"`
	Error      string                  `json:"error,omitempty"`
}

// pythonSlideImageData represents slide image data from Python script
type pythonSlideImageData struct {
	SlideNumber  int    `json:"slide_number"`
	ThumbnailPath string `json:"thumbnail_path"`
	FullsizePath string `json:"fullsize_path"`
}

// ExtractSlides extracts slide images from a PPTX file
func (e *SlideExtractor) ExtractSlides(ctx context.Context, pptxPath string, outputDir string) (int, error) {
	// Check for early cancellation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Validate pptxPath
	if err := e.validatePath(pptxPath); err != nil {
		return 0, fmt.Errorf("invalid pptx path: %w", err)
	}

	// Validate outputDir
	if err := e.validatePath(outputDir); err != nil {
		return 0, fmt.Errorf("invalid output directory: %w", err)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Check again before expensive operation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Prepare command arguments: python3 extract_slides.py <pptxPath> <outputDir>
	args := []string{e.pythonScript, pptxPath, outputDir}

	// Determine python command name
	cmdName := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		cmdName = "python"
	}

	// Execute Python script
	cmd := exec.CommandContext(ctx, cmdName, args...)

	// Capture output (stdout only, per Phase 2 D-14 decision)
	output, err := cmd.Output()
	if err != nil {
		e.logger.Error("Python extract script failed",
			zap.String("output", string(output)),
			zap.Error(err))

		// Clean up partial output directory to prevent cache corruption
		if cleanupErr := os.RemoveAll(outputDir); cleanupErr != nil {
			e.logger.Warn("Failed to clean up partial extraction",
				zap.String("output_dir", outputDir),
				zap.Error(cleanupErr))
		}

		return 0, fmt.Errorf("failed to extract slides: %w (output: %s)", err, string(output))
	}

	// Parse JSON result
	var result pythonSlideResult
	if err := json.Unmarshal(output, &result); err != nil {
		e.logger.Error("Failed to parse Python output",
			zap.String("output", string(output)),
			zap.Error(err))

		// Clean up partial output directory
		if cleanupErr := os.RemoveAll(outputDir); cleanupErr != nil {
			e.logger.Warn("Failed to clean up partial extraction",
				zap.String("output_dir", outputDir),
				zap.Error(cleanupErr))
		}

		return 0, fmt.Errorf("failed to parse extract output: %w (output: %s)", err, string(output))
	}

	// Check for success
	if !result.Success {
		e.logger.Error("Python script reported failure",
			zap.String("error", result.Error))

		// Clean up partial output directory
		if cleanupErr := os.RemoveAll(outputDir); cleanupErr != nil {
			e.logger.Warn("Failed to clean up partial extraction",
				zap.String("output_dir", outputDir),
				zap.Error(cleanupErr))
		}

		return 0, fmt.Errorf("slide extraction failed: %s", result.Error)
	}

	e.logger.Info("Slides extracted successfully",
		zap.String("pptx_path", pptxPath),
		zap.Int("slide_count", result.SlideCount))

	return result.SlideCount, nil
}

// validatePath validates that a path is safe and within allowed directories
// Reuses the pattern from PPTXGenerator
func (e *SlideExtractor) validatePath(path string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute path: %w", err)
	}

	// Check for suspicious characters that could enable injection
	if strings.ContainsAny(path, "\n\r\t") {
		return fmt.Errorf("path contains invalid characters")
	}

	// Ensure path is within allowed storage directory (project root)
	projectRoot := getProjectRoot()
	allowedDir := filepath.Clean(projectRoot)
	if !strings.HasPrefix(absPath, allowedDir) {
		return fmt.Errorf("path outside allowed directory: %s", path)
	}

	return nil
}
