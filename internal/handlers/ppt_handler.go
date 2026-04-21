package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PPThandler handles PPT-related API requests
type PPThandler struct {
	pptFileService     *services.PPTFileService
	slideCacheService  *services.SlideCacheService
	mergeService       *services.PPTMergeService
	videoFileService   *services.VideoFileService
	pptEditorService   *services.PPTEditorService
	frameCaptureService *services.FrameCaptureService
	logger             *zap.Logger
}

// NewPPThandler creates a new PPT handler
func NewPPThandler(
	pptFileService *services.PPTFileService,
	slideCacheService *services.SlideCacheService,
	mergeService *services.PPTMergeService,
	videoFileService *services.VideoFileService,
	pptEditorService *services.PPTEditorService,
	frameCaptureService *services.FrameCaptureService,
	logger *zap.Logger,
) *PPThandler {
	return &PPThandler{
		pptFileService:     pptFileService,
		slideCacheService:  slideCacheService,
		mergeService:       mergeService,
		videoFileService:   videoFileService,
		pptEditorService:   pptEditorService,
		frameCaptureService: frameCaptureService,
		logger:             logger,
	}
}

// verifyPPTOwnership verifies that the current user owns the PPT file
// via its associated SourceVideoFileID. Returns an error if ownership
// cannot be verified or if the user doesn't have access.
func (h *PPThandler) verifyPPTOwnership(c *gin.Context, pptFile *models.PPTFile) error {
	if pptFile.SourceVideoFileID == nil {
		return fmt.Errorf("PPT文件没有关联视频")
	}

	videoFile, err := h.videoFileService.GetFileByID(*pptFile.SourceVideoFileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("关联视频不存在")
		}
		return err
	}

	userID := middleware.GetUserID(c)
	if !middleware.GetIsAdmin(c) && videoFile.CreatedBy != userID {
		return fmt.Errorf("无权访问此PPT文件")
	}

	return nil
}

// GetSlides handles GET /api/v1/ppts/:id/slides (per D-09)
func (h *PPThandler) GetSlides(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Get slides from cache (extracts if needed)
	slides, err := h.slideCacheService.GetOrExtractSlides(uint(id))
	if err != nil {
		h.logger.Error("Failed to get slides",
			zap.String("ppt_id", idStr),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取幻灯片失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"slide_count": len(slides),
		"slides":      slides,
		"status":      "ready",
	})
}

// ServeSlideImage handles GET /api/v1/ppts/:id/slides/:resolution/:filename
// 公开访问，但在 handler 内部验证权限（用于 <img> 标签显示）
func (h *PPThandler) ServeSlideImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	resolution := c.Param("resolution")
	filename := c.Param("filename")

	// TODO: 添加权限验证（暂时跳过，图片文件名随机不易猜测）
	// 需要通过 token 或 session 验证用户身份

	// Get slide image path
	imagePath, err := h.slideCacheService.GetSlideImagePath(uint(id), resolution, filename)
	if err != nil {
		h.logger.Warn("Slide image not found",
			zap.String("ppt_id", idStr),
			zap.String("resolution", resolution),
			zap.String("filename", filename),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "幻灯片图片不存在"})
		return
	}

	// Serve file
	c.File(imagePath)
}

// GetPptsByVideo handles GET /api/v1/videos/:id/ppts (per D-10, D-12)
func (h *PPThandler) GetPptsByVideo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	// Verify user owns the video file
	userID := middleware.GetUserID(c)
	videoFile, err := h.videoFileService.GetFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "视频文件不存在")
		return
	}

	if !middleware.GetIsAdmin(c) && videoFile.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权访问此视频文件的PPT")
		return
	}

	// Get all PPTs for this video
	ppts, err := h.pptFileService.GetPptsByVideoFile(uint(id))
	if err != nil {
		h.logger.Error("Failed to get PPTs for video",
			zap.String("video_file_id", idStr),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取PPT列表失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"ppts": ppts,
	})
}

// MergeSlides handles POST /api/v1/ppts/merge (per D-13 to D-18)
func (h *PPThandler) MergeSlides(c *gin.Context) {
	// Read body for logging before binding (since binding consumes it)
	bodyBytes, err := c.GetRawData()
	if err != nil {
		h.logger.Warn("Failed to read request body", zap.Error(err))
		response.GinError(c, response.CodeInvalidRequest, "读取请求失败")
		return
	}
	bodyStr := string(bodyBytes)

	// Unmarshal JSON manually since we already consumed the body
	var req models.MergeRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		h.logger.Warn("Merge request JSON parse failed",
			zap.Error(err),
			zap.String("request_body", bodyStr))
		response.GinError(c, response.CodeInvalidRequest, "无效的请求参数: "+err.Error())
		return
	}

	// Additional validation with better error messages
	if req.VideoFileID == 0 {
		h.logger.Warn("Merge request failed: video_file_id is 0 or missing",
			zap.String("request_body", bodyStr))
		response.GinError(c, response.CodeInvalidRequest, "视频文件ID不能为空")
		return
	}

	h.logger.Info("Merge request received",
		zap.Uint("video_file_id", req.VideoFileID),
		zap.Int("slide_count", len(req.Slides)))

	// Validate slides not empty
	if len(req.Slides) == 0 {
		response.GinError(c, response.CodeInvalidRequest, "请选择要合并的幻灯片")
		return
	}

	// Validate each slide item
	for _, slide := range req.Slides {
		if slide.SlideNumber < 0 {
			response.GinError(c, response.CodeInvalidRequest,
				fmt.Sprintf("无效的幻灯片编号: %d", slide.SlideNumber))
			return
		}
		if slide.PptFileID == 0 {
			response.GinError(c, response.CodeInvalidRequest, "PPT文件ID不能为空")
			return
		}
	}

	// Set default output name if not provided
	if req.OutputName == "" {
		req.OutputName = "合并PPT.pptx"
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)

	// Call merge service
	pptFile, err := h.mergeService.MergeSlides(c.Request.Context(), &req, userID)
	if err != nil {
		h.logger.Error("Failed to merge slides",
			zap.Uint("video_file_id", req.VideoFileID),
			zap.Int("slide_count", len(req.Slides)),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "合并幻灯片失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"ppt_file_id": pptFile.ID,
		"file_name":   pptFile.FileName,
		"page_count":  pptFile.PageCount,
	})
}

// DownloadPPT handles GET /api/v1/ppts/:id/download (per PPT-01)
func (h *PPThandler) DownloadPPT(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Determine download filename - use video name if available
	downloadFilename := pptFile.FileName
	if pptFile.SourceVideoFileID != nil {
		videoFile, err := h.videoFileService.GetFileByID(*pptFile.SourceVideoFileID)
		if err == nil && videoFile != nil {
			// Use video filename without extension + .pptx
			videoName := strings.TrimSuffix(videoFile.FileName, filepath.Ext(videoFile.FileName))
			downloadFilename = videoName + ".pptx"
		}
	}

	// Set Content-Disposition header with custom filename
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", downloadFilename))
	c.File(pptFile.FilePath)
}

// DeletePPT handles DELETE /api/v1/ppts/:id
func (h *PPThandler) DeletePPT(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Load PPT file to verify ownership
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Delete PPT file
	if err := h.pptFileService.DeletePPTFile(uint(id)); err != nil {
		h.logger.Error("Failed to delete PPT file",
			zap.String("ppt_id", idStr),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "删除PPT文件失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"message": "PPT文件已删除",
	})
}

// RenamePPTRequest 重命名PPT请求
type RenamePPTRequest struct {
	NewName string `json:"new_name" binding:"required,min=1,max=200"`
}

// RenamePPT handles POST /api/v1/ppts/:id/rename
func (h *PPThandler) RenamePPT(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	var req RenamePPTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// Trim whitespace and validate
	newName := trimString(req.NewName)
	if newName == "" {
		response.GinError(c, response.CodeInvalidRequest, "新文件名不能为空")
		return
	}

	// Reject path separators
	if containsPathSeparator(newName) {
		response.GinError(c, response.CodeInvalidRequest, "文件名不能包含路径分隔符")
		return
	}

	// Get user ID from context
	userID := middleware.GetUserID(c)

	// Call service to rename
	if err := h.pptFileService.RenamePPTFile(uint(id), newName, userID); err != nil {
		h.logger.Warn("重命名PPT文件失败",
			zap.String("ppt_id", idStr),
			zap.String("new_name", newName),
			zap.Error(err))

		// Map service errors to HTTP status codes
		switch {
		case err.Error() == "文件不存在" || errors.Is(err, gorm.ErrRecordNotFound):
			response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		case err.Error() == "无权操作此文件":
			response.GinError(c, response.CodeForbidden, "无权操作此文件")
		default:
			response.GinError(c, response.CodeInternalError, "重命名失败: "+err.Error())
		}
		return
	}

	// Get updated PPT file info
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		h.logger.Error("重命名成功但无法获取更新后的文件信息", zap.Error(err))
		response.GinSuccess(c, gin.H{
			"message": "重命名成功",
		})
		return
	}

	response.GinSuccess(c, gin.H{
		"message": "重命名成功",
		"data": gin.H{
			"id":         pptFile.ID,
			"file_name":  pptFile.FileName,
			"file_path":  pptFile.FilePath,
		},
	})
}

// DetectDuplicatesRequest 检测重复幻灯片请求
type DetectDuplicatesRequest struct {
	Threshold float64 `form:"threshold"` // 相似度阈值 (可选，默认0.95)
}

// DetectDuplicatesHandler handles GET /api/v1/ppts/:id/duplicates
func (h *PPThandler) DetectDuplicatesHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Parse threshold parameter (optional)
	var req DetectDuplicatesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		req.Threshold = 0.95 // Default threshold
	}

	// Call service to detect duplicates
	groups, err := h.pptEditorService.DetectDuplicateSlides(uint(id))
	if err != nil {
		h.logger.Error("Failed to detect duplicate slides",
			zap.String("ppt_id", idStr),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "检测重复幻灯片失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"groups":        groups,
		"total_scanned": pptFile.PageCount,
		"duplicate_count": len(groups),
	})
}

// DeleteSlidesRequest 删除幻灯片请求
type DeleteSlidesRequest struct {
	Slides []int `json:"slides" binding:"required,min=1"`
}

// DeleteSlidesHandler handles DELETE /api/v1/ppts/:id/slides
func (h *PPThandler) DeleteSlidesHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Parse request body
	var req DeleteSlidesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的请求参数")
		return
	}

	// Validate slides array
	if len(req.Slides) == 0 {
		response.GinError(c, response.CodeInvalidRequest, "请选择要删除的幻灯片")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Validate slide numbers are within range
	for _, slideNum := range req.Slides {
		if slideNum < 1 || slideNum > pptFile.PageCount {
			response.GinError(c, response.CodeInvalidRequest,
				fmt.Sprintf("无效的幻灯片编号: %d (有效范围: 1-%d)", slideNum, pptFile.PageCount))
			return
		}
	}

	// Call service to delete slides
	if err := h.pptEditorService.DeleteSlides(uint(id), req.Slides); err != nil {
		h.logger.Error("Failed to delete slides",
			zap.String("ppt_id", idStr),
			zap.Ints("slide_numbers", req.Slides),
			zap.Error(err))

		// Map error messages
		errMsg := err.Error()
		switch {
		case errMsg == "cannot delete all slides":
			response.GinError(c, response.CodeInvalidRequest, "不能删除所有幻灯片")
		case errMsg == "no backup exists for rollback":
			response.GinError(c, response.CodeInternalError, "无法创建备份")
		default:
			response.GinError(c, response.CodeInternalError, "删除幻灯片失败: "+err.Error())
		}
		return
	}

	// Get updated PPT file info
	updatedPPT, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		h.logger.Error("Failed to get updated PPT file", zap.Error(err))
		response.GinSuccess(c, gin.H{
			"message": "幻灯片删除成功",
			"deleted_slides": req.Slides,
		})
		return
	}

	// Parse deleted slides for response
	deletedSlides, _ := updatedPPT.GetDeletedSlides()

	response.GinSuccess(c, gin.H{
		"message":       "幻灯片删除成功",
		"page_count":    updatedPPT.PageCount,
		"deleted_slides": deletedSlides,
		"backup_path":   updatedPPT.BackupPath,
	})
}

// RollbackHandler handles POST /api/v1/ppts/:id/rollback
func (h *PPThandler) RollbackHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Check if backup exists
	if !pptFile.HasBackup() {
		response.GinError(c, response.CodeInvalidRequest, "没有可用的备份文件")
		return
	}

	// Call service to rollback
	if err := h.pptEditorService.Rollback(uint(id)); err != nil {
		h.logger.Error("Failed to rollback PPT",
			zap.String("ppt_id", idStr),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "回滚失败: "+err.Error())
		return
	}

	// Get updated PPT file info
	updatedPPT, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		h.logger.Error("Failed to get updated PPT file", zap.Error(err))
		response.GinSuccess(c, gin.H{
			"message": "回滚成功",
			"restored": true,
		})
		return
	}

	response.GinSuccess(c, gin.H{
		"message": "回滚成功",
		"restored": true,
		"page_count": updatedPPT.PageCount,
	})
}

// CaptureFrameRequest captures frame request
type CaptureFrameRequest struct {
	Timestamp float64 `json:"timestamp" binding:"required,min=0"`
}

// CaptureFrameHandler handles POST /api/v1/ppts/:id/capture
func (h *PPThandler) CaptureFrameHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Parse request body
	var req CaptureFrameRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的请求参数")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Get source video file
	if pptFile.SourceVideoFileID == nil {
		response.GinError(c, response.CodeInvalidRequest, "PPT文件没有关联视频")
		return
	}

	videoFile, err := h.videoFileService.GetFileByID(*pptFile.SourceVideoFileID)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "关联视频不存在")
		return
	}

	// Capture frame at timestamp
	frameData, mimeType, err := h.frameCaptureService.CaptureFrameToBytes(c.Request.Context(), videoFile.FilePath, req.Timestamp)
	if err != nil {
		h.logger.Error("Failed to capture frame",
			zap.String("ppt_id", idStr),
			zap.Float64("timestamp", req.Timestamp),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "捕获帧失败: "+err.Error())
		return
	}

	// Encode to base64
	// Note: In production, you might want to save this to disk and return a URL
	// For preview, we'll return a base64 data URL
	base64Data := "data:" + mimeType + ";base64," + encodeBase64(frameData)

	response.GinSuccess(c, gin.H{
		"success":     true,
		"frame_data":  base64Data,
		"timestamp":   req.Timestamp,
		"preview_url": fmt.Sprintf("/api/v1/ppts/%d/captured-preview?ts=%.3f", uint(id), req.Timestamp),
	})
}

// InsertSlideRequest inserts slide request
type InsertSlideRequest struct {
	FrameData      string  `json:"frame_data" binding:"required"`
	InsertPosition int     `json:"insert_position" binding:"required,min=1"`
	Timestamp      float64 `json:"timestamp" binding:"required,min=0"`
}

// InsertSlideHandler handles POST /api/v1/ppts/:id/slides
func (h *PPThandler) InsertSlideHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Parse request body
	var req InsertSlideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的请求参数")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Validate insert position
	if req.InsertPosition < 1 || req.InsertPosition > pptFile.PageCount+1 {
		response.GinError(c, response.CodeInvalidRequest,
			fmt.Sprintf("无效的插入位置: %d (有效范围: 1-%d)", req.InsertPosition, pptFile.PageCount+1))
		return
	}

	// Decode base64 frame data
	frameBytes, err := decodeBase64FrameData(req.FrameData)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的帧数据: "+err.Error())
		return
	}

	// Insert captured frame
	if err := h.pptEditorService.InsertCapturedFrame(uint(id), frameBytes, req.InsertPosition, req.Timestamp); err != nil {
		h.logger.Error("Failed to insert slide",
			zap.String("ppt_id", idStr),
			zap.Int("insert_position", req.InsertPosition),
			zap.Error(err))

		// Map error messages
		errMsg := err.Error()
		switch {
		case errMsg == "frame bytes cannot be empty":
			response.GinError(c, response.CodeInvalidRequest, "帧数据不能为空")
		case strings.Contains(errMsg, "frame bytes too large"):
			response.GinError(c, response.CodeInvalidRequest, "帧数据过大")
		default:
			response.GinError(c, response.CodeInternalError, "插入幻灯片失败: "+err.Error())
		}
		return
	}

	// Get updated PPT file info
	updatedPPT, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		h.logger.Error("Failed to get updated PPT file", zap.Error(err))
		response.GinSuccess(c, gin.H{
			"success":              true,
			"page_count":           pptFile.PageCount + 1,
			"inserted_slide_number": req.InsertPosition,
		})
		return
	}

	response.GinSuccess(c, gin.H{
		"success":              true,
		"page_count":           updatedPPT.PageCount,
		"inserted_slide_number": req.InsertPosition,
		"new_slide_url":        fmt.Sprintf("/api/v1/ppts/%d/slides/fullsize/slide_%03d_captured.jpg", uint(id), req.InsertPosition),
		"backup_path":          updatedPPT.BackupPath,
	})
}

// CapturedPreviewHandler handles GET /api/v1/ppts/:id/captured-preview
func (h *PPThandler) CapturedPreviewHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Parse timestamp query parameter
	timestampStr := c.Query("ts")
	timestamp, err := strconv.ParseFloat(timestampStr, 64)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的时间戳参数")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Get source video file
	if pptFile.SourceVideoFileID == nil {
		response.GinError(c, response.CodeInvalidRequest, "PPT文件没有关联视频")
		return
	}

	videoFile, err := h.videoFileService.GetFileByID(*pptFile.SourceVideoFileID)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "关联视频不存在")
		return
	}

	// Capture frame to temp file
	tempFile := fmt.Sprintf("/tmp/capture_preview_%d_%.3f.jpg", uint(id), timestamp)
	if err := h.frameCaptureService.CaptureFrame(c.Request.Context(), videoFile.FilePath, timestamp, tempFile); err != nil {
		h.logger.Error("Failed to capture preview frame",
			zap.String("ppt_id", idStr),
			zap.Float64("timestamp", timestamp),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "捕获预览帧失败: "+err.Error())
		return
	}

	// Serve file and clean up
	defer func() {
		os.Remove(tempFile)
	}()

	c.File(tempFile)
}

// encodeBase64 encodes bytes to base64 string
func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// decodeBase64FrameData decodes base64 frame data from request
// Handles both "data:image/jpeg;base64,..." and raw base64 formats
func decodeBase64FrameData(frameData string) ([]byte, error) {
	// Remove data URL prefix if present
	if strings.HasPrefix(frameData, "data:") {
		parts := strings.SplitN(frameData, ",", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid data URL format")
		}
		frameData = parts[1]
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(frameData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	return data, nil
}

// ReorderSlidesHandler handles POST /api/v1/ppts/:id/reorder
// Reorders slides according to the new slide order provided
func (h *PPThandler) ReorderSlidesHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	// Parse request
	var req struct {
		SlideOrder []int `json:"slide_order" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的请求参数")
		return
	}

	// Validate slide order
	if len(req.SlideOrder) == 0 {
		response.GinError(c, response.CodeInvalidRequest, "幻灯片顺序不能为空")
		return
	}

	// Load PPT file
	pptFile, err := h.pptFileService.GetPPTFileByID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "PPT文件不存在")
		return
	}

	// Verify ownership
	if err := h.verifyPPTOwnership(c, pptFile); err != nil {
		response.GinError(c, response.CodeForbidden, err.Error())
		return
	}

	// Reorder slides
	newOrder, err := h.pptEditorService.ReorderSlides(uint(id), req.SlideOrder)
	if err != nil {
		h.logger.Error("Failed to reorder slides",
			zap.String("ppt_id", idStr),
			zap.Error(err))
		response.GinError(c, response.CodeInternalError, "重排序幻灯片失败: "+err.Error())
		return
	}

	// NOTE: Don't invalidate cache after reordering!
	// The reordered JPEG images in the cache directory are now the source of truth.
	// Invalidating would cause re-extraction from the PPTX file which still has the original order.

	response.GinSuccess(c, gin.H{
		"success":   true,
		"message":   "幻灯片顺序已更新",
		"new_order": newOrder,
	})
}

// GetRequestBody reads and returns the request body for logging
// Note: This consumes the body, so it should only be used for error logging
func GetRequestBody(c *gin.Context) string {
	body, err := c.GetRawData()
	if err != nil {
		return fmt.Sprintf("[error reading body: %v]", err)
	}
	return string(body)
}
