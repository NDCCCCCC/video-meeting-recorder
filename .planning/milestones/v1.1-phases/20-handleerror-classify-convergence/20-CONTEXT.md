# Phase 20: 错误处理统一收敛 + sentinel 体系增强 - Context

**Gathered:** 2026-08-01
**Status:** Ready for planning

<domain>
## Phase Boundary

在 Phase 19（HandleError + 24 sentinels + ~356 散点收敛）的基础上深化错误处理体系，**仅聚焦三项**：handler ad-hoc classify 全量替换为 HandleError、zap logger 集成 errors.Is 链输出 `sentinel_type` 字段、基于 `internal/errors/errors.go` 自动生成 `docs/errors.md` 文档。**不**做 typed error kind 字段（Sentinel vs BusinessError vs ad-hoc 的运行时类型区分），该项 deferred 到下一阶段。

**属于本阶段:**

1. handler 层 ad-hoc classify 全量清理：9 个文件 27 处 `err.Error()`/string-match inline 分支 + 删除 `classifyAuthLoginError` 1 处 formal 函数。全部走 `if response.HandleError(c, err) { return }`
2. zap logger `sentinel_type` 字段接入：在 `pkg/response` 或新增 `pkg/logging` 提供 `SentinelField(err)` helper；handler/service 现存 `zap.Error(err)` 调用点零侵入升级
3. 自动生成 `docs/errors.md`：基于 `internal/errors/errors.go` const 集合和 `mapping.go` 的 MapToHTTPStatus 输出 sentinel 表格（name | kind | HTTP 状态 | call-site count）
4. 仅鉴收 cross-package local error var 重复（即查不主动迁）：survey 现存 service 文件是否仍声明 `var ErrXxx = errors.New(...)` 而未走 `internal/errors`

**不属于本阶段:**

- typed error kind 字段 / 三层 enum（Sentinel vs BusinessError vs ad-hoc）—— 工作量大 + 需 spike；deferred 到独立 phase
- Phase 17 deferred 全量收尾：STYLE-001 全库 `%w` 迁移、SEC-003b 华为密码 DB 加密、PERF-003 全库 403 处 ctx、STYLE-009 包名冗余
- Phase 19 deferred：`taskServiceAdapter` 与 VideoFileService 合并、HMAC jti DB 表（Redis vs DB 架构决策）、`internal/errors` 包 import 全量迁移
- 引入新依赖（`koanf`、`golangci-lint + errcheck/gosec`、`audit` 包迁移）—— 工具链改进独立 phase
- 前端相关修改 —— 仅后端 / 文档
- 修改 `docs/audits/*.md` —— 不可变 source of truth

</domain>

<decisions>
## Implementation Decisions

### D-01 Phase 范围聚焦
- **D-01.1:** 本阶段**仅落地 3 项**：(a) classify 全量清理，(b) zap logger errors.Is 集成，(c) docs/errors.md 自动生成。typed error kind 留 `<deferred>`
- **D-01.2:** 改动仅限 `internal/` + `pkg/` + `docs/errors.md`（新增文件），不动 `frontend/`、`.planning/`、`docs/audits/*.md`、worktrees
- **D-01.3:** 不主动迁 cross-package local error var，仅在 (a) 触及文件时顺带调研、产出 survey 报告（不入 final deliverable，但入 commit message body 列出文件:行号）

### D-02 classify 全量清理（27 处 + 1 处 formal 函数）
- **D-02.1:** 一次性全量扫荡 27 处 inline 分支 + 删除 `classifyAuthLoginError`（auth_handler.go:46）。按 Phase 17 D-01.4 "一次完成全量改" 原则；与 Phase 16 D-01.2 一致
- **D-02.2:** 替换原则：**复用现有 sentinel + 补漏**。如 `ErrUserNotFound`/`ErrADAccountNotFound`/`ErrPPTFileNotFound`/`ErrAPIKeyNotFound` 等 Phase 19 D5-D21 已建；不对应场景改用 `BusinessError`（typed，已支持按 Code 字段映射）
- **D-02.3:** 27 处对应的源 / 路径（plan-phase 时务必验证行号，最终以 runtime grep 计数为准）：
  - `internal/handlers/ppt_handler.go` — 27 处（其中 ppt_handler.go:916 `strings.Contains(errMsg, "frame bytes too large")` 为典型 case）
  - `internal/handlers/input_config_handler.go` — 7 处
  - `internal/handlers/file_handler.go` — 5 处
  - `internal/handlers/auth_handler.go` — 5 处（含 `classifyAuthLoginError` formal 函数）
  - `internal/handlers/video_file_handler.go` — 3 处
  - `internal/handlers/admin_handler.go` — 3 处
  - `internal/handlers/user_handler.go` — 2 处
  - `internal/handlers/transcription_handler.go` — 2 处
  - `internal/handlers/split_handler.go` — 2 处
  - `internal/handlers/role_handler.go` — 2 处
- **D-02.4:** Service 层错误返回约定：service 返回 `BusinessError`（typed，Code 字段）或 sentinel `%w` 包装的 wrapped error；handler 一律 `if response.HandleError(c, err) { return }`，**不**再允许内联 switch
- **D-02.5:** regression 测试：每个 handler path 加表驱动单测，验证传入原 string-match error 后 HTTP 状态码与之前一致。具体策略（Phase 17 D-04.1 纪律参考）：
  - 每个被触达的 handler 函数加 1 个表驱动 `_test.go`
  - 子测试覆盖至少 4 类错误：(i) sentinel 直接返回 (ii) sentinel wrap (iii) BusinessError (iv) 未知 error
  - 验证响应 `http.StatusXxx` + `CodeXxx` 双字段
- **D-02.6:** P0/P1 节奏：本阶段只 1 个 plan 层级（不是 4 个 wave）；按 handler 文件 1 个 atomic commit，9+1 = 10 atomic commits，docs/errors.md 与 D-03 一起一个 atomic commit

### D-03 zap logger errors.Is 集成
- **D-03.1:** 在 `pkg/response/response.go` 或新 `pkg/logging/` 提供 helper `SentinelField(err error) zap.Field`，返回 `zap.String("sentinel_type", "...")`。helper 内部调用 `internal/errors.IsKnownError` 第一个命中项
- **D-03.2:** 调用方配合：`h.logger.Error("xxx", zap.Error(err), response.SentinelField(err))` —— 一行调用即可，无需修改 zap core
- **D-03.3:** 匹配优先级 = `IsKnownError` 第一个命中（按 `mapping.go` IsKnownError 内部 slice 的顺序，常以"更具体"优先排列：例如 `ErrTaskNotFound` 在 `ErrNotFound` 之前）
- **D-03.4:** BusinessError 处理：`sentinel_type="BusinessError(code=NOT_FOUND)"` —— `kind + parenthesis` 形态；与普通 sentinel 字符串值区分。示例：`sentinel_type:"BusinessError(code=FOREIGN_KEY_CONSTRAINT)"`
- **D-03.5:** 未知 error（不命中任何 IsKnownError）→ `sentinel_type="ad-hoc"`（与 D-03.4 的 BusinessError 形态不同）；监控 dashboard 用此字符串识别"未收口"风险点
- **D-03.6:** 集成位置：handler / service 现存所有 `zap.Error(err)` 调用点改为 `zap.Error(err), response.SentinelField(err)`；不引入 wrapper logger，不侵入 zap core
- **D-03.7:** 不动 `internal/middleware/error_mapper.go`（Phase 19 已带开，本阶段不扩展）

### D-04 docs/errors.md 自动生成
- **D-04.1:** 触发方式：`//go:generate go run ./cmd/error-doc-gen` 注释加在 `internal/errors/errors.go` 顶部；CI / Makefile target 跑 `go generate ./internal/errors/...` 校验
- **D-04.2:** Generator 位置：新增 `cmd/error-doc-gen/main.go` 单一 binary；纯文本扫 `internal/errors/errors.go` 的 `var (...)` 块和 `mapping.go` 的 `MapToHTTPStatus` switch case —— 不依赖 AST/reflection
- **D-04.3:** 文档内容（每行一个 sentinel）：
  | Sentinel name | Kind | HTTP Status | Call-site count |
  |---|---|---|---|
  | `ErrNotFound` | Sentinel | 404 | N |
  | `BusinessError(CodeNotFound)` | BusinessError | 404 | N |
  
  Kind 列只取 `Sentinel / BusinessError` 两值（ad-hoc 不入此文档）；Call-site count = `grep -rE '\bErr\w+' internal/ --include='*.go' | wc -l` 单次生成结果（每次生成重算）
- **D-04.4:** 同步检查：Makefile `check-errors-doc` target 跑 `go generate` 后用 `git diff --quiet docs/errors.md` 检查；CI 命中非空 diff 即 fail。`docs/errors.md` 每次 const 集变动必须随 commit 更新
- **D-04.5:** docs/errors.md 末尾附加"未收口 ad-hoc 错误审计段"—— 从 `internal/handler/*.go` 动态 grep 当前 inline `err.Error()` 分支数（目标：0）。CI 渲染为统计段（如 `> 当前 27 处已收敛到 0`）

### D-05 测试纪律
- **D-05.1:** classify 替换（D-02.5）每个 handler file 至少 1 个 `_test.go` 表驱动测试
- **D-05.2:** zap logger helper（D-03）增加 `pkg/response` 或 `pkg/logging` 测试：(i) sentinel 命中返回 `sentinel_type=ErrXxx` (ii) BusinessError 返回 `sentinel_type=BusinessError(code=yyy)` (iii) 未知返回 `sentinel_type=ad-hoc` (iv) nil 返回 nil（safe default）
- **D-05.3:** docs/errors.md generator 增加：(i) 已知 sentinel 全表 (ii) BusinessError 全表 (iii) missing filepath → error (iv) diff vs 上次输出 = 一致
- **D-05.4:** 跨包回归：`go test -race ./...` 12 个 phase-17 触及包全绿
- **D-05.5:** 不删既有 handler 测试（如 `auth_handler_test.go`）；改动函数签名需同步更新

### D-06 文档与提交
- **D-06.1:** 提交策略（与 Phase 17 D-02 类似）：
  - `refactor(20-classify): 全量替换 27 处 handler ad-hoc classify 为 HandleError`（10 atomic commits，每文件 1 个）
  - `feat(20-logger): zap logger 接入 errors.Is 链输出 sentinel_type`（D-03）
  - `docs(20-errors): 自动生成 docs/errors.md + Makefile check target`（D-04）
- **D-06.2:** 每个 atomic commit 必须保证 `go build ./...` + 涉及包 `go test -race ./...` 通过
- **D-06.3:** docs/errors.md 与对应 constants 同步 commit；Makefile check target 集成到现有 CI

### Claude's Discretion
- D-02.4 中 service 返回 `BusinessError` vs `%w` wrap 的具体挑选粒度（基于是否需要 Code 字段标识）
- D-03.1 中 helper 是放 `pkg/response` 还是新 `pkg/logging/`（根据 call-site 跨包性决定）
- D-04.2 中 generator 对 const 注释（godoc 风格）是否提取到文档第二列
- D-04.3 中 Call-site count 列入方式（粗略 grep vs 精确 grep；前者更轻量且足够）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 19 baseline（lock-in）
- `.planning/STATE.md` §"Phase 19 Final Status" + "Phase 19 — ctx 全量级联 + SEC-004 replay 修复 + STYLE-001 error 迁移" — 11 commits + HandleError + mapping.go 架构
- `internal/errors/errors.go` — 30+ sentinels + `BusinessError` typed 类型（Phase 19 D5-D21 落地）
- `internal/errors/mapping.go` — `MapToHTTPStatus` (typed BusinessError first, sentinel via errors.Is) + `IsKnownError` (slice iteration first hit)
- `pkg/response/response.go:173` — `HandleError(c, err) bool` handler 统一入口
- `internal/middleware/error_mapper.go` — middleware 侧 mapping（Phase 19 已带开，本阶段不扩展）

### Phase 17 修复纪律（保持一致）
- `.planning/phases/17-56-p0-p1-p2/17-CONTEXT.md` D-01.4 "一次完成全量改"、D-04 测试纪律
- `docs/audits/2026-07-30-backend-code-review.md` — 审计 source of truth（不可变，**禁止**为 Phase 20 改动）

### 代码起点（本阶段需修改的文件）
- `internal/handlers/ppt_handler.go` — 27 处 string-match 分支（最密集）
- `internal/handlers/auth_handler.go:46` — `classifyAuthLoginError` formal 函数待删
- `internal/handlers/input_config_handler.go` / `file_handler.go` / `video_file_handler.go` / `admin_handler.go` / `user_handler.go` / `transcription_handler.go` / `split_handler.go` / `role_handler.go` — 9 个 handler 文件共 27 处
- `internal/handlers/auth_handler_test.go` — 既有测试需保持兼容
- `cmd/server/app.go` — logger 注入点（不动）

### 已有架构文档（约束）
- `.planning/PROJECT.md` §"Auth" — SM4-GCM Token 认证（HandleError 路径必走 middleware）
- `.planning/PROJECT.md` §"Context" — zap logger 全栈使用

### 工具链（不引入新依赖）
- Go 1.24 `//go:generate` 已支持
- zap 现有 zapcore.Field interface 已就位
- 不引入 cspell/markdownlint 等

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets

- **`pkg/response/response.go:173 HandleError`** — handler 统一入口，已写 `if response.HandleError(c, err) { return }`；handler 改造时一对一替换字符串匹配即可
- **`internal/errors/mapping.go:32 MapToHTTPStatus`** — typed BusinessError 优先 + sentinel errors.Is 链；新 sentinel 加进 `mapping.go` 即自动映射
- **`internal/errors/mapping.go:127 IsKnownError`** — 用 slice 顺序枚举，**第一个命中** 即返回 true（D-03.3 优先级规则）
- **`internal/errors/errors.go` `BusinessError{ Code, Message, Err }`** — typed 错误已支持按 Code 字段映射（D-02.4 / D-03.4）
- **`internal/errors/mapping.go:164 FromGORM`** — service 边界从 gorm 错误的统一转换
- **`internal/errors/mapping.go:180 NotFound(what, id)`** — Wrap 风格 NotFound 构造助手

### Established Patterns

- **Phase 19 D4**：service 边界 `errors.Is(err, gorm.ErrRecordNotFound)` 替代 `err ==` 已落地全库（ppt_file/notification/timestamp_mapper/video_file 等包）
- **Phase 19 D6**：ad_auth + local_auth Login 路径统一化通过 `classifyAuthLoginError` + mapping.go 共同作用（**删除该函数后映射由 mapping.go 接续**）
- **handler 风格**：handler 文件 `h.logger` zap logger，DI 注入；`c.JSON(http.StatusXxx, response.Response{...})` 统一响应
- **logging 风格**：`logger.Error("msg", zap.Error(err), zap.String("k", v))` 模式普遍

### Integration Points

- **`pkg/response/response.go`** 加 `SentinelField(err error) zap.Field` 公开 helper —— handler/service 直接 `response.SentinelField(err)` 调用
- **`cmd/error-doc-gen/main.go`** 新增 —— 单一 binary，纯文本扫 const 集
- **`docs/errors.md`** 新增 —— CI/Makefile check 验证同步
- **`internal/handlers/*.go`** —— 9 文件 27 处 inline 分支 + 1 处 formal 函数清理
- **`pkg/response/response.go:173 HandleError`** —— 不动内部，但 mapping 表驱动变更需同步测试

</code_context>

<specifics>
## Specific Ideas

### `classifyAuthLoginError` 替换示意

```go
// 删除: internal/handlers/auth_handler.go:46
// func classifyAuthLoginError(err error) (code int, httpStatus int) { ... }

// Login handler 内调用点 (line 91) 替换:
code, _ := classifyAuthLoginError(err)
//  → // (无需调用 classify；mapping.go 自动接续)
// if response.HandleError(c, err) { return }
// mapping 已实现 typed/sentinel/dup→409/disabled→403/...
```

### ppt_handler.go:916 替换示意

```go
// 替换前
errMsg := err.Error()
case strings.Contains(errMsg, "frame bytes too large"):
    c.JSON(http.StatusBadRequest, response.Response{Code: response.CodeInvalidRequest, Message: "frame bytes too large"})

// 替换后
// service 层: return fmt.Errorf("frame bytes too large: %w", internalerrors.ErrInvalidInput)
// handler 层: 
if response.HandleError(c, err) { return }
// mapping.go 命中 ErrInvalidInput → 400/1001/"请求参数无效"
```

### D-03 `SentinelField` 示意

```go
// 新增: pkg/response/response.go 或 pkg/logging/sentinel.go

// SentinelField 返回 zap.Field "sentinel_type"：
//   - sentinel 命中 → zap.String("sentinel_type", "ErrXxx")
//   - BusinessError → zap.String("sentinel_type", "BusinessError(code=XXX)")
//   - 未知 err      → zap.String("sentinel_type", "ad-hoc")
//   - err==nil     → zap.Skip() (跳过字段)
func SentinelField(err error) zap.Field {
    if err == nil { return zap.Skip() }
    var be *internalerrors.BusinessError
    if errors.As(err, &be) {
        return zap.String("sentinel_type", fmt.Sprintf("BusinessError(code=%s)", be.Code))
    }
    if name, ok := firstMatchingSentinel(err); ok {
        return zap.String("sentinel_type", name)
    }
    return zap.String("sentinel_type", "ad-hoc")
}

func firstMatchingSentinel(err error) (string, bool) {
    for _, sentinel := range []error{ /* mirror of internal/errors IsKnownError slice */ } {
        if errors.Is(err, sentinel) {
            return sentinelName(sentinel), true
        }
    }
    return "", false
}
```

调用点：
```go
h.logger.Error("Failed to save auth config", zap.Error(err), response.SentinelField(err))
```

### D-04 generator 示意

```go
// cmd/error-doc-gen/main.go
//go:build ignore
package main

// 1. 正则匹配 errors.go var(...) 块中 `ErrXxx = errors.New("...")`
// 2. 正则匹配 mapping.go switch case 收集 HTTP status
// 3. 写入 docs/errors.md：
//    | Sentinel | Kind | HTTP Status | Call-site count |
//    |----------|------|-------------|-----------------|
//    | ErrNotFound | Sentinel | 404 | 12 |
//    | BusinessError(CodeNotFound) | BusinessError | 404 | 3 |
// 4. 末尾附加 "ad-hoc 审计段"：
//    > 当前 `err.Error()`/string-match inline 分支: N 处 (target: 0)
```

### D-02.5 表驱动单测示意

```go
// internal/handlers/auth_handler_test.go
func TestLogin_HandleError_ClassifyDrop(t *testing.T) {
    cases := []struct{
        name string
        err error
        wantStatus int
        wantCode int
    }{
        {"ErrADAccountNotFound", internalerrors.ErrADAccountNotFound, 404, 1004},
        {"ErrUserDisabled_wrapped", fmt.Errorf("user %s disabled: %w", "alice", internalerrors.ErrUserDisabled), 403, 1003},
        {"ErrADConfigError_unwrapped", internalerrors.ErrADConfigError, 503, 1005},
        {"ErrInvalidInput_BusinessError", internalerrors.NewBusinessError(internalerrors.CodeInvalidInput, "bad", nil), 400, 1001},
        {"Ad-hoc_unknown", errors.New("random"), 500, 1005},
    }
    for _, tt := range cases {
        t.Run(tt.name, func(t *testing.T) {
            // gin.TestMode + 构造 c → HandleError → 验证 c.Writer.Status() + JSON Code
        })
    }
}
```

</specifics>

<deferred>
## Deferred Ideas

### Phase 20 自身延后
- **typed error kind 字段**（Sentinel vs BusinessError vs ad-hoc 三层 enum 或 marker interface）—— 用户在 D-01 明确排除；本阶段只输出 `sentinel_type` 字符串形态。运行时类型区分留给下一 phase；如需 spike 验证 marker interface vs reflect 性能

### Phase 17 deferred 仍未触
- STYLE-001 全库 `%w` 包装迁移（168 处 `errors.New` + 474 处 `fmt.Errorf`）
- SEC-003b 华为密码 DB 加密存储（`models.InputConfig.Password`）
- PERF-003 全库 403 处 GORM `WithContext(ctx)` 透传
- STYLE-009 包名冗余清理（133 处 `Get*` rename）
- 引入 `koanf` 替代 viper
- audit 包从 `internal/services/audit` 迁移到 `internal/audit`
- 测试覆盖稀疏问题（44 测试 vs 153 源）
- 引入 `golangci-lint` + `errcheck`/`gosec` 规则

### Phase 19 deferred 仍未触
- `taskServiceAdapter` 与 `VideoFileService` 合并（含 Phase 18 SM4-GCM 解密逻辑）
- HMAC jti 服务端 `used_jtis` 表（Redis vs GORM 架构决策）
- `internal/errors` 包被 0 service 文件 import 的全量迁移（部分 service 仍用 `fmt.Errorf` + `errors.Is(gorm.ErrRecordNotFound)`）

### Phase 20 D-01.3 调研后产出（仅 commit message body）
- cross-package local error var survey 报告：grep 各 service 文件是否仍声明 `var ErrXxx = errors.New(...)` 而非走 `internal/errors`；本阶段**不**主动迁，调研结果落 commit message

</deferred>

---

*Phase: 20-handleerror-classify-convergence*
*Context gathered: 2026-08-01*
