package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"go.uber.org/zap"
)

// PythonDependencyInfo contains information about Python dependencies
type PythonDependencyInfo struct {
	OK            bool              `json:"ok"`
	PythonVersion string            `json:"python_version,omitempty"`
	Packages      map[string]string `json:"packages,omitempty"`
	Error         string            `json:"error,omitempty"`
	Command       string            `json:"command,omitempty"` // "python", "uv", "python3"
}

// PythonDepsManager manages Python dependencies for the application
type PythonDepsManager struct {
	logger      *zap.Logger
	projectRoot string
	checkScript string
	preferUV    bool // Whether to prefer uv over system Python
}

// NewPythonDepsManager creates a new Python dependencies manager
func NewPythonDepsManager(logger *zap.Logger, preferUV bool) *PythonDepsManager {
	// Get the directory where the executable is located
	// Scripts are in ./scripts relative to the executable
	baseDir := "."
	if exePath, err := os.Executable(); err == nil {
		baseDir = filepath.Dir(exePath)
		logger.Debug("Python deps: using executable directory",
			zap.String("exe_path", exePath),
			zap.String("base_dir", baseDir))
	} else {
		logger.Warn("Failed to get executable path, using current directory",
			zap.Error(err), response.SentinelField(err))
		// Fallback to current working directory
		if wd, err := os.Getwd(); err == nil {
			baseDir = wd
		}
	}

	return &PythonDepsManager{
		logger:      logger,
		projectRoot: baseDir,
		checkScript: "scripts/check_python_deps.py",
		preferUV:    preferUV,
	}
}

// CheckDependencies verifies Python and required packages are available
func (m *PythonDepsManager) CheckDependencies(ctx context.Context) (*PythonDependencyInfo, error) {
	// Try different methods in order of preference
	methods := []string{"uv", "python3", "python"}

	if !m.preferUV {
		// If uv is not preferred, try system python first
		methods = []string{"python3", "python", "uv"}
	}

	var lastErr error
	for _, method := range methods {
		info, err := m.checkWithMethod(ctx, method)
		if err == nil && info.OK {
			m.logger.Info("Python dependencies verified",
				zap.String("method", method),
				zap.String("python_version", info.PythonVersion),
				zap.Strings("packages", packageList(info.Packages)),
			)
			return info, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf("all Python methods failed: %w", lastErr)
}

// checkWithMethod checks dependencies using a specific method (uv, python3, python)
func (m *PythonDepsManager) checkWithMethod(ctx context.Context, method string) (*PythonDependencyInfo, error) {
	checkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	switch method {
	case "uv":
		// Use uv run to execute the check script
		// Set working directory to project root and use relative path
		cmd = exec.CommandContext(checkCtx, "uv", "run", "python", m.checkScript)
		cmd.Dir = m.projectRoot
	case "python3", "python":
		// Use system Python directly
		// Set working directory to project root and use relative path
		cmd = exec.CommandContext(checkCtx, method, m.checkScript)
		cmd.Dir = m.projectRoot
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		m.logger.Info("Python check method failed",
			zap.String("method", method),
			zap.String("working_dir", m.projectRoot),
			zap.String("script", m.checkScript),
			zap.String("output", string(output)),
			zap.Error(err),
			response.SentinelField(err),
		)
		return nil, fmt.Errorf("%s check failed: %w", method, err)
	}

	// Extract JSON from output (uv may add warnings before JSON)
	jsonStart := bytes.Index(output, []byte("{"))
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in output")
	}
	jsonOutput := output[jsonStart:]

	var info PythonDependencyInfo
	if err := json.Unmarshal(jsonOutput, &info); err != nil {
		m.logger.Info("Failed to parse Python check output",
			zap.String("method", method),
			zap.String("output", string(output)),
			zap.Error(err),
			response.SentinelField(err),
		)
		return nil, fmt.Errorf("failed to parse check output: %w", err)
	}

	info.Command = method
	return &info, nil
}

// EnsureDependencies attempts to install missing dependencies using uv
func (m *PythonDepsManager) EnsureDependencies(ctx context.Context) error {
	// Check if uv is available
	if _, err := exec.LookPath("uv"); err != nil {
		return fmt.Errorf("uv not found in PATH; install from https://astral.sh/uv: %w", err)
	}

	m.logger.Info("Running uv sync to ensure Python dependencies")

	syncCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(syncCtx, "uv", "sync")
	cmd.Dir = m.projectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("uv sync failed: %w", err)
	}

	m.logger.Info("Python dependencies synchronized successfully")
	return nil
}

// GetPythonCommand returns the best available Python command for executing scripts
func (m *PythonDepsManager) GetPythonCommand(ctx context.Context) (string, []string, error) {
	// If uv is preferred and available, use "uv run python"
	if m.preferUV {
		if _, err := exec.LookPath("uv"); err == nil {
			return "uv", []string{"run", "python"}, nil
		}
	}

	// Fall back to system Python
	for _, cmdName := range []string{"python3", "python"} {
		if _, err := exec.LookPath(cmdName); err == nil {
			return cmdName, nil, nil
		}
	}

	return "", nil, fmt.Errorf("no Python interpreter found")
}

// InstallUV provides instructions for installing uv
func (m *PythonDepsManager) InstallUV() string {
	return `# Install uv (fast Python package manager)

# On Linux/macOS:
curl -LsSf https://astral.sh/uv/install.sh | sh

# On Windows:
powershell -ExecutionPolicy ByPass -c "irm https://astral.sh/uv/install.ps1 | iex"

# Or via pip:
pip install uv

# After installation, run: uv sync`
}

// packageList extracts package names from info map
func packageList(packages map[string]string) []string {
	names := make([]string, 0, len(packages))
	for name := range packages {
		names = append(names, name)
	}
	return names
}

// CheckPythonDependenciesLegacy provides backward-compatible check without uv support
// Deprecated: Use CheckDependencies instead
func CheckPythonDependenciesLegacy(logger *zap.Logger) error {
	mgr := NewPythonDepsManager(logger, false)
	_, err := mgr.CheckDependencies(context.Background())
	return err
}
