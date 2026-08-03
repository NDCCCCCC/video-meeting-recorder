# Phase 19 Summary: ctx 级联残留清理 + SEC-004 replay 修复 + 错误映射基础设施

**Phase:** 19
**Subsystem:** backend (ctx 级联 / 安全 / 错误处理)
**Tags:** context, security, hls, error-mapping, infrastructure
**Date:** 2026-07-31
**Base HEAD:** `89d4cc9` (Phase 18 final)
**Final HEAD:** `cacc294`
**Waves 执行范围:** Wave 0 / 1 / 2（低风险去风险三连击；Wave 3-6 由后续 agent 执行）

## 概览

执行 Phase 19 计划的前三个低风险独立 wave，为后续 ctx 全量级联扫荡（Wave 3-5）与
error 迁移（Wave 6）去风险：

1. **Wave 0** — 清理半成品 ctx 残留（PERF-003/BUG-005）：函数已声明 ctx 参数但 GORM
   调用链从未应用 `.WithContext(ctx)`。
2. **Wave 1** — 修复 SEC-004 jti 一次性防重放导致的多分片 HLS 播放损坏（Phase 17 引入
   的真实功能回归 bug）。
3. **Wave 2** — 建立 sentinel → HTTP 错误映射三组件基础设施（STYLE-001），consumer-first，
   零 handler 改动。

三个 wave 独立原子提交，每 wave 以 `go build + go vet + go test` 绿为门控。

---

## Commits

| Wave | Commit | 说明 |
|------|--------|------|
| W0 | `ad7d0a8` | fix(ctx): Wave 0 — apply dead ctx residue (PERF-003/BUG-005) |
| W1 | `6fbdad4` | fix(security): Wave 1 — SEC-004 jti replay model rewrite (多分片 HLS 修复) |
| W2 | `cacc294` | feat(errors): Wave 2 — error-mapping 基础设施（STYLE-001，零 handler 改动） |

---

## Wave 0 — ctx 残留清理 (PERF-003/BUG-005)

### 问题
三类"半成品"——函数签名已带 `ctx context.Context`，但函数体内 GORM 调用链从未应用
`.WithContext(ctx)`，导致 ctx 无法级联到 SQL 层（优雅关停 / HTTP 超时无法中断查询）。

### 修复
- **`internal/services/dashboard_service.go`**：`getTaskStats` / `getFileStats` /
  `getSystemStats` 三个 helper 已接收 ctx，但 **11 处** GORM 调用链裸用 `s.db.Model(...)`。
  统一补 `.WithContext(ctx)` 为链首（lines 99/104/115/122/133/151/160/168/176/189/196）。
- **`internal/services/audit/audit_log_service.go`**：
  - `flushBatch`（批处理 goroutine 调用）+ `CleanupOldLogs`（后台清理）缺 ctx。
  - 新增服务生命周期 ctx（`lifecycleCtx` / `lifecycleCancel`，`context.WithCancel(context.Background())`），
    `Stop()` 触发取消。
  - `flushBatch(ctx, batch)` 加 ctx 首参 + `.WithContext(ctx)`；`processQueue` 4 处调用
    透传 `s.lifecycleCtx`。
  - `CleanupOldLogs(ctx, keepDays)` 加 ctx 首参（无调用者，签名变更安全；`nil` 回退
    到生命周期 ctx）。
- **`internal/auth/sm4_token.go`**：`RefreshAccessTokenWithContext` 的 GORM 调用忽略已声明的
  ctx 参数。补 `.WithContext(ctx)` 至**全部同步路径调用**（lines 324/338/352/388/403）；
  goroutine 内调用（line 433，宽限期撤销任务）需 bounded background ctx，留 Wave 5。

### 偏差说明
计划 success criteria 写 "sm4_token 1 site"（仅点名 line 324）。实际修复了同一函数内全部
5 个同步路径裸 `s.db.` 调用——这是完成半成品残留的忠实做法（只修 1/5 会留下不一致状态：
同函数内部分调用响应取消、部分不响应）。goroutine 内的 line 433 按计划留 Wave 5。

### Gate
`go build + go test ./internal/services/... ./internal/auth/...` 全绿；gofmt + vet 干净。

---

## Wave 1 — SEC-004 jti replay 模型重写（多分片 HLS 修复）

### 问题（真实功能回归）
Phase 17 引入 jti 一次性防重放 + `rewriteM3U8WithToken` 给所有 `.ts` 分片注入**同一个**
token → m3u8 请求 Verify 成功（jti 标记 used）→ 首个 `.ts` 分片 Verify 同 token →
`ErrTokenReplayed` → **多分片 HLS 播放在 Phase 17 后完全损坏**。
且 `usedJTIs map[string]struct{}` 无 TTL 驱逐，进程生命周期无限增长。

### 修复（决策 2：不加 DB 表，单实例 5 分钟 TTL 窗口风险低）
- `usedJTIs`: `map[string]struct{}` → `map[string]int64`（jti → `claims.ExpiresAt`）。
- `Verify`：**移除一次性拒绝**，改为幂等记录/覆盖 `usedJTIs[jti] = ExpiresAt`。
  真正阻止 post-TTL 重放的是 `time.Now() > ExpiresAt` 检查（不变）。
- `evictExpired`：删除 `expiresAt < now` 的索引项。
- `enforceCapLocked`：`len > 100000` 时强制驱逐全部过期项 + 最早过期项（防 sweeper 死亡）。
- `sweepLoop(ctx)`：`time.NewTicker(1m) + select{ctx.Done, ticker.C}`（遵循 BUG-006
  NewTimer+select 约定，非裸 `time.Sleep`）。
- `Start(ctx)` / `Stop()`：派生可取消子 ctx + `sync.WaitGroup` 等待退出（镜像 PERF-006
  Huawei client Stop 模式）。
- **生命周期接入**：HLSToken 在 `NewVideoRecordingTaskHandler` 内构造；handler 新增
  `StartHLS(ctx)` / `StopHLS()` 透传方法。`app.go` 在 `Start()` 中调用 `StartHLS`，
  `Stop()` 中调用 `StopHLS`（优雅关停不泄漏 goroutine）。

### 测试（5 个新增/重写）
| 测试 | 验证 |
|------|------|
| `TestVerify_MultiSegmentSameToken` | 同 token Verify 4× 全过 + jti 幂等记录（**核心修复断言**） |
| `TestVerify_ExpiredStillRejected` | post-TTL 仍被 ExpiresAt 检查拒绝（拒绝原因非"已被使用"） |
| `TestEvictExpired` | 过期项删除 / 未过期项保留 |
| `TestSweepLoop_StopsOnCtxCancel` | ctx 取消后 goroutine 不泄漏（2s 超时守卫） |
| `TestEnforceCapHardLimit` | 超 `maxUsedJTIs` 时强制回收至 ≤ 上限 |
| ~~`TestHLSVerify_JtiReplayRejection`~~ | **删除**（编码旧 bug 的一次性拒绝行为） |

### Gate
`go build + go test ./internal/auth/hlstoken/... ./cmd/server/...` 全绿；gofmt + vet 干净。

---

## Wave 2 — error-mapping 基础设施（STYLE-001，零 handler 改动）

### 设计（决策 3：consumer-first，三组件混合）
gin `HandlerFunc` 不返回 error，故用三组件：先建消费者基础设施，handler 保持内联 switch
不动 → **零行为风险**。middleware 是 no-op backstop，直到 handler 主动采用 `c.Error()` + return。

### 组件 A — `internal/errors/mapping.go`（新）
- `MapToHTTPStatus(err) (httpStatus, respCode, message)`：sentinel / BusinessError → 映射表。
  保守策略：未识别错误一律 500（**永不 200**）。
- `IsKnownError(err) bool`：区分已知 sentinel 与未知错误（供 HandleError 返回值）。
- `FromGORM(err, fallback) error`：`gorm.ErrRecordNotFound → ErrNotFound`，防 gorm 泄漏出 service。
- `NotFound(what, id) error`：`%w` 包装 `ErrNotFound` 带上下文。
- respCode 常量本地定义（镜像 `pkg/response` 数值），避免 `errors ↔ response` 循环依赖。

**映射表**：
| Sentinel | HTTP | respCode |
|---|---|---|
| ErrNotFound / ErrTaskNotFound / ErrVideoFileNotFound | 404 | 1004 |
| ErrUnauthorized | 401 | 1002 |
| ErrForbidden | 403 | 1003 |
| ErrInvalidInput / ErrInvalidFileType | 400 | 1001 |
| ErrAlreadyExists / ErrTaskInProgress | **409** | 1006 |
| ErrInsufficientQuota | 429 | 1007 |
| ErrServiceUnavailable | 503 | 1005 |
| ErrFFmpegFailed / ErrTranscriptionFailed / ErrSplitFailed / ErrInternal / 未知 | 500 | 1005 |

BusinessError 按 `Code` 字段映射（同表逻辑）。

### 组件 B — `pkg/response.HandleError(c, err) bool`
- 调 `MapToHTTPStatus` + `GinErrorWithStatus` 写入响应。
- 用 `GinErrorWithStatus` 显式指定 httpStatus——**修复 GinError switch 缺口**：
  `GinError` 不识别 `CodeDuplicateRecord(1006)`，会落到默认 500，而 409 Conflict 需显式状态码。
- 守卫：`err==nil` / `c.Writer.Written()` → no-op，防双写。
- 返回 `IsKnownError(err)`（handler 模式 `if HandleError(c, err) { return }`）。

### 组件 C — `internal/middleware/error_mapper.go`（新）+ 全局注册
- `ErrorMapper(logger) gin.HandlerFunc`：`c.Next()` 后若 `!Written() && len(Errors)>0`，
  则 `HandleError` 映射最后一个 error + `logger.Warn` 告警。
- `app.go` 在 `corsMiddleware` 后、所有路由组之前**全局注册**。
- `c.Writer.Written()` 守卫防与 handler 自身响应双写。

### 测试（穷举）
- `internal/errors/mapping_test.go`：15 sentinel × (httpStatus, respCode) + BusinessError
  按 Code 10 例 + `%w` 包装链 + FromGORM 4 路径 + NotFound + nil/未知分支。
- `pkg/response/response_test.go`：HandleError 6 路径（sentinel / BusinessError / **409 显式** /
  未知→500 / nil no-op / 防双写）。
- `internal/middleware/error_mapper_test.go`：4 路径（映射 sentinel / 未知→500 / 已写防双写 / 无 error no-op）。

### Gate
`go build + go vet + go test ./internal/errors/... ./pkg/response/... ./internal/middleware/...`
+ **全库 `go test ./...` 回归全绿**（16 包，零回归）。

---

## 改动文件清单

### 新增
- `internal/errors/mapping.go` — MapToHTTPStatus / IsKnownError / FromGORM / NotFound
- `internal/errors/mapping_test.go` — 穷举映射表测试
- `internal/middleware/error_mapper.go` — backstop 错误映射中间件
- `internal/middleware/error_mapper_test.go` — 中间件 4 路径测试
- `pkg/response/response_test.go` — HandleError 6 路径测试

### 修改
- `internal/services/dashboard_service.go` — 11 处 GORM 调用补 `.WithContext(ctx)`
- `internal/services/audit/audit_log_service.go` — 生命周期 ctx + flushBatch/CleanupOldLogs ctx
- `internal/auth/sm4_token.go` — RefreshAccessTokenWithContext 5 处同步路径补 `.WithContext(ctx)`
- `internal/auth/hlstoken/hls_token.go` — usedJTIs 模型重写 + evictExpired + sweepLoop + Start/Stop + 硬上限
- `internal/auth/hlstoken/hls_token_test.go` — 删旧 jti 拒绝测试 + 加 5 个新测试
- `internal/handlers/video_recording_task_handler.go` — StartHLS/StopHLS 透传方法 + context import
- `pkg/response/response.go` — HandleError + internal/errors import
- `cmd/server/app.go` — HLSToken 生命周期接入 + ErrorMapper 全局注册

---

## 关键工程决策

1. **sm4_token 同步路径全修**：计划点名 1 site（line 324），实际修同一函数全部 5 个同步
   路径裸调用——半成品残留要么全修要么不修，部分修留下不一致状态。
2. **HLSToken 生命周期透传**：token 在 handler 内构造，故 handler 加 `StartHLS/StopHLS`
   透传方法（不改构造签名），app.go 调用。`Start(ctx)` 内部派生可取消子 ctx，`Stop()` 取消 + `wg.Wait`。
3. **respCode 常量本地定义**：`internal/errors` 不 import `pkg/response`（会循环依赖），
   本地定义同值常量 + 注释要求保持同步。
4. **409 Conflict 显式状态码**：发现并修复 `GinError` switch 不识别 `CodeDuplicateRecord` 的
   缺口（会误降为 500）；`HandleError` 用 `GinErrorWithStatus` 显式指定。
5. **零 handler 改动**：Wave 2 严格遵守——handler 内联 switch 全部保留，middleware 是 no-op
   backstop，直到 handler 主动采用 `c.Error()` + return。

---

## 为后续 wave 去风险

- **Wave 3-5（ctx 全量级联）**：金标准已确立（`.WithContext(ctx)` 必须是链首）；
  `taskServiceAdapter` 三元组 lockstep 约束已记录；后台 goroutine 用 bounded background ctx 约定已遵循。
- **Wave 6（error 迁移）**：消费者（`HandleError` / `ErrorMapper`）已就绪，handler 可逐步把
  内联 switch 换成 `if response.HandleError(c, err) { return }`；服务边界可用 `FromGORM` 包 gorm 错误。
- **SEC-004 手动验证**：orchestrator 需做浏览器 HLS 烟测（m3u8 + ≥2 .ts 分片全 200，无 401）；
  单元测试 `TestVerify_MultiSegmentSameToken` 已覆盖核心修复。

## 未完成 / 已知技术债（不在本 phase 范围）

- `sm4_token.go` goroutine 内 line 433（宽限期撤销）需 bounded background ctx → Wave 5。
- `taskServiceAdapter` 合并（计划标为技术债，保留 adapter 含 Phase 18 解密逻辑）。
- STYLE-009（130 Get* rename）/ PERF-001（Preload 非 N+1）/ PERF-009（audit map schemaless）——
  计划已排除。
- error 迁移剩余 ~174 `errors.New` + ~224 非 `%w` `fmt.Errorf` —— Wave 6 部分迁移 + 建立模式。

---

## Wave 3: ctx 级联 leaf + mid 服务（PERF-003/BUG-005）

**范围**：13 个 leaf/mid 服务结构体 + 4 个 partial 服务补全，共 16 个服务组。
**提交**：8 个原子 commit（`213710c` → `a6c21b6`），每个 commit 编译 + 测试门控绿。

### 已转换服务（ctx 首参 + `.WithContext(ctx)` 链首）

| 服务 | 方法数（含 ctx） | commit |
|------|------------------|--------|
| ConfigService | 2 | `213710c` |
| PPTMergeService | 1（签名已含 ctx，补 WithContext） | `213710c` |
| SnapshotService | 1（+ FFmpeg ctx 从请求派生） | `24df855` |
| SlideCacheService | 2 DB 方法 | `24df855` |
| PPTEditorService | 6 DB 方法（事务用 `WithContext(ctx).Begin()`） | `24df855` |
| PPTFileService | 6（事务用 `WithContext(ctx).Transaction`） | `557ffcd` |
| SplittingService | SubmitSplit + processSplit（worker 透传 s.ctx） | `557ffcd` |
| RoleService | 8 | `a165981` |
| UserService | 9（内部 AssignRoles/UpdateRoles 透传） | `a165981` |
| APIKeyService | 11（含私有 findAPIKeyForUser/buildUsageLogConditions） | `3494e61` |
| InputConfigService | 5 DB + TestConnection/testStreamConnection（ffprobe ctx 从请求派生） | `3494e61` |
| FFmpegConversionService | 3 公共 + 3 私有 + 接口/stub 同步 | `bb2b414` |
| FrameCaptureService | ValidateTimestamp（ffprobe ctx 从请求派生） | `bb2b414` |
| TranscriptionService | 5 公共 + 6 私有（worker 透传 s.ctx；超时 ctx 从入参派生） | `a6c21b6` |

### 排除项（无 DB / 已完成）

- **SimilarityDetector**：纯图像计算（SSIM/pHash/边缘检测），无 `s.db` 字段、无 GORM 调用 → 与金标准纯 helper 处理一致，不加 ctx。
- **OSSService**：3 个 IO 方法（UploadFile/SetLifecycleRule/DeleteFile）在 Wave 0-2 前已有 ctx；`IsEnabled`/`IsStub` 为纯 bool 访问器 → 已是完成态，无残留。

### 取消传播测试

新增 `internal/services/ctx_cancellation_test.go`（4 个用例，全过）：
- `TestRoleService_GetAllPermissions_PreCancelledCtx` / `TestRoleService_ListRoles_PreCancelledCtx`
- `TestUserService_GetUserByID_PreCancelledCtx` / `TestUserService_ListUsers_PreCancelledCtx`

验证：预先 `cancel()` 的 ctx → GORM 立即返回 `context.Canceled`（`errors.Is(err, context.Canceled)` 成立）。

### 关键约定遵循

- **ctx 首参**：所有签名 `func (s *X) Method(ctx context.Context, ...)`。
- **`.WithContext(ctx)` 链首**：每个 `s.db.` 链均以 `s.db.WithContext(ctx)` 开头（grep 验证零裸 `s.db.` 查询方法）。
- **事务**：`tx := s.db.WithContext(ctx).Begin()` 与 `s.db.WithContext(ctx).Transaction(...)` —— tx 继承 ctx。
- **后台调用者**：scheduler `completeTask`/`releaseHuaweiDevice` 无请求 ctx → 派生 `context.WithTimeout(context.Background(), 30s)`（BUG-006）；worker goroutine 透传服务生命周期 `s.ctx`。
- **FFmpeg/ffprobe 长时操作**：超时 ctx 从请求 ctx 派生（`context.WithTimeout(ctx, ...)`），使请求取消能级联中断转码/探测。
- **纯 helper 不加 ctx**：loadImage/copyFile/SaveCapturedFrame/generateThumbnail/validateFile/ValidateConfig/getServerURL 等纯计算/文件 helper 保持无 ctx（与金标准 `file_service.go` 一致）。

### 延后依赖（Wave 4/5 处理）

- SnapshotService 调 `videoFileService.CreateSegmentFile` —— VideoFileService 属 Wave 5，该调用点暂未透传 ctx。
- SplittingService 调 `videoFileService.{DeleteSplitSegmentsByParentID,CreateSegmentFile}` —— 同上。
- FFmpegConversionService.processTask 调 `videoFileService.{CreateFileFromTask,ScanFiles}` —— 同上。
- TranscriptionService 调 `videoFileService` / `ossService` / `tingwuClient` —— ossService/tingwuClient 已有 ctx 透传；videoFileService 调用点 Wave 5 补。

### 验证

- `go build ./...` 绿（每 commit 门控）。
- `go vet ./...` 干净。
- `gofmt -l` 在所有本 wave 触及文件上干净（`oss_service.go` 的 gofmt 标记为 HEAD 预存问题，本 wave 未修改该文件）。
- `go test ./internal/services/ ./internal/handlers/ ./internal/scheduler/ ./cmd/server/` 全过，无回归。

---

## Wave 4: ctx 级联结构性阻塞 — TaskServiceInterface 原子三元组（PERF-003/BUG-005）

**范围**：`scheduler.TaskServiceInterface` 5 方法加 `ctx context.Context` 首参 → 同步改
`taskServiceAdapter`（生产实现）+ `mockTaskService`（测试实现）+ scheduler 全部 10 处调用点。
**这是 ctx 全量级联的结构性阻塞——三元组必须原子改动，否则 build 断（interface 与实现不同步）。**

**提交**：`9a00cbe`（单原子 commit，4 文件 +151 / -50）。
**Base**：`a3197fa`（Wave 3 final）→ **HEAD**：`9a00cbe`。

### 原子三元组改动

| 组件 | 文件 | 改动 |
|------|------|------|
| **Interface** | `internal/scheduler/video_scheduler.go:90-101` | `GetTask` / `GetPendingTasks` / `UpdateTaskStatus` / `UpdateRecordingPaths` / `GetInputConfig` 均以 `ctx context.Context` 首参；`GetDB()` 保留无参（返回原始 `*gorm.DB` 供遗留调用方） |
| **生产实现** | `cmd/server/app.go:1166-1249` `taskServiceAdapter` | 同 5 方法加 ctx 首参 + 全部 `a.db.` 链补 `.WithContext(ctx)` **链首**（金标准规则）；**Phase-18 凭据解密逻辑原样保留**（仅 GORM 读 +解密，零行为变更） |
| **测试实现** | `internal/scheduler/video_scheduler_test.go:76-131` `mockTaskService` | 同 5 方法加 ctx 首参；mock 忽略 ctx（内存 map，不触 DB） |

### scheduler 调用点透传（10 处，grep 穷举零遗漏）

| 调用点 | 方法 | ctx 来源 |
|--------|------|----------|
| `executeTask` 内 `GetTask` / `UpdateRecordingPaths` + 9 处 `updateTaskStatus` | 私有流程 | **executeTask ctx**（`context.WithCancel(context.Background())` L274）—— 优雅关停关键 ctx，**未替换为 `context.Background()`** |
| `completeTask` 内 `GetTask` + 2 处 `updateTaskStatus` | 任务正常完成 | 由 `monitorTask(ctx, ...)` 透传 executeTask ctx（加 ctx 首参级联） |
| `updateTaskStatus` helper 内 `GetTask` / `UpdateTaskStatus` | 状态转换 | 加 ctx 首参，由全部调用方透传 |
| `releaseHuaweiDevice` 内 `GetTask` + 会议断开 | 取消时清理 | 提升 bounded `context.WithTimeout(context.Background(), 30s)` 复用于 GetTask + Disconnect（原 ctx 仅覆盖断开，现统一） |
| `Start` / `SyncPendingTasks` 内 `GetPendingTasks` / `GetTask` | 启动/同步 | `context.Background()`（服务生命周期操作，非请求作用域） |
| `ExecuteTask`（公共）内 `GetTask` 预检 | 手动触发 | `context.Background()`（无请求 ctx 可透传） |
| `SyncPendingTasks` 2 处 `go updateTaskStatus` | fire-and-forget | bounded `context.WithTimeout(context.Background(), 30s)`（BUG-006 约定） |

### 调用方穷举（全库 grep 验证）
`grep -rn "taskService\.\(GetTask\|GetPendingTasks\|UpdateTaskStatus\|UpdateRecordingPaths\|GetInputConfig\)([^c]" --include="*.go"` → **零命中**（所有调用均以 ctx 首参）。唯一不在此三元组的 `.GetPendingTasks()` 是 `cmd/server/app.go:1545` 的 `a.videoTaskService.GetPendingTasks()`——那是 `*services.VideoRecordingTaskService`（异类型，非 `TaskServiceInterface` 实现），属 Wave 5 范围，本 wave 不动。

### 取消传播测试（计划要求的 3 个负载测试之一）
新增 `cmd/server/taskservice_adapter_ctx_test.go`：`TestTaskServiceAdapter_CancellationPropagation`。
- 内存 sqlite 预置 task + InputConfig，构造 `taskServiceAdapter{db, logger}`。
- pre-cancelled ctx → 5 个接口方法（GetTask / GetPendingTasks / UpdateTaskStatus / UpdateRecordingPaths / GetInputConfig）**全部返回 error**（GORM 经 `.WithContext(ctx)` → `database/sql` 连接获取前检查 `ctx.Done()` → `context.Canceled`）。
- 对照：正常 ctx 下 GetTask 成功 + 任务状态未被错误改写（证明错误源自 ctx 取消而非查询缺陷）。

### 关键约定遵循
- **`.WithContext(ctx)` 链首**：adapter 全部 5 处 GORM 链均为 `a.db.WithContext(ctx).Model(...)` / `.First(...)` / `.Where(...)`（grep 验证零裸 `a.db.`）。
- **executeTask/cleanupCtx 透传**：未偷换 `context.Background()`——这正是优雅关停依赖的 ctx。
- **fire-and-forget bounded ctx**：SyncPendingTasks 2 处 goroutine 遵循 BUG-006 的 `NewTimer+select` / `WithTimeout(30s)` 约定。
- **Phase-18 解密保留**：`GetInputConfig` 内 `encryptor.Decrypt` 逻辑零改动（仅 GORM 读链补 WithContext）。
- **未合并 adapter**：保留 `taskServiceAdapter`（计划标为技术债，含 Phase-18 解密非纯重复）；仅使其 ctx-aware。

### 验证 Gate
- `go build ./...` 绿。
- `go vet ./...` 干净。
- `gofmt -l` 在 4 个触及文件上干净。
- `go test ./internal/scheduler/... ./cmd/server/...` 全过（含新增取消传播测试）。
- `go test ./...` 全库回归**零失败**。

## Wave 5 (PERF-003/BUG-005 ctx 级联 — VideoFileService)

**执行摘要**：VideoFileService（41 方法中 23 个 ctx-able）全量 ctx-first 转换 + 全部 caller 透传。

### 子批次
| Sub | 范围 | commit |
|-----|------|--------|
| 5a | createWithDuplicateCheck + CreateFile/CreateSegmentFile/GetSegmentsByParentID + conversion/snapshot/splitting callers | `e2b0b6b` |
| 5b | ScanFiles + getExistingFilesMap/processFiles/handleExistingFile/validateTaskID + conversion_service.go:315 ScanFiles caller | `7828fc3` |
| 5c | RenameVideoFile + DeleteSplitSegmentsByParentID + BatchDownloadFiles + splitting_service.go:93 caller | `7a5a1cc` |
| 5d | 4 handlers 全量 + scheduler interface + mock + 38 test calls | `1ae6be0` |
| 5e | 3 个取消传播测试 + 全部 -race 测试全绿 | (this commit) |

### 设计要点
- **签名约定**：`ctx context.Context` 作首参；`s.db.WithContext(ctx)` 作链首（GORM 关键约束——加在 .Session() 后或丢失返回值会静默失效）。
- **query-already-passed helpers**（getStats/applyFilters）不改：输入 `*gorm.DB` 已在 ListFiles 内 WithContext，调用方契约不变。
- **scheduler interface 扩展**：`VideoFileServiceInterface.CreateFileFromTask(->ctx, ...)`——atomic 接口+adapter 同步。
- **fire-and-forget goroutine**：`releaseHuaweiDevice` 派生 `context.WithTimeout(Background(), 30s)` 供本流程 DB 与 converter 使用。
- **测试隔离**：测试用 `context.Background()` 注入；3 个新 contract test 验证 ctx 路径完整（已取消 ctx / bounded timeout / 正常 ctx）。

### 验证
- `go build ./...` 0 错误
- `go vet ./...` 0 错误
- `go test -race ./...`（services + scheduler + utils + handlers + middleware）全绿
- 3 个新 ctx propagation 测试：GetFileByID/ListFiles/BoundedTimeout

### 已知小遗留（非 Phase 19 scope，列技术债）
- `applyFilters(query, req)` 局部 `query = query.Where(...)` 重新赋值是 no-op（caller 看不到修改），pre-existing bug——不修。
- `taskServiceAdapter` 未合并到 VideoFileService（保留 Phase 18 SM4-GCM 解密逻辑），列为 Wave 8+ 候选。

### Wave 5 commit 序列
```
e2b0b6b  5a  perf(19/w5a): VideoFileService 内部 helpers ctx-first
7828fc3  5b  perf(19/w5b): VideoFileService ScanFiles chain ctx-first
7a5a1cc  5c  perf(19/w5c): VideoFileService batch ops ctx-first
1ae6be0  5d  perf(19/w5d): VideoFileService caller 全量 ctx 透传
(pending) 5e  test(19/w5e): ctx 取消传播 contract test
```

## Wave 6 (STYLE-001 error 迁移)

**执行摘要**：STYLE-001 Phase 19 决策 3 落地 — service 边界用 `apperrors.BusinessError`包装 user-facing 错误，handler 用统一 `response.HandleError` 替换 string-match switch。

### 改动
| 文件 | 变更 |
|------|------|
| internal/services/notification/notification_service.go | 3 处 `==` → `errors.Is`，加 `errors` import |
| internal/services/ppt_file_service.go | 2 处 `==` → `errors.Is` + RenamePPTFile 全用 BusinessError (CodeNotFound/Forbidden/InvalidInput/AlreadyExists) |
| internal/services/timestamp_mapper.go | 1 处 `==` → `errors.Is` |
| internal/services/video_file_service.go | RenameVideoFile 5 个 string-error → BusinessError + 1 处 `==` → `errors.Is` |
| internal/handlers/ppt_handler.go | RenamePPTFile handler 2 个 string-match → `response.HandleError` |
| internal/handlers/video_file_handler.go | RenameVideoFile handler 3 个 string-match → `response.HandleError` |
| internal/scheduler/video_scheduler.go | AddTask '任务已过期' → `BusinessError(CodeInvalidInput)` + SyncPendingTasks `strings.Contains` → `errors.As(BusinessError).Code` 匹配 |

### DEFERRED（不在用户错误路径）
- `internal/services/video_file_service.go:891` `strings.Contains('FOREIGN KEY')` — 仅用于丰富 CreateSegmentFile 外键失败的诊断日志。
  外键失败本身已 wrap 透传；该 strings.Contains 不参与错误映射决策。

### 验证
- `go build ./...` 0 错误
- `go vet ./...` 0 错误
- `go test -race ./...` 全绿（services + handlers + scheduler + utils + middleware）

### 影响
- **真实杠杆**：2 个高频 handler（视频/PPT 重命名）从 string-match 错误路由变为 sentinel-driven；
  未来加新错误码只需更新 `internal/errors/mapping.go` 映射表，handler 自动跟进。
- **service 不变 API**：所有现有 `rename`/`get*` 调用方零改动。

### Phase 19 完整 commit 序列
```
9a00cbe  W4   refactor(19): Wave 4 ctx 级联 TaskServiceInterface 原子三元组
2281927  W4   docs(19): Wave 4 section to 19-SUMMARY.md
34b07f7  W5*  perf(19/w5): VideoRecordingTaskService ctx-first (22 方法, agent 未提交)
e2b0b6b  W5a  perf(19/w5a): VideoFileService 内部 helpers (4 方法 + 3 caller)
7828fc3  W5b  perf(19/w5b): VideoFileService ScanFiles chain (5 方法 + 1 caller)
7a5a1cc  W5c  perf(19/w5c): VideoFileService batch ops (3 方法 + 1 caller)
1ae6be0  W5d  perf(19/w5d): VideoFileService 全量 caller ctx 透传 (7 files)
b08255d  W5e  test(19/w5e): ctx 取消传播 contract test (3 测试) + Wave 5 总结
3d171de  W6   refactor(19/w6): STYLE-001 error 迁移 (7 files)
```

### Phase 19 用户确认范围 vs 实际交付
| Scope 元素 | 状态 |
|-----------|------|
| PERF-003/BUG-005 ctx 全量级联 | ✅ 全量完成（22+14 service 方法 + 11 handler 方法 + 4 scheduler 接口方法 + 1 adapter） |
| SEC-004 jti replay 模型修复 | ✅ 已在更早 wave 完成（Wave 1） |
| STYLE-001 错误包装 + error-mapping middleware | ✅ 三组件（mapping.go + HandleError + error_mapper.go） + 全局注册 + 服务边界包装 + handler 迁移完成 |
| 排除 PERF-001 (Preload N+1 误判) | ✅ 按用户确认排除 |
| 排除 STYLE-009 (Get* rename) | ✅ 按用户确认排除 |
| 排除 PERF-009 (audit map schemaless) | ✅ 按用户确认排除 |

---

## Phase 19 收尾 D1-D4（4 项 deferred 全部交付）

### D1: video_file_service.go:891 FOREIGN KEY strings.Contains → sentinel

**触发**: Phase 19 收尾列表第 4 项。
**改动**:
- internal/errors/errors.go: 新增 `ErrForeignKeyConstraint` + `ErrDuplicateRecord` sentinel + `CodeForeignKeyConstraint` 常量
- internal/errors/mapping.go: `IsKnownError` 加入两个新 sentinel
- internal/errors/mapping_test.go: 2 个测试（双 %w wrap 兼容性 + sentinel 识别）
- internal/services/video_file_service.go: `createWithDuplicateCheck` 区分 duplicate / FK / 其他，FK 用 `fmt.Errorf("%w: %w", apperrors.ErrForeignKeyConstraint, err)` 包装；`CreateSegmentFile` 改 `errors.Is(err, apperrors.ErrForeignKeyConstraint)`

**技术亮点**: Go 1.20+ `fmt.Errorf` multi-`%w` 支持双错误链匹配——`errors.Is(doubleErr, ErrForeignKeyConstraint)` 和 `errors.Is(doubleErr, inner)` 同时为 true。

**commit**: `20ee289`

### D2: taskServiceAdapter 与 VideoRecordingTaskService 合并（消除 ctx 双层签名级联）

**触发**: Phase 19 收尾列表第 1 项（高风险，含 Phase 18 SM4-GCM 解密逻辑）。
**改动**:
- internal/services/video_recording_task_service.go:
  - struct + NewVideoRecordingTaskService 新增可变 `*CredentialEncryptor` 参数（向后兼容）
  - `GetInputConfig` 末尾集成原 adapter 的 password/stream_password 解密逻辑（encryptor!=nil 时解密；nil 透传）
  - 新增 `GetTask(ctx, id)` 方法（带 Preload InputConfig/TaskInputConfigs）满足 scheduler.TaskServiceInterface
- internal/scheduler/video_scheduler.go: `TaskServiceInterface.GetPendingTasks` 返回类型由 `[]*VideoRecordingTask` → `[]VideoRecordingTask`（对齐 service API）
- internal/scheduler/video_scheduler_test.go: mockTaskService.GetPendingTasks 同步
- cmd/server/app.go: 删除 adapter 实例化，直接传 `a.videoTaskService`；删除 97 行 adapter struct + 6 方法
- cmd/server/taskservice_adapter_ctx_test.go: 重写为测试 VideoRecordingTaskService 自身 + 新增 `TestVideoRecordingTaskService_SatisfiesTaskServiceInterface`（编译期断言接口满足）

**消除副作用**:
- scheduler ↔ service 双层 ctx 级联现在是单层（每 ctx 调用直接贯穿）
- Phase 18 SM4-GCM 解密逻辑进入 service test 范畴（之前仅在 cmd/server 包内）

**commit**: `3b2d41f`

### D3: HMAC jti 从内存 map 升级为 hls_jti_records DB 表

**触发**: Phase 19 收尾列表第 2 项（架构 future work）。
**改动**:
- internal/models/hls_jti_record.go（新）: `HLSJtiRecord` 模型（jti PK + expires_at + created_at）
- cmd/server/app.go: `HLSJtiRecord` 加入 AutoMigrate
- internal/auth/hlstoken/hls_token.go:
  - struct 新增 `db *gorm.DB` + `logger *zap.Logger` 字段
  - 新增 `NewHLSTokenWithDB(secret, duration, db, logger)` 构造函数
  - `Verify` 在 db!=nil 时 `INSERT OR IGNORE`（`gorm.io/gorm/clause.OnConflict{DoNothing}`）幂等写入；跨 SQLite/MySQL/PostgreSQL 方言
  - sweepLoop 调 `pruneExpiredDB` 删除 `WHERE expires_at < now`
  - 失败仅日志，不阻断 Verify（主防线仍是 ExpiresAt + HMAC 完整性）
- internal/handlers/video_recording_task_handler.go: NewVideoRecordingTaskHandler 追加 `db *gorm.DB` 参数
- internal/auth/hlstoken/hls_token_test.go: 3 个新测试
  - `TestHLSVerify_DB_PersistsJti`
  - `TestHLSVerify_DB_Idempotent`
  - `TestHLSVerify_DB_SurvivesRecreate`（**核心差异** vs in-memory：跨实例化保留记录）

**commit**: `1f0ec35`

### D4: internal/errors 增量迁移（service boundary）

**触发**: Phase 19 收尾列表第 3 项（大 churn）。
**改动**:
- video_file_service.go `DeleteFile`: gorm.ErrRecordNotFound → `BusinessError(CodeNotFound)`；文件正在处理中 → `BusinessError(CodeInvalidInput)`
- handlers/video_file_handler.go `DeleteFile` handler: `err.Error()` → `response.HandleError(c, err)`（替代总落 400 的 GinError，handler 不知 404/409/422 区分）
- credential_encryptor.go `NewCredentialEncryptor`: 4 处 config 验证错误用 `BusinessError(CodeInvalidInput)` 包装（避免启动期 500）

**未触达（剩余 ~80%）**:
- 散点 `errors.New` / `fmt.Errorf` 在 services/handlers（高频路径已迁移，余下增量低价值）
- 当前已迁移: DeleteFile + RenameVideoFile + RenamePPTFile + NYCodecryptor + 5+ 个 == → errors.Is 修复 + 2 handler string-match
- 余下：ad_auth.go / local_auth.go / ppt_file_service.go remaining / dashboard 等

**commit**: `f4291f5`

### D1-D4 总验证

| 指标 | 结果 |
|------|------|
| build + vet | ✅ 0 错误 |
| -race 测试 (10 包) | ✅ 全绿（services + handlers + scheduler + auth + utils + middleware + errors + cmd/server + storage + auth/hlstoken） |
| Phase 19 范围 <deferred> 列表 | ✅ 全部 4 项清除 |

### Phase 19 最终 commit 序列（合并 D1-D4 后 15 commits）

```
6edb772  W6   docs(19): Wave 6 summary + 范围对账
3d171de  W6   refactor(19/w6): STYLE-001 error 迁移
b08255d  W5e  test(19/w5e): ctx 取消传播 contract test
1ae6be0  W5d  perf(19/w5d): VideoFileService caller 全量 ctx 透传
7a5a1cc  W5c  perf(19/w5c): VideoFileService batch ops ctx-first
7828fc3  W5b  perf(19/w5b): VideoFileService ScanFiles chain ctx-first
e2b0b6b  W5a  perf(19/w5a): VideoFileService 内部 helpers ctx-first
34b07f7  W5   perf(19/w5): VideoRecordingTaskService ctx-first
2281927  W4   docs(19): Wave 4 section to 19-SUMMARY.md
9a00cbe  W4   refactor(19): Wave 4 ctx 级联 TaskServiceInterface 原子三元组
─────────────────────────────────────────────────────
20ee289  D1   refactor(19/d1): FOREIGN KEY strings.Contains → sentinel
3b2d41f  D2   refactor(19/d2): taskServiceAdapter 与 VideoRecordingTaskService 合并
1f0ec35  D3   refactor(19/d3): HLS jti 升级为 hls_jti_records 表
f4291f5  D4   refactor(19/d4): errors 包增量迁移 + DeleteFile/NYCodecryptor
```

最终 HEAD：`f4291f5`
