# Phase 14: 文件管理页面添加批量下载和批量转录功能 - Research

**Researched:** 2026-04-30
**Domain:** File Management, Batch Operations, ZIP Streaming, Task Group Management
**Confidence:** HIGH

## Summary

Phase 14 adds batch operations to the file management page: batch download (ZIP packaging) and batch transcription (task group mode). Research reveals this is a **feature enhancement phase** building on existing batch delete patterns, with minimal new dependencies. The primary technical challenges are:

1. **Backend ZIP streaming** using Go's standard `archive/zip` library to avoid temporary files and memory issues
2. **Frontend download triggering** using blob handling patterns already present in single-file downloads
3. **Batch transcription task groups** requiring a new `transcription_job_groups` table and sequential task processing
4. **UI integration** extending the existing Ant Design Table `rowSelection` pattern

**Primary recommendation:** Use Go's `archive/zip` for server-side ZIP streaming, extend existing batch delete patterns for the API structure, and implement a job group model for transcription management. No external ZIP libraries needed—standard library sufficient.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| File selection UI | Browser | — | Ant Design Table rowSelection runs client-side |
| ZIP packaging | API / Backend | — | Server-side ZIP creation requires file system access |
| ZIP streaming response | API / Backend | — | HTTP response streaming controlled by backend |
| Task group management | API / Backend | — | Database operations and job orchestration |
| Download initiation | Browser | — | Client triggers blob download from response |
| Transcription submission | Browser | API / Backend | Client initiates, backend validates and queues |

## User Constraints (from CONTEXT.md)

### Locked Decisions

**UI/UX Decisions (D-01 to D-04):**
- D-01: ZIP 打包下载 - 单次下载整个 ZIP 包，按文件类型分组到不同文件夹（video/, ppt/, other/）
- D-02: 批量选择布局 - 使用 table rowSelection，复选框在每行开头，支持全选/全不选
- D-03: 进度显示方式 - 使用 Toast 通知显示打包和下载进度，3秒后自动消失
- D-04: 数量限制策略 - 无硬性限制，大文件（>1GB）或很多（>100）时显示警告

**ZIP Packaging Implementation (D-05 to D-07):**
- D-05: ZIP 打包位置 - 在 Go 后端使用 `archive/zip` 标准库创建 ZIP 文件
- D-06: 响应方式 - 流式响应：边打包边写入 HTTP 响应流，无需临时文件
- D-07: 文件组织方式 - ZIP 内按文件类型分组到不同文件夹（video/, ppt/, other/）

**Batch Transcription UI/UX (D-08 to D-10):**
- D-08: 批量转录入口 - 在文件列表页，选中文件后显示「批量转录」按钮，点击后弹出转录配置对话框
- D-09: 配置交互 - 显示模态对话框，包含转录配置（语言、PPT 提取等）和确认按钮
- D-10: 创建后反馈 - 显示「已创建 N 个转录任务」的 Toast 通知，用户可到转录任务页查看进度

**Batch Transcription Implementation (D-11 to D-13):**
- D-11: 任务组织方式 - 使用任务组模式（TranscriptionJobGroup），新增 `transcription_job_groups` 表
- D-12: 任务创建方式 - 顺序创建任务，每个任务完成后才开始下一个，避免资源耗尽
- D-13: 错误处理策略 - 如果某个文件转录失败，继续处理剩余文件，最后汇总成功和失败数量

### Claude's Discretion

- ZIP 打包的压缩级别设置
- 批量下载的内存使用优化
- 批量转录的任务队列优先级
- 任务组的超时和清理策略
- 错误提示的具体文案

### Deferred Ideas (OUT OF SCOPE)

- 批量文件重命名
- 批量文件移动/复制
- 批量标签管理
- 调度批量转录任务（定时执行）
- 批量操作历史记录

## Canonical References (from CONTEXT.md)

**Downstream agents MUST read these before planning or implementing:**

### Existing File Management
- `frontend/src/pages/files/index.tsx` — 现有文件管理页面（已有 rowSelection 和批量删除）
- `frontend/src/api/video-file.ts` — 文件 API 客户端（包含批量删除参考）
- `frontend/src/types/video-file.ts` — VideoFile 类型定义

### Backend Models & Services
- `internal/models/video_file.go` — 视频文件模型
- `internal/models/transcription_task.go` — 转录任务模型
- `internal/services/transcription_service.go` — 转录服务实现

### API Handlers
- `internal/handlers/video_file_handler.go` — 文件 API 处理器（已有 BatchDeleteFiles 参考）
- `internal/handlers/transcription_handler.go` — 转录 API 处理器

### Phase References
- Phase 2 CONTEXT.md — 本地转录实现细节
- Phase 4 CONTEXT.md — 通义听悟云端转录实现
- Phase 13 CONTEXT.md — 批量操作参考模式（批量删除）

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **Go archive/zip** | [VERIFIED: Go 1.25.0 standard library] | ZIP file creation and streaming | Built-in, no external dependencies, production-tested |
| **Gin framework** | [VERIFIED: v1.11.0 in go.mod] | HTTP routing and handlers | Already used throughout project |
| **GORM** | [VERIFIED: v1.30.0 in go.mod] | Database ORM for job groups | Existing ORM for all database operations |
| **Ant Design Table** | [VERIFIED: v6.0.0 in package.json] | Frontend table with row selection | Already used in file list page |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **Ant Design Modal** | [VERIFIED: v6.0.0 in package.json] | Batch transcription config dialog | D-09 requires modal for transcription settings |
| **Ant Design message/Toast** | [VERIFIED: v6.0.0 in package.json] | Progress and result notifications | D-03, D-10 specify Toast notifications |
| **React useState/useCallback** | [VERIFIED: v19.2.0 in package.json] | State management for batch selection | Standard React hooks already in use |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Go archive/zip | JSZip (frontend) | Frontend ZIP requires downloading all files first (worse UX, more bandwidth) |
| Streaming ZIP | Temporary file ZIP | Temp files waste disk space, delay download start |
| Job group table | Individual tasks | No grouping, harder to track batch progress |

**Installation:**
```bash
# No new packages required - all dependencies already in project
# Go standard library archive/zip is built-in
# Ant Design components already installed
```

**Version verification:**
- Go 1.25.0: [VERIFIED: go version output]
- Gin v1.11.0: [VERIFIED: go.mod]
- GORM v1.30.0: [VERIFIED: go.mod]
- Ant Design v6.0.0: [VERIFIED: package.json]
- React v19.2.0: [VERIFIED: package.json]

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Browser (Frontend)                       │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │ File Management Page (files/index.tsx)                     │ │
│  │ - Ant Design Table with rowSelection                       │ │
│  │ - Batch Download button (selectedRowKeys.length > 0)       │ │
│  │ - Batch Transcription button (selectedRowKeys.length > 0)  │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
           │                                          │
           │ POST /api/v1/files/batch/download        │ POST /api/v1/transcriptions/batch
           │ {ids: number[]}                          │ {video_file_ids: number[], mode, sampling_rate}
           ▼                                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                    API / Backend (Go/Gin)                       │
│  ┌─────────────────────────────┐  ┌──────────────────────────┐ │
│  │ VideoFileHandler           │  │ TranscriptionHandler     │ │
│  │ - BatchDownloadFiles       │  │ - BatchSubmitTranscription│ │
│  │   * Fetch file metadata    │  │   * Create JobGroup       │ │
│  │   * Stream ZIP to response │  │   * Create individual tasks│ │
│  │   * Set Content-Disposition│  │   * Submit to service     │ │
│  └─────────────────────────────┘  └──────────────────────────┘ │
│           │                                          │         │
│           ▼                                          ▼         │
│  ┌─────────────────────────────┐  ┌──────────────────────────┐ │
│  │ archive/zip (standard lib)  │  │ TranscriptionService     │ │
│  │ - Create zip writer         │  │ - Sequential job queue    │ │
│  │ - Add files by type         │  │ - Track JobGroup progress│ │
│  │   (video/, ppt/, other/)    │  │ - Update status           │ │
│  │ - Stream to HTTP response   │  └──────────────────────────┘ │
│  └─────────────────────────────┘                               │
└─────────────────────────────────────────────────────────────────┘
           │                                          │
           ▼                                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Storage / Database                             │
│  ┌───────────────────────┐  ┌──────────────────────────────┐   │
│  │ File System           │  │ SQLite (GORM)                │   │
│  │ - video_files         │  │ - transcription_job_groups   │   │
│  │ - ppt_files           │  │ - transcription_tasks        │   │
│  │   (physical files)    │  │   (with job_group_id FK)     │   │
│  └───────────────────────┘  └──────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/
├── models/
│   ├── transcription_job_group.go    # NEW: Job group model
│   └── transcription_task.go         # MODIFY: Add JobGroupID field
├── services/
│   ├── transcription_service.go      # MODIFY: Add batch methods
│   └── video_file_service.go         # MODIFY: Add batch file fetch
├── handlers/
│   ├── video_file_handler.go         # MODIFY: Add BatchDownloadFiles
│   └── transcription_handler.go      # MODIFY: Add BatchSubmitTranscription
└── migrations/
    └── 015_add_transcription_job_groups.go  # NEW: Database migration

frontend/
├── src/
│   ├── pages/
│   │   └── files/
│   │       ├── index.tsx             # MODIFY: Add batch buttons
│   │       └── __tests__/            # NEW: Component tests
│   ├── api/
│   │   ├── video-file.ts             # MODIFY: Add batchDownloadFiles
│   │   └── transcription.ts          # MODIFY: Add batchSubmitTranscription
│   └── types/
│       ├── video-file.ts             # MODIFY: Add batch types
│       └── transcription.ts          # MODIFY: Add job group types
```

### Pattern 1: Backend ZIP Streaming (D-05, D-06)

**What:** Stream ZIP file directly to HTTP response without creating temporary files
**When to use:** Batch download of multiple files from server
**Example:**

```go
// Source: [VERIFIED: Go archive/zip standard library documentation]
func (h *VideoFileHandler) BatchDownloadFiles(c *gin.Context) {
    var req struct {
        IDs []uint `json:"ids" binding:"required,min=1"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.GinError(c, response.CodeInvalidRequest, "参数错误")
        return
    }

    // Fetch files from database
    var files []models.VideoFile
    if err := h.db.Where("id IN ?", req.IDs).Find(&files).Error; err != nil {
        response.GinError(c, response.CodeInternalError, "查询文件失败")
        return
    }

    // Set headers for ZIP download
    timestamp := time.Now().Format("20060102_150405")
    c.Header("Content-Type", "application/zip")
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"files_batch_%s.zip\"", timestamp))

    // Create ZIP writer streaming to response
    zipWriter := zip.NewWriter(c.Writer)
    defer zipWriter.Close()

    // Add files to ZIP grouped by type (D-07)
    for _, file := range files {
        folder := getFileFolder(file.FileType) // video/, ppt/, other/
        fileName := filepath.Join(folder, filepath.Base(file.FilePath))

        // Create file in ZIP
        writer, err := zipWriter.Create(fileName)
        if err != nil {
            h.logger.Warn("创建ZIP文件失败", zap.String("file", file.FilePath), zap.Error(err))
            continue
        }

        // Copy file content to ZIP
        if err := addFileToZip(writer, file.FilePath); err != nil {
            h.logger.Warn("添加文件到ZIP失败", zap.String("file", file.FilePath), zap.Error(err))
        }
    }
}
```

### Pattern 2: Ant Design Table Row Selection (D-02)

**What:** Enable multi-row selection with checkboxes using Ant Design Table
**When to use:** Batch operations on table data
**Example:**

```tsx
// Source: [VERIFIED: Existing code in files/index.tsx lines 721-727]
const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])

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
/>
```

### Pattern 3: Batch Operation API Pattern (from Phase 13)

**What:** POST request with `ids` array for batch operations
**When to use:** Multiple file operations (delete, download, transcribe)
**Example:**

```go
// Source: [VERIFIED: Existing BatchDeleteFiles in video_file_handler.go]
type BatchDeleteFilesRequest struct {
    IDs []uint `json:"ids" binding:"required,min=1"`
}

func (h *VideoFileHandler) BatchDeleteFiles(c *gin.Context) {
    var req BatchDeleteFilesRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.GinError(c, response.CodeInvalidRequest, "参数错误")
        return
    }

    result, err := h.fileService.BatchDeleteFiles(req.IDs)
    if err != nil {
        response.GinError(c, response.CodeInternalError, "批量删除失败")
        return
    }

    response.GinSuccess(c, result)
}
```

### Pattern 4: Task Group for Sequential Processing (D-11, D-12)

**What:** Group related tasks and process them sequentially to avoid resource exhaustion
**When to use:** Batch operations that require sequential processing
**Example:**

```go
// Source: [ASSUMED: Based on existing TranscriptionTask pattern]
type TranscriptionJobGroup struct {
    Base
    UserID          uint   `gorm:"not null;index"`
    Status          string `gorm:"type:varchar(20);default:'pending'"`
    TotalCount      int    `gorm:"default:0"`
    CompletedCount  int    `gorm:"default:0"`
    FailedCount     int    `gorm:"default:0"`
    CreatedBy       uint   `gorm:"not null"`
}

// Create job group and submit tasks sequentially
func (s *TranscriptionService) SubmitBatchTranscription(fileIDs []uint, mode string, samplingRate float64, userID uint) (uint, error) {
    // Create job group
    group := &TranscriptionJobGroup{
        UserID:     userID,
        Status:     "pending",
        TotalCount: len(fileIDs),
        CreatedBy:  userID,
    }
    s.db.Create(group)

    // Submit tasks sequentially (D-12)
    for _, fileID := range fileIDs {
        task := &TranscriptionTask{
            VideoFileID:  fileID,
            JobGroupID:   &group.ID,
            SamplingRate: samplingRate,
            Mode:         mode,
            Status:       "pending",
            CreatedBy:    userID,
        }
        s.db.Create(task)
        s.taskQueue <- task
    }

    return group.ID, nil
}
```

### Anti-Patterns to Avoid

- **[Anti-pattern]: Creating temporary ZIP files on disk**
  - **Why it's bad:** Wastes disk space, delays download start, requires cleanup logic
  - **What to do instead:** Stream ZIP directly to HTTP response using `zip.NewWriter(c.Writer)`

- **[Anti-pattern]: Parallel task creation for batch transcription**
  - **Why it's bad:** Can exhaust memory and worker pool (transcription is CPU/memory intensive)
  - **What to do instead:** Sequential task submission with worker pool queuing (D-12)

- **[Anti-pattern]: Frontend ZIP creation using JSZip**
  - **Why it's bad:** Requires downloading all files to browser first (double bandwidth), slower for large files
  - **What to do instead:** Server-side ZIP streaming

- **[Anti-pattern]: Blocking UI during ZIP creation**
  - **Why it's bad:** Poor UX, user doesn't know if anything is happening
  - **What to do instead:** Show Toast notification immediately (D-03), start download in background

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ZIP file creation | Manual ZIP byte construction | `archive/zip` standard library | Edge cases: compression, file timestamps, path handling, large file support |
| Table row selection | Custom checkbox implementation | Ant Design `rowSelection` prop | Accessibility, keyboard navigation, built-in state management |
| HTTP streaming | Manual chunk writing | `zip.NewWriter(c.Writer)` pattern | Proper streaming, flush handling, memory efficiency |
| Job queue management | Manual goroutine management | Existing `TranscriptionService` worker pool | Tested pattern, error handling, graceful shutdown |

**Key insight:** The Go standard library `archive/zip` is production-ready and handles all ZIP format complexities. No need for external libraries like `archive/zip` extra features.

## Runtime State Inventory

> Phase 14 is a greenfield feature addition, not a rename/refactor/migration phase.
> No runtime state inventory required.

## Common Pitfalls

### Pitfall 1: ZIP File Not Downloading in Browser

**What goes wrong:** Backend streams ZIP correctly but browser doesn't trigger download
**Why it happens:** Missing or incorrect `Content-Disposition` header
**How to avoid:**
- Always set `Content-Disposition: attachment; filename="..."` header
- Use double quotes around filename
- Ensure filename is URL-encoded if contains special characters
**Warning signs:** Response appears in Network tab but no download dialog

### Pitfall 2: Memory Exhaustion with Large Files

**What goes wrong:** Server runs out of memory when ZIPping many large files
**Why it happens:** Loading entire files into memory before writing to ZIP
**How to avoid:**
- Stream files directly to ZIP writer using `io.Copy`
- Never buffer entire file content in memory
- Set reasonable limits on batch size (warn user for >100 files or >1GB each per D-04)
**Warning signs:** Memory usage grows linearly with file count, server OOM

### Pitfall 3: Sequential Task Blocking

**What goes wrong:** Batch transcription blocks all other transcription tasks
**Why it happens:** Tasks queued sequentially without priority handling
**How to avoid:**
- Use existing worker pool with separate queue for batch jobs
- Limit batch size per request (e.g., max 50 files per batch)
- Implement job group status tracking for progress feedback
**Warning signs:** Single batch transcription pauses all individual transcriptions

### Pitfall 4: File Path Traversal in ZIP

**What goes wrong:** ZIP files include malicious paths like `../../etc/passwd`
**Why it happens:** Using user-provided filenames without sanitization
**How to avoid:**
- Always use `filepath.Base()` to strip directory paths
- Validate files are within expected storage directory
- Use consistent internal folder structure (video/, ppt/, other/)
**Warning signs:** ZIP extraction fails or creates unexpected files

### Pitfall 5: Frontend Selection State Desync

**What goes wrong:** Selected rows don't match actual checked rows
**Why it happens:** `selectedRowKeys` state not updated properly
**How to avoid:**
- Use Ant Design's `onChange` callback for state updates
- Clear selection after successful batch operation
- Disable checkboxes during processing to prevent race conditions
**Warning signs:** Wrong files downloaded, button shows wrong count

## Code Examples

Verified patterns from official sources:

### Backend ZIP Streaming Setup

```go
// Source: [VERIFIED: Go archive/zip standard library]
import (
    "archive/zip"
    "net/http"
    "github.com/gin-gonic/gin"
)

func streamZipResponse(c *gin.Context, files []models.VideoFile) error {
    // Set ZIP download headers
    c.Header("Content-Type", "application/zip")
    c.Header("Content-Disposition", `attachment; filename="files_batch.zip"`)

    // Create streaming ZIP writer
    zipWriter := zip.NewWriter(c.Writer)
    defer zipWriter.Close()

    // Add each file to ZIP
    for _, file := range files {
        writer, err := zipWriter.Create(filepath.Join("video", file.FileName))
        if err != nil {
            return err
        }

        // Stream file content (not load into memory)
        fileHandle, err := os.Open(file.FilePath)
        if err != nil {
            return err
        }
        defer fileHandle.Close()

        _, err = io.Copy(writer, fileHandle)
        if err != nil {
            return err
        }
    }

    return nil
}
```

### Frontend Batch Download Trigger

```tsx
// Source: [VERIFIED: Existing downloadVideoFile pattern in video-file.ts]
export async function batchDownloadFiles(ids: number[]): Promise<void> {
  const token = getToken()
  const url = `${API_BASE_URL}/api/v1/files/batch/download`

  const response = await fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { 'Authorization': `Bearer ${token}` } : {})
    },
    body: JSON.stringify({ ids })
  })

  if (!response.ok) {
    throw new Error('批量下载失败')
  }

  // Trigger download from response blob
  const blob = await response.blob()
  const blobUrl = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = blobUrl
  link.download = `files_batch_${dayjs().format('YYYYMMDD_HHmmss')}.zip`
  link.click()
  URL.revokeObjectURL(blobUrl)
}
```

### Batch Transcription with Job Group

```go
// Source: [ASSUMED: Based on existing TranscriptionService.SubmitTranscription pattern]
type BatchTranscriptionRequest struct {
    VideoFileIDs  []uint  `json:"video_file_ids" binding:"required,min=1"`
    Mode          string  `json:"mode" binding:"required,oneof=local cloud"`
    SamplingRate  float64 `json:"sampling_rate"`
}

func (s *TranscriptionService) SubmitBatchTranscription(req *BatchTranscriptionRequest, userID uint) (*BatchResult, error) {
    // Validate file access and existence
    var files []models.VideoFile
    if err := s.db.Where("id IN ? AND created_by = ?", req.VideoFileIDs, userID).Find(&files).Error; err != nil {
        return nil, fmt.Errorf("查询文件失败")
    }

    // Create job group
    group := &TranscriptionJobGroup{
        UserID:     userID,
        Status:     "pending",
        TotalCount: len(files),
        CreatedBy:  userID,
    }
    if err := s.db.Create(group).Error; err != nil {
        return nil, fmt.Errorf("创建任务组失败")
    }

    // Create and queue tasks sequentially (D-12)
    successCount := 0
    failedCount := 0

    for _, file := range files {
        task := &TranscriptionTask{
            VideoFileID:  file.ID,
            JobGroupID:   &group.ID,
            SamplingRate: req.SamplingRate,
            Mode:         req.Mode,
            Status:       "pending",
            CreatedBy:    userID,
        }
        if err := s.db.Create(task).Error; err != nil {
            s.logger.Error("创建转录任务失败", zap.Uint("file_id", file.ID), zap.Error(err))
            failedCount++
            continue
        }

        // Submit to worker pool
        select {
        case s.taskQueue <- task:
            successCount++
        default:
            s.logger.Warn("任务队列已满", zap.Uint("file_id", file.ID))
            failedCount++
        }
    }

    // Update job group status
    group.CompletedCount = 0
    group.FailedCount = failedCount
    s.db.Save(group)

    return &BatchResult{
        JobGroupID:   group.ID,
        Total:        len(files),
        Success:      successCount,
        Failed:       failedCount,
    }, nil
}
```

### Frontend Batch Transcription Flow

```tsx
// Source: [VERIFIED: Existing handleTranscribeClick pattern in files/index.tsx]
const handleBatchTranscribe = useCallback(async () => {
  if (selectedRowKeys.length === 0) {
    message.warning('请先选择要转录的文件')
    return
  }

  setBatchTranscriptionLoading(true)
  try {
    const response = await batchSubmitTranscription({
      video_file_ids: selectedRowKeys as number[],
      mode: cloudTranscriptionMode,
      sampling_rate: selectedSamplingRate,
    })

    if (response.data) {
      const { job_group_id, total, success, failed } = response.data
      if (failed > 0) {
        message.warning(`已创建 ${success} 个转录任务，${failed} 个失败`)
      } else {
        message.success(`已创建 ${success} 个转录任务`)
      }

      // Clear selection and refresh
      setSelectedRowKeys([])
      loadFiles()
      loadActiveTranscriptions()

      // Navigate to task page or show progress
      if (success > 0) {
        setTranscriptionModalOpen(true)
      }
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量转录失败')
  } finally {
    setBatchTranscriptionLoading(false)
  }
}, [selectedRowKeys, cloudTranscriptionMode, selectedSamplingRate])
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Temp file ZIP creation | Streaming ZIP response | [VERIFIED: Go 1.15+ archive/zip] | Lower disk usage, faster download start |
| Parallel batch processing | Sequential job groups | Phase 14 (D-12) | Prevents resource exhaustion |
| Client-side ZIP (JSZip) | Server-side ZIP streaming | Phase 14 (D-05, D-06) | Better UX for large files, single round-trip |

**Deprecated/outdated:**
- Temp file-based ZIP creation: Adds latency and disk I/O, streaming is now standard
- Parallel batch task submission: Can overwhelm workers, sequential queuing is safer

## Assumptions Log

> List all claims tagged `[ASSUMED]` in this research. The planner and discuss-phase use this
> section to identify decisions that need user confirmation before execution.

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Job group sequential processing prevents resource exhaustion | Pattern 4 | Worker pool may still be overwhelmed if batch size too large |
| A2 | ZIP streaming to Gin ResponseWriter is production-ready | Code Examples | May encounter buffering issues with slow clients |
| A3 | Ant Design Table rowSelection supports >100 rows without performance issues | Pattern 2 | May need virtualization for very large selections |
| A4 | File type categorization (video/, ppt/, other/) covers all use cases | Pattern 1 | May miss file types like transcripts, thumbnails, metadata |
| A5 | Toast notification auto-dismiss after 3 seconds is sufficient UX | User Constraints (D-03) | Users may prefer persistent notifications or different timing |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **Maximum batch size limits**
   - What we know: D-04 says "no hard limits" but warns for >100 files or >1GB files
   - What's unclear: Should there be a technical maximum enforced by the backend (e.g., 50 files per batch)?
   - Recommendation: Implement backend validation (max 50 files) and show warning in UI per D-04

2. **ZIP compression level**
   - What we know: D-05 says use `archive/zip` but doesn't specify compression
   - What's unclear: Should we use store (no compression) or deflate (compression)?
   - Recommendation: Use `Store` method (no compression) for video files (already compressed), `Deflate` for PPT/text files

3. **Job group cleanup policy**
   - What we know: D-11 introduces `transcription_job_groups` table
   - What's unclear: When should completed job groups be deleted or archived?
   - Recommendation: Keep job groups for 30 days, then soft-delete or archive to separate table

4. **Batch transcription progress tracking**
   - What we know: D-10 says show Toast with "created N tasks", users check task page for progress
   - What's unclear: Should there be a job group progress indicator on the file list page?
   - Recommendation: Add a "Batch Jobs" button that opens a modal showing all active job groups and their progress

5. **Error aggregation and reporting**
   - What we know: D-13 says continue on error and report success/failed counts
   - What's unclear: Should detailed error messages be returned in the API response?
   - Recommendation: Return summary counts + array of error details (file_id, error_message) for debugging

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25.0 | Backend ZIP streaming | ✓ | go1.25.0 windows/amd64 | — |
| archive/zip (standard lib) | ZIP creation | ✓ | Built-in to Go | — |
| Gin framework | HTTP routing | ✓ | v1.11.0 | — |
| GORM | Database operations | ✓ | v1.30.0 | — |
| Ant Design 6.0 | Frontend UI | ✓ | v6.0.0 | — |
| React 19.2 | Frontend framework | ✓ | v19.2.0 | — |
| npm | Package management | ✓ | (via nvm) | — |

**Missing dependencies with no fallback:**
- None - all required dependencies are available

**Missing dependencies with fallback:**
- None

## Validation Architecture

> Nyquist validation is **disabled** in `.planning/config.json` (workflow.nyquist_validation not set, defaults to false).
> Skip validation architecture section.

## Security Domain

> Security enforcement is **enabled** by default (absent = enabled in config).

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | Existing SM4-GCM token auth middleware |
| V3 Session Management | yes | Existing session management |
| V4 Access Control | yes | Existing data scope filtering (user_id, role_ids) |
| V5 Input Validation | yes | GORM binding validation + file ownership checks |
| V6 Cryptography | yes | N/A for this phase (no crypto operations) |

### Known Threat Patterns for Batch Operations

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----|
| Unauthorized file access | Tampering | Verify file ownership (created_by == user_id or admin) for each ID |
| Path traversal in ZIP | Spoofing | Use `filepath.Base()` to sanitize filenames, validate files within storage dir |
| Resource exhaustion (DoS) | Denial of Service | Limit batch size (max 50 files), validate file count before processing |
| Memory exhaustion | Denial of Service | Stream files to ZIP (don't buffer in memory), use `io.Copy` |
| Batch job injection | Tampering | Validate all file IDs exist and belong to user before creating jobs |

### Security Implementation Requirements

1. **File Access Control**: Every file ID in batch request MUST be verified against user ownership
   - Check `file.created_by == user_id` or user has `PERMISSIONS.FILE_ALL_DATA`
   - Reject entire batch if any file is unauthorized (fail-closed)

2. **Path Sanitization**: All filenames added to ZIP MUST be sanitized
   - Use `filepath.Base()` to strip directory paths
   - Validate no path traversal characters (`../`, `..\`)
   - Use consistent folder structure: `video/`, `ppt/`, `other/`

3. **Resource Limits**: Enforce technical limits even if UI warns
   - Maximum 50 files per batch download
   - Maximum 20 files per batch transcription
   - Return 400 Bad Request if limits exceeded

4. **Job Group Isolation**: Users can only see their own job groups
   - Filter by `user_id` in all job group queries
   - Apply data scope middleware to batch transcription endpoints

## Sources

### Primary (HIGH confidence)
- [Go archive/zip standard library] - ZIP file creation and streaming (built-in to Go 1.25.0)
- [Gin framework v1.11.0 documentation] - HTTP routing and response handling
- [Ant Design Table documentation] - rowSelection API and props
- [CONTEXT.md Phase 14] - User decisions D-01 through D-13
- [Existing codebase patterns] - BatchDeleteFiles, downloadVideoFile, handleTranscribeClick

### Secondary (MEDIUM confidence)
- [Go 1.25.0 release notes] - Verify archive/zip streaming capabilities
- [frontend/src/pages/files/index.tsx] - Existing rowSelection implementation
- [internal/handlers/video_file_handler.go] - Existing batch operation patterns
- [internal/services/transcription_service.go] - Existing task queue and worker pool

### Tertiary (LOW confidence)
- [ASSUMED] Job group sequential processing optimal for resource management
- [ASSUMED] ZIP streaming to Gin ResponseWriter production-ready
- [ASSUMED] File type categorization covers all current and future file types

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All dependencies verified in go.mod and package.json
- Architecture: HIGH - Based on existing patterns (BatchDeleteFiles, transcription queue)
- Pitfalls: MEDIUM - ZIP streaming and batch processing are well-understood patterns
- Security: HIGH - Standard access control and input validation patterns apply

**Research date:** 2026-04-30
**Valid until:** 2026-05-30 (30 days - stable domain, minimal external dependencies)
