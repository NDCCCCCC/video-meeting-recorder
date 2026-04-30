---
phase: 14-batch-operations
plan: 01
type: execute
wave: 1
status: pending-human-verify
completed_at: "2026-04-30T14:30:00+08:00"
---

# Plan 14-01 Summary: Backend Batch Download Implementation

## Objective
实现批量下载功能：用户选择多个文件后，后端将文件打包为ZIP并流式响应，ZIP内按文件类型分组到不同文件夹。

## What Was Built

### Backend Service Layer (Task 1)
**File:** `internal/services/video_file_service.go`

- **BatchDownloadFilesRequest** - 批量下载请求结构
- **BatchDownloadFilesResponse** - 批量下载响应结构（包含 io.ReadCloser 流）
- **BatchDownloadFiles()** - 核心批量下载方法
  - 查询文件并验证所有权（isAdmin 或 file.CreatedBy == userID）
  - 检查物理文件是否存在
  - 使用 io.Pipe 创建流式 ZIP
  - 使用 goroutine 后台写入 ZIP
- **getFileFolder()** - 根据文件扩展名返回 ZIP 文件夹（video/, ppt/, other/）
- **addFileToZip()** - 将文件添加到 ZIP，使用 zip.Deflate 压缩

### Backend Handler Layer (Task 2)
**Files:** `internal/handlers/video_file_handler.go`, `cmd/server/app.go`

- **BatchDownloadFiles()** - HTTP 处理器方法
  - 解析请求中的文件 ID 列表
  - 获取用户信息（userID, isAdmin）
  - 调用服务层获取 ZIP 流
  - 设置正确的 HTTP 响应头
    - Content-Disposition: attachment; filename="files_batch_YYYYMMDD_HHMMSS.zip"
    - Content-Type: application/zip
    - Accept-Ranges: none
  - 使用 c.DataFromReader() 流式响应
- **路由注册:** POST /api/v1/files/batch/download

### Frontend API Client (Task 3)
**Files:** `frontend/src/api/video-file.ts`, `frontend/src/types/video-file.ts`

- **BatchDownloadFilesRequest** - 批量下载请求类型定义
- **BatchDownloadFilesResult** - 批量下载结果类型定义
- **batchDownloadFiles()** - 前端批量下载函数
  - 显示打包进度提示（message.loading）
  - 发送 POST 请求到 /api/v1/files/batch/download
  - 接收 ZIP 文件流（response.blob()）
  - 创建临时下载链接并触发浏览器下载
  - 使用 dayjs 生成时间戳文件名
  - 清理 blob URL
  - 显示成功/失败消息

## Files Modified
- `internal/services/video_file_service.go` - 添加批量下载服务方法
- `internal/handlers/video_file_handler.go` - 添加 BatchDownloadFiles handler
- `cmd/server/app.go` - 注册 POST /api/v1/files/batch/download 路由
- `frontend/src/types/video-file.ts` - 添加批量下载类型定义
- `frontend/src/api/video-file.ts` - 添加 batchDownloadFiles 函数

## Self-Check: PASSED

- [x] 后端服务层实现完成
- [x] 后端 Handler 实现完成
- [x] 路由注册完成
- [x] 前端 API 客户端实现完成
- [x] 后端代码编译通过
- [x] 前端代码构建通过

## Deviations
None

## Human Verification Required

此计划包含人工验证检查点。请验证以下功能：

1. 启动后端服务
2. 启动前端开发服务器
3. 访问文件管理页面
4. 选择多个文件（混合视频和PPT）
5. 点击「批量下载」按钮（需要在 Wave 3 的 UI 实现中添加）
6. 验证：
   - 浏览器下载了 ZIP 文件
   - ZIP 内文件按类型分组（video/, ppt/, other/）
   - 所有选中文件都包含在 ZIP 中
   - 下载进度有 Toast 提示

## Next Steps
Wave 2 将实现 Plan 14-02 (后端批量转录)，Wave 3 将实现 Plan 14-03 (前端批量操作UI，包括批量下载和批量转录按钮)。
