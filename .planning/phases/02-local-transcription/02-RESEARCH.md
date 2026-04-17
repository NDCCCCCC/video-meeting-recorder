# Phase 02 Research: PPTX Generation Libraries

## Context

Original plan specified unidoc/unioffice/v2, but discovered it requires a commercial license for PPTX file generation (FREE_TIER_LIMITS error in unioffice). User decided: **Switch to Muprprpr/Go-pptx (MIT license, free).**

**CRITICAL DISCOVERY:** Muprprpr/Go-pptx is NOT a library - it's a command-line program that imports unioffice/v2 as a dependency. When we added it, go.mod showed `github.com/unidoc/unioffice/v2` as a transitive dependency, and tests failed with "unioffice license required" error.

**NEW DECISION:** Switch to SiliconCatalyst/officeforge (MIT license, pure Go, zero external dependencies).

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

### Muprprpr/Go-pptx (REJECTED - NOT A LIBRARY)

**Repository:** github.com/Muprprpr/Go-pptx

**License:** MIT License (free, no restrictions)

**CRITICAL ISSUE:** This is NOT an importable Go library - it's a command-line program. When we added it with `go get`, Go's module system downloaded it, but it imports `github.com/unidoc/unioffice/v2` as a dependency. Tests immediately failed with:

```
Unlicensed version of UniOffice
- Get a trial license on https://unidoc.io
Error: failed to save PPTX file: unioffice license required
```

**Verdict:** REJECTED - Same licensing problem as unioffice (it wraps unioffice)

---

### SiliconCatalyst/officeforge (SELECTED - FINAL CHOICE)

**Repository:** github.com/SiliconCatalyst/officeforge

**License:** MIT License (free, no restrictions)

**Documentation:** https://github.com/SiliconCatalyst/officeforge

**Key Features:**
- Pure Go library for generating Word, Excel, and PowerPoint documents
- Zero external dependencies (built on standard library)
- Active development (last updated: 2026-01-23)
- Includes `pptx` subpackage for PowerPoint generation
- MIT licensed with no commercial restrictions

**Repository Structure:**
```
officeforge/
├── pptx/           # PowerPoint generation
├── docx/           # Word generation
├── examples/       # Usage examples
├── go.mod          # Module: github.com/siliconcatalyst/officeforge
└── README.md
```

**Pros:**
- ✅ MIT license - no cost, no restrictions
- ✅ Pure Go - no external dependencies
- ✅ Zero licensing issues
- ✅ Can generate PPTX files from scratch
- ✅ Support for images, text, shapes
- ✅ Standard library only - maximum portability

**Cons:**
- Newer library (2 stars, limited adoption)
- Less mature than unioffice
- Documentation may be limited
- API may differ significantly from unioffice

**Verdict:** SELECTED - Only viable free option that actually works

## Pattern 3: officeforge PPTX Usage for Full-Frame 16:9 Image Slides

### Key API Concepts

**Note:** As of this research, we need to explore the officeforge API to understand:
1. How to create a new presentation
2. How to add slides
3. How to add images to slides
4. How to position and size images (full-frame 16:9)
5. How to save the PPTX file

**Estimated API Pattern** (to be verified during implementation):
```go
import "github.com/siliconcatalyst/officeforge/pptx"

// Create new presentation
ppt := pptx.NewPresentation()

// Add slide
slide := ppt.AddSlide()

// Add image to slide
// officeforge likely uses standard units (pixels/inches) or EMU
err := slide.AddImage("path/to/image.jpg", x, y, width, height)
```

### Implementation Approach

Since officeforge is a newer library with limited documentation, the implementation will follow this discovery process:

1. **Add dependency:** `go get github.com/siliconcatalyst/officeforge`
2. **Explore API:** Check the `pptx` package structure and available functions
3. **Review examples:** Look at the `examples/` directory for usage patterns
4. **Implement PPTXGenerator:** Adapt our existing structure to use officeforge API
5. **Test generation:** Verify PPTX files are valid and open in PowerPoint/LibreOffice

### Key Differences from unioffice

| Aspect | unioffice | officeforge |
|--------|-----------|-------------|
| **License** | Commercial (FREE_TIER_LIMITS) | MIT (free) |
| **Dependencies** | Multiple external deps | Zero external deps (pure Go) |
| **API Maturity** | Mature, well-documented | Newer, limited docs |
| **Unit System** | measurement.Inch | TBD (likely pixels or EMU) |
| **Image Support** | Full-featured | TBD (basic support expected) |

## Migration Notes

**From unioffice to officeforge:**

1. **Replace imports:**
   - Remove: `github.com/unidoc/unioffice/v2/presentation`
   - Remove: `github.com/unidoc/unioffice/v2/measurement`
   - Remove: `github.com/unidoc/unioffice/v2/common`
   - Add: `github.com/siliconcatalyst/officeforge/pptx`

2. **API exploration required:**
   - Check if officeforge uses EMU or different unit system
   - Verify image addition API differs from unioffice's two-step process
   - Confirm slide size customization is supported

3. **Error handling:**
   - officeforge likely returns standard Go errors
   - No licensing errors to handle

4. **Testing:**
   - Must verify generated PPTX files open correctly in PowerPoint/LibreOffice
   - Test with actual image files to ensure full-frame layout works

## Threat Model Considerations

**T-02-05: Path Traversal**
- Validate output path is within configured storage directory
- Use filepath.Clean to prevent path traversal attacks
- Same mitigation as unioffice approach

**T-02-06: Denial of Service**
- Limit maximum slides (cap at 500) to prevent OOM
- Same mitigation as unioffice approach

## Implementation Plan

1. Remove Muprprpr/Go-pptx dependency from go.mod
2. Add SiliconCatalyst/officeforge dependency
3. Explore officeforge/pptx API to understand usage
4. Rewrite pptx_generator.go using officeforge API
5. Update tests if needed to match new API
6. Verify tests pass and generate valid PPTX files
7. Create SUMMARY.md documenting the migration

## References

- officeforge Repository: https://github.com/SiliconCatalyst/officeforge
- officeforge License: MIT (https://github.com/SiliconCatalyst/officeforge/blob/main/LICENSE)
- 16:9 Aspect Ratio: 10" x 5.625" (standard widescreen)
- EMU Documentation: https://docs.microsoft.com/en-us/windows/win32/vss/english-metric-units
