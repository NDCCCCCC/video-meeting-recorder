# Phase 1: Video Splitting - Pattern Map

**Mapped:** 2026-04-17
**Files analyzed:** 15 new/modified files
**Analogs found:** 14 / 15

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/splitting_service.go` | service | CRUD | `internal/services/conversion_service.go` | exact |
| `internal/handlers/split_handler.go` | handler | request-response | `internal/handlers/video_file_handler.go` | role-match |
| `internal/models/video_file.go` | model | CRUD | `internal/models/video_file.go` | self-extension |
| `internal/migrations/001_add_segment_fields.go` | migration | transform | `internal/migrations/` (registry pattern) | pattern-match |
| `frontend/src/pages/split/index.tsx` | component | event-driven | `frontend/src/pages/tasks/index.tsx` | role-match |
| `frontend/src/components/TimelineWithMarkers.tsx` | component | event-driven | `frontend/src/components/VideoPlayerModal.tsx` | partial-match |
| `frontend/src/api/split.ts` | utility | request-response | `frontend/src/api/video-file.ts` | exact |
| `internal/services/recording_service.go` (extend) | service | event-driven | `internal/services/conversion_service.go` | pattern-match |
| `frontend/src/pages/tasks/index.tsx` (extend) | component | event-driven | `frontend/src/pages/tasks/index.tsx` | self-extension |
| `frontend/src/pages/files/index.tsx` (extend) | component | request-response | `frontend/src/pages/files/index.tsx` | self-extension |

## Pattern Assignments

### `internal/services/splitting_service.go` (service, CRUD)

**Analog:** `internal/services/conversion_service.go`

**Imports pattern** (lines 1-18):
```go
package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

**Service interface pattern** (lines 19-35):
```go
// ConversionService 转换服务接口
type ConversionService interface {
	// SubmitConversion 提交转换任务
	SubmitConversion(taskID uint) error
	
	// GetConversionStatus 获取转换状态
	GetConversionStatus(taskID uint) (models.ConversionStatus, error)
	
	// Start 启动服务
	Start() error
	
	// Stop 停止服务
	Stop()
}
```

**Worker pool pattern** (lines 43-58):
```go
type FFmpegConversionService struct {
	db               *gorm.DB
	logger           *zap.Logger
	config           *config.Config
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
```

**Service callback pattern** (lines 279-291):
```go
// 转换成功，创建 MP4 文件记录，并获取实际视频时长
if s.videoFileService != nil {
	mp4 := "mp4"
	videoFile, err := s.videoFileService.CreateFileFromTask(&task, &mp4)
	if err != nil {
		s.logger.Error("创建MP4文件记录失败",
			zap.Uint("task_id", taskID),
			zap.Error(err),
		)
	} else if videoFile != nil && videoFile.Duration > 0 {
		// 更新录制时长（从视频文件元数据获取）
		updates["recording_duration"] = videoFile.Duration
	}
}
```

**FFmpeg execution pattern** (lines 334-356):
```go
// 执行转换，捕获输出用于错误诊断
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

**Error handling with retry pattern** (lines 374-428):
```go
// handleConversionError 处理转换错误
func (s *FFmpegConversionService) handleConversionError(task *models.VideoRecordingTask, err error) {
	task.ConversionRetryCount++
	
	// 检查是否超过最大重试次数
	if task.ConversionRetryCount >= s.maxRetries {
		// 标记为失败，同时更新任务状态为失败
		s.db.Model(task).Updates(map[string]interface{}{
			"conversion_status":    models.ConversionStatusFailed,
			"conversion_error_msg": err.Error(),
			// 同时更新任务状态为失败
			"status":    models.VideoStatusFailed,
			"error_msg": fmt.Sprintf("转换失败: %s", err.Error()),
		})
		s.logger.Error("转换失败，已达最大重试次数，任务已标记为失败",
			zap.Uint("task_id", task.ID),
			zap.Int("retry_count", task.ConversionRetryCount),
			zap.Error(err),
		)
		return
	}
	
	// 计算退避时间
	backoffDuration := s.calculateBackoff(task.ConversionRetryCount)
	
	// 安排重试
	go func() {
		select {
		case <-time.After(backoffDuration):
			select {
			case s.taskQueue <- task.ID:
				s.logger.Info("重试已安排",
					zap.Uint("task_id", task.ID),
					zap.Int("attempt", task.ConversionRetryCount),
				)
			case <-s.ctx.Done():
				return
			}
		case <-s.ctx.Done():
			return
		}
	}()
}
```

---

### `internal/handlers/split_handler.go` (handler, request-response)

**Analog:** `internal/handlers/video_file_handler.go`

**Imports pattern** (lines 1-14):
```go
package handlers

import (
	"net/http"

	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)
```

**Handler structure pattern** (lines 16-31):
```go
// VideoFileHandler 视频文件处理器
type VideoFileHandler struct {
	fileService *services.VideoFileService
	logger      *zap.Logger
}

// NewVideoFileHandler 创建视频文件处理器
func NewVideoFileHandler(
	fileService *services.VideoFileService,
	logger *zap.Logger,
) *VideoFileHandler {
	return &VideoFileHandler{
		fileService: fileService,
		logger:      logger,
	}
}
```

**Request handling pattern** (lines 34-62):
```go
// ListFiles 获取文件列表
func (h *VideoFileHandler) ListFiles(c *gin.Context) {
	var req services.ListFilesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	// 设置数据范围过滤参数
	req.UserID = middleware.GetUserID(c)
	req.IsAdmin = middleware.GetIsAdmin(c)
	req.ApplyDataScope = true

	result, err := h.fileService.ListFiles(&req)
	if err != nil {
		h.logger.Error("获取文件列表失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取文件列表失败")
		return
	}

	response.GinSuccess(c, result)
}
```

**Error response pattern** (lines 56-59):
```go
if err != nil {
	h.logger.Error("获取文件列表失败", zap.Error(err))
	response.GinError(c, response.CodeInternalError, "获取文件列表失败")
	return
}
```

---

### `internal/models/video_file.go` (model, CRUD)

**Analog:** Self-extension (add fields to existing model)

**Current model structure** (lines 10-27):
```go
// VideoFile 视频文件模型
type VideoFile struct {
	Base
	FileName      string              `gorm:"type:varchar(200);not null" json:"file_name"`
	FilePath      string              `gorm:"type:varchar(500);not null;uniqueIndex:idx_file_path" json:"file_path"`
	FileSize      int64               `gorm:"default:0" json:"file_size"`
	Duration      int                 `gorm:"default:0" json:"duration"`
	Format        string              `gorm:"type:varchar(20)" json:"format"`
	TaskID        *uint               `gorm:"index" json:"task_id,omitempty"`
	CreatedBy     uint                `gorm:"not null" json:"created_by"`
	Status        string              `gorm:"type:varchar(20);default:'ready';index:idx_status_created,priority:1" json:"status"`
	ThumbnailPath *string             `json:"thumbnail_path,omitempty"`
	RecordedAt    *time.Time          `gorm:"index" json:"recorded_at,omitempty"`
}
```

**Add fields pattern** (new fields to add):
```go
	ParentID      *uint               `gorm:"index" json:"parent_id,omitempty"`       // 父视频ID（用于分割段、快照）
	SourceType    string              `gorm:"type:varchar(20);default:'recording'" json:"source_type"` // recording, snapshot, split
	Parent        *VideoFile          `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
```

---

### `internal/migrations/001_add_segment_fields.go` (migration, transform)

**Analog:** Migration registry pattern from `cmd/server/app.go` (lines 235-264)

**Migration registration pattern** (lines 236-264):
```go
// runCustomMigrations 执行自定义迁移（SQL迁移）
func (a *MinimalApp) runCustomMigrations() error {
	a.logger.Info("正在执行自定义数据库迁移...")

	// 获取注册的迁移
	migrations := migrations.GetRegisteredMigrations()

	// 执行每个迁移的 Up 方法
	for _, m := range migrations {
		migrationName := ""
		if mi, ok := m.(interface{ Name() string }); ok {
			migrationName = mi.Name()
		}

		a.logger.Info("执行迁移", zap.String("migration", migrationName))

		if mu, ok := m.(interface{ Up(*gorm.DB) error }); ok {
			if err := mu.Up(a.db); err != nil {
				a.logger.Error("迁移失败",
					zap.String("migration", migrationName),
					zap.Error(err),
				)
				return fmt.Errorf("migration %s failed: %w", migrationName, err)
			}
			a.logger.Info("迁移成功", zap.String("migration", migrationName))
		}
	}

	return nil
}
```

**Migration structure pattern** (to be created):
```go
package migrations

import (
	"gorm.io/gorm"
)

// AddSegmentFieldsMigration 添加分割段相关字段
type AddSegmentFieldsMigration struct{}

func (m *AddSegmentFieldsMigration) Name() string {
	return "001_add_segment_fields"
}

func (m *AddSegmentFieldsMigration) Up(db *gorm.DB) error {
	// 添加 parent_id 和 source_type 字段
	return db.Exec(`
		ALTER TABLE video_files 
		ADD COLUMN parent_id INTEGER,
		ADD COLUMN source_type VARCHAR(20) DEFAULT 'recording',
		CREATE INDEX idx_video_files_parent_id ON video_files(parent_id)
	`).Error
}

func init() {
	RegisterMigration(&AddSegmentFieldsMigration{})
}
```

---

### `frontend/src/pages/split/index.tsx` (component, event-driven)

**Analog:** `frontend/src/pages/tasks/index.tsx`

**Component imports pattern** (lines 1-61):
```typescript
import { useState, useEffect, useMemo, useCallback } from 'react'
import {
	Table,
	Button,
	Space,
	Input,
	Modal,
	message,
	Popconfirm,
	Tag,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import * as taskApi from '../../api/task'
import { PermissionGuard } from '../../components/PermissionGuard'
import { PERMISSIONS } from '../../utils/permissions'
import type {
	VideoRecordingTask,
	VideoRecordingTaskStatus,
	TaskListParams,
} from '../../types/task'
```

**State management pattern** (lines 167-186):
```typescript
export default function TaskManagement() {
	const [tasks, setTasks] = useState<VideoRecordingTask[]>([])
	const [total, setTotal] = useState(0)
	const [loading, setLoading] = useState(false)
	const [modalVisible, setModalVisible] = useState(false)
	const [editingTask, setEditingTask] = useState<VideoRecordingTask | null>(null)
	const [form] = Form.useForm()

	// 查询参数
	const [params, setParams] = useState<TaskListParams>({
		page: 1,
		page_size: DEFAULT_PAGE_SIZE,
	})
```

**Data loading pattern** (lines 192-205):
```typescript
	// 加载任务列表
	const loadTasks = useCallback(async (showLoading = false) => {
		if (showLoading) setLoading(true)
		try {
			const response = await taskApi.getTaskList(params)
			if (response.data) {
				setTasks(response.data.items)
				setTotal(response.data.total)
			}
		} catch (error) {
			message.error(error instanceof Error ? error.message : '加载任务列表失败')
		} finally {
			if (showLoading) setLoading(false)
		}
	}, [params])
```

**Table columns pattern** (lines 551-613):
```typescript
	// 表格列定义
	const columns: ColumnsType<VideoRecordingTask> = useMemo(() => [
		{
			title: 'ID',
			dataIndex: 'id',
			width: 80,
		},
		{
			title: '任务名称',
			dataIndex: 'name',
			width: 200,
			ellipsis: true,
		},
		// ... more columns
		{
			title: '操作',
			key: 'action',
			width: 200,
			fixed: 'right' as const,
			render: (_: unknown, record: VideoRecordingTask) => renderActions(record),
		},
	], [renderStatus, renderActions])
```

---

### `frontend/src/components/TimelineWithMarkers.tsx` (component, event-driven)

**Analog:** Partial pattern from existing Slider usage in codebase

**Props interface pattern** (to be created):
```typescript
interface TimelineWithMarkersProps {
	duration: number           // 视频总时长（秒）
	markers: number[]          // 分割标记点（秒）
	currentTime: number        // 当前播放时间
	onMarkerAdd: (time: number) => void
	onMarkerRemove: (time: number) => void
	onSeek: (time: number) => void
	editable?: boolean         // 是否允许编辑标记
}
```

**State management pattern** (from task list):
```typescript
const [markers, setMarkers] = useState<number[]>([]) // Array of timestamps in seconds

// Add marker on timeline click
const handleTimelineClick = (timestamp: number) => {
	setMarkers([...markers, timestamp].sort((a, b) => a - b))
}

// Remove marker
const handleMarkerRemove = (timestamp: number) => {
	setMarkers(markers.filter(m => m !== timestamp))
}
```

---

### `frontend/src/api/split.ts` (utility, request-response)

**Analog:** `frontend/src/api/video-file.ts`

**API client pattern** (lines 1-32):
```typescript
import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'

// 获取文件列表
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

**POST request pattern** (from video-file.ts lines 69-73):
```typescript
// 扫描并导入文件
export async function scanVideoFiles(): Promise<ApiResponse<ScanResult>> {
	return apiRequest<ScanResult>('/api/v1/files/scan', {
		method: 'POST',
	})
}
```

---

### `frontend/src/pages/tasks/index.tsx` (extend - snapshot button)

**Analog:** Self-extension

**Add inline action button pattern** (from existing task list, lines 116-164):
```typescript
const TaskActions = memo(function TaskActions({
	record,
	onStart,
	onStop,
	onCancel,
	onRetry,
	onDelete,
	onEdit,
}: TaskActionsProps) {
	return (
		<Space size="small">
			{canPreviewTask(record.status) && (
				<HLSPreview taskId={record.id} taskName={record.name} status={record.status} />
			)}
			<PermissionGuard permission={PERMISSIONS.TASK_START}>
				{canStartTask(record.status) && (
					<Tooltip title="启动任务">
						<Button type="link" size="small" icon={<PlayCircleOutlined />} onClick={() => onStart(record.id)} />
					</Tooltip>
				)}
			</PermissionGuard>
			// ... more buttons
		</Space>
	)
})
```

**Snapshot button pattern** (to be added):
```typescript
// Add to TaskActions component
{canGenerateSnapshot(record.status) && (
	<PermissionGuard permission={PERMISSIONS.RECORDING_SNAPSHOT}>
		<Button
			type="link"
			size="small"
			icon={<CameraOutlined />}
			loading={record.snapshotGenerating}
			onClick={() => onGenerateSnapshot(record.id)}
			disabled={record.snapshotGenerating}
		>
			{record.snapshotGenerating ? '生成中...' : '生成MP4快照'}
		</Button>
	</PermissionGuard>
)}
```

---

### `frontend/src/pages/files/index.tsx` (extend - source column)

**Analog:** Self-extension

**Add table column pattern** (from existing file list, lines 271-330):
```typescript
	// 表格列定义
	const columns: ColumnsType<VideoFile> = useMemo(() => [
		{
			title: 'ID',
			dataIndex: 'id',
			width: 60,
		},
		{
			title: '文件名',
			dataIndex: 'file_name',
			width: 250,
			ellipsis: true,
			render: (name: string) => (
				<Space>
					<VideoCameraOutlined />
					<Tooltip title={name}>{name}</Tooltip>
				</Space>
			),
		},
		// ... more columns
	], [renderStatus, viewDetail, handleDownload, handleDelete])
```

**Source column pattern** (to be added):
```typescript
		{
			title: '来源',
			dataIndex: 'source_type',
			width: 100,
			render: (sourceType: string, record: VideoFile) => {
				const SOURCE_CONFIG = {
					recording: { label: '录制', color: 'blue' },
					snapshot: { label: '快照', color: 'green' },
					split: { label: '分割', color: 'orange' },
				}
				const config = SOURCE_CONFIG[sourceType] || SOURCE_CONFIG.recording
				return (
					<Space>
						<Tag color={config.color}>{config.label}</Tag>
						{record.parent_id && (
							<Button
								type="link"
								size="small"
								onClick={() => navigateToParent(record.parent_id)}
							>
								查看原视频
							</Button>
						)}
					</Space>
				)
			},
		},
```

---

## Shared Patterns

### Authentication Middleware
**Source:** `internal/middleware/auth.go`
**Apply to:** All API route groups

```go
// From app.go lines 541-543
api := a.router.Group("/api/v1")
api.Use(middleware.MultiAuth(a.db, a.tokenService)) // 支持SM4 Token和API Key认证
```

### Permission Guards
**Source:** `frontend/src/components/PermissionGuard.tsx`
**Apply to:** All frontend action buttons

```typescript
// From tasks/index.tsx lines 120-126
<PermissionGuard permission={PERMISSIONS.TASK_START}>
	{canStartTask(record.status) && (
		<Tooltip title="启动任务">
			<Button type="link" size="small" icon={<PlayCircleOutlined />} onClick={() => onStart(record.id)} />
		</Tooltip>
	)}
</PermissionGuard>
```

### Error Response Format
**Source:** `pkg/response/` (implied from usage)
**Apply to:** All handler error responses

```go
// From video_file_handler.go lines 56-59
if err != nil {
	h.logger.Error("获取文件列表失败", zap.Error(err))
	response.GinError(c, response.CodeInternalError, "获取文件列表失败")
	return
}
```

### GORM Transaction Pattern
**Source:** `internal/services/video_file_service.go` (lines 221-231)
**Apply to:** All multi-step database operations

```go
// 使用事务删除数据库记录
err := s.db.Transaction(func(tx *gorm.DB) error {
	// 删除任务的所有视频文件记录
	if err := tx.Where("task_id = ?", *taskID).Delete(&models.VideoFile{}).Error; err != nil {
		return err
	}
	// 删除任务记录
	if err := tx.Delete(&models.VideoRecordingTask{}, *taskID).Error; err != nil {
		return err
	}
	return nil
})
```

### Service Lifecycle Pattern
**Source:** `cmd/server/app.go` (lines 721-728)
**Apply to:** All long-running services (SplittingService)

```go
// 启动转换服务
if a.conversionService != nil {
	if err := a.conversionService.Start(); err != nil {
		return fmt.Errorf("failed to start conversion service: %w", err)
	}
	a.logger.Info("转换服务启动成功")
}
```

### FFmpeg Command Construction Pattern
**Source:** `internal/services/conversion_service.go` (lines 316-325)
**Apply to:** All FFmpeg operations (split, snapshot)

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
```

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `frontend/src/components/TimelineWithMarkers.tsx` | component | event-driven | No interactive timeline component exists yet - must extend Slider with custom marks |

---

## Implementation Notes

### Critical Integration Points

1. **VideoFileService Callback Pattern**: All services (RecordingService, ConversionService, SplittingService) must call `videoFileService.CreateFileFromTask()` or `CreateSegmentFile()` after MP4 generation completes.

2. **Migration Registry**: New migrations must be registered in `internal/migrations/registry.go` (need to verify if this file exists or create it following the pattern from `app.go`).

3. **Route Registration**: New split routes must be added to `cmd/server/app.go` in the `registerRoutes()` function (around line 513-688).

4. **Handler Initialization**: New handlers must be added to the `Handlers` struct and initialized in `initHandlers()` (around line 442-511).

5. **Service Registration**: SplittingService must be started in `registerServices()` (around line 690-731).

6. **Permission Constants**: New permissions (FILE_SPLIT, RECORDING_SNAPSHOT) must be added to `internal/models/permission_constants.go`.

---

## Metadata

**Analog search scope:**
- `internal/services/*.go`
- `internal/handlers/*.go`
- `internal/models/*.go`
- `frontend/src/pages/**/*.tsx`
- `frontend/src/api/*.ts`
- `cmd/server/app.go`

**Files scanned:** 20+ files
**Pattern extraction date:** 2026-04-17
**Confidence level:** HIGH - All major patterns extracted from production codebase

---

*Phase: 01-video-splitting*
*Pattern mapping complete: 2026-04-17*
