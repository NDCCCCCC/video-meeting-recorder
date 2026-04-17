# Project Research Summary

**Project:** Record V2 - Video Splitting, Aliyun Tingwu Transcription, and PPT Extraction
**Domain:** Video Recording Management with AI Transcription
**Researched:** 2025-04-17
**Confidence:** MEDIUM

## Executive Summary

Record V2 is a meeting recording management system that requires three core capabilities: manual video splitting with timeline UI, Aliyun Tingwu transcription integration, and PPT slide extraction from transcriptions. This is an established product category with well-defined patterns from platforms like Microsoft Teams, Zoom, and Otter.ai. Expert implementations separate video processing from transcription workflows, use cloud storage for external API integrations, and implement robust task queue systems for long-running operations.

The recommended approach is to build in three distinct phases: (1) video splitting infrastructure using existing FFmpeg integration with timeline UI components, (2) Aliyun Tingwu transcription service with OSS file upload and task polling, and (3) PPT extraction and generation leveraging the transcription infrastructure. This phased approach mitigates the two highest risks identified: OSS file orphaning (which causes indefinite storage cost accumulation) and Tingwu API rate limiting during status polling. Critical mitigation strategies include implementing OSS lifecycle rules for automatic cleanup, using jittered exponential backoff for API polling, and persisting task IDs immediately to survive service restarts.

Key technical decisions include using the Aliyun OSS Go SDK v2 for cloud storage (not the legacy v1), implementing manual REST API calls to Tingwu with HMAC-SHA256 signature generation (no official Go SDK exists), and using FFmpeg's `-c copy` mode for fast lossless splitting with documented precision limitations (±2 seconds at keyframes). The architecture should follow existing patterns in the codebase: service layer for business logic, handler layer for HTTP endpoints, and background worker pools for async operations. Video processing and transcription should be completely separate workflows to allow independent scaling and error handling.

## Key Findings

### Recommended Stack

**Core Additions to Existing Go/Gin + React Stack:**

- **alibabacloud-oss-go-sdk-v2** (Latest) — Aliyun OSS file upload. Official SDK v2 with modern API design, supports Go 1.18+, environment-based credentials. Avoid legacy v1 SDK.
- **net/http + crypto/hmac** (stdlib) — Tingwu API calls with SignatureV4 signing. No official Go SDK exists for Tingwu; manual REST implementation required.
- **FFmpeg CLI** (already integrated) — Video splitting. Use `-ss` seek + `-codec copy` for fast, lossless multi-point splitting. Accept ±2s keyframe precision limitation.
- **Muprprpr/Go-pptx** (Latest) — PPTX file generation. Pure Go library, no license restrictions. Avoid unidoc/unioffice (requires paid commercial license).

**Stack Integration Pattern:**
```
Local Video → OSS SDK v2 → Public URL → Tingwu API → Poll for Results → Download & Store
```

### Expected Features

**Must have (table stakes) — users expect these features or product feels incomplete:**
- Manual split point marking with timeline UI — Basic requirement for video segmentation
- Video preview while marking — Users expect to see video while setting split points
- Split into multiple segments — Core splitting functionality using FFmpeg `-c copy`
- Manual transcription trigger — Cost control, users choose when to transcribe
- Transcription status tracking — Users need to see pending/processing/completed/failed states
- Download transcription text — Basic deliverable from transcription
- PPT images download — Key differentiator feature, extracted slides from video
- Task history/audit log — Leverage existing audit system for transparency
- Error feedback — Clear error messages when operations fail
- OSS upload progress — Essential feedback for large file uploads (2-5GB typical)

**Should have (competitive advantages):**
- Multi-point timeline UI — Visual timeline with multiple markers vs single split (rare in basic systems)
- Smart split suggestions — Suggest split points based on silence detection (advanced feature)
- PPT extraction + transcript sync — PPT images linked to transcript timestamps (unique value)
- Segment-specific transcription — Transcribe only selected segments (cost/time saving)
- Search across transcriptions — Full-text search in transcribed content (valuable for meetings)
- Speaker identification — Tingwu provides speaker diarization (included in API)
- Lossless splitting — No re-encoding means fast processing (FFmpeg capability)
- Auto-delete from OSS — Cost management by removing files after transcription

**Explicitly NOT building (anti-features):**
- Real-time transcription — Only manual trigger permitted (per PROJECT.md constraints)
- Auto-transcription after recording — Cost control and user choice (per PROJECT.md)
- AI-generated PPT from text — Only using PPT extraction from video (per PROJECT.md)
- Multi-language translation — Out of scope, focus on Chinese transcription
- Advanced video editing — Not a video editor, simple splitting only

### Architecture Approach

**Standard three-tier architecture with background workers:**

```
Frontend (React 19 + Ant Design) → API Layer (Gin) → Service Layer → Data Layer (GORM/SQLite)
                                                              ↓
                                                 External APIs (OSS, Tingwu, FFmpeg)
```

**Major components:**
1. **Frontend Layer** — React components for timeline UI, task management interface, and transcription results viewer. Uses existing patterns with Zustand for state management and TanStack Query for API caching.
2. **API Layer (Gin)** — HTTP handlers for transcription and split operations. Leverages existing MultiAuth middleware (SM4 Token + API Key). Handlers should be thin, delegating to services.
3. **Service Layer** — Business logic for transcription workflow, video splitting, and OSS integration. Includes background worker pools for async operations. Interface-based design for testability.
4. **Data Layer** — GORM models for Transcription and SplitSegment entities. Extends existing video_file and ppt_file relationships.
5. **External Services** — Aliyun OSS SDK v2 for storage, manual HTTP client for Tingwu API with signature generation, FFmpeg CLI for video processing.

**Key architectural patterns:**
- **Service Interface Pattern** — Define interfaces for TranscriptionService, OSSService, SplitService to enable testing and provider flexibility
- **Background Worker Pattern** — Worker pools for long-running operations (video processing, API polling) with controlled concurrency
- **Repository Pattern for External APIs** — Wrap Aliyun APIs in repository-like interfaces to isolate dependencies and enable mocking
- **Two-Phase Commit for OSS** — Upload to OSS first, then create DB record; on DB failure, delete from OSS to prevent orphaning

### Critical Pitfalls

**Top 5 pitfalls with prevention strategies:**

1. **OSS File Orphaning** — Temporary video files uploaded to OSS are never deleted, causing indefinite storage cost accumulation. Prevention: Wrap OSS operations in cleanup handlers using `defer`, set OSS lifecycle rules to auto-delete after 24-48 hours, track uploads in database with timestamps, implement periodic cleanup job for orphaned files (>48h old).

2. **Tingwu Status Polling Thundering Herd** — Multiple transcription tasks completing simultaneously barrage Tingwu API with status requests, triggering rate limiting (429 errors). Prevention: Implement jittered polling with ±30% variation, exponential backoff (5s → 30s max), staggered initial polls, global rate limiter (max 10 req/sec), priority queue for newer tasks.

3. **FFmpeg Keyframe Misalignment** — Video splitting at arbitrary timestamps (not on keyframes) creates output segments with frozen video for first 1-2 seconds and audio-video drift. Prevention: Document ±2s precision limitation with `-c copy` mode, offer re-encode option for frame-accurate cuts, use ffprobe to analyze keyframe positions, implement "smart split" that adjusts timestamps to nearest keyframe.

4. **PPT Image URL Download Timeouts** — Tingwu returns 50-200 image URLs for PPT extraction; sequential downloads take 5-20 minutes causing jobs to appear "stuck". Prevention: Parallel downloads with worker pool (10-20 concurrent), progress tracking in DB shown to users via API, retry with exponential backoff (3x), allow partial completion with warnings, 30s timeout per image.

5. **Database Transaction Mismatch with OSS Operations** — DB transaction rolls back after OSS upload succeeds, leaving orphaned files on OSS. Prevention: Two-phase commit pattern (upload to OSS with unique ID → create DB record with OSS key → on DB failure, delete from OSS), idempotent OSS operations using same key for retries, state machine for transcription tasks tracking OSS upload status separately.

## Implications for Roadmap

Based on combined research from stack, features, architecture, and pitfalls, the recommended phase structure:

### Phase 1: Foundation - OSS Integration & Video Splitting
**Rationale:** OSS integration is the foundation for Tingwu transcription (requires publicly accessible video URLs). Video splitting is independent of external APIs and provides immediate user value. Building both first establishes critical infrastructure and prevents OSS file orphaning from day one.

**Delivers:** OSS upload/download service with lifecycle management, FFmpeg video splitting service with timeline UI, split segment management with preview, OSS auto-cleanup job.

**Addresses:** Manual split point marking, video preview while marking, split into multiple segments, segment naming, split preview before commit, download segments, OSS upload with progress.

**Avoids:** OSS file orphaning (Pitfall #1), database transaction mismatch with OSS (Pitfall #5), large video upload timeouts (Pitfall #6), FFmpeg keyframe misalignment (Pitfall #3).

**Stack elements:** alibabacloud-oss-go-sdk-v2, FFmpeg CLI with `-c copy`, net/http stdlib

**Architecture components:** OSSService, SplitService, OSS/Split handlers, OSS/Split models

### Phase 2: Core Value - Tingwu Transcription Integration
**Rationale:** Depends on Phase 1 OSS infrastructure. Core transcription workflow is the primary value proposition. Establishes task queue patterns and API polling strategies that prevent rate limiting issues.

**Delivers:** Tingwu API client with signature generation, transcription service with task submission, status polling with exponential backoff, result download and storage, task history/audit log integration, cost tracking display.

**Addresses:** Manual transcription trigger, Tingwu task submission, transcription status tracking, download transcription text, error feedback, task history/audit log, file size/length display, cost warning before submission.

**Avoids:** Tingwu status polling thundering herd (Pitfall #2), Tingwu task ID not persisted (Pitfall #7), no cost warning (UX pitfall).

**Stack elements:** net/http with crypto/hmac for Tingwu API, context for cancellation

**Architecture components:** TingwuClient, TranscriptionService, TranscriptionHandler, Transcription model, background worker for polling

### Phase 3: Differentiator - PPT Extraction & Generation
**Rationale:** Builds on Phase 2 transcription infrastructure. PPT extraction is a key differentiator that links transcript timestamps to slide images. Can be developed independently after transcription is stable.

**Delivers:** PPT extraction from Tingwu results, parallel image download with worker pool, PPTX file generation using Go-pptx, PPT + transcript timestamp synchronization, PPT download endpoints, transcription result preview viewer.

**Addresses:** PPT extraction trigger, PPT images download, transcription result preview, PPT + transcript sync, speaker identification display, keyword extraction display.

**Avoids:** PPT image URL download timeouts (Pitfall #4), PPT filename not validated (Pitfall #9).

**Stack elements:** Muprprpr/Go-pptx for PPTX generation, worker pool for parallel downloads

**Architecture components:** ResultProcessor, PPT generation service, extends Transcription model

### Phase 4: Enhancement - Smart Features & Search
**Rationale:** Advanced features that build on stable Phase 1-3 foundation. Can be developed incrementally based on user feedback.

**Delivers:** Smart split suggestions based on silence detection, full-text search across transcriptions, auto-chapter detection using Tingwu chapter data, segment-specific transcription option.

**Addresses:** Smart split suggestions, search across transcriptions, auto-chapter detection, segment-specific transcription.

**Research needs:** Silence detection algorithms (FFmpeg), search implementation options (Elasticsearch/whoosh)

### Phase 5: Collaboration - Advanced Editing & Real-time
**Rationale:** Requires separate infrastructure (real-time backend). Lowest priority for MVP.

**Delivers:** Transcription editing interface, collaborative annotations, real-time updates.

**Addresses:** Transcription editing, collaborative annotations.

**Research needs:** Real-time collaboration systems (WebSocket/WebRTC), rich text editor with timestamp handling

### Phase Ordering Rationale

**Why this order based on dependencies:**
- Phase 1 (OSS + Splitting) must come first because Tingwu requires publicly accessible video URLs from OSS, and splitting is independent of external APIs
- Phase 2 (Transcription) depends on Phase 1 OSS infrastructure but is independent of splitting features
- Phase 3 (PPT) depends on Phase 2 transcription infrastructure (uses Tingwu PPT extraction API)
- Phase 4 (Smart Features) builds on all previous phases, requires stable transcription data
- Phase 5 (Collaboration) requires separate real-time infrastructure, lowest business value for MVP

**Why this grouping based on architecture:**
- Phase 1 groups infrastructure (OSS) and independent feature (splitting) to establish foundational services
- Phase 2 focuses on core external API integration (Tingwu) with proper async patterns
- Phase 3 extends existing transcription service rather than creating new architecture
- Phase 4 and 5 are separate enhancement tracks that can be prioritized based on feedback

**How this avoids pitfalls from research:**
- Building OSS integration first with lifecycle rules prevents Pitfall #1 (file orphaning) from day one
- Implementing task queue pattern in Phase 2 prevents Pitfall #2 (polling thundering herd) and Pitfall #7 (task ID not persisted)
- Addressing video splitting in Phase 1 allows early testing of Pitfall #3 (keyframe misalignment) and Pitfall #6 (large upload timeouts)
- Deferring PPT to Phase 3 allows focus on Pitfall #4 (image download timeouts) in isolation

### Research Flags

**Phases likely needing deeper research during planning:**

- **Phase 1 (OSS Integration):** Timeline UI component design — MEDIUM priority research needed for video editor timeline UX best practices. Can start with simple implementation and iterate.
- **Phase 2 (Tingwu Integration):** Tingwu API error handling — HIGH priority research needed to resolve API documentation gap (specific API docs returned 404). Need to research error codes, retry logic, and polling optimization during phase planning.
- **Phase 2 (Tingwu Integration):** Task queue implementation — MEDIUM priority research needed for Go-based task queue patterns for long-running operations. Standard patterns exist but need adaptation for Tingwu polling.
- **Phase 4 (Smart Features):** Silence detection algorithms — HIGH priority research needed if implementing smart split suggestions. FFmpeg silence detection requires research on threshold tuning and accuracy.
- **Phase 4 (Smart Features):** Search implementation — HIGH priority research needed for full-text search options (Elasticsearch vs whoosh vs SQLite FTS5). Major infrastructure decision.
- **Phase 5 (Collaboration):** Real-time systems — HIGH priority research needed for collaborative features. WebSocket/WebRTC patterns, operational transformation/CRDTs for collaborative editing.

**Phases with standard patterns (can skip research-phase):**

- **Phase 1 (Video Splitting):** FFmpeg CLI integration is well-documented, existing codebase already has FFmpeg conversion service to use as reference. Standard patterns for process management and timeout handling.
- **Phase 1 (OSS Integration):** Aliyun OSS SDK v2 has official documentation and examples. Standard file upload/download patterns with retry logic.
- **Phase 3 (PPT Generation):** Go-pptx library has documented API. Standard PPTX file generation patterns. No exotic requirements.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Official SDK documentation verified for OSS v2 and Go-pptx. FFmpeg patterns proven in existing codebase. Tingwu manual REST approach is standard practice when no SDK exists. |
| Features | MEDIUM | Table stakes features identified from industry-standard platforms (Teams, Zoom, Otter). Some UI/UX patterns inferred from general knowledge (web search was rate-limited). Differentiator features based on Tingwu documented capabilities. |
| Architecture | HIGH | Based on existing proven architecture in codebase. Three-tier pattern with service layer and background workers is established practice. Integration points clearly defined. |
| Pitfalls | MEDIUM | OSS and FFmpeg pitfalls based on well-documented technical constraints and industry experience. Tingwu-specific pitfalls based on general async API patterns (specific API docs had 404 errors, but patterns are standard). |

**Overall confidence:** MEDIUM

Stack and architecture recommendations are HIGH confidence based on official documentation and existing proven implementation. Feature recommendations are MEDIUM confidence because table stakes are well-established but UI/UX patterns for timeline components were inferred. Pitfall identification is MEDIUM confidence because technical constraints are well-understood but Tingwu-specific issues are based on general async API patterns due to documentation gaps.

### Gaps to Address

**Areas where research was inconclusive or needs validation:**

1. **Tingwu API specifics:** Detailed API documentation for PPT extraction parameters, response formats, and error codes. Official API docs returned 404 errors. *How to handle:* During Phase 2 planning, use Aliyun support, API testing with trial account, and community forums to resolve documentation gaps. Plan for API exploration spike.

2. **Timeline UI patterns:** Detailed research on video editor timeline UX best practices. Web search was rate-limited. *How to handle:* During Phase 1 planning, research UI patterns from open source video editors (Shotcut, OpenShot, Kdenlive). Start with simple clickable timeline and iterate based on user testing.

3. **OSS integration patterns:** Best practices for temporary file hosting with auto-cleanup beyond lifecycle rules. *How to handle:* During Phase 1 implementation, reference Aliyun OSS best practices documentation, implement tracking table for uploads, and monitor for orphaned files in testing.

4. **Task queue implementation:** Go-based task queue patterns for long-running operations with database-backed persistence. *How to handle:* During Phase 2 planning, research patterns like Asynq, River, or database-backed polling. Choose based on scalability requirements (start with database-backed polling for simplicity).

5. **Search implementation:** Full-text search options for transcriptions (deferred to Phase 4). *How to handle:* During Phase 4 planning, evaluate SQLite FTS5 (simple, built-in), whoosh (Python-based, may not fit), or Elasticsearch (scalable but complex). Decision depends on transcription volume.

6. **Silence detection algorithms:** FFmpeg silence detection accuracy and threshold tuning for smart split suggestions (deferred to Phase 4). *How to handle:* During Phase 4 planning, research FFmpeg silence detection parameters, test with sample videos, and document acceptable accuracy thresholds.

## Sources

### Primary (HIGH confidence)
- **Alibaba Cloud OSS Go SDK v2 GitHub** — Official OSS SDK documentation, installation, examples
- **Muprprpr/Go-pptx GitHub** — PPTX library features, installation, examples
- **UniDoc UniOffice License** — Commercial license requirement (avoid)
- **FFmpeg Documentation** — FFmpeg seeking modes, stream copying vs re-encoding, codec options
- **Go stdlib documentation** — net/http, crypto/hmac, crypto/sha256, context, encoding/json
- **Existing Record V2 codebase** — Current FFmpeg conversion service, database models, configuration patterns, authentication middleware

### Secondary (MEDIUM confidence)
- **Aliyun Tingwu product overview** — Service capabilities, API endpoints (verified via webReader, but specific API docs had 404 errors)
- **Alibaba Cloud Credentials Guide** — Credential management for Go SDKs
- **Microsoft Teams Recording Documentation** — Industry standard features for recording and transcription
- **Zoom Smart Recording Features** — Industry standard AI companion features
- **Otter.ai and Sonix** — AI transcription software feature comparisons
- **Async Task Processing Patterns** — Polling strategies, exponential backoff, rate limiting (AWS blog, general patterns)

### Tertiary (LOW confidence)
- **Video editor timeline UX patterns** — Inferred from general knowledge (web search rate-limited, limited to forum discussions)
- **Tingwu API error codes and retry logic** — Specific API documentation unavailable (404 errors), based on general REST API patterns
- **Silence detection algorithms** — Deferred to Phase 4 research, not critical for MVP

---
*Research completed: 2025-04-17*
*Ready for roadmap: yes*
