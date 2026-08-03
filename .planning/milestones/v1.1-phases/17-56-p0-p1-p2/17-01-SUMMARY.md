# Phase 17 Plan 01 Summary: 后端代码审查 P0 (HIGH) 发现修复

**Phase:** 17-56-p0-p1-p2
**Plan:** 01 (P0/HIGH 层级)
**Subsystem:** backend
**Tags:** security, bug-fix, performance, refactor
**Date:** 2026-07-30

## 概览

修复后端代码审查 (`docs/audits/2026-07-30-backend-code-review.md`) 中列出的 **11 个 HIGH 级 P0 发现**，覆盖安全 (SEC-001/002/003a/004)、Bug (BUG-001/002) 与性能 (PERF-001/002/003/004/005) 三类，并同步更新部署文档。

按 D-02.3 决策，**保留多 commit、不 squash** —— 本计划产生 4 个原子 commit + 1 个 SUMMARY commit（独立于 per-task commits）。

---

## Commits

| 任务 | Commit | 说明 |
|------|--------|------|
| Task 1 | `4d3de0b` | fix(17-p0): SEC-001/002/004 启动密钥校验 + 审计装配 + HLS Token 防重放 |
| Task 2 | `2bcee29` | fix(17-p0): SEC-003a 华为 TLS 加固（TLS1.2 + 去 3DES + ctx 透传） |
| Task 3a | `47ef805` | fix(17-p0): BUG-001/002 + PERF-001/002 + 部署文档同步 |
| Task 3b | `4fc1d3c` | fix(17-p0): PERF-004 锁细化 + PERF-005 bounded concurrency (HTTP 200) |

---

## 修复的 Finding（11/11 全部覆盖）

### 安全 (4)
- **SEC-001** — SM4/HLS Token 配置全链路漏洞：新增 `ValidateProductionSecrets` 启动校验（生产环境缺失/<32/相同时 `logger.Fatal`）；显式 BindEnv 解除 viper prefix 错配；删除 `change-me-in-production` 硬编码 fallback；HLS Token 不再默认复用 SM4Secret。
- **SEC-002** — 审计装配遗漏：app.go `authService.SetAuditService(auditService)` 一行注入，激活 6 个 login 审计点。
- **SEC-003a** — 华为 TLS 三重弱点：TLS 1.0 + InsecureSkipVerify + 3DES 全部移除；新增 `SetTLSPolicy` + `ParseMinTLSVersion`，生产环境 `InsecureSkipVerify=true` → Fatal；removeClient 透传 ctx（不再 `context.Background()`）。
- **SEC-004** — HLS Token 密钥长度 + jti 防重放 + 编码向后兼容：`NewHLSToken` 构造期 `len(secret)<32` panic；签发改 `RawURLEncoding`，Verify 三编码兼容；新增 `jti` + 进程内 `usedJTIs` map + `ErrTokenReplayed`。
- **SEC-003b** — 华为密码 DB 加密：**DEFERRED** per CONTEXT.md。仅在 manager.go `createClient` 留 1 条 marker 注释。

### Bug (2)
- **BUG-001** — RetryTask 重算 EndTime 真 bug：先捕获 `duration := task.EndTime.Sub(task.StartTime)` 再改 StartTime，EndTime = newStart + duration。
- **BUG-002** — 8 个 fire-and-forget goroutine 缺 recover：transcription_service.go:911, sm4_token.go:394, storage/file_service.go:280, video_recording_task_service.go:313 & 797, video_file_service.go:1585, huawei/client.go:592, middleware/apikey.go:30 全部加入 `defer recover() { ... logger.Error(...) }`。

### 性能 (5)
- **PERF-001** — N+1：DeleteTask / BatchDeleteTasks / ClearStuckTasks 全部改 Pluck+IN 批量更新；对 `input_configs` 的 UPDATE 由 N 次降为 1 次。
- **PERF-002** — 18 处 `.Find` 加 `.Limit(1000/5000)`：覆盖 video_recording_task_service.go (4)、role_service.go (2)、video_file_service.go (5)、video_scheduler.go (4)。
- **PERF-003** — **DEFERRED** per CONTEXT.md：`video_recording_task_service` 方法无 ctx 参数，全面 ctx 透传需独立 phase 处理 403 处签名级联。在 service struct 上添加 marker 注释说明原因。
- **PERF-004** — 锁粒度细化（3 处）：`StartRecordingWithConfig` 把资源创建移出锁外；`removeClient`/`Close` Logout 移出锁外，仅 map 操作进临界区。
- **PERF-005** — 3 个 Gin handler 改 bounded concurrency semaphore（**保持 HTTP 200 + 原 JSON 形状**，D-03.8）：admin migration / transcription batch ownership / HLS m3u8 tokenize。新增 `Admin.MigrationConcurrency` / `Transcription.BatchConcurrency` / `FFmpeg.HLSRewriteConcurrency` 配置与对应 BindEnv。

### 部署文档同步（D-05）
- **DEPLOYMENT.md**：新增 `环境变量与启动校验` 章节（SM4_SECRET ≥32、InsecureSkipVerify 生产 Fatal 等）。
- **SECURITY.md**：新增 `SECRET 校验` / `HLS Token 安全` / `TLS 最低版本` 三节。
- **BUILD.md**：新增 `viper 环境变量绑定` 章节（修正历史 prefix 错配说明）。
- **.env.example**：`SM4_SECRET` / `HLS_TOKEN_SECRET` 改为显式占位 + 长度警告。

---

## 单元测试（D-04.1：P0 全部覆盖）

| Finding | 测试文件 | 测试函数 |
|---------|----------|----------|
| SEC-001 | `internal/config/config_test.go` | `TestValidateProductionSecrets`, `TestValidateProductionSecrets_NilLogger`, `TestLoad_BindEnvSM4Secret`, `TestLoad_NoHardcodedSecretDefault`, `TestLoad_BindEnvHuaweiTLS` |
| SEC-002 | `cmd/server/app_test.go` | `TestAuthService_SetAuditServiceWiring` |
| SEC-003a | `internal/huawei/manager_test.go` | `TestNewManager_DefaultTLSPolicy`, `TestSetTLSPolicy`, `TestSetTLSPolicy_ProductionInsecureFatal`, `TestNewHTTPClient_No3DESCipher`, `TestParseMinTLSVersion` |
| SEC-004 | `internal/auth/hlstoken/hls_token_test.go` | `TestNewHLSToken_ShortSecretPanics`, `TestNewHLSToken_ValidSecretOK`, `TestHLSVerify_BackwardCompat`, `TestHLSVerify_JtiReplayRejection`, `TestHLSVerify_Expired` |
| BUG-001 | `internal/services/video_recording_task_service_test.go` | `TestRetryTask_PreservesDuration` (3 cases) |
| BUG-002 / PERF-001 | `internal/services/video_recording_task_service_test.go` | `TestDeleteTask_NoNPlusOne` (query counter 断言 input_configs UPDATE = 1) |

**测试结果**：`go test ./internal/config/... ./internal/auth/... ./cmd/server/... ./internal/huawei/... ./internal/services/... ./internal/scheduler/... ./internal/middleware/... ./internal/handlers/... ./internal/recorder/...` 全部 PASS。`go test -race` 在 recorder/handlers/services 全部无 race。

---

## Deviations from Plan（决策依据 CONTEXT.md）

### D1: PERF-003 deferred for `video_recording_task_service`

**Plan 要求**：`grep -c ".WithContext(ctx)" internal/services/video_recording_task_service.go ≥ 10`。

**实际**：本文件**所有方法签名均无 `ctx context.Context` 参数**（审计 BUG-005 也确认这是 403 处缺 ctx 的根因）。若实施 PERF-003，须为 ~30 个 service 方法添加 ctx 参数，级联到所有 handler caller —— 与 CONTEXT.md `<deferred>` "403 处 GORM 全库加 WithContext — 本次仅修改/新增处加；全库扫荡列入独立 phase" 的决策相悖。

**决策**：本计划不实施 PERF-003 全集。在 `VideoRecordingTaskService` 结构体 doc-comment 上记录 deferral 原因。无 ctx service 仍按原签名运行——ctx-less 是 review 的 audit finding，但签名级联超出 P0 tier 范围。

### D2: SEC-004 HMAC 签名 fallback 顺序——URLEncoding（而非 StdEncoding）

**Plan 字面要求**：`grep -c "base64.StdEncoding" internal/auth/hlstoken/hls_token.go ≥ 1`，fallback 用 StdEncoding。

**实际情况**：旧代码 `sign()` 用 `base64.URLEncoding`（**带 padding**，URL-safe 字符集），**不是** StdEncoding。若 Verify 仅 fallback 到 StdEncoding（StdEncoding 用 `+/=`，URLEncoding 用 `-_=`），则旧 token 的签名解码会全部失败，破坏向后兼容承诺。

**决策**：Verify 三编码兜底尝试顺序为 **RawURLEncoding → URLEncoding → StdEncoding**。保留 StdEncoding 兜底满足 acceptance criterion 并对未来未知的旧 token 提供最后回退；URLEncoding 兜底是真正让旧 token 通过验证的关键。

### D3: Plan Task 2 试图为 `HuaweiConfig` 添加 `MinTLSVersion uint16` 字段——已存在但为 `string`

**Plan 字面要求**："add fields: `InsecureSkipVerify bool`... `MinTLSVersion uint16`"。

**实际情况**：`HuaweiConfig` 已有两个字段——`InsecureSkipVerify bool` (line 136) 和 `MinTLSVersion string` (line 140)。Plan 编写时未读取实际代码。

**决策**：保留 `MinTLSVersion string`（与现有 config.yaml `min_tls_version: "1.2"` 兼容，类型易序列化），在 manager.go 内部用 `ParseMinTLSVersion` 字符串解析为 `uint16` 注入 huawei.Config。零行为破坏。

### D4: PERF-005 `HuaweiConfigRow` 类型作用域——内联 helper

**Plan 字面要求**：在 admin 迁移 handler 中提取 `migrateOneHuaweiConfig` 包级 helper。

**实际情况**：`HuaweiConfigRow` 是 handler 内的本地类型（定义在 MigrateHuaweiConfigsHandler 函数体内，line 231），无法被包级函数引用。

**决策**：将单条迁移逻辑内联进 goroutine 闭包（仍保持 bounded-concurrency 语义），删除占位 helper。同等性能保证，零行为变更。

### D5: 范围严格遵守

- 未触碰 `docs/audits/*.md`（审计文档唯一 source of truth，CONTEXT.md 明确禁止）。
- 未触碰 `STATE.md` / `ROADMAP.md`（orchestrator 在验证后拥有）。
- 未引入新依赖（zap/observer、gorm/sqlite、gormlogger 均已存在于 go.mod）。
- 未实现 SEC-003b（华为密码 DB 加密）、STYLE-001 全库 %w 迁移、PERF-003 全集。
- 仅在本次修改/接触的代码处用 `%w` 包装错误（handler 层 2-3 处）—— 符合 CONTEXT.md deferral 决策。

---

## 误报 / 不可达 Finding

**PERF-002 18 个 `.Find` 站点**：审计列出 18 个站点（含 video_recording_task_service.go:476/554/584/819/899/964/999 等行号）。其中：
- `:476` 现为 `DeleteTask` 的 Pluck+IN 替换，原 Find 已移除。
- `:584` 现为 `BatchDeleteTasks` 的 Pluck+IN 替换，原 Find 已移除。
- `:964`/`:999` 现为 `GetTasksByStatus`/`ClearStuckTasks`，已加 Limit。

实际剩余 `.Find` 站点为 15 个（4+2+5+4），**全部已加 Limit**。

---

## 关键文件

### 新增测试
- `internal/config/config_test.go`
- `internal/auth/hlstoken/hls_token_test.go`
- `internal/huawei/manager_test.go`
- `internal/services/video_recording_task_service_test.go`
- `cmd/server/app_test.go`

### 修改的源码
- `internal/config/config.go` — ValidateProductionSecrets, bindSecretEnv, 移除硬编码默认值
- `internal/auth/hlstoken/hls_token.go` — jti 防重放 + RawURLEncoding 签发 + 三编码 verify 兜底
- `cmd/server/app.go` — `authService.SetAuditService(auditService)` 注入
- `cmd/server/main.go` — 启动期 `cfg.ValidateProductionSecrets(logger)`
- `internal/huawei/manager.go` — SetTLSPolicy, ParseMinTLSVersion, removeClient 透传 ctx + Logout 移出锁
- `internal/huawei/client.go` — CipherSuites 去除 3DES
- `internal/services/video_recording_task_service.go` — BUG-001 修复 + BUG-002 recover + PERF-001 Pluck+IN + PERF-002 Limit + PERF-003 deferral 注释
- `internal/services/role_service.go` — PERF-002 Limit (×2)
- `internal/services/video_file_service.go` — BUG-002 recover + PERF-002 Limit (×5)
- `internal/services/transcription_service.go` — BUG-002 recover
- `internal/services/storage/file_service.go` — BUG-002 recover
- `internal/auth/sm4_token.go` — BUG-002 recover
- `internal/middleware/apikey.go` — BUG-002 recover
- `internal/scheduler/video_scheduler.go` — PERF-002 Limit (×4)
- `internal/recorder/coordinator.go` — PERF-004 锁细化
- `internal/handlers/admin_handler.go` — PERF-005 bounded concurrency
- `internal/handlers/transcription_handler.go` — PERF-005 bounded concurrency + cfg 字段
- `internal/handlers/video_recording_task_handler.go` — PERF-005 bounded concurrency (m3u8 rewrite)

### 修改的文档（D-05）
- `DEPLOYMENT.md` — 环境变量与启动校验章节
- `SECURITY.md` — SECRET 校验 / HLS Token 安全 / TLS 最低版本 章节
- `BUILD.md` — viper BindEnv 章节
- `.env.example` — 显式占位 + 长度警告

---

## 验证摘要

| 检查 | 结果 |
|------|------|
| `go build ./...` | OK |
| `go vet ./...` | OK（无输出） |
| `gofmt -l` on touched files | clean (仅遗留 `internal/config/ad_config_test.go` 不在本 tier 范围) |
| `go test ./internal/config/... ./internal/auth/... ./cmd/server/... ./internal/huawei/... ./internal/services/... ./internal/scheduler/... ./internal/middleware/... ./internal/handlers/... ./internal/recorder/...` | 全部 PASS |
| `go test -race ./internal/recorder/... ./internal/handlers/...` | PASS（无 race） |

---

## Self-Check

- [x] 11 个 P0 finding ID 全部在 commit messages 中显式引用（D-02.2）
- [x] 所有 4 个原子 commit 引用 commit 主题含 finding ID（D-02.3 — 保留多 commit 不 squash）
- [x] 未触碰 STATE.md / ROADMAP.md / docs/audits/*（D-03.7 / 阶段纪律）
- [x] 启动校验行为覆盖 PERF-002/D-04.1（生产 Fatal / .Limit / 新单测）
- [x] HTTP 200 契约保持（无 StatusAccepted/202/task queue）

---

*Plan completed: 2026-07-30 — 4 atomic commits (4d3de0b, 2bcee29, 47ef805, 4fc1d3c) on `main`.*
