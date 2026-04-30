# Phase 14: 文件管理页面添加批量下载和批量转录功能 - Context

**Gathered:** 2026-04-30
**Status:** Ready for planning

## Phase Boundary

为现有文件管理页面添加批量操作功能：
- **批量下载**：选中多个文件，打包为 ZIP 下载
- **批量转录**：选中多个视频文件，批量提交转录任务

**核心变更：**
- 文件列表页添加行选择功能（复选框）
- 批量下载：后端 ZIP 打包，流式响应
- 批量转录：任务组模式，顺序创建任务

## Implementation Decisions

### 批量下载 UI/UX

**D-01: 下载交互方式**
- 使用 ZIP 打包下载，用户点击一次下载整个 ZIP 包
- ZIP 内按文件类型分组到不同文件夹（video/, ppt/, other/）

**D-02: 批量选择布局**
- 使用 table rowSelection，复选框在每行开头
- 支持全选/全不选

**D-03: 进度显示方式**
- 使用 Toast 通知显示打包和下载进度
- 3秒后自动消失，用户可以继续操作

**D-04: 数量限制策略**
- 无硬性限制
- 如果文件很大（>1GB）或很多（>100）时显示警告提示

### 批量下载实现

**D-05: ZIP 打包位置**
- 在 Go 后端使用 `archive/zip` 标准库创建 ZIP 文件

**D-06: 响应方式**
- 流式响应：边打包边写入 HTTP 响应流
- 无需临时文件，用户立即开始下载

**D-07: 文件组织方式**
- ZIP 内按文件类型分组到不同文件夹：
  - `video/` - 视频文件（.mp4, .mkv 等）
  - `ppt/` - PPT 文件（.pptx）
  - `other/` - 其他文件

### 批量转录 UI/UX

**D-08: 批量转录入口**
- 在文件列表页，选中文件后显示「批量转录」按钮
- 点击后弹出转录配置对话框

**D-09: 配置交互**
- 显示模态对话框，包含转录配置（语言、PPT 提取等）和确认按钮

**D-10: 创建后反馈**
- 显示「已创建 N 个转录任务」的 Toast 通知
- 用户可到转录任务页查看进度

### 批量转录实现

**D-11: 任务组织方式**
- 使用任务组模式（TranscriptionJobGroup）
- 新增 `transcription_job_groups` 表，包含：
  - id, user_id, status, total_count, completed_count
  - created_at, updated_at
- 每个子任务关联 `job_group_id`

**D-12: 任务创建方式**
- 顺序创建任务，每个任务完成后才开始下一个
- 避免资源耗尽

**D-13: 错误处理策略**
- 如果某个文件转录失败，继续处理剩余文件
- 最后汇总成功和失败数量

## Claude's Discretion

- ZIP 打包的压缩级别设置
- 批量下载的内存使用优化
- 批量转录的任务队列优先级
- 任务组的超时和清理策略
- 错误提示的具体文案

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Existing File Management
- `frontend/src/pages/files/index.tsx` — 现有文件管理页面
- `frontend/src/api/video-file.ts` — 文件 API 客户端（包含批量删除参考）
- `frontend/src/types/video-file.ts` — VideoFile 类型定义

### Backend Models & Services
- `internal/models/video_file.go` — 视频文件模型
- `internal/models/transcription_task.go` — 转录任务模型
- `internal/services/transcription_service.go` — 转录服务实现

### API Handlers
- `internal/handlers/video_file_handler.go` — 文件 API 处理器
- `internal/handlers/transcription_handler.go` — 转录 API 处理器

### Phase References
- `Phase 2 CONTEXT.md` — 本地转录实现细节
- `Phase 4 CONTEXT.md` — 通义听悟云端转录实现
- `Phase 13 CONTEXT.md` — 批量操作参考模式（批量删除）

## Existing Code Insights

### Reusable Assets
- Ant Design Table `rowSelection` prop — 支持多选行功能
- `batchDeleteFiles` API — 批量删除实现参考
- 转录任务系统 — 已支持单文件转录，可直接复用

### Established Patterns
- 批量操作 API 模式：`POST /api/v1/files/batch` with `{ids: number[]}`
- 任务状态管理：pending → running → completed/failed
- 文件分组逻辑：按 `file_type` 或扩展名分组

### Integration Points
- 文件列表页：添加批量操作按钮和行选择
- 后端路由：`/api/v1/files/batch/download` (POST), `/api/v1/transcriptions/batch` (POST)
- 转录任务页：显示 JobGroup 列表和子任务详情
- 数据库：新增 `transcription_job_groups` 表

## Specific Ideas

- 批量下载按钮图标：使用 DownloadOutlined 图标
- 批量转录按钮图标：使用 ThunderboltOutlined 图标
- ZIP 文件名：`files_batch_YYYYMMDD_HHMMSS.zip`
- 任务组状态徽章：颜色区分（灰色=待执行，蓝色=进行中，绿色=完成，红色=失败）
- 进度 Toast：显示当前处理文件和进度百分比

## Deferred Ideas

- 批量文件重命名
- 批量文件移动/复制
- 批量标签管理
- 调度批量转录任务（定时执行）
- 批量操作历史记录

---

*Phase: 14-batch-operations*
*Context gathered: 2026-04-30*
