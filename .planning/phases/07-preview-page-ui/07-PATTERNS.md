# Phase 07: Preview Page UI Improvements - Pattern Map

**Mapped:** 2026-04-20
**Files analyzed:** 6
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/pages/results/index.tsx` | page/component | request-response | `frontend/src/pages/results/index.tsx` | exact (modify) |
| `frontend/src/components/VideoPreviewPanel.tsx` | component | event-driven | `frontend/src/components/VideoPreviewPanel.tsx` | exact (modify) |
| `frontend/src/components/EditableProgressBar.tsx` | component | event-driven | `frontend/src/components/VideoPreviewPanel.tsx` (progress bar) | role-match |
| `frontend/src/components/PPTResultsDropdown.tsx` | component | event-driven | `frontend/src/pages/files/index.tsx` (transcription dropdown) | role-match |
| `frontend/src/components/SlideThumbnail.tsx` | component | event-driven | `frontend/src/components/SlideThumbnail.tsx` | exact (modify) |
| `frontend/src/styles/global.css` | config | CSS | `frontend/src/styles/global.css` | exact (modify) |

## Pattern Assignments

### `frontend/src/pages/results/index.tsx` (page/component, request-response)

**Analog:** `frontend/src/pages/results/index.tsx` (existing file - lines 1-810)

**Imports pattern** (lines 1-56):
```typescript
import { useState, useEffect, useCallback, useRef } from 'react'
import {
  Button,
  Card,
  Descriptions,
  Space,
  message,
  Popconfirm,
  Tabs,
  Dropdown,
} from 'antd'
import {
  ArrowLeftOutlined,
  DownloadOutlined,
  RedoOutlined,
  MergeCellsOutlined,
  DeleteOutlined,
  CloudOutlined,
  LaptopOutlined,
  ScanOutlined,
  CameraOutlined,
  VideoCameraOutlined,
  DragOutlined,
} from '@ant-design/icons'
import { useNavigate, useParams } from 'react-router-dom'
import dayjs from 'dayjs'
import PPTPreview from '../../components/PPTPreview'
import PPTGalleryStrip from '../../components/PPTGalleryStrip'
import { VideoPreviewPanel } from '../../components/VideoPreviewPanel'
import SlideThumbnail from '../../components/SlideThumbnail'
import type { PPTResult, SlideImage } from '../../types/ppt'
import { getPptsByVideo, getSlides, deletePpt, getPptDownloadUrl } from '../../api/ppt'
```

**Layout structure pattern** (lines 552-622):
```typescript
{/* Preview Area with Side-by-Side Layout */}
<div className="ppt-preview-grid" style={previewAreaStyle}>
  {/* Left: Thumbnail Sidebar (160px) */}
  <div
    ref={thumbnailContainerRef}
    style={{
      overflowY: 'auto',
      borderRight: '1px solid #f0f0f0',
      padding: 8,
      background: '#fafafa',
      maxHeight: 'calc(100vh - 200px)',
      scrollBehavior: 'smooth',
    }}
  >
    {/* Thumbnail list with SlideThumbnail components */}
  </div>

  {/* Center: PPT Preview (16:9) */}
  <div style={previewBoxStyle}>
    <PPTPreview
      slides={slides}
      currentSlide={currentSlide}
      onSlideChange={handleSlideChange}
      hideThumbnailSidebar={true}
    />
  </div>

  {/* Right: Video Preview (16:9) */}
  {isVideoPanelVisible && (
    <div style={previewBoxStyle}>
      <VideoPreviewPanel
        videoFileId={videoFileIdNum}
        currentSlide={currentSlide + 1}
        onSlideClick={handleVideoSlideChange}
      />
    </div>
  )}
</div>
```

**Info/Operations bar pattern** (lines 624-750):
```typescript
<Card size="small" style={{ marginBottom: 16 }}>
  <Tabs
    defaultActiveKey="info"
    items={[
      {
        key: 'info',
        label: '基本信息',
        children: (
          <Descriptions column={1} size="small">
            <Descriptions.Item label="视频名称">{videoName}</Descriptions.Item>
            <Descriptions.Item label="转录时间">
              {currentPpt ? dayjs(currentPpt.created_at).format('YYYY-MM-DD HH:mm') : '—'}
            </Descriptions.Item>
            <Descriptions.Item label="页数">{currentPpt?.page_count || 0} 页</Descriptions.Item>
            <Descriptions.Item label="文件大小">{formatFileSize(currentPpt?.file_size || 0)}</Descriptions.Item>
            <Descriptions.Item label="类型">
              {currentPpt?.source_type === 'merge' ? '合并' : '转录'}
            </Descriptions.Item>
          </Descriptions>
        ),
      },
      {
        key: 'operations',
        label: '操作',
        children: (
          <Space direction="vertical" style={{ width: '100%' }}>
            <Button block icon={<DownloadOutlined />} onClick={handleDownloadPpt}>
              下载PPT
            </Button>
            {/* More operation buttons */}
          </Space>
        ),
      },
    ]}
  />
</Card>
```

**State management pattern** (lines 82-119):
```typescript
const [ppts, setPpts] = useState<PPTResult[]>([])
const [currentPptId, setCurrentPptId] = useState<number>(0)
const [slides, setSlides] = useState<SlideImage[]>([])
const [currentSlide, setCurrentSlide] = useState(0)
const [isMergeMode, setIsMergeMode] = useState(false)
const [selectedSlides, setSelectedSlides] = useState<SelectedSlide[]>([])
const [videoName, setVideoName] = useState('')
const [loading, setLoading] = useState(false)

// Refs for DOM manipulation
const slidesPollCleanupRef = useRef<(() => void) | null>(null)
const thumbnailContainerRef = useRef<HTMLDivElement>(null)
const videoRef = useRef<HTMLVideoElement>(null)
```

---

### `frontend/src/components/VideoPreviewPanel.tsx` (component, event-driven)

**Analog:** `frontend/src/components/VideoPreviewPanel.tsx` (lines 1-438)

**Imports pattern** (lines 1-17):
```typescript
import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Card, Alert, Space, Button, message } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  StepForwardOutlined,
  StepBackwardOutlined,
  FullscreenOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import { getTimestampMap } from '../api/transcription'
import type { SlideTimestamp } from '../types/transcription'
import { getToken } from '../api/apiClient'
import { PlaybackSpeedControl, usePlaybackSpeed } from './PlaybackSpeedControl'
```

**Time formatting utility** (lines 24-37):
```typescript
// 格式化时间（秒 -> MM:SS 或 HH:MM:SS）
function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00'

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

**Video element styling pattern** (lines 333-356):
```typescript
<div
  style={{
    position: 'relative',
    backgroundColor: '#000',
    borderRadius: '8px',
    overflow: 'hidden',
  }}
>
  <video
    ref={videoRef}
    src={videoUrl}
    style={{
      width: '100%',
      maxHeight: '400px',
      display: 'block',
    }}
    preload="metadata"
    onLoadedMetadata={handleLoadedMetadata}
    onError={handleError}
    onTimeUpdate={handleTimeUpdate}
    onPlay={handlePlay}
    onPause={handlePause}
  />
  {/* Custom controls overlay */}
</div>
```

**Progress bar pattern** (lines 371-382):
```typescript
{/* 进度条 */}
<input
  type="range"
  min={0}
  max={duration}
  value={currentTime}
  onChange={(e) => handleSeek(Number(e.target.value))}
  style={{
    width: '100%',
    cursor: 'pointer',
  }}
/>
```

**Video API usage pattern** (lines 210-236):
```typescript
// Play/Pause control
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

// Seek control
const handleSeek = useCallback((value: number) => {
  const video = videoRef.current
  if (!video) return
  video.currentTime = value
  video.playbackRate = playbackRate  // Restore speed after seek
}, [playbackRate])
```

---

### `frontend/src/components/EditableProgressBar.tsx` (component, event-driven) - NEW

**Analog:** `frontend/src/components/VideoPreviewPanel.tsx` (progress bar + time display, lines 371-416)

**InputNumber pattern from PPTPreview** (`frontend/src/components/PPTPreview.tsx`, lines 197-204):
```typescript
<InputNumber
  min={1}
  max={slides.length}
  value={currentSlide + 1}
  onChange={(v) => v && onSlideChange(v - 1)}
  style={{ width: 80 }}
  aria-label={`跳转到页码，共${slides.length}页`}
/>
```

**Space component layout** (`VideoPreviewPanel.tsx`, lines 385-424):
```typescript
<div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
  <Space>
    <Button type="text" icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />} />
    <Button type="text" icon={<StepBackwardOutlined />} />
    <span style={{ color: '#fff', marginLeft: '8px' }}>
      {formatTime(currentTime)} / {formatTime(duration)}
    </span>
  </Space>
  <Button type="text" icon={<FullscreenOutlined />} />
</div>
```

**Key pattern for time input**:
- Use `InputNumber` with `formatter` and `parser` props for MM:SS ↔ seconds conversion
- Sync with video `currentTime` on change
- Display formatted time, store raw seconds

---

### `frontend/src/components/PPTResultsDropdown.tsx` (component, event-driven) - NEW

**Analog:** `frontend/src/pages/files/index.tsx` (transcription dropdown, lines 446-468)

**Dropdown menu pattern** (lines 446-468):
```typescript
<Dropdown
  menu={{
    items: [
      {
        key: 'local',
        icon: <LaptopOutlined />,
        label: '本地转录',
        onClick: () => handleTranscribeClick(record),
      },
      {
        key: 'cloud',
        icon: <CloudOutlined />,
        label: '云端转录（通义听悟）',
        onClick: () => handleCloudTranscribe(record),
      },
    ],
  }}
  trigger={['click']}
>
  <Button type="primary" size="small" icon={<CloudOutlined />}>
    转录
  </Button>
</Dropdown>
```

**Alternative gallery pattern** (`frontend/src/components/PPTGalleryStrip.tsx`, lines 14-103):
```typescript
export default function PPTGalleryStrip({
  ppts,
  currentPptId,
  onSelect,
}: PPTGalleryStripProps) {
  return (
    <div style={{ display: 'flex', gap: 8, overflowX: 'auto', padding: '12px 0' }}>
      {ppts.map((ppt) => (
        <Tooltip key={ppt.id} title={`${dayjs(ppt.created_at).format('HH:mm')} 转录 · ${ppt.page_count} 页`}>
          <div
            onClick={() => onSelect(ppt.id)}
            style={{
              cursor: 'pointer',
              width: 100,
              height: 80,
              border: currentPptId === ppt.id ? '2px solid #1890ff' : '1px solid #f0f0f0',
              background: currentPptId === ppt.id ? '#e6f7ff' : '#ffffff',
            }}
          >
            <div>{dayjs(ppt.created_at).format('HH:mm')}</div>
            <div>{ppt.page_count}页</div>
          </div>
        </Tooltip>
      ))}
    </div>
  )
}
```

**Key patterns**:
- Use Ant Design `Dropdown` with `menu.items` array
- Each item has `key`, `label`, `icon`, and `onClick`
- Use `dayjs` for date formatting
- Visual indicator for current selection (checkmark or border color)

---

### `frontend/src/components/SlideThumbnail.tsx` (component, event-driven)

**Analog:** `frontend/src/components/SlideThumbnail.tsx` (lines 1-182)

**Image component pattern** (lines 98-121):
```typescript
{slide.thumbnail_url ? (
  <Image
    src={slide.thumbnail_url}
    alt={`幻灯片 ${slideNumber}`}
    width={160}
    height={90}
    preview={false}
    loading="lazy"
    onError={(e) => {
      const img = e.currentTarget as HTMLImageElement
      img.src = 'data:image/svg+xml;base64,...'  // Fallback placeholder
    }}
    style={{
      objectFit: 'cover',
      display: 'block',
    }}
  />
) : (
  <Skeleton.Image active style={{ width: 160, height: 90 }} />
)}
```

**Container styling pattern** (lines 69-90):
```typescript
<div
  style={{
    position: 'relative',
    cursor: isDraggable ? 'move' : isSelectable ? 'pointer' : 'default',
    border: isCurrent || isSelected ? '2px solid #1890ff' : '2px solid transparent',
    borderRadius: 4,
    overflow: 'hidden',
    opacity: isDragging ? 0.3 : isCurrent ? 1 : 0.6,
    transition: 'opacity 0.2s, transform 0.2s',
  }}
  onMouseEnter={(e) => {
    if (!isCurrent && !isDragging) {
      e.currentTarget.style.opacity = '0.8'
      e.currentTarget.style.transform = 'scale(1.02)'
    }
  }}
  onMouseLeave={(e) => {
    if (!isCurrent && !isDragging) {
      e.currentTarget.style.opacity = '0.6'
      e.currentTarget.style.transform = 'scale(1)'
    }
  }}
>
  {/* Image content */}
</div>
```

**Key pattern modification**:
- Add fixed height constraint to match 16:9 preview area
- Use `maxHeight` in conjunction with existing `overflowY: auto` on parent container

---

### `frontend/src/styles/global.css` (config, CSS)

**Analog:** `frontend/src/styles/global.css` (lines 59-74)

**Existing PPT preview grid** (lines 59-74):
```css
/* PPT Results Page - Side-by-Side Preview Layout */
.ppt-preview-grid {
  display: grid;
  grid-template-columns: 160px 1fr 1fr;
  gap: 16px;
  margin-bottom: 16px;
}

/* Responsive breakpoint: Stack vertically on screens < 1200px */
@media (max-width: 1200px) {
  .ppt-preview-grid {
    grid-template-columns: 1fr;
  }
}
```

**Key pattern for thumbnail sidebar**:
- Use CSS Grid with `align-items: start` to align sidebar to top
- Add `grid-template-rows: auto` to let row height match content
- Use `calc()` for dynamic height matching preview area

---

## Shared Patterns

### Ant Design Dropdown Component
**Source:** `frontend/src/pages/files/index.tsx` (lines 446-468)
**Apply to:** `PPTResultsDropdown.tsx`
```typescript
<Dropdown
  menu={{
    items: [
      {
        key: string,
        icon: <Icon />,
        label: string | JSX.Element,
        onClick: () => void,
      },
    ],
  }}
  trigger={['click']}
>
  <Button icon={<Icon />}>Button Label</Button>
</Dropdown>
```

### Time Formatting Utility
**Source:** `frontend/src/components/VideoPreviewPanel.tsx` (lines 24-37)
**Apply to:** `EditableProgressBar.tsx`, `VideoPreviewPanel.tsx`
```typescript
function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00'

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

### InputNumber with Validation
**Source:** `frontend/src/components/PPTPreview.tsx` (lines 197-204)
**Apply to:** `EditableProgressBar.tsx`
```typescript
<InputNumber
  min={number}
  max={number}
  value={number}
  onChange={(v) => v && callback(v)}
  style={{ width: number }}
  aria-label={string}
/>
```

### Video Element API
**Source:** `frontend/src/components/VideoPreviewPanel.tsx` (lines 210-236)
**Apply to:** `EditableProgressBar.tsx` (for video sync)
```typescript
const video = videoRef.current
if (!video) return

// Seek
video.currentTime = value

// Play/Pause
video.play().catch(() => message.error('播放失败'))
video.pause()

// Current time
const currentTime = video.currentTime
const duration = video.duration
```

### CSS object-fit for Aspect Ratio
**Source:** `frontend/src/components/SlideThumbnail.tsx` (lines 111-115)
**Apply to:** `VideoPreviewPanel.tsx` video element
```css
object-fit: cover;  /* Crop to fill container, remove black bars */
```

### dayjs Date Formatting
**Source:** `frontend/src/pages/results/index.tsx` (lines 635-636)
**Apply to:** `PPTResultsDropdown.tsx`
```typescript
import dayjs from 'dayjs'

dayjs(ppt.created_at).format('MM-DD HH:mm')
dayjs(ppt.created_at).format('YYYY-MM-DD HH:mm:ss')
```

### Space Component for Layout
**Source:** `frontend/src/components/VideoPreviewPanel.tsx` (lines 385-424)
**Apply to:** Horizontal operations bar in results page
```typescript
<Space direction="horizontal" wrap>
  <Button icon={<Icon />}>Button</Button>
  <Button icon={<Icon />}>Button</Button>
</Space>
```

### Descriptions Component for Info Display
**Source:** `frontend/src/pages/results/index.tsx` (lines 633-643)
**Apply to:** Info section in results page (remove Tabs wrapper)
```typescript
<Descriptions column={1} size="small">
  <Descriptions.Item label="标签">{value}</Descriptions.Item>
  <Descriptions.Item label="标签">{value}</Descriptions.Item>
</Descriptions>
```

### React State Management with Refs
**Source:** `frontend/src/pages/results/index.tsx` (lines 82-119)
**Apply to:** All component state
```typescript
const [state, setState] = useState<Type>(initialValue)
const ref = useRef<HTMLDivElement>(null)

// Access DOM
if (ref.current) {
  ref.current.scrollIntoView({ behavior: 'smooth' })
}
```

---

## No Analog Found

**None identified** - All files have close analogs in the codebase.

---

## Metadata

**Analog search scope:**
- `frontend/src/pages/results/` - Main preview page
- `frontend/src/components/` - Reusable components
- `frontend/src/styles/` - Global styles
- `frontend/src/types/` - TypeScript type definitions

**Files scanned:** 20+ components and pages

**Pattern extraction date:** 2026-04-20

**Key findings:**
1. **Strong existing patterns** for video controls (VideoPreviewPanel), dropdown menus (files page), and thumbnail display (SlideThumbnail)
2. **InputNumber component** already used in PPTPreview for slide navigation - can adapt for time input
3. **Time formatting utility** exists in VideoPreviewPanel - reuse for editable progress bar
4. **CSS Grid layout** already established for side-by-side preview - extend for thumbnail sidebar height matching
5. **Dropdown pattern** well-established in files page for transcription modes - adapt for PPT results switching
