# Roadmap: Record V2 — v1.0 Milestone

**Milestone:** v1.0 - 视频切割与会议转录PPT
**Granularity:** Standard
**Planned:** 2026-04-17
**Phases:** 4

## Phases

- [ ] **Phase 1: Video Splitting** - Multi-point video splitting, recording snapshot, and auto scan
- [ ] **Phase 2: Local Transcription** - Frame extraction, SSIM/pHash/edge detection, and PPTX generation
- [ ] **Phase 3: PPT Management** - PPT preview, multi-result management, and merge features
- [ ] **Phase 4: Cloud Services** - Aliyun OSS integration, Tingwu transcription, and cloud/local switching

## Phase Details

### Phase 1: Video Splitting

**Goal**: Users can split videos at multiple time points, generate MP4 snapshots during recording, and all new MP4 files are automatically scanned into the file management system

**Depends on**: Nothing (first phase, all local)

**Requirements**: SPLIT-01, SPLIT-02, SPLIT-03, SPLIT-04, SPLIT-05, SNAP-01, SNAP-02, SCAN-01, SCAN-02, UI-01

**Success Criteria** (what must be TRUE):
1. User can watch a video in the browser and click on the timeline to add split markers at precise time points
2. User can click "生成MP4快照" during active recording and download an MP4 without stopping the recording
3. MP4 files generated from any source (recording completion, snapshot, or splitting) automatically appear in the file management page within 5 seconds
4. User can click "确认分割" and FFmpeg splits the video into multiple MP4 segments using -c copy mode
5. Split segments appear in the file list and can be renamed, deleted, downloaded, or individually transcribed

**Plans**: 4 plans

Plans:
- [ ] 01-01-PLAN.md — Database model extension (parent_id, source_type) + VideoFileService segment creation
- [ ] 01-02-PLAN.md — SplittingService, SnapshotService, SplitHandler backend + app.go wiring
- [ ] 01-03-PLAN.md — Frontend split page with TimelineWithMarkers + split API client
- [ ] 01-04-PLAN.md — Task list snapshot button + file list source column + auto-refresh

---

### Phase 2: Local Transcription

**Goal**: System extracts video frames, detects slide changes using multi-dimensional similarity (SSIM + pHash + edge analysis), generates PPTX files locally, with real-time progress tracking

**Depends on**: Phase 1 (video splitting and segment management)

**Requirements**: LCL-01, LCL-02, LCL-03, LCL-04, TRAN-01 (local trigger), TRAN-04 (status tracking), TRAN-06 (segment transcription)

**Success Criteria** (what must be TRUE):
1. User can click "转录" button on any video or split segment to trigger local transcription
2. System extracts video frames at a configurable sampling rate (e.g., 1 frame per 2 seconds) using FFmpeg
3. System analyzes each new frame using three detection methods: SSIM structural similarity (<0.85 indicates change), pHash perceptual hash (difference >10 indicates change), and edge change rate (>0.25 indicates change)
4. User can see real-time progress feedback showing "已处理 45/200 帧 (22%)" during local transcription
5. After processing completes, a PPTX file is generated with each unique frame as a separate slide page

**Plans**: TBD

---

### Phase 3: PPT Management

**Goal**: Users can preview PPT in browser, manage multiple transcription results, and merge slides from different PPT files

**Depends on**: Phase 2 (transcription produces PPT results)

**Requirements**: PPT-01, PPT-02, PPT-03, PPT-04, PPT-05, PPT-06, UI-03

**Success Criteria** (what must be TRUE):
1. User can download the PPT file independently from the original video in the file list
2. PPT files are displayed in the file list linked to their source video with clear visual association
3. User can click "预览PPT" to browse slide pages in the browser without downloading the file
4. If a PPT has insufficient slides (e.g., detection missed changes), user can click "重新转录" to submit the video again
5. System retains all historical PPT results from multiple transcriptions of the same video
6. User can select specific slide pages from multiple PPT results and merge them into a final PPT file
7. User can view transcription results in a dedicated page showing text content, PPT preview, and actions (download/retry/merge)

**Plans**: TBD

**UI hint**: yes

---

### Phase 4: Cloud Services

**Goal**: Integrate Aliyun OSS for file relay and Aliyun Tingwu for cloud transcription, with cloud/local choice and automatic fallback

**Depends on**: Phase 2 (local transcription for fallback), Phase 3 (PPT management)

**Requirements**: OSS-01, OSS-02, TRAN-01 (cloud option), TRAN-02, TRAN-03, TRAN-05, UI-02

**Success Criteria** (what must be TRUE):
1. System can upload video files to Aliyun OSS and generate publicly accessible URLs
2. User can choose between "云端转录（通义听悟）" and "本地转录" when triggering transcription
3. Cloud transcription uploads to OSS, submits to Tingwu API, and tracks real-time status: 排队中 → 处理中 → 完成/失败
4. When cloud transcription fails, system automatically falls back to local transcription and notifies the user
5. Cloud transcription completes with text content that user can view with timestamps
6. OSS files are automatically cleaned up within 24 hours after transcription completes to prevent storage costs

**Plans**: TBD

**UI hint**: yes

---

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Video Splitting | 0/4 | Planned | - |
| 2. Local Transcription | 0/3 | Not started | - |
| 3. PPT Management | 0/3 | Not started | - |
| 4. Cloud Services | 0/3 | Not started | - |

---
*Roadmap created: 2026-04-17*
*Last updated: 2026-04-17 after phase planning for Phase 1*
