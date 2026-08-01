# Roadmap: Record V2 - 视频切割与会议转录PPT

## Milestones

- ✅ **v1.0 视频切割与会议转录PPT** — Phases 01-14 (shipped 2026-05-06)
- 📋 **v1.1** — Planning next milestone

## Phases

<details>
<summary>✅ v1.0 视频切割与会议转录PPT (Phases 01-14) — SHIPPED 2026-05-06</summary>

- [x] Phase 01: 视频分割 (5/5 plans) — completed 2026-04-17
- [x] Phase 02: 本地转录 (4/4 plans) — completed 2026-04-17
- [x] Phase 03: PPT 管理 (2/2 plans) — completed 2026-04-18
- [x] Phase 04: 云服务 (5/5 plans) — completed 2026-04-20
- [x] Phase 05: 文件重命名 (2/2 plans) — completed 2026-04-20
- [x] Phase 06: PPT 编辑器 (9/9 plans) — completed 2026-04-20
- [x] Phase 07: 预览页面UI改进 (4/4 plans) — completed 2026-04-20
- [x] Phase 08: 视频快照播放器增强 (5/5 plans) — completed 2026-04-21
- [x] Phase 09: 多角色权限 (5/5 plans) — completed 2026-04-28
- [x] Phase 10: 管理员审计日志 (5/5 plans) — completed 2026-04-28
- [x] Phase 11: IP-IP 录制 (6/6 plans) — completed 2026-04-28
- [x] Phase 12: Windows AD 域控 (6/6 plans) — completed 2026-04-30
- [x] Phase 13: 重构华为配置，支持USB和流媒体 (6/6 plans) — completed 2026-04-29
- [x] Phase 14: 批量操作 (4/4 plans) — completed 2026-04-30

**Total:** 69 plans completed

**See** `.planning/milestones/v1.0-ROADMAP.md` **for full milestone details.**

</details>

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 01. 视频分割 | v1.0 | 5/5 | Complete | 2026-04-17 |
| 02. 本地转录 | v1.0 | 4/4 | Complete | 2026-04-17 |
| 03. PPT 管理 | v1.0 | 2/2 | Complete | 2026-04-18 |
| 04. 云服务 | v1.0 | 5/5 | Complete | 2026-04-20 |
| 05. 文件重命名 | v1.0 | 2/2 | Complete | 2026-04-20 |
| 06. PPT 编辑器 | v1.0 | 9/9 | Complete | 2026-04-20 |
| 07. 预览页面UI改进 | v1.0 | 4/4 | Complete | 2026-04-20 |
| 08. 视频快照播放器增强 | v1.0 | 5/5 | Complete | 2026-04-21 |
| 09. 多角色权限 | v1.0 | 5/5 | Complete | 2026-04-28 |
| 10. 管理员审计日志 | v1.0 | 5/5 | Complete | 2026-04-28 |
| 11. IP-IP 录制 | v1.0 | 6/6 | Complete | 2026-04-28 |
| 12. Windows AD 域控 | v1.0 | 6/6 | Complete | 2026-04-30 |
| 13. 重构华为配置，支持USB和流媒体 | v1.0 | 6/6 | Complete | 2026-04-29 |
| 14. 批量操作 | v1.0 | 4/4 | Complete | 2026-04-30 |
| 17. 后端代码审查 56 发现修复 P0/P1/P2 | v1.1 | 4/4 | Complete | 2026-07-30 |
| 18. 凭据静态加密 + 密钥轮换 (SEC-003b) | v1.1 | 1/1 | Complete | 2026-07-31 |
| 19. ctx 全量级联 + SEC-004 replay + STYLE-001 error | v1.1 | 4/4 | Complete | 2026-07-31 |
| 20. 错误处理统一收敛 + sentinel 体系增强 | v1.1 | 5/5 | Complete    | 2026-08-01 |

## Backlog

(No backlog items)

### Phase 1: 新功能：在视频播放中添加外挂字幕支持（预览视频、切割视频、预览PPT页面）

**Goal:** 在VideoPlayerModal、切割视频页面、PPTPreview三个视频播放场景中实现外挂字幕功能，使用WebVTT格式，字幕显示在视频下方独立区域，支持开关、字号调整和样式配置。
**Requirements**: D-01 through D-11 (from CONTEXT.md)
**Depends on:** None (standalone feature)
**Plans:** 3 plans

Plans:

- [ ] 01-01-PLAN.md -- Backend subtitle API endpoints + frontend type definitions
- [ ] 01-02-PLAN.md -- Subtitle sync hook + SubtitlePanel component with style controls
- [ ] 01-03-PLAN.md -- Integration into VideoPlayerModal and PPTPreview

### Phase 15: 前端去 AI 味

**Goal:** 消除 Record V2 前端 UI 中的"AI 生成感"——替换模板化文案、删除硬编码 mock、补齐空/错/加载态、统一设计令牌与品牌基线、加入克制的微交互，并以 Playwright 截图回归保障全量改的回归风险。
**Requirements**: D-01 through D-08 (from CONTEXT.md)
**Depends on:** Phase 14
**Plans:** 6/6 plans complete

Plans:

**Wave 1** *(foundation — no blockers)*

- [x] 15-01-PLAN.md -- Extend theme.ts (accent/surface/border/muted/elevation/motion/radius) + antd 6 flat motion token mapping in main.tsx + StatCards/global.css dehardcode

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 15-02-PLAN.md -- MotionProvider (LazyMotion + reducedMotion) + App.tsx route fade + dashboard m.div stagger
- [x] 15-03-PLAN.md -- 5 inline SVG illustrations + Files/Tasks/Audit empty/error states + useErrorHandler rewrite
- [x] 15-04-PLAN.md -- Delete taskTrendData mock + delete Line chart card + StatCards all-zero empty state

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 15-05-PLAN.md -- Product name "录播服务系统" in 4 touch points + NotFound rewrite (Result + SVG + stagger)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 15-06-PLAN.md -- Install @playwright/test + config + 4 visual regression specs with page.route mocks (R-1)

### Phase 16: 视觉重塑（去 AI 模板感）

**Goal:** Phase 15 的改动集中在基础设施层，用户视觉感受不到明显变化。本 phase 集中在用户每次打开页面**第一眼看到**的视觉重塑：登录页动效背景、统一页面壳组件、Dashboard 重新布局、列表页布局统一、字体排版分级。
**Requirements**: D-01 through D-08 (from CONTEXT.md)
**Depends on:** Phase 15
**Plans:** 6 plans (planned)

Plans:

**Wave 1** *(foundation — no blockers)*

- [ ] 16-01-PLAN.md -- 登录页动效背景重做（深青渐变 + 漂浮光晕 + 点阵网格 + 鼠标光晕 + 磨砂玻璃卡）

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 16-02-PLAN.md -- 共享 PageShell 组件 + 全站替换 PageHeader + 删除 antd 默认浅灰背景

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 16-03-PLAN.md -- Dashboard 重新布局（实时状态条 + 紧凑统计条 + 图表 2:1 比例）
- [ ] 16-04-PLAN.md -- 列表页布局统一（files / tasks / audit / results / split / system）

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 16-05-PLAN.md -- 字体排版统一（标题分级 + 等宽数字 + 字重 / 行高）
- [ ] 16-06-PLAN.md -- 更新视觉回归 spec baseline（login / dashboard / files / tasks / audit）

### Phase 17: 后端代码审查 56 个发现修复 - P0/P1/P2 全量

**Goal:** 后端代码库通过 56 项审查发现的分级修复（P0 HIGH + P1 MEDIUM + P2 LOW），配齐 P0/P1 单测、同步部署文档，go build/vet/fmt 全绿、既有测试不回归。
**Requirements**: SEC-001..015, BUG-001..016, PERF-001..016, STYLE-001..010 (per docs/audits/2026-07-30-backend-code-review.md; STYLE-001 全库迁移 / SEC-003b 华为密码 DB 加密 / STYLE-009 Get* rename 三项 deferred)
**Depends on:** Phase 16
**Plans:** 4 plans

Plans:
**Wave 1**

- [x] 17-01-PLAN.md — P0 HIGH 发现修复（11 项 + 部署文档同步；SEC-003b deferred） — 4 atomic commits (4d3de0b/2bcee29/47ef805/4fc1d3c) on main, build+tests verified

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 17-02-PLAN.md — P1a 错误处理 + 安全加固（12 项：BUG-003..006 + SEC-005..010 + STYLE-004/005） — 12 atomic commits on main (d27903f..e040d2d, +b53cc8c regression), build+tests -race verified

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 17-03-PLAN.md — P1b 性能 + 接口归位（7 项：PERF-006..011 + STYLE-003） — 7 atomic commits on main (9150e95→0190f83), build+tests -race verified; STYLE-003 接口迁移到消费方包 + compile-level 断言

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 17-04-PLAN.md — P2 LOW 发现清理（20 项；STYLE-001 partial / STYLE-002 误报 / STYLE-009 deferred） — 18 atomic commits on main (4f5579a..857bb55), go build/vet/fmt + 12 包 -race tests green

### Phase 20: 错误处理统一收敛 + sentinel 体系增强

**Goal:** 在 Phase 19（D18-D21 完成 24 个 sentinel 散点统一 + ~356 散点收敛）的基础上进一步深化错误处理体系：handler 层移除 ad-hoc classify 函数全部走 HandleError；zap logger 集成 errors.Is 链并输出 sentinel_type 字段；自动生成 sentinel 文档；引入 typed error kind 字段区分 Sentinel / BusinessError / ad-hoc。
**Requirements**: D-22 候选清单（1）handler 错误处理统一收敛（移除 classify 函数，全部走 HandleError） / （2）zap logger 集成 errors.Is 链输出 sentinel_type 字段 / （3）自动生成 sentinel 文档 / （4）typed error kind 字段区分（Sentinel vs BusinessError vs ad-hoc）
**Depends on:** Phase 19
**Plans:** 5/5 plans complete

Plans:

**Wave 1** *(foundation — SentinelField helper + ErrADUserNotRegistered migration)*

- [x] 20-01-PLAN.md — Foundation: FirstKnownSentinelName export + pkg/response.SentinelField helper + R-3 ErrADUserNotRegistered migration to internal/errors

**Wave 2** *(blocked on Wave 1 — handler classify convergence)*

- [x] 20-02-PLAN.md — 4 heavy handlers (ppt/video_recording_task/auth/input_config) + classifyAuthLoginError delete + ppt_file_service %w wrapping + table-driven tests
- [x] 20-03-PLAN.md — 8 light handlers (file/video_file/admin/user/transcription/split/role/apikey) + table-driven tests  [parallel with 20-02, no file overlap]

**Wave 3** *(blocked on Wave 2 — service zap upgrade + docs generator)*

- [x] 20-04-PLAN.md — Service-layer ~160 zap.Error sites upgraded with SentinelField (services/auth/scheduler/huawei; middleware DEFERRED per D-03.7) [parallel with 20-05]
- [x] 20-05-PLAN.md — cmd/error-doc-gen generator + docs/errors.md + //go:generate directive + CI sync-check (no Makefile per R-2) [parallel with 20-04]

### Phase 21: Close v1.1 gaps: retro-verify phases 17/18/19 + create REQUIREMENTS.md + fix auth_handler.go:57 WARNING

**Goal:** [To be planned]
**Requirements**: TBD
**Depends on:** Phase 20
**Plans:** 0 plans

Plans:
- [ ] TBD (run /gsd-plan-phase 21 to break down)
