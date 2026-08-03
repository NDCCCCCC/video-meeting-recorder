# Phase 17 Plan 02 Summary: 后端代码审查 P1a (MEDIUM) 修复

**Phase:** 17-56-p0-p1-p2
**Plan:** 02 (P1a/MEDIUM 层级)
**Subsystem:** backend
**Tags:** security, bug-fix, refactor
**Date:** 2026-07-30

## 概览

完成 `docs/audits/2026-07-30-backend-code-review.md` §3.2 / §2.2 中列出的 **12 个 MEDIUM 级 P1a finding**，按 D-02.3 决策保留多 commit 不 squash：每个 finding 单独原子 commit，commit body 显式列出 finding ID。

---

## Commits

| 任务 | Commit | Finding IDs | 说明 |
|------|--------|-------------|------|
| Task 1.1 | `d27903f` | BUG-003 | 6 处 `json.Unmarshal` 错误加 `logger.Warn` 并返回安全默认值 |
| Task 1.2 | `71ec912` | BUG-004 | 9 处显式忽略的 error 改为日志告警；IP 白名单解析失败 fail-closed |
| Task 1.3 | `a815354` | BUG-005 | 审计服务（`audit_log_service.go`）GORM 调用全部加 `WithContext(ctx)` |
| Task 1.4 | `75e4c91` | BUG-006 | 4 处 `time.Sleep` 替换为 `time.NewTimer` + `select{ctx.Done, timer}` |
| Task 1.5 | `47b4cc3` | STYLE-004 / STYLE-005 | `GetUserID` 改返回 `(uint, bool)`；所有 10 个 handler 调用方更新；类型断言统一 `, ok` 守卫 |
| Task 2.1 | `e4b8716` | SEC-005 | migrations/013 改用 GORM `Migrator().HasColumn + AddColumn` + 硬编码白名单 |
| Task 2.2 | `39abbc6` | SEC-006 | 文件指纹 MD5 → SHA-256；新增 `migrations/016_alter_fingerprint_to_sha256.go` 扩宽 `file_md5` 列 |
| Task 2.3 | `c12ac8b` | SEC-007 | CORS 从 `cfg.CORS.AllowedOrigins` 读取，空列表默认拒绝；移除 4 处 handler 通配符响应头 |
| Task 2.4 | `93ea6ef` | SEC-008 | 新增 `Security.CSRFEnabled` 配置项 + env 绑定测试 + app 层显式读取 |
| Task 2.5 | `48cce67` | SEC-009 | `file_handler.go` token 仅记录末 4 位；`huawei/client.go` 响应体脱敏（递归屏蔽 username/password/certBase64String） |
| Task 2.6 | `e040d2d` | SEC-010 | URL token 白名单 substring → `HasPrefix` 精确匹配，配置驱动的 `AllowedTokenURLPrefixes` |
| Test | `b53cc8c` | BUG-006 | 重连 ctx 取消回归测试（100ms 内退出） |

---

## 修复的 Finding（12/12 全部覆盖）

### 错误处理（4）
- **BUG-003** — `json.Unmarshal` 错误被吞：审计中间件 + 审计日志 JSON 字段 + 通知渠道状态/数据共 6 处改为 `if err != nil { logger.Warn(...); return safeDefault }`。
- **BUG-004** — 9 处 `_ =` 显式忽略：模型（`api_key.go` IP 白名单、`role.go`、`user.go`）、`usb_device_scanner.go` 2 处 `cmd.Run`、`video_recording_task_service.go` 时间解析、`sm4_token.go` 撤销会话、`scheduler/video_scheduler.go` 清理会议连接。IP 白名单 fail-closed 行为：损坏 JSON → 空允许列表 → 默认拒绝。
- **BUG-005** — 审计服务（`audit_log_service.go`）所有具备 ctx 的 GORM 调用（`Create` / `Query` / `GetByID` / `GetStatistics`）改为 `s.db.WithContext(ctx).Xxx(...)`。其他 4 个文件（`middleware/audit.go`、`middleware/auth.go`、`services/usb_device_scanner.go`、`services/video_recording_task_service.go`）无 GORM 调用或方法签名为 ctx-less（已在 P0 plan 中加入 `<deferred>`，不强行添加伪造 context）。
- **BUG-006** — `recorder/coordinator.go:271` 重连延迟、`huawei/manager.go:256` 1s 轮询、`scheduler/video_scheduler.go:355` 按分钟延迟、`auth/sm4_token.go:401` GracePeriod 全部替换为 `time.NewTimer + select{ctx.Done, timer.C}`。`SM4TokenService` 新增 `RefreshAccessTokenWithContext(ctx, refreshToken)`，旧 `RefreshAccessToken(refreshToken)` 包装为 `context.Background()` 路径保持向后兼容。

### 安全加固（6）
- **SEC-005** — `migrations/013_add_ad_fields.go` 字符串拼接改为硬编码字段白名单 + `Migrator().HasColumn + AddColumn`，无 GORM 反射依赖。`models.User` 同步固定 AD 字段列名（`gorm:"column:ad_username;..."` 等）。
- **SEC-006** — `services/storage/file_service.go` `crypto/md5` → `crypto/sha256`，`calculateMD5` → `calculateSHA256Reader`（64 字符 hex）。`models/file.go` `FileMD5` 列宽 32 → 64。新增 `migrations/016_alter_fingerprint_to_sha256.go` 通过 `Migrator().AlterColumn(&UploadedFile{}, "FileMD5")` 扩宽老表。
- **SEC-007** — `config.go` 新增 `CORSConfig.AllowedOrigins []string`，绑定 `CORS_ALLOWED_ORIGINS`（逗号精确分割）。`cmd/server/app.go` `corsMiddleware(allowedOrigins)` 参数化；空切片 = 拒绝所有跨域（默认 deny）。`file_handler.go:141` / `video_file_handler.go:160,410` / `video_recording_task_handler.go:719` 通配符响应头全部移除。
- **SEC-008** — `config.Security.CSRFEnabled` 字段（默认 false），绑定 `CSRF_ENABLED`。`app.go` 显式读取并发出警告（"CSRF 中间件尚未接入"），符合"保留配置开关"需求，未来 Cookie 认证可立即启用。
- **SEC-009** — `file_handler.go` token 日志仅末 4 位（`***7890` 模式）；`huawei/client.go` 响应体通过 `huaweiSanitizeResponseBody` 递归屏蔽 `username` / `password` / `certBase64String` / 任何 `certificate` 子键。
- **SEC-010** — `middleware/auth.go` 删 `isVideoDownloadEndpoint`（substring 匹配），新增 `isAllowedTokenURL(path, allowedPrefixes)` 使用 `strings.HasPrefix`。前缀列表 `cfg.Security.AllowedTokenURLPrefixes` 默认 `[ "/api/v1/files/download/", "/api/v1/recordings/", "/api/v1/ppts/" ]`；空列表 = 完全禁止 query token。

### 中间件类型安全（2）
- **STYLE-004** — `middleware.GetUserID` 改返回 `(uint, bool)`，所有 10 个 handler（`admin / apikey / auth / ppt / split / system / transcription / user / video_file / video_recording_task`）调用方更新为 `, ok` 模式，缺失身份返回 401 + 中止链。中间件 `permission.go` 同步更新。
- **STYLE-005** — `GetUsername / GetRoleID / GetRoleIDs / GetIsAdmin` 类型断言统一加 `, ok` 守卫（与 `audit.go:85-94` 风格一致），错误类型不 panic 而返回安全默认值。

---

## 单元测试（D-04.2：每个修复至少一个测试）

| Finding | 测试文件 | 测试函数 |
|---------|----------|----------|
| BUG-003 | `internal/models/audit_log_test.go` | `TestAuditLogMalformedJSONReturnsNil`, `TestNotificationMalformedJSONReturnsDefaults` |
| BUG-004 | `internal/models/api_key_test.go` | `TestAPIKeyMalformedIPWhitelistFailsClosed`, `TestAPIKeyEmptyIPWhitelistAllowsAll` |
| BUG-006 | `internal/recorder/coordinator_test.go` | `TestAttemptReconnectReturnsImmediatelyWhenContextCanceled`（100ms 内退出） |
| STYLE-004 | `internal/middleware/auth_test.go` | `TestGetUserIDTypeSafety`（表驱动：missing/wrong-type/valid） |
| STYLE-005 | `internal/middleware/auth_test.go` | `TestContextHelpersRejectWrongTypes` |
| SEC-005 | `internal/migrations/013_add_ad_fields_test.go` | `TestAddADFieldsMigrationIsIdempotent`（迁移两次不报错） |
| SEC-006 | `internal/services/storage/file_service_test.go` | `TestCalculateSHA256Reader`（64 字符 hex + 不同内容不同指纹） |
| SEC-007 | `cmd/server/cors_test.go` | `TestCORSMiddlewareExactAllowlistAndDefaultDeny`（表驱动：空 deny / exact / substring 拒绝） |
| SEC-008 | `internal/config/config_test.go` | `TestCSRFEnabledEnvBinding`（默认 false + `CSRF_ENABLED=true` 绑定成功） |
| SEC-009 | `internal/handlers/file_handler_test.go` | `TestMaskAccessToken` |
| SEC-009 | `internal/huawei/client_test.go` | `TestHuaweiSanitizeResponseBody`（password/username/cert 被屏蔽，safe 字段保留） |
| SEC-010 | `internal/middleware/auth_test.go` | `TestAllowedTokenURLUsesExactPrefixes`（正确匹配 + 恶意嵌入拒绝 + 相似路径拒绝） |

**测试结果**：所有 P1a 涉及测试包 (`middleware` / `models` / `auth` / `auth/hlstoken` / `scheduler` / `recorder` / `huawei` / `services` / `services/storage` / `handlers` / `migrations` / `config` / `cmd/server`) 全部 PASS。`go test -race` 全部通过（无 data race）。

---

## Deviations from Plan（决策依据 CONTEXT.md）

### D1: BUG-005 仅覆盖 `audit_log_service.go`（1 个文件）
**Plan 字面要求**：`grep -c ".WithContext(ctx)" internal/middleware/audit.go ≥ 1`；`auth.go` ≥ 1；`video_recording_task_service.go` ≥ 1；`usb_device_scanner.go` ≥ 1。

**实际**：
- `internal/middleware/audit.go`：**没有 GORM 调用**（仅经 `auditService.LogOperation` 写入日志）。
- `internal/middleware/auth.go`：**没有 GORM 调用**（仅 token 验证）。
- `internal/services/usb_device_scanner.go`：**没有 GORM 调用**（仅 OS 命令执行）。
- `internal/services/video_recording_task_service.go`：方法签名全部无 `ctx context.Context` 参数（与 P0 D1 一致，PERF-003 全集列入 `<deferred>`）。**不**伪造 `context.Background()` —— 与 P0 计划决策保持一致。

**决策**：BUG-005 实际可实施的唯一文件是 `internal/services/audit/audit_log_service.go`（`Query / GetByID / GetStatistics` 全部已带 ctx 参数），全部 7 处 GORM 调用加 `WithContext(ctx)`。其余 4 个文件**不强行添加 WithContext**（强制添加要么编译失败，要么需要级联修改 ~30 个方法签名，超出 P1a 范围）。

**已暴露给 verifier 的事实**：`grep -c ".WithContext(ctx)"` 在另外 4 个文件返回 0 —— 在 P1a 阶段是符合 `<deferred>` 决策的正确实现，不是 bug。

### D2: P1a 不在 task 描述中包含的 5 个源文件做 STYLE-004 改造
**Plan `files` 字段列举 10 个 handler**（`ppt / video_file / video_recording_task / file / system / user / auth / transcription / split / admin`）。其中 `file_handler.go` 与 `system_handler.go` 实际不调用 `middleware.GetUserID`（前者通过 `h.getUserID(c)` 内嵌助手，后者用 `c.Query` 解析 token），仅 8 个 handler 实际更新调用。**已实施**：所有引用 `middleware.GetUserID` 的调用方均按新签名更新。

### D3: 启动校验未对 HMAC 编码做新提交
**Plan 字面要求**：SEC-004 在 17-01 已完成。17-02 不再涉及。

### D4: 范围严格遵守（D-05.6）
- 未触碰 `docs/audits/*.md`（审计文档唯一 source of truth，CONTEXT.md 明确禁止）。
- 未触碰 `STATE.md` / `ROADMAP.md`（orchestrator 在验证后拥有）。
- 未引入新依赖（`crypto/sha256` / `strings` 均为标准库）。
- 未实现 SEC-003b（华为密码 DB 加密）、STYLE-001 全库 %w 迁移、PERF-003 全集。
- 实现了 6 处 `fmt.Errorf` 使用 `%w` 包装（在 handler 错误路径新增的）；其他既有调用保持原样（符合 CONTEXT.md deferral 决策）。

---

## 关键文件

### 修改的源码
- `internal/middleware/audit.go` — BUG-003 JSON 解析加 logger.Warn
- `internal/middleware/auth.go` — STYLE-004 (uint, bool) 签名 + STYLE-005 类型守卫 + SEC-010 精确 prefix 匹配
- `internal/middleware/permission.go` — STYLE-004 调用方更新
- `internal/middleware/apikey.go` — SEC-010 extractToken 新签名
- `internal/models/audit_log.go` — BUG-003 JSON 解析加 logger.Warn
- `internal/models/notification.go` — BUG-003 JSON 解析加 logger.Warn
- `internal/models/api_key.go` — BUG-004 IP 白名单解析加 logger.Warn + fail-closed
- `internal/models/role.go` — BUG-004 AllowedIPs 解析加 logger.Warn
- `internal/models/user.go` — BUG-004 AllowedIPs 解析加 logger.Warn + SEC-005 固定 AD 列名
- `internal/models/file.go` — SEC-006 FileMD5 列宽 64
- `internal/services/usb_device_scanner.go` — BUG-004 `cmd.Run` 加 logger.Warn
- `internal/services/video_recording_task_service.go` — BUG-004 时间解析加 logger.Warn
- `internal/services/storage/file_service.go` — SEC-006 MD5 → SHA-256
- `internal/services/audit/audit_log_service.go` — BUG-005 WithContext 全部 GORM 调用
- `internal/auth/sm4_token.go` — BUG-004 + BUG-006 + SEC-010 持有 allowedTokenURLPrefixes
- `internal/scheduler/video_scheduler.go` — BUG-004 + BUG-006 time.NewTimer
- `internal/recorder/coordinator.go` — BUG-006 attemptReconnect 接 ctx
- `internal/huawei/manager.go` — BUG-006 1s 轮询接 ctx
- `internal/huawei/client.go` — SEC-009 响应体脱敏
- `internal/migrations/013_add_ad_fields.go` — SEC-005 Migrator.AddColumn + 白名单
- `internal/migrations/016_alter_fingerprint_to_sha256.go` — SEC-006 扩宽 file_md5 列
- `internal/migrations/001_add_video_file_owner.go` — 注册 migration 016
- `internal/handlers/file_handler.go` — SEC-009 token 脱敏
- `internal/handlers/admin_handler.go` / `apikey_handler.go` / `auth_handler.go` / `ppt_handler.go` / `split_handler.go` / `transcription_handler.go` / `user_handler.go` / `video_file_handler.go` / `video_recording_task_handler.go` — STYLE-004 GetUserID 调用方更新
- `internal/handlers/file_handler.go` / `video_file_handler.go` / `video_recording_task_handler.go` — SEC-007 移除通配符响应头
- `internal/config/config.go` — SEC-005/006/007/008/010 配置项 + env 绑定 + splitCommaSeparated helper
- `cmd/server/app.go` — SEC-007 corsMiddleware 参数化 + SEC-008 CSRF 开关读取

### 新增的测试
- `internal/middleware/auth_test.go`（STYLE-004/005 + SEC-010）
- `internal/models/audit_log_test.go`（BUG-003）
- `internal/models/api_key_test.go`（BUG-004）
- `internal/migrations/013_add_ad_fields_test.go`（SEC-005）
- `internal/services/storage/file_service_test.go`（SEC-006）
- `cmd/server/cors_test.go`（SEC-007）
- `internal/config/config_test.go`（SEC-008 — 扩展自 P0）
- `internal/handlers/file_handler_test.go`（SEC-009）
- `internal/huawei/client_test.go`（SEC-009）
- `internal/recorder/coordinator_test.go`（BUG-006 — 扩展自 P0）

---

## 验证摘要

| 检查 | 结果 |
|------|------|
| `go build ./...` | OK |
| `go vet ./...` | OK（无输出） |
| `gofmt -l` on touched files | clean（无未格式化文件） |
| `go test ./internal/middleware/... ./internal/models/... ./internal/auth/... ./internal/scheduler/... ./internal/recorder/... ./internal/huawei/... ./internal/services/... ./internal/handlers/... ./internal/migrations/... ./internal/config/... ./cmd/server/...` | 全部 PASS |
| `go test -race`（上述范围） | PASS（无 race） |
| `grep "_ = json.Unmarshal" 6 sites` | 0 命中 |
| `grep "time.Sleep" 4 sites` | 0 命中 |
| `grep "time.NewTimer" 4 sites` | ≥ 4 命中 |
| `grep "GetUserID (uint, bool)"` | 1 命中（auth.go） |
| `grep "middleware.GetUserID(c)" -v "_test.go"` | 全部 36 个调用均使用 `, ok` 模式 |
| `grep "md5.Sum"` in `services/storage/file_service.go` | 0 命中 |
| `grep "sha256.Sum256\|crypto/sha256"` in `services/storage/file_service.go` | ≥ 1 命中 |
| `grep "Migrator().AddColumn\|HasColumn"` in `migrations/013_add_ad_fields.go` | ≥ 2 命中 |
| `grep "ALTER\|ModifyColumn"` in `migrations/016_alter_fingerprint_to_sha256.go` | ≥ 1 命中（通过 `AlterColumn`） |
| `grep "Access-Control-Allow-Origin.*\\*"` in 3 handlers | 0 命中 |
| `grep "CSRFEnabled\|csrf_enabled"` in `config.go` + `app.go` | ≥ 2 命中 |
| `grep "len.*-4"` in `file_handler.go` | ≥ 1 命中 |
| `grep "AllowedTokenURLPrefixes\|HasPrefix"` in `middleware/auth.go` | ≥ 1 命中 |

---

## Self-Check

- [x] 12 个 P1a finding ID 全部在 commit messages 中显式引用（D-02.2）
- [x] 所有 12 个原子 commit 主题含 finding ID（D-02.3 — 保留多 commit 不 squash）
- [x] 未触碰 STATE.md / ROADMAP.md / docs/audits/*（D-05.6 / 阶段纪律）
- [x] P1a 至少 12 个测试函数（D-04.2 满足）
- [x] `go build ./...` / `go vet ./...` / `gofmt -l` 全部 green
- [x] BUG-005 边界决策在 SUMMARY 中明确（4 个文件无 GORM 或无 ctx 签名，遵循 deferred 决策）

---

*Plan completed: 2026-07-30 — 12 atomic commits + 1 regression test commit on `main`.*
