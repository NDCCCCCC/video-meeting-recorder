# Phase 2: Local Transcription - Pattern Map

**Mapped:** 2026-04-17
**Files analyzed:** 11
**Analogs found:** 11 / 11

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/transcription_task.go` | model | CRUD | `internal/models/video_file.go` | exact |
| `internal/services/transcription_service.go` | service | event-driven | `internal/services/splitting_service.go` | exact |
| `internal/services/similarity_detector.go` | service | transform | `internal/services/conversion_service.go` | role-match |
| `internal/services/frame_extractor.go` | service | file-I/O | `internal/services/conversion_service.go` | role-match |
| `internal/services/pptx_generator.go` | service | file-I/O | `internal/models/ppt_file.go` | role-match |
| `internal/handlers/transcription_handler.go` | handler | request-response | `internal/handlers/split_handler.go` | exact |
| `internal/migrations/[timestamp]_create_transcription_tasks.go` | migration | CRUD | `internal/migrations/001_add_video_file_owner.go` | role-match |
| `frontend/src/api/transcription.ts` | api-client | request-response | `frontend/src/api/split.ts` | exact |
| `frontend/src/pages/files/index.tsx` | component | request-response | `frontend/src/pages/files/index.tsx` | modify |
| `frontend/src/components/TranscriptionProgressModal.tsx` | component | event-driven | `frontend/src/pages/split/index.tsx` | role-match |
| `frontend/src/types/transcription.ts` | types | CRUD | `frontend/src/types/video-file.ts` | exact |

## Pattern Assignments

### `internal/models/transcription_task.go` (model, CRUD)

**Analog:** `internal/models/video_file.go`

**Imports pattern** (lines 1-7):
```go
package models

import (
	"fmt"
	"os"
	"time"
)
```

**Model structure pattern** (lines 10-31):
```go
// VideoFile 视频文件模型
type VideoFile struct {
	Base
	FileName       string              `gorm:"type:varchar(200);not null" json:"file_name"`
	FilePath       string              `gorm:"type:varchar(500);not null;uniqueIndex:idx_file_path" json:"file_path"`
	FileSize       int64               `gorm:"default:0" json:"file_size"`
	Duration       int                 `gorm:"default:0" json:"duration"`
	Format         string              `gorm:"type:varchar(20)" json:"format"`
	TaskID         *uint               `gorm:"index" json:"task_id,omitempty"`
	Task           *VideoRecordingTask `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	ParentID       *uint               `gorm:"index" json:"parent_id,omitempty"`
	CreatedBy      uint                `gorm:"not null" json:"created_by"`
	Creator        *User               `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Status         string              `gorm:"type:varchar(20);default:'ready';index" json:"status"`
}
```

**Status constants pattern** (lines 34-46):
```go
// 文件状态常量
const (
	FileStatusReady      = "ready"
	FileStatusProcessing = "processing"
	FileStatusError      = "error"
	FileStatusDeleting   = "deleting"
)

// 视频来源类型常量
const (
	SourceTypeRecording = "recording"
	SourceTypeSnapshot  = "snapshot"
	SourceTypeSplit     = "split"
)
```

**Helper methods pattern** (lines 49-83):
```go
// GetSizeMB 获取文件大小（MB）
func (v *VideoFile) GetSizeMB() float64 {
	return float64(v.FileSize) / (1024 * 1024)
}

// Exists 检查文件是否存在
func (v *VideoFile) Exists() bool {
	_, err := os.Stat(v.FilePath)
	return err == nil
}

// TableName 指定表名
func (VideoFile) TableName() string {
	return "video_files"
}
```

---

### `internal/services/transcription_service.go` (service, event-driven)

**Analog:** `internal/services/splitting_service.go`

**Imports pattern** (lines 1-18):
```go
package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

**Task structure pattern** (lines 20-28):
```go
// SplitTask represents a pending split operation
type SplitTask struct {
	ID          uint
	VideoFileID uint
	Markers     []float64 // timestamps in seconds, sorted
	ReEncode    bool
	CreatedBy   uint
	CreatedAt   time.Time
}
```

**Service structure pattern** (lines 30-47):
```go
type SplittingService struct {
	db               *gorm.DB
	logger           *zap.Logger
	config           *config.Config
	taskQueue        chan *SplitTask
	workers          int
	maxRetries       int
	cancelFuncs      map[uint]context.CancelFunc
	mu               sync.RWMutex
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
	ffmpegPath       string
	videoFileService *VideoFileService
	// Track active splits: videoFileID -> status ("processing" / "completed" / "failed")
	statusMap map[uint]string
	statusMu  sync.RWMutex
}
```

**Constructor pattern** (lines 49-68):
```go
func NewSplittingService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, videoFileService *VideoFileService) *SplittingService {
	ctx, cancel := context.WithCancel(context.Background())
	ffmpegPath := cfg.FFmpeg.Path
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &SplittingService{
		db:               db,
		logger:           logger,
		config:           cfg,
		taskQueue:        make(chan *SplitTask, 100),
		workers:          2,
		maxRetries:       3,
		cancelFuncs:      make(map[uint]context.CancelFunc),
		ctx:              ctx,
		cancel:           cancel,
		ffmpegPath:       ffmpegPath,
		videoFileService: videoFileService,
		statusMap:        make(map[uint]string),
	}
}
```

**Submit task pattern** (lines 86-108):
```go
// SubmitSplit submits a split task
func (s *SplittingService) SubmitSplit(videoFileID uint, markers []float64, reEncode bool, createdBy uint) error {
	task := &SplitTask{
		VideoFileID: videoFileID,
		Markers:     markers,
		ReEncode:    reEncode,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}
	s.statusMu.Lock()
	s.statusMap[videoFileID] = "processing"
	s.statusMu.Unlock()

	select {
	case s.taskQueue <- task:
		s.logger.Info("分割任务已提交", zap.Uint("video_file_id", videoFileID), zap.Int("markers", len(markers)))
		return nil
	default:
		s.statusMu.Lock()
		s.statusMap[videoFileID] = "failed"
		s.statusMu.Unlock()
		return fmt.Errorf("分割任务队列已满")
	}
}
```

**Status check pattern** (lines 110-118):
```go
// GetSplitStatus returns the current split status for a video file
func (s *SplittingService) GetSplitStatus(videoFileID uint) string {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	if status, ok := s.statusMap[videoFileID]; ok {
		return status
	}
	return ""
}
```

**Worker pattern** (lines 120-130):
```go
func (s *SplittingService) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case task := <-s.taskQueue:
			s.processSplit(task)
		case <-s.ctx.Done():
			return
		}
	}
}
```

**Process task pattern** (lines 132-151):
```go
func (s *SplittingService) processSplit(task *SplitTask) {
	// 1. Load source video file
	var sourceFile models.VideoFile
	if err := s.db.First(&sourceFile, task.VideoFileID).Error; err != nil {
		s.logger.Error("源视频文件不存在", zap.Uint("video_file_id", task.VideoFileID), zap.Error(err))
		s.statusMu.Lock()
		s.statusMap[task.VideoFileID] = "failed"
		s.statusMu.Unlock()
		return
	}

	// 2. Create output directory
	outputDir := filepath.Join(filepath.Dir(sourceFile.FilePath), "segments")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		s.logger.Error("创建输出目录失败", zap.Error(err))
		s.statusMu.Lock()
		s.statusMap[task.VideoFileID] = "failed"
		s.statusMu.Unlock()
		return
	}
	// ... rest of processing
}
```

---

### `internal/services/similarity_detector.go` (service, transform)

**Analog:** `internal/services/conversion_service.go` (FFmpeg execution pattern)

**FFmpeg command pattern** (lines 317-335):
```go
// 构建FFmpeg命令
args := []string{
	"-y", // 覆盖输出文件
	"-i", inputPath,
	"-c:v", "copy", // 视频直接复制（不重新编码）
	"-c:a", "aac", // 音频转AAC
	"-b:a", "128k",
	"-movflags", "+faststart",
	outputPath,
}

// 执行转换，捕获输出用于错误诊断
cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
```

**Error handling pattern** (lines 345-356):
```go
if err := cmd.Run(); err != nil {
	// 记录详细的错误信息
	s.logger.Error("FFmpeg转换失败",
		zap.Uint("task_id", task.ID),
		zap.String("input", inputPath),
		zap.String("output", outputPath),
		zap.Error(err),
		zap.String("stderr", stderr.String()),
		zap.String("stdout", stdout.String()),
	)
	return "", fmt.Errorf("FFmpeg转换失败: %w, stderr: %s", err, stderr.String())
}
```

---

### `internal/services/frame_extractor.go` (service, file-I/O)

**Analog:** `internal/services/conversion_service.go` (FFmpeg invocation)

**FFmpeg with context pattern** (lines 334-356):
```go
cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr

s.logger.Info("正在执行FFmpeg转换",
	zap.Uint("task_id", task.ID),
	zap.String("input", inputPath),
	zap.String("output", outputPath),
)

if err := cmd.Run(); err != nil {
	s.logger.Error("FFmpeg转换失败",
		zap.Uint("task_id", task.ID),
		zap.String("input", inputPath),
		zap.String("output", outputPath),
		zap.Error(err),
		zap.String("stderr", stderr.String()),
	)
	return "", fmt.Errorf("FFmpeg转换失败: %w, stderr: %s", err, stderr.String())
}
```

---

### `internal/services/pptx_generator.go` (service, file-I/O)

**Analog:** `internal/models/ppt_file.go`

**PPTFile model pattern** (lines 4-16):
```go
// PPTFile PPT文件模型
type PPTFile struct {
	Base
	FileName  string `gorm:"type:varchar(200);not null" json:"file_name"`
	FilePath  string `gorm:"type:varchar(500);not null" json:"file_path"`
	FileSize  int64  `gorm:"default:0" json:"file_size"`
	PageCount int    `gorm:"default:0" json:"page_count"`
	Format    string `gorm:"type:varchar(20)" json:"format"` // pptx
	SourceVideoFileID *uint      `json:"source_video_file_id,omitempty"`
	SourceVideoFile   *VideoFile `gorm:"foreignKey:SourceVideoFileID" json:"source_video_file,omitempty"`
}
```

**Placeholder method pattern** (lines 18-25):
```go
// GenerateFromVideo 从视频生成PPT（占位符实现）
func (p *PPTFile) GenerateFromVideo(videoFile *VideoFile) error {
	// TODO: 实现从视频提取帧并生成PPT的逻辑
	// 1. 使用ffmpeg提取关键帧
	// 2. 使用模板生成PPT
	// 3. 保存PPT文件
	return nil
}
```

---

### `internal/handlers/transcription_handler.go` (handler, request-response)

**Analog:** `internal/handlers/split_handler.go`

**Handler structure pattern** (lines 13-32):
```go
type SplitHandler struct {
	splittingService *services.SplittingService
	snapshotService  *services.SnapshotService
	videoFileService *services.VideoFileService
	logger           *zap.Logger
}

func NewSplitHandler(
	splittingService *services.SplittingService,
	snapshotService *services.SnapshotService,
	videoFileService *services.VideoFileService,
	logger *zap.Logger,
) *SplitHandler {
	return &SplitHandler{
		splittingService: splittingService,
		snapshotService:  snapshotService,
		videoFileService: videoFileService,
		logger:           logger,
	}
}
```

**POST handler pattern** (lines 36-85):
```go
// SubmitSplit handles POST /api/v1/videos/:id/split
// Body: {"markers": [10.5, 30.0, 50.0], "re_encode": false}
func (h *SplitHandler) SubmitSplit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	var req struct {
		Markers  []float64 `json:"markers" binding:"required"`
		ReEncode bool      `json:"re_encode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// Validate markers
	if len(req.Markers) == 0 {
		response.GinError(c, response.CodeInvalidRequest, "至少需要一个分割标记点")
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

	if err := h.splittingService.SubmitSplit(uint(id), req.Markers, req.ReEncode, userID); err != nil {
		response.GinError(c, response.CodeInternalError, "提交分割任务失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"video_file_id": id,
		"status":        "processing",
		"segment_count": len(req.Markers) + 1,
	})
}
```

**GET status handler pattern** (lines 113-131):
```go
// GetSplitStatus handles GET /api/v1/videos/:id/split-status
func (h *SplitHandler) GetSplitStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	status := h.splittingService.GetSplitStatus(uint(id))

	// If completed or empty, also return segment list
	result := gin.H{"status": status}
	if status == "completed" || status == "" {
		segments, _ := h.videoFileService.GetSegmentsByParentID(uint(id))
		result["segments"] = segments
	}

	response.GinSuccess(c, result)
}
```

---

### `internal/migrations/[timestamp]_create_transcription_tasks.go` (migration, CRUD)

**Analog:** `internal/migrations/001_add_video_file_owner.go`

**Migration interface pattern** (from migration registry):
```go
type Migration interface {
	Up(*gorm.DB) error
	Down(*gorm.DB) error
}

type NamedMigration interface {
	Migration
	Name() string
}
```

**Up migration pattern** (typical SQL migration):
```go
func (m *MigrationName) Up(db *gorm.DB) error {
	return db.Exec(`
		CREATE TABLE IF NOT EXISTS transcription_tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			video_file_id INTEGER NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			sampling_rate REAL DEFAULT 0.5,
			current_stage VARCHAR(50),
			frames_processed INTEGER DEFAULT 0,
			total_frames INTEGER DEFAULT 0,
			percentage INTEGER DEFAULT 0,
			result_ppt_file_id INTEGER,
			error_message TEXT,
			created_by INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME,
			FOREIGN KEY (video_file_id) REFERENCES video_files(id),
			FOREIGN KEY (result_ppt_file_id) REFERENCES ppt_files(id),
			FOREIGN KEY (created_by) REFERENCES users(id)
		);
		CREATE INDEX idx_transcription_tasks_video_file ON transcription_tasks(video_file_id);
		CREATE INDEX idx_transcription_tasks_status ON transcription_tasks(status);
	`).Error
}
```

---

### `frontend/src/api/transcription.ts` (api-client, request-response)

**Analog:** `frontend/src/api/split.ts`

**Imports and types pattern** (lines 1-16):
```typescript
// 视频分割 API 客户端

import type { ApiResponse } from '../types/auth'
import type { VideoFile } from '../types/video-file'
import { apiRequest } from './apiClient'

export interface SplitRequest {
  markers: number[]
  re_encode?: boolean
}

export interface SplitResponse {
  video_file_id: number
  status: string
  segment_count: number
}

export interface SplitStatusResponse {
  status: string
  segments?: VideoFile[]
}
```

**API request functions pattern** (lines 24-47):
```typescript
// 提交分割任务
export async function submitSplit(
  videoFileId: number,
  markers: number[],
  reEncode: boolean = false
): Promise<ApiResponse<SplitResponse>> {
  return apiRequest<SplitResponse>(`/api/v1/videos/${videoFileId}/split`, {
    method: 'POST',
    body: JSON.stringify({ markers, re_encode: reEncode }),
  })
}

// 获取分割状态
export async function getSplitStatus(
  videoFileId: number
): Promise<ApiResponse<SplitStatusResponse>> {
  return apiRequest<SplitStatusResponse>(`/api/v1/videos/${videoFileId}/split-status`)
}

// 获取分割段落列表
export async function getSegments(
  videoFileId: number
): Promise<ApiResponse<VideoFile[]>> {
  return apiRequest<VideoFile[]>(`/api/v1/videos/${videoFileId}/segments`)
}
```

---

### `frontend/src/pages/files/index.tsx` (component, request-response) - MODIFY

**Analog:** `frontend/src/pages/files/index.tsx` (existing file)

**Action button rendering pattern** (lines 243-293):
```typescript
// 渲染操作按钮
const renderActions = useCallback((record: VideoFile) => (
  <Space size="small">
    <RenderVideoPreview {...record} />
    <Button
      type="link"
      size="small"
      icon={<EyeOutlined />}
      onClick={() => viewDetail(record)}
    >
      详情
    </Button>
    {record.format === 'mp4' && (
      <PermissionGuard permission={PERMISSIONS.FILE_SPLIT}>
        <Tooltip title="视频分割">
          <Button
            type="link"
            size="small"
            icon={<ScissorsOutlined />}
            onClick={() => navigate(`/split/${record.id}`)}
          />
        </Tooltip>
      </PermissionGuard>
    )}
    <Button
      type="link"
      size="small"
      icon={<DownloadOutlined />}
      onClick={() => handleDownload(record.id, record.file_name)}
      disabled={record.status !== 'ready'}
    >
      下载
    </Button>
    <PermissionGuard permission={PERMISSIONS.FILE_DELETE}>
      <Popconfirm
        title="确定要删除这个文件吗？"
        onConfirm={() => handleDelete(record.id)}
        disabled={record.status === 'processing'}
      >
        <Button
          type="link"
          size="small"
          danger
          icon={<DeleteOutlined />}
          disabled={record.status === 'processing'}
        >
          删除
        </Button>
      </Popconfirm>
    </PermissionGuard>
  </Space>
), [viewDetail, handleDownload, handleDelete, navigate])
```

**Add transcription button in this section after the split button**

---

### `frontend/src/components/TranscriptionProgressModal.tsx` (component, event-driven)

**Analog:** `frontend/src/pages/split/index.tsx` (polling and progress pattern)

**Polling pattern** (lines 211-239):
```typescript
// 轮询分割状态
const pollInterval = setInterval(async () => {
  try {
    const statusResponse = await splitApi.getSplitStatus(parseInt(id, 10))

    if (statusResponse.data) {
      const { status } = statusResponse.data

      if (status === 'completed') {
        clearInterval(pollInterval)
        setSplitProgress(null)
        setSplitting(false)
        message.success(`分割完成！已生成 ${markers.length + 1} 个视频段落`)
        navigate('/files')
      } else if (status === 'failed') {
        clearInterval(pollInterval)
        setSplitProgress(null)
        setSplitting(false)
        message.error('视频分割失败，请检查视频文件是否完整或尝试使用重新编码模式')
      } else if (status === 'processing') {
        setSplitProgress(`正在分割中...`)
      }
    }
  } catch (err) {
    clearInterval(pollInterval)
    setSplitProgress(null)
    setSplitting(false)
    message.error(err instanceof Error ? err.message : '查询分割状态失败')
  }
}, POLL_INTERVAL)
```

**Modal with progress pattern** (lines 196-248):
```typescript
Modal.confirm({
  title: '确认分割',
  content: `确认将视频分割为 ${markers.length + 1} 个段落？`,
  okText: '确认分割',
  cancelText: '取消',
  onOk: async () => {
    setSplitting(true)
    setSplitProgress('正在分割中...')

    try {
      // 提交分割任务
      const splitResponse = await splitApi.submitSplit(parseInt(id, 10), markers, false)

      if (splitResponse.data) {
        // 轮询分割状态
        // ... polling logic
      }
    } catch (err) {
      setSplitProgress(null)
      setSplitting(false)
      message.error(err instanceof Error ? err.message : '提交分割任务失败')
    }
  },
})
```

---

### `frontend/src/types/transcription.ts` (types, CRUD)

**Analog:** `frontend/src/types/video-file.ts`

**Type definition pattern** (from video-file.ts):
```typescript
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
  task_id?: number
  task?: VideoRecordingTask
  parent_id?: number
  source_type: string
  status: VideoFileStatus
  thumbnail_path?: string
  created_at: string
  updated_at: string
}

export type VideoFileStatus = 'ready' | 'processing' | 'error' | 'deleting'

export interface VideoFileListParams {
  page?: number
  page_size?: number
  keyword?: string
  task_id?: number
  status?: VideoFileStatus
  format?: string
  start_date?: string
  end_date?: string
}

export interface VideoFileListResponse {
  items: VideoFile[]
  total: number
  page: number
  page_size: number
}

export interface VideoFileStats {
  total: number
  total_size_gb: number
}
```

---

## Shared Patterns

### Authentication & Authorization
**Source:** `internal/middleware/auth.go`, `internal/middleware/permission.go`
**Apply to:** All controller handlers
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
**Source:** `pkg/response/response.go`
**Apply to:** All service and handler files
```go
// Unified error response
response.GinError(c, response.CodeInvalidRequest, "错误消息")
response.GinSuccess(c, gin.H{"key": "value"})
```

### FFmpeg Command Execution
**Source:** `internal/services/conversion_service.go`
**Apply to:** All services that invoke FFmpeg
```go
// Command with context and error capture
cmd := exec.CommandContext(ctx, s.ffmpegPath, args...)
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr

if err := cmd.Run(); err != nil {
    s.logger.Error("FFmpeg执行失败",
        zap.Error(err),
        zap.String("stderr", stderr.String()),
    )
    return "", fmt.Errorf("FFmpeg执行失败: %w", err)
}
```

### Worker Pool Service Lifecycle
**Source:** `internal/services/splitting_service.go`
**Apply to:** TranscriptionService
```go
// Service structure with task queue and workers
type Service struct {
    taskQueue   chan *Task
    workers     int
    ctx         context.Context
    cancel      context.CancelFunc
    statusMap   map[uint]string
    statusMu    sync.RWMutex
}

// Start method
func (s *Service) Start() error {
    for i := 0; i < s.workers; i++ {
        s.wg.Add(1)
        go s.worker(i)
    }
    return nil
}

// Worker loop
func (s *Service) worker(id int) {
    defer s.wg.Done()
    for {
        select {
        case task := <-s.taskQueue:
            s.processTask(task)
        case <-s.ctx.Done():
            return
        }
    }
}
```

### Frontend Polling Pattern
**Source:** `frontend/src/pages/split/index.tsx`
**Apply to:** TranscriptionProgressModal
```typescript
const POLL_INTERVAL = 5000 // 5 seconds for transcription

const pollInterval = setInterval(async () => {
  try {
    const statusResponse = await api.getStatus(id)
    if (statusResponse.data?.status === 'completed') {
      clearInterval(pollInterval)
      // Handle completion
    }
  } catch (err) {
    clearInterval(pollInterval)
    // Handle error
  }
}, POLL_INTERVAL)
```

### API Client Wrapper
**Source:** `frontend/src/api/apiClient.ts`
**Apply to:** All API client modules
```typescript
// Use apiRequest wrapper for consistent error handling
export async function apiFunction(
  id: number
): Promise<ApiResponse<ResponseDataType>> {
  return apiRequest<ResponseDataType>(`/api/v1/resource/${id}`, {
    method: 'POST',
    body: JSON.stringify({ /* request data */ }),
  })
}
```

### Permission-Guarded UI Components
**Source:** `frontend/src/components/PermissionGuard.tsx`
**Apply to:** All button rendering in file list
```typescript
<PermissionGuard permission={PERMISSIONS.PERMISSION_CODE}>
  <Button onClick={handler}>
    Button Text
  </Button>
</PermissionGuard>
```

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/services/similarity_detector.go` | service | transform | No existing image similarity detection service - use RESEARCH.md SSIM/pHash/edge detection patterns |
| `internal/services/pptx_generator.go` | service | file-I/O | No existing PPTX generation service - use RESEARCH.md unioffice library patterns |

## Metadata

**Analog search scope:** 
- Backend: `internal/services/`, `internal/handlers/`, `internal/models/`, `internal/migrations/`
- Frontend: `frontend/src/api/`, `frontend/src/pages/`, `frontend/src/components/`, `frontend/src/types/`

**Files scanned:** 20+
**Pattern extraction date:** 2026-04-17
**Primary analog sources:**
- `internal/services/splitting_service.go` - Worker pool, task queue, status tracking
- `internal/services/conversion_service.go` - FFmpeg invocation, error handling
- `internal/handlers/split_handler.go` - Handler structure, request/response patterns
- `internal/models/video_file.go` - Model structure, status constants
- `frontend/src/api/split.ts` - API client patterns
- `frontend/src/pages/split/index.tsx` - Polling and progress tracking
