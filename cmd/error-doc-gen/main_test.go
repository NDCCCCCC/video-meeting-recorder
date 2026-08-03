package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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

// TestGenerate_SentinelTableStatuses asserts that every sentinel row has a
// valid HTTP status (WR-01). The previous implementation only captured
// sentinels explicitly named in a case clause; sentinels routed via the
// default branch (ErrInternal, ErrDuplicateRecord, ErrForeignKeyConstraint)
// rendered as HTTP status 0.
func TestGenerate_SentinelTableStatuses(t *testing.T) {
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

	// Valid HTTP statuses are 100..599 (covers 1xx..5xx informational
	// and error codes). Anything outside that range is a parser bug.
	const minHTTP, maxHTTP = 100, 599
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "| Err") {
			continue
		}
		fields := strings.Split(line, "|")
		// fields: [empty, ErrName, Sentinel, HTTPStatus, CallSiteCount, empty]
		if len(fields) < 5 {
			continue
		}
		statusStr := strings.TrimSpace(fields[3])
		status, err := strconv.Atoi(statusStr)
		if err != nil {
			t.Errorf("sentinel row has non-numeric HTTP status %q: %s", statusStr, line)
			continue
		}
		if status < minHTTP || status > maxHTTP {
			t.Errorf("sentinel row has invalid HTTP status %d: %s", status, line)
		}
	}

	// Specific assertions for the bug fixed by WR-01: sentinels that
	// fall through to the default branch must render as HTTP 500.
	for _, sentinel := range []string{
		"ErrInternal", "ErrDuplicateRecord", "ErrForeignKeyConstraint",
	} {
		row := findSentinelRow(content, sentinel)
		if row == "" {
			t.Errorf("missing sentinel row for %s", sentinel)
			continue
		}
		if !strings.Contains(row, "| 500 |") {
			t.Errorf("%s expected HTTP 500, got row: %s", sentinel, row)
		}
	}
}

// findSentinelRow returns the markdown table row for the given sentinel name.
func findSentinelRow(content, sentinel string) string {
	for _, line := range strings.Split(content, "\n") {
		if strings.Contains(line, "| "+sentinel+" |") {
			return line
		}
	}
	return ""
}

// TestGrepCount_ExcludesShouldBindJSONBlock verifies the audit's
// ShouldBindJSON exclusion uses source context (block-tracking), not
// filename suffix (which can never match a `.go` file). The fixture
// contains both a `ShouldBindJSON` block that must NOT count and a
// real `err.Error()` classifier that MUST count.
func TestGrepCount_ExcludesShouldBindJSONBlock(t *testing.T) {
	pat := regexp.MustCompile(`\berr\.Error\(\)`)
	src := `package handlers

func (h *Handler) Foo(c *gin.Context) {
	var req struct {
		Name string ` + "`json:\"name\"`" + `
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "request error: "+err.Error())
		return
	}
}

func (h *Handler) Bar(c *gin.Context) {
	result, err := h.svc.Get(c)
	if err != nil {
		c.JSON(500, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(200, result)
}
`

	count := grepCountInSource(src, pat)
	// Only the err.Error() in Bar() should count; the ShouldBindJSON
	// block in Foo() must be excluded.
	if count != 1 {
		t.Errorf("expected 1 classified err.Error() (Foo excluded), got %d", count)
	}
}

// TestGrepCount_NestedBindingBlock verifies the brace tracker correctly
// exits nested blocks (e.g. when a `ShouldBindJSON` branch contains
// helper calls inside a nested `if`).
func TestGrepCount_NestedBindingBlock(t *testing.T) {
	pat := regexp.MustCompile(`\berr\.Error\(\)`)
	src := `package handlers

func (h *Handler) Foo(c *gin.Context) {
	if err := c.ShouldBindJSON(&req); err != nil {
		if err.Error() != "bad" {
			c.JSON(400, gin.H{"msg": err.Error()})
		}
		return
	}
}

func (h *Handler) Bar(c *gin.Context) {
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := h.svc.Do(); err != nil {
		c.JSON(500, err.Error())
	}
}
`

	count := grepCountInSource(src, pat)
	// Both Foo() and Bar() include ShouldBindJSON blocks that must be
	// excluded; only the Bar() tail `err.Error()` outside the binding
	// block must count.
	if count != 1 {
		t.Errorf("expected 1 classified err.Error() (both binding blocks excluded), got %d", count)
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
