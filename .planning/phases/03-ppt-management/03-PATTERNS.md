# Phase 3: PPT Management - Pattern Map

**Mapped:** 2026-04-17
**Files analyzed:** 16 new files, 3 extended files
**Analogs found:** 18 / 19

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/ppt_file.go` | model | CRUD | `internal/models/transcription_task.go` | exact |
| `internal/models/slide_merge.go` | model | request-response | `internal/models/transcription_task.go` | role-match |
| `internal/services/slide_extractor.go` | service | file-I/O | `internal/services/pptx_generator.go` | exact |
| `internal/services/slide_cache_service.go` | service | CRUD | `internal/services/video_file_service.go` | role-match |
| `internal/services/ppt_merge_service.go` | service | event-driven | `internal/services/transcription_service.go` | role-match |
| `internal/services/ppt_file_service.go` | service | CRUD | `internal/services/video_file_service.go` | exact |
| `internal/handlers/ppt_handler.go` | handler | request-response | `internal/handlers/transcription_handler.go` | exact |
| `internal/migrations/[timestamp]_add_ppt_cache_fields.go` | migration | CRUD | `internal/migrations/001_add_video_file_owner.go` | exact |
| `frontend/src/api/ppt.ts` | api-client | request-response | `frontend/src/api/transcription.ts` | exact |
| `frontend/src/pages/results/[videoFileId]/index.tsx` | page | request-response | `frontend/src/pages/files/index.tsx` | role-match |
| `frontend/src/components/PPTPreview.tsx` | component | request-response | `frontend/src/components/TranscriptionProgressModal.tsx` | role-match |
| `frontend/src/components/PPTGalleryStrip.tsx` | component | event-driven | `frontend/src/components/TimelineWithMarkers.tsx` | partial |
| `frontend/src/components/MergeSelectionBar.tsx` | component | event-driven | `frontend/src/components/TimelineWithMarkers.tsx` | partial |
| `frontend/src/components/SlideThumbnail.tsx` | component | event-driven | `frontend/src/components/VideoPlayerSimple.tsx` | partial |
| `frontend/src/types/ppt.ts` | types | N/A | `frontend/src/types/transcription.ts` | exact |
| `scripts/extract_slides.py` | script | file-I/O | `scripts/create_pptx.py` | exact |
| `scripts/merge_slides.py` | script | file-I/O | `scripts/create_pptx.py` | exact |

## Pattern Assignments

### `internal/models/ppt_file.go` (model, CRUD) - EXTEND

**Analog:** `internal/models/transcription_task.go`

**Imports pattern** (lines 1-2):
```go
package models

import (
    "gorm.io/gorm"
)
```

**Model struct pattern** (lines 4-19):
```go
type TranscriptionTask struct {
    Base
    VideoFileID       uint       `gorm:"not null;index" json:"video_file_id"`
    VideoFile         *VideoFile `gorm:"foreignKey:VideoFileID" json:"video_file,omitempty"`
    SamplingRate      float64    `gorm:"default:0.5" json:"sampling_rate"`
    Status            string     `gorm:"type:varchar(20);default:'pending';index" json:"status"`
    ResultPPTFileID   *uint      `json:"result_ppt_file_id,omitempty"`
    ResultPPTFile     *PPTFile   `gorm:"foreignKey:ResultPPTFileID" json:"result_ppt_file,omitempty"`
    ErrorMessage      string     `gorm:"type:text" json:"error_message,omitempty"`
    CreatedBy         uint       `gorm:"not null" json:"created_by"`
    Creator           *User      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName 指定表名
func (TranscriptionTask) TableName() string {
    return "transcription_tasks"
}
```

**Extension fields to add:**
- `SlideCachePath string` - Path to extracted slide images cache directory
- `SourceType string` - Enum: "transcription" or "merge"
- `MergedFrom string` - JSON array of source PPT IDs for merge results

---

### `internal/models/slide_merge.go` (model, request-response) - NEW

**Analog:** `internal/models/transcription_task.go`

**Request struct pattern** (lines 44-46):
```go
type TranscriptionTriggerRequest struct {
    SamplingRate float64 `json:"sampling_rate"`
}
```

**Create new struct:**
```go
type MergeRequest struct {
    SlideIDs    []string `json:"slide_ids" binding:"required"`    // Format: "pptID_slideNumber"
    OutputName  string   `json:"output_name" binding:"required"`   // Output filename
    VideoFileID uint     `json:"video_file_id" binding:"required"` // For ownership validation
}
```

---

### `internal/services/slide_extractor.go` (service, file-I/O) - NEW

**Analog:** `internal/services/pptx_generator.go`

**Service struct pattern** (lines 16-19):
```go
type PPTXGenerator struct {
    logger       *zap.Logger
    pythonScript string
}
```

**Constructor pattern** (lines 22-30):
```go
func NewPPTXGenerator(logger *zap.Logger) *PPTXGenerator {
    projectRoot := getProjectRoot()
    return &PPTXGenerator{
        logger:       logger,
        pythonScript: filepath.Join(projectRoot, "scripts", "create_pptx.py"),
    }
}
```

**Python execution pattern** (lines 113-127):
```go
cmdName := "python3"
if _, err := exec.LookPath("python3"); err != nil {
    cmdName = "python"
}
cmd := exec.CommandContext(ctx, cmdName, args...)
output, err := cmd.CombinedOutput()
if err != nil {
    g.logger.Error("Python script failed",
        zap.String("output", string(output)),
        zap.Error(err))
    return 0, fmt.Errorf("failed to generate PPTX: %w", err)
}
```

**JSON response parsing** (lines 130-136):
```go
var result pythonResult
if err := json.Unmarshal(output, &result); err != nil {
    g.logger.Error("Failed to parse Python output",
        zap.String("output", string(output)),
        zap.Error(err))
    return 0, fmt.Errorf("failed to parse Python output: %w", err)
}
```

**Path validation pattern** (lines 206-227):
```go
func (g *PPTXGenerator) validatePath(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return fmt.Errorf("cannot resolve absolute path: %w", err)
    }

    if strings.ContainsAny(path, "\n\r\t") {
        return fmt.Errorf("path contains invalid characters")
    }

    projectRoot := getProjectRoot()
    allowedDir := filepath.Clean(projectRoot)
    if !strings.HasPrefix(absPath, allowedDir) {
        return fmt.Errorf("path outside allowed directory: %s", path)
    }

    return nil
}
```

---

### `internal/services/slide_cache_service.go` (service, CRUD) - NEW

**Analog:** `internal/services/video_file_service.go`

**Service struct pattern** (from video_file_service.go):
```go
type VideoFileService struct {
    db     *gorm.DB
    logger *zap.Logger
    config *config.Config
}
```

**Constructor pattern**:
```go
func NewVideoFileService(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *VideoFileService {
    return &VideoFileService{
        db:     db,
        logger: logger,
        config: cfg,
    }
}
```

**CRUD query pattern** (from video_file_service.go):
```go
func (s *VideoFileService) GetFileByID(id uint) (*models.VideoFile, error) {
    var file models.VideoFile
    err := s.db.Where("id = ?", id).First(&file).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, fmt.Errorf("文件不存在")
        }
        return nil, err
    }
    return &file, nil
}
```

---

### `internal/services/ppt_merge_service.go` (service, event-driven) - NEW

**Analog:** `internal/services/transcription_service.go`

**Worker pool pattern** (lines 32-50):
```go
type TranscriptionService struct {
    db                 *gorm.DB
    logger             *zap.Logger
    config             *config.Config
    taskQueue          chan *models.TranscriptionTask
    workers            int
    cancelFuncs        map[uint]context.CancelFunc
    mu                 sync.RWMutex
    wg                 sync.WaitGroup
    ctx                context.Context
    cancel             context.CancelFunc
    pptxGenerator      *PPTXGenerator
}
```

**Context with timeout pattern** (lines 222-223):
```go
ctx, cancel := context.WithTimeout(s.ctx, 10*time.Minute)
defer cancel()
```

**Database transaction pattern** (lines 376-393):
```go
pptFile := &models.PPTFile{
    FileName:            fmt.Sprintf("%s_转录.pptx", strings.TrimSuffix(videoFile.FileName, filepath.Ext(videoFile.FileName))),
    FilePath:            pptxOutputPath,
    FileSize:            fileInfo.Size(),
    PageCount:           pageCount,
    Format:              "pptx",
    SourceVideoFileID:   &task.VideoFileID,
    TranscriptionTaskID: &task.ID,
}
if err := s.db.Create(pptFile).Error; err != nil {
    s.logger.Error("创建PPT文件记录失败",
        zap.Uint("video_file_id", task.VideoFileID),
        zap.Error(err))
    return
}
```

---

### `internal/services/ppt_file_service.go` (service, CRUD) - EXTEND

**Analog:** `internal/services/video_file_service.go`

**Add methods following this pattern:**
```go
func (s *VideoFileService) GetFileByID(id uint) (*models.VideoFile, error) {
    var file models.VideoFile
    err := s.db.Where("id = ?", id).First(&file).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, fmt.Errorf("文件不存在")
        }
        return nil, err
    }
    return &file, nil
}
```

**New methods needed:**
- `GetSlides(pptFileID uint) ([]SlideImage, error)` - Query slide cache
- `GetPptsByVideoFile(videoFileID uint) ([]models.PPTFile, error)` - Query all PPT results for a video
- `CreatePPTFile(file *models.PPTFile) error` - Create PPT record
- `UpdatePPTFile(id uint, updates map[string]interface{}) error` - Update PPT record

---

### `internal/handlers/ppt_handler.go` (handler, request-response) - NEW

**Analog:** `internal/handlers/transcription_handler.go`

**Handler struct pattern** (lines 14-18):
```go
type TranscriptionHandler struct {
    transcriptionService *services.TranscriptionService
    videoFileService     *services.VideoFileService
    logger               *zap.Logger
}
```

**Constructor pattern** (lines 21-31):
```go
func NewTranscriptionHandler(
    transcriptionService *services.TranscriptionService,
    videoFileService *services.VideoFileService,
    logger *zap.Logger,
) *TranscriptionHandler {
    return &TranscriptionHandler{
        transcriptionService: transcriptionService,
        videoFileService:     videoFileService,
        logger:               logger,
    }
}
```

**Request handling pattern** (lines 36-72):
```go
func (h *TranscriptionHandler) SubmitTranscription(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
        return
    }

    var req struct {
        SamplingRate float64 `json:"sampling_rate"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        req.SamplingRate = 0.5 // Default
    }

    // Validate sampling rate
    validRates := map[float64]bool{1.0: true, 0.5: true, 0.2: true}
    if !validRates[req.SamplingRate] {
        response.GinError(c, response.CodeInvalidRequest, "无效的采样率，必须是 1.0, 0.5 或 0.2")
        return
    }

    // Verify user owns the video file
    userID := middleware.GetUserID(c)
    file, err := h.videoFileService.GetFileByID(uint(id))
    if err != nil {
        response.GinError(c, response.CodeInternalError, "视频文件不存在")
        return
    }
    if !middleware.GetIsAdmin(c) && file.CreatedBy != userID {
        response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
        return
    }

    if err := h.transcriptionService.SubmitTranscription(uint(id), req.SamplingRate, userID); err != nil {
        response.GinError(c, response.CodeInternalError, "提交转录任务失败: "+err.Error())
        return
    }

    response.GinSuccess(c, gin.H{
        "video_file_id": id,
        "status":        "processing",
        "sampling_rate": req.SamplingRate,
    })
}
```

**Success response pattern** (lines 79-84):
```go
response.GinSuccess(c, gin.H{
    "video_file_id": id,
    "status":        "processing",
    "sampling_rate": req.SamplingRate,
})
```

**Error response pattern** (lines 39, 55, 69, 75):
```go
response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
response.GinError(c, response.CodeInvalidRequest, "无效的采样率")
response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
response.GinError(c, response.CodeInternalError, "提交转录任务失败: "+err.Error())
```

---

### `internal/migrations/[timestamp]_add_ppt_cache_fields.go` (migration, CRUD) - NEW

**Analog:** `internal/migrations/001_add_video_file_owner.go`

**Migration struct pattern** (lines 9-14):
```go
type AddVideoFileOwnerMigration struct{}

func (m *AddVideoFileOwnerMigration) Name() string {
    return "001_add_video_file_owner"
}
```

**Up migration pattern** (lines 16-42):
```go
func (m *AddVideoFileOwnerMigration) Up(db *gorm.DB) error {
    // 使用 PRAGMA table_info 检查列是否已存在（更可靠）
    var columnName string
    checkErr := db.Raw("SELECT name FROM pragma_table_info('video_files') WHERE name = 'created_by'").Scan(&columnName).Error

    // 如果查询成功且找到了列，说明已存在，跳过迁移
    if checkErr == nil && columnName != "" {
        return nil
    }

    // 列不存在，执行添加
    addResult := db.Exec("ALTER TABLE video_files ADD COLUMN created_by INTEGER NOT NULL DEFAULT 1")
    if addResult.Error != nil {
        // 检查是否是"重复列名"错误，如果是则忽略（幂等）
        if addResult.Error != nil && len(addResult.Error.Error()) > 0 {
            errStr := addResult.Error.Error()
            if contains(errStr, "duplicate column name") {
                return nil // 列已存在，忽略错误
            }
        }
        return addResult.Error
    }

    return nil
}
```

**Down migration pattern** (lines 44-48):
```go
func (m *AddVideoFileOwnerMigration) Down(db *gorm.DB) error {
    // SQLite 不支持 DROP COLUMN，需要重建表
    // 这里简单处理：不执行回滚
    return nil
}
```

**Helper functions** (lines 51-63):
```go
func contains(s, substr string) bool {
    return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
        indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
    for i := 0; i <= len(s)-len(substr); i++ {
        if s[i:i+len(substr)] == substr {
            return i
        }
    }
    return -1
}
```

**Migration registration** (lines 259-266):
```go
func GetRegisteredMigrations() []interface{} {
    return []interface{}{
        &AddVideoFileOwnerMigration{},
        &AddStreamConfigMigration{},
        &AddSegmentFieldsMigration{},
        &CreateTranscriptionTasksMigration{},
    }
}
```

---

### `frontend/src/api/ppt.ts` (api-client, request-response) - NEW

**Analog:** `frontend/src/api/transcription.ts`

**Imports pattern** (lines 1-9):
```typescript
import type { ApiResponse } from '../types/auth'
import type {
  TranscriptionTriggerRequest,
  TranscriptionTriggerResponse,
  TranscriptionStatusResponse
} from '../types/transcription'
import { apiRequest } from './apiClient'
```

**API function pattern** (lines 12-23):
```typescript
export async function submitTranscription(
  videoFileId: number,
  samplingRate: number = 0.5
): Promise<ApiResponse<TranscriptionTriggerResponse>> {
  return apiRequest<TranscriptionTriggerResponse>(
    `/api/v1/videos/${videoFileId}/transcribe`,
    {
      method: 'POST',
      body: JSON.stringify({ sampling_rate: samplingRate }),
    }
  )
}
```

**GET request pattern** (lines 26-32):
```typescript
export async function getTranscriptionStatus(
  videoFileId: number
): Promise<ApiResponse<TranscriptionStatusResponse>> {
  return apiRequest<TranscriptionStatusResponse>(
    `/api/v1/videos/${videoFileId}/transcription-status`
  )
}
```

---

### `frontend/src/pages/results/[videoFileId]/index.tsx` (page, request-response) - NEW

**Analog:** `frontend/src/pages/files/index.tsx`

**Component imports pattern** (lines 1-49):
```typescript
import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Table,
  Button,
  Space,
  Modal,
  message,
  Popconfirm,
  Tag,
  Card,
  Statistic,
  Row,
  Col,
  Tooltip,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { useNavigate } from 'react-router-dom'
import * as videoFileApi from '../../api/video-file'
import { submitTranscription } from '../../api/transcription'
```

**State management pattern** (lines 86-109):
```typescript
const [files, setFiles] = useState<VideoFile[]>([])
const [stats, setStats] = useState<VideoFileStats | null>(null)
const [total, setTotal] = useState(0)
const [loading, setLoading] = useState(false)
const [detailVisible, setDetailVisible] = useState(false)
const [viewingFile, setViewingFile] = useState<VideoFile | null>(null)
```

**Data fetching pattern** (lines 112-127):
```typescript
const loadFiles = useCallback(async (showLoading = true) => {
  if (showLoading) setLoading(true)
  try {
    const response = await videoFileApi.getVideoFileList(params)
    if (response.data) {
      setFiles(response.data.items)
      setTotal(response.data.total)
    }
  } catch (error) {
    if (showLoading) {
      message.error(error instanceof Error ? error.message : '加载文件列表失败')
    }
  } finally {
    if (showLoading) setLoading(false)
  }
}, [params])
```

**Effect for initial load** (lines 142-144):
```typescript
useEffect(() => {
  Promise.all([loadFiles(), loadStats()])
}, [loadFiles, loadStats])
```

**Navigation pattern** (line 87, 309):
```typescript
const navigate = useNavigate()
// ...
onClick={() => navigate(`/split/${record.id}`)}
```

---

### `frontend/src/components/PPTPreview.tsx` (component, request-response) - NEW

**Analog:** `frontend/src/components/TranscriptionProgressModal.tsx`

**Component props pattern** (lines 15-22):
```typescript
interface TranscriptionProgressModalProps {
  open: boolean
  onClose: () => void
  videoFileId: number
  fileName: string
  samplingRate: number
  onCompleted: (pptFileId: number) => void
}
```

**State management pattern** (lines 39-46):
```typescript
const [status, setStatus] = useState<TranscriptionTaskStatus>('pending')
const [stage, setStage] = useState<TranscriptionStage | ''>('')
const [framesProcessed, setFramesProcessed] = useState(0)
const [totalFrames, setTotalFrames] = useState(0)
const [percentage, setPercentage] = useState(0)
```

**Polling effect pattern** (lines 49-83):
```typescript
useEffect(() => {
  if (!open) return

  const fetchStatus = async () => {
    try {
      const response = await getTranscriptionStatus(videoFileId)
      if (response.data) {
        const data = response.data
        setStatus(data.status)
        setStage(data.current_stage)
        // ... update other state
      }
    } catch (error) {
      console.error('获取转录状态失败:', error)
    }
  }

  fetchStatus()
  const interval = setInterval(fetchStatus, 5000)
  return () => clearInterval(interval)
}, [open, videoFileId, onCompleted])
```

**Modal structure pattern** (lines 147-161):
```typescript
return (
  <Modal
    title={`本地转录进度 - ${fileName}`}
    open={open}
    onCancel={onClose}
    footer={[
      <Button key="retry" icon={<ReloadOutlined />} onClick={handleRetry}>
        重新转录
      </Button>,
      <Button key="download" type="primary" icon={<DownloadOutlined />} onClick={handleDownloadPpt}>
        下载PPT
      </Button>,
    ]}
    width={600}
  >
    {/* Content */}
  </Modal>
)
```

---

### `frontend/src/components/PPTGalleryStrip.tsx` (component, event-driven) - NEW

**Analog:** `frontend/src/components/TimelineWithMarkers.tsx`

**Component pattern for horizontal strip:**
```typescript
interface PPTGalleryStripProps {
  ppts: PPTResult[]
  currentPptId: number
  onSelect: (pptId: number) => void
}

export default function PPTGalleryStrip({ ppts, currentPptId, onSelect }: PPTGalleryStripProps) {
  return (
    <div style={{ display: 'flex', gap: 8, overflowX: 'auto', padding: '12px 0' }}>
      {ppts.map((ppt) => (
        <div
          key={ppt.id}
          onClick={() => onSelect(ppt.id)}
          style={{
            cursor: 'pointer',
            border: currentPptId === ppt.id ? '2px solid #1890ff' : '2px solid transparent',
            borderRadius: 4,
            padding: 8,
          }}
        >
          {/* Thumbnail and info */}
        </div>
      ))}
    </div>
  )
}
```

---

### `frontend/src/components/MergeSelectionBar.tsx` (component, event-driven) - NEW

**Analog:** `frontend/src/components/TimelineWithMarkers.tsx`

**State management for drag-to-reorder:**
```typescript
const [selectedSlides, setSelectedSlides] = useState<SelectedSlide[]>([])
const [draggedIndex, setDraggedIndex] = useState<number | null>(null)

const handleReorder = useCallback((slides: SelectedSlide[]) => {
  setSelectedSlides(slides)
}, [])
```

**Drag handlers pattern:**
```typescript
const handleDragStart = (index: number) => {
  setDraggedIndex(index)
}

const handleDragOver = (e: React.DragEvent, index: number) => {
  e.preventDefault()
  if (draggedIndex === null || draggedIndex === index) return

  const newSlides = [...selectedSlides]
  const [draggedItem] = newSlides.splice(draggedIndex, 1)
  newSlides.splice(index, 0, draggedItem)

  setSelectedSlides(newSlides)
  setDraggedIndex(index)
}

const handleDragEnd = () => {
  setDraggedIndex(null)
}
```

---

### `frontend/src/components/SlideThumbnail.tsx` (component, event-driven) - NEW

**Analog:** `frontend/src/components/VideoPlayerSimple.tsx`

**Component props pattern:**
```typescript
interface SlideThumbnailProps {
  slide: SlideImage
  isSelected: boolean
  isSelectable: boolean
  onClick: () => void
  onDownload?: () => void
  onCopy?: () => void
}

export default function SlideThumbnail({ slide, isSelected, isSelectable, onClick, onDownload, onCopy }: SlideThumbnailProps) {
  return (
    <div
      onClick={isSelectable ? onClick : undefined}
      style={{
        position: 'relative',
        cursor: isSelectable ? 'pointer' : 'default',
        border: isSelected ? '2px solid #1890ff' : '2px solid transparent',
        borderRadius: 4,
        overflow: 'hidden',
      }}
    >
      <Image src={slide.thumbnail_url} width={160} height={90} preview={false} />
      {isSelected && (
        <div style={{
          position: 'absolute',
          top: 4,
          right: 4,
          background: '#1890ff',
          color: 'white',
          borderRadius: '50%',
          width: 20,
          height: 20,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          fontSize: 12,
        }}>✓</div>
      )}
    </div>
  )
}
```

---

### `frontend/src/types/ppt.ts` (types, N/A) - NEW

**Analog:** `frontend/src/types/transcription.ts`

**Type definition pattern** (lines 1-38):
```typescript
// 转录触发请求
export interface TranscriptionTriggerRequest {
  sampling_rate: number
}

// 转录触发响应
export interface TranscriptionTriggerResponse {
  video_file_id: number
  status: string
}

// 转录进度阶段
export type TranscriptionStage = 'extracting' | 'detecting' | 'generating'

// 转录任务状态
export type TranscriptionTaskStatus = 'pending' | 'processing' | 'completed' | 'failed'

// 转录状态响应
export interface TranscriptionStatusResponse {
  status: TranscriptionTaskStatus
  current_stage: TranscriptionStage | ''
  frames_processed: number
  total_frames: number
  percentage: number
  error_message: string
  result_ppt_file_id: number | null
}

// 采样率选项
export interface SamplingRateOption {
  label: string
  value: number
  secondsPerFrame: number
  description: string
}
```

**New types needed:**
```typescript
export interface SlideImage {
  slide_number: number
  thumbnail_url: string
  fullsize_url: string
}

export interface PPTResult {
  id: number
  file_name: string
  page_count: number
  file_size: number
  source_type: 'transcription' | 'merge'
  created_at: string
  slides?: SlideImage[]
}

export interface MergeRequest {
  slide_ids: string[]  // Format: "pptID_slideNumber"
  output_name: string
  video_file_id: number
}

export interface MergeResponse {
  ppt_file_id: number
  file_name: string
  page_count: number
}
```

---

### `scripts/extract_slides.py` (script, file-I/O) - NEW

**Analog:** `scripts/create_pptx.py`

**Script structure pattern** (lines 1-18):
```python
#!/usr/bin/env python3
"""
Create PowerPoint files from image frames.

Usage:
    python3 create_pptx.py <output_path> <image1> <image2> ...

Output:
    JSON: {"success": true, "page_count": N, "output_path": "..."}
"""

import sys
import json
import os
from pptx import Presentation
from pptx.util import Inches
```

**Function pattern** (lines 27-117):
```python
def create_pptx_from_images(image_paths, output_path):
    """
    Create a PowerPoint file from a list of images.

    Args:
        image_paths: List of image file paths
        output_path: Output PPTX file path

    Returns:
        Tuple (success: bool, result_dict: dict, exit_code: int)
    """
    try:
        # Validate inputs
        if not image_paths:
            result = {
                "success": False,
                "error": "No image paths provided"
            }
            return False, result, 1

        # Process images
        # ...

        # Return success result
        result = {
            "success": True,
            "page_count": page_count,
            "output_path": output_path,
        }
        return True, result, 0

    except Exception as e:
        result = {
            "success": False,
            "error": str(e)
        }
        return False, result, 1
```

**Main entry point** (lines 120-144):
```python
def main():
    if len(sys.argv) < 3:
        print(json.dumps({
            "success": False,
            "error": "Usage: create_pptx.py <output_path> <image1> <image2> ..."
        }), file=sys.stderr)
        sys.exit(1)

    output_path = sys.argv[1]
    image_paths = sys.argv[2:]

    success, result, exit_code = create_pptx_from_images(image_paths, output_path)

    if success:
        print(json.dumps(result))
    else:
        print(json.dumps(result), file=sys.stderr)

    sys.exit(exit_code)
```

---

### `scripts/merge_slides.py` (script, file-I/O) - NEW

**Analog:** `scripts/create_pptx.py`

**Same structure as extract_slides.py, with merge-specific logic:**

```python
def merge_slides(source_pptx_paths, output_path):
    """
    Merge slides from multiple PPTX files into a single presentation.

    Args:
        source_pptx_paths: List of source PPTX file paths
        output_path: Output merged PPTX file path

    Returns:
        Tuple (success: bool, result_dict: dict, exit_code: int)
    """
    try:
        # Create output presentation
        output_prs = Presentation()
        slides_merged = 0

        for pptx_path in source_pptx_paths:
            if not os.path.exists(pptx_path):
                continue

            source_prs = Presentation(pptx_path)

            # Copy each slide
            for slide in source_prs.slides:
                blank_layout = output_prs.slide_layouts[6]
                output_slide = output_prs.slides.add_slide(blank_layout)

                # Copy shapes (implementation depends on requirements)
                # ...

                slides_merged += 1

        # Save merged presentation
        output_prs.save(output_path)

        result = {
            "success": True,
            "slides_merged": slides_merged,
            "output_path": output_path
        }
        return True, result, 0

    except Exception as e:
        result = {
            "success": False,
            "error": str(e)
        }
        return False, result, 1
```

---

## Shared Patterns

### Authentication/Authorization

**Source:** `internal/middleware` (used in handlers)
**Apply to:** All controller files

```go
// Get user ID from context
userID := middleware.GetUserID(c)

// Check admin status
if !middleware.GetIsAdmin(c) && file.CreatedBy != userID {
    response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
    return
}
```

### Error Handling

**Source:** `pkg/response` (used throughout handlers)
**Apply to:** All service and handler files

```go
// Success response
response.GinSuccess(c, gin.H{
    "video_file_id": id,
    "status":        "processing",
})

// Error response
response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
response.GinError(c, response.CodeInternalError, "提交转录任务失败: "+err.Error())
```

### Input Validation

**Source:** `internal/handlers/transcription_handler.go`
**Apply to:** All handler POST/PUT endpoints

```go
// Parse and validate parameter
idStr := c.Param("id")
id, err := strconv.ParseUint(idStr, 10, 32)
if err != nil {
    response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
    return
}

// Parse and validate JSON body
var req struct {
    SamplingRate float64 `json:"sampling_rate"`
}
if err := c.ShouldBindJSON(&req); err != nil {
    req.SamplingRate = 0.5 // Default
}

// Validate business rules
validRates := map[float64]bool{1.0: true, 0.5: true, 0.2: true}
if !validRates[req.SamplingRate] {
    response.GinError(c, response.CodeInvalidRequest, "无效的采样率")
    return
}
```

### Logging

**Source:** `go.uber.org/zap` (used throughout services)
**Apply to:** All service files

```go
// Info logging
s.logger.Info("转录任务已提交",
    zap.Uint("video_file_id", videoFileID),
    zap.Uint("task_id", task.ID),
    zap.Float64("sampling_rate", samplingRate))

// Error logging
s.logger.Error("创建PPT文件记录失败",
    zap.Uint("video_file_id", task.VideoFileID),
    zap.Error(err))
```

### React State Management

**Source:** `frontend/src/pages/files/index.tsx`
**Apply to:** All page components

```typescript
const [files, setFiles] = useState<VideoFile[]>([])
const [loading, setLoading] = useState(false)

const loadFiles = useCallback(async (showLoading = true) => {
  if (showLoading) setLoading(true)
  try {
    const response = await videoFileApi.getVideoFileList(params)
    if (response.data) {
      setFiles(response.data.items)
    }
  } catch (error) {
    if (showLoading) {
      message.error(error instanceof Error ? error.message : '加载失败')
    }
  } finally {
    if (showLoading) setLoading(false)
  }
}, [params])
```

### API Client Pattern

**Source:** `frontend/src/api/apiClient.ts`
**Apply to:** All API client files

```typescript
export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = `${API_BASE_URL}${endpoint}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
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

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `frontend/src/components/PPTGalleryStrip.tsx` | component | event-driven | No horizontal gallery strip component exists yet |
| `frontend/src/components/MergeSelectionBar.tsx` | component | event-driven | No drag-to-reorder selection bar exists yet |
| `frontend/src/components/SlideThumbnail.tsx` | component | event-driven | No selectable thumbnail component exists yet |

**Note:** These components should use Ant Design components (Image, Space, Button) and dnd-kit for drag-and-drop as specified in RESEARCH.md.

---

## Metadata

**Analog search scope:**
- `internal/models/*.go`
- `internal/services/*.go`
- `internal/handlers/*.go`
- `internal/migrations/*.go`
- `frontend/src/api/*.ts`
- `frontend/src/pages/**/*.tsx`
- `frontend/src/components/*.tsx`
- `frontend/src/types/*.ts`
- `scripts/*.py`

**Files scanned:** 45
**Pattern extraction date:** 2026-04-17
**Primary analog sources:**
- `internal/models/transcription_task.go`
- `internal/services/pptx_generator.go`
- `internal/services/transcription_service.go`
- `internal/handlers/transcription_handler.go`
- `internal/migrations/001_add_video_file_owner.go`
- `frontend/src/api/transcription.ts`
- `frontend/src/pages/files/index.tsx`
- `frontend/src/components/TranscriptionProgressModal.tsx`
- `scripts/create_pptx.py`
