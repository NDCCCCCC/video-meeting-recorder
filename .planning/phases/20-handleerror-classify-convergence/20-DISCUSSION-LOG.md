# Phase 20: 错误处理统一收敛 + sentinel 体系增强 - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-01
**Phase:** 20-handleerror-classify-convergence
**Areas discussed:** Scope focus, Classify replacement strategy, Logger field shape, Doc generation, Replacement gaps, Error flow direction, Status regression testing, Logger integration location, Match priority, BusinessError logger output, Doc metadata, Sync timing, Generator implementation, Phase 20 scope vs deferred

---

## Scope focus (D-01)

| Option | Description | Selected |
|--------|-------------|----------|
| 先聚焦分类收敛 | 仅 classify 全量替换 + zap logger + 文档生成；typed error kind 留 deferred | ✓ |
| 四项目标全量落地 | 4 项全部纳入，预计 5-6 plans | |
| 只做单点深化 | 4 项里只挑 1 项快进快出 | |

**User's choice:** 先聚焦分类收敛
**Notes:** typed error kind 字段（三层 enum）工作量大 + 需 spike；phase 20 仅落 3 项深化

---

## Classify replacement strategy (D-02.1)

| Option | Description | Selected |
|--------|-------------|----------|
| 一次性全量清理 27 处 | 按 Phase 17 D-01.4 "一次完成全量改" 原则 | ✓ |
| 按文件分批迁移 | 1 file 1 PR | |
| 只清理 formal classify | 只删 classifyAuthLoginError，保留 inline 分支 | |

**User's choice:** 一次性全量清理 27 处（推荐）
**Notes:** 与 Phase 16 D-01.2 / Phase 17 D-01.4 一致；1 mega commit + 单测保证状态码不回归

---

## Logger field shape (D-03.x)

| Option | Description | Selected |
|--------|-------------|----------|
| 单个字符串（最高优先级） | `zap.String("sentinel_type", "ErrNotFound")` | ✓ |
| 字符串数组 | `zap.Strings("sentinel_chain", ["ErrNotFound","ErrVideoFileNotFound"])` | |
| 双字段 | 单字符串 + 全链数组 | |

**User's choice:** 单个字符串（取最高优先级 match）
**Notes:** 与 errors.Is 单 match 一致；日志聚合友好

---

## Doc generation trigger & placement (D-04)

| Option | Description | Selected |
|--------|-------------|----------|
| go:generate + 单文件 docs/errors.md | Makefile check + git diff 验证 | ✓ |
| CI 检查脚本 + 多格式 | godoc + markdown + html | |
| 不自动生成 | 手动维护 godoc | |

**User's choice:** go:generate + 单文件 docs/errors.md（推荐）
**Notes:** 单一 source of truth；CI 强制同步

---

## Replacement gaps (D-02.2)

| Option | Description | Selected |
|--------|-------------|----------|
| 复用现有 + 补漏，不建新 | 不主动加新 sentinel，对应不到走 BusinessError | ✓ |
| 强制复用现有 | 完全不加新；issue 推下 phase | |
| 照需加新 sentinel | IsKnownError 集合同期增长 | |

**User's choice:** 复用现有 + 补漏，不建新（推荐）
**Notes:** Phase 19 D5-D21 已建 30+ sentinels 收底；本阶段冷启动不加新

---

## Error flow direction (D-02.4)

| Option | Description | Selected |
|--------|-------------|----------|
| Service BusinessError + handler 转换 | handler 一律 `if response.HandleError(c, err) { return }` | ✓ |
| 全走 sentinel 包装 | service 用 `Wrap(err,...)` 包装 sentinel | |
| status code 仅由 handler 决定 | 仅初始化时包 IsKnownError | |

**User's choice:** Service 返回 BusinessError + handler 转换（推荐）
**Notes:** 保留 service 细分 context；handler 不再允许内联 switch

---

## Status regression testing (D-02.5)

| Option | Description | Selected |
|--------|-------------|----------|
| 表驱动单测验证 HTTP 状态码不回归 | 每个 handler path 1 表驱动 `_test.go` | ✓ |
| 不增单测，仅 CI | 靠手验 + CI 验证 | |
| 仅 e2e integration test | Phase 17 D-04.6 纪律 | |

**User's choice:** 表驱动单测验证 HTTP 状态码不回归（推荐）
**Notes:** Phase 17 D-04.1 纪律；4 类错误全覆盖（sentinel / sentinel wrap / BusinessError / 未知）

---

## Logger integration location (D-03.1)

| Option | Description | Selected |
|--------|-------------|----------|
| middleware + helper 函数 + logger 注入 | `SentinelField(err)` helper，各 handler/service 自调 | ✓ |
| Wrapper logger | 封装一层 LoggerWrapper | |
| 中层 log hook | middleware/logger pre/post-process | |

**User's choice:** middleware + helper 函数 + logger 注入（推荐）
**Notes:** Phase 19 error_mapper.go 已带开；helper 放 pkg/response 或 pkg/logging 由 call-site 决定

---

## Match priority (D-03.3)

| Option | Description | Selected |
|--------|-------------|----------|
| IsKnownError 第一个命中 | 按 mapping.go slice 顺序 | ✓ |
| 最后一个命中 | 反向遍历 | |
| 最 specific sentinel | ErrTaskNotFound vs ErrNotFound 区分 | |

**User's choice:** 取 IsKnownError 第一个命中（推荐）
**Notes:** slice 顺序常以更具体优先（如 ErrTaskNotFound 在 ErrNotFound 之前）

---

## BusinessError logger output (D-03.4)

| Option | Description | Selected |
|--------|-------------|----------|
| Code + sentinel_type 双输出 | `sentinel_type="BusinessError(code=NOT_FOUND)"` | ✓ |
| 全用 sentinel_type | BusinessError 走 "BusinessError" 单字符串 | |
| 本阶段独立处理 | BusinessError 不走 helper | |

**User's choice:** Code + sentinel_type 双输出（推荐）
**Notes:** 与 sentinel 字符串值区分；unknown → `sentinel_type="ad-hoc"`

---

## Doc metadata (D-04.3)

| Option | Description | Selected |
|--------|-------------|----------|
| 错误名 + path + source + status + count | name \| kind \| HTTP status \| call-site count | ✓ |
| 只列错误 + HTTP状态 | 简版 | |
| 文档 + mapping 表不写 | 仅 MapToHTTPStatus 现有表 | |

**User's choice:** 错误名 + path + source + status + count（推荐）
**Notes:** 机读 + review 友好；call-site count 每次生成重算

---

## Sync timing (D-04.4)

| Option | Description | Selected |
|--------|-------------|----------|
| go:generate + Makefile check | `git diff --quiet docs/errors.md` | ✓ |
| 含 commit hook | pre-commit 强制 generator | |
| CI 静默错 | 走 GitHub Actions | |

**User's choice:** go:generate + Makefile check（推荐）
**Notes:** 已有 Makefile pattern；CI 验证 generator diff 为空

---

## Generator implementation (D-04.2)

| Option | Description | Selected |
|--------|-------------|----------|
| 文本 generator (go script) | 单一 binary cmd/error-doc-gen/main.go | ✓ |
| go AST + reflection | 高复杂度，可重生成 | |
| 从 source grep | 难度低但易变 | |

**User's choice:** 文本 generator（go 脚本，推荐）
**Notes:** 纯文本扫 const 集合；不依赖 AST；可控易改

---

## Phase 20 scope vs Phase 17/19 deferred (D-01.3)

| Option | Description | Selected |
|--------|-------------|----------|
| 不抱，不处理遗留 | 本阶段不碰 Phase 17/19 deferred items | ✓ |
| 顺手抱低难 leftovers | taskServiceAdapter merge 等 | |
| 搭 HMAC jti DB 表架构 spike | phase 19 deferred | |

**User's choice:** 不抱，不处理遗留（推荐）
**Notes:** 仅鉴收 cross-package local error var 重复（survey only，不主动迁）

---

## Claude's Discretion

- D-02.4 service 返回 `BusinessError` vs `%w` wrap 的具体挑选粒度
- D-03.1 helper 放 `pkg/response` vs 新 `pkg/logging/`
- D-04.2 generator 提取 const godoc 注释作为第二列
- D-04.3 Call-site count 粗略 grep vs 精确 grep

## Deferred Ideas

### Phase 20 deferred
- typed error kind 字段（Sentinel vs BusinessError vs ad-hoc marker interface）—— 用户在 D-01 明确排除；本阶段仅输出 `sentinel_type` 字符串形态

### Phase 17 deferred (still untouched)
- STYLE-001 全库 `%w` 迁移 / SEC-003b 华为密码 DB / PERF-003 全库 ctx / STYLE-009 Get* rename
- koanf / audit 包迁移 / golangci-lint+errcheck/gosec

### Phase 19 deferred (still untouched)
- taskServiceAdapter merge / HMAC jti DB 表 / internal/errors 全量 service import

### Survey (commit body only)
- cross-package local error var 调研：本阶段仅查不主动迁；结果入 commit message body
