---
phase: quick
plan: 260729-mwt
subsystem: audit
type: execute
wave: 1
depends_on: ["260729-m8l"]
files_modified:
  - internal/handlers/input_config_handler.go
  - internal/services/input_config_service.go
  - internal/handlers/system_handler.go
  - internal/handlers/video_file_handler.go
  - internal/services/video_file_service.go
  - cmd/server/app.go
autonomous: true
tags: [audit, old-data, update, delete, batch-delete, sanitization, backend, input-config, system, file]
requires:
  - "audit.RecordChange(ctx, opts) helper exists (task 260729-m8l)"
  - "audit middleware active on input-config / system / file write routes (task 260729-lr4)"
  - "AuditLogService.LogOperation sanitizer pipeline covers passwords/tokens"
  - "UserHandler/RoleHandler prove the (old, new, err) return pattern (task 260729-m8l)"
provides:
  - "input-config (Create/Update/Delete) OldData capture wired to RecordChange"
  - "system (UpdateConfig) OldData capture with sub-key partial-update snapshot"
  - "video file (DeleteFile/BatchDeleteFiles) OldData capture wired to RecordChange"
  - "Rollout template for remaining high-risk update/delete sites"
affects:
  - "internal/handlers/input_config_handler.go (CreateConfig/UpdateConfig/DeleteConfig + audit field)"
  - "internal/services/input_config_service.go (UpdateConfig/DeleteConfig return old)"
  - "internal/handlers/system_handler.go (UpdateConfig + audit field)"
  - "internal/handlers/video_file_handler.go (DeleteFile/BatchDeleteFiles + audit field)"
  - "internal/services/video_file_service.go (DeleteFile return old, BatchDeleteFiles return old[])"
  - "cmd/server/app.go (wire auditService to 3 handlers)"
tech-stack:
  added: []
  patterns:
    - "follow m8l pattern exactly — service-layer snapshot → (old, new, err) return → handler.RecordChange"
    - "BatchDeleteFiles: option (b) — single RecordChange with Resource = comma-joined IDs and OldData = []models.VideoFile"
    - "system.UpdateConfig: snapshot only actually-mutated sub-keys (LogLevel/LogFormat/LogOutput/MaxDiskUsage), not restart-required ones"
key-files:
  created:
    - .planning/quick/260729-mwt-input-config-system-file-olddata-6/260729-mwt-PLAN.md
    - .planning/quick/260729-mwt-input-config-system-file-olddata-6/260729-mwt-SUMMARY.md
  modified:
    - internal/handlers/input_config_handler.go
    - internal/services/input_config_service.go
    - internal/handlers/system_handler.go
    - internal/handlers/video_file_handler.go
    - internal/services/video_file_service.go
    - cmd/server/app.go
decisions:
  - "Follow m8l pattern exactly: service returns (old, new, err) and handler calls auditService.RecordChange(c.Request.Context(), opts). No new helper code."
  - "CreateConfig OldData is nil (insert, no prior state). NewData is the new InputConfig model. Use models.ActionCreate."
  - "UpdateConfig OldData is the pre-mutation model, NewData is the post-mutation model. Use models.ActionUpdate."
  - "DeleteConfig OldData is the pre-delete model, NewData is nil. Use models.ActionDelete."
  - "system.UpdateConfig OldData/NewData are map[string]interface{}{...} containing only ACTUALLY-MUTATED sub-keys (LogLevel/LogFormat/LogOutput/MaxDiskUsage); Resource is system_config:<subkey> comma-joined for forensic clarity. Restart-required fields (RecordingsPath/HLSPath/TempPath/FFmpegPath/FFprobePath) are NOT mutated in-memory — they have no OldData to snapshot, and recording them as 'requested but pending restart' is a different concern (out of scope)."
  - "DeleteFile OldData is the pre-delete VideoFile model, NewData is nil. Use models.ActionDelete. Module is models.ModuleFile (matches existing auditOp registration in app.go:833)."
  - "BatchDeleteFiles: choose option (b) — single RecordChange with OldData = []models.VideoFile of all IDs that were requested (even skipped processing ones — for forensic completeness, the audit row shows what was ASKED to delete), NewData = nil, Resource = 'video_file:<comma-joined-ids>'. Use Action 'batch_delete' (matches existing auditOp string in app.go:829). This keeps the audit table's single-Resource model intact and groups the operation as one logical event."
  - "Sanitizer auto-covers Password / StreamPassword (input_config), no explicit redaction needed. ResetPassword-style PasswordHash exclusion NOT needed for these sites (no password changes)."
  - "Do NOT modify RecordChange helper signature. Do NOT change service method names — only return signatures. Do NOT introduce new audit row formats."
  - "Add auditService *audit.AuditLogService field to 3 handlers (InputConfigHandler, SystemHandler, VideoFileHandler) and wire in cmd/server/app.go (auditService is already constructed at line 580)."
must_haves:
  truths:
    - "PUT /api/v1/input-configs/{id} populates audit_logs.old_data with pre-mutation config (sans sanitized Password/StreamPassword)"
    - "DELETE /api/v1/input-configs/{id} populates audit_logs.old_data with pre-delete config, new_data is null"
    - "PUT /api/v1/system/config populates audit_logs.old_data/new_data with only the actually-mutated sub-keys (LogLevel/LogFormat/LogOutput/MaxDiskUsage), resource field is 'system_config:<comma-joined-keys>'"
    - "DELETE /api/v1/files/{id} populates audit_logs.old_data with the pre-delete VideoFile record"
    - "DELETE /api/v1/files/batch emits exactly one audit_logs row per request, old_data = []VideoFile of requested IDs (including processing-skipped), resource = 'video_file:<comma-joined-ids>'"
    - "POST /api/v1/input-configs populates audit_logs.new_data with the new config, old_data is null"
    - "Each RecordChange call failure is logged via h.logger.Warn but does NOT block the business response (matches m8l pattern)"
  artifacts:
    - path: "internal/handlers/input_config_handler.go"
      provides: "InputConfigHandler with auditService field + RecordChange calls in CreateConfig/UpdateConfig/DeleteConfig"
      contains: "auditService"
    - path: "internal/services/input_config_service.go"
      provides: "UpdateConfig returns (oldConfig, newConfig, error); DeleteConfig returns (oldConfig, error)"
      exports: ["UpdateConfig", "DeleteConfig"]
    - path: "internal/handlers/system_handler.go"
      provides: "SystemHandler with auditService field + RecordChange call in UpdateConfig with partial sub-key snapshot"
      contains: "auditService"
    - path: "internal/handlers/video_file_handler.go"
      provides: "VideoFileHandler with auditService field + RecordChange calls in DeleteFile/BatchDeleteFiles"
      contains: "auditService"
    - path: "internal/services/video_file_service.go"
      provides: "DeleteFile returns (oldFile, error); BatchDeleteFiles returns (oldFiles []models.VideoFile, result, error)"
      exports: ["DeleteFile", "BatchDeleteFiles"]
    - path: "cmd/server/app.go"
      provides: "auditService wired to NewInputConfigHandler/NewSystemHandler/NewVideoFileHandler constructors"
      contains: "auditService"
  key_links:
    - from: "internal/handlers/input_config_handler.go"
      to: "internal/services/audit/audit_log_service.go"
      via: "auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{...})"
      pattern: "RecordChange\\(c\\.Request\\.Context\\(\\)"
    - from: "internal/handlers/system_handler.go"
      to: "internal/services/audit/audit_log_service.go"
      via: "auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{...})"
      pattern: "RecordChange\\(c\\.Request\\.Context\\(\\)"
    - from: "internal/handlers/video_file_handler.go"
      to: "internal/services/audit/audit_log_service.go"
      via: "auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{...})"
      pattern: "RecordChange\\(c\\.Request\\.Context\\(\\)"
    - from: "internal/services/input_config_service.go"
      to: "models.InputConfig"
      via: "First(&config) then Save(&config), returning pre-mutation snapshot"
      pattern: "oldConfig := config"
    - from: "internal/services/video_file_service.go"
      to: "models.VideoFile"
      via: "First(&file) returning pre-delete snapshot; BatchDeleteFiles uses Where('id IN ?') for full list snapshot"
      pattern: "oldFile := file"
    - from: "cmd/server/app.go"
      to: "internal/handlers/{input_config_handler,system_handler,video_file_handler}.go"
      via: "auditService parameter passed to 3 NewXxxHandler constructors"
      pattern: "auditService, a\\.logger"
metrics:
  duration: "约 15-20 分钟"
  tasks: 1
  files: 6
  commits: 2
  completed: null
---

# Quick Task 260729-mwt: OldData 捕获 — input-config / system / file (6 站点)

## 1. 一句话结论

**严格遵循 m8l 模式**（参见 `.planning/quick/260729-m8l-olddata-update-delete/260729-m8l-PLAN.md` §3 与 `260729-m8l-SUMMARY.md`），在 input-config（3）+ system（1）+ file（2）共 **6 个高危 P0 update/delete 站点**接入 service-layer snapshot → `audit.RecordChange(ctx, opts)` 模式。**不新增 helper 代码**，只改造 service 返回签名 + handler 增 `auditService` 字段 + handler 增 RecordChange 调用。

m8l 已固化模式：

```
service.First(&oldModel)  →  oldSnapshot = oldModel (拷贝)
service.mutate(oldModel → newModel)
return &oldModel, &newModel, nil
                          ↓
handler:  auditService.RecordChange(c.Request.Context(), RecordChangeOpts{
              Action: ..., Module: ..., Resource: fmt.Sprintf("...:%d", id),
              ResourceID: &id, OldData: oldModel, NewData: newModel })
                          ↓
LogOperation → Sanitizer(oldData) → Sanitizer(newData) → async queue → audit_logs
```

---

## 2. 6 个接入点（按风险递减）

| # | 文件:方法 | 模式 | Action | Module | OldData | NewData | Resource |
|---|----------|------|--------|--------|---------|---------|----------|
| 1 | `input_config_handler.go:CreateConfig` | Create（无 prior） | `ActionCreate` | `ModuleInputConfig` | `nil` | 新 `*models.InputConfig` | `input_config:%d` |
| 2 | `input_config_handler.go:UpdateConfig` | Update (old, new) | `ActionUpdate` | `ModuleInputConfig` | 旧 `*models.InputConfig` | 新 `*models.InputConfig` | `input_config:%d` |
| 3 | `input_config_handler.go:DeleteConfig` | Delete (old only) | `ActionDelete` | `ModuleInputConfig` | 旧 `*models.InputConfig` | `nil` | `input_config:%d` |
| 4 | `system_handler.go:UpdateConfig` | Partial sub-key snapshot | `"update_config"` | `ModuleSystem` | `map[string]interface{}{...}` 仅已变 sub-key | 同上（new 值） | `system_config:%s` (已变 sub-key 名) |
| 5 | `video_file_handler.go:DeleteFile` | Delete (old only) | `ActionDelete` | `ModuleFile` | 旧 `*models.VideoFile` | `nil` | `video_file:%d` |
| 6 | `video_file_handler.go:BatchDeleteFiles` | Batch (option b: 1 row, 切片 OldData) | `"batch_delete"` | `ModuleFile` | `[]models.VideoFile`（请求 IDs 对应的旧记录，**含 skipped processing**） | `nil` | `video_file:%s` (逗号拼接 IDs) |

### 2.1 Sub-key 决定 (system.UpdateConfig)

阅读 `internal/handlers/system_handler.go` UpdateConfig 全文（L72-133）后，对**实际变更内存 h.config 的子键**归类如下：

| 字段 | 是否 mutate `h.config` | 备注 |
|------|----------------------|------|
| `LogLevel` | ✅ mutate `h.config.Logging.Level` + 尝试动态更新 zap | snapshot 实际变更 |
| `LogFormat` | ✅ mutate `h.config.Logging.Format` | snapshot 实际变更 |
| `LogOutput` | ✅ mutate `h.config.Logging.Output` | snapshot 实际变更 |
| `MaxDiskUsage` | ✅ mutate `h.config.Storage.MaxDiskUsage` | snapshot 实际变更 |
| `RecordingsPath` | ❌ 仅记录 `changes` 字符串，不 mutate | **不**进 OldData（重启才生效，无法在内存中回滚） |
| `HLSPath` | ❌ 同上 | 同上 |
| `TempPath` | ❌ 同上 | 同上 |
| `FFmpegPath` | ❌ 同上 | 同上 |
| `FFprobePath` | ❌ 同上 | 同上 |

**结论**：OldData/NewData 只包含**实际 mutate 的 4 个 sub-keys**。Resource 字符串格式：`"system_config:logging.level,logging.format,logging.output,storage.max_disk_usage"`（实际只拼接本次请求中**非 nil** 且**与现值不同**的 key 列表）。

### 2.2 BatchDeleteFiles 决策 (option b)

两个候选：

| 选项 | 描述 | 优点 | 缺点 |
|------|------|------|------|
| (a) | 对每个 file 各 emit 一条 RecordChange | 粒度细，每条记录独立 | audit_logs 行数 = N，处理中跳过的也会落 N 条；前端查询 N 倍压力 |
| (b) | emit 一条 RecordChange，OldData = `[]models.VideoFile`（含 skipped），Resource = `video_file:1,2,3,4,5` | 一条审计行对应一次用户操作；Resource 即可定位"动了哪些 ID" | 失去每条文件的独立审计时间线 |

**选 (b)** 理由：与 m8l 模式"一次用户操作 = 一条 RecordChange"一致；audit_logs 表的 Resource 是单值字段，b 方案与之天然契合；后续若有 per-file 审计需求可加专门的 `video_file_audit` 表（out of scope）。

---

## 3. 任务列表

### Task 1（auto）：input-config / system / file 模块接入 6 站点

**预期产出**：

#### 3.1 service 改造（4 处签名变更）

```go
// internal/services/input_config_service.go
// UpdateConfig 现在返回 (old, new, err)
func (s *InputConfigService) UpdateConfig(id uint, req *UpdateInputConfigRequest) (*models.InputConfig, *models.InputConfig, error) {
    var config models.InputConfig
    if err := s.db.First(&config, id).Error; err != nil {
        return nil, nil, err
    }
    oldConfig := config  // ★ snapshot before mutation

    // ... 现有 if req.X != nil { config.X = *req.X } 逻辑保持不变 ...

    if err := s.ValidateConfig(&config); err != nil {
        return nil, nil, err
    }
    if err := s.db.Save(&config).Error; err != nil {
        return nil, nil, err
    }
    return &oldConfig, &config, nil
}

// DeleteConfig 现在返回 (old, err) — old 是删除前的 config
func (s *InputConfigService) DeleteConfig(id uint) (*models.InputConfig, error) {
    var config models.InputConfig
    if err := s.db.First(&config, id).Error; err != nil {
        return nil, err
    }
    oldConfig := config  // ★ snapshot before delete

    // 现有检查"配置正在被任务使用"逻辑
    var count int64
    s.db.Table("task_input_configs").Where("input_config_id = ?", id).Count(&count)
    if count > 0 {
        return nil, errors.New("配置正在被任务使用，无法删除")
    }
    if err := s.db.Delete(&models.InputConfig{}, id).Error; err != nil {
        return nil, err
    }
    return &oldConfig, nil
}

// CreateConfig 签名不变 (返回 *models.InputConfig, error)
// handler 自行 RecordChange(OldData: nil, NewData: result)
```

```go
// internal/services/video_file_service.go
// DeleteFile 现在返回 (old, err)
func (s *VideoFileService) DeleteFile(id uint) (*models.VideoFile, error) {
    var file models.VideoFile
    if err := s.db.First(&file, id).Error; err != nil {
        return nil, errors.New("文件不存在")
    }
    oldFile := file  // ★ snapshot before delete

    if file.Status == models.FileStatusProcessing {
        return nil, errors.New("文件正在处理中，无法删除")
    }
    // ... 现有 delete 逻辑 ...
    return &oldFile, nil
}

// BatchDeleteFiles 现在返回 (oldFiles []models.VideoFile, result, err)
// 关键：先批量查询所有要删的 files，snapshot，再走原 delete 流程
func (s *VideoFileService) BatchDeleteFiles(ids []uint) ([]models.VideoFile, *BatchDeleteFilesResult, error) {
    result := &BatchDeleteFilesResult{}
    if len(ids) == 0 {
        return nil, result, nil
    }

    // ★ Snapshot 全部 requested files BEFORE any delete (含 processing 状态)
    var oldFiles []models.VideoFile
    if err := s.db.Where("id IN ?", ids).Find(&oldFiles).Error; err != nil {
        return nil, result, err
    }

    // ... 现有 processing/orphan/filesWithTask 分类与 delete 逻辑不变 ...

    return oldFiles, result, nil
}
```

#### 3.2 handler 改造（3 个 handler 加 auditService 字段 + 6 处 RecordChange）

```go
// internal/handlers/input_config_handler.go
type InputConfigHandler struct {
    configService *services.InputConfigService
    auditService  *audit.AuditLogService   // ★ 新增
    logger        *zap.Logger
    usbScanner    *services.USBDeviceScanner
}

func NewInputConfigHandler(
    configService *services.InputConfigService,
    auditService  *audit.AuditLogService,  // ★ 新增
    logger *zap.Logger,
    usbScanner *services.USBDeviceScanner,
) *InputConfigHandler {
    return &InputConfigHandler{
        configService: configService,
        auditService:  auditService,        // ★ 新增
        logger:        logger,
        usbScanner:    usbScanner,
    }
}

// CreateConfig: OldData=nil, NewData=result
func (h *InputConfigHandler) CreateConfig(c *gin.Context) {
    // ... 现有 bind/validate ...
    config, err := h.configService.CreateConfig(&req)
    if err != nil { ... return }

    resourceID := config.ID
    if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
        Action:     models.ActionCreate,
        Module:     models.ModuleInputConfig,
        Resource:   fmt.Sprintf("input_config:%d", config.ID),
        ResourceID: &resourceID,
        OldData:    nil,
        NewData:    config,
    }); err != nil {
        h.logger.Warn("Failed to record input config create change", zap.Error(err), zap.Uint("config_id", config.ID))
    }

    h.logger.Info("Input config created", ...)
    response.GinSuccess(c, config)
}

// UpdateConfig: OldData=oldConfig, NewData=newConfig
func (h *InputConfigHandler) UpdateConfig(c *gin.Context) {
    // ... 现有 bind ...
    oldConfig, config, err := h.configService.UpdateConfig(uint(id), &req)
    if err != nil { ... return }

    resourceID := config.ID
    if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
        Action:     models.ActionUpdate,
        Module:     models.ModuleInputConfig,
        Resource:   fmt.Sprintf("input_config:%d", config.ID),
        ResourceID: &resourceID,
        OldData:    oldConfig,
        NewData:    config,
    }); err != nil {
        h.logger.Warn("Failed to record input config update change", zap.Error(err), zap.Uint("config_id", config.ID))
    }

    response.GinSuccess(c, config)
}

// DeleteConfig: OldData=oldConfig, NewData=nil
func (h *InputConfigHandler) DeleteConfig(c *gin.Context) {
    // ... 现有 parse ...
    oldConfig, err := h.configService.DeleteConfig(uint(id))
    if err != nil { ... return }

    resourceID := oldConfig.ID
    if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
        Action:     models.ActionDelete,
        Module:     models.ModuleInputConfig,
        Resource:   fmt.Sprintf("input_config:%d", oldConfig.ID),
        ResourceID: &resourceID,
        OldData:    oldConfig,
        NewData:    nil,
    }); err != nil {
        h.logger.Warn("Failed to record input config delete change", zap.Error(err), zap.Uint("config_id", id))
    }

    response.GinSuccess(c, gin.H{"message": "配置已删除"})
}
```

```go
// internal/handlers/system_handler.go
type SystemHandler struct {
    db           *gorm.DB
    auditService *audit.AuditLogService   // ★ 新增
    logger       *zap.Logger
    config       *config.Config
}

func NewSystemHandler(db *gorm.DB, auditService *audit.AuditLogService, logger *zap.Logger, cfg *config.Config) *SystemHandler {
    return &SystemHandler{
        db:           db,
        auditService: auditService,        // ★ 新增
        logger:       logger,
        config:       cfg,
    }
}

// UpdateConfig: 仅 snapshot 实际 mutate 的 sub-keys
func (h *SystemHandler) UpdateConfig(c *gin.Context) {
    // ... 现有 bind ...

    oldMap := map[string]interface{}{}
    newMap := map[string]interface{}{}
    changedKeys := []string{}

    if req.LogLevel != nil && *req.LogLevel != h.config.Logging.Level {
        oldMap["logging.level"] = h.config.Logging.Level
        newMap["logging.level"] = *req.LogLevel
        h.config.Logging.Level = *req.LogLevel
        changedKeys = append(changedKeys, "logging.level")
        if err := h.updateLogLevel(*req.LogLevel); err == nil { ... }
    }
    if req.LogFormat != nil && *req.LogFormat != h.config.Logging.Format {
        oldMap["logging.format"] = h.config.Logging.Format
        newMap["logging.format"] = *req.LogFormat
        h.config.Logging.Format = *req.LogFormat
        changedKeys = append(changedKeys, "logging.format")
    }
    if req.LogOutput != nil && *req.LogOutput != h.config.Logging.Output {
        oldMap["logging.output"] = h.config.Logging.Output
        newMap["logging.output"] = *req.LogOutput
        h.config.Logging.Output = *req.LogOutput
        changedKeys = append(changedKeys, "logging.output")
    }
    if req.MaxDiskUsage != nil && *req.MaxDiskUsage != h.config.Storage.MaxDiskUsage {
        oldMap["storage.max_disk_usage"] = h.config.Storage.MaxDiskUsage
        newMap["storage.max_disk_usage"] = *req.MaxDiskUsage
        h.config.Storage.MaxDiskUsage = *req.MaxDiskUsage
        changedKeys = append(changedKeys, "storage.max_disk_usage")
    }

    // ... 现有 changes 字符串数组与 response 不变 ...

    if len(changedKeys) > 0 {
        resourceID := uint(0)  // system config 没有单 ID
        if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
            Action:     "update_config",
            Module:     models.ModuleSystem,
            Resource:   "system_config:" + strings.Join(changedKeys, ","),
            ResourceID: &resourceID,
            OldData:    oldMap,
            NewData:    newMap,
        }); err != nil {
            h.logger.Warn("Failed to record system config update change", zap.Error(err))
        }
    }

    response.GinSuccess(c, gin.H{...})
}
```

```go
// internal/handlers/video_file_handler.go
type VideoFileHandler struct {
    fileService  *services.VideoFileService
    auditService *audit.AuditLogService   // ★ 新增
    logger       *zap.Logger
}

func NewVideoFileHandler(
    fileService *services.VideoFileService,
    auditService *audit.AuditLogService,  // ★ 新增
    logger *zap.Logger,
) *VideoFileHandler {
    return &VideoFileHandler{
        fileService:  fileService,
        auditService: auditService,        // ★ 新增
        logger:       logger,
    }
}

// DeleteFile: OldData=oldFile, NewData=nil
func (h *VideoFileHandler) DeleteFile(c *gin.Context) {
    id, err := parseUintParam(c, "id")
    if err != nil { ... return }

    oldFile, err := h.fileService.DeleteFile(id)
    if err != nil { ... return }

    resourceID := oldFile.ID
    if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
        Action:     models.ActionDelete,
        Module:     models.ModuleFile,
        Resource:   fmt.Sprintf("video_file:%d", oldFile.ID),
        ResourceID: &resourceID,
        OldData:    oldFile,
        NewData:    nil,
    }); err != nil {
        h.logger.Warn("Failed to record video file delete change", zap.Error(err), zap.Uint("file_id", id))
    }

    h.logger.Info("视频文件已删除", zap.Uint("file_id", id))
    response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// BatchDeleteFiles: 单条 RecordChange, OldData=[]models.VideoFile, Resource=video_file:1,2,3
func (h *VideoFileHandler) BatchDeleteFiles(c *gin.Context) {
    // ... 现有 bind ...
    oldFiles, result, err := h.fileService.BatchDeleteFiles(req.IDs)
    if err != nil {
        h.logger.Error("批量删除文件失败", zap.Error(err))
        response.GinError(c, response.CodeInternalError, "批量删除失败")
        return
    }

    // 拼 Resource: video_file:1,2,3,4,5
    idStrs := make([]string, len(oldFiles))
    for i, f := range oldFiles {
        idStrs[i] = strconv.FormatUint(uint64(f.ID), 10)
    }
    resource := "video_file:" + strings.Join(idStrs, ",")
    resourceID := uint(0)  // batch operation 没有单 ID

    if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
        Action:     "batch_delete",
        Module:     models.ModuleFile,
        Resource:   resource,
        ResourceID: &resourceID,
        OldData:    oldFiles,  // []models.VideoFile 切片
        NewData:    nil,
    }); err != nil {
        h.logger.Warn("Failed to record batch video file delete change", zap.Error(err))
    }

    h.logger.Info("批量删除文件成功", ...)
    response.GinSuccess(c, result)
}
```

#### 3.3 cmd/server/app.go wiring（3 处构造调用）

```go
// 在 initHandlers() 中（约 L669-674）：

// 改前：
InputConfig:   handlers.NewInputConfigHandler(inputConfigService, a.logger, usbScanner),
VideoFile:     handlers.NewVideoFileHandler(a.videoFileService, a.logger),
System:        handlers.NewSystemHandler(a.db, a.logger, a.config),

// 改后：插入 auditService (L580 已构造)
InputConfig:   handlers.NewInputConfigHandler(inputConfigService, auditService, a.logger, usbScanner),
VideoFile:     handlers.NewVideoFileHandler(a.videoFileService, auditService, a.logger),
System:        handlers.NewSystemHandler(a.db, auditService, a.logger, a.config),
```

#### 3.4 import 检查

3 个 handler 文件需新增 import：

| 文件 | 新增 import |
|------|------------|
| `input_config_handler.go` | `"github.com/NDCCCCCC/video-meeting-recorder/internal/models"`, `"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"` |
| `system_handler.go` | `"github.com/NDCCCCCC/video-meeting-recorder/internal/models"`, `"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"`, `"strings"`（如尚未 import） |
| `video_file_handler.go` | `"github.com/NDCCCCCC/video-meeting-recorder/internal/models"`, `"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"`, `"strconv"`, `"strings"`（如尚未 import） |

#### 3.5 Verify

```bash
# 1. Build
go build ./... && echo "BUILD OK"

# 2. Tests
go test -count=1 ./internal/services/audit/... ./internal/handlers/... \
                  ./internal/services/input_config_service_test.go \
                  ./internal/services/video_file_service_test.go

# 3. RecordChange 出现次数（应 >= 15 = m8l 的 9 + 这次的 6 calls）
grep -rn "RecordChange" internal/ | wc -l
# 期望 >= 15

# 4. 旧 m8l 6 站点未受影响
grep -c "RecordChange" internal/handlers/user_handler.go  # 应 = 3
grep -c "RecordChange" internal/handlers/role_handler.go  # 应 = 3

# 5. 手动触发 + 数据库核对
# 触发一次 PUT /api/v1/input-configs/1 + DELETE /api/v1/input-configs/<id> + POST /api/v1/files/batch
# 查询：
#   SELECT module, action, resource, JSON_EXTRACT(old_data, '$') AS od, JSON_EXTRACT(new_data, '$') AS nd
#   FROM audit_logs
#   WHERE module IN ('input_config', 'system', 'file')
#     AND action IN ('create', 'update', 'delete', 'batch_delete', 'update_config')
#   ORDER BY id DESC LIMIT 6;
# 期望：6 行，old_data 和 new_data 都不为 NULL（除非 create 的 old_data 是 NULL 设计上允许）
# input-config 的 Password / StreamPassword 应被 Sanitizer 脱敏为 '***'

# 6. system.UpdateConfig partial-update sanity check
# 仅改 LogLevel，期望：
#   resource = 'system_config:logging.level'
#   old_data = {"logging.level": "info"}
#   new_data = {"logging.level": "debug"}
# 同时改 LogLevel + MaxDiskUsage：
#   resource = 'system_config:logging.level,storage.max_disk_usage'
```

#### 3.6 Self-check

- [ ] `RecordChange` helper 签名未改（仍是 `func (s *AuditLogService) RecordChange(ctx, opts) error`）
- [ ] 6 个站点全部接入（input_config ×3 + system ×1 + file ×2）
- [ ] `cmd/server/app.go` 中 3 个 handler 构造调用更新
- [ ] input-config 的 Password / StreamPassword 字段经 Sanitizer 脱敏（即使在 OldData 中也以 `***` 形式落审计表）
- [ ] BatchDeleteFiles 选 option (b) — 单条 RecordChange，Resource 包含逗号拼接的所有 ID
- [ ] system.UpdateConfig OldData/NewData 只含**实际 mutate 的 4 个 sub-keys**（不包含 restart-required 字段）
- [ ] Sanitizer 继承自 LogOperation 管道，对 OldData 生效
- [ ] 现有 63 个中间件审计路径不受影响（input-config / system / file 的 create/update/delete 路由仍走 auditOp 中间件，新增的 RecordChange 是同请求第二条审计行）
- [ ] `go build ./...` 通过
- [ ] handlers + services 测试通过
- [ ] 1-2 次原子提交（"feat(audit): wire 6 high-risk OldData sites in input-config/system/file" + "chore(app): wire auditService to 3 handlers" 或合并为 1 次提交）

---

## 4. 已知遗留项 / 后续任务（写入 SUMMARY 的"后续步骤"章节）

**剩余 ~21 个 update/delete 站点未在本任务覆盖**（来自 m8l SUMMARY 的 ~50 站点 - m8l 6 - 本批 6 - 估计的另外 ~17 个其他已 P2 跳过 = ~21）：

| 批次 | 模块 | 站点数 | 优先级 | 建议任务 |
|------|------|--------|--------|----------|
| 中危 | recording (UpdateTask/DeleteTask/BatchDeleteTasks/StartTask/StopTask/CancelTask) | 6 | P1 | 下一 quick task |
| 中危 | storage (Upload/Delete/Share) | 3 | P1 | 批量 |
| 中危 | ppts (DeletePPT/DeleteSlides/Rollback/ReorderSlides) | 4 | P1 | 批量 |
| 中危 | apikey (UpdateAPIKey/DeleteAPIKey/ToggleAPIKeyStatus) | 3 | P1 | 批量 |
| 中危 | notification (UpdateUserSetting) | 1 | P1 | 批量 |
| 中危 | admin (UpdateAuthConfig) | 1 | P1 | 批量 |
| 中危 | system (ClearFiles) | 1 | P1 | 批量 |
| 中危 | input-config (TestConnection) | 1 | P2 | 跳过（无状态变更） |

**接入模板**已固化到 `RecordChange` helper，每站点成本 ~10 行（service 返回 old + handler 一处 RecordChange 调用）。批量接入 ~21 个剩余站点预计 1 个 quick task。

---

## 5. Self-Check 任务完成清单

- [ ] `internal/services/input_config_service.go` UpdateConfig/DeleteConfig 改造为返回 old
- [ ] `internal/services/video_file_service.go` DeleteFile/BatchDeleteFiles 改造为返回 old（BatchDeleteFiles 返回 `[]models.VideoFile`）
- [ ] `internal/handlers/input_config_handler.go` CreateConfig/UpdateConfig/DeleteConfig 调用 RecordChange + 新增 auditService 字段
- [ ] `internal/handlers/system_handler.go` UpdateConfig 调用 RecordChange（partial sub-key snapshot）+ 新增 auditService 字段
- [ ] `internal/handlers/video_file_handler.go` DeleteFile/BatchDeleteFiles 调用 RecordChange + 新增 auditService 字段
- [ ] `cmd/server/app.go` 3 个 handler 构造调用更新（传入 auditService）
- [ ] `go build ./...` 通过
- [ ] handlers + services 测试通过
- [ ] 1-2 次原子提交
- [ ] SUMMARY.md 包含"剩余 ~21 站点接入计划"章节
- [ ] 旧 audit_logs 行（含 NULL OldData）仍然可读
- [ ] input-config 的 Password / StreamPassword 字段经 Sanitizer 脱敏（`***`）
- [ ] system.UpdateConfig 不会因 restart-required 字段（路径类）触发空 OldData RecordChange（仅在 `len(changedKeys) > 0` 时调用）
- [ ] BatchDeleteFiles 选 option (b)，单条 RecordChange，Resource 包含所有 ID

---

## 6. 与 m8l 任务的对齐声明

本任务**严格遵循** m8l 的所有约束与决策（参见 `260729-m8l-PLAN.md` §3 末尾的约束段）：

- service 方法不新增 ctx 参数；handler 用 `c.Request.Context()`
- Sanitizer 自动覆盖 password/secret/token，OldData 里出现密码字段也会被 redact
- 不修改 RecordChange helper 签名
- 不引入与中间件审计行冲突的新行格式（同一请求可共存 2 条 audit_logs：中间件 HTTP 上下文行 + RecordChange OldData 业务上下文行）
- 失败处理：RecordChange 调用失败仅 `h.logger.Warn`，不阻断业务响应（与 m8l §3.6 UserHandler UpdateUser 一致）
- 文件命名：`{padded_phase}-{NN}-PLAN.md` = `260729-mwt-PLAN.md` ✅

---

## 7. 风险与回退

| 风险 | 缓解 |
|------|------|
| service 签名变更可能 break 其他调用者 | `grep` 确认每个被改 service 方法的调用方；m8l 改造已确立"controller 是唯一调用者"的事实 |
| BatchDeleteFiles 返回 `[]models.VideoFile` 大小可能很大（100 个文件 × 全部字段） | audit_logs.old_data JSON 字段无明确大小上限；LogOperation 已异步队列化；如遇性能问题可后续改为只 snapshot `id, file_name, file_path, status` 等关键字段 |
| system.UpdateConfig partial snapshot 可能遗漏 sub-key | 4 个 mutate sub-keys 已在 §2.1 表格明确列出；任何对 system_handler.go 后续添加 mutate 子键的 PR 需同时更新本表 |
| app.go 修改 3 处构造调用易漏 | 1 次 commit 完成；CI build 立即报错 |
