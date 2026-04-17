# Domain Pitfalls

**Domain:** Video splitting, Aliyun Tingwu transcription, OSS integration, PPT generation
**Researched:** 2025-04-17
**Overall confidence:** MEDIUM

## Critical Pitfalls

### Pitfall 1: OSS File Orphaning

**What goes wrong:**
Temporary video files uploaded to Aliyun OSS for Tingwu transcription are never deleted, causing storage costs to accumulate indefinitely. A single 2GB meeting video left on OSS costs ~¥3-4/month per file, which escalates quickly with regular usage.

**Why it happens:**
Developers focus on the "happy path" (upload → transcribe → download results) but neglect cleanup in error scenarios:
- Transcription API failures leave uploaded files
- Service crashes during processing leave files in limbo
- Database rollback doesn't trigger OSS deletion
- No automated lifecycle rules for temporary uploads

**Prevention:**
1. **Wrap OSS operations in cleanup handlers**: Use `defer` for guaranteed cleanup, even on errors
2. **Set OSS lifecycle rules**: Configure bucket to auto-delete objects after 24-48 hours
3. **Track OSS uploads in database**: Create `OSSUpload` model to track all uploads with timestamps
4. **Periodic cleanup job**: Run cron job to delete orphaned files (>48h old, no associated task)
5. **Idempotent cleanup**: Design cleanup to be safely retryable

**Detection:**
- OSS bucket size grows consistently
- Billing shows increasing storage costs
- Database has fewer transcription tasks than OSS objects

**Phase to address:** Phase 1 (OSS Integration) — Prevents accumulation from day one

---

### Pitfall 2: Tingwu Status Polling Thundering Herd

**What goes wrong:**
Multiple transcription tasks complete simultaneously, causing the backend to barrage Tingwu API with status requests. This triggers rate limiting (429 errors), delays result retrieval, and can cause tasks to be marked as failed when they're actually completing.

**Why it happens:**
Naive polling implementation:
```go
// BAD: Fixed interval for all tasks
for task.Status != "completed" {
    time.Sleep(5 * time.Second)
    status := tingwu.GetStatus(taskID)
}
```
When 10 tasks finish at once, this creates 10 simultaneous API calls every 5 seconds.

**Prevention:**
1. **Jittered polling**: Add random ±30% variation to polling intervals
2. **Exponential backoff**: Start at 5s, increase to 30s max over time
3. **Staggered initial polls**: Don't poll all tasks immediately on startup
4. **Rate limiter**: Global rate limiter for Tingwu API calls (max 10 req/sec)
5. **Priority queue**: Poll newer tasks more frequently than older ones

**Detection:**
- Tingwu API returns HTTP 429 errors
- Tasks marked "failed" but actually completed in Tingwu console
- Status polling takes longer than expected for multiple tasks

**Phase to address:** Phase 2 (Tingwu Integration) — Test with 5+ concurrent tasks

---

### Pitfall 3: FFmpeg Keyframe Misalignment Causes Audio-Video Desync

**What goes wrong:**
Video splitting at arbitrary timestamps (not on keyframes) creates output segments with:
- First 1-2 seconds have frozen video but audio continues
- Audio drifts out of sync over long segments
- Segment duration doesn't match user-specified timing

**Why it happens:**
FFmpeg's `-c copy` (stream copy) mode only splits on keyframes (I-frames), which occur every 2-10 seconds depending on video. When users request splits at non-keyframe timestamps, FFmpeg silently rounds to nearest keyframe.

**Prevention:**
1. **Use accurate seeking with re-encoding for precise cuts**:
   ```bash
   # For precision < 1 second, re-encode
   ffmpeg -i input.mp4 -ss 00:01:23.5 -to 00:05:45.0 -c:v libx264 -c:a copy output.mp4
   ```
2. **Document keyframe limitation**: Warn users about ±2s precision with `-c copy`
3. **Two-pass approach**: Fast split for preview, re-encode for final output
4. **Analyze keyframe positions**: Use ffprobe to find actual keyframe timestamps
5. **Offer "smart split"**: Automatically adjust timestamps to nearest keyframe

**Detection:**
- Users complain segments are "slightly off"
- Video thumbnails don't match expected timestamps
- Audio-video sync issues in split segments

**Phase to address:** Phase 1 (Video Splitting) — Test with various video encodings

---

### Pitfall 4: PPT Image URL Download Timeouts

**What goes wrong:**
Tingwu returns 50-200 image URLs for PPT extraction. Downloading all sequentially takes 5-20 minutes, causing the transcription job to appear "stuck". Individual image downloads may timeout or fail, corrupting the PPT package.

**Why it happens:**
Naive sequential download:
```go
// BAD: Downloads one at a time
for _, url := range tingwuResult.ImageURLs {
    img := downloadImage(url)  // Each takes 2-5 seconds
    addImageToPPT(img)
}
```
With 100 images at 3s each = 5 minutes total, and one failure breaks everything.

**Prevention:**
1. **Parallel downloads with worker pool**: Limit to 10-20 concurrent downloads
2. **Progress tracking**: Store progress in DB, show to users via API
3. **Retry with backoff**: Failed image downloads retry 3x with exponential backoff
4. **Partial completion**: Allow PPT generation with some failed images (log warnings)
5. **Streaming PPT generation**: Start PPT creation while images download
6. **Timeout per image**: 30s timeout per image, skip on failure

**Detection:**
- Transcription "completes" but PPT generation takes 10+ minutes
- Users report "stuck" transcription jobs
- Error logs show timeout errors for image downloads

**Phase to address:** Phase 3 (PPT Generation) — Test with 100+ image URLs

---

## Moderate Pitfalls

### Pitfall 5: Database Transaction Mismatch with OSS Operations

**What goes wrong:**
Database transaction rolls back after OSS upload succeeds, leaving orphaned files on OSS. Or OSS upload fails but database records persist, causing inconsistent state.

**Why it happens:**
OSS operations can't be rolled back by database transactions. Developers wrap DB operations in transactions but forget OSS is outside that boundary.

**Prevention:**
1. **Two-phase commit pattern**: 
   - Phase 1: Upload to OSS with unique ID
   - Phase 2: Create DB record with OSS key
   - On DB failure, delete from OSS
2. **Idempotent OSS operations**: Use same key for retries
3. **Compensation actions**: On rollback, explicitly delete OSS files
4. **State machine for transcription tasks**: Track OSS upload status separately

**Detection:**
- Database shows transcription tasks with no matching OSS objects
- OSS has objects not referenced in database
- Transaction logs show rollback after OSS success

**Phase to address:** Phase 1 (OSS Integration) — Add integration tests for rollback scenarios

---

### Pitfall 6: Large Video Upload Timeout

**What goes wrong:**
Uploading 2-5GB video files to OSS exceeds HTTP timeout (default 30-60s), causing "upload failed" errors even though transfer is progressing. Users retry, creating duplicate uploads.

**Why it happens:**
Default HTTP clients have short timeouts. Large video uploads at 10MB/s take 200-500s for 2GB files.

**Prevention:**
1. **Chunked uploads**: Use OSS multipart upload API (5MB chunks)
2. **Resumable uploads**: Support pause/resume for large files
3. **Long timeouts**: Set HTTP client timeout to 30+ minutes
4. **Progress callbacks**: Report upload progress to database
5. **Deduplication**: Check if file already uploaded by content hash

**Detection:**
- Upload failures for files >1GB
- Users report "hangs" during upload
- OSS logs show partial uploads

**Phase to address:** Phase 1 (OSS Integration) — Test with 5GB files

---

### Pitfall 7: Tingwu Task ID Not Persisted

**What goes wrong:**
Service restart loses track of in-progress Tingwu transcription tasks. No way to query status or retrieve results after restart, causing "lost" transcriptions.

**Why it happens:**
Tingwu task IDs only stored in memory during polling loop. Service crash/restart loses this state.

**Prevention:**
1. **Persist Tingwu task ID immediately**: Store in DB when transcription submitted
2. **Resume polling on startup**: Query DB for "submitted" tasks and resume polling
3. **Task status synchronization**: Sync DB status with Tingwu status on every poll
4. **Idempotent status checks**: Handle duplicate status responses

**Detection:**
- Service restart causes "lost" transcriptions
- Tingwu console shows completed tasks not reflected in system
- Database has tasks stuck in "submitted" state

**Phase to address:** Phase 2 (Tingwu Integration) — Test service restart during active transcriptions

---

## Minor Pitfalls

### Pitfall 8: FFmpeg Process Leaks

**What goes wrong:**
FFmpeg processes spawned for video splitting aren't properly cleaned up on errors or timeouts, causing zombie processes that consume CPU/memory.

**Why it happens:**
Go's `exec.Command` creates subprocesses. If parent process crashes or context isn't properly canceled, subprocesses become orphaned.

**Prevention:**
1. **Always use context.WithTimeout**: Set 30-60 minute timeout for split operations
2. **Kill process on error**: Explicit `cmd.Process.Kill()` in error handlers
3. **Process cleanup on startup**: Scan for orphaned FFmpeg processes on service start
4. **Resource limits**: Set FFmpeg `MaxProcesses` in config (already exists)

**Detection:**
- Server shows high CPU usage when idle
- `ps aux | grep ffmpeg` shows many old processes
- Video splitting tasks hang indefinitely

**Phase to address:** Phase 1 (Video Splitting) — Add process monitoring

---

### Pitfall 9: PPT File Path Not Validated

**What goes wrong:**
PPT generation fails because file paths contain invalid characters, exceed length limits, or use reserved names (e.g., `CON`, `PRN` on Windows).

**Why it happens:**
User input for filenames isn't sanitized before file system operations.

**Prevention:**
1. **Filename sanitization**: Replace invalid chars (`/`, `\`, `:`, `*`, `?`, `<`, `>`, `|`) with `_`
2. **Length limits**: Truncate to 255 chars (common FS limit)
3. **Reserved names**: Check OS-specific reserved names
4. **Path validation**: Verify path is within allowed directory (prevent directory traversal)

**Detection:**
- PPT generation fails with "invalid path" errors
- Windows-specific failures for certain filenames
- Directory traversal attempts in logs

**Phase to address:** Phase 3 (PPT Generation) — Add filename validation tests

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| **Synchronous polling** (no message queue) | Simpler implementation, no infrastructure | Server must be always running, hard to scale horizontally | Never — use database-backed polling from day one |
| **Hardcoded Tingwu endpoints** | Faster initial development | Difficult to test, can't switch regions | Only for MVP prototype |
| **Inline PPT generation** (no separate service) | Single codebase, simpler deployment | PPT failures block transcription UI | Only if PPT generation is fast (<2 min) |
| **No download retry logic** | Simpler error handling | Failed downloads require manual intervention | Never — users expect reliability |
| **Skip OSS cleanup in tests** | Faster test execution | Test buckets accumulate junk files | Acceptable with automated cleanup job |
| **Use `-c copy` for all splits** | Fast splitting, no quality loss | Imprecise cuts, keyframe limitations | Acceptable for preview, add re-encode option later |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| **Aliyun OSS** | Using HTTP client with 30s timeout for large file uploads | Use multipart upload API with chunked transfer and 30+ min timeout |
| **Aliyun OSS** | Not setting lifecycle rules for temporary files | Configure bucket rule: auto-delete objects with prefix `temp/` after 48h |
| **Tingwu API** | Polling status every 5s for all tasks | Use exponential backoff starting at 10s, jittered intervals, max 60s |
| **Tingwu API** | Not handling 429 rate limit responses | Implement retry with exponential backoff, respect `Retry-After` header |
| **Tingwu API** | Assuming synchronous transcription completion | Use async pattern: submit → poll status → fetch results |
| **FFmpeg** | Splitting at arbitrary timestamps with `-c copy` | Use accurate seek with re-encoding for precision, or warn users about keyframe rounding |
| **FFmpeg** | Not killing subprocess on context cancellation | Always defer `cmd.Process.Kill()` in error handlers |
| **OSS + Tingwu** | Passing private OSS URL to Tingwu API | Generate signed URL with 24h expiry, or make bucket public for Tingwu IPs |
| **Database + OSS** | Wrapping only DB ops in transactions | Use two-phase commit: OSS upload → DB record → (rollback + OSS delete) on error |

---

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| **Sequential image downloads** | PPT generation takes 10-20 minutes for 100 images | Use worker pool with 10-20 concurrent downloads, progress tracking | Immediately with >20 images |
| **Fixed-interval status polling** | API rate limits, thundering herd on task completion | Exponential backoff with jitter, rate limiter | With >5 concurrent tasks |
| **No OSS upload progress tracking** | Users think upload is frozen, retry causing duplicates | Emit progress events, store in DB, show via API | For files >500MB |
| **Re-encoding all video splits** | Splitting 1-hour video takes 30+ minutes | Use `-c copy` for preview, offer re-encode option for final | For videos >30 min |
| **Not cleaning up temp files** | Disk space fills up, crashes recording service | Periodic cleanup job, monitor disk usage, enforce quotas | After 100+ recordings |
| **Single-threaded FFmpeg operations** | Only 1 video split at a time despite multiple CPU cores | FFmpeg worker pool (already exists for conversion, reuse for splitting) | When >3 concurrent splits |

---

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| **OSS credentials in config file** | Credentials leaked if repo is public | Use environment variables or encrypted secrets |
| **Signed URLs with long expiry** | If leaked, allows unauthorized access to videos | Use 1-24h expiry, IP restriction if possible |
| **Tingwu API key in logs** | API key leaked in logs/error reports | Sanitize API keys from logs, use structured logging |
| **No validation of Tingwu webhooks** (if used) | Fake webhook requests could corrupt database | Verify HMAC signatures if implementing webhooks |
| **PPT download allows directory traversal** | Could download system files | Validate paths are within allowed directory |
| **No rate limiting on transcription submission** | User could submit 1000 tasks and exhaust quota | Per-user rate limits (e.g., 10 tasks/hour) |

---

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| **No progress indication for long operations** | Users think system is broken, retry causing duplicates | Store progress in DB, query via API, show progress bar in UI |
| **Precision not communicated for video splits** | Users expect frame-perfect cuts, get ±2s offsets | Show actual split points after keyframe rounding, offer re-encode option |
| **PPT download only after full completion** | 20-minute wait with no feedback | Stream PPT generation, allow partial download with warnings |
| **No error recovery for partial failures** | Single failed image breaks entire PPT | Generate PPT with placeholders for failed images, log warnings |
| **Transcription status only shows "processing"** | No indication if Tingwu is actually working or stuck | Show detailed status: "uploading → transcribing (75%) → extracting PPT (30/100 images)" |
| **No cost warning before transcription** | Users surprised by OSS + Tingwu fees | Show estimated cost before submission, track spending |

---

## "Looks Done But Isn't" Checklist

- [ ] **Video splitting**: Often missing keyframe alignment — verify split timestamps match user request within acceptable tolerance
- [ ] **OSS uploads**: Often missing cleanup on error — verify orphaned files don't accumulate after failed operations
- [ ] **Tingwu polling**: Often missing backoff strategy — verify no 429 errors under load
- [ ] **PPT generation**: Often missing progress tracking — verify users see progress for multi-minute operations
- [ ] **Error recovery**: Often missing retry logic — verify transient failures (network, timeout) are retried
- [ ] **Service restarts**: Often missing state persistence — verify in-progress tasks resume after restart
- [ ] **Large files**: Often missing chunked uploads — verify 2GB+ files upload successfully
- [ ] **Concurrent operations**: Often missing resource limits — verify 10+ simultaneous splits don't crash system
- [ ] **Cost tracking**: Often missing usage metrics — verify can track OSS + Tingwu costs per user/task
- [ ] **Filename handling**: Often missing sanitization — verify special characters don't break file operations

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| **OSS file orphaning** | MEDIUM | 1. Query DB for all transcription tasks<br>2. List OSS objects with `temp/` prefix<br>3. Delete objects not in DB or >48h old<br>4. Add lifecycle rule to prevent recurrence |
| **Tingwu task lost** | LOW | 1. Query DB for tasks stuck in "submitted"<br>2. Manually check Tingwu console for task status<br>3. Update DB to actual status or mark as failed<br>4. Implement task ID persistence |
| **PPT image download timeout** | LOW | 1. Check Tingwu result for failed image URLs<br>2. Retry download for failed images<br>3. Generate PPT with available images<br>4. Add timeout and parallel download |
| **FFmpeg process leaks** | LOW | 1. `ps aux | grep ffmpeg` to find orphaned processes<br>2. Kill orphaned processes<br>3. Add process cleanup on startup<br>4. Implement context cancellation |
| **Database-OSS inconsistency** | HIGH | 1. Compare DB records with OSS bucket listing<br>2. Delete OSS objects not in DB<br>3. Create DB records for orphaned OSS objects<br>4. Implement two-phase commit |
| **Disk space full from temp files** | MEDIUM | 1. Identify temp directories consuming space<br>2. Delete files older than 24h<br>3. Add automated cleanup job<br>4. Implement disk usage monitoring |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| **OSS file orphaning** | Phase 1 (OSS Integration) | Test: Upload → crash service → verify cleanup on restart + lifecycle rules configured |
| **Tingwu polling thundering herd** | Phase 2 (Tingwu Integration) | Test: Submit 10 tasks simultaneously → verify no 429 errors → verify polling intervals are jittered |
| **FFmpeg keyframe misalignment** | Phase 1 (Video Splitting) | Test: Split at non-keyframe timestamps → verify output timing matches request ±2s with `-c copy` |
| **PPT image download timeouts** | Phase 3 (PPT Generation) | Test: Mock 100 image URLs → verify parallel downloads → verify completion <5 minutes |
| **Database-OSS transaction mismatch** | Phase 1 (OSS Integration) | Test: Upload → trigger DB rollback → verify OSS file deleted |
| **Large video upload timeout** | Phase 1 (OSS Integration) | Test: Upload 5GB file → verify completes with chunked upload + progress tracking |
| **Tingwu task ID not persisted** | Phase 2 (Tingwu Integration) | Test: Submit task → restart service → verify polling resumes using persisted task ID |
| **FFmpeg process leaks** | Phase 1 (Video Splitting) | Test: Split video → cancel operation → verify no orphaned FFmpeg processes |
| **PPT filename not validated** | Phase 3 (PPT Generation) | Test: Generate PPT with special chars in filename → verify sanitization works |
| **No cost warning** | Phase 2 (Tingwu Integration) | Test: Submit transcription → verify UI shows estimated cost before submission |

---

## Sources

- [Aliyun OSS Pricing and Data Processing Fees](https://help.aliyun.com/zh/oss/data-processing-fees) — OSS pricing structure, hidden processing fees for video operations (HIGH confidence)
- [Aliyun OSS Best Practices](https://zhuanlan.zhihu.com/p/1890848177072099423) — OSS lifecycle management, cost optimization strategies (MEDIUM confidence)
- [Aliyun Tingwu Official Documentation](https://tingwu.aliyun.com) — Tingwu API usage, status polling patterns (HIGH confidence)
- [FFmpeg Documentation](https://ffmpeg.org/documentation.html) — FFmpeg seeking modes, stream copying vs re-encoding (HIGH confidence)
- [Go exec.Command Best Practices](https://pkg.go.dev/os/exec) — Proper subprocess cleanup, context cancellation (HIGH confidence)
- [Async Task Processing Patterns](https://aws.amazon.com/blogs/architecture/pattern-for-building-background-tasks-with-amazon-sqs-and-lambda/) — Polling strategies, exponential backoff, rate limiting (MEDIUM confidence)
- [Multipart Upload Best Practices](https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html) — Chunked upload pattern for large files (MEDIUM confidence, AWS S3 but applicable to OSS)
- [Existing codebase analysis] — Current FFmpeg conversion service implementation, database models, configuration patterns (HIGH confidence)
- [Industry experience] — Common video processing pitfalls, cloud storage integration issues (MEDIUM confidence)

---

*Pitfalls research for: Video splitting, Aliyun Tingwu transcription, OSS integration, PPT generation*
*Researched: 2025-04-17*
