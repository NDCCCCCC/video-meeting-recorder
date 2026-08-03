---
phase: 21-close-v1-1-gaps-retro-verify-phases-17-18-19-create-requirem
verified: 2026-08-03T10:30:00Z
status: passed
score: 4/4 must-haves verified (D-05.1..D-05.4)
overrides_applied: 0
overrides: []
gaps: []
deferred:
  - item: "Nyquist VALIDATION.md 补齐 (phase 16/17 无 VALIDATION.md, phase 20 仍 nyquist_compliant: false draft)"
    reason: "21-CONTEXT `<deferred>` 显式排除 — 独立关注点, phase 21 标题未涉及, 列入独立 phase"
    delivered_by: "future phase (非本阶段交付)"
  - item: "Phase 16 归属裁定 (v1.1 ROADMAP 表内外歧义)"
    reason: "21-CONTEXT `<deferred>` + D-03.4 — 留待 milestone 决策, REQUIREMENTS.md 仅记观察不裁定"
    delivered_by: "milestone decision (非 phase 21 范围)"
  - item: "审计 tech_debt 10+ 项 (bare zap.Error 31 处 / ppt_handler 9 处 err.Error() 泄漏 / auth_handler 另 3 处 RefreshToken:90 + ChangePassword:179 + LogoutAll:142 / STYLE-001 全库 / PERF-003 全库 / STYLE-009 包名 rename 等)"
    reason: "21-CONTEXT D-01.3 显式拒绝任何顺手清理 — 各为后续独立 phase"
    delivered_by: "future phases (out of v1.1 scope)"
  - item: "重跑 /gsd:audit-milestone v1.1"
    reason: "21-CONTEXT `<deferred>` — phase 21 验收之后的动作, 非本阶段交付 (但其通过条件是本阶段成功判据)"
    delivered_by: "milestone re-audit step"
human_verification: []
---

# Phase 21: Close v1.1 gaps — retro-verify 17/18/19 + REQUIREMENTS.md + auth:57 fix - Verification Report

**Phase Goal (from ROADMAP.md):** 关闭 v1.1 里程碑审计 (`v1.1-MILESTONE-AUDIT.md` gaps_found) 发现的 5 项过程缺口 (REQUIREMENTS.md 缺失 + phase 17/18/19 未验证 + auth_handler.go:57 WARNING), 使里程碑可在重审时诚实归档为 passed. 代码库本身已功能完整 (`go test -race ./...` 全绿), 本阶段不改业务功能 (仅 1 行 auth_handler.go 规范化), 只补齐过程产物.

**Verified:** 2026-08-03T10:30:00Z
**Status:** passed (4/4 must-haves verified)
**Verifier:** Claude (gsd-verifier) — goal-backward, LIVE filesystem + git evidence

> 本验证基于 LIVE 文件系统 + git history + `go build`/`go test` 重跑, 不依赖 SUMMARY.md 的自述. 每条 must-have 与每项 gap 都有独立证据指针.

## Goal Achievement

### Observable Truths (Must-Haves D-05.1..D-05.4)

| # | Truth | Status | Evidence (LIVE) |
|---|-------|--------|-----------------|
| D-05.1 | 3 个 retro-VERIFICATION.md 各含 `status: passed`, must_haves 全部有证据指向 | ✓ VERIFIED | `17-VERIFICATION.md` frontmatter `status: passed` + `score: 7/7` + 8 wave commit hash 全引用 (4d3de0b/2bcee29/47ef805/4fc1d3c/d27903f/9150e95/4f5579a/c04f805); `18-VERIFICATION.md` `status: passed` + `score: 11/11` + 39 VERIFIED 标记 + 9 wave commits (W1a..W4d) 全引用 + 5d536ec 显式标预存证据 (12 处提及); `19-VERIFICATION.md` `status: passed` + `score: 10/10` + 41 VERIFIED 标记 + 11 wave + 17 D5-D21 commits 全引用. 三文件均含 `method: retro-active goal-backward` + `## Evidence Limitations` + `## Deferred (confirmed not done, by design)` 章节. |
| D-05.2 | `.planning/REQUIREMENTS.md` 存在, v1.1 四 phase 全覆盖, 无 REQ-ID orphan (除显式 deferred) | ✓ VERIFIED | 文件存在 251 行 / 28312 bytes; 5 列表头 `\| REQ-ID \| Phase \| 来源 \| 状态 \| 验证证据 \|` 完整; 行计数 REQ-17-* = 52 + REQ-18-* = 5 + REQ-19-* = 4 + REQ-20-* = 11 (共 72 行, ~80 含跨 phase 兑现注解); 11 个 REQ-20a/b/c ID 全部 grep 命中; 跨 phase 兑现项 REQ-17-SEC-003b (2 处) + REQ-17-PERF-003 (3 处) 显式标 "delivered by Phase 18/19"; `## Coverage` 段含 4 条 grep orphan 检测规则 + Orphans: 0; `## Out-of-scope observation` 段记录 phase 16 归属歧义含 "不强行裁定" 字样; `## Canonical References` 段登记 root 18/19-SUMMARY.md 路径. 无 TBD/FIXME/XXX. |
| D-05.3 | `auth_handler.go:57` 为规范模式; `go build ./...` + `go test -race ./internal/handlers/...` 全绿; 既有 auth 测试不回归 | ✓ VERIFIED | LIVE `sed -n '50,65p' internal/handlers/auth_handler.go` 确认 line 57 `response.HandleError(c, err)` + line 58 `return` (无 `if` 包裹, 无 "兜底" 注释); line 53-56 mapping.go 行为注释保留 (含 "Phase 20 (20-02)" + "R-3 要求" + "R-4"); LIVE `go build ./...` exit=0; LIVE `go test -race -count=1 ./internal/handlers/...` ok 3.356s exit=0; LIVE `go test -run TestLogin_HandleError_ClassifyDrop` 10 sub-tests 全部 PASS (ErrADUserNotRegistered→403 / wrapped→403 / ErrADAccountNotFound→404 / ErrUserDisabled→403 / ErrADConfigError→503 / ErrADUnreachable→503 / ErrUnauthorized→401 / wrapped→401 / BusinessError→400 / unknown→500). 其他 3 处 tech_debt 保留 (line 90 RefreshToken + line 142 LogoutAll + line 179 ChangePassword, 因 -3 行偏移与 CONTEXT D-04.4 原 line 号 93/182 不同但实质同一). |
| D-05.4 | `v1.1-MILESTONE-AUDIT.md` 的 5 项 gap 全部可标记关闭 | ✓ VERIFIED | 见下 `## 5-Gap Closure Table` — 5/5 CLOSED. 重跑 `/gsd:audit-milestone v1.1` 应由 gaps_found → passed (重跑本身不在本阶段, 但产物支撑该结论). |

**Score:** 4/4 truths verified (no functional gap)

### Required Artifacts

| Artifact | Expected | Status | Details (L1 Existence / L2 Substantive / L3 Wired) |
|----------|----------|--------|----------------------------------------------------|
| `.planning/REQUIREMENTS.md` | v1.1 4-phase REQ-ID 追溯表 | VERIFIED | L1 存在 251 行; L2 5 列表结构 + 4 phase 子段 + Coverage/Out-of-scope/Canonical References 三段齐全, REQ-17/18/19/20 全覆盖; L3 验证证据列引用 17/18/19/20-VERIFICATION.md 全部 4 个路径 + commit hash. |
| `.planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md` | phase 17 retro-verify 报告 | VERIFIED | L1 存在 40152 bytes / 180 行 (≥80 阈值); L2 frontmatter status: passed + 7 Observable Truths (M1-M7) 全 VERIFIED + Required Artifacts + Key Link + Behavioral Spot-Checks + Evidence Limitations + Deferred 6 行表格; L3 引用 8 个 wave commit hash + 跨 phase 兑现去向 (SEC-003b→18 / PERF-003→19 / HMAC jti→19 D3). |
| `.planning/phases/18-credential-static-encryption-sec-003b/18-VERIFICATION.md` | phase 18 retro-verify 报告 | VERIFIED | L1 存在 46129 bytes / 174 行 (≥100 阈值); L2 status: passed 11/11 + 39 VERIFIED + Evidence Limitations + Deferred; L3 L3 wiring two-hop 验证: `input_config_service.go:48 s.encryptor.Encrypt → credential_encryptor.go:92 utils.EncryptGCM`; 9 wave commits + 5d536ec 预存证据标注. |
| `.planning/phases/18-credential-static-encryption-sec-003b/` 目录 | 重建的最小目录 | VERIFIED | 3 文件齐全: `.gitkeep` (0 bytes 占位) + `18-SUMMARY.md` (21704 bytes, `diff -q` 与 root 原版 IDENTICAL) + `18-VERIFICATION.md`. Root `18-SUMMARY.md` 保持原位 (未移动). |
| `.planning/phases/19-ctx-cascade-sec-004-style-001-error/19-VERIFICATION.md` | phase 19 retro-verify 报告 | VERIFIED | L1 存在 65323 bytes / 大段 (≥120 阈值); L2 status: passed 10/10 + 41 VERIFIED + Evidence Limitations + Deferred; L3 L3 wiring: `grep -c '.WithContext(ctx)' internal/services/video_recording_task_service.go` = 42 + `cmd/server/app.go:340 &models.HLSJtiRecord{}` AutoMigrate 注册; 11 wave + 17 D5-D21 commits 引用. |
| `.planning/phases/19-ctx-cascade-sec-004-style-001-error/` 目录 | 重建的最小目录 | VERIFIED | 4 文件齐全: `.gitkeep` + `19-SUMMARY.md` (33058 bytes, IDENTICAL to root) + `phase-19-D5-D21-summary.md` (8557 bytes, IDENTICAL to docs/audits/) + `19-VERIFICATION.md`. Root `19-SUMMARY.md` + `docs/audits/phase-19-D5-D21-summary.md` 保持原位. |
| `internal/handlers/auth_handler.go` | Login handler canonical HandleError 模式 | VERIFIED | L1 文件存在; L2 line 57 `response.HandleError(c, err)` + line 58 `return` (无 `if` 包裹, grep 命中 1 处规范模式 + 0 处 `if response.HandleError(c, err)` 在 Login 范围); L3 紧邻 `return` 实际退出 Login handler, 无第二条 GinError 写入路径. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| 5 phase 21 plan commits | main branch | git log a93f98b..HEAD | VERIFIED | 13 commits 落地: 2c679f2 (17-VERIF) + d76d47d (18-VERIF + dir) + 4b52463 (19-VERIF + dir) + 695b4fe (REQUIREMENTS) + 4959e9c (auth:57 fix) + 5 plan-completion SUMMARY commits + 3 STATE/ROADMAP/review bookkeeping commits. |
| auth:57 fix commit message body | 行为等价性论据 | 含 "Behavior-equivalent" + "IsKnownError" 字样, 不含 "always returns true" | VERIFIED | `git log -1 --format=%B 4959e9c` 显示 commit body 用控制流论据 (per CONTEXT D-04.2 更正 + 21-RESEARCH §6), 非事实错误的 "always returns true" 论据. |
| 18-VERIFICATION D18-5 (encrypt-on-write) | utils.EncryptGCM (SM4-GCM 原语) | 两跳 wiring: input_config_service.go:48 → CredentialEncryptor.Encrypt → credential_encryptor.go:92 utils.EncryptGCM | VERIFIED | 18-VERIFICATION 含 grep `s\.encryptor\.\(Encrypt\|Decrypt\)` = 2 + grep `utils\.EncryptGCM` in credential_encryptor.go = 1 (line 92) 双重 L3 证据. |
| 19-VERIFICATION D19-1 (ctx 全量级联) | video_recording_task_service.go (42 WithContext sites) | grep `.WithContext(ctx)` count | VERIFIED | LIVE 重跑命令 (本 verifier 执行): `grep -c ".WithContext(ctx)" internal/services/video_recording_task_service.go` = 42, 与 19-VERIFICATION 声明一致. |
| 19-VERIFICATION D19-6 (hls_jti_records 表) | cmd/server/app.go:340 AutoMigrate 注册 | grep `HLSJtiRecord` | VERIFIED | 17-VERIFICATION Behavioral Spot-Checks 含 `grep -n "HLSJtiRecord" cmd/server/app.go` 命中 line 340, 跨 phase 契约 (phase 17 HMAC jti deferred → phase 19 D3 兑现) 显式登记. |
| REQUIREMENTS.md REQ-17-SEC-003b 行 | 18-VERIFICATION D18-1..D18-5 | 验证证据列引用 VERIFICATION 路径 | VERIFIED | REQUIREMENTS.md 第 38 行 `REQ-17-SEC-003b` 状态列 "done (delivered by Phase 18)" + 验证证据列引用 `18-VERIFICATION D18-1..D18-5 + commits e6315ce + 5d536ec`. |
| REQUIREMENTS.md REQ-20a-classify 行 | 20-VERIFICATION (passed 10/10) | 验证证据列 | VERIFIED | REQUIREMENTS.md 第 132 行验证证据列引用 `20-VERIFICATION (10/10) — 9 文件 27 处 ad-hoc classify 全替换`. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|-------------------|--------|
| `auth_handler.go:57` HandleError 调用 | err (5 类 sentinel/BusinessError + unknown) | ShouldBindJSON 失败 + authService.Login error | Yes (TestLogin_HandleError_ClassifyDrop 10 sub-tests LIVE PASS 验证 403/404/503/401/400/500 全状态码) | VERIFIED |
| 18-VERIFICATION D18-1 SM4-GCM envelope | nonce_12B + ciphertext + tag_16B | utils.EncryptGCM (sm4_password.go:254) → CredentialEncryptor.Encrypt → input_config_service:48 写入 | Yes (LIVE grep 三层调用链完整) | VERIFIED |
| 19-VERIFICATION D19-1 ctx 级联 | ctx context.Context | scheduler → VideoRecordingTaskService 22 方法 + VideoFileService 23 方法 + 13 leaf 服务 | Yes (LIVE grep video_recording_task_service.go = 42 WithContext sites) | VERIFIED |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| go build 全绿 | `go build ./...` | exit=0 | ✓ PASS |
| handlers race test 全绿 | `go test -race -count=1 ./internal/handlers/...` | ok 3.356s exit=0 | ✓ PASS |
| TestLogin_HandleError_ClassifyDrop 10 sub-tests | `go test -run TestLogin_HandleError_ClassifyDrop -v ./internal/handlers/...` | 10/10 PASS (覆盖 5 类错误 × wrapped/unwrapped + R-3/R-4) | ✓ PASS |
| auth:57 规范模式无 if 包裹 | `grep -n 'response.HandleError(c, err)' internal/handlers/auth_handler.go` | line 57 命中 (canonical 模式), Login 范围内 `if response.HandleError` 命中 0 处 | ✓ PASS |
| "兜底" 注释已删 | `grep -n '兜底' internal/handlers/auth_handler.go` | 0 命中 | ✓ PASS |
| 18-SUMMARY.md 副本与 root 一致 | `diff -q 18-SUMMARY.md .planning/phases/18-credential-static-encryption-sec-003b/18-SUMMARY.md` | IDENTICAL | ✓ PASS |
| 19-SUMMARY.md + D5-D21 副本与 root 一致 | `diff -q 19-SUMMARY.md ...` + `diff -q docs/audits/phase-19-D5-D21-summary.md ...` | IDENTICAL (两个) | ✓ PASS |
| REQUIREMENTS.md 11 REQ-20 IDs 全覆盖 | `for id in REQ-20a-classify ... REQ-20-typed-kind; do grep -q "$id" ...; done` | 11/11 OK | ✓ PASS |
| phase 21 commits 落地 main | `git log --oneline a93f98b..HEAD` | 13 commits (5 plan-commits + 5 plan-complete commits + 3 bookkeeping) | ✓ PASS |
| REQUIREMENTS.md 无 debt marker | `grep -E 'TBD\|FIXME\|XXX' .planning/REQUIREMENTS.md` | 0 命中 | ✓ PASS |

### Probe Execution

Not applicable — phase 21 is process-gap closure (docs + 1-line code normalization). No `scripts/*/tests/probe-*.sh` declared in PLAN or convention for this phase type. The role-equivalent runnable checks are the Behavioral Spot-Checks above (go build / go test -race / grep patterns), all PASS.

## 5-Gap Closure Table

源 spec: `.planning/v1.1-MILESTONE-AUDIT.md` (status: gaps_found, immutable).

| # | Audit Gap | Deliverable | LIVE Evidence | Closed? |
|---|-----------|-------------|---------------|---------|
| 1 | REQUIREMENTS.md missing — no REQ-ID traceability, orphan detection impossible | `.planning/REQUIREMENTS.md` (21-04) | 文件存在 251 行; 5 列表结构; ~80 REQ-IDs 全覆盖 (52 REQ-17 + 5 REQ-18 + 4 REQ-19 + 11 REQ-20); Orphans: 0; 4 条 grep orphan 检测规则; 跨 phase 兑现项显式标注 (SEC-003b→18 / PERF-003→19 / BUG-005→19 / HMAC-jti→19 D3); Out-of-scope phase 16 观察含 "不强行裁定"; Canonical References 段登记 root SUMMARY 路径. Commit `695b4fe`. | ✓ CLOSED |
| 2 | phase-17 unverified (blocker) — 4 SUMMARYs 存在但缺 VERIFICATION.md | `.planning/phases/17-56-p0-p1-p2/17-VERIFICATION.md` (21-01) | 文件存在 180 行; frontmatter status: passed + score 7/7 + method: retro-active; 7 Observable Truths (M1-M7) 全 VERIFIED 含 8 wave commit hash (4d3de0b..c04f805) + live grep targets; Evidence Limitations 段诚实; Deferred 段 6 行表格含跨 phase 去向. Commit `2c679f2`. | ✓ CLOSED |
| 3 | phase-18 missing-directory (blocker) — 目录被早期清理 | `.planning/phases/18-credential-static-encryption-sec-003b/` + 18-VERIFICATION.md (21-02) | 目录存在含 3 文件 (.gitkeep + 18-SUMMARY.md IDENTICAL 副本 + 18-VERIFICATION.md 174 行); status: passed 11/11; 39 VERIFIED 标记; 9 wave commits (W1a..W4d, body 为准) + 5d536ec 显式标预存证据; L3 两跳 wiring (input_config_service → CredentialEncryptor → utils.EncryptGCM); root 18-SUMMARY.md 保持原位. Commit `d76d47d`. | ✓ CLOSED |
| 4 | phase-19 missing-directory (blocker) — 目录被早期清理 | `.planning/phases/19-ctx-cascade-sec-004-style-001-error/` + 19-VERIFICATION.md (21-03) | 目录存在含 4 文件 (.gitkeep + 19-SUMMARY.md IDENTICAL 副本 + phase-19-D5-D21-summary.md IDENTICAL 副本 + 19-VERIFICATION.md); status: passed 10/10; 41 VERIFIED 标记; 11 wave + 17 D5-D21 commits 引用; L3 wiring (42 WithContext + HLSJtiRecord AutoMigrate); cacc294/6edb772 HEAD 不一致标注 body 为准; root 原版保持原位. Commit `4b52463`. | ✓ CLOSED |
| 5 | auth_handler.go:57 WARNING (latent CR-01 reintroduce risk) — `if HandleError { return }; return` 不规范 | `internal/handlers/auth_handler.go` 规范化 (21-05) | LIVE line 57 `response.HandleError(c, err)` + line 58 `return` (无 if 包裹, 无 "兜底" 注释); line 53-56 mapping.go 注释保留; `go build ./...` exit=0; `go test -race ./internal/handlers/...` ok 3.356s exit=0; TestLogin_HandleError_ClassifyDrop 10 sub-tests 全 PASS; 其他 3 处 tech_debt (line 90/142/179) 保留; 21-REVIEW.md status: clean 0 findings; commit body 用控制流论据 (Behavior-equivalent + IsKnownError, 非 "always returns true"). Commit `4959e9c`. | ✓ CLOSED |

**5/5 gaps CLOSED.** v1.1-MILESTONE-AUDIT.md 的 `gaps:` 段 5 项 (requirements + phase-17 + phase-18 + phase-19 + auth:57 WARNING) 全部有产物支撑重审由 gaps_found → passed.

## Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| P21-R1 | 21-01 | phase 17 retro-verify (关闭 phase-17 unverified gap) | SATISFIED | 17-VERIFICATION.md status: passed 7/7 + commit `2c679f2` |
| P21-R2 | 21-02 | phase 18 retro-verify + 目录重建 (关闭 phase-18 missing-directory gap) | SATISFIED | 18-VERIFICATION.md status: passed 11/11 + dir 重建 + commit `d76d47d` |
| P21-R3 | 21-03 | phase 19 retro-verify + 目录重建 (关闭 phase-19 missing-directory gap) | SATISFIED | 19-VERIFICATION.md status: passed 10/10 + dir 重建 + commit `4b52463` |
| P21-R4 | 21-04 | 创建 REQUIREMENTS.md (关闭 REQUIREMENTS.md missing gap) | SATISFIED | REQUIREMENTS.md 251 行 + ~80 REQ-IDs + commit `695b4fe` |
| P21-R5 | 21-05 | auth:57 规范化 (关闭唯一 WARNING) | SATISFIED | auth_handler.go:57 canonical + go build/test 全绿 + commit `4959e9c` |

P21-R1..R5 是从 21-CONTEXT.md D-01..D-05 派生的 requirement IDs (per phase goal footnote: "by design REQUIREMENTS.md is itself a deliverable that defines the v1.1 REQ-ID system"). 5/5 SATISFIED, 无 orphan.

## Anti-Patterns Found

无 BLOCKER / WARNING 级 anti-pattern.

| File | Line(s) | Pattern | Severity | Impact |
|------|---------|---------|----------|--------|
| `.planning/REQUIREMENTS.md` + 3 retro-VERIFICATION.md | — | (无) | — | LIVE `grep -E 'TBD\|FIXME\|XXX'` 全部 0 命中 |
| `internal/handlers/auth_handler.go` | 90, 142, 179, 250 | 4 处 `response.GinError(c, ..., err.Error())` raw leak (RefreshToken / LogoutAll / ChangePassword / VerifyTotp) | INFO | 显式 deferred per CONTEXT D-04.4 — 21-05 plan 显式拒绝顺手清理, 归后续独立 phase. 非 phase 21 引入, 非 regression. |
| `internal/handlers/auth_handler_test.go` | 124-126 | 测试注释引用旧 calling pattern `if response.HandleError(c, err) { return }`, refactor 后 stale (21-REVIEW §Observation 标注) | INFO | 不影响测试可靠性 (测试直接调 HandleError 本身, 不经 Login if 语句). 21-REVIEW 明确不作为 finding. 未来 minor doc cleanup. |

## Human Verification Required

无 — 本阶段为过程产物补齐 + 1 行代码规范化, 全部行为程序化验证:
- go build / go test -race LIVE 重跑通过
- 文件存在 + 内容正确性 + grep wiring 全程序化验证
- auth:57 行为等价性由控制流论据 + 21-REVIEW 独立验证 + 10 sub-tests 回归网保证, 无需人工

## Evidence Limitations

- **18/19 retro-verify 的固有约束**: phase 18/19 的原 PLAN.md / CONTEXT.md / DISCUSSION-LOG 永久丢失 (从未 git 入库) — 这是 21-CONTEXT `<deferred>` §"数据局限" 标注的既定事实, 不是 phase 21 可补救的 gap. 18/19-VERIFICATION.md 已在各自 `## Evidence Limitations` 段诚实说明验证基于 SUMMARY + commit + live code 三源交叉, 非基于原始计划文档.
- **5d536ec 是预存证据**: commit `5d536ec` (2026-08-02) 早于 phase 21 启动 (2026-08-03), 是 phase 21 启动前的预存证据 NOT phase 21 交付物 (per 21-RESEARCH §Pitfall 6). 18-VERIFICATION.md 已显式标注此点 (12 处提及).
- **未重跑 `/gsd:audit-milestone v1.1`**: phase 21 交付物的设计意图是支撑重审由 gaps_found → passed, 但重跑本身按 21-CONTEXT `<deferred>` 不属本阶段. 本 VERIFICATION 的 5-Gap Closure Table 已基于 LIVE 证据独立确认 5 项 gap 全部有产物关闭.
- **未跑全库 `go test -race ./...`**: 仅跑 `go test -race ./internal/handlers/...` (D-05.3 要求范围). phase 21 不改其他代码 (auth:57 是 1 行 control-flow normalization + 4 phase 的 VERIFICATION.md 是纯文档), 故全库 race test 不属 phase 21 验证必须项. v1.1-MILESTONE-AUDIT.md Executive Summary 已记录 "all 24 tested packages pass `go test -race ./...`" 作为基线.

## Gaps Summary

**No functional gaps found.** v1.1-MILESTONE-AUDIT.md 标记的 5 项过程缺口全部关闭:

1. REQUIREMENTS.md missing → `.planning/REQUIREMENTS.md` 创建 (251 行, ~80 REQ-IDs, orphan 0)
2. phase-17 unverified → 17-VERIFICATION.md 重建 (status: passed 7/7)
3. phase-18 missing-directory → 目录重建 + 18-VERIFICATION.md (status: passed 11/11)
4. phase-19 missing-directory → 目录重建 + 19-VERIFICATION.md (status: passed 10/10)
5. auth_handler.go:57 WARNING → 1 行规范化 (canonical 模式 + go build/test 全绿 + 21-REVIEW clean)

阶段目标 (关闭 5 项过程缺口, 使里程碑可重审诚实归档) **达成**. 代码库功能完整性未变 (phase 21 设计上不改业务功能, 仅 1 行 auth_handler.go 规范化 + 纯文档产物).

---

_Verified: 2026-08-03T10:30:00Z_
_Verifier: Claude (gsd-verifier)_
