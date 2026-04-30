# Phase 14: 文件管理页面添加批量下载和批量转录功能 - Pattern Map

**Mapped:** 2026-04-30
**Files analyzed:** 11
**Analogs found:** 11 / 11

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/models/transcription_job_group.go` | model | CRUD | `internal/models/transcription_task.go` | exact |
| `internal/handlers/video_file_handler.go` | handler | request-response | `internal/handlers/video_file_handler.go` (BatchDeleteFiles) | exact |
| `internal/handlers/transcription_handler.go` | handler | request-response | `internal/handlers/transcription_handler.go` (SubmitTranscription) | exact |
| `internal/services/video_file_service.go` | service | CRUD | `internal/services/video_file_service.go` (BatchDeleteFiles) | exact |
| `internal/services/transcription_service.go` | service | event-driven | `internal/services/transcription_service.go` (SubmitTranscription) | exact |
| `internal/migrations/015_add_transcription_job_groups.go` | migration | batch | `internal/migrations/014_create_input_configs.go` | role-match |
| `frontend/src/pages/files/index.tsx` | component | request-response | `frontend/src/pages/files/index.tsx` (rowSelection, batchDelete) | exact |
| `frontend/src/api/video-file.ts` | api-client | request-response | `frontend/src/api/video-file.ts` (batchDeleteFiles, downloadVideoFile) | exact |
| `frontend/src/api/transcription.ts` | api-client | request-response | `frontend/src/api/transcription.ts` (submitTranscriptionWithMode) | exact |
| `frontend/src/types/video-file.ts` | types | static | `frontend/src/types/video-file.ts` | exact |
| `frontend/src/types/transcription.ts` | types | static | `frontend/src/types/transcription.ts` | exact |

## Pattern Assignments

### `internal/models/transcription_job_group.go` (model, CRUD)

**Analog:** `internal/models/transcription_task.go`

**Imports pattern** (lines 1-8):
```go
package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)
```

**Base model pattern** (lines 16-36):
```go
type TranscriptionTask struct {
	Base
	VideoFileID     uint       `gorm:"not null;index" json:"video_file_id"`
	VideoFile       *VideoFile `gorm:"foreignKey:VideoFileID" json:"video_file,omitempty"`
	Status          string     `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	// ... other fields
	CreatedBy       uint       `gorm:"not null" json:"created_by"`
	Creator         *User      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Mode            string     `gorm:"type:varchar(20);default:'local'" json:"mode"`
}
```

**Status constants pattern** (lines 38-57):
```go
const (
	TranscriptionStatusPending    = "pending"
	TranscriptionStatusProcessing = "processing"
	TranscriptionStatusCompleted  = "completed"
	TranscriptionStatusFailed     = "failed"
)

const (
	TranscriptionModeLocal = "local"
	TranscriptionModeCloud = "cloud"
)
```

**JSON field handling pattern** (lines 72-113):
```go
func (t *TranscriptionTask) GetSlideTimestamps() ([]SlideTimestamp, error) {
	if t.SlideTimestamps == "" {
		return []SlideTimestamp{}, nil
	}
	var timestamps []SlideTimestamp
	if err := json.Unmarshal([]byte(t.SlideTimestamps), &timestamps); err != nil {
		return []SlideTimestamp{}, nil
	}
	return timestamps, nil
}

func (t *TranscriptionTask) SetSlideTimestamps(timestamps []SlideTimestamp) error {
	data, err := json.Marshal(timestamps)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	t.SlideTimestamps = string(data)
	return nil
}
```

---

### `internal/handlers/video_file_handler.go` (handler, request-response)

**Analog:** `internal/handlers/video_file_handler.go` (BatchDeleteFiles method, lines 177-204)

**Imports pattern** (lines 1-15):
```go
package handlers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

**Batch operation request pattern** (lines 177-180):
```go
type BatchDeleteFilesRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}
```

**Batch handler pattern** (lines 182-204):
```go
func (h *VideoFileHandler) BatchDeleteFiles(c *gin.Context) {
	var req BatchDeleteFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "参数错误")
		return
	}

	result, err := h.fileService.BatchDeleteFiles(req.IDs)
	if err != nil {
		h.logger.Error("批量删除文件失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "批量删除失败")
		return
	}

	h.logger.Info("批量删除文件成功",
		zap.Int("total", len(req.IDs)),
		zap.Int("success", result.Success),
		zap.Int("failed", result.Failed),
	)

	response.GinSuccess(c, result)
}
```

**Streaming download pattern** from `internal/handlers/file_handler.go` (lines 102-129):
```go
func (h *FileHandler) Download(c *gin.Context) {
	accessToken := c.Param("token")
	userID := h.getUserID(c)

	reader, filename, err := h.fileService.Download(c.Request.Context(), accessToken, userID)
	if err != nil {
		h.logger.Warn("文件下载失败",
			zap.Uint("user_id", userID),
			zap.String("token", accessToken),
			zap.Error(err),
		)
		response.GinError(c, response.CodeNotFound, err.Error())
		return
	}
	defer reader.Close()

	// Set headers for download
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Accept-Ranges", "bytes")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Expose-Headers", "Content-Length, Content-Range, Accept-Ranges")

	// Stream file to response
	c.DataFromReader(200, -1, "application/octet-stream", reader, nil)
}
```

---

### `internal/handlers/transcription_handler.go` (handler, request-response)

**Analog:** `internal/handlers/transcription_handler.go` (SubmitTranscription method, lines 37-101)

**Request pattern with mode selection** (lines 47-77):
```go
var req struct {
	SamplingRate float64 `json:"sampling_rate"`
	Mode         string  `json:"mode"`
}
if err := c.ShouldBindJSON(&req); err != nil {
	req.SamplingRate = 0.5
	req.Mode = "local"
}

// Validate mode
validModes := map[string]bool{"local": true, "cloud": true}
if !validModes[req.Mode] {
	response.GinError(c, response.CodeInvalidRequest, "无效的转录模式，必须是 local 或 cloud")
	return
}
```

**File ownership verification pattern** (lines 78-88):
```go
userID := middleware.GetUserID(c)
file, err := h.videoFileService.GetFileByID(uint(id))
if err != nil {
	response.GinError(c, response.CodeInternalError, "视频文件不存在")
	return
}
if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
	response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
	return
}
```

**Service call and response pattern** (lines 90-100):
```go
if err := h.transcriptionService.SubmitTranscriptionWithMode(uint(id), req.SamplingRate, req.Mode, userID); err != nil {
	response.GinError(c, response.CodeInternalError, "提交转录任务失败: "+err.Error())
	return
}

response.GinSuccess(c, gin.H{
	"video_file_id": id,
	"status":        "processing",
	"sampling_rate": req.SamplingRate,
	"mode":          req.Mode,
})
```

---

### `internal/services/video_file_service.go` (service, CRUD)

**Analog:** `internal/services/video_file_service.go` (BatchDeleteFiles method, lines 329-420)

**Batch request/result types** (lines 329-339):
```go
type BatchDeleteFilesRequest struct {
	IDs []uint `json:"ids"`
}

type BatchDeleteFilesResult struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}
```

**Batch operation pattern** (lines 341-390):
```go
func (s *VideoFileService) BatchDeleteFiles(ids []uint) (*BatchDeleteFilesResult, error) {
	result := &BatchDeleteFilesResult{}

	if len(ids) == 0 {
		return result, nil
	}

	// Query all files to delete
	var files []models.VideoFile
	if err := s.db.Where("id IN ?", ids).Find(&files).Error; err != nil {
		return result, err
	}

	// Categorize files by status and task association
	var filesWithTask []models.VideoFile
	var orphanFiles []models.VideoFile
	var processingFileIDs []uint

	for _, file := range files {
		if file.Status == models.FileStatusProcessing {
			processingFileIDs = append(processingFileIDs, file.ID)
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("文件 %d 正在处理中，无法删除", file.ID))
		} else if file.TaskID == nil {
			orphanFiles = append(orphanFiles, file)
		} else {
			filesWithTask = append(filesWithTask, file)
		}
	}

	// Process each category
	for _, file := range orphanFiles {
		if err := s.deleteOrphanFile(&file); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("删除孤立文件 %d 失败: %v", file.ID, err))
		} else {
			result.Success++
		}
	}

	// ... continue with task-based deletion
	return result, nil
}
```

---

### `internal/services/transcription_service.go` (service, event-driven)

**Analog:** `internal/services/transcription_service.go` (lines 32-91)

**Service struct pattern** (lines 32-53):
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
	ffmpegPath         string
	frameExtractor     *FrameExtractor
	similarityDetector *SimilarityDetector
	pptxGenerator      *PPTXGenerator
	videoFileService   *VideoFileService
	ossService         *OSSService
	tingwuClient       *TingwuClient
	statusMap          map[uint]*TranscriptionProgress
	statusMu           sync.RWMutex
}
```

**Worker pool initialization** (lines 72-91):
```go
return &TranscriptionService{
	db:                 db,
	logger:             logger,
	config:             cfg,
	taskQueue:          make(chan *models.TranscriptionTask, 100),
	workers:            1, // Conservative for memory-intensive transcription
	cancelFuncs:        make(map[uint]context.CancelFunc),
	ctx:                ctx,
	cancel:             cancel,
	ffmpegPath:         ffmpegPath,
	frameExtractor:     frameExtractor,
	similarityDetector: similarityDetector,
	pptxGenerator:      pptxGenerator,
	videoFileService:   videoFileService,
	ossService:         ossService,
	tingwuClient:       tingwuClient,
	statusMap:          make(map[uint]*TranscriptionProgress),
}
```

**Task submission pattern** (inferred from existing SubmitTranscriptionWithMode):
```go
// Create task in database
task := &models.TranscriptionTask{
	VideoFileID:  videoFileID,
	SamplingRate: samplingRate,
	Mode:         mode,
	Status:       models.TranscriptionStatusPending,
	CreatedBy:    userID,
}
if err := s.db.Create(task).Error; err != nil {
	return fmt.Errorf("创建转录任务失败: %w", err)
}

// Submit to worker queue
select {
case s.taskQueue <- task:
	s.logger.Info("转录任务已提交到队列",
		zap.Uint("task_id", task.ID),
		zap.Uint("video_file_id", videoFileID),
	)
default:
	return fmt.Errorf("任务队列已满")
}

return nil
```

---

### `internal/migrations/015_add_transcription_job_groups.go` (migration, batch)

**Analog:** `internal/migrations/014_create_input_configs.go`

**Migration structure pattern** (lines 1-30):
```go
package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// AddIPRestrictionsMigration 为用户和角色添加IP限制字段
type AddIPRestrictionsMigration struct{}

func (m *AddIPRestrictionsMigration) Name() string {
	return "011_add_ip_restrictions"
}

func (m *AddIPRestrictionsMigration) Up(db *gorm.DB) error {
	// Step 1: Add column if not exists
	exists, err := columnExists(db, "users", "allowed_ips")
	if err != nil {
		return fmt.Errorf("failed to check allowed_ips column in users: %w", err)
	}
	if !exists {
		if err := db.Exec("ALTER TABLE users ADD COLUMN allowed_ips TEXT").Error; err != nil {
			return fmt.Errorf("failed to add allowed_ips column to users: %w", err)
		}
		log.Println("INFO: Added allowed_ips column to users table")
	} else {
		log.Println("INFO: allowed_ips column already exists in users table, skipping")
	}

	// Step 2: Add indexes
	if err := db.Exec("CREATE INDEX idx_users_allowed_ips ON users(allowed_ips)").Error; err != nil {
		log.Printf("WARN: Failed to create index on allowed_ips: %v", err)
	}

	return nil
}

func (m *AddIPRestrictionsMigration) Down(db *gorm.DB) error {
	// Remove columns
	db.Exec("ALTER TABLE users DROP COLUMN allowed_ips")
	return nil
}
```

---

### `frontend/src/pages/files/index.tsx` (component, request-response)

**Analog:** `frontend/src/pages/files/index.tsx` (existing rowSelection and batchDelete)

**Row selection state pattern** (lines 104-105):
```tsx
const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
const [batchDeleting, setBatchDeleting] = useState(false)
```

**Table rowSelection configuration** (lines 721-727):
```tsx
<Table
  columns={columns}
  dataSource={files}
  rowSelection={{
    selectedRowKeys,
    onChange: (selectedKeys) => setSelectedRowKeys(selectedKeys),
    getCheckboxProps: (record: VideoFile) => ({
      disabled: record.status === 'processing',
    }),
  }}
  pagination={{
    current: params.page,
    pageSize: params.page_size,
    total,
    showSizeChanger: true,
    showTotal: (t) => `共 ${t} 条`,
  }}
  onChange={handleTableChange}
/>
```

**Batch operation handler pattern** (lines 241-270):
```tsx
const handleBatchDelete = useCallback(async () => {
  if (selectedRowKeys.length === 0) {
    message.warning('请先选择要删除的文件')
    return
  }

  setBatchDeleting(true)
  try {
    const response = await videoFileApi.batchDeleteFiles(selectedRowKeys as number[])
    if (response.data) {
      const { success, failed } = response.data
      if (failed > 0) {
        message.warning(`成功删除 ${success} 个文件，${failed} 个失败`)
      } else {
        message.success(`成功删除 ${success} 个文件`)
      }
      setSelectedRowKeys([])
      loadFiles()
      loadStats()
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量删除失败')
  } finally {
    setBatchDeleting(false)
  }
}, [selectedRowKeys, loadFiles, loadStats])
```

**Conditional batch button rendering** (lines 676-696):
```tsx
{selectedRowKeys.length > 0 && (
  <Space style={{ marginBottom: 16 }}>
    <Popconfirm
      title={`确定要删除选中的 ${selectedRowKeys.length} 个文件吗？`}
      onConfirm={handleBatchDelete}
      okText="确定"
      cancelText="取消"
    >
      <Button
        danger
        icon={<DeleteOutlined />}
        loading={batchDeleting}
        disabled={batchDeleting}
      >
        批量删除 ({selectedRowKeys.length})
      </Button>
    </Popconfirm>
  </Space>
)}
```

---

### `frontend/src/api/video-file.ts` (api-client, request-response)

**Analog:** `frontend/src/api/video-file.ts` (batchDeleteFiles method, lines 88-108)

**Batch API pattern** (lines 88-108):
```typescript
export interface BatchDeleteFilesRequest {
  ids: number[]
}

export interface BatchDeleteFilesResult {
  success: number
  failed: number
  errors: string[]
}

export async function batchDeleteFiles(
  ids: number[]
): Promise<ApiResponse<BatchDeleteFilesResult>> {
  return apiRequest<BatchDeleteFilesResult>('/api/v1/files/batch', {
    method: 'DELETE',
    body: JSON.stringify({ ids }),
  })
}
```

**Download with blob pattern** (lines 40-58):
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

### `frontend/src/api/transcription.ts` (api-client, request-response)

**Analog:** `frontend/src/api/transcription.ts` (submitTranscriptionWithMode method, lines 39-59)

**API with mode selection pattern** (lines 39-59):
```typescript
export async function submitTranscriptionWithMode(
  videoFileId: number,
  mode: TranscriptionMode,
  samplingRate?: number
): Promise<ApiResponse<TranscriptionTriggerResponseExtended>> {
  const body: Record<string, unknown> = { mode }
  // Only include sampling_rate for local mode
  if (mode === 'local' && samplingRate) {
    body.sampling_rate = samplingRate
  }
  return apiRequest<TranscriptionTriggerResponseExtended>(
    `/api/v1/videos/${videoFileId}/transcribe`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    }
  )
}
```

---

### `frontend/src/types/video-file.ts` (types, static)

**Analog:** `frontend/src/types/video-file.ts`

**Status type definition** (line 9):
```typescript
export type VideoFileStatus = 'ready' | 'processing' | 'error' | 'deleting'
```

**Interface pattern** (lines 11-39):
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
  task_id?: number | null
  status: VideoFileStatus
  thumbnail_path: string | null
  created_at: string
  updated_at: string
}
```

---

### `frontend/src/types/transcription.ts` (types, static)

**Analog:** `frontend/src/types/transcription.ts`

**Mode type definition** (line 42):
```typescript
export type TranscriptionMode = 'local' | 'cloud'
```

**Batch result interface pattern** (lines 79-83):
```typescript
export interface TranscriptionTriggerResponseExtended {
  video_file_id: number
  status: string
  mode: TranscriptionMode
}
```

---

## Shared Patterns

### Authentication/Authorization
**Source:** `internal/handlers/transcription_handler.go` lines 79-88
**Apply to:** All batch operation handlers

```go
userID := middleware.GetUserID(c)
file, err := h.videoFileService.GetFileByID(uint(id))
if err != nil {
	response.GinError(c, response.CodeInternalError, "视频文件不存在")
	return
}
if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
	response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
	return
}
```

### Error Handling
**Source:** `pkg/response/response.go` lines 17-33
**Apply to:** All handler files

```go
const (
	CodeSuccess           = 0
	CodeInvalidRequest    = 1001
	CodeUnauthorized      = 1002
	CodeForbidden         = 1003
	CodeNotFound          = 1004
	CodeInternalError     = 1005
)

// Usage in handlers
response.GinError(c, response.CodeInvalidRequest, "参数错误")
response.GinSuccess(c, result)
```

### Batch Result Structure
**Source:** `internal/services/video_file_service.go` lines 334-339
**Apply to:** All batch operation services

```go
type BatchDeleteFilesResult struct {
	Success int      `json:"success"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}
```

### File Ownership Validation
**Source:** `internal/services/video_file_service.go` lines 351-354
**Apply to:** All batch operations accessing files

```go
var files []models.VideoFile
if err := s.db.Where("id IN ?", ids).Find(&files).Error; err != nil {
	return result, err
}
// Always validate ownership before processing
```

### Toast Notification Pattern
**Source:** `frontend/src/pages/files/index.tsx` lines 249-256
**Apply to:** All batch operation UI handlers

```tsx
const { success, failed } = response.data
if (failed > 0) {
  message.warning(`成功删除 ${success} 个文件，${failed} 个失败`)
} else {
  message.success(`成功删除 ${success} 个文件`)
}
```

### ZIP Streaming Headers
**Source:** `internal/handlers/file_handler.go` lines 121-125
**Apply to:** Batch download handler

```go
c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
c.Header("Content-Type", "application/zip")  // Changed for ZIP
c.Header("Accept-Ranges", "none")  // Disable range for ZIP
c.Header("Access-Control-Allow-Origin", "*")
```

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| None | - | - | All files have close analogs in the existing codebase |

## Metadata

**Analog search scope:**
- `internal/handlers/*.go`
- `internal/services/*.go`
- `internal/models/*.go`
- `internal/migrations/*.go`
- `frontend/src/pages/files/*.tsx`
- `frontend/src/api/*.ts`
- `frontend/src/types/*.ts`
- `pkg/response/*.go`

**Files scanned:** 25+
**Pattern extraction date:** 2026-04-30

**Key insights:**
1. **Batch delete pattern** in `video_file_service.go` provides perfect template for batch operations
2. **Streaming download** in `file_handler.go` shows how to stream files to HTTP response
3. **Row selection** in files page already implements the UI pattern needed
4. **Transcription service** has worker pool queue pattern for sequential task processing
5. **Migration pattern** is consistent across all migrations with Up/Down methods
