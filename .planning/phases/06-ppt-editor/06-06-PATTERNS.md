# Phase 06-06: PPT Preview UI Improvements - Pattern Map

**Mapped:** 2026-04-20
**Files analyzed:** 7
**Analogs found:** 7 / 7

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `frontend/src/components/PlaybackSpeedControl.tsx` | component | request-response | `frontend/src/components/SlideCapturePanel.tsx` (Select usage) | exact |
| `frontend/src/components/OperationsBar.tsx` | component | request-response | `frontend/src/components/MergeSelectionBar.tsx` | role-match |
| `frontend/src/components/VideoPreviewPanel.tsx` | component | event-driven | `frontend/src/components/VideoPreviewPanel.tsx` (existing) | self-modify |
| `frontend/src/components/PPTPreview.tsx` | component | event-driven | `frontend/src/components/PPTPreview.tsx` (existing) | self-modify |
| `frontend/src/components/SlideCapturePanel.tsx` | component | request-response | `frontend/src/components/SlideCapturePanel.tsx` (existing) | self-modify |
| `frontend/src/components/SlideThumbnail.tsx` | component | event-driven | `frontend/src/components/SlideThumbnail.tsx` (existing) | self-modify |
| `frontend/src/pages/results/index.tsx` | page | event-driven | `frontend/src/pages/results/index.tsx` (existing) | self-modify |

## Pattern Assignments

### `frontend/src/components/PlaybackSpeedControl.tsx` (component, request-response)

**Analog:** `frontend/src/components/SlideCapturePanel.tsx` (lines 2-4, 271-276)

**Why this analog:** Both use Ant Design `Select` component with controlled state, similar prop pattern (callbacks for changes), and Ant Design icons.

**Imports pattern** (from `SlideCapturePanel.tsx` lines 2-4):
```typescript
import { Modal, Button, Space, InputNumber, Image, message, Progress, Select } from 'antd'
import { CameraOutlined, PlayCircleOutlined, PauseCircleOutlined, CheckOutlined } from '@ant-design/icons'
```

**Select component pattern** (from `SlideCapturePanel.tsx` lines 271-276):
```typescript
<Select
  value={insertPositionOption}
  onChange={handleInsertPositionOptionChange}
  options={insertPositionOptions}
  style={{ width: '100%' }}
/>
```

**Options array pattern** (from `SlideCapturePanel.tsx` lines 169-174):
```typescript
const insertPositionOptions = [
  { label: `当前幻灯片之后 (位置 ${currentSlide + 1})`, value: 'after' },
  { label: `当前幻灯片之前 (位置 ${currentSlide})`, value: 'before' },
  { label: `最后 (位置 ${totalSlides + 1})`, value: 'end' },
  { label: '自定义位置', value: 'custom' },
]
```

**Component interface pattern** (from `SlideCapturePanel.tsx` lines 13-20):
```typescript
interface SlideCapturePanelProps {
  pptFileId: number
  videoFileId: number
  currentSlide: number
  totalSlides: number
  onSlideInserted?: (newSlideNumber: number) => void
  onCancel?: () => void
  open?: boolean
}
```

**State management pattern** (from `VideoPreviewPanel.tsx` lines 63-66):
```typescript
const [isLoading, setIsLoading] = useState(true)
const [isPlaying, setIsPlaying] = useState(false)
const [currentTime, setCurrentTime] = useState(0)
const [duration, setDuration] = useState(0)
```

**Video ref pattern** (from `VideoPreviewPanel.tsx` lines 60, 322-336):
```typescript
const videoRef = useRef<HTMLVideoElement>(null)

// Video element with ref
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
```

**Event handler pattern** (from `VideoPreviewPanel.tsx` lines 193-204):
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
```

---

### `frontend/src/components/OperationsBar.tsx` (component, request-response)

**Analog:** `frontend/src/components/MergeSelectionBar.tsx`

**Why this analog:** Both are horizontal bar components with action buttons, similar layout structure (flexbox with space-between), and Ant Design Space/Button patterns.

**Imports pattern** (from `MergeSelectionBar.tsx` lines 4-8):
```typescript
import { Button, Space, Spin, Image } from 'antd'
import {
  CheckOutlined,
  CloseOutlined,
} from '@ant-design/icons'
import type { SelectedSlide } from '../types/ppt'
```

**Layout container pattern** (from `MergeSelectionBar.tsx` lines 167-173):
```typescript
<div
  style={{
    borderTop: '2px solid #1890ff',
    background: '#ffffff',
    padding: 16,
  }}
>
```

**Header section pattern** (from `MergeSelectionBar.tsx` lines 174-194):
```typescript
<div
  style={{
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 8,
  }}
>
  <span
    style={{
      fontSize: 16,
      fontWeight: 500,
    }}
  >
    已选择 {selectedSlides.length}/200 页
  </span>
  <Space>
    <Button onClick={onCancel} disabled={isMerging}>
      取消合并
    </Button>
    <Button
      type="primary"
      onClick={onConfirm}
      disabled={selectedSlides.length === 0}
      icon={isMerging ? <Spin size="small" /> : <CheckOutlined />}
    >
      {isMerging ? '合并中...' : '确认合并'}
    </Button>
  </Space>
</div>
```

**Component interface pattern** (from `MergeSelectionBar.tsx` lines 29-36):
```typescript
interface MergeSelectionBarProps {
  selectedSlides: SelectedSlide[]
  onReorder: (slides: SelectedSlide[]) => void
  onRemove: (id: string) => void
  onConfirm: () => void
  onCancel: () => void
  isMerging: boolean
}
```

**Space component pattern** (from `results/index.tsx` lines 542-613):
```typescript
<Space direction="vertical" style={{ width: '100%' }}>
  <Button
    block
    icon={<DownloadOutlined />}
    onClick={handleDownloadPpt}
  >
    下载PPT
  </Button>
  {/* More buttons... */}
</Space>
```

**Tabs pattern** (from `results/index.tsx` lines 503-538):
```typescript
<Tabs
  defaultActiveKey="info"
  style={{ marginBottom: 16 }}
  items={[
    {
      key: 'info',
      label: '基本信息',
      children: (
        <Descriptions column={1} size="small">
          {/* Info content */}
        </Descriptions>
      ),
    },
    {
      key: 'text',
      label: '文字内容',
      children: (
        <TextContentTab videoFileId={videoFileIdNum} />
      ),
    },
  ]}
/>
```

---

### `frontend/src/components/VideoPreviewPanel.tsx` (component, event-driven) - MODIFY

**Analog:** Self-modification - existing `VideoPreviewPanel.tsx`

**Existing imports** (lines 1-12):
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
```

**Constants pattern** (lines 17-19):
```typescript
const SKIP_SECONDS = 10
const TIME_UPDATE_DEBOUNCE_MS = 1000  // Debounce timeupdate events to avoid excessive updates
```

**Component interface** (lines 40-47):
```typescript
interface VideoPreviewPanelProps {
  videoFileId: number
  currentSlide?: number  // 1-based slide number
  onSlideClick?: (slideNumber: number) => void  // Callback for video -> slide sync
  style?: React.CSSProperties
  autoPlay?: boolean  // Auto-play video when seeking to slide
  showControls?: boolean  // Show custom playback controls
}
```

**State management** (lines 59-69):
```typescript
const videoRef = useRef<HTMLVideoElement>(null)
const containerRef = useRef<HTMLDivElement>(null)

const [isLoading, setIsLoading] = useState(true)
const [isPlaying, setIsPlaying] = useState(false)
const [currentTime, setCurrentTime] = useState(0)
const [duration, setDuration] = useState(0)
const [error, setError] = useState<string>()
const [timestampMap, setTimestampMap] = useState<Map<number, number>>(new Map())
const [timestampError, setTimestampError] = useState<string>()
```

**Video controls layout** (lines 338-398):
```typescript
{showControls && duration > 0 && (
  <div
    style={{
      position: 'absolute',
      bottom: 0,
      left: 0,
      right: 0,
      background: 'linear-gradient(transparent, rgba(0,0,0,0.8))',
      padding: '12px 16px',
      display: 'flex',
      flexDirection: 'column',
      gap: '8px',
    }}
  >
    {/* Progress bar */}
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

    {/* Control buttons row */}
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
      <Space>
        <Button
          type="text"
          icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
          onClick={handlePlayPause}
          style={{ color: '#fff' }}
        />
        {/* More controls... */}
      </Space>
    </div>
  </div>
)}
```

**Modification point:** Insert speed control between skip controls and fullscreen button (around line 390).

---

### `frontend/src/components/PPTPreview.tsx` (component, event-driven) - MODIFY

**Analog:** Self-modification - existing `PPTPreview.tsx`

**Existing imports** (lines 1-14):
```typescript
import { Button, Image, InputNumber, Spin, message } from 'antd'
import {
  DownloadOutlined,
  CopyOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
} from '@ant-design/icons'
import type { SlideImage, SelectedSlide } from '../types/ppt'
import { useState, useEffect, useCallback } from 'react'
import SlideThumbnail from './SlideThumbnail'
```

**Component interface** (lines 15-24):
```typescript
interface PPTPreviewProps {
  slides: SlideImage[]
  currentSlide: number
  onSlideChange: (index: number) => void
  isMergeMode: boolean
  selectedSlides: SelectedSlide[]
  onToggleSelect: (slide: SlideImage, index: number) => void
  isLoading: boolean
  currentPptId?: number
}
```

**Layout structure** (lines 92-123):
```typescript
return (
  <div style={{ display: 'flex', height: '100%', minHeight: 400 }}>
    {/* Sidebar thumbnails */}
    <div
      style={{
        width: 200,
        overflowY: 'auto',
        borderRight: '1px solid #f0f0f0',
        padding: 8,
        background: '#fafafa',
        display: isFullscreen ? 'none' : 'block',
      }}
    >
      {/* Thumbnails */}
    </div>

    {/* Main view */}
    <div
      style={{
        flex: 1,
        display: 'flex',
        flexDirection: 'column',
        padding: 24,
        background: '#ffffff',
        position: 'relative',
      }}
    >
      {/* Slide image */}
    </div>
  </div>
)
```

**Modification point:** Change width from fixed 200px to 160px, ensure vertical scrolling optimization.

---

### `frontend/src/components/SlideCapturePanel.tsx` (component, request-response) - MODIFY

**Analog:** Self-modification - existing `SlideCapturePanel.tsx`

**Existing imports** (lines 1-5):
```typescript
import React, { useState, useRef, useEffect } from 'react'
import { Modal, Button, Space, InputNumber, Image, message, Progress, Select } from 'antd'
import { CameraOutlined, PlayCircleOutlined, PauseCircleOutlined, CheckOutlined } from '@ant-design/icons'
import type { SlideCapturePanelProps } from '../types/ppt'
import { captureFrame, insertSlide } from '../api/ppt'
```

**API call pattern** (lines 95-114):
```typescript
const handleCaptureFrame = async () => {
  setIsCapturing(true)
  setCapturedFrame(null)

  try {
    const response = await captureFrame(pptFileId, videoState.currentTime)

    if (response.data?.success && response.data?.frame_data) {
      setCapturedFrame(response.data.frame_data)
      message.success('帧捕获成功')
    } else {
      message.error('帧捕获失败')
    }
  } catch (error) {
    console.error('Failed to capture frame:', error)
    message.error('帧捕获失败: ' + (error as Error).message)
  } finally {
    setIsCapturing(false)
  }
}
```

**Insert slide pattern** (lines 117-159):
```typescript
const handleInsertSlide = async () => {
  if (!capturedFrame) {
    message.warning('请先捕获帧')
    return
  }

  setIsInserting(true)

  try {
    const response = await insertSlide(
      pptFileId,
      capturedFrame,
      insertPosition,
      videoState.currentTime
    )

    if (response.data?.success) {
      message.success(`幻灯片插入成功，位置: ${response.data.inserted_slide_number}`)

      // Call callback with new slide number
      if (onSlideInserted) {
        onSlideInserted(response.data.inserted_slide_number)
      }

      // Reset captured frame
      setCapturedFrame(null)

      // Close modal after short delay
      setTimeout(() => {
        if (onCancel) {
          onCancel()
        }
      }, 1000)
    } else {
      message.error('幻灯片插入失败')
    }
  } catch (error) {
    console.error('Failed to insert slide:', error)
    message.error('幻灯片插入失败: ' + (error as Error).message)
  } finally {
    setIsInserting(false)
  }
}
```

**Modification point:** Remove Modal wrapper, create direct capture function that combines capture + insert, expose as simpler component or hook.

---

### `frontend/src/components/SlideThumbnail.tsx` (component, event-driven) - MODIFY

**Analog:** Self-modification - existing `SlideThumbnail.tsx`

**Existing imports** (lines 1-5):
```typescript
import { Image } from 'antd'
import type { SlideImage } from '../types/ppt'
```

**Component structure** (lines 17-95):
```typescript
export default function SlideThumbnail({
  slide,
  slideNumber,
  totalSlides,
  isSelected,
  isSelectable,
  isCurrent,
  onClick,
}: SlideThumbnailProps) {
  return (
    <div
      onClick={onClick}
      role="button"
      aria-label={`幻灯片${slideNumber}，共${totalSlides}页${isCurrent ? '，当前幻灯片' : ''}`}
      tabIndex={0}
      style={{
        position: 'relative',
        cursor: isSelectable ? 'pointer' : 'default',
        border: isCurrent || isSelected ? '2px solid #1890ff' : '2px solid transparent',
        borderRadius: 4,
        overflow: 'hidden',
        opacity: isCurrent ? 1 : 0.6,
        transition: 'opacity 0.2s',
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
    >
      <Image
        src={slide.thumbnail_url}
        alt={`幻灯片 ${slideNumber}`}
        width={160}
        height={88}
        preview={false}
        style={{
          objectFit: 'cover',
          display: 'block',
        }}
      />
      {/* Selection indicators */}
    </div>
  )
}
```

**Modification point:** Add `loading="lazy"` attribute to Image component for performance optimization.

---

### `frontend/src/pages/results/index.tsx` (page, event-driven) - MODIFY

**Analog:** Self-modification - existing `results/index.tsx`

**Existing imports** (lines 1-54):
```typescript
import { useState, useEffect, useCallback, useRef } from 'react'
import {
  Button,
  Card,
  Row,
  Col,
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
} from '@ant-design/icons'
import { useNavigate, useParams } from 'react-router-dom'
import dayjs from 'dayjs'
import PPTPreview from '../../components/PPTPreview'
import PPTGalleryStrip from '../../components/PPTGalleryStrip'
import MergeSelectionBar from '../../components/MergeSelectionBar'
import TextContentTab from '../../components/TextContentTab'
import DuplicateDetectionPanel from '../../components/DuplicateDetectionPanel'
import SlideCapturePanel from '../../components/SlideCapturePanel'
import { VideoPreviewPanel } from '../../components/VideoPreviewPanel'
```

**Layout structure** (lines 456-498):
```typescript
<Row gutter={24}>
  {/* Left column - Preview area (70%) */}
  <Col span={17}>
    <Card bodyStyle={{ padding: 0 }}>
      <PPTPreview
        slides={slides}
        currentSlide={currentSlide}
        onSlideChange={handleSlideChange}
        isMergeMode={isMergeMode}
        selectedSlides={selectedSlides}
        onToggleSelect={handleToggleSelect}
        isLoading={isLoadingSlides}
        currentPptId={currentPptId}
      />
    </Card>

    {/* Video preview panel */}
    {isVideoPanelVisible && (
      <VideoPreviewPanel
        videoFileId={videoFileIdNum}
        currentSlide={currentSlide + 1}
        onSlideClick={handleVideoSlideChange}
        style={{ marginTop: 16 }}
        autoPlay={false}
        showControls={true}
      />
    )}

    {/* Merge selection bar */}
    {isMergeMode && (
      <MergeSelectionBar
        selectedSlides={selectedSlides}
        onReorder={setSelectedSlides}
        onRemove={handleRemoveSlide}
        onConfirm={handleConfirmMerge}
        onCancel={() => {
          setIsMergeMode(false)
          setSelectedSlides([])
        }}
        isMerging={isMerging}
      />
    )}
  </Col>

  {/* Right column - Info panel (30%) */}
  <Col span={7}>
    {/* Tabbed info panel */}
    <Tabs
      defaultActiveKey="info"
      style={{ marginBottom: 16 }}
      items={[...]}
    />

    {/* Operations button card */}
    <Card title="操作" size="small" style={{ marginBottom: 16 }}>
      <Space direction="vertical" style={{ width: '100%' }}>
        {/* Operation buttons */}
      </Space>
    </Card>
  </Col>
</Row>
```

**State management** (lines 67-94):
```typescript
const [ppts, setPpts] = useState<PPTResult[]>([])
const [currentPptId, setCurrentPptId] = useState<number>(0)
const [slides, setSlides] = useState<SlideImage[]>([])
const [currentSlide, setCurrentSlide] = useState(0)
const [isLoadingSlides, setIsLoadingSlides] = useState(false)
const [isMergeMode, setIsMergeMode] = useState(false)
const [selectedSlides, setSelectedSlides] = useState<SelectedSlide[]>([])
const [isMerging, setIsMerging] = useState(false)
const [videoName, setVideoName] = useState('')
const [loading, setLoading] = useState(false)
const [isCapturePanelOpen, setIsCapturePanelOpen] = useState(false)
const [isVideoPanelVisible, setIsVideoPanelVisible] = useState(true)
```

**Modification point:** Reorganize layout to side-by-side 16:9 previews with left thumbnail sidebar, move operations bar below preview area.

---

## Shared Patterns

### Video Ref and Playback Control
**Source:** `frontend/src/components/VideoPreviewPanel.tsx` (lines 60, 193-217)
**Apply to:** `PlaybackSpeedControl.tsx`, `SlideCapturePanel.tsx` (direct capture)
```typescript
const videoRef = useRef<HTMLVideoElement>(null)

// Control functions
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

// Playback rate control
const handleSpeedChange = useCallback((rate: number) => {
  const video = videoRef.current
  if (!video) return

  video.playbackRate = rate
  setPlaybackRate(rate)
}, [])
```

### Ant Design Select Component
**Source:** `frontend/src/components/SlideCapturePanel.tsx` (lines 271-276)
**Apply to:** `PlaybackSpeedControl.tsx`
```typescript
<Select
  value={value}
  onChange={handleChange}
  options={options}
  style={{ width: 100 }}
  size="small"
/>
```

### API Call Pattern with Loading State
**Source:** `frontend/src/components/SlideCapturePanel.tsx` (lines 95-114)
**Apply to:** `SlideCapturePanel.tsx` (direct capture), `OperationsBar.tsx` (if needed)
```typescript
const [isLoading, setIsLoading] = useState(false)

const handleApiCall = async () => {
  setIsLoading(true)
  try {
    const response = await apiFunction(params)
    if (response.data?.success) {
      message.success('操作成功')
      // Handle success
    } else {
      message.error('操作失败')
    }
  } catch (error) {
    console.error('API call failed:', error)
    message.error('操作失败: ' + (error as Error).message)
  } finally {
    setIsLoading(false)
  }
}
```

### Lazy Loading Images
**Source:** HTML5 native lazy loading (browser API)
**Apply to:** `SlideThumbnail.tsx`
```typescript
<img
  src={url}
  loading="lazy"
  alt={alt}
  style={{ /* ... */ }}
/>

// Or with Ant Design Image component
<Image
  src={url}
  loading="lazy"
  preview={false}
  style={{ /* ... */ }}
/>
```

### CSS Grid Side-by-Side Layout
**Source:** CSS Grid specification + aspect-ratio property
**Apply to:** `results/index.tsx` (preview area layout)
```typescript
const previewAreaStyle: React.CSSProperties = {
  display: 'grid',
  gridTemplateColumns: '1fr 1fr',
  gap: '16px',
  marginBottom: '16px',
}

const previewBoxStyle: React.CSSProperties = {
  position: 'relative',
  width: '100%',
  aspectRatio: '16 / 9',
  backgroundColor: '#000',
  borderRadius: '8px',
  overflow: 'hidden',
}
```

### Message Feedback Pattern
**Source:** Throughout codebase (Ant Design message API)
**Apply to:** All new/modified components
```typescript
import { message } from 'antd'

// Success
message.success('操作成功')

// Error
message.error('操作失败: ' + error.message)

// Warning
message.warning('请先完成前置步骤')

// Loading
message.loading('处理中...', 0) // 0 = no auto-dismiss
```

### Button with Loading State
**Source:** `frontend/src/components/MergeSelectionBar.tsx` (lines 199-206)
**Apply to:** `OperationsBar.tsx`, direct capture button
```typescript
<Button
  type="primary"
  onClick={handleClick}
  disabled={isLoading}
  loading={isLoading}
  icon={isLoading ? <Spin size="small" /> : <CheckOutlined />}
>
  {isLoading ? '处理中...' : '确认操作'}
</Button>
```

### Component Callback Pattern
**Source:** All existing components
**Apply to:** All new components
```typescript
interface ComponentProps {
  onEvent?: (data: SomeType) => void
  onComplete?: (result: ResultType) => void
  onCancel?: () => void
}

// Usage
const handleEvent = useCallback((data: SomeType) => {
  // Handle event
  if (onEvent) {
    onEvent(processedData)
  }
}, [onEvent])
```

### Style Object Pattern
**Source:** All existing components (inline styles)
**Apply to:** All new components
```typescript
const containerStyle: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
  padding: 16,
  backgroundColor: '#ffffff',
}

// Or inline
<div style={{ display: 'flex', gap: 8 }}>
  {/* Content */}
</div>
```

### Type Import Pattern
**Source:** All existing components
**Apply to:** All new components
```typescript
import type { SlideImage, SelectedSlide } from '../types/ppt'
import type { SlideTimestamp } from '../types/transcription'
```

## No Analog Found

None - All files have exact or role-match analogs in the existing codebase.

## Metadata

**Analog search scope:**
- `frontend/src/components/*.tsx` (17 component files)
- `frontend/src/pages/**/*.tsx` (13 page files)
- `frontend/src/api/*.ts` (12 API files)
- `frontend/src/types/*.ts` (11 type files)

**Files scanned:** 53
**Pattern extraction date:** 2026-04-20

**Key findings:**
1. **Video playback speed control:** No existing speed control component, but `VideoPreviewPanel.tsx` provides video ref and control patterns
2. **Operations bar:** `MergeSelectionBar.tsx` provides excellent analog for horizontal bar layout with buttons
3. **Select component:** `SlideCapturePanel.tsx` has Select usage for position options
4. **API patterns:** `SlideCapturePanel.tsx` has complete capture/insert API call patterns
5. **Layout patterns:** `results/index.tsx` has existing Row/Col layout that needs reorganization
6. **Thumbnail optimization:** `SlideThumbnail.tsx` needs lazy loading attribute added
7. **All components use:** Ant Design components, TypeScript interfaces, React hooks (useState, useCallback, useRef, useEffect)
