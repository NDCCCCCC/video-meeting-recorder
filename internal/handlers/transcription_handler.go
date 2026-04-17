package handlers

import (
	"strconv"

	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
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
		req.SamplingRate = 0.5 // Default: 1 frame per 2 seconds per D-02
	}

	// Validate sampling rate (per D-02: 1s/2s/5s intervals = 1.0/0.5/0.2 fps)
	if req.SamplingRate != 0 {
		validRates := map[float64]bool{1.0: true, 0.5: true, 0.2: true}
		if !validRates[req.SamplingRate] {
			response.GinError(c, response.CodeInvalidRequest, "无效的采样率，必须是 1.0, 0.5 或 0.2")
			return
		}
	} else {
		req.SamplingRate = 0.5 // Default to 0.5 if not provided
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
