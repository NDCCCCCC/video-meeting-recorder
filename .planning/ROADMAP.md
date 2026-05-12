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
