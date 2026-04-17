# Phase 1: Video Splitting - Research

**Researched:** 2026-04-17
**Domain:** Video processing, FFmpeg operations, React timeline UI, file management
**Confidence:** MEDIUM

## Summary

Phase 1 focuses on implementing video splitting capabilities, recording snapshots, and automatic file scanning. The phase requires extending the existing FFmpeg integration to support:
1. Multi-point video splitting with timeline markers (fast mode using `-c copy` with ±2s precision limitation, re-encode mode for frame accuracy)
2. MP4 snapshot generation during active recording without stopping the recording process
3. Automatic file scanning that triggers whenever new MP4 files are created (recording completion, snapshot, or splitting)
4. A new split page with an extended video player featuring timeline markers and segment management

The research reveals that FFmpeg's `-c copy` mode has inherent keyframe alignment limitations (±2s precision) which is a known technical constraint of H.264/H.265 codecs. The solution requires offering both fast (imprecise) and precise (re-encode) splitting modes.

**Primary recommendation:** Use the existing ConversionService worker pool pattern for split operations, extend VideoPlayerModal's Ant Design Slider into a marker-enabled timeline, and implement service callbacks for automatic file scanning.

## User Constraints (from CONTEXT.md)

### Locked Decisions

**Split Marker Interaction**
- D-01: Users add split markers by clicking on the video timeline OR by manually entering a timestamp in an input field — both methods are supported
- D-02: Markers can be repositioned by dragging along the timeline
- D-03: Precision is second-level only — no frame-level micro-adjustment or timeline zoom needed
- D-04: Markers display as vertical lines on the timeline with hover tooltip showing the timestamp; clicking a marker shows delete/edit actions

**Split Precision vs Speed**
- D-05: Default split uses FFmpeg `-c copy` mode (fast, lossless, potential ±2s keyframe misalignment)
- D-06: If split results are imprecise, user can re-run with re-encode mode for frame-accurate cuts
- D-07: The UI should communicate that fast mode may have slight imprecision at split points

**Recording Snapshot**
- D-08: "生成MP4快照" button appears inline on the active recording task row in the task list page
- D-09: After clicking, button text changes to "生成中..." and the system uses the existing notification system to alert on completion
- D-10: Snapshot MP4 file automatically appears in the file management list via service callback (same auto-scan mechanism as other MP4 generation)

**Segment Management & Auto Scan**
- D-11: Split segments are stored as independent VideoFile records with a `parent_id` field linking back to the source video
- D-12: Segments appear in the existing file list with an additional column showing "来源" (source: 录制/快照/分割) and a link to the original video
- D-13: Auto-scan uses service callbacks — recording service, conversion service, and splitting service call VideoFileService directly when new MP4 files are produced (no file system watching, no polling)
- D-14: Segments can be independently renamed, deleted, downloaded, and triggered for transcription

### Claude's Discretion

- Exact timeline marker component implementation (custom component vs extending existing Slider)
- FFmpeg command construction for split and snapshot operations
- Snapshot extraction technique (copy partial MKV vs dual-output FFmpeg)
- Database model additions (parent_id field on VideoFile, new migration)
- API endpoint design for split operations
- Segment naming convention (e.g., "原视频名_段落1.mp4")
- File storage paths for split segments
- How to handle split during recording vs split of completed video
- Re-encode mode UI trigger (per-segment "re-split with precision" button or global option)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SPLIT-01 | User can mark multiple split points on video player timeline | Ant Design Slider with custom marks/markers, React state for marker array |
| SPLIT-02 | User can preview video and precisely locate split points | Existing VideoPlayerModal with seek controls, add timestamp input field |
| SPLIT-03 | FFmpeg splits video by markers using -c copy mode | ConversionService pattern with FFmpeg `-ss` `-to` `-c copy` commands |
| SPLIT-04 | Split segments appear in list with independent management | VideoFile model with parent_id, source_type enum, separate CRUD operations |
| SPLIT-05 | Split segments can be individually transcribed | Segment records inherit transcription capability from parent VideoFile |
| SNAP-01 | Active recording shows "生成MP4快照" button | Task list inline button with loading state, API endpoint |
| SNAP-02 | Snapshot exports MP4 without stopping recording | FFmpeg copy partial MKV or dual-output technique, background job |
| SCAN-01 | New MP4 files automatically scanned into file management | Service callback pattern in RecordingService, ConversionService, SplittingService |
| SCAN-02 | File list updates in real-time without manual trigger | WebSocket or polling-based refresh after callback completion |
| UI-01 | Split page layout with video player + timeline + segment list | New page route, extended VideoPlayerModal with markers, segment Table component |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Video timeline marker rendering | Browser / Client | — | Pure UI interaction, React state management |
| Split marker state management | Browser / Client | — | Frontend maintains marker array during user interaction |
| FFmpeg split execution | API / Backend | — | CPU-intensive operation, requires server-side FFmpeg access |
| MP4 file generation (snapshot/split) | API / Backend | — | File system operations, FFmpeg processing |
| VideoFile database operations | API / Backend | — | Database persistence, transaction management |
| Auto-scan triggering | API / Backend | — | Service callbacks after file generation completion |
| File list real-time updates | Browser / Client | API / Backend | Frontend polls or receives WebSocket events |
| Recording snapshot button | Browser / Client | — | Inline UI component on task list page |
| Segment file metadata extraction | API / Backend | — | ffprobe execution, database record creation |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **FFmpeg** | Existing install | Video splitting, snapshot extraction | Already integrated for recording/conversion, proven reliability |
| **Go 1.24** | Project version | Backend service implementation | Consistent with existing codebase |
| **React 19** | Project version | Frontend UI components | Consistent with existing codebase |
| **Ant Design 6** | Project version | UI component library (Slider, Button, Table) | Already used in project, provides timeline Slider with marks |
| **GORM** | Project version | Database ORM for VideoFile model extensions | Existing database layer, supports migrations |
| **Gin** | Project version | HTTP API handlers for split/snapshot endpoints | Existing API framework |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **Zustand** | Project version | Frontend state management for split page | Existing state management pattern |
| **dayjs** | Project version | Timeline timestamp formatting | Already used in project for time display |
| **Zap Logger** | Project version | Structured logging for split operations | Existing logging infrastructure |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Ant Design Slider | Custom canvas timeline | More control but higher implementation cost, Slider is sufficient for second-level precision |
| Service callback auto-scan | File system watcher | More complex, OS-dependent, service callbacks are simpler and more reliable |
| FFmpeg -c copy split | Re-encode split by default | Fast by default, re-encode available as precision option |

**Installation:**
```bash
# No new packages required - all dependencies already in project
# Existing FFmpeg installation will be used
```

**Version verification:** All core libraries are already installed in the project. FFmpeg version should be verified to support `-c copy` and tee muxer (already in use for recording).

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (Browser)                       │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                 Video Split Page                          │  │
│  │  ┌─────────────┐  ┌─────────────────┐  ┌──────────────┐ │  │
│  │  │ VideoPlayer │->│ Timeline Slider │->│ Marker List  │ │  │
│  │  │ (extended)  │  │ (with marks)    │  │              │ │  │
│  │  └─────────────┘  └─────────────────┘  └──────────────┘ │  │
│  │           │                   │                   │       │  │
│  │           ▼                   ▼                   ▼       │  │
│  │  ┌─────────────────────────────────────────────────────┐ │  │
│  │  │            Segment Preview Table                     │ │  │
│  │  └─────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                  │
│                              ▼                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Task List Page (Snapshot)                     │  │
│  │  Active Recording Row → [生成MP4快照] Button               │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      API / Backend Layer                         │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              SplittingService (new)                        │  │
│  │  - SubmitSplit(videoID, markers[])                        │  │
│  │  - ExecuteSplit() → FFmpeg worker pool                    │  │
│  │  - Callback VideoFileService on completion                │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                   │                              │
│  ┌───────────────────────────────┼────────────────────────────┐ │
│  │                               │                            │ │
│  ▼                               ▼                            ▼ │
│  ┌─────────────────┐   ┌─────────────────┐   ┌──────────────────┐
│  │RecordingService │   │ConversionService│   │VideoFileService  │
│  │(existing)       │   │(existing)       │   │(extended)        │
│  │+ SnapshotAPI    │   │+ Callback on    │   │+ CreateSegment() │
│  │  (new endpoint) │   │  completion     │   │+ AutoScan logic  │
│  └─────────────────┘   └─────────────────┘   └──────────────────┘
│          │                       │                      │        │
│          └───────────────────────┴──────────────────────┘        │
│                                  │                               │
│                                  ▼                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    FFmpeg Operations                       │  │
│  │  - Split: ffmpeg -i input.mp4 -ss XX -to YY -c copy seg.mp4│  │
│  │  - Snapshot: Copy partial MKV / Dual-output tee           │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Database & Storage Layer                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              SQLite Database (GORM)                        │  │
│  │  video_files table + parent_id, source_type columns       │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                  │                               │
│                                  ▼                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              File System Storage                           │  │
│  │  /recordings/task_{id}/source.mp4                         │  │
│  │  /recordings/task_{id}/segments/segment_001.mp4           │  │
│  │  /recordings/task_{id}/snapshot_{timestamp}.mp4           │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/
├── services/
│   ├── splitting_service.go         # NEW: Split operations worker pool
│   ├── conversion_service.go        # EXTEND: Add VideoFileService callback
│   ├── recording_service.go         # EXTEND: Add snapshot endpoint
│   └── video_file_service.go        # EXTEND: CreateSegment(), source handling
├── handlers/
│   ├── split_handler.go             # NEW: Split API endpoints
│   └── video_file_handler.go        # EXTEND: Segment CRUD endpoints
├── models/
│   └── video_file.go                # EXTEND: ParentID, SourceType fields
└── migrations/
    └── 001_add_segment_fields.go    # NEW: Database migration

frontend/
├── src/
│   ├── pages/
│   │   ├── split/
│   │   │   └── index.tsx            # NEW: Split page with timeline
│   │   ├── tasks/
│   │   │   └── index.tsx            # EXTEND: Add snapshot button
│   │   └── files/
│   │       └── index.tsx            # EXTEND: Add source column
│   ├── components/
│   │   ├── VideoPlayerModal.tsx     # EXTEND: Timeline markers
│   │   └── TimelineWithMarkers.tsx  # NEW: Marker-enabled slider
│   └── api/
│       └── split.ts                 # NEW: Split API client
```

### Pattern 1: Worker Pool for Split Operations (from ConversionService)

**What:** Reuse the existing ConversionService worker pool pattern for split operations to ensure consistent error handling, retry logic, and process management.

**When to use:** Any long-running FFmpeg operation that needs to be cancellable and should not block API requests.

**Example:**
```go
// Source: internal/services/conversion_service.go (existing pattern)
type FFmpegConversionService struct {
    taskQueue        chan uint
    workers          int
    maxRetries       int
    cancelFuncs      map[uint]context.CancelFunc
    mu               sync.RWMutex
    wg               sync.WaitGroup
    ctx              context.Context
    cancel           context.CancelFunc
    ffmpegPath       string
    videoFileService *VideoFileService
}

// Apply same pattern to SplittingService
type SplittingService struct {
    splitQueue      chan SplitTask
    workers         int
    maxRetries      int
    cancelFuncs     map[uint]context.CancelFunc
    mu              sync.RWMutex
    wg              sync.WaitGroup
    ctx             context.Context
    cancel          context.CancelFunc
    ffmpegPath      string
    videoFileService *VideoFileService
}
```

### Pattern 2: Service Callback Pattern for Auto-Scan

**What:** Services call VideoFileService directly when new MP4 files are created, eliminating the need for file system watching or polling.

**When to use:** Any service that generates MP4 files as output (RecordingService, ConversionService, SplittingService).

**Example:**
```go
// In ConversionService, after successful MP4 conversion
func (s *FFmpegConversionService) processTask(taskID uint) {
    // ... FFmpeg conversion logic ...

    // Conversion successful - create MP4 file record via callback
    if s.videoFileService != nil {
        mp4 := "mp4"
        videoFile, err := s.videoFileService.CreateFileFromTask(&task, &mp4)
        if err != nil {
            s.logger.Error("创建MP4文件记录失败", zap.Uint("task_id", taskID), zap.Error(err))
        }
    }
}

// Same pattern in SplittingService after split completion
func (s *SplittingService) processSplit(splitTask *SplitTask) {
    // ... FFmpeg split logic ...

    // Split successful - create segment file records via callback
    for i, segmentPath := range segmentPaths {
        videoFile, err := s.videoFileService.CreateSegmentFile(
            segmentPath,
            &splitTask.SourceVideoID,
            "split",
            splitTask.CreatedBy,
        )
        // ... error handling ...
    }
}
```

### Pattern 3: React State Management for Timeline Markers

**What:** Use React useState to maintain an array of marker timestamps, render them as Ant Design Slider marks, and allow click-to-add and drag-to-reposition.

**When to use:** Interactive timeline UI where users need to mark multiple points.

**Example:**
```typescript
// Frontend marker state management
const [markers, setMarkers] = useState<number[]>([]) // Array of timestamps in seconds

// Add marker on timeline click
const handleTimelineClick = (timestamp: number) => {
  setMarkers([...markers, timestamp].sort((a, b) => a - b))
}

// Remove marker
const handleMarkerRemove = (timestamp: number) => {
  setMarkers(markers.filter(m => m !== timestamp))
}

// Render as Slider marks
const marks = markers.reduce((acc, marker) => {
  acc[marker] = formatTime(marker)
  return acc
}, {} as Record<number, string>)
```

### Anti-Patterns to Avoid

- **File system watching for auto-scan**: Unreliable across different OS, high resource usage, race conditions. Use service callbacks instead.
- **Blocking FFmpeg operations in HTTP handlers**: Prevents request cancellation, causes timeout issues. Use worker pool pattern.
- **Storing markers only in frontend state**: Lost on page refresh. Should persist to database before split execution.
- **Re-encoding by default for splits**: Slow and unnecessary for most use cases. Default to `-c copy`, offer re-encode as precision option.
- **Polling for file list updates**: Inefficient and chatty. Use WebSocket or trigger refresh only after operations complete.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| FFmpeg command construction | Manual string concatenation with escaping issues | Existing patterns from `coordinator.go` (escapePathForTeeMuxer, normalizePathForFFmpeg) | Path escaping is complex, especially on Windows with tee muxer |
| Worker pool for async operations | Custom goroutine management | Copy ConversionService pattern (worker pool with cancellable contexts) | Proven reliability, consistent error handling, retry logic |
| Timeline UI component | Custom canvas implementation | Extend Ant Design Slider with marks property | Already in project, sufficient for second-level precision |
| API request authentication | Custom fetch wrappers | Existing apiRequest<T>() from apiClient.ts | Handles token refresh, 401 errors, retry logic |
| Database transactions | Manual transaction handling | GORM's db.Transaction() wrapper | Automatic rollback on error, consistent with existing code |

**Key insight:** The existing codebase has proven patterns for FFmpeg integration, worker pools, and API handling. Reusing these patterns reduces risk and implementation time.

## Runtime State Inventory

> **Skip for this phase** - Phase 1 is greenfield (new features), not a rename/refactor/migration phase. No runtime state migration required.

## Common Pitfalls

### Pitfall 1: FFmpeg Keyframe Misalignment with `-c copy`

**What goes wrong:** User expects frame-accurate splits, but `-c copy` mode can only split on keyframe boundaries (I-frames), resulting in ±2s imprecision.

**Why it happens:** H.264/H.265 codecs use inter-frame compression where most frames depend on previous frames. FFmpeg can only start a new segment at a keyframe without re-encoding.

**How to avoid:**
1. Document the limitation clearly in the UI (D-07: "快速分割模式可能有±2秒误差")
2. Offer re-encode mode as a precision option (D-06)
3. Show actual split start/end times after fast mode completes
4. Consider auto-adjusting markers to nearest keyframe before splitting

**Warning signs:** User complaints that splits don't start/end at the exact timestamp they marked.

### Pitfall 2: Snapshot Interrupting Recording

**What goes wrong:** Attempting to copy or process the MKV file during active recording causes the recording to fail or corrupts the output.

**Why it happens:** The MKV file is being actively written by the FFmpeg recording process. File locking or concurrent access causes issues.

**How to avoid:**
1. **Copy partial MKV technique:** Copy the file to a temporary location first, then process the copy
2. **Dual-output tee muxer:** Already using tee muxer for MKV+HLS, could add third output for snapshots
3. **FFmpeg segment technique:** Use `-f segment` to create periodic snapshots during recording
4. Test thoroughly with long recordings to ensure no interruption

**Warning signs:** Recording stops unexpectedly after snapshot button is clicked, snapshot file is corrupted.

### Pitfall 3: Orphaned Files in Database

**What goes wrong:** MP4 files are created on disk but VideoFile records are not created due to service crash, callback failure, or transaction error.

**Why it happens:** File system operations and database operations are not atomic. If the callback fails after file creation, the record is never created.

**How to avoid:**
1. Use transactions for database operations
2. Implement retry logic in service callbacks
3. Create VideoFile record BEFORE file operations where possible
4. Add robust error logging for callback failures
5. Provide manual "扫描导入" button to recover orphaned files (already exists in file list page)

**Warning signs:** Files exist in recordings directory but don't appear in file management page.

### Pitfall 4: Frontend State Desync

**What goes wrong:** Split markers are added/removed in UI but backend API uses stale data, or split executes with different markers than shown.

**Why it happens:** Race condition between user interactions and API calls, or state not properly synchronized between components.

**How to avoid:**
1. Send marker array directly in split API request (don't rely on backend state)
2. Validate markers on backend (sort, deduplicate, range check)
3. Show loading state during split execution to prevent further modifications
4. Return actual split timestamps in API response to update frontend

**Warning signs:** User reports splits happened at wrong times, or marker count doesn't match segment count.

### Pitfall 5: Segment File Path Conflicts

**What goes wrong:** Multiple splits of the same video create segments with identical filenames, causing overwrites or database constraint violations.

**Why it happens:** Insufficient uniqueness in segment filename generation (e.g., using only segment index without timestamp or split ID).

**How to avoid:**
1. Include timestamp in segment filenames: `{source_name}_seg{index}_{timestamp}.mp4`
2. Use subdirectories: `/recordings/task_{id}/segments/`
3. Add unique constraint check before creating file records
4. Handle file_exists error gracefully by auto-renaming

**Warning signs:** Database errors on unique constraint violation, segments disappearing after second split.

## Code Examples

### FFmpeg Split Command (Fast Mode - `-c copy`)

```go
// Source: Based on existing FFmpeg patterns in coordinator.go
// Split video at marker timestamps using stream copy (fast, imprecise)
func buildSplitCommand(inputPath string, markers []time.Duration, outputDir string) []string {
    args := []string{
        "-y", // Overwrite output files
        "-i", inputPath,
    }

    // Generate segments for each marker interval
    // markers: [10s, 30s, 50s] → segments: [0-10s, 10-30s, 30-50s, 50s-end]
    for i, marker := range markers {
        startTime := marker
        var endTime time.Duration
        if i < len(markers)-1 {
            endTime = markers[i+1]
        }

        outputPath := filepath.Join(outputDir, fmt.Sprintf("segment_%03d.mp4", i))

        segmentArgs := []string{
            "-ss", formatDuration(startTime),
        }
        if endTime > 0 {
            segmentArgs = append(segmentArgs, "-to", formatDuration(endTime))
        }
        segmentArgs = append(segmentArgs,
            "-c", "copy", // Stream copy mode (fast)
            "-avoid_negative_ts", "1",
            outputPath,
        )

        args = append(args, segmentArgs...)
    }

    return args
}
```

### React Timeline with Markers

```typescript
// Source: Based on existing VideoPlayerModal.tsx Slider implementation
import { Slider } from 'antd'

function TimelineWithMarkers({
  duration,
  markers,
  onMarkerAdd,
  onMarkerRemove,
  currentTime,
  onSeek
}: TimelineProps) {
  const marks = useMemo(() => {
    return markers.reduce((acc, marker, index) => {
      acc[marker] = {
        style: { color: '#1890ff' },
        label: formatTime(marker)
      }
      return acc
    }, {} as Record<number, { style: React.CSSProperties; label: string }>)
  }, [markers])

  const handleSliderChange = (value: number) => {
    onSeek(value)
  }

  const handleSliderClick = (e: React.MouseEvent) => {
    // Calculate timestamp from click position
    const slider = e.currentTarget
    const rect = slider.getBoundingClientRect()
    const clickX = e.clientX - rect.left
    const ratio = clickX / rect.width
    const timestamp = ratio * duration

    // Check if close to existing marker (toggle)
    const existingMarker = markers.find(m => Math.abs(m - timestamp) < 1)
    if (existingMarker) {
      onMarkerRemove(existingMarker)
    } else {
      onMarkerAdd(timestamp)
    }
  }

  return (
    <div onClick={handleSliderClick}>
      <Slider
        min={0}
        max={duration}
        value={currentTime}
        onChange={handleSliderChange}
        marks={marks}
        tooltip={{ formatter: (value) => formatTime(value) }}
        trackStyle={{ backgroundColor: '#1890ff' }}
      />
    </div>
  )
}
```

### Service Callback Pattern

```go
// Source: Extension of existing ConversionService pattern
func (s *FFmpegConversionService) processTask(taskID uint) {
    // ... FFmpeg conversion logic ...

    // Conversion successful - trigger VideoFileService callback
    if s.videoFileService != nil {
        mp4Format := "mp4"
        videoFile, err := s.videoFileService.CreateFileFromTask(&task, &mp4Format)
        if err != nil {
            s.logger.Error("创建MP4文件记录失败",
                zap.Uint("task_id", taskID),
                zap.Error(err),
            )
        } else {
            s.logger.Info("MP4文件已自动扫描",
                zap.Uint("file_id", videoFile.ID),
                zap.String("file_path", videoFile.FilePath),
            )
        }
    }
}

// Same pattern for SplittingService
func (s *SplittingService) processSplit(splitTask *SplitTask) error {
    // ... FFmpeg split logic ...

    // Split successful - create segment records via callback
    for i, segmentPath := range s.segmentPaths {
        segmentFile, err := s.videoFileService.CreateSegmentFile(
            segmentPath,
            &splitTask.SourceVideoID,
            "split", // source_type
            splitTask.CreatedBy,
        )
        if err != nil {
            s.logger.Error("创建segment文件记录失败",
                zap.Int("segment", i),
                zap.String("path", segmentPath),
                zap.Error(err),
            )
            continue
        }
        s.logger.Info("Segment文件已自动扫描",
            zap.Uint("segment_id", segmentFile.ID),
            zap.String("file_path", segmentFile.FilePath),
        )
    }

    return nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual file scanning | Service callback auto-scan | Phase 1 | Instant file appearance, no manual scan needed |
| Single split point | Multi-point timeline markers | Phase 1 | Efficient workflow, mark all points then split once |
| Only post-recording splits | In-progress recording snapshots | Phase 1 | Export partial recordings without stopping |
| Fast split only | Fast + re-encode precision mode | Phase 1 | User choice between speed and accuracy |

**Deprecated/outdated:**
- **File system polling for new files**: Replaced by service callbacks
- **Single-marker split UI**: Replaced by multi-marker timeline
- **Blocking FFmpeg in HTTP handlers**: Replaced by worker pool pattern

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Ant Design Slider marks property supports dynamic marker addition/removal | Architecture Patterns | May need custom timeline component if marks property insufficient |
| A2 | Copy partial MKV during recording does not interrupt FFmpeg process | Recording Snapshot | Snapshot may corrupt recording or cause FFmpeg to crash |
| A3 | Service callbacks are sufficient for real-time file list updates | Auto Scan | May need WebSocket if users don't see files immediately |
| A4 | FFmpeg `-c copy` split with multiple segments can be done in single command | Code Examples | May require multiple FFmpeg invocations, affecting performance |
| A5 | React 19 and Ant Design 6 are compatible (no breaking changes) | Standard Stack | May encounter UI component compatibility issues |

## Open Questions

1. **Snapshot technique selection**
   - What we know: Three potential approaches (copy partial MKV, dual-output tee, segment output)
   - What's unclear: Which approach works reliably without interrupting ongoing recording
   - Recommendation: Prototype and test all three approaches with 30+ minute recordings to verify no interruption

2. **Timeline marker drag-to-reposition implementation**
   - What we know: Ant Design Slider doesn't natively support draggable marks
   - What's unclear: Whether custom drag implementation is needed or if there's a simpler alternative
   - Recommendation: Investigate if clicking a marker opens a modal with timestamp input (simpler) vs implementing full drag-and-drop

3. **Real-time file list update mechanism**
   - What we know: Service callbacks create database records, but frontend needs to know about updates
   - What's unclear: Whether WebSocket infrastructure exists or if polling is acceptable
   - Recommendation: Check if project has WebSocket support, otherwise implement 5-second polling on file list page

4. **Segment file storage organization**
   - What we know: Segments need unique names and organized storage
   - What's unclear: Whether to use `/recordings/task_{id}/segments/` subdirectory or store alongside source video
   - Recommendation: Use subdirectory for better organization, easier cleanup

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| FFmpeg | Split/snapshot operations | ✓ | Existing installation | — |
| Go 1.24 | Backend services | ✓ | Project version | — |
| React 19 | Frontend UI | ✓ | Project version | — |
| Ant Design 6 | UI components | ✓ | Project version | — |
| GORM | Database operations | ✓ | Project version | — |
| SQLite | Database storage | ✓ | Project version | — |

**Missing dependencies with no fallback:** None

**Missing dependencies with fallback:** None

All required dependencies are already available in the project environment.

## Validation Architecture

> **Skip condition:** No explicit `workflow.nyquist_validation` setting found in `.planning/config.json` (file doesn't exist). Defaulting to including validation architecture.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | None detected - need to verify if testing infrastructure exists |
| Config file | TBD - check for pytest.ini, jest.config.*, vitest.config.* |
| Quick run command | TBD - needs investigation |
| Full suite command | TBD - needs investigation |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SPLIT-01 | User can mark multiple split points on timeline | E2E | ❌ Manual-only (UI interaction) | N/A |
| SPLIT-02 | User can preview and locate split points | E2E | ❌ Manual-only (video playback) | N/A |
| SPLIT-03 | FFmpeg splits video by markers | Integration | `go test ./internal/services/... -run TestSplitService -v` | ❌ Wave 0 |
| SPLIT-04 | Segments appear in list with management | Integration | `go test ./internal/handlers/... -run TestSplitHandler -v` | ❌ Wave 0 |
| SPLIT-05 | Segments can be transcribed | Integration | `go test ./internal/services/... -run TestSegmentTranscription -v` | ❌ Wave 0 |
| SNAP-01 | Snapshot button appears on active recording | E2E | ❌ Manual-only (UI visibility) | N/A |
| SNAP-02 | Snapshot exports MP4 without stopping recording | Integration | `go test ./internal/services/... -run TestSnapshot -v` | ❌ Wave 0 |
| SCAN-01 | New MP4 files auto-scanned | Integration | `go test ./internal/services/... -run TestAutoScan -v` | ❌ Wave 0 |
| SCAN-02 | File list updates in real-time | E2E | ❌ Manual-only (UI refresh) | N/A |
| UI-01 | Split page layout | E2E | ❌ Manual-only (visual) | N/A |

### Sampling Rate

- **Per task commit:** `go test ./... -short` (quick smoke test)
- **Per wave merge:** `go test ./... -v` (full suite)
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/services/splitting_service_test.go` — covers SPLIT-03, SNAP-02
- [ ] `internal/handlers/split_handler_test.go` — covers SPLIT-04
- [ ] `internal/services/video_file_service_test.go` — extends for SCAN-01, auto-scan callbacks
- [ ] `frontend/src/components/__tests__/TimelineWithMarkers.test.tsx` — covers SPLIT-01 (unit test for component logic)
- [ ] Test framework setup: Verify if Go testing framework exists, add `go.mod` test dependencies if needed
- [ ] Integration test helpers: Mock FFmpeg execution, test database fixtures

**Priority:** High - Need to establish testing infrastructure before implementation begins.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | N/A (uses existing auth system) |
| V3 Session Management | No | N/A (uses existing session system) |
| V4 Access Control | Yes | Existing permission guards (PERMISSIONS.FILE_SPLIT, PERMISSIONS.RECORDING_SNAPSHOT) |
| V5 Input Validation | Yes | Marker timestamp validation (range check, deduplication, sort), file path sanitization |
| V6 Cryptography | No | N/A (no encryption in this phase) |

### Known Threat Patterns for Go FFmpeg Video Processing

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| FFmpeg command injection | Tampering | Use structured command construction (exec.CommandContext), never string concatenation for user input |
| Path traversal in segment filenames | Tampering | Validate/sanitize file paths, use filepath.Join(), reject paths with `..` |
| Resource exhaustion (CPU/disk) | Denial of Service | Worker pool limits, max concurrent splits, disk space checks before operations |
| Unauthorized file access | Spoofing | Verify user owns source video before allowing split, check parent_id relationships |
| Database record orphaning | Information Disclosure | Use transactions for file+record creation, implement cleanup jobs |

**Key security considerations:**
- All FFmpeg commands must use `exec.CommandContext` with array arguments (no shell injection)
- File paths must be validated and sanitized before use
- Split operations must verify user permission on source video (created_by check or admin role)
- Worker pool must enforce resource limits to prevent DoS

## Sources

### Primary (HIGH confidence)

- **Existing codebase analysis:**
  - `internal/recorder/coordinator.go` — FFmpeg integration patterns, path escaping, tee muxer usage
  - `internal/services/conversion_service.go` — Worker pool pattern, error handling, retry logic
  - `internal/services/video_file_service.go` — File scanning, metadata extraction, database operations
  - `internal/models/video_file.go` — VideoFile model structure, fields
  - `frontend/src/components/VideoPlayerModal.tsx` — Video player with seek slider
  - `frontend/src/pages/files/index.tsx` — File list page structure
  - `frontend/src/pages/tasks/index.tsx` — Task list page structure
  - `frontend/src/api/apiClient.ts` — API client patterns

### Secondary (MEDIUM confidence)

- **[Stack Overflow: How to split video so each chunk starts with a keyframe](https://stackoverflow.com/questions/14005110/how-to-split-a-video-using-ffmpeg-so-that-each-chunk-starts-with-a-key-frame)** — FFmpeg split techniques and keyframe alignment
- **[Video Stack Exchange: Using ffmpeg to cut videos with more precision than key frames allow](https://video.stackexchange.com/questions/16750/using-ffmpeg-to-cut-videos-with-more-precision-than-key-frames-allow)** — Confirmation that `-c copy` has precision limitations
- **[Ant Design Slider Documentation](https://ant.design/components/slider/)** — Slider component API, marks property, customization options
- **[Build a Custom Time Slider Component with Ant Design and Next.js](https://www.paigeniedringhaus.com/blog/build-a-custom-time-slider-component-with-ant-design-and-nextjs/)** — Custom time slider implementation patterns

### Tertiary (LOW confidence)

- **[React Video Timeline Slider examples](https://codesandbox.io/examples/package/react-video-timelines-slider)** — Timeline component patterns (not verified)
- **[RVE - React Video Editor Timeline](https://www.reactvideoeditor.com/features/timeline)** — Video editor timeline features (not verified)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All dependencies already verified in existing codebase
- Architecture: MEDIUM - Patterns based on existing codebase, but new components (SplittingService, TimelineWithMarkers) need implementation
- Pitfalls: MEDIUM - Keyframe alignment and snapshot interruption risks are documented but need prototyping verification

**Research date:** 2026-04-17
**Valid until:** 2026-05-17 (30 days - tech stack is stable, but FFmpeg techniques may have updates)
