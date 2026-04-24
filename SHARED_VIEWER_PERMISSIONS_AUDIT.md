# Shared Viewer 权限检查审计报告

**审计日期**: 2026-04-21
**更新日期**: 2026-04-23
**审计范围**: 所有影响 shared_viewer 角色的权限检查
**角色 ID**: 5 (shared_viewer)

---

## 修复进度

| 优先级 | 问题 | 状态 | 日期 |
|--------|------|------|------|
| P0 | HLS 预览权限检查 | ✅ 已修复 | 2026-04-23 |
| P1 | 视频文件重命名 | ✅ 已修复 | 2026-04-23 |
| P1 | PPT 文件重命名 | ✅ 已修复 | 2026-04-23 |
| P1 | PPT 幻灯片合并 | ✅ 已修复 | 2026-04-23 |
| P1 | 视频分割提交 | ✅ 已修复（之前） | - |
| P1 | 转录相关权限 | ✅ 已修复（之前） | - |
| P2 | 录制任务操作权限 | ✅ 已修复 | 2026-04-23 |

**总计**: 16 个权限检查点全部修复完成 ✅

---

## 设计原则回顾

根据 shared_viewer 角色的设计原则：
1. **数据可见性控制** - shared_viewers 可以看到所有用户创建的数据
2. **操作权限不由 shared_viewer 控制** - 操作权限由用户的其他角色（操作员、查看者等）决定
3. **组合行为示例**：
   - shared_viewer + 操作员 = 对所有数据执行操作员允许的操作
   - shared_viewer + 查看者 = 对所有数据只能查看
   - 无 shared_viewer = 只能看到和操作自己创建的数据

---

## 已修复的问题

### P0 - 关键功能

#### 1. HLS 实时预览权限 ✅
**文件**: `internal/handlers/video_recording_task_handler.go`

**修复位置**:
- `GetHLSPreviewHandler` (行 474-490)
- `ServeHLSStream` (行 581-603)

**修复内容**:
```go
hasSharedViewer := middleware.GetHasSharedViewer(c)
if !isAdmin && !hasSharedViewer && task.CreatedBy != userID {
    response.GinError(c, response.CodeForbidden, "无权限访问此预览")
    return
}
```

---

### P1 - 常用操作

#### 2. 视频文件重命名 ✅
**文件**: `internal/services/video_file_service.go`

**修复内容**:
- 方法签名: `RenameVideoFile(id, newName, userID, hasSharedViewer bool)`
- 权限检查: `if !hasSharedViewer && videoFile.CreatedBy != userID`

**Handler**: `internal/handlers/video_file_handler.go:282`

---

#### 3. PPT 文件重命名 ✅
**文件**: `internal/services/ppt_file_service.go`

**修复内容**:
- 方法签名: `RenamePPTFile(id, newName, userID, hasSharedViewer bool)`
- 权限检查: `if !hasSharedViewer && pptFile.SourceVideoFile.CreatedBy != userID`

**Handler**: `internal/handlers/ppt_handler.go:423`

---

#### 4. PPT 幻灯片合并 ✅
**文件**: `internal/services/ppt_merge_service.go`

**修复内容**:
- 方法签名: `MergeSlides(ctx, req, userID, hasSharedViewer bool)`
- 权限检查: `if !hasSharedViewer && videoFile.CreatedBy != userID`

**Handler**: `internal/handlers/ppt_handler.go:278`

---

#### 5. 视频分割提交 ✅
**文件**: `internal/handlers/split_handler.go`

**修复内容**:
- 使用辅助函数: `middleware.CanAccessAllData(c)`
- 权限检查: `if !middleware.CanAccessAllData(c) && file.CreatedBy != userID`

---

#### 6. 转录相关权限 ✅
**文件**: `internal/handlers/transcription_handler.go`

**修复内容**:
- 所有 5 个位置都使用 `middleware.CanAccessAllData(c)`

---

### P2 - 录制任务操作权限 ✅

#### 7-11. 录制任务操作 ✅
**文件**: `internal/services/video_recording_task_service.go`

**修复的方法**:
- `UpdateTask(id, req, userID, hasSharedViewer bool)`
- `StopTask(id, userID, hasSharedViewer bool)`
- `CancelTask(id, userID, hasSharedViewer bool)`
- `RetryTask(id, userID, hasSharedViewer bool)`

**修复内容**:
```go
// 检查权限 (shared_viewers 可以操作任何任务)
if !hasSharedViewer && task.CreatedBy != userID {
    return nil, errors.New("无权限操作此任务")
}
```

**Handler 更新**: `internal/handlers/video_recording_task_handler.go`
- UpdateTask (行 193)
- StopTask (行 320)
- CancelTask (行 346)
- RetryTask (行 371)

---

## 架构改进

### 新增辅助函数

**文件**: `internal/middleware/auth.go`

```go
// GetHasSharedViewer 从context检查用户是否有 shared_viewer 角色
func GetHasSharedViewer(c *gin.Context) bool {
    roleIDs := GetRoleIDs(c)
    for _, roleID := range roleIDs {
        if roleID == 5 { // RoleSharedViewer ID
            return true
        }
    }
    return false
}

// CanAccessAllData 检查用户是否可以访问所有数据
func CanAccessAllData(c *gin.Context) bool {
    return GetIsAdmin(c) || GetHasSharedViewer(c)
}
```

---

## 测试结果

### 单元测试

**视频文件重命名测试**: ✅ 6/6 通过
- TestVideoFileService_RenameVideoFile_Success
- TestVideoFileService_RenameVideoFile_PreservesExtension
- TestVideoFileService_RenameVideoFile_OriginalRecordingImmutable
- TestVideoFileService_RenameVideoFile_OwnershipValidation
- TestVideoFileService_RenameVideoFile_RollbackOnFilesystemError
- TestVideoFileService_RenameVideoFile_DuplicateDetection

**编译状态**: ✅ 通过

---

## 提交记录

1. `9ec9afb` - fix(shared_viewer): P0/P1 权限问题修复 - HLS预览、文件重命名、PPT合并
2. `c95a580` - fix(shared_viewer): P2 录制任务操作权限重构

---

## Phase 09 总结

**Phase 09: Multi-Role Permissions** 已完成 ✅

所有 shared_viewer 权限检查点已修复：
- Handler 层使用 `middleware.CanAccessAllData(c)` 辅助函数
- Service 层接收 `hasSharedViewer bool` 参数
- 权限检查统一模式: `!hasSharedViewer && resource.CreatedBy != userID`

**下一步建议**:
1. 集成测试 - 创建多角色用户场景测试
2. 手动验证 - 启动服务验证实际使用场景
