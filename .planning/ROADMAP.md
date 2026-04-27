# Roadmap: Record V2

## Milestones

- ✅ **v1.0 视频切割与会议转录PPT** — Phases 1-4 (shipped 2026-04-18)
- 🔄 **v1.1 文件管理与编辑增强** — Phases 5-11 (in progress)

## Phases

<details>
<summary>✅ v1.0 视频切割与会议转录PPT (Phases 1-4) — SHIPPED 2026-04-18</summary>

- [x] Phase 1: Video Splitting (5/5 plans) — completed 2026-04-17
- [x] Phase 2: Local Transcription (4/4 plans) — completed 2026-04-18
- [x] Phase 3: PPT Management (2/2 plans) — completed 2026-04-18
- [x] Phase 4: Cloud Services (5/5 plans) — completed 2026-04-18

</details>

<details>
<summary>🔄 v1.1 文件管理与编辑增强 (Phases 5-11) — IN PROGRESS</summary>

- [x] Phase 5: File Rename & Smart Cleanup (2/2 plans) — **completed 2026-04-20**
    - [x] 05-01-PLAN.md — File rename API and UI for split videos and PPTs
    - [x] 05-02-PLAN.md — Smart cleanup logic for re-splitting videos
- [x] Phase 6: PPT Editor UI Improvements (7/7 plans) — **completed 2026-04-20**
    - [x] 06-01-PLAN.md — Duplicate slide detection and deletion
    - [x] 06-02-PLAN.md — Video preview integration with timestamp sync
    - [x] 06-03-PLAN.md — Slide capture from video and insertion
    - [x] 06-06-01-PLAN.md — Video playback speed control (0.5x-2x)
    - [x] 06-06-02-PLAN.md — Side-by-side 16:9 layout for PPT and video previews
    - [x] 06-06-03-PLAN.md — Direct slide capture without modal
    - [x] 06-06-04-PLAN.md — Optimized vertical scrolling thumbnails with lazy loading
- [x] Phase 7: Preview Page UI Improvements (4/4 plans) — **completed 2026-04-20**
    - [x] 07-01-PLAN.md — Thumbnail sidebar fixed height & video aspect ratio correction
    - [x] 07-02-PLAN.md — Editable progress bar time input & PPT results dropdown
    - [x] 07-03-PLAN.md — Info display reorganization & operations bar horizontal layout
    - [x] 07-04-PLAN.md — Gap: Fix thumbnail height alignment & video black box
- [x] Phase 8: Video Snapshot & Player Enhancement (5/5 plans) — **completed 2026-04-20**
    - [x] 08-00-PLAN.md — Wave 0: Test stubs for snapshot service and player enhancements
    - [x] 08-01-PLAN.md — Snapshot service: concurrent safety, enhanced naming, validation
    - [x] 08-02-PLAN.md — Keyboard shortcuts hook and utility constants
    - [x] 08-03-PLAN.md — Frame-level navigation hook and component
    - [x] 08-04-PLAN.md — VideoPlayerModal integration of all enhancements
- [ ] Phase 9: Multi-Role Permissions & Shared Viewer (0/0 plans) — **not started**
    - [ ] TBD (run /gsd-plan-phase 9 to break down)
- [x] Phase 10: Admin Dashboard, Audit Logs, and UI Enhancements (5/5 plans) — **completed 2026-04-24**
    - [x] 10-01-PLAN.md — Backend dashboard statistics API (aggregations, service, handler, routes)
    - [x] 10-02-PLAN.md — Audit log export functionality and frontend API client
    - [x] 10-03-PLAN.md — Admin dashboard frontend (StatCards, ChartsSection, QuickActions, useDashboardStats)
    - [x] 10-04-PLAN.md — Audit logs viewer frontend (AuditTable, FilterBar, DiffModal, ExportButton, useAuditLogs)
    - [x] 10-05-PLAN.md — Design tokens system and reusable hooks (useLoadingState, useErrorHandler, API error interceptor)
- [ ] Phase 11: IP地址登录限制 (6/6 plans) — **ready to execute**
    - [ ] 11-00-PLAN.md — Wave 0: Test infrastructure for IP validation and restriction
    - [ ] 11-01-PLAN.md — Backend IP validation, model fields, and login enforcement (TDD)
    - [ ] 11-02-PLAN.md — Audit logging integration for IP restriction failures
    - [ ] 11-03-PLAN.md — Frontend TypeScript types and API client support
    - [ ] 11-04-PLAN.md — Frontend UI components for IP management (with checkpoints)
    - [ ] 11-05-PLAN.md — Comprehensive testing and documentation (with checkpoints)

</details>

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Video Splitting | v1.0 | 5/5 | Complete | 2026-04-17 |
| 2. Local Transcription | v1.0 | 4/4 | Complete | 2026-04-18 |
| 3. PPT Management | v1.0 | 2/2 | Complete | 2026-04-18 |
| 4. Cloud Services | v1.0 | 5/5 | Complete | 2026-04-18 |
| 5. File Rename & Smart Cleanup | v1.1 | 2/2 | **Complete** | 2026-04-20 |
| 6. PPT Editor UI Improvements | v1.1 | 7/7 | **Complete** | 2026-04-20 |
| 7. Preview Page UI Improvements | v1.1 | 4/4 | **Complete** | 2026-04-20 |
| 8. Video Snapshot & Player Enhancement | v1.1 | 5/5 | **Complete** | 2026-04-20 |
| 9. Multi-Role Permissions & Shared Viewer | v1.1 | 0/0 | **Not Started** | — |
| 10. Admin Dashboard, Audit Logs, and UI Enhancements | v1.1 | 5/5 | **Complete** | 2026-04-24 |
| 11. IP地址登录限制 | v1.1 | 0/6 | **Ready to Execute** | — |

### Phase 10: Admin Dashboard, Audit Logs, and UI Enhancements

**Goal:** Deliver admin dashboard with statistics visualization, comprehensive audit log viewer with diff capabilities and export, and foundational UI design tokens and reusable hooks for consistent user experience.

**Requirements:**
- D-01 to D-12: Admin dashboard layout, statistics, charts, quick actions, admin-only access
- D-13 to D-26: Audit logs table, filters, diff modal, CSV/JSON export
- D-27 to D-39: Design tokens, loading states, error handling, API error interceptor

**Depends on:** Phase 9
**Plans:** 5/5 plans complete

Plans:
- [x] 10-01-PLAN.md — Backend dashboard statistics API (aggregations, service, handler, routes) — Wave 1
- [x] 10-02-PLAN.md — Audit log export functionality and frontend API client — Wave 1
- [x] 10-03-PLAN.md — Admin dashboard frontend (StatCards, ChartsSection, QuickActions, useDashboardStats) — Wave 2
- [x] 10-04-PLAN.md — Audit logs viewer frontend (AuditTable, FilterBar, DiffModal, ExportButton, useAuditLogs) — Wave 2
- [x] 10-05-PLAN.md — Design tokens system and reusable hooks (useLoadingState, useErrorHandler, API error interceptor) — Wave 3

**Wave Structure:**
- Wave 1 (parallel): 10-01 (backend dashboard API), 10-02 (audit export + API client)
- Wave 2 (parallel): 10-03 (dashboard frontend, depends on 10-01), 10-04 (audit frontend, depends on 10-02)
- Wave 3 (sequential): 10-05 (design tokens + hooks, depends on 10-03, 10-04)

### Phase 11: IP地址登录限制 - 为用户和角色添加IP地址组，限制只有组内地址才能登录系统

**Goal:** Implement IP address-based access control for users and roles with OR logic merging, supporting IPv4 single addresses, CIDR ranges, and IP ranges, integrated with login flow and audit logging.

**Requirements:**
- D-01 to D-17: User and role IP restrictions, OR logic merging, IPv4-only support, login-time enforcement, audit logging

**Depends on:** Phase 10
**Plans:** 6/6 plans ready

Plans:
- [ ] 11-00-PLAN.md — Wave 0: Test infrastructure for IP validation and restriction — Wave 0
- [ ] 11-01-PLAN.md — Backend IP validation, model fields, and login enforcement (TDD) — Wave 1
- [ ] 11-02-PLAN.md — Audit logging integration for IP restriction failures — Wave 2
- [ ] 11-03-PLAN.md — Frontend TypeScript types and API client support — Wave 2
- [ ] 11-04-PLAN.md — Frontend UI components for IP management (with checkpoints) — Wave 2
- [ ] 11-05-PLAN.md — Comprehensive testing and documentation (with checkpoints) — Wave 3

**Wave Structure:**
- Wave 0 (sequential): 11-00 (test stubs for all IP functionality)
- Wave 1 (sequential): 11-01 (backend IP validation, models, CheckIPRestriction, migration - TDD)
- Wave 2 (parallel): 11-02 (audit logging), 11-03 (frontend types/API), 11-04 (frontend UI with checkpoints)
- Wave 3 (sequential): 11-05 (E2E testing, docs, verification)

**Key Decisions:**
- D-01 to D-03: User + role IP restrictions with OR logic merging
- D-04 to D-05: JSON field storage (no separate IP address table)
- D-06 to D-09: IPv4 only (single IP, CIDR, IP range formats)
- D-10 to D-12: ClientIP() extraction, direct deployment
- D-13 to D-15: User-friendly error messages, audit logging, no admin exemption
- D-16 to D-17: IP check after password validation, before token generation

---
*Roadmap created: 2026-04-17*
*Last updated: 2026-04-27 - Phase 11 planning complete (6 plans)*
