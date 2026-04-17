package handlers

import (
	"strconv"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type SplitHandler struct {
	splittingService *services.SplittingService
	snapshotService  *services.SnapshotService
	videoFileService *services.VideoFileService
	logger           *zap.Logger
}

func NewSplitHandler(
	splittingService *services.SplittingService,
	snapshotService *services.SnapshotService,
	videoFileService *services.VideoFileService,
	logger *zap.Logger,
) *SplitHandler {
	return &SplitHandler{
		splittingService: splittingService,
		snapshotService:  snapshotService,
		videoFileService: videoFileService,
		logger:           logger,
	}
}

// SubmitSplit handles POST /api/v1/videos/:id/split
// Body: {"markers": [10.5, 30.0, 50.0], "re_encode": false}
func (h *SplitHandler) SubmitSplit(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	var req struct {
		Markers  []float64 `json:"markers" binding:"required"`
		ReEncode bool      `json:"re_encode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// Validate markers
	if len(req.Markers) == 0 {
		response.GinError(c, response.CodeInvalidRequest, "至少需要一个分割标记点")
		return
	}
	if len(req.Markers) > 20 {
		response.GinError(c, response.CodeInvalidRequest, "最多支持20个分割标记点")
		return
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

	if err := h.splittingService.SubmitSplit(uint(id), req.Markers, req.ReEncode, userID); err != nil {
		response.GinError(c, response.CodeInternalError, "提交分割任务失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"video_file_id": id,
		"status":        "processing",
		"segment_count": len(req.Markers) + 1,
	})
}

// GenerateSnapshot handles POST /api/v1/tasks/:id/snapshot
func (h *SplitHandler) GenerateSnapshot(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	userID := middleware.GetUserID(c)

	snapshotFile, err := h.snapshotService.GenerateSnapshot(uint(id), userID)
	if err != nil {
		response.GinError(c, response.CodeInternalError, "生成快照失败: "+err.Error())
		return
	}

	response.GinSuccess(c, gin.H{
		"snapshot_file_id": snapshotFile.ID,
		"file_name":        snapshotFile.FileName,
		"file_size":        snapshotFile.FileSize,
		"duration":         snapshotFile.Duration,
	})
}

// GetSplitStatus handles GET /api/v1/videos/:id/split-status
func (h *SplitHandler) GetSplitStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	status := h.splittingService.GetSplitStatus(uint(id))

	// If completed or empty, also return segment list
	result := gin.H{"status": status}
	if status == "completed" || status == "" {
		segments, _ := h.videoFileService.GetSegmentsByParentID(uint(id))
		result["segments"] = segments
	}

	response.GinSuccess(c, result)
}

// GetSegments handles GET /api/v1/videos/:id/segments
func (h *SplitHandler) GetSegments(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的视频ID")
		return
	}

	segments, err := h.videoFileService.GetSegmentsByParentID(uint(id))
	if err != nil {
		response.GinError(c, response.CodeInternalError, "获取分割段落失败")
		return
	}

	response.GinSuccess(c, segments)
}
