# Feature Landscape

**Domain:** Video recording management with splitting, transcription, and PPT extraction
**Researched:** 2025-04-17
**Overall confidence:** MEDIUM

## Table Stakes

Features users expect in a meeting recording management system. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Manual split point marking** | Users need to mark specific time points for segmentation (LOW confidence - industry standard) | Low | Basic timeline UI with clickable markers |
| **Video preview while marking** | Users expect to see video while setting split points (LOW confidence - standard UX) | Medium | Requires video player integration with timeline |
| **Split into multiple segments** | Users expect to create multiple clips from one recording (LOW confidence - basic requirement) | Medium | FFmpeg `-c copy` split at keyframes |
| **Manual transcription trigger** | Users want to control when transcription happens (cost control) | Low | Simple API button/endpoint |
| **Transcription status tracking** | Users need to know task progress (pending/processing/completed/failed) | Medium | Task queue with status polling |
| **Download transcription text** | Users expect to download transcribed content | Low | File download from stored results |
| **PPT images download** | Users expect separate PPT image downloads | Low | ZIP file of extracted images |
| **Task history/audit log** | Users expect to see past transcription tasks (already have audit logging) | Low | Leverage existing audit system |
| **Error feedback** | Users expect clear error messages when tasks fail | Low | Standard error handling |
| **File size/length display** | Users need video metadata before transcription | Low | Display duration and file size |
| **OSS upload progress** | Users need feedback during large file uploads to OSS | Medium | Upload progress indicator |
| **Transcription result preview** | Users expect to preview results before download | Medium | In-app viewer for text/PPT |
| **Split preview before commit** | Users want to verify segments before finalizing | Medium | Preview mode for split points |
| **Segment naming** | Users expect to name their split segments | Low | Simple text input per segment |
| **Batch operations** | Power users expect to split multiple files at once | Medium | Queue system for batch processing |

## Differentiators

Features that set product apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Multi-point timeline UI** | Visual timeline with multiple markers vs single split (HIGH confidence - rare in basic systems) | High | Interactive timeline component |
| **Smart split suggestions** | Suggest split points based on silence detection (MEDIUM confidence - advanced feature) | High | FFmpeg silence detection + heuristics |
| **PPT extraction + transcript sync** | PPT images linked to transcript timestamps (HIGH confidence - unique value) | High | Requires parsing Tingwu output structure |
| **Segment-specific transcription** | Transcribe only selected segments (cost/time saving) | Medium | Conditional task creation |
| **Auto-chapter detection** | Use Tingwu's chapter detection for auto-splitting (MEDIUM confidence - API dependent) | Medium | Parse Tingwu chapter data |
| **Search across transcriptions** | Full-text search in transcribed content (MEDIUM confidence - valuable for meetings) | High | Search index on transcripts |
| **Speaker identification** | Tingwu provides speaker diarization (HIGH confidence - documented feature) | Low | Already in Tingwu output |
| **Keyword extraction** | Tingwu provides keyword extraction (HIGH confidence - documented feature) | Low | Already in Tingwu output |
| **Summary generation** | Tingwu provides AI summaries (HIGH confidence - documented feature) | Low | Already in Tingwu output |
| **Meeting insights** | Action items, Q&A extraction (HIGH confidence - Tingwu feature) | Medium | Parse structured output |
| **Lossless splitting** | No re-encoding means fast processing (HIGH confidence - FFmpeg capability) | Low | Use `-c copy` mode |
| **Segment preview playlists** | Create HLS playlist from segments for quick review | Medium | Generate m3u8 from split segments |
| **Transcription editing** | Allow users to correct transcription errors | High | Full text editor with timestamps |
| **Collaborative annotations** | Multiple users can add notes to segments/transcripts | High | Real-time collaboration system |
| **Auto-delete from OSS** | Cost management by removing files after transcription (HIGH confidence - cost control) | Low | Scheduled cleanup task |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| **Real-time transcription** | Out of scope - only manual trigger permitted (per PROJECT.md constraints) | Manual button to start transcription |
| **Auto-transcription after recording** | Out of scope - cost control and user choice (per PROJECT.md) | Manual trigger only |
| **AI-generated PPT from text** | Not using Tingwu's text-to-PPT feature - only PPT extraction (per PROJECT.md) | Use Tingwu's PPT extraction API |
| **Multi-language translation** | Out of scope - Tingwu translation not enabled (per PROJECT.md) | Focus on Chinese transcription |
| **Video scene detection** | Out of scope - only manual split points (per PROJECT.md) | User manually marks points |
| **Live transcription during recording** | Out of scope - offline transcription only | Process after recording completes |
| **Advanced video editing** | Not a video editor - simple splitting only | Use dedicated video editing software |
| **Transcription translation** | Out of scope - translation features disabled | Keep transcriptions in original language |
| **Automatic chapter generation** | Out of scope - manual segment control | User-defined segments only |
| **Voice activity detection for split** | Out of scope - manual markers only | User marks split points explicitly |

## Feature Dependencies

```
Video splitting features:
├── FFmpeg integration (already exists)
├── Timeline UI component (new)
├── Video player with timeline (new)
└── Segment management (new)

Transcription features:
├── Aliyun Tingwu API integration (new)
├── OSS file upload (new)
├── Task queue system (new)
├── Status tracking (new)
├── Result storage (new)
└── Notification system (already exists)

PPT extraction features:
├── Tingwu PPT extraction API (new)
├── Image storage (new)
├── ZIP file generation (new)
└── Download endpoints (new)

Cross-cutting:
├── RBAC permissions (already exists)
├── Audit logging (already exists)
├── API authentication (already exists)
└── Error handling (already exists)
```

## MVP Recommendation

Based on table stakes and complexity, prioritize in this order:

### Phase 1 - Basic Splitting (Foundation)
1. **Manual split point marking** - Timeline UI with clickable markers
2. **Video preview while marking** - Basic player integration
3. **Split into multiple segments** - FFmpeg `-c copy` implementation
4. **Segment naming** - Simple text input
5. **Split preview before commit** - Verify segments before finalizing
6. **Download segments** - File download endpoints

**Rationale:** Core splitting functionality is independent of Tingwu and provides immediate value. Establishes timeline UI patterns.

### Phase 2 - Transcription Integration (Core Value)
1. **Manual transcription trigger** - API button/endpoint
2. **OSS upload with progress** - File transfer to cloud
3. **Tingwu task submission** - API integration
4. **Transcription status tracking** - Task polling
5. **Download transcription text** - Result retrieval
6. **Error feedback** - Clear error messages
7. **Task history/audit log** - Leverage existing system

**Rationale:** Core transcription workflow. Establishes task queue patterns and OSS integration.

### Phase 3 - PPT Extraction (Differentiator)
1. **PPT extraction trigger** - Part of transcription task
2. **PPT images download** - ZIP file generation
3. **Transcription result preview** - In-app viewer
4. **PPT + transcript sync** - Link images to timestamps

**Rationale:** Builds on transcription infrastructure. Key differentiator feature.

### Defer to Future Releases
- **Smart split suggestions** - Phase 4 (requires silence detection)
- **Search across transcriptions** - Phase 4 (requires search index)
- **Segment-specific transcription** - Phase 4 (UI complexity)
- **Transcription editing** - Phase 5 (full editor needed)
- **Collaborative annotations** - Phase 5 (real-time system)
- **Auto-chapter detection** - Phase 4 (Tingwu chapter parsing)

**Reasons for deferring:**
- Smart suggestions: High complexity, optional value
- Search: Requires separate infrastructure (Elasticsearch/whoosh)
- Segments-specific: UI complexity, can batch process instead
- Editing: Requires rich text editor with timestamp handling
- Collaboration: Requires real-time backend (WebSocket/WebRTC)
- Auto-chapters: Advanced feature, manual control preferred initially

## Technical Considerations

### Video Splitting
- **Precision limitation:** FFmpeg `-c copy` only splits at keyframes (I-frames), not frame-accurate
- **Workaround accepted:** Users accept slight imprecision for speed/quality tradeoff
- **Alternative:** Re-encode for frame accuracy (slower, quality loss)
- **Multiple segments:** Can process multiple splits in one FFmpeg command or sequentially

### Tingwu Transcription
- **API version:** 2023-09-30 (per PROJECT.md)
- **Endpoint:** tingwu.cn-beijing.aliyuncs.com (per PROJECT.md)
- **Requires:** Publicly accessible video URL (hence OSS requirement)
- **Output formats:** Text, JSON with timestamps, speaker labels, PPT images
- **Processing time:** Typically 10-20% of video duration
- **Cost:** ~¥1-2 per transcription (per PROJECT.md)
- **Features:** Speaker diarization, keyword extraction, summarization (included)

### PPT Extraction
- **Tingwu capability:** Extracts PPT slide images from video
- **Output format:** Individual image files with timestamps
- **Quality:** Depends on video resolution and slide visibility
- **Synchronization:** Images linked to transcript timestamps
- **Storage:** Need to store multiple images per transcription

### OSS Integration
- **Purpose:** Temporary file hosting for Tingwu access
- **Lifecycle:** Upload → Tingwu processes → Download results → Delete from OSS
- **Cost control:** Auto-delete after transcription completes
- **Security:** Signed URLs with expiration

## Sources

### Video Splitting
- [Stack Overflow - Split video without re-encoding at timestamps](https://stackoverflow.com/questions/78743822/split-a-video-with-ffmpeg-without-reencoding-at-timestamps-given-in-a-txt-file) - HIGH confidence (technical documentation)
- [Ask Ubuntu - Auto-segment video without re-encoding](https://askubuntu.com/questions/948304/how-to-automatically-segment-video-using-ffmpeg-without-re-encoding) - HIGH confidence (technical documentation)
- [Super User - Cut video with no re-encoding](https://superuser.com/questions/1850814/how-to-cut-a-video-with-ffmpeg-with-no-or-minimal-re-encoding) - HIGH confidence (technical documentation)
- [GitHub Gist - Cut without re-encoding](https://gist.github.com/joshschmelzle/f7a34fa54a7ba1307cea1fa41577a298) - HIGH confidence (code example)
- [Brandon Pugh Blog - Trim without re-encoding](https://www.brandonpugh.com/til/video/trim-without-reencoding/) - MEDIUM confidence (blog post)
- [Video Help Forum - Cutting without re-encode](https://forum.videohelp.com/threads/406241-Cutting-away-sections-from-a-video-using-ffmpeg-without-re-encode) - MEDIUM confidence (forum discussion)

### Meeting Recording Platforms (Feature Comparison)
- [Microsoft Teams - Recording and transcription overview](https://learn.microsoft.com/en-us/microsoftteams/recording-transcription-overview) - HIGH confidence (official documentation)
- [Zoom - Smart Recording with AI Companion](https://support.zoom.com/hc/en/article?id=zm_kb&sysparm_article=KB0061101) - HIGH confidence (official documentation)
- [Panopto - Meeting recording features](https://www.panopto.com/features/meeting-recording/) - MEDIUM confidence (product page)
- [Otter.ai - AI transcript summarization](https://otter.ai/blog/ai-to-summarize-transcripts) - MEDIUM confidence (blog post)
- [Sonix - AI transcription software](https://sonix.ai/) - MEDIUM confidence (product page)
- [Video Highlight - Meeting recording features](https://videohighlight.com/features/audio-meetings) - MEDIUM confidence (product page)

### Aliyun Tingwu
- [Aliyun Tingwu product overview](https://help.aliyun.com/zh/tingwu) - HIGH confidence (official documentation, though some API pages returned 404)
- Note: Specific API documentation pages returned 404 errors, but product overview confirms capabilities

### Confidence Levels
- **Video splitting FFmpeg technical details:** HIGH - Multiple technical sources agree
- **Meeting platform feature comparison:** MEDIUM - Product pages, may be marketing-focused
- **Aliyun Tingwu capabilities:** MEDIUM - Official product overview accessible, but API docs had 404 errors
- **UI/UX patterns for timeline:** LOW - Web search rate-limited, based on general knowledge

## Gaps Requiring Phase-Specific Research

1. **Tingwu API specifics:** Detailed API documentation for PPT extraction parameters, response formats (blocked by 404 on official docs)
2. **Timeline UI patterns:** Detailed research on video editor timeline UX best practices (web search rate-limited)
3. **OSS integration patterns:** Best practices for temporary file hosting with auto-cleanup
4. **Task queue implementation:** Go-based task queue patterns for long-running operations
5. **Error handling strategies:** Tingwu API error codes and retry logic
6. **Storage optimization:** Efficient storage of PPT images and transcripts
7. **Search implementation:** Full-text search options for transcriptions (deferred feature)

### Next Steps for Roadmap

When creating phases:
1. **Phase 1 (Splitting):** Focus on UI patterns for timeline markers - may need additional research
2. **Phase 2 (Transcription):** Focus on Tingwu API integration - resolve API documentation gap
3. **Phase 3 (PPT):** Leverage Phase 2 infrastructure, focus on storage and presentation
4. **Phase 4 (Advanced):** Research silence detection algorithms, search implementation
5. **Phase 5 (Collaboration):** Research real-time systems for collaborative features

**Risk areas flagged for deeper research:**
- Timeline UI component design (MEDIUM priority - can start with simple implementation)
- Tingwu API error handling (HIGH priority - critical for core feature)
- OSS auto-cleanup reliability (HIGH priority - cost control)
- Storage scalability for PPT images (MEDIUM priority - can optimize later)
