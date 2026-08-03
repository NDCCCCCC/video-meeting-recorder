# Phase 21: Close v1.1 gaps — retro-verify 17/18/19 + REQUIREMENTS.md + auth:57 fix - Context

**Gathered:** 2026-08-03
**Status:** Ready for planning
**Source:** Synthesized from `v1.1-MILESTONE-AUDIT.md` (orchestrator-generated, per user decision — audit treated as locked spec)

<domain>
## Phase Boundary

关闭 v1.1 里程碑审计（`v1.1-MILESTONE-AUDIT.md`，status: `gaps_found`）发现的三类**过程缺口**，使里程碑可在重审时诚实归档为 `passed`。代码库本身已功能完整（`go test -race ./...` 全绿），本阶段**不改业务功能**，只补齐过程产物 + 1 处 1 行代码 WARNING。

**属于本阶段（3 项交付）:**

1. **Retro-verify phase 17 / 18 / 19** — 用 gsd-verifier 的 goal-backward 方法，基于 SUMMARY + git history + 实时代码，为三个未验证阶段重建 `VERIFICATION.md`：
   - Phase 17：目录存在（CONTEXT + 4 PLAN + 4 SUMMARY + REVIEWS 齐全），**仅缺 VERIFICATION.md**
   - Phase 18：目录缺失 —— 重建最小目录，证据源 = 根目录 `18-SUMMARY.md` + STATE.md §Phase 18 + commit `5d536ec` + 实时代码
   - Phase 19：目录缺失 —— 重建最小目录，证据源 = 根目录 `19-SUMMARY.md` + `docs/audits/phase-19-D5-D21-summary.md` + STATE.md §Phase 19 + 21+ `refactor(19/dN)` commits + 实时代码
2. **创建 `.planning/REQUIREMENTS.md`** — 建立 v1.1 milestone（phase 17-20）的 REQ-ID 追溯表，使 orphan 检测 + 3-source 交叉引用成为可能
3. **修复 `internal/handlers/auth_handler.go:57`** —— 1 行替换，关闭审计唯一的 WARNING（latent CR-01 重引入风险）

**不属于本阶段:**

- **Nyquist VALIDATION.md 缺口**（phase 16/17 无 VALIDATION.md、phase 20 仍是 `nyquist_compliant: false` draft）—— 独立关注点，phase 标题未涉及，列入 `<deferred>`
- **Phase 16 归属歧义**（不在 v1.1 ROADMAP Progress 表但 init 计数含 16）—— 仅在 REQUIREMENTS.md 里以观察记录，不强行裁定
- **审计 `tech_debt` 清单的 10+ 项**（bare zap.Error 残留、ppt_handler 9 处、STYLE-001/PERF-003 全库迁移等）—— 均为后续独立 phase，本阶段不触碰代码
- **重跑 `/gsd:audit-milestone v1.1`** —— 这是 phase 21 验收**之后**的动作，不属于本阶段交付（但其通过条件是本阶段的成功判据）
- 任何业务功能变更、前端改动、数据库 schema 变更、新依赖引入

</domain>

<decisions>
## Implementation Decisions

### D-01 范围聚焦（3 项 + 硬边界）
- **D-01.1:** 本阶段**仅交付 3 项**：(a) retro-verify 17/18/19 的 VERIFICATION.md，(b) `.planning/REQUIREMENTS.md`，(c) `auth_handler.go:57` 1 行修复。其余审计 findings/tech_debt 不触
- **D-01.2:** 改动面 = `.planning/`（新增 REQUIREMENTS.md + 3 个 VERIFICATION.md + 重建 18/19 目录）+ `internal/handlers/auth_handler.go`（1 行）。**不动** `frontend/`、`docs/audits/*.md`（不可变 source of truth）、业务 service 代码、DB schema
- **D-01.3:** 代码侧只允许 `auth_handler.go:57` 一处变更；任何「顺手清理」bare zap.Error / ppt_handler / err.Error() 泄漏的冲动都必须拒绝（归后续 phase）

### D-02 Retro-verify 方法论（goal-backward，证据驱动）
- **D-02.1:** 采用 gsd-verifier 的 **goal-backward** 方法：从各 phase 的 ROADMAP `**Goal:**` 出发，逐条验证「代码库是否真的交付了 goal 承诺的东西」，而不是「任务清单是否打勾」。每个 VERIFICATION.md 必须含 `must_haves` + 证据 + `status: passed|partial` + 诚实的数据局限说明
- **D-02.2:** 证据层级（从强到弱）：① 实时代码 grep / `go build` / `go test -race` → ② git commit history（diff + message）→ ③ SUMMARY.md / STATE.md 记录 → ④ 推断（必须标注 "inferred"）。**禁止**把推断当事实
- **D-02.3:** **Phase 17 retro-verify** —— 目录已齐全，直接基于 `17-CONTEXT.md` 的 must_haves（D-01.4 全量 56 项、D-04 测试纪律、D-03 破坏性变更）+ 4 个 SUMMARY + REVIEWS.md，对实时代码做 goal-backward 验证。重点核验 deferred 项（SEC-003b / STYLE-001 全库 / STYLE-009）确实未做且确实 deferred
- **D-02.4:** **Phase 18 retro-verify** —— 重建 `.planning/phases/18-凭据静态加密-SEC-003b/`（或 planner 选定的 slug），把根目录 `18-SUMMARY.md` **复制**（非移动）为目录内 `18-SUMMARY.md`，新写 `18-VERIFICATION.md`。证据：commit `5d536ec`（`cmd/server/app_test.go` 61 行生产解密不变式 + `internal/huawei/manager.go`）+ STATE.md §Phase 18（SM4-GCM AEAD、envelope 格式 `SM4:<version>:<base64(nonce|ciphertext|tag)>`、密钥族隔离、fail-closed 10 步扫描）+ 实时代码。核验：密码字段确为加密存储、解密路径在生产配置下不变、密钥轮换手册存在
- **D-02.5:** **Phase 19 retro-verify** —— 重建 `.planning/phases/19-ctx-全量级联-SEC-004-replay-STYLE-001-error/`（或 planner 选定 slug），复制根目录 `19-SUMMARY.md` + `docs/audits/phase-19-D5-D21-summary.md` 进目录，新写 `19-VERIFICATION.md`。证据：21+ `refactor(19/dN)` commits（D1-D21 逐项）+ STATE.md §Phase 19 Final Status + 实时代码。核验：ctx 全量级联（`WithContext` 覆盖）、SEC-004 jti 防重放（`hls_jti_records` 表）、STYLE-001 error 迁移（gorm wrap → BusinessError + HandleError）、24 sentinels + ~356 散点收敛
- **D-02.6:** **目录重建策略** —— 根目录 `18-SUMMARY.md` / `19-SUMMARY.md` 是 git-tracked 历史记录，**保持原位不动**；只在 `.planning/phases/` 下重建目录并复制副本（`.planning/` 在 `.gitignore`，用 `git add -f` 入库）。这让每个 phase 目录自洽，同时不破坏 git 历史
- **D-02.7:** **诚实标注局限** —— phase 18/19 的原始 PLAN.md / CONTEXT.md / DISCUSSION-LOG 无法恢复（从未入库），VERIFICATION.md 必须在 "Evidence Limitations" 段如实说明：验证基于 SUMMARY + commit + 代码，非基于原始计划文档

### D-03 REQUIREMENTS.md 内容与范围
- **D-03.1:** 范围 = **v1.1 milestone（phase 17/18/19/20）**。v1.0（phase 01-14）已 shipped 归档，不入此表（如需可另开 phase 补 v1.0 追溯）
- **D-03.2:** REQ-ID 体系（planner 可微调命名，但必须覆盖这些来源）：
  - **REQ-17-*** : phase 17 的 56 个 finding（SEC-001..015 / BUG-001..016 / PERF-001..016 / STYLE-001..010），来源 `docs/audits/2026-07-30-backend-code-review.md`。deferred 项（SEC-003b / STYLE-001 全库 / STYLE-009）标注 status
  - **REQ-18-* : SEC-003b 凭据静态加密 + 密钥轮换（SM4-GCM envelope、密钥族隔离、fail-closed 校验、轮换手册）
  - **REQ-19-* : ctx 全量级联 + SEC-004 jti 防重放 + STYLE-001 error 迁移 + D5-D21 sentinel 体系
  - **REQ-20-* : 沿用 phase 20 已有 `REQ-20a/b/c`（classify 收敛 / sentinel-field / generator）—— 来源 20-01/20-05 PLAN frontmatter + 20-CONTEXT.md
- **D-03.3:** 表结构至少含：`REQ-ID | Phase | 来源 (audit/CONTEXT/PLAN) | 状态 (done/deferred/partial) | 验证证据 (VERIFICATION.md / commit / test)`。须支持 orphan 检测（REQ-ID 无 plan 覆盖）与 3-source 交叉引用（REQUIREMENTS ↔ ROADMAP ↔ PLAN）
- **D-03.4:** phase 16 归属歧义：在 REQUIREMENTS.md 加一段 `## Out-of-scope observation`，记录 phase 16 不在 v1.1 ROADMAP Progress 表但 init 计数含 16，**不强行裁定**（留待 milestone 决策）
- **D-03.5:** 不伪造追溯 —— 凡 SUMMARY frontmatter `requirements_completed` 为空的（审计指出 phase 20 多个 SUMMARY 如此），按实际交付物回填，并在表里标注 "backfilled from deliverables, SUMMARY frontmatter was empty"

### D-04 auth_handler.go:57 修复（1 行，行为不变）
- **D-04.1:** 当前代码（line 57-61）：
  ```go
  if response.HandleError(c, err) {
      return
  }
  // 兜底：unknown error（response.HandleError 已写 500）。
  return
  ```
  替换为规范模式：
  ```go
  response.HandleError(c, err)
  return
  ```
- **D-04.2:** **行为等价性证明（基于控制流，非返回值）** —— `HandleError` 实际返回 `errors.IsKnownError(err)`（known sentinel/BusinessError → 写对应状态码 + 返回 `true`；unknown error → 写 500 + 返回 `false`；`err==nil`/`Written()` → 不写 + 返回 `false`。详见 `pkg/response/response.go:173-180`）。当前 `if HandleError(c,err){return}; return` 的两条分支——known（进 if 内 return）/ unknown（落到裸 return）——**都是「先写响应再 return」**，与 `HandleError(c,err); return` 观察等价；返回值无关紧要。**关键不变式**：在 auth:57 调用点 `c.Writer.Written()` 必为 `false`（ShouldBindJSON 失败已在 line 38 提前 return，line 48-52 的 `Warn` 日志不写 HTTP），故 HandleError 必走 `GinErrorWithStatus` 写响应分支，绝不会「未写响应就 return」。回归网 = `auth_handler_test.go::TestLogin_HandleError_ClassifyDrop` 10 个 sub-tests（5 类错误 + R-3/R-4），改后须 `go test -race ./internal/handlers/...` 全绿。（更正：早期版本此条曾误称「HandleError 始终非 false」——事实错误，由 21-RESEARCH §6 推翻，本条已按控制流论证修订。）
- **D-04.3:** 注释清理：删除 line 60 「兜底」注释（已无兜底分支）；保留 line 53-56 解释 mapping.go 行为的注释
- **D-04.4:** 不动 auth_handler.go 其他 3 处 tech_debt（`:93 RefreshToken` / `:182 ChangePassword` / `LogoutAll` 的 raw `GinError + err.Error()` 泄漏）—— 归后续 phase

### D-05 验收标准（goal-backward）
- **D-05.1:** 3 个 VERIFICATION.md 各含 `status: passed`（或 `partial` + 明确 partial 理由），must_haves 全部有证据指向
- **D-05.2:** `.planning/REQUIREMENTS.md` 存在、v1.1 四个 phase 全覆盖、无 REQ-ID orphan（除非显式标 deferred）
- **D-05.3:** `auth_handler.go:57` 为规范模式；`go build ./...` + `go test -race ./internal/handlers/...` 全绿；既有 auth 测试不回归
- **D-05.4:** `v1.1-MILESTONE-AUDIT.md` 的 5 项 gap（REQUIREMENTS 缺失 / phase17 未验证 / phase18 目录缺 / phase19 目录缺 / auth:57 WARNING）全部可标记关闭（重跑 `/gsd:audit-milestone v1.1` 应由 gaps_found → passed —— 重跑本身不在本阶段，但产物须支撑该结论）

### D-06 提交策略
- **D-06.1:** 提交分组（`commit_docs: true`，但 `.planning/` 需 `git add -f`）：
  - `docs(21): retro-verify phase 17 — reconstruct VERIFICATION.md`
  - `docs(21): retro-verify phase 18 — reconstruct dir + VERIFICATION.md`
  - `docs(21): retro-verify phase 19 — reconstruct dir + VERIFICATION.md`
  - `docs(21): create REQUIREMENTS.md — v1.1 REQ-ID traceability`
  - `fix(handlers/SEC-008): auth_handler.go:57 canonical HandleError pattern`（代码改动单独提交，与 docs 分离 —— 与用户「debug 改动与 phase 工作分提交」偏好一致）
- **D-06.2:** 代码提交（auth:57）必须 `go build ./...` + `go test -race ./internal/handlers/...` 通过方可提交
- **D-06.3:** 不修改 `docs/audits/*.md`（不可变）；REQUIREMENTS.md 是新建文件，不冲突

### Claude's Discretion
- D-02.4/D-02.5 中重建目录的具体 slug 命名（遵循 `.planning/phases/NN-<slug>/` 既有约定即可）
- D-03.2 中 REQ-ID 的确切前缀格式（`REQ-17-SEC-001` vs `REQ-17.1` 等），只要覆盖全部来源且无歧义
- D-02 中每个 VERIFICATION.md 的 must_haves 具体条目措辞（从对应 phase 的 ROADMAP Goal + CONTEXT must_haves 派生）
- 是否把根目录 `18-SUMMARY.md` / `19-SUMMARY.md` 在 REQUIREMENTS.md 里登记为 canonical ref（建议登记，便于审计追溯）

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 审计与状态（本阶段 spec 来源）
- `.planning/v1.1-MILESTONE-AUDIT.md` — 审计报告 source of truth（gaps_found；§Recommendation 列出 3 条 remediation = 本阶段 3 项交付）。**不可修改**
- `.planning/STATE.md` §Phase 18（line 52-78）+ §Phase 19（line 145-206）+ §Phase 19 Final Status（line 513+）—— phase 18/19 scope / base HEAD / deviations / final status 的权威记录
- `.planning/ROADMAP.md` — 各 phase 的 `**Goal:**`（retro-verify 的 goal-backward 起点）+ Progress 表

### Retro-verify 证据源
- **Phase 17**（目录齐全，`.planning/phases/17-56-p0-p1-p2/`）：`17-CONTEXT.md`（must_haves: D-01.4/D-03/D-04）、`17-01..04-SUMMARY.md`、`17-REVIEWS.md`、`17-01..04-PLAN.md`
- **Phase 18**（目录缺失）：根目录 `18-SUMMARY.md`（21.7KB）、commit `5d536ec fix(huawei/SEC-003b)`（`cmd/server/app_test.go` + `internal/huawei/manager.go`）、STATE.md §Phase 18、实时代码 `internal/huawei/manager.go` + `cmd/server/app_test.go`
- **Phase 19**（目录缺失）：根目录 `19-SUMMARY.md`（33KB）、`docs/audits/phase-19-D5-D21-summary.md`（8.5KB，D5-D21 逐项）、21+ commits（`refactor(19/d1)`..`refactor(19/d21)` + Wave 0-6）、STATE.md §Phase 19、实时代码（`internal/errors/*.go`、`internal/services/*.go`、`internal/middleware/*.go`）

### auth:57 修复起点
- `internal/handlers/auth_handler.go:50-62` — 当前 `if HandleError { return }; return` 模式（line 57）
- `internal/handlers/auth_handler_test.go` — 既有表驱动测试（5 类错误），回归保护
- `pkg/response/response.go` `HandleError(c, err) bool` — 始终返回 true（行为等价性依据）
- `v1.1-MILESTONE-AUDIT.md` §Cross-Phase Integration / Findings — WARNING 条目与 fix 建议

### REQUIREMENTS.md 来源
- `docs/audits/2026-07-30-backend-code-review.md` — phase 17 的 56 个 finding（REQ-17-* 来源，**不可修改**）
- `.planning/phases/20-handleerror-classify-convergence/20-CONTEXT.md` §D-22 候选 + R-3/R-4/R-5/R-7 user-locked decisions
- `.planning/phases/20-*/20-01-PLAN.md` + `20-05-PLAN.md` frontmatter — REQ-20a/b/c 已有定义
- `.planning/phases/20-*/20-VERIFICATION.md`（passed, 10/10）—— phase 20 交付物回填依据

</canonical_refs>

<specifics>
## Specific Ideas

### Retro-verify VERIFICATION.md 骨架（每 phase 一份）
```markdown
# Phase NN Verification
**Phase:** NN — <name>
**Verified:** 2026-08-03 (retro-active)
**Method:** goal-backward, evidence = SUMMARY + git history + live code
**status:** passed | partial

## Goal (from ROADMAP)
<原 ROADMAP Goal 原文>

## Must-Haves (from CONTEXT / audit)
- [x] <must_have> — 证据: <commit | test | grep 结果>
- ...

## Evidence
- Live code: <grep/build/test 结果>
- Git: <commit hash + 文件>
- SUMMARY: <18/19-SUMMARY.md 段落>

## Evidence Limitations (诚实)
- 原 PLAN.md / CONTEXT.md 未入库，验证基于 SUMMARY + commit + 代码，非原始计划
- <phase 18/19 特有局限>

## Deferred (confirmed not done, by design)
- <deferred 项 + 去向>
```

### auth_handler.go:57 diff 预览
```diff
-		if response.HandleError(c, err) {
-			return
-		}
-		// 兜底：unknown error（response.HandleError 已写 500）。
-		return
+		response.HandleError(c, err)
+		return
```

### REQUIREMENTS.md 表骨架
```markdown
| REQ-ID | Phase | 来源 | 状态 | 验证证据 |
|--------|-------|------|------|----------|
| REQ-17-SEC-001 | 17 | audit §6.2 | done | 17-VERIFICATION + commit <h> |
| REQ-17-SEC-003b | 17→18 | audit / CONTEXT D-deferred | done (phase 18) | 18-VERIFICATION + 5d536ec |
| REQ-17-STYLE-001 | 17 | audit | partial (handler 层 done, 全库 deferred) | 17-VERIFICATION |
| REQ-18-001 | 18 | 18-SUMMARY | done | 18-VERIFICATION |
| REQ-19-ctx | 19 | 19-SUMMARY | done | 19-VERIFICATION |
| REQ-20a-classify | 20 | 20-01-PLAN | done | 20-VERIFICATION (10/10) |
```

</specifics>

<deferred>
## Deferred Ideas

### 明确不在本阶段
- **Nyquist VALIDATION.md 补齐**（phase 16/17 缺、phase 20 draft）—— 独立 phase，本阶段只补 VERIFICATION.md（goal-backward），不补 VALIDATION.md（Nyquist dimension）
- **Phase 16 归属裁定**（v1.1 表内外歧义）—— 留待 milestone 决策，REQUIREMENTS.md 仅记观察
- **审计 tech_debt 10+ 项**（bare zap.Error 31 处、ppt_handler 9 处 err.Error() 泄漏、auth_handler 另 3 处、STYLE-001 全库、PERF-003 全库、STYLE-009 包名、typed error kind 字段等）—— 各为后续独立 phase
- **重跑 `/gsd:audit-milestone v1.1`** —— phase 21 交付后的验收动作，非本阶段交付物
- **v1.0（phase 01-14）REQ-ID 追溯** —— v1.0 已 shipped 归档，如需另开 phase

### 数据局限（非 deferred，是约束）
- Phase 18/19 原 PLAN.md / CONTEXT.md / DISCUSSION-LOG 永久丢失（从未 git 入库），VERIFICATION.md 只能基于 SUMMARY + commit + 代码 —— 这不是可补救的 gap，是既定事实

</deferred>

---

*Phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem*
*Context gathered: 2026-08-03 via orchestrator synthesis from v1.1-MILESTONE-AUDIT.md*
</content>
</invoke>
