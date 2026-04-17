---
phase: 01-video-splitting
reviewed: 2025-04-17T12:30:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - cmd/server/app.go
  - frontend/src/pages/files/index.tsx
  - frontend/src/pages/tasks/index.tsx
  - frontend/src/utils/permissions.ts
  - internal/handlers/split_handler.go
  - internal/handlers/split_handler_test.go
  - internal/models/video_file.go
  - internal/services/snapshot_service.go
  - internal/services/snapshot_service_test.go
  - internal/services/splitting_service.go
  - internal/services/splitting_service_test.go
  - internal/services/video_file_service.go
  - internal/services/video_file_service_test.go
findings:
  critical: 3
  warning: 8
  info: 4
  total: 15
status: issues_found
---

# Phase 1: Code Review Report

**Reviewed:** 2025-04-17T12:30:00Z
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Reviewed 13 source files implementing video splitting and snapshot functionality. The codebase demonstrates good structure with proper separation of concerns between handlers, services, and models. However, several critical issues were identified including potential race conditions, missing error handling, and security concerns.

### Key Findings:
- **Critical:** Race conditions in status map updates, missing validation on user input, and potential command injection vulnerabilities
- **Warnings:** Unchecked error returns, missing nil checks, and potential resource leaks
- **Info:** Code style improvements and minor optimizations

## Critical Issues

### CR-01: Race Condition in SplittingService Status Map Updates

**File:** `internal/services/splitting_service.go:94-96, 137-140`
**Issue:** The status map is updated without holding the mutex lock during `SubmitSplit`, creating a race condition where the status could be "processing" when the queue is full, but later updated to "failed" by a different goroutine.

**Fix:**
```go
func (s *SplittingService) SubmitSplit(videoFileID uint, markers []float64, reEncode bool, createdBy uint) error {
    task := &SplitTask{
        VideoFileID: videoFileID,
        Markers:     markers,
        ReEncode:    reEncode,
        CreatedBy:   createdBy,
        CreatedAt:   time.Now(),
    }
    
    s.statusMu.Lock()
    defer s.statusMu.Unlock() // Hold lock for entire operation
    s.statusMap[videoFileID] = "processing"

    select {
    case s.taskQueue <- task:
        s.logger.Info("分割任务已提交", zap.Uint("video_file_id", videoFileID), zap.Int("markers", len(markers)))
        return nil
    default:
        s.statusMap[videoFileID] = "failed"
        return fmt.Errorf("分割任务队列已满")
    }
}
```

### CR-02: Missing Validation on Marker Values

**File:** `internal/handlers/split_handler.go:44-61`
**Issue:** The handler validates marker count but doesn't validate that marker values are within the video's duration range. This could cause FFmpeg to fail silently or produce incorrect segments.

**Fix:**
```go
// Validate markers
if len(req.Markers) == 0 {
    response.GinError(c, response.CodeInvalidRequest, "至少需要一个分割标记点")
    return
}
if len(req.Markers) > 20 {
    response.GinError(c, response.CodeInvalidRequest, "最多支持20个分割标记点")
    return
}

// Validate marker values are positive and within video duration
videoFile, err := h.videoFileService.GetFileByID(uint(id))
if err != nil {
    response.GinError(c, response.CodeInternalError, "视频文件不存在")
    return
}
maxDuration := float64(videoFile.Duration)
for _, marker := range req.Markers {
    if marker < 0 {
        response.GinError(c, response.CodeInvalidRequest, "分割标记点不能为负数")
        return
    }
    if marker > maxDuration {
        response.GinError(c, response.CodeInvalidRequest, 
            fmt.Sprintf("分割标记点 %.2f 超出视频时长 %.2f 秒", marker, maxDuration))
        return
    }
}
```

### CR-03: Potential Command Injection in FFmpeg Arguments

**File:** `internal/services/splitting_service.go:186-199, snapshot_service.go:101-110`
**Issue:** While file paths are controlled internally, the use of `fmt.Sprintf` for FFmpeg arguments without explicit sanitization could be vulnerable if file paths contain malicious characters. Additionally, the `seg.end` value of 0 is used to mean "until end" which could be confused.

**Fix:**
```go
// Validate and sanitize file paths before using in FFmpeg commands
func validateFilePath(path string) error {
    // Check for shell metacharacters
    dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "<", ">"}
    for _, char := range dangerousChars {
        if strings.Contains(path, char) {
            return fmt.Errorf("file path contains dangerous character: %s", char)
        }
    }
    return nil
}

// In processSplit:
if err := validateFilePath(sourceFile.FilePath); err != nil {
    s.logger.Error("无效的文件路径", zap.Error(err))
    return
}

// Use explicit check for end-of-file instead of magic 0 value
if seg.end > 0 && seg.end <= seg.start {
    s.logger.Error("Invalid segment: end time must be greater than start time")
    continue
}
```

## Warnings

### WR-01: Unchecked Error Returns in VideoFileService

**File:** `internal/services/video_file_service.go:789, 801`
**Issue:** `os.Stat` and `os.Stat` errors are not checked, which could lead to files with size 0 being created in the database.

```go
// Line 789 - unchecked error
fileInfo, _ := os.Stat(filePath)
fileSize := int64(0)
if fileInfo != nil {
    fileSize = fileInfo.Size()
}
```

**Fix:**
```go
fileInfo, err := os.Stat(filePath)
if err != nil {
    return nil, fmt.Errorf("无法获取文件信息: %w", err)
}
fileSize := fileInfo.Size()
```

### WR-02: Potential Nil Pointer Dereference

**File:** `internal/services/snapshot_service.go:124-128`
**Issue:** If `parentFile.ID` is 0 (which is valid for nil pointer), it could be passed as a pointer causing issues.

**Fix:**
```go
var parentID *uint
if err := s.db.Where("task_id = ? AND source_type = ?", taskID, models.SourceTypeRecording).First(&parentFile).Error; err == nil {
    parentID = &parentFile.ID
}
```

### WR-03: Missing Transaction Rollback on Error

**File:** `internal/services/video_file_service.go:226-236`
**Issue:** The transaction doesn't explicitly rollback on error, relying on GORM's automatic rollback. This could leave the database in an inconsistent state if the transaction fails partway through.

**Fix:**
```go
err := s.db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Where("task_id = ?", *taskID).Delete(&models.VideoFile{}).Error; err != nil {
        return err // Explicit return causes rollback
    }
    if err := tx.Delete(&models.VideoRecordingTask{}, *taskID).Error; err != nil {
        return err // Explicit return causes rollback
    }
    return nil // Commit
})
if err != nil {
    return fmt.Errorf("删除数据库记录失败: %w", err)
}
```

### WR-04: Resource Leak in copyFile

**File:** `internal/services/snapshot_service.go:148-175`
**Issue:** If `destFile.Write` fails, the source file is closed but the destination file is not cleaned up, potentially leaving temporary files.

**Fix:**
```go
func copyFile(src, dst string) error {
    sourceFile, err := os.Open(src)
    if err != nil {
        return err
    }
    defer sourceFile.Close()

    destFile, err := os.Create(dst)
    if err != nil {
        return err
    }
    defer func() {
        destFile.Close()
        if err != nil {
            os.Remove(dst) // Clean up on error
        }
    }()

    buf := make([]byte, 32*1024)
    for {
        n, err := sourceFile.Read(buf)
        if n > 0 {
            if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
                return writeErr
            }
        }
        if err != nil {
            if err == io.EOF {
                return nil
            }
            return err
        }
    }
}
```

### WR-05: Missing Context Cancellation Handling

**File:** `internal/services/splitting_service.go:182-185`
**Issue:** The context is stored in `cancelFuncs` but never cleaned up after completion, potentially causing memory leaks for long-running splits.

**Fix:**
```go
defer func() {
    s.mu.Lock()
    delete(s.cancelFuncs, task.VideoFileID)
    s.mu.Unlock()
}()
```

### WR-06: Inconsistent Error Handling in Split Handler

**File:** `internal/handlers/split_handler.go:75-78`
**Issue:** The error from `SubmitSplit` is wrapped but the HTTP status code is always `CodeInternalError`, even for validation errors that should return 400.

**Fix:**
```go
if err := h.splittingService.SubmitSplit(uint(id), req.Markers, req.ReEncode, userID); err != nil {
    if strings.Contains(err.Error(), "队列已满") {
        response.GinError(c, response.CodeInvalidRequest, err.Error())
    } else {
        response.GinError(c, response.CodeInternalError, "提交分割任务失败: "+err.Error())
    }
    return
}
```

### WR-07: Missing Authorization Check in GetSplitStatus

**File:** `internal/handlers/split_handler.go:113-131`
**Issue:** The handler doesn't verify that the user has permission to view the split status for the requested video file.

**Fix:**
```go
func (h *SplitHandler) GetSplitStatus(c *gin.Context) {
    idStr := c.Param("id")
    id, err := strconv.ParseUint(idStr, 10, 32)
    if err != nil {
        response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
        return
    }

    // Verify user has access to this video
    userID := middleware.GetUserID(c)
    file, err := h.videoFileService.GetFileByID(uint(id))
    if err != nil {
        response.GinError(c, response.CodeInternalError, "视频文件不存在")
        return
    }
    if !middleware.GetIsAdmin(c) && file.CreatedBy != userID {
        response.GinError(c, response.CodeInvalidRequest, "无权查看此视频文件")
        return
    }

    status := h.splittingService.GetSplitStatus(uint(id))
    // ... rest of function
}
```

### WR-08: Unbounded Goroutine Creation in Workers

**File:** `internal/services/splitting_service.go:71-78`
**Issue:** The worker count is hardcoded to 2, but the task queue can hold 100 items. If all 100 items are submitted quickly, only 2 workers will process them, but there's no backpressure mechanism to prevent queue overflow.

**Fix:**
```go
func NewSplittingService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, videoFileService *VideoFileService) *SplittingService {
    // Calculate workers based on CPU cores or configuration
    workers := runtime.NumCPU()
    if workers > 4 {
        workers = 4 // Cap at 4 workers
    }
    
    return &SplittingService{
        // ...
        workers:  workers,
        taskQueue: make(chan *SplitTask, workers * 10), // Scale queue with workers
        // ...
    }
}
```

## Info

### IN-01: Inconsistent Naming Convention

**File:** `frontend/src/pages/files/index.tsx:352-359`
**Issue:** The "查看原视频" button navigates to `/files` instead of navigating to the specific parent video file.

**Fix:**
```tsx
{record.parent_id && (
  <Button
    type="link"
    size="small"
    style={{ padding: 0, fontSize: 12 }}
    onClick={() => navigate(`/files/${record.parent_id}`)}
  >
    查看原视频
  </Button>
)}
```

### IN-02: Magic Numbers in Timeout Values

**File:** `internal/services/snapshot_service.go:97, splitting_service.go:182`
**Issue:** Timeout values (10 minutes, 30 minutes) are hardcoded without constants.

**Fix:**
```go
const (
    SnapshotTimeout = 10 * time.Minute
    SplitSegmentTimeout = 30 * time.Minute
)
```

### IN-03: Missing Unit Tests for Critical Paths

**Files:** `internal/handlers/split_handler_test.go, internal/services/splitting_service_test.go`
**Issue:** Test files contain only stubs with `t.Skip()` calls. No actual tests are implemented for critical functionality.

**Fix:** Implement the test cases outlined in the stub comments.

### IN-04: Duplicate Code in Status Map Access

**File:** `internal/services/splitting_service.go:94-96, 111-118`
**Issue:** The pattern of acquiring lock, reading map, and releasing lock is duplicated. Could be extracted to a helper method.

**Fix:**
```go
func (s *SplittingService) getStatus(videoFileID uint) string {
    s.statusMu.RLock()
    defer s.statusMu.RUnlock()
    if status, ok := s.statusMap[videoFileID]; ok {
        return status
    }
    return ""
}

func (s *SplittingService) setStatus(videoFileID uint, status string) {
    s.statusMu.Lock()
    defer s.statusMu.Unlock()
    s.statusMap[videoFileID] = status
}
```

---

_Reviewed: 2025-04-17T12:30:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
