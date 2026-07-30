---
quick_id: 260730-bc3
slug: 38-recording-5-storage-3-ppts-4-apikey-3
status: complete
completed_at: 2026-07-30
---

# Quick Task 260730-bc3: OldData 捕获 — recording/storage/ppts/apikey/notification（16 站点）

## 任务结论

在用户明确列出的 16 个 P1 变更站点接入 `audit.RecordChange(ctx, opts)`，与 m8l + mwt 已验证的 `service-layer snapshot → return (old, new, err) → handler.RecordChange(c.Request.Context(), opts)` 模式完全一致。每个站点都遵循：

```
service.First(&model)  →  oldSnapshot = *model (value-copy 拷贝)
service.mutate(model → newModel)
return &oldSnapshot, &newModel, nil
                              ↓
handler:  auditService.RecordChange(c.Request.Context(), RecordChangeOpts{
              Action: ..., Module: ..., Resource: fmt.Sprintf("...:%d", id),
              ResourceID: &id, OldData: oldModel, NewData: newModel })
                              ↓
LogOperation → Sanitizer(oldData) → Sanitizer(newData) → async queue → audit_logs
```

不修改 `RecordChange` helper 签名，不引入新 helper 或 service 方法 ctx 透传，所有 Sanitizer 继承自 LogOperation 管道对 OldData/NewData 生效。

**关键交付**：

| 维度 | 数据 |
|------|------|
| 原子提交数 | 3 |
| 涉及代码文件 | 13（11 改造 + 2 test 适配） |
| 代码行变化 | +499 / −165 |
| RecordChange 出现次数 | 16（5 recording + 3 storage + 4 ppts + 3 apikey + 1 notification） |
| Build | `go build ./...` ✅ |
| Tests | `go test -count=1 ./internal/services/... ./internal/handlers/...` ✅ |
| Frontend 文件触动 | 0 |

## 提交清单

| Commit | 标题 | 改动文件数 |
|--------|------|-----------|
| `7c30874` | feat(audit): wire recording OldData capture (5 sites) | 3 |
| `a111f01` | feat(audit): wire storage + apikey OldData capture (6 sites) | 6 |
| `c51f549` | feat(audit): wire ppts + notification OldData capture (5 sites) | 7 |

## 16 个接入点（按模块分组）

| # | 文件:方法 | 模式 | Action | Module | OldData | NewData | Resource |
|---|----------|------|--------|--------|---------|---------|----------|
| 1 | `video_recording_task_handler.go:UpdateTask` | Update (old, new) | `ActionUpdate` | `ModuleTask` | `*oldTask` | `*newTask` | `task:<id>` |
| 2 | `video_recording_task_handler.go:DeleteTask` | Delete (old only) | `ActionDelete` | `ModuleTask` | `*oldTask` | `nil` | `task:<id>` |
| 3 | `video_recording_task_handler.go:BatchDeleteTasks` | Batch (option b: 1 row, slice OldData) | `"batch_delete"` | `ModuleTask` | `[]VideoRecordingTask`（初始 `id IN ?` 查询的快照） | `nil` | `task:<id1>,<id2>` |
| 4 | `video_recording_task_handler.go:StartTask` | Update (old, new) | `"start"` | `ModuleTask` | `*oldTask` | `*newTask`（scheduler dispatch 后 reload） | `task:<id>` |
| 5 | `video_recording_task_handler.go:StopTask` | Update (old, new) | `"stop"` | `ModuleTask` | `*oldTask`（CancelTaskExecution 前快照） | `*newTask`（converting/cancelled） | `task:<id>` |
| 6 | `file_handler.go:Upload` | Create (无 prior) | `ActionCreate` | `ModuleStorage` | `nil` | `*FileUploadResult` | `storage:<fileID>` |
| 7 | `file_handler.go:Delete` | Delete (old only) | `ActionDelete` | `ModuleStorage` | `*UploadedFile`（soft-delete 前快照） | `nil` | `storage:<fileID>` |
| 8 | `file_handler.go:Share` | Update (old, new) | `"share"` | `ModuleStorage` | `*FileShare`（同 file/shared_by 最近一条；nil 表示无 prior） | `*FileShare`（含生成 token，Sanitizer 自动 redact） | `storage:<fileID>` |
| 9 | `ppt_handler.go:DeletePPT` | Delete (old only) | `ActionDelete` | `ModulePPT` | `*PPTFile` | `nil` | `ppt:<id>` |
| 10 | `ppt_handler.go:DeleteSlidesHandler` | Update (old, new) | `"delete_slides"` | `ModulePPT` | `*PPTFile`（backup/cache 之前） | `*PPTFile`（reload 后含 `PageCount`/`DeletedSlides`/`EditHistory`/`BackupPath`） | `ppt:<id>` |
| 11 | `ppt_handler.go:RollbackHandler` | Update (old, new) | `"rollback"` | `ModulePPT` | `*PPTFile`（backup copy 前） | `*PPTFile`（reload 后 `backup_path` 清空、`deleted_slides` 重置） | `ppt:<id>` |
| 12 | `ppt_handler.go:ReorderSlidesHandler` | Update (old, new) | `"reorder_slides"`（**非**中间件的 `"reorder"`） | `ModulePPT` | `*PPTFile`（backup/reorder 前） | `*PPTFile`（reload 后新 `BackupPath`） | `ppt:<id>` |
| 13 | `apikey_handler.go:UpdateAPIKey` | Update (old, new) | `ActionUpdate` | `ModuleAPIKey` | `*APIKey`（SetScopes/SetIPWhitelist 前） | `*APIKey` | `apikey:<id>` |
| 14 | `apikey_handler.go:DeleteAPIKey` | Delete (old only) | `ActionDelete` | `ModuleAPIKey` | `*APIKey` | `nil` | `apikey:<id>` |
| 15 | `apikey_handler.go:ToggleAPIKeyStatus` | Update (old, new) | `"toggle"` | `ModuleAPIKey` | `*APIKey` | `*APIKey` | `apikey:<id>` |
| 16 | `notification_handler.go:UpdateUserSetting` | Update (old, new) | `ActionUpdate` | `ModuleNotification` | `*UserNotificationSetting`（无 row 时使用默认设置快照） | `*UserNotificationSetting`（reload 后） | `notification_setting:<userID>` |

## 关键设计决策

### 1. service 返回 (old, new) — 与 m8l/mwt 严格一致

| Service 方法 | 改前 | 改后 |
|-------------|------|------|
| `VideoRecordingTaskService.UpdateTask` | `(*VideoRecordingTask, error)` | `(*VideoRecordingTask /* old */, *VideoRecordingTask /* new */, error)` |
| `VideoRecordingTaskService.DeleteTask` | `error` | `(*VideoRecordingTask /* old */, error)` |
| `VideoRecordingTaskService.BatchDeleteTasks` | `(*BatchDeleteTasksResult, error)` | `([]VideoRecordingTask /* old */, *BatchDeleteTasksResult, error)` |
| `VideoRecordingTaskService.StartTask` | `(*VideoRecordingTask, error)` | `(*VideoRecordingTask /* old */, *VideoRecordingTask /* new */, error)` |
| `VideoRecordingTaskService.StopTask` | `(*VideoRecordingTask, error)` | `(*VideoRecordingTask /* old */, *VideoRecordingTask /* new */, error)` |
| `FileService.Delete` | `error` | `(*UploadedFile /* old */, error)` |
| `FileService.ShareFile` | `(string, error)` | `(*FileShare /* old */, *FileShare /* new */, string, error)` |
| `PPTFileService.DeletePPTFile` | `error` | `(*PPTFile /* old */, error)` |
| `PPTEditorService.DeleteSlides` | `error` | `(*PPTFile /* old */, *PPTFile /* new */, error)` |
| `PPTEditorService.Rollback` | `error` | `(*PPTFile /* old */, *PPTFile /* new */, error)` |
| `PPTEditorService.ReorderSlides` | `([]int, error)` | `([]int, *PPTFile /* old */, *PPTFile /* new */, error)` |
| `APIKeyService.UpdateAPIKey` | `(*APIKey, error)` | `(*APIKey /* old */, *APIKey /* new */, error)` |
| `APIKeyService.DeleteAPIKey` | `error` | `(*APIKey /* old */, error)` |
| `APIKeyService.ToggleAPIKeyStatus` | `(*APIKey, error)` | `(*APIKey /* old */, *APIKey /* new */, error)` |
| `NotificationService.UpdateUserSetting` | `error` | `(*UserNotificationSetting /* old */, *UserNotificationSetting /* new */, error)` |

每处快照都在**第一个 mutation 之前**取 `oldSnapshot := *model` 或 `oldTask := task`（value-copy），保证后续 service 内的 mutation 不影响 OldData。`BatchDeleteTasks` 在最开始的 `Where("id IN ?").Find()` 阶段就取 `oldTasks`，含 processing 状态跳过的记录（forensic 完整性）。

### 2. StartTask 的 NewData 取 scheduler dispatch 后 reload

`StartTask` 调用 `s.scheduler.ExecuteTask(id)` 后再次 `s.db.First(&task, id)` reload，`NewData` 反映 scheduler 异步修改的真实状态（status/recording_file 等可能已被 scheduler 改写）。这是 plan §任务 1 action 的明确要求（"NewData obtained from the post-dispatch state actually observable from the existing asynchronous scheduler path"），**不**阻塞 scheduler 完成，仅 reload 同步 DB 可见状态。

### 3. BatchDeleteTasks 选 option (b) — 单条 RecordChange

OldData = `[]models.VideoRecordingTask`（含 processing 跳过的全部 ID，forensic 完整性），Resource = `task:<id1>,<id2>`，ResourceID = `&zero`。一行审计 = 一次用户操作，与 audit_logs Resource 单值字段模型契合；与 mwt §2.2 BatchDeleteFiles 模式一致。

### 4. Storage ShareFile 的 oldShare 查询策略

`ShareFile` 返回 `(oldShare, newShare, shareURL, error)`：
- `oldShare` 查询 `WHERE file_id = ? AND shared_by = ?` 最近一条（`Order("created_at DESC")`），`gorm.ErrRecordNotFound` 映射为 nil OldData，其他错误透传
- `newShare` 是新建的 share，包含生成的 `ShareToken`（Sanitizer 自动 redact 为 `***`）
- `shareURL` 仍由 `s.getServerURL()` 派生，handler 的 `share_url` 响应键保持不变

### 5. PPT snapshot 字段不引入新 model 字段

`models.PPTFile` 没有 `Slides` 字段（plan §源对齐映射说明），其持久化的 slide 状态由 `PageCount`、`DeletedSlides`、`EditHistory`、`BackupPath` 表达。每处服务的 `oldPPT := pptFile` value-copy 包含全部字段，handler 的 `NewData` 直接用 service reload 返回的 `*newPPT`。

### 6. Notification 旧值用 default snapshot

`UpdateUserSetting` 当 DB 无 row 时 `oldSetting = *s.getDefaultSetting(userID)`（与 `getDefaultSetting` 一致），不阻塞审计 OldData capture。

### 7. Sanitizer 自动覆盖

| 字段 / 模块 | Sanitizer 规则 |
|------------|---------------|
| `APIKey.Key` / `KeyHash` | 自动 redact 为 `***` |
| `FileShare.Password` / `ShareToken` | 自动 redact 为 `***` |
| PPT / Recording / Notification / Storage | 不含敏感字段，无需手动 redact |

未引入任何手动 redact；所有敏感字段由 `LogOperation → s.sanitizer.Sanitize(oldData/newData)` 自动处理。

### 8. 失败处理

RecordChange 失败仅 `h.logger.Warn`，不阻断业务响应（与 m8l §3.6 UserHandler.UpdateUser、mwt §3 模式一致）。所有 16 处调用都遵循此 guard。

### 9. handler 新增 `auditService *audit.AuditLogService` 字段 + 构造函数注入

| Handler | 新增字段 | app.go wiring 变化 |
|---------|----------|-------------------|
| `VideoRecordingTaskHandler` | `auditService` | `NewVideoRecordingTaskHandler(a.videoTaskService, auditService, a.logger, a.config)` |
| `FileHandler` | `auditService` | `NewFileHandler(fileService, auditService, a.logger)` |
| `PPThandler` | `auditService` | `NewPPThandler(..., auditService, a.logger)` |
| `APIKeyHandler` | `auditService` | `NewAPIKeyHandler(apikeyService, auditService, a.logger)` |
| `NotificationHandler` | `auditService` | `NewNotificationHandler(notificationService, auditService)` |

app.go 中 `auditService` 提前到 `FileHandler` 创建之前（mwt 已有 UserHandler/RoleHandler/InputConfigHandler/VideoFileHandler/SystemHandler 的 auditService 注入，但 auditService 在更下方的位置；本次需要提前以支持 `FileHandler` 也接收 auditService）。

### 10. Module 名锁定（与中间件 auditOp 解耦）

| 端点 | 中间件 auditOp | 本次 RecordChange 用 |
|------|---------------|---------------------|
| `/api/v1/recordings/...` | `ModuleRecording` | `ModuleTask` |
| `/api/v1/storage/...` | `ModuleFile` | `ModuleStorage` |
| `/api/v1/ppts/...` | `ModulePPT` | `ModulePPT` |
| `/api/v1/apikeys/...` | `ModuleAPIKey` | `ModuleAPIKey` |
| `/api/v1/notifications/...` | `ModuleNotification` | `ModuleNotification` |

`/api/v1/recordings/PUT` 等路由的中间件 auditOp 仍输出 `ModuleRecording` HTTP 上下文行；新 RecordChange 输出 `ModuleTask` 业务上下文行。两者共存不冲突（同请求两条 audit_logs 行）。

### 11. Action 名 — 锁定字符串

| 端点 | Action 字面量 |
|------|--------------|
| recording update/delete | `models.ActionUpdate` / `models.ActionDelete` |
| recording start/stop/batch | `"start"` / `"stop"` / `"batch_delete"` |
| storage upload/delete | `models.ActionCreate` / `models.ActionDelete` |
| storage share | `"share"` |
| ppts delete | `models.ActionDelete` |
| ppts slide edit | `"delete_slides"` / `"rollback"` / `"reorder_slides"`（**非**中间件的 `"reorder"`） |
| apikey update/delete | `models.ActionUpdate` / `models.ActionDelete` |
| apikey toggle | `"toggle"` |
| notification setting update | `models.ActionUpdate` |

## 关键执行偏差（deviations）

### 1. worktree 基底不一致（auto-fixed via reset）

工作分支 `worktree-agent-a8824b8807f898d2d` 起始 HEAD = `8cdba2e`（mwt 之前的前端修复 commit），但 `EXPECTED_BASE` = `1c90af6`（bc3 预派单 plan commit）。merge-base 检查触发后，对 worktree 进行 `git reset --hard 1c90af6`（不会破坏 main repo，因为 Step 1 已确认 HEAD 在 `worktree-agent-*` 命名空间），保留 Task 1 已编辑的 3 个文件用 `git checkout stash@{0} -- <files>` 恢复（避免 `git stash pop` 跨 worktree 全局副作用，遵守 destructive_git_prohibition §stash），然后 drop stash。后续编辑全部基于正确的 1c90af6 基底进行。

### 2. app.go NewFileHandler 创建顺序依赖

`FileHandler` 的 auditService 注入需要在 `auditService` 之后创建。原 1c90af6 的 app.go 顺序是 `FileHandler → auditService`，与新需求冲突。修复：把 `auditService` 创建提到 `FileHandler` 之前（紧跟 `inputConfigService`），后续 UserHandler、NotificationHandler 等依赖 auditService 的 handler 顺序不变。原有 mwt 接入的 UserHandler/RoleHandler/InputConfigHandler/VideoFileHandler/SystemHandler 完全不受影响。

### 3. test 适配（expected, 8 处）

`internal/services/storage/file_service_test.go` 中 `service.Delete` / `service.ShareFile` 调用从 1/2 值改为 2/4 值，并新增 `oldFile.ID`/`oldShare`/`newShare` 断言。`internal/services/ppt_editor_service_test.go` 中 `service.DeleteSlides` / `service.Rollback` 调用从 1 值改为 3 值，并新增 `oldPPT.PageCount == 5` / `oldPPT.BackupPath == ""` 等 pre-edit snapshot 断言。

## 16 站点自我验证

### Service 签名变化（15 处）

详见上文 §1. service 返回 (old, new) 表格。

### Handler 字段变化

5 个 handler（VideoRecordingTaskHandler、FileHandler、PPThandler、APIKeyHandler、NotificationHandler）新增 `auditService *audit.AuditLogService` 字段 + 构造函数参数。

### app.go 构造调用变化

```go
// 改前（1c90af6 基底）
VideoTask:     handlers.NewVideoRecordingTaskHandler(a.videoTaskService, a.logger, a.config),
File:          fileHandler,  // fileHandler := handlers.NewFileHandler(fileService, a.logger)
Notification:  notificationHandler,  // notificationHandler := handlers.NewNotificationHandler(notificationService)
APIKey:        apikeyHandler,  // apikeyHandler := handlers.NewAPIKeyHandler(apikeyService, a.logger)
PPT:           handlers.NewPPThandler(pptFileService, a.slideCacheService, a.pptMergeService, a.videoFileService, a.pptEditorService, a.frameCaptureService, a.logger),

// 改后（bc3 提交后）
VideoTask:     handlers.NewVideoRecordingTaskHandler(a.videoTaskService, auditService, a.logger, a.config),
File:          fileHandler,  // fileHandler := handlers.NewFileHandler(fileService, auditService, a.logger)
Notification:  notificationHandler,  // notificationHandler := handlers.NewNotificationHandler(notificationService, auditService)
APIKey:        apikeyHandler,  // apikeyHandler := handlers.NewAPIKeyHandler(apikeyService, auditService, a.logger)
PPT:           handlers.NewPPThandler(pptFileService, a.slideCacheService, a.pptMergeService, a.videoFileService, a.pptEditorService, a.frameCaptureService, auditService, a.logger),
```

## Self-check 完成清单

- [x] RecordChange helper 签名未改（仍是 `func (s *AuditLogService) RecordChange(ctx context.Context, opts RecordChangeOpts) error`）
- [x] 16 个站点全部接入（recording ×5 + storage ×3 + ppts ×4 + apikey ×3 + notification ×1）
- [x] `cmd/server/app.go` 中 5 个 handler 构造调用更新，auditService 提前到 FileHandler 之前
- [x] APIKey.Key / KeyHash / FileShare.Password / ShareToken 字段经 Sanitizer 脱敏（无需手动 redact，Sanitizer 自动覆盖）
- [x] BatchDeleteTasks 选 option (b) — 单条 RecordChange，Resource 包含逗号拼接的所有 ID
- [x] StartTask NewData 是 scheduler dispatch 后 reload 的状态（不阻塞 scheduler）
- [x] StopTask oldTask 在 CancelTaskExecution 之前快照
- [x] ReorderSlides action 用 `"reorder_slides"` 而非中间件的 `"reorder"`
- [x] Notification.UpdateUserSetting 旧值用 default setting snapshot（无 row 时）
- [x] Sanitizer 继承自 LogOperation 管道，对 OldData/NewData 生效
- [x] 现有 auditOp 中间件路由不受影响（recording/storage/ppts/apikey/notification 的写路由仍走 auditOp，新增 RecordChange 是同请求第二条审计行）
- [x] `go build ./...` 通过
- [x] handlers + services 测试通过（含 8 处 service 新返回值适配 + old/new snapshot 断言）
- [x] 3 次原子提交（recording / storage+apikey / ppts+notification，与 plan §约束建议一致）
- [x] 23 个 frontend 未提交文件未触动（`git diff 1c90af6 --name-only -- frontend` 输出为空）
- [x] 旧 audit_logs 行（含 NULL OldData）继续可读（Sanitizer 对 nil 输入直接返回 nil）
- [x] go.mod / go.sum / audit_log_service.go / audit_log.go 未修改（`git diff 1c90af6 -- go.mod go.sum internal/services/audit/audit_log_service.go internal/models/audit_log.go` 输出为空）

## 剩余 ~22 站点接入计划

**未在本任务覆盖**的 P1/P2 站点（按风险递减）：

| 批次 | 模块 | 站点数 | 优先级 | 说明 |
|------|------|--------|--------|------|
| P1 | admin (UpdateAuthConfig) | 1 | P1 | admin 鉴权配置更新；handler 在 admin.go，与 m8l/mwt 模式一致 |
| P1 | system (ClearFiles) | 1 | P1 | 清空文件数据库（高危批量写操作） |
| P1 | video_file (RenameFile) | 1 | P1 | 视频文件重命名 |
| P1 | video_file (BatchDownloadFiles) | 1 | P1 | 批量下载（数据导出，mwt 已有中间件 auditOp ActionExport） |
| P2 | input-config (TestConnection) | 1 | P2 | 跳过（无状态变更） |
| P2 | user (ToggleUserStatus) | 1 | P2 | 用户状态切换（与 apikey ToggleAPIKeyStatus 模式一致） |
| P2 | user (ResetPassword) | 1 | P2 | m8l 已用 OldData snapshot map（不含 PasswordHash） |
| P2 | user (UpdateCurrentProfile) | 1 | P2 | 当前用户资料更新 |
| P2 | video_task (CancelTask / RetryTask / ClearStuckTasks) | 3 | P2 | 任务取消/重试/清理卡住任务 |
| P2 | video_task (GetConversionStatus / RetryConversion / GetHLSPreview) | 3 | P2 | 只读或状态查询 |
| P2 | transcription (SubmitTranscription / SubmitBatchTranscription) | 2 | P2 | 转录提交 |
| P2 | split (SubmitSplit / GenerateSnapshot) | 2 | P2 | 分割/快照 |
| P2 | system (TestConnection) | 1 | P2 | 系统测试连接 |

**接入模板**（已固化到 RecordChange），每站点成本：~10 行（service 返回 old + handler 一处 RecordChange 调用 + app.go 一行构造调用）。剩余 ~22 个站点分两批：
- P1 批量（约 4 站点）：1 个 quick task
- P2 批量（约 13 站点）：1 个 quick task

## 相关文档

- `.planning/quick/260730-bc3-38-recording-5-storage-3-ppts-4-apikey-3/260730-bc3-PLAN.md` — 完整计划
- `.planning/quick/260729-mwt-input-config-system-file-olddata-6/260729-mwt-SUMMARY.md` — 前置任务（input-config/system/file 6 站点）
- `.planning/quick/260729-m8l-olddata-update-delete/260729-m8l-SUMMARY.md` — 前置任务（RecordChange helper + user/role 6 站点）
- `internal/services/audit/audit_log_service.go` — RecordChange + LogOperation 实现位置
- `internal/services/audit/sanitizer.go` — 自动脱敏管道（Key/KeyHash/ShareToken/Password）