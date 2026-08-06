---
gsd_state_version: 1.0
milestone: v2.0
milestone_name: 智能录制收尾（Smart Recording End）
status: in_progress
last_updated: "2026-08-06T09:40:00.000Z"
last_activity: 2026-08-06
progress:
  total_phases: 4
  completed_phases: 2
  total_plans: 14
  completed_plans: 10
  percent: 71
---

# STATE.md - Project Memory

**Project:** Record V2
**Milestone:** v2.0 - 智能录制收尾（Smart Recording End）
**Last Updated:** 2026-08-06
**Last Activity:** 2026-08-06

---

## Project Reference

### Core Value

会议视频从录制到PPT的一站式处理，让会议内容可检索、可回顾、可分享。

### What This Is

视频会议录制管理系统 V2.0，专为华为会议终端设计的自动化录制、管理、转录和PPT生成平台。支持自动录制华为会议、USB设备录制、RTSP流录制，提供视频多点分割、阿里通义听悟AI转录、PPT自动提取等能力。面向企业内部用户，通过Web界面和API进行管理。

### Current Focus

v2.0 智能录制收尾（Smart Recording End）— Phase 25 计划 4 plans, plan 01 已执行（ActivitySnapshot 2 字段扩 + service 单入口）。下一步：执行 plan 02 (scheduler select 三路)。

**v2.0 Goal:** 让华为会议录制时长智能贴合会议真实时长——到点未结束自动延时（30min × 4 = 2h 上限），提前结束由 TE40 `WEB_GetMailboxDataAPI`（`confState=="" && joinSum==0`）主信号 + silencedetect + 文件停滞双兜底任一触发即收尾转码，无需人工干预。

### v2.0 PRD 来源

`docs/plans/2026-08-05-smart-meeting-recording-end-design.md`（v2：纳入 TE40 邮箱 API 主信号）

### v2.0 Phase 结构（已批准 A/B/C 三段）

| Phase | 名称 | Goal | REQ-IDs | Plans |
|-------|------|------|---------|-------|
| 23 | 华为 API 扩展 + GORM 字段 + sentinel 错误码 | 落地 H 信号数据通路与可观测基线 | DETECT-01/04, AUDIT-01/05, CFG-01/02 (6) | 5/5 |
| 24 | ActivityWatcher + silencedetect + 文件停滞 | 整合 H+A+B + 多级降级 | DETECT-02/03, WATCH-01..05, EXTEND-03 (8) | 3/4 |
| 25 | scheduler 多信号驱动 + service 封装 + E2E + CI | 端到端闭环 + 余项 | SCHED-01..04, EXTEND-01/02, EARLY-01..04, AUDIT-02/03/04, CFG-03/04, OBS-01..05 (20) | 1/4 |
| 26 | 华为终端 TLS 私有 CA 加载 (SEC-003a hotfix) | 私有 CA 加载到 RootCAs | SEC-003a-01..05 (5) | 1/1 |

---

## Current Position

Phase: 25
Plan: 01 (ActivitySnapshot + service entry points) — COMPLETED
Plan 02: scheduler select 三路 + multi-input merge + CFG-03 双守门 — next
Status: In Progress
Last activity: 2026-08-06 -- Phase 25 plan 01 executed (3 commits: f8f8f6b + e527ecb + fbae631)

> 注：HANDOFF.json + .continue-here.md (2026-08-05 旧版 design-v2-review pause) 已删除。
> 实际进度：Phase 23 验证通过 (5/5)；Phase 24 plans 1-3 done；24-04 待执行。
> Phase 25 plan 01 (AUDIT-02/03/04 + EXTEND-01 + EARLY-01/02) 已执行：
> - ActivitySnapshot 2 字段扩 (FileSizeBytes + FileGrowthBps) + fileTicker 维护 lastFileGrowthBps
> - VideoRecordingTaskService 增 auditSvc + cfg 字段 + SetAuditService/SetConfig setters
> - UpdateTaskExtension (含 cfg.SmartEnd.MaxExtendCount 守门 wrapped ErrRecordingSmartExtend) + MarkTaskEndedEarly 单入口
> - SmartEndSnapshot JSON 序列化结构 (8 字段) 供 audit log 使用
> - 5 个新单元测试 + 1 个 audit snapshot 测试均通过
> Phase 26 (TLS CA SEC-003a hotfix) 26-01 已执行完毕，4 commits（3 feat + 1 docs）。
> SEC-003a-01..05 全部 satisfied：atomic SetCABundle + caCertPool RootCAs + cmd/server fail-closed 启动 + CABundleFile + BindEnv + 5-scenario 回归测试。

---

## Performance Metrics

| Phase | Duration | Tasks | Files |
|-------|----------|-------|-------|
| Phase 25 P01 | 25min | 2 tasks | 4 files |
| Phase 26 P01 | 27min | 3 tasks | 7 files |

### Requirements Coverage

- Total v2.0 requirements: 34 (DETECT 4 + WATCH 5 + SCHED 4 + EXTEND 3 + EARLY 4 + AUDIT 5 + CFG 4 + OBS 5)
- Mapped to phases: 34 (100%)
- Unmapped: 0
- Phase 23: 6 reqs / Phase 24: 8 reqs / Phase 25: 20 reqs

### Milestone Velocity

- v1.0 = 69 plans / 19 days = 3.6 plans/day
- v1.1 = 25 plans / 5 days = 5 plans/day
- v2.0 = 0 plans / 0 days (planning)

### v1.1 Audit Status (archived 2026-08-03)

- status: tech_debt (gaps 全空, 60/60 reqs, 5/5 phases, 5/5 integration, 4/4 flows)
- 5/5 phase VERIFICATION passed

---

## Accumulated Context

### Key Architectural Decisions (v1.1 carry-over)

1. **All Go Implementation** (No Python microservice) — 单进程部署，运维简单 ✓
2. **Aliyun OSS for File Relay** — 无公网 IP，Tingwu 需公网 URL ✓
3. **SM4-GCM envelope 凭据加密（SEC-003b，v1.1 Phase 18）** — fail-closed 启动 10 步不变量 ✓
4. **HandleError 全量收敛 + sentinel 体系（v1.1 Phase 20）** — 9 handler 收敛 + 42 sentinels + 自动 docs/errors.md + CI sync-check 门禁（`.github/workflows/ci.yml:44-51`）✓
6. **AutoMigrate 列表同步（v1.1）** — 走 AutoMigrate 不进 dormant `runCustomMigrations` ✓
7. **ctx 全量级联（v1.1 Phase 19）** — 403 处 GORM + ~190 service 方法 ✓
8. **jti replay 防御（v1.1 Phase 19）** — TTL sweeper，不加 DB 表（单实例 5min 窗口可接受）

### Tech Debt Deferred (v1.1 close)

- STYLE-001 全库 %w 迁移（~117 errors.New + ~474 fmt.Errorf）— deferred
- STYLE-009 Get* rename（124 处）— deferred
- KMS/Vault 自动注入凭据 — deferred
- 真实生产数据 post-audit — deferred
- jti replay 多实例需 Redis — deferred

### v2.0 Scope Decisions (locked by commit `506904a`)

1. **3-phase A/B/C split** — A: 基础设施 / B: watcher / C: E2E + 余项
2. **不改 `VideoRecordingTaskStatus` 枚举** — 沿用 `completed / failed / canceled`
3. **不做"会议提前结束预测"** — 无 ML/ASR
4. **不重写 `monitorProcessWithKey`** — 保留断流重连逻辑（仅在 watcher 侧处理重连期间状态，见 WATCH-05）
5. **不动前端 UI** — 仅后端 + config
6. **不接 `MSG_CONF_STATE_CHANGE` 推送通道** — 30s 轮询已足够
7. **默认 `extend_step_min=30`，`max_extend_count=4`** — 2h 总上限

### Project Constraints (carry-over)

- `.planning/` gitignored — commit 需 `git add -f`
- 4 个 commit 拆分（debug 改动与 Phase 工作分提交）— `commit-boundary-separation.md`
- 本机 transparent HTTPS MITM — local-repo `.git/config` http.sslVerify=false
- golangci-lint v2.12.2+（go1.25 要求）+ action v7+

### Future Requirements（v2.0 不做，下个 milestone 候选）

- FUTURE-01: MSG_CONF_STATE_CHANGE 推送接入（需项目具备消息推送基础设施）
- FUTURE-02: TE40 T.140 字幕信号（避免纯静音会议被 A+B 误杀）
- FUTURE-03: 跨 input 一致性（多 input 任务软结束）
- FUTURE-04: 前端任务详情页显示 ExtensionCount / EndedEarlyReason / EndedByHuaWeAPI
- FUTURE-05: 机器学习预测（用历史会议数据训练提前结束概率模型）

---

## Decisions Log

### 2026-08-06 - Phase 25 Plan 01 ActivitySnapshot + service entry points executed

**Decision:** ActivitySnapshot 2 字段扩 (FileSizeBytes / FileGrowthBps) + VideoRecordingTaskService 增 UpdateTaskExtension / MarkTaskEndedEarly 单入口 + SetAuditService / SetConfig setter 注入 + SmartEndSnapshot JSON 序列化结构。
**Rationale:** Phase 25 scheduler 多信号驱动需要 (1) audit log 含 snapshot 6 字段 (AUDIT-02/03)，其中 `file_size_bytes` / `file_growth_bps` 此前 ActivitySnapshot 未暴露 — Phase 24 RESEARCH.md Pitfall 8 已标注；(2) 收敛 smart-end 写入入口到 service 层 (AUDIT-04)，避免 scheduler 直 GORM 散落调用导致 audit log 漏写。Setter 注入 (vs 增 variadic) 保留 Phase 19 D2 encryptor 变参兼容性,deps 渐进注入不破坏既有调用点。
**Outcome:** 2 feat commits (f8f8f6b / e527ecb) + 1 docs commit (fbae631). 5 个新单元测试 (TestUpdateTaskExtension_Exists / _AuditSnapshot / _MaxLimit / TestMarkTaskEndedEarly_HuaweiSignal / _BothSilenceAndStall) + 原 5 个 Phase 24 recorder 测仍绿。`MaxExtendCount` 守门 wrapped ErrRecordingSmartExtend,errors.Is 可达 — scheduler 据此走 EXTEND-02 max_extend_reached 路径。5 deviations auto-fixed (1 JSON omitempty bug, 1 audit Stop race, 3 test infra bugs)。

### 2026-08-06 - Phase 26 Plan 01 SEC-003a hotfix executed

**Decision:** Atomic PEM bundle publish under Manager.mu + caCertPool *x509.CertPool parameter on NewHTTPClient + fail-closed startup wiring in cmd/server/app.go + HUAWEI_CA_BUNDLE_FILE env override.
**Rationale:** x509: certificate signed by unknown authority on `https://10.62.10.3` (Huawei TE40 private CA) blocked the entire v2.0 DETECT-01 mailbox polling chain. Phase 17 TLS hardening requires InsecureSkipVerify=false + chain validation, so the only safe option is to load the private CA bundle into a *x509.CertPool and pass it via tls.Config.RootCAs. Atomic-only publish prevents partial-trust states; logger.Fatal at startup makes misconfiguration loud rather than silent.
**Outcome:** 3 feat commits (c8ef568 / f311e86 / c2357d7) + 1 docs commit (7818db5). 5-scenario regression suite covers ValidPEM / InvalidOrMissing (6 subtests) / EmptyPath / CertPoolBranches (pointer-identity) / ServerAndRootChain. All Phase 17 invariants preserved (MinVersion=tls.VersionTLS12, no 3DES, InsecureSkipVerify=false, ForceAttemptHTTP2=false).

### 2026-08-06 - v2.0 Roadmap 创建

**Decision:** 3-phase split（A 基础设施 → B watcher → C E2E + 余项），Phase 23/24/25。34 REQ-IDs 全映射，0 orphan。
**Rationale:** PRD §9 建议 11 步任务按 A/B/C 三段切分，A 段落地数据通路与可观测基线（DETECT 字段 + AUDIT 表 + CFG config + AUDIT sentinel），B 段整合 H+A+B 三类信号（DETECT 信号采集 + EXTEND-03 默认值），C 段完成 scheduler 多信号驱动 + service 封装 + 端到端闭环 + 余项 AUDIT/CFG/OBS。
**Outcome:** ROADMAP.md 写入 v2.0 sections（Phases 23-25 详情 + Progress 表 + Backlog 保留 Phase 15/16/字幕候选）；STATE.md 更新 v2.0 进度；REQUIREMENTS.md Traceability 表填充 34 行。

### 2026-08-03 - Phase 22 审计 tech debt 收尾 (v1.1 归档)

详见 v1.1-MILESTONE-AUDIT-REAUDIT.md。6 plans 全部完成，errors.md footer=16（CI SYNC_OK）。

### 2026-08-03 - Phase 19-22 v1.1 收尾完成

25 plans 落地（247 commits, 267 files, +35261/-5976 LOC），shipped 2026-08-03。详见 `.planning/milestones/v1.1-ROADMAP.md`。

---

## Session Continuity

### Last Session

2026-08-06 — Phase 25 plan 01 executed (3 commits landed: f8f8f6b + e527ecb + fbae631). ActivitySnapshot FileSizeBytes/FileGrowthBps 扩展 + VideoRecordingTaskService UpdateTaskExtension / MarkTaskEndedEarly 单入口 + audit 注入 + 5 个新单元测试。6/6 must_haves satisfied, 5 deviations auto-fixed.

### Next Steps

1. `/gsd:execute-phase 25` — 执行 25-02 plan (scheduler monitorTask select 三路 + WatcherChannels 多 input merge + CFG-03 双守门)
2. `/gsd:plan-phase 25` 后续：25-03 (OBS 5 类日志 + smart_end_metrics.go atomic) + 25-04 (Nyquist E2E 7+ 场景 + audit golden JSON)
3. 24-04 Nyquist 仍待执行 — 与 25-02 可并行（24-04 不依赖 25 反之亦然）

### Open Questions

无（PRD `docs/plans/2026-08-05-smart-meeting-recording-end-design.md` v2 已明确全部 34 REQ-ID）

### Blockers

无

---

*STATE.md updated: 2026-08-06 — Phase 25 plan 01 executed (3 commits landed), plan 02 next*

**Active Phase:** 25 (4 plans, 1/4 done)
