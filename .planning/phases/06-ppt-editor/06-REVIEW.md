---
phase: 06-ppt-editor
reviewed: 2025-01-04T12:00:00Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - frontend/src/api/ppt.ts
  - frontend/src/api/transcription.ts
  - frontend/src/components/DuplicateDetectionPanel.tsx
  - frontend/src/components/SlideCapturePanel.tsx
  - frontend/src/components/VideoPreviewPanel.tsx
  - frontend/src/pages/results/index.tsx
  - frontend/src/types/ppt.ts
  - frontend/src/types/transcription.ts
  - internal/handlers/transcription_handler.go
  - internal/migrations/003_add_slide_timestamps.go
  - internal/models/transcription_task.go
  - internal/services/frame_capture_service.go
  - internal/services/ppt_editor_service.go
  - internal/services/similarity_detector.go
  - internal/services/timestamp_mapper.go
findings:
  critical: 2
  warning: 8
  info: 5
  total: 15
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2025-01-04T12:00:00Z
**Depth:** standard
**Files Reviewed:** 15
**Status:** issues_found

## Summary

This review examined 15 files across frontend (TypeScript/React) and backend (Go) for phase 06 (PPT Editor). The implementation adds video preview synchronization, slide capture/insertion, and duplicate slide detection features.

**Key findings:**
- 2 critical issues (memory leaks, race conditions)
- 8 warnings (error handling, validation, type safety)
- 5 info items (code quality, maintainability)

## Critical Issues

### CR-01: Memory Leak in Video Preview Polling

**File:** `frontend/src/pages/results/index.tsx:127-172`

**Issue:** The `loadSlides` function returns a cleanup function for polling, but the async `.then()` pattern creates a race condition where cleanup may not be properly registered. If the component unmounts before the promise resolves, the interval will never be cleared, causing a memory leak.

```typescript
// Current code (lines 199-203):
loadSlides(currentPptId).then((cleanup) => {
  if (cleanup) {
    slidesPollCleanupRef.current = cleanup
  }
})
```

**Fix:** Store the promise and cleanup in the same callback, or use async/await in useEffect directly:

```typescript
// Option 1: Store cleanup immediately
useEffect(() => {
  if (slidesPollCleanupRef.current) {
    slidesPollCleanupRef.current()
    slidesPollCleanupRef.current = null
  }

  if (currentPptId > 0) {
    setCurrentSlide(0)
    const cleanupPromise = loadSlides(currentPptId)

    // Store cleanup synchronously
    const registerCleanup = async () => {
      const cleanup = await cleanupPromise
      if (cleanup) {
        slidesPollCleanupRef.current = cleanup
      }
    }
    registerCleanup()

    return () => {
      if (slidesPollCleanupRef.current) {
        slidesPollCleanupRef.current()
        slidesPollCleanupRef.current = null
      }
    }
  }
}, [currentPptId, loadSlides])
```

**OR (Option 2 - Preferred):** Inline the polling logic directly in useEffect to avoid the async cleanup pattern:

```typescript
useEffect(() => {
  if (slidesPollCleanupRef.current) {
    slidesPollCleanupRef.current()
  }

  if (currentPptId <= 0) return

  setCurrentSlide(0)
  let cancelled = false
  let intervalId: NodeJS.Timeout | null = null

  const poll = async () => {
    if (cancelled) return
    try {
      const response = await getSlides(currentPptId)
      if (cancelled) return

      if (response.data?.status === 'ready') {
        if (intervalId) clearInterval(intervalId)
        if (!cancelled) {
          setSlides(response.data.slides)
          setIsLoadingSlides(false)
        }
      }
    } catch (error) {
      if (!cancelled) {
        console.error('Polling error:', error)
      }
    }
  }

  // Initial check
  poll().then(() => {
    // If still extracting, start polling
    if (!cancelled && isLoadingSlides) {
      intervalId = setInterval(poll, 2000)
    }
  })

  return () => {
    cancelled = true
    if (intervalId) clearInterval(intervalId)
  }
}, [currentPptId])
```

### CR-02: Race Condition in Slide Cache Invalidations

**File:** `internal/services/ppt_editor_service.go:536-676`

**Issue:** The `InsertCapturedFrame` function does not invalidate the timestamp cache after inserting a new slide. This causes the video preview synchronization to use stale timestamp mappings, where slide numbers no longer match video timestamps.

At line 568-574, the code saves the captured frame to cache and regenerates the PPT, but the `timestampMapper.InvalidateCache()` is never called. The `SlideCapturePanel` component will show incorrect timestamps after insertion.

**Fix:** Add cache invalidation after successfully inserting the slide:

```go
// After line 633 (after invalidating slide cache)
// Invalidate timestamp cache as well
if s.timestampMapper != nil {
  s.timestampMapper.InvalidateCache(videoFileID)
}
```

**Additionally**, the `PPTEditorService` needs a reference to `TimestampMapper` to invalidate it. Update the constructor:

```go
// In constructor (line 52-67):
type PPTEditorService struct {
  db                 *gorm.DB
  logger             *zap.Logger
  config             *config.Config
  slideCache         *SlideCacheService
  similarityDetector *SimilarityDetector
  pptxGenerator      *PPTXGenerator
  timestampMapper    *TimestampMapper  // Add this field
}
```

## Warnings

### WR-01: Missing Error Handling in Video Frame Capture

**File:** `internal/services/frame_capture_service.go:36-94`

**Issue:** The `CaptureFrame` function validates the video path but doesn't validate that the file is actually a video file (could be any file type). Passing a non-video file to FFmpeg causes cryptic errors.

**Fix:** Add MIME type or extension validation:

```go
// After line 40, add:
// Validate file extension
ext := strings.ToLower(filepath.Ext(videoPath))
validExts := []string{".mp4", ".avi", ".mov", ".mkv", ".webm", ".flv"}
isValidExt := false
for _, valid := range validExts {
  if ext == valid {
    isValidExt = true
    break
  }
}
if !isValidExt {
  return fmt.Errorf("invalid video file extension: %s (supported: %v)", ext, validExts)
}
```

### WR-02: Type Assertion Without Check

**File:** `frontend/src/components/VideoPreviewPanel.tsx:93-97`

**Issue:** The code uses optional chaining on line 93 but doesn't validate that `response.data.slide_timestamps` is actually an array before calling `.forEach()`. A malformed API response could crash the component.

```typescript
response.data.slide_timestamps.forEach((ts: SlideTimestamp) => {
  map.set(ts.slide_number, ts.timestamp)
})
```

**Fix:** Add array validation:

```typescript
if (response.data?.success && Array.isArray(response.data?.slide_timestamps)) {
  const map = new Map<number, number>()
  response.data.slide_timestamps.forEach((ts: SlideTimestamp) => {
    if (ts.slide_number && typeof ts.timestamp === 'number') {
      map.set(ts.slide_number, ts.timestamp)
    }
  })
  setTimestampMap(map)
  // ... rest of code
}
```

### WR-03: Unvalidated User Input in Shell Command

**File:** `internal/services/frame_capture_service.go:59-83`

**Issue:** While `exec.CommandContext` with separate args prevents shell injection, the `validatePath` function (line 209-231) is defined but **never called**. Any path validation comments are misleading since the validation isn't executed.

**Fix:** Call `validatePath` in both `CaptureFrame` and `GetVideoDuration`:

```go
// In CaptureFrame (after line 38):
if err := s.validatePath(videoPath); err != nil {
  return fmt.Errorf("invalid video path: %w", err)
}

// In GetVideoDuration (after line 155):
if err := s.validatePath(videoPath); err != nil {
  return "", fmt.Errorf("invalid video path: %w", err)
}
```

### WR-04: Missing Timeout on Video Duration Query

**File:** `internal/services/frame_capture_service.go:152-206`

**Issue:** The `GetVideoDuration` function runs `ffprobe` without a timeout. If a video file is corrupted or has invalid metadata, ffprobe may hang indefinitely, blocking the HTTP request.

**Fix:** Add context timeout:

```go
// Change function signature to accept context:
func (s *FrameCaptureService) GetVideoDuration(ctx context.Context, videoPath string) (float64, error) {
  // ... after line 169:
  cmd := exec.CommandContext(ctx, s.ffprobePath, args...)
  // ... rest of code
}

// Call with timeout in CaptureFrame:
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
duration, err := s.GetVideoDuration(ctx, videoPath)
```

### WR-05: Silent Failure on Thumbnail Generation

**File:** `internal/services/ppt_editor_service.go:704-708`

**Issue:** Thumbnail generation failures are only logged as warnings, not returned as errors. This means users may see broken thumbnail images in the UI without knowing why.

**Fix:** Return error if thumbnail generation fails critically (e.g., disk full):

```go
if err := s.generateThumbnail(fullsizePath, thumbnailPath); err != nil {
  // Log and return error for critical failures
  if os.IsNotExist(err) || os.IsPermission(err) {
    return "", fmt.Errorf("failed to generate thumbnail: %w", err)
  }
  s.logger.Warn("Failed to generate thumbnail", zap.Error(err))
  // Continue for decode errors (non-critical)
}
```

### WR-06: Potential Integer Overflow in Slide Number

**File:** `internal/services/ppt_editor_service.go:558`

**Issue:** The insert position validation checks against `pptFile.PageCount+1`, but `PageCount` is an `int`. If PPT has `INT_MAX` slides, this would overflow. While unlikely, it violates defensive coding.

**Fix:** Check for overflow before validation:

```go
// Before line 558:
if pptFile.PageCount == math.MaxInt32 {
  return fmt.Errorf("cannot insert slide: maximum page count reached")
}
if insertPosition < 1 || insertPosition > pptFile.PageCount+1 {
  return fmt.Errorf("invalid insert position: %d (valid range: 1-%d)", insertPosition, pptFile.PageCount+1)
}
```

### WR-07: Duplicate Type Definition

**File:** `frontend/src/components/SlideCapturePanel.tsx:7-15`

**Issue:** `SlideCapturePanelProps` is defined twice (lines 7-15 and 67-75 in types/ppt.ts). This creates confusion and maintenance burden. The component should import the type from `../types/ppt` instead of redefining it.

**Fix:** Remove the duplicate definition:

```typescript
// Remove lines 7-15 entirely
// Add import:
import type { SlideCapturePanelProps } from '../types/ppt'
```

### WR-08: Missing Null Check on Map Access

**File:** `frontend/src/components/VideoPreviewPanel.tsx:272-277`

**Issue:** The sync button callback accesses `timestampMap.get(currentSlide)` without checking if `currentSlide` is undefined first, despite the conditional `if (currentSlide)` on line 272. If `currentSlide` is 0 (falsy but valid), the check fails silently.

**Fix:** Use explicit undefined check:

```typescript
onClick={() => {
  if (currentSlide !== undefined) {
    const timestamp = timestampMap.get(currentSlide)
    if (timestamp !== undefined && videoRef.current) {
      videoRef.current.currentTime = timestamp
    }
  }
}}
```

## Info

### IN-01: Inconsistent Error Message Format

**File:** Multiple files

**Issue:** Error messages use both Chinese and English inconsistently. For example:
- `frontend/src/api/ppt.ts`: Chinese messages ("帧捕获成功")
- `internal/services/frame_capture_service.go`: English messages ("video file not found")

**Fix:** Standardize on one language or implement i18n. Prefer English for backend, use frontend i18n library for UI.

### IN-02: Magic Numbers Without Constants

**File:** `frontend/src/components/VideoPreviewPanel.tsx:18-19`

**Issue:** Constants `SKIP_SECONDS = 10` and `TIME_UPDATE_DEBOUNCE_MS = 1000` are defined but hardcoded values appear elsewhere:
- Line 177: `minDiff < 5` (5 seconds threshold)
- Line 353: `maxHeight: '400px'` hardcoded

**Fix:** Extract to named constants:

```typescript
const SLIDE_SYNC_THRESHOLD_SECONDS = 5
const VIDEO_MAX_HEIGHT = 400
```

### IN-03: Unused Parameter Warning

**File:** `frontend/src/components/SlideCapturePanel.tsx:223`

**Issue:** Parameter `_index` in `handleToggleSelect` callback is unused but causes confusion with `slide.slide_number`.

**Fix:** Remove underscore prefix or use the index:

```typescript
const handleToggleSelect = useCallback(
  (slide: SlideImage, index: number) => {
    // Use index for validation if needed
    const slideId = `${currentPptId}_${slide.slide_number}`
    // ...
  },
  [currentPptId, selectedSlides, currentPpt]
)
```

### IN-04: Missing Comments for Complex Algorithms

**File:** `internal/services/similarity_detector.go:109-152`

**Issue:** The SSIM calculation with sliding windows is complex but lacks comments explaining the 8x8 window choice or C1/C2 constants.

**Fix:** Add docstring:

```go
// calculateSSIM computes Structural Similarity Index using 8x8 sliding windows.
// C1, C2 are stabilization constants based on dynamic range (255 for 8-bit images).
// Returns average SSIM score (0.0-1.0, higher = more similar).
```

### IN-05: Redundant Database Query

**File:** `internal/services/ppt_editor_service.go:303`

**Issue:** After creating backup (line 299), the code reloads the PPT file from database. This query is unnecessary since the backup path is already known from the creation step.

**Fix:** Update the model in memory instead of reloading:

```go
// Remove lines 302-303:
// s.db.First(&pptFile, pptFileID)

// Replace with:
pptFile.BackupPath = backupPath
```

---

_Reviewed: 2025-01-04T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
