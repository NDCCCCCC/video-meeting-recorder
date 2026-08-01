package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot returns the absolute path to the repository root for the test
// environment. Falls back to two levels up from this test file (i.e. the
// layout that matches cmd/error-doc-gen living under <repo>/cmd/error-doc-gen).
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed")
	}
	// this file lives at <repo>/cmd/error-doc-gen/main_test.go
	dir := filepath.Dir(thisFile)
	root := filepath.Clean(filepath.Join(dir, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("cannot locate repo root (missing go.mod at %s): %v", root, err)
	}
	return root
}

// TestGenerate_SentinelTableComplete verifies the generator emits the full
// sentinel table (>=42 rows) when scanning the real errors.go file.
func TestGenerate_SentinelTableComplete(t *testing.T) {
	root := repoRoot(t)
	errorsFile := filepath.Join(root, "internal", "errors", "errors.go")
	mappingFile := filepath.Join(root, "internal", "errors", "mapping.go")
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "errors.md")

	if err := Generate(errorsFile, mappingFile, root, outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	content := string(data)

	// Must have a header
	if !strings.Contains(content, "Sentinel") {
		t.Errorf("output missing 'Sentinel' header; got:\n%s", content)
	}

	// Count sentinel rows: lines beginning with "| Err" (not the BusinessError table).
	rows := countRowsWithPrefix(content, "| Err")
	if rows < 42 {
		t.Errorf("expected >= 42 sentinel rows, got %d", rows)
	}

	// Each sentinel row must include kind and HTTP status columns.
	if !strings.Contains(content, "| Sentinel |") {
		t.Errorf("output missing '| Sentinel |' rows; got:\n%s", content)
	}
}

// TestGenerate_BusinessErrorTableComplete verifies the generator emits the
// BusinessError sub-table with all 10 Code constants mapped to HTTP statuses.
func TestGenerate_BusinessErrorTableComplete(t *testing.T) {
	root := repoRoot(t)
	errorsFile := filepath.Join(root, "internal", "errors", "errors.go")
	mappingFile := filepath.Join(root, "internal", "errors", "mapping.go")
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "errors.md")

	if err := Generate(errorsFile, mappingFile, root, outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "BusinessError") {
		t.Errorf("output missing 'BusinessError' table; got:\n%s", content)
	}

	// Count BusinessError(code=XXX) rows.
	rows := countRowsWithPrefix(content, "| BusinessError(code=")
	if rows != 10 {
		t.Errorf("expected exactly 10 BusinessError rows, got %d", rows)
	}
}

// TestGenerate_MissingFilepath verifies the generator fails cleanly when given
// a non-existent path (no panic, returns error).
func TestGenerate_MissingFilepath(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "errors.md")

	// Non-existent errors file path.
	err := Generate(
		filepath.Join(tmpDir, "no-such-errors.go"),
		filepath.Join(tmpDir, "no-such-mapping.go"),
		tmpDir,
		outPath,
	)
	if err == nil {
		t.Fatal("expected error for missing filepath, got nil")
	}
}

// TestGenerate_Deterministic verifies that running the generator twice on the
// same input produces byte-identical output (no timestamps, no randomness).
func TestGenerate_Deterministic(t *testing.T) {
	root := repoRoot(t)
	errorsFile := filepath.Join(root, "internal", "errors", "errors.go")
	mappingFile := filepath.Join(root, "internal", "errors", "mapping.go")
	tmpDir := t.TempDir()
	outA := filepath.Join(tmpDir, "a.md")
	outB := filepath.Join(tmpDir, "b.md")

	if err := Generate(errorsFile, mappingFile, root, outA); err != nil {
		t.Fatalf("first Generate failed: %v", err)
	}
	if err := Generate(errorsFile, mappingFile, root, outB); err != nil {
		t.Fatalf("second Generate failed: %v", err)
	}

	a, err := os.ReadFile(outA)
	if err != nil {
		t.Fatalf("read a: %v", err)
	}
	b, err := os.ReadFile(outB)
	if err != nil {
		t.Fatalf("read b: %v", err)
	}

	if !bytes.Equal(a, b) {
		t.Fatalf("output is not deterministic:\n--- A ---\n%s\n--- B ---\n%s", string(a), string(b))
	}
}

// TestGenerate_AuditFooter verifies the generator writes an ad-hoc audit
// footer counting inline classify branches in internal/handlers/.
func TestGenerate_AuditFooter(t *testing.T) {
	root := repoRoot(t)
	errorsFile := filepath.Join(root, "internal", "errors", "errors.go")
	mappingFile := filepath.Join(root, "internal", "errors", "mapping.go")
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "errors.md")

	if err := Generate(errorsFile, mappingFile, root, outPath); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	content := string(data)

	// Footer must mention "ad-hoc" so reviewers see the audit signal.
	if !strings.Contains(strings.ToLower(content), "ad-hoc") {
		t.Errorf("output missing 'ad-hoc' audit footer; got:\n%s", content)
	}
}

// countRowsWithPrefix counts non-header lines that start with the given
// prefix (used to assert table row counts).
func countRowsWithPrefix(content, prefix string) int {
	n := 0
	for _, line := range strings.Split(content, "\n") {
		s := strings.TrimSpace(line)
		if strings.HasPrefix(s, prefix) {
			n++
		}
	}
	return n
}