# Phase 20: 错误处理统一收敛 + sentinel 体系增强 - Research

**Researched:** 2026-08-01
**Domain:** Go error handling (sentinel + typed BusinessError) + zap structured logging + go:generate docs
**Confidence:** HIGH (lightweight verification mode — every claim below verified by runtime grep/Read against `main` HEAD `570a2bc`)
**Mode:** 轻量验证 (CONTEXT.md 已详尽，本文件验证关键声明 + 修正差异)

## Summary

CONTEXT.md 的核心架构声明 **全部 confirmed**（行号精确匹配：HandleError@173, MapToHTTPStatus@32, IsKnownError@127, FromGORM@164, NotFound@180；sentinel 总数 41 > "30+"；BusinessError typed 类型完整）。Phase 19 baseline 健康，`go build ./...` 0 错误，`go test ./internal/handlers/...` 全绿。

但发现 **5 项必须在 plan 阶段显式处理的差异**（不阻断，但需 planner 调整范围与策略）：

1. **⚠ 总散点计数 27 是错的——实际是 ~57**。CONTEXT 顶部说"9 文件共 27 处"，但 D-02.3 自己列的 10 个文件 (ppt=27, input_config=7, file=5, auth=5, video_file=3, admin=3, user=2, transcription=2, split=2, role=2) 求和 = **58**。"27" 实际只是 `ppt_handler.go` 单文件的计数。CONTEXT 自身内部不一致（"9 文件" vs D-02.3 列了 10 个文件）。
2. **⚠ CONTEXT D-02.3 漏列 2 个 handler 文件**：`video_recording_task_handler.go`（13 处 err.Error() GinError）与 `apikey_handler.go`（2 处 parse-error GinError；其余已走 HandleError）。实际散点文件数 = 12，不是 9 也不是 10。
3. **⚠ Makefile 不存在**——D-04.4 声明"已有 Makefile pattern"错误。`Makefile` / `makefile` 都不存在；`cmd/` 目录只有 `server/` 子目录。Planner 必须**新建 Makefile**（或改用 `tools/` shell 脚本 + `.github/workflows/` 直接调用）。
4. **⚠ AD config error 状态码 WILL 变化 (500 → 503)**——`classifyAuthLoginError` 把 `ErrADConfigError`/`ErrADUnreachable` 映射到 500；mapping.go 映射到 **503** (StatusServiceUnavailable)。auth_handler.go:38-39 注释已承认此差异，但 CONTEXT.md D-02.5 测试策略"HTTP 状态码与之前一致"在此 case **会失败**——必须显式接受 503，并更新 `TestClassifyAuthLoginError` 表项。
5. **⚠ `auth.ErrADUserNotRegistered` 未在 mapping.go IsKnownError slice 内**——目前走 `classifyAuthLoginError` 显式判 `auth.IsADUserNotRegistered` 返回 403；切到 HandleError 后将落入 default → **500 ad-hoc**。要么补 mapping.go case，要么补 internal/errors 新 sentinel（违反 D-02.2 "不主动加新 sentinel"——需 planner 决断）。

其余声明（IsKnownError slice 顺序、BusinessError 字段、Code 常量集、error_mapper.go 职责、Phase 19 已收敛 24 散点 + HandleError + mapping.go 三组件）全部 ✅ 验证通过。

**Primary recommendation:** Planner 接受 D-01..D-06 全部 locked decisions，但需在 PLAN 中：(a) 修正散点总数为 ~57 并扩到 12 个文件，(b) 新增 Wave 0 任务创建 Makefile，(c) 在 classify 替换表驱动测试中显式声明 AD config error 503 + ADUserNotRegistered 兜底策略，(d) 在 commit message body 中列出 cross-package 名称冲突 (`hlstoken.ErrTokenReplayed` vs `internal/errors.ErrTokenReplayed`)。

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions (D-01 .. D-06, 逐字摘自 CONTEXT.md `<decisions>`)

**D-01 Phase 范围聚焦**
- D-01.1: 本阶段**仅落地 3 项**——(a) classify 全量清理, (b) zap logger errors.Is 集成, (c) docs/errors.md 自动生成。typed error kind 留 `<deferred>`
- D-01.2: 改动仅限 `internal/` + `pkg/` + `docs/errors.md`（新增文件），不动 `frontend/`、`.planning/`、`docs/audits/*.md`、worktrees
- D-01.3: 不主动迁 cross-package local error var，仅在 (a) 触及文件时顺带调研、产出 survey 报告（入 commit message body）

**D-02 classify 全量清理** — D-02.1 一次全量扫荡 + 删 classifyAuthLoginError；D-02.2 复用现有 sentinel + 补漏不主动加新；D-02.3 文件清单（**计数有误，见 §1**）；D-02.4 service 返回 BusinessError 或 sentinel `%w`，handler 一律 HandleError；D-02.5 表驱动单测 4 类错误全覆盖；D-02.6 单层 plan，每文件 1 atomic commit

**D-03 zap logger errors.Is 集成** — D-03.1 `SentinelField(err) zap.Field` helper（放 pkg/response 或 pkg/logging 由 Claude 决定）；D-03.2 调用方 `zap.Error(err), response.SentinelField(err)` 一行升级；D-03.3 优先级 = IsKnownError 第一个命中；D-03.4 BusinessError → `sentinel_type="BusinessError(code=XXX)"`；D-03.5 未知 → `sentinel_type="ad-hoc"`；D-03.6 现存 `zap.Error(err)` 调用点零侵入升级；D-03.7 不动 error_mapper.go

**D-04 docs/errors.md 自动生成** — D-04.1 `//go:generate` 注释；D-04.2 `cmd/error-doc-gen/main.go` 单 binary 纯文本扫；D-04.3 文档表格 name | kind | HTTP | call-site count；D-04.4 Makefile `check-errors-doc` target + `git diff --quiet`；D-04.5 末尾 ad-hoc 审计段

**D-05 测试纪律** — D-05.1 每文件至少 1 表驱动 _test.go；D-05.2 SentinelField 4 类输入断言；D-05.3 generator 4 项验证；D-05.4 跨包回归 12 个 phase-17 包 `go test -race` 全绿；D-05.5 不删既有 handler 测试

**D-06 文档与提交** — D-06.1 3 类 atomic commit (refactor classify / feat logger / docs errors)；D-06.2 每 commit `go build ./...` + 涉及包 `go test -race ./...` 通过；D-06.3 docs/errors.md 与 constants 同步 commit

### Claude's Discretion (逐字摘自 CONTEXT.md `<decisions>`)

- D-02.4 service 返回 `BusinessError` vs `%w` wrap 的具体挑选粒度（基于是否需要 Code 字段标识）
- D-03.1 helper 放 `pkg/response` vs 新 `pkg/logging/`（根据 call-site 跨包性决定）
- D-04.2 generator 对 const 注释（godoc 风格）是否提取到文档第二列
- D-04.3 Call-site count 粗略 grep vs 精确 grep（前者更轻量且足够）

### Deferred Ideas (OUT OF SCOPE)

- **typed error kind 字段**（Sentinel vs BusinessError vs ad-hoc 三层 enum 或 marker interface）—— 用户 D-01 明确排除；本阶段仅输出 `sentinel_type` 字符串形态
- **Phase 17 deferred 仍未触**：STYLE-001 全库 %w 迁移 / SEC-003b 华为密码 DB / PERF-003 全库 ctx / STYLE-009 Get* rename / koanf / audit 包迁移 / golangci-lint+errcheck/gosec
- **Phase 19 deferred 仍未触**：taskServiceAdapter merge / HMAC jti DB 表 / internal/errors 全量 service import
- **cross-package local error var 主动迁移**——仅 survey 入 commit body
</user_constraints>

<phase_requirements>
## Phase Requirements (D-22 候选清单前 3 项；第 4 项 deferred)

| ID | Description | Research Support |
|----|-------------|------------------|
| REQ-20a | handler ad-hoc classify 全量清理（移除 inline string-match + 删除 classifyAuthLoginError formal 函数，全部走 HandleError） | §1 全量计数表（实际 12 文件 ~57 处，非 CONTEXT 声明的 9 文件 27 处）；§2 IsKnownError slice 顺序；§3 HandleError 签名；§9 AD config 503 + ADUserNotRegistered 兜底风险 |
| REQ-20b | zap logger 集成 errors.Is 链输出 sentinel_type 字段 | §2 BusinessError typed + IsKnownError slice（SentinelField 需 mirror 此 slice）；§4 zap.Error call-site 全表（~208 处零侵入升级 scope） |
| REQ-20c | 自动生成 docs/errors.md | §2 sentinel 全表 + Code 常量集（generator 输入源）；§5 cmd/Makefile 现状（**无 Makefile——需新建**） |
| REQ-20d (DEFERRED) | typed error kind 字段区分 Sentinel/BusinessError/ad-hoc | §9 风险与边界守护——本阶段不实现，但 SentinelField 字符串形态需为下阶段 typed kind 留扩展点 |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

`D:\CODE\ClaudeCode\record_V2\CLAUDE.md` 不含 error-handling 相关 hard rules，仅声明：项目结构（internal/auth, internal/models, internal/services, frontend）+ Auto-loaded Skill `spike-findings-record-v2`（Windows AD 实施蓝图，未含 error 约束）。全局 `~/.claude/CLAUDE.md` 是另一项目（智能记忆系统 hooks）与本 phase 无关。无 Phase 20 特定 forbidden patterns / 必选工具 / 测试规则——以 CONTEXT.md D-01..D-06 为唯一权威。

---

## §1 classify 散点全量计数表（D-02.3 runtime 验证）

**Grep 模式：** `err\.Error\(\)|strings\.Contains\(err|errors\.Is\(err|switch.*err|classify` 在 `internal/handlers/` 范围，配合 Read 逐文件审查。

| 文件 | CONTEXT 声明 | 实际 grep 计数 | 差异 | 备注 |
|------|-------------|---------------|------|------|
| `ppt_handler.go` | 27 | **26** | ⚠ -1 | err.Error() GinError 调用 24 处 + 2 处 `switch errMsg` 块 (line 670-678, 912-920)；典型 case line 916 ✅ 仍存在 |
| `input_config_handler.go` | 7 | **7** | ✅ | 全为 err.Error() GinError (parse + service) |
| `file_handler.go` | 5 | **5** | ✅ | 全为 err.Error() GinError |
| `auth_handler.go` | 5 (含 formal fn) | **5 + 1 formal** | ✅ 计数同 | formal fn @ line 46 + 4 inline err.Error() GinError + 1 调用点 (line 91) |
| `video_file_handler.go` | 3 | **3** | ✅ | |
| `admin_handler.go` | 3 | **3** | ✅ | |
| `user_handler.go` | 2 | **2** | ✅ | |
| `transcription_handler.go` | 2 | **2** | ✅ | |
| `split_handler.go` | 2 | **2** | ✅ | |
| `role_handler.go` | 2 | **2** | ✅ | |
| **⚠ `video_recording_task_handler.go`** | **未列** | **13** | ⚠ 漏列 | 全为 err.Error() GinError；多数 CodeInvalidRequest |
| **⚠ `apikey_handler.go`** | **未列** | **2** | ⚠ 漏列 | 2 处 parse-error GinError；其余已走 HandleError (line 107, 157 等) |
| **总计** | **CONTEXT 顶部"27"** / D-02.3 求和 58 | **实际 70 (26+7+5+5+3+3+2+2+2+2+13+2)** | ⚠ 严重不一致 | 见下文分析 |

### 差异分析（planner 必读）

**(a) CONTEXT 自身内部矛盾**：顶部 `<domain>` 说"9 个 handler 文件共 27 处"；D-02.3 自身列了 **10 个**文件求和=58。两数都对不上。从 D-02.3 各文件逐项匹配实际 grep（除 video_recording_task/apikey 漏列外，其余 10 个文件计数全对）推断：**"27" 实际只是 `ppt_handler.go` 单文件的数字**，被误抄到顶部成为"总数"。

**(b) 漏列的 2 个文件**：
- `video_recording_task_handler.go` 共 13 处 err.Error() GinError 调用 (lines 175, 190, 211, 226, 254, 270, 317, 366, 441, 488, 534, 569, 630)。其中 line 190/226/270/317/366/441/488/534/569/630 是 `response.GinError(c, response.CodeInvalidRequest, err.Error())`——把任何 task service 错误硬编码为 400，是典型 classify 散点。
- `apikey_handler.go` 仅 2 处 parse-error (lines 87, 229)；其余 endpoint 已在前 phase 切到 HandleError。

**(c) `ppt_handler.go` 实际 26 而非 27**：差异 1 处，可能是 CONTEXT 写时基于不同 HEAD 或把 `line 916 case` 单独算了一行；不影响工作量量级。建议 planner 在 Wave 0 用 grep 重新锁定精确行号清单作为任务输入。

### 建议 plan 调整

- 总散点数应声明为 **~70**（不是 27）；文件数 12（不是 9 或 10）
- atomic commit 数从 "10" 调整为 **"12 + 1（classifyAuthLoginError formal fn 删除可单独 commit 或合并到 auth_handler commit）"**
- `video_recording_task_handler.go` 与 `apikey_handler.go` 必须纳入 D-02.5 表驱动测试覆盖范围

---

## §2 internal/errors 架构（D-02.4 / D-03.1 / D-04 全部 confirmed）

### 2.1 Sentinel 全表（41 个，CONTEXT "30+" ✅）

`internal/errors/errors.go` `var (...)` 块 line 9-89，按业务域分组：

| 域 | Sentinels | 计数 |
|----|-----------|------|
| 通用 | ErrNotFound, ErrAlreadyExists, ErrInvalidInput, ErrUnauthorized, ErrForbidden, ErrInternal | 6 |
| 业务核心 | ErrVideoFileNotFound, ErrTaskNotFound, ErrTaskInProgress, ErrInvalidFileType, ErrFFmpegFailed, ErrTranscriptionFailed, ErrSplitFailed, ErrInsufficientQuota, ErrServiceUnavailable, ErrDuplicateRecord, ErrForeignKeyConstraint | 11 |
| 用户/角色 | ErrUserNotFound, ErrUsernameExists, ErrEmailExists, ErrRoleNotFound, ErrSystemAdminProtected | 5 |
| 认证 | ErrADAccountNotFound, ErrUserDisabled, ErrADConfigError, ErrADUnreachable | 4 |
| Token | ErrTokenInvalid, ErrTokenExpired, ErrTokenNotYetValid, ErrTokenReplayed | 4 |
| Role 补充 | ErrRoleNameExists, ErrSystemRoleProtected, ErrRoleInUse, ErrPermissionNotFound | 4 |
| APIKey | ErrAPIKeyNotFound, ErrAPIKeyInvalid, ErrAPIKeyExpired, ErrAPIKeyDisabled, ErrAPIKeyIPNotAllowed | 5 |
| PPT file | ErrPPTFileNotFound | 1 |
| Transcription 补充 | ErrTranscriptionUnavailable | 1 |
| **总计** | | **41** |

### 2.2 `BusinessError` typed 类型（D-02.4 / D-03.4 confirmed）

```go
// internal/errors/errors.go:117-144
type BusinessError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Err     error  `json:"-"`
}

func (e *BusinessError) Error() string { /* ... */ }
func (e *BusinessError) Unwrap() error { return e.Err }

func NewBusinessError(code, message string, err error) *BusinessError {
    return &BusinessError{Code: code, Message: message, Err: err}
}
```

`errors.As(err, &be)` 是检测 BusinessError 的标准手法——`SentinelField` helper 必须使用此手法（D-03.4）。

### 2.3 Code 常量集（D-02.5 表驱动测试 wantCode 依据）

```go
// internal/errors/errors.go:147-158
const (
    CodeNotFound             = "NOT_FOUND"
    CodeAlreadyExists        = "ALREADY_EXISTS"
    CodeInvalidInput         = "INVALID_INPUT"
    CodeUnauthorized         = "UNAUTHORIZED"
    CodeForbidden            = "FORBIDDEN"
    CodeInternalError        = "INTERNAL_ERROR"
    CodeServiceUnavailable   = "SERVICE_UNAVAILABLE"
    CodeTaskInProgress       = "TASK_IN_PROGRESS"
    CodeFFmpegError          = "FFMPEG_ERROR"
    CodeForeignKeyConstraint = "FOREIGN_KEY_CONSTRAINT"
)
```

**注意：Code 是 string 类型**，不是 int。CONTEXT D-02.5 表驱动测试示意中 `wantCode int` 是 **handler 层 `response.Code*` int 常量**，不是 `internal/errors.Code*` string 常量。两者关系：

| `response.Code*` (int, 在 pkg/response) | 数值 | 对应 `internal/errors.Code*` (string, 在 internal/errors) |
|---|---|---|
| CodeInvalidRequest | 1001 | CodeInvalidInput |
| CodeUnauthorized | 1002 | CodeUnauthorized |
| CodeForbidden | 1003 | CodeForbidden |
| CodeNotFound | 1004 | CodeNotFound |
| CodeInternalError | 1005 | CodeInternalError / CodeServiceUnavailable / CodeFFmpegError / CodeForeignKeyConstraint (均映射到此 int) |
| CodeDuplicateRecord | 1006 | CodeAlreadyExists / CodeTaskInProgress |
| CodeTooManyRequests | 1007 | (无对应 string Code) |

`internal/errors/mapping.go:15-23` 用本地 `respCode*` int 镜像 `response.Code*`，避免循环依赖（注释明示"新增时务必同步两边"）。

### 2.4 `IsKnownError` slice 顺序全文（D-03 SentinelField helper 必须镜像）

```go
// internal/errors/mapping.go:127-158
func IsKnownError(err error) bool {
    if err == nil {
        return false
    }
    var be *BusinessError
    if errors.As(err, &be) {
        return true
    }
    for _, sentinel := range []error{
        ErrNotFound, ErrTaskNotFound, ErrVideoFileNotFound,
        ErrUserNotFound, ErrRoleNotFound, ErrADAccountNotFound, ErrPermissionNotFound,
        ErrAPIKeyNotFound, ErrPPTFileNotFound,
        ErrUnauthorized, ErrForbidden, ErrInvalidInput, ErrInvalidFileType,
        ErrAlreadyExists, ErrTaskInProgress, ErrUsernameExists, ErrEmailExists,
        ErrRoleNameExists, ErrRoleInUse,
        ErrSystemAdminProtected, ErrSystemRoleProtected, ErrUserDisabled,
        ErrAPIKeyDisabled, ErrAPIKeyIPNotAllowed,
        ErrTokenInvalid, ErrTokenExpired, ErrTokenNotYetValid, ErrTokenReplayed,
        ErrAPIKeyInvalid, ErrAPIKeyExpired,
        ErrInsufficientQuota,
        ErrServiceUnavailable, ErrADConfigError, ErrADUnreachable,
        ErrTranscriptionUnavailable,
        ErrFFmpegFailed, ErrTranscriptionFailed,
        ErrSplitFailed, ErrInternal,
        ErrDuplicateRecord, ErrForeignKeyConstraint,
    } {
        if errors.Is(err, sentinel) {
            return true
        }
    }
    return false
}
```

**Slice 顺序 = D-03.3 优先级规则**。`SentinelField` helper 必须用**同一 slice**（建议直接 import 并调用 `errors.IsKnownError` 内部逻辑，或导出 `FirstKnownSentinelName(err) (string, bool)` helper 至 `internal/errors`，避免在 `pkg/response` 重新维护 slice 副本）。

观察：slice 中 NotFound 类优先（`ErrTaskNotFound` 在 `ErrNotFound` 之前），符合 D-03.3 注释"更具体优先"。但**`ErrNotFound` 排在第三位（其实 slice 是 `ErrNotFound` 在最前，"通用" NotFound 在 task/video 之后是顺序的）**——读源码：slice 开头是 `ErrNotFound, ErrTaskNotFound, ErrVideoFileNotFound,...`。即"通用 ErrNotFound 在最前"。这意味着如果一个 err 同时 wrap 了 `ErrTaskNotFound` 和 `ErrNotFound`（罕见），`errors.Is` 在 `ErrNotFound` 处即返回——SentinelField 会显示 `ErrNotFound` 而非更具体的 `ErrTaskNotFound`。

**⚠ SentinelField helper 注意点**：若想 "更具体优先"，需把 slice 重排为 task/video 在前；但 CONTEXT D-03.3 明示"按 `mapping.go` IsKnownError slice 顺序"，故应**忠实镜像**而非重排。Planner 不应在此处自作主张优化。

### 2.5 `MapToHTTPStatus` / `FromGORM` / `NotFound` 行号确认

| 函数 | 行号 | CONTEXT 声明 | 状态 |
|------|------|-------------|------|
| `MapToHTTPStatus(err) (int, int, string)` | mapping.go:32 | "mapping.go:32" | ✅ |
| `mapBusinessError(be)` (内部) | mapping.go:100 | — | 私有，BusinessError.Code 分支 |
| `IsKnownError(err) bool` | mapping.go:127 | "mapping.go:127" | ✅ |
| `FromGORM(err, fallback) error` | mapping.go:164 | "mapping.go:164" | ✅ |
| `NotFound(what, id) error` | mapping.go:180 | "mapping.go:180" | ✅ |

---

## §3 `pkg/response.HandleError` 现状（D-02 / canonical_refs confirmed）

```go
// pkg/response/response.go:161-180
// HandleError 把 err 通过 errors.MapToHTTPStatus 映射为 HTTP 响应并写入。
//
// STYLE-001 (Phase 19) 决策 3 组件 B：handler 把内联 switch 换成
// `if response.HandleError(c, err) { return }`——命中已知 sentinel 即写入响应并返回 true；
// 未识别的错误也写入（保守 500，永不 200）但返回 false，调用方可选择继续自行处理。
//
// 守卫：
//   - err==nil → no-op，返回 false。
//   - c.Writer.Written() 已为 true（handler 已写响应）→ no-op，返回 false（防双写）。
//
// 用 GinErrorWithStatus 显式指定 httpStatus，因为 GinError 的 switch 不识别
// CodeDuplicateRecord（会落到默认 500，而 409 Conflict 需要显式状态码）。
func HandleError(c *gin.Context, err error) bool {
    if err == nil || c.Writer.Written() {
        return false
    }
    httpStatus, respCode, message := errors.MapToHTTPStatus(err)
    GinErrorWithStatus(c, httpStatus, respCode, message)
    return errors.IsKnownError(err)
}
```

**关键设计点（planner 与执行器必读）：**
1. **返回值是 `errors.IsKnownError(err)`**——即"是否已知 sentinel/BusinessError"。**未识别 err 也写入 500 响应**，只是返回 false。
2. CONTEXT D-02.4 的 handler 模式 `if response.HandleError(c, err) { return }` 含义：**只要 err != nil 就 return**（因为已写响应）；返回值仅在调用方需要"err 已识别 vs 未知"分支时有用。本 phase 替换后所有 handler 都应直接 `if response.HandleError(c, err) { return }`，不依赖返回值分支。
3. **`pkg/response` 已 import `internal/errors`**（line 7）——`SentinelField` helper 放 `pkg/response` 不引入新循环依赖。但反方向不可（`internal/errors` 不能 import `pkg/response`，mapping.go 注释 line 11-14 明示）。
4. **`zap` 未在 `pkg/response` import**——若 `SentinelField` 放此包，需新增 `go.uber.org/zap` 依赖（zap 已在 go.mod，无新外部包）。
5. **无既有 SentinelField / zap 相关 helper** ✅（CONTEXT 声明的"应为无"确认）。

---

## §4 `zap.Error(err)` 调用点统计（D-03.6 scope）

**Grep:** `zap\.Error\(err\)` 在 `internal/handlers/` + `internal/services/` 范围。

### 4.1 Handler 层（98 处）

| 文件 | 调用点数 | 行号 |
|------|---------|------|
| admin_handler.go | 5 | 145, 211, 240, 281, 412 |
| auth_handler.go | 3 | 88, 148, 174 |
| audit_handler.go | 7 | 79, 98, 145, 198, 225, 251, 269 |
| apikey_handler.go | 10 | 105, 155, 249, 264, 307, 322, 367, 382, 436, 480 |
| dashboard_handler.go | 1 | 35 |
| file_handler.go | 9 | 92, 106, 128, 163, 177, 218, 232, 293, 310 |
| input_config_handler.go | 5 | 65, 91, 131, 181, 219 |
| notification_handler.go | 7 | 78, 103, 121, 139, 157, 182, 196 |
| ppt_handler.go | 23 | 120, 155, 209, 289, 299, 369, 446, 460, 531, 545, 600, 667, 685, 702, 749, 757, 774, 836, 909, 927, 993, 1075, 1089 |
| role_handler.go | 4 | 57, 155, 192, 261 |
| system_handler.go | 2 | 180, 209 |
| transcription_handler.go | 1 | 439 |
| user_handler.go | 4 | 60, 165, 202, 249 |
| video_file_handler.go | 10 | 68, 133, 214, 236, 259, 279, 291, 357, 372, 415 |
| video_recording_task_handler.go | 7 | 112, 283, 330, 387, 454, 501, 962 |
| **小计** | **98** | |

### 4.2 Service 层（110 处，主要文件）

| 文件 | 调用点数 |
|------|---------|
| video_file_service.go | 22 |
| transcription_service.go | 20 |
| video_recording_task_service.go | 9 |
| ppt_editor_service.go | 11 |
| usb_device_scanner.go | 6 |
| auth/ad_auth.go | 11 |
| auth/local_auth.go | 6 |
| dashboard_service.go | 5 |
| video_scheduler.go (scheduler) | 28 |
| conversion_service.go | 7 |
| splitting_service.go | 5 |
| config_service.go | 3 |
| frame_extractor.go | 4 |
| input_config_service.go | 3 |
| ppt_file_service.go | 5 |
| ppt_merge_service.go | 3 |
| python_deps.go | 3 |
| storage/file_service.go | 3 |
| huawei/manager.go | 5 |
| huawei/client.go | 2 |
| auth/sm4_token.go | 4 |
| (其他散布) | ~14 |
| **小计** | **~110** |

### 4.3 总量与 plan 影响

**handler + service = ~208 处 `zap.Error(err)` 调用点**需在 D-03.6 升级为 `zap.Error(err), response.SentinelField(err)`。

**建议：**
- 这不是 27 处级的小工作量，而是 **200+ 处的机械替换**
- 由于"零侵入"（只加一个 field），可用单次 grep+sed 风格批量替换，每文件 1 commit
- Planner 应在 D-06.1 commit 计划中新增独立的 `feat(20-logger): batch upgrade zap.Error call-sites` 系列原子 commit，**不与 classify 替换混在一起**（避免 atomic commit 过大）
- 或：执行器可在每个 classify 替换 commit 中**顺带**升级该文件 `zap.Error` 调用点（合并到 12 文件 atomic commit 内，每文件 1 commit 完成"classify + logger"双升级）

---

## §5 cmd / docs / Makefile 现状（D-04 generator analog）

| 项 | 路径 | 存在 | 备注 |
|----|------|------|------|
| `cmd/server/` | 是 | ✅ | 含 `main.go` + `app.go` + 3 个 `*_test.go` (cors_test, phase18_integration_test, taskservice_adapter_ctx_test) |
| `cmd/error-doc-gen/` | 否 | — | **本 phase 新建**（D-04.2） |
| `docs/errors.md` | 否 | — | **本 phase 新建**（D-04） |
| `docs/` 目录 | 是 | ✅ | 已存在 `docs/audits/2026-07-30-backend-code-review.md`（**不可变 source of truth，禁止改动**）+ 其他 |
| `Makefile` | **否** | ⚠ | **不存在！** CONTEXT D-04.4 "已有 Makefile pattern" 声明错误 |
| `makefile` (小写) | 否 | — | 同上 |
| `.github/workflows/` | 未在本 phase 调研范围 | — | CI 是否存在 planner 可选查 |

**Plan 调整建议（D-04.4）：**

由于 Makefile 不存在，planner 有 3 个备选方案（Claude's Discretion 范围内）：

1. **新建 Makefile**（D-04.4 原意）：从零写一个简单 Makefile，至少含 `check-errors-doc` target。**风险**：项目从未有 Makefile，可能引入 build 工具链期望差异；需评估 CI 是否能跑 make
2. **改用 `tools/check-errors-doc.sh` shell 脚本** + CI 直接调脚本：避免引入 Make 工具依赖；与项目"Go 单 binary"风格一致
3. **直接 `go generate ./internal/errors/...` + CI step**：最轻量，无 Makefile / 无 shell 脚本；CI 配置文件中直接加一步

**推荐方案 2 或 3**——项目历史上未用 Make，强行引入徒增 onboarding 成本。Planner 应在 PLAN Wave 0 把此决策明确（或 escalate 给用户在 discuss-phase 补一个决策）。

---

## §6 `internal/middleware/error_mapper.go` 职责（D-03.7 confirmed）

```go
// internal/middleware/error_mapper.go (全文 48 行)
// ErrorMapper 是 backstop 错误映射中间件（STYLE-001 决策 3 组件 C）。
//
// c.Next() 后，若响应尚未写入且 c.Errors 非空，则把最后一个 error 通过
// response.HandleError 映射为 HTTP 响应，避免客户端因 handler 只 c.Error(err)
// 未写响应而收到空体。
//
// 零行为风险保证：此中间件不改变任何现有 handler 行为——handler 仍可自行 c.JSON
// 错误响应；仅当 handler 通过 c.Error(err) 记录错误但未写入响应时，本中间件兜底。
// c.Writer.Written() 守卫防止与 handler 自身响应双写。
```

**职责（D-03.7 confirmed "Phase 19 已带开，本阶段不扩展"）：**
- backstop middleware：handler 只调 `c.Error(err)` 未写响应时兜底
- 不改变现有 handler 行为；`c.Writer.Written()` 守卫防双写
- **本 phase 不修改此文件**——D-03.7 locked

**与 D-03 SentinelField 的关系**：error_mapper.go 自身 logger 已在 backstop 时输出 `zap.Error(lastErr)`（line 40, 45）——D-03.6 升级时这两处也是 call-site。但 D-03.7 锁定"不动"，建议执行器把 middleware 的 zap.Error 升级与 D-03.6 合并（或显式 deferred 到下个 phase）。Planner 需在 PLAN 中明示。

---

## §7 cross-package local error var survey（D-01.3 仅 survey）

**Grep:** `^var\s+Err\w+\s*=\s*errors\.New` 与 `var\s+Err\w+\s*=` 在 `internal/` 范围。

| 文件:行号 | 声明 | 是否走 internal/errors | 风险 |
|-----------|------|----------------------|------|
| `internal/auth/ad_auth.go:22` | `var ErrADUserNotRegistered = errors.New("账号未在系统中注册，请联系管理员添加")` | ❌ 未走 | 当前 `classifyAuthLoginError` 显式判 `auth.IsADUserNotRegistered` 返回 403；切到 HandleError 后落入 default → **500 ad-hoc**。**⚠ 影响 Phase 20 REQ-20a**——必须在 D-02.2 "复用现有 + 补漏" 原则下处理 |
| `internal/auth/hlstoken/hls_token.go:29` | `var ErrTokenReplayed = errors.New("token 已被使用（防重放）")` | ❌ 未走 | **⚠ 名称冲突**：与 `internal/errors/errors.go:59 ErrTokenReplayed = errors.New("token reuse detected")` 同名不同实体。当前 hlstoken 包内自用，不影响 mapping.go；但若 hlstoken 错误传到 handler，HandleError 走 IsKnownError 会匹配到 internal/errors 版本（因 `errors.Is` 比的是 identity）——可能误判 |
| `internal/scheduler/video_scheduler_test.go:437` | `var ErrTaskNotFound = fmt.Errorf("task not found")` | ❌ 未走（test-only） | **⚠ 测试 fixture 名称冲突**；test-only 不影响生产；记入 commit body 即可 |
| `internal/huawei/error_codes.go:38` | `var ErrorMessages = map[int]string{...}` | — | 不是 sentinel，是错误码到消息的查找表；不相关 |

**总结（D-01.3 仅 survey 入 commit body）：**
- 2 个生产代码 local sentinel var 未走 internal/errors（`ad_auth.ErrADUserNotRegistered`, `hlstoken.ErrTokenReplayed`）
- 1 个 test-only fixture var 名称冲突
- 本 phase **不主动迁**——但 `ErrADUserNotRegistered` 因 REQ-20a 切换会触发行为变化（见 §9）

---

## §8 测试现状（D-05）

### 8.1 现有 handler 测试文件清单（D-05.5 不删既有）

| 文件 | 覆盖测试 |
|------|---------|
| `admin_ad_test.go` | TestAdminHandler_GetAuthConfig, _UpdateAuthConfig_LocalMode/ADMode, _TestADConnection, _TestADConnection_Port389Warning, TestMaskAccessToken |
| `auth_handler_test.go` | **TestClassifyAuthLoginError**（9 子测试，含 sentinel/wrapped/AD config 等所有 case） |
| `file_handler_test.go` | （待 runtime 确认；存在即可） |
| `input_config_handler_test.go` | TestInputConfigHandler_ListConfigs, _GetConfig, _CreateConfig, _UpdateConfig, _DeleteConfig, _TestConnection, _ScanUSBDevices |
| `split_handler_test.go` | TestSplitHandler_SubmitSplit, _SubmitSplit_InvalidMarkers, _SubmitSplit_UnauthorizedVideo, _GenerateSnapshot, _GetSplitStatus, _GetSegments, TestModeParameterValidation |
| `transcription_handler_test.go` | TestTranscriptionHandler_SubmitBatchTranscription_* (6 子测试), _GetBatchTranscriptionStatus, _GetBatchTranscriptionStatus_NotFound |
| `transcription_handler_cloud_test.go` | TestCloudNoSamplingRateRequired, TestLocalModeRequiresSamplingRate, TestModeDefaultToLocal |
| `video_file_handler_test.go` | TestVideoFileHandler_BatchDownloadFiles_* (6 子测试) |

**`auth_handler_test.go` 关键约束（D-05.5）：**
- `TestClassifyAuthLoginError` line 17-99 直接调用 `classifyAuthLoginError(tt.err)`（line 84）—— **删除 formal 函数后此测试编译失败**
- Planner 必须在 PLAN 中显式声明：**删除 `classifyAuthLoginError` 时同步重写 `TestClassifyAuthLoginError` 为 `TestLogin_HandleError_ClassifyDrop`**（CONTEXT D-02.5 示意名字），改用 `response.HandleError(ctx, tt.err)` + 断言 `testRecorder.Code` + JSON `code` 字段
- 9 个子测试中至少 **2 个 wantStatus 需更新**：
  - `"AD configuration error maps to 500"` (line 62-66) → 实际将变 **503** (StatusServiceUnavailable)
  - `"AD unreachable maps to 500"` (line 67-72) → 同上 **503**
  - `"sentinel maps to forbidden"` 用 `auth.ErrADUserNotRegistered` (line 25-29) → 若不补 mapping.go case 将变 **500 ad-hoc**（见 §9 风险 R-3）

### 8.2 Phase 17 触及的 12 个包（D-05.4 跨包回归 scope）

源自 `.planning/phases/17-56-p0-p1-p2/17-02-SUMMARY.md:73` + `17-REVIEWS.md:38`：

| # | 包路径 | 备注 |
|---|--------|------|
| 1 | `internal/config` | |
| 2 | `internal/auth` | |
| 3 | `internal/auth/hlstoken` | 子包 |
| 4 | `cmd/server` | |
| 5 | `internal/huawei` | |
| 6 | `internal/services` | |
| 7 | `internal/services/storage` | 子包 |
| 8 | `internal/scheduler` | |
| 9 | `internal/recorder` | |
| 10 | `internal/middleware` | |
| 11 | `internal/handlers` | 本 phase 主战场 |
| 12 | `internal/models` | (17-02 SUMMARY 提到 migrations；17-01 SUMMARY 提到 models 测试) |

**D-05.4 跨包回归命令：**
```bash
go test -race ./internal/config/... ./internal/auth/... ./cmd/server/... ./internal/huawei/... \
  ./internal/services/... ./internal/scheduler/... ./internal/recorder/... \
  ./internal/middleware/... ./internal/handlers/... ./internal/models/...
```

或简化：`go test -race ./...`（全量，但会包含未触及包——成本低，建议直接全量）。

---

## §9 风险与坑（含 deferred 第 4 项边界守护）

### R-1: classify 散点总数与文件数与 CONTEXT 不符（**高**，已在 §1 详细）
- 实际 12 文件 ~70 处（非 9 文件 27 处）
- Planner 必须修正 commit 数与文件清单，否则会漏改 `video_recording_task_handler.go` / `apikey_handler.go`

### R-2: Makefile 不存在（**中**，已在 §5 详细）
- D-04.4 "已有 Makefile pattern" 声明错误
- 决策点：新建 Makefile vs shell 脚本 vs CI 直接调 go generate

### R-3: `auth.ErrADUserNotRegistered` 切到 HandleError 后变 500 ad-hoc（**高**）
**根因：** `internal/auth/ad_auth.go:22` 定义了 local sentinel `ErrADUserNotRegistered`，未走 `internal/errors`。`classifyAuthLoginError` 显式调 `auth.IsADUserNotRegistered(err)` 返回 403；但 mapping.go `IsKnownError` slice 内**没有**此 sentinel。

**影响：** `Login` 切到 `response.HandleError(c, err)` 后：
- err = `auth.ErrADUserNotRegistered`（未 wrap）→ `errors.As` 不是 BusinessError → `IsKnownError` slice 不命中 → 返回 false → `HandleError` 走 default → **500 + respCodeInternalError**
- 当前行为：403 + CodeForbidden

**3 个处理方案（planner 决断）：**
1. **补 mapping.go case**：在 Forbidden 分支加 `errors.Is(err, auth.ErrADUserNotRegistered)` —— 但需 `internal/errors` import `internal/auth`，可能引入循环依赖（需验证）
2. **补 internal/errors sentinel**：新增 `ErrADUserNotRegistered = errors.New(...)` 到 internal/errors，`ad_auth.go` 改用 internal/errors 版本——违反 D-02.2 "不主动加新 sentinel"
3. **接受 500**：把此 case 当作"未知错误"通过 SentinelField 输出 `sentinel_type="ad-hoc"`——但破坏现有 403 语义，**用户登录 UX 回退**

**推荐方案 2 + 同步删除 `ad_auth.go:22` local var**：把 ad_auth 的 ErrADUserNotRegistered 迁到 internal/errors（属 D-02.2 "补漏" 范畴，非"主动加新"），mapping.go Forbidden 分支添加 case。`ad_auth.go` 改 import `internal/errors` 并 alias。**但这超出 D-01.3 "不主动迁 cross-package" 约束——需 planner 在 PLAN 中显式声明此为 D-02.2 补漏例外，或 escalate 用户决策。**

### R-4: `ErrADConfigError` / `ErrADUnreachable` 状态码 500 → 503（**中**）
**根因：** `classifyAuthLoginError` line 54-56 把这两个映射到 `http.StatusInternalServerError (500)`；mapping.go line 84-87 映射到 `http.StatusServiceUnavailable (503)`。

**影响：** `TestClassifyAuthLoginError` 的 2 个子测试 wantStatus 需更新（500 → 503）。auth_handler.go:38-39 注释已承认此差异（"如未来改用 HandleError 则走 mapping.go 的 503 ServiceUnavailable"）。

**处置：** 接受 503，更新测试。这是 **更准确的状态码**（503 表达"AD 基础设施不可用"比 500 更精确），不是 regression。Plan 需明示。

### R-5: `hlstoken.ErrTokenReplayed` 名称冲突（**低**，D-01.3 survey only）
- 与 `internal/errors.ErrTokenReplayed` 同名不同实体
- 当前 hlstoken 错误若传到 handler，`HandleError` 走 IsKnownError 用 `errors.Is` identity 比对——不会误匹配（identity 不等）
- 但若 `SentinelField` helper 输出 sentinel 名时用 reflect 包名而非 errors.Is，可能输出错误名字
- **本 phase 不处理，仅入 commit body**

### R-6: typed error kind deferred 边界守护（**低**，REQ-20d）
- 本 phase 不实现 typed kind 三层 enum
- **但 `SentinelField` 字符串形态必须为下阶段 typed kind 留扩展点**：
  - 当前 `"sentinel_type": "ErrXxx"` / `"BusinessError(code=YYY)"` / `"ad-hoc"` 三态字符串
  - 下阶段若改为 typed kind enum（如 `kind=Sentinel/BusinessError/AdHoc` + `sentinel_type=ErrXxx` 双字段），SentinelField 返回 `zap.Field` 的 API 需稳定
- **建议**：本 phase SentinelField 返回 `zap.Field`（不是 zap.String 直接），未来可扩展为 `zap.Object("error_kind", ...)` 多字段而不破坏调用方

### R-7: `video_recording_task_handler.go` 大量 CodeInvalidRequest 误用（**中**）
- 该 handler 把 task service 任何错误（含 server-side 故障）硬编码为 `response.CodeInvalidRequest (400)`——line 190/226/270 等
- 切到 HandleError 后这些将根据 sentinel/BusinessError 正确映射（多为 500/503）
- **行为变化大**——可能破坏前端期望（前端可能根据 400 vs 500 显示不同 UX）
- Plan 需在 D-02.5 表驱动测试中覆盖此 handler 全部 10 个 endpoint，**显式声明"前后状态码可能不同，新状态码才是正确语义"**

---

## Validation Architecture (Nyquist)

> `workflow.nyquist_validation` 未显式 false，按 enabled 处理。

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing + `github.com/stretchr/testify/assert` (已有) |
| Config file | 无（Go 标准约定：`*_test.go` 同包；`go test` 自动发现） |
| Quick run command | `go test ./internal/handlers/... ./internal/errors/... ./pkg/response/... -count=1` |
| Full suite command | `go test -race ./...` （~12 个 phase-17 触及包） |
| Sanity build | `go build ./... && go vet ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| REQ-20a-classify | classify 散点全量替换为 HandleError，HTTP 状态码符合 mapping.go 语义 | unit (table-driven) | `go test ./internal/handlers/ -run TestClassifyReplacement -count=1` | ❌ Wave 0 新建（12 个文件每个 1 个 _test.go） |
| REQ-20a-formal | 删除 `classifyAuthLoginError` 后 Login 走 HandleError | unit (rewrite existing) | `go test ./internal/handlers/ -run TestLogin_HandleError_ClassifyDrop -count=1` | ❌ 重写 `auth_handler_test.go::TestClassifyAuthLoginError` |
| REQ-20a-ad-user-not-registered | `auth.ErrADUserNotRegistered` 切 HandleError 后行为（403/500 取决于 R-3 决策） | unit | `go test ./internal/handlers/ -run TestLogin_HandleError_ADUserNotRegistered -count=1` | ❌ Wave 0 新建 |
| REQ-20b-sentinel-field | SentinelField 4 类输入 (sentinel/BusinessError/unknown/nil) | unit | `go test ./pkg/response/ -run TestSentinelField -count=1` （或 ./pkg/logging/） | ❌ Wave 0 新建 |
| REQ-20b-priority | SentinelField 优先级 = IsKnownError slice 顺序 | unit | `go test ./pkg/response/ -run TestSentinelField_Priority -count=1` | ❌ Wave 0 新建 |
| REQ-20c-generator | docs/errors.md generator 输出 (i) 已知 sentinel 全表 (ii) BusinessError 全表 (iii) missing filepath 错误 (iv) diff 一致性 | unit + smoke | `go test ./cmd/error-doc-gen/... -count=1` + `go generate ./internal/errors/... && git diff --quiet docs/errors.md` | ❌ Wave 0 新建 |
| REQ-20c-call-site-count | call-site count 列正确 grep | smoke | `go generate ./internal/errors/... && grep -c 'Err[       ]' docs/errors.md` | ❌ Wave 0 新建 |
| REQ-20-regression | 12 phase-17 触及包 go test -race 全绿 | regression | `go test -race ./internal/config/... ./internal/auth/... ./cmd/server/... ./internal/huawei/... ./internal/services/... ./internal/scheduler/... ./internal/recorder/... ./internal/middleware/... ./internal/handlers/... ./internal/models/...` | ✅ 已有（Phase 17 baseline） |
| REQ-20-build | go build ./... 0 错误 | smoke | `go build ./...` | ✅ 已确认（基线绿） |

### Sampling Rate

- **Per task commit (atomic commit 级)：** `go build ./... && go test ./internal/handlers/ -count=1` （< 5 秒）
- **Per wave merge：** `go build ./... && go vet ./... && go test -race ./internal/handlers/... ./internal/errors/... ./pkg/response/...` （< 30 秒）
- **Phase gate (verify-work 前)：** `go build ./... && go vet ./... && go test -race ./...` 全绿

### Wave 0 Gaps

- [ ] 新建 `internal/handlers/{ppt,input_config,file,video_file,admin,user,transcription,split,role,auth,video_recording_task,apikey}_handleerror_test.go` —— 每文件 1 个表驱动测试覆盖 4 类错误 (sentinel/sentinel wrap/BusinessError/unknown)
- [ ] 重写 `internal/handlers/auth_handler_test.go::TestClassifyAuthLoginError` → `TestLogin_HandleError_ClassifyDrop`（同步更新 AD config 503 + ADUserNotRegistered 行为）
- [ ] 新建 `pkg/response/sentinel_field_test.go`（或 `pkg/logging/sentinel_field_test.go`）—— 4 类输入断言
- [ ] 新建 `cmd/error-doc-gen/main_test.go` —— generator 4 项验证
- [ ] 新建 `docs/errors.md` —— 由 generator 产出
- [ ] 新建 `Makefile` 或 `tools/check-errors-doc.sh` —— CI 集成（R-2 决策依赖）
- [ ] 无新框架依赖——`testify/assert` 已在 go.mod（auth_handler_test.go:14 已用）

---

## 关键代码摘录

### 摘录 1：`internal/errors/mapping.go::IsKnownError` 全文

见 §2.4。Slice 顺序 = D-03.3 SentinelField 优先级源。共 41 个 sentinel（含 ErrDuplicateRecord, ErrForeignKeyConstraint 末尾两项，与 errors.go 全表对齐）。

### 摘录 2：`internal/errors/errors.go::BusinessError`

见 §2.2。`Code/Message/Err` 三字段；`Unwrap()` 返回 `e.Err`；`NewBusinessError(code, message, err)` 构造。

### 摘录 3：`pkg/response/response.go::HandleError`

见 §3。签名 `HandleError(c *gin.Context, err error) bool`；返回 `errors.IsKnownError(err)`；err==nil 或 Writer.Written() 时 no-op 返 false。

### 摘录 4：`internal/handlers/auth_handler.go::classifyAuthLoginError` (line 46-61)

```go
func classifyAuthLoginError(err error) (code int, httpStatus int) {
    switch {
    case auth.IsADUserNotRegistered(err):
        return response.CodeForbidden, http.StatusForbidden
    case errors.Is(err, apperrors.ErrADAccountNotFound):
        return response.CodeNotFound, http.StatusNotFound
    case errors.Is(err, apperrors.ErrUserDisabled):
        return response.CodeForbidden, http.StatusForbidden
    case errors.Is(err, apperrors.ErrADConfigError),
        errors.Is(err, apperrors.ErrADUnreachable):
        return response.CodeInternalError, http.StatusInternalServerError   // ⚠ 切到 HandleError 后将变 503
    case errors.Is(err, apperrors.ErrUnauthorized):
        return response.CodeUnauthorized, http.StatusUnauthorized
    }
    return response.CodeInvalidCredential, http.StatusInternalServerError
}
```

调用点：`auth_handler.go:91 code, _ := classifyAuthLoginError(err)` → `line 92 response.GinError(c, code, err.Error())`。

### 摘录 5：`internal/handlers/ppt_handler.go:912-920` (典型 string-match 散点)

```go
// Map error messages
errMsg := err.Error()
switch {
case errMsg == "frame bytes cannot be empty":
    response.GinError(c, response.CodeInvalidRequest, "帧数据不能为空")
case strings.Contains(errMsg, "frame bytes too large"):
    response.GinError(c, response.CodeInvalidRequest, "帧数据过大")
default:
    response.GinError(c, response.CodeInternalError, "插入幻灯片失败: "+err.Error())
}
```

**替换示意（CONTEXT specifics 已给出）：**
- service 层: `return fmt.Errorf("frame bytes too large: %w", internalerrors.ErrInvalidInput)`
- handler 层: `if response.HandleError(c, err) { return }`
- mapping.go 命中 `ErrInvalidInput` → 400/1001/"请求参数无效"

### 摘录 6：`internal/middleware/error_mapper.go::ErrorMapper` (line 18-48)

见 §6。Backstop middleware，c.Next() 后扫描 c.Errors，未写入则调 HandleError 兜底。本 phase 不动（D-03.7）。

---

## Assumptions Log

> 本 phase 不引入新外部包（无 slopcheck 触发条件）；所有 `[VERIFIED]` 标签均经 runtime grep/Read 验证。

| # | Claim | Section | Confidence | Source |
|---|-------|---------|------------|--------|
| A1 | 散点总数 ~70（12 文件），非 CONTEXT 声明的 27（9 文件） | §1 | HIGH | runtime grep `internal/handlers/` |
| A2 | Makefile 不存在 | §5 | HIGH | `ls D:/CODE/ClaudeCode/record_V2/Makefile` exit 2 |
| A3 | `auth.ErrADUserNotRegistered` 未在 IsKnownError slice 内，切 HandleError 后变 500 | §9 R-3 | HIGH | Read ad_auth.go:22 + mapping.go:127-158 |
| A4 | AD config/unreachable 状态码 500 → 503 | §9 R-4 | HIGH | Read classifyAuthLoginError vs mapping.go:84-87 |
| A5 | Go 1.25.0（CONTEXT 说 1.24） | §5 / go.mod | HIGH | `cat go.mod` line 3 |
| A6 | `go build ./...` 0 错误 | §8 | HIGH | `go build ./...` exit 0 |
| A7 | `go test ./internal/handlers/...` 全绿 | §8 | HIGH | `go test -list '.*'` 输出 `ok ... 0.153s` |
| A8 | zap.Error(err) 调用点 ~208 处（handler 98 + service 110） | §4 | HIGH | runtime grep `internal/handlers/` + `internal/services/` |
| A9 | Phase 17 触及 12 个包 | §8.2 | HIGH | `.planning/phases/17-56-p0-p1-p2/17-02-SUMMARY.md:73` + `17-REVIEWS.md:38` |

无 `[ASSUMED]` 标签——本 phase 为轻量验证模式，所有 claim 都有 runtime 证据。

---

## Open Questions (RESOLVED)

> All 4 questions resolved during /gsd:plan-phase 20 via user decision (Q-1, Q-2) or Claude's Discretion locked in planner prompt (Q-3, Q-4). Resolutions landed in the PLAN.md files as noted.

### Q-1: `auth.ErrADUserNotRegistered` 处理方案（R-3） — ✅ RESOLVED
- **What we know:** 当前 classifyAuthLoginError 显式判 → 403；切 HandleError → 500 ad-hoc
- **What's unclear:** 用户是否接受方案 2（迁到 internal/errors，违反 D-01.3 不主动迁 cross-package）？
- **Recommendation:** Planner 在 PLAN Wave 0 中**默认采用方案 2**（补漏范畴，非"主动加新 sentinel"），并在 commit message 中显式声明"R-3 决策：迁移 ad_auth.ErrADUserNotRegistered 至 internal/errors 属 D-02.2 补漏例外"。若用户反对，则在 discuss-phase 补 D-07 决策。
- **✅ RESOLVED:** 用户在 plan-phase 拍板采用方案 2（迁移到 internal/errors，D-02.2 补漏例外）。落地于 20-01 Task 3。

### Q-2: Makefile vs shell 脚本 vs CI 直接调（R-2） — ✅ RESOLVED
- **What we know:** 项目无 Makefile 历史；CONTEXT D-04.4 期望"已有 pattern"是错的
- **What's unclear:** 用户偏好哪种 check 集成方式？
- **Recommendation:** Planner 在 PLAN 中**默认采用方案 3**（go:generate + CI step，最轻量），并在 commit message 中说明。或 escalate discuss-phase 补 D-08。
- **✅ RESOLVED:** 用户在 plan-phase 拍板采用方案 3（go:generate + CI step，不建 Makefile）。落地于 20-05 Task 2。

### Q-3: SentinelField helper 放 pkg/response 还是 pkg/logging（D-03.1 Claude's Discretion） — ✅ RESOLVED
- **What we know:** pkg/response 已 import internal/errors；pkg/logging 不存在（需新建）；call-site 跨 handler + service（200+ 处都需 import）
- **What's unclear:** 是否值得新建 pkg/logging 包？
- **Recommendation:** **放 pkg/response**。理由：(a) 已有循环依赖方向 `pkg/response → internal/errors`，再加 `zap.Field` 不引入新方向；(b) handler 已普遍 import pkg/response；(c) service 层 import pkg/response 也无障碍（无循环）。新建 pkg/logging 徒增包数。Planner 可在 PLAN Wave 0 锁定此选择。
- **✅ RESOLVED:** Claude's Discretion 锁定为 pkg/response（用户在 plan-phase 确认）。落地于 20-01 Task 2，调用 internal/errors.FirstKnownSentinelName 避免 slice 副本。

### Q-4: 散点精确行号是否在 Wave 0 重新锁定（R-1） — ✅ RESOLVED
- **What we know:** CONTEXT D-02.3 各文件计数大体对（除漏列 2 文件），但行号会随每次 commit 微变
- **What's unclear:** 是否在 PLAN 中固定行号清单？
- **Recommendation:** **不在 PLAN 中固定行号**——行号会随前序 commit 漂移；改为每文件给出"grep 模式 + 期望计数"作为验收条件。每个 atomic commit 自验"该文件 err.Error() GinError 计数 = 0"。
- **✅ RESOLVED:** 采用 grep 模式 + 期望计数作为 acceptance_criteria（非行号），已贯穿 20-02/20-03/20-04 各 plan 的 verify 块。

---

## Sources

### Primary (HIGH confidence)
- `internal/errors/errors.go` (183 lines, full read) — sentinel 全表 + BusinessError + Code 常量
- `internal/errors/mapping.go` (183 lines, full read) — MapToHTTPStatus + IsKnownError slice + FromGORM + NotFound
- `pkg/response/response.go` (181 lines, full read) — HandleError 签名 + Response/Code 常量
- `internal/middleware/error_mapper.go` (48 lines, full read) — backstop middleware 职责
- `internal/handlers/auth_handler.go` (295 lines, full read) — classifyAuthLoginError + Login 调用点
- `internal/handlers/auth_handler_test.go` (100 lines, full read) — TestClassifyAuthLoginError 9 子测试
- `internal/handlers/ppt_handler.go` lines 660-750 + 880-1010 (read) — string-match 散点典型 case
- runtime grep `internal/handlers/` (`err\.Error\(\)|strings\.Contains\(err|errors\.Is\(err|switch.*err|classify`) — 散点全量计数
- runtime grep `internal/` (`zap\.Error\(err\)`, `var\s+Err\w+\s*=`) — call-site 与 cross-package survey
- `go.mod` — `go 1.25.0`, module `github.com/NDCCCCCC/video-meeting-recorder`
- `go build ./...` exit 0 — 基线绿
- `go test -list '.*' ./internal/handlers/...` — 测试发现 OK

### Secondary (MEDIUM confidence)
- `.planning/phases/17-56-p0-p1-p2/17-02-SUMMARY.md:73` + `17-REVIEWS.md:38` — Phase 17 12 个触及包
- `.planning/STATE.md` §"Phase 19 Final Status" — Phase 19 baseline（11 commits + HandleError + mapping.go 三组件）

### Tertiary (LOW confidence)
- 无 — 本 phase 所有 claim 均有 runtime 证据

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — internal/errors + pkg/response 架构完整 verified，行号精确匹配
- Architecture: HIGH — Phase 19 baseline + HandleError/mapping.go/error_mapper.go 三组件全 confirmed
- Pitfalls: HIGH — 5 项差异全部 runtime 验证，行号 + 计数都有证据
- Validation: HIGH — 12 个 phase-17 包清单 + 现有测试文件清单 + 测试命令均 verified

**Research date:** 2026-08-01
**Valid until:** 2026-08-31 (30 days; Go 1.25 + 内部架构稳定，不易变)
**Base HEAD verified:** `570a2bc` (docs(state): record phase 20 context session)

## RESEARCH COMPLETE

**Phase:** 20 - 错误处理统一收敛 + sentinel 体系增强
**Confidence:** HIGH

### Key Findings

- **5 项 CONTEXT 差异已识别**（不阻断，但 planner 必须 PLAN 中显式处理）：散点总数 ~70 非 27 / 漏列 2 文件 / 无 Makefile / AD config 500→503 / ADUserNotRegistered 兜底
- **架构基线全部 confirmed**：HandleError@173 + MapToHTTPStatus@32 + IsKnownError@127 + 41 sentinels + BusinessError typed + 10 Code 常量 + error_mapper.go backstop
- **D-03.6 scope ~208 处 `zap.Error(err)`**（handler 98 + service 110），不是小工作量——建议每文件 1 atomic commit 合并 classify + logger 双升级
- **D-04 需新建 Makefile 或 shell 脚本**（项目无 Make 历史），推荐方案 3：go:generate + CI step
- **`TestClassifyAuthLoginError` 重写为 D-05.5 关键路径**——同步处理 AD config 503 + ADUserNotRegistered 兜底语义

### File Created

`D:\CODE\ClaudeCode\record_V2\.planning\phases\20-handleerror-classify-convergence\20-RESEARCH.md`

### Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | internal/errors + pkg/response 全文 Read，行号 + 类型定义 + slice 顺序逐项验证 |
| Architecture | HIGH | HandleError/mapping.go/error_mapper.go 三组件 + Phase 19 baseline 全 confirmed；build/test 绿 |
| Pitfalls | HIGH | 5 项差异全部 runtime grep + Read 双重证据，给出文件:行号 |
| Validation | HIGH | 12 phase-17 包 + 8 个现有 handler test 文件 + 测试命令均 verified |

### Open Questions (escalation candidates)

1. **R-3 ADUserNotRegistered 处理方案** — 推荐"迁到 internal/errors（D-02.2 补漏例外）"，需用户确认或 discuss-phase 补 D-07
2. **R-2 Makefile 决策** — 推荐"方案 3 go:generate + CI step"，需用户确认或 discuss-phase 补 D-08
3. **D-03.1 SentinelField 放 pkg/response** — Claude's Discretion 范围内，推荐放 pkg/response（不新建 pkg/logging）
4. **行号清单是否固定在 PLAN** — 推荐用 grep 模式 + 计数作为验收条件，不锁行号

### Ready for Planning

Research complete. Planner 可基于本文件 + CONTEXT.md 创建 PLAN.md。

**Planner 必读优先级：** §1 (散点修正) → §9 R-1..R-7 (风险) → §3 (HandleError 设计点) → §2.4 (IsKnownError slice) → Validation Architecture (Wave 0 gaps)。
