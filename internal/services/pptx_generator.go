package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// PPTXGenerator handles PowerPoint file generation
type PPTXGenerator struct {
	pythonScript  string
	logger        *zap.Logger
	depsManager   *PythonDepsManager
	pythonCmd     string   // "python", "python3", or "uv"
	pythonCmdArgs []string // Additional args (e.g., ["run", "python"] for uv)
	preferUV      bool
	baseDir       string   // Directory where scripts are located (executable directory)
}

// NewPPTXGenerator creates a new PPTX generator instance
func NewPPTXGenerator(logger *zap.Logger, preferUV bool) *PPTXGenerator {
	mgr := NewPythonDepsManager(logger, preferUV)

	// Get the base directory (same as PythonDepsManager uses)
	baseDir := "."
	if exePath, err := os.Executable(); err == nil {
		baseDir = filepath.Dir(exePath)
	}

	return &PPTXGenerator{
		pythonScript: "scripts/create_pptx.py",
		logger:       logger,
		depsManager:  mgr,
		preferUV:     preferUV,
		baseDir:      baseDir,
	}
}

// initializePythonCommand sets up the Python command on first use
func (g *PPTXGenerator) initializePythonCommand(ctx context.Context) error {
	if g.pythonCmd != "" {
		return nil // Already initialized
	}

	cmd, args, err := g.depsManager.GetPythonCommand(ctx)
	if err != nil {
		return fmt.Errorf("failed to find Python interpreter: %w", err)
	}

	g.pythonCmd = cmd
	g.pythonCmdArgs = args

	g.logger.Info("Python command initialized",
		zap.String("command", cmd),
		zap.Strings("args", args))

	return nil
}

// GeneratePPTX creates a PowerPoint file from a list of image frames
func (g *PPTXGenerator) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
	if len(framePaths) == 0 {
		return 0, fmt.Errorf("no frame paths provided")
	}

	// Initialize Python command if not already done
	if err := g.initializePythonCommand(ctx); err != nil {
		return 0, err
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
	// Usage: uv run python scripts/create_pptx.py <output_path> <image1> <image2> ...
	//     or: python3 scripts/create_pptx.py <output_path> <image1> <image2> ...
	args := append(g.pythonCmdArgs, g.pythonScript, outputPath)
	args = append(args, sanitizedPaths...)

	cmd := exec.CommandContext(ctx, g.pythonCmd, args...)
	// Don't set cmd.Dir - use current working directory to resolve paths correctly

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		g.logger.Error("Python script failed",
			zap.String("command", g.pythonCmd),
			zap.String("output", string(output)),
			zap.Error(err))
		return 0, fmt.Errorf("python script failed: %w, output: %s", err, string(output))
	}

	// Parse JSON output from script
	var result struct {
		Success      bool     `json:"success"`
		PageCount    int      `json:"page_count"`
		OutputPath   string   `json:"output_path"`
		Error        string   `json:"error"`
		SkippedCount int      `json:"skipped_count"`
		InputCount   int      `json:"input_count,omitempty"`
		MissingFiles []string `json:"missing_files,omitempty"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		g.logger.Warn("Failed to parse JSON output from create_pptx.py",
			zap.String("output", string(output)),
			zap.Error(err))
		// Try legacy format fallback
		pageCount := len(framePaths)
		return pageCount, nil
	}

	if !result.Success {
		return 0, fmt.Errorf("PPTX generation failed: %s", result.Error)
	}

	// Log if there were missing files (debug for the 2-page issue)
	if result.SkippedCount > 0 {
		g.logger.Warn("PPTX生成时有部分图片文件缺失",
			zap.Int("input_count", len(framePaths)),
			zap.Int("skipped_count", result.SkippedCount),
			zap.Int("pages_created", result.PageCount),
			zap.Any("missing_files", result.MissingFiles))
	}

	// Sanity check: if we expected N pages but got significantly fewer, log it
	if result.InputCount > 0 && result.PageCount < result.InputCount/2 {
		g.logger.Error("PPTX页面数量异常，可能大部分文件丢失",
			zap.Int("expected_frames", result.InputCount),
			zap.Int("actual_pages", result.PageCount),
			zap.Int("skipped_count", result.SkippedCount))
	}

	return result.PageCount, nil
}

// validatePath validates that a path is safe and within allowed directories
func (g *PPTXGenerator) validatePath(path string) error {
	// Check for suspicious characters that could enable injection
	if strings.ContainsAny(path, "\n\r\t") {
		return fmt.Errorf("path contains invalid characters")
	}

	var absPath string
	var err error

	// If path is relative, resolve it relative to project root
	if filepath.IsAbs(path) {
		// Path is already absolute
		absPath = filepath.Clean(path)
	} else {
		// Path is relative - resolve relative to project root
		projectRoot := getProjectRoot()
		absPath = filepath.Join(projectRoot, path)
		absPath = filepath.Clean(absPath)
	}

	// Resolve any remaining .. or . components
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return fmt.Errorf("cannot resolve absolute path: %w", err)
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

// getProjectRoot returns the project root directory by searching for go.mod
func getProjectRoot() string {
	// Start from executable directory instead of current working directory
	// This ensures scripts are found when running from any location
	execPath, err := os.Executable()
	if err != nil {
		// Fallback to current directory if executable path can't be determined
		dir, err := os.Getwd()
		if err != nil {
			return "."
		}
		return findGoMod(dir)
	}

	// Get the directory containing the executable
	execDir := filepath.Dir(execPath)
	return findGoMod(execDir)
}

// findGoMod searches upward from dir for go.mod
func findGoMod(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root, return current directory
			return dir
		}
		dir = parent
	}
}

// CheckPythonDependencies verifies that python and required packages are installed
func (g *PPTXGenerator) CheckPythonDependencies() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := g.depsManager.CheckDependencies(ctx)
	if err != nil {
		return err
	}

	// Cache the command for future use
	g.pythonCmd = info.Command
	if info.Command == "uv" {
		g.pythonCmdArgs = []string{"run", "python"}
	}

	g.logger.Info("Python dependencies verified",
		zap.String("python_version", info.PythonVersion),
		zap.Any("packages", info.Packages))

	return nil
}

// EnsureDependencies attempts to install missing Python dependencies using uv
func (g *PPTXGenerator) EnsureDependencies(ctx context.Context) error {
	return g.depsManager.EnsureDependencies(ctx)
}
