// Command error-doc-gen scans internal/errors/errors.go and
// internal/errors/mapping.go to emit docs/errors.md.
//
// Generated content (per Phase 20 CONTEXT D-04.2 / D-04.3 / D-04.5):
//
//   - Header describing auto-generation contract.
//   - Sentinel table: Sentinel | Kind | HTTP Status | Call-site count
//     (>=42 rows; one per ErrXxx = errors.New(...) in errors.go).
//   - BusinessError sub-table: 10 Code constants → HTTP status.
//   - Ad-hoc audit footer: inline err.Error() classify branches remaining
//     in internal/handlers/*.go (target: 0 after Phase 20 convergence).
//
// The binary is wired via `//go:generate go run ./cmd/error-doc-gen` in
// internal/errors/errors.go and enforced in CI (no Makefile per R-2).
//
// Triggered locally:
//
//	go run ./cmd/error-doc-gen \
//	    -errors-file internal/errors/errors.go \
//	    -mapping-file internal/errors/mapping.go \
//	    -output docs/errors.md \
//	    -repo-root .
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Sentinel captures one entry from errors.go's `var (...)` block.
type Sentinel struct {
	Name    string // e.g. "ErrNotFound"
	Message string // e.g. "resource not found"
}

// Code captures one entry from errors.go's `const (...)` block.
type Code struct {
	Name  string // e.g. "CodeNotFound"
	Value string // e.g. "NOT_FOUND"
}

// sentinelRow is one rendered row in the sentinel table.
type sentinelRow struct {
	Name       string
	HTTPStatus int
	CallSites  int
}

// codeRow is one rendered row in the BusinessError table.
type codeRow struct {
	Name       string
	HTTPStatus int
	CallSites  int
}

// Pattern groups the regexes used to parse source files. Compiled once.
type patterns struct {
	sentinel     *regexp.Regexp
	code         *regexp.Regexp
	caseSentinel *regexp.Regexp
	caseCode     *regexp.Regexp
	returnHTTP   *regexp.Regexp
}

func compilePatterns() *patterns {
	return &patterns{
		sentinel: regexp.MustCompile(
			`(?m)^\s*(Err\w+)\s*=\s*errors\.New\("([^"]*)"\)`,
		),
		code: regexp.MustCompile(
			`(?m)^\s*(Code\w+)\s*=\s*"([^"]*)"`,
		),
		// Matches both `case errors.Is(err, ErrXxx),` (the head of a multi-line
		// case clause) and continuation lines like
		// `errors.Is(err, ErrYyy),` inside the same case.
		caseSentinel: regexp.MustCompile(
			`(?m)^\s*(?:case\s+)?errors\.Is\(err,\s*(Err\w+)\)`,
		),
		caseCode: regexp.MustCompile(
			`(?m)^\s*case\s+((?:Code\w+\s*,\s*)*Code\w+)\s*:`,
		),
		returnHTTP: regexp.MustCompile(
			`(?m)return\s+http\.Status(\w+)\s*,`,
		),
	}
}

// httpStatusConst is a numeric constant from net/http (subset of what we need).
type httpStatusConst struct {
	Name string
	Code int
}

// netHTTPStatusConsts returns the well-known subset of net/http status codes
// referenced by mapping.go. We hard-code the subset to avoid a runtime import
// of net/http (the generator depends only on std library text/regex utilities).
//
// Source of truth: https://pkg.go.dev/net/http#pkg-constants
// Listed in numeric order for stability.
func netHTTPStatusConsts() []httpStatusConst {
	return []httpStatusConst{
		{"OK", 200},
		{"BadRequest", 400},
		{"Unauthorized", 401},
		{"Forbidden", 403},
		{"NotFound", 404},
		{"Conflict", 409},
		{"TooManyRequests", 429},
		{"InternalServerError", 500},
		{"ServiceUnavailable", 503},
	}
}

// resolveRelative returns the first path that exists when probed in this
// order: as-given from cwd → from each ancestor directory up to the
// module root (directory containing go.mod). If nothing matches, it
// returns the as-given path (which will fail downstream with a clear
// "no such file" error).
func resolveRelative(p string) string {
	if _, err := os.Stat(p); err == nil {
		return p
	}
	// Walk up from cwd looking for go.mod, then try the path anchored at
	// each ancestor directory.
	dir, err := os.Getwd()
	if err != nil {
		return p
	}
	for {
		candidate := filepath.Join(dir, p)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		// Stop at filesystem root.
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		// Stop at module root (directory containing go.mod).
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		dir = parent
	}
	return p
}

// Generate is the testable core of the generator. It scans the given Go
// source files, computes HTTP statuses / call-site counts, and writes
// the rendered markdown to outputPath. The repoRoot is used as the base
// for the call-site grep (which searches "internal/" under repoRoot).
//
// If any of the path arguments is relative and not found as-given, the
// generator walks up from the current working directory looking for a
// go.mod file (the module root). This makes the generator robust when
// invoked via `go generate ./internal/errors/...`, which sets the
// working directory to the package directory containing the directive.
func Generate(errorsPath, mappingPath, repoRoot, outputPath string) error {
	if errorsPath == "" || mappingPath == "" || outputPath == "" {
		return fmt.Errorf("Generate: all of errorsPath, mappingPath, outputPath must be non-empty")
	}

	if !filepath.IsAbs(errorsPath) {
		errorsPath = resolveRelative(errorsPath)
	}
	if !filepath.IsAbs(mappingPath) {
		mappingPath = resolveRelative(mappingPath)
	}
	if !filepath.IsAbs(repoRoot) {
		repoRoot = resolveRelative(repoRoot)
	}
	if !filepath.IsAbs(outputPath) {
		outputPath = resolveRelative(outputPath)
	}

	errorsSrc, err := os.ReadFile(errorsPath)
	if err != nil {
		return fmt.Errorf("read errors file %s: %w", errorsPath, err)
	}
	mappingSrc, err := os.ReadFile(mappingPath)
	if err != nil {
		return fmt.Errorf("read mapping file %s: %w", mappingPath, err)
	}

	pats := compilePatterns()
	statusByName := indexHTTPStatusConsts(pats)

	sentinels := parseSentinels(string(errorsSrc), pats)
	codes := parseCodes(string(errorsSrc), pats)
	statusBySentinel := mapSentinelsToStatus(string(mappingSrc), pats, statusByName)
	statusByCode := mapCodesToStatus(string(mappingSrc), pats, statusByName)
	// Apply default switch branch status to any Code constants not explicitly listed.
	if defaultStatus, ok := statusByCode["__default__"]; ok {
		for _, c := range codes {
			if _, listed := statusByCode[c.Name]; !listed {
				statusByCode[c.Name] = defaultStatus
			}
		}
		delete(statusByCode, "__default__")
	}
	// Apply default switch branch status to any sentinels not explicitly listed.
	// Mirrors the BusinessError logic above; without this every sentinel routed
	// via the default branch renders as HTTP 0 in the generated docs (WR-01).
	if defaultStatus, ok := statusBySentinel["__default__"]; ok {
		for _, s := range sentinels {
			if _, listed := statusBySentinel[s.Name]; !listed {
				statusBySentinel[s.Name] = defaultStatus
			}
		}
		delete(statusBySentinel, "__default__")
	}

	sentinelCallSites := countCallSites(sentinels, repoRoot)
	codeCallSites := countCodeCallSites(codes, repoRoot)

	adHocCount := auditAdHocErrors(repoRoot)

	doc := renderMarkdown(sentinels, codes, statusBySentinel, statusByCode,
		sentinelCallSites, codeCallSites, adHocCount)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("mkdir for output: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(doc), 0o644); err != nil {
		return fmt.Errorf("write output %s: %w", outputPath, err)
	}
	return nil
}

func indexHTTPStatusConsts(pats *patterns) map[string]int {
	// Build from the hard-coded subset. We deliberately do NOT load the
	// net/http source — the mapping only references a small known subset
	// and a fixed list keeps the generator deterministic and dependency-free.
	idx := make(map[string]int, len(netHTTPStatusConsts()))
	for _, s := range netHTTPStatusConsts() {
		idx[s.Name] = s.Code
	}
	return idx
}

func parseSentinels(src string, pats *patterns) []Sentinel {
	matches := pats.sentinel.FindAllStringSubmatch(src, -1)
	out := make([]Sentinel, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, Sentinel{Name: name, Message: m[2]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func parseCodes(src string, pats *patterns) []Code {
	matches := pats.code.FindAllStringSubmatch(src, -1)
	out := make([]Code, 0, len(matches))
	seen := make(map[string]bool, len(matches))
	for _, m := range matches {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, Code{Name: name, Value: m[2]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// caseBranch groups a queue of sentinels/codes waiting for the next
// `return http.StatusXxx, ...` line within the same case clause.
type caseBranch struct {
	sentinels []string
	codes     []string
}

// mapSentinelsToStatus walks the MapToHTTPStatus switch and assigns each
// sentinel to the HTTP status of the case it's listed under. The switch
// structure has the form:
//
//	case errors.Is(err, ErrNotFound),
//	    errors.Is(err, ErrTaskNotFound),
//	    ...:
//	    return http.StatusNotFound, ...
//
// so each `case` may list multiple sentinels, all sharing one `return`.
// We accumulate sentinels per case branch and apply the next matching
// `return http.StatusXxx, ...` line as their status.
//
// The default branch is captured separately under the "__default__" key so
// the caller can apply it to sentinels that fall through (e.g. ErrInternal,
// ErrDuplicateRecord, ErrForeignKeyConstraint). Without this, the rendered
// docs would show HTTP status 0 for those entries (WR-01).
func mapSentinelsToStatus(src string, pats *patterns, statusByName map[string]int) map[string]int {
	out := make(map[string]int)
	lines := strings.Split(src, "\n")
	var branch *caseBranch
	inSwitch := false
	switchDepth := 0
	var defaultStatus int

	flush := func() {
		if branch == nil {
			return
		}
		// Will be called once we've read the `return http.StatusXxx, ...` line.
		branch = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "switch") {
			inSwitch = true
			switchDepth = 1
			branch = nil
			continue
		}
		if !inSwitch {
			continue
		}
		switchDepth += strings.Count(line, "{") - strings.Count(line, "}")

		// Sentinel case: `case errors.Is(err, ErrXxx),`
		if m := pats.caseSentinel.FindStringSubmatch(line); m != nil {
			if branch == nil {
				branch = &caseBranch{}
			}
			branch.sentinels = append(branch.sentinels, m[1])
			continue
		}

		// Default branch: remember its status for unmatched sentinels.
		if strings.HasPrefix(trimmed, "default:") {
			branch = &caseBranch{}
			continue
		}

		// Return line: apply status to the open branch.
		if m := pats.returnHTTP.FindStringSubmatch(line); m != nil {
			if branch != nil {
				if code, ok := statusByName[m[1]]; ok {
					if len(branch.sentinels) == 0 {
						// Default branch return — remember for later.
						defaultStatus = code
					} else {
						for _, s := range branch.sentinels {
							out[s] = code
						}
					}
				}
				flush()
			}
			continue
		}

		if switchDepth <= 0 {
			inSwitch = false
			switchDepth = 0
			branch = nil
		}
	}

	// Expose default branch status to the caller via the reserved "__default__" key.
	if defaultStatus != 0 {
		out["__default__"] = defaultStatus
	}
	return out
}

// mapCodesToStatus walks the mapBusinessError switch and assigns each Code
// constant to the HTTP status of the case it's listed under. Only cases
// whose label matches `case CodeXxx:` are captured (skipping the sentinel
// switch's `case errors.Is(...)` entries — those are handled by
// mapSentinelsToStatus). Codes that are not matched by any explicit case
// fall through to the `default:` branch, whose status we also capture.
func mapCodesToStatus(src string, pats *patterns, statusByName map[string]int) map[string]int {
	out := make(map[string]int)
	lines := strings.Split(src, "\n")
	var branch *caseBranch
	inSwitch := false
	switchDepth := 0
	var defaultStatus int

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "switch") {
			inSwitch = true
			switchDepth = 1
			branch = nil
			continue
		}
		if !inSwitch {
			continue
		}
		switchDepth += strings.Count(line, "{") - strings.Count(line, "}")

		// Code case: `case CodeXxx:` or `case CodeXxx, CodeYyy:`
		if m := pats.caseCode.FindStringSubmatch(line); m != nil {
			if branch == nil {
				branch = &caseBranch{}
			}
			// m[1] may contain comma-separated names like "CodeAlreadyExists, CodeTaskInProgress".
			for _, name := range strings.Split(m[1], ",") {
				name = strings.TrimSpace(name)
				if name != "" {
					branch.codes = append(branch.codes, name)
				}
			}
			continue
		}

		// Default branch: remember its status for unmatched codes.
		if strings.HasPrefix(trimmed, "default:") {
			branch = &caseBranch{}
			continue
		}

		if m := pats.returnHTTP.FindStringSubmatch(line); m != nil {
			if branch != nil {
				if code, ok := statusByName[m[1]]; ok {
					if len(branch.codes) == 0 {
						// Default branch return — remember for later.
						defaultStatus = code
					} else {
						for _, c := range branch.codes {
							out[c] = code
						}
					}
				}
				branch = nil
			}
			continue
		}

		if switchDepth <= 0 {
			inSwitch = false
			switchDepth = 0
			branch = nil
		}
	}

	// Apply default status to codes not explicitly listed in the switch
	// (i.e. they fall through to the default case). The caller passes the
	// complete Code set, so we read it from the file again via parseCodes —
	// but we don't have it here. Instead, callers fill in zero entries by
	// iterating parseCodes() output. We just need to return the default.
	if defaultStatus != 0 {
		// Store under a sentinel key for the caller to apply.
		out["__default__"] = defaultStatus
	}
	return out
}

// countCallSites does a deterministic word-boundary grep of internal/**/*.go
// for each sentinel name and counts total matches. Implements the D-04.3
// Claude's Discretion: rough grep is sufficient (no AST/regex for callers).
func countCallSites(sentinels []Sentinel, repoRoot string) map[string]int {
	out := make(map[string]int, len(sentinels))
	internalDir := filepath.Join(repoRoot, "internal")
	for _, s := range sentinels {
		// Pattern: \b<Name>\b — word boundary, so ErrNotFound won't match
		// ErrNotFoundExt or similar. We compile one regex per sentinel for
		// clarity rather than per-line substring search.
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(s.Name) + `\b`)
		count := grepCount(internalDir, pattern, false)
		out[s.Name] = count
	}
	return out
}

// countCodeCallSites counts how often each Code constant is referenced.
// Pattern: `Code<Name>` (must include the prefix so we don't match unrelated
// identifiers like CodeReviewer).
func countCodeCallSites(codes []Code, repoRoot string) map[string]int {
	out := make(map[string]int, len(codes))
	internalDir := filepath.Join(repoRoot, "internal")
	for _, c := range codes {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(c.Name) + `\b`)
		count := grepCount(internalDir, pattern, false)
		out[c.Name] = count
	}
	return out
}

// auditAdHocErrors counts inline `err.Error()` classify branches remaining in
// internal/handlers/. The pattern targets the Phase 17 / 19 anti-pattern:
// `c.JSON(...)` or `response.GinError(c, ..., err.Error())` style ad-hoc
// classification. Per D-04.5 the target is 0 after convergence.
func auditAdHocErrors(repoRoot string) int {
	handlersDir := filepath.Join(repoRoot, "internal", "handlers")
	// Two patterns: (a) `response.GinError(c, ..., err.Error())` style
	// and (b) bare `err.Error()` inside handlers — a strong signal of
	// ad-hoc classification. We exclude `_test.go` and lines inside
	// `ShouldBindJSON` blocks (request-binding validation errors are
	// expected in handlers, not service-classifier ad-hoc counts).
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`\berr\.Error\(\)`),
		regexp.MustCompile(`\berrMsg\s*:=`),
	}
	count := 0
	for _, pat := range patterns {
		count += grepCount(handlersDir, pat, true)
	}
	return count
}

// grepCount walks dir, returning the total number of matching lines across
// all *.go files (excluding _test.go if excludeTests is true). Files are
// walked directly (filepath.WalkDir) — no exec of external grep, so the
// generator stays portable to Windows without Git Bash / Cygwin.
//
// Lines inside a `ShouldBindJSON` block are excluded by source context,
// not by filename (the previous implementation checked `HasSuffix(name,
// "ShouldBindJSON")` which could never match a `.go` filename). WR-02.
func grepCount(dir string, pattern *regexp.Regexp, excludeTests bool) int {
	count := 0
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		if excludeTests && strings.HasSuffix(name, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		count += grepCountInSource(string(data), pattern)
		return nil
	})
	return count
}

// grepCountInSource counts lines matching pattern in src, excluding lines
// that fall inside a `c.ShouldBindJSON(...)` block. Block boundaries are
// tracked by counting `{` and `}` from the line that contains the binding
// call until the depth returns to 0. This is the WR-02 fix: the previous
// implementation checked filename suffix, which is impossible to match.
func grepCountInSource(src string, pattern *regexp.Regexp) int {
	count := 0
	lines := strings.Split(src, "\n")
	inBindingBlock := false
	blockDepth := 0

	for _, line := range lines {
		if inBindingBlock {
			opens := strings.Count(line, "{")
			closes := strings.Count(line, "}")
			blockDepth += opens
			blockDepth -= closes
			if blockDepth <= 0 {
				inBindingBlock = false
				blockDepth = 0
			}
			continue
		}
		if strings.Contains(line, "ShouldBindJSON") {
			opens := strings.Count(line, "{")
			closes := strings.Count(line, "}")
			blockDepth = opens - closes
			if blockDepth > 0 {
				inBindingBlock = true
			}
			continue
		}
		if pattern.MatchString(line) {
			count++
		}
	}
	return count
}

func renderMarkdown(
	sentinels []Sentinel,
	codes []Code,
	statusBySentinel map[string]int,
	statusByCode map[string]int,
	sentinelCallSites map[string]int,
	codeCallSites map[string]int,
	adHocCount int,
) string {
	var b bytes.Buffer

	fmt.Fprintln(&b, "# Error Sentinel Reference")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "> Auto-generated by `cmd/error-doc-gen`.")
	fmt.Fprintln(&b, "> Do not edit by hand — regenerate via `go generate ./internal/errors/...`.")
	fmt.Fprintln(&b, "> Source of truth: `internal/errors/errors.go` (sentinel/Code definitions) +")
	fmt.Fprintln(&b, "> `internal/errors/mapping.go` (`MapToHTTPStatus` switch → HTTP status).")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "## Sentinel Table")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Sentinel | Kind | HTTP Status | Call-site count |")
	fmt.Fprintln(&b, "|----------|------|-------------|-----------------|")
	for _, s := range sentinels {
		status := statusBySentinel[s.Name]
		fmt.Fprintf(&b, "| %s | Sentinel | %d | %d |\n",
			s.Name, status, sentinelCallSites[s.Name])
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## BusinessError Table")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Errors returned as `internal/errors.BusinessError{Code: ...}`. HTTP status is")
	fmt.Fprintln(&b, "resolved by `mapBusinessError` in `internal/errors/mapping.go`.")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| Sentinel | Kind | HTTP Status | Call-site count |")
	fmt.Fprintln(&b, "|----------|------|-------------|-----------------|")
	for _, c := range codes {
		status := statusByCode[c.Name]
		fmt.Fprintf(&b, "| BusinessError(code=%s) | BusinessError | %d | %d |\n",
			c.Value, status, codeCallSites[c.Name])
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## Ad-hoc Error Audit")
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "Counts of remaining inline `err.Error()` classify branches in")
	fmt.Fprintln(&b, "`internal/handlers/*.go` (excluding `_test.go` and `ShouldBindJSON`).")
	fmt.Fprintln(&b, "After Phase 20 convergence (`20-02` + `20-03`) the target is **0**.")
	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "> Current `err.Error()` / inline classify branches: **%d** (target: 0)\n", adHocCount)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "If this number grows, a handler has regressed to the pre-Phase-20 anti-pattern.")
	fmt.Fprintln(&b, "Move the classification into the service layer (return a sentinel / `BusinessError`)")
	fmt.Fprintln(&b, "and let `response.HandleError` route via `internal/errors/mapping.go`.")
	fmt.Fprintln(&b)

	return b.String()
}

func main() {
	fs := flag.NewFlagSet("error-doc-gen", flag.ExitOnError)
	errorsFile := fs.String("errors-file", "internal/errors/errors.go", "path to internal/errors/errors.go")
	mappingFile := fs.String("mapping-file", "internal/errors/mapping.go", "path to internal/errors/mapping.go")
	output := fs.String("output", "docs/errors.md", "path to output markdown file")
	repoRoot := fs.String("repo-root", ".", "repository root for grep-based call-site count")
	_ = fs.Parse(os.Args[1:])

	if err := Generate(*errorsFile, *mappingFile, *repoRoot, *output); err != nil {
		fmt.Fprintf(os.Stderr, "error-doc-gen: %v\n", err)
		os.Exit(1)
	}
}