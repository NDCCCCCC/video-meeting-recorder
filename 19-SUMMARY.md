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
