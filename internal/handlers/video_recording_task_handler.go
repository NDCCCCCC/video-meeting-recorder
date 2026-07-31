package handlers

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/auth/hlstoken"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/scheduler"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// VideoRecordingTaskHandler 视频录制任务处理器
type VideoRecordingTaskHandler struct {
	taskService       *services.VideoRecordingTaskService
	conversionService scheduler.ConversionService
	auditService      *audit.AuditLogService
	logger            *zap.Logger
	config            *config.Config
	hlsToken          *hlstoken.HLSToken
}

// NewVideoRecordingTaskHandler 创建视频录制任务处理器
func NewVideoRecordingTaskHandler(taskService *services.VideoRecordingTaskService, auditService *audit.AuditLogService, logger *zap.Logger, cfg *config.Config) *VideoRecordingTaskHandler {
	return &VideoRecordingTaskHandler{
		taskService:  taskService,
		auditService: auditService,
		logger:       logger,
		config:       cfg,
		hlsToken:     hlstoken.NewHLSToken(cfg.Auth.HLSTokenSecret, cfg.Auth.HLSTokenDuration),
	}
}

// SetConversionService 设置转换服务
func (h *VideoRecordingTaskHandler) SetConversionService(conversionService scheduler.ConversionService) {
	h.conversionService = conversionService
}

// StartHLS 启动 HLS Token 的后台 sweeper（周期性驱逐过期 jti 索引项）。
// SEC-004 (Phase 19)：在 app.Start 中调用，StopHLS 在 app.Stop 中调用，
// 确保优雅关停不泄漏 sweeper goroutine。
func (h *VideoRecordingTaskHandler) StartHLS(ctx context.Context) {
	if h.hlsToken != nil {
		h.hlsToken.Start(ctx)
	}
}

// StopHLS 停止 HLS Token 的后台 sweeper 并等待 goroutine 退出。
func (h *VideoRecordingTaskHandler) StopHLS() {
	if h.hlsToken != nil {
		h.hlsToken.Stop()
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

	// 设置数据范围过滤参数
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	req.UserID = userID
	req.IsAdmin = middleware.GetIsAdmin(c)
	req.ApplyDataScope = true
	req.RoleIDs = middleware.GetRoleIDs(c)

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

	// 检查用户是否有权限访问此任务
	if !middleware.CanAccessAllData(c) {
		userID, ok := middleware.GetUserID(c)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
			return
		}
		// 非管理员和共享查看者只能访问自己创建的任务
		if task.CreatedBy != userID {
			h.logger.Warn("用户尝试访问无权限的录制任务",
				zap.Uint("user_id", userID),
				zap.Uint("task_id", id),
				zap.Uint("task_created_by", task.CreatedBy))
			response.GinError(c, response.CodeForbidden, "无权访问此任务")
			return
		}
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	task, err := h.taskService.CreateTask(&req, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已创建", zap.Uint("task_id", task.ID), zap.String("name", task.Name))
	response.GinSuccess(c, task)
}

// CreateTaskAuto 自动创建录制任务（固定华为配置ID为1）
// @Summary 自动创建录制任务
// @Description 创建新的视频录制任务，自动使用默认华为配置
// @Tags 录制任务
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.CreateTaskAutoRequest true "创建任务请求"
// @Success 200 {object} response.Response{data=models.VideoRecordingTask}
// @Router /api/v1/recordings/auto [post]
func (h *VideoRecordingTaskHandler) CreateTaskAuto(c *gin.Context) {
	var req services.CreateTaskAutoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	task, err := h.taskService.CreateTaskAuto(&req, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("录制任务已自动创建", zap.Uint("task_id", task.ID), zap.String("name", task.Name))
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	hasSharedViewer := middleware.GetHasSharedViewer(c)
	oldTask, task, err := h.taskService.UpdateTask(id, &req, userID, hasSharedViewer)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	resourceID := task.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionUpdate,
		Module:     models.ModuleTask,
		Resource:   fmt.Sprintf("task:%d", task.ID),
		ResourceID: &resourceID,
		OldData:    oldTask,
		NewData:    task,
	}); err != nil {
		h.logger.Warn("Failed to record task update change", zap.Error(err), zap.Uint("task_id", id))
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)
	oldTask, err := h.taskService.DeleteTask(id, userID, isAdmin)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	resourceID := oldTask.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionDelete,
		Module:     models.ModuleTask,
		Resource:   fmt.Sprintf("task:%d", oldTask.ID),
		ResourceID: &resourceID,
		OldData:    oldTask,
		NewData:    nil,
	}); err != nil {
		h.logger.Warn("Failed to record task delete change", zap.Error(err), zap.Uint("task_id", id))
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)
	oldTasks, result, err := h.taskService.BatchDeleteTasks(req.IDs, userID, isAdmin)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	// Emit single RecordChange for batch operation (option b per plan): OldData is the
	// pre-delete slice captured from initial id IN ? query, NewData is nil.
	if len(oldTasks) > 0 {
		idStrs := make([]string, len(oldTasks))
		for i, t := range oldTasks {
			idStrs[i] = strconv.FormatUint(uint64(t.ID), 10)
		}
		resource := "task:" + strings.Join(idStrs, ",")
		resourceID := uint(0)
		if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
			Action:     "batch_delete",
			Module:     models.ModuleTask,
			Resource:   resource,
			ResourceID: &resourceID,
			OldData:    oldTasks,
			NewData:    nil,
		}); err != nil {
			h.logger.Warn("Failed to record batch task delete change", zap.Error(err))
		}
	}

	// 根据结果构造响应消息
	var message string
	if result.TotalFailed > 0 {
		message = fmt.Sprintf("成功删除 %d 个任务，%d 个任务无法删除", result.TotalDeleted, result.TotalFailed)
	} else {
		message = fmt.Sprintf("成功删除 %d 个任务", result.TotalDeleted)
	}

	h.logger.Info("批量删除录制任务成功",
		zap.Int("deleted", result.TotalDeleted),
		zap.Int("failed", result.TotalFailed),
		zap.Uint("deleted_by", userID),
	)

	response.GinSuccess(c, gin.H{
		"message":       message,
		"deleted_ids":   result.DeletedIDs,
		"failed_ids":    result.FailedIDs,
		"failed_tasks":  result.FailedTasks,
		"total_deleted": result.TotalDeleted,
		"total_failed":  result.TotalFailed,
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	oldTask, task, err := h.taskService.StartTask(id, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	resourceID := task.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     "start",
		Module:     models.ModuleTask,
		Resource:   fmt.Sprintf("task:%d", task.ID),
		ResourceID: &resourceID,
		OldData:    oldTask,
		NewData:    task,
	}); err != nil {
		h.logger.Warn("Failed to record task start change", zap.Error(err), zap.Uint("task_id", id))
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	hasSharedViewer := middleware.GetHasSharedViewer(c)
	oldTask, task, err := h.taskService.StopTask(id, userID, hasSharedViewer)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	resourceID := task.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     "stop",
		Module:     models.ModuleTask,
		Resource:   fmt.Sprintf("task:%d", task.ID),
		ResourceID: &resourceID,
		OldData:    oldTask,
		NewData:    task,
	}); err != nil {
		h.logger.Warn("Failed to record task stop change", zap.Error(err), zap.Uint("task_id", id))
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	hasSharedViewer := middleware.GetHasSharedViewer(c)
	if err := h.taskService.CancelTask(id, userID, hasSharedViewer); err != nil {
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

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	hasSharedViewer := middleware.GetHasSharedViewer(c)
	task, err := h.taskService.RetryTask(id, userID, hasSharedViewer)
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

	status, err := h.conversionService.GetConversionStatus(c.Request.Context(), id)
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

	if err := h.conversionService.RetryConversion(c.Request.Context(), id); err != nil {
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

	// 验证权限：任务创建者、管理员或 shared_viewer 可以访问
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})
		return
	}
	isAdmin := middleware.GetIsAdmin(c)
	roleIDs := middleware.GetRoleIDs(c)

	hasSharedViewer := false
	for _, roleID := range roleIDs {
		if roleID == 5 { // RoleSharedViewer ID
			hasSharedViewer = true
			break
		}
	}

	if !isAdmin && !hasSharedViewer && task.CreatedBy != userID {
		response.GinError(c, response.CodeForbidden, "无权限访问此预览")
		return
	}

	// 根据状态和文件存在性返回不同响应
	// 返回完整的 m3u8 播放列表 URL（包含 token）
	playbackURL := fmt.Sprintf("/api/v1/recordings/%d/preview/stream/index.m3u8", id)

	// 生成访问 token
	accessToken := h.hlsToken.Generate(id, userID)
	playbackURLWithToken := fmt.Sprintf("%s?token=%s", playbackURL, accessToken)

	switch {
	case task.Status == "recording" && !m3u8Exists:
		response.GinSuccess(c, gin.H{
			"task_id":      id,
			"playback_url": playbackURLWithToken,
			"status":       task.Status,
			"ready":        false,
			"message":      "HLS预览正在准备中，请稍后刷新",
		})
	case !m3u8Exists:
		response.GinError(c, response.CodeNotFound, "HLS预览文件不存在")
	default:
		response.GinSuccess(c, gin.H{
			"task_id":      id,
			"playback_url": playbackURLWithToken,
			"status":       task.Status,
			"ready":        true,
		})
	}
}

// ServeHLSStream 提供HLS流文件服务
// @Summary 提供HLS流文件
// @Description 返回HLS m3u8播放列表或TS分片文件（需要有效的访问 token）
// @Tags 录制任务
// @Param id path int true "任务ID"
// @Param file path string true "文件名 (index.m3u8 或 segment_xxx.ts)"
// @Param token query string true "访问 token"
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

	// 获取并验证访问 token
	token := c.Query("token")
	if token == "" {
		response.GinError(c, response.CodeUnauthorized, "缺少访问 token")
		return
	}

	// 验证 token
	claims, err := h.hlsToken.Verify(token)
	if err != nil {
		h.logger.Warn("HLS流访问 token 验证失败",
			zap.Uint("task_id", id),
			zap.String("error", err.Error()),
		)
		response.GinError(c, response.CodeUnauthorized, "无效或已过期的访问 token")
		return
	}

	// 验证 token 中的任务 ID 是否匹配
	if claims.TaskID != id {
		response.GinError(c, response.CodeForbidden, "Token 与请求的任务不匹配")
		return
	}

	// 获取任务信息
	task, err := h.taskService.GetTaskByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "任务不存在")
		return
	}

	// 验证用户权限：管理员、shared_viewer 或任务创建者可以访问
	// 从当前上下文获取用户角色信息（而非仅依赖 token）
	isAdmin := middleware.GetIsAdmin(c)
	roleIDs := middleware.GetRoleIDs(c)

	hasSharedViewer := false
	for _, roleID := range roleIDs {
		if roleID == 5 { // RoleSharedViewer ID
			hasSharedViewer = true
			break
		}
	}

	// token 用户身份验证 + 数据访问权限检查
	if !isAdmin && !hasSharedViewer && claims.UserID != task.CreatedBy {
		h.logger.Warn("HLS流访问权限拒绝：用户不匹配",
			zap.Uint("task_id", id),
			zap.Uint("token_user_id", claims.UserID),
			zap.Uint("task_creator_id", task.CreatedBy),
		)
		response.GinError(c, response.CodeForbidden, "无权限访问此预览")
		return
	}

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

	// 如果是 m3u8 文件，需要重写内容，在分段 URL 中添加 token
	if strings.HasSuffix(filePath, ".m3u8") {
		content, err := os.ReadFile(fullPath)
		if err != nil {
			response.GinError(c, response.CodeInternalError, "读取文件失败")
			return
		}

		// 重写 m3u8 内容：在分段 URL 中添加 token 参数
		rewrittenContent := h.rewriteM3U8WithToken(string(content), token, id)
		c.String(200, rewrittenContent)
		return
	}

	// 返回文件内容
	c.File(fullPath)
}

// rewriteM3U8WithToken 重写 m3u8 播放列表，在分段 URL 中添加 token 参数
// PERF-005/D-03.8: 对 .ts/.m3u8 分段 URL 的 tokenize 改用 bounded concurrency，
// 限制在 cfg.FFmpeg.HLSRewriteConcurrency（默认 2）以避免突发请求耗尽 FFmpeg 子进程槽位。
func (h *VideoRecordingTaskHandler) rewriteM3U8WithToken(content string, token string, taskID uint) string {
	lines := strings.Split(content, "\n")
	tokenParam := fmt.Sprintf("?token=%s", token)
	_ = taskID

	result := make([]string, len(lines))
	copy(result, lines)

	// 收集需要添加 token 的行下标。
	indices := make([]int, 0, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") &&
			(strings.HasSuffix(trimmed, ".ts") || strings.HasSuffix(trimmed, ".m3u8")) {
			indices = append(indices, i)
		}
	}

	if len(indices) == 0 {
		return content
	}

	concurrency := 2
	if h.config != nil && h.config.FFmpeg.HLSRewriteConcurrency > 0 {
		concurrency = h.config.FFmpeg.HLSRewriteConcurrency
	}
	if concurrency > len(indices) {
		concurrency = len(indices)
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, idx := range indices {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			defer func() {
				if r := recover(); r != nil {
					h.logger.Error("m3u8 rewrite worker panicked",
						zap.Any("recover", r), zap.Stack("stack"))
				}
			}()
			result[i] = strings.TrimSpace(lines[i]) + tokenParam
		}(idx)
	}
	wg.Wait()

	return strings.Join(result, "\n")
}

// containsPathTraversal 检查文件名是否包含路径遍历字符（包括 URL 编码）
func containsPathTraversal(filename string) bool {
	// 先尝试 URL 解码
	decoded, err := url.PathUnescape(filename)
	if err == nil {
		// 检查解码后的路径
		filename = decoded
	}

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

// ClearStuckTasks 清理卡住的任务
// @Summary 清理卡住的任务
// @Description 将转换中状态超过指定时间的任务标记为失败，并释放终端锁
// @Tags 录制任务
// @Security Bearer
// @Param timeout_minutes query int false "超时时间（分钟），默认30分钟" default(30)
// @Success 200 {object} response.Response{data=services.ClearStuckTasksResult}
// @Router /api/v1/tasks/clear-stuck [post]
func (h *VideoRecordingTaskHandler) ClearStuckTasks(c *gin.Context) {
	// 获取超时参数
	timeoutMinutes := 30 // 默认30分钟
	if timeoutStr := c.Query("timeout_minutes"); timeoutStr != "" {
		if parsed, err := parseUintParamFromString(timeoutStr); err == nil && parsed > 0 {
			timeoutMinutes = int(parsed)
		}
	}

	result, err := h.taskService.ClearStuckTasks(timeoutMinutes)
	if err != nil {
		h.logger.Error("清理卡住任务失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "清理卡住任务失败")
		return
	}

	h.logger.Info("清理卡住任务成功",
		zap.Int("total_cleared", result.TotalCleared),
		zap.Int("total_unlocked", result.TotalUnlocked),
	)

	response.GinSuccess(c, result)
}

// parseUintParamFromString 从字符串解析uint参数
func parseUintParamFromString(s string) (uint, error) {
	var result uint
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
