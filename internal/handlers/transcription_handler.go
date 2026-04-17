package handlers

import (
	"strconv"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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

// SubmitTranscription handles POST /api/v1/videos/:id/transcribe
// Body: {"sampling_rate": 0.5, "mode": "local"} (both optional, mode defaults to local)
func (h *TranscriptionHandler) SubmitTranscription(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	var req struct {
		SamplingRate float64 `json:"sampling_rate"`
		Mode         string  `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// If body is empty or parsing fails, use defaults
		req.SamplingRate = 0.5 // Default: 1 frame per 2 seconds per D-02
		req.Mode = "local"
	}

	// Default mode to local if not specified (backward compatible)
	if req.Mode == "" {
		req.Mode = "local"
	}

	// Validate mode (per V5 Input Validation)
	validModes := map[string]bool{"local": true, "cloud": true}
	if !validModes[req.Mode] {
		response.GinError(c, response.CodeInvalidRequest, "无效的转录模式，必须是 local 或 cloud")
		return
	}

	// Per D-03: cloud mode does not need sampling_rate
	// The SubmitTranscriptionWithMode method handles this:
	// - For cloud mode: sampling_rate validation is skipped, any value is ignored
	// - For local mode: sampling_rate is validated (existing behavior)
	// For local mode, apply default sampling rate if not provided
	if req.Mode == "local" && req.SamplingRate == 0 {
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

	if err := h.transcriptionService.SubmitTranscriptionWithMode(uint(id), req.SamplingRate, req.Mode, userID); err != nil {
		response.GinError(c, response.CodeInternalError, "提交转录任务失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"video_file_id": id,
		"status":        "processing",
		"sampling_rate": req.SamplingRate,
		"mode":          req.Mode,
	})
}

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
		"status":             progress.Status,
		"current_stage":      progress.CurrentStage,
		"frames_processed":   progress.FramesProcessed,
		"total_frames":       progress.TotalFrames,
		"percentage":         progress.Percentage,
		"error_message":      progress.ErrorMessage,
		"result_ppt_file_id": progress.ResultPPTFileID,
		"mode":               progress.Mode,
	})
}

// GetTranscriptionText handles GET /api/v1/videos/:id/transcription-text
func (h *TranscriptionHandler) GetTranscriptionText(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	// Verify user owns the video file (per T-04-08 file ownership check)
	userID := middleware.GetUserID(c)
	file, err := h.videoFileService.GetFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "视频文件不存在")
		return
	}
	if !middleware.GetIsAdmin(c) && file.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权访问此视频文件的转录内容")
		return
	}

	// Find latest transcription task for this video
	var task models.TranscriptionTask
	if err := h.transcriptionService.GetDB().
		Where("video_file_id = ? AND status = ?", uint(id), models.TranscriptionStatusCompleted).
		Order("created_at DESC").
		First(&task).Error; err != nil {
		response.GinSuccess(c, gin.H{
			"segments":    []interface{}{},
			"total_count": 0,
		})
		return
	}

	// Get text segments
	var texts []models.TranscriptionText
	if err := h.transcriptionService.GetDB().
		Where("transcription_task_id = ?", task.ID).
		Order("segment_index ASC").
		Find(&texts).Error; err != nil {
		response.GinError(c, response.CodeInternalError, "获取文字内容失败")
		return
	}

	// Convert to response format
	segments := make([]gin.H, 0, len(texts))
	for _, t := range texts {
		segments = append(segments, gin.H{
			"text":          t.Text,
			"begin_time":    t.BeginTime,
			"end_time":      t.EndTime,
			"segment_index": t.SegmentIndex,
		})
	}

	response.GinSuccess(c, gin.H{
		"segments":    segments,
		"total_count": len(segments),
	})
}
