# Phase 8: Video Snapshot & Player Enhancement - Research

**Researched:** 2026-04-20
**Domain:** Video processing, FFmpeg operations, React video player UI, snapshot naming
**Confidence:** MEDIUM

## Summary

Phase 8 focuses on enhancing the video snapshot functionality and video player precision. The phase requires improving:
1. **Snapshot time range logic** - Analyzing and validating the incremental snapshot implementation (D-15 from Phase 1)
2. **Snapshot naming conventions** - Current naming uses `snapshot_20060102_150405.mp4` format; potential enhancements for better organization
3. **Video player precision** - Seeking accuracy, frame-level navigation, and precise time input
4. **Video player controls** - Enhanced playback controls, keyboard shortcuts for accessibility and power users
5. **Edge cases** - Concurrent snapshots, recording interruption, and error handling

The research reveals that the current snapshot implementation (in `snapshot_service.go`) already handles incremental snapshots using `SnapshotOffset` tracking, but there may be opportunities to improve naming conventions, add validation, and enhance the video player with better precision controls and keyboard shortcuts.

**Primary recommendation:** Enhance the existing incremental snapshot implementation with better naming conventions (e.g., including task name or sequence number), add comprehensive keyboard shortcuts to VideoPlayerModal following industry standards (YouTube/VLC patterns), implement frame-level seeking with FFmpeg re-encode mode, and add robust validation for edge cases like concurrent snapshots and recording interruptions.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SNAPSHOT-01 | Analyze and validate incremental snapshot time range logic | Current implementation in `snapshot_service.go` lines 62-103, uses `SnapshotOffset` tracking |
| SNAPSHOT-02 | Implement improved snapshot naming conventions | Current format `snapshot_20060102_150405.mp4`, can enhance with task context |
| PLAYER-01 | Add frame-level seeking precision | HTML5 video API has limitations, FFmpeg re-encode mode for frame accuracy |
| PLAYER-02 | Implement keyboard shortcuts for video controls | Industry standard patterns (Space, arrows, J/K/L, M, F) |
| PLAYER-03 | Enhanced playback controls (slow-mo, frame-by-frame) | `playbackRate` property, `requestVideoFrameCallback` API |
| EDGE-01 | Handle concurrent snapshot requests | Mutex/locking in Go service layer, request queuing |
| EDGE-02 | Handle recording interruption during snapshot | Validation checks, graceful error handling |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Snapshot time range calculation | API / Backend | -- | Server-side FFmpeg processing, database state management |
| Snapshot file naming | API / Backend | -- | File system operations, database record creation |
| Video player seeking | Browser / Client | -- | HTML5 video API, React state management |
| Keyboard shortcuts | Browser / Client | -- | Client-side event handling, accessibility |
| Frame-level navigation | Browser / Client | API / Backend | Client-side UI triggers server-side re-encode if needed |
| Concurrent request handling | API / Backend | -- | Server-side locking, queue management |
| Recording interruption handling | API / Backend | -- | Server-side validation, state machine |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **FFmpeg** | Existing install | Video processing, snapshot extraction, frame-accurate seeking | Already integrated, supports `-ss` seeking with precision modes |
| **Go 1.24** | Project version | Backend service, snapshot logic | Consistent with existing codebase, supports concurrency primitives |
| **React 19** | Project version | Frontend video player enhancements | Existing VideoPlayerModal component, hooks for state management |
| **Ant Design 6** | Project version | UI components (Slider, Button, Tooltip) | Already used, provides accessible components |
| **GORM** | Project version | Database operations for VideoFile records | Existing ORM layer, supports `SnapshotOffset` field |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **html5-video-api** | Browser native | Video seeking, playback control | All video player interactions |
| **useRef/useEffect** | React 19 | Video element references, event listeners | Managing HTMLVideoElement state |
| **requestVideoFrameCallback** | Browser API | Frame-level navigation (if supported) | Frame-by-frame navigation in supported browsers |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| HTML5 video element | Video.js, Plyr | External libraries add complexity; HTML5 is sufficient for current needs |
| FFmpeg re-encode | Waveform Precision Seek | Re-encode is slower but more compatible; waveform tools add heavy dependencies |
| Keyboard event listeners | react-hotkeys-hook | Library provides abstraction but adds dependency; native listeners are simple |

**Installation:**
```bash
# No additional packages needed - all dependencies already in project
# Video player enhancements use existing React 19 + Ant Design 6
```

**Version verification:**
```bash
# All packages are already installed in the project
# Verify existing versions:
npm view react version  # Should match project's React 19
npm view antd version   # Should match project's Ant Design 6
```

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                            │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                     VideoPlayerModal                          │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐   │  │
│  │  │   HTML5     │  │  Timeline    │  │  Control Bar     │   │  │
│  │  │   Video     │  │  Slider      │  │  (Keyboard + UI) │   │  │
│  │  │   Element   │  │  + Markers   │  │                  │   │  │
│  │  └─────────────┘  └──────────────┘  └──────────────────┘   │  │
│  │         │                  │                  │              │  │
│  │         └──────────────────┴──────────────────┘              │  │
│  │                            │                                  │  │
│  │                   ┌────────▼────────┐                         │  │
│  │                   │  React State    │                         │  │
│  │                   │  (currentTime,  │                         │  │
│  │                   │   playbackRate,  │                         │  │
│  │                   │   markers, etc.) │                         │  │
│  │                   └────────┬────────┘                         │  │
│  └────────────────────────────┼──────────────────────────────────┘  │
└───────────────────────────────┼──────────────────────────────────────┘
                                │
                                │ HTTP API
                                ▼
┌─────────────────────────────────────────────────────────────────────┐
│                    Backend (Go API Server)                          │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │                   Snapshot Handler                            │  │
│  │  POST /api/v1/tasks/{id}/snapshot                            │  │
│  └───────────────────────────┬──────────────────────────────────┘  │
│                              │                                      │
│  ┌───────────────────────────▼──────────────────────────────────┐  │
│  │                   SnapshotService                             │  │
│  │  ┌────────────────────────────────────────────────────────┐  │  │
│  │  │  1. Load recording task                                │  │  │
│  │  │  2. Find last snapshot (D-15 incremental logic)        │  │  │
│  │  │  3. Calculate seek offset                              │  │  │
│  │  4. Copy partial MKV to temp                              │  │  │
│  │  │  5. FFmpeg convert with -ss and -to                    │  │  │
│  │  │  6. Register via VideoFileService                      │  │  │
│  │  └────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────┬──────────────────────────────────┘  │
│                              │                                      │
│  ┌───────────────────────────▼──────────────────────────────────┐  │
│  │                   VideoFileService                            │  │
│  │  - CreateSegmentFile()                                        │  │
│  │  - Store SnapshotOffset                                       │  │
│  │  - Return snapshot metadata                                  │  │
│  └───────────────────────────┬──────────────────────────────────┘  │
│                              │                                      │
│  ┌───────────────────────────▼──────────────────────────────────┐  │
│  │                   Database (SQLite/GORM)                      │  │
│  │  - video_files table (id, file_name, snapshot_offset, etc.)  │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/
├── services/
│   ├── snapshot_service.go          # [EXISTING] Snapshot generation logic
│   │   # Enhancements needed:
│   │   # - Add mutex for concurrent snapshot protection
│   │   # - Improve naming convention with task context
│   │   # - Add validation for recording interruption
│   │
│   └── video_file_service.go        # [EXISTING] File management
│       # Already supports CreateSegmentFile with SnapshotOffset
│
├── handlers/
│   └── task_handler.go              # [EXISTING] Snapshot endpoint
│       # Already has POST /api/v1/tasks/{id}/snapshot
│
└── models/
    └── video_file.go                # [EXISTING] VideoFile model
        # Already has SnapshotOffset field

frontend/src/
├── components/
│   ├── VideoPlayerModal.tsx         # [EXISTING] Main video player
│   │   # Enhancements needed:
│   │   # - Add keyboard shortcuts (Space, arrows, J/K/L, M, F)
│   │   # - Add frame-by-frame navigation
│   │   # - Add slow-motion playback
│   │   # - Improve seeking precision
│   │
│   ├── TimelineWithMarkers.tsx      # [EXISTING] Timeline with split markers
│   │   # Already supports marker-based seeking
│   │
│   └── EditableProgressBar.tsx      # [EXISTING] Time input + progress bar
│       # Already supports MM:SS time input seeking
│
├── hooks/
│   ├── useKeyboardShortcuts.ts      # [NEW] Keyboard shortcut management
│   └── useVideoFrameNavigation.ts   # [NEW] Frame-level navigation
│
└── utils/
    └── videoPlayerHotkeys.ts        # [NEW] Keyboard shortcut definitions
```

### Pattern 1: Incremental Snapshot with Offset Tracking (D-15)

**What:** Each snapshot starts from where the previous snapshot ended, using FFmpeg's `-ss` seek parameter to skip already-captured content.

**When to use:** Recording snapshot feature where users want to export incremental portions of an ongoing recording.

**Example:**
```go
// Source: internal/services/snapshot_service.go (lines 62-77)
// Find the last snapshot for this task to determine incremental offset
var lastSnapshot models.VideoFile
seekOffset := 0.0
if err := s.db.Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeSnapshot).
    Order("created_at DESC").First(&lastSnapshot).Error; err == nil {
    // Last snapshot found — calculate its end offset (snapshot_offset + duration)
    if lastSnapshot.SnapshotOffset > 0 || lastSnapshot.Duration > 0 {
        seekOffset = lastSnapshot.SnapshotOffset + float64(lastSnapshot.Duration)
    }
    s.logger.Info("增量快照: 从上次结束点开始",
        zap.Uint("task_id", taskID),
        zap.Float64("last_offset", lastSnapshot.SnapshotOffset),
        zap.Int("last_duration", lastSnapshot.Duration),
        zap.Float64("seek_offset", seekOffset),
    )
}

// Use seekOffset in FFmpeg command
args := []string{"-y"}
if seekOffset > 0 {
    args = append(args, "-ss", fmt.Sprintf("%.3f", seekOffset))
}
args = append(args,
    "-i", tempMKV,
    "-to", fmt.Sprintf("%.3f", recordingDuration),
    "-c", "copy",
    "-movflags", "+faststart",
    outputMP4,
)
```

**Status:** `[VERIFIED: codebase analysis]` - Already implemented in `snapshot_service.go`

### Pattern 2: Keyboard Shortcut Management

**What:** Capture keyboard events at the modal/container level and map them to video player actions.

**When to use:** Enhancing video player accessibility and power-user workflows.

**Example:**
```typescript
// Source: [ASSUMED] - Standard React keyboard event pattern
// Note: This pattern is NOT currently in the codebase - needs to be implemented

import { useEffect, useCallback } from 'react'
import { message } from 'antd'

interface KeyboardShortcutConfig {
  key: string
  ctrlKey?: boolean
  shiftKey?: boolean
  action: () => void
  description: string
}

export function useKeyboardShortcuts(
  videoRef: React.RefObject<HTMLVideoElement>,
  isPlaying: boolean,
  playbackRate: number,
  onPlayPause: () => void,
  onSeek: (seconds: number) => void,
  onVolumeChange: (volume: number) => void,
  onPlaybackRateChange: (rate: number) => void,
  onFullscreen: () => void,
) {
  const shortcuts: KeyboardShortcutConfig[] = [
    { key: ' ', action: onPlayPause, description: '播放/暂停' },
    { key: 'ArrowLeft', action: () => onSeek(-10), description: '快退10秒' },
    { key: 'ArrowRight', action: () => onSeek(10), description: '快进10秒' },
    { key: 'ArrowUp', action: () => onVolumeChange(0.1), description: '音量+10%' },
    { key: 'ArrowDown', action: () => onVolumeChange(-0.1), description: '音量-10%' },
    { key: 'j', shiftKey: true, action: () => onSeek(-1), description: '快退1秒' },
    { key: 'l', shiftKey: true, action: () => onSeek(1), description: '快进1秒' },
    { key: 'm', action: () => onVolumeChange(0), description: '静音/取消静音' },
    { key: 'f', action: onFullscreen, description: '全屏' },
    { key: '>', shiftKey: true, action: () => onPlaybackRateChange(1), description: '正常速度' },
  ]

  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    // Ignore if typing in input fields
    if (event.target instanceof HTMLInputElement || 
        event.target instanceof HTMLTextAreaElement) {
      return
    }

    const shortcut = shortcuts.find(s => 
      s.key.toLowerCase() === event.key.toLowerCase() &&
      (s.ctrlKey === undefined || s.ctrlKey === event.ctrlKey) &&
      (s.shiftKey === undefined || s.shiftKey === event.shiftKey)
    )

    if (shortcut) {
      event.preventDefault()
      shortcut.action()
      message.info(shortcut.description)
    }
  }, [shortcuts])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}
```

**Status:** `[ASSUMED]` - Standard React pattern, not currently implemented in VideoPlayerModal

### Pattern 3: Frame-Level Navigation

**What:** Use `requestVideoFrameCallback` API (if supported) or fine-grained seeking for frame-by-frame navigation.

**When to use:** Users need to precisely locate specific frames in the video (e.g., for slide capture).

**Example:**
```typescript
// Source: [ASSUMED] - Based on MDN documentation for requestVideoFrameCallback
// Note: This pattern is NOT currently in the codebase

export function useVideoFrameNavigation(videoRef: React.RefObject<HTMLVideoElement>) {
  const nextFrame = useCallback(() => {
    const video = videoRef.current
    if (!video) return

    // Advance by one frame (assuming 30fps = 0.033s per frame)
    const frameTime = 1 / 30
    video.currentTime = Math.min(video.duration, video.currentTime + frameTime)
  }, [])

  const prevFrame = useCallback(() => {
    const video = videoRef.current
    if (!video) return

    // Go back by one frame
    const frameTime = 1 / 30
    video.currentTime = Math.max(0, video.currentTime - frameTime)
  }, [])

  // Detect if browser supports requestVideoFrameCallback
  const supportsFrameCallback = useCallback(() => {
    const video = videoRef.current
    return video && typeof (video as any).requestVideoFrameCallback === 'function'
  }, [])

  return { nextFrame, prevFrame, supportsFrameCallback }
}
```

**Status:** `[ASSUMED]` - Based on HTML5 video API documentation, browser support varies

### Anti-Patterns to Avoid

- **Anti-pattern:** Using FFmpeg `-c copy` for frame-accurate cuts
  - **Why it's bad:** `-c copy` mode has +/-2s keyframe alignment limitations
  - **What to do instead:** Use re-encode mode for frame accuracy, or document the limitation

- **Anti-pattern:** Attaching keyboard event listeners to individual video elements
  - **Why it's bad:** Creates event propagation issues, doesn't work with modal overlays
  - **What to do instead:** Attach listeners to modal/container level, use event delegation

- **Anti-pattern:** Using `setState` in keyboard event handlers without debouncing
  - **Why it's bad:** Rapid key presses (e.g., holding down arrow key) cause excessive re-renders
  - **What to do instead:** Debounce seek operations, use requestAnimationFrame for updates

- **Anti-pattern:** Not validating snapshot offset against recording duration
  - **Why it's bad:** Can cause FFmpeg errors or invalid snapshots
  - **What to do instead:** Always validate `seekOffset < recordingDuration` before FFmpeg call

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Video player controls | Custom video controls from scratch | HTML5 video API + Ant Design components | Browser native playback is optimized, handles format compatibility, accessibility built-in |
| Keyboard shortcut system | Custom event handling logic | React hooks pattern + standard key mappings | Proven pattern, easier to maintain, follows accessibility guidelines |
| Time formatting | Custom time string manipulation | Existing `formatTime` utility in EditableProgressBar | Already tested, handles MM:SS and HH:MM:SS formats |
| FFmpeg command building | String concatenation for commands | `exec.CommandContext` with argument arrays | Safer (shell injection protection), easier to debug |
| Seek validation | Manual boundary checks | Existing validation in TimelineWithMarkers | Already handles edge cases (negative values, duration overflow) |

**Key insight:** The codebase already has robust utilities for time formatting, seeking, and FFmpeg command construction. Phase 8 should enhance and extend these, not replace them.

## Common Pitfalls

### Pitfall 1: FFmpeg Keyframe Misalignment
**What goes wrong:** Using `-c copy` mode results in split points being off by +/-2 seconds from the intended timestamp.

**Why it happens:** H.264/H.265 codecs use keyframes (I-frames) spaced several seconds apart. `-c copy` can only split at keyframes, not arbitrary timestamps.

**How to avoid:**
1. Document the limitation clearly in the UI
2. Offer re-encode mode for frame-accurate splits: `ffmpeg -i input.mp4 -ss 00:01:23 -to 00:02:45 -c:v libx264 -c:a aac output.mp4`
3. Use smart keyframe detection: `ffprobe -select_streams v:0 -show_frames -show_entries frame=pict_type` to find nearest keyframe

**Warning signs:** User complains that split doesn't start/end at the exact timestamp they specified.

### Pitfall 2: Concurrent Snapshot Race Conditions
**What goes wrong:** Two snapshot requests for the same task execute simultaneously, causing both to start from the same offset (the previous snapshot before either request), resulting in duplicate content.

**Why it happens:** The current implementation queries for the last snapshot, then calculates the new offset. Without locking, two requests can read the same "last snapshot" before either writes a new one.

**How to avoid:**
1. Add a mutex in `SnapshotService` to serialize snapshot requests per task:
   ```go
   type SnapshotService struct {
       // ... existing fields
       snapshotMutexes sync.Map // map[uint]*sync.Mutex
   }
   
   func (s *SnapshotService) getMutex(taskID uint) *sync.Mutex {
       mutex, _ := s.snapshotMutexes.LoadOrStore(taskID, &sync.Mutex{})
       return mutex.(*sync.Mutex)
   }
   ```
2. Use database transactions with row locking: `SELECT ... FOR UPDATE`
3. Return HTTP 429 (Too Many Requests) if a snapshot is already in progress for that task

**Warning signs:** Snapshot files have overlapping time ranges, duplicate content between consecutive snapshots.

### Pitfall 3: Recording Interruption During Snapshot
**What goes wrong:** Recording stops or fails while a snapshot is being generated, causing the snapshot to fail or capture incomplete data.

**Why it happens:** Snapshot copies the partial MKV file, but if recording stops during the copy, the file may be corrupted or shorter than expected.

**How to avoid:**
1. Check task status before starting snapshot: `if task.Status != models.VideoStatusRecording { return error }`
2. Use file locking during copy: `os.Open` with read-only mode won't block writes
3. Validate recording duration is positive: `if recordingDuration <= 0 { return error }`
4. Validate seek offset doesn't exceed duration: `if seekOffset >= recordingDuration { return error }`
5. Add transaction logging to track snapshot lifecycle

**Warning signs:** FFmpeg error "Output file is empty or contains no streams", snapshot duration is 0.

### Pitfall 4: Keyboard Shortcut Conflicts
**What goes wrong:** Keyboard shortcuts trigger browser actions (e.g., Ctrl+P for print, Space for page scroll) instead of video controls.

**Why it happens:** Event handlers don't call `event.preventDefault()`, or shortcuts are attached at the wrong level.

**How to avoid:**
1. Always call `event.preventDefault()` for handled shortcuts
2. Attach listener at modal level, use `event.stopPropagation()` to stop bubbling
3. Ignore events from input elements: `if (event.target instanceof HTMLInputElement) return`
4. Provide visual feedback (toast message) when shortcut is triggered

**Warning signs:** Pressing Space scrolls the page instead of toggling playback, Ctrl+P opens print dialog.

### Pitfall 5: Browser Incompatibility for Frame Navigation
**What goes wrong:** Frame-by-frame navigation works in Chrome but fails in Firefox or Safari.

**Why it happens:** `requestVideoFrameCallback` API is not supported in all browsers (as of 2026).

**How to avoid:**
1. Feature-detect before using: `if ('requestVideoFrameCallback' in video)`
2. Provide fallback: use fine-grained seeking (1/30 second increments)
3. Document browser compatibility in user-facing help
4. Consider using a polyfill or alternative approach for unsupported browsers

**Warning signs:** Console errors in Firefox/Safari, frame navigation buttons don't respond.

## Code Examples

### Example 1: Enhanced Snapshot Naming with Task Context

**Current implementation** (snapshot_service.go line 119):
```go
timestamp := time.Now().Format("20060102_150405")
tempMKV := filepath.Join(tempDir, fmt.Sprintf("snapshot_%s.mkv", timestamp))
outputMP4 := filepath.Join(tempDir, fmt.Sprintf("snapshot_%s.mp4", timestamp))
```

**Enhanced version** with task name and sequence:
```go
// Source: [PROPOSED] - Enhancement to existing snapshot_service.go
// Add task context to snapshot filename for better organization

func (s *SnapshotService) generateSnapshotFilename(task models.VideoRecordingTask, sequence int) string {
    // Sanitize task name for filename use
    sanitizedName := strings.Map(func(r rune) rune {
        if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
            return r
        }
        return '_'
    }, task.Name)
    
    // Limit name length to avoid excessively long filenames
    if len(sanitizedName) > 30 {
        sanitizedName = sanitizedName[:30]
    }
    
    timestamp := time.Now().Format("20060102_150405")
    return fmt.Sprintf("%s_snapshot_%03d_%s.mp4", sanitizedName, sequence, timestamp)
}

// Usage in GenerateSnapshot:
// 1. Count existing snapshots for this task
var snapshotCount int64
s.db.Model(&models.VideoFile{}).
    Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeSnapshot).
    Count(&snapshotCount)
sequence := int(snapshotCount) + 1

// 2. Generate filename with context
filename := s.generateSnapshotFilename(task, sequence)
outputMP4 := filepath.Join(tempDir, filename)
```

**Status:** `[PROPOSED]` - Enhancement to existing code, not yet implemented

### Example 2: Mutex Protection for Concurrent Snapshots

```go
// Source: [PROPOSED] - Addition to SnapshotService struct
// Prevents race conditions when multiple snapshot requests arrive simultaneously

type SnapshotService struct {
    db               *gorm.DB
    logger           *zap.Logger
    config           *config.Config
    ffmpegPath       string
    videoFileService *VideoFileService
    snapshotMutexes  sync.Map // map[uint]*sync.Mutex - one mutex per task
}

// getMutex returns (or creates) a mutex for the specified task
func (s *SnapshotService) getMutex(taskID uint) *sync.Mutex {
    mutex, _ := s.snapshotMutexes.LoadOrStore(taskID, &sync.Mutex{})
    return mutex.(*sync.Mutex)
}

func (s *SnapshotService) GenerateSnapshot(taskID uint, createdBy uint) (*models.VideoFile, error) {
    // Acquire mutex for this task to prevent concurrent snapshots
    mutex := s.getMutex(taskID)
    mutex.Lock()
    defer mutex.Unlock()
    
    // ... rest of existing implementation
    
    s.logger.Info("快照生成完成 (互斥锁已释放)",
        zap.Uint("task_id", taskID),
        zap.Uint("snapshot_file_id", snapshotFile.ID),
    )
    
    return snapshotFile, nil
}
```

**Status:** `[PROPOSED]` - Enhancement needed for concurrent request handling

### Example 3: Keyboard Shortcuts Hook for VideoPlayerModal

```typescript
// Source: [PROPOSED] - New hook for VideoPlayerModal
// Manages keyboard shortcuts following industry standards (YouTube/VLC patterns)

import { useEffect, useCallback } from 'react'
import { message } from 'antd'

interface UseKeyboardShortcutsProps {
  videoRef: React.RefObject<HTMLVideoElement>
  isPlaying: boolean
  playbackRate: number
  volume: number
  enabled: boolean // Only enable when modal is open
  onPlayPause: () => void
  onSeek: (seconds: number) => void
  onVolumeChange: (volume: number) => void
  onPlaybackRateChange: (rate: number) => void
  onFullscreen: () => void
  onMuteToggle: () => void
}

export function useKeyboardShortcuts({
  videoRef,
  isPlaying,
  playbackRate,
  volume,
  enabled,
  onPlayPause,
  onSeek,
  onVolumeChange,
  onPlaybackRateChange,
  onFullscreen,
  onMuteToggle,
}: UseKeyboardShortcutsProps) {
  const handleKeyDown = useCallback((event: KeyboardEvent) => {
    if (!enabled) return
    
    // Ignore if typing in input, textarea, or select elements
    const target = event.target as HTMLElement
    if (target.tagName === 'INPUT' || 
        target.tagName === 'TEXTAREA' || 
        target.tagName === 'SELECT' ||
        target.isContentEditable) {
      return
    }

    const key = event.key.toLowerCase()
    const ctrl = event.ctrlKey || event.metaKey
    const shift = event.shiftKey

    // Prevent default for all handled shortcuts
    let handled = false

    switch (true) {
      case key === ' ':
        event.preventDefault()
        onPlayPause()
        message.info(isPlaying ? '暂停' : '播放')
        handled = true
        break

      case key === 'arrowleft':
        event.preventDefault()
        onSeek(shift ? -1 : -10)
        message.info(shift ? '快退 1 秒' : '快退 10 秒')
        handled = true
        break

      case key === 'arrowright':
        event.preventDefault()
        onSeek(shift ? 1 : 10)
        message.info(shift ? '快进 1 秒' : '快进 10 秒')
        handled = true
        break

      case key === 'arrowup':
        event.preventDefault()
        const newVolumeUp = Math.min(1, volume + 0.1)
        onVolumeChange(newVolumeUp)
        message.info(`音量: ${Math.round(newVolumeUp * 100)}%`)
        handled = true
        break

      case key === 'arrowdown':
        event.preventDefault()
        const newVolumeDown = Math.max(0, volume - 0.1)
        onVolumeChange(newVolumeDown)
        message.info(`音量: ${Math.round(newVolumeDown * 100)}%`)
        handled = true
        break

      case key === 'j':
        event.preventDefault()
        onSeek(-10)
        message.info('快退 10 秒')
        handled = true
        break

      case key === 'l':
        event.preventDefault()
        onSeek(10)
        message.info('快进 10 秒')
        handled = true
        break

      case key === 'k':
        event.preventDefault()
        onPlayPause()
        message.info(isPlaying ? '暂停' : '播放')
        handled = true
        break

      case key === 'm':
        event.preventDefault()
        onMuteToggle()
        message.info(volume > 0 ? '静音' : '取消静音')
        handled = true
        break

      case key === 'f':
        event.preventDefault()
        onFullscreen()
        message.info('全屏')
        handled = true
        break

      case key === '>':
      case key === '.':
        if (shift) {
          event.preventDefault()
          const speeds = [0.5, 1, 1.25, 1.5, 2]
          const currentIndex = speeds.indexOf(playbackRate)
          const nextSpeed = speeds[(currentIndex + 1) % speeds.length]
          onPlaybackRateChange(nextSpeed)
          message.info(`播放速度: ${nextSpeed}x`)
          handled = true
        }
        break

      case key === '<':
      case key === ',':
        if (shift) {
          event.preventDefault()
          const speeds = [0.5, 1, 1.25, 1.5, 2]
          const currentIndex = speeds.indexOf(playbackRate)
          const prevSpeed = speeds[(currentIndex - 1 + speeds.length) % speeds.length]
          onPlaybackRateChange(prevSpeed)
          message.info(`播放速度: ${prevSpeed}x`)
          handled = true
        }
        break

      case key === 'home':
        event.preventDefault()
        onSeek(-Infinity) // Seek to start
        message.info('跳转到开始')
        handled = true
        break

      case key === 'end':
        event.preventDefault()
        onSeek(Infinity) // Seek to end
        message.info('跳转到结束')
        handled = true
        break

      case (key >= '0' && key <= '9'):
        event.preventDefault()
        const percentage = parseInt(key) / 10
        const video = videoRef.current
        if (video) {
          video.currentTime = video.duration * percentage
          message.info(`跳转到 ${Math.round(percentage * 100)}%`)
        }
        handled = true
        break
    }
  }, [enabled, isPlaying, playbackRate, volume, onPlayPause, onSeek, onVolumeChange, onPlaybackRateChange, onFullscreen, onMuteToggle, videoRef])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}
```

**Status:** `[PROPOSED]` - New hook to be implemented for VideoPlayerModal

### Example 4: Frame-Level Navigation Component

```typescript
// Source: [PROPOSED] - New component for frame-by-frame navigation
// Adds precision navigation controls for video editing workflows

import { Button, Space, Tooltip } from 'antd'
import { StepBackwardOutlined, StepForwardOutlined, UndoOutlined } from '@ant-design/icons'
import { useVideoFrameNavigation } from '../hooks/useVideoFrameNavigation'

interface FrameNavigationProps {
  videoRef: React.RefObject<HTMLVideoElement>
  disabled?: boolean
}

export function FrameNavigation({ videoRef, disabled }: FrameNavigationProps) {
  const { nextFrame, prevFrame, supportsFrameCallback } = useVideoFrameNavigation(videoRef)

  if (!supportsFrameCallback()) {
    return null // Don't show if browser doesn't support frame-level API
  }

  return (
    <Space size="small">
      <Tooltip title="上一帧 (Shift+←)">
        <Button
          type="text"
          icon={<UndoOutlined />}
          onClick={prevFrame}
          disabled={disabled}
          size="small"
        >
          -1帧
        </Button>
      </Tooltip>
      <Tooltip title="下一帧 (Shift+→)">
        <Button
          type="text"
          icon={<StepForwardOutlined />}
          onClick={nextFrame}
          disabled={disabled}
          size="small"
        >
          +1帧
        </Button>
      </Tooltip>
    </Space>
  )
}
```

**Status:** `[PROPOSED]` - New component to be implemented

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual snapshot from beginning | Incremental snapshots (D-15) | Phase 1 (2026-04-17) | Each snapshot captures only new content since last snapshot |
| Timestamp-only filenames | Context-aware filenames | Phase 8 (proposed) | Filenames include task name and sequence for better organization |
| Mouse-only controls | Keyboard + mouse controls | Phase 8 (proposed) | Improved accessibility and power-user workflows |
| Second-level seeking | Frame-level navigation | Phase 8 (proposed) | Precision navigation for slide capture and editing |
| Single snapshot at a time | Concurrent-safe snapshots | Phase 8 (proposed) | Prevents race conditions from rapid clicks |

**Deprecated/outdated:**
- FFmpeg 2.x syntax: Old `-ss` position matters bug (fixed in FFmpeg 3+)
- Non-incremental snapshots: Always capturing from 0s wastes time and storage

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Incremental snapshot logic (D-15) is working correctly in current implementation | Snapshot Time Range Logic | If bugs exist, need to fix before enhancing |
| A2 | Browser support for `requestVideoFrameCallback` is sufficient for frame navigation | Frame-Level Navigation | May need fallback for Firefox/Safari if support is poor |
| A3 | Current naming convention `snapshot_20060102_150405.mp4` is adequate for users | Snapshot Naming | If users struggle with filename organization, enhancement is higher priority |
| A4 | Ant Design Slider with marks is sufficient for timeline markers | Video Player Controls | May need custom timeline component if requirements evolve |
| A5 | HTML5 video API seeking precision is acceptable for current use cases | Video Player Precision | If users complain about precision, may need FFmpeg-based seeking |

## Open Questions

1. **Snapshot naming priority**
   - What we know: Current naming uses timestamp only (`snapshot_20060102_150405.mp4`)
   - What's unclear: Do users find this confusing? Do they need task name context?
   - Recommendation: Survey users or analyze usage patterns. If users frequently rename snapshots, enhance naming convention.

2. **Frame navigation browser support**
   - What we know: `requestVideoFrameCallback` is Chrome/Edge-only as of 2026
   - What's unclear: What percentage of users use Firefox/Safari? Is frame navigation critical for them?
   - Recommendation: Implement feature detection and provide fallback (fine-grained seeking) for unsupported browsers.

3. **Keyboard shortcut conflicts**
   - What we know: Standard shortcuts (Space, arrows) can conflict with browser/system shortcuts
   - What's unclear: Should we allow users to customize shortcuts? Which shortcuts are non-negotiable?
   - Recommendation: Start with standard YouTube/VLC shortcuts, add customization if users request it.

4. **Concurrent snapshot frequency**
   - What we know: Current implementation has no mutex protection
   - What's unclear: How often do users click snapshot button multiple times rapidly? Is this a real problem?
   - Recommendation: Add mutex protection regardless (defensive programming), but prioritize based on actual user behavior.

5. **Snapshot duration display**
   - What we know: Snapshots have `SnapshotOffset` and `Duration` fields
   - What's unclear: Should UI display the time range (e.g., "0:00 - 5:23") for each snapshot?
   - Recommendation: Yes, display time range in file list to help users understand snapshot content.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| FFmpeg | Snapshot generation, frame-accurate seeking | ✓ | Existing install | — |
| Go 1.24 | Backend service | ✓ | Project version | — |
| React 19 | Frontend components | ✓ | Project version | — |
| Ant Design 6 | UI components | ✓ | Project version | — |
| SQLite/GORM | Database operations | ✓ | Project version | — |
| HTML5 video API | Video playback | ✓ | Browser native | — |
| requestVideoFrameCallback | Frame-level navigation | Partial | Chrome/Edge only | Fine-grained seeking fallback |

**Missing dependencies with no fallback:**
- None

**Missing dependencies with fallback:**
- `requestVideoFrameCallback` - Falls back to fine-grained seeking (1/30 second increments)

## Validation Architecture

> Note: workflow.nyquist_validation is not set in config.json, assuming enabled by default.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing + testify (backend), React Testing Library (frontend) |
| Config file | No specific config — uses standard Go test conventions |
| Quick run command | `go test ./internal/services/... -run TestSnapshot` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SNAPSHOT-01 | Incremental snapshot time range logic | integration | `go test ./internal/services/... -run TestGenerateSnapshot_Incremental` | ❌ Need to create |
| SNAPSHOT-02 | Snapshot naming conventions | unit | `go test ./internal/services/... -run TestGenerateSnapshot_Naming` | ❌ Need to create |
| PLAYER-01 | Frame-level seeking precision | integration | `npm test -- VideoPlayerModal.frame` | ❌ Need to create |
| PLAYER-02 | Keyboard shortcuts functionality | unit | `npm test -- useKeyboardShortcuts` | ❌ Need to create |
| PLAYER-03 | Enhanced playback controls | unit | `npm test -- PlaybackSpeedControl` | ✅ Already exists |
| EDGE-01 | Concurrent snapshot handling | unit | `go test ./internal/services/... -run TestGenerateSnapshot_Concurrent` | ❌ Need to create |
| EDGE-02 | Recording interruption handling | integration | `go test ./internal/services/... -run TestGenerateSnapshot_Interrupted` | ❌ Need to create |

### Sampling Rate

- **Per task commit:** `go test ./internal/services/... -run TestSnapshot && npm test -- --testPathPattern=VideoPlayer`
- **Per wave merge:** `go test ./... && npm test`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/services/snapshot_service_test.go` — Test for incremental snapshot logic with offset tracking
- [ ] `internal/services/snapshot_service_test.go` — Test for concurrent snapshot mutex protection
- [ ] `internal/services/snapshot_service_test.go` — Test for recording interruption during snapshot
- [ ] `internal/services/snapshot_service_test.go` — Test for enhanced naming convention
- [ ] `frontend/src/components/__tests__/VideoPlayerModal.test.tsx` — Test for keyboard shortcuts
- [ ] `frontend/src/components/__tests__/VideoPlayerModal.test.tsx` — Test for frame-level navigation
- [ ] `frontend/src/hooks/__tests__/useKeyboardShortcuts.test.ts` — Test for keyboard shortcuts hook
- [ ] `frontend/src/hooks/__tests__/useVideoFrameNavigation.test.ts` — Test for frame navigation hook
- [ ] Frontend test setup: Verify `@testing-library/react` is installed — if none detected

*(If no gaps: "None — existing test infrastructure covers all phase requirements")*

## Security Domain

> Required when `security_enforcement` is enabled (absent = enabled).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V1 Architecture | yes | Service layer mutex prevents race conditions |
| V5 Input Validation | yes | Validate task status, duration, offset before FFmpeg operations |
| V6 Cryptography | no | N/A for this phase |
| V7 Error Handling | yes | Graceful handling of FFmpeg failures, recording interruptions |
| V8 Data Protection | yes | File paths sanitized, no directory traversal in snapshot filenames |

### Known Threat Patterns for Go + React Video Processing

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal in snapshot filename | Tampering | Sanitize task name with `strings.Map`, limit length, validate no `..` or `/` in output |
| Concurrent snapshot race condition | Tampering | Mutex per task, database row locking with `SELECT FOR UPDATE` |
| FFmpeg command injection | Tampering | Use `exec.Command` with argument array, never shell string concatenation |
| Recording interruption DoS | Denial of Service | Validate task status before snapshot, handle partial file errors gracefully |
| Keyboard shortcut XSS | Spoofing | React escapes JSX by default, avoid `dangerouslySetInnerHTML` |

## Sources

### Primary (HIGH confidence)

- [Codebase analysis] - `internal/services/snapshot_service.go` — Current snapshot implementation with incremental logic
- [Codebase analysis] - `internal/models/video_file.go` — VideoFile model with SnapshotOffset field
- [Codebase analysis] - `frontend/src/components/VideoPlayerModal.tsx` — Current video player implementation
- [Codebase analysis] - `frontend/src/components/TimelineWithMarkers.tsx` — Timeline marker component
- [Codebase analysis] - `frontend/src/components/EditableProgressBar.tsx` — Time input component
- [Phase 1 Context] - `.planning/phases/01-video-splitting/01-CONTEXT.md` — Decisions D-08 through D-15 for snapshot logic
- [Phase 1 Research] - `.planning/phases/01-video-splitting/01-RESEARCH.md` — FFmpeg patterns and pitfalls

### Secondary (MEDIUM confidence)

- [MDN Web Docs] - HTML5 video API documentation (`currentTime`, `playbackRate`, `requestVideoFrameCallback`)
- [React Documentation] - Event handling patterns, `useEffect` and `useCallback` hooks
- [FFmpeg Documentation] - `-ss`, `-to`, `-c copy` vs re-encode modes

### Tertiary (LOW confidence)

- [Web search results] - Video player keyboard shortcuts patterns (rate-limited during research, unable to verify)
- [Web search results] - FFmpeg snapshot incremental patterns (rate-limited during research, unable to verify)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All dependencies already in project, verified by codebase analysis
- Architecture: HIGH - Based on existing implementation patterns in codebase
- Pitfalls: MEDIUM - Some identified from code, others assumed from general knowledge (need verification)
- Video player enhancements: MEDIUM - Based on React/HTML5 video patterns, but browser support varies

**Research date:** 2026-04-20
**Valid until:** 2026-05-20 (30 days - stable technology domain)
