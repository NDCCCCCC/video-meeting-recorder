package handlers

import (
	"strconv"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/notification"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NotificationHandler 通知处理器
type NotificationHandler struct {
	notificationService *notification.NotificationService
	logger              *zap.Logger
}

// NewNotificationHandler 创建通知处理器
func NewNotificationHandler(notificationService *notification.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// SetLogger 设置日志记录器
func (h *NotificationHandler) SetLogger(logger *zap.Logger) {
	h.logger = logger
}

// Stop 停止处理器
func (h *NotificationHandler) Stop() {
	if h.notificationService != nil {
		h.notificationService.Stop()
	}
}

// getUserID 从上下文获取用户ID
func (h *NotificationHandler) getUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// ListNotifications 获取通知列表
// @Summary 获取通知列表
// @Tags 通知
// @Produce json
// @Param type query string false "通知类型"
// @Param is_read query bool false "是否已读"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "页大小" default(20)
// @Success 200 {object} response.Response
// @Router /api/v1/notifications [get]
func (h *NotificationHandler) ListNotifications(c *gin.Context) {
	var req notification.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	result, err := h.notificationService.Query(c.Request.Context(), h.getUserID(c), &req)
	if err != nil {
		h.logger.Warn("查询通知列表失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "查询失败")
		return
	}

	response.GinSuccess(c, result)
}

// MarkAsRead 标记为已读
// @Summary 标记通知为已读
// @Tags 通知
// @Produce json
// @Param id path int true "通知ID"
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/:id/read [put]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的ID")
		return
	}

	userID := h.getUserID(c)
	err = h.notificationService.MarkAsRead(c.Request.Context(), uint(id), userID)
	if err != nil {
		h.logger.Warn("标记已读失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "标记失败")
		return
	}

	response.GinSuccess(c, gin.H{"message": "标记成功"})
}

// MarkAllAsRead 全部标记为已读
// @Summary 全部标记为已读
// @Tags 通知
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/read-all [put]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := h.getUserID(c)
	err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		h.logger.Warn("全部标记已读失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "标记失败")
		return
	}

	response.GinSuccess(c, gin.H{"message": "标记成功"})
}

// GetUnreadCount 获取未读数量
// @Summary 获取未读消息数量
// @Tags 通知
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := h.getUserID(c)
	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.logger.Warn("获取未读数量失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取失败")
		return
	}

	response.GinSuccess(c, gin.H{"count": count})
}

// GetUserSetting 获取用户通知配置
// @Summary 获取用户通知配置
// @Tags 通知
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/settings [get]
func (h *NotificationHandler) GetUserSetting(c *gin.Context) {
	userID := h.getUserID(c)
	setting, err := h.notificationService.GetUserSetting(c.Request.Context(), userID)
	if err != nil {
		h.logger.Warn("获取通知配置失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取失败")
		return
	}

	response.GinSuccess(c, setting)
}

// UpdateUserSetting 更新用户通知配置
// @Summary 更新用户通知配置
// @Tags 通知
// @Produce json
// @Param request body models.UserNotificationSetting true "更新配置请求"
// @Success 200 {object} response.Response
// @Router /api/v1/notifications/settings [put]
func (h *NotificationHandler) UpdateUserSetting(c *gin.Context) {
	var req models.UserNotificationSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	userID := h.getUserID(c)
	err := h.notificationService.UpdateUserSetting(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Warn("更新通知配置失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "更新失败")
		return
	}

	response.GinSuccess(c, gin.H{"message": "更新成功"})
}
