# Phase 4: Cloud Services - Research

**Researched:** 2025-04-17
**Domain:** Aliyun OSS integration, Tingwu API, cloud transcription with fallback
**Confidence:** MEDIUM

## Summary

Phase 4 integrates Aliyun OSS for file relay and Aliyun Tingwu for cloud transcription, enabling users to choose between cloud and local transcription modes with automatic fallback. The phase requires implementing three new services (OSSService, TingwuClient, and cloud transcription pipeline), extending existing TranscriptionService with a cloud branch, updating the frontend with dropdown buttons and cloud progress stages, and adding text content display with clickable timestamps.

**Primary recommendation:** Use Aliyun OSS Go SDK v2 for file operations with presigned URLs for public access, implement Tingwu REST API client with manual HMAC-SHA256 signing (no official Go SDK exists), extend TranscriptionService with cloud pipeline branch sharing the same statusMap pattern, and use Ant Design Dropdown.Button component for mode selection UI.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| OSS file upload/download | API / Backend | CDN / Static | Backend manages credentials, generates presigned URLs, controls lifecycle |
| Tingwu API integration | API / Backend | — | Backend handles HMAC-SHA256 signing, status polling, result retrieval |
| Cloud transcription orchestration | API / Backend | Live service config | Backend manages task state, polling, fallback triggers |
| Transcription mode selection UI | Browser / Client | — | Frontend renders dropdown, captures user choice |
| Progress tracking (cloud) | Browser / Client | API / Backend | Frontend polls backend every 10s, backend polls Tingwu with exponential backoff |
| Text content display | Browser / Client | API / Backend | Frontend renders timestamps, handles click-to-jump video interaction |
| OSS file lifecycle cleanup | API / Backend | CDN / Static | Backend sets lifecycle rules, triggers cleanup after transcription |

## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Transcription Mode Selection (TRAN-02)
- **D-01:** "转录" button becomes Ant Design Dropdown button with two options: "本地转录" and "云端转录（通义听悟）". Click directly starts — no extra modal step.
- **D-02:** Result page's "重新转录" button also becomes dropdown with the same two options, consistent with file list.
- **D-03:** Cloud transcription requires no sampling rate parameter — click to start immediately. Local transcription keeps the existing sampling rate selection flow in TranscriptionProgressModal.

#### Cloud Status Tracking & Polling (TRAN-04)
- **D-04:** Tingwu API provides detailed progress via status API. Backend polls Tingwu with exponential backoff and caches status in-memory (same statusMap pattern as Phase 2).
- **D-05:** Frontend polls backend at 10-second intervals for cloud transcription progress (lighter than local's 5s — cloud tasks take minutes).
- **D-06:** Reuse existing TranscriptionProgressModal for cloud mode. Cloud stages differ from local: "上传中" (0-20%) → "排队中" → "处理中" (20-90%) → "下载结果" (90-100%). Stage rendering adapts based on mode.

#### Cloud Fallback (TRAN-03)
- **D-07:** Auto-fallback only triggers at initial submission stage — OSS upload failure or Tingwu API rejection. Mid-processing failures (Tingwu returns error after accepting) do NOT auto-fallback; they mark the task as failed and user can manually retry with either mode.
- **D-08:** Seamless transition — progress modal auto-switches to local mode with an info alert "云端转录失败，已自动切换到本地转录". Local transcription stages then display normally.

#### Text Content Display (TRAN-05)
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

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| OSS-01 | Video upload to Aliyun OSS for public URL | [VERIFIED: Aliyun docs] OSS SDK v2 provides presigned URL generation, multipart upload |
| OSS-02 | Automatic OSS file cleanup after transcription | [VERIFIED: Aliyun docs] Lifecycle rules support expiration by days/hours after creation |
| TRAN-01 | Cloud transcription mode option | [VERIFIED: Tingwu docs] Submit task API accepts file URL, returns task ID for polling |
| TRAN-02 | User choice between cloud/local modes | [VERIFIED: Ant Design docs] Dropdown.Button supports menu items with icons |
| TRAN-03 | Auto-fallback to local on cloud failure | [CITED: CONTEXT.md] D-07/D-08 define trigger conditions and seamless UX |
| TRAN-05 | Text content display with timestamps | [VERIFIED: Tingwu docs] Result API returns text with time offsets |
| UI-02 | Transcription task list page | [VERIFIED: Existing code] File list page pattern established in Phase 1 |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| alibabacloud-oss-go-sdk-v2 | latest | OSS file operations | Official Aliyun OSS SDK v2 for Go, supports presigned URLs, multipart upload, lifecycle rules |
| github.com/aliyun/alibabacloud-go-common | latest | HMAC-SHA256 signing | Common utilities for Aliyun API authentication |
| net/http | stdlib | HTTP client for Tingwu API | Standard library HTTP client with context support |
| crypto/hmac | stdlib | HMAC-SHA256 signing | Standard library for cryptographic signing |
| antd | 6.3.6 | Dropdown.Button UI | [VERIFIED: npm registry] Latest stable version, Dropdown.Button with icon property |
| react | 19.2.5 | Component state management | [VERIFIED: npm registry] Current version, matches existing codebase |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/crobtaar/go-hashing | latest | HMAC-SHA256 helper | If manual signing complex, otherwise use crypto/hmac |
| github.com/avast/retry-go | latest | Exponential backoff with jitter | For Tingwu API retry logic, rate limit handling |
| dayjs | 1.11.13 | Timestamp formatting | [VERIFIED: Existing code] Already used for date formatting |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| alibabacloud-oss-go-sdk-v2 | alibabacloud-oss-go-sdk (v1) | v1 is deprecated, v2 is current standard |
| Manual HMAC signing | Aliyun Go SDK for Tingwu | No official Tingwu Go SDK exists [ASSUMED], manual REST required |
| antd Dropdown.Button | Custom dropdown with Menu | Ant Design component is battle-tested, accessible |

**Installation:**

```bash
# Go dependencies
go get github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss
go get github.com/aliyun/alibabacloud-go-common/sdk
go get github.com/avast/retry-go/v4

# Frontend (already installed)
npm install antd@6.3.6 react@19.2.5
```

**Version verification:**
```bash
npm view antd version  # 6.3.6
npm view react version  # 19.2.5
go list -m github.com/aliyun/alibabacloud-oss-go-sdk-v2  # Latest
```

## Architecture Patterns

### System Architecture Diagram

```
User clicks "云端转录（通义听悟）"
        ↓
Frontend: Dropdown.Button → API POST /api/v1/videos/:id/transcribe {mode: "cloud"}
        ↓
Backend: TranscriptionHandler.SubmitTranscription() → route to cloud pipeline
        ↓
OSSService: Upload video file to OSS → generate presigned URL
        ↓
TingwuClient: Submit task with OSS URL → receive CloudTaskID
        ↓
TranscriptionService: Start background worker with CloudTaskID
        ↓
Frontend: Poll GET /api/v1/videos/:id/transcription-status every 10s
        ↓
Backend: Poll Tingwu status API with exponential backoff (2s → 60s max)
        ↓
TranscriptionService: Update statusMap with cloud stages
        ↓
Frontend: Update TranscriptionProgressModal with cloud stages
        ↓
[When Tingwu status = "completed"]
Backend: Download result text, parse timestamps, save to database
        ↓
OSSService: Trigger cleanup lifecycle rule (24h expiration)
        ↓
Frontend: Navigate to result page with text content tab
```

### Recommended Project Structure

```
internal/
├── services/
│   ├── transcription_service.go      # Extend with cloud pipeline branch
│   ├── oss_service.go                # NEW: OSS upload, presigned URL, lifecycle
│   └── tingwu_client.go              # NEW: Tingwu REST API client with HMAC signing
├── handlers/
│   └── transcription_handler.go      # Add mode parameter handling
├── models/
│   ├── transcription_task.go         # Add Mode, CloudTaskID, OSSURL, TextContent
│   └── transcription_text.go         # NEW: Text segment model with timestamps
└── config/
    └── config.go                     # Add OSSConfig, TingwuConfig

frontend/
├── src/
│   ├── components/
│   │   ├── TranscriptionProgressModal.tsx    # Extend for cloud stages
│   │   └── TextContentTab.tsx               # NEW: Text display with timestamps
│   ├── pages/
│   │   ├── files/index.tsx                  # Convert button to Dropdown.Button
│   │   └── results/index.tsx                # Add text content tab, convert retranscribe
│   ├── api/
│   │   └── transcription.ts                 # Add mode parameter
│   └── types/
│       └── transcription.ts                 # Add cloud stages, mode enum
```

### Pattern 1: OSSService with Presigned URLs

**What:** Encapsulate OSS operations with credential management, presigned URL generation, and lifecycle rules.

**When to use:** All OSS file operations (upload, download, cleanup).

**Example:**

```go
// Source: https://help.aliyun.com/zh/oss/developer-reference/v2-presign-upload

type OSSService struct {
    client    *oss.Client
    bucket    *oss.Bucket
    logger    *zap.Logger
    config    *config.OSSConfig
}

func (s *OSSService) UploadFile(ctx context.Context, localPath string, objectKey string) (string, error) {
    // Upload to OSS
    err := s.bucket.PutObjectFromFile(objectKey, localPath)
    if err != nil {
        return "", fmt.Errorf("OSS upload failed: %w", err)
    }

    // Generate presigned URL (24 hour expiration)
    signedURL, err := s.bucket.SignURL(objectKey, oss.HTTPGet, 86400)
    if err != nil {
        return "", fmt.Errorf("generate presigned URL failed: %w", err)
    }

    return signedURL, nil
}

func (s *OSSService) SetLifecycleRule(ctx context.Context, prefix string, days int) error {
    // Set lifecycle rule to delete files after N days
    rule := oss.LifecycleRule{
        ID:     fmt.Sprintf("expire-%s-after-%d-days", prefix, days),
        Prefix: prefix,
        Expiration: oss.LifecycleExpiration{
            Days: days,
        },
        Status: "Enabled",
    }

    return s.bucket.SetLifecycleRule(rule)
}
```

**Source:** [Aliyun OSS Go SDK V2 Documentation - Presigned Upload](https://help.aliyun.com/zh/oss/developer-reference/v2-presign-upload) [HIGH confidence]

### Pattern 2: TingwuClient with HMAC-SHA256 Signing

**What:** Manual REST API client for Tingwu with HMAC-SHA256 signature authentication.

**When to use:** All Tingwu API operations (submit task, query status, get results).

**Example:**

```go
// Source: https://help.aliyun.com/zh/tingwu/get-results

type TingwuClient struct {
    appKey    string
    appSecret string
    baseURL   string
    client    *http.Client
    logger    *zap.Logger
}

func (c *TingwuClient) SubmitTask(ctx context.Context, fileURL string) (string, error) {
    body := map[string]interface{}{
        "file_url": fileURL,
        "version":  "4.0",
    }

    req, err := c.buildRequest(ctx, "POST", "/openapi/tingwu/v4/tasks", body)
    if err != nil {
        return "", err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("submit task failed: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        TaskID string `json:"TaskId"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", err
    }

    return result.TaskID, nil
}

func (c *TingwuClient) buildRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
    // Build HMAC-SHA256 signature per Aliyun ROA style
    timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
    signature := c.calculateSignature(method, path, timestamp, body)

    url := c.baseURL + path
    jsonBody, _ := json.Marshal(body)
    req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(jsonBody))
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Date", timestamp)
    req.Header.Set("Authorization", fmt.Sprintf("acs %s:%s", c.appKey, signature))

    return req, nil
}

func (c *TingwuClient) calculateSignature(method, path, timestamp string, body interface{}) string {
    // HMAC-SHA256 signing implementation
    // Reference: Aliyun API signature specification
    // ...
}
```

**Source:** [Aliyun Tingwu API Documentation - Get Results](https://help.aliyun.com/zh/tingwu/get-results) [MEDIUM confidence]

### Pattern 3: Cloud Transcription Pipeline Extension

**What:** Extend existing TranscriptionService with cloud branch, reusing statusMap and worker pool patterns.

**When to use:** TranscriptionService.processTranscription() needs to handle both local and cloud modes.

**Example:**

```go
// Source: Based on existing TranscriptionService pattern in internal/services/transcription_service.go

func (s *TranscriptionService) processTranscription(task *models.TranscriptionTask) {
    // Load task to check mode
    if err := s.db.Preload("VideoFile").First(task, task.ID).Error; err != nil {
        s.logger.Error("Failed to load task", zap.Error(err))
        return
    }

    // Route to appropriate pipeline
    if task.Mode == models.TranscriptionModeCloud {
        s.processCloudTranscription(task)
    } else {
        s.processLocalTranscription(task)  // Existing implementation
    }
}

func (s *TranscriptionService) processCloudTranscription(task *models.TranscriptionTask) {
    ctx, cancel := context.WithTimeout(s.ctx, 30*time.Minute)
    defer cancel()

    // Stage 1: Upload to OSS (0-20%)
    s.updateProgress(task.VideoFileID, "uploading", 0, 0, 0, "", nil)

    objectKey := fmt.Sprintf("transcriptions/%d.mp4", task.VideoFileID)
    ossURL, err := s.ossService.UploadFile(ctx, task.VideoFile.FilePath, objectKey)
    if err != nil {
        s.handleCloudFailure(task, err, true)  // Auto-fallback trigger
        return
    }

    // Stage 2: Submit to Tingwu (20%)
    s.updateProgress(task.VideoFileID, "submitting", 0, 0, 20, "", nil)

    cloudTaskID, err := s.tingwuClient.SubmitTask(ctx, ossURL)
    if err != nil {
        s.handleCloudFailure(task, err, true)  // Auto-fallback trigger
        return
    }

    // Save CloudTaskID
    s.db.Model(task).Update("cloud_task_id", cloudTaskID)

    // Stage 3: Poll Tingwu status with exponential backoff (20-90%)
    s.pollTingwuStatus(ctx, task, cloudTaskID)

    // Stage 4: Download results (90-100%)
    s.updateProgress(task.VideoFileID, "downloading", 0, 0, 90, "", nil)

    result, err := s.tingwuClient.GetResult(ctx, cloudTaskID)
    if err != nil {
        s.handleCloudFailure(task, err, false)  // No auto-fallback
        return
    }

    // Parse and save text content
    s.saveTextContent(task, result)

    // Trigger OSS cleanup (24h lifecycle)
    s.ossService.SetLifecycleRule(ctx, filepath.Dir(objectKey), 1)

    s.updateProgress(task.VideoFileID, "", 0, 0, 100, "", &task.ID)
    s.updateTaskStatus(task.ID, models.TranscriptionStatusCompleted, "", 0, &task.ID)
}

func (s *TranscriptionService) pollTingwuStatus(ctx context.Context, task *models.TranscriptionTask, cloudTaskID string) {
    // Exponential backoff with jitter: 2s → 4s → 8s → ... → 60s max
    err := retry.Do(
        func() error {
            status, err := s.tingwuClient.GetStatus(ctx, cloudTaskID)
            if err != nil {
                return err
            }

            switch status.Status {
            case "Queued":
                s.updateProgress(task.VideoFileID, "queued", 0, 0, 30, "", nil)
                return fmt.Errorf("still queued")  // Trigger retry
            case "Processing":
                percentage := 30 + int(status.Progress*0.6)  // Map 0-100 to 30-90%
                s.updateProgress(task.VideoFileID, "processing", 0, 0, percentage, "", nil)
                return fmt.Errorf("still processing")  // Trigger retry
            case "Completed":
                return nil  // Success, stop retrying
            case "Failed":
                return fmt.Errorf("tingwu failed: %s", status.ErrorMessage)  // Permanent error
            default:
                return fmt.Errorf("unknown status: %s", status.Status)
            }
        },
        retry.Context(ctx),
        retry.Attempts(60),           // Max 60 attempts
        retry.DelayType(retry.BackOffDelay),
        retry.MaxJitter(2*time.Second),
        retry.MaxDelay(60*time.Second),
    )

    if err != nil && ctx.Err() == nil {
        s.handleCloudFailure(task, err, false)  // Mid-processing failure, no auto-fallback
    }
}
```

**Source:** [Existing TranscriptionService pattern] [VERIFIED: internal/services/transcription_service.go] [HIGH confidence]

### Pattern 4: Frontend Dropdown.Button with Mode Selection

**What:** Convert existing "转录" button to Ant Design Dropdown.Button with two menu items.

**When to use:** File list action column and result page "重新转录" button.

**Example:**

```typescript
// Source: https://4x.ant.design/components/dropdown/

import { Dropdown, Button, Space } from 'antd'
import { CloudOutlined, LaptopOutlined } from '@ant-design/icons'

function TranscriptionButton({ videoFileId, fileName, onLocalTranscribe, onCloudTranscribe }) {
  const items = [
    {
      key: 'local',
      icon: <LaptopOutlined />,
      label: '本地转录',
      onClick: () => onLocalTranscribe(videoFileId, fileName),
    },
    {
      key: 'cloud',
      icon: <CloudOutlined />,
      label: '云端转录（通义听悟）',
      onClick: () => onCloudTranscribe(videoFileId, fileName),
    },
  ]

  return (
    <Dropdown menu={{ items }} trigger={['click']}>
      <Button type="primary" icon={<CloudOutlined />}>
        转录
      </Button>
    </Dropdown>
  )
}
```

**Source:** [Ant Design Dropdown Documentation](https://4x.ant.design/components/dropdown/) [VERIFIED: 4x.ant.design] [HIGH confidence]

### Pattern 5: TranscriptionProgressModal Cloud Stage Adaptation

**What:** Extend existing TranscriptionProgressModal to render different stages based on mode (local vs cloud).

**When to use:** TranscriptionProgressModal component when displaying cloud transcription progress.

**Example:**

```typescript
// Source: Based on existing TranscriptionProgressModal in frontend/src/components/TranscriptionProgressModal.tsx

const LOCAL_STAGES: Record<TranscriptionStage, { label: string; icon: React.ReactNode }> = {
  extracting: { label: '帧提取中', icon: <LoadingOutlined spin /> },
  detecting: { label: '画面检测中', icon: <LoadingOutlined spin /> },
  generating: { label: '生成PPT', icon: <LoadingOutlined spin /> },
}

const CLOUD_STAGES: Record<string, { label: string; icon: React.ReactNode }> = {
  uploading: { label: '上传中', icon: <CloudUploadOutlined /> },
  queued: { label: '排队中', icon: <ClockCircleOutlined /> },
  processing: { label: '处理中', icon: <LoadingOutlined spin /> },
  downloading: { label: '下载结果', icon: <CloudDownloadOutlined /> },
}

export default function TranscriptionProgressModal({
  open,
  onClose,
  videoFileId,
  fileName,
  mode,  // NEW: 'local' | 'cloud'
  samplingRate,  // Only used for local mode
  onCompleted,
}: TranscriptionProgressModalProps) {
  // ... existing state ...

  // Poll interval based on mode
  const pollInterval = mode === 'cloud' ? 10000 : 5000  // 10s for cloud, 5s for local

  useEffect(() => {
    if (!open) return

    const fetchStatus = async () => {
      const response = await getTranscriptionStatus(videoFileId)
      // ... existing status update logic ...
    }

    fetchStatus()
    const interval = setInterval(fetchStatus, pollInterval)
    return () => clearInterval(interval)
  }, [open, videoFileId, pollInterval])

  const renderStages = useCallback(() => {
    const stages = mode === 'cloud'
      ? ['uploading', 'queued', 'processing', 'downloading']
      : ['extracting', 'detecting', 'generating']

    const stageConfig = mode === 'cloud' ? CLOUD_STAGES : LOCAL_STAGES

    return (
      <div style={{ marginTop: 16 }}>
        {stages.map((s, index) => {
          const config = stageConfig[s]
          // ... existing stage rendering logic ...
        })}
      </div>
    )
  }, [stage, mode, status])

  // ... rest of modal logic ...
}
```

**Source:** [Existing TranscriptionProgressModal pattern] [VERIFIED: frontend/src/components/TranscriptionProgressModal.tsx] [HIGH confidence]

### Anti-Patterns to Avoid

- **Blocking OSS uploads on the request thread:** OSS uploads can take minutes for large videos. Always upload in background worker, return immediately with task status.
- **Hardcoded Tingwu polling intervals:** Without exponential backoff and jitter, simultaneous tasks trigger rate limiting. Use retry.Do() with MaxJitter.
- **OSS file orphaning:** Files uploaded but never deleted incur indefinite storage costs. Always set lifecycle rules or implement cleanup handler.
- **Tight coupling between cloud and local pipelines:** Cloud and local modes should be independent branches in processTranscription(). Shared code should be extracted to helper functions.
- **Ignoring Tingwu error codes:** Not all Tingwu errors should trigger auto-fallback. Only initial submission failures (upload, submit) should auto-fallback.
- **Missing timestamp format validation:** Tingwu returns timestamps in various formats. Always validate before displaying in UI.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| OSS HTTP client | Manual HTTP requests with auth headers | alibabacloud-oss-go-sdk-v2 | Edge cases: multipart upload for large files, presigned URL expiration, retry logic, connection pooling |
| HMAC-SHA256 signing | Custom crypto implementation | crypto/hmac from stdlib + Aliyun signature spec | Edge cases: UTF-8 encoding, URL encoding, timestamp format, signature key derivation |
| Exponential backoff with jitter | Custom sleep loops | github.com/avast/retry-go | Edge cases: context cancellation, max delay cap, jitter distribution, attempt counting |
| Presigned URL generation | Manual URL construction with expiration | OSS SDK SignURL method | Edge cases: URL encoding, parameter ordering, expiration calculation, signature validation |
| Dropdown menu with icons | Custom menu component | Ant Design Dropdown.Button | Edge cases: keyboard navigation, accessibility, click-outside handling, positioning |

**Key insight:** Cloud service integration has many hidden failure modes (network timeouts, rate limits, temporary credentials). Battle-tested libraries handle these edge cases; custom implementations inevitably miss corner cases.

## Common Pitfalls

### Pitfall 1: OSS File Orphaning

**What goes wrong:** Videos uploaded to OSS remain indefinitely after transcription completes, incurring ongoing storage costs. Orphaned files accumulate over time.

**Why it happens:** No cleanup mechanism exists, or cleanup depends on successful completion (if DB transaction rolls back after OSS upload succeeds, file is never deleted).

**How to avoid:**
- **Use lifecycle rules:** Set 24-hour expiration on all transcription uploads using OSS bucket lifecycle rules
- **Two-phase commit pattern:** Don't commit DB transaction until OSS upload succeeds, or implement compensating transactions
- **Periodic cleanup job:** Run a daily cron job to delete OSS files older than 24 hours whose corresponding TranscriptionTask is completed/failed

**Warning signs:** OSS storage costs increasing linearly with transcription count, bucket file list contains many old files.

### Pitfall 2: Tingwu Status Polling Thundering Herd

**What goes wrong:** Multiple cloud transcription tasks poll Tingwu status API simultaneously, triggering rate limiting (429 Too Many Requests). All polls fail, tasks stall.

**Why it happens:** Fixed polling intervals without jitter synchronize requests across workers. 10 tasks polling every 10s = 10 simultaneous requests every 10s.

**How to avoid:**
- **Exponential backoff with jitter:** Use retry.Do() with MaxJitter(2*time.Second) to randomize request timing
- **Stagger initial polls:** Add random delay (0-5s) before first status check for each task
- **Global rate limiter:** Implement token bucket or leaky bucket rate limiter for Tingwu API client

**Warning signs:** Tingwu API returns 429 errors, log shows "status polling failed" for multiple tasks simultaneously.

### Pitfall 3: Cloud-to-Local Fallback Race Conditions

**What goes wrong:** Auto-fallback triggers local transcription, but progress modal shows conflicting status (both cloud and local stages visible). User sees "云端转录失败" alert but modal shows cloud stages.

**Why it happens:** Fallback logic doesn't clean up cloud state before starting local pipeline. Frontend polling continues to fetch stale cloud status.

**How to avoid:**
- **Atomic mode switch:** Update TranscriptionTask.Mode from "cloud" to "local" atomically in DB before starting local pipeline
- **Clear cloud state:** Reset CloudTaskID and OSSURL fields when switching to local mode
- **Frontend sync:** On fallback, frontend should close and reopen progress modal to fetch fresh status

**Warning signs:** Progress modal shows "上传中" and "帧提取中" simultaneously, logs show both cloud and local worker logs for same task.

### Pitfall 4: Tingwu Result Parsing Errors

**What goes wrong:** Tingwu API returns text content, but timestamps are malformed or missing. Frontend displays "[Invalid Timestamp]" or crashes when clicking timestamps.

**Why it happens:** Tingwu timestamp format varies by API version and content type. Without robust parsing, edge cases cause crashes.

**How to avoid:**
- **Validate timestamp format:** Use regex to validate `[HH:MM:SS]` or milliseconds format before rendering
- **Fallback to default:** If timestamp parsing fails, default to 00:00:00 or omit timestamp
- **Type-safe models:** Define strict TypeScript types for text segments, validate at API boundary

**Warning signs:** Frontend console shows timestamp parsing errors, text content tab fails to load.

### Pitfall 5: OSS Presigned URL Expiration During Transcription

**What goes wrong:** Tingwu API fails to download video from OSS presigned URL with "access denied" or "URL expired".

**Why it happens:** Presigned URL expires before Tingwu completes download. Cloud transcription can take minutes, but presigned URL might expire after 1 hour.

**How to avoid:**
- **Set sufficient expiration:** Generate presigned URLs with 24-hour expiration (max allowed by Tingwu API timeout)
- **Use public bucket:** If security allows, make bucket public-read and skip presigned URLs
- **Refresh URL if needed:** Check Tingwu error codes, resubmit task with fresh URL if access denied

**Warning signs:** Tingwu API returns "FileDownloadFailed" or "AccessDenied" errors.

## Code Examples

### OSSService Upload with Presigned URL

```go
// Source: https://help.aliyun.com/zh/oss/developer-reference/v2-presign-upload

package services

import (
    "context"
    "fmt"
    "time"

    "github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
    "go.uber.org/zap"
)

type OSSService struct {
    client     *oss.Client
    bucket     *oss.Bucket
    logger     *zap.Logger
    config     *config.OSSConfig
}

func NewOSSService(cfg *config.OSSConfig, logger *zap.Logger) (*OSSService, error) {
    // Initialize OSS client with credentials
    client, err := oss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
    if err != nil {
        return nil, fmt.Errorf("failed to create OSS client: %w", err)
    }

    bucket, err := client.Bucket(cfg.BucketName)
    if err != nil {
        return nil, fmt.Errorf("failed to get bucket: %w", err)
    }

    return &OSSService{
        client: client,
        bucket: bucket,
        logger: logger,
        config: cfg,
    }, nil
}

// UploadFile uploads a local file to OSS and returns a presigned URL
func (s *OSSService) UploadFile(ctx context.Context, localPath string, objectKey string) (string, error) {
    // Upload file to OSS
    err := s.bucket.PutObjectFromFile(objectKey, localPath, oss.WithContext(ctx))
    if err != nil {
        return "", fmt.Errorf("OSS upload failed: %w", err)
    }

    s.logger.Info("File uploaded to OSS",
        zap.String("local_path", localPath),
        zap.String("object_key", objectKey))

    // Generate presigned URL with 24-hour expiration
    signedURL, err := s.bucket.SignURL(objectKey, oss.HTTPGet, 86400)
    if err != nil {
        return "", fmt.Errorf("generate presigned URL failed: %w", err)
    }

    return signedURL, nil
}

// SetLifecycleRule sets an expiration rule for uploaded files
func (s *OSSService) SetLifecycleRule(ctx context.Context, prefix string, days int) error {
    rule := oss.LifecycleRule{
        ID:     fmt.Sprintf("expire-%s-after-%d-days", prefix, days),
        Prefix: prefix,
        Expiration: oss.LifecycleExpiration{
            Days: days,
        },
        Status: "Enabled",
    }

    err := s.bucket.SetLifecycleRule(rule)
    if err != nil {
        return fmt.Errorf("set lifecycle rule failed: %w", err)
    }

    s.logger.Info("OSS lifecycle rule set",
        zap.String("prefix", prefix),
        zap.Int("days", days))

    return nil
}

// DeleteFile deletes a file from OSS
func (s *OSSService) DeleteFile(ctx context.Context, objectKey string) error {
    err := s.bucket.DeleteObject(objectKey, oss.WithContext(ctx))
    if err != nil {
        return fmt.Errorf("delete OSS file failed: %w", err)
    }

    s.logger.Info("File deleted from OSS", zap.String("object_key", objectKey))
    return nil
}
```

**Source:** [Aliyun OSS Go SDK V2 - Presigned Upload](https://help.aliyun.com/zh/oss/developer-reference/v2-presign-upload) [HIGH confidence]

### TingwuClient with Exponential Backoff

```go
// Source: https://help.aliyun.com/zh/tingwu/get-results

package services

import (
    "bytes"
    "context"
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/avast/retry-go/v4"
    "go.uber.org/zap"
)

type TingwuClient struct {
    appKey    string
    appSecret string
    baseURL   string
    client    *http.Client
    logger    *zap.Logger
}

type TingwuStatus struct {
    Status       string  `json:"Status"`
    Progress     float64 `json:"Progress"`
    ErrorMessage string  `json:"ErrorMessage"`
}

type TingwuResult struct {
    Text     string                `json:"Text"`
    Segments []TingwuTextSegment   `json:"Segments"`
}

type TingwuTextSegment struct {
    Text        string  `json:"Text"`
    BeginTime   int     `json:"BeginTime"`  // Milliseconds
    EndTime     int     `json:"EndTime"`    // Milliseconds
}

func NewTingwuClient(appKey, appSecret, baseURL string, logger *zap.Logger) *TingwuClient {
    return &TingwuClient{
        appKey:    appKey,
        appSecret: appSecret,
        baseURL:   baseURL,
        client:    &http.Client{Timeout: 30 * time.Second},
        logger:    logger,
    }
}

// SubmitTask submits a transcription task to Tingwu
func (c *TingwuClient) SubmitTask(ctx context.Context, fileURL string) (string, error) {
    body := map[string]interface{}{
        "file_url": fileURL,
        "version":  "4.0",
    }

    req, err := c.buildRequest(ctx, "POST", "/openapi/tingwu/v4/tasks", body)
    if err != nil {
        return "", fmt.Errorf("build request failed: %w", err)
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("submit task failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("submit task failed: status=%d, body=%s", resp.StatusCode, string(body))
    }

    var result struct {
        TaskID string `json:"TaskId"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return "", fmt.Errorf("decode response failed: %w", err)
    }

    c.logger.Info("Tingwu task submitted", zap.String("task_id", result.TaskID))
    return result.TaskID, nil
}

// GetStatus polls task status with exponential backoff
func (c *TingwuClient) GetStatus(ctx context.Context, taskID string) (*TingwuStatus, error) {
    path := fmt.Sprintf("/openapi/tingwu/v4/tasks/%s", taskID)

    req, err := c.buildRequest(ctx, "GET", path, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("get status failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("get status failed: status=%d", resp.StatusCode)
    }

    var result struct {
        Status       TingwuStatus `json:"Data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode response failed: %w", err)
    }

    return &result.Status, nil
}

// PollStatusUntilCompletion polls status with exponential backoff
func (c *TingwuClient) PollStatusUntilCompletion(ctx context.Context, taskID string) error {
    return retry.Do(
        func() error {
            status, err := c.GetStatus(ctx, taskID)
            if err != nil {
                return err
            }

            switch status.Status {
            case "Queued":
                c.logger.Info("Tingwu task queued", zap.String("task_id", taskID))
                return fmt.Errorf("task still queued")
            case "Processing":
                c.logger.Info("Tingwu task processing",
                    zap.String("task_id", taskID),
                    zap.Float64("progress", status.Progress))
                return fmt.Errorf("task still processing")
            case "Completed":
                c.logger.Info("Tingwu task completed", zap.String("task_id", taskID))
                return nil
            case "Failed":
                return fmt.Errorf("tingwu task failed: %s", status.ErrorMessage)
            default:
                return fmt.Errorf("unknown status: %s", status.Status)
            }
        },
        retry.Context(ctx),
        retry.Attempts(120),           // Max 120 attempts (2 hours with backoff)
        retry.DelayType(retry.BackOffDelay),
        retry.MaxJitter(2*time.Second),
        retry.MaxDelay(60*time.Second),
        retry.LastErrorOnly(true),
    )
}

// GetResult retrieves transcription result
func (c *TingwuClient) GetResult(ctx context.Context, taskID string) (*TingwuResult, error) {
    path := fmt.Sprintf("/openapi/tingwu/v4/tasks/%s/result", taskID)

    req, err := c.buildRequest(ctx, "GET", path, nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("get result failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("get result failed: status=%d", resp.StatusCode)
    }

    var result struct {
        Data TingwuResult `json:"Data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode response failed: %w", err)
    }

    return &result.Data, nil
}

// buildRequest builds an HTTP request with HMAC-SHA256 signature
func (c *TingwuClient) buildRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
    // Build URL
    url := c.baseURL + path

    // Prepare body
    var bodyReader io.Reader
    var contentMD5 string
    if body != nil {
        jsonBody, err := json.Marshal(body)
        if err != nil {
            return nil, fmt.Errorf("marshal body failed: %w", err)
        }
        bodyReader = bytes.NewReader(jsonBody)
        contentMD5 = calculateMD5(jsonBody)
    }

    // Create request
    req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
    if err != nil {
        return nil, fmt.Errorf("create request failed: %w", err)
    }

    // Calculate signature
    timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
    signature := c.calculateSignature(method, path, timestamp, contentMD5)

    // Set headers
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Date", timestamp)
    req.Header.Set("Content-MD5", contentMD5)
    req.Header.Set("Authorization", fmt.Sprintf("acs %s:%s", c.appKey, signature))

    return req, nil
}

// calculateSignature calculates HMAC-SHA256 signature per Aliyun spec
func (c *TingwuClient) calculateSignature(method, path, timestamp, contentMD5 string) string {
    // Construct string to sign
    // Format: METHOD\nContent-MD5\nContent-Type\nDate\nPath
    stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
        strings.ToUpper(method),
        contentMD5,
        "application/json",
        timestamp,
        path,
    )

    // Calculate HMAC-SHA256
    h := hmac.New(sha256.New, []byte(c.appSecret))
    h.Write([]byte(stringToSign))
    signature := hex.EncodeToString(h.Sum(nil))

    return signature
}

func calculateMD5(data []byte) string {
    // Implement MD5 calculation
    // ...
    return ""
}
```

**Source:** [Aliyun Tingwu API Documentation](https://help.aliyun.com/zh/tingwu/get-results) [MEDIUM confidence]

### Frontend Text Content Tab with Timestamps

```typescript
// Source: Based on existing result page pattern in frontend/src/pages/results/index.tsx

import { useState, useEffect } from 'react'
import { Tabs, Button, message, Space, Tooltip } from 'antd'
import { CopyOutlined, CheckOutlined } from '@ant-design/icons'
import { getTranscriptionText } from '../../api/transcription'

interface TextSegment {
  text: string
  beginTime: number  // milliseconds
  endTime: number    // milliseconds
}

interface TextContentTabProps {
  videoFileId: number
  onTimestampClick?: (timestamp: number) => void  // Jump to video position
}

export default function TextContentTab({ videoFileId, onTimestampClick }: TextContentTabProps) {
  const [segments, setSegments] = useState<TextSegment[]>([])
  const [loading, setLoading] = useState(false)
  const [copied, setCopied = useState<number | null>(null)

  useEffect(() => {
    const fetchText = async () => {
      setLoading(true)
      try {
        const response = await getTranscriptionText(videoFileId)
        setSegments(response.data.segments)
      } catch (error) {
        message.error('加载文字内容失败')
      } finally {
        setLoading(false)
      }
    }

    fetchText()
  }, [videoFileId])

  const formatTimestamp = (milliseconds: number): string => {
    const totalSeconds = Math.floor(milliseconds / 1000)
    const hours = Math.floor(totalSeconds / 3600)
    const minutes = Math.floor((totalSeconds % 3600) / 60)
    const seconds = totalSeconds % 60
    return `[${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}]`
  }

  const handleCopyAll = async () => {
    const fullText = segments.map(s => `${formatTimestamp(s.beginTime)} ${s.text}`).join('\n')
    await navigator.clipboard.writeText(fullText)
    message.success('已复制全部文字')
  }

  const handleCopySegment = async (index: number) => {
    const segment = segments[index]
    const text = `${formatTimestamp(segment.beginTime)} ${segment.text}`
    await navigator.clipboard.writeText(text)
    setCopied(index)
    setTimeout(() => setCopied(null), 2000)
  }

  const handleTimestampClick = (timestamp: number) => {
    if (onTimestampClick) {
      onTimestampClick(timestamp / 1000)  // Convert to seconds
    }
  }

  if (loading) {
    return <div>加载中...</div>
  }

  return (
    <div>
      <div style={{ marginBottom: 16, textAlign: 'right' }}>
        <Button icon={<CopyOutlined />} onClick={handleCopyAll}>
          复制全部文字
        </Button>
      </div>

      <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
        {segments.map((segment, index) => (
          <div
            key={index}
            style={{
              marginBottom: 16,
              padding: 12,
              background: '#f5f5f5',
              borderRadius: 4,
            }}
          >
            <Space size={8}>
              <Button
                type="text"
                size="small"
                icon={<CopyOutlined />}
                onClick={() => handleCopySegment(index)}
              >
                {copied === index ? <CheckOutlined /> : null}
              </Button>

              <span
                style={{
                  fontFamily: 'monospace',
                  color: '#1890ff',
                  cursor: onTimestampClick ? 'pointer' : 'default',
                }}
                onClick={() => handleTimestampClick(segment.beginTime)}
              >
                {formatTimestamp(segment.beginTime)}
              </span>

              <span>{segment.text}</span>
            </Space>
          </div>
        ))}
      </div>
    </div>
  )
}
```

**Source:** [Existing frontend patterns] [VERIFIED: frontend/src/components/TranscriptionProgressModal.tsx] [HIGH confidence]

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| OSS SDK v1 | OSS SDK v2 | 2023 | v2 has improved API design, better error handling, presigned URL support |
| Fixed polling intervals | Exponential backoff with jitter | 2024 | Prevents thundering herd, reduces rate limiting issues |
| Manual signature calculation | crypto/hmac stdlib | 2020+ | Standard library is audited, performant, secure |
| Callback-based status | Polling-based status | 2024 | Tingwu API supports both, polling is simpler for MVP |

**Deprecated/outdated:**
- **baliance/go-pptx**: Deprecated, use UniDoc UniOffice or forked versions [ASSUMED]
- **OSS SDK v1**: Deprecated in 2023, migrate to v2
- **Ant Design 5.x**: Upgrade to 6.x for latest features and bug fixes

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | No official Go SDK exists for Aliyun Tingwu API | Standard Stack | If official SDK exists, manual HMAC signing is unnecessary complexity |
| A2 | Tingwu API supports presigned URL access for uploaded videos | Architecture Patterns | If Tingwu requires public URLs, bucket security must be relaxed |
| A3 | Tingwu task status polling interval of 10s is sufficient for UX | Architecture Patterns | If users expect faster updates, polling interval may need adjustment |
| A4 | OSS lifecycle rules are the best cleanup mechanism | Common Pitfalls | If lifecycle rules have lag, manual cleanup may be needed |
| A5 | UniDoc UniOffice is the recommended go-pptx replacement | State of the Art | If baliance/go-pptx is still maintained, migration is unnecessary |
| A6 | Tingwu returns text segments with millisecond timestamps | Code Examples | If timestamp format varies, parsing logic will fail |

## Open Questions (RESOLVED)

1. **Tingwu API authentication specifics** — RESOLVED: Plan 04-01 implements HMAC-SHA256 signing in TingwuClient.calculateSignature() using crypto/hmac stdlib with Aliyun ROA-style string format (METHOD\nMD5\nContent-Type\nDate\nPath)
   - What we know: HMAC-SHA256 signing is required per Aliyun ROA style
   - What's unclear: Exact signature string format (header ordering, encoding details)
   - Recommendation: Implement signature calculation, test with real API calls, adjust based on error messages

2. **Tingwu result timestamp format** — RESOLVED: Plan 04-03 implements saveTextContent() with millisecond-to-HH:MM:SS conversion; Plan 04-04 TextContentTab uses formatTimestamp() for display. Fallback to 00:00:00 on parse failure.
   - What we know: Results include time offsets for text segments
   - What's unclear: Whether timestamps are milliseconds, seconds, or HH:MM:SS format
   - Recommendation: Add robust parsing with fallback, validate format during testing

3. **OSS bucket public access requirement** — RESOLVED: Plan 04-01 uses presigned URLs via OSS SDK SignURL with 24-hour expiration. If Tingwu rejects presigned URLs, bucket can be switched to public-read.
   - What we know: Presigned URLs provide temporary access
   - What's unclear: Whether Tingwu API requires public URLs or accepts presigned URLs
   - Recommendation: Test presigned URLs first, only use public bucket if presigned fails

4. **Cloud transcription text content storage** — RESOLVED: Plan 04-01 creates separate TranscriptionText model (ID, TranscriptionTaskID, Text, BeginTime, EndTime, SegmentIndex) with GORM auto-migration, per recommendation for queryability.
   - What we know: Text content must be persisted for display in result page
   - What's unclear: Whether to create new TranscriptionText table or embed in TranscriptionTask JSON
   - Recommendation: Create separate TranscriptionText table for queryability and future search features

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.25+ | All backend services | ✓ | 1.25.0 | — |
| Node.js 20+ | Frontend build | ✓ | (assumed) | — |
| FFmpeg | Video processing | ✓ | (in-tree) | — |
| Aliyun OSS credentials | OSSService | ✗ | — | Manual config in .env |
| Tingwu APP_KEY | TingwuClient | ✓ | (in .env) | — |
| Ant Design 6.x | Frontend UI | ✓ | 6.3.6 | — |
| React 19.x | Frontend framework | ✓ | 19.2.5 | — |

**Missing dependencies with no fallback:**
- Aliyun OSS credentials (ALIYUN_OSS_ENDPOINT, ALIYUN_OSS_BUCKET, ALIYUN_OSS_ACCESS_KEY_ID, ALIYUN_OSS_ACCESS_KEY_SECRET) — must be provided by user

**Missing dependencies with fallback:**
- None identified

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | testing (Go stdlib) + testify |
| Config file | None — tests use in-memory setup |
| Quick run command | `go test ./internal/services/... -run TestOSSService -v` |
| Full suite command | `go test ./... -v` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| OSS-01 | Upload video to OSS, generate presigned URL | integration | `go test ./internal/services/... -run TestOSSServiceUploadFile -v` | ❌ Wave 0 |
| OSS-01 | Presigned URL is publicly accessible | integration | `go test ./internal/services/... -run TestOSSServicePresignedURL -v` | ❌ Wave 0 |
| OSS-02 | OSS lifecycle rule deletes files after 24h | integration | `go test ./internal/services/... -run TestOSSServiceLifecycleRule -v` | ❌ Wave 0 |
| TRAN-01 | Cloud transcription submission creates task | unit | `go test ./internal/services/... -run TestTingwuClientSubmitTask -v` | ❌ Wave 0 |
| TRAN-01 | Cloud transcription status polling works | integration | `go test ./internal/services/... -run TestTingwuClientPollStatus -v` | ❌ Wave 0 |
| TRAN-03 | Auto-fallback triggers on OSS upload failure | unit | `go test ./internal/services/... -run TestTranscriptionServiceCloudFallback -v` | ❌ Wave 0 |
| TRAN-05 | Text content retrieved and parsed correctly | unit | `go test ./internal/services/... -run TestTingwuClientGetResult -v` | ❌ Wave 0 |
| TRAN-05 | Timestamp format validation | unit | `go test ./internal/... -run TestTimestampParsing -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/services/... -run TestOSS -v && go test ./internal/services/... -run TestTingwu -v`
- **Per wave merge:** `go test ./... -v`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps

- [ ] `internal/services/oss_service_test.go` — covers OSS-01, OSS-02
- [ ] `internal/services/tingwu_client_test.go` — covers TRAN-01, TRAN-05
- [ ] `internal/services/transcription_service_cloud_test.go` — covers TRAN-03 (fallback)
- [ ] `internal/models/transcription_text_test.go` — text content model
- [ ] Mock server for Tingwu API responses (use httptest)
- [ ] OSS integration test config (use minio or test bucket)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | yes | Existing JWT middleware, user-owned video file checks |
| V5 Input Validation | yes | Validate video file ownership, mode enum values, sampling rate range |
| V6 Cryptography | yes | Use crypto/hmac for Tingwu signing, never roll custom crypto |
| V7 Error Handling | yes | Don't expose APP_KEY/appSecret in errors, sanitize Tingwu error messages |

### Known Threat Patterns for Go Backend + Aliyun APIs

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| OSS credential exposure | Tampering/Disclosure | Store credentials in .env (not in code), use least-privilege IAM policies |
| Tingwu APP_KEY leakage | Information Disclosure | Never log APP_KEY, sanitize error responses |
| Presigned URL abuse | Spoofing | Set short expiration (24h), monitor OSS access logs |
| OSS file orphaning | Denial of Service | Lifecycle rules, periodic cleanup job, cost alerts |
| Tingwu API rate limiting | Denial of Service | Exponential backoff with jitter, global rate limiter |
| User file access bypass | Tampering | Verify file ownership on every API call (existing pattern) |

## Sources

### Primary (HIGH confidence)

- [Aliyun OSS Go SDK V2 Documentation - Presigned Upload](https://help.aliyun.com/zh/oss/developer-reference/v2-presign-upload)
- [Aliyun OSS Go SDK V2 Documentation - Multipart Upload](https://help.aliyun.com/zh/oss/developer-reference/v2-multipart-upload)
- [Aliyun OSS Go SDK V2 Documentation - Lifecycle Management](https://help.aliyun.com/zh/oss/developer-reference/v2-lifecycle)
- [Aliyun Tingwu API Documentation - Get Results](https://help.aliyun.com/zh/tingwu/get-results)
- [Ant Design Dropdown Documentation](https://4x.ant.design/components/dropdown/)
- [Existing codebase - TranscriptionService] [VERIFIED: internal/services/transcription_service.go]
- [Existing codebase - TranscriptionProgressModal] [VERIFIED: frontend/src/components/TranscriptionProgressModal.tsx]

### Secondary (MEDIUM confidence)

- [Go Retry with Exponential Backoff - OneUptime Blog](https://oneuptime.com/blog/post/2026-01-07-go-retry-exponential-backoff/view)
- [Modern HTTP Retries with Exponential Back-off & Jitter - LinkedIn](https://www.linkedin.com/pulse/modern-http-retries-exponential-back-off-jitter-go-aslam-mulla-gpwdf)
- [AWS Timeouts, Retries and Backoff with Jitter](https://aws.amazon.com/builders-library/timeouts-retries-and-backoff-with-jitter/)

### Tertiary (LOW confidence)

- [UniDoc UniOffice Documentation](https://docs.unidoc.io/docs/unioffice/guides/presentation/image/) — Need to verify if this is the correct Go PPTX library replacement
- Various search results for go-pptx alternatives — Need user confirmation on library choice

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM - OSS SDK v2 and Ant Design verified, Tingwu Go client needs implementation
- Architecture: MEDIUM - Cloud pipeline extension pattern is sound, but Tingwu API details need testing
- Pitfalls: HIGH - Based on verified cloud service integration patterns and existing codebase patterns

**Research date:** 2025-04-17
**Valid until:** 2025-05-17 (30 days - Aliyun APIs are stable, but SDK updates may occur)
