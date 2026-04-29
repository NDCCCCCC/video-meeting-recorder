# Phase 13: 重构华为配置，支持USB设备和流媒体录制模式 - Context

**Gathered:** 2026-04-29
**Status:** Ready for planning

## Phase Boundary

重构录制配置架构，将华为终端控制从"必填"改为"可选"，支持多种录制输入源（USB直录、流媒体），实现灵活的录制任务调度。

**核心变更：**
- 华为终端配置从必填改为可选（作为调度触发器）
- 支持三种录制输入：USB采集卡、流媒体(RTMP/RTSP)、华为终端自动控制
- 前端页面改名为"输入配置"
- 调度器支持所有类型配置的定时任务

## Implementation Decisions

### 配置数据结构

**D-01: 单一配置模型**
- 保持现有配置表结构，华为终端字段改为可选
- 每个配置包含：华为控制信息（可选）+ 录制源（USB 或流媒体二选一）
- 一个配置同时包含调度器信息和录制源信息

**D-02: 配置类型互斥**
- 添加 `config_type` 字段：`huawei_auto` | `usb` | `stream`
- 创建配置时选择类型，类型决定显示哪些字段
- 每个配置只能有一个录制源（USB 或流媒体，不能同时）

**D-03: 华为开关控制**
- 添加 `huawei_enabled` 布尔字段
- 开关关闭：隐藏华为终端字段
- 开关打开：华为终端字段必填（server, port, username, password, terminal_number）

### 验证规则

**D-04: 录制源必填验证**
- 至少需要填写一个录制源：
  - USB：`usb_camera_name` + `usb_camera_device`
  - 流媒体：`stream_url`（可选 `stream_protocol`, `stream_username`, `stream_password`）
  - 华为（当 `huawei_enabled=true`）：完整的华为终端字段

**D-05: 测试连接功能**
- USB：调用设备列表扫描 API，验证设备可用
- 流媒体：调用测试连通性 API，验证 URL 可访问
- 华为：调用华为终端 API，验证连接成功

### 调度触发机制

**D-06: 统一调度器**
- 所有类型配置都可以创建定时任务
- 调度逻辑：
  - 华为自动控制：到时间 → 发送 API 控制终端进入会议 → 开始录制
  - USB/流媒体：到时间 → 直接开始录制
- 移除 `VideoRecordingTask.IsValid()` 中 `HuaweiConfigID` 必填的检查

### 前端重构

**D-07: 全面重命名**
- 页面路由：`/system/huawei-configs` → `/system/input-configs`
- API 路由：`/api/huawei-configs` → `/api/input-configs`
- 文件重命名：
  - `huawei-config.ts` → `input-config.ts`
  - `huawei-configs/` 目录 → `input-configs/`
- 类型重命名：`HuaweiConfig` → `InputConfig`
- 组件重命名：所有 `HuaweiConfig*` 组件 → `InputConfig*`
- 菜单文案："华为配置" → "输入配置"

**D-08: 配置表单重构**
- 添加配置类型选择器（华为自动控制 / USB / 流媒体）
- 添加华为控制开关（仅在选择相关类型时显示）
- 根据类型动态显示/隐藏字段
- USB 设备选择器（调用扫描 API）
- 流媒体协议选择器（RTMP / RTSP / SRT / HLS）

### 数据库变更

**D-09: 新建 input_configs 表**
- 创建新表 `input_configs`，保持与 `huawei_configs` 相同的字段结构
- 添加新字段：
  - `config_type` VARCHAR(20): 'huawei_auto' | 'usb' | 'stream'
  - `huawei_enabled` BOOLEAN DEFAULT false
- 旧表 `huawei_configs` 保留，现有数据不受影响

**D-10: 数据迁移（可选）**
- 提供迁移功能：将 `huawei_configs` 中的数据转换为 `input_configs` 格式
- 旧数据类型设置为 `huawei_auto`，`huawei_enabled` 设置为 true
- 迁移脚本：`/admin/migrate-input-configs` API

**D-11: 关联表更新**
- `task_huawei_configs` 表保留（支持多配置关联）
- `config_type` 字段值：'usb' | 'stream' | 'huawei_auto'

### API 变更

**D-12: API 路由重命名**
- `GET/POST /api/input-configs` - 配置列表和创建
- `GET/PUT/DELETE /api/input-configs/:id` - 配置详情、更新、删除
- `POST /api/input-configs/:id/test` - 测试连接
- `GET /api/input-configs/usb-devices` - 扫描 USB 设备
- 保留旧路由 `/api/huawei-configs` 作为兼容性重定向（可选）

**D-13: 录制任务 API**
- `VideoRecordingTask` 的 `huawei_config_id` 字段改为可选
- 支持通过 `task_huawei_configs` 关联多个输入配置
- 任务创建时验证：至少关联一个有效配置

## Claude's Discretion

- 配置列表页面的筛选和排序设计
- 配置详情页面的布局和信息展示优先级
- 错误提示的具体文案和样式
- 测试连接的超时时间和重试策略
- 数据迁移的详细交互流程（是否需要确认、进度显示等）

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Database Models
- `internal/models/huawei_config.go` — 现有华为配置模型结构
- `internal/models/task_huawei_config.go` — 任务与配置的关联表
- `internal/models/video_recording_task.go` — 录制任务模型

### Services
- `internal/services/huawei_config_service.go` — 华为配置服务实现
- `internal/services/config_service.go` — 通用配置服务

### API Handlers
- `internal/handlers/huawei_config_handler.go` — 华为配置 API 处理器

### Frontend
- `frontend/src/pages/system/huawei-configs/index.tsx` — 华为配置页面
- `frontend/src/api/huawei-config.ts` — 华为配置 API 客户端
- `frontend/src/types/huawei-config.ts` — TypeScript 类型定义

### Recording Infrastructure
- `internal/services/video_recording/huawei_conference_connector.go` — 华为会议连接器
- Phase 1-4 相关文档 — 视频录制、分割、转换的基础架构

## Existing Code Insights

### Reusable Assets
- `TaskHuaweiConfig` 关联表：已支持多配置关联，`ConfigType` 字段可区分 USB/Stream
- USB 设备扫描功能：已在系统中实现，可直接复用
- 流媒体配置字段：已存在于 `HuaweiConfig` 模型中（`stream_protocol`, `stream_url` 等）

### Established Patterns
- 配置服务的 CRUD 模式：`List`, `Create`, `Update`, `Delete`, `Test`
- 设备绑定状态管理：`unbound` | `binding` | `bound` | `error`
- 任务与配置的多对多关联：通过 `TaskHuaweiConfig` 中间表

### Integration Points
- 路由：`internal/routes/` 需要更新 `/api/input-configs` 路由
- 导航菜单：`frontend/src/` 需要更新菜单文案和路由
- 调度器：`internal/services/scheduler/` 需要适配新的配置类型
- 录制服务：需要根据配置类型选择不同的录制启动逻辑

## Specific Ideas

- 配置类型图标：华为（终端图标）、USB（USB图标）、流媒体（网络图标）
- 华为开关的 UI：Toggle 组件，关闭时显示"手动模式"，打开时显示"自动控制"
- USB 设备扫描：显示可用设备列表，支持设备名称搜索
- 流媒体测试：显示连接状态码和响应时间
- 配置克隆功能：基于现有配置快速创建新配置

## Deferred Ideas

- 配置模板功能：预定义常用配置模板
- 配置导入/导出：支持批量导入配置
- 配置分组：支持将配置按地点/部门分组管理
- 录制源预览：配置时实时预览 USB 或流媒体画面
- 高级调度规则：支持复杂的调度条件（工作日、节假日等）

---

*Phase: 13-usb*
*Context gathered: 2026-04-29*
