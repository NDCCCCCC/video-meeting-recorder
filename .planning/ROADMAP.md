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
**Plans:** 4/6 plans complete

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
