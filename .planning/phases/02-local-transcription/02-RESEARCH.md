# Phase 2: Local Transcription - Research

**Researched:** 2026-04-17
**Domain:** Video processing, image similarity detection, PPTX generation in Go
**Confidence:** MEDIUM

## Summary

Phase 2 implements local video transcription by extracting frames, detecting slide changes using multi-dimensional similarity analysis (SSIM + pHash + edge detection), and generating PPTX files. The system uses FFmpeg for frame extraction, pure Go libraries for image processing, and UniOffice for PPTX generation. Progress tracking follows the established SplittingService pattern with real-time status updates via polling.

**Primary recommendation:** Use unidoc/unioffice v2.9.0 for PPTX generation, corona10/goimagehash v1.1.0 for perceptual hashing, and implement SSIM and edge detection in Go using golang.org/x/image v0.39.0. Follow the SplittingService worker pool pattern exactly for concurrent transcription task management.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Video frame extraction | API / Backend | — | FFmpeg execution via Go backend service |
| Image similarity detection | API / Backend | — | Compute-intensive image processing in Go |
| PPTX file generation | API / Backend | — | Server-side file creation using unioffice |
| Progress tracking | Frontend | API / Backend | Frontend polls backend for status updates |
| Transcription trigger | Frontend | API / Backend | User action triggers API call |
| File storage | API / Backend | — | Server filesystem stores extracted frames and PPTX |

## User Constraints (from CONTEXT.md)

### Locked Decisions

**Frame Extraction Strategy:**
- **D-01:** Dual-layer strategy: first extract frames at fixed interval, then apply similarity detection to deduplicate. Two-pass filtering for higher accuracy.
- **D-02:** Default sampling rate is 1 frame per 2 seconds. User can choose from preset options (1s / 2s / 5s) in the transcription trigger UI.
- **D-03:** Frames are extracted as JPEG (quality 95) to temporary directory. JPEG is 5-10x smaller than PNG with negligible impact on similarity detection accuracy. Temp files cleaned up after processing.
- **D-04:** For comparison, frames are downscaled to 720p resolution. For the final PPTX, original-resolution frames are re-extracted by FFmpeg at the detected keyframe timestamps.
- **D-05:** Temp directory cleaned up after transcription completes (both success and failure paths).

**Similarity Detection Algorithm:**
- **D-06:** Pure Go implementation using `golang.org/x/image` for image decoding and `github.com/corona10/goimagehash` for perceptual hashing. SSIM and edge detection implemented in Go. No FFmpeg filter dependency for comparison logic.
- **D-07:** OR logic: any single detection method registering a change causes the frame to be retained. Per ROADMAP: SSIM < 0.85 OR pHash difference > 10 OR edge change rate > 0.25.
- **D-08:** Fixed thresholds from ROADMAP: SSIM < 0.85, pHash diff > 10, edge change rate > 0.25. No user-adjustable parameters for detection thresholds.

**PPTX Generation:**
- **D-09:** Use `unidoc/unioffice` library for PPTX generation (user selected over Go-pptx for better documentation and community support).
- **D-10:** Slide layout: full-frame image with no margins, padding, or decorations. Image fills the entire slide area.
- **D-11:** Slide dimensions: 16:9 (standard widescreen). Modern standard, matches most displays and projectors.
- **D-12:** Each unique (deduplicated) frame becomes one slide page. No additional metadata (page numbers, timestamps) on slides.

**Progress & UX Flow:**
- **D-13:** "转录" button lives in the file list action column (same row as download/delete/split buttons). Applies to both full videos and split segments.
- **D-14:** After clicking "转录", a modal popup shows real-time progress with staged phases: "帧提取中..." → "画面检测中 (45/200)..." → "生成PPT...". User can close the modal and continue browsing; transcription continues in the background.
- **D-15:** On completion, modal displays "转录完成" with a "下载PPT" button and a "关闭" button. File list auto-refreshes to show the new PPT association.
- **D-16:** Polling interval is 5 seconds (slower than Phase 1's 2s — transcription tasks take minutes, 5s reduces server load while still feeling responsive).
- **D-17:** Progress data structure includes: current stage (extracting/detecting/generating), frames processed, total frames, percentage.

### Claude's Discretion

- Exact Go implementation of SSIM calculation
- Exact Go implementation of edge change rate detection
- TranscriptionTask database model design (fields, status enum)
- API endpoint paths and request/response structures
- Temporary directory naming and cleanup strategy
- goimagehash library usage for pHash (dhash vs phash implementation choice)
- How to handle concurrent transcription tasks (queue with workers like SplittingService)
- Error handling for individual frame processing failures

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LCL-01 | FFmpeg按可配置采样率提取视频帧 | FFmpeg frame extraction using `-vf fps=1/2` pattern (1 frame per 2 seconds), configurable via UI presets |
| LCL-02 | 三维联合变化检测去重 | Multi-dimensional similarity detection: SSIM structural similarity, pHash perceptual hashing, edge change rate analysis |
| LCL-03 | 去重后的帧直接生成为PPTX文件 | unioffice library provides slide creation with image insertion, full-frame layout supported |
| LCL-04 | 本地转录进度实时反馈 | Worker pool pattern from SplittingService provides statusMap, polling every 5 seconds for progress updates |
| TRAN-01 | 用户可以手动点击"转录"按钮触发转录 | File list action column button pattern established in Phase 1, add "转录" button following same design |
| TRAN-04 | 转录任务状态实时跟踪（排队中/处理中/完成/失败） | Status enum with polling mechanism, progress modal shows staged phases |
| TRAN-06 | 分割后的视频段落可以单独触发转录 | VideoFile model supports segment files, same transcription API applies to all VideoFile records |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| unidoc/unioffice | v2.9.0 | PPTX generation with slide and image support | [VERIFIED: go list] Latest stable release, excellent documentation, active maintenance, supports image positioning and sizing |
| corona10/goimagehash | v1.1.0 | Perceptual hashing (pHash/dHash) for image comparison | [VERIFIED: go list] Latest stable, proven library for image hashing, Hamming distance calculation built-in |
| golang.org/x/image | v0.39.0 | Image decoding and processing primitives | [VERIFIED: go list] Official Go image extension library, supports multiple formats, required for SSIM implementation |
| FFmpeg | 2021-03-24-git-a77beea6c8-full_build | Frame extraction from video | [VERIFIED: command -v ffmpeg] Already integrated in project, proven frame extraction with `-vf fps` filter |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| GORM | v1.30.0 (existing) | Database ORM for TranscriptionTask model | Already in project, use for transcription task persistence |
| Gin framework | v1.11.0 (existing) | HTTP handlers for transcription API | Already in project, follow handler pattern from split_handler.go |
| Zap logger | v1.27.0 (existing) | Structured logging for transcription tasks | Already in project, use for debugging and monitoring |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| unioffice | Muprprpr/Go-pptx | unioffice has better documentation and community support; Go-pptx is less maintained |
| goimagehash pHash | FFmpeg phase filter | FFmpeg filter is less flexible, harder to integrate with Go-based SSIM/edge detection |
| Custom SSIM | Third-party SSIM library | No mature Go SSIM library exists; custom implementation allows exact threshold control per requirements |

**Installation:**
```bash
go get github.com/unidoc/unioffice/v2@v2.9.0
go get github.com/corona10/goimagehash@v1.1.0
go get golang.org/x/image@v0.39.0
```

**Version verification:** Before writing the Standard Stack table, I verified each recommended package version is current using `go list -m -versions`. All versions listed are the latest stable releases as of 2026-04-17.

## Architecture Patterns

### System Architecture Diagram

```
User clicks "转录" button in file list
        ↓
Frontend: POST /api/v1/videos/{id}/transcribe with sampling_rate
        ↓
Backend (TranscriptionHandler): Validates request, submits task to TranscriptionService
        ↓
Backend (TranscriptionService Worker Pool): 
  ├─ Queue task (videoFileID, samplingRate, createdBy)
  ├─ Update statusMap: "processing"
  └─ Worker picks up task
        ↓
Stage 1: Frame Extraction
  ├─ FFmpeg: ffmpeg -i input.mp4 -vf fps=1/2 -q:v 2 frame_%04d.jpg
  ├─ Extract frames to temp directory (JPEG quality 95)
  ├─ Downscale extracted frames to 720p for comparison
  └─ Update progress: "extracting", frameCount, totalEstimated
        ↓
Stage 2: Similarity Detection
  ├─ For each frame pair (previous, current):
  │   ├─ Calculate SSIM structural similarity
  │   ├─ Calculate pHash difference (Hamming distance)
  │   └─ Calculate edge change rate
  ├─ OR logic: If SSIM < 0.85 OR pHash diff > 10 OR edge rate > 0.25 → keep frame
  ├─ Store unique frame timestamps
  └─ Update progress: "detecting", processedCount, totalFrames
        ↓
Stage 3: PPTX Generation
  ├─ Re-extract unique frames at original resolution using stored timestamps
  ├─ Create presentation with unioffice (16:9 dimensions)
  ├─ For each unique frame: Add slide, insert image, fill entire slide
  ├─ Save PPTX file to storage
  ├─ Register PPTFile in database with SourceVideoFileID
  └─ Update progress: "generating", slidesCreated, totalUniqueFrames
        ↓
Cleanup & Completion
  ├─ Delete temp directory (all extracted frames)
  ├─ Update statusMap: "completed" or "failed"
  ├─ Update PPTFile record with page count and file path
  └─ Return task completion status
        ↓
Frontend Polling (every 5 seconds):
  ├─ GET /api/v1/videos/{id}/transcription-status
  ├─ Update modal with current stage and progress
  └─ On completion: Show "下载PPT" button, refresh file list
```

### Recommended Project Structure

```
internal/
├── models/
│   ├── transcription_task.go         # NEW: TranscriptionTask model
│   └── ppt_file.go                    # EXTEND: Add TranscriptionTaskID, status fields
├── services/
│   ├── transcription_service.go       # NEW: Worker pool, frame extraction, similarity detection, PPTX generation
│   ├── similarity_detector.go         # NEW: SSIM, pHash, edge detection algorithms
│   ├── frame_extractor.go             # NEW: FFmpeg frame extraction with temp management
│   └── pptx_generator.go              # NEW: unioffice-based PPTX creation
├── handlers/
│   └── transcription_handler.go       # NEW: API endpoints (submit, status, cancel)
└── migrations/
    └── [timestamp]_create_transcription_tasks.go  # NEW: Database migration

frontend/
├── src/
│   ├── api/
│   │   └── transcription.ts           # NEW: API client for transcription endpoints
│   ├── pages/
│   │   └── files/
│   │       └── index.tsx              # MODIFY: Add "转录" button to action column
│   ├── components/
│   │   └── TranscriptionProgressModal.tsx  # NEW: Progress modal with staged phases
│   └── types/
│       └── transcription.ts           # NEW: TypeScript interfaces for transcription
```

### Pattern 1: Worker Pool Service (SplittingService Template)

**What:** Task queue with N workers, in-memory status map, cancellable contexts, lifecycle management (Start/Stop).

**When to use:** Long-running background tasks that need concurrent processing with progress tracking.

**Example:**
```go
// Source: internal/services/splitting_service.go (existing code)
type TranscriptionService struct {
    db               *gorm.DB
    logger           *zap.Logger
    config           *config.Config
    taskQueue        chan *TranscriptionTask
    workers          int
    maxRetries       int
    cancelFuncs      map[uint]context.CancelFunc
    mu               sync.RWMutex
    wg               sync.WaitGroup
    ctx              context.Context
    cancel           context.CancelFunc
    ffmpegPath       string
    pptFileService   *PPTFileService
    // Track active transcriptions: videoFileID -> status
    statusMap map[uint]string
    statusMu  sync.RWMutex
}

func (s *TranscriptionService) SubmitTranscription(videoFileID uint, samplingRate float64, createdBy uint) error {
    task := &TranscriptionTask{
        VideoFileID:  videoFileID,
        SamplingRate: samplingRate,
        CreatedBy:    createdBy,
        CreatedAt:    time.Now(),
    }
    s.statusMu.Lock()
    s.statusMap[videoFileID] = "processing"
    s.statusMu.Unlock()
    
    select {
    case s.taskQueue <- task:
        return nil
    default:
        s.statusMu.Lock()
        s.statusMap[videoFileID] = "failed"
        s.statusMu.Unlock()
        return fmt.Errorf("转录任务队列已满")
    }
}

func (s *TranscriptionService) GetTranscriptionStatus(videoFileID uint) string {
    s.statusMu.RLock()
    defer s.statusMu.RUnlock()
    if status, ok := s.statusMap[videoFileID]; ok {
        return status
    }
    return ""
}
```

### Pattern 2: FFmpeg Frame Extraction

**What:** Extract frames at fixed intervals using FFmpeg's fps filter, output as JPEG with quality control.

**When to use:** Converting video to individual frames for analysis or processing.

**Example:**
```go
// Source: FFmpeg documentation and existing conversion_service.go pattern
func extractFrames(ctx context.Context, ffmpegPath, videoPath, outputDir string, fps float64) ([]string, error) {
    // FFmpeg command: -vf fps=1/2 extracts 1 frame every 2 seconds
    // -q:v 2 sets JPEG quality (1-31, lower is better, 2 is high quality)
    args := []string{
        "-y", // Overwrite output files
        "-i", videoPath,
        "-vf", fmt.Sprintf("fps=%f", fps), // 1/fps = seconds per frame (e.g., fps=0.5 = 1 frame per 2s)
        "-q:v", "2", // JPEG quality (high quality)
        filepath.Join(outputDir, "frame_%04d.jpg"),
    }
    
    cmd := exec.CommandContext(ctx, ffmpegPath, args...)
    var stderrBuf bytes.Buffer
    cmd.Stderr = &stderrBuf
    
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("FFmpeg frame extraction failed: %w, stderr: %s", err, stderrBuf.String())
    }
    
    // Read extracted frame files
    frames, err := filepath.Glob(filepath.Join(outputDir, "frame_*.jpg"))
    if err != nil {
        return nil, fmt.Errorf("failed to list extracted frames: %w", err)
    }
    
    sort.Strings(frames)
    return frames, nil
}
```

### Pattern 3: PPTX Generation with unioffice

**What:** Create PowerPoint presentations with images using unioffice library.

**When to use:** Generating slide decks from extracted frames.

**Example:**
```go
// Source: Context7 documentation for unidoc/unioffice
import (
    "github.com/unidoc/unioffice/v2/presentation"
    "github.com/unidoc/unioffice/v2/measurement"
)

func generatePPTX(frames []string, outputPath string) error {
    pres := presentation.New()
    
    for _, framePath := range frames {
        slide := pres.AddSlide()
        
        // Load image
        img, err := common.ImageFromFile(framePath)
        if err != nil {
            return fmt.Errorf("failed to load image %s: %w", framePath, err)
        }
        
        imgRef, err := pres.AddImage(img)
        if err != nil {
            return fmt.Errorf("failed to add image to presentation: %w", err)
        }
        
        // Add image to slide
        slideImg, err := slide.AddImage(imgRef)
        if err != nil {
            return fmt.Errorf("failed to add image to slide: %w", err)
        }
        
        // Full-frame layout: position at (0,0), size to fill entire slide
        // Standard 16:9 slide: 10" x 5.625"
        slideImg.Properties().SetPosition(0*measurement.Inch, 0*measurement.Inch)
        slideImg.Properties().SetSize(10*measurement.Inch, 5.625*measurement.Inch)
    }
    
    if err := pres.SaveToFile(outputPath); err != nil {
        return fmt.Errorf("failed to save presentation: %w", err)
    }
    
    return nil
}
```

### Pattern 4: Perceptual Hashing with goimagehash

**What:** Calculate perceptual hashes and Hamming distance for image similarity detection.

**When to use:** Detecting visual changes between consecutive frames.

**Example:**
```go
// Source: Context7 documentation for corona10/goimagehash
import (
    "github.com/corona10/goimagehash"
    "image/jpeg"
    "os"
)

func calculatePerceptualHashDifference(imgPath1, imgPath2 string) (int, error) {
    file1, err := os.Open(imgPath1)
    if err != nil {
        return 0, err
    }
    defer file1.Close()
    
    file2, err := os.Open(imgPath2)
    if err != nil {
        return 0, err
    }
    defer file2.Close()
    
    img1, err := jpeg.Decode(file1)
    if err != nil {
        return 0, err
    }
    
    img2, err := jpeg.Decode(file2)
    if err != nil {
        return 0, err
    }
    
    // Calculate perceptual hash
    hash1, err := goimagehash.PerceptionHash(img1)
    if err != nil {
        return 0, err
    }
    
    hash2, err := goimagehash.PerceptionHash(img2)
    if err != nil {
        return 0, err
    }
    
    // Calculate Hamming distance (number of differing bits)
    distance, err := hash1.Distance(hash2)
    if err != nil {
        return 0, err
    }
    
    return distance, nil
}
```

### Anti-Patterns to Avoid

- **Sequential frame processing**: Processing frames one-by-one in a single goroutine will be extremely slow. Use worker pools with multiple goroutines for parallel similarity detection.
- **In-memory frame storage**: Loading all frames into memory at once will cause OOM errors for long videos. Process frames in batches, cleanup temp files after each stage.
- **Blocking FFmpeg calls**: Running FFmpeg without timeout/context can hang the worker. Always use `exec.CommandContext` with timeout and cancellation support.
- **Ignoring cleanup errors**: Failure to cleanup temp directories leaves disk space consumed. Implement cleanup in defer statements with error logging.
- **Hard-coded thresholds**: Using magic numbers for similarity thresholds makes tuning difficult. Define constants at package level with clear documentation.
- **Single-method similarity detection**: Relying on only SSIM or only pHash produces false positives/negatives. Multi-dimensional OR logic provides robust change detection.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| PPTX file format generation | Custom XML manipulation, ZIP archives with manual content types | unidoc/unioffice library | PPTX format is complex (XML schemas, relationships, content types), library handles edge cases and compatibility |
| Perceptual hashing algorithms | Custom pHash/dHash implementation with bit manipulation | corona10/goimagehash library | Well-tested implementation, supports multiple hash algorithms, built-in distance calculation |
| FFmpeg command construction | Manual string concatenation for args | exec.CommandContext with structured args array | Proper escaping, cross-platform compatibility, easier debugging |
| Worker pool management | Custom goroutine scheduling, error-prone concurrency patterns | SplittingService pattern (copy structure) | Proven pattern in codebase, handles cancellation, status tracking, graceful shutdown |
| Image decoding | Manual JPEG/PNG parsing | golang.org/x/image with standard image/jpeg | Supports multiple formats, handles edge cases, memory-efficient streaming |
| Progress tracking | Custom WebSocket implementation, pub/sub systems | HTTP polling with statusMap (established pattern) | Simpler, works with existing infrastructure, no additional dependencies |

**Key insight:** Custom implementations of image processing, file format generation, and concurrent worker management are error-prone and reinvent well-solved problems. Use established libraries and patterns from the existing codebase.

## Runtime State Inventory

> N/A for this phase (greenfield implementation, no rename/refactor/migration).

## Common Pitfalls

### Pitfall 1: FFmpeg Keyframe Misalignment

**What goes wrong:** Frame extraction at exact timestamps may miss actual keyframes, resulting in duplicate or blurry frames.

**Why it happens:** FFmpeg's `-ss` parameter has ±2 second precision when used with `-c copy`, and keyframes don't align with exact timestamps.

**How to avoid:** 
1. Use `-vf fps=1/N` filter instead of `-ss` for fixed interval extraction (FFmpeg handles keyframe alignment automatically)
2. For precise timestamp re-extraction in Stage 3, use `-ss` before `-i` (seek before decode) for better precision
3. Document the precision limitation in user-facing text

**Warning signs:** Extracted frames have duplicate content or are blurry/interpolated instead of clean keyframes.

### Pitfall 2: Memory Exhaustion from Large Frame Sets

**What goes wrong:** Processing hundreds of high-resolution frames in parallel causes out-of-memory errors.

**Why it happens:** Loading full-resolution frames into memory for similarity detection consumes GB of RAM for long videos.

**How to avoid:**
1. Downscale frames to 720p for comparison stage (D-04: comparison uses downscaled, PPTX uses original)
2. Process frames in batches (e.g., 50 frames at a time) instead of loading all at once
3. Cleanup temp frame files immediately after each stage completes

**Warning signs:** Process memory usage grows continuously, OOM kills, or system swapping during transcription.

### Pitfall 3: Temp Directory Leaks

**What goes wrong:** Temporary frame files are not deleted, consuming disk space over time.

**Why it happens:** Cleanup logic only runs on success path; errors or crashes leave temp directories.

**How to avoid:**
1. Create temp directory with unique name (e.g., `transcription_{videoFileID}_{timestamp}`)
2. Use `defer os.RemoveAll(tempDir)` at the start of task processing
3. Log cleanup errors separately without failing the entire task

**Warning signs:** Disk space decreasing over time, `temp/` directory growing large, many `transcription_*` directories.

### Pitfall 4: Similarity Detection False Positives

**What goes wrong:** Motion or camera changes trigger false slide change detection, resulting in too many PPT slides.

**Why it happens:** Single detection method (e.g., only pHash) is sensitive to motion, lighting changes, or camera movement.

**How to avoid:**
1. Use multi-dimensional OR logic (SSIM + pHash + edge) as specified in D-07
2. Tune thresholds based on real meeting video testing (D-08: fixed thresholds, but validate empirically)
3. Consider adding a minimum frame gap (e.g., don't allow changes within 3 seconds of previous change)

**Warning signs:** PPTX has too many slides (e.g., 200 slides for a 20-minute video), many slides look identical.

### Pitfall 5: UI Responsiveness During Long Transcriptions

**What goes wrong:** Browser becomes unresponsive or modal blocks navigation during multi-minute transcription tasks.

**Why it happens:** Synchronous API calls or blocking UI operations prevent user from doing other tasks.

**How to avoid:**
1. Use background processing with worker pool (SplittingService pattern)
2. Non-blocking modal that can be closed (D-14: user can close modal, transcription continues)
3. Polling every 5 seconds (D-16) instead of WebSockets (simpler, sufficient for 5-minute tasks)
4. Show notification/banner when transcription completes (even if modal was closed)

**Warning signs:** User complaints about UI freezing, unable to navigate away during transcription.

## Code Examples

Verified patterns from official sources:

### FFmpeg Frame Extraction with Fixed Interval

```go
// Source: FFmpeg official documentation (https://ffmpeg.org/ffmpeg-filters.html#fps-1)
// Extract 1 frame every 2 seconds: fps=1/2 = 0.5 fps
// Extract 1 frame every 5 seconds: fps=1/5 = 0.2 fps
func buildFFmpegFrameExtractionCommand(inputPath, outputPattern string, framesPerSecond float64) []string {
    return []string{
        "-y", // Overwrite output
        "-i", inputPath,
        "-vf", fmt.Sprintf("fps=%f", framesPerSecond), // Fixed interval extraction
        "-q:v", "2", // JPEG quality (1-31, 2 is high quality)
        "-vsync", "0", // Pass through timestamps without re-encoding
        outputPattern, // e.g., "/path/to/temp/frame_%04d.jpg"
    }
}
```

### Perceptual Hash Similarity Detection

```go
// Source: Context7 documentation for corona10/goimagehash (verified 2026-04-17)
// https://context7.com/corona10/goimagehash/llms.txt
func detectSlideChangeByPerceptualHash(framePath1, framePath2 string, threshold int) (bool, error) {
    img1, err := loadImage(framePath1)
    if err != nil {
        return false, err
    }
    
    img2, err := loadImage(framePath2)
    if err != nil {
        return false, err
    }
    
    hash1, err := goimagehash.PerceptionHash(img1)
    if err != nil {
        return false, err
    }
    
    hash2, err := goimagehash.PerceptionHash(img2)
    if err != nil {
        return false, err
    }
    
    distance, err := hash1.Distance(hash2)
    if err != nil {
        return false, err
    }
    
    // D-07: pHash difference > 10 indicates change
    return distance > threshold, nil // threshold = 10 per requirements
}
```

### PPTX Generation with Full-Frame Images

```go
// Source: Context7 documentation for unidoc/unioffice (verified 2026-04-17)
// https://context7.com/unidoc/unioffice/llms.txt
func addFullFrameImageToSlide(slide *presentation.Slide, imgRef common.ImageRef) error {
    slideImg, err := slide.AddImage(imgRef)
    if err != nil {
        return err
    }
    
    // D-10: Full-frame layout with no margins
    // D-11: Standard 16:9 slide dimensions (10" x 5.625")
    slideImg.Properties().SetPosition(0*measurement.Inch, 0*measurement.Inch)
    slideImg.Properties().SetSize(10*measurement.Inch, 5.625*measurement.Inch)
    
    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Python-based transcription | Pure Go implementation | This phase | Single-process deployment, no Python microservice dependency |
| Single-method similarity detection | Multi-dimensional OR logic (SSIM + pHash + edge) | This phase | Reduced false positives, more robust slide change detection |
| Sequential frame processing | Worker pool with parallel processing | This phase | Faster transcription for long videos, better resource utilization |
| Manual cleanup | Defer-based automatic temp cleanup | This phase | Prevents disk space leaks, cleaner error handling |

**Deprecated/outdated:**
- **Python OpenCV for image processing**: Replaced by pure Go libraries (goimagehash, golang.org/x/image) per D-06
- **FFmpeg scene detection filter**: Replaced by Go-based similarity detection for better control and integration
- **Cloud-only transcription**: Local transcription first (Phase 2), cloud fallback later (Phase 4) per architecture

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | SSIM < 0.85 threshold is appropriate for slide change detection | Similarity Detection Algorithm | May produce too many/false slides if threshold needs tuning for actual meeting videos |
| A2 | pHash Hamming distance > 10 indicates perceptual change | Similarity Detection Algorithm | Sensitive to motion/camera movement; may need adjustment based on real-world testing |
| A3 | Edge change rate > 0.25 threshold detects content changes | Similarity Detection Algorithm | [ASSUMED] Edge detection algorithm not yet specified; actual implementation may require different threshold |
| A4 | unioffice library can handle 100+ slide PPTX generation without memory issues | PPTX Generation | Large PPTX files may hit memory limits; need testing with real-world video lengths |
| A5 | JPEG quality 95 (FFmpeg -q:v 2) preserves enough detail for similarity detection | Frame Extraction Strategy | Lower quality may cause false positives; may need quality tuning |
| A6 | 720p downscaling for comparison is sufficient for similarity detection | Frame Extraction Strategy | May miss subtle slide changes; need empirical validation |
| A7 | 5-second polling interval feels responsive for multi-minute tasks | Progress & UX Flow | Users may perceive 5-second delay as sluggish; UX testing needed |
| A8 | Existing SplittingService worker pool pattern scales to transcription workloads | Architecture Patterns | Transcription tasks are longer and more memory-intensive; may need worker count tuning |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed. (This table IS NOT empty; 8 assumptions require user validation).

## Open Questions

1. **SSIM Implementation Algorithm**
   - What we know: Need pure Go implementation of SSIM structural similarity
   - What's unclear: Which specific SSIM variant to implement (standard SSIM, multi-scale SSIM, etc.) and window size parameter
   - Recommendation: Implement standard SSIM with 8x8 Gaussian window (common default), validate empirically with real meeting videos

2. **Edge Detection Algorithm Choice**
   - What we know: Need edge change rate detection > 0.25 threshold
   - What's unclear: Which edge detection algorithm (Sobel, Canny, Laplacian) and how to calculate "change rate"
   - Recommendation: Start with Sobel edge detection (simpler, faster), calculate edge pixel ratio change; if results are poor, consider Canny

3. **TranscriptionTask Database Model Fields**
   - What we know: Need TranscriptionTask model to track status and progress
   - What's unclear: Exact fields needed (videoFileID FK, status enum, progress tracking, samplingRate, resultPPTXFileID, etc.)
   - Recommendation: Design model to support: status (queued/processing/completed/failed), currentStage (extracting/detecting/generating), framesProcessed, totalFrames, percentage, resultPPTXFileID, error message

4. **Concurrent Task Limits**
   - What we know: Use worker pool pattern from SplittingService
   - What's unclear: How many concurrent transcription tasks to allow (SplittingService uses 2 workers, but transcription is more memory-intensive)
   - Recommendation: Start with 1 worker (serial processing) to validate memory usage, then increase to 2-3 workers if memory allows

5. **Error Handling for Individual Frame Failures**
   - What we know: Some frames may fail to decode or process
   - What's unclear: Should individual frame failures fail the entire transcription, or skip and continue?
   - Recommendation: Skip failed frames with error logging, but fail the entire task if >10% of frames fail (indicates corrupted video)

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| FFmpeg | Frame extraction | ✓ | 2021-03-24-git-a77beea6c8-full_build | — |
| Go 1.24+ | All Go code | ✓ | 1.24.5 | — |
| unioffice v2.9.0 | PPTX generation | ✗ | — | Must install via `go get` |
| goimagehash v1.1.0 | Perceptual hashing | ✗ | — | Must install via `go get` |
| golang.org/x/image v0.39.0 | Image processing | ✗ | — | Must install via `go get` |

**Missing dependencies with no fallback:**
- unioffice v2.9.0: Required for PPTX generation, no alternative library meets requirements
- goimagehash v1.1.0: Required for perceptual hashing, no alternative Go library provides pHash
- golang.org/x/image v0.39.0: Required for image decoding and processing primitives

**Missing dependencies with fallback:**
- None

**Installation required before execution:**
```bash
go get github.com/unidoc/unioffice/v2@v2.9.0
go get github.com/corona10/goimagehash@v1.1.0
go get golang.org/x/image@v0.39.0
```

## Validation Architecture

> workflow.nyquist_validation is not explicitly false in config.json (default: enabled) — include this section.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing package (testing) + testify assertions |
| Config file | None — uses standard Go test conventions |
| Quick run command | `go test -v -run TestTranscriptionService ./internal/services/` |
| Full suite command | `go test -v ./internal/services/... ./internal/handlers/...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| LCL-01 | FFmpeg extracts frames at configurable sampling rate | integration | `go test -v -run TestFrameExtraction ./internal/services/` | ❌ Wave 0 |
| LCL-02 | Multi-dimensional similarity detection (SSIM + pHash + edge) | unit | `go test -v -run TestSimilarityDetection ./internal/services/` | ❌ Wave 0 |
| LCL-03 | PPTX generation with unique frames | integration | `go test -v -run TestPPTXGeneration ./internal/services/` | ❌ Wave 0 |
| LCL-04 | Real-time progress tracking | unit | `go test -v -run TestProgressTracking ./internal/services/` | ❌ Wave 0 |
| TRAN-01 | Transcription trigger from file list | integration | `go test -v -run TestSubmitTranscription ./internal/handlers/` | ❌ Wave 0 |
| TRAN-04 | Status tracking (queued/processing/completed/failed) | unit | `go test -v -run TestTranscriptionStatus ./internal/services/` | ❌ Wave 0 |
| TRAN-06 | Segment transcription | integration | `go test -v -run TestSegmentTranscription ./internal/handlers/` | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go test -v -run TestTranscriptionService ./internal/services/`
- **Per wave merge:** `go test -v ./internal/services/... ./internal/handlers/...`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/services/transcription_service_test.go` — Tests for TranscriptionService worker pool, frame extraction, similarity detection, PPTX generation
- [ ] `internal/services/similarity_detector_test.go` — Unit tests for SSIM, pHash, edge detection algorithms with known image pairs
- [ ] `internal/services/frame_extractor_test.go` — Integration tests for FFmpeg frame extraction with mock videos
- [ ] `internal/services/pptx_generator_test.go` — Integration tests for unioffice PPTX generation
- [ ] `internal/handlers/transcription_handler_test.go` — API handler tests for transcription endpoints
- [ ] Test fixtures: Sample MP4 videos, test frame images, expected PPTX outputs
- [ ] Framework install: None required — uses standard Go testing package

*(If no gaps: "None — existing test infrastructure covers all phase requirements")*
**Actual Status:** Wave 0 gaps exist — transcription service is new code, needs comprehensive test coverage before implementation.

## Security Domain

> security_enforcement is enabled (absent = enabled in config.json) — include this section.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | Validate samplingRate parameter (1s/2s/5s presets only), validate videoFileID exists and belongs to user, sanitize file paths |
| V8 Error Handling | yes | Log errors without exposing system details, return generic error messages to API clients, handle FFmpeg failures gracefully |
| V9 Memory Management | yes | Cleanup temp directories with defer, limit concurrent transcription tasks to prevent OOM, process frames in batches |
| V11 File Handling | yes | Validate file paths are within expected directories, sanitize PPTX filenames, prevent path traversal attacks |

### Known Threat Patterns for {Go Video Processing with FFmpeg}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command Injection via FFmpeg args | Tampering | Use structured args array (not string concatenation), validate samplingRate is numeric, whitelist allowed FPS values |
| Path Traversal in temp file paths | Tampering | Use filepath.Join() (not string concatenation), validate paths are within temp directory, disallow absolute paths |
| DoS via large video uploads | Denial of Service | Limit concurrent transcription tasks (worker pool size), set FFmpeg timeout context, validate video file size before processing |
| Temp Directory Disk Fill | Denial of Service | Enforce temp directory size quotas, cleanup on success/failure paths, monitor disk space usage |
| Memory Exhaustion from frame loading | Denial of Service | Downscale frames to 720p for comparison, process in batches (not all at once), use streaming image decode |
| Unauthorized file access | Spoofing | Validate SourceVideoFileID ownership before transcription, check user permissions on video file, audit PPTX file access |

## Sources

### Primary (HIGH confidence)

- [Context7: unidoc/unioffice v2] - PPTX creation, slide manipulation, image insertion and positioning
- [Context7: corona10/goimagehash v1] - Perceptual hashing algorithms (AverageHash, DifferenceHash, PerceptionHash), Hamming distance calculation
- [FFmpeg Official Documentation] - fps filter for frame extraction at fixed intervals (`-vf fps=1/N`), JPEG quality control (`-q:v` parameter)
- [Go Module Registry] - Verified latest stable versions: unioffice v2.9.0, goimagehash v1.1.0, golang.org/x/image v0.39.0
- [Existing Codebase] - SplittingService worker pool pattern (internal/services/splitting_service.go), FFmpeg invocation pattern (internal/services/conversion_service.go), PPTFile model (internal/models/ppt_file.go)

### Secondary (MEDIUM confidence)

- [Project CONTEXT.md Decisions D-01 through D-17] - User-approved implementation approach for frame extraction, similarity detection, PPTX generation, progress tracking
- [Project REQUIREMENTS.md] - LCL-01 through LCL-04 acceptance criteria, TRAN-01/TRAN-04/TRAN-06 requirements
- [Project STATE.md] - Tech stack context, critical pitfalls (OSS file orphaning, FFmpeg keyframe misalignment, PPT download timeouts)

### Tertiary (LOW confidence)

- [Web Search: Go SSIM implementation] - No Go-specific SSIM library found; custom implementation required (LOW confidence due to lack of official Go SSIM reference)
- [Web Search: Go edge detection] - Rate limited during search; based on training knowledge of Sobel/Canny algorithms (LOW confidence, needs verification)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All library versions verified via `go list -m -versions`, unioffice and goimagehash usage verified via Context7
- Architecture: MEDIUM - Worker pool pattern proven in existing codebase, FFmpeg frame extraction well-documented, but SSIM/edge detection implementations are custom (no library)
- Pitfalls: MEDIUM - FFmpeg keyframe misalignment and temp directory leaks are well-documented issues, but similarity detection false positives depend on empirical threshold tuning

**Research date:** 2026-04-17
**Valid until:** 2026-05-17 (30 days - stable dependencies, but threshold tuning may invalidate assumptions)
