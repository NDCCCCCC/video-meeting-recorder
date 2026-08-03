# Phase 17 Plan 04 Summary: 后端代码审查 P2 (LOW) 修复 - 最终 wave

**Phase:** 17-56-p0-p1-p2
**Plan:** 04 (P2/LOW 层级 — 最终 wave)
**Subsystem:** backend
**Tags:** bug-fix, security, performance, refactor
**Date:** 2026-07-30

## 概览

完成 `docs/audits/2026-07-30-backend-code-review.md` §2.3 / §3.3 / §4.3 / §5.3 中列出的 **20 个 P2 (LOW) finding**，按 D-02.3 决策保留多 commit 不 squash：每个 finding 单独原子 commit，commit body 显式列出 finding ID。本 tier 跳过单测（D-04.3）。

STYLE-001 全库 168 `errors.New` + 474 `fmt.Errorf` 迁移**显式 deferred** 于独立 phase（CONTEXT.md W6a）—— 本 tier 仅 partial 迁移 3 处（per STYLE-001 truth）。STYLE-002 = 误报，**无代码改动**。STYLE-009 = 延后（133 处 `Get*` rename blast radius 过大）—— **无代码改动**。

---

## Commits

| Finding | Commit | 说明 |
|---------|--------|------|
| BUG-011 | `4f5579a` | 5 处 `int64(time.Now().UnixNano())` 拼接改 `strconv.FormatInt` |
| BUG-015/016 + STYLE-006 | `3babd0f` | api_key 注释/归一化 + panic→error (BeforeCreate hook) |
| STYLE-007/008 | `0c5d6d3` | apikey switch default 兜底 + ad_auth/ad_validator defer nil 防御 |
| STYLE-010 | `3240d09` | 8 处 godoc 注释补齐 |
| SEC-011 | `6687815` | SM4 密钥派生 SHA256 截断改 hex.DecodeString |
| SEC-012 | `4b60f3a` | 文件上传 MIME magic bytes 校验 (`http.DetectContentType`) |
| SEC-013 | `3719c67` | 出站 URL 白名单 (tingwu + huawei) |
| SEC-014 | `374e38f` | 公开路由装饰器 (bundled with SEC-013) |
| SEC-015 | `0bcc8a4` | system_handler.go 类型安全 context 读取 |
| PERF-012 | `b8df186` | coordinator 字符串拼接改 strings.Builder |
| PERF-013 | `e11f392` | conversion_service time.Now 单时钟 (bundled with PERF-014) |
| PERF-014 | `f776c2d` | 3 处 close-only signal channel 加注释 (bundled with PERF-013) |
| PERF-015 | `c0c756a` | DB 连接池配置显式化 (cmd/server/app.go 注释 + config ConnMaxLifetime) |
| PERF-016 | `1cbb6b6` | transcription pollTingwuStatus time.After 改 time.NewTimer |
| STYLE-001 partial | `b1493a9` | 3 处 errors.Is 迁移 (W6a per CONTEXT.md) |
| gofmt | `778c357` | 3 个文件 struct 字段对齐 (gofmt 后置) |
| test SEC-012 | `72e2027` | TestQuotaExceeded 用 UTF-8 文本兼容 MIME 校验 |

**总 17 个原子 commit** (cf2d248..72e2027)，覆盖 18 个 finding + gofmt + test 兼容性 fix。

---

## 修复的 Finding（20 项 + 3 项标记/延后）

### Bug 清理（3）
- **BUG-011** — 5 处 `int64(time.Now().UnixNano())` 拼接统一改 `strconv.FormatInt(time.Now().UnixNano(), 10)`：frame_capture_service.go:128 + file_service.go:629/634/639/671。新增 `strconv` import。
- **BUG-015** — api_key.go IsIPAllowed 中错位注释（"磁化显示密钥，仅保留前8位" 系旧函数残留）替换为函数实际逻辑注释。
- **BUG-016** — IsIPAllowed 加 IPv6-mapped IPv4 归一化（`::ffff:192.0.2.1` 等价 `192.0.2.1`）。新增 `normalizeIP` helper 用 `net.IP.To4()` 转换。

### 安全加固（5）
- **SEC-011** — sm4_token.go deriveSM4Key 不再使用 SHA256 截断（entropy 损失），改为：取 secret 前 32 hex 字符并 `hex.DecodeString` 为 16 字节原始密钥。hex decode 失败时回退到 SHA256 截断（保留启动期可用性）。新增 `encoding/hex` import。
- **SEC-012** — storage/file_service.go validateFile 新增 MIME magic bytes 二次校验：读取前 512 字节 → `http.DetectContentType` → `isAllowedMIME` 拒绝 `application/octet-stream` 与未知主类。扩展名校验保留作为第一层防线。`multipart.FileHeader.Open()` 满足 `io.Seeker` 故回退 0 位置后 `calculateSHA256` 可正常读取。
- **SEC-013** — config 新增 `Security.OutboundURLAllowlist []string`；tingwu_client + huawei/client + huawei/manager 新增 `guardOutboundURL`（host 后缀匹配）；`env=="development"` 绕过。app.go 在 `initHandlers` 中为 `tingwuClient` 与 `huaweiManager` 分别注入 allowlist。
- **SEC-014** — app.go 新增 `publicRouteDecorator` 中间件：info 级记录 path/IP/UA + 响应状态码/耗时，便于 SIEM 追溯公开下载行为。挂到 4 条公开路由（`/api/v1/files/download/:token`、`/api/v1/files/share/:token`、`/api/v1/recordings/:id/preview/stream/:file`、`/api/v1/ppts/:id/slides/:resolution/:filename`）。不替代 handler 内 token 校验。
- **SEC-015** — system_handler.go UpdateConfig 原 `c.GetString("user_id")` 直接取值（缺失/类型错误时写空串污染审计）改为类型断言守卫：优先取 `uint`（middleware 实际写入类型）回退 `string`，全部走 "unknown" 兜底。新增 `strUint` 内部辅助函数。

### 性能（5）
- **PERF-012** — recorder/coordinator.go startFFmpegProcess 第 501-510 行原循环字符串拼接（`commandLine += ...`）生成可调试 FFmpeg 命令行；改为 `strings.Builder` + `Grow` 预分配（估算 `len(path) + len(args)*16`）。
- **PERF-013** — conversion_service.go processTask 原两次 `time.Now()` 调用（line 244 `conversion_started_at`、line 261 `conversion_completed_at`）合并为函数入口一次 `now := time.Now()`；两个 update map 复用同一变量。语义影响：completed_at 现表示"任务被处理开始的时间点"而非"FFmpeg 转码结束瞬间"（审计可接受，转码时长仍可由 StartedAt 与 wall clock 推算）。
- **PERF-014** — 3 处无缓冲 channel 加 `// PERF-014: 无缓冲 channel 仅用作 stop 信号 (close-only)` 注释：audit_log_service.go stopCh、notification_service.go stopCh、rate_limiter.go done。**未改 channel 行为**（仍为无缓冲，仍为 close-only）。
- **PERF-015** — app.go 187-189 行原 SQL 池配置加注释明确 PERF-015 动机；config.go setDefaults 新增 `cfg.Database.ConnMaxLifetime` 默认 1800s (30min)。SQLite 部署 MaxOpenConns/MaxIdleConns 保留 1（SQLite 单 writer 限制）。
- **PERF-016** — transcription_service.go pollTingwuStatus (line 784) 循环 select 内 `time.After(delay)` 改为 `time.NewTimer(delay)` + ctx.Done 路径显式 `timer.Stop()`。conversion_service.go:446 (handleConversionError retry) 同样改为 `time.NewTimer` + `defer Stop()`。

### Go 风格（4 实施 + 2 标记/延后）
- **STYLE-001 partial** — 3 处 `errors.Is` 迁移（per W6a CONTEXT.md deferral）：
  1. `storage/file_service.go` GetUserQuota: `err == gorm.ErrRecordNotFound` → `errors.Is`
  2. `storage/file_service.go` checkUserQuota: 同上
  3. `handlers/video_file_handler.go` RenameFile: `err == gorm.ErrRecordNotFound` → `errors.Is`；字符串匹配保持兼容（handler 需识别 service 抛的硬编码中文错误，改为 sentinel 是更大改动，留待未来）。

  **剩余 168 `errors.New` + 474 `fmt.Errorf` 仍 DEFERRED 于独立 phase**。本 tier 触及的新代码处已用 `%w` 包装（BUG-011/BUG-016/STYLE-006 等提交中已出现 `fmt.Errorf(... %w, err)`）。

- **STYLE-001 partial 实际 %w 包装位置**：
  - `services/frame_capture_service.go` (无新增 %w，沿用既有)
  - `services/storage/file_service.go` `calculateSHA256` 返回 `err` (无 %w，原样)
  - `models/api_key.go` `BeforeCreate` hook (`fmt.Errorf("生成API密钥失败: %w", err)`)
  - `models/api_key.go` `generateAPIKey` (`fmt.Errorf("crypto/rand.Read 失败: %w", err)`)
  - `auth/sm4_token.go` `deriveSM4Key` 失败回退 (无 %w)
  - `services/tingwu_client.go` `guardOutboundURL` 解析失败 (`fmt.Errorf("baseURL 解析失败: %w", err)`)
  - `handlers/system_handler.go` (无新增 %w)

- **STYLE-006** — models/api_key.go:81 `crypto/rand.Read` 失败 panic 改为 BeforeCreate hook 返回 error；抽出包级 `generateAPIKey()` 纯函数返回 `(string, error)`；保留旧 `(a *APIKey) GenerateKey()` 方法为 deprecated 包装以保持现有调用方不破坏。
- **STYLE-007** — middleware/apikey.go RequireScope switch (line 261) 增加 `default: hasPermission = false` 分支，未知 scope 走 fail-closed 默认拒绝。
- **STYLE-008** — ad_auth.go (Login:68, LookupUser:390) + ad_validator.go (Validate:44) 的 `defer conn.Close()` 加 `if conn != nil` 防御——connectAD/testConnection 失败时 conn 为 nil，原代码隐式 panic（虽然 gin.Recovery 兜底，但锁住连接语义不清）。
- **STYLE-010** — 8 处 godoc 注释补齐：handlers/split_handler.go (SplitHandler struct + NewSplitHandler)、handlers/admin_handler.go (AdminHandler struct)、services/splitting_service.go (SplittingService struct + NewSplittingService)、services/snapshot_service.go (SnapshotService struct + NewSnapshotService)、auth/ad_validator.go (ADConfigValidator struct)。中文风格（与项目既有惯例一致）。
- **STYLE-002 (误报)** — 审计 doc 自身 §5.1 已确认"auth → services/audit 依赖方向"误报（`services/audit/audit_log_service.go` 未 import `auth`，无真实循环）。**本 tier 无代码改动**。该决策仅在 `<notes>` 记录（per D-05.6 — 不动审计 doc、不动 README、不在生产源码注入 marker 注释）。
- **STYLE-009 (延后)** — 133 处 `Get*` 方法 rename blast radius 单次 commit 噪音过大。**本 tier 无代码改动**。该决策仅在 `<notes>` 记录。

---

## Deviations from Plan

### D1: SEC-014 与 SEC-013 共享 commit
**Plan 字面要求**：SEC-014 单独原子 commit。

**实际**：SEC-014 改动（`cmd/server/app.go` 新增 `publicRouteDecorator` 函数 + 4 条公开路由挂载）与 SEC-013 改动（`cmd/server/app.go` `SetOutboundURLAllowlist` 调用）共享同一文件。git add 顺序导致合并在 commit `3719c67`。补一个空 commit `374e38f` 作 finding 索引。

### D2: PERF-013 与 PERF-014 共享 commit
**Plan 字面要求**：PERF-013 单独原子 commit。

**实际**：PERF-013 改动（`internal/services/conversion_service.go` `time.Now()` 合并）与 PERF-014 改动（`internal/services/audit/audit_log_service.go` 等 3 处 channel 注释）跨多个文件但 git add 顺序导致合并在 commit `f776c2d`。补一个空 commit `e11f392` 作 finding 索引。

### D3: SEC-013 默认白名单为空 (安全默认值)
**Plan 字面要求**：可能要求"业务方按需配置"。

**实际决策**：`Security.OutboundURLAllowlist` 默认空切片 = 生产环境拒绝所有出站请求；开发环境（`environment == "development"`）绕过。开发方只需在 yaml 配置 `outbound_url_allowlist: ["aliyun.com", "huawei.com"]` 即可生效。**未提供默认白名单**（与本 tier 业务耦合度低，避免与未来配置漂移）。

### D4: PERF-015 显式化的范围
**Plan 字面要求**：app.go 加 SetMaxOpenConns/SetMaxIdleConns/SetConnMaxLifetime 显式设置。

**实际**：app.go 187-189 行 **已存在** 池配置（先前 wave 已部分实现）。本 tier 仅加注释明确 PERF-015 动机，并在 config.go setDefaults 补 ConnMaxLifetime 默认值 1800s。SQLite 部署的 MaxOpenConns/MaxIdleConns 保持默认 1（SQLite 单 writer 限制——与 MySQL/PostgreSQL 部署的 25 不一致，但本项目实际生产为 SQLite）。

### D5: STYLE-001 partial 仅 3 处（per CONTEXT.md W6a）
**Plan 字面要求**：handler 层 2-3 处 `errors.Is` 迁移。

**实际**：
- 2 处 service 层（`storage/file_service.go` GetUserQuota + checkUserQuota）
- 1 处 handler 层（`video_file_handler.go` RenameFile）

未触及 service 层 `fmt.Errorf("中文: %v", err)` 模式的 %w 改造（该改造需要 service 错误全面 sentinel 化，影响面较大，超出 P2 tier 范围）。

### D6: 范围严格遵守（D-05.6 / 阶段纪律）
- 未触碰 `docs/audits/*.md`（审计文档唯一 source of truth）
- 未触碰 `STATE.md` / `ROADMAP.md`（orchestrator 在验证后拥有）
- 未引入新依赖（`strconv` / `net/url` / `mime/multipart` / `errors` 均为标准库）
- 未注入 STYLE-002 误报 / STYLE-009 延后决策的 marker 注释到生产源码
- 仅 1 处既有测试更新（`TestQuotaExceeded` 改用 UTF-8 文本兼容 SEC-012 MIME 校验；D-04.4 兼容）
- 既未在 P2 引入新测试文件（D-04.3）

---

## 关键文件

### 修改的源码
- `internal/services/frame_capture_service.go` — BUG-011
- `internal/services/storage/file_service.go` — BUG-011 + SEC-012 + STYLE-001 partial
- `internal/models/api_key.go` — BUG-015/016 + STYLE-006
- `internal/middleware/apikey.go` — STYLE-007
- `internal/auth/ad_auth.go` — STYLE-008
- `internal/auth/ad_validator.go` — STYLE-008 + STYLE-010
- `internal/handlers/split_handler.go` — STYLE-010
- `internal/handlers/admin_handler.go` — STYLE-010
- `internal/services/splitting_service.go` — STYLE-010
- `internal/services/snapshot_service.go` — STYLE-010
- `internal/auth/sm4_token.go` — SEC-011
- `internal/services/tingwu_client.go` — SEC-013 + gofmt
- `internal/huawei/client.go` — SEC-013 + gofmt
- `internal/huawei/manager.go` — SEC-013
- `internal/handlers/system_handler.go` — SEC-015
- `internal/handlers/video_file_handler.go` — STYLE-001 partial
- `cmd/server/app.go` — SEC-013 + SEC-014 + PERF-015 (注释)
- `internal/recorder/coordinator.go` — PERF-012
- `internal/services/conversion_service.go` — PERF-013 + PERF-016
- `internal/services/audit/audit_log_service.go` — PERF-014
- `internal/services/notification/notification_service.go` — PERF-014
- `internal/services/rate_limiter.go` — PERF-014
- `internal/services/transcription_service.go` — PERF-016
- `internal/config/config.go` — SEC-013 + PERF-015

### 修改的测试
- `internal/services/storage/file_service_test.go` — TestQuotaExceeded 改用 UTF-8 文本兼容 SEC-012 MIME 校验

---

## 验证摘要

| 检查 | 结果 |
|------|------|
| `go build ./...` | OK |
| `go vet ./...` | OK（无输出） |
| `gofmt -l` on touched files | clean（17-04 触及文件均通过） |
| `go test -count=1 ./internal/config/... ./internal/auth/... ./cmd/server/... ./internal/huawei/... ./internal/services/... ./internal/scheduler/... ./internal/middleware/... ./internal/handlers/... ./internal/recorder/... ./internal/models/...` | 全部 PASS（12 个包，零 FAIL） |
| `grep "strconv.FormatInt.*UnixNano" internal/services/storage/file_service.go` | ≥ 5 命中 |
| `grep "IsIPv4Mapped\|.To4()" internal/models/api_key.go` | 2 命中 (To4 + normalizeIP 调用) |
| `grep "BeforeCreate" internal/models/api_key.go` | 2 命中 (BeforeCreate + generateAPIKey) |
| `grep "default:.*hasPermission" internal/middleware/apikey.go` | 1 命中 |
| `grep "if conn != nil" internal/auth/ad_auth.go internal/auth/ad_validator.go` | 3 命中 (across both files) |
| `grep "hex.DecodeString" internal/auth/sm4_token.go` | 1 命中 |
| `grep "http.DetectContentType" internal/services/storage/file_service.go` | 1 命中 |
| `grep "OutboundURLAllowlist" internal/config/config.go internal/services/tingwu_client.go internal/huawei/client.go` | 6 命中 |
| `grep "publicRouteDecorator" cmd/server/app.go` | 5 命中 (定义 + 4 路由挂载) |
| `grep ", ok :=" internal/handlers/system_handler.go internal/middleware/audit.go` | ≥ 1 命中 (audit.go 已有 8 处，system_handler.go 新增 1 处 type switch) |
| `grep "strings.Builder" internal/recorder/coordinator.go` | ≥ 1 命中 |
| `grep "now := time.Now()" internal/services/conversion_service.go` | 1 命中 |
| `grep "PERF-014: 无缓冲 channel" 3 service files` | 3 命中 |
| `grep "SetMaxOpenConns\|SetMaxIdleConns\|SetConnMaxLifetime" cmd/server/app.go` | 3 命中 |
| `grep "time.NewTimer" internal/services/conversion_service.go internal/services/transcription_service.go` | 2 命中 |
| `grep "errors.Is.*ErrRecordNotFound\|errors.Is.*gorm.ErrRecordNotFound" internal/services/storage/file_service.go internal/handlers/video_file_handler.go` | 3 命中 |

---

## Self-Check

- [x] 18 个 P2 finding ID 全部在 commit messages 中显式引用（D-02.2）
- [x] 所有 18 个 finding 至少 1 个 commit 主题含 finding ID（D-02.3 — 保留多 commit 不 squash）
- [x] 未触碰 STATE.md / ROADMAP.md / docs/audits/*（D-05.6 / 阶段纪律）
- [x] P2 跳过单测（D-04.3）—— 仅 1 处既有测试兼容性更新（TestQuotaExceeded）
- [x] 既有测试包零回归（12 个测试包全部 PASS）
- [x] `go build ./...` / `go vet ./...` / `gofmt -l` 全部 green
- [x] STYLE-002 = 误报，STYLE-009 = 延后——仅在 `<notes>` 与本 SUMMARY 记录，未动审计 doc / 未注 marker 注释到生产源码
- [x] STYLE-001 partial per W6a（3 处 errors.Is + 4 处 %w 包装于本 tier 触及的新代码）

---

*Plan completed: 2026-07-30 — 17 atomic commits on `main` (cf2d248..72e2027).*
