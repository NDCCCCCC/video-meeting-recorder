package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// SlideExtractor extracts slide images from PPTX files using Python script
type SlideExtractor struct {
	logger       *zap.Logger
	pythonScript string
	depsManager  *PythonDepsManager
	preferUV     bool
}

// NewSlideExtractor creates a new SlideExtractor instance
func NewSlideExtractor(logger *zap.Logger, preferUV bool) *SlideExtractor {
	return &SlideExtractor{
		logger:       logger,
		pythonScript: "scripts/extract_slides.py", // 相对路径，确保从项目根目录启动
		depsManager:  NewPythonDepsManager(logger, preferUV),
		preferUV:     preferUV,
	}
}

// pythonSlideResult represents the JSON output from the Python script
type pythonSlideResult struct {
	Success    bool                   `json:"success"`
	SlideCount int                    `json:"slide_count"`
	Slides     []pythonSlideImageData `json:"slides,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// pythonSlideImageData represents slide image data from Python script
type pythonSlideImageData struct {
	SlideNumber   int    `json:"slide_number"`
	ThumbnailPath string `json:"thumbnail_path"`
	FullsizePath  string `json:"fullsize_path"`
}

// ExtractSlides extracts slide images from a PPTX file
func (e *SlideExtractor) ExtractSlides(ctx context.Context, pptxPath string, outputDir string) (int, error) {
	// Check for early cancellation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Validate pptxPath (input file must exist)
	if err := e.validateInputPath(pptxPath); err != nil {
		return 0, fmt.Errorf("invalid pptx path: %w", err)
	}

	// Validate outputDir format (directory doesn't need to exist yet)
	if err := e.validateOutputPath(outputDir); err != nil {
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

	// Get Python command (uses PythonDepsManager like PPTXGenerator)
	pythonCmd, pythonCmdArgs, err := e.depsManager.GetPythonCommand(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to find Python interpreter: %w", err)
	}

	// Prepare command arguments: python3 extract_slides.py <pptxPath> <outputDir>
	// Prepend pythonCmdArgs (e.g., ["run", "python"] for uv)
	args := append(pythonCmdArgs, e.pythonScript, pptxPath, outputDir)

	e.logger.Info("Executing Python slide extraction script",
		zap.String("python_cmd", pythonCmd),
		zap.String("script", e.pythonScript),
		zap.String("pptx_path", pptxPath),
		zap.String("output_dir", outputDir))

	// Execute Python script
	cmd := exec.CommandContext(ctx, pythonCmd, args...)

	// Capture output (stdout only, per Phase 2 D-14 decision)
	output, err := cmd.Output()
	if err != nil {
		e.logger.Error("Python extract script failed",
			zap.String("command", pythonCmd),
			zap.String("script", e.pythonScript),
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

// validateInputPath validates that an input file path is safe and exists
func (e *SlideExtractor) validateInputPath(path string) error {
	// Check for suspicious characters that could enable injection
	if strings.ContainsAny(path, "\n\r\t") {
		return fmt.Errorf("path contains invalid characters")
	}

	// Validate the path exists and is accessible
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("path not accessible: %w", err)
	}

	return nil
}

// validateOutputPath validates that an output directory path is safe (doesn't need to exist yet)
func (e *SlideExtractor) validateOutputPath(path string) error {
	// Check for suspicious characters that could enable injection
	if strings.ContainsAny(path, "\n\r\t") {
		return fmt.Errorf("path contains invalid characters")
	}

	return nil
}
