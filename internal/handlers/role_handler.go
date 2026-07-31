package handlers

import (
	"fmt"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RoleHandler 角色处理器
type RoleHandler struct {
	roleService  *services.RoleService
	auditService *audit.AuditLogService
	logger       *zap.Logger
}

// NewRoleHandler 创建角色处理器
func NewRoleHandler(roleService *services.RoleService, auditService *audit.AuditLogService, logger *zap.Logger) *RoleHandler {
	return &RoleHandler{
		roleService:  roleService,
		auditService: auditService,
		logger:       logger,
	}
}

// ListRoles 获取角色列表
// @Summary 获取角色列表
// @Description 分页获取角色列表，支持关键词搜索
// @Tags 角色管理
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} response.Response{data=services.ListRolesResponse}
// @Router /api/v1/roles [get]
func (h *RoleHandler) ListRoles(c *gin.Context) {
	var req services.ListRolesRequest
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

	result, err := h.roleService.ListRoles(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to list roles", zap.Error(err))
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, result)
}

// GetRole 获取角色详情
// @Summary 获取角色详情
// @Description 根据ID获取角色详细信息
// @Tags 角色管理
// @Security Bearer
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response{data=models.Role}
// @Router /api/v1/roles/{id} [get]
func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的角色ID")
		return
	}

	role, err := h.roleService.GetRoleByID(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, role)
}

// CreateRole 创建角色
// @Summary 创建角色
// @Description 创建新角色
// @Tags 角色管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.CreateRoleRequest true "创建角色请求"
// @Success 200 {object} response.Response{data=models.Role}
// @Router /api/v1/roles [post]
func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req services.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	role, err := h.roleService.CreateRole(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	h.logger.Info("Role created", zap.Uint("role_id", role.ID), zap.String("name", role.Name))
	response.GinSuccess(c, role)
}

// UpdateRole 更新角色
// @Summary 更新角色
// @Description 更新角色信息
// @Tags 角色管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "角色ID"
// @Param request body services.UpdateRoleRequest true "更新角色请求"
// @Success 200 {object} response.Response{data=models.Role}
// @Router /api/v1/roles/{id} [put]
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的角色ID")
		return
	}

	var req services.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	oldRole, role, err := h.roleService.UpdateRole(c.Request.Context(), id, &req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resourceID := role.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionUpdate,
		Module:     models.ModuleRole,
		Resource:   fmt.Sprintf("role:%d", role.ID),
		ResourceID: &resourceID,
		OldData:    oldRole,
		NewData:    role,
	}); err != nil {
		h.logger.Warn("Failed to record role update change", zap.Error(err), zap.Uint("role_id", id))
	}

	h.logger.Info("Role updated", zap.Uint("role_id", id))
	response.GinSuccess(c, role)
}

// DeleteRole 删除角色
// @Summary 删除角色
// @Description 删除指定角色
// @Tags 角色管理
// @Security Bearer
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response
// @Router /api/v1/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的角色ID")
		return
	}

	oldRole, _, err := h.roleService.DeleteRole(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resourceID := oldRole.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionDelete,
		Module:     models.ModuleRole,
		Resource:   fmt.Sprintf("role:%d", oldRole.ID),
		ResourceID: &resourceID,
		OldData:    oldRole,
		NewData:    nil,
	}); err != nil {
		h.logger.Warn("Failed to record role delete change", zap.Error(err), zap.Uint("role_id", id))
	}

	h.logger.Info("Role deleted", zap.Uint("role_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// GetRolePermissions 获取角色权限
// @Summary 获取角色权限
// @Description 获取指定角色的权限列表
// @Tags 角色管理
// @Security Bearer
// @Param id path int true "角色ID"
// @Success 200 {object} response.Response{data=[]models.Permission}
// @Router /api/v1/roles/{id}/permissions [get]
func (h *RoleHandler) GetRolePermissions(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的角色ID")
		return
	}

	permissions, err := h.roleService.GetRolePermissions(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, permissions)
}

// AssignPermissions 分配权限
// @Summary 分配权限
// @Description 为角色分配权限
// @Tags 角色管理
// @Security Bearer
// @Accept json
// @Param id path int true "角色ID"
// @Param request body services.AssignPermissionsRequest true "分配权限请求"
// @Success 200 {object} response.Response
// @Router /api/v1/roles/{id}/permissions [post]
func (h *RoleHandler) AssignPermissions(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的角色ID")
		return
	}

	var req services.AssignPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	oldPermissions, newPermissions, err := h.roleService.AssignPermissions(c.Request.Context(), id, &req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	resourceID := id
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     "assign_permissions",
		Module:     models.ModuleRole,
		Resource:   fmt.Sprintf("role:%d", id),
		ResourceID: &resourceID,
		OldData:    oldPermissions,
		NewData:    newPermissions,
	}); err != nil {
		h.logger.Warn("Failed to record permission assignment change", zap.Error(err), zap.Uint("role_id", id))
	}

	h.logger.Info("Role permissions assigned", zap.Uint("role_id", id))
	response.GinSuccess(c, gin.H{"message": "权限分配成功"})
}

// GetAllPermissions 获取所有权限
// @Summary 获取所有权限
// @Description 获取系统中所有可用权限
// @Tags 角色管理
// @Security Bearer
// @Success 200 {object} response.Response{data=[]models.Permission}
// @Router /api/v1/permissions [get]
func (h *RoleHandler) GetAllPermissions(c *gin.Context) {
	permissions, err := h.roleService.GetAllPermissions(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, permissions)
}
