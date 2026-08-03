# Roadmap: Record V2 - 视频切割与会议转录PPT

## Milestones

- ✅ **v1.0 视频切割与会议转录PPT** — Phases 01-14 (shipped 2026-05-06)
- ✅ **v1.1 文件管理与编辑增强 / 后端安全加固** — Phases 17-22 (shipped 2026-08-03)
- 📋 **v2.0** — Planning next milestone

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

<details>
<summary>✅ v1.1 文件管理与编辑增强 / 后端安全加固 (Phases 17-22) — SHIPPED 2026-08-03</summary>

- [x] Phase 17: 后端代码审查 56 项修复 P0/P1/P2 (4/4 plans) — completed 2026-07-30
- [x] Phase 18: 凭据静态加密 + 密钥轮换 SEC-003b SM4-GCM (1/1 plan) — completed 2026-07-31
- [x] Phase 19: ctx 全量级联 + SEC-004 jti + STYLE-001 error (4/4 plans) — completed 2026-07-31
- [x] Phase 20: 错误处理统一收敛 + sentinel 体系增强 (5/5 plans) — completed 2026-08-01
- [x] Phase 21: Close v1.1 gaps — retro-verify 17/18/19 + REQUIREMENTS.md + auth:57 fix (5/5 plans) — completed 2026-08-03
- [x] Phase 22: Address v1.1 audit tech debt — regenerate errors.md + backfill VALIDATION.md (6/6 plans) — completed 2026-08-03

**Total:** 25 plans completed

**Audit:** 重审 `status: tech_debt`（gaps 全空，requirements 60/60、phases 5/5、integration 5/5、flows 4/4），5/5 phase VERIFICATION passed。
**See** `.planning/milestones/v1.1-ROADMAP.md` **for full milestone details.**

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
| 17. 后端代码审查 56 项修复 | v1.1 | 4/4 | Complete | 2026-07-30 |
| 18. 凭据静态加密 SEC-003b | v1.1 | 1/1 | Complete | 2026-07-31 |
| 19. ctx 全量级联 + SEC-004 + STYLE-001 | v1.1 | 4/4 | Complete | 2026-07-31 |
| 20. 错误处理统一收敛 + sentinel | v1.1 | 5/5 | Complete | 2026-08-01 |
| 21. Close v1.1 gaps | v1.1 | 5/5 | Complete | 2026-08-03 |
| 22. v1.1 audit tech debt 收尾 | v1.1 | 6/6 | Complete | 2026-08-03 |

## Backlog

未归入已交付里程碑的候选项（下个里程碑 `/gsd:new-milestone` 裁定归属）：

### Phase 1: 新功能：在视频播放中添加外挂字幕支持（预览视频、切割视频、预览PPT页面）

**Goal:** 在VideoPlayerModal、切割视频页面、PPTPreview三个视频播放场景中实现外挂字幕功能，使用WebVTT格式，字幕显示在视频下方独立区域，支持开关、字号调整和样式配置。
**Requirements**: D-01 through D-11 (from CONTEXT.md)
**Depends on:** None (standalone feature)
**Plans:** 3 plans

Plans:

- [ ] 01-01-PLAN.md -- Backend subtitle API endpoints + frontend type definitions
- [ ] 01-02-PLAN.md -- Subtitle sync hook + SubtitlePanel component with style controls
- [ ] 01-03-PLAN.md -- Integration into VideoPlayerModal and PPTPreview

### Phase 15: 前端去 AI 味（6/6 complete — unassigned to milestone）

**Goal:** 消除 Record V2 前端 UI 中的"AI 生成感"——替换模板化文案、删除硬编码 mock、补齐空/错/加载态、统一设计令牌与品牌基线、加入克制的微交互，并以 Playwright 截图回归保障全量改的回归风险。
**Requirements**: D-01 through D-08 (from CONTEXT.md)
**Depends on:** Phase 14
**Plans:** 6/6 plans complete
**Status:** 代码已全部落地（6 plans），但未归入 v1.0/v1.1 任何里程碑。下个里程碑可追溯归档或作为 v2.0 前端基线。

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

### Phase 16: 视觉重塑（去 AI 模板感）（planned — 1/6 complete）

**Goal:** Phase 15 的改动集中在基础设施层，用户视觉感受不到明显变化。本 phase 集中在用户每次打开页面**第一眼看到**的视觉重塑：登录页动效背景、统一页面壳组件、Dashboard 重新布局、列表页布局统一、字体排版分级。
**Requirements**: D-01 through D-08 (from CONTEXT.md)
**Depends on:** Phase 15
**Plans:** 6 plans (1 executed, 5 planned)

Plans:

**Wave 1** *(foundation — no blockers)*

- [x] 16-01-PLAN.md -- 登录页动效背景重做（深青渐变 + 漂浮光晕 + 点阵网格 + 鼠标光晕 + 磨砂玻璃卡）

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 16-02-PLAN.md -- 共享 PageShell 组件 + 全站替换 PageHeader + 删除 antd 默认浅灰背景

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 16-03-PLAN.md -- Dashboard 重新布局（实时状态条 + 紧凑统计条 + 图表 2:1 比例）
- [ ] 16-04-PLAN.md -- 列表页布局统一（files / tasks / audit / results / split / system）

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 16-05-PLAN.md -- 字体排版统一（标题分级 + 等宽数字 + 字重 / 行高）
- [ ] 16-06-PLAN.md -- 更新视觉回归 spec baseline（login / dashboard / files / tasks / audit）
