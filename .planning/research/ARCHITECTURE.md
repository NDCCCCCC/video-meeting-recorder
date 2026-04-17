# Architecture Research

**Domain:** Video Recording Management with AI Transcription
**Researched:** 2025-04-17
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend Layer                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │ Video Player │  │ Task Manager │  │ File Browser │         │
│  │   Controls   │  │   Interface  │  │   Interface  │         │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘         │
│         │                 │                 │                  │
├─────────┼─────────────────┼─────────────────┼──────────────────┤
│         │         API Layer (Gin Router)     │                  │
│         │    ┌──────────────────────────┐    │                  │
│         │    │  Multi-Auth Middleware   │    │                  │
│         │    │  (SM4 Token + API Key)   │    │                  │
│         │    └───────────┬──────────────┘    │                  │
│         │                │                    │                  │
│         │    ┌───────────┴──────────────┐    │                  │
│         │    │    Handler Layer         │    │                  │
│         │    │  - TaskHandler           │    │                  │
│         │    │  - TranscriptionHandler  │    │                  │
│         │    │  - SplitHandler          │    │                  │
│         │    │  - FileHandler           │    │                  │
│         │    └───────────┬──────────────┘    │                  │
│         │                │                    │                  │
├─────────┼────────────────┼────────────────────┼──────────────────┤
│         │         Service Layer               │                  │
│         │    ┌───────────┴──────────────┐    │                  │
│         │    │  - TaskService           │    │                  │
│         │    │  - TranscriptionService  │    │                  │
│         │    │  - SplitService          │    │                  │
│         │    │  - OSSService            │    │                  │
│         │    │  - ConversionService     │    │                  │
│         │    └───────────┬──────────────┘    │                  │
│         │                │                    │                  │
├─────────┼────────────────┼────────────────────┼──────────────────┤
│         │    Data Layer   │    External APIs   │                  │
│         │  ┌──────────────┴──────────────┐    │                  │
│         │  │      GORM (SQLite)          │    │                  │
│         │  │  - Tasks                    │    │                  │
│         │  │  - Files                    │    │                  │
│         │  │  - Transcriptions           │    │                  │
│         │  │  - Splits                   │    │                  │
│         │  └──────────────┬──────────────┘    │                  │
│         │                 │                   │                  │
│         │    ┌────────────┴────────────┐      │                  │
│         │    │  - FFmpeg (video)       │      │                  │
│         │    │  - Aliyun OSS           │      │                  │
│         │    │  - Aliyun Tingwu        │      │                  │
│         │    └─────────────────────────┘      │                  │
└─────────┴────────────────┴────────────────────┴──────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **Frontend** | User interface, state management, API communication | React 19 + Ant Design 6 + Zustand + TanStack Query |
| **API Layer** | HTTP routing, authentication, request validation | Gin router with middleware chain |
| **Handler Layer** | Request/response handling, parameter validation, business orchestration | Struct handlers with service dependencies |
| **Service Layer** | Business logic, external API integration, data transformation | Interface-based services with error handling |
| **Data Layer** | Persistent storage, relationships, constraints | GORM ORM with SQLite database |
| **External Services** | Video processing, cloud storage, AI transcription | FFmpeg, Aliyun OSS SDK, Aliyun Tingwu API |

## Recommended Project Structure

```
internal/
├── handlers/                   # HTTP request handlers
│   ├── video_recording_task_handler.go  # Existing: task CRUD
│   ├── transcription_handler.go         # NEW: transcription tasks
│   ├── video_split_handler.go           # NEW: split operations
│   └── file_handler.go                  # Existing: file management
├── services/                   # Business logic layer
│   ├── video_recording_task_service.go  # Existing: task service
│   ├── transcription/                    # NEW: transcription service
│   │   ├── service.go                   # Main transcription logic
│   │   ├── tingwu_client.go             # Aliyun Tingwu API client
│   │   └── result_processor.go          # Process transcription results
│   ├── oss/                              # NEW: OSS integration
│   │   ├── service.go                   # OSS upload/download service
│   │   └── client.go                    # Aliyun OSS client wrapper
│   ├── video_split/                      # NEW: video splitting
│   │   ├── service.go                   # Split business logic
│   │   └── ffmpeg_split.go              # FFmpeg split operations
│   └── conversion_service.go             # Existing: MKV→MP4 conversion
├── models/                     # GORM data models
│   ├── video_recording_task.go          # Existing: tasks
│   ├── video_file.go                    # Existing: video files
│   ├── ppt_file.go                      # Existing: PPT files
│   ├── transcription.go                 # NEW: transcription records
│   └── video_split.go                   # NEW: split segments
├── middleware/                 # HTTP middleware
│   ├── auth.go                          # Existing: authentication
│   └── permission.go                    # Existing: authorization
├── config/                     # Configuration
│   └── config.go                        # Extended with OSS/Tingwu configs
└── utils/                      # Utility functions
    ├── ffmpeg.go                        # Existing: FFmpeg helpers
    └── tingwu_signature.go              # NEW: API signature generation
```

### Structure Rationale

- **handlers/**: Separate handlers by feature domain (transcription, split) following existing pattern
- **services/**: Group related services in subdirectories to manage complexity
  - **transcription/**: Isolates Aliyun API interaction and result processing
  - **oss/**: Encapsulates cloud storage operations, easy to swap providers
  - **video_split/**: Contains split-specific logic separate from general conversion
- **models/**: Add new models for transcription and splits, following existing GORM patterns
- **middleware/**: Reuse existing auth/permission middleware for new endpoints
- **config/**: Extend existing config structure with new sections for external services

## Architectural Patterns

### Pattern 1: Service Interface Pattern

**What:** Define interfaces for services to enable testing and flexibility

**When to use:** 
- Services that may have multiple implementations
- Complex business logic that needs unit testing
- External API integrations

**Trade-offs:** 
- Pros: Testable, flexible, clear contracts
- Cons: More boilerplate, initial complexity

**Example:**
```go
// Service interface definition
type TranscriptionService interface {
    SubmitTask(taskID uint, videoURL string) (string, error)
    GetStatus(transcriptionID string) (TranscriptionStatus, error)
    GetResult(transcriptionID string) (*TranscriptionResult, error)
    CancelTask(transcriptionID string) error
}

// Concrete implementation
type AliyunTingwuService struct {
    client *TingwuClient
    db     *gorm.DB
    logger *zap.Logger
    config *config.TingwuConfig
}

func (s *AliyunTingwuService) SubmitTask(taskID uint, videoURL string) (string, error) {
    // Implementation
}
```

### Pattern 2: Background Worker Pattern

**What:** Use worker pools for long-running async operations

**When to use:**
- Video processing (splitting, conversion)
- External API polling (transcription status)
- File uploads to cloud storage

**Trade-offs:**
- Pros: Controlled concurrency, error isolation, scalable
- Cons: Complexity in state management, requires persistence

**Example:**
```go
type TranscriptionWorker struct {
    taskQueue  chan TranscriptionTask
    workers    int
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
    service    TranscriptionService
}

func (w *TranscriptionWorker) Start() {
    for i := 0; i < w.workers; i++ {
        w.wg.Add(1)
        go w.worker(i)
    }
}

func (w *TranscriptionWorker) worker(id int) {
    defer w.wg.Done()
    for {
        select {
        case task := <-w.taskQueue:
            w.processTask(task)
        case <-w.ctx.Done():
            return
        }
    }
}
```

### Pattern 3: Repository Pattern for External APIs

**What:** Wrap external API calls in repository-like interfaces

**When to use:**
- External service integration (Aliyun APIs)
- Need to mock external dependencies in tests
- Want to switch providers without changing business logic

**Trade-offs:**
- Pros: Isolated external dependencies, easier testing
- Cons: Additional abstraction layer

**Example:**
```go
type TingwuClient interface {
    SubmitTask(ctx context.Context, req *SubmitTaskRequest) (*SubmitTaskResponse, error)
    GetTaskInfo(ctx context.Context, taskID string) (*TaskInfo, error)
    GetResult(ctx context.Context, taskID string) (*TranscriptionResult, error)
}

type AliyunTingwuClient struct {
    client     *http.Client
    credential *credentials.Credential
    endpoint   string
}

func (c *AliyunTingwuClient) SubmitTask(ctx context.Context, req *SubmitTaskRequest) (*SubmitTaskResponse, error) {
    // API call implementation with signature
}
```

## Data Flow

### Request Flow

```
User Action (React Frontend)
    ↓ (HTTP request with SM4 Token/API Key)
Gin Router → MultiAuth Middleware → Handler
    ↓              ↓                          ↓
Permission Check → User Context Set → Request Validation
    ↓                                          ↓
Service Layer → Business Logic Execution
    ↓              ↓
External APIs → Database Operations
    ↓              ↓
Response ← Data Transformation ← Service Result
    ↓
JSON Response to Frontend
```

### Transcription Flow

```
User: Start Transcription
    ↓
POST /api/v1/recordings/{id}/transcribe
    ↓
TranscriptionHandler.SubmitTranscription()
    ↓
1. Check task status (must be completed)
2. Check existing transcription (no duplicates)
3. Get video file path
    ↓
OSSService.UploadFile() ← Upload video to Aliyun OSS
    ↓
TingwuService.SubmitTask(videoURL)
    ↓
1. Generate API signature
2. Call Aliyun Tingwu API
3. Get transcription ID
4. Save Transcription record (status: pending)
    ↓
Background Worker: Poll for status
    ↓
While status != completed:
    TingwuService.GetStatus(transcriptionID)
    If failed: Update status, notify user
    If completed: Download results
    ↓
ResultProcessor.Process()
    ↓
1. Download text transcript
2. Download PPT slides (images)
3. Generate PPT file
4. Save PPT record
5. Update Transcription record (status: completed)
    ↓
Notify user (NotificationService)
```

### Video Split Flow

```
User: Mark split points on video timeline
    ↓
POST /api/v1/recordings/{id}/splits
    ↓
SplitHandler.CreateSplits()
    ↓
1. Validate split points (ascending order, within video duration)
2. Check video file exists
3. Create SplitSegment records (status: pending)
    ↓
Background Worker: Process splits
    ↓
For each split segment:
    SplitService.ProcessSplit(segment)
    ↓
    FFmpegSplitService.Split(inputPath, segment)
    ↓
    1. Calculate start/end timestamps
    2. Build FFmpeg command (seek + copy)
    3. Execute split
    4. Validate output file
    5. Create VideoFile record
    6. Update SplitSegment (status: completed)
    ↓
Notify user (NotificationService)
```

### Key Data Flows

1. **Video → OSS → Tingwu:** Local file uploaded to OSS gets public URL for Tingwu access
2. **Tingwu → Local Storage:** Transcription results downloaded and stored locally
3. **FFmpeg Operations:** All video processing (conversion, split) uses existing FFmpeg integration
4. **State Synchronization:** Database status updates trigger frontend polling reactions

## Integration Points

### External Services

| Service | Integration Pattern | Implementation Notes |
|---------|---------------------|---------------------|
| **Aliyun OSS** | SDK client wrapper | - Use `github.com/aliyun/aliyun-oss-go-sdk/oss`<br>- Initialize in app.go, inject into services<br>- Implement retry logic with exponential backoff<br>- Support multipart upload for large files<br>- Set public-read ACL for Tingwu access |
| **Aliyun Tingwu** | HTTP client with signature | - Implement RPC-style API client<br>- Use Alibaba Cloud credentials for signature<br>- RPC signature: SHA256 with HMAC<br>- Domain: tingwu.cn-beijing.aliyuncs.com<br>- Version: 2023-09-30<br>- Poll status every 30 seconds<br>- Implement webhook for async notifications |
| **FFmpeg** | Command execution | - Already integrated via `exec.Command`<br>- For splits: Use `-ss` seek + `-codec copy`<br>- Fast splitting: Seek to keyframes first<br>- Validate split points with ffprobe |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| **Handler ↔ Service** | Direct method calls | Handlers should be thin, services contain business logic |
| **Service ↔ External API** | Interface-based | Mockable for testing, isolated error handling |
| **Service ↔ Database** | GORM ORM | Use transactions for multi-step operations |
| **Background Workers** | Channel-based queue | Workers consume from shared queue, update DB status |
| **Frontend ↔ Backend** | REST API | Existing patterns with TanStack Query for caching |

### New vs Modified Components

**New Components:**
- `internal/handlers/transcription_handler.go` - Transcription task management
- `internal/handlers/video_split_handler.go` - Video split operations
- `internal/services/transcription/` - Complete transcription service package
- `internal/services/oss/` - OSS service for file upload/download
- `internal/services/video_split/` - Video splitting service
- `internal/models/transcription.go` - Transcription task model
- `internal/models/video_split.go` - Video split segment model

**Modified Components:**
- `internal/config/config.go` - Add OSS and Tingwu configuration sections
- `cmd/server/app.go` - Initialize new services and handlers
- `internal/models/video_file.go` - May need relation to splits/transcriptions
- `internal/models/ppt_file.go` - Link to transcription results
- Frontend: Add new pages/components for transcription and split UI

**Integration Strategy:**
1. Add new models with migrations
2. Implement OSS service (independent, testable)
3. Implement transcription service using OSS
4. Implement split service (independent, reuses FFmpeg)
5. Wire up handlers and routes
6. Frontend integration

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| **0-100 users** | Single process, SQLite, local FFmpeg, background worker pools |
| **100-1000 users** | Add job queue (Redis/RabbitMQ) for async tasks, PostgreSQL instead of SQLite |
| **1000+ users** | Microservice split (video processing separate), distributed task queue, cloud storage for all media |

### Scaling Priorities

1. **First bottleneck:** FFmpeg processing
   - Splitting/transcoding consumes CPU
   - Solution: Worker pool with max concurrency limit, queue system
   
2. **Second bottleneck:** External API rate limits
   - Aliyun Tingwu has concurrent task limits
   - Solution: Implement local queue with throttling, prioritize tasks

3. **Third bottleneck:** Storage
   - Video files + transcriptions + PPTs consume disk
   - Solution: OSS for all media, local cache only for hot data

### Performance Optimization

- **FFmpeg operations:** Use `-codec copy` for fast splitting (no re-encoding)
- **OSS uploads:** Multipart upload for files > 100MB
- **Database indexing:** Add indexes on status columns for efficient polling
- **Caching:** Cache Tingwu API responses to reduce calls
- **Concurrent processing:** Limit concurrent FFmpeg processes (system-dependent)

## Anti-Patterns

### Anti-Pattern 1: Synchronous External API Calls in Handlers

**What people do:** Call Aliyun APIs directly in HTTP handlers

**Why it's wrong:** 
- Blocks HTTP response (timeouts)
- No retry logic on failures
- No visibility into async operations

**Do this instead:**
```go
// Handler: Submit task and return immediately
func (h *TranscriptionHandler) SubmitTranscription(c *gin.Context) {
    transcriptionID, err := h.service.SubmitTask(taskID)
    // Return 202 Accepted with transcription ID
    response.GinSuccess(c, gin.H{
        "transcription_id": transcriptionID,
        "status": "pending",
    })
}

// Service: Queue for background processing
func (s *TranscriptionService) SubmitTask(taskID uint) (string, error) {
    // Create transcription record
    // Submit to worker queue
    // Return transcription ID immediately
}
```

### Anti-Pattern 2: Hardcoded External Service URLs

**What people do:** Hardcode OSS endpoint, Tingwu domain

**Why it's wrong:** 
- Cannot test with different environments
- Difficult to switch regions/providers

**Do this instead:**
```go
type TingwuConfig struct {
    Endpoint string `mapstructure:"endpoint" json:"endpoint"`
    Region   string `mapstructure:"region" json:"region"`
    // ... other config
}

// In config.yaml
tingwu:
  endpoint: "tingwu.cn-beijing.aliyuncs.com"
  region: "cn-beijing"
```

### Anti-Pattern 3: No Transaction Management for Multi-Step Operations

**What people do:** Save transcription, then PPT, then update task separately

**Why it's wrong:** 
- Partial updates on failure
- Inconsistent state

**Do this instead:**
```go
func (s *TranscriptionService) ProcessResult(taskID uint, result *TranscriptionResult) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 1. Save transcription
        // 2. Save PPT
        // 3. Update task status
        // All or nothing
        return nil
    })
}
```

### Anti-Pattern 4: Ignoring External API Rate Limits

**What people do:** Submit all transcription tasks immediately

**Why it's wrong:** 
- API throttling
- Failed tasks
- Wasted retries

**Do this instead:**
```go
type ThrottledTingwuClient struct {
    client       TingwuClient
    rateLimiter  *rate.Limiter
}

func (c *ThrottledTingwuClient) SubmitTask(ctx context.Context, req *SubmitTaskRequest) error {
    // Wait for rate limit
    if err := c.rateLimiter.Wait(ctx); err != nil {
        return err
    }
    return c.client.SubmitTask(ctx, req)
}
```

## Build Order

Given the dependencies between components, recommended build order:

### Phase 1: Foundation (No dependencies)
1. **OSS Service** - Independent, testable with mock uploads
2. **Configuration** - Add OSS/Tingwu config sections
3. **Database Models** - Add Transcription and SplitSegment models

### Phase 2: Core Services (Depends on Phase 1)
4. **Tingwu Client** - HTTP client with signature (depends on config)
5. **Transcription Service** - Uses OSS + Tingwu client
6. **Video Split Service** - Uses existing FFmpeg integration

### Phase 3: API Layer (Depends on Phase 2)
7. **Transcription Handler** - Exposes transcription endpoints
8. **Split Handler** - Exposes split endpoints

### Phase 4: Frontend Integration
9. **React Components** - Transcription UI, Split timeline UI
10. **State Management** - Zustand stores for new features

### Phase 5: Polish
11. **Error Handling** - Comprehensive error messages
12. **Notifications** - User notifications for async operations
13. **Testing** - Integration tests for happy paths

## Sources

- **Aliyun OSS Go SDK:** [https://github.com/aliyun/aliyun-oss-go-sdk](https://github.com/aliyun/aliyun-oss-go-sdk) (HIGH confidence - official SDK)
- **Aliyun Tingwu API:** Based on PROJECT.md specification (MEDIUM confidence - requires verification of API changes)
- **FFmpeg Documentation:** Existing codebase patterns (HIGH confidence - proven implementation)
- **Existing Architecture:** Code analysis of Record V2 codebase (HIGH confidence - actual implementation)

---
*Architecture research for: Video Recording Management with AI Transcription*
*Researched: 2025-04-17*
