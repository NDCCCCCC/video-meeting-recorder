# Phase 4: Cloud Services - Pattern Map

**Mapped:** 2026-04-17
**Files analyzed:** 13
**Analogs found:** 11 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/oss_service.go` | service | file-I/O | `internal/services/video_file_service.go` | role-match |
| `internal/services/tingwu_client.go` | service | request-response | `internal/services/huawei_config_service.go` | role-match |
| `internal/services/transcription_service.go` | service | event-driven | `internal/services/transcription_service.go` | self-extend |
| `internal/handlers/transcription_handler.go` | handler | request-response | `internal/handlers/transcription_handler.go` | self-extend |
| `internal/models/transcription_task.go` | model | CRUD | `internal/models/transcription_task.go` | self-extend |
| `internal/models/transcription_text.go` | model | CRUD | `internal/models/ppt_file.go` | role-match |
| `internal/config/config.go` | config | read-only | `internal/config/config.go` | self-extend |
| `frontend/src/components/TranscriptionProgressModal.tsx` | component | event-driven | `frontend/src/components/TranscriptionProgressModal.tsx` | self-extend |
| `frontend/src/components/TextContentTab.tsx` | component | request-response | `frontend/src/components/PPTPreview.tsx` | partial-match |
| `frontend/src/pages/files/index.tsx` | page | request-response | `frontend/src/pages/files/index.tsx` | self-extend |
| `frontend/src/pages/results/index.tsx` | page | request-response | `frontend/src/pages/results/index.tsx` | self-extend |
| `frontend/src/api/transcription.ts` | api | request-response | `frontend/src/api/transcription.ts` | self-extend |
| `frontend/src/types/transcription.ts` | types | read-only | `frontend/src/types/transcription.ts` | self-extend |

## Pattern Assignments

### `internal/services/oss_service.go` (service, file-I/O)

**Analog:** `internal/services/video_file_service.go`

**Imports pattern** (lines 1-18):
```go
package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

**Service struct pattern** (lines 29-50):
```go
// VideoFileService 视频文件服务
type VideoFileService struct {
	db             *gorm.DB
	logger         *zap.Logger
	recordingsPath string
	hlsPath        string
	ffprobePath    string
}

// NewVideoFileService 创建视频文件服务
func NewVideoFileService(db *gorm.DB, logger *zap.Logger, recordingsPath string, ffprobePath string) *VideoFileService {
	if ffprobePath == "" {
		ffprobePath = "./bin/ffprobe"
	}
	return &VideoFileService{
		db:             db,
		logger:         logger,
		recordingsPath: recordingsPath,
		hlsPath:        "",
		ffprobePath:    ffprobePath,
	}
}
```

**Error handling pattern** (lines 100-110):
```go
// ListFiles 获取文件列表
func (s *VideoFileService) ListFiles(req *ListFilesRequest) (*ListFilesResponse, error) {
	var files []models.VideoFile
	var total int64

	// Build query
	query := s.db.Model(&models.VideoFile{})

	// Apply filters
	if req.Keyword != "" {
		query = query.Where("file_name LIKE ?", "%"+req.Keyword+"%")
	}

	// Execute query
	if err := query.Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&files).Error; err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return &ListFilesResponse{
		Total: total,
		Items: files,
	}, nil
}
```

---

### `internal/services/tingwu_client.go` (service, request-response)

**Analog:** `internal/services/huawei_config_service.go`

**Imports pattern** (lines 1-15):
```go
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)
```

**Service struct pattern** (lines 17-31):
```go
// HuaweiConfigService 华为配置服务
type HuaweiConfigService struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

// NewHuaweiConfigService 创建华为配置服务
func NewHuaweiConfigService(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *HuaweiConfigService {
	return &HuaweiConfigService{
		db:     db,
		logger: logger,
		config: cfg,
	}
}
```

**Request/Response pattern** (lines 33-70):
```go
// ListConfigsRequest 配置列表请求
type ListConfigsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size" binding:"max=100"`
	Keyword  string `form:"keyword"`
	IsActive *bool  `form:"is_active"`
}

// ListConfigsResponse 配置列表响应
type ListConfigsResponse struct {
	Total int64                 `json:"total"`
	Items []models.HuaweiConfig `json:"items"`
}

// CreateConfigRequest 创建配置请求
type CreateConfigRequest struct {
	Name             string `json:"name" binding:"required,max=100"`
	Description      string `json:"description" binding:"max=500"`
	Server           string `json:"server" binding:"required,max=100"`
	Port             int    `json:"port" binding:"min=1,max=65535"`
	// ... more fields
}
```

---

### `internal/services/transcription_service.go` (service, event-driven - EXTEND)

**Analog:** `internal/services/transcription_service.go` (self-extension)

**Existing worker pool pattern** (lines 31-84):
```go
// TranscriptionService handles video transcription with worker pool
type TranscriptionService struct {
	db                 *gorm.DB
	logger             *zap.Logger
	config             *config.Config
	taskQueue          chan *models.TranscriptionTask
	workers            int
	cancelFuncs        map[uint]context.CancelFunc
	mu                 sync.RWMutex
	wg                 sync.WaitGroup
	ctx                context.Context
	cancel             context.CancelFunc
	ffmpegPath         string
	frameExtractor     *FrameExtractor
	similarityDetector *SimilarityDetector
	pptxGenerator      *PPTXGenerator
	videoFileService   *VideoFileService
	statusMap          map[uint]*TranscriptionProgress
	statusMu           sync.RWMutex
}

// NewTranscriptionService creates a new transcription service
func NewTranscriptionService(
	db *gorm.DB,
	logger *zap.Logger,
	cfg *config.Config,
	frameExtractor *FrameExtractor,
	similarityDetector *SimilarityDetector,
	pptxGenerator *PPTXGenerator,
	videoFileService *VideoFileService,
) *TranscriptionService {
	ctx, cancel := context.WithCancel(context.Background())
	ffmpegPath := cfg.FFmpeg.Path
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}

	return &TranscriptionService{
		db:                 db,
		logger:             logger,
		config:             cfg,
		taskQueue:          make(chan *models.TranscriptionTask, 100),
		workers:            1,
		cancelFuncs:        make(map[uint]context.CancelFunc),
		ctx:                ctx,
		cancel:             cancel,
		ffmpegPath:         ffmpegPath,
		frameExtractor:     frameExtractor,
		similarityDetector: similarityDetector,
		pptxGenerator:      pptxGenerator,
		videoFileService:   videoFileService,
		statusMap:          make(map[uint]*TranscriptionProgress),
	}
}
```

**Status map pattern** (lines 156-174):
```go
// GetTranscriptionStatus returns the current transcription progress
func (s *TranscriptionService) GetTranscriptionStatus(videoFileID uint) *TranscriptionProgress {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()

	if progress, ok := s.statusMap[videoFileID]; ok {
		// Return a copy to avoid concurrent access issues
		return &TranscriptionProgress{
			Status:           progress.Status,
			CurrentStage:     progress.CurrentStage,
			FramesProcessed:  progress.FramesProcessed,
			TotalFrames:      progress.TotalFrames,
			Percentage:       progress.Percentage,
			ErrorMessage:     progress.ErrorMessage,
			ResultPPTFileID:  progress.ResultPPTFileID,
		}
	}
	return nil
}
```

**Progress update pattern** (lines 406-449):
```go
// updateProgress updates the progress map for a video file
func (s *TranscriptionService) updateProgress(
	videoFileID uint,
	stage string,
	processed int,
	total int,
	percentage int,
	errorMsg string,
	resultPPTFileID *uint,
) {
	s.statusMu.Lock()
	defer s.statusMu.Unlock()

	// Get or create progress entry atomically
	progress, ok := s.statusMap[videoFileID]
	if !ok {
		progress = &TranscriptionProgress{
			Status: models.TranscriptionStatusProcessing,
		}
		s.statusMap[videoFileID] = progress
	}

	// Now update all fields atomically while holding lock
	if stage != "" {
		progress.CurrentStage = stage
	}
	if processed > 0 {
		progress.FramesProcessed = processed
	}
	if total > 0 {
		progress.TotalFrames = total
	}
	if percentage > 0 {
		progress.Percentage = percentage
	}
	if errorMsg != "" {
		progress.Status = models.TranscriptionStatusFailed
		progress.ErrorMessage = errorMsg
	}
	if resultPPTFileID != nil {
		progress.Status = models.TranscriptionStatusCompleted
		progress.ResultPPTFileID = resultPPTFileID
	}
}
```

---

### `internal/handlers/transcription_handler.go` (handler, request-response - EXTEND)

**Analog:** `internal/handlers/transcription_handler.go` (self-extension)

**Handler struct pattern** (lines 13-31):
```go
// TranscriptionHandler handles transcription API requests
type TranscriptionHandler struct {
	transcriptionService *services.TranscriptionService
	videoFileService     *services.VideoFileService
	logger               *zap.Logger
}

// NewTranscriptionHandler creates a new transcription handler
func NewTranscriptionHandler(
	transcriptionService *services.TranscriptionService,
	videoFileService *services.VideoFileService,
	logger *zap.Logger,
) *TranscriptionHandler {
	return &TranscriptionHandler{
		transcriptionService: transcriptionService,
		videoFileService:     videoFileService,
		logger:               logger,
	}
}
```

**Request handling pattern** (lines 33-84):
```go
// SubmitTranscription handles POST /api/v1/videos/:id/transcribe
// Body: {"sampling_rate": 0.5} (optional, defaults to 0.5 = 1 frame per 2 seconds)
func (h *TranscriptionHandler) SubmitTranscription(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	var req struct {
		SamplingRate float64 `json:"sampling_rate"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// If body is empty or parsing fails, use default sampling rate
		req.SamplingRate = 0.5
	}

	// Validate sampling rate
	if req.SamplingRate != 0 {
		validRates := map[float64]bool{1.0: true, 0.5: true, 0.2: true}
		if !validRates[req.SamplingRate] {
			response.GinError(c, response.CodeInvalidRequest, "无效的采样率，必须是 1.0, 0.5 或 0.2")
			return
		}
	} else {
		req.SamplingRate = 0.5
	}

	// Verify user owns the video file
	userID := middleware.GetUserID(c)
	file, err := h.videoFileService.GetFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeInternalError, "视频文件不存在")
		return
	}
	if !middleware.GetIsAdmin(c) && file.CreatedBy != userID {
		response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
		return
	}

	if err := h.transcriptionService.SubmitTranscription(uint(id), req.SamplingRate, userID); err != nil {
		response.GinError(c, response.CodeInternalError, "提交转录任务失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"video_file_id": id,
		"status":        "processing",
		"sampling_rate": req.SamplingRate,
	})
}
```

**Response pattern** (lines 86-122):
```go
// GetTranscriptionStatus handles GET /api/v1/videos/:id/transcription-status
func (h *TranscriptionHandler) GetTranscriptionStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	// Verify user owns the video file
	userID := middleware.GetUserID(c)
	file, err := h.videoFileService.GetFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "视频文件不存在")
		return
	}
	if !middleware.GetIsAdmin(c) && file.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权访问此视频文件的转录状态")
		return
	}

	progress := h.transcriptionService.GetTranscriptionStatus(uint(id))
	if progress == nil {
		response.GinError(c, response.CodeNotFound, "转录任务不存在")
		return
	}

	response.GinSuccess(c, gin.H{
		"status":           progress.Status,
		"current_stage":    progress.CurrentStage,
		"frames_processed": progress.FramesProcessed,
		"total_frames":     progress.TotalFrames,
		"percentage":       progress.Percentage,
		"error_message":    progress.ErrorMessage,
		"result_ppt_file_id": progress.ResultPPTFileID,
	})
}
```

---

### `internal/models/transcription_task.go` (model, CRUD - EXTEND)

**Analog:** `internal/models/transcription_task.go` (self-extension)

**Model struct pattern** (lines 3-19):
```go
// TranscriptionTask 转录任务模型
type TranscriptionTask struct {
	Base
	VideoFileID       uint       `gorm:"not null;index" json:"video_file_id"`
	VideoFile         *VideoFile `gorm:"foreignKey:VideoFileID" json:"video_file,omitempty"`
	SamplingRate      float64    `gorm:"default:0.5" json:"sampling_rate"`
	Status            string     `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	CurrentStage      string     `gorm:"type:varchar(50)" json:"current_stage"`
	FramesProcessed   int        `gorm:"default:0" json:"frames_processed"`
	TotalFrames       int        `gorm:"default:0" json:"total_frames"`
	Percentage        int        `gorm:"default:0" json:"percentage"`
	ResultPPTFileID   *uint      `json:"result_ppt_file_id,omitempty"`
	ResultPPTFile     *PPTFile   `gorm:"foreignKey:ResultPPTFileID" json:"result_ppt_file,omitempty"`
	ErrorMessage      string     `gorm:"type:text" json:"error_message,omitempty"`
	CreatedBy         uint       `gorm:"not null" json:"created_by"`
	Creator           *User      `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}
```

**Constants pattern** (lines 21-34):
```go
// 转录状态常量
const (
	TranscriptionStatusPending    = "pending"
	TranscriptionStatusProcessing = "processing"
	TranscriptionStatusCompleted  = "completed"
	TranscriptionStatusFailed     = "failed"
)

// 转录阶段常量
const (
	TranscriptionStageExtracting = "extracting"
	TranscriptionStageDetecting  = "detecting"
	TranscriptionStageGenerating = "generating"
)
```

---

### `internal/models/transcription_text.go` (model, CRUD)

**Analog:** `internal/models/ppt_file.go`

**Model struct pattern** (lines 1-30):
```go
package models

import "time"

// PPTFile PPT文件模型
type PPTFile struct {
	Base
	FileName            string     `gorm:"not null;size:255" json:"file_name"`
	FilePath            string     `gorm:"not null;size:500" json:"file_path"`
	FileSize            int64      `gorm:"not null" json:"file_size"`
	PageCount           int        `gorm:"default:0" json:"page_count"`
	Format              string     `gorm:"size:10" json:"format"` // ppt, pptx
	SourceVideoFileID   *uint      `json:"source_video_file_id,omitempty"`
	SourceVideoFile     *VideoFile `gorm:"foreignKey:SourceVideoFileID" json:"source_video_file,omitempty"`
	TranscriptionTaskID *uint      `json:"transcription_task_id,omitempty"`
	TranscriptionTask   *TranscriptionTask `gorm:"foreignKey:TranscriptionTaskID" json:"transcription_task,omitempty"`
}

// TableName 指定表名
func (PPTFile) TableName() string {
	return "ppt_files"
}
```

**Base model pattern** (from `internal/models/base.go`):
```go
package models

import (
	"time"

	"gorm.io/gorm"
)

// Base 基础模型
type Base struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
```

---

### `internal/config/config.go` (config, read-only - EXTEND)

**Analog:** `internal/config/config.go` (self-extension)

**Config struct pattern** (lines 13-23):
```go
// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server" json:"server" yaml:"server"`
	Database DatabaseConfig `mapstructure:"database" json:"database" yaml:"database"`
	Auth     AuthConfig     `mapstructure:"auth" json:"auth" yaml:"auth"`
	Logging  LoggingConfig  `mapstructure:"logging" json:"logging" yaml:"logging"`
	Storage  StorageConfig  `mapstructure:"storage" json:"storage" yaml:"storage"`
	Huawei   HuaweiConfig   `mapstructure:"huawei" json:"huawei" yaml:"huawei"`
	RTSP     RTSPConfig     `mapstructure:"rtsp" json:"rtsp" yaml:"rtsp"`
	FFmpeg   FFmpegConfig   `mapstructure:"ffmpeg" json:"ffmpeg" yaml:"ffmpeg"`
}
```

**Sub-config pattern** (lines 89-101):
```go
// HuaweiConfig 华为会议系统配置
type HuaweiConfig struct {
	ConferenceServer   string        `mapstructure:"conference_server" json:"conference_server" yaml:"conference_server"`
	ConferencePort     int           `mapstructure:"conference_port" json:"conference_port" yaml:"conference_port"`
	Username           string        `mapstructure:"username" json:"username" yaml:"username"`
	Password           string        `mapstructure:"password" json:"password" yaml:"password"`
	HTTPS              bool          `mapstructure:"https" json:"https" yaml:"https"`
	InsecureSkipVerify bool          `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	APITimeout         time.Duration `mapstructure:"api_timeout" json:"api_timeout" yaml:"api_timeout"`
	SessionTimeout     time.Duration `mapstructure:"session_timeout" json:"session_timeout" yaml:"session_timeout"`
	KeepAliveInterval  time.Duration `mapstructure:"keep_alive_interval" json:"keep_alive_interval" yaml:"keep_alive_interval"`
	MinTLSVersion      string        `mapstructure:"min_tls_version" json:"min_tls_version" yaml:"min_tls_version"`
}
```

**Environment variable expansion pattern** (lines 136-161):
```go
// expandEnvWithDefault 展开环境变量，支持 ${VAR:default} 格式
func expandEnvWithDefault(s string) string {
	// 匹配 ${VAR:default} 或 ${VAR} 格式
	re := regexp.MustCompile(`\$\{([^:}]+)(?::([^}]*))?\}`)

	return re.ReplaceAllStringFunc(s, func(match string) string {
		parts := re.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		varName := parts[1]
		defaultValue := ""
		if len(parts) >= 3 {
			defaultValue = parts[2]
		}

		// 优先使用环境变量的值
		if envValue := os.Getenv(varName); envValue != "" {
			return envValue
		}

		// 否则使用默认值
		return defaultValue
	})
}
```

---

### `frontend/src/components/TranscriptionProgressModal.tsx` (component, event-driven - EXTEND)

**Analog:** `frontend/src/components/TranscriptionProgressModal.tsx` (self-extension)

**Component props pattern** (lines 15-22):
```typescript
interface TranscriptionProgressModalProps {
  open: boolean
  onClose: () => void
  videoFileId: number
  fileName: string
  samplingRate: number
  onCompleted: (pptFileId: number) => void
}
```

**State management pattern** (lines 39-46):
```typescript
const [status, setStatus] = useState<TranscriptionTaskStatus>('pending')
const [stage, setStage] = useState<TranscriptionStage | ''>('')
const [framesProcessed, setFramesProcessed] = useState(0)
const [totalFrames, setTotalFrames] = useState(0)
const [percentage, setPercentage] = useState(0)
const [errorMessage, setErrorMessage] = useState('')
const [pptFileId, setPptFileId] = useState<number | null>(null)
```

**Polling pattern** (lines 48-83):
```typescript
// 轮询获取转录状态 (per D-16, 5-second interval)
useEffect(() => {
  if (!open) return

  // 立即获取一次状态
  const fetchStatus = async () => {
    try {
      const response = await getTranscriptionStatus(videoFileId)
      if (response.data) {
        const data = response.data
        setStatus(data.status)
        setStage(data.current_stage)
        setFramesProcessed(data.frames_processed)
        setTotalFrames(data.total_frames)
        setPercentage(data.percentage)
        setErrorMessage(data.error_message)
        setPptFileId(data.result_ppt_file_id)

        // 转录完成
        if (data.status === 'completed' && data.result_ppt_file_id) {
          onCompleted(data.result_ppt_file_id)
        }
      }
    } catch (error) {
      console.error('获取转录状态失败:', error)
      // 不显示错误消息，静默失败继续轮询
    }
  }

  fetchStatus()

  // 设置轮询间隔
  const interval = setInterval(fetchStatus, 5000) // 5 seconds per D-16

  return () => clearInterval(interval) // 清理定时器
}, [open, videoFileId, onCompleted])
```

**Stage rendering pattern** (lines 96-142):
```typescript
const renderStages = useCallback(() => {
  const stages: TranscriptionStage[] = ['extracting', 'detecting', 'generating']
  const stageIndex = stages.indexOf(stage as TranscriptionStage)

  return (
    <div style={{ marginTop: 16 }}>
      {stages.map((s, index) => {
        const config = STAGE_CONFIG[s]
        const isCompleted = index < stageIndex
        const isActive = index === stageIndex && status === 'processing'
        const isPending = index > stageIndex

        let icon: React.ReactNode
        let text: string
        let textStyle: React.CSSObject = {}

        if (isCompleted) {
          icon = <CheckCircleOutlined style={{ color: '#52c41a' }} />
          text = `✓ ${config.label}...`
          textStyle = { color: '#52c41a' }
        } else if (isActive) {
          icon = config.icon
          if (s === 'detecting' && framesProcessed > 0 && totalFrames > 0) {
            text = `● ${config.label} (${framesProcessed}/${totalFrames})...`
          } else {
            text = `● ${config.label}...`
          }
          textStyle = { color: '#1890ff' }
        } else {
          icon = <span style={{ color: '#d9d9d9' }}>○</span>
          text = `○ ${config.label}...`
          textStyle = { color: '#d9d9d9' }
        }

        return (
          <div key={s} style={{ marginBottom: 8, fontSize: 14, ...textStyle }}>
            <Space size={8}>
              {icon}
              <span>{text}</span>
            </Space>
          </div>
        )
      })}
    </div>
  )
}, [stage, status, framesProcessed, totalFrames])
```

---

### `frontend/src/components/TextContentTab.tsx` (component, request-response)

**Analog:** `frontend/src/components/PPTPreview.tsx`

**Component structure pattern** (from `PPTPreview.tsx`):
```typescript
import { useState, useEffect } from 'react'
import { Spin, Empty } from 'antd'
import type { SlideImage } from '../types/ppt'

interface PPTPreviewProps {
  pptId: number
  currentPage: number
  onPageChange: (page: number) => void
}

export default function PPTPreview({ pptId, currentPage, onPageChange }: PPTPreviewProps) {
  const [slides, setSlides] = useState<SlideImage[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    const loadSlides = async () => {
      setLoading(true)
      try {
        // Load data
      } catch (error) {
        console.error('Error loading slides:', error)
      } finally {
        setLoading(false)
      }
    }
    loadSlides()
  }, [pptId])

  if (loading) {
    return <Spin />
  }

  return (
    <div>
      {/* Render content */}
    </div>
  )
}
```

---

### `frontend/src/pages/files/index.tsx` (page, request-response - EXTEND)

**Analog:** `frontend/src/pages/files/index.tsx` (self-extension)

**Button action pattern** (from lines 200-300):
```typescript
// 转录处理
const handleTranscribe = useCallback(
  async (file: VideoFile, samplingRate: number) => {
    setTranscriptionVideoFile(file)
    setTranscriptionModalOpen(true)
    setSelectedSamplingRate(samplingRate)
    
    try {
      await submitTranscription(file.id, samplingRate)
      message.success('转录任务已提交')
    } catch (error) {
      message.error(error instanceof Error ? error.message : '提交转录任务失败')
    }
  },
  []
)
```

**Table column pattern** (from existing table):
```typescript
const columns: ColumnsType<VideoFile> = [
  // ... other columns
  {
    title: '操作',
    key: 'action',
    width: 200,
    render: (_, record) => (
      <Space size="small">
        <Button
          type="primary"
          size="small"
          icon={<FileTextOutlined />}
          onClick={() => handleTranscribe(record, 0.5)}
        >
          转录
        </Button>
        {/* Other actions */}
      </Space>
    ),
  },
]
```

---

### `frontend/src/pages/results/index.tsx` (page, request-response - EXTEND)

**Analog:** `frontend/src/pages/results/index.tsx` (self-extension)

**Page structure pattern** (lines 55-102):
```typescript
export default function ResultDetailPage() {
  const navigate = useNavigate()
  const { videoFileId } = useParams<{ videoFileId: string }>()
  const videoFileIdNum = parseInt(videoFileId || '0', 10)

  // State
  const [ppts, setPpts] = useState<PPTResult[]>([])
  const [currentPptId, setCurrentPptId] = useState<number>(0)
  const [slides, setSlides] = useState<SlideImage[]>([])
  const [currentSlide, setCurrentSlide] = useState(0)
  const [loading, setLoading] = useState(false)

  // Load data
  const loadPpts = useCallback(async () => {
    if (!videoFileIdNum) return

    setLoading(true)
    try {
      const response = await getPptsByVideo(videoFileIdNum)
      if (response.data && response.data.ppts) {
        const sortedPpts = response.data.ppts.sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )
        setPpts(sortedPpts)
        if (sortedPpts.length > 0) {
          setCurrentPptId(sortedPpts[0].id)
        }
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 PPT 列表失败')
    } finally {
      setLoading(false)
    }
  }, [videoFileIdNum])

  useEffect(() => {
    loadPpts()
  }, [loadPpts])

  return (
    <div>
      {/* Render content */}
    </div>
  )
}
```

---

### `frontend/src/api/transcription.ts` (api, request-response - EXTEND)

**Analog:** `frontend/src/api/transcription.ts` (self-extension)

**API client pattern** (lines 1-33):
```typescript
// 本地转录 API 客户端

import type { ApiResponse } from '../types/auth'
import type {
  TranscriptionTriggerRequest,
  TranscriptionTriggerResponse,
  TranscriptionStatusResponse
} from '../types/transcription'
import { apiRequest } from './apiClient'

// 提交转录任务
export async function submitTranscription(
  videoFileId: number,
  samplingRate: number = 0.5
): Promise<ApiResponse<TranscriptionTriggerResponse>> {
  return apiRequest<TranscriptionTriggerResponse>(
    `/api/v1/videos/${videoFileId}/transcribe`,
    {
      method: 'POST',
      body: JSON.stringify({ sampling_rate: samplingRate }),
    }
  )
}

// 获取转录状态
export async function getTranscriptionStatus(
  videoFileId: number
): Promise<ApiResponse<TranscriptionStatusResponse>> {
  return apiRequest<TranscriptionStatusResponse>(
    `/api/v1/videos/${videoFileId}/transcription-status`
  )
}
```

---

### `frontend/src/types/transcription.ts` (types, read-only - EXTEND)

**Analog:** `frontend/src/types/transcription.ts` (self-extension)

**Type definitions pattern** (lines 1-38):
```typescript
// 本地转录 API 类型定义

// 转录触发请求
export interface TranscriptionTriggerRequest {
  sampling_rate: number
}

// 转录触发响应
export interface TranscriptionTriggerResponse {
  video_file_id: number
  status: string
}

// 转录进度阶段
export type TranscriptionStage = 'extracting' | 'detecting' | 'generating'

// 转录任务状态
export type TranscriptionTaskStatus = 'pending' | 'processing' | 'completed' | 'failed'

// 转录状态响应
export interface TranscriptionStatusResponse {
  status: TranscriptionTaskStatus
  current_stage: TranscriptionStage | ''
  frames_processed: number
  total_frames: number
  percentage: number
  error_message: string
  result_ppt_file_id: number | null
}

// 采样率选项
export interface SamplingRateOption {
  label: string
  value: number
  secondsPerFrame: number
  description: string
}
```

---

## Shared Patterns

### Authentication/Authorization
**Source:** `internal/handlers/transcription_handler.go` (lines 62-72)
**Apply to:** All handler files
```go
// Verify user owns the video file
userID := middleware.GetUserID(c)
file, err := h.videoFileService.GetFileByID(uint(id))
if err != nil {
	response.GinError(c, response.CodeInternalError, "视频文件不存在")
	return
}
if !middleware.GetIsAdmin(c) && file.CreatedBy != userID {
	response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
	return
}
```

### Error Handling
**Source:** `pkg/response` (used throughout handlers)
**Apply to:** All handler files
```go
// Success response
response.GinSuccess(c, gin.H{
	"video_file_id": id,
	"status":        "processing",
	"sampling_rate": req.SamplingRate,
})

// Error response
response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
```

### Service Registration
**Source:** `cmd/server/app.go` (lines 450-542)
**Apply to:** New OSS and Tingwu services
```go
// In initHandlers()
ossService := services.NewOSSService(cfg, logger)
tingwuClient := services.NewTingwuClient(cfg, logger)

// Update transcription service with cloud dependencies
a.transcriptionService = services.NewTranscriptionService(
	a.db, a.logger, a.config,
	frameExtractor, similarityDetector, pptxGenerator,
	a.videoFileService,
	ossService, tingwuClient, // New dependencies
)

// In Handlers struct
a.handlers.Transcription = handlers.NewTranscriptionHandler(
	a.transcriptionService, 
	a.videoFileService, 
	a.logger,
)
```

### Config Environment Variable Expansion
**Source:** `internal/config/config.go` (lines 136-161)
**Apply to:** OSS and Tingwu config sections
```go
// expandEnvWithDefault supports ${VAR:default} format
// Example .env usage:
// ALIYUN_OSS_ENDPOINT=${ALIYUN_OSS_ENDPOINT:https://oss-cn-hangzhou.aliyuncs.com}
// ALIYUN_OSS_BUCKET=${ALIYUN_OSS_BUCKET:my-bucket}
```

### Frontend API Request Pattern
**Source:** `frontend/src/api/apiClient.ts`
**Apply to:** All new API endpoints
```typescript
import { apiRequest } from './apiClient'

export async function newApiFunction(
  param: number
): Promise<ApiResponse<ResponseType>> {
  return apiRequest<ResponseType>(
    `/api/v1/resource/${param}/endpoint`,
    {
      method: 'POST',
      body: JSON.stringify({ key: 'value' }),
    }
  )
}
```

### Frontend Polling Pattern
**Source:** `frontend/src/components/TranscriptionProgressModal.tsx` (lines 48-83)
**Apply to:** Cloud transcription status polling
```typescript
useEffect(() => {
  if (!open) return

  const fetchStatus = async () => {
    try {
      const response = await getTranscriptionStatus(videoFileId)
      // Process response
    } catch (error) {
      console.error('Error:', error)
    }
  }

  fetchStatus()
  const interval = setInterval(fetchStatus, pollInterval) // 5000 for local, 10000 for cloud
  return () => clearInterval(interval)
}, [open, videoFileId, pollInterval])
```

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/services/tingwu_client.go` | service | request-response | No existing REST API client with HMAC signing |
| `internal/services/oss_service.go` | service | file-I/O | No existing cloud storage service (all local file operations) |
| `frontend/src/components/TextContentTab.tsx` | component | request-response | No existing text display with clickable timestamps |

**Note:** For TingwuClient and OSSService, use the code examples provided in RESEARCH.md sections:
- TingwuClient: Use RESEARCH.md "Pattern 2: TingwuClient with HMAC-SHA256 Signing" (lines 240-313)
- OSSService: Use RESEARCH.md "Pattern 1: OSSService with Presigned URLs" (lines 189-238)
- TextContentTab: Use RESEARCH.md "Frontend Text Content Tab with Timestamps" (lines 990-1113)

---

## Metadata

**Analog search scope:** 
- `internal/services/**/*.go`
- `internal/handlers/**/*.go`
- `internal/models/**/*.go`
- `internal/config/config.go`
- `frontend/src/components/**/*.tsx`
- `frontend/src/pages/**/*.tsx`
- `frontend/src/api/**/*.ts`
- `frontend/src/types/**/*.ts`
- `cmd/server/app.go`

**Files scanned:** 25
**Pattern extraction date:** 2026-04-17
**Phase:** 04-cloud-services
**Confidence:** HIGH (11/13 files have close analogs in existing codebase)

---

## Implementation Notes

1. **Self-extension files** (TranscriptionService, TranscriptionHandler, TranscriptionTask, etc.) should follow the exact patterns established in the existing files, adding cloud mode branches alongside local mode logic.

2. **New services** (OSSService, TingwuClient) should follow the structural patterns from existing services but implement the specific functionality from RESEARCH.md code examples.

3. **Frontend components** should maintain consistency with existing Ant Design patterns, using the same component libraries and styling approaches.

4. **Config extensions** should follow the environment variable expansion pattern established for HuaweiConfig, allowing flexible credential management via .env file.

5. **API endpoint extensions** should maintain the existing RESTful patterns and response formats, adding `mode` parameter support without breaking existing functionality.
