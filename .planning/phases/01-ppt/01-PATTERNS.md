# Phase 01: 在视频播放中添加外挂字幕支持 - Pattern Map

**Mapped:** 2026-05-12
**Files analyzed:** 7 (3 new, 4 modified)
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/SubtitlePanel.tsx` | component | event-driven | `frontend/src/components/VideoPlayerModal.tsx` | role-match |
| `frontend/src/hooks/useSubtitleSync.ts` | hook | event-driven | `frontend/src/hooks/useKeyboardShortcuts.ts` | role-match |
| `frontend/src/types/subtitle.ts` | type | transform | `frontend/src/types/video-file.ts` | exact |
| `frontend/src/components/VideoPlayerModal.tsx` | component | request-response | Existing file (self-analog) | self-modify |
| `frontend/src/components/PPTPreview.tsx` | component | event-driven | Existing file (self-analog) | self-modify |
| `frontend/src/api/video-file.ts` | api-client | request-response | Existing file (self-analog) | self-modify |
| `internal/handlers/video_file_handler.go` | handler | request-response | Existing file (self-analog) | self-modify |

## Pattern Assignments

### `frontend/src/components/SubtitlePanel.tsx` (component, event-driven)

**Analog:** `frontend/src/components/VideoPlayerModal.tsx`

**Imports pattern** (lines 1-18):
```typescript
import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Modal, Button, Space, Alert, message, Slider } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  // ... other icons
} from '@ant-design/icons'
import type { VideoFile } from '../types/video-file'
import { getToken } from '../api/apiClient'
import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts'
```

**STYLES constant pattern** (lines 29-78):
```typescript
// 样式常量
const STYLES = {
  container: {
    position: 'relative',
    backgroundColor: '#000',
    borderRadius: '8px',
    overflow: 'hidden',
  } as const,
  loadingOverlay: {
    position: 'absolute',
    inset: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#000',
    zIndex: 1,
  } as const,
  // ... more style definitions
} as const
```

**Component structure pattern** (lines 125-139):
```typescript
interface ComponentProps {
  file: VideoFile
  visible: boolean
  onClose: () => void
}

export function ComponentName({ file, visible, onClose }: ComponentProps) {
  // ==================== 状态 ====================
  const videoRef = useRef<HTMLVideoElement>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)

  // ==================== 计算值 ====================
  const videoUrl = useMemo(() => {
    const API_BASE_URL = import.meta.env.VITE_API_URL || ''
    const token = getToken()
    return token
      ? `${API_BASE_URL}/api/v1/files/${file.id}/download?token=${token}`
      : `${API_BASE_URL}/api/v1/files/${file.id}/download`
  }, [file.id])

  // ==================== 回调函数 ====================
  const handlePlayPause = useCallback(() => {
    const video = videoRef.current
    if (!video) return
    // ... implementation
  }, [isPlaying])

  // ==================== 渲染 ====================
  return (
    <div style={STYLES.container}>
      {/* JSX content */}
    </div>
  )
}
```

**Ant Design integration pattern** (lines 364-371):
```typescript
return (
  <Modal
    title={`${file.file_name} - 视频预览`}
    open={visible}
    onCancel={handleClose}
    footer={null}
    width={900}
    centered
  >
    {/* Modal content */}
  </Modal>
)
```

---

### `frontend/src/hooks/useSubtitleSync.ts` (hook, event-driven)

**Analog:** `frontend/src/hooks/useKeyboardShortcuts.ts`

**Imports pattern** (lines 1-14):
```typescript
import { useEffect, useCallback } from 'react'
import { message } from 'antd'

// ==================== Constants ====================
/**
 * Available playback speeds for video player
 */
const PLAYBACK_SPEEDS = [0.5, 1, 1.25, 1.5, 2]
```

**Hook interface pattern** (lines 17-29):
```typescript
// ==================== Hook Interface ====================

interface UseKeyboardShortcutsProps {
  videoRef: React.RefObject<HTMLVideoElement | null>
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

**Hook implementation pattern** (lines 33-177):
```typescript
// ==================== Hook Implementation ====================

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
    if (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.tagName === 'SELECT' ||
      target.isContentEditable
    ) {
      return
    }

    // Event handling logic
  }, [enabled, isPlaying, playbackRate, volume, onPlayPause, onSeek, onVolumeChange, onPlaybackRateChange, onFullscreen, onMuteToggle, videoRef])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}
```

**Event listener cleanup pattern** (lines 179-183):
```typescript
useEffect(() => {
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [handleKeyDown])
```

---

### `frontend/src/types/subtitle.ts` (type, transform)

**Analog:** `frontend/src/types/video-file.ts`

**Type definition pattern** (lines 9-39):
```typescript
export type VideoFileStatus = 'ready' | 'processing' | 'error' | 'deleting'

export interface VideoFile {
  id: number
  file_name: string
  file_path: string
  file_size: number
  duration: number
  format: string
  resolution: string
  bitrate: number
  codec: string
  task_id?: number | null
  task?: {
    id: number
    name: string
  } | null
  parent_id?: number | null
  source_type: string
  snapshot_offset: number
  parent?: {
    id: number
    file_name: string
  } | null
  status: VideoFileStatus
  thumbnail_path: string | null
  recorded_at: string | null
  has_ppt?: boolean
  created_at: string
  updated_at: string
}
```

**List params pattern** (lines 41-51):
```typescript
export interface VideoFileListParams {
  page?: number
  page_size?: number
  keyword?: string
  task_id?: number
  status?: VideoFileStatus
  format?: string
  source_type?: string
  start_date?: string
  end_date?: string
}
```

**Response type pattern** (lines 53-58):
```typescript
export interface VideoFileListResponse {
  total: number
  items: VideoFile[]
  total_size: number
  total_size_gb: number
}
```

---

### `frontend/src/components/VideoPlayerModal.tsx` (component, request-response - MODIFY)

**Analog:** Self-analog (existing file to modify)

**Where to add subtitle support:**

1. **Import section** (after line 17): Add subtitle hook import
```typescript
import { useSubtitleSync } from '../hooks/useSubtitleSync'
```

2. **State section** (after line 144): Add subtitle state
```typescript
const [subtitlesEnabled, setSubtitlesEnabled] = useState(false)
const [subtitleText, setSubtitleText] = useState<string>('')
const [subtitleStyle, setSubtitleStyle] = useState({
  fontSize: 'medium',
  position: 'bottom',
  color: '#ffffff',
  backgroundColor: 'rgba(0, 0, 0, 0.7)'
})
```

3. **Custom hook integration** (after line 339): Add subtitle sync hook
```typescript
// ==================== 字幕同步 ====================
const currentSubtitle = useSubtitleSync({
  videoRef,
  vttContent: subtitlesEnabled ? subtitleContent : null,
  enabled: visible && subtitlesEnabled
})

useEffect(() => {
  if (currentSubtitle) {
    setSubtitleText(currentSubtitle.text)
  } else {
    setSubtitleText('')
  }
}, [currentSubtitle])
```

4. **Control bar section** (after line 493): Add subtitle toggle button
```typescript
<ControlButton
  icon={<SubtitlesOutlined />}
  onClick={() => setSubtitlesEnabled(!subtitlesEnabled)}
  title={subtitlesEnabled ? '关闭字幕' : '开启字幕'}
/>
```

5. **Video section** (after line 418): Add subtitle panel
```typescript
{/* 字幕显示区域 */}
{subtitlesEnabled && subtitleText && (
  <div style={{
    padding: '12px 16px',
    backgroundColor: subtitleStyle.backgroundColor,
    color: subtitleStyle.color,
    fontSize: subtitleStyle.fontSize === 'small' ? '14px' : subtitleStyle.fontSize === 'large' ? '20px' : '16px',
    textAlign: 'center',
    minHeight: '40px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center'
  }}>
    {subtitleText}
  </div>
)}
```

---

### `frontend/src/components/PPTPreview.tsx` (component, event-driven - MODIFY)

**Analog:** Self-analog (existing file to modify)

**Where to add subtitle support for embedded videos:**

1. **State section** (after line 43): Add subtitle state for embedded videos
```typescript
const [videoSubtitlesEnabled, setVideoSubtitlesEnabled] = useState(false)
```

2. **Video rendering section**: If component contains video elements, add subtitle panel similar to VideoPlayerModal pattern

---

### `frontend/src/api/video-file.ts` (api-client, request-response - MODIFY)

**Analog:** Self-analog (existing file to modify)

**Where to add subtitle APIs:**

1. **After line 37**: Add subtitle check API
```typescript
// 检查字幕文件是否存在
export async function checkSubtitleExists(id: number): Promise<ApiResponse<{ exists: boolean; subtitle_url?: string }>> {
  return apiRequest<{ exists: boolean; subtitle_url?: string }>(`/api/v1/files/${id}/subtitle`)
}

// 获取字幕文件内容
export async function getSubtitleContent(id: number): Promise<string> {
  const token = getToken()
  const url = `${API_BASE_URL}/api/v1/files/${id}/subtitle/download`

  const response = await fetch(url, {
    headers: {
      ...(token ? { 'Authorization': `Bearer ${token}` } : {})
    }
  })

  if (!response.ok) {
    throw new Error('Failed to fetch subtitle')
  }

  return await response.text()
}
```

**API client pattern** (lines 14-31):
```typescript
export async function getVideoFileList(
  params: VideoFileListParams
): Promise<ApiResponse<VideoFileListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)

  const query = queryParams.toString()
  return apiRequest<VideoFileListResponse>(
    `/api/v1/files${query ? `?${query}` : ''}`
  )
}
```

**Token-based fetch pattern** (lines 40-58):
```typescript
export function downloadVideoFile(id: number, fileName?: string): void {
  const token = getToken()
  const url = `${API_BASE_URL}/api/v1/files/${id}/download`

  fetch(url, {
    headers: {
      ...(token ? { 'Authorization': `Bearer ${token}` } : {})
    }
  })
  .then(response => response.blob())
  .then(blob => {
    const blobUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = fileName || `video_${id}.mp4`
    link.click()
    URL.revokeObjectURL(blobUrl)
  })
}
```

---

### `internal/handlers/video_file_handler.go` (handler, request-response - MODIFY)

**Analog:** Self-analog (existing file to modify)

**Where to add subtitle endpoints:**

1. **After line 333**: Add subtitle check and download handlers
```go
// GetSubtitle 检查字幕文件是否存在并返回URL
func (h *VideoFileHandler) GetSubtitle(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
        return
    }

    file, err := h.fileService.GetFileByID(id)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "文件不存在")
        return
    }

    // 构造字幕文件路径
    subtitlePath := strings.TrimSuffix(file.FilePath, filepath.Ext(file.FilePath)) + ".vtt"

    // 检查文件是否存在
    if _, err := os.Stat(subtitlePath); os.IsNotExist(err) {
        response.GinSuccess(c, gin.H{"exists": false})
        return
    }

    // 返回字幕文件URL
    subtitleURL := fmt.Sprintf("/api/v1/files/%d/subtitle/download", id)
    response.GinSuccess(c, gin.H{"exists": true, "subtitle_url": subtitleURL})
}

// DownloadSubtitle 下载字幕文件
func (h *VideoFileHandler) DownloadSubtitle(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
        return
    }

    file, err := h.fileService.GetFileByID(id)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "文件不存在")
        return
    }

    // 检查数据访问权限（复用现有权限检查逻辑）
    userID := middleware.GetUserID(c)
    if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
        response.GinError(c, response.CodeForbidden, "无权访问此文件")
        return
    }

    // 构造字幕文件路径
    subtitlePath := strings.TrimSuffix(file.FilePath, filepath.Ext(file.FilePath)) + ".vtt"

    // 读取字幕文件
    content, err := os.ReadFile(subtitlePath)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "字幕文件不存在")
        return
    }

    // 设置正确的Content-Type
    c.Header("Content-Type", "text/vtt; charset=utf-8")
    c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.vtt\"", file.FileName))
    c.Header("Access-Control-Allow-Origin", "*")
    c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
    c.Header("Access-Control-Allow-Headers", "Range, Content-Type, Authorization")
    c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")

    c.Data(http.StatusOK, "text/vtt; charset=utf-8", content)
}
```

**Handler pattern** (lines 67-81):
```go
// GetFile 获取文件详情
func (h *VideoFileHandler) GetFile(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的文件ID")
        return
    }

    file, err := h.fileService.GetFileByID(id)
    if err != nil {
        response.GinError(c, response.CodeNotFound, "文件不存在")
        return
    }

    response.GinSuccess(c, file)
}
```

**Permission check pattern** (lines 99-109):
```go
// 检查数据访问权限
// shared_viewer 和 admin 可以访问所有文件，普通用户只能访问自己创建的文件
userID := middleware.GetUserID(c)
if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
    h.logger.Warn("用户无权访问文件",
        zap.Uint("user_id", userID),
        zap.Uint("file_id", id),
        zap.Uint("file_owner", file.CreatedBy))
    response.GinError(c, response.CodeForbidden, "无权访问此文件")
    return
}
```

**File serving pattern** (lines 144-158):
```go
// setVideoHeaders 设置视频流响应头
func setVideoHeaders(c *gin.Context, file *models.VideoFile, fileSize int64) {
    // 根据文件格式设置 Content-Type
    contentType := getContentType(file.Format)

    c.Header("Content-Type", contentType)
    c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", file.FileName))

    // 设置视频流所需的 CORS 头
    c.Header("Access-Control-Allow-Origin", "*")
    c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
    c.Header("Access-Control-Allow-Headers", "Range, Content-Type, Authorization")
    c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
}
```

---

## Shared Patterns

### React Component State Management
**Source:** `frontend/src/components/VideoPlayerModal.tsx` (lines 132-145)
**Apply to:** `SubtitlePanel.tsx`, `VideoPlayerModal.tsx` (modifications)
```typescript
// ==================== 状态 ====================
const videoRef = useRef<HTMLVideoElement>(null)
const [isPlaying, setIsPlaying] = useState(false)
const [currentTime, setCurrentTime] = useState(0)
const [duration, setDuration] = useState(0)
const [error, setError] = useState<string>()

// ==================== 计算值 ====================
const videoUrl = useMemo(() => {
  const API_BASE_URL = import.meta.env.VITE_API_URL || ''
  const token = getToken()
  return token
    ? `${API_BASE_URL}/api/v1/files/${file.id}/download?token=${token}`
    : `${API_BASE_URL}/api/v1/files/${file.id}/download`
}, [file.id])

// ==================== 回调函数 ====================
const handlePlayPause = useCallback(() => {
  const video = videoRef.current
  if (!video) return
  // ... implementation
}, [isPlaying])
```

### Custom Hook with Cleanup
**Source:** `frontend/src/hooks/useKeyboardShortcuts.ts` (lines 179-183)
**Apply to:** `useSubtitleSync.ts`
```typescript
useEffect(() => {
  window.addEventListener('keydown', handleKeyDown)
  return () => window.removeEventListener('keydown', handleKeyDown)
}, [handleKeyDown])
```

### API Request with Token Authentication
**Source:** `frontend/src/api/apiClient.ts` (lines 140-155)
**Apply to:** `video-file.ts` (new subtitle APIs)
```typescript
export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = `${API_BASE_URL}${endpoint}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }

  let token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  const data: ApiResponse<T> = await response.json()

  if (!response.ok) {
    throw new Error(data.message || 'Request failed')
  }

  return data
}
```

### Go Handler with Permission Check
**Source:** `internal/handlers/video_file_handler.go` (lines 99-109)
**Apply to:** `video_file_handler.go` (subtitle endpoints)
```go
// 检查数据访问权限
userID := middleware.GetUserID(c)
if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
    h.logger.Warn("用户无权访问文件",
        zap.Uint("user_id", userID),
        zap.Uint("file_id", id),
        zap.Uint("file_owner", file.CreatedBy))
    response.GinError(c, response.CodeForbidden, "无权访问此文件")
    return
}
```

### File Path Construction Pattern
**Source:** `internal/handlers/video_file_handler.go` (lines 195-196, 360)
**Apply to:** `video_file_handler.go` (subtitle endpoints)
```go
// 构造字幕文件路径（替换扩展名为.vtt）
subtitlePath := strings.TrimSuffix(file.FilePath, filepath.Ext(file.FilePath)) + ".vtt"
```

### CORS Headers for File Serving
**Source:** `internal/handlers/video_file_handler.go` (lines 144-158)
**Apply to:** `video_file_handler.go` (subtitle download endpoint)
```go
c.Header("Content-Type", "text/vtt; charset=utf-8")
c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.vtt\"", file.FileName))
c.Header("Access-Control-Allow-Origin", "*")
c.Header("Access-Control-Allow-Methods", "GET, HEAD, OPTIONS")
c.Header("Access-Control-Allow-Headers", "Range, Content-Type, Authorization")
c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")
```

### Ant Design Message Pattern
**Source:** `frontend/src/components/VideoPlayerModal.tsx` (lines 166, 192, 252)
**Apply to:** `SubtitlePanel.tsx`
```typescript
import { message } from 'antd'

// Show success message
message.success('开始下载')

// Show info message
message.info(`播放速度: ${nextRate}x`)

// Show error message
message.error('播放失败，请稍后重试')
```

### STYLES Constant Pattern
**Source:** `frontend/src/components/VideoPlayerModal.tsx` (lines 29-78)
**Apply to:** `SubtitlePanel.tsx`
```typescript
const STYLES = {
  container: {
    position: 'relative',
    backgroundColor: '#000',
    borderRadius: '8px',
    overflow: 'hidden',
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

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| None | - | - | All files have strong analogs in the existing codebase |

## Metadata

**Analog search scope:** 
- Frontend: `frontend/src/components/`, `frontend/src/hooks/`, `frontend/src/types/`, `frontend/src/api/`
- Backend: `internal/handlers/`, `internal/services/`

**Files scanned:** 20+ (React components, hooks, types, API clients, Go handlers)

**Pattern extraction date:** 2026-05-12

**Key insights:**
1. **React components** follow a consistent structure: imports → constants → interface → state → computed values → callbacks → render
2. **Custom hooks** use useCallback for event handlers and useEffect for setup/cleanup
3. **API clients** use centralized `apiRequest` function with token authentication
4. **Go handlers** follow a consistent pattern: parse params → check permissions → business logic → set headers → send response
5. **File serving** requires proper CORS headers and UTF-8 charset for text files
