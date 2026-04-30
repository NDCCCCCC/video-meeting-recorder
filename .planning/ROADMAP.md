### Phase 13: 重构华为配置，支持USB设备和流媒体录制模式

**Goal:** 重构录制配置架构，将华为终端控制从"必填"改为"可选"，支持USB直录和流媒体(RTMP/RTSP)录制模式；修改前端页面名称从"华为配置"改为"输入配置"。

**Requirements:**
- D-01 to D-03: 单一配置模型，配置类型互斥，华为开关控制
- D-04 to D-05: 录制源必填验证，测试连接功能
- D-06: 统一调度器（所有类型配置都可创建定时任务）
- D-07 to D-08: 全面重命名，配置表单重构
- D-09 to D-11: 数据库变更（新建input_configs表，数据迁移可选，关联表更新）
- D-12 to D-13: API变更（路由重命名，录制任务API）

**Depends on:** Phase 12
**Plans:** 6/6 plans complete

Plans:
- [x] 13-00-PLAN.md — Wave 0: Test infrastructure stubs (4 test files, 28 test stubs) — Wave 0
- [x] 13-01-PLAN.md — InputConfig model with config_type and huawei_enabled fields — Wave 1
- [x] 13-02-PLAN.md — Database migration, VideoRecordingTask refactor, and data migration API — Wave 1
- [x] 13-03-PLAN.md — InputConfigService with validation and connection testing — Wave 2
- [x] 13-04-PLAN.md — InputConfigHandler API endpoints and route registration — Wave 3
- [x] 13-05-PLAN.md — Frontend refactoring (types, API client, management page, routing/menu) — Wave 4

**Wave Structure:**
- Wave 0 (sequential): 13-00 (test stubs for all InputConfig functionality)
- Wave 1 (parallel): 13-01 (InputConfig model), 13-02 (database migration + migration API)
- Wave 2 (sequential): 13-03 (InputConfigService with validation and testing)
- Wave 3 (sequential): 13-04 (InputConfigHandler + routes, depends on 13-03)
- Wave 4 (sequential): 13-05 (frontend refactoring with 3 checkpoints, depends on 13-04)

**Key Decisions:**
- D-01: 单一配置模型 - 保持现有配置表结构，华为终端字段改为可选
- D-02: 配置类型互斥 - 添加 config_type 字段：huawei_auto | usb | stream
- D-03: 华为开关控制 - 添加 huawei_enabled 布尔字段

### Phase 14: 文件管理页面添加批量下载和批量转录功能

**Goal:** 为现有文件管理页面添加批量操作功能：批量下载（打包为ZIP）和批量转录（任务组模式）。

**Requirements:**
- D-01 to D-07: 批量下载UI/UX（ZIP打包、行选择、进度显示、数量限制、后端ZIP打包、流式响应、文件分组）
- D-08 to D-10: 批量转录UI/UX（入口按钮、配置对话框、创建后反馈）
- D-11 to D-13: 批量转录实现（任务组模型、顺序创建、错误处理）

**Depends on:** Phase 13
**Plans:** 4 plans

Plans:
- [ ] 14-00-PLAN.md — Wave 0: Test infrastructure stubs (5 test files, 32 test stubs) — Wave 0
- [ ] 14-01-PLAN.md — Backend batch download (service layer + handler + API client) — Wave 1
- [ ] 14-02-PLAN.md — Backend batch transcription (job group model + migration + service + handler + API) — Wave 2
- [ ] 14-03-PLAN.md — Frontend batch operations UI (batch download button + batch transcription button + config modal) — Wave 3

**Wave Structure:**
- Wave 0 (sequential): 14-00 (test stubs for all batch operations)
- Wave 1 (sequential): 14-01 (batch download, depends on 14-00)
- Wave 2 (sequential): 14-02 (batch transcription, depends on 14-00)
- Wave 3 (sequential): 14-03 (frontend UI, depends on 14-01 and 14-02)

**Key Decisions:**
- D-01: 下载交互方式 - 使用 ZIP 打包下载，用户点击一次下载整个 ZIP 包
- D-02: 批量选择布局 - 使用 table rowSelection，复选框在每行开头
- D-03: 进度显示方式 - 使用 Toast 通知显示打包和下载进度
- D-04: 数量限制策略 - 无硬性限制，大文件或大量文件时显示警告
- D-05: ZIP 打包位置 - 在 Go 后端使用 archive/zip 标准库
- D-06: 响应方式 - 流式响应，边打包边写入 HTTP 响应流
- D-07: 文件组织方式 - ZIP 内按文件类型分组（video/, ppt/, other/）
- D-08: 批量转录入口 - 选中文件后显示批量转录按钮，点击弹出配置对话框
- D-09: 配置交互 - 模态对话框包含转录配置（语言、PPT 提取等）
- D-10: 创建后反馈 - 显示「已创建 N 个转录任务」的 Toast 通知
- D-11: 任务组织方式 - 使用任务组模式（TranscriptionJobGroup）
- D-12: 任务创建方式 - 顺序创建任务，每个任务完成后才开始下一个
- D-13: 错误处理策略 - 某个文件转录失败，继续处理剩余文件
