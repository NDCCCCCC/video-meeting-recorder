# Milestones

## v1.0 视频切割与会议转录PPT (Shipped: 2026-04-18)

**Phases completed:** 4 phases, 16 plans, 19 tasks

**Key accomplishments:**

- Wave 0 test stubs establishing Nyquist validation contract for Phase 1 video splitting, snapshot generation, and auto-scan features
- VideoFile model extended with parent_id, source_type, snapshot_offset fields supporting split segments and incremental snapshots with idempotent migration 003
- FFmpeg-based video splitting with worker pool and incremental snapshot generation using service callback pattern
- Files Created:
- Task list and file list UI extensions for snapshot generation, source tracking, and split navigation
- File:
- Choice:
- Backend foundation for PPT preview, multi-result management, and slide merge with dual-resolution caching, Python-pptx integration, and secure API endpoints
- PPT result detail page with preview, multi-result gallery switching, slide merge with drag-to-reorder, and file list integration using React, Ant Design, and dnd-kit
- Found during:
- Frontend Types Extended
- File:
- TextContentTab Component Created
- Commit:

---

## v1.0 — 视频切割与会议转录PPT

**Status:** In Progress
**Started:** 2026-04-17
**Goal:** 添加视频多点分割和阿里通义听悟转录PPT功能

### Target Features

- 视频多点分割
- 阿里通义听悟转录集成
- 阿里云OSS文件中转
- 转录结果管理
- PPT独立下载

### Phases

(To be defined by roadmap)

---
*No prior milestones — this is the first milestone.*
