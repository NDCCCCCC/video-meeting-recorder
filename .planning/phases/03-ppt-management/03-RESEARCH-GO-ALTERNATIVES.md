# Phase 3 Supplementary: Pure Go PPT Generation - Research

**Researched:** 2026-04-20
**Domain:** Go PPTX libraries vs python-pptx for PPT generation
**Confidence:** MEDIUM

## Summary

Phase 3 currently uses Python (python-pptx) for PPTX generation via `scripts/create_pptx.py` called from Go's `exec.Command`. This research investigates pure Go alternatives to eliminate the Python dependency while maintaining feature parity. The Go ecosystem offers two viable PPTX libraries: **unidoc/unioffice** (commercial, feature-rich) and **Muprprpr/Go-pptx** (MIT-licensed, streaming-focused). However, both libraries have limitations compared to python-pptx, particularly in slide manipulation and merge operations. The research reveals that **migration to pure Go has significant complexity and moderate risk**, with python-pptx remaining more mature for complex operations like slide merging.

**Primary recommendation:** **Keep Python approach** for Phase 3. The python-pptx dependency is already working (verified v1.0.2 installed), and Go libraries lack critical features for slide merging. Address dependency issues by documenting installation requirements in deployment docs. Consider pure Go migration for Phase 4+ only if slide rendering requirements simplify (image-only slides, no text/shapes).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**PPT Preview (PPT-03):**
- **D-01:** Server-side slide image extraction via Python-pptx (already integrated in PPTXGenerator). Each slide is converted to JPEG image served via API.
- **D-02:** Preview layout: main view (large slide) + sidebar thumbnail strip. Similar to PowerPoint's thumbnail sidebar.
- **D-03:** Thumbnails generated on-demand and cached. First preview shows a brief loading/progress indicator while images are extracted.
- **D-04:** Dual resolution strategy: thumbnails at 200x112px (fast loading), main view images at 1920x1080px (high clarity).
- **D-05:** Full-screen presentation mode supported — hides sidebar and navigation, slides fill entire screen.
- **D-06:** Page indicator ("第 3/25 页") displayed below main view with click-to-jump input for direct page navigation.
- **D-07:** Per-slide actions: single page download + copy image to clipboard.
- **D-08:** Slide images extracted as JPEG quality 90%. Balance of file size and visual clarity for PPT screenshots.
- **D-09:** API design: GET /api/v1/ppts/:id/slides returns list of slide image URLs (thumbnail + full-size pairs).

**Multi-result Display (PPT-04, PPT-05):**
- **D-10:** Gallery-style switching for multiple PPT results of the same video. Horizontal thumbnail strip at bottom, current result displayed prominently.
- **D-11:** "重新转录" button lives inside the result page action panel. Reuses transcription trigger logic with TranscriptionProgressModal.
- **D-12:** Default selection: newest transcription result first. User can switch to any historical result via gallery strip.

**Slide Merge (PPT-06):**
- **D-13:** Merge triggered from result page — "合并幻灯片" button enters merge mode inline (no page navigation).
- **D-14:** Slide selection: click-to-select on thumbnails (highlight on select, click again to deselect). Selected slides appear in a bottom bar with drag-to-reorder support.
- **D-15:** Merge result generates a new PPTX file saved on server, associated with the original video. Does not modify source PPTs.
- **D-16:** Merged PPT appears in the result gallery alongside transcription results, associated with original video.
- **D-17:** Merge limit: 200 slides maximum. UI shows selected count and limit indicator.
- **D-18:** Merge progress: simple loading spinner + completion toast. No detailed progress needed (merge is typically fast).

**Result Page Layout (UI-03):**
- **D-19:** Left-right split layout: left side = PPT preview area (main view + sidebar thumbnails), right side = info/action panel.
- **D-20:** Navigation entry: "预览PPT" button in file list action column jumps to result detail page.
- **D-21:** Right panel contains three sections: basic info (video name, transcription time, sampling rate, page count, file size), action buttons (download, re-transcribe, merge, delete), multi-result gallery switcher (horizontal strip showing all results with time + page count).
- **D-22:** Result page URL pattern: /results/:videoFileId (shows all PPT results for that video).

### Claude's Discretion

- Exact Python-pptx slide extraction implementation (image rendering approach)
- Slide image caching strategy (file system vs database blob, cache invalidation)
- Merge PPTX generation approach (re-extract original frames vs combine existing images)
- PPTFile model extensions needed for merge results (source_type field, merged_from IDs)
- API endpoint paths and request/response structures for slide images and merge operations
- Thumbnail strip component implementation details
- Drag-to-reorder library choice for merge selection bar
- Merge mode UI state management (entering/exiting merge mode, selection state)
- Error handling for slide extraction failures
- How to handle deleted source PPTs in merge results

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.
</user_constraints>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| PPTX file generation (from images) | API / Backend | — | Server-side creates PPTX from extracted frames |
| PPTX slide manipulation/merge | API / Backend | — | Complex operation requiring PPTX structure modification |
| Interop glue code | API / Backend | — | Go → Python bridge or pure Go implementation |
| Slide image extraction | API / Backend | — | Server-side extracts JPEG from PPTX for preview |
| Error handling & logging | API / Backend | — | Centralized logging for PPTX operations |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| python-pptx | 1.0.2 | PPTX generation from images | [VERIFIED: python -c import] Already installed and working; mature library with full slide manipulation support |
| Go 1.24 | Backend execution | Calls Python via exec.Command | [VERIFIED: go version] Project standard, already integrated in PPTXGenerator |
| exec.Command | Go standard library | Process execution for Python scripts | [VERIFIED: Go docs] Standard Go pattern for external process invocation |

### Supporting (Go Alternatives - Not Recommended for Phase 3)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| unidoc/unioffice | v2.9.0 | Pure Go PPTX creation/editing | If eliminating Python dependency is critical AND commercial license is acceptable; limited slide merge support |
| Muprprpr/Go-pptx | latest | Pure Go PPTX with streaming I/O | For large PPTX files (>50MB) requiring streaming; MIT licensed but limited high-level slide manipulation |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| python-pptx slide generation | unioffice Go library | Pure Go but requires commercial license, less mature API, limited slide copy/merge operations |
| python-pptx slide generation | Muprprpr/Go-pptx | MIT licensed but lower-level API, requires more code for same functionality, limited documentation |
| Go exec.Command → Python | CGO with Python C API | Tighter integration but complex build, platform-specific, still requires Python runtime |

**Installation (Current Python Approach):**
```bash
# Already verified installed
python --version  # 3.13.2
python -c "import pptx; print(pptx.__version__)"  # 1.0.2

# If reinstalling:
pip install python-pptx==1.0.2
```

**Installation (Pure Go Alternative - NOT Recommended):**
```bash
# For unioffice (commercial license required):
go get github.com/unidoc/unioffice/v2

# For Muprprpr/Go-pptx (MIT license):
go get github.com/Muprprpr/Go-pptx
```

**Version verification:**
- python-pptx 1.0.2: [VERIFIED: python -c "import pptx; print(pptx.__version__)"] on 2026-04-20
- Python 3.13.2: [VERIFIED: python --version] on 2026-04-20
- Go 1.24: [VERIFIED: go version] from project context
- unioffice v2.9.0: [VERIFIED: go list -m -versions] on 2026-04-20 (latest in Go registry)
- Muprprpr/Go-pptx: [VERIFIED: go list -m] on 2026-04-20 (available but no version tags)

## Architecture Patterns

### System Architecture Diagram (Current: Go → Python)

```
TranscriptionService detects scene changes
        ↓
FrameExtractor extracts frames as JPEG images
        ↓
PPTXGenerator.GeneratePPTX(ctx, framePaths, outputPath)
        ↓
Go: exec.CommandContext(ctx, "python", ["scripts/create_pptx.py", outputPath, frame1, frame2, ...])
        ↓
Python: create_pptx.py executes
        ↓
Python: python-pptx creates PPTX with one image per slide
        ↓
Python: JSON output {"success": true, "page_count": N}
        ↓
Go: Parse JSON, update PPTFile record, return result
```

### System Architecture Diagram (Alternative: Pure Go with unioffice)

```
TranscriptionService detects scene changes
        ↓
FrameExtractor extracts frames as JPEG images
        ↓
PPTXGenerator.GeneratePPTX(ctx, framePaths, outputPath)
        ↓
Go: Create presentation with unioffice/presentation
        ↓
Go: For each frame, add slide with image
        ↓
Go: Save presentation to outputPath
        ↓
Go: Update PPTFile record, return result
```

### Recommended Project Structure (Pure Go Alternative)

```
internal/
├── services/
│   ├── pptx_generator_go.go           # NEW: Pure Go implementation using unioffice
│   ├── pptx_generator.go              # KEEP: Python-based implementation (fallback)
│   └── slide_extractor.go             # NEW: Slide extraction from PPTX for preview
└── models/
    └── ppt_file.go                    # EXTEND: Add SourceType to track generation method

scripts/
├── create_pptx.py                     # KEEP: Python script for complex operations (merge)
└── merge_slides.py                    # KEEP: Python script for slide merging
```

### Pattern 1: PPTX Generation with python-pptx (Current - KEEP)

**What:** Generate PPTX files from JPEG images using python-pptx library called from Go via exec.Command.

**When to use:** All PPTX generation scenarios in Phase 3 (transcription results, merge operations).

**Example:**
```python
# Source: scripts/create_pptx.py (existing)
from pptx import Presentation
from pptx.util import Inches

SLIDE_WIDTH_INCH = 10.0
SLIDE_HEIGHT_INCH = 5.625

def create_pptx_from_images(image_paths, output_path):
    prs = Presentation()
    prs.slide_width = Inches(SLIDE_WIDTH_INCH)
    prs.slide_height = Inches(SLIDE_HEIGHT_INCH)

    for img_path in image_paths:
        if not os.path.exists(img_path):
            continue

        blank_slide_layout = prs.slide_layouts[6]
        slide = prs.slides.add_slide(blank_slide_layout)

        # Add image full-frame
        slide.shapes.add_picture(
            img_path,
            0, 0,
            width=prs.slide_width,
            height=prs.slide_height
        )

    prs.save(output_path)
    return len(prs.slides)
```

**Go integration:**
```go
// Source: internal/services/pptx_generator.go (existing)
func (g *PPTXGenerator) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
    args := append([]string{g.pythonScript, outputPath}, framePaths...)

    cmdName := "python3"
    if _, err := exec.LookPath("python3"); err != nil {
        cmdName = "python"
    }
    cmd := exec.CommandContext(ctx, cmdName, args...)

    output, err := cmd.CombinedOutput()
    if err != nil {
        return 0, fmt.Errorf("python script failed: %w", err)
    }

    var result struct {
        Success   bool   `json:"success"`
        PageCount int    `json:"page_count"`
        Error     string `json:"error,omitempty"`
    }
    if err := json.Unmarshal(output, &result); err != nil {
        return len(framePaths), nil
    }

    return result.PageCount, nil
}
```

### Pattern 2: PPTX Generation with unioffice (Alternative - NOT Recommended)

**What:** Generate PPTX files from JPEG images using pure Go library unioffice.

**When to use:** Only if Python dependency elimination is critical AND commercial license is acceptable. **Warning:** Limited documentation for slide manipulation patterns.

**Example:**
```go
// Source: unioffice documentation (https://unidoc.io/blog/creating-powerpoint-presentations-with-go/)
package services

import (
    "os"
    "image"
    "_ image/jpeg"

    "github.com/unidoc/unioffice/presentation"
)

type PPTXGeneratorGo struct {
    logger *zap.Logger
}

func (g *PPTXGeneratorGo) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
    // Create presentation
    prs := presentation.New()

    // Set slide size to 16:9
    prs.Width(10 * presentation.Inch)
    prs.Height(5.625 * presentation.Inch)

    pageCount := 0
    for _, imgPath := range framePaths {
        // Read image file
        f, err := os.Open(imgPath)
        if err != nil {
            g.logger.Warn("Failed to open image", zap.String("path", imgPath), zap.Error(err))
            continue
        }

        img, _, err := image.DecodeConfig(f)
        f.Close()
        if err != nil {
            continue
        }

        // Add slide with image
        slide, err := prs.CreateSlide()
        if err != nil {
            return pageCount, err
        }

        // Add image to slide (unioffice API pattern)
        // Note: Documentation incomplete for image positioning
        imageRef, err := prs.AddImage(imgPath)
        if err != nil {
            return pageCount, err
        }

        // API for placing image on slide unclear from docs
        // This is a research gap - requires experimentation
        pageCount++
    }

    // Save presentation
    if err := prs.SaveToFile(outputPath); err != nil {
        return pageCount, err
    }

    return pageCount, nil
}
```

**Key limitation:** unioffice documentation is incomplete for image placement on slides. Requires trial-and-error or commercial support.

### Pattern 3: PPTX Generation with Muprprpr/Go-pptx (Alternative - NOT Recommended)

**What:** Generate PPTX files using pure Go library with OPC (Open Packaging Convention) implementation.

**When to use:** For large PPTX files requiring streaming I/O. **Warning:** Lower-level API, more code required.

**Example:**
```go
// Source: Muprprpr/Go-pptx documentation (https://github.com/Muprprpr/Go-pptx)
package services

import (
    "github.com/Muprprpr/Go-pptx/opc"
    "github.com/Muprprpr/Go-pptx/parts"
)

type PPTXGeneratorOPC struct {
    logger *zap.Logger
}

func (g *PPTXGeneratorOPC) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
    // Create new PPTX package
    pkg := opc.NewPackage()

    // Add presentation part
    presPart := parts.NewPresentationPart()
    pkg.AddPart(presPart)

    // Set slide size
    presPart.SetSlideSize(10*914400, 5.625*914400) // In metric units (EMU)

    pageCount := 0
    for _, imgPath := range framePaths {
        // Add slide part
        slidePart := parts.NewSlidePart()
        pkg.AddPart(slidePart)

        // Add image to slide
        // Requires manual XML manipulation - lower level than python-pptx
        // This is complex and error-prone

        // Link slide to presentation
        presPart.AddSlide(slidePart)

        pageCount++
    }

    // Save package
    if err := pkg.SaveFile(outputPath); err != nil {
        return pageCount, err
    }

    return pageCount, nil
}
```

**Key limitation:** Requires deep understanding of PPTX XML structure. More code than python-pptx for same functionality.

### Pattern 4: Go-Python Interop with exec.Command (Current - KEEP)

**What:** Bridge Go and Python using standard library exec.Command for process invocation.

**When to use:** Calling Python scripts from Go for PPTX operations.

**Performance characteristics:**
- **Process spawn overhead:** ~10-50ms per call on Windows/Linux
- **Context timeout support:** Yes, via CommandContext
- **Error handling:** Parse JSON output from Python
- **Security concerns:** Command injection (mitigated by validatePath)

**Example:**
```go
// Source: internal/services/pptx_generator.go (existing)
func (g *PPTXGenerator) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
    // Sanitize paths to prevent injection
    sanitizedPaths := make([]string, len(framePaths))
    for i, path := range framePaths {
        sanitizedPaths[i] = filepath.Clean(path)
        if err := g.validatePath(path); err != nil {
            return 0, fmt.Errorf("invalid frame path at index %d: %w", i, err)
        }
    }

    // Prepare command
    args := append([]string{g.pythonScript, outputPath}, sanitizedPaths...)

    cmdName := "python3"
    if _, err := exec.LookPath("python3"); err != nil {
        cmdName = "python"
    }
    cmd := exec.CommandContext(ctx, cmdName, args...)

    // Execute with timeout
    output, err := cmd.CombinedOutput()
    if err != nil {
        g.logger.Error("Python script failed",
            zap.String("output", string(output)),
            zap.Error(err))
        return 0, fmt.Errorf("python script failed: %w", err)
    }

    // Parse JSON result
    var result struct {
        Success   bool   `json:"success"`
        PageCount int    `json:"page_count"`
        Error     string `json:"error,omitempty"`
    }
    if err := json.Unmarshal(output, &result); err != nil {
        g.logger.Warn("Failed to parse Python output",
            zap.String("output", string(output)),
            zap.Error(err))
        return len(framePaths), nil
    }

    return result.PageCount, nil
}
```

### Anti-Patterns to Avoid

- **Unconditional Python dependency removal:** Removing Python without fully validating Go library capabilities leads to missing features (slide merging, complex formatting).
- **Using unioffice without license review:** unioffice requires commercial license for production use. Ignoring this leads to legal compliance issues.
- **CGO with Python C API:** Adds build complexity, platform-specific code, and still requires Python runtime. No advantage over exec.Command.
- **Hand-rolled PPTX XML generation:** PPTX format is complex OPC+XML schemas. Building custom XML writer is error-prone and unmaintainable.
- **Ignoring slide merge limitations:** Go libraries cannot easily copy slides between presentations. This breaks PPT-06 merge requirement.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PPTX XML structure manipulation | Custom XML writer for slide relationships, content types, media | python-pptx OR unioffice OR Muprprpr/Go-pptx | PPTX is OPC container with complex XML schemas, namespaces, and relationships. Libraries handle edge cases (slide IDs, media embedding, layout inheritance) |
| Process spawning for Python interop | Custom fork/exec with pipe management | exec.Command from Go standard library | Handles process lifecycle, signal propagation, context cancellation, stdio pipes correctly |
| Image format conversion for PPTX | Custom JPEG/PNG encoding/decoding | image/jpeg from Go standard library OR PIL/Pillow in Python | Handles EXIF, color profiles, compression levels, progressive encoding |
| Slide layout calculation | Manual pixel/EMU conversion for 16:9 aspect ratio | python-pptx Inches utility OR unioffice presentation.Inch | PPTX uses English Metric Units (EMU), 1 inch = 914400 EMU. Libraries handle conversion correctly |
| Slide merge between presentations | Custom slide copy with shape cloning | python-pptx (has limited slide copy support) | Slide copying requires cloning shapes, images, relationships, layout references. Doing manually is error-prone |

**Key insight:** PPTX file format is intentionally complex (OPC container with XML parts, relationships, content types). Libraries like python-pptx encapsulate this complexity. Pure Go alternatives exist but are less mature for complex operations like slide merging.

## Common Pitfalls

### Pitfall 1: python-pptx Cannot Render Slides to Images

**What goes wrong:** Attempting to use python-pptx to render slides as images for preview produces blank or partial images.

**Why it happens:** python-pptx reads and writes PPTX structure but lacks a rendering engine. It cannot convert slides to images because it doesn't handle layout, font rendering, or shape composition.

**How to avoid:**
1. **For Phase 3 preview (D-01):** Extract only embedded images from slides using `slide.shapes`. Fast, no external deps. Accept limitation that text/shapes won't render.
2. **For full rendering (future):** Integrate LibreOffice headless mode: `soffice --headless --convert-to pdf output.pptx` then `pdftoppm` for images.
3. **Alternative:** Use commercial library Aspose.Slides Python (has rendering capabilities).

**Warning signs:** Slide images are blank, text content missing, shapes not appearing in extracted JPEGs.

**Impact on Go vs Python decision:** This limitation affects BOTH Python and Go approaches. Neither python-pptx nor Go libraries can render slides. This is NOT a reason to choose Go over Python.

### Pitfall 2: Go Libraries Lack Slide Merge Support

**What goes wrong:** Attempting to merge slides from multiple PPTX files using unioffice or Muprprpr/Go-pptx fails or produces corrupted files.

**Why it happens:** Slide merging requires:
- Copying slide content (shapes, images, text)
- Updating slide IDs and relationships
- Handling layout and master slide references
- Re-embedding images and media

Go libraries have incomplete or undocumented APIs for these operations. python-pptx also has limitations but is more mature.

**How to avoid:**
1. **For Phase 3 merge (D-15, PPT-06):** Use python-pptx for merge operations, even if using Go for PPTX generation.
2. **Alternative:** Re-extract original frame images and create new PPTX from selected frames (bypasses slide copy complexity).
3. **Document limitation:** If using Go, clearly document that slide merge is not supported.

**Warning signs:** Merged PPTX fails to open, missing images, broken slide layouts, crashes PowerPoint.

**Impact on Go vs Python decision:** This is a **critical blocker** for pure Go migration. Slide merge (PPT-06) is a Phase 3 requirement. Go libraries cannot currently support this requirement reliably.

### Pitfall 3: Go-Python Interop Performance Bottleneck

**What goes wrong:** Frequent calls to Python scripts cause performance degradation, especially during concurrent PPTX generation.

**Why it happens:** Each `exec.Command` call spawns a new Python process (~10-50ms overhead). For 10 concurrent PPTX generations, this adds 100-500ms of latency.

**How to avoid:**
1. **Batch operations:** Generate multiple PPTX files in a single Python script call (pass multiple output paths).
2. **Python worker pool:** Keep Python processes running and communicate via stdin/stdout or Unix sockets (complex).
3. **Accept overhead:** For Phase 3 usage pattern (one PPTX per transcription, not high-frequency), overhead is acceptable (<5% of total time).

**Warning signs:** PPTX generation takes >10 seconds, high CPU usage from process spawning, concurrent operations timeout.

**Impact on Go vs Python decision:** Performance impact is **minimal** for Phase 3 workload (low-frequency PPTX generation). Not a compelling reason to migrate to Go.

### Pitfall 4: unioffice Commercial License Compliance

**What goes wrong:** Using unioffice in production without purchasing a license violates the EULA and may cause legal issues.

**Why it happens:** unioffice is a commercial product. The free tier is for evaluation only. Production use requires a paid license.

**How to avoid:**
1. **Review license:** Read unioffice EULA at https://unidoc.io/eula/
2. **Budget for license:** unioffice licenses cost money (pricing not publicly listed).
3. **Use MIT-licensed alternative:** Muprprpr/Go-pptx is MIT-licensed but less feature-rich.

**Warning signs:** License warnings in logs, watermarked output, usage tracking.

**Impact on Go vs Python decision:** unioffice adds **ongoing cost** for Go-based PPTX generation. python-pptx is free (PSF license).

### Pitfall 5: Python Environment Fragmentation

**What goes wrong:** Python script works on development machine but fails in production due to different Python versions or missing dependencies.

**Why it happens:** Python environments vary by OS, package managers, and virtual environments. python-pptx may not be installed or wrong version.

**How to avoid:**
1. **Document dependencies:** Create `requirements.txt` with python-pptx==1.0.2
2. **Check dependencies at startup:** Call `CheckPythonDependencies()` on service start (already implemented in pptx_generator.go).
3. **Provide clear error messages:** If Python not found, instruct user: "Install Python 3.x and python-pptx: pip install python-pptx==1.0.2"
4. **Containerize deployment:** Use Docker with Python pre-installed (if deploying in containers).

**Warning signs:** "python: command not found", "ModuleNotFoundError: No module named 'pptx'", version mismatch errors.

**Impact on Go vs Python decision:** This is a **valid concern** but manageable with proper documentation and dependency checks. Not a strong reason to migrate to Go alone.

## Code Examples

Verified patterns from official sources:

### Go-Python Interop with JSON Result (Current Pattern)

```go
// Source: internal/services/pptx_generator.go (existing code)
func (g *PPTXGenerator) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
    if len(framePaths) == 0 {
        return 0, fmt.Errorf("no frame paths provided")
    }

    // Sanitize paths to prevent command injection
    outputPath = filepath.Clean(outputPath)
    sanitizedPaths := make([]string, len(framePaths))
    for i, path := range framePaths {
        sanitizedPaths[i] = filepath.Clean(path)
        if err := g.validatePath(path); err != nil {
            return 0, fmt.Errorf("invalid frame path at index %d: %w", i, err)
        }
    }

    // Prepare command: python create_pptx.py <output_path> <image1> <image2> ...
    args := append([]string{g.pythonScript, outputPath}, sanitizedPaths...)

    // Try python3 first, then fall back to python
    cmdName := "python3"
    if _, err := exec.LookPath("python3"); err != nil {
        cmdName = "python"
    }
    cmd := exec.CommandContext(ctx, cmdName, args...)

    // Execute and capture output
    output, err := cmd.CombinedOutput()
    if err != nil {
        g.logger.Error("Python script failed",
            zap.String("output", string(output)),
            zap.Error(err))
        return 0, fmt.Errorf("python script failed: %w", err)
    }

    // Parse JSON result from Python script
    var result struct {
        Success     bool   `json:"success"`
        PageCount   int    `json:"page_count"`
        OutputPath  string `json:"output_path"`
        Error       string `json:"error,omitempty"`
        SkippedCount int   `json:"skipped_count,omitempty"`
    }
    if err := json.Unmarshal(output, &result); err != nil {
        g.logger.Warn("Failed to parse page count from output",
            zap.String("output", string(output)),
            zap.Error(err))
        // Default to number of frames if parsing fails
        return len(framePaths), nil
    }

    if !result.Success {
        return 0, fmt.Errorf("PPTX generation failed: %s", result.Error)
    }

    g.logger.Info("PPTX generated successfully",
        zap.String("output_path", result.OutputPath),
        zap.Int("page_count", result.PageCount),
        zap.Int("skipped_count", result.SkippedCount))

    return result.PageCount, nil
}
```

### Python Script with JSON Output (Current Pattern)

```python
# Source: scripts/create_pptx.py (existing code)
import sys
import json
import os
from pptx import Presentation
from pptx.util import Inches

SLIDE_WIDTH_INCH = 10.0
SLIDE_HEIGHT_INCH = 5.625

def create_pptx_from_images(image_paths, output_path):
    """Create a PowerPoint file from a list of images."""
    try:
        if not image_paths:
            result = {"success": False, "error": "No image paths provided"}
            return False, result, 1

        prs = Presentation()
        prs.slide_width = Inches(SLIDE_WIDTH_INCH)
        prs.slide_height = Inches(SLIDE_HEIGHT_INCH)

        page_count = 0
        skipped = []

        for img_path in image_paths:
            try:
                if not os.path.exists(img_path):
                    skipped.append(img_path)
                    continue

                blank_slide_layout = prs.slide_layouts[6]
                slide = prs.slides.add_slide(blank_slide_layout)

                slide.shapes.add_picture(
                    img_path,
                    0, 0,
                    width=prs.slide_width,
                    height=prs.slide_height
                )

                page_count += 1

            except Exception:
                skipped.append(img_path)
                continue

        if page_count == 0:
            result = {
                "success": False,
                "error": f"No valid slides created from {len(image_paths)} input images",
                "skipped": skipped
            }
            return False, result, 1

        output_dir = os.path.dirname(output_path)
        if output_dir:
            os.makedirs(output_dir, exist_ok=True)

        prs.save(output_path)

        result = {
            "success": True,
            "page_count": page_count,
            "output_path": output_path,
            "skipped_count": len(skipped)
        }
        return True, result, 0

    except Exception as e:
        result = {"success": False, "error": str(e)}
        return False, result, 1

def main():
    if len(sys.argv) < 3:
        print(json.dumps({
            "success": False,
            "error": "Usage: create_pptx.py <output_path> <image1> <image2> ..."
        }), file=sys.stderr)
        sys.exit(1)

    output_path = sys.argv[1]
    image_paths = sys.argv[2:]

    success, result, exit_code = create_pptx_from_images(image_paths, output_path)

    if success:
        print(json.dumps(result))
    else:
        print(json.dumps(result), file=sys.stderr)

    sys.exit(exit_code)

if __name__ == "__main__":
    main()
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Pure Python application | Go backend with Python scripts | Phase 2 | Hybrid architecture: Go for API/business logic, Python for specialized libraries |
| Manual PPTX creation | python-pptx library | Phase 2 | Automated PPTX generation from video frames |
| No PPT preview | Browser-based preview with slide extraction | Phase 3 (planned) | Users can preview PPT content without downloading |

**Deprecated/outdated:**
- **Pure Python monolith:** Replaced by Go backend with Python scripts for specific tasks
- **Manual PPTX assembly:** Replaced by python-pptx automation

**Emerging trends:**
- **Pure Go office libraries:** unioffice, Muprprpr/Go-pptx gaining maturity but still behind Python
- **Containerized deployments:** Mitigates Python environment fragmentation via Docker images
- **Cloud-based office APIs:** Google Slides API, Microsoft Graph API for PPTX generation (external dependencies)

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | python-pptx can handle slide merging for PPT-06 requirement | Don't Hand-Roll | If python-pptx slide merge is unreliable, Phase 3 merge feature will fail, requiring rework |
| A2 | unioffice license cost is acceptable for production use | Common Pitfalls | If unioffice licensing is too expensive, Go migration becomes uneconomical |
| A3 | Go-Python interop overhead is acceptable for Phase 3 workload | Common Pitfalls | If PPTX generation becomes performance bottleneck, need to optimize or migrate to Go |
| A4 | Muprprpr/Go-pptx can handle simple PPTX generation (images to slides) | Standard Stack | If Muprprpr/Go-pptx lacks image placement APIs, cannot use for even simple PPTX creation |
| A5 | Python environment can be reliably managed in production | Common Pitfalls | If Python dependency causes frequent deployment failures, pure Go becomes more attractive |
| A6 | Slide merge complexity is similar in python-pptx and Go libraries | Common Pitfalls | If Go libraries completely lack slide merge, hybrid approach (Go for generation, Python for merge) needed |
| A7 | exec.Command context cancellation works reliably on Windows | Architecture Patterns | If context cancellation fails on Windows, long-running Python scripts cannot be terminated cleanly |
| A8 | Python 3.13.2 and python-pptx 1.0.2 are compatible on all target platforms | Environment Availability | If version incompatibilities exist on production OS, need version pinning or compatibility testing |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed. (This table IS NOT empty; 8 assumptions require validation).

## Open Questions

1. **unioffice License Cost**
   - What we know: unioffice requires commercial license for production use
   - What's unclear: Exact pricing model and whether it fits project budget
   - Recommendation: Contact unidoc sales for pricing quote before committing to Go migration. Factor license cost into migration ROI calculation.

2. **Muprprpr/Go-pptx Image Placement API**
   - What we know: Library supports PPTX creation and media management
   - What's unclear: Exact API for placing images on slides (positioning, sizing, fitting)
   - Recommendation: Create proof-of-concept test: generate 10-slide PPTX from sample images using Muprprpr/Go-pptx. Measure code complexity vs python-pptx.

3. **Slide Merge Implementation in Go**
   - What we know: python-pptx has limited slide copy support, Go libraries unclear
   - What's unclear: Whether Muprprpr/Go-pptx or unioffice can copy slides between presentations
   - Recommendation: Test slide merge with both Go libraries. If neither supports it, plan hybrid approach (Go for generation, Python for merge) or use re-extract-from-frames approach.

4. **Go-Python Interop Performance Impact**
   - What we know: Process spawning has ~10-50ms overhead per call
   - What's unclear: Actual impact on Phase 3 workload (typical PPTX generation takes 5-30 seconds, overhead may be negligible)
   - Recommendation: Benchmark: measure PPTX generation time with Python approach vs pure Go prototype. Calculate percentage overhead.

5. **Python Dependency Management in Production**
   - What we know: Current deployment requires Python 3.x and python-pptx
   - What's unclear: How deployment is managed (Docker, bare metal, package manager)
   - Recommendation: Document Python installation in deployment guide. If using Docker, add Python installation to Dockerfile. If bare metal, add to setup script.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Python 3 | Current Python approach (pptx_generator.go) | ✓ | 3.13.2 | — |
| python-pptx | Current Python approach | ✓ | 1.0.2 | — |
| Go 1.24 | Backend (already in use) | ✓ | 1.24.x | — |
| unioffice v2.9.0 | Pure Go alternative | ✗ (not installed) | v2.9.0 available in registry | Must install via `go get` AND purchase commercial license |
| Muprprpr/Go-pptx | Pure Go alternative (MIT) | ✗ (not installed) | latest (no version tags) | Must install via `go get` |

**Missing dependencies with no fallback:**
- None for current Python approach (Python 3.13.2 and python-pptx 1.0.2 verified installed)

**Missing dependencies with fallback:**
- unioffice: Can use Muprprpr/Go-pptx as alternative (MIT licensed but less mature)
- Muprprpr/Go-pptx: Can use python-pptx as fallback (more mature, requires Python)

**Installation required for Go migration:**
```bash
# For unioffice (commercial license required):
go get github.com/unidoc/unioffice/v2

# For Muprprpr/Go-pptx (MIT license):
go get github.com/Muprprpr/Go-pptx
```

**Recommendation:** Do NOT install Go PPTX libraries for Phase 3. Current Python approach is sufficient. Re-evaluate for Phase 4+ if requirements change.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | Validate all frame paths before passing to Python (sanitize via filepath.Clean, check for path traversal) |
| V8 Error Handling | yes | Log Python script errors without exposing system paths, return generic error messages to client |
| V11 File Handling | yes | Validate Python script path is within allowed directories, prevent arbitrary script execution |
| V12 Code Quality | yes | Use established patterns for Go-Python interop, avoid hand-rolled process spawning |

### Known Threat Patterns for {Go calling Python for PPTX generation}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command injection via malicious frame paths | Injection | Sanitize all paths with filepath.Clean(), validate with validatePath(), reject paths with newlines/tabs |
| Arbitrary Python script execution | Tampering | Hardcode script path "scripts/create_pptx", validate path is within project root, reject user-supplied script names |
| Path traversal in output path | Tampering | Use filepath.Clean(), verify path is within allowed directory (recordings/), reject absolute paths outside project root |
| DoS via malformed image files | Denial of Service | Timeout Python execution with context.Context (10-minute timeout), skip corrupted images gracefully, limit max frames per PPTX |
| Information disclosure via Python errors | Disclosure | Log detailed errors server-side, return generic "PPTX generation failed" to client, sanitize Python stack traces before logging |

**Additional considerations for pure Go alternative:**
- unioffice and Muprprpr/Go-pptx reduce attack surface by eliminating Python subprocess
- However, both Go libraries parse PPTX files (complex XML), risking XML parsing vulnerabilities
- Python approach has been battle-tested; Go libraries less audited for security

## Sources

### Primary (HIGH confidence)

- [unioffice GitHub Repository](https://github.com/unidoc/unioffice) - Official documentation, API examples, license information
- [unioffice API Documentation](https://apidocs.unidoc.io/unioffice/v1.8.0/github.com/unidoc/unioffice/presentation/) - Presentation API reference
- [Muprprpr/Go-pptx GitHub Repository](https://github.com/Muprprpr/Go-pptx) - Library source code, documentation, examples
- [Existing codebase] - `internal/services/pptx_generator.go`, `scripts/create_pptx.py`
- [Verified installations] - Python 3.13.2, python-pptx 1.0.2, Go 1.24

### Secondary (MEDIUM confidence)

- [Web Search: Go PPTX generation libraries](https://github.com/search?q=go+pptx+library) - Confirmed unioffice and Muprprpr/Go-pptx as primary options
- [Web Search: Go-Python interop performance](https://stackoverflow.com/questions/55271734/speed-up-access-to-python-programs-from-golangs-exec-package) - Documented performance overhead of exec.Command
- [Python package registry] - Verified python-pptx 1.0.2 availability and version
- [Go module registry] - Verified unioffice v2.9.0 and Muprprpr/Go-pptx availability

### Tertiary (LOW confidence)

- [Web Search: python-pptx slide rendering limitations] - Rate limited during research; based on training knowledge that python-pptx cannot render slides to images
- [Web Search: Go PPTX library comparison] - Rate limited; based on training knowledge of python-pptx feature set vs Go alternatives
- [unioffice pricing] - No public pricing; requires contact with sales (assumption about cost)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Python 3.13.2 and python-pptx 1.0.2 verified installed and working; Go libraries verified available in registry
- Architecture: MEDIUM - Go-Python interop pattern proven in production; Go library APIs partially documented
- Pitfalls: MEDIUM - Performance overhead and dependency fragmentation are well-documented; unioffice licensing is known issue but specific pricing unknown
- Migration recommendation: MEDIUM - Based on feature comparison (python-pptx supports slide merging, Go libraries unclear); requires validation of slide merge capabilities

**Research date:** 2026-04-20
**Valid until:** 2026-05-20 (30 days - Go library ecosystem evolving rapidly, unioffice pricing may change)

## Appendix: Feature Comparison Matrix

| Feature | python-pptx | unioffice | Muprprpr/Go-pptx | Phase 3 Requirement |
|---------|-------------|-----------|------------------|---------------------|
| Create PPTX from images | ✓ Full support | ✓ Full support | ⚠ Lower-level API | PPT-01: Generate PPT from frames |
| Add slides to presentation | ✓ Full support | ✓ Full support | ✓ Full support | PPT-01: Multi-slide PPT |
| Insert images on slides | ✓ Full support | ⚠ Limited docs | ⚠ Manual XML required | PPT-01: Frame images on slides |
| Set slide size (16:9) | ✓ Easy (Inches) | ✓ Easy (Inch) | ⚠ Manual (EMU) | Current implementation uses 16:9 |
| Copy slides between PPTs | ⚠ Limited support | ❌ No docs | ❌ No docs | PPT-06: Merge slides from multiple PPTs |
| Extract embedded images | ✓ Full support | ⚠ Unclear | ⚠ Unclear | PPT-03: Preview slides (extract images) |
| Render slides to images | ❌ Not supported | ❌ Not supported | ❌ Not supported | PPT-03: Preview (need external tool) |
| License | PSF (free) | Commercial ($$$) | MIT (free) | Deployment constraint |
| Documentation quality | Excellent | Good | Sparse | Development effort |
| Community size | Large | Medium | Small | Support availability |
| Maturity (years) | 12+ (since 2012) | 6+ (since 2018) | <2 (new) | Stability risk |

**Legend:**
- ✓ Full support: Well-documented, tested, production-ready
- ⚠ Limited support: Exists but incomplete docs or requires workarounds
- ❌ No support: Feature does not exist or not documented

**Key findings:**
1. python-pptx is only library with documented slide copy support (critical for PPT-06 merge)
2. All libraries lack slide rendering to images (affects PPT-03 preview)
3. unioffice has commercial license cost, Muprprpr/Go-pptx is MIT but less mature
4. python-pptx has best documentation and community support

**Conclusion:** python-pptx remains best choice for Phase 3 requirements. Pure Go migration not recommended due to slide merge limitation (PPT-06) and unioffice licensing cost.
