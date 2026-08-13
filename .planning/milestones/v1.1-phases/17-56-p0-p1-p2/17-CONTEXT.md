# Phase 17: 后端代码审查 56 个发现修复 - P0/P1/P2 全量 - Context

**Gathered:** 2026-07-30
**Status:** Ready for planning

<domain>
## Phase Boundary

修复 `docs/audits/2026-07-30-backend-code-review.md` 中列出的 56 个 Go 后端代码审查发现（13 HIGH + 18 MEDIUM + 25 LOW），覆盖 bug、安全、性能、Go 风格四类。仅修改 `internal/` 与 `cmd/server/app.go`，不动前端、文档、迁移 SQL 文件。

**属于本阶段:**

- 13 个 HIGH 级发现（SEC-001/002/003/004 + BUG-001/002 + PERF-001/002/003/004/005）
- 18 个 MEDIUM 级发现（BUG-003/004/005/006 + SEC-005/006/007/008/009/010 + PERF-006/007/008/009/010/011 + STYLE-003/004/005）
- 25 个 LOW 级发现（BUG-011/015/016 + SEC-011/012/013/014/015 + PERF-012/013/014/015/016 + STYLE-001/002/006/007/008/009/010）
- P0 修复配套的启动校验、配置文档同步更新、`DEPLOYMENT.md` / `BUILD.md` / `SECURITY.md` / `.env.example` 中相关环境变量名变更说明

**不属于本阶段:**

- 前端代码审查（`frontend/`，由前端审查负责）
- 数据库迁移 SQL 文件（除 `migrations/*.go` 内的 Go 代码）
- 新增功能（仅修复，不增功能）
- `bin/`、`data/`、`certs/`、`.claude/`、`.planning/`、worktrees
- 文档类项目（README 重写等）
- **审计文档** (`docs/audits/*.md`) — 不可变，作为唯一 source of truth，不允许修改或追加注释
- 引入新依赖（除非修复必须且 spike 已验证）— 例：MD5→SHA-256 不引入 `crypto/sha256`（已标准库）

</domain>

<decisions>
## Implementation Decisions

### D-01 范围与策略
- **D-01.1:** 范围=**一次性完成全量 56 个**（与 Phase 16 D-01.2 "一次完成全量改"一致）
- **D-01.2:** 节奏=按优先级 **P0 → P1 → P2 顺序**推进，P0 必须先合并通过验证再做 P1
- **D-01.3:** 改动仅限 `internal/` + `cmd/server/app.go`，不动 `frontend/`、`.planning/`、worktrees
- **D-01.4:** **P0 11 项代码发现全部需配单测**（11 处必做；STYLE-001/002 虽为 HIGH 项目级但归入 P2 处理或误报标记）；**P1 至少每个修复都加单测**（18 处）；**P2 跳过单测**（25 处，仅做代码修改）
- **D-01.5:** 不重新执行代码审查本身（审计文档已是最终产物）；只按其清单逐项修复
- **D-01.6:** 修复顺序参考审计文档 6.2 节"优先级修复清单"，不另设顺序

### D-02 提交/合并策略
- **D-02.1:** **4 个 mega commit** 按 P0/P1a/P1b/P2 分组（P1 因规模拆为 a/b 两个 commit）：
  - `fix(17-p0): P0 HIGH 发现修复（11 项）`
  - `refactor(17-p1a): P1 MEDIUM 发现修复 - 错误处理 + 安全加固（12 项）`
  - `refactor(17-p1b): P1 MEDIUM 发现修复 - 性能 + 接口归位（7 项）`
  - `chore(17-p2): P2 LOW 发现清理（20 项 + STYLE-002 误报 + STYLE-009 延后）`
- **D-02.2:** 每个 mega commit 的 commit message body 列出**所有引用的 finding ID**（如 `SEC-001`, `BUG-001`, `PERF-003`）
- **D-02.3:** 每个 mega commit 内部按 finding ID 顺序提交多个普通 commit，最后 squash 成 mega commit（或保留多个 commit 由 planner 决定）
- **D-02.4:** 每个 mega commit 必须保证 `go build ./...` 通过 + 该 tier 涉及的相关测试包通过
- **D-02.5:** 不再为每个 finding 单独开分支或 PR；4 个 mega commit 直接推 `main`

### D-03 破坏性变更处理（SEC-001/003/004 等）
- **D-03.1:** **内置启动校验**：SEC-001 中 `Environment=="production" && Secret==""` 时 `logger.Fatal`；非 production 仅打印严重警告
- **D-03.2:** **同步更新部署文档**：`DEPLOYMENT.md`、`BUILD.md`、`SECURITY.md`、`.env.example` 中所有相关环境变量名（`RECORD_*` 前缀修正、SM4 secret 长度要求等）
- **D-03.3:** **HMAC 编码变更保持向后兼容**（SEC-004）：新代码签发用 `base64.RawURLEncoding`，但 `Verify()` 接受新旧两种编码，重启后旧 token 仍可验证
- **D-03.4:** **删除硬编码 fallback secret**（SEC-001/2）：不再保留 `change-me-in-production` 默认值；启动时检查 secret 长度 ≥ 32 字节
- **D-03.5:** **TLS 最低版本强制 1.2**（SEC-003a：仅 TLS 三项）：移除 `MinTLSVersion: 0x0301` 硬编码，改为从配置读取 `cfg.MinTLSVersion` 默认 `tls.VersionTLS12`
- **D-03.6:** **CORS 收紧**（SEC-007）：从配置 `cfg.CORS.AllowedOrigins`（数组）读取允许的 origin 列表；默认拒绝所有跨域
- **D-03.7:** 不保留 deprecated 兼容开关（与审计建议一致，避免技术债）
- **D-03.8:** **PERF-005 保持 HTTP 200 契约**：sync-IO handler 改用 **bounded concurrency + streaming**（信号量限流 + 分块处理），**不**改响应码或 JSON 形状；不为 PERF-005 引入 async/202 + 任务队列基础设施

### D-04 测试纪律
- **D-04.1:** P0 修复（11 项代码发现）**必须**配单元测试，测试用例覆盖正常路径 + 至少 1 个回归路径
- **D-04.2:** P1 修复（18 项）**至少配一个测试**，可用表驱动测试合并多个 finding 的边界用例
- **D-04.3:** P2 修复（25 项）**跳过单测**，仅 `go vet` + `gofmt -l` 通过即可
- **D-04.4:** 既有测试包不删（如 `handlers` 测试包），修复若改动既有函数签名需同步更新测试
- **D-04.5:** 新增测试文件命名遵循 `internal/<package>/<file>_test.go` 既有约定
- **D-04.6:** 测试中涉及 DB 的用 SQLite in-memory 或 `sqlmock`，不引入新外部依赖

### D-05 文档同步更新（D-03 的实施细节）
- **D-05.1:** `DEPLOYMENT.md`：列出新环境变量名（`SM4_SECRET` → 仍用此名但需明确 viper prefix 修正）、最小长度（≥ 32 字符）、启动校验行为
- **D-05.2:** `BUILD.md`：移除文档中错误的 `SetEnvPrefix("RECORD")` 示例（应改为显式 BindEnv 或空 prefix）
- **D-05.3:** `SECURITY.md`：补充 SECRET 校验章节、HLS Token jti 防重放说明、TLS 最低版本说明
- **D-05.4:** `.env.example`：所有 secret 占位改为 `<必须显式设置，最小 32 字符>` 警告文案，移除可用的默认值示例
- **D-05.5:** 文档更新随 D-02 的 P0 mega commit 一起提交（文档与代码同步上线）
- **D-05.6:** **仅授权**修改 `DEPLOYMENT.md` / `BUILD.md` / `SECURITY.md` / `.env.example` 四个文档；**禁止**修改 `docs/audits/*.md`、禁止新增 `internal/**/README.md`、禁止向生产源码注入 STYLE-002/SYTLE-009 等延后决策的注释

### Claude's Discretion
- BUG-006（time.Sleep → time.NewTimer + select）的具体重构写法
- PERF-007/008/009（sync.Pool、正则包级、类型化 struct）的局部实现选择
- STYLE-009（包名冗余 133 处 Get*）是否真的在本次一并清理（用户没说必须，仅 LOW） — 默认跳过以减少 PR 噪音
- STYLE-010（godoc 缺失 8 处）的注释措辞（中文/英文）
- 中间件类型断言 `, ok` 守卫的具体错误返回值（`uint(0)` vs `0, false`）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 审计报告（修复清单的唯一来源）
- `docs/audits/2026-07-30-backend-code-review.md` — 647 行审计报告，列出全部 56 个 finding，每项含文件:行号、触发场景、修复建议。**这是 planning 的唯一 source of truth，不允许重新解读 finding 含义，也不允许修改/追加内容**
- `docs/audits/2026-07-30-backend-code-review.md` §1.3 — Top 5 最严重问题（爆炸半径排序）
- `docs/audits/2026-07-30-backend-code-review.md` §6.2 — 优先级修复清单（P0/P1/P2）

### 已有架构文档（用于理解约束）
- `cmd/server/app.go` — 服务装配入口，SEC-002 修复点（line 557-598）
- `internal/auth/service.go` — AuthService，SEC-002 注入点
- `internal/config/config.go` — viper 配置加载，SEC-001 修复点
- `internal/errors/errors.go` — 错误类型定义，STYLE-001 统一目标

### 已有相关代码（修复的起点）
- `internal/services/audit/audit_log_service.go` — audit service 实现
- `internal/services/audit/sanitizer.go` — sanitizer（commit 7169029 已通过 DI 注入）
- `internal/middleware/audit.go` — audit 中间件
- `internal/middleware/auth.go` — auth 中间件（STYLE-004/005 修复点）

### 部署文档（需同步更新）
- `DEPLOYMENT.md` — 部署指南
- `BUILD.md` — 构建说明（含 viper prefix 示例）
- `SECURITY.md` — 安全说明
- `.env.example` — 环境变量示例

### 前置阶段决策（用于保持一致）
- `.planning/phases/12-windows-ad/12-CONTEXT.md` D-04/D-15–D-17/D-31–D-32 — 与 SEC-001/002 修复方向一致
- `.planning/phases/16-visual-reshape/16-CONTEXT.md` D-01.2 — "一次完成全量改"决策
- `.claude/skills/spike-findings-record-v2/SKILL.md` — TLS 1.2+ 强制、AD 密码经 SM4 解密（与 SEC-003 一致）

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **`apikeyService.SetAuditService(auditService)` 模式**（`cmd/server/app.go:598`）— 已正确装配，SEC-002 修复应复用此模式（一行调用插入 557 与 582 之间）
- **`internal/errors` 包**（`internal/errors/errors.go`）— 已定义 `ErrNotFound/ErrInvalidInput/ErrUnauthorized/ErrForbidden`，STYLE-001 统一错误处理的目标
- **Sanitizer DI 模式**（`middleware.NewAuditMiddleware(auditService, logger, sanitizer)`）— nil 容忍（commit 7169029 验证），SEC-009 修复时可参考其防御式设计
- **审计中间件已有 `, ok` 类型断言**（`audit.go:85-94`）— STYLE-005 中间件类型断言应与其对齐
- **既有 `recovery()` 兜底**（`cmd/server/app.go:519` `gin.Recovery()`）— BUG-002 fire-and-forget recover 的目标格式可参考

### Established Patterns

- **Go 错误处理**：项目已有 168 处 `errors.New("中文")` + 474 处 `fmt.Errorf`（仅 59% 用 `%w`），STYLE-001 修复**仅在本次新修改/接触的代码处用 `%w` 包装**，不做全库扫荡（全库迁移列入 `<deferred>`）
- **GORM 调用**：项目 GORM 调用均无 `WithContext(ctx)`（0/403），PERF-003 修复涉及 403 处，工作量大；建议**只在新增/修改的 GORM 调用上加 WithContext**（避免 PR 噪音），不在本次做全库扫荡式改造（除非 planner 评估后可接受）
- **环境变量命名**：项目用 viper `SetEnvPrefix("RECORD")` 但文档写 `SM4_SECRET`（无前缀），SEC-001 修复需明确文档/代码统一
- **审计日志格式**：sanitizer 已 DI 注入，新日志字段遵循既有 schema，不引入新结构

### Integration Points

- **`cmd/server/app.go`**：装配序列中插入 `authService.SetAuditService(auditService)`（SEC-002）
- **`internal/config/config.go`**：`SetEnvPrefix` / `setDefaults` / `Load()` 三处修改（SEC-001）
- **`internal/auth/hlstoken/hls_token.go`**：`NewHLSToken` 启动校验 + base64 编码（SEC-004）
- **`internal/middleware/auth.go`**：所有 GORM 调用加 `WithContext`（PERF-003，**仅限本次修改的函数**）
- **`internal/services/video_recording_task_service.go`**：BUG-001 RetryTask 修复 + PERF-001 N+1 修复 + WithContext 改造
- **`internal/huawei/manager.go` + `client.go`**：SEC-003a TLS 加固 + ctx 透传
- **`internal/migrations/013_add_ad_fields.go`**：SEC-005 SQL 拼接改 GORM Migrator

</code_context>

<specifics>
## Specific Ideas

### SEC-002 修复（一行）
```go
// cmd/server/app.go:557 之后插入
authService.SetAuditService(auditService)
```
参考 `apikeyService.SetAuditService(auditService)` 已有的模式（line 598）。

### BUG-001 修复（RetryTask）
```go
// 替换 internal/services/video_recording_task_service.go:786-789
duration := task.EndTime.Sub(task.StartTime)
task.StartTime = newTriggerTime.Add(time.Duration(task.PreJoinMinutes) * time.Minute)
task.EndTime = task.StartTime.Add(duration)
```

### SEC-001 启动校验伪代码
```go
// internal/config/config.go Load() 末尾
if a.env == "production" && len(a.config.Auth.SM4Secret) < 32 {
    logger.Fatal("SM4_SECRET 必须显式设置且 ≥ 32 字符（生产环境）")
}
if len(a.config.HLS.HLSTokenSecret) < 32 {
    logger.Fatal("HLS_TOKEN_SECRET 必须显式设置且 ≥ 32 字符")
}
```

### SEC-004 HMAC 编码（向后兼容）
```go
// 签发用新编码
mac := hmac.New(sha256.New, secret)
mac.Write(payload)
sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
// Verify 时两种编码都接受
sigRaw, err1 := base64.RawURLEncoding.DecodeString(provided)
if err1 != nil {
    sigRaw, err2 = base64.StdEncoding.DecodeString(provided)
}
```

### PERF-005 修复（保持 HTTP 200，D-03.8）
```go
// bounded concurrency: handler 用 semaphore 限制并发重操作，
// 同时保持原有 200 + JSON 响应形状不变（不改 202、不引任务队列）
sem := make(chan struct{}, cfg.Admin.MigrationConcurrency) // default 4
var wg sync.WaitGroup
for _, item := range batch {
    wg.Add(1)
    sem <- struct{}{}
    go func(it Item) {
        defer wg.Done()
        defer func() { <-sem }()
        processChunk(ctx, it) // 已有处理函数
    }(item)
}
wg.Wait()
// 返回原有 200 响应
```

### P0 单测最低标准
- BUG-001：表驱动测试覆盖"原时长 > 新 StartTime 偏移"和"原时长 < 偏移"两个边界
- SEC-001：测试覆盖 `production + secret=""` → Fatal；`dev + secret=""` → Warn；`production + secret>=32` → OK
- SEC-002：测试覆盖 `SetAuditService(nil)` 不 panic；`auditLogger != nil` 时调用被路由
- SEC-003a：测试覆盖 `MinTLSVersion == 0x0302`；`InsecureSkipVerify == false` 默认
- SEC-004：测试覆盖 `len(secret) < 32` → Fatal；jti 重放 → 拒绝

</specifics>

<deferred>
## Deferred Ideas

- **STYLE-001 全库错误包装迁移**（168 处 `errors.New` + 474 处 `fmt.Errorf` + 5 处 `err.Error()=="中文"` + 13 处 `err == gorm.ErrRecordNotFound`）— 影响面过大，与审计核心修复无关；本次**仅在新修改/接触的代码处**用 `%w` 包装 + 在 handler 层迁移 2-3 处 `errors.Is(err, internalerrors.ErrNotFound)`，全库扫荡留作独立 phase
- **SEC-003b 华为密码 DB 加密存储**（`models.InputConfig.Password` 明文 → SM4-ECB 加密）— 需独立迁移 + 前端/配置联动（解密路径 + 旧明文数据兼容），留作独立 phase；本 phase 仅完成 SEC-003a（TLS 三项 + ctx 透传）
- STYLE-009 包名冗余清理（133 处 Get*）— 影响面过大且与审计核心修复无关，列入下个 LOW 清理 phase
- 引入 `koanf` 替代 viper（审计 6.3 节建议）— 范围外的依赖迁移，留作独立 phase
- audit 包从 `internal/services/audit` 挪到 `internal/audit`（审计 6.3 节建议）— 影响 import 链，留作独立 phase
- 403 处 GORM 全库加 `WithContext`（PERF-003 全集）— 本次仅修改/新增处加；全库扫荡列入独立 phase
- 引入 `golangci-lint` + `errcheck`/`gosec` 规则（审计 6.3 节建议）— 工具链改造，独立 phase
- 测试覆盖稀疏问题（44 个测试文件 vs 153 个源文件）— 本次 per-fix 增量补；全面覆盖率提升独立 phase
- HMAC jti 服务端 `used_jtis` 表（Redis 或 DB）— 需要架构决策（Redis 引入/还是 DB 表），独立 phase

---

*Phase: 17-56-p0-p1-p2*
*Context gathered: 2026-07-30*
*Revised: 2026-07-30 (per checker feedback — D-01.4/D-02.1/D-03.8/D-05.6 clarified; STYLE-001 full migration + SEC-003b added to `<deferred>`)*
</content>
</invoke>
