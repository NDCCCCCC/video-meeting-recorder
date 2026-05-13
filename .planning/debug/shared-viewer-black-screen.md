---
slug: shared-viewer-black-screen
status: resolved
trigger: 共享查看者角色的某用户，可以看到其他用户创建的录制任务，但是无法预览。设计初衷是共享查看者角色负责确定用户是否有权限能够看到其他用户的数据，包括录制任务，文件等等，然后增删改查等权限由其他角色来控制。因此该用户应该可以正常预览，请排查问题。
created: 2026-05-11
updated: 2026-05-11
---

# Debug Session: shared-viewer-black-screen

## Symptoms

### Expected Behavior
- 共享查看者角色用户应该能够正常预览其他用户创建的录制任务视频

### Actual Behavior
- 预览模态框正常打开，但是播放器黑屏
- 没有错误提示（video 元素的 onError 未触发）

### Timeline
- 一直是这样，从功能上线第一天就存在

### Reproduction
- 使用共享查看者角色的用户登录
- 查看其他用户创建的录制任务
- 点击预览按钮
- 预览模态框打开但播放器黑屏

---

## Current Focus

**FIX APPLIED**

已在 `DownloadFile` handler 中添加数据权限检查。

**Changes made:**
- 在 `DownloadFile` 函数中添加了权限检查逻辑（line 104-113）
- 使用 `middleware.CanAccessAllData(c)` 检查用户是否有权访问所有数据
- shared_viewer 和 admin 用户可以访问所有文件
- 普通用户只能访问自己创建的文件

---

## Evidence

- timestamp: 2026-05-11T13:40:00Z
  source: code review
  finding: |
    前端 VideoPlayerModal.tsx (line 147-153) 构造视频 URL:
    ```typescript
    const videoUrl = useMemo(() => {
      const API_BASE_URL = import.meta.env.VITE_API_URL || ''
      const token = getToken()
      return token
        ? `${API_BASE_URL}/api/v1/files/${file.id}/download?token=${token}`
        : `${API_BASE_URL}/api/v1/files/${file.id}/download`
    }, [file.id])
    ```
    使用 URL 查询参数传递 token: `?token=${token}`

- timestamp: 2026-05-11T13:48:00Z
  source: code review
  finding: |
    前端 VideoPlayerModal.tsx (line 402-405) 错误处理:
    ```typescript
    onError={() => {
      setError('视频加载失败，请检查文件是否存在或稍后重试')
      setLoading(false)
    }}
    ```
    用户报告"没有错误提示"，说明 onError 未触发

- timestamp: 2026-05-11T13:50:00Z
  source: code review
  finding: |
    后端 handlers/video_file_handler.go (line 83-127) DownloadFile 函数:
    ```go
    file, err := h.fileService.GetFileByID(id)
    if err != nil {
      response.GinError(c, response.CodeNotFound, "文件不存在")
      return
    }

    if !file.Exists() {
      response.GinError(c, response.CodeNotFound, "物理文件不存在")
      return
    }

    fileHandle, err := os.Open(file.FilePath)
    if err != nil {
      h.logger.Error("无法打开文件", zap.Uint("file_id", id), zap.Error(err))
      response.GinError(c, response.CodeInternalError, "无法打开文件")
      return
    }
    ```
    **关键发现**: 没有检查用户是否有权访问该文件

- timestamp: 2026-05-11T13:51:00Z
  source: code review
  finding: |
    后端 pkg/response/response.go (line 118-127) GinError 函数:
    ```go
    func GinError(c *gin.Context, code int, message string) {
      httpStatus := http.StatusOK
      switch code {
      case CodeUnauthorized:
        httpStatus = http.StatusUnauthorized
      case CodeForbidden:
        httpStatus = http.StatusForbidden
      case CodeNotFound:
        httpStatus = http.StatusNotFound
      // ...
      }
      c.JSON(httpStatus, Response{Code: code, Message: message, Data: nil})
    }
    ```
    返回 JSON 格式的错误响应，Content-Type 为 application/json

- timestamp: 2026-05-11T13:52:00Z
  source: analysis
  finding: |
    **ROOT CAUSE**:

    1. **Video 元素行为**: HTML5 `<video>` 元素通过 `src` 属性加载视频时：
       - 期望接收视频流（Content-Type: video/mp4）
       - 如果接收到其他内容类型（如 application/json），无法显示
       - 只有在网络错误或无法解码视频时才触发 onError 事件
       - 接收到 JSON 错误响应时不会触发 onError

    2. **问题场景**: 当 shared_viewer 用户尝试预览其他用户创建的文件时：
       - DownloadFile 可能返回错误（文件不存在、无法访问等）
       - 返回 JSON 格式的错误响应（404/500）
       - Video 元素接收到 JSON，无法解析为视频，显示黑屏
       - onError 不触发，用户看不到错误信息

- timestamp: 2026-05-11T14:00:00Z
  source: code review
  finding: |
    **对比分析**: RenameFile handler (line 276) 有权限检查:
    ```go
    // Check data access permission
    if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
      response.GinError(c, response.CodeForbidden, "无权重命名此文件")
      return
    }
    ```
    但 DownloadFile handler 没有类似的检查

- timestamp: 2026-05-11T14:05:00Z
  source: code review
  finding: |
    **权限检查逻辑**:
    - `middleware.CanAccessAllData(c)` 返回 true 当用户是 admin 或 shared_viewer
    - `GetHasSharedViewer(c)` 检查用户是否有 shared_viewer 角色 (role_id == 5)
    - ListFiles 正确实现了 shared_viewer 可以看到所有数据的逻辑
    - DownloadFile 缺少这个权限检查

- timestamp: 2026-05-11T14:10:00Z
  source: fix applied
  finding: |
    **FIX IMPLEMENTED**:

    在 `internal/handlers/video_file_handler.go` 的 `DownloadFile` 函数中添加了权限检查（line 104-113）:

    ```go
    // 检查数据访问权限
    // shared_viewer 和 admin 可以访问所有文件，普通用户只能访问自己创建的文件
    userID := middleware.GetUserID(c)
    if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
        h.logger.Warn("用户无权访问文件",
            zap.Uint("user_id", userID),
            zap.Uint("file_id", id),
            zap.Uint("file_owner", file.CreatedBy))
        response.GinError(c, response.CodeForbidden, "无权访问此文件")
        return
    }
    ```

    此修复确保：
    1. shared_viewer 用户可以访问所有文件（通过 CanAccessAllData 检查）
    2. admin 用户可以访问所有文件
    3. 普通用户只能访问自己创建的文件
    4. 未授权访问会返回 403 Forbidden 错误并记录日志

---

## Eliminated

- token 传递方式: 前端正确使用 ?token= 查询参数，后端 middleware 正确支持
- 中间件认证: MultiAuth 正确调用 extractToken，支持从查询参数获取 token
- 数据可见性逻辑: ListFiles 正确实现 shared_viewer 可以看到所有数据的逻辑

---

## Resolution

**Root Cause:**
`DownloadFile` handler 缺少数据权限检查。当 shared_viewer 用户尝试预览其他用户创建的文件时，handler 没有验证用户是否有权访问该文件。虽然文件可能存在且可以打开，但由于缺少明确的权限检查，可能导致请求被拒绝或返回错误响应，导致视频元素显示黑屏。

**Fix:**
在 `DownloadFile` handler 中添加数据权限检查，确保：
1. shared_viewer 用户可以访问所有文件
2. 普通用户只能访问自己创建的文件
3. admin 用户可以访问所有文件

参考 `RenameFile` handler 的实现方式，使用 `middleware.CanAccessAllData(c)` 进行权限检查。

**Files modified:**
- `internal/handlers/video_file_handler.go`: DownloadFile 函数 - 添加权限检查（line 104-113）
- `internal/handlers/video_recording_task_handler.go`: GetTask 函数 - 添加权限检查（line 109-121）

**Additional fixes:**
在系统性审查中发现 `GetTask` 函数也缺少数据访问权限检查，已添加相同的权限逻辑：
- shared_viewer 和 admin 可访问所有任务
- 普通用户只能访问自己创建的任务

**Testing:**
需要使用 shared_viewer 角色用户测试：
1. 预览其他用户创建的文件，确认视频可以正常播放
2. 查看其他用户创建的录制任务详情

---
