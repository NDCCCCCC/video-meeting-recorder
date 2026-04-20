# Phase 4: Cloud Services - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-17
**Phase:** 04-cloud-services
**Areas discussed:** Transcription Mode Selection, Cloud Status Tracking & Polling, Cloud Fallback Experience, Text Content Display

---

## Transcription Mode Selection

| Option | Description | Selected |
|--------|-------------|----------|
| 两步弹窗 | Click "转录" → small modal asking mode → then progress modal | |
| 下拉按钮 | "转录" button becomes dropdown with "本地转录" and "云端转录" options, click to start | ✓ |
| 复用现有弹窗 | Add radio selector at top of TranscriptionProgressModal before starting | |

**User's choice:** 下拉按钮 (Dropdown button) — direct action, no extra modal step
**Notes:** Also applies to result page "重新转录" button — both use dropdown for consistency. Cloud transcription needs no sampling rate parameter.

---

## Cloud Status Tracking & Polling

| Option | Description | Selected |
|--------|-------------|----------|
| 详细阶段 (5s polling) | Show each stage: uploading → queued → processing → complete, 5s frontend poll | |
| 详细阶段 (10s polling) | Same stages with 10s frontend poll — lighter load for cloud tasks | ✓ |
| 简化进度 | Show overall progress without distinguishing Tingwu internal stages | |
| 仅状态指示 | Show only "processing..." and final status, no percentage | |

**User's choice:** 详细阶段 with 10s polling. User confirmed Tingwu API provides detailed progress.
**Notes:** Reuse existing TranscriptionProgressModal with adapted cloud stages: 上传中 (0-20%) → 排队中 → 处理中 (20-90%) → 下载结果 (90-100%).

---

## Cloud Fallback Experience

| Option | Description | Selected |
|--------|-------------|----------|
| 无缝切换 + 提示 | Progress modal auto-switches to local mode with info alert | ✓ |
| 通知 + 手动确认 | Show notification, user must click to confirm local fallback | |
| 静默降级 + 完成通知 | Silent fallback, only notify at completion | |

**User's choice:** 无缝切换 + 提示 (seamless switch with notification)
**Notes:** Fallback only triggers at initial submission stage (OSS upload failure or Tingwu API rejection). Mid-processing failures do NOT auto-fallback.

---

## Text Content Display

| Option | Description | Selected |
|--------|-------------|----------|
| Tab标签页切换 | Add "文字内容" tab in result page right panel alongside basic info | ✓ |
| PPT预览下方展示 | Show text below PPT preview area | |
| 弹窗展示 | Separate modal for text content | |

**User's choice:** Tab标签页切换 (Tab switching in right panel)

### Text Format

| Option | Description | Selected |
|--------|-------------|----------|
| 可点击时间戳 | [HH:MM:SS] prefix per segment, click to jump to video position | ✓ |
| 纯文本展示 | Timestamps shown but not interactive | |
| 时间轴布局 | Left timeline + right text, subtitle editor style | |

**User's choice:** 可点击时间戳 (clickable timestamps)

### Copy Functionality

| Option | Description | Selected |
|--------|-------------|----------|
| 全文复制 + 段落复制 | "复制全部文字" button + per-segment copy icon | ✓ |
| 仅全文复制 | Only full text copy button | |

**User's choice:** 全文复制 + 段落复制

---

## Claude's Discretion

- OSS Go SDK v2 integration details (multipart upload, presigned URL)
- Tingwu REST API client implementation (HMAC-SHA256 signing)
- TranscriptionTask model extensions (new fields, migration)
- Backend Tingwu polling parameters (exponential backoff)
- OSS cleanup mechanism
- Config struct additions
- API endpoint design
- Text content storage model
- Progress modal stage adaptation logic
- Error classification for fallback trigger
- Dropdown button Ant Design component choice

## Deferred Ideas

None — discussion stayed within phase scope.
