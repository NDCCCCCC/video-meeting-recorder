package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
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
	timestampMapper      *services.TimestampMapper
	cfg                  *config.Config
	logger               *zap.Logger
}

// NewTranscriptionHandler creates a new transcription handler
func NewTranscriptionHandler(
	transcriptionService *services.TranscriptionService,
	videoFileService *services.VideoFileService,
	timestampMapper *services.TimestampMapper,
	cfg *config.Config,
	logger *zap.Logger,
) *TranscriptionHandler {
	return &TranscriptionHandler{
		transcriptionService: transcriptionService,
		videoFileService:     videoFileService,
		timestampMapper:      timestampMapper,
		cfg:                  cfg,
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
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	file, err := h.videoFileService.GetFileByID(c.Request.Context(),uint(id))
	if err != nil {
		response.GinError(c, response.CodeInternalError, "视频文件不存在")
		return
	}
	if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
		response.GinError(c, response.CodeInvalidRequest, "无权操作此视频文件")
		return
	}

	if err := h.transcriptionService.SubmitTranscriptionWithMode(c.Request.Context(), uint(id), req.SamplingRate, req.Mode, userID); err != nil {
		h.logger.Warn("提交转录任务失败", zap.Uint64("video_id", id), zap.String("mode", req.Mode), zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
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
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	file, err := h.videoFileService.GetFileByID(c.Request.Context(),uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "视频文件不存在")
		return
	}
	if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
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
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	file, err := h.videoFileService.GetFileByID(c.Request.Context(),uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "视频文件不存在")
		return
	}
	if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权访问此视频文件的转录内容")
		return
	}

	// Find latest transcription task for this video
	var task models.TranscriptionTask
	if err := h.transcriptionService.GetDB().WithContext(c.Request.Context()).
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
	if err := h.transcriptionService.GetDB().WithContext(c.Request.Context()).
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

// ListActiveTasks handles GET /api/v1/transcriptions/active
// Returns all active transcription tasks (pending or processing)
func (h *TranscriptionHandler) ListActiveTasks(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}

	tasks, err := h.transcriptionService.GetActiveTasks(c.Request.Context())
	if err != nil {
		response.GinError(c, response.CodeInternalError, "获取活跃任务失败")
		return
	}

	// 优化：批量查询避免 N+1 问题
	if len(tasks) == 0 {
		response.GinSuccess(c, gin.H{
			"tasks": []gin.H{},
			"total": 0,
		})
		return
	}

	// 1. 收集所有 VideoFileID
	videoFileIDs := make([]uint, len(tasks))
	for i, task := range tasks {
		videoFileIDs[i] = task.VideoFileID
	}

	// 2. 一次性查询所有 VideoFile（PERF-002: IN 子句天然限定,但加 Limit 上限防止
	//    videoFileIDs 长度过大时的全表扫描风险）
	var videoFiles []models.VideoFile
	h.transcriptionService.GetDB().WithContext(c.Request.Context()).Where("id IN ?", videoFileIDs).Limit(1000).Find(&videoFiles)

	// 3. 创建 map 快速查找
	vfMap := make(map[uint]models.VideoFile, len(videoFiles))
	for _, vf := range videoFiles {
		vfMap[vf.ID] = vf
	}

	// 4. 过滤和构建响应
	filteredTasks := make([]gin.H, 0, len(tasks))
	for _, task := range tasks {
		videoFile, ok := vfMap[task.VideoFileID]
		if !ok {
			continue // Skip if video file not found
		}

		// Only show tasks owned by user (or all if admin/shared_viewer)
		if !middleware.CanAccessAllData(c) && videoFile.CreatedBy != userID {
			continue
		}

		filteredTasks = append(filteredTasks, gin.H{
			"id":              task.ID,
			"video_file_id":   task.VideoFileID,
			"status":          task.Status,
			"mode":            task.Mode,
			"sampling_rate":   task.SamplingRate,
			"current_stage":   task.CurrentStage,
			"percentage":      task.Percentage,
			"error_message":   task.ErrorMessage,
			"created_at":      task.CreatedAt,
			"video_file_name": videoFile.FileName,
		})
	}

	response.GinSuccess(c, gin.H{
		"tasks": filteredTasks,
		"total": len(filteredTasks),
	})
}

// GetTimestampMapHandler handles GET /api/v1/transcriptions/:videoFileId/timestamps
// Returns slide-to-timestamp mappings for video preview synchronization
func (h *TranscriptionHandler) GetTimestampMapHandler(c *gin.Context) {
	idStr := c.Param("videoFileId")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	videoFileID := uint(id)

	// Verify user owns the video file
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	file, err := h.videoFileService.GetFileByID(c.Request.Context(),videoFileID)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "视频文件不存在")
		return
	}
	if !middleware.CanAccessAllData(c) && file.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权访问此视频文件的时间戳映射")
		return
	}

	// Get timestamp map from service
	timestamps, err := h.timestampMapper.GetTimestampMap(c.Request.Context(), videoFileID)
	if err != nil {
		// Return empty array instead of error for graceful degradation
		response.GinSuccess(c, gin.H{
			"success":          true,
			"slide_timestamps": []interface{}{},
		})
		return
	}

	// Return timestamp map
	response.GinSuccess(c, gin.H{
		"success":          true,
		"slide_timestamps": timestamps,
	})
}

// SubmitBatchTranscription handles POST /api/v1/transcriptions/batch
func (h *TranscriptionHandler) SubmitBatchTranscription(c *gin.Context) {
	var req struct {
		VideoFileIDs []uint  `json:"video_file_ids" binding:"required,min=1,dive,min=1"`
		SamplingRate float64 `json:"sampling_rate"`
		Mode         string  `json:"mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "参数错误")
		return
	}

	// 设置默认值
	if req.Mode == "" {
		req.Mode = "local"
	}
	if req.Mode == "local" && req.SamplingRate == 0 {
		req.SamplingRate = 0.5
	}

	// 验证模式
	validModes := map[string]bool{"local": true, "cloud": true}
	if !validModes[req.Mode] {
		response.GinError(c, response.CodeInvalidRequest, "无效的转录模式，必须是 local 或 cloud")
		return
	}

	// 获取用户信息
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	isAdmin := middleware.CanAccessAllData(c)

	// PERF-005/D-03.8: 所有权校验改为 bounded concurrency，缩短大批量请求的延迟。
	concurrency := h.cfg.Transcription.BatchConcurrency
	if concurrency <= 0 {
		concurrency = 4
	}
	if concurrency > len(req.VideoFileIDs) {
		concurrency = len(req.VideoFileIDs)
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var checkErr string
	var checkMu sync.Mutex
	for _, videoFileID := range req.VideoFileIDs {
		wg.Add(1)
		sem <- struct{}{}
		go func(id uint) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					h.logger.Error("batch ownership check panicked",
						zap.Any("recover", r), zap.Stack("stack"))
				}
			}()
			file, err := h.videoFileService.GetFileByID(c.Request.Context(),id)
			if err != nil {
				checkMu.Lock()
				if checkErr == "" {
					checkErr = fmt.Sprintf("视频文件 %d 不存在", id)
				}
				checkMu.Unlock()
				return
			}
			if !isAdmin && file.CreatedBy != userID {
				checkMu.Lock()
				if checkErr == "" {
					checkErr = fmt.Sprintf("无权操作视频文件 %d", id)
				}
				checkMu.Unlock()
			}
		}(videoFileID)
	}
	wg.Wait()
	if checkErr != "" {
		if checkErr != "" && (checkErr[:len("视频文件")] == "视频文件") {
			response.GinError(c, response.CodeNotFound, checkErr)
		} else {
			response.GinError(c, response.CodeForbidden, checkErr)
		}
		return
	}

	// 调用服务层
	batchReq := &services.BatchTranscriptionRequest{
		VideoFileIDs: req.VideoFileIDs,
		SamplingRate: req.SamplingRate,
		Mode:         req.Mode,
		UserID:       userID,
	}
	result, err := h.transcriptionService.SubmitBatchTranscription(c.Request.Context(), batchReq)
	if err != nil {
		h.logger.Error("批量转录失败",
			zap.Uint("user_id", userID),
			zap.Int("file_count", len(req.VideoFileIDs)),
			zap.Error(err),
			response.SentinelField(err),
		)
		response.HandleError(c, err)
		return
	}

	h.logger.Info("批量转录已提交",
		zap.Uint("user_id", userID),
		zap.Uint("job_group_id", result.JobGroupID),
		zap.Int("total", result.TotalCount),
		zap.Int("submitted", result.SubmittedCount),
		zap.Int("failed", result.FailedCount),
	)

	// PERF-005/PR-F: 异步转录批任务返 202 + status URL,与 POST /admin/migrate-input-configs 同模式。
	c.Header("Location", fmt.Sprintf("/api/v1/transcriptions/batch/%d", result.JobGroupID))
	c.JSON(http.StatusAccepted, gin.H{
		"code":    response.CodeSuccess,
		"message": "已提交,异步执行中",
		"data": gin.H{
			"job_group_id": result.JobGroupID,
			"total":        result.TotalCount,
			"submitted":    result.SubmittedCount,
			"failed":       result.FailedCount,
			"status_url":   fmt.Sprintf("/api/v1/transcriptions/batch/%d", result.JobGroupID),
		},
	})
}

// GetBatchTranscriptionStatus handles GET /api/v1/transcriptions/batch/:id
func (h *TranscriptionHandler) GetBatchTranscriptionStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务组ID")
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.CanAccessAllData(c)

	jobGroup, err := h.transcriptionService.GetJobGroupStatus(c.Request.Context(), uint(id), userID, isAdmin)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "任务组不存在")
		return
	}

	response.GinSuccess(c, jobGroup)
}
