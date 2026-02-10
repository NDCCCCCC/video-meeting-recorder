package handlers

import (
	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VideoRecordingTaskHandler 视频录制任务处理器
type VideoRecordingTaskHandler struct {
	taskService *services.VideoRecordingTaskService
	logger      *zap.Logger
}

// NewVideoRecordingTaskHandler 创建视频录制任务处理器
func NewVideoRecordingTaskHandler(taskService *services.VideoRecordingTaskService, logger *zap.Logger) *VideoRecordingTaskHandler {
	return &VideoRecordingTaskHandler{
		taskService: taskService,
		logger:      logger,
	}
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
		h.logger.Error("Failed to list video recording tasks", zap.Error(err))
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

	h.logger.Info("Video recording task created", zap.Uint("task_id", task.ID), zap.String("name", task.Name))
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

	h.logger.Info("Video recording task updated", zap.Uint("task_id", id))
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

	h.logger.Info("Video recording task deleted", zap.Uint("task_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
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

	h.logger.Info("Video recording task started", zap.Uint("task_id", id))
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

	h.logger.Info("Video recording task stopped", zap.Uint("task_id", id))
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

	h.logger.Info("Video recording task cancelled", zap.Uint("task_id", id))
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

	h.logger.Info("Video recording task retried", zap.Uint("task_id", id))
	response.GinSuccess(c, task)
}
