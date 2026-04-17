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

// PPTXGenerator generates PowerPoint files from image frames using Python script
type PPTXGenerator struct {
	logger       *zap.Logger
	pythonScript string
}

// NewPPTXGenerator creates a new PPTXGenerator instance
func NewPPTXGenerator(logger *zap.Logger) *PPTXGenerator {
	// Get the project root directory (assuming we're in internal/services during tests)
	// For production use, this should be configurable
	projectRoot := getProjectRoot()
	return &PPTXGenerator{
		logger:       logger,
		pythonScript: filepath.Join(projectRoot, "scripts", "create_pptx.py"),
	}
}

// getProjectRoot attempts to find the project root directory
func getProjectRoot() string {
	// Start from current directory and search for go.mod
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}

	// Search up for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, return current directory
			return "."
		}
		dir = parent
	}
}

const (
	// 16:9 slide dimensions in inches (for reference)
	SLIDE_WIDTH_INCH  = 10.0
	SLIDE_HEIGHT_INCH = 5.625

	// Maximum number of slides to prevent OOM (T-02-06 mitigation)
	MAX_SLIDES = 500
)

// pythonResult represents the JSON output from the Python script
type pythonResult struct {
	Success     bool   `json:"success"`
	PageCount   int    `json:"page_count"`
	OutputPath  string `json:"output_path"`
	SkippedCount int   `json:"skipped_count,omitempty"`
	Error       string `json:"error,omitempty"`
}

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

	// Sanitize and validate all frame paths to prevent command injection
	sanitizedPaths := make([]string, len(framePaths))
	for i, path := range framePaths {
		sanitizedPaths[i] = filepath.Clean(path)
		// Validate path is safe
		if err := g.validatePath(path); err != nil {
			return 0, fmt.Errorf("invalid frame path at index %d: %w", i, err)
		}
	}

	// Prepare command arguments
	// Usage: python create_pptx.py <output_path> <image1> <image2> ...
	args := append([]string{g.pythonScript, outputPath}, sanitizedPaths...)

	// Execute Python script
	// Try 'python3' first, then fall back to 'python'
	cmdName := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		cmdName = "python"
	}
	cmd := exec.CommandContext(ctx, cmdName, args...)

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		g.logger.Error("Python script failed",
			zap.String("output", string(output)),
			zap.Error(err))
		return 0, fmt.Errorf("failed to generate PPTX: %w (output: %s)", err, string(output))
	}

	// Parse JSON result
	var result pythonResult
	if err := json.Unmarshal(output, &result); err != nil {
		g.logger.Error("Failed to parse Python output",
			zap.String("output", string(output)),
			zap.Error(err))
		return 0, fmt.Errorf("failed to parse Python output: %w (output: %s)", err, string(output))
	}

	// Check for success
	if !result.Success {
		g.logger.Error("Python script reported failure",
			zap.String("error", result.Error))
		return 0, fmt.Errorf("PPTX generation failed: %s", result.Error)
	}

	// Verify page count
	if result.PageCount == 0 {
		return 0, fmt.Errorf("no slides were created from %d input frames", len(framePaths))
	}

	g.logger.Info("PPTX generated successfully",
		zap.String("output", outputPath),
		zap.Int("page_count", result.PageCount),
		zap.Int("skipped_count", result.SkippedCount))

	return result.PageCount, nil
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

// CheckPythonAvailability checks if python and python-pptx are available
func (g *PPTXGenerator) CheckPythonAvailability(ctx context.Context) error {
	// Determine python command name
	cmdName := "python3"
	if _, err := exec.LookPath("python3"); err != nil {
		cmdName = "python"
	}

	// Check if python is available
	cmd := exec.CommandContext(ctx, cmdName, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("python not available: %w", err)
	}

	// Check if python-pptx is installed
	cmd = exec.CommandContext(ctx, cmdName, "-c", "import pptx; print(pptx.__version__)")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python-pptx not installed: %w (install with: pip install python-pptx)", err)
	}

	version := strings.TrimSpace(string(output))
	g.logger.Info("python-pptx available", zap.String("version", version), zap.String("command", cmdName))

	return nil
}

// validatePath validates that a path is safe and within allowed directories
func (g *PPTXGenerator) validatePath(path string) error {
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
	// This prevents access to sensitive system directories
	projectRoot := getProjectRoot()
	allowedDir := filepath.Clean(projectRoot)
	if !strings.HasPrefix(absPath, allowedDir) {
		return fmt.Errorf("path outside allowed directory: %s", path)
	}

	return nil
}
