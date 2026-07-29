---
quick_id: 260729-mwt
slug: input-config-system-file-olddata-6
status: complete
completed_at: 2026-07-29
---

# Quick Task 260729-mwt: OldData 捕获 — input-config / system / file (6 站点)

## 任务结论

在 6 个高危 P0 update/delete 站点接入 `audit.RecordChange(ctx, opts)`，与 m8l 任务形成的 helper + user/role 6 站点模式完全一致。每个站点都遵循 `service-layer snapshot → return (old, new, err) → handler.RecordChange(c.Request.Context(), opts)` 流程，自动复用 Sanitizer 管道对密码/token 脱敏。

**关键交付**：

| 维度 | 数据 |
|------|------|
| 原子提交数 | 2 |
| 涉及代码文件 | 7（6 改造 + 1 test 适配） |
| 代码行变化 | +200 / −48 |
| RecordChange 出现次数 | 21（m8l 9 + mwt 12） |
| Build | `go build ./...` ✅ |
| Tests | `go test -count=1 ./internal/services/audit/... ./internal/handlers/ ./internal/services/` ✅ |
| Frontend 文件触动 | 0 |

## 提交清单

| Commit | 标题 | 改动文件 |
|--------|------|---------|
| `3176ff8` | feat(audit): wire input-config OldData capture (Create/Update/Delete) | internal/handlers/input_config_handler.go, internal/services/input_config_service.go, cmd/server/app.go |
| `2c341b5` | feat(audit): wire system + file OldData capture (UpdateConfig, DeleteFile, BatchDeleteFiles) | internal/handlers/system_handler.go, internal/handlers/video_file_handler.go, internal/services/video_file_service.go, internal/services/video_file_service_test.go, cmd/server/app.go |

## 6 个接入点

| # | 文件:方法 | 模式 | Action | Module | OldData | NewData | Resource |
|---|----------|------|--------|--------|---------|---------|----------|
| 1 | `input_config_handler.go:CreateConfig` | Create (无 prior) | `ActionCreate` | `ModuleInputConfig` | `nil` | 新 `*models.InputConfig` | `input_config:%d` |
| 2 | `input_config_handler.go:UpdateConfig` | Update (old, new) | `ActionUpdate` | `ModuleInputConfig` | 旧 `*models.InputConfig` | 新 `*models.InputConfig` | `input_config:%d` |
| 3 | `input_config_handler.go:DeleteConfig` | Delete (old only) | `ActionDelete` | `ModuleInputConfig` | 旧 `*models.InputConfig` | `nil` | `input_config:%d` |
| 4 | `system_handler.go:UpdateConfig` | Partial sub-key snapshot | `"update_config"` | `ModuleSystem` | `map[string]interface{}{...}` 仅已变 sub-key | 同上 | `system_config:<key1>,<key2>` |
| 5 | `video_file_handler.go:DeleteFile` | Delete (old only) | `ActionDelete` | `ModuleFile` | 旧 `*models.VideoFile` | `nil` | `video_file:%d` |
| 6 | `video_file_handler.go:BatchDeleteFiles` | Batch (option b: 1 row, 切片 OldData) | `"batch_delete"` | `ModuleFile` | `[]models.VideoFile`（请求 IDs 全部，含 skipped） | `nil` | `video_file:%d1,%d2,...` |

## 关键设计决策

### 1. service 返回 (old, new) — 与 m8l 一致

`InputConfigService.UpdateConfig` / `VideoFileService.DeleteFile` 改为返回 `(*old, *new, error)` 或 `(*old, error)`，handler 调 `c.Request.Context()` 走 RecordChange，**不引入 ctx 透传**。

### 2. system.UpdateConfig 选用 partial sub-key snapshot

只 snapshot 实际 mutate `h.config` 的 4 个 sub-key（LogLevel/LogFormat/LogOutput/MaxDiskUsage），restart-required 字段（RecordingsPath/HLSPath/TempPath/FFmpegPath/FFprobePath）仅记录到 `changes` 字符串，**不**进 OldData（内存未变更，pending restart）。Resource 字段动态拼接为 `system_config:logging.level,storage.max_disk_usage` 等，便于 forensic 定位。

`len(changedKeys) > 0` 才 emit RecordChange，纯路径类请求不写审计行。

### 3. BatchDeleteFiles 选 option (b)：单条 RecordChange

OldData = `[]models.VideoFile`（含 processing 跳过的 ID，forensic 完整性），Resource = `video_file:1,2,3`。一行审计 = 一次用户操作，与 audit_logs 表 Resource 单值字段模型契合；与中间件 AuditOperation 已有 `Action="batch_delete"` 模式一致（app.go:829）。

### 4. Sanitizer 自动覆盖

- `input_config.Password` / `StreamPassword` 字段由 `LogOperation → s.sanitizer.Sanitize(oldData/newData)` 自动 redact 为 `***`
- `system` / `file` 模块不含敏感字段
- `ResetPassword`-style PasswordHash 排除规则不适用于本批 6 站点

### 5. 失败处理

RecordChange 失败仅 `h.logger.Warn`，不阻断业务响应（与 m8l §3.6 UserHandler.UpdateUser 一致）。

## 6 站点自我验证

### Service 签名变化

| Service 方法 | 改前 | 改后 |
|-------------|------|------|
| `InputConfigService.UpdateConfig` | `(*models.InputConfig, error)` | `(*models.InputConfig /* old */, *models.InputConfig /* new */, error)` |
| `InputConfigService.DeleteConfig` | `error` | `(*models.InputConfig /* old */, error)` |
| `VideoFileService.DeleteFile` | `error` | `(*models.VideoFile /* old */, error)` |
| `VideoFileService.BatchDeleteFiles` | `(*BatchDeleteFilesResult, error)` | `([]models.VideoFile /* old */, *BatchDeleteFilesResult, error)` |

### Handler 字段变化

| Handler | 新增字段 |
|---------|----------|
| `InputConfigHandler` | `auditService *audit.AuditLogService` |
| `SystemHandler` | `auditService *audit.AuditLogService` |
| `VideoFileHandler` | `auditService *audit.AuditLogService` |

### app.go 构造调用变化

```go
// 改前
InputConfig: handlers.NewInputConfigHandler(inputConfigService, a.logger, usbScanner),
VideoFile:   handlers.NewVideoFileHandler(a.videoFileService, a.logger),
System:      handlers.NewSystemHandler(a.db, a.logger, a.config),

// 改后
InputConfig: handlers.NewInputConfigHandler(inputConfigService, auditService, a.logger, usbScanner),
VideoFile:   handlers.NewVideoFileHandler(a.videoFileService, auditService, a.logger),
System:      handlers.NewSystemHandler(a.db, auditService, a.logger, a.config),
```

## Self-check 完成清单

- [x] RecordChange helper 签名未改（仍是 `func (s *AuditLogService) RecordChange(ctx, opts) error`）
- [x] 6 个站点全部接入（input_config ×3 + system ×1 + file ×2）
- [x] `cmd/server/app.go` 3 个 handler 构造调用更新
- [x] input-config 的 Password / StreamPassword 字段经 Sanitizer 脱敏（无需手动 redact，Sanitizer 自动覆盖）
- [x] BatchDeleteFiles 选 option (b) — 单条 RecordChange，Resource 包含逗号拼接的所有 ID
- [x] system.UpdateConfig OldData/NewData 只含实际 mutate 的 4 个 sub-keys（不包含 restart-required 字段）
- [x] system.UpdateConfig 仅在 `len(changedKeys) > 0` 时 emit RecordChange
- [x] Sanitizer 继承自 LogOperation 管道，对 OldData 生效
- [x] 现有 audit middleware 路径不受影响（input-config / system / file 的 create/update/delete 路由仍走 auditOp 中间件，新增的 RecordChange 是同请求第二条审计行）
- [x] `go build ./...` 通过
- [x] `go test -count=1 ./internal/services/audit/... ./internal/handlers/ ./internal/services/` 通过（含 4 处 `service.DeleteFile` 测试签名适配）
- [x] 2 次原子提交
- [x] 23 个 frontend 未提交文件未触动
- [x] 旧 audit_logs 行（含 NULL OldData）继续可读（Sanitizer 对 nil 输入直接返回 nil）

## 后续任务 — 剩余 ~21 站点接入计划

**未在本任务覆盖**的 update/delete 站点按风险分批：

| 批次 | 模块 | 站点数 | 优先级 | 建议任务 |
|------|------|--------|--------|----------|
| 中危 | recording (UpdateTask/DeleteTask/BatchDeleteTasks/StartTask/StopTask/CancelTask) | 6 | P1 | 下一 quick task |
| 中危 | storage (Upload/Delete/Share) | 3 | P1 | 批量 |
| 中危 | ppts (DeletePPT/DeleteSlides/Rollback/ReorderSlides) | 4 | P1 | 批量 |
| 中危 | apikey (UpdateAPIKey/DeleteAPIKey/ToggleAPIKeyStatus) | 3 | P1 | 批量 |
| 中危 | notification (UpdateUserSetting) | 1 | P1 | 批量 |
| 中危 | admin (UpdateAuthConfig) | 1 | P1 | 批量 |
| 中危 | system (ClearFiles) | 1 | P1 | 批量 |
| P2 | input-config (TestConnection) | 1 | P2 | 跳过（无状态变更） |

**接入模板**（已固化到 RecordChange），每站点成本：~10 行（service 返回 old + handler 一处 RecordChange 调用 + app.go 一行构造调用）。批量接入 ~19 个剩余站点预计 1 个 quick task。

## 相关文档

- `.planning/quick/260729-mwt-input-config-system-file-olddata-6/260729-mwt-PLAN.md` — 完整计划
- `.planning/quick/260729-m8l-olddata-update-delete/260729-m8l-SUMMARY.md` — 前置任务（RecordChange helper + user/role 6 站点）
- `.planning/quick/260729-m8l-olddata-update-delete/260729-m8l-PLAN.md` — 模板参考
- `internal/services/audit/audit_log_service.go` — RecordChange + LogOperation 实现位置
