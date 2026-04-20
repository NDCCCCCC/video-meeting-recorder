# Phase 2: Local Transcription - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-17
**Phase:** 02-local-transcription
**Areas discussed:** Frame Extraction Strategy, Similarity Detection Algorithm, PPTX Generation, Progress & UX Flow

---

## Frame Extraction Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed interval sampling | 1 frame per N seconds, simple and predictable. May miss fast transitions | |
| FFmpeg scene detection | Automatic scene change detection. Smart but uncontrollable | |
| Dual-layer strategy | Fixed sampling + similarity dedup. Two-pass filtering for higher accuracy | ✓ |

**User's choice:** Dual-layer strategy
**Notes:** User chose the most thorough approach — fixed sampling first, then similarity dedup as second filter

### Sampling Rate

| Option | Description | Selected |
|--------|-------------|----------|
| 1 frame/second | 300 frames per 10min. More precise but slower | |
| 1 frame/2 seconds | 150 frames per 10min. Balanced precision and speed | ✓ |
| 1 frame/5 seconds | 60 frames per 10min. Fast but may miss quick slides | |

**User's choice:** 1 frame/2 seconds (default), with user-adjustable presets (1s/2s/5s)

### Comparison Resolution

| Option | Description | Selected |
|--------|-------------|----------|
| Downscale to 720p | Fast comparison, sufficient for PPT text/chart detection | ✓ |
| Original resolution | More precise but much slower (SSIM scales with pixel count) | |
| Downscale to 480p | Fastest but may lose detail | |

**User's choice:** Downscale to 720p for comparison, re-extract at original resolution for PPT

### Frame Storage

| Option | Description | Selected |
|--------|-------------|----------|
| PNG temp files | Lossless but large, high disk I/O | |
| In-memory buffers | Fast but high memory usage for long videos | |
| JPEG temp files (Q95) | 5-10x smaller than PNG, negligible impact on detection | ✓ |

**User's choice:** JPEG temp files with quality 95

### Keyframe Source for PPT

| Option | Description | Selected |
|--------|-------------|----------|
| Use downscaled frames | Fast, no re-extraction, but lower quality in PPT | |
| Re-extract at original resolution | Extra FFmpeg call but highest quality PPT images | ✓ |
| Extract both at once | No re-extraction overhead but doubles disk usage | |

**User's choice:** Re-extract at original resolution for PPT quality

---

## Similarity Detection Algorithm

| Option | Description | Selected |
|--------|-------------|----------|
| Pure Go libraries | golang.org/x/image + goimagehash. No external deps, precise threshold control | ✓ |
| FFmpeg filters | ssim/signature filters. No new deps but limited (no pHash/edge support) | |
| Pixel difference histogram | Simple but ineffective for PPT text changes | |

**User's choice:** Pure Go library implementation

### Voting Logic

| Option | Description | Selected |
|--------|-------------|----------|
| Any method triggers | Never miss a change, may have extra slides | ✓ |
| Majority vote (2/3) | Cleaner PPT but may miss subtle changes | |
| Weighted scoring | Flexible but needs tuning | |

**User's choice:** OR logic (any detection triggers retention) — aligns with ROADMAP requirement

### Threshold Adjustability

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed defaults | SSIM<0.85, pHash>10, edge>0.25. Simple, ROADMAP-specified | ✓ |
| User-adjustable UI | More flexible but most users don't understand these params | |

**User's choice:** Use fixed default thresholds from ROADMAP

---

## PPTX Generation

| Option | Description | Selected |
|--------|-------------|----------|
| Muprprpr/Go-pptx | Lightweight, meets needs, smaller community | |
| unidoc/unioffice | More powerful, better docs, larger community | ✓ |
| Claude decides | Pick the most suitable library | |

**User's choice:** unidoc/unioffice

### Slide Layout

| Option | Description | Selected |
|--------|-------------|----------|
| Full-frame fill | Image fills entire slide, maximized display | ✓ |
| With margins & page numbers | More formal but wastes space | |
| With timestamp annotation | Preserves timing info but adds clutter | |

**User's choice:** Full-frame fill, no margins or decorations

### Slide Dimensions

| Option | Description | Selected |
|--------|-------------|----------|
| 16:9 widescreen | Modern standard, matches most displays | ✓ |
| 4:3 traditional | Legacy projector format | |
| Match video resolution | Non-standard PPT size | |

**User's choice:** 16:9 widescreen

---

## Progress & UX Flow

### Trigger Location

| Option | Description | Selected |
|--------|-------------|----------|
| File list action column | Inline with download/delete/split buttons. Most convenient | ✓ |
| File detail page | Requires navigation to detail page first | |
| Both locations | Most flexible but more maintenance | |

**User's choice:** File list action column

### Progress Display

| Option | Description | Selected |
|--------|-------------|----------|
| Modal popup | Shows progress, can be closed to continue browsing | ✓ |
| Dedicated status page | More info but interrupts workflow | |
| Inline in file row | Compact but may affect list layout | |

**User's choice:** Modal popup with staged progress

### Completion Behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Show complete + download button | User chooses next action | ✓ |
| Auto-download PPT | Saves a step but user may not know where file went | |
| Silent completion | Low-key, list refreshes | |

**User's choice:** Display completion status with download button in modal

### Polling Interval

| Option | Description | Selected |
|--------|-------------|----------|
| 2 seconds | Consistent with Phase 1 split polling | |
| 5 seconds | Reduces server load for longer tasks | ✓ |

**User's choice:** 5 seconds — transcription tasks take minutes, 5s is sufficient

### Progress Granularity

| Option | Description | Selected |
|--------|-------------|----------|
| Staged phases | "帧提取中..." → "画面检测中 (45/200)..." → "生成PPT..." | ✓ |
| Simple total progress | "已处理 45/200 帧 (22%)" only | |

**User's choice:** Staged phases with frame counts

---

## Claude's Discretion

- SSIM calculation implementation details
- Edge change rate detection implementation
- TranscriptionTask database model design
- API endpoint paths and request/response structures
- Temp directory management and cleanup
- goimagehash usage (dhash vs phash choice)
- Concurrent task queue design
- Individual frame error handling

## Deferred Ideas

None — discussion stayed within phase scope.
