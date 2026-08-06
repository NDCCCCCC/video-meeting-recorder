# Phase 25: scheduler 多信号驱动 + service 封装 + E2E + CI — Research

**Researched:** 2026-08-06
**Domain:** Go 后端 — scheduler `select`-multichannel / GORM 字段写入 / audit log snapshot 序列化 / `cfg.SmartEnd.*` 运行时开关接线 / E2E 测试金字塔 / CI gate 增强
**Confidence:** HIGH（核心代码事实（scheduler `monitorTask`、`taskUpdateChans`、`RecordingProcess.taskEndedCh`、`ActivityWatcher.EndedCh/Snapshot/IsActive`、`VideoRecordingTask` 5 GORM 字段、`AuditLogService.RecordChange`、`cfg.SmartEnd.14 字段`）全部在 codebase 验证；E2E 测试栈与 CI 现状来自 `.github/workflows/ci.yml` 直读）

## User Constraints

无独立 `25-CONTEXT.md`；以下约束来自 `STATE.md:107-114`、`ROADMAP.md:97-108`、`REQUIREMENTS.md` 与用户给定 phase context。**全部锁定**。

### Locked Decisions（来自 STATE/ROADMAP/REQUIREMENTS，不可更动）

- **范围**：完成 `monitorTask` `select` 驱动 + 服务层封装 + 端到端闭环 + AUDIT-02/03/04 + CFG-03/04 + OBS-01..05；不再扩展 DETECT/WATCH 信号采集（已 Phase 23/24 落地）。`ROADMAP.md:97-108` [VERIFIED: codebase read]
- **信号消费契约**：scheduler `select` 等待 3 路 — `timer.C(EndTime)` + `<-taskEndedCh` + `<-updateChan(taskUpdateChans[task.ID])`；三者中 **`taskEndedCh` 永远优先**（channel close 后 `<-timer.C` 不再生效）。`REQUIREMENTS.md:30-37`、Phase 24 RESEARCH.md `Pitfall 2` [VERIFIED: PRD read]
- **延时上限**：单任务 `ExtensionCount` 上限 = `cfg.SmartEnd.MaxExtendCount`（默认 4），累计 = `MaxExtendCount × ExtendStepMin`（默认 30min）= **2h 总上限**；达上限后仍活跃 → `completeTask("max_extend_reached")` 且保留 `completed`（**不**标 `failed`）。`REQUIREMENTS.md:39-41` [VERIFIED: PRD read]
- **手动 `UpdateTaskEndTime` 不重置 ExtensionCount**（SCHED-04）：仅重置 timer，不绕开上限；watcher 不感知此事件。`REQUIREMENTS.md:35-37` [VERIFIED: PRD read]
- **提前结束策略**（EARLY-04）：多 input（huawei_auto + usb 同时录制）任务下，**任一 watcher 判定结束 → 整个任务结束**（保守策略）；当前 `coordinator.processes` 已按 `taskID_configType` 分键，scheduler 选一个 watcher 为主触发源或 `OR` 全部 `EndedCh` 即可。`REQUIREMENTS.md:43-44`、Phase 24 RESEARCH.md EARLY-04 [VERIFIED: PRD read]
- **service 封装契约**：`UpdateTaskExtension(task, deltaMin, reason)` + `MarkTaskEndedEarly(task, reason, byHuaWeiAPI bool)`，**统一**写入 GORM 5 字段 + audit log（含 snapshot），避免散落调用点。`REQUIREMENTS.md:51-52` [VERIFIED: PRD read]
- **运行开关**（CFG-03/04）：`smart_end.enabled=false` 时 scheduler 不读 `taskEndedCh`、watcher 不启；`smart_end.huawei_enabled=false` 时 watcher 的 `huaweiPoller` 跳过（Phase 24 已实现），scheduler 不变。`REQUIREMENTS.md:55-56` [VERIFIED: PRD read]
- **可观测日志**（OBS-01..04 + 05）：
  - `INFO smart_extend task=<id> count=<n> new_end=<ts> reason=<text>`（每次自动延时）
  - `INFO smart_early_end task=<id> reason=<text> snapshot=<json>`（每次提前结束）
  - `WARN max_extend_reached task=<id> force_end=true`
  - `ERROR activity_watcher_degraded reason=<text>`（Phase 24 已发，需保证 Phase 25 仍触发）
  - OBS-05：项目无 prometheus 集成，仅预留 counter 接入点（`vars.go` 暴露 `AtomicInt64` 或同款全局变量），不引入新依赖。`REQUIREMENTS.md:60-64` [VERIFIED: PRD read]
- **audience**：scheduler `monitorTask` 现行为保留（用于手动 `UpdateTaskEndTime` 与 scheduler 内部其他 case），不重写框架；选 `time.NewTimer` + `time.NewTicker` 模式。`REQUIREMENTS.md:90-92` [VERIFIED: PRD read]
- **Project Constraints (CLAUDE.md)**：
  - `.planning/` 被 gitignore — RESEARCH.md / PLAN.md / SUMMARY.md `git add -f` 提交
  - 4 commit 拆分（debug ↔ Phase 工作）— `commit-boundary-separation.md`
  - golangci-lint v2.12.2+ (action v7+) — 项目已 pinned
  - AutoMigrate 仅放已确认 model，不进 dormant `runCustomMigrations`（Phase 23 已加 5 字段，Phase 25 不再新增 column）

### Claude's Discretion（research 推荐 + planner 决策）

- `select` 内 case 顺序（建议 taskEndedCh 优先 → timer → updateChan，与 SCHED-03 锁定契约一致）
- `UpdateTaskExtension` / `MarkTaskEndedEarly` 实现位置（建议放 `internal/services/video_recording_task_service.go` 与既有服务同文件，注入 `audit.AuditLogService`）
- 多 input `EndedCh` 合并策略（建议在 monitorTask 内拉单一 `endedCh := mergeWatchers(processes[taskID_...] taskEndedCh)`；或简化为按 configType 选主 watcher + OR 收信号，后者改动最小）
- audit log snapshot 字段序列化（建议单一 `map[string]interface{}` JSON 序列化进 `AuditLog.NewData`，与现有 `RecordChange` 复用）
- Prometheus 接入点（建议 `internal/observability/smart_end_metrics.go` 暴露 3 个 atomic.Int64 计数器 + `Record*` 函数；仍仅日志，**不**导入 `github.com/prometheus/client_golang`）

### Deferred Ideas (OUT OF SCOPE)

- 改 `VideoRecordingTaskStatus` 枚举为 `smart_extended`
- 做 ML/ASR "提前结束预测"
- 重写 `monitorProcessWithKey` 断流重连（保留现有；watcher 已通过 OnReconnect 兼容 WATCH-05）
- 前端 UI 展示 `ExtensionCount`/`EndedEarlyReason`/`EndedByHuaWeAPI`（FUTURE-04，单独 phase）
- 接 `MSG_CONF_STATE_CHANGE` 推送（FUTURE-01）
- 跨 input "软结束"（FUTURE-03）

## Project Constraints (from CLAUDE.md)

- 后端 Go 主目录 `internal/`，分层 `auth/` / `models/` / `services/` / `scheduler/` / `recorder/` / `huawei/` / `config/` / `errors/` / `audit/`。`CLAUDE.md:10-17` [VERIFIED: codebase read]
- 自动技能 `spike-findings-record-v2`（Windows AD 认证蓝图）— 与本阶段技术域无直接约束，未发现仓库内 skill 文件可进一步加载。`CLAUDE.md:6-8` [VERIFIED: codebase read]
- 4 commit 拆分约定（debug 与 Phase 工作分提交），按 `commit-boundary-separation.md` [VERIFIED: project memory]
- 本机 transparent HTTPS MITM — local-repo `.git/config` http.sslVerify=false（仅本仓库）[VERIFIED: project memory]
- AutoMigrate 仅放已确认 model；不动 dormant `runCustomMigrations` [VERIFIED: project memory]
- `.planning/` gitignored — RESEARCH/PLAN/SUMMARY 需 `git add -f` [VERIFIED: project memory]
- golangci-lint v2.12.2+（go1.25 要求）+ action v7+ [VERIFIED: project memory]

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SCHED-01 | `monitorTask` 改 `select` on EndTime timer + `taskEndedCh` + `taskUpdateChans[task.ID]` | `video_scheduler.go:541-606` 已用 `select`，但仅含 `timer.C` + `ctx.Done()` + `updateChan`；缺 `taskEndedCh`。scheduler 须 `RecorderCoordinatorInterface` 新增 `GetWatcher(taskID) (*recorder.ActivityWatcher, bool)` 或读 `processes` map 直接取 `taskEndedCh` 字段 |
| SCHED-02 | EndTime 到点先问 `watcher.IsActive()`，活跃则 `EndTime += extend_step_min`，否则 `completeTask("endtime_no_activity")` | `ActivityWatcher.IsActive()` 已实现（`activity_watcher.go:156-163`），`ExtendStepMin()` 已暴露（`activity_watcher.go:195-197`）。Phase 25 service 调 `UpdateTaskExtension` 后改写 `task.EndTime` 并同步 `taskUpdateChans[task.ID]` 信号回 monitorTask |
| SCHED-03 | `taskEndedCh` 信号永远优先于 EndTime timer（channel close 后 EndTime.C 不再生效） | 在 `select` 内把 `<-taskEndedCh` case 放最先；Go `select` 多 case 随机选，但 channel close 立刻可读 → 实际总是先触发；但需补充 `default` 阻止 timer 提前触发的 nonce（非必须，select 本身就保证） |
| SCHED-04 | 用户手动 `UpdateTaskEndTime` 触发 `taskUpdateChans` 时 `ExtensionCount` 不重置 | `ExtensionCount` 字段读写均在 service 层（`UpdateTaskExtension`）；scheduler 在收到 `updateChan` 信号时**只**改 `endTime` 局部变量，不调 `UpdateTaskExtension`，即不重置计数 |
| EXTEND-01 | `ExtensionCount` 上限 = `cfg.SmartEnd.MaxExtendCount`（默认 4），累计 = `MaxExtendCount × ExtendStepMin` | service 方法需读 `cfg.SmartEnd.MaxExtendCount` 做 `>=` 比较，超阈值走 EXTEND-02 |
| EXTEND-02 | 上限达 4 后 EndTime 到点仍活跃 → `completeTask("max_extend_reached")`，任务保留 `completed`（**不**标 failed） | scheduler 需在 `case <-timer.C` 重新检查 `watcher.IsActive()` + `task.ExtensionCount >= cfg.SmartEnd.MaxExtendCount`。Status `completed` 走 `updateTaskStatus(ctx, taskID, VideoStatusConverting, "")`（复用现有 `completeTask`），不发 `Failed`；audit log Status=`warn` |
| EARLY-01 | H 信号触发 → `completeTask("smart_early_end")`，`task.EndedByHuaWeAPI=true`，`task.EndedEarlyReason="huawei_state_empty"` | `ActivityWatcher.Snapshot().EndedReason` 已携带 `"huawei_state_empty"`；Phase 25 scheduler 在 `case <-taskEndedCh` 触发后从 watcher 取 snapshot，调 `service.MarkTaskEndedEarly(task, snapshot.EndedReason, true)` |
| EARLY-02 | A+B 双 AND → `smart_early_end`，`EndedByHuaWeAPI=false`，`EndedEarlyReason="both_silence_and_stall"` | `watcher.EndedReason == "both_silence_and_stall"` 时 `byHuaWeiAPI=false` |
| EARLY-03 | 提前结束信号永远优先于 EndTime timer | 同 SCHED-03；`select` 把 `<-taskEndedCh` 放第一 case |
| EARLY-04 | 多 input 任一 watcher 触发整体结束（保守策略） | `coordinator.processes` 按 `taskID_configType` 分键，每键一个 `taskEndedCh`。scheduler `monitorTask` 启一个聚合 goroutine 用 `select` OR 所有 input 的 `taskEndedCh`，首个关即 return |
| AUDIT-02 | 每次"延时"事件 audit log 含 snapshot `silence_since` / `last_file_growth` / `file_size_bytes` / `file_growth_bps` + `extension_count` + `new_end_time` | `ActivitySnapshot` 字段差 `file_size_bytes` / `file_growth_bps` — 需在 `activity_watcher.go` `ActivitySnapshot` 加 2 字段并 `fileTicker` 维护；audit log `OldData`/`NewData`/`DiffData` JSON 序列化 |
| AUDIT-03 | 每次"提前结束"事件 audit log 含 `task_id` + `reason` + `ended_by_huawei_api` + `ended_early_reason` + final snapshot | 同 snapshot + GORM 字段写入 |
| AUDIT-04 | service 层封装 `UpdateTaskExtension(task, deltaMin, reason)` + `MarkTaskEndedEarly(task, reason, byHuaWeiAPI bool)` | `services/video_recording_task_service.go` 注入 `audit.AuditLogService`（新增字段 + 构造函数参数）；与既有 `UpdateTaskStatus` 并列 |
| CFG-03 | `smart_end.enabled=false` 时 scheduler 不读 `taskEndedCh`，watcher 不启 | `coordinator.go:199` 已守门 `cfg.SmartEnd.Enabled`（watcher 不启）；scheduler `monitorTask` 增 `if !cfg.SmartEnd.Enabled { /* old EndTime-only loop */ }` 分支 |
| CFG-04 | `smart_end.huawei_enabled=false` 时 watcher 的 huaweiPoller 不启，scheduler 不变 | `activity_watcher.go:381-384` 已守门 — Phase 25 不动 scheduler 即可 |
| OBS-01 | `INFO smart_extend task=<id> count=<n> new_end=<ts> reason=<text>` | `UpdateTaskExtension` 内 + scheduler select case 后各发 1 次（service 内发更佳，单一入口） |
| OBS-02 | `INFO smart_early_end task=<id> reason=<text> snapshot=<json>` | `MarkTaskEndedEarly` 内 + `ActivityWatcher.closeEnded` 已发 INFO（需调整含 snapshot JSON） |
| OBS-03 | `WARN max_extend_reached task=<id> force_end=true` | scheduler 在 `case <-timer.C` 触发上限分支时发，**或** `UpdateTaskExtension` 返回 `errors.Is(apperrors.ErrMaxExtendReached)` 让上头发 — 推荐前者 |
| OBS-04 | `ERROR activity_watcher_degraded reason=<text>` | Phase 24 `activity_watcher.go:274` + `:390` + `:419` 已发，需补 unit test 覆盖 OBS-04 字段 schema |
| OBS-05 | 可选 Prometheus counter 接入点；项目无 prometheus 集成则仅做日志 | 新增 `internal/observability/smart_end_metrics.go` 暴露 3 个 atomic.Int64（`smart_extend_total` / `smart_early_end_total` / `watcher_degraded_total`）+ `Record*` 函数；scheduler 调用；**不** import `github.com/prometheus/client_golang` |
</phase_requirements>

## Summary

Phase 25 把 Phase 23/24 已落地的三类产物（H 信号、A 信号、B 信号 + 5 GORM 字段 + 3 sentinel + SmartEndConfig）在调度层串成闭环。核心改动在 **scheduler `monitorTask` 单 loop 改 `select` 三路信号**（timer + `taskEndedCh` + `taskUpdateChans[task.ID]`），其中 `<-taskEndedCh` 必须最早 case 化以满足 SCHED-03；**service 层抽出 `UpdateTaskExtension` 与 `MarkTaskEndedEarly` 两个公开方法**作为延时 / 提前结束的唯一入口，把 GORM 字段写入 + audit log（含 watcher snapshot）统一收敛，避免散落到 scheduler / coordinator / recorder 多处（满足 AUDIT-04）。

最重要的规划纠偏有两点：(1) **`taskEndedCh` 在多 input 任务下需要 OR 合并** — 当前 `coordinator.processes` 按 `taskID_configType` 分键（`process.taskEndedCh` 是 per-(task,configType)），EARLY-04 要求"任一 watcher 触发整体结束"，scheduler `monitorTask` 必须 select 所有 `processes[taskID_*].taskEndedCh`；最稳的实现是 `RecorderCoordinatorInterface` 加 `WatcherChannels(taskID uint) []<-chan struct{}` getter（或直接暴露 `GetWatcher`，由 monitorTask 自行 select），scheduler 在 select 前一次性拉所有 ch 用聚合 `select` 监听。(2) **`smart_end.enabled=false` 必须既不读 `taskEndedCh` 也不调 `MarkTaskEndedEarly`** — 否则任务会被 watcher 提早关闭而 EndTime-timer 路径不感知；建议 `monitorTask` 保留旧 timer-only path 作 `else` 分支（cfg 守门）。

AUDIT-02/03 的 audit log snapshot 有 6 字段（`silence_since` / `last_file_growth` / `file_size_bytes` / `file_growth_bps` / `extension_count` / `new_end_time`），其中 `file_size_bytes` 与 `file_growth_bps` 当前 `ActivitySnapshot` 不暴露 — Phase 25 需在 `internal/recorder/activity_watcher.go` 扩 2 字段 + `fileTicker` 维护（`activity_watcher.go:316-368` 已计算 `growthBps` 局部值）；这是一个 **Phase 24 落地未完整** 的子项，Phase 25 PLAN-01 必须先补齐再谈写 audit log。

OBS-05 项目无 prometheus，**禁止引入新依赖**；按 PRD 建议仅暴露 atomic.Int64 计数 + `Record*` 函数即满足"预留接入点"。E2E 测试栈沿用 `internal/services/video_recording_task_service_test.go:18-29` `newTestDB(t)` 模式（SQLite :memory: + AutoMigrate），scheduler E2E 沿用 `internal/scheduler/video_scheduler_test.go:138-159` 的 `mockCoordinator` 模式扩 fake `taskEndedCh` 注入。CI 新增 gate 仅加一条：`cmd/...` 与 `internal/...` 的 `-race` 必过（已存在 `go test -race ./...`），无需新增 step。

**Primary recommendation:** Phase 25 用 4 个 plans 落地 — (01) ActivitySnapshot 扩字段 + service 层 `UpdateTaskExtension`/`MarkTaskEndedEarly` + 注入 `audit.AuditLogService`；(02) scheduler `monitorTask` `select` 改造 + 多 watcher 合并 + `cfg.SmartEnd.Enabled` 守门；(03) 5 类 OBS 日志 + `smart_end_metrics.go` atomic 计数器接入点；(04) Nyquist E2E 覆盖 7+ scenario（多次延时正常上限 / 强制 max_extend_reached / H 触发 / A+B 触发 / 手动延时不重置 / 多 input 一到全停 / CFG-03 disabled 降级）。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| scheduler `select` 多信号 | API / Backend (scheduler/video_scheduler.go) | recorder/coordinator (consumer of `processes[taskID_*].taskEndedCh`) | 改动限定在 `monitorTask` 单函数；其他调度逻辑不变 |
| `IsActive()` 决策点 | API / Backend (scheduler) → recorder/activity_watcher | service (UpdateTaskExtension 写入 ExtensionCount) | 每次 EndTime timer fire 调一次；watcher 单方法已实现 |
| 延时 GORM 字段写入 + audit log | API / Backend (services/video_recording_task_service) | audit (RecordChange)、recorder (Snapshot) | 单入口 `UpdateTaskExtension` / `MarkTaskEndedEarly`；不允许 scheduler 直接 UPDATE columns |
| 提前结束 GORM 字段 + status 转换 | API / Backend (services/video_recording_task_service) | scheduler (`completeTask` already transitions to `converting` via `updateTaskStatus`) | service 写 `EndedEarly*` 5 字段；scheduler 复用现有 `completeTask` 走 converting 状态机 |
| 多 input taskEndedCh 合并 | API / Backend (scheduler) → recorder/coordinator | recorder/activity_watcher | scheduler 一次性 `select` 多个 `<-chan struct{}` |
| `smart_end.enabled=false` 守门 | API / Backend (scheduler) → recorder/coordinator | config.SmartEnd | scheduler `monitorTask` 入口 `if cfg.SmartEnd.Enabled` 分支；coordinator `StartRecordingWithConfig` 已有守门 |
| `huawei_enabled=false` 守门 | API / Backend (recorder/activity_watcher) | config.SmartEnd | Phase 24 已实现；Phase 25 不变 |
| audit log snapshot 序列化 | API / Backend (audit/AuditLogService.RecordChange) | recorder (Snapshot) + models (VideoRecordingTask) | 复用 `AuditLogData.OldData/NewData/Diff`，JSON marshal `interface{}` 即可 |
| OBS 日志字段 | API / Backend (scheduler + service) | recorder (existing `activity_watcher_degraded`) | scheduler 主循环日志 + service 写 audit 前 INFO 1 次；watcher 已有 ERROR |
| Prometheus 接入点 | API / Backend (observability/smart_end_metrics.go) | scheduler + service + recorder 调用方 | 单文件暴露 atomic + Record*；不引入 prom 库 |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (`time` / `sync` / `context` / `errors` / `encoding/json` / `sync/atomic`) | Go 1.25.0 | `select` 三路 / Mutex / Timer / channel close / atomic.Int64 计数 | 已 in use — `video_scheduler.go:4-9`、`activity_watcher.go:10-15`、`audit_log_service.go:1-9` |
| `github.com/NDCCCCCC/video-meeting-recorder/internal/config` | (项目内) | `cfg.SmartEnd.*` 14 字段读取 + 守门 | Phase 23 已交付 — `internal/config/smart_end.go:20-69` |
| `github.com/NDCCCCCC/video-meeting-recorder/internal/audit` | (项目内) | `AuditLogService.RecordChange(ctx, opts)` | 现有 service，`models.AuditLog.Action=ActionUpdate / Module=ModuleTask` |
| `github.com/NDCCCCCC/video-meeting-recorder/internal/recorder` | (项目内) | `ActivityWatcher.EndedCh()/IsActive()/Snapshot()/ExtendStepMin()` | Phase 24 已交付，本阶段消费 |
| `github.com/NDCCCCCC/video-meeting-recorder/internal/models` | (项目内) | `VideoRecordingTask` 5 GORM 字段 + `VideoRecordingTaskStatus` 枚举 | Phase 23 AUDIT-01 已交付 — `internal/models/video_recording_task.go:39-44` |
| `github.com/NDCCCCCC/video-meeting-recorder/internal/errors` | (项目内) | `apperrors.ErrRecordingSmartExtend` / `ErrRecordingSmartEarlyEnd` | Phase 23 AUDIT-05 已交付 — `internal/errors/errors.go:101-105` |
| `go.uber.org/zap` | v1.27.0 | 结构化日志（5 类 OBS 字段） | 项目标准 logger — `go.mod:23` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` | stdlib | audit log NewData/OldData/Diff JSON marshal | `AuditLog.OldData/NewData/DiffData` 是 `gorm:type:text`，由 service marshal 字符串 |
| `database/sql/driver` (modernc.org/sqlite) | v1.45.0 | E2E 测试 in-memory SQLite | test only — `internal/services/video_recording_task_service_test.go:22` |
| `sync/atomic.Int64` | Go 1.19+ | OBS-05 Prometheus 计数器接入点 | 暴露 3 个全局变量供后续 prom 实现 import；本阶段不引入 `prometheus/client_golang` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| service 层注入 `*AuditLogService` 直接调 `RecordChange` | scheduler 内联 `auditService.RecordChange(...)` | service 收敛更好写 audit log（含 GORM 字段）；scheduler 只持有 watch + 调用 service；选 service 入口 |
| `RecorderCoordinatorInterface.WatcherChannels(taskID)` getter | scheduler 直接读 `coordinator` 私有 `processes` map | 接口化便于 mock 测试（unit-only fake），不改 recorder 内部结构 |
| `sync/atomic.Int64` 全局计数器 | `github.com/prometheus/client_golang/promauto` | 项目无 prometheus 集成；OBS-05 仅"接入点"；**禁止**引入新依赖 — 选 atomic |
| `<-chan struct{}` 关闭语义（close-once） | `chan struct{ Reason string }` 同步发原因 | Phase 24 已选 `chan struct{}` + `Snapshot().EndedReason` 字段；Phase 25 不改 watcher 接口 |

**Version verification:** 无新增 package（go.mod 不变）。[VERIFIED: go.mod lockfile unchanged]

## Architecture Patterns

### System Architecture Diagram

```
[executeTask cron / AddTask path]
   │
   ▼
[StartRecordingWithConfig per input config]
   │  cfg.SmartEnd.Enabled == true
   │  → rec.processes[taskID_configType].taskEndedCh = endedCh  (NEW: Phase 24 已加字段, Phase 25 消费)
   │  → rec.ActivityWatcher = NewActivityWatcher(...) ; Start
   │
   ▼
[monitorTask(ctx, task)]  (Phase 25 改造核心)
   │
   │  if !cfg.SmartEnd.Enabled  → 保留旧 timer-only loop（CFG-03 守门）
   │
   │  channels := []<-chan struct{}{}
   │  for each inputConfig ∈ task.TaskInputConfigs :
   │      if proc.taskEndedCh != nil : append(channels)
   │
   │  updateChan := taskUpdateChans[task.ID]
   │  endTime := task.EndTime
   │  loop {
   │    timer := time.NewTimer(time.Until(endTime))
   │    select {
   │    case <-mergeTaskEnded(channels):           // SCHED-03 优先
   │        snap := firstWatcher.Snapshot()
   │        MarkTaskEndedEarly(task, snap.EndedReason, snap.LastHuaWeiStateEmpty)
   │        completeTask("smart_early_end")    // 复用现有路径
   │        return
   │    case newEndTime := <-updateChan:            // SCHED-04: ExtensionCount 不动
   │        endTime = newEndTime                     // 仅 timer 重置
   │    case <-timer.C:                              // EndTime 到点
   │        if watcher.IsActive() :
   │            if task.ExtensionCount >= cfg.SmartEnd.MaxExtendCount :
   │                log.WARN("max_extend_reached", ...)
   │                completeTask("max_extend_reached")  // EXTEND-02, status=completed 不变
   │                return
   │            UpdateTaskExtension(task, ExtendStepMin, reason)  // 写 5 字段 + audit
   │            endTime = task.EndTime + ExtendStepMin
   │        else :
   │            completeTask("endtime_no_activity")
   │            return
   │    case <-ctx.Done():
   │        return
   │    }
   │  }
```

### Recommended Project Structure

```
internal/
├── scheduler/
│   ├── video_scheduler.go              # 修改 — monitorTask 改 select 多信号 + CFG-03 守门
│   ├── video_scheduler_test.go         # 扩展 — 7+ E2E subtest
│   └── task_ended_channel_helper.go    # 新增 — mergeWatchers(select 多 chan) 工具
├── recorder/
│   ├── activity_watcher.go              # 修改 — ActivitySnapshot 加 2 字段 (file_size_bytes / file_growth_bps)
│   ├── activity_watcher_test.go        # 扩展 — 新字段断言 + 多 input OR 合并测试
│   └── coordinator.go                  # 修改 — RecorderCoordinatorInterface 实现 GetWatcher(taskID)
├── services/
│   ├── video_recording_task_service.go # 修改 — 新增 UpdateTaskExtension + MarkTaskEndedEarly
│   └── video_recording_task_service_test.go # 扩展 — 5 字段读写 + audit snapshot JSON 断言
├── observability/                       # 新增目录
│   └── smart_end_metrics.go            # OBS-05: atomic.Int64 计数 + Record* 函数
└── ...

internal/services/audit/
└── (无改动 — 复用 RecordChange API)

.golangci.yml  (或 make / scripts/run_lint.sh)
    # 已固定 v2.12.2 + action v7；Phase 25 无新增 lint 规则
```

### Pattern 1: `select` 三路 case 顺序固定 taskEndedCh 优先

**What:** scheduler `monitorTask` 单循环 `select` 内 case 顺序固定为 `<-taskEndedCh`（先行）→ `<-updateChan` → `<-timer.C`，Go runtime 在多 case ready 时随机选，但 `taskEndedCh` close 后立即 always-ready 而 `timer.C` 仅在 time 到达时 ready —— 实际保证"提前结束永远先触发"。

**When to use:** 当 channel close 语义代表"终止信号"且必须强优先于定时器时。

**Why:** 用户在 PRD §11 "任务提前结束应不晚于 timer 误触发的窗口"要求；SCHED-03 + EARLY-03 双向契约用同一机制实现。

**Code sketch:**
```go
// Source: PRD §5.2 + Phase 24 RESEARCH.md "Pattern 3"
for {
    timer := time.NewTimer(time.Until(endTime))
    select {
    case <-taskEndedCh:
        timer.Stop()
        snap := w.Snapshot()
        if err := s.taskService.MarkTaskEndedEarly(ctx, task.ID, snap.EndedReason, snap.LastHuaWeiStateEmpty); err != nil {
            s.logger.Error(...)
        }
        s.completeTask(ctx, task.ID)
        return
    case newEndTime, ok := <-updateChan:
        timer.Stop()
        if !ok { return }
        endTime = newEndTime // SCHED-04: 仅 timer 重置，不动 ExtensionCount
    case <-timer.C:
        if !w.IsActive() {
            s.completeTask(ctx, task.ID)
            return
        }
        // 上限判定在 service UpdateTaskExtension 内
        // EXTEND-01/02 + OBS-03
        ...
    case <-ctx.Done():
        timer.Stop()
        return
    }
}
```

### Pattern 2: Snapshot 字段透传到 audit log

**What:** ActivityWatcher 暴露 `Snapshot() ActivitySnapshot`（Phase 24 已实现）—— Phase 25 仅扩展 2 字段（`file_size_bytes` / `file_growth_bps`），service `UpdateTaskExtension` / `MarkTaskEndedEarly` 接 `snapshot ActivitySnapshot` 入参，构造 `audit.AuditLogData{OldData:..., NewData: snapshot + GORM 字段}` JSON marshal 进 `audit_logs.new_data`。

**When to use:** 后端需要把 watcher 状态 + DB 字段同时落 audit log，但 audit handler 不直读 recorder（保持解耦）。

**Why:** AUDIT-04 要求"统一接口"；避免 scheduler 把 snapshot 序列化两次或漏字段。

**Code sketch:**
```go
// Source: Phase 24 activity_watcher.go:65-77 (现有) + Phase 25 扩展
type ActivitySnapshot struct {
    // ...Phase 24 已有
    FileSizeBytes    int64 `json:"file_size_bytes"`     // NEW Phase 25
    FileGrowthBps    int64 `json:"file_growth_bps"`     // NEW Phase 25
}

// service 内
type SmartEndSnapshot struct {
    SilenceSince     *time.Time `json:"silence_since,omitempty"`
    LastFileGrowthAt *time.Time `json:"last_file_growth,omitempty"`
    FileSizeBytes    int64      `json:"file_size_bytes"`
    FileGrowthBps    int64      `json:"file_growth_bps"`
    ExtensionCount   int        `json:"extension_count"`
    NewEndTime       time.Time  `json:"new_end_time"`
}

func (s *VideoRecordingTaskService) UpdateTaskExtension(
    ctx context.Context, taskID uint, deltaMin int, reason string, snap ActivitySnapshot,
) error {
    // 1. 读当前 task
    var task models.VideoRecordingTask
    if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil { ... }

    // 2. EXTEND-02 上限守门
    if task.ExtensionCount >= s.cfg.SmartEnd.MaxExtendCount {
        return fmt.Errorf("extension limit reached: %w", apperrors.ErrRecordingSmartExtend)
    }

    // 3. UPDATE GORM
    newEnd := task.EndTime.Add(time.Duration(deltaMin) * time.Minute)
    updates := map[string]interface{}{
        "end_time":             newEnd,
        "extension_count":      task.ExtensionCount + 1,
        "last_extension_reason": reason,
    }
    if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil { ... }

    // 4. OBS-01 + audit
    s.logger.Info("smart_extend",
        zap.Uint("task_id", taskID),
        zap.Int("count", task.ExtensionCount+1),
        zap.Time("new_end", newEnd),
        zap.String("reason", reason),
    )
    observability.RecordSmartExtend() // OBS-05 atomic +1

    // 5. audit log (AUDIT-02 snapshot 6 字段)
    if s.auditSvc != nil {
        _ = s.auditSvc.RecordChange(ctx, audit.RecordChangeOpts{
            Action:     models.ActionUpdate,
            Module:     models.ModuleTask,
            Resource:   "video_recording_task",
            ResourceID: &taskID,
            OldData:    map[string]interface{}{"end_time": task.EndTime, "extension_count": task.ExtensionCount},
            NewData: SmartEndSnapshot{
                FileSizeBytes: snap.FileSizeBytes, FileGrowthBps: snap.FileGrowthBps,
                LastFileGrowthAt: timePtr(snap.LastFileGrowthAt),
                ExtensionCount:   task.ExtensionCount + 1,
                NewEndTime:       newEnd,
                // SilenceSince 视 watcher 暴露
            },
        })
    }
    return nil
}
```

### Pattern 3: 多 input `taskEndedCh` OR 合并

**What:** 多 input 任务下 scheduler 一次性拉 `coordinator.WatcherChannels(taskID)` 返回 `[]<-chan struct{}`，在 monitorTask 入口建一个聚合 goroutine `chan struct{}` 任一 close 即传播；或直接用 `reflect.Select` —— 二者选其一。

**When to use:** EARLY-04 "任一 watcher 触发整体结束"。

**Why:** Go `select` 不支持变长 channel slice；用聚合 ch 简单且可控。

**Code sketch:**
```go
// mergeWatchers 用 goroutine 把 N 个 channel 收拢到 1 个
func mergeWatchers(ctx context.Context, chans []<-chan struct{}, logger *zap.Logger) <-chan struct{} {
    out := make(chan struct{}, 1)
    go func() {
        defer close(out)
        if len(chans) == 0 {
            <-ctx.Done()
            return
        }
        cases := make([]reflect.SelectCase, len(chans)+1)
        for i, ch := range chans {
            cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
        }
        cases[len(chans)] = reflect.SelectCase{Dir: reflect.SelectDefault}
        // reflect.Select 不支持直接 select on ctx.Done 与 N chan + ctx 一起；
        // 改用 select 内嵌 goroutine 关闭
        for _, ch := range chans {
            go func(c <-chan struct{}) {
                select {
                case <-c:
                    select {
                    case out <- struct{}{}:
                    default: // 已关闭则忽略
                    }
                case <-ctx.Done():
                }
            }(ch)
        }
        // wait for ctx
        <-ctx.Done()
    }()
    return out
}
```

> ⚠ Plan-02 阶段需 caution 测试：close-once + buffered 1 + 多 goroutine 推 out 的 race。建议改 `chan struct{}` + `sync.Once` 而非 reflect.Select。

### Pattern 4: CFG-03 双守门（scheduler + coordinator）

**What:** `smart_end.enabled=false` 时：(a) `coordinator.StartRecordingWithConfig` 不构造 watcher（已 Phase 24 落地 — `coordinator.go:199`）；(b) scheduler `monitorTask` 入口 if-else 选旧 timer-only 路径；二者独立但必须都生效。

**When to use:** 全局回退开关。

**Why:** 双层守门避免单点失效；scheduler 不知道 watcher 是否启，统一走 `cfg.SmartEnd.Enabled` 入口分支。

### Anti-Patterns to Avoid

- **直接把 `task.ExtendStepMin()` 写在 scheduler 内做 `task.EndTime.Add(time.Hour * 30 / 60)`**：违反 AUDIT-04；service 入口唯一。
- **scheduler 内调用 `s.db.Update(...).Updates(...)` 直接改 GORM**：绕开 service，违反 Phase 19 ctx 级联约定；audit log 也漏掉。
- **`sync.Once` 在 scheduler 与 service 各做一次 close**：可能导致 audit log 双写；统一 service 入口。
- **`<-time.After(d)` 替代 `time.NewTimer(d).Stop()`**：GC 不释放 timer，Phase 24 Phase 25 监控长跑会累积。
- **改写 `VideoRecordingTaskStatus` 枚举**：PRD §2.2 明确 Non-Goal。
- **引入 `github.com/prometheus/client_golang`**：项目无 prom 集成；仅暴露 atomic + Record* 即可（OBS-05："可选 Prometheus counter 接入点"；本阶段**不接 prom**）。
- **E2E 测试用真实 ffmpeg / 真实 Huawei HTTPS**：Phase 23-1-04 已落 fake / fixture 模式；Phase 25 E2E 沿用 `mockCoordinator` + `mockTaskService` + fake `recorder.ActivityWatcher` 接口注入。
- **`<-timer.C` 与 `<-taskEndedCh` 用 same `time.After` 共 channel**：Phase 24 已用 `time.NewTimer`，本阶段不变。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GORM column update with snapshot | `db.Exec("UPDATE ...")` 直写 SQL | `db.Model(&task).Updates(map[string]interface{}{...})` | 与 `UpdateTaskStatus`/`UpdateRecordingPaths` 现有用法对齐，ctx 级联 + AutoMigrate 字段映射可控 |
| audit log 写入 | `db.Exec("INSERT INTO audit_logs ...")` | `audit.AuditLogService.RecordChange(ctx, opts)` | 复用脱敏 + 异步队列 + 错误码映射；现有 12 处调用都走这条路 |
| 多 channel 并发 select | 手写 `reflect.Select` 复杂 case slice | 聚合 goroutine + `sync.Once` 合并到单一 `chan struct{}` | reflect.Select 不可与 `<-ctx.Done()` 优雅结合；本阶段选 goroutine fan-in（已在 Pattern 3 标 ⚠） |
| Atomic 计数器 | `sync/atomic.AddInt64` 全局 var | `obs.SmartEndMetrics.Record{Extend,EarlyEnd,WatcherDegraded}*` 包装函数 | 后续 Prometheus 接入只改 Record* 函数包装；scheduler / service 只调函数不直读 atomic |
| close-once channel | `if !ended { close(ch); ended = true }` | Phase 24 已在 watcher 内部 `sync.Once.Do(func(){ close(...) })` 物理保证；Phase 25 仅消费不重写 | 多次 close panic；sync.Once 是惯例 |
| 时间归一化 (Beijing → UTC) | 自写时区逻辑 | 现有 `s.encryptor != nil` / `time.FixedZone("CST", 8*3600)` 模式 — `video_recording_task_service.go:323` | 与 CreateTask 路径对齐，避免时区错误 |
| audit log JSON 序列化 | 手写 `json.Marshal` 各处 | 统一 `audit.RecordChangeOpts.OldData/NewData interface{}` 走 `AuditLog.MarshalJSON` | 现有 12 处调用统一路径，sanitize + diff 共用 |

**Key insight:** Phase 25 唯一新路径是 `service.UpdateTaskExtension` 与 `service.MarkTaskEndedEarly`，所有 audit / GORM / OBS 日志都复用既有底座；scheduler 改动仅是 `select` 三路 + 调用 service。

## Common Pitfalls

### Pitfall 1: `taskEndedCh` close-once 在多 input 任务下导致聚合 ch 重复关闭

**What goes wrong:** EARLY-04 多 input 下 scheduler 启聚合 goroutine，configType A 触发 close → 推 out → 主 `select` 收 → completeTask；100ms 后 configType B 也触发时聚合 goroutine 已退出但 channel 已 close；若设计 buggy 二次 close 可能 panic。

**Why it happens:** Go close-once 规则，多 goroutine 推 `out ch` + 单次 close 容易出错。

**How to avoid:** 聚合 ch 设计为 `chan struct{}` + `sync.Once.Do(func(){ close(out) })`，每个 source channel 启独立 goroutine 监听 `c <- struct{}{}`；recv 一次就 done 不重试。

**Warning signs:** `go test -race` 偶发 panic / `close of closed channel`。

### Pitfall 2: `monitorTask` select case 顺序依赖 Go runtime 调度

**What goes wrong:** PRD §5.2 要求 `taskEndedCh` 永远先于 timer.C，但 Go 文档只承诺多 ready case 随机选；scheduler 误把 `<-timer.C` 放第一个 case 时同样多 ready 时触发 timer。

**Why it happens:** Go spec 不保证 case 优先级。

**How to avoid:** Go 语义本身保证：channel close 立即 ready + timer 仅在时间到 ready —— 实际场景中两者**不同时**ready；不必依赖 case 顺序，纯靠物理时序已满足 EARLY-03。但保留 `<-taskEndedCh` 放第一个 case 是惯例。

**Warning signs:** E2E 测试 1ms 精度下偶发 timer 先触发。

### Pitfall 3: `smart_end.enabled=false` 时 scheduler 仍读 `taskEndedCh`

**What goes wrong:** `cfg.SmartEnd.Enabled=false` 应"退回纯 EndTime 行为"（CFG-03），但 scheduler `monitorTask` 无守门 → watcher 不启 → `taskEndedCh` 为 nil channel → `<-nil` 永远不触发 → 任务等到 timer 结束（行为对但浪费一个 case slot）。

**Why it happens:** Phase 24 coordinator 已守门，scheduler 假设 watcher 一定启。

**How to avoid:** `monitorTask` 入口 `if !s.config.SmartEnd.Enabled` 直接走旧 timer-only 路径；不读 watcher / taskEndedCh / IsActive。

**Warning signs:** 单元测试 `smart_end.enabled=false` 路径下 `IsActive` panic（watcher nil）。

### Pitfall 4: service 写 audit 时 task 上下文丢失（update_time / extension_count 旧值）

**What goes wrong:** `UpdateTaskExtension` 内 `First(&task)` 拿当前 ExtensionCount；接着 `Updates(...)` 改 ExtensionCount；写 audit `NewData` 时引用 `task.ExtensionCount + 1` —— OK。但 audit `OldData` 引用 `task.ExtensionCount`（旧值）需要在 `Updates` 之前 marshal。

**Why it happens:** `Updates` 之后 `task.ExtensionCount` 字段未刷新（除非重新 `First`）。

**How to avoid:** service 内 `extensionOld := task.ExtensionCount; newCount := extensionOld + 1; updates := map[...]{"extension_count": newCount}; ...` 显式保存局部变量；audit OldData/NewData 用局部变量。

**Warning signs:** audit log diff 出现 `extension_count: 0 → 0` 静默丢失。

### Pitfall 5: scheduler `completeTask("smart_early_end")` 与现有状态机冲突

**What goes wrong:** `completeTask` 内部 `updateTaskStatus(ctx, taskID, VideoStatusConverting, "")`（`video_scheduler.go:659`）；但 EARLY-01/02 路径调 `MarkTaskEndedEarly` 写 `EndedEarly=true` 然后 `completeTask` 再改 `Status=converting` —— 二者不冲突（`EndedEarly` 是独立字段），但 audit log 双写风险（service 写 + scheduler 写）。

**Why it happens:** completeTask 内部不写 audit log；scheduler 的 `completeTask` 不调 `MarkTaskEndedEarly`。

**How to avoid:** 限定 `service.MarkTaskEndedEarly(taskID, reason, byHuaWeiAPI)` 仅写 GORM 5 字段 + audit log，不动 `Status`；scheduler 仍调现有 `completeTask` 改 Status=converting；二者正交。

**Warning signs:** audit log 出现 `EndedEarly=true → false` 反向 diff。

### Pitfall 6: 多 input 任务 channel merge 时 `select` 与 `reflect.Select` 互操作 bug

**What goes wrong:** `reflect.Select` Case 要求 `reflect.ValueOf(<-chan)`，但 `<-chan struct{}` 双向转换易出错；多次 `reflect.Select` 调用不一致 channel set 导致漏读。

**Why it happens:** reflect API 在 select 上对 close 语义不直观。

**How to avoid:** Phase 25 直接用 fan-in goroutine 模式（Pattern 3）而非 reflect.Select；测试覆盖 close-once + race detector。

**Warning signs:** `go test -race` 报 `Send on closed channel`。

### Pitfall 7: Prometheus 接入点过早引入 client_golang

**What goes wrong:** OBS-05 字面读"可选 Prometheus counter"，planner 引入 `github.com/prometheus/client_golang/promauto` 加重依赖。

**Why it happens:** 字面读 OBS-05 误以为必接 prom。

**How to avoid:** 仅暴露 `observability/smart_end_metrics.go` 3 个 `atomic.Int64` + `Record*`；README 注释"后续 prom 实现 import 此文件即可"。**禁止**改 go.mod。

**Warning signs:** `go.mod` diff 含 `github.com/prometheus/client_golang`。

### Pitfall 8: Phase 24 ActivitySnapshot 字段缺 `file_size_bytes` / `file_growth_bps`，Phase 25 audit log 缺字段

**What goes wrong:** AUDIT-02 / AUDIT-03 要求 snapshot 6 字段，但 Phase 24 `ActivitySnapshot` 只有 4 字段（`SilenceSince` / `LastFileGrowthAt` / `LastHuaWeiStateEmpty` / `HuaWeiEmptySince` 等）—— `file_size_bytes` / `file_growth_bps` 未透出。

**Why it happens:** Phase 24 RESEARCH.md Pitfall 6 已注明：`fileTicker` 局部计算 `growthBps` 但未存到 state。

**How to avoid:** Phase 25 PLAN-01 第一步先改 `activity_watcher.go` `ActivitySnapshot` 加 2 字段 + `fileTicker` 维护 `lastFileSize` 已是 expose（已是 public 字段）；`growthBps` 需新增 `lastFileGrowthBps int64` 字段在 ticker 同步。

**Warning signs:** audit log snapshot 缺 `file_size_bytes` / `file_growth_bps`，CI golden file 校验失败。

## Files to Create/Modify (with paths and rationale)

### Create

| Path | Rationale |
|------|-----------|
| `internal/observability/smart_end_metrics.go` | OBS-05 atomic 计数器接入点，3 个 `Record*` 函数；scheduler / service / recorder 调用 |
| `internal/scheduler/task_ended_channel_helper.go` | Pattern 3 `mergeWatchers(ctx, chans, logger)` fan-in goroutine；close-once `sync.Once` |

### Modify

| Path | Rationale |
|------|-----------|
| `internal/scheduler/video_scheduler.go:542-606` | `monitorTask` 改 select 三路 + CFG-03 守门；`<-taskEndedCh` 优先 case；服务方法改返回 error 含 max_extend_reached；扩展 `RecorderCoordinatorInterface`（或新增 `WatcherChannels` getter） |
| `internal/scheduler/video_scheduler_test.go:138-159` | 加 `mockCoordinator.WatcherChannels` 实现（返回 fake `chan struct{}`）+ 7+ E2E subtest |
| `internal/recorder/activity_watcher.go:65-77` | `ActivitySnapshot` 加 `FileSizeBytes` / `FileGrowthBps` 2 字段；`fileTicker` 同步维护 `lastFileGrowthBps` |
| `internal/recorder/activity_watcher_test.go` | 扩 Snapshot 字段断言；多 input 合并测试在 scheduler 测（watcher 仅返回 `<-chan struct{}`） |
| `internal/recorder/coordinator.go` | `RecorderCoordinatorInterface` 加 `WatcherChannels(taskID uint) []<-chan struct{}`（scheduler 消费）；`SimpleRecordingCoordinator` 实现：遍历 `c.processes` 找 key 以前缀 `taskID_` 开头且 `process.taskEndedCh != nil` |
| `internal/services/video_recording_task_service.go` | 新增 `UpdateTaskExtension(ctx, taskID, deltaMin, reason, snap)` + `MarkTaskEndedEarly(ctx, taskID, reason, byHuaWeiAPI bool, snap)` + 注入 `*AuditLogService` + `*config.Config`（cfg.SmartEnd 读 MaxExtendCount） |
| `internal/services/video_recording_task_service_test.go` | 扩 5 字段读写 + audit snapshot JSON marshal 断言 + max_extend_reached 错误路径 |

### Explicitly do not modify in Phase 25

- `internal/recorder/activity_watcher.go:127-135` `Start()`：Phase 24 4 goroutine 已稳；不动
- `internal/recorder/activity_watcher.go:139-145` `Stop()`：Phase 24 close-once 已稳；不动
- `internal/recorder/silence_parser.go` + `huawei_state_client.go`：Phase 24 已交付；不动
- `internal/huawei/client.go`：Phase 23 交付；不动
- `internal/errors/errors.go` / `mapping.go` / `docs/errors.md`：Phase 23 3 sentinel 已加；不动
- `internal/config/smart_end.go`：Phase 23 已加 14 字段；不动
- `config.yaml` / `bin/config.yaml`：Phase 23 已加 smart_end 段；不动
- `internal/models/video_recording_task.go`：Phase 23 已加 5 字段；不动
- `.github/workflows/ci.yml`：现有 `go test -race ./...` + errors.md sync-check 已覆盖；不加新 step
- 前端：Non-Goal

## Patterns & Conventions (closest analogs in codebase)

| New work | Closest analog | Reuse |
|----------|----------------|-------|
| service 层 audit log 写入 | `audit.AuditLogService.RecordChange(ctx, opts)` | 12 处现有调用一致签名：`OldData/NewData interface{}` 自动 JSON marshal |
| scheduler 内 select 多 channel | `scheduler.monitorTask:541-606` 现状 | `select` 内 `case <-timer.C` + `case <-ctx.Done()` + `case newEndTime := <-updateChan` 三路；Phase 25 加第 4 路 `<-taskEndedCh` |
| service 注入 `*AuditLogService` | `VideoRecordingTaskService.SetScheduler(scheduler.SchedulerInterface)` | Service 构造函数 `NewVideoRecordingTaskService(db, logger, encryptor...)` 加变参 `auditSvc ...*audit.AuditLogService`；call site 增量改不破坏既有 |
| scheduler 读 `processes` map 拿 `taskEndedCh` | `coordinator.StopRecording:620-648` 遍历 `c.processes` 按前缀筛选 | 同模式，迁移到 `RecorderCoordinatorInterface.WatcherChannels(taskID)` 返回 `[]<-chan struct{}` |
| close-once channel | `scheduler.taskUpdateChans[task.ID]` 删除 + `make(chan time.Time, 1)` | 已知惯例：`defer delete(s.taskUpdateChans, task.ID); close(ch)` |
| OBS 日志结构化 | existing `s.logger.Info("任务触发时间计算", zap.Uint("task_id", ...), zap.Time("end_time", ...))` | 与既有日志格式对齐 |

## Validation Architecture (Nyquist — test coverage requirements)

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` + testify v1.11.1 |
| Config file | none（standard `go test`） |
| Quick run command | `go test ./internal/scheduler ./internal/services ./internal/recorder ./internal/observability -run 'TestSchedulerMonitorTask|TestUpdateTaskExtension|TestMarkTaskEndedEarly|TestSmartEndMetrics|TestActivityWatcher_SnapshotExtension|TestRecorderCoordinator_WatcherChannels' -count=1 -v` |
| Full suite command | `go test -race ./...` |
| Estimated runtime | quick ~10s；full ~120s |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SCHED-01 | monitorTask select 三路 — 任一触发即结束 | unit (fake watcher + fake updateChan) | `go test ./internal/scheduler -run 'TestMonitorTask_TripleSelect' -count=1` | ❌ Wave 0 |
| SCHED-02 | EndTime 到点 watcher.IsActive() 活跃 → ExtendStepMin | unit | `go test ./internal/scheduler -run 'TestMonitorTask_OnTimerActive_Extends' -count=1` | ❌ Wave 0 |
| SCHED-03 | taskEndedCh close 后 EndTime.C 不再生效 | unit (close-then-no-fire) | `go test ./internal/scheduler -run 'TestMonitorTask_TaskEnded_PreemptsTimer' -count=1` | ❌ Wave 0 |
| SCHED-04 | 手动 updateChan 不重置 ExtensionCount | unit (snapshot before/after) | `go test ./internal/scheduler -run 'TestMonitorTask_ManualUpdateDoesNotResetCount' -count=1` | ❌ Wave 0 |
| EXTEND-01 | ExtensionCount 上限 4，4×30min=2h 总上限 | unit (service) | `go test ./internal/services -run 'TestUpdateTaskExtension_MaxLimit' -count=1` | ❌ Wave 0 |
| EXTEND-02 | 上限达后 → max_extend_reached，status=completed | unit (gold assertion on `completeTask("max_extend_reached")`) | `go test ./internal/scheduler -run 'TestMonitorTask_MaxExtendReached' -count=1` | ❌ Wave 0 |
| EARLY-01 | H 信号触发 → smart_early_end + EndedByHuaWeAPI=true | unit (fake watcher EndedReason="huawei_state_empty") | `go test ./internal/services -run 'TestMarkTaskEndedEarly_HuaweiSignal' -count=1` | ❌ Wave 0 |
| EARLY-02 | A+B → smart_early_end + EndedByHuaWeAPI=false | unit (fake EndedReason="both_silence_and_stall") | `go test ./internal/services -run 'TestMarkTaskEndedEarly_BothSilenceAndStall' -count=1` | ❌ Wave 0 |
| EARLY-03 | 提前结束优先 timer | 复用 SCHED-03 | 同 | ❌ Wave 0 |
| EARLY-04 | 多 input 任一 watcher 触发整体结束 | unit (mock multi input) | `go test ./internal/scheduler -run 'TestMonitorTask_MultiInput_AnyEndsAll' -count=1` | ❌ Wave 0 |
| AUDIT-02 | 延时 audit log 含 6 字段 snapshot | unit (golden JSON) | `go test ./internal/services -run 'TestUpdateTaskExtension_AuditSnapshot' -count=1` | ❌ Wave 0 |
| AUDIT-03 | 提前结束 audit log 含 5 字段 | unit (golden JSON) | `go test ./internal/services -run 'TestMarkTaskEndedEarly_AuditSnapshot' -count=1` | ❌ Wave 0 |
| AUDIT-04 | service 封装 + 统一入口 | unit (验证 scheduler 直 db.Update 调用次数 = 0) | `go test ./internal/services -run 'TestServiceEntrypoint_OnlyPath' -count=1` | ❌ Wave 0 |
| CFG-03 | `enabled=false` 退回 timer-only | unit | `go test ./internal/scheduler -run 'TestMonitorTask_SmartEndDisabled' -count=1` | ❌ Wave 0 |
| CFG-04 | `huawei_enabled=false` 跳过 huaweiPoller（Phase 24 已覆盖） | 复用 24-04 测试 | `go test ./internal/recorder -run 'TestActivityWatcher_HuaweiDisabled' -count=1` | ✅ 24-04 |
| OBS-01 | `smart_extend` 日志字段 | unit (zap observer) | `go test ./internal/services -run 'TestUpdateTaskExtension_LogFields' -count=1` | ❌ Wave 0 |
| OBS-02 | `smart_early_end` 日志字段 | unit (zap observer) | `go test ./internal/services -run 'TestMarkTaskEndedEarly_LogFields' -count=1` | ❌ Wave 0 |
| OBS-03 | `max_extend_reached` 日志 | unit (zap observer) | `go test ./internal/scheduler -run 'TestMonitorTask_LogMaxExtendReached' -count=1` | ❌ Wave 0 |
| OBS-04 | `activity_watcher_degraded` 日志（Phase 24 已覆盖） | 复用 24-04 测试 | `go test ./internal/recorder -run 'TestActivityWatcher_DegradedLog' -count=1` | ✅ 24-04 |
| OBS-05 | `smart_extend_total` atomic +1 | unit | `go test ./internal/observability -run 'TestSmartEndMetrics_RecordExtend' -count=1` | ❌ Wave 0 |

### snapshot field coverage required

| Field | Type | Source | Required by |
|-------|------|--------|-------------|
| `silence_since` | `*time.Time` (nullable) | `ActivitySnapshot.SilenceSince` (zero → null) | AUDIT-02 |
| `last_file_growth` | `*time.Time` (nullable) | `ActivitySnapshot.LastFileGrowthAt` (zero → null) | AUDIT-02 |
| `file_size_bytes` | `int64` | `ActivitySnapshot.FileSizeBytes` (Phase 25 NEW) | AUDIT-02 |
| `file_growth_bps` | `int64` | `ActivitySnapshot.FileGrowthBps` (Phase 25 NEW) | AUDIT-02 |
| `extension_count` | `int` | `task.ExtensionCount` after update | AUDIT-02 |
| `new_end_time` | `time.Time` | `task.EndTime` after update | AUDIT-02 |
| `ended_by_huawei_api` | `bool` | `task.EndedByHuaWeAPI` | AUDIT-03 |
| `ended_early_reason` | `string` | `task.EndedEarlyReason` | AUDIT-03 |

### Sampling Rate

- **Per task commit:** `go test ./internal/scheduler ./internal/services -count=1`（包内快速反馈，< 5s）
- **Per wave merge:** quick run command（含 scheduler + service + recorder + observability 子测）
- **Phase gate:** `go test -race ./...` + `go vet ./...` + `go build ./...`，全绿再 `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/recorder/activity_watcher.go` 扩 `ActivitySnapshot` 2 字段（FileSizeBytes + FileGrowthBps）+ `fileTicker` 维护 `lastFileGrowthBps`
- [ ] `internal/recorder/activity_watcher_test.go` 加新字段断言
- [ ] `internal/recorder/coordinator.go` 加 `WatcherChannels(taskID uint) []<-chan struct{}` + `RecorderCoordinatorInterface` 升级
- [ ] `internal/scheduler/task_ended_channel_helper.go` 增 `mergeWatchers` fan-in 实现
- [ ] `internal/scheduler/video_scheduler.go:542-606` `monitorTask` 改 select 三路 + CFG-03 守门
- [ ] `internal/services/video_recording_task_service.go` 加 `UpdateTaskExtension` + `MarkTaskEndedEarly` + 注入 `audit` + `cfg`
- [ ] `internal/observability/smart_end_metrics.go` 增 3 个 atomic.Int64 + Record* 函数
- [ ] `internal/scheduler/video_scheduler_test.go` 加 `mockCoordinator.WatcherChannels` fake
- [ ] `internal/services/video_recording_task_service_test.go` 加 5 字段读写 + audit JSON 断言 + 上限错误路径
- [ ] `internal/scheduler/video_scheduler_test.go` 加 7+ E2E subtest
- [ ] `internal/observability/smart_end_metrics_test.go` 加 atomic +1 断言

## Code Examples

Verified patterns from official sources / codebase:

### Scheduler select 三路（代码草图）

```go
// Source: PRD §5.2 + 既有 monitorTask:542-606 + Phase 24 RESEARCH.md "Pattern 3"
func (s *VideoSimpleScheduler) monitorTask(ctx context.Context, task *models.VideoRecordingTask) {
    s.mu.Lock()
    updateChan := make(chan time.Time, 1)
    s.taskUpdateChans[task.ID] = updateChan
    s.taskEndTimes[task.ID] = task.EndTime
    s.mu.Unlock()
    defer func() {
        s.mu.Lock()
        delete(s.taskUpdateChans, task.ID)
        delete(s.taskEndTimes, task.ID)
        s.mu.Unlock()
    }()

    // CFG-03 守门
    if !s.config.SmartEnd.Enabled {
        s.monitorTaskEndTimeOnly(ctx, task, updateChan) // 旧 timer-only 路径
        return
    }

    // EARLY-04: 多 input 任一 taskEndedCh 触发即结束
    chans := s.coordinator.WatcherChannels(task.ID)
    taskEndedCh := mergeWatchers(ctx, chans, s.logger)

    var watcher recorder.ActivityWatcher // 由 coordinator 暴露；本阶段重构暂用接口
    _ = watcher // Phase 25-02 取 snapshot 路径独立

    endTime := task.EndTime
    for {
        timer := time.NewTimer(time.Until(endTime))
        select {
        case <-ctx.Done():
            timer.Stop()
            return
        case <-taskEndedCh: // SCHED-03 优先
            timer.Stop()
            s.handleTaskEnded(ctx, task) // 含 MarkTaskEndedEarly + audit log + completeTask
            return
        case newEndTime, ok := <-updateChan:
            timer.Stop()
            if !ok { return }
            endTime = newEndTime // SCHED-04: 不重置 ExtensionCount
            s.logger.Info("任务结束时间已更新",
                zap.Uint("task_id", task.ID),
                zap.Time("new_end_time", newEndTime),
            )
        case <-timer.C:
            // SCHED-02 + EXTEND-01/02
            s.handleEndTimeReached(ctx, task, endTime)
            // 可能 return (收尾) 或更新 endTime 继续 loop
            if /* 已经 completeTask */ {
                return
            }
            endTime = task.EndTime // UpdateTaskExtension 已改 task.EndTime
        }
    }
}
```

### service UpdateTaskExtension（代码草图）

```go
// Source: PRD §8 AUDIT-02 + Phase 24 activity_watcher.go ActivitySnapshot
type SmartEndSnapshot struct {
    SilenceSince     *time.Time `json:"silence_since,omitempty"`
    LastFileGrowthAt *time.Time `json:"last_file_growth,omitempty"`
    FileSizeBytes    int64      `json:"file_size_bytes"`
    FileGrowthBps    int64      `json:"file_growth_bps"`
    ExtensionCount   int        `json:"extension_count"`
    NewEndTime       time.Time  `json:"new_end_time"`
    EndedByHuaWeiAPI bool       `json:"ended_by_huawei_api"`
    EndedEarlyReason string     `json:"ended_early_reason,omitempty"`
}

func (s *VideoRecordingTaskService) UpdateTaskExtension(
    ctx context.Context, taskID uint, deltaMin int, reason string, snap ActivitySnapshot,
) error {
    var task models.VideoRecordingTask
    if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
        return err
    }

    // EXTEND-01 上限守门
    if task.ExtensionCount >= s.cfg.SmartEnd.MaxExtendCount {
        return fmt.Errorf("extension limit %d reached for task %d: %w",
            s.cfg.SmartEnd.MaxExtendCount, taskID, apperrors.ErrRecordingSmartExtend)
    }

    extensionOld := task.ExtensionCount
    newEndTime := task.EndTime.Add(time.Duration(deltaMin) * time.Minute)
    newCount := extensionOld + 1

    updates := map[string]interface{}{
        "end_time":              newEndTime,
        "extension_count":       newCount,
        "last_extension_reason": reason,
    }
    if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
        return err
    }
    task.EndTime = newEndTime // 让 scheduler 读回去
    task.ExtensionCount = newCount

    // OBS-01
    s.logger.Info("smart_extend",
        zap.Uint("task", taskID),
        zap.Int("count", newCount),
        zap.Time("new_end", newEndTime),
        zap.String("reason", reason),
    )
    observability.RecordSmartExtend()

    // AUDIT-02
    if s.auditSvc != nil {
        snapSt := SmartEndSnapshot{
            SilenceSince: nilIfZero(snap.SilenceSince),
            LastFileGrowthAt: nilIfZero(snap.LastFileGrowthAt),
            FileSizeBytes:    snap.FileSizeBytes,
            FileGrowthBps:    snap.FileGrowthBps,
            ExtensionCount:   newCount,
            NewEndTime:       newEndTime,
        }
        _ = s.auditSvc.RecordChange(ctx, audit.RecordChangeOpts{
            Action:     models.ActionUpdate,
            Module:     models.ModuleTask,
            Resource:   "video_recording_task",
            ResourceID: &taskID,
            OldData:    map[string]interface{}{"end_time": task.EndTime, "extension_count": extensionOld},
            NewData:    snapSt,
        })
    }
    return nil
}

func (s *VideoRecordingTaskService) MarkTaskEndedEarly(
    ctx context.Context, taskID uint, reason string, byHuaWeiAPI bool, snap ActivitySnapshot,
) error {
    var task models.VideoRecordingTask
    if err := s.db.WithContext(ctx).First(&task, taskID).Error; err != nil {
        return err
    }

    updates := map[string]interface{}{
        "ended_early":           true,
        "ended_early_reason":    reason,
        "ended_by_huawei_api":   byHuaWeiAPI,
    }
    if err := s.db.WithContext(ctx).Model(&task).Updates(updates).Error; err != nil {
        return err
    }

    // OBS-02
    s.logger.Info("smart_early_end",
        zap.Uint("task", taskID),
        zap.String("reason", reason),
        zap.Any("snapshot", SmartEndSnapshot{
            SilenceSince: nilIfZero(snap.SilenceSince),
            LastFileGrowthAt: nilIfZero(snap.LastFileGrowthAt),
            FileSizeBytes: snap.FileSizeBytes,
            FileGrowthBps: snap.FileGrowthBps,
            EndedByHuaWeiAPI: byHuaWeiAPI,
            EndedEarlyReason: reason,
        }),
    )
    observability.RecordSmartEarlyEnd()

    // AUDIT-03
    if s.auditSvc != nil {
        _ = s.auditSvc.RecordChange(ctx, audit.RecordChangeOpts{
            Action:     models.ActionUpdate,
            Module:     models.ModuleTask,
            Resource:   "video_recording_task",
            ResourceID: &taskID,
            NewData: map[string]interface{}{
                "ended_early":          true,
                "ended_early_reason":   reason,
                "ended_by_huawei_api":  byHuaWeiAPI,
                "snapshot": SmartEndSnapshot{ /* same */ },
            },
        })
    }
    return nil
}

func nilIfZero(t time.Time) *time.Time {
    if t.IsZero() {
        return nil
    }
    return &t
}
```

### ActivitySnapshot 字段扩展（代码草图）

```go
// Source: Phase 24 activity_watcher.go:65-77 现状 + Phase 25 AUDIT-02 需求
type ActivitySnapshot struct {
    SilenceSince         time.Time
    LastFileGrowthAt     time.Time
    HuaWeiEmptySince     time.Time
    LastHuaWeiStateEmpty bool
    HuaWeiDegraded       bool
    SilenceDegraded      bool
    Ended                bool
    EndedReason          string
    LastSilenceStart     time.Time
    TotalSilenceDuration time.Duration
    // Phase 25 新增
    FileSizeBytes        int64 `json:"file_size_bytes"`  // NEW
    FileGrowthBps        int64 `json:"file_growth_bps"`  // NEW
}
```

```go
// activity_watcher.go fileTicker (Phase 24 activity_watcher.go:316-368) 增维护 lastFileGrowthBps
type ActivityWatcher struct {
    // ...Phase 24 已加的字段
    lastFileGrowthBps int64 // Phase 25 NEW (fileTicker 写, Snapshot 读)
}

func (w *ActivityWatcher) Snapshot() ActivitySnapshot {
    w.mu.Lock()
    defer w.mu.Unlock()
    return ActivitySnapshot{
        // ...Phase 24 已有
        FileSizeBytes: w.lastFileSize,    // Phase 24 已有
        FileGrowthBps: w.lastFileGrowthBps, // Phase 25 NEW
    }
}
```

```go
// fileTicker loop 内（Phase 24 activity_watcher.go:351-365）
w.mu.Lock()
deltaBytes := size - w.lastFileSize
if deltaBytes < 0 { deltaBytes = 0 }
growthBps := deltaBytes * 8 / checkInterval
w.lastFileSize = size
if growthBps >= w.cfg.SmartEnd.FileMinGrowthBPS {
    w.lastFileGrowthAt = w.now()
    w.lastFileGrowthBps = growthBps // Phase 25 NEW: 缓存上一次的 growthBps 供 Snapshot
    w.statConsecFailures = 0
} else {
    // not growth: 保留旧 lastFileGrowthBps 不清零 (Pitfall 6 同样原则)
}
w.mu.Unlock()
```

### observability/smart_end_metrics.go 接入点

```go
// Source: OBS-05 PRD §10 + 现有 sync/atomic 惯例 (无 prom 集成)
package observability

import "sync/atomic"

var (
    SmartExtendTotal     atomic.Int64
    SmartEarlyEndTotal   atomic.Int64
    WatcherDegradedTotal atomic.Int64
)

func RecordSmartExtend()     { SmartExtendTotal.Add(1) }
func RecordSmartEarlyEnd()   { SmartEarlyEndTotal.Add(1) }
func RecordWatcherDegraded() { WatcherDegradedTotal.Add(1) }

// 后续 Prometheus 接入: 在 internal/observability/prom_init.go 用 promauto.NewCounterFunc 包装以上 atomic
// (本阶段不实施 — 仅暴露接入点)
```

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 不参与 auth 路径 |
| V3 Session Management | no | 不参与 token 路径 |
| V4 Access Control | no | scheduler 内部触发，无 endpoint |
| V5 Input Validation | yes | `cfg.SmartEnd.MaxExtendCount` 已 Validate；`UpdateTaskExtension` 入参 deltaMin 在 service 层做 `> 0` 断言；audit log 通过现有 Sanitizer |
| V6 Cryptography | no | 无新 crypto |
| V7 Error Handling | yes | `ErrRecordingSmartExtend` / `ErrRecordingSmartEarlyEnd` sentinel + mapping.go 已加 (Phase 23)；service 包 error 时 `errors.Is` 检测 |
| V9 Logging | yes | OBS-01..04 结构化日志（task_id / reason / snapshot）通过 zap；不含 password / token |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| `task_id` 注入 audit log 字段 | Tampering | audit log 字段由 GORM model 与 ActivitySnapshot 强类型控制；不接 user input 字符串 |
| 恶意 delay（外部手动 `UpdateTaskEndTime`）绕过上限 | Elevation of Privilege | SCHED-04: 手动延时不重置 ExtensionCount，仅 scheduler monitorTask 调 service `UpdateTaskExtension` 走守门 |
| false watcher 触发导致任务提前结束 | DoS | watcher 在 Phase 24 已实现 multi-level degrade；Phase 25 仅消费 Snapshot EndedReason，不绕过状态机 |
| audit log JSON marshal 嵌套过深 | DoS | SmartEndSnapshot 是 flat struct（≤8 字段），无嵌套 marshal 攻击面 |
| Prometheus counter 整数溢出 | DoS | atomic.Int64 上限约 9.2e18，单任务延时频次远小于此 |

## Package Legitimacy Audit

> 本阶段不新增任何 Go module / npm package。`go.mod` 在 Phase 23/24 已锁定，本阶段不变。**无 slopcheck 审计需求**。

| Package | Action |
|---------|--------|
| (none) | — |

## Runtime State Inventory (OUT OF SCOPE — no rename)

本阶段不涉及 rename / rebrand / 字符串替换；Runtime State Inventory 不适用。

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| scheduler `monitorTask` 单 `time.NewTimer` + `case <-updateChan` | scheduler `select` 三路 + `<-taskEndedCh` | Phase 25 SCHED-01 | 提前结束信号不再延迟 timer 误触发的窗口 |
| 每处手动 UPDATE GORM column | service `UpdateTaskExtension` / `MarkTaskEndedEarly` 单入口 | Phase 25 AUDIT-04 | audit log 与 GORM 字段一致性由 service 保证；scheduler 仅 consumer |
| audit log 字符串拼 JSON | `audit.RecordChange(opts)` `interface{}` 自动 marshal | Phase 10 已交付 | 复用既有 12 处调用，无新写法 |
| `time.NewTimer` 单次 + 手动 reset | 单循环 `select` + `time.NewTimer(remaining)` 每轮 new timer | Phase 25 monitorTask | timer 提前触发路径消失，select 内 timer.C 与 taskEndedCh 物理互斥 |

**Deprecated/outdated:**
- (无)

## Assumptions Log

> Claims tagged [ASSUMED] need user confirmation. Phase 25 仅调度层 + service 层整合，无新合规/安全/性能目标争议；故此表空。所有"功能契约"皆由 Phase 23/24 PRD + REQUIREMENTS.md 锁定。

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| (none) | — | — | — |

## Open Questions

1. **`taskEndedCh` 多 input 合并是否需要 `sync.Once` 二次保护？**
   - What we know: Phase 24 已用 `sync.Once` 在 watcher 内部 close；scheduler 聚合 fan-in goroutine 收到 close 后也只 close 一次（外部 sync.Once）。
   - What's unclear: 第一个 goroutine close 后其他 goroutine 仍监听 `case <-c:` 是否会因 c 已 close 而永不触发？→ 是，close 之后 `<-c` 立即可读；但我们是 `<-c` → `out <- struct{}{}`，一旦多个 goroutine 看到 c 已 close 同时想推 out，可能 race。
   - Recommendation: 聚合 ch 用 `chan struct{}` buffered 1 + `select` push + `default` 丢弃；或外部 `sync.Once` guard 推 out。**Phase 25 PLAN-02 决定**，本研究仅指出风险。

2. **`OBS-05 atomic 接入点是否要导出 `smart_end_metrics_test.go`？**
   - What we know: prom 接入点需 atomic.Int64 + Record* 函数；测试 assertion `atomic.LoadInt64 == 1` 后 `Record*` 调用。
   - What's unclear: 是否要在 cmd/server 启动时打印一次 "smart_end.metrics exposed at /metrics" log，便于运维知晓？
   - Recommendation: 暂不加 verbose startup log；如需则放 Phase 26+ 或后续 observability phase。

3. **`smart_end.enabled=false` 时 scheduler 是否仍订阅 taskEndedCh 作为兜底？**
   - What we know: PRD §6 写"false 时系统退回纯 EndTime 行为"。
   - What's unclear: 若运维临时 `enabled=true → false` 在重启窗口，旧 watcher 已被 Stop；新调度 watch 不读 channel；但监控代码可能误判 scheduler 死锁。
   - Recommendation: 本研究不强制；建议 PLAN-02 选"严格 if-else 守门"（不订阅）。

4. **audit log 的 OldData 应否含"前 extension_count"或仅 NewData 含"新 count"？**
   - What we know: 现有 12 处 audit 都同时给 OldData + NewData 做 diff；`audit_logs.diff_data` 列存 diff。
   - What's unclear: `extension_count: 0 → 1` 的 diff 在 audit UI 上是否值得展示？
   - Recommendation: 沿用 OldData + NewData 模式，diff 自动生成。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | build/test | ✓ | go1.25.0 windows/amd64 | — |
| SQLite :memory: | E2E 测试 | ✓ | gorm sqlite v1.6.0 + modernc | — |
| ffmpeg binary | 不需要 | n/a | — | service E2E 走 mockCoordinator，integration 不启 ffmpeg |
| Huawei HTTPS | 不需要 | n/a | — | scheduler E2E 走 fake `HuaweiStateClient`，不依赖真实 CA |
| Git | CI sync-check | ✓ | repo clean | — |
| prometheus | 不需要 | n/a | — | OBS-05 接入点仅 atomic，**不**引入 prom 库 |

**Missing dependencies with no fallback:** none（所有外部依赖有 mock 替代）

## Sources

### Primary (HIGH confidence)

- `D:/CODE/ClaudeCode/record_V2/internal/scheduler/video_scheduler.go:541-606` — `monitorTask` 现状（select timer + updateChan）+ `:609-715 completeTask` + `:1259-1294 UpdateTaskEndTime`
- `D:/CODE/ClaudeCode/record_V2/internal/recorder/activity_watcher.go:65-77` ActivitySnapshot + `:149-228 EndedCh/IsActive/Snapshot/ExtendStepMin/closeEnded`
- `D:/CODE/ClaudeCode/record_V2/internal/recorder/coordinator.go:130-217` StartRecordingWithConfig（per-task owned watcher） + `:613-653 StopRecording`
- `D:/CODE/ClaudeCode/record_V2/internal/recorder/huawei_state_client.go:20-22` HuaweiStateClient interface
- `D:/CODE/ClaudeCode/record_V2/internal/services/video_recording_task_service.go:1056-1073` `UpdateTaskStatus` 同款签名模式 + `:311-318 GetTaskByID`
- `D:/CODE/ClaudeCode/record_V2/internal/services/audit/audit_log_service.go:62-82` `RecordChangeOpts + RecordChange`
- `D:/CODE/ClaudeCode/record_V2/internal/models/audit_log.go:11-51` AuditLog model + `AuditLogData:124-128` interface{} 字段
- `D:/CODE/ClaudeCode/record_V2/internal/models/video_recording_task.go:39-44` 5 GORM 字段（Phase 23 AUDIT-01）
- `D:/CODE/ClaudeCode/record_V2/internal/config/smart_end.go:20-185` 14 字段 SmartEndConfig + Validate
- `D:/CODE/ClaudeCode/record_V2/internal/errors/errors.go:101-105` 3 sentinel（Phase 23 AUDIT-05）
- `D:/CODE/ClaudeCode/record_V2/.github/workflows/ci.yml:62-77` `go test -race -coverprofile` + golangci-lint v2.12.2
- `D:/CODE/ClaudeCode/record_V2/cmd/server/app.go:1131-1155` scheduler 构造（cmd/server 注入 videoTaskService 直接满足 TaskServiceInterface）
- `D:/CODE/ClaudeCode/record_V2/internal/scheduler/video_scheduler_test.go:138-159` mockCoordinator 模式（Phase 25 E2E 复用）

### Secondary (MEDIUM confidence)

- `PRD §5.2 + §6` (docs/plans/2026-08-05-smart-meeting-recording-end-design.md) — select 三路契约 + 14 项配置 + audit log snapshot 字段
- Phase 24 RESEARCH.md Pitfall 2 (close-once panic) + Pitfall 6 (file ticker 误触发) — 设计依据

### Tertiary (LOW confidence)

- `reflect.Select` 与 `chan struct{}` fan-in 互操作的详细语义 — 本研究推荐 Pattern 3 (fan-in goroutine) 而非 reflect，**未**实测 reflect 路径；planner 可在 PLAN-02 选其一
- `smart_end.enabled=false` 路径下是否要 `cfg.SmartEnd.Enabled` 一处守门（coordinator 已守，scheduler 是否需守）—— 推荐双守门（research 主线建议），planner 可裁剪

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 零新增依赖，全部 stdlib + 现有项目依赖
- Architecture: HIGH — `monitorTask` `select` 改造边界清晰，service 方法签名固定
- GORM 5 字段写入: HIGH — Phase 23 已交付，model 字段稳定
- audit log snapshot 序列化: HIGH — `AuditLogData.OldData/NewData` interface{} 自动 marshal 成熟
- OBS 日志字段: HIGH — zap structured logging 既有惯例，5 字段 schema 锁定
- 多 input watcher 合并: MEDIUM — fan-in goroutine 模式为 [ASSUMED] 推荐；reflect.Select 为备选
- ActivitySnapshot 2 字段扩展: HIGH — 仅在现有 struct 上加字段，无架构变更
- E2E 测试栈: HIGH — `internal/services/video_recording_task_service_test.go:18-29` `newTestDB` + `internal/scheduler/video_scheduler_test.go:138-159` `mockCoordinator` 模式已存在

**Research date:** 2026-08-06
**Valid until:** 2026-09-05（Go stdlib select / sync.Once / atomic API 稳定，无外部库版本风险）

## Research Complete

Phase 25 规划信息已齐备。建议按 4 plans 落地：(01) ActivitySnapshot 2 字段扩展 + `service.UpdateTaskExtension/MarkTaskEndedEarly` 引入 + audit 注入；(02) scheduler `monitorTask` `select` 三路 + `mergeWatchers` fan-in + CFG-03 守门；(03) `observability/smart_end_metrics.go` 原子计数器 + OBS-01..04 zap 日志；(04) Nyquist 全覆盖（scheduler 7+ scenario + service 8+ scenario + observability 3 atomic 测试）。**关键风险** 已在 Pitfall 1/2/3/4/8 标注 — planner 需 (1) EARLY-04 fan-in 选用 fan-in goroutine 而非 reflect.Select；(2) scheduler 必须实现 CFG-03 双守门（既不读 taskEndedCh 也不订阅 taskEndedCh）；(3) ActivitySnapshot `FileSizeBytes`/`FileGrowthBps` 2 字段先于 service 写入之前就绪；(4) Audit log snapshot JSON marshal 时若 `time.Time{}` 零值需要 omitempty 防 `0001-01-01` 进 DB。**禁止引入新依赖**（含 prometheus client）；仅暴露 atomic + Record* 函数即满足 OBS-05 接入点语义。
