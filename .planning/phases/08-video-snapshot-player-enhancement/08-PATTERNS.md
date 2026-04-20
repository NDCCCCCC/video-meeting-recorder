# Phase 8: Video Snapshot & Player Enhancement - Pattern Map

**Mapped:** 2026-04-20
**Files analyzed:** 8
**Analogs found:** 7 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/snapshot_service.go` | service | CRUD | `internal/services/snapshot_service.go` | exact (self-enhancement) |
| `frontend/src/components/VideoPlayerModal.tsx` | component | request-response | `frontend/src/components/VideoPlayerModal.tsx` | exact (self-enhancement) |
| `frontend/src/hooks/useKeyboardShortcuts.ts` | hook | event-driven | `frontend/src/stores/authStore.ts` | pattern-match |
| `frontend/src/hooks/useVideoFrameNavigation.ts` | hook | request-response | `frontend/src/components/PlaybackSpeedControl.tsx` | role-match |
| `frontend/src/utils/videoPlayerHotkeys.ts` | utility | static | `frontend/src/utils/permissions.ts` | role-match |
| `frontend/src/components/FrameNavigation.tsx` | component | request-response | `frontend/src/components/PlaybackSpeedControl.tsx` | role-match |
| `internal/services/snapshot_service_test.go` | test | CRUD | `internal/handlers/split_handler_test.go` | role-match |
| `frontend/src/hooks/__tests__/useKeyboardShortcuts.test.ts` | test | event-driven | `frontend/src/pages/files/__tests__/TranscriptionDropdown.test.ts` | pattern-match |

## Pattern Assignments

### `internal/services/snapshot_service.go` (service, CRUD)

**Analog:** `internal/services/snapshot_service.go` (self-enhancement)

**Service struct pattern** (lines 17-23):
```go
type SnapshotService struct {
    db               *gorm.DB
    logger           *zap.Logger
    config           *config.Config
    ffmpegPath       string
    videoFileService *VideoFileService
}
```

**Constructor pattern** (lines 25-36):
```go
func NewSnapshotService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, videoFileService *VideoFileService) *SnapshotService {
    ffmpegPath := cfg.FFmpeg.Path
    if ffmpegPath == "" {
        ffmpegPath = "ffmpeg"
    }
    return &SnapshotService{
        db:               db,
        logger:           logger,
        config:           cfg,
        ffmpegPath:       ffmpegPath,
        videoFileService: videoFileService,
    }
}
```

**Incremental snapshot logic** (lines 62-77):
```go
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
```

**FFmpeg command building pattern** (lines 137-150):
```go
// Build FFmpeg args with -ss for incremental seeking
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

cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
```

**Error handling pattern** (lines 44-52):
```go
// 1. Load recording task
var task models.VideoRecordingTask
if err := s.db.First(&task, taskID).Error; err != nil {
    return nil, fmt.Errorf("录制任务不存在: %w", err)
}

// 2. Verify task is recording
if task.Status != models.VideoStatusRecording {
    return nil, fmt.Errorf("任务不在录制状态，无法生成快照")
}
```

**Validation pattern** (lines 95-103):
```go
// Validate recording duration is positive
if recordingDuration <= 0 {
    return nil, fmt.Errorf("录制时长无效: %.0f秒，无法生成快照", recordingDuration)
}

// Validate seek offset doesn't exceed recording duration
if seekOffset >= recordingDuration {
    return nil, fmt.Errorf("快照偏移量 %.2f 秒超过或等于录制时长 %.2f 秒", seekOffset, recordingDuration)
}
```

---

### `frontend/src/components/VideoPlayerModal.tsx` (component, request-response)

**Analog:** `frontend/src/components/VideoPlayerModal.tsx` (self-enhancement)

**Imports pattern** (lines 1-19):
```typescript
import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Modal, Button, Space, Alert, message, Slider } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  StepForwardOutlined,
  StepBackwardOutlined,
  DownloadOutlined,
  SoundOutlined,
  FullscreenOutlined,
} from '@ant-design/icons'
import type { VideoFile } from '../types/video-file'
import { getToken } from '../api/apiClient'
```

**Constants pattern** (lines 17-19):
```typescript
const PLAYBACK_RATES: readonly number[] = [0.5, 1, 1.25, 1.5, 2]
const SKIP_SECONDS = 10
```

**Style constants pattern** (lines 22-71):
```typescript
const STYLES = {
  container: {
    position: 'relative',
    backgroundColor: '#000',
    borderRadius: '8px',
    overflow: 'hidden',
  } as const,
  video: {
    width: '100%',
    maxHeight: '500px',
    display: 'block',
  } as const,
  controlBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    background: 'linear-gradient(transparent, rgba(0,0,0,0.8))',
    padding: '12px 16px',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  } as const,
} as const
```

**Video reference pattern** (lines 126-127):
```typescript
const videoRef = useRef<HTMLVideoElement>(null)
const containerRef = useRef<HTMLDivElement>(null)
```

**State management pattern** (lines 129-136):
```typescript
const [isPlaying, setIsPlaying] = useState(false)
const [currentTime, setCurrentTime] = useState(0)
const [duration, setDuration] = useState(0)
const [volume, setVolume] = useState(1)
const [playbackRate, setPlaybackRate] = useState(1)
const [error, setError] = useState<string>()
const [loading, setLoading] = useState(true)
```

**Control callback pattern** (lines 149-191):
```typescript
const handlePlayPause = useCallback(() => {
  const video = videoRef.current
  if (!video) return

  if (isPlaying) {
    video.pause()
  } else {
    video.play().catch(() => {
      message.error('播放失败，请稍后重试')
    })
  }
}, [isPlaying])

const handleSkip = useCallback((seconds: number) => {
  const video = videoRef.current
  if (!video || !duration) return
  video.currentTime = Math.max(0, Math.min(duration, video.currentTime + seconds))
}, [duration])

const handleSeek = useCallback((value: number) => {
  const video = videoRef.current
  if (!video) return
  video.currentTime = value
}, [])
```

**Effect for video metadata pattern** (lines 243-267):
```typescript
useEffect(() => {
  if (!visible) return

  const video = videoRef.current
  if (!video) return

  // 检查视频是否已经加载（处理浏览器缓存）
  const checkVideoLoaded = () => {
    if (video.readyState >= 1) { // HAVE_METADATA
      setDuration(video.duration)
      setLoading(false)
    }
  }

  // 延迟检查，给视频元素一些时间加载
  const timer = setTimeout(checkVideoLoaded, 100)

  // 如果视频已经加载，立即检查
  if (video.readyState >= 1) {
    clearTimeout(timer)
    checkVideoLoaded()
  }

  return () => clearTimeout(timer)
}, [visible])
```

**Video event handlers pattern** (lines 321-336):
```typescript
<video
  ref={videoRef}
  src={videoUrl}
  style={STYLES.video}
  preload="metadata"
  onLoadedMetadata={() => {
    const video = videoRef.current
    if (video) {
      setDuration(video.duration)
      setLoading(false)
    }
  }}
  onError={() => {
    setError('视频加载失败，请检查文件是否存在或稍后重试')
    setLoading(false)
  }}
  onTimeUpdate={() => {
    const video = videoRef.current
    if (video) setCurrentTime(video.currentTime)
  }}
/>
```

---

### `frontend/src/hooks/useKeyboardShortcuts.ts` (hook, event-driven)

**Analog:** `frontend/src/stores/authStore.ts` (Zustand store pattern)

**Hook interface pattern** (inspired by authStore):
```typescript
interface UseKeyboardShortcutsProps {
  videoRef: React.RefObject<HTMLVideoElement>
  isPlaying: boolean
  playbackRate: number
  volume: number
  enabled: boolean
  onPlayPause: () => void
  onSeek: (seconds: number) => void
  onVolumeChange: (volume: number) => void
  onPlaybackRateChange: (rate: number) => void
  onFullscreen: () => void
  onMuteToggle: () => void
}
```

**Custom hook pattern** (from PlaybackSpeedControl):
```typescript
export function usePlaybackSpeed(videoRef: React.RefObject<HTMLVideoElement | null>) {
  const [playbackRate, setPlaybackRate] = useState(1.0)

  const changeSpeed = useCallback((rate: number) => {
    const video = videoRef.current
    if (video) {
      video.playbackRate = rate
      setPlaybackRate(rate)
    }
  }, [videoRef])

  return { playbackRate, changeSpeed }
}
```

**Event listener setup pattern** (from VideoPlayerModal):
```typescript
useEffect(() => {
  const handleKeyDown = (event: KeyboardEvent) => {
    // Event handling logic
  }

  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [dependencies])
```

**Input element guard pattern** (from RESEARCH.md Example 3):
```typescript
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
  
  // Handle shortcuts
}, [enabled])
```

---

### `frontend/src/hooks/useVideoFrameNavigation.ts` (hook, request-response)

**Analog:** `frontend/src/components/PlaybackSpeedControl.tsx` (custom hook pattern)

**Hook pattern** (lines 19-42):
```typescript
export function usePlaybackSpeed(videoRef: React.RefObject<HTMLVideoElement | null>) {
  const [playbackRate, setPlaybackRate] = useState(1.0)

  const changeSpeed = useCallback((rate: number) => {
    const video = videoRef.current
    if (video) {
      video.playbackRate = rate
      setPlaybackRate(rate)
    }
  }, [videoRef])

  // Re-apply speed after video events (seek, load, etc.)
  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    const handleRateChange = () => setPlaybackRate(video.playbackRate)

    video.addEventListener('ratechange', handleRateChange)
    return () => video.removeEventListener('ratechange', handleRateChange)
  }, [videoRef])

  return { playbackRate, changeSpeed }
}
```

**Frame navigation pattern** (from RESEARCH.md Example 3):
```typescript
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

---

### `frontend/src/utils/videoPlayerHotkeys.ts` (utility, static)

**Analog:** `frontend/src/utils/permissions.ts` (constants + utility functions)

**Constants export pattern** (lines 2-49):
```typescript
export const PERMISSIONS = {
  DASHBOARD_VIEW: 'dashboard:view',
  TASK_VIEW: 'tasks:view',
  TASK_CREATE: 'tasks:create',
  // ... more permissions
} as const
```

**Type-safe interface pattern** (lines 82-87):
```typescript
export function hasPermission(user: User | null, permission: string): boolean {
  if (!user) return false
  // 管理员拥有所有权限
  if (user.is_admin) return true
  return user.permissions?.includes(permission) ?? false
}
```

**Keyboard shortcut constants pattern** (from RESEARCH.md Example 3):
```typescript
export const KEYBOARD_SHORTCUTS = {
  PLAY_PAUSE: { key: ' ', description: '播放/暂停' },
  SEEK_BACK_10: { key: 'ArrowLeft', description: '快退10秒' },
  SEEK_FORWARD_10: { key: 'ArrowRight', description: '快进10秒' },
  VOLUME_UP: { key: 'ArrowUp', description: '音量+10%' },
  VOLUME_DOWN: { key: 'ArrowDown', description: '音量-10%' },
  SEEK_BACK_1: { key: 'j', shiftKey: true, description: '快退1秒' },
  SEEK_FORWARD_1: { key: 'l', shiftKey: true, description: '快进1秒' },
  MUTE: { key: 'm', description: '静音/取消静音' },
  FULLSCREEN: { key: 'f', description: '全屏' },
  SPEED_UP: { key: '>', shiftKey: true, description: '播放速度+' },
  SPEED_DOWN: { key: '<', shiftKey: true, description: '播放速度-' },
  SEEK_TO_START: { key: 'Home', description: '跳转到开始' },
  SEEK_TO_END: { key: 'End', description: '跳转到结束' },
} as const
```

---

### `frontend/src/components/FrameNavigation.tsx` (component, request-response)

**Analog:** `frontend/src/components/PlaybackSpeedControl.tsx` (control component)

**Component interface pattern** (lines 46-50):
```typescript
interface PlaybackSpeedControlProps {
  currentSpeed: number
  onSpeedChange: (speed: number) => void
  style?: React.CSSProperties
}
```

**Component structure pattern** (lines 54-69):
```typescript
export function PlaybackSpeedControl({
  currentSpeed,
  onSpeedChange,
  style
}: PlaybackSpeedControlProps) {
  return (
    <Select
      value={currentSpeed}
      onChange={onSpeedChange}
      options={SPEED_OPTIONS}
      style={{ width: 80, ...style }}
      size="small"
      prefix={<DashboardOutlined />}
    />
  )
}
```

**Button control pattern** (from VideoPlayerModal lines 93-114):
```typescript
function ControlButton({
  icon,
  onClick,
  title,
  disabled = false,
}: {
  icon: React.ReactNode
  onClick: () => void
  title: string
  disabled?: boolean
}) {
  return (
    <Button
      type="text"
      icon={icon}
      onClick={onClick}
      title={title}
      disabled={disabled}
      style={STYLES.controlBtn}
    />
  )
}
```

**Frame navigation component pattern** (from RESEARCH.md Example 4):
```typescript
import { Button, Space, Tooltip } from 'antd'
import { StepBackwardOutlined, StepForwardOutlined, UndoOutlined } from '@ant-design/icons'

interface FrameNavigationProps {
  videoRef: React.RefObject<HTMLVideoElement>
  disabled?: boolean
}

export function FrameNavigation({ videoRef, disabled }: FrameNavigationProps) {
  const { nextFrame, prevFrame, supportsFrameCallback } = useVideoFrameNavigation(videoRef)

  if (!supportsFrameCallback()) {
    return null
  }

  return (
    <Space size="small">
      <Tooltip title="上一帧 (Shift+←)">
        <Button type="text" icon={<UndoOutlined />} onClick={prevFrame} disabled={disabled} size="small">
          -1帧
        </Button>
      </Tooltip>
      <Tooltip title="下一帧 (Shift+→)">
        <Button type="text" icon={<StepForwardOutlined />} onClick={nextFrame} disabled={disabled} size="small">
          +1帧
        </Button>
      </Tooltip>
    </Space>
  )
}
```

---

### `internal/services/snapshot_service_test.go` (test, CRUD)

**Analog:** `internal/handlers/split_handler_test.go`

**Go test structure pattern**:
```go
package services

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestGenerateSnapshot_Incremental(t *testing.T) {
    // Setup test database and service
    // Test incremental snapshot logic
}
```

**Table-driven test pattern** (common in Go):
```go
func TestGenerateSnapshot_Validation(t *testing.T) {
    tests := []struct {
        name        string
        taskID      uint
        createdBy   uint
        setup       func(*gorm.DB)
        wantErr     bool
        errContains string
    }{
        {
            name:      "task not found",
            taskID:    999,
            createdBy: 1,
            wantErr:   true,
        },
        {
            name:        "task not recording",
            taskID:      1,
            createdBy:   1,
            wantErr:     true,
            errContains: "不在录制状态",
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

---

### `frontend/src/hooks/__tests__/useKeyboardShortcuts.test.ts` (test, event-driven)

**Analog:** `frontend/src/pages/files/__tests__/TranscriptionDropdown.test.ts`

**Type-level test pattern** (lines 1-28):
```typescript
/**
 * Type-level tests for keyboard shortcuts hook
 */
import { renderHook, act } from '@testing-library/react'

// Test: Hook initializes correctly
const { result } = renderHook(() => useKeyboardShortcuts({...}))
if (!result.current) throw new Error('Hook should return value')

// Test: Keyboard event handling
const mockVideo = {
  currentTime: 0,
  duration: 100,
  play: vi.fn(),
  pause: vi.fn(),
} as unknown as HTMLVideoElement
```

**Event simulation pattern**:
```typescript
// Test: Space key toggles play/pause
const event = new KeyboardEvent('keydown', { key: ' ' })
Object.defineProperty(event, 'target', { value: document.body })
window.dispatchEvent(event)

// Assert play/pause was called
```

---

## Shared Patterns

### Time Formatting Utility
**Source:** `frontend/src/components/VideoPlayerModal.tsx` (lines 76-90)
**Apply to:** All components displaying video time
```typescript
function formatTime(seconds: number): string {
  if (!seconds || !Number.isFinite(seconds)) return '0:00'

  const s = Math.floor(seconds)
  const hours = Math.floor(s / 3600)
  const minutes = Math.floor((s % 3600) / 60)
  const secs = s % 60

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}
```

### Video Reference Management
**Source:** `frontend/src/components/VideoPlayerModal.tsx` (lines 126-127)
**Apply to:** All video-related hooks
```typescript
const videoRef = useRef<HTMLVideoElement>(null)
```

### React Hook Callback Pattern
**Source:** `frontend/src/components/VideoPlayerModal.tsx` (lines 149-160)
**Apply to:** All video control callbacks
```typescript
const handleCallback = useCallback(() => {
  const video = videoRef.current
  if (!video) return
  
  // Control logic
}, [dependencies])
```

### Go Service Constructor Pattern
**Source:** `internal/services/snapshot_service.go` (lines 25-36)
**Apply to:** All Go services
```go
func NewSnapshotService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, videoFileService *VideoFileService) *SnapshotService {
    ffmpegPath := cfg.FFmpeg.Path
    if ffmpegPath == "" {
        ffmpegPath = "ffmpeg"
    }
    return &SnapshotService{
        db:               db,
        logger:           logger,
        config:           cfg,
        ffmpegPath:       ffmpegPath,
        videoFileService: videoFileService,
    }
}
```

### Go Error Wrapping Pattern
**Source:** `internal/services/snapshot_service.go` (lines 45-46)
**Apply to:** All Go service methods
```go
if err := s.db.First(&task, taskID).Error; err != nil {
    return nil, fmt.Errorf("录制任务不存在: %w", err)
}
```

### FFmpeg Command Building Pattern
**Source:** `internal/services/snapshot_service.go` (lines 152-156)
**Apply to:** All FFmpeg operations
```go
cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
output, err := cmd.CombinedOutput()
if err != nil {
    return nil, fmt.Errorf("FFmpeg快照转换失败: %w, output: %s", err, string(output))
}
```

### Ant Design Message Feedback Pattern
**Source:** `frontend/src/components/VideoPlayerModal.tsx` (line 157, 183)
**Apply to:** All user feedback
```typescript
message.error('播放失败，请稍后重试')
message.info(`播放速度: ${nextRate}x`)
```

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `frontend/src/hooks/useKeyboardShortcuts.ts` | hook | event-driven | No existing keyboard shortcut hooks in codebase |
| `frontend/src/hooks/useVideoFrameNavigation.ts` | hook | request-response | No existing frame navigation hooks |
| `frontend/src/utils/videoPlayerHotkeys.ts` | utility | static | No existing keyboard shortcut utilities |

**Note:** These files should follow the patterns outlined in RESEARCH.md Examples 2-4 and the authStore/PlaybackSpeedControl patterns identified above.

---

## Metadata

**Analog search scope:** 
- `internal/services/*.go`
- `internal/handlers/*.go`
- `frontend/src/components/*.tsx`
- `frontend/src/hooks/*.ts`
- `frontend/src/utils/*.ts`

**Files scanned:** 20+ service/handler files, 20+ component files, 5 utility files

**Pattern extraction date:** 2026-04-20

**Key findings:**
1. Backend uses consistent service pattern with constructor, dependency injection, and error wrapping
2. Frontend uses Ant Design components consistently with TypeScript interfaces
3. Custom hooks follow useCallback/useEffect patterns for video element references
4. Time formatting utility is duplicated across components - should extract to shared utility
5. No existing keyboard shortcut implementation - this is new functionality
