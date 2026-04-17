# Phase 1: Video Splitting - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-17
**Phase:** 01-video-splitting
**Areas discussed:** Split Marker Interaction, Split Precision vs Speed, Recording Snapshot Experience, Segment Management & Auto Scan

---

## Split Marker Interaction

| Option | Description | Selected |
|--------|-------------|----------|
| Click + drag (recommended) | Click timeline to add markers, drag to reposition, click marker for delete/edit. Most intuitive, like video editors. | |
| Timestamp input only | Manually enter timestamps (01:23:45) in a split point list. Precise but slow interaction. | |
| Both | Click to add + manual timestamp input. Most flexible but more complex UI. | ✓ |

**User's choice:** Both — click-to-add and manual timestamp input supported
**Notes:** User wants maximum flexibility for adding split points

| Option | Description | Selected |
|--------|-------------|----------|
| Zoom + frame-level micro-adjust (recommended) | Timeline zoom + arrow key frame stepping. Video editor precision. | |
| Second-level only | Markers align to seconds. Simple, lightweight implementation. | ✓ |
| Zoom + 0.1s step | Timeline zoom + 0.1s step adjustment. Compromise. | |

**User's choice:** Second-level precision is sufficient
**Notes:** Keeps implementation simple — no need for frame-level controls

| Option | Description | Selected |
|--------|-------------|----------|
| Vertical lines + hover tooltip (recommended) | Markers as vertical lines on timeline, hover shows time, click for actions. Clean. | ✓ |
| Markers with time labels | Markers show timestamp labels below, each segment shows duration. More info but visually dense. | |

**User's choice:** Vertical line markers with hover tooltips
**Notes:** Clean visual style, consistent with minimalist approach

---

## Split Precision vs Speed

| Option | Description | Selected |
|--------|-------------|----------|
| Default fast + re-encode option (recommended) | Default -c copy (fast, ±2s error), offer re-encode if precision insufficient. | ✓ |
| Always fast copy | Always use -c copy, no re-encode option. Simple but no precision fallback. | |
| Choose mode before each split | User picks "fast" or "precise" before splitting. Flexible but adds cognitive load. | |

**User's choice:** Default fast copy with re-encode option when precision is insufficient
**Notes:** Users understand the speed/precision tradeoff and can opt into re-encode when needed

---

## Recording Snapshot Experience

| Option | Description | Selected |
|--------|-------------|----------|
| Task list row button (recommended) | Inline button on active recording task row. Click → confirm → background export → notify. | ✓ |
| HLS preview page button | Button on live preview page. Capture while watching live stream. | |
| Both locations | Button in both task list and preview page. Broader coverage. | |

**User's choice:** Inline button on the active recording task row
**Notes:** Simple and direct — user doesn't need to be watching the preview to take a snapshot

| Option | Description | Selected |
|--------|-------------|----------|
| Button state change + notification (recommended) | Button changes to "生成中...", notification on completion. Non-intrusive. | ✓ |
| Progress modal + download link | Modal with progress percentage, download link on completion. More detailed but interrupts workflow. | |

**User's choice:** Button state change with notification
**Notes:** Uses existing notification system — consistent with app patterns

---

## Segment Management & Auto Scan

| Option | Description | Selected |
|--------|-------------|----------|
| Independent records + parent link (recommended) | Segments as independent VideoFile records with parent_id. Visible in file list, independently manageable. | ✓ |
| Sub-list under original video | Segments only visible in a sub-list under the source video. Cleaner but harder to find. | |
| Both | Main list + sub-list under original. Full coverage but more development. | |

**User's choice:** Independent VideoFile records with parent_id linking to source video
**Notes:** Consistent with existing file management patterns, segments are fully independent

| Option | Description | Selected |
|--------|-------------|----------|
| Service callback (recommended) | Recording/conversion/split services call VideoFileService directly. Event-driven, real-time. | ✓ |
| File system watcher (fsnotify) | Watch recordings directory for new MP4 files. Decoupled but race condition risks. | |
| Callback + periodic scan | Callback primary + periodic scan fallback. Double insurance but more complexity. | |

**User's choice:** Service callback — direct call from producing services to VideoFileService
**Notes:** Most reliable approach, leverages existing service layer patterns

| Option | Description | Selected |
|--------|-------------|----------|
| Extend existing file list (recommended) | Add "来源" column and parent video link to existing file list. Simple, no new pages. | ✓ |
| Embedded result list in split page | Split page shows results inline. Better split experience but duplicates file management. | |

**User's choice:** Extend existing file list with source info
**Notes:** Minimal UI changes, consistent with existing patterns

---

## Claude's Discretion

- Timeline marker component implementation approach
- FFmpeg command construction for split and snapshot
- Snapshot extraction technique (partial MKV copy vs dual-output)
- Database migration for parent_id and source_type fields
- API endpoint design and route organization
- Segment naming convention
- File storage paths for split segments
- Re-encode trigger UI design
- Split page navigation (how user gets to the split interface)

## Deferred Ideas

No scope creep during discussion — all areas stayed within Phase 1 boundary.
