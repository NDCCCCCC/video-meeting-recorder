package handlers

import (
	"errors"
	"net/http"
	"strconv"

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
	pptFileService    *services.PPTFileService
	slideCacheService *services.SlideCacheService
	mergeService      *services.PPTMergeService
	videoFileService  *services.VideoFileService
	logger            *zap.Logger
}

// NewPPThandler creates a new PPT handler
func NewPPThandler(
	pptFileService *services.PPTFileService,
	slideCacheService *services.SlideCacheService,
	mergeService *services.PPTMergeService,
	videoFileService *services.VideoFileService,
	logger *zap.Logger,
) *PPThandler {
	return &PPThandler{
		pptFileService:    pptFileService,
		slideCacheService: slideCacheService,
		mergeService:      mergeService,
		videoFileService:  videoFileService,
		logger:            logger,
	}
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
func (h *PPThandler) ServeSlideImage(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的PPT ID")
		return
	}

	resolution := c.Param("resolution")
	filename := c.Param("filename")

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
	var req models.MergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的请求参数")
		return
	}

	// Validate slides not empty
	if len(req.Slides) == 0 {
		response.GinError(c, response.CodeInvalidRequest, "请选择要合并的幻灯片")
		return
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

	// Verify ownership via SourceVideoFileID
	if pptFile.SourceVideoFileID == nil {
		response.GinError(c, response.CodeForbidden, "PPT文件没有关联视频，无法验证权限")
		return
	}

	videoFile, err := h.videoFileService.GetFileByID(*pptFile.SourceVideoFileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.GinError(c, response.CodeForbidden, "关联视频不存在")
		} else {
			response.GinError(c, response.CodeInternalError, "获取视频信息失败")
		}
		return
	}

	userID := middleware.GetUserID(c)
	if !middleware.GetIsAdmin(c) && videoFile.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权下载此PPT文件")
		return
	}

	// Serve file
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

	// Verify ownership via SourceVideoFileID
	if pptFile.SourceVideoFileID == nil {
		response.GinError(c, response.CodeForbidden, "PPT文件没有关联视频，无法验证权限")
		return
	}

	videoFile, err := h.videoFileService.GetFileByID(*pptFile.SourceVideoFileID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.GinError(c, response.CodeForbidden, "关联视频不存在")
		} else {
			response.GinError(c, response.CodeInternalError, "获取视频信息失败")
		}
		return
	}

	userID := middleware.GetUserID(c)
	if !middleware.GetIsAdmin(c) && videoFile.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权删除此PPT文件")
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
