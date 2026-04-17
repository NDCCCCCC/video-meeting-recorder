---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: planning
last_updated: "2026-04-17T04:30:28.044Z"
progress:
  total_phases: 4
  completed_phases: 1
  total_plans: 5
  completed_plans: 5
  percent: 100
---

# STATE.md - Project Memory

**Project:** Record V2
**Milestone:** v1.0 - 视频切割与会议转录PPT
**Last Updated:** 2026-04-17

---

## Project Reference

### Core Value

会议视频从录制到PPT的一站式处理，让会议内容可检索、可回顾、可分享。

### What This Is

视频会议录制管理系统 V2.0，专为华为会议终端设计的自动化录制、管理、转录和PPT生成平台。支持自动录制华为会议、USB设备录制、RTSP流录制，提供视频多点分割、阿里通义听悟AI转录、PPT自动提取等能力。

### Current Focus

Phase 1: Video Splitting - Multi-point video splitting, recording snapshot, and auto scan (all local, no external dependencies).

---

## Current Position

Phase: 01 (video-splitting) — EXECUTING
Plan: 1 of 5
**Phase:** 2
**Plan:** Not started
**Status:** Ready to plan
**Progress:** ▱▱▱▱▱▱▱▱▱▱ 0/4 phases (0%)

### Blockers

None identified.

---

## Performance Metrics

### Requirements Coverage

- Total v1 requirements: 30
- Mapped to phases: 30 (100%)
- Unmapped: 0

### Phase Breakdown

| Phase | Requirements | Status |
|-------|--------------|--------|
| Phase 1 - Video Splitting | 10 | Pending |
| Phase 2 - Local Transcription | 7 | Pending |
| Phase 3 - PPT Management | 7 | Pending |
| Phase 4 - Cloud Services | 6 | Pending |

---

## Accumulated Context

### Key Architectural Decisions

1. **All Go Implementation** (No Python microservice)
   - Rationale: Consistency with existing codebase, single-process deployment, simpler operations
   - Status: Pending validation

2. **Aliyun OSS for File Relay**
   - Rationale: Server has no public IP; Tingwu API requires publicly accessible URLs
   - Status: Pending validation

3. **Tingwu REST API with Manual HMAC-SHA256 Signing**
   - Rationale: No official Go SDK exists for Tingwu API
   - Status: Pending validation

4. **FFmpeg for Splitting and Frame Extraction**
   - Rationale: FFmpeg already integrated for recording/conversion
   - Status: Pending validation

5. **Local PPT Generation with Go-pptx Library**
   - Rationale: No need to generate PPT for cloud paths (download directly)
   - Status: Pending validation

6. **Manual Transcription Trigger Only**
   - Rationale: Cost control, user choice, simplified error handling
   - Status: Pending validation

### Tech Stack Context

**Backend:**

- Go 1.24 (Gin framework)
- SQLite database with GORM
- SM4-GCM encryption for Token authentication
- FFmpeg for video processing (already integrated)

**Frontend:**

- React 19
- Ant Design 6
- Zustand for state management
- TanStack Query for API caching

**External Dependencies (New):**

- Aliyun OSS Go SDK v2 (alibabacloud-oss-go-sdk-v2)
- Aliyun Tingwu API (manual REST with HMAC-SHA256)
- Muprprpr/Go-pptx for PPTX generation

### Dependency Analysis

**Phase 1 (Video Splitting):**

- SPLIT-01 to SPLIT-05: Video multi-point splitting with FFmpeg
- SNAP-01, SNAP-02: Recording snapshot without interrupting
- SCAN-01, SCAN-02: Auto file scanning
- UI-01: Split page layout
- Depends on: Nothing (first phase, all local)

**Phase 2 (Local Transcription):**

- LCL-01 to LCL-04: Frame extraction + SSIM/pHash/edge detection + PPTX generation
- TRAN-01 (local), TRAN-04, TRAN-06: Local transcription trigger, status, segment
- Depends on: Phase 1 (split segments need transcription)

**Phase 3 (PPT Management):**

- PPT-01 to PPT-06: PPT preview, download, multi-result, merge
- UI-03: PPT result page layout
- Depends on: Phase 2 (transcription produces PPT results)

**Phase 4 (Cloud Services):**

- OSS-01, OSS-02: Aliyun OSS file relay
- TRAN-01 (cloud), TRAN-02, TRAN-03, TRAN-05: Cloud transcription + fallback
- UI-02: Transcription task page layout
- Depends on: Phase 2 (local fallback), Phase 3 (PPT management)

### Critical Pitfalls from Research

1. **OSS File Orphaning** - Temporary files never deleted, causing indefinite storage costs
   - Mitigation: Lifecycle rules, cleanup handlers, periodic cleanup job

2. **Tingwu Status Polling Thundering Herd** - Rate limiting from simultaneous status requests
   - Mitigation: Jittered exponential backoff, staggered polls, global rate limiter

3. **FFmpeg Keyframe Misalignment** - ±2s precision limitation with -c copy mode
   - Mitigation: Document limitation, offer re-encode option, smart split to nearest keyframe

4. **PPT Image URL Download Timeouts** - Sequential downloads take 5-20 minutes
   - Mitigation: Parallel downloads with worker pool, progress tracking, retry with backoff

5. **Database Transaction Mismatch with OSS** - DB rollback leaves orphaned OSS files
   - Mitigation: Two-phase commit pattern, idempotent operations, state machine

---

## Decisions Log

### 2026-04-17 - Roadmap Creation

**Decision:** 4-phase structure with external services last
**Rationale:** Build local features first (splitting, local transcription, PPT management) that work without external dependencies, then add cloud services as enhancement
**Outcome:** ROADMAP.md created with 30/30 requirements mapped (100% coverage)

---

## Todos

### Immediate

- [ ] Execute `/gsd-plan-phase 1` to create Phase 1 implementation plan
- [ ] Set up OSS integration development environment
- [ ] Review existing recording infrastructure for snapshot implementation

### Short-term

- [ ] Validate OSS SDK v2 integration patterns
- [ ] Test FFmpeg snapshot extraction without interrupting recording
- [ ] Design file scan trigger mechanism

---

## Session Continuity

### Last Session

- Initial milestone v1.0 setup completed
- Requirements defined (30 total)
- Research completed (MEDIUM confidence)
- Roadmap created with 5 phases

### Next Steps

1. Plan Phase 1 implementation
2. Set up OSS development environment
3. Implement OSS upload/download service
4. Implement recording snapshot feature
5. Implement auto file scanning

---

*STATE.md initialized: 2026-04-17*
