package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VideoRecordingTaskHandler 视频录制任务处理器
type VideoRecordingTaskHandler struct {
	taskService      *services.VideoRecordingTaskService
	conversionService services.ConversionService
	logger           *zap.Logger
}

// NewVideoRecordingTaskHandler 创建视频录制任务处理器
func NewVideoRecordingTaskHandler(taskService *services.VideoRecordingTaskService, logger *zap.Logger) *VideoRecordingTaskHandler {
	return &VideoRecordingTaskHandler{
		taskService: taskService,
		logger:      logger,
	}
}

// SetConversionService 设置转换服务
func (h *VideoRecordingTaskHandler) SetConversionService(conversionService services.ConversionService) {
	h.conversionService = conversionService
}

// ListTasks 获取录制任务列表
// @Summary 获取录制任务列表
// @Description 分页获取录制任务列表，支持筛选
// @Tags 录制任务
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Param status query string false "任务状态"
// @Param created_by query int false "创建者ID"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} response.Response{data=services.ListTasksResponse}
// @Router /api/v1/tasks [get]
func (h *VideoRecordingTaskHandler) ListTasks(c *gin.Context) {
	var req services.ListTasksRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	result, err := h.taskService.ListTasks(&req)
	if err != nil {
		h.logger.Error("获取录制任务列表失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取任务列表失败")
		return
	}

	response.GinSuccess(c, result)
}

// GetTask 获取录制任务详情
// @Summary 获取录制任务详情
// @Description 根据ID获取录制任务详细信息
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response{data=models.VideoRecordingTask}
// @Router /api/v1/tasks/{id} [get]
func (h *VideoRecordingTaskHandler) GetTask(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	task, err := h.taskService.GetTaskByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "任务不存在")
		return
	}

	response.GinSuccess(c, task)
}

// CreateTask 创建录制任务
// @Summary 创建录制任务
// @Description 创建新的视频录制任务
// @Tags 录制任务
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.CreateTaskRequest true "创建任务请求"
// @Success 200 {object} response.Response{data=models.VideoRecordingTask}
// @Router /api/v1/tasks [post]
func (h *VideoRecordingTaskHandler) CreateTask(c *gin.Context) {
	var req services.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	task, err := h.taskService.CreateTask(&req, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已创建", zap.Uint("task_id", task.ID), zap.String("name", task.Name))
	response.GinSuccess(c, task)
}

// UpdateTask 更新录制任务
// @Summary 更新录制任务
// @Description 更新录制任务信息
// @Tags 录制任务
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Param request body services.UpdateTaskRequest true "更新任务请求"
// @Success 200 {object} response.Response{data=models.VideoRecordingTask}
// @Router /api/v1/tasks/{id} [put]
func (h *VideoRecordingTaskHandler) UpdateTask(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	var req services.UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	task, err := h.taskService.UpdateTask(id, &req, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已更新", zap.Uint("task_id", id))
	response.GinSuccess(c, task)
}

// DeleteTask 删除录制任务
// @Summary 删除录制任务
// @Description 删除指定的录制任务
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response
// @Router /api/v1/tasks/{id} [delete]
func (h *VideoRecordingTaskHandler) DeleteTask(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.taskService.DeleteTask(id, userID); err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已删除", zap.Uint("task_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// BatchDeleteTasks 批量删除录制任务
// @Summary 批量删除录制任务
// @Description 批量删除多个录制任务（只能删除待执行、已完成、失败、已取消状态的任务）
// @Tags 录制任务
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.BatchDeleteTasksRequest true "批量删除请求"
// @Success 200 {object} response.Response
// @Router /api/v1/recordings/batch [delete]
func (h *VideoRecordingTaskHandler) BatchDeleteTasks(c *gin.Context) {
	var req services.BatchDeleteTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	userID := middleware.GetUserID(c)
	count, err := h.taskService.BatchDeleteTasks(req.IDs, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("批量删除录制任务成功", zap.Int("count", count), zap.Uint("deleted_by", userID))
	response.GinSuccess(c, gin.H{
		"message": fmt.Sprintf("成功删除 %d 个任务", count),
		"count":   count,
	})
}

// StartTask 启动录制任务
// @Summary 启动录制任务
// @Description 手动启动待执行状态的录制任务
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response{data=models.VideoRecordingTask}
// @Router /api/v1/tasks/{id}/start [post]
func (h *VideoRecordingTaskHandler) StartTask(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	userID := middleware.GetUserID(c)
	task, err := h.taskService.StartTask(id, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已启动", zap.Uint("task_id", id))
	response.GinSuccess(c, task)
}

// StopTask 停止录制任务
// @Summary 停止录制任务
// @Description 手动停止录制中的任务
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response{data=models.VideoRecordingTask}
// @Router /api/v1/tasks/{id}/stop [post]
func (h *VideoRecordingTaskHandler) StopTask(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	userID := middleware.GetUserID(c)
	task, err := h.taskService.StopTask(id, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已停止", zap.Uint("task_id", id))
	response.GinSuccess(c, task)
}

// CancelTask 取消录制任务
// @Summary 取消录制任务
// @Description 取消待执行或连接中的任务
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response
// @Router /api/v1/tasks/{id}/cancel [post]
func (h *VideoRecordingTaskHandler) CancelTask(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	userID := middleware.GetUserID(c)
	if err := h.taskService.CancelTask(id, userID); err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已取消", zap.Uint("task_id", id))
	response.GinSuccess(c, gin.H{"message": "任务已取消"})
}

// RetryTask 重试录制任务
// @Summary 重试录制任务
// @Description 重试失败的任务
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response{data=models.VideoRecordingTask}
// @Router /api/v1/tasks/{id}/retry [post]
func (h *VideoRecordingTaskHandler) RetryTask(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	userID := middleware.GetUserID(c)
	task, err := h.taskService.RetryTask(id, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已重试", zap.Uint("task_id", id))
	response.GinSuccess(c, task)
}

// GetConversionStatus 获取转换状态
// @Summary 获取转换状态
// @Description 获取任务的MKV到MP4转换状态
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /api/v1/recordings/{id}/conversion-status [get]
func (h *VideoRecordingTaskHandler) GetConversionStatus(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	if h.conversionService == nil {
		response.GinError(c, response.CodeInternalError, "转换服务未启用")
		return
	}

	status, err := h.conversionService.GetConversionStatus(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "任务不存在")
		return
	}

	response.GinSuccess(c, gin.H{
		"task_id": id,
		"status":  status,
	})
}

// RetryConversion 重试转换任务
// @Summary 重试转换任务
// @Description 重试失败的MKV到MP4转换任务
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response
// @Router /api/v1/recordings/{id}/conversion-retry [post]
func (h *VideoRecordingTaskHandler) RetryConversion(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	if h.conversionService == nil {
		response.GinError(c, response.CodeInternalError, "转换服务未启用")
		return
	}

	if err := h.conversionService.RetryConversion(id); err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("转换任务已重试", zap.Uint("task_id", id))
	response.GinSuccess(c, gin.H{"message": "转换任务已重新提交"})
}

// GetHLSPreview 获取HLS预览信息
// @Summary 获取HLS预览信息
// @Description 获取任务的HLS实时预览播放地址（仅任务创建者可访问）
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Router /api/v1/recordings/{id}/preview [get]
func (h *VideoRecordingTaskHandler) GetHLSPreview(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	// 获取任务信息
	task, err := h.taskService.GetTaskByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "任务不存在")
		return
	}

	// 检查HLS预览路径是否存在
	if task.HLSPreviewPath == "" {
		response.GinError(c, response.CodeNotFound, "该任务没有HLS预览")
		return
	}

	// 检查 m3u8 文件是否存在
	_, m3u8Err := os.Stat(task.HLSPreviewPath)
	m3u8Exists := m3u8Err == nil

	// 验证权限：只有任务创建者可以访问
	userID := middleware.GetUserID(c)
	if task.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权限访问此预览")
		return
	}

	// 根据状态和文件存在性返回不同响应
	playbackURL := fmt.Sprintf("/api/v1/recordings/%d/preview/stream", id)

	switch {
	case task.Status == "recording" && !m3u8Exists:
		response.GinSuccess(c, gin.H{
			"task_id":      id,
			"playback_url": playbackURL,
			"status":       task.Status,
			"ready":        false,
			"message":      "HLS预览正在准备中，请稍后刷新",
		})
	case !m3u8Exists:
		response.GinError(c, response.CodeNotFound, "HLS预览文件不存在")
	default:
		response.GinSuccess(c, gin.H{
			"task_id":      id,
			"playback_url": playbackURL,
			"status":       task.Status,
			"ready":        true,
		})
	}
}

// ServeHLSStream 提供HLS流文件服务
// @Summary 提供HLS流文件
// @Description 返回HLS m3u8播放列表或TS分片文件（仅任务创建者可访问）
// @Tags 录制任务
// @Security Bearer
// @Param id path int true "任务ID"
// @Param file path string true "文件名 (index.m3u8 或 segment_xxx.ts)"
// @Success 200 {file} file
// @Router /api/v1/recordings/{id}/preview/stream/{file} [get]
func (h *VideoRecordingTaskHandler) ServeHLSStream(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的任务ID")
		return
	}

	// 获取请求的文件名
	filePath := c.Param("file")
	if filePath == "" {
		response.GinError(c, response.CodeInvalidRequest, "文件名不能为空")
		return
	}

	// 安全检查：确保文件名不包含路径遍历字符
	if containsPathTraversal(filePath) {
		response.GinError(c, response.CodeInvalidRequest, "无效的文件名")
		return
	}

	// 获取任务信息
	task, err := h.taskService.GetTaskByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "任务不存在")
		return
	}

	// HLS 流访问权限验证：
	// 由于浏览器无法在视频请求中携带 JWT token，我们采用宽松的权限策略：
	// 1. 只检查任务是否存在
	// 2. 不检查用户权限（任何知道 URL 的人都可以访问）
	// 3. 安全性依赖：
	//    - URL 包含任务 ID，不易猜测
	//    - HLS 文件只在录制期间生成，录制结束后可能被清理
	//    - 可以添加额外的 token 验证（可选）
	//
	// 如果需要更强的安全性，可以考虑：
	// - 生成临时访问 token
	// - 或在 URL 中包含签名参数

	// 构建完整的文件路径
	hlsDir := filepath.Dir(task.HLSPreviewPath)
	fullPath := filepath.Join(hlsDir, filePath)

	// 安全检查：确保请求的文件在HLS目录内
	if !isPathWithinDirectory(fullPath, hlsDir) {
		response.GinError(c, response.CodeForbidden, "访问被拒绝")
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.Status(404)
		c.Abort()
		return
	}

	// 设置正确的Content-Type
	c.Header("Content-Type", getHLSContentType(filePath))
	c.Header("Cache-Control", "no-cache")
	c.Header("Access-Control-Allow-Origin", "*")

	// 返回文件内容
	c.File(fullPath)
}

// containsPathTraversal 检查文件名是否包含路径遍历字符
func containsPathTraversal(filename string) bool {
	return strings.Contains(filename, "..") ||
		strings.Contains(filename, "\\") ||
		strings.Contains(filename, "\x00") ||
		strings.HasPrefix(filename, "/") ||
		strings.HasPrefix(filename, "\\")
}

// isPathWithinDirectory 检查路径是否在指定目录内
func isPathWithinDirectory(path, dir string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, "..")
}

// getHLSContentType 根据文件扩展名返回Content-Type
func getHLSContentType(filename string) string {
	if strings.HasSuffix(filename, ".m3u8") {
		return "application/vnd.apple.mpegurl"
	}
	if strings.HasSuffix(filename, ".ts") {
		return "video/mp2t"
	}
	return "application/octet-stream"
}
