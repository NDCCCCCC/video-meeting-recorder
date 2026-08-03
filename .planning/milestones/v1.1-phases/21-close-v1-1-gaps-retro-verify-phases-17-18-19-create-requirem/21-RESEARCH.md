# Phase 21: Close v1.1 gaps — retro-verify 17/18/19 + REQUIREMENTS.md + auth:57 fix - Research

**Researched:** 2026-08-03
**Domain:** GSD 工作流过程缺口补救（retrospective verification + requirement traceability + 1-line code fix）
**Confidence:** HIGH（证据源全部本地可验证：git history + 实时代码 + STATE.md + 现有 VERIFICATION.md 模板）

## Summary

Phase 21 是一个**纯过程/文档阶段**（外加 1 行代码修复）。代码库功能已完整（`go test -race ./...` 全绿，phase 20 通过 10/10 goal-backward verification），本阶段只补齐 v1.1 里程碑审计（`v1.1-MILESTONE-AUDIT.md`，`status: gaps_found`）发现的三类**过程缺口**：3 份重建的 VERIFICATION.md（phase 17/18/19）、1 份全新的 REQUIREMENTS.md、1 处 auth_handler.go:57 规范化。

研究核心结论：**全部所需证据已存在于代码库与 git 历史中**，无需新增代码或运行时验证。3 份 retro-verify 的证据链完整且可交叉验证（SUMMARY → commit → live code 三层一致）；REQUIREMENTS.md 有 GSD 官方模板可参照，但需按 D-03.3 扩展为 5 列追溯表；auth:57 fix 的行为等价性结论成立但**CONTEXT D-04.2 给出的理由是错的**（见 §6），planner 必须用正确的论据。

**Primary recommendation:** planner 可以直接基于本研究列出的证据点（每项含 file:line / commit / test name）生成 4 个原子任务（17-VERIFY / 18-dir+VERIFY / 19-dir+VERIFY / REQUIREMENTS）+ 1 个独立代码任务（auth:57）。`gsd-sdk` 无加速器，全部人工组装。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Phase 17 retro-verify | `.planning/phases/17-*/17-VERIFICATION.md` | live code + 17-SUMMARYs + git log | 目录已存在，仅缺 VERIFICATION.md；证据源全部 in-tree |
| Phase 18 retro-verify | `.planning/phases/18-<slug>/` (重建) | 根目录 `18-SUMMARY.md` + commit `e6315ce..0c018f2` + `5d536ec` + STATE.md §Phase 18 | 目录完全缺失，需重建并复制 SUMMARY 副本 |
| Phase 19 retro-verify | `.planning/phases/19-<slug>/` (重建) | 根目录 `19-SUMMARY.md` + `docs/audits/phase-19-D5-D21-summary.md` + 21 个 `refactor(19/dN)` commits + STATE.md §Phase 19 | 同上；额外有 D5-D21 增量总结 |
| REQUIREMENTS.md 追溯表 | `.planning/REQUIREMENTS.md` (新建) | 17-CONTEXT + 18/19-SUMMARY + 20-VERIFICATION + audit §6.2 | 横跨 4 个 phase 的 REQ-ID 集中登记 |
| auth:57 规范化 | `internal/handlers/auth_handler.go` | `pkg/response/response.go` + `internal/handlers/auth_handler_test.go` | 1 行代码改动，回归网已存在 |
| Nyquist VALIDATION.md（**OUT OF SCOPE**） | — | — | CONTEXT `<deferred>` 显式排除 |

## User Constraints (from CONTEXT.md)

### Locked Decisions

**D-01 范围聚焦（3 项 + 硬边界）**
- D-01.1: 仅交付 3 项 — retro-verify 17/18/19 的 VERIFICATION.md + `.planning/REQUIREMENTS.md` + `auth_handler.go:57` 1 行修复
- D-01.2: 改动面 = `.planning/`（新增 + 重建目录）+ `internal/handlers/auth_handler.go`（1 行）；**不动** `frontend/`、`docs/audits/*.md`（不可变 source of truth）、业务 service 代码、DB schema
- D-01.3: 代码侧只允许 `auth_handler.go:57` 一处变更；任何顺手清理（bare zap.Error / ppt_handler / err.Error() 泄漏）必须拒绝

**D-02 Retro-verify 方法论（goal-backward，证据驱动）**
- D-02.1: gsd-verifier goal-backward 方法；每个 VERIFICATION.md 必须含 `must_haves` + 证据 + `status: passed|partial` + 诚实的数据局限说明
- D-02.2: 证据层级（强→弱）：① 实时代码 grep / `go build` / `go test -race` → ② git commit history → ③ SUMMARY.md / STATE.md → ④ 推断（必须标 "inferred"）；禁止把推断当事实
- D-02.3: Phase 17 直接基于 17-CONTEXT must_haves + 4 SUMMARY + REVIEWS 对实时代码验证；重点核验 deferred 项确实未做
- D-02.4: Phase 18 重建目录，复制（非移动）根目录 `18-SUMMARY.md`，新写 `18-VERIFICATION.md`；证据 = commit `5d536ec` + STATE.md §Phase 18 + 实时代码
- D-02.5: Phase 19 重建目录，复制根目录 `19-SUMMARY.md` + `docs/audits/phase-19-D5-D21-summary.md`，新写 `19-VERIFICATION.md`；证据 = 21+ `refactor(19/dN)` commits + STATE.md §Phase 19 + 实时代码
- D-02.6: 根目录 `18-SUMMARY.md` / `19-SUMMARY.md` 保持原位不动；`.planning/phases/` 下重建目录并复制副本（`.planning/` 在 `.gitignore`，需 `git add -f`）
- D-02.7: 诚实标注局限 — phase 18/19 原 PLAN.md / CONTEXT.md / DISCUSSION-LOG 永久丢失，VERIFICATION.md 必须在 "Evidence Limitations" 段说明

**D-03 REQUIREMENTS.md 内容与范围**
- D-03.1: 范围 = v1.1 milestone（phase 17/18/19/20）
- D-03.2: REQ-ID 体系覆盖：REQ-17-*（56 findings）/ REQ-18-*（SM4-GCM）/ REQ-19-*（ctx + SEC-004 + STYLE-001）/ REQ-20-*（沿用 REQ-20a/b/c）
- D-03.3: 表结构至少含 `REQ-ID | Phase | 来源 | 状态 (done/deferred/partial) | 验证证据`；须支持 orphan 检测与 3-source 交叉引用
- D-03.4: phase 16 归属歧义 — 加 `## Out-of-scope observation` 记录，**不强行裁定**
- D-03.5: 不伪造追溯 — SUMMARY frontmatter `requirements_completed` 为空的按实际交付物回填并标 "backfilled from deliverables, SUMMARY frontmatter was empty"

**D-04 auth_handler.go:57 修复（1 行，行为不变）**
- D-04.1: `if response.HandleError(c, err) { return }; return` → `response.HandleError(c, err); return`
- D-04.2: 行为等价性必须保持 — **CONTEXT 此处理由有误，见本 RESEARCH §6 的更正**
- D-04.3: 删除 line 60 「兜底」注释；保留 line 53-56 解释 mapping.go 行为的注释
- D-04.4: 不动 auth_handler.go 其他 3 处 tech_debt（`:93 RefreshToken` / `:182 ChangePassword` / `LogoutAll`）

**D-05 验收标准（goal-backward）**
- D-05.1: 3 个 VERIFICATION.md 各含 `status: passed`（或 `partial` + 理由），must_haves 全部有证据指向
- D-05.2: `.planning/REQUIREMENTS.md` 存在、v1.1 四个 phase 全覆盖、无 REQ-ID orphan（除非显式标 deferred）
- D-05.3: `auth_handler.go:57` 为规范模式；`go build ./...` + `go test -race ./internal/handlers/...` 全绿；既有 auth 测试不回归
- D-05.4: 审计的 5 项 gap 全部可标记关闭（重跑 `/gsd:audit-milestone v1.1` 由 `gaps_found` → `passed`）

**D-06 提交策略**
- D-06.1: 提交分组（5 个 commit，`commit_docs: true`，`.planning/` 需 `git add -f`）：
  - `docs(21): retro-verify phase 17 — reconstruct VERIFICATION.md`
  - `docs(21): retro-verify phase 18 — reconstruct dir + VERIFICATION.md`
  - `docs(21): retro-verify phase 19 — reconstruct dir + VERIFICATION.md`
  - `docs(21): create REQUIREMENTS.md — v1.1 REQ-ID traceability`
  - `fix(handlers/SEC-008): auth_handler.go:57 canonical HandleError pattern`（代码改动单独提交）
- D-06.2: 代码提交（auth:57）必须 `go build ./...` + `go test -race ./internal/handlers/...` 通过
- D-06.3: 不修改 `docs/audits/*.md`（不可变）；REQUIREMENTS.md 是新建文件

### Claude's Discretion
- D-02.4/D-02.5 重建目录的具体 slug 命名（遵循 `.planning/phases/NN-<slug>/` 既有约定）
- D-03.2 REQ-ID 的确切前缀格式（如 `REQ-17-SEC-001` vs `REQ-17.1`）
- D-02 每个 VERIFICATION.md 的 must_haves 具体条目措辞
- 是否把根目录 `18-SUMMARY.md` / `19-SUMMARY.md` 在 REQUIREMENTS.md 里登记为 canonical ref

### Deferred Ideas (OUT OF SCOPE)
- **Nyquist VALIDATION.md 补齐**（phase 16/17 缺、phase 20 draft）— 独立 phase
- **Phase 16 归属裁定** — 留待 milestone 决策，REQUIREMENTS.md 仅记观察
- **审计 tech_debt 10+ 项**（bare zap.Error 31 处、ppt_handler 9 处、auth_handler 另 3 处、STYLE-001 全库、PERF-003 全库、STYLE-009 包名、typed error kind 字段等）— 各为后续独立 phase
- **重跑 `/gsd:audit-milestone v1.1`** — phase 21 交付后的验收动作
- **v1.0（phase 01-14）REQ-ID 追溯** — v1.0 已 shipped 归档

### 数据局限（非 deferred，是约束）
- Phase 18/19 原 PLAN.md / CONTEXT.md / DISCUSSION-LOG 永久丢失（从未 git 入库）

## Phase Requirements

Phase 21 无 inbound REQ-ID（phase_req_ids 为 null）。这是 by design：本阶段的**交付物之一**就是创建 `.planning/REQUIREMENTS.md` 定义 v1.1 REQ-ID 体系。下表是本阶段从 CONTEXT.md 派生的内部需求映射（不是入站 REQ-ID）。

| ID (派生) | Description | Research Support |
|-----------|-------------|------------------|
| P21-R1 | 重建 phase 17 VERIFICATION.md（goal-backward，passed） | §2 + §9（证据点已枚举） |
| P21-R2 | 重建 phase 18 目录 + VERIFICATION.md | §3（commit + STATE + live code 三源一致） |
| P21-R3 | 重建 phase 19 目录 + VERIFICATION.md | §4（21 个 dN commits + D5-D21 summary） |
| P21-R4 | 创建 `.planning/REQUIREMENTS.md`（v1.1 四 phase 全覆盖） | §5（GSD 模板 + D-03.3 扩展表结构） |
| P21-R5 | 修复 `auth_handler.go:57`（1 行，行为等价） | §6（含 CONTEXT D-04.2 理由更正） |
| P21-R6 | 不动业务代码 / frontend / docs/audits / DB schema | CONTEXT D-01.2/D-01.3 |

---

## Focus Area 1: VERIFICATION.md Format Contract

### Source of Truth (Gold References)

两份权威 source 决定 retro-verify 文档的格式约定：

1. **`$HOME/.claude/get-shit-done/templates/verification-report.md`** [VERIFIED: GSD template] — GSD 标准 VERIFICATION.md 模板
2. **`.planning/phases/20-handleerror-classify-convergence/20-VERIFICATION.md`** [VERIFIED: project-local] — 本项目唯一一份通过的 VERIFICATION.md（10/10 passed），是事实模板

### Required Frontmatter

```yaml
---
phase: <NN-name>           # e.g. 17-56-p0-p1-p2
verified: <ISO timestamp>  # 2026-08-03T...Z（retro-verify 标注 "retro-active"）
status: passed | gaps_found | human_needed
score: N/M must-haves verified
---
```

20-VERIFICATION.md 额外用了这些可选字段（planner 可沿用）：
`overrides_applied`, `overrides`, `gaps`, `gap_closure`, `deferred`, `human_verification`。**对 retro-verify 场景，`deferred` 与 `gap_closure` 字段尤其有用**——分别记录"原 phase deferred 项"与"补齐过程产物的 gap"。

### Status 值语义（GSD 标准）

| 值 | 含义 | 适用本阶段？ |
|----|------|--------------|
| `passed` | 所有 must-haves 验证通过，无 blocker | ✅ Phase 17/18/19 目标状态 |
| `gaps_found` | 发现 ≥1 critical gap | ❌ 不期望（除非触发 partial 重做） |
| `human_needed` | 自动检查通过但需人工验证 | ❌ |

### Required Sections (mirror 20-VERIFICATION.md structure)

20-VERIFICATION.md 实际产出的章节结构（planner 应镜像）：

1. `# Phase NN: <Name> - Verification Report`
2. `**Phase Goal:**` — 从 ROADMAP `**Goal:**` 拷贝原文
3. `**Verified:** <ts>` / `**Status:** <status>` / `**Verifier:** Claude (gsd-verifier) — retro-active, goal-backward methodology`
4. `## Goal Achievement`
   - `### Observable Truths` — 表格：`| # | Truth | Status | Evidence |`（Status = VERIFIED / FAILED / UNCERTAIN）
   - `### Required Artifacts` — 表格：`| Artifact | Expected | Status | Details |`（EXISTS + SUBSTANTIVE 模式）
   - `### Key Link Verification` — 表格：`| From | To | Via | Status | Details |`（wiring 验证）
   - `### Data-Flow Trace (Level 4)` — 可选，20-VERIFICATION 用过
   - `### Behavioral Spot-Checks` — 表格：`| Behavior | Command | Result | Status |`（命令必须 < 30s 可运行）
5. `## Requirements Coverage` — 表格：`| Requirement | Source Plan | Description | Status | Evidence |`
6. `## Anti-Patterns Found` — 表格：`| File | Line(s) | Pattern | Severity | Impact |`
7. `## Human Verification Required`（若无可写 "None required — all behaviors programmatically verified"）
8. `## Gaps Summary`
9. **retro-verify 必加**：`## Evidence Limitations` — 诚实说明"基于 SUMMARY + commit + 代码，非基于原始 PLAN/CONTEXT"
10. **retro-verify 必加**：`## Deferred (confirmed not done, by design)` — 列出原 phase 的 deferred 项

### gsd-verifier 三层证据模型 (from `references/few-shot-examples/verifier.md`)

[VERIFIED: GSD reference] verifier 对每个 must-have artifact 独立执行三层检查：

| Level | 检查内容 | 示例 |
|-------|----------|------|
| L1 (Existence) | 文件存在、行数 ≥ 阈值 | "internal/utils/sm4_password.go exists, 400+ lines" |
| L2 (Substantive) | 无 TODO/FIXME stub、有真实导出符号 | "exports EncryptGCM/DecryptGCM/ParseCredentialEnvelope" |
| L3 (Wired) | 被消费者实际 import/调用 | "credential_encryptor.go calls ParseCredentialEnvelope at line ..." |

校准语料统计：**80% pass rate，gap 分布 = 37% missing wiring + 25% missing tests + 38% other**。planner 写 retro-verify 时必须为每条 must-have 给到 L1+L2+L3 三层证据，不能只给"文件存在"。

### 关键诚实规则（来自 verifier few-shot）

- ❌ 禁止 "blanket pass with no per-criterion evidence"
- ❌ 禁止只查 L1 existence 不查 L2/L3
- ✅ 区分 planning gap（计划遗漏）vs execution failure（执行失败）— retro-verify 多数会是 "execution delivered, planning doc lost"
- ✅ PASS_WITH_NOTES 允许（执行正确但发现未计划的 follow-up）

## Focus Area 2: Phase 17 Retro-Verify Readiness

### 目录现状（齐全）

[VERIFIED: filesystem]
```
.planning/phases/17-56-p0-p1-p2/
├── 17-CONTEXT.md          (15.7 KB — 含 D-01.4/D-03/D-04 must_haves + <deferred>)
├── 17-DISCUSSION-LOG.md   (4.1 KB)
├── 17-01-PLAN.md          (43.5 KB) ─┐
├── 17-01-SUMMARY.md       (12.7 KB) │
├── 17-02-PLAN.md          (32.9 KB) │
├── 17-02-SUMMARY.md       (15.8 KB) │ 4 PLAN + 4 SUMMARY 齐全
├── 17-03-PLAN.md          (20.4 KB) │
├── 17-03-SUMMARY.md       (18.6 KB) │
├── 17-04-PLAN.md          (31.3 KB) │
├── 17-04-SUMMARY.md       (16.7 KB) ─┘
├── 17-REVIEWS.md          (28.1 KB — opencode 单一外部 reviewer，post-execution)
└── .gitkeep
```

**仅缺 VERIFICATION.md。** 无需重建目录，直接新写一份即可。

### Must-Haves 来源（17-CONTEXT.md）

[VERIFIED: 17-CONTEXT.md, lines 33-85]

ROADMAP Phase 17 Goal：[VERIFIED: `.planning/ROADMAP.md` line 128]
> 后端代码库通过 56 项审查发现的分级修复（P0 HIGH + P1 MEDIUM + P2 LOW），配齐 P0/P1 单测、同步部署文档，go build/vet/fmt 全绿、既有测试不回归。

派生 must-haves（来自 D-01.4 全量 56 项 + D-03 破坏性变更 + D-04 测试纪律 + D-05 文档同步）：

| # | Must-Have | Source | Live Code Evidence |
|---|-----------|--------|--------------------|
| M1 | 13 HIGH 全修（SEC-001/002/003a/004 + BUG-001/002 + PERF-001..005） | 17-01-SUMMARY | `internal/config/config.go` ValidateProductionSecrets / `cmd/server/app.go` SetAuditService wiring / `internal/huawei/manager.go` SetTLSPolicy / `internal/auth/hlstoken/hls_token.go` jti + RawURLEncoding |
| M2 | 18 MEDIUM 全修（BUG-003..006 + SEC-005..010 + PERF-006..011 + STYLE-003/004/005） | 17-02/03-SUMMARY | 12 个 commit hash（d27903f..0190f83）+ grep 验证（见 SUMMARY 表） |
| M3 | 25 LOW 全修（BUG-011/015/016 + SEC-011..015 + PERF-012..016 + STYLE-001 partial + STYLE-006/007/008/010） | 17-04-SUMMARY | 17 atomic commits（4f5579a..72e2027） |
| M4 | P0/P1 单测配齐（D-04.1/D-04.2） | 17-01/02-SUMMARY 测试表 | `internal/config/config_test.go` / `internal/huawei/manager_test.go` / `internal/auth/hlstoken/hls_token_test.go` / 等 |
| M5 | 部署文档同步（D-05：DEPLOYMENT.md/BUILD.md/SECURITY.md/.env.example） | 17-01-SUMMARY §"部署文档同步" | grep "ValidateProductionSecrets\|环境变量与启动校验" 应有命中 |
| M6 | `go build ./...` + `go vet ./...` + touched files `gofmt -l` 全绿 | 全部 SUMMARY | `go build ./...`（< 30s）|
| M7 | 既有测试不回归 | 全部 SUMMARY | `go test -race ./...` |

### Confirmed `<deferred>` Items（必须验证确实未做）

[VERIFIED: 17-CONTEXT.md lines 222-234 + 17-01-SUMMARY §D5 + 17-04-SUMMARY §"修复的 Finding"]

| Item | Claimed Deferred | Live-Code 验证 |
|------|------------------|----------------|
| **SEC-003b**（华为密码 DB 加密） | 17-CONTEXT W6 deferred；17-01-SUMMARY 标 DEFERRED | 在 phase 17 范围内确实未做；**实际由 phase 18 落地**（见 §3） |
| **STYLE-001 全库 %w 迁移** | 17-CONTEXT deferred（168 errors.New + 474 fmt.Errorf） | 17-04 仅 partial 移 3 处；全库仍 ~642 处未迁移；**部分由 phase 19/20 收敛** |
| **PERF-003 全库 ctx 级联** | 17-CONTEXT deferred（403 处 GORM） | 17-01 D1 明确未做；**实际由 phase 19 落地**（见 §4） |
| **STYLE-009** 包名冗余 Get* rename（133 处） | 17-CONTEXT Claude's Discretion 默认跳过 | 实际未做（不影响 API） |
| **HMAC jti `used_jtis` 表** | 17-CONTEXT deferred（架构决策） | 17-01 用进程内 map；**phase 19 D3 升级为 `hls_jti_records` 表** |
| koanf / audit 包迁移 / golangci-lint | 17-CONTEXT deferred | 实际未做 |

**验证 deferred 项"确实未做"的 grep 命令**（planner 写 VERIFICATION 时用）：

```bash
# STYLE-001 全库迁移未做（应仍见大量 errors.New）
grep -rc "errors\.New" internal/ --include="*.go" | awk -F: '{s+=$2} END {print s}'
# 期望：数百处（不是 0）

# PERF-003 全库 ctx（应仍见 ctx-less GORM 调用）
grep -rn "s\.db\." internal/ --include="*.go" -A 0 | grep -v "WithContext" | wc -l
# 期望：非零（虽然 phase 19 收敛了 high-frequency 路径）

# STYLE-009 Get* rename（应仍见 133 处包名冗余）
grep -rcE "func \(.*\) Get[A-Z]" internal/ --include="*.go" | awk -F: '{s+=$2} END {print s}'
# 期望：~100+
```

### 17-REVIEWS.md 关键信息（post-execution，opencode 单一 reviewer）

[VERIFIED: 17-REVIEWS.md lines 1-80]

- **reviewers**: `[opencode]`（v1.14.20 via GitHub Copilot）— 唯一外部 reviewer
- **risk_assessment**: LOW
- **execution_base**: `cf2d248` (planning) → `3b10afa` (phase 17 complete HEAD)
- **45 commits**（41 code/test + 3 docs/state + 1 housekeeping）
- **13 deviations** 全部已接受（典型：PERF-003 因方法签名级联而 deferred、SEC-004 fallback 顺序为 RawURLEncoding→URLEncoding→StdEncoding、`MinTLSVersion` 实际是 string 不是 uint16、PERF-005 HuaweiConfigRow 类型作用域）
- **reviewer 评价**：计划质量"exceptionally thorough"，但指出"systematic defects — assuming field types/signatures without code verification"（与 deferred 项相互印证）

### Phase 17 retro-verify 关键 commit hash（按 wave）

| Wave | Commit | Finding IDs |
|------|--------|-------------|
| 17-01 P0 | `4d3de0b` / `2bcee29` / `47ef805` / `4fc1d3c` | SEC-001/002/003a/004 + BUG-001/002 + PERF-001..005 |
| 17-02 P1a | `d27903f`..`b53cc8c` | BUG-003..006 + SEC-005..010 + STYLE-004/005 |
| 17-03 P1b | `9150e95`..`0190f83` | PERF-006..011 + STYLE-003 |
| 17-04 P2 | `4f5579a`..`72e2027` | BUG-011/015/016 + SEC-011..015 + PERF-012..016 + STYLE-001 partial/006/007/008/010 |
| 最终 HEAD | `c04f805` | housekeeping + gofmt clean |

**Goal-backward 验证建议 status: `passed`** —— 全部 56 findings 都有 commit + 单测覆盖（M1-M4），deferred 项已明确登记（5 项），无 functional gap。

## Focus Area 3: Phase 18 Retro-Verify Evidence Pack

### 目录现状：缺失，需重建

[VERIFIED: filesystem] `.planning/phases/18-*/` 不存在（ROADMAP 表显示 1/1 Complete 2026-07-31，但目录被清理/从未建）。

证据源（按 D-02.4 强弱排序）：
1. **实时代码**：`internal/services/credential_encryptor.go` / `internal/utils/sm4_password.go` / `internal/models/hls_jti_record.go` / `cmd/server/app_test.go` / `cmd/server/phase18_integration_test.go`
2. **git commits**：`e6315ce` (W1a) / `1dbb3b0` (W1b) / `edaa4ae` (W1c) / `558f723` (W2) / `bd84fe2` (W3) / `8796ca3` (W4a) / `3822497` (W4b) / `a182cd6` (W4c) / `0c018f2` (W4d) + 后续 `5d536ec` (SEC-003b invariant)
3. **SUMMARY**：根目录 `18-SUMMARY.md`（21.7 KB）
4. **STATE.md**：§Phase 18（line 52-78）+ Phase 18 Base HEAD 记录

### Deliverables 与证据映射

| # | Deliverable | Live Code Evidence | Git Evidence | SUMMARY Ref |
|---|-------------|--------------------|--------------|-------------|
| D18-1 | **SM4-GCM envelope 格式** `SM4:<version>:<base64(nonce_12B\|ciphertext\|tag_16B)>` | `internal/utils/sm4_password.go:19` 注释 + `EncryptGCM`/`DecodeCredentialEnvelope`/`ParseCredentialEnvelope` 导出 [VERIFIED: grep] | `e6315ce` | 18-SUMMARY §"算法" |
| D18-2 | **PKCS#7 padding**（gmsm v1.4.1 GCMDecrypt 块对齐要求） | `internal/utils/sm4_password.go` PKCS7 实现注释 | `e6315ce` | 18-SUMMARY §"核心决策" + §"与计划差异" #2 |
| D18-3 | **密钥族隔离**（`CREDENTIAL_SM4_*` vs `SM4_SECRET` vs `HLS_TOKEN_SECRET`） | `internal/config/config.go` AuthConfig.CredentialSM4* 字段 + `ValidateCredentialSM4Config` [VERIFIED: 17/18-SUMMARY] | `1dbb3b0` | 18-SUMMARY §"密钥族分离" 表 |
| D18-4 | **CredentialEncryptor service**（encrypt/decrypt/rotate/invariant scan） | `internal/services/credential_encryptor.go` 存在（`ls` 确认） | `edaa4ae` | 18-SUMMARY §"改动文件清单" |
| D18-5 | **encrypt-on-write / decrypt-on-read 接入业务层**（input_config_service / config_service） | `internal/services/input_config_service.go` 注入 encryptor + `internal/services/config_service.go` 删除 base64-stub | `558f723` | 18-SUMMARY §"核心决策" #死代码删除 |
| D18-6 | **fail-closed 启动期 10 步扫描** | `cmd/server/app.go` `Initialize()` 内 `MigratePlaintextToGCM` + 双 `InvariantScan` + `RotateIfNeeded` [VERIFIED: SUMMARY + STATE.md] | `bd84fe2` | 18-SUMMARY §"启动期 fail-closed (10 步)" |
| D18-7 | **生产解密不变式测试**（huaweiDBAdapter 必须解密 SM4 信封） | `cmd/server/app_test.go:53` `TestHuaweiDBAdapter_ProductionDecrypts` [VERIFIED: grep + 5d536ec diff] | `5d536ec` | — (post-audit 增补) |
| D18-8 | **operator 轮换手册** + 物理残留章节 | `DEPLOYMENT.md` 「凭据密钥配置与轮换」+「凭据存储的物理残留」两章节 | `0c018f2` | 18-SUMMARY §Wave 4 §W4d |
| D18-9 | **重复轮换测试**（v1→v2→v3） | `internal/services/credential_encryptor_test.go::TestRepeatedRotation_V1ToV2ToV3` | `3822497` | 18-SUMMARY §W4b |
| D18-10 | **per-site/version 计数日志**（`after_migrate`/`after_rotate`/`after_invariant` 三阶段） | `internal/services/credential_encryptor.go::CountByVersion`/`LogVersionCounts` + `cmd/server/app.go::Initialize()` 三处调用 | `8796ca3` | 18-SUMMARY §W4a |
| D18-11 | **SEC-003b marker comment 更新**（deferred→done） | `internal/huawei/manager.go:132-134` 注释 "SEC-003b/Phase 21: ... 此层视为明文边界" [VERIFIED: grep + 5d536ec diff] | `5d536ec` | — (commit message 显式说明) |
| D18-12 | **`hls_jti_records` 表**（jti 持久化）— 跨 phase 19 D3 协作 | `internal/models/hls_jti_record.go` + `cmd/server/app.go:340` AutoMigrate 注册 [VERIFIED: grep] | — | 18-SUMMARY 未覆盖（属 phase 19 D3） |

### Phase 18 Base/Final HEAD（STATE.md §Phase 18）

[VERIFIED: STATE.md lines 64-72]

- 规划基线：`e294ae9` (Phase 17 cross-AI review)
- After W3 docs: `3b1cb79`
- After W1+2+3 state: `8c69e33`
- After W4: `7e9baaf`
- 最终 HEAD（18-SUMMARY 标注）：`0c018f2`（W4d docs）

**注**：18-SUMMARY frontmatter 标 `Final HEAD: bd84fe2`（仅 W3），但 §Wave 4 实际延伸到 `0c018f2`（§"Wave 4" line 175 显式说明 "W1a..W4d 共 9 个原子 commit"）。planner 在 VERIFICATION 里应取 `0c018f2` 为 phase 18 真正最终 HEAD，并标注"SUMMARY frontmatter 与 body 不一致，body 为准（inferred from body detail）"。

### 关键技术细节（来自 18-SUMMARY §"与计划差异"）

1. **gmsm v1.4.1 nonce mutation**：GCM 内部 `IV = append(IV, 0001)` 修改 nonce slice backing array → 实现层做 defensive copy
2. **PKCS#7 padding 必需**：gmsm v1.4.1 GCMDecrypt 仅支持块对齐密文
3. **No `migrations/017_*` file**：列宽扩展由 `cmd/server/app.go:widenPasswordColumns()` 在 Initialize() 内直接执行（dormant registry 不触发）
4. **每轮旋转 5 envelopes**（非 4）：Live3.stream_password 首次 Migrate 时就是 v1，也被覆盖
5. **GORM `.Scan(&rows)` 不支持** → `db.Raw(...).Rows()` + manual scan workaround（在 `CountByVersion` 中）

### SEC-003b 后续 invariant commit（关键时间线）

[VERIFIED: `git show 5d536ec`]

```
commit 5d536ecb14e4e2b9530c020d5e1eb2a4fba69b4d
Date:   Sun Aug 2 23:06:30 2026 +0800   ← 注意：审计 (2026-08-01) 之后
    fix(huawei/SEC-003b): 标注密码 DB 加密已落地 + 生产解密不变式测试
```

此 commit 是 **phase 21 的预存证据**（21-CONTEXT D-02.4 显式引用）：
- `internal/huawei/manager.go`：注释从 "SEC-003b deferred" 改为 "SEC-003b/Phase 21: done"
- `cmd/server/app_test.go`：新增 `TestHuaweiDBAdapter_ProductionDecrypts`（61 行）

**planner 在 18-VERIFICATION 里应明确**：此 commit 在 phase 21 启动前已落地，作为 retro-verify 证据使用，**不是 phase 21 的交付物**（phase 21 不改业务代码）。

### Phase 18 retro-verify 验证建议

**Goal-backward 起点**（ROADMAP Progress table）：[VERIFIED: ROADMAP.md line 18]
> 18. 凭据静态加密 + 密钥轮换 (SEC-003b) — 1/1 Complete 2026-07-31

**关键 must-haves**：见上表 D18-1..D18-11。**建议 status: `passed`**（功能交付 + 61 个测试覆盖 + 文档同步）。

**Evidence Limitations 段（必加）**：
- 原 PLAN.md / CONTEXT.md / DISCUSSION-LOG 永久丢失
- SUMMARY frontmatter `Final HEAD` 与 body 不一致（bd84fe2 vs 0c018f2）
- 真实生产数据 post-audit 未做（18-SUMMARY §"不在本 phase 范围" 列出）

### Phase 18 与 Phase 17/19 的契约（integration）

- **上游（Phase 17）**：SEC-003b 在 phase 17 标 deferred，phase 18 兑现
- **下游（Phase 19/20）**：`huaweiDBAdapter` 解密路径不变；`TestHuaweiDBAdapter_ProductionDecrypts` 是后续阶段的回归网

## Focus Area 4: Phase 19 Retro-Verify Evidence Pack

### 目录现状：缺失，需重建

[VERIFIED: filesystem] `.planning/phases/19-*/` 不存在（ROADMAP 表显示 4/4 Complete 2026-07-31）。

证据源（按 D-02.5 强弱排序）：
1. **实时代码**：`internal/errors/errors.go` + `mapping.go` + `internal/middleware/error_mapper.go` + `pkg/response/response.go::HandleError` + `internal/services/*_service.go` ctx 透传
2. **git commits**：21 个 `refactor(19/dN)` commits（D1-D21）+ Wave 0-6 commits
3. **SUMMARY**：根目录 `19-SUMMARY.md`（33 KB）+ `docs/audits/phase-19-D5-D21-summary.md`（8.5 KB）
4. **STATE.md**：§Phase 19（line 145-206）+ §Phase 19 Final Status（line 513-533）

### Commit Range（STATE.md 权威记录）

[VERIFIED: STATE.md line 203-207]

- Phase 19 进入基线：`2281927` (W4 docs commit 前)
- 最终 HEAD：`6edb772` (Phase 19 docs 总结)
- `git log --oneline 2281927..6edb772 | wc -l` = **8 commits**（仅 W4 docs 后的范围）

但 phase 19 全量还包括 Wave 0/1/2/3 + D1-D4（更早的 commits），共 11 个 wave commits + 21 个 D1-D21 commits ≈ 32+ commits 总计。

### Wave Commit 序列（19-SUMMARY line 397-408 + 511-519）

| Wave | Commit | 范围 |
|------|--------|------|
| W0 | `ad7d0a8` | ctx 残留清理 (PERF-003/BUG-005) — dashboard_service 11 处 / audit_log_service lifecycle ctx / sm4_token 5 处 |
| W1 | `6fbdad4` | SEC-004 jti replay 模型重写（多分片 HLS 修复） |
| W2 | `cacc294` | STYLE-001 error-mapping 基础设施三组件 |
| W3 | `213710c`..`a6c21b6` (8 commits) | ctx 级联 13 leaf/mid 服务 |
| W4 | `9a00cbe` + `2281927` | TaskServiceInterface 原子三元组（interface + adapter + mock + scheduler 10 调用点） |
| W5 | `34b07f7` (VideoRecordingTaskService 22 方法 ctx-first) + `e2b0b6b`/`7828fc3`/`7a5a1cc`/`1ae6be0`/`b08255d` (VideoFileService + caller) | ctx 全量级联 |
| W6 | `3d171de` | STYLE-001 error 迁移（gorm wrap → BusinessError + HandleError） |
| 收尾 D1 | `20ee289` | FOREIGN KEY strings.Contains → sentinel |
| 收尾 D2 | `3b2d41f` | taskServiceAdapter 与 VideoRecordingTaskService 合并 |
| 收尾 D3 | `1f0ec35` | HLS jti 升级为 `hls_jti_records` 表 |
| 收尾 D4 | `f4291f5` | errors 包增量迁移 |
| 最终 docs | `6edb772` | Wave 6 summary + 范围对账 |

### D5-D21 增量（21 个 commits，`docs/audits/phase-19-D5-D21-summary.md`）

[VERIFIED: phase-19-D5-D21-summary.md]

| Commit | dN | 范围 | Sentinel 增量 | 散点迁移 |
|--------|----|----|---------------|---------|
| `7a0a7af` | d5 | user_service + handler | +5 | 14 |
| `da8aaf9` | d6 | ad_auth + local_auth Login | +4 | 33 |
| `964fb8f` | d7 | sm4_token + middleware | +4 | 11 |
| `c8ed97f` | d8 | ip_validator | 0 | 9 |
| `2a807a6` | d9 | role_service + handler | +4 | 9 |
| `d277f37` | d10 | apikey_service + handler | +5 | 11 |
| `7b0d817` | d11 | hls_token | 0 | 6 |
| `00df988` | d12 | auth/service.go | 0 | 4 |
| `e1dd2dd` | d13 | ppt_file_service + handler | +1 | 1 |
| `1fa66d8` | d14 | tingwu_client | +1 | 23 |
| `98e14ca` | d15 | storage/file_service | 0 | 22 |
| `8eec84a` | d16 | scheduler + recorder | 0 | 33 |
| `cc49867` | d17 | huawei SDK 适配 | 0 | 25 |
| `71e5be0` | d18 | utils/sm4_password | 0 | 28 |
| `ddf047c` | d19 | migrations + models + input_config | 0 | 47 |
| `ffdc0c6` | d20 | transcription/oss/notification/local_driver/config | 0 | 53 |
| `f358602` | d21 | video_recording_task_service | 0 | 24 |

**合计**：17 D5-D21 commits + 4 D1-D4 收尾 = **21 `refactor(19/dN)` commits**，新增 **24 sentinels**，迁移 **~356 散点**，行变化 +461 / -270。

### Deliverables 与 Live-Code 证据映射

| # | Deliverable | Live Code Evidence (grep target) | Git Evidence |
|---|-------------|----------------------------------|--------------|
| D19-1 | **ctx 全量级联 ~190 service 方法** | `grep -c ".WithContext(ctx)" internal/services/video_recording_task_service.go` = **42** [VERIFIED: live grep] | W3-W5 commits |
| D19-2 | **SEC-004 jti 防重放（多分片 HLS 修复）** — 不加 DB 表 + TTL sweeper | `internal/auth/hlstoken/hls_token.go::TestVerify_MultiSegmentSameToken` | `6fbdad4` |
| D19-3 | **STYLE-001 三组件（mapping.go + HandleError + error_mapper.go）** | `internal/errors/mapping.go` + `pkg/response/response.go::HandleError` + `internal/middleware/error_mapper.go` [VERIFIED: Read] | `cacc294` |
| D19-4 | **24 sentinels**（D5-D21 新增，见 `knownSentinels` slice） | `internal/errors/mapping.go:157-203` `knownSentinels` slice 共 **42 条**（19 pre-existing + 23 new，去重后 42 行 — 20-VERIFICATION 标 42 sentinels） | d5-d21 |
| D19-5 | **~356 散点 sentinel 化**（13 service + 9 handler + middleware + utility） | grep `apperrors.Err\|response.HandleError` across services/handlers | d5-d21 |
| D19-6 | **HMAC jti 升级为 `hls_jti_records` 表**（跨实例/重启持久化） | `internal/models/hls_jti_record.go::HLSJtiRecord` + `cmd/server/app.go:340` `&models.HLSJtiRecord{}` AutoMigrate 注册 + `internal/auth/hlstoken/hls_token.go:55,92,95` 注释 [VERIFIED: live grep] | `1f0ec35` |
| D19-7 | **taskServiceAdapter 与 VideoRecordingTaskService 合并** | `cmd/server/app.go` 无独立 adapter struct（grep `taskServiceAdapter` 应减少）；`NewVideoRecordingTaskService` 接受 `*CredentialEncryptor` 可变参数 | `3b2d41f` |
| D19-8 | **gorm wrap → BusinessError + HandleError**（handler string-match → sentinel-driven） | `internal/services/notification/notification_service.go` + `ppt_file_service.go::RenamePPTFile` + `video_file_service.go::RenameVideoFile` + `internal/handlers/ppt_handler.go::RenamePPTFile` + `video_file_handler.go::RenameVideoFile` [VERIFIED: SUMMARY] | `3d171de` (W6) |
| D19-9 | **取消传播测试**（pre-cancelled ctx → GORM `context.Canceled`） | `internal/services/ctx_cancellation_test.go` (4 测试) + `cmd/server/taskservice_adapter_ctx_test.go::TestTaskServiceAdapter_CancellationPropagation` | W3 + W4 |
| D19-10 | **foreignKey sentinel + dual-%w wrap**（Go 1.20+ 双错误链） | `internal/errors/errors.go::ErrForeignKeyConstraint` + `internal/services/video_file_service.go::createWithDuplicateCheck` | `20ee289` (D1) |
| D19-11 | **Run-time invariant scan**（fail-closed）— 跨 phase 18 共享模式 | — | — (D18-6 协同) |

### Phase 19 retro-verify 验证建议

**Goal-backward 起点**（ROADMAP Progress table）：[VERIFIED: ROADMAP.md line 19]
> 19. ctx 全量级联 + SEC-004 replay + STYLE-001 error — 4/4 Complete 2026-07-31

**关键 must-haves**：见上表 D19-1..D19-10。**建议 status: `passed`**（11 commits + 21 D1-D21 commits 全部落地 main，`go test -race` 全绿，无回归）。

**Evidence Limitations 段（必加）**：
- 原 PLAN.md / CONTEXT.md / DISCUSSION-LOG 永久丢失（与 phase 18 同）
- SUMMARY 显示 base HEAD `89d4cc9`（W0 起点），但 STATE.md §Phase 19 line 204 标 `2281927` 为 "进入基线"——这是 W4 docs commit 前，与 SUMMARY 的 `89d4cc9` 起点不冲突（一个是 phase 19 整体起点，一个是 W4 docs 之前）
- SUMMARY frontmatter `Final HEAD: cacc294`（W2 终点）与 body 实际最终 `6edb772`（docs 终点）不一致——body 为准

### Phase 19 与 Phase 18/20 的契约

- **上游（Phase 18）**：D2（taskServiceAdapter 合并）保留 Phase 18 SM4-GCM 解密逻辑（`VideoRecordingTaskService` 接受 `*CredentialEncryptor` 可选参数）
- **下游（Phase 20）**：`HandleError` + 24 sentinels + `BusinessError` 三件套是 phase 20 的直接前置（20-VERIFICATION §"Key Link Verification" 验证过）
- **20-VERIFICATION 已验证**（[VERIFIED: 20-VERIFICATION.md lines 77-82]）：Phase 19 → Phase 20 contract intact；Phase 17 ↔ Phase 20 contract intact

## Focus Area 5: REQUIREMENTS.md Format

### GSD 标准 REQUIREMENTS.md 模板

[VERIFIED: `$HOME/.claude/get-shit-done/templates/requirements.md`]

GSD 官方模板结构：
- `# Requirements: [Project Name]` 标题
- v1 Requirements（按 category 分组：AUTH-01, CONT-01, ...）
- v2 Requirements（deferred）
- Out of Scope 表
- **Traceability 表**：`| Requirement | Phase | Status |`（**仅 3 列**）

**模板是项目初始化（greenfield）导向**，不是 milestone 追溯导向。CONTEXT D-03.3 的 5 列扩展（`REQ-ID | Phase | 来源 | 状态 | 验证证据`）超出标准模板——这是合理的，因为本 phase 是事后追溯。

### 推荐结构（基于 D-03.3 + GSD 模板融合）

```markdown
# Requirements: Record V2 - v1.1 Milestone Traceability

**Defined:** 2026-08-03 (retro-active)
**Milestone:** v1.1 — 文件管理与编辑增强
**Scope:** Phases 17, 18, 19, 20 (per ROADMAP Progress table)
**Method:** REQ-ID backfilled from deliverables (SUMMARY frontmatter `requirements_completed` was empty for most phase-20 plans — per v1.1 audit recommendation)

## v1.1 Requirements

### REQ-17-*: 后端代码审查 56 项修复 (Phase 17)

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-17-SEC-001 | 17 | audit §6.2 (SEC-001) | done | `internal/config/config.go::ValidateProductionSecrets` + commit 4d3de0b |
| REQ-17-SEC-002 | 17 | audit §6.2 (SEC-002) | done | `cmd/server/app.go::authService.SetAuditService` + commit 4d3de0b |
| REQ-17-SEC-003a | 17 | audit §6.2 (SEC-003a) | done | `internal/huawei/manager.go::SetTLSPolicy` + commit 2bcee29 |
| REQ-17-SEC-003b | 17→18 | audit §6.2 (SEC-003b) | done (delivered by Phase 18) | `internal/services/credential_encryptor.go` + commit 5d536ec + 18-VERIFICATION |
| REQ-17-SEC-004 | 17 | audit §6.2 (SEC-004) | done | `internal/auth/hlstoken/hls_token.go` jti + commit 4d3de0b |
| REQ-17-SEC-005..015 | 17 | audit §6.2 | done | 见 17-02/04 SUMMARY + 17-VERIFICATION |
| REQ-17-BUG-001..016 | 17 | audit §2.x | done (除 BUG-007..014 计划外发现不存在) | 见 17-VERIFICATION |
| REQ-17-PERF-001..016 | 17 | audit §4.x | partial (PERF-003 全库 deferred → Phase 19 兑现) | 见 17-VERIFICATION + 19-VERIFICATION |
| REQ-17-STYLE-001 | 17 | audit §5.x | partial (handler 层 done Phase 19/20; 全库 ~642 处 deferred) | 17-VERIFICATION |
| REQ-17-STYLE-002 | 17 | audit §5.x | N/A (false positive) | 17-04 SUMMARY §"误报" |
| REQ-17-STYLE-009 | 17 | audit §5.x | deferred (133 Get* rename, blast radius 大) | 17-VERIFICATION §Deferred |

### REQ-18-*: 凭据静态加密 + 密钥轮换 (Phase 18)

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-18-001 | 18 | 18-SUMMARY §"算法" | done | SM4-GCM envelope `SM4:<v>:<base64>` @ `internal/utils/sm4_password.go:19` + commit e6315ce |
| REQ-18-002 | 18 | 18-SUMMARY §"密钥族分离" | done | `CREDENTIAL_SM4_*` vs `SM4_SECRET` 隔离 + commit 1dbb3b0 |
| REQ-18-003 | 18 | 18-SUMMARY §"启动期 fail-closed" | done | `Initialize()` 10 步 + 双 InvariantScan + commit bd84fe2 |
| REQ-18-004 | 18 | 18-SUMMARY §Wave 4 §W4d | done | `DEPLOYMENT.md` operator runbook + 物理残留章节 + commit 0c018f2 |
| REQ-18-005 | 18 | post-audit | done | `TestHuaweiDBAdapter_ProductionDecrypts` + commit 5d536ec |

### REQ-19-*: ctx 级联 + SEC-004 replay + STYLE-001 error (Phase 19)

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-19-ctx | 19 | 19-SUMMARY §Wave 3-5 | done | `grep -c ".WithContext(ctx)" internal/services/video_recording_task_service.go` = 42 |
| REQ-19-SEC-004 | 19 | 19-SUMMARY §Wave 1 | done | `TestVerify_MultiSegmentSameToken` + commit 6fbdad4 |
| REQ-19-STYLE-001 | 19 | 19-SUMMARY §Wave 2/6 + D5-D21 | done | mapping.go + HandleError + error_mapper.go + 24 sentinels + ~356 散点 + 21 dN commits |
| REQ-19-jti-db | 19 | 19-SUMMARY §D3 | done | `hls_jti_records` 表 + commit 1f0ec35 |

### REQ-20-*: HandleError 收敛 + sentinel 增强 (Phase 20)

沿用 20-01/20-05 PLAN frontmatter 已定义的 REQ-ID（来源 [VERIFIED: v1.1-MILESTONE-AUDIT.md §"Requirements Traceability"]）：

| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-20a-classify | 20 | 20-01-PLAN frontmatter | done | 20-VERIFICATION (10/10) |
| REQ-20a-formal | 20 | 20-01-PLAN | done | 20-VERIFICATION |
| REQ-20a-ad-user-not-registered | 20 | 20-01-PLAN (R-3) | done | 20-VERIFICATION |
| REQ-20b-sentinel-field | 20 | 20-01-PLAN | done | 20-VERIFICATION |
| REQ-20b-priority | 20 | 20-01-PLAN | done | 20-VERIFICATION |
| REQ-20b-upgrade | 20 | 20-01-PLAN | done | 20-VERIFICATION |
| REQ-20c-generator | 20 | 20-05-PLAN | done | 20-VERIFICATION |
| REQ-20c-doc-sync | 20 | 20-05-PLAN | done | 20-VERIFICATION |
| REQ-20-regression | 20 | 20-01-PLAN | done | 20-VERIFICATION |
| REQ-20-build | 20 | 20-01-PLAN | done | 20-VERIFICATION |
| REQ-20-typed-kind | 20 | 20-CONTEXT §D-01.1 | **deferred** | 显式 out-of-scope per CONTEXT D-01.1 |

## Coverage

- v1.1 requirements: ~80 total (56 from phase 17 + 5 phase 18 + 4 phase 19 + 11 phase 20)
- Mapped to phases: ~80 (100%)
- Orphans: 0 ✓ (deferred items explicitly marked, not orphaned)

## Out-of-scope observation

Phase 16 (visual-reshape) 目录存在（`.planning/phases/16-visual-reshape/16-01-SUMMARY.md`）但**不在 v1.1 ROADMAP Progress 表**（表中只列 17-20）。`init.milestone-op` 报告的 phase_count: 5 可能含 phase 16 — 这是归属歧义。本表**不强行裁定** phase 16 归属，留待 milestone 决策。

---

*Requirements defined: 2026-08-03 (retro-active from audit gaps_found)*
*Last updated: 2026-08-03 after phase 21 retro-verify*
```

### REQ-ID 命名建议（D-03.2 discretion）

推荐 `REQ-<phase>-<source-id>` 格式（如 `REQ-17-SEC-001`），理由：
1. 与 audit 文档原生 ID（SEC-001/BUG-001/...）一一对应，便于 3-source 交叉引用
2. 跨 phase 流转清晰可见（如 `REQ-17-SEC-003b` 在 phase 17 deferred → phase 18 done）
3. orphan 检测简单：任何 audit finding 无对应 REQ-ID 即 orphan

### orphan 检测规则

1. `docs/audits/2026-07-30-backend-code-review.md` 列出 56 findings → 须对应 56 个 REQ-17-* ID（含 deferred/partial/N/A 状态）
2. 18-SUMMARY 列出 5 deliverables → 须对应 ≥ 5 个 REQ-18-* ID
3. 19-SUMMARY 列出 4 deliverables + 21 dN items → 须对应 ≥ 4 个 REQ-19-* 高层 ID（dN items 可选细分）
4. 20-PLAN frontmatter 列出 11 REQ-20-* → 须全部出现

## Focus Area 6: auth_handler.go:57 Behavior Equivalence

### 关键发现：CONTEXT D-04.2 的理由是错的

CONTEXT D-04.2 称：
> HandleError 内部对已知 sentinel/BusinessError 写对应状态码并返回 true，对 unknown error 写 500 并返回 true（**始终非 false**）

**实际代码（`pkg/response/response.go:173-180`）**[VERIFIED: Read]：
```go
func HandleError(c *gin.Context, err error) bool {
    if err == nil || c.Writer.Written() {
        return false  // ← 这里返回 false，不是 true
    }
    httpStatus, respCode, message := errors.MapToHTTPStatus(err)
    GinErrorWithStatus(c, httpStatus, respCode, message)
    return errors.IsKnownError(err)  // ← unknown error 返回 false，不是 true
}
```

`HandleError` 返回值是 `errors.IsKnownError(err)`：
- 已知 sentinel / BusinessError → 写响应 + 返回 `true`
- **unknown error → 写 500 响应 + 返回 `false`**（不是 CONTEXT 所说 `true`）
- `err == nil` 或 `c.Writer.Written()` → 不写，返回 `false`

### 但等价性结论仍然成立（理由不同）

**当前代码**（auth_handler.go:57-61）：
```go
if response.HandleError(c, err) {  // 已知 → true；未知 → false
    return                          // 分支 A：已知错 → 写响应 + return
}
// 兜底：unknown error（response.HandleError 已写 500）。
return                              // 分支 B：未知错 → 已写 500 + return
```

**等价改写**：
```go
response.HandleError(c, err)  // 任何错都写响应（含 unknown → 500）
return                         // 无条件 return
```

**等价性证明**（基于控制流，不依赖 HandleError 返回值）：
- 分支 A（known）：写响应 + `return` === `HandleError(c,err); return`（写响应 + return）
- 分支 B（unknown）：写 500 + `return` === `HandleError(c,err); return`（写 500 + return）
- 两条路径都"先写响应再 return"，与 `HandleError(c,err); return` 完全等价

**关键不变式**：在 auth_handler.go:57 的调用点，`c.Writer.Written()` 必为 `false`（因为 ShouldBindJSON 失败已在 line 38 提前 return，且 line 48-52 的 `h.logger.Warn` 不写 HTTP 响应）。所以 HandleError 必然走到 `GinErrorWithStatus` 写响应分支，不会出现"未写响应就 return"的情况。

### 回归网已存在（auth_handler_test.go）

[VERIFIED: Read] `internal/handlers/auth_handler_test.go::TestLogin_HandleError_ClassifyDrop` 有 **10 个表驱动 sub-tests**，覆盖 5 类错误：

| Sub-test | 错误类 | 期望 Status | 期望 Code | HandleError 返回 |
|----------|--------|-------------|-----------|------------------|
| ErrADUserNotRegistered (R-3) | sentinel | 403 Forbidden | 1003 | true |
| wrapped ErrADUserNotRegistered | wrapped sentinel | 403 | 1003 | true |
| ErrADAccountNotFound | sentinel | 404 NotFound | 1004 | true |
| ErrUserDisabled | sentinel | 403 | 1003 | true |
| ErrADConfigError (R-4) | sentinel | 503 ServiceUnavailable | 1005 | true |
| ErrADUnreachable (R-4) | sentinel | 503 | 1005 | true |
| ErrUnauthorized | sentinel | 401 Unauthorized | 1002 | true |
| wrapped ErrUnauthorized | wrapped sentinel | 401 | 1002 | true |
| BusinessError(CodeInvalidInput) | typed BusinessError | 400 InvalidRequest | 1001 | true |
| unknown ad-hoc error | unknown | 500 InternalServerError | 1005 | **false** |

**重要语义**：测试**直接调用 `response.HandleError(ctx, tt.err)`**（line 127），**不经过 Login handler 的 `if` 语句**。所以测试验证的是 HandleError 本身的写入契约，不是 Login 控制流。

**对 phase 21 的意义**：移除 `if` 后跑 `go test -race ./internal/handlers/...`：
- 该测试针对 `HandleError` 行为，**不受 `if` 移除影响**（仍 10/10 通过）
- Login handler 的实际控制流由代码审查保证（无需新测试）

### 注释清理细节（D-04.3）

当前 `auth_handler.go:53-61`：
```go
// Phase 20 (20-02): Login 错误统一走 response.HandleError；mapping.go 通过
// errors.Is 链自动识别 sentinel → 对应 401/403/404/503/500 状态码。
//   - ErrADUserNotRegistered → 403 (R-3 要求)。
//   - ErrADConfigError / ErrADUnreachable → 503 (R-4: 500 → 503)。
if response.HandleError(c, err) {
    return
}
// 兜底：unknown error（response.HandleError 已写 500）。
return
```

修改后（line 53-56 注释保留，line 57-61 替换）：
```go
// Phase 20 (20-02): Login 错误统一走 response.HandleError；mapping.go 通过
// errors.Is 链自动识别 sentinel → 对应 401/403/404/503/500 状态码。
//   - ErrADUserNotRegistered → 403 (R-3 要求)。
//   - ErrADConfigError / ErrADUnreachable → 503 (R-4: 500 → 503)。
response.HandleError(c, err)
return
```

### 验证命令（acceptance）

```bash
# 1. 静态行为不变
go build ./...

# 2. 既有测试不回归（包含 TestLogin_HandleError_ClassifyDrop 10 个 sub-tests）
go test -race ./internal/handlers/...

# 3. grep 确认规范模式
grep -A 1 "response.HandleError(c, err)" internal/handlers/auth_handler.go | head -5
# 期望：含 `return` 紧随其后，无 `if response.HandleError` 包裹
```

### 注意：planner 必须更正 CONTEXT D-04.2 的理由

写 PLAN.md 时**不要照抄 CONTEXT D-04.2 "HandleError 始终非 false" 的论据**（事实错误）。改用：
> 行为等价性论据：当前 `if HandleError(c, err) { return }; return` 的两条分支（known→`return`、unknown→`return`）都"先写响应再 return"，与 `HandleError(c, err); return` 等价。HandleError 实际返回 `IsKnownError(err)`（known→true、unknown→false），但因两条分支都 return，返回值无关紧要。回归网 `TestLogin_HandleError_ClassifyDrop` 10 sub-tests 验证 HandleError 写入契约不变。

## Focus Area 7: gsd-sdk Helpers

### 可用 native query handlers

[VERIFIED: `gsd-sdk query --help` + 实际调用]

| Handler | 用途 | 本 phase 是否有用 |
|---------|------|-------------------|
| `init.milestone-op` | 里程碑初始化 | ❌（v1.1 已存在） |
| `init.phase-op` | 阶段初始化 | ✅ 已用于读 `commit_docs: true`、`phase_dir` 等 |
| `state.planned-phase --phase <N>` | 查询已规划阶段 | ⚠️ 可选 |
| `check.decision-coverage-plan` | 决策覆盖检查 | ⚠️ 可选（planner 阶段用） |

### 不存在但有用的 handler

**无 verifier-reconstruct 子命令** — retro-verify 无法自动化，必须人工组装 VERIFICATION.md（读 SUMMARY → 查 commit → grep 实时代码）。

**无 requirements 子命令** — REQUIREMENTS.md 必须按 §5 推荐结构人工编写。

### fallback 失败的 handler（环境问题）

`check.plan-state-consistency` 与 `roadmap.phase-counts` 不在 native registry，fallback 到 `gsd-tools.cjs` 时报错（本机 `npx` 缓存路径问题）。不影响 phase 21 工作——planner 不依赖这两个 handler。

### gsd-verifier / gsd-planner 文件位置（供 planner 参考）

[VERIFIED: `find`]
- `$HOME/.claude/get-shit-done/references/few-shot-examples/verifier.md` — 校准用 few-shot
- `$HOME/.claude/get-shit-done/references/verification-patterns.md` / `verification-overrides.md` — 模式与 override 规则
- `$HOME/.claude/get-shit-done/templates/verification-report.md` — VERIFICATION.md 模板
- `$HOME/.claude/get-shit-done/templates/requirements.md` — REQUIREMENTS.md 模板
- `$HOME/.claude/get-shit-done/templates/planner-subagent-prompt.md` — planner 模板（含 VERIFICATION frontmatter 要求）

## Focus Area 8: Validation Architecture (Nyquist Dimension 8)

### Status: EXPLICITLY DEFERRED — DO NOT SCOPE IN

[VERIFIED: 21-CONTEXT.md `<deferred>` line 191]
> Nyquist VALIDATION.md 补齐（phase 16/17 缺、phase 20 draft）—— 独立 phase，本阶段只补 VERIFICATION.md（goal-backward），不补 VALIDATION.md（Nyquist dimension）

[VERIFIED: v1.1-MILESTONE-AUDIT.md lines 55-60 + 188-198]
- Nyquist overall: **incomplete**（0 compliant / 1 partial / 4 missing）
- Phase 16/17: ❌ missing VALIDATION.md
- Phase 18/19: ❌ missing（dir missing）
- Phase 20: ⚠ partial（VALIDATION.md exists but `nyquist_compliant: false`, draft）

### planner 必须忽略 Nyquist

- **不要**在 phase 21 创建任何 VALIDATION.md
- **不要**在 17/18/19-VERIFICATION.md 里包含 Nyquist dimension 章节
- **VERIFICATION.md 的 `## Validation Architecture` 章节标 N/A-deferred**：本 phase 不涉及
- 如果 roadmap 决定补 Nyquist，应另开 phase 22+

### Test Framework Reference (for planner awareness only)

本 phase 唯一需要测试的环节是 auth:57 fix（D-05.3）。框架现状：

| Property | Value |
|----------|-------|
| Framework | Go testing + testify/assert + testify/require |
| Config file | none（Go 标准 `go test`） |
| Quick run command | `go test -race ./internal/handlers/...`（< 30s） |
| Full suite command | `go test -race ./...`（> 60s） |

auth:57 fix 不需要新测试，只需跑既有 `TestLogin_HandleError_ClassifyDrop`（10 sub-tests）+ `go build ./...`。

---

## Common Pitfalls

### Pitfall 1: CONTEXT D-04.2 的理由是错的，照抄会让 PLAN 含事实错误
**What goes wrong:** planner 若照抄 CONTEXT D-04.2 "HandleError 始终非 false" 写 PLAN.md acceptance criteria，会在 review 时被发现事实错误，导致返工。
**Why it happens:** CONTEXT.md 由 orchestrator 在 phase 启动前合成，可能未实际读 `pkg/response/response.go`。
**How to avoid:** 用本 RESEARCH §6 提供的正确论据（基于控制流而非返回值）。
**Warning signs:** PLAN 出现 "HandleError always returns true" 字样。

### Pitfall 2: ROOT-level `18-SUMMARY.md` frontmatter 与 body 的 HEAD 不一致
**What goes wrong:** 18-SUMMARY frontmatter 标 `Final HEAD: bd84fe2`（W3 终点），但 §Wave 4 body 实际写到 `0c018f2`。直接信 frontmatter 会让 VERIFICATION 漏掉 Wave 4 交付物（operator runbook + 重复轮换测试 + per-site 计数日志）。
**Why it happens:** Wave 4 是后续 agent 接力追加的，frontmatter 没同步更新。
**How to avoid:** 取 body 为准（`0c018f2` 是真正终点）；在 VERIFICATION "Evidence Limitations" 段注明此不一致。
**Warning signs:** 18-VERIFICATION 只列 5 commits（W1a-W3）而非 9 commits（W1a-W4d）。

### Pitfall 3: phase 18/19 deferred 项与 phase 17 deferred 项混淆
**What goes wrong:** SEC-003b 在 phase 17 标 deferred，但在 phase 18 兑现；PERF-003 在 phase 17 标 deferred，在 phase 19 兑现。VERIFICATION 若孤立看每个 phase，会把这些误标为"未做"。
**Why it happens:** 三个 phase 的 deferred 项有跨 phase 兑现关系，不是独立列表。
**How to avoid:** 在 17-VERIFICATION 的 Deferred 段明确"SEC-003b → delivered by Phase 18（见 18-VERIFICATION）"、"PERF-003 → delivered by Phase 19（见 19-VERIFICATION）"。
**Warning signs:** 17-VERIFICATION 的 Deferred 段只列"deferred"不列"where delivered"。

### Pitfall 4: `.planning/` 在 `.gitignore`，提交需 `git add -f`
**What goes wrong:** 用普通 `git add .planning/REQUIREMENTS.md` 会失败（被 gitignore 忽略）。
**Why it happens:** `.gitignore` line 74 是 `/.planning/`。
**How to avoid:** D-06.1 已明确——用 `git add -f .planning/...`。planner 在每个 docs commit 的 task action 里必须显式写 `-f`。
**Warning signs:** `git status` 看不到新增的 `.planning/REQUIREMENTS.md`。

### Pitfall 5: retro-verify 的"L3 Wired"证据容易被遗漏
**What goes wrong:** VERIFICATION 只列"文件存在 + 有导出符号"（L1+L2），不验证消费者实际调用（L3），导致校准语料 37% 的 "missing wiring" gap 重现。
**Why it happens:** L3 需要额外 grep 跨文件引用。
**How to avoid:** 每个 must-have 至少给一个 L3 证据（如"credential_encryptor.go 的 EncryptGCM 被 input_config_service.go line N 调用"）。
**Warning signs:** Evidence 列只写"exists at path"无引用关系。

### Pitfall 6: 误把 phase 21 的 5d536ec invariant 当作 phase 21 交付物
**What goes wrong:** 5d536ec commit 是 2026-08-02（phase 21 启动前），但 commit message 提到 "Phase 21"。planner 误以为还要再写代码补这个 invariant。
**Why it happens:** commit 是预存证据（21-CONTEXT D-02.4 显式引用为证据源）。
**How to avoid:** 在 18-VERIFICATION 明确："此 invariant commit 在 phase 21 启动前已落地，作为 retro-verify 证据使用，不是 phase 21 的交付物"。
**Warning signs:** PLAN 出现"为 phase 18 重写 TestHuaweiDBAdapter_ProductionDecrypts"任务。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| VERIFICATION.md 格式 | 自创结构 | 镜像 20-VERIFICATION.md 章节 + `$HOME/.claude/get-shit-done/templates/verification-report.md` | 已有通过 10/10 验证的本地 gold reference |
| REQUIREMENTS.md 表结构 | 自创列 | GSD 模板 + D-03.3 5 列扩展 | 模板有 traceability 表范例，D-03.3 指定 5 列 |
| REQ-ID 命名 | 新造命名方案 | `REQ-<phase>-<source-id>`（如 REQ-17-SEC-001） | 与 audit 文档原生 ID 一一对应，便于追溯 |
| `hls_jti_records` 表存在性验证 | 跑迁移看表 | grep `AutoMigrate(&models.HLSJtiRecord{})` 在 cmd/server/app.go | 已注册（line 340） |
| HandleError 行为验证 | 写新 contract test | 跑既有 `TestLogin_HandleError_ClassifyDrop`（10 sub-tests） | 已覆盖 5 类错误 + R-3/R-4 |

## Code Examples

### Example 1: 18-VERIFICATION.md frontmatter 骨架（retro-verify）

```yaml
---
phase: 18-credential-static-encryption
verified: 2026-08-03T00:00:00Z
status: passed
score: 11/11 must-haves verified
method: retro-active goal-backward (SUMMARY + git history + live code)
deferred:
  - item: "Wave 5 候选 (KMS/Vault 集成 / staging 真实数据 post-audit / FileShare.Password 加密)"
    reason: "out-of-v1.1 scope per 18-SUMMARY §'不在本 phase 范围'"
human_verification: []
evidence_limitations:
  - "原 PLAN.md / CONTEXT.md / DISCUSSION-LOG 永久丢失（从未 git 入库）"
  - "18-SUMMARY frontmatter Final HEAD (bd84fe2) 与 body 实际终点 (0c018f2) 不一致 — body 为准"
  - "真实生产数据 post-audit 未做（18-SUMMARY 列为 out-of-scope）"
---
```

### Example 2: 17-VERIFICATION.md Observable Truths 行（含跨 phase deferred）

```markdown
| 14 | SEC-003b 华为密码 DB 加密未在 phase 17 范围内实施（per CONTEXT `<deferred>`） | VERIFIED (deferred-by-design) | `internal/huawei/manager.go:132` 注释原标 "deferred (per CONTEXT.md)"；phase 18 兑现 — 见 18-VERIFICATION D18-1..D18-5；commit 5d536ec 把注释改为 "SEC-003b/Phase 21: done" |
```

### Example 3: REQUIREMENTS.md orphan 检测段

```markdown
## Coverage

- v1.1 requirements: 80 total (56 REQ-17-* + 5 REQ-18-* + 4 REQ-19-* + 11 REQ-20-* + 4 跨 phase 兑现)
- Mapped to phases: 80 (100%)
- Orphans: 0 ✓
- Deferred (explicit): 4 (REQ-17-STYLE-001 全库, REQ-17-STYLE-009, REQ-17-PERF-003 全库 → REQ-19-ctx 兑现, REQ-20-typed-kind)
- Delivered by other phase: 2 (REQ-17-SEC-003b → REQ-18-*, REQ-17-PERF-003 → REQ-19-ctx)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Phase dir 完整后清理（phase 18/19） | Phase dir 保留（phase 20、21） | v1.1 中段 | retro-verify 需重建目录 |
| `taskServiceAdapter` 单独 struct | 合并到 `VideoRecordingTaskService` | Phase 19 D2 | ctx 级联单层化 |
| jti in-memory map | `hls_jti_records` DB 表 | Phase 19 D3 | 跨实例/重启持久化 |
| `if HandleError(c, err) { return }; return` | `HandleError(c, err); return` | Phase 20（多数 handler）/ Phase 21（auth:57 收尾） | 关闭 CR-01 重引入风险 |
| `base64-stub` password 加密 | SM4-GCM envelope | Phase 18 | 真正 at-rest 加密 |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Phase 16 不在 v1.1 ROADMAP Progress 表（仅 17-20） | §5 + REQUIREMENTS Out-of-scope | LOW — 已通过 grep ROADMAP 验证 [VERIFIED] |
| A2 | `hls_jti_records` 是正确的表名（不是 `hls_jti_record`） | §3 D18-12, §4 D19-6 | LOW — 已通过 `models/hls_jti_record.go:28` `return "hls_jti_records"` 验证 [VERIFIED]；CONTEXT D-03.2 用复数正确 |
| A3 | `5d536ec` 在 phase 21 启动前已存在（不是 phase 21 交付物） | §3, §Pitfalls 6 | LOW — commit date 2026-08-02 早于 phase 21 启动；21-CONTEXT D-02.4 显式引用为证据 [VERIFIED] |
| A4 | phase 18 真正最终 HEAD 是 `0c018f2`（不是 frontmatter 的 `bd84fe2`） | §3 | MEDIUM — 18-SUMMARY body §Wave 4 明确"W1a..W4d 共 9 个原子 commit"，frontmatter 与 body 不一致 [VERIFIED: body] |
| A5 | phase 19 全部交付物由 11 wave commits + 21 D1-D21 commits 落地 | §4 | LOW — 19-SUMMARY body 与 STATE.md §Phase 19 Final Status 一致 [VERIFIED] |
| A6 | CONTEXT D-04.2 "HandleError 始终非 false" 是事实错误 | §6 | HIGH — 已通过 Read `pkg/response/response.go:173-180` 验证；planner 必须用正确理由 [VERIFIED] |

## Open Questions (RESOLVED — all 3 implemented in 21-0x plans)

1. **RESOLVED (implemented in 21-04):** REQUIREMENTS.md 是否登记根目录 `18-SUMMARY.md` / `19-SUMMARY.md` 为 canonical ref？
   - What we know: CONTEXT D-02.6 让根目录 SUMMARY 保持原位，`.planning/phases/` 内只放副本
   - What's unclear: REQUIREMENTS.md 的"来源"列应指向哪个路径（根目录原版？副本？都登记？）
   - Recommendation: 在 REQUIREMENTS.md 加 `## Canonical References` 段，明确两路径都是 canonical（根目录 = git 历史 / `.planning/phases/` 副本 = phase 目录自洽），来源列指向 `.planning/phases/NN-*/NN-SUMMARY.md` 副本（与 VERIFICATION 同目录便于追溯）。

2. **RESOLVED (implemented in 21-01/02/03 — no new reviewer):** 17-VERIFICATION 是否需要 opencode 等 reviewer 共识？
   - What we know: 17-REVIEWS.md 是 post-execution 评审，仅 opencode 一位外部 reviewer
   - What's unclear: retro-verify 是否需要再请 reviewer
   - Recommendation: **不需要**。phase 21 的 retro-verify 是基于已有证据的文档重建，不是新代码评审。VERIFICATION.md 的 Evidence Limitations 段诚实记录"基于 SUMMARY + commit + 代码，非基于原始 PLAN"即可。

3. **RESOLVED (implemented — Wave 2 = 21-04 depends_on [21-01,21-02,21-03]):** D-06.1 提交分组中 phase 17 / 18 / 19 / REQUIREMENTS 是 4 个独立 docs commit 还是合并？
   - What we know: CONTEXT D-06.1 明确 5 个 commit（4 docs + 1 fix）
   - What's unclear: 顺序与依赖（17 是否先于 18/19？REQUIREMENTS 是否最后？）
   - Recommendation: 按 D-06.1 顺序：17 → 18 → 19 → REQUIREMENTS → auth:57 fix。理由：REQUIREMENTS 引用 17/18/19 的 VERIFICATION 路径作为验证证据，需先有 VERIFICATION 文件；auth:57 fix 与 docs 完全独立，最后单独提交。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| git | 提交 / 历史查询 / `git show` | ✓ | 默认 | — |
| Go toolchain | auth:57 build + test（D-05.3） | ✓ | 1.24（per STATE.md） | — |
| gsd-sdk | `init.phase-op` / `state.planned-phase` | ✓ | npx cache | 部分查询 handler fallback 失败（不影响 phase 21） |
| `gsd-tools.cjs` graphify | 知识图谱查询 | ⚠ 未探测 | — | 不需要（本 phase 是文档重建，无代码域） |
| slopcheck | 包合法性检查 | N/A | — | 本 phase **不安装任何外部包**（纯 docs + 1 行 Go 代码），跳过 Package Legitimacy Gate |

**Missing dependencies with no fallback:** 无。

**Missing dependencies with fallback:** `gsd-tools.cjs` 的 `check.plan-state-consistency` / `roadmap.phase-counts` 在本环境 fallback 失败，但 planner 不依赖这两个 handler（D-05 验收标准用 `go build` / `go test` / grep，不用 gsd-sdk handler）。

## Validation Architecture

**Status:** N/A — DEFERRED per CONTEXT `<deferred>`。

本 phase 不创建任何 VALIDATION.md（Nyquist dimension）。phase 21 的"验证"概念是 VERIFICATION.md（goal-backward），不是 VALIDATION.md（Nyquist）。两者不可混淆。

唯一与测试相关的 acceptance 是 auth:57 fix 的回归测试：
- `go build ./...`
- `go test -race ./internal/handlers/...`（含既有 `TestLogin_HandleError_ClassifyDrop` 10 sub-tests）

详见 §6 验证命令。

## Security Domain

**Status:** N/A — 本 phase 不引入新代码 / 新依赖 / 新 API。

- auth:57 fix 是**纯控制流规范化**（`if X { return }; return` → `X; return`），不改变任何安全语义
- retro-verify 与 REQUIREMENTS.md 是**纯文档**，不触及运行时安全
- 无新依赖引入（跳过 Package Legitimacy Gate）

唯一与安全相关的观察：auth_handler.go:57 规范化后**降低 CR-01 重引入风险**（per v1.1-MILESTONE-AUDIT.md §Cross-Phase Integration Findings WARNING）。这是关闭审计唯一 WARNING 的安全收益，但不改变当前安全状态（CURRENT 行为已正确，只是规范化防未来回归）。

## Sources

### Primary (HIGH confidence)
- `.planning/phases/21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem/21-CONTEXT.md` — phase 21 决策 source of truth
- `.planning/v1.1-MILESTONE-AUDIT.md` — 审计报告 source of truth（gaps_found）
- `.planning/STATE.md` §Phase 18 / §Phase 19 / §Phase 19 Final Status — scope / base HEAD / deviations 权威记录
- `.planning/phases/20-handleerror-classify-convergence/20-VERIFICATION.md` — 本项目唯一通过的 VERIFICATION.md gold reference
- `internal/huawei/manager.go:132-134` — SEC-003b marker comment（live code）
- `internal/handlers/auth_handler.go:34-70` — Login handler 含 :57 待修模式
- `pkg/response/response.go:173-180` — HandleError 实现（推翻 CONTEXT D-04.2 理由）
- `internal/errors/mapping.go:32-203` — MapToHTTPStatus + knownSentinels slice（42 sentinels）
- `internal/handlers/auth_handler_test.go:35-156` — TestLogin_HandleError_ClassifyDrop 10 sub-tests
- `git show 5d536ec` — SEC-003b invariant commit（manager.go + app_test.go 完整 diff）
- `git log --oneline 2281927..6edb772` — phase 19 commit range（8 commits post-W4-docs）
- `.planning/ROADMAP.md` lines 17-20 — Progress table phase 17-20 完整记录
- `18-SUMMARY.md` / `19-SUMMARY.md` / `docs/audits/phase-19-D5-D21-summary.md` — phase 18/19 历史记录
- `.planning/phases/17-56-p0-p1-p2/17-{CONTEXT,01..04-SUMMARY,REVIEWS}.md` — phase 17 完整目录

### Secondary (MEDIUM confidence)
- `$HOME/.claude/get-shit-done/templates/verification-report.md` — GSD VERIFICATION.md 模板（status 值、frontmatter 字段、章节结构）
- `$HOME/.claude/get-shit-done/templates/requirements.md` — GSD REQUIREMENTS.md 模板（traceability 表范例）
- `$HOME/.claude/get-shit-done/references/few-shot-examples/verifier.md` — verifier L1/L2/L3 三层证据模型 + 校准 gap 分布

### Tertiary (LOW confidence)
- 无（全部 claim 已通过本地证据或 GSD 模板验证）

## Metadata

**Confidence breakdown:**
- VERIFICATION.md format contract: HIGH — 20-VERIFICATION.md 是本项目通过的 gold reference + GSD 模板双重确认
- Phase 17 retro-verify readiness: HIGH — 4 SUMMARY + CONTEXT + REVIEWS + git commits 全部本地可验证
- Phase 18 evidence pack: HIGH — SUMMARY + 9 commits + STATE.md + live code（含 `5d536ec` 后续 invariant）三源一致
- Phase 19 evidence pack: HIGH — SUMMARY + D5-D21 summary + 21 dN commits + STATE.md + live code（42 WithContext sites）多源一致
- REQUIREMENTS.md format: HIGH — GSD 模板存在 + D-03.3 扩展定义清晰
- auth:57 行为等价性: HIGH — Read 源码 + 控制流证明 + 10 sub-test 回归网
- CONTEXT D-04.2 理由错误: HIGH — 直接 Read `pkg/response/response.go:173-180` 反证

**Research date:** 2026-08-03
**Valid until:** 2026-09-03（30 天，stable；除非 phase 21 已开工则按 PLAN 时间）

## RESEARCH COMPLETE

**Phase:** 21 - Close v1.1 gaps
**Confidence:** HIGH

### Key Findings

1. **CONTEXT D-04.2 关于 HandleError "始终非 false" 的论据是事实错误**（`pkg/response/response.go:179` 返回 `IsKnownError(err)`，unknown 错误时为 `false`）。但 auth:57 行为等价性结论仍成立——基于控制流（两条分支都 return），不基于返回值。planner 必须用正确论据，否则 PLAN 会含事实错误。

2. **3 份 retro-verify 证据链完整且三源一致**（SUMMARY ↔ commit ↔ live code），无需补做代码或运行时验证。phase 17 目录已齐全仅缺 VERIFICATION；phase 18/19 需重建目录并复制 SUMMARY 副本。所有 commit hash 已枚举（17: 45 commits / 18: 9 commits + 5d536ec / 19: 11 wave + 21 dN commits）。

3. **REQUIREMENTS.md 需基于 GSD 模板扩展为 5 列表**（D-03.3：REQ-ID | Phase | 来源 | 状态 | 验证证据），覆盖 ~80 个 REQ-ID（56 REQ-17 + 5 REQ-18 + 4 REQ-19 + 11 REQ-20 + 4 跨 phase 兑现）。REQ-17-SEC-003b / REQ-17-PERF-003 跨 phase 兑现关系必须在表里显式标注。

4. **`5d536ec` 是 phase 21 启动前已落地的预存证据**（2026-08-02，commit message 提 "Phase 21"），含 `TestHuaweiDBAdapter_ProductionDecrypts` invariant 测试。作为 phase 18 retro-verify 的强证据使用，**不是** phase 21 的交付物（phase 21 不改业务代码）。

5. **`.planning/` 在 `.gitignore`（line 74）**，所有新文档提交必须用 `git add -f`。`gsd-sdk` 无 verifier-reconstruct / requirements 加速器，全部人工组装。

### File Created
`D:\CODE\ClaudeCode\record_V2\.planning\phases\21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem\21-RESEARCH.md`

### Confidence Assessment

| Area | Level | Reason |
|------|-------|--------|
| VERIFICATION.md format contract | HIGH | 20-VERIFICATION.md gold reference + GSD template 双源确认 |
| Phase 17 retro-verify readiness | HIGH | 4 SUMMARY + CONTEXT + REVIEWS + 45 commits 全本地可验证 |
| Phase 18 evidence pack | HIGH | SUMMARY + 9 commits + STATE.md + live code + 5d536ec invariant 三源一致 |
| Phase 19 evidence pack | HIGH | SUMMARY + D5-D21 summary + 32 commits + STATE.md + 42 live WithContext sites 多源一致 |
| REQUIREMENTS.md format | HIGH | GSD 模板 + D-03.3 扩展定义清晰 |
| auth:57 行为等价性 | HIGH | Read 源码 + 控制流证明 + 10 sub-tests 回归网 |
| CONTEXT D-04.2 错误检测 | HIGH | 直接 Read 反证 |

### Open Questions
- Q1: REQUIREMENTS.md "来源"列指向根目录 SUMMARY 还是 `.planning/phases/` 副本？（推荐：副本，便于 VERIFICATION 同目录追溯；root 原版在 Canonical References 段登记）
- Q2: retro-verify 是否需再请 reviewer？（推荐：不需要，VERIFICATION 的 Evidence Limitations 段诚实记录即够）
- Q3: 5 个 commit 顺序？（推荐：17 → 18 → 19 → REQUIREMENTS → auth:57 fix，因 REQUIREMENTS 引用前 3 个 VERIFICATION 路径）

### Ready for Planning
Research complete. Planner can now create PLAN.md files. 建议生成 4 个 docs plan + 1 个 code plan（按 D-06.1 提交分组），每个 plan 任务引用本 RESEARCH 的具体 file:line / commit / test name 作为 acceptance criteria 证据。
