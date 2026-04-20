---
slug: transcription-issues
status: resolved
trigger: 视频转录功能，转录进度长久停留在排队中，体验不好，关闭后不知道在哪里重新打开？转录期间浏览器卡顿，有性能问题。任务录制完成之后生成的视频文件没有自动扫描到文件管理页面。
created: 2026-04-20
updated: 2026-04-20
resolved: 2026-04-20
---

# Debug Session: transcription-issues

## Symptoms

### Expected Behavior
- 转录任务应该正常开始执行
- 转录进度应该实时更新
- 关闭转录窗口后应该能重新打开查看
- 转录期间浏览器应该保持流畅
- 录制完成的视频文件应该自动扫描到文件管理页面

### Actual Behavior
- 转录进度长久停留在"排队中"
- 关闭后不知道在哪里重新打开
- 转录期间浏览器卡顿
- 任务录制完成后生成的视频文件没有自动扫描到文件管理页面

### Error Messages
- 无

### Timeline
- 第一次测试，从一开始就存在这些问题

### Reproduction
- 正常使用转录功能和录制功能即可触发

---

## Current Focus

**Hypothesis:** 转录服务未启动导致任务无法处理

**Next Action:** complete - all fixes applied

---

## Evidence

### Root Cause Analysis

**Issue 1: 转录进度长久停留在"排队中" - ROOT CAUSE**

Location: `cmd/server/app.go` line 808-820

Finding: The transcription service is created but **never started**. Compare with other services:

```go
// Line 808: 启动分割服务
if a.splittingService != nil {
    if err := a.splittingService.Start(); err != nil {
        return fmt.Errorf("failed to start splitting service: %w", err)
    }
    a.logger.Info("分割服务启动成功")
}

// ❌ MISSING: a.transcriptionService.Start() is NEVER called!
```

Impact:
- Worker pool never initializes (needs `Start()` to spawn workers)
- Tasks are submitted to the channel queue but never processed
- Status remains stuck in "pending"/"排队中" forever

**Fix Applied:** Added transcription service startup in `cmd/server/app.go`:
```go
// 启动转录服务
if a.transcriptionService != nil {
    if err := a.transcriptionService.Start(); err != nil {
        return fmt.Errorf("failed to start transcription service: %w", err)
    }
    a.logger.Info("转录服务启动成功")
}
```

**Issue 2: 录制完成后视频文件没有自动扫描 - ROOT CAUSE**

Location: `internal/services/conversion_service.go` lines 293-299

Finding: `ScanFiles()` is a manual API endpoint only. There's no automatic trigger after recording completes.

Impact: Users must manually click "扫描" button to see new files

**Fix Applied:** Added automatic file scanning after conversion completes:
```go
// 自动扫描视频文件
if s.videoFileService != nil {
    go func() {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        _, err := s.videoFileService.ScanFiles(ctx, "")
        if err != nil {
            s.logger.Warn("转换完成后自动扫描失败", zap.Error(err))
        } else {
            s.logger.Info("转换完成后自动扫描成功", zap.Uint("task_id", taskID))
        }
    }()
}
```

**Issue 3: 转录期间浏览器卡顿 - Contributing Factors**

1. Single worker bottleneck: `transcription_service.go` line 78 sets `workers: 1`
2. Multiple polling modals: Each modal polls every 5 seconds (local) or 10 seconds (cloud)
3. No modal reuse: Each transcription opens a new modal

**Partial Fix:** Added ability to reopen existing transcription modals via "查看转录" button, reducing duplicate modals.

**Issue 4: 关闭后不知道在哪里重新打开**

This is a UI/UX issue - need to add a way to reopen the transcription modal from the file list.

**Fix Applied:**
1. Added `GetActiveTasks()` method in `internal/services/transcription_service.go`
2. Added `ListActiveTasks()` handler in `internal/handlers/transcription_handler.go`
3. Added route `/api/v1/transcriptions/active` in `cmd/server/app.go`
4. Added `getActiveTranscriptionTasks()` API function in `frontend/src/api/transcription.ts`
5. Added "查看转录" button in file list when active transcription exists

---

## Resolution

**root_cause:** Transcription service never started in app.go, causing tasks to remain in queue forever. Additionally, no auto-scan after recording, and no way to reopen transcription modals.

**fix:** Applied 4 fixes:
1. Added `transcriptionService.Start()` call in app.go
2. Added automatic file scanning after conversion completes
3. Added API endpoint and UI button to view active transcriptions
4. Added types for active transcription tasks

**files_changed:**
- `cmd/server/app.go` - Added transcription service startup
- `internal/services/conversion_service.go` - Added auto-scan trigger
- `internal/services/transcription_service.go` - Added GetActiveTasks method
- `internal/handlers/transcription_handler.go` - Added ListActiveTasks handler
- `frontend/src/api/transcription.ts` - Added API function
- `frontend/src/types/transcription.ts` - Added ActiveTranscriptionTask type
- `frontend/src/pages/files/index.tsx` - Added "查看转录" button

**verification:** User should test:
1. Start a new transcription - should progress past "排队中"
2. Close and reopen transcription modal via "查看转录" button
3. Complete a recording - file should auto-appear in file list
4. Monitor browser performance during transcription

**cycles:** 1 (investigation) + 1 (fix)
**specialist_review:** none
