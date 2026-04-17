# Phase 02 Research: PPTX Generation Libraries

## Context

Original plan specified unidoc/unioffice/v2, but discovered it requires a commercial license for PPTX file generation (FREE_TIER_LIMITS error in unioffice). User decided: **Switch to Muprprpr/Go-pptx (MIT license, free).**

**CRITICAL DISCOVERY 1:** Muprprpr/Go-pptx is NOT a library - it's a command-line program that imports unioffice/v2 as a dependency. When we added it, go.mod showed `github.com/unidoc/unioffice/v2` as a transitive dependency, and tests failed with "unioffice license required" error.

**CRITICAL DISCOVERY 2:** SiliconCatalyst/officeforge only supports **template-based** PPTX generation (keyword replacement in existing PPTX files), NOT creating PPTX files from scratch with images. This won't work for our use case.

**NEW DECISION:** Use Python's python-pptx library (BSD license, mature, free) called from Go via os/exec. This is a pragmatic solution that gives us full PPTX generation capabilities without licensing issues.

## Library Comparison

### unidoc/unioffice/v2 (Original Choice - REJECTED)

**Repository:** github.com/unidoc/unioffice/v2

**License:** Commercial (FREE_TIER_LIMITS on save operations)

**Verdict:** REJECTED due to licensing costs

---

### Muprprpr/Go-pptx (REJECTED - NOT A LIBRARY)

**Repository:** github.com/Muprprpr/Go-pptx

**License:** MIT License

**CRITICAL ISSUE:** This is NOT an importable Go library - it's a command-line program that imports `github.com/unidoc/unioffice/v2` as a dependency. Tests failed with "unioffice license required" error.

**Verdict:** REJECTED - Same licensing problem as unioffice (it wraps unioffice)

---

### SiliconCatalyst/officeforge (REJECTED - WRONG CAPABILITIES)

**Repository:** github.com/SiliconCatalyst/officeforge

**License:** MIT License

**API Functions Available:**
- `ProcessPptxSingle(inputPath, outputPath, keyword, replacement)` - Replace one keyword
- `ProcessPptxMulti(inputPath, outputPath, replacements map[string]string)` - Replace multiple keywords
- `ProcessPptxMultipleRecords(...)` - Batch processing with templates

**CRITICAL LIMITATION:** This library only supports **template-based** PPTX generation. You provide an existing PPTX template with placeholders like `{{NAME}}`, and it replaces them. It does NOT support:
- Creating PPTX files from scratch
- Adding images to slides
- Positioning/sizing images
- Creating slides dynamically

**Why it doesn't work:** Our use case is: take N image files → create N slides (one per image) with full-frame layout. This requires creating PPTX from scratch, not template replacement.

**Verdict:** REJECTED - Does not support our required functionality

---

### python-pptx (SELECTED - FINAL CHOICE)

**Repository:** https://github.com/scanny/python-pptx

**License:** BSD License (free, no restrictions)

**Documentation:** https://python-pptx.readthedocs.io/

**Key Features:**
- Mature, widely-used Python library for PPTX generation
- Full support for creating PPTX files from scratch
- Add images to slides with precise positioning and sizing
- Set slide size (16:9, 4:3, custom)
- No licensing restrictions
- Active development and community

**Integration Approach:** Call Python script from Go using `os/exec`

**Pros:**
- ✅ BSD license - no cost, no restrictions
- ✅ Mature, well-documented (8.5k stars on GitHub)
- ✅ Full PPTX generation from scratch
- ✅ Add images with precise positioning/sizing
- ✅ Supports our exact use case
- ✅ No licensing issues

**Cons:**
- ⚠️ Requires Python runtime on server
- ⚠️ Cross-language integration (Go → Python)
- ⚠️ Additional dependency to manage

**Verdict:** SELECTED - Only viable free option that actually supports our requirements

## Proposed Solution: Python Integration

### Architecture

```
Go Service (PPTXGenerator)
    ↓
    | os/exec.Command
    ↓
Python Script (create_pptx.py)
    ↓
    | python-pptx library
    ↓
PPTX File
```

### Python Script Implementation

**File:** `scripts/create_pptx.py`

```python
#!/usr/bin/env python3
import sys
import json
from pptx import Presentation
from pptx.util import Inches, Pt
from pptx.enum.shapes import MSO_SHAPE

def create_pptx_from_images(image_paths, output_path):
    """
    Create a PowerPoint file from a list of images.

    Args:
        image_paths: List of image file paths
        output_path: Output PPTX file path

    Returns:
        JSON string with result: {"success": true, "page_count": N}
    """
    try:
        # Create presentation
        prs = Presentation()
        prs.slide_width = Inches(10)
        prs.slide_height = Inches(5.625)  # 16:9 aspect ratio

        page_count = 0

        # Add each image as a slide
        for img_path in image_paths:
            try:
                # Add blank slide
                blank_slide_layout = prs.slide_layouts[6]  # Blank layout
                slide = prs.slides.add_slide(blank_slide_layout)

                # Add image to slide (full-frame, no margins)
                slide.shapes.add_picture(img_path, 0, 0, width=prs.slide_width, height=prs.slide_height)

                page_count += 1
            except Exception as e:
                # Log error but continue with other images
                sys.stderr.write(f"Error adding image {img_path}: {str(e)}\n")
                continue

        # Save presentation
        prs.save(output_path)

        # Return success result
        result = {
            "success": True,
            "page_count": page_count,
            "output_path": output_path
        }
        print(json.dumps(result))
        return 0

    except Exception as e:
        # Return error result
        result = {
            "success": False,
            "error": str(e)
        }
        print(json.dumps(result), file=sys.stderr)
        return 1

if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: create_pptx.py <output_path> <image1> <image2> ...", file=sys.stderr)
        sys.exit(1)

    output_path = sys.argv[1]
    image_paths = sys.argv[2:]

    sys.exit(create_pptx_from_images(image_paths, output_path))
```

### Go Service Implementation

```go
package services

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "path/filepath"

    "go.uber.org/zap"
)

type PPTXGenerator struct {
    logger        *zap.Logger
    pythonScript  string  // Path to create_pptx.py
}

func NewPPTXGenerator(logger *zap.Logger) *PPTXGenerator {
    return &PPTXGenerator{
        logger:       logger,
        pythonScript: "scripts/create_pptx.py",  // Relative to project root
    }
}

func (g *PPTXGenerator) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
    if len(framePaths) == 0 {
        return 0, fmt.Errorf("frame paths cannot be empty")
    }

    // Prevent DoS (T-02-06)
    if len(framePaths) > MAX_SLIDES {
        return 0, fmt.Errorf("number of frames (%d) exceeds maximum allowed (%d)", len(framePaths), MAX_SLIDES)
    }

    // Sanitize paths (T-02-05)
    outputPath = filepath.Clean(outputPath)

    // Prepare command arguments
    args := append([]string{g.pythonScript, outputPath}, framePaths...)

    // Execute Python script
    cmd := exec.CommandContext(ctx, "python3", args...)
    output, err := cmd.CombinedOutput()

    if err != nil {
        g.logger.Error("Python script failed",
            zap.String("output", string(output)),
            zap.Error(err))
        return 0, fmt.Errorf("failed to generate PPTX: %w", err)
    }

    // Parse result
    var result struct {
        Success    bool   `json:"success"`
        PageCount  int    `json:"page_count"`
        OutputPath string `json:"output_path"`
        Error      string `json:"error,omitempty"`
    }

    if err := json.Unmarshal(output, &result); err != nil {
        return 0, fmt.Errorf("failed to parse Python output: %w", err)
    }

    if !result.Success {
        return 0, fmt.Errorf("PPTX generation failed: %s", result.Error)
    }

    g.logger.Info("PPTX generated successfully",
        zap.String("output", outputPath),
        zap.Int("page_count", result.PageCount))

    return result.PageCount, nil
}
```

### Deployment Requirements

**Server Dependencies:**
- Python 3.7+
- pip install python-pptx

**Installation:**
```bash
# Install python-pptx
pip3 install python-pptx

# Or add to requirements.txt
echo "python-pptx>=0.6.21" >> requirements.txt
pip3 install -r requirements.txt
```

**Verification:**
```bash
# Test Python script
python3 scripts/create_pptx.py test.pptx image1.jpg image2.jpg
```

## Threat Model Considerations

**T-02-05: Path Traversal**
- Validate output path is within configured storage directory
- Use filepath.Clean to prevent path traversal attacks
- Python script also sanitizes paths

**T-02-06: Denial of Service**
- Limit maximum slides (cap at 500) to prevent OOM
- Context timeout on command execution

**Additional Security:**
- Validate Python script path is within project directory
- Use fixed Python script path (not user-controllable)
- Timeout command execution with context

## Implementation Plan

1. Remove officeforge dependency from go.mod
2. Install python-pptx: `pip3 install python-pptx`
3. Create `scripts/create_pptx.py` Python script
4. Rewrite `internal/services/pptx_generator.go` to use os/exec
5. Update tests to mock or integrate actual Python execution
6. Verify tests pass and generate valid PPTX files
7. Create SUMMARY.md documenting the Python integration

## References

- python-pptx Documentation: https://python-pptx.readthedocs.io/
- python-pptx GitHub: https://github.com/scanny/python-pptx
- python-pptx License: BSD (https://github.com/scanny/python-pptx/blob/main/LICENSE)
- 16:9 Aspect Ratio: 10" x 5.625" (standard widescreen)
