# Phase 02 Research: PPTX Generation Libraries

## Context

Original plan specified unidoc/unioffice/v2, but discovered it requires a commercial license for PPTX file generation (FREE_TIER_LIMITS error in unioffice). User decided: **Switch to Muprprpr/Go-pptx (MIT license, free).**

## Library Comparison

### unidoc/unioffice/v2 (Original Choice - REJECTED)

**Repository:** github.com/unidoc/unioffice/v2

**License:** Commercial (FREE_TIER_LIMITS on save operations)

**Pros:**
- Mature, well-documented
- Active community
- Comprehensive Office suite support

**Cons:**
- **Commercial license required for PPTX generation** (dealbreaker)
- Free tier has limitations

**Verdict:** REJECTED due to licensing costs

---

### Muprprpr/Go-pptx (Selected)

**Repository:** github.com/Muprprpr/Go-pptx

**License:** MIT License (free, no restrictions)

**Documentation:** https://github.com/Muprprpr/Go-pptx

**API Overview:**
```go
import "github.com/Muprprpr/Go-pptx"

// Create new presentation
ppt := pptx.NewPresentation()

// Add slide
slide := ppt.AddSlide()

// Add image to slide
imgPath := "path/to/image.jpg"
imgRef, err := ppt.AddImage(imgPath)
if err != nil {
    return err
}

// Position and size image on slide
// Go-pptx usesEMU (English Metric Units) for positioning
// 1 inch = 914400 EMU
// 16:9 slide size: 12192000 x 6858000 EMU (10" x 5.625")

// Image positioning is done during AddImage call
// or through properties on the returned image reference
```

**Pros:**
- ✅ MIT license - no cost, no restrictions
- ✅ Active development
- ✅ Simple API
- ✅ Supports image addition to slides
- ✅ Can set slide size and image positioning

**Cons:**
- Less mature than unioffice
- Smaller community
- Less comprehensive documentation
- API may differ from unioffice patterns

**Verdict:** SELECTED - Licensing makes it the only viable option

## Pattern 3: Go-pptx Usage for Full-Frame 16:9 Image Slides

### Key API Concepts

**1. EMU (English Metric Units)**
- Go-pptx uses EMU for all positioning and sizing
- Conversion: 1 inch = 914400 EMU
- 16:9 slide dimensions: 10" x 5.625" = 9,144,000 x 5,143,250 EMU

**2. Presentation Creation**
```go
import "github.com/Muprprpr/Go-pptx"

ppt := pptx.NewPresentation()
// Optionally set slide size
ppt.SetSlideSize(10*914400, 5.625*914400) // 16:9 in EMU
```

**3. Adding Images to Slides**
```go
// Add image to presentation (returns image reference)
imgRef, err := ppt.AddImage("path/to/image.jpg")
if err != nil {
    return fmt.Errorf("failed to add image: %w", err)
}

// Add slide
slide := ppt.AddSlide()

// Add image to slide with positioning
// Go-pptx API: slide.AddImage(imgRef, x, y, width, height)
// All values in EMU
err = slide.AddImage(imgRef, 0, 0, 9144000, 5143250)
if err != nil {
    return fmt.Errorf("failed to add image to slide: %w", err)
}
```

**4. Saving**
```go
err := ppt.SaveToFile("output.pptx")
if err != nil {
    return fmt.Errorf("failed to save PPTX: %w", err)
}
```

### Implementation Pattern for Full-Frame Slides

```go
package services

import (
    "context"
    "fmt"
    "os"

    "github.com/Muprprpr/Go-pptx"
    "go.uber.org/zap"
)

type PPTXGenerator struct {
    logger *zap.Logger
}

func NewPPTXGenerator(logger *zap.Logger) *PPTXGenerator {
    return &PPTXGenerator{
        logger: logger,
    }
}

const (
    // EMU per inch
    EMU_PER_INCH = 914400

    // 16:9 slide dimensions in inches
    SLIDE_WIDTH_INCH  = 10.0
    SLIDE_HEIGHT_INCH = 5.625

    // 16:9 slide dimensions in EMU
    SLIDE_WIDTH_EMU  = 10 * 914400  // 9,144,000
    SLIDE_HEIGHT_EMU = 5.625 * 914400 // 5,143,250
)

func (g *PPTXGenerator) GeneratePPTX(ctx context.Context, framePaths []string, outputPath string) (int, error) {
    // Validate inputs
    if len(framePaths) == 0 {
        return 0, fmt.Errorf("frame paths cannot be empty")
    }

    // Ensure output directory exists
    outputDir := filepath.Dir(outputPath)
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return 0, fmt.Errorf("failed to create output directory: %w", err)
    }

    // Create presentation
    ppt := pptx.NewPresentation()

    // Set slide size to 16:9
    ppt.SetSlideSize(SLIDE_WIDTH_EMU, SLIDE_HEIGHT_EMU)

    pageCount := 0

    // Add each frame as a slide
    for _, framePath := range framePaths {
        // Check if file exists
        if _, err := os.Stat(framePath); os.IsNotExist(err) {
            g.logger.Warn("Image file does not exist, skipping",
                zap.String("path", framePath))
            continue
        }

        // Add image to presentation
        imgRef, err := ppt.AddImage(framePath)
        if err != nil {
            g.logger.Error("Failed to add image to presentation, skipping",
                zap.String("path", framePath),
                zap.Error(err))
            continue
        }

        // Create slide
        slide := ppt.AddSlide()

        // Add image to slide (full-frame, no margins)
        // Position: (0, 0) - top-left corner
        // Size: full slide size (16:9)
        err = slide.AddImage(imgRef, 0, 0, SLIDE_WIDTH_EMU, SLIDE_HEIGHT_EMU)
        if err != nil {
            g.logger.Error("Failed to add image to slide, skipping",
                zap.String("path", framePath),
                zap.Error(err))
            continue
        }

        pageCount++
    }

    // Save presentation
    if err := ppt.SaveToFile(outputPath); err != nil {
        return 0, fmt.Errorf("failed to save PPTX: %w", err)
    }

    g.logger.Info("PPTX generated successfully",
        zap.String("output", outputPath),
        zap.Int("page_count", pageCount))

    return pageCount, nil
}
```

## Key Differences from unioffice

| Aspect | unioffice | Go-pptx |
|--------|-----------|---------|
| **License** | Commercial (FREE_TIER_LIMITS) | MIT (free) |
| **Unit System** | measurement.Inch | EMU (English Metric Units) |
| **Slide Creation** | pres.AddSlide() | ppt.AddSlide() |
| **Image Addition** | pres.AddImage() then slide.AddImage() | ppt.AddImage() then slide.AddImage() |
| **Positioning** | SetPosition() + SetSize() | AddImage(x, y, width, height) |
| **Saving** | pres.SaveToFile() | ppt.SaveToFile() |

## Migration Notes

**From unioffice to Go-pptx:**

1. **Replace imports:**
   - Remove: `github.com/unidoc/unioffice/v2/presentation`
   - Remove: `github.com/unidoc/unioffice/v2/measurement`
   - Add: `github.com/Muprprpr/Go-pptx`

2. **Convert measurement units:**
   - `measurement.Inch` → EMU (multiply by 914400)

3. **Update API calls:**
   - `slideImg.Properties().SetPosition(x, y)` → `slide.AddImage(imgRef, x, y, w, h)`
   - `slideImg.Properties().SetSize(w, h)` → Included in AddImage call

4. **Error handling:**
   - Both libraries return errors on image load/save failures
   - Go-pptx error messages may differ

## Threat Model Considerations

**T-02-05: Path Traversal**
- Validate output path is within configured storage directory
- Use filepath.Clean to prevent path traversal attacks
- Same mitigation as unioffice approach

**T-02-06: Denial of Service**
- Limit maximum slides (cap at 500) to prevent OOM
- Same mitigation as unioffice approach

## References

- Go-pptx Repository: https://github.com/Muprprpr/Go-pptx
- EMU Documentation: https://docs.microsoft.com/en-us/windows/win32/vss/english-metric-units
- 16:9 Aspect Ratio: 10" x 5.625" (12192000 x 6858000 EMU)
