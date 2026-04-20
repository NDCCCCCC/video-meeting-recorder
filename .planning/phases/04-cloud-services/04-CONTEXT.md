# Phase 4: Cloud Services - Context

**Gathered:** 2026-04-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Integrate Aliyun OSS for file relay (upload video to get public URL) and Aliyun Tingwu for cloud transcription, with cloud/local mode switching via dropdown button, automatic fallback from cloud to local when initial submission fails, and transcription text content display with timestamps.

Requirements covered: OSS-01, OSS-02, TRAN-01 (cloud option), TRAN-02, TRAN-03, TRAN-05, UI-02.

</domain>

<decisions>
## Implementation Decisions

### Transcription Mode Selection (TRAN-02)
- **D-01:** "转录" button becomes Ant Design Dropdown button with two options: "本地转录" and "云端转录（通义听悟）". Click directly starts — no extra modal step.
- **D-02:** Result page's "重新转录" button also becomes dropdown with the same two options, consistent with file list.
- **D-03:** Cloud transcription requires no sampling rate parameter — click to start immediately. Local transcription keeps the existing sampling rate selection flow in TranscriptionProgressModal.

### Cloud Status Tracking & Polling (TRAN-04)
- **D-04:** Tingwu API provides detailed progress via status API. Backend polls Tingwu with exponential backoff and caches status in-memory (same statusMap pattern as Phase 2).
- **D-05:** Frontend polls backend at 10-second intervals for cloud transcription progress (lighter than local's 5s — cloud tasks take minutes).
- **D-06:** Reuse existing TranscriptionProgressModal for cloud mode. Cloud stages differ from local: "上传中" (0-20%) → "排队中" → "处理中" (20-90%) → "下载结果" (90-100%). Stage rendering adapts based on mode.

### Cloud Fallback (TRAN-03)
- **D-07:** Auto-fallback only triggers at initial submission stage — OSS upload failure or Tingwu API rejection. Mid-processing failures (Tingwu returns error after accepting) do NOT auto-fallback; they mark the task as failed and user can manually retry with either mode.
- **D-08:** Seamless transition — progress modal auto-switches to local mode with an info alert "云端转录失败，已自动切换到本地转录". Local transcription stages then display normally.

### Text Content Display (TRAN-05)
- **D-09:** Result page right panel adds "文字内容" tab alongside existing info/preview sections. Tab switching between basic info and text content in the same panel area.
- **D-10:** Clickable timestamps — each text segment shows `[HH:MM:SS]` prefix. Clicking a timestamp jumps to the corresponding video position (if video player is available). Interactive timeline-text linking.
- **D-11:** Copy functionality: "复制全部文字" button at top + per-segment copy icon. Users can easily extract text to other tools.

### Claude's Discretion
- OSS Go SDK v2 integration details (multipart upload, presigned URL generation)
- Tingwu REST API client implementation (HMAC-SHA256 signing, request/response structs)
- TranscriptionTask model extensions (Mode, CloudTaskID, OSSURL fields)
- Backend polling interval for Tingwu status (exponential backoff parameters)
- OSS cleanup mechanism (scheduled task vs callback vs lifecycle rule)
- Config struct additions for OSS and Tingwu credentials
- API endpoint design for cloud transcription (reuse existing /transcribe with mode param or new endpoints)
- Text content storage model (new table or embed in TranscriptionTask)
- TranscriptionProgressModal stage adaptation logic (conditional rendering based on mode)
- Error classification for fallback trigger (which errors = initial stage vs mid-processing)
- Dropdown button implementation details (Ant Design Dropdown.Button or SplitButton)

### Folded Todos
None.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Cloud services requirements
- `.planning/REQUIREMENTS.md` §OSS Integration — OSS-01, OSS-02 acceptance criteria
- `.planning/REQUIREMENTS.md` §Transcription — TRAN-01 (cloud option), TRAN-02, TRAN-03, TRAN-05 acceptance criteria
- `.planning/REQUIREMENTS.md` §UI Layout — UI-02 acceptance criteria (转录任务列表页面)
- `.planning/ROADMAP.md` §Phase 4 — Success criteria and phase boundary

### Phase 2 context (prior decisions and patterns)
- `.planning/phases/02-local-transcription/02-CONTEXT.md` — TranscriptionService worker pool, statusMap pattern, "转录" button, TranscriptionProgressModal, 5s polling, TranscriptionTask model, PPTX generation

### Phase 3 context (prior decisions and patterns)
- `.planning/phases/03-ppt-management/03-CONTEXT.md` — Result page layout (left PPT preview + right info panel), gallery switcher, "重新转录" button placement, slide extraction pattern

### Phase 1 context (prior decisions)
- `.planning/phases/01-video-splitting/01-CONTEXT.md` — File list action column pattern, auto-refresh

### Existing code to reuse or extend
- `internal/services/transcription_service.go` — Worker pool, statusMap, processTranscription pipeline (extend with cloud mode branch)
- `internal/handlers/transcription_handler.go` — Transcription API (add mode parameter)
- `internal/models/transcription_task.go` — TranscriptionTask model (extend with Mode, CloudTaskID fields)
- `internal/services/pptx_generator.go` — PPTX generation (cloud transcription may also produce PPT)
- `internal/config/config.go` — Config infrastructure (add OSS/Tingwu config sections)
- `frontend/src/components/TranscriptionProgressModal.tsx` — Reuse for cloud mode with adapted stages
- `frontend/src/pages/files/index.tsx` — File list "转录" button (convert to dropdown)
- `frontend/src/pages/results/index.tsx` — Result page (add text content tab, convert "重新转录" to dropdown)
- `frontend/src/api/transcription.ts` — API client (add mode parameter)
- `frontend/src/types/transcription.ts` — Type definitions (add cloud stages, mode enum)
- `cmd/server/app.go` — Service/handler registration pattern
- `.env` — Already has Tingwu APP_KEY and OSS credential placeholders

### Project constraints
- `.planning/PROJECT.md` — Tech stack (Go 1.24/Gin, React 19/Ant Design 6, SQLite/GORM), all Go implementation, OSS as file relay, manual HMAC-SHA256 signing for Tingwu
- `.planning/STATE.md` — Tech stack context, critical pitfalls (OSS file orphaning, Tingwu polling thundering herd, DB/OSS transaction mismatch)

No external specs — requirements are fully captured in REQUIREMENTS.md and PROJECT.md.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **TranscriptionService** (`internal/services/transcription_service.go`): Worker pool with task channel, in-memory statusMap with sync.RWMutex, cancellable per-task contexts. The cloud branch can reuse the same statusMap and polling infrastructure.
- **TranscriptionHandler** (`internal/handlers/transcription_handler.go`): POST /transcribe + GET /transcription-status pattern. Add `mode` field to request, route to local or cloud pipeline.
- **TranscriptionTask model** (`internal/models/transcription_task.go`): Has VideoFileID, Status, CurrentStage, Percentage fields. Needs extension: Mode (local/cloud), CloudTaskID (Tingwu task ID), OSSURL.
- **TranscriptionProgressModal** (`frontend/src/components/TranscriptionProgressModal.tsx`): Already renders 3 local stages. Adapt to detect mode and render cloud stages (uploading → queued → processing → downloading).
- **Config infrastructure** (`internal/config/config.go`): Viper-based config with env var expansion. Add OSSConfig (endpoint, bucket, keys) and TingwuConfig (appKey, API version, domain).
- **.env file**: Tingwu APP_KEY already filled. OSS credential placeholders exist.

### Established Patterns
- **Worker pool services**: Struct-based service with task channel, N workers, status map, Start/Stop lifecycle
- **API handlers**: Gin handlers with `ShouldBindQuery`/`ShouldBindJSON`, unified `response.GinSuccess/GinError`
- **Frontend state**: Local useState for page state, API calls via `apiRequest<T>()`
- **Frontend polling**: setInterval for progress tracking, clear on complete/fail
- **Config**: Viper with YAML + env var expansion, prefix `RECORD_`

### Integration Points
- **TranscriptionService**: Add cloud transcription pipeline branch alongside existing local pipeline
- **TranscriptionHandler**: Add `mode` parameter to POST /transcribe request
- **TranscriptionTask model**: Add Mode, CloudTaskID, OSSURL fields via migration
- **New: OSSService**: Upload, generate presigned URL, delete — new service
- **New: TingwuClient**: REST API client with HMAC-SHA256 signing — new service
- **Frontend dropdown**: Convert "转录" button to Ant Design Dropdown.Button in file list and result page
- **TranscriptionProgressModal**: Adapt stage rendering for cloud mode
- **Result page**: Add text content tab with clickable timestamps and copy functionality
- **cmd/server/app.go**: Register new OSS/Tingwu services, update handler creation
- **config.go**: Add OSSConfig and TingwuConfig structs

</code_context>

<specifics>
## Specific Ideas

- Dropdown button should visually indicate two modes clearly — "本地转录" with a standard icon, "云端转录" with a cloud icon
- Cloud stages in progress modal should feel distinct from local stages — perhaps different progress bar color or stage icon
- Text content tab should have a clean, readable layout — monospace timestamps, paragraph-style text segments
- Timestamp click-to-jump is a nice UX touch — makes text content actionable, not just passive reading

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 04-cloud-services*
*Context gathered: 2026-04-17*
