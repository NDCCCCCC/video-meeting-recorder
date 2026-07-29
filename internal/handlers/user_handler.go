package handlers

import (
	"fmt"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/middleware"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService  *services.UserService
	auditService *audit.AuditLogService
	logger       *zap.Logger
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService *services.UserService, auditService *audit.AuditLogService, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		userService:  userService,
		auditService: auditService,
		logger:       logger,
	}
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 分页获取用户列表，支持关键词搜索和筛选
// @Tags 用户管理
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Param role_id query int false "角色ID"
// @Param is_active query bool false "是否激活"
// @Success 200 {object} response.Response{data=services.ListUsersResponse}
// @Router /api/v1/users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req services.ListUsersRequest
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

	result, err := h.userService.ListUsers(&req)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取用户列表失败")
		return
	}

	response.GinSuccess(c, result)
}

// GetUser 获取用户详情
// @Summary 获取用户详情
// @Description 根据ID获取用户详细信息
// @Tags 用户管理
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=models.User}
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的用户ID")
		return
	}

	user, err := h.userService.GetUserByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "用户不存在")
		return
	}

	response.GinSuccess(c, user)
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户
// @Tags 用户管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.CreateUserRequest true "创建用户请求"
// @Success 200 {object} response.Response{data=models.User}
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req services.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	user, err := h.userService.CreateUser(&req)
	if err != nil {
		response.GinError(c, response.CodeDuplicateRecord, err.Error())
		return
	}

	h.logger.Info("User created", zap.Uint("user_id", user.ID), zap.String("username", user.Username))
	response.GinSuccess(c, user)
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新用户信息
// @Tags 用户管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param request body services.UpdateUserRequest true "更新用户请求"
// @Success 200 {object} response.Response{data=models.User}
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的用户ID")
		return
	}

	var req services.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取当前用户ID用于审计日志
	currentUserID := middleware.GetUserID(c)

	oldUser, user, err := h.userService.UpdateUser(id, &req, currentUserID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	resourceID := user.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionUpdate,
		Module:     models.ModuleUser,
		Resource:   fmt.Sprintf("user:%d", user.ID),
		ResourceID: &resourceID,
		OldData:    oldUser,
		NewData:    user,
	}); err != nil {
		h.logger.Warn("Failed to record user update change", zap.Error(err), zap.Uint("user_id", id))
	}

	h.logger.Info("User updated", zap.Uint("user_id", id))
	response.GinSuccess(c, user)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除指定用户
// @Tags 用户管理
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的用户ID")
		return
	}

	oldUser, _, err := h.userService.DeleteUser(id)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	resourceID := oldUser.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionDelete,
		Module:     models.ModuleUser,
		Resource:   fmt.Sprintf("user:%d", oldUser.ID),
		ResourceID: &resourceID,
		OldData:    oldUser,
		NewData:    nil,
	}); err != nil {
		h.logger.Warn("Failed to record user delete change", zap.Error(err), zap.Uint("user_id", id))
	}

	h.logger.Info("User deleted", zap.Uint("user_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// ResetPassword 重置用户密码
// @Summary 重置用户密码
// @Description 重置指定用户的密码
// @Tags 用户管理
// @Security Bearer
// @Accept json
// @Param id path int true "用户ID"
// @Param request body object{password=string} true "新密码"
// @Success 200 {object} response.Response
// @Router /api/v1/users/{id}/reset-password [post]
func (h *UserHandler) ResetPassword(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的用户ID")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	oldSnapshot, newSnapshot, err := h.userService.ResetPassword(id, req.Password)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	resourceID := id
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     "reset_password",
		Module:     models.ModuleUser,
		Resource:   fmt.Sprintf("user:%d", id),
		ResourceID: &resourceID,
		OldData:    oldSnapshot,
		NewData:    newSnapshot,
	}); err != nil {
		h.logger.Warn("Failed to record password reset change", zap.Error(err), zap.Uint("user_id", id))
	}

	h.logger.Info("User password reset", zap.Uint("user_id", id))
	response.GinSuccess(c, gin.H{"message": "密码重置成功"})
}

// ToggleUserStatus 切换用户状态
// @Summary 切换用户状态
// @Description 启用/禁用用户
// @Tags 用户管理
// @Security Bearer
// @Param id path int true "用户ID"
// @Success 200 {object} response.Response{data=models.User}
// @Router /api/v1/users/{id}/toggle-status [post]
func (h *UserHandler) ToggleUserStatus(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的用户ID")
		return
	}

	user, err := h.userService.ToggleUserStatus(id)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("User status toggled", zap.Uint("user_id", id), zap.Bool("is_active", user.IsActive))
	response.GinSuccess(c, user)
}

// GetCurrentProfile 获取当前用户资料
func (h *UserHandler) GetCurrentProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "用户不存在")
		return
	}

	response.GinSuccess(c, user)
}

// UpdateCurrentProfile 更新当前用户资料
func (h *UserHandler) UpdateCurrentProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req struct {
		Email    string `json:"email" binding:"omitempty,email"`
		FullName string `json:"full_name" binding:"omitempty,max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 构造更新请求
	updateReq := &services.UpdateUserRequest{
		Email:    req.Email,
		FullName: req.FullName,
	}

	_, user, err := h.userService.UpdateUser(userID, updateReq, userID)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("User profile updated", zap.Uint("user_id", userID))
	response.GinSuccess(c, user)
}
