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

// APIKeyHandler API密钥处理器
type APIKeyHandler struct {
	apiKeyService *services.APIKeyService
	auditService  *audit.AuditLogService
	logger        *zap.Logger
}

// NewAPIKeyHandler 创建API密钥处理器
func NewAPIKeyHandler(apiKeyService *services.APIKeyService, auditService *audit.AuditLogService, logger *zap.Logger) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
		auditService:  auditService,
		logger:        logger,
	}
}

// SetLogger 设置日志记录器
func (h *APIKeyHandler) SetLogger(logger *zap.Logger) {
	h.logger = logger
}

// maskKey 脱敏显示密钥，仅保留前8位
func maskKey(key string) string {
	if len(key) > 8 {
		return key[:8] + "..."
	}
	return key
}

// buildAPIKeyResponse 构建 API Key 响应（隐藏完整密钥）
func buildAPIKeyResponse(apiKey *models.APIKey) map[string]interface{} {
	return map[string]interface{}{
		"id":            apiKey.ID,
		"name":          apiKey.Name,
		"key":           maskKey(apiKey.Key),
		"user_id":       apiKey.UserID,
		"expires_at":    apiKey.ExpiresAt,
		"is_active":     apiKey.IsActive,
		"scopes":        apiKey.GetScopeList(),
		"ip_whitelist":  apiKey.GetIPWhitelist(),
		"description":   apiKey.Description,
		"inherit_perms": apiKey.InheritPerms,
		"last_used_at":  apiKey.LastUsedAt,
		"created_at":    apiKey.CreatedAt,
		"updated_at":    apiKey.UpdatedAt,
	}
}

// normalizePagination 规范化分页参数
func normalizePagination(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}

// CreateAPIKey 创建API密钥
// @Summary 创建API密钥
// @Description 创建新的永久API密钥
// @Tags API密钥管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.CreateAPIKeyRequest true "创建请求"
// @Success 200 {object} response.Response{data=models.APIKey}
// @Router /api/v1/apikeys [post]
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
	var req services.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}

	apiKey, fullKey, err := h.apiKeyService.CreateAPIKey(userID, &req)
	if err != nil {
		h.logger.Warn("创建API密钥失败",
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	// 返回完整密钥（仅此一次）
	responseData := buildAPIKeyResponse(apiKey)
	responseData["key"] = fullKey

	h.logger.Info("API密钥创建成功",
		zap.Uint("user_id", userID),
		zap.Uint("key_id", apiKey.ID),
	)

	response.GinSuccess(c, responseData)
}

// ListAPIKeys 获取API密钥列表
// @Summary 获取API密钥列表
// @Description 获取当前用户的API密钥列表（管理员可查看所有）
// @Tags API密钥管理
// @Security Bearer
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "关键词搜索"
// @Param is_active query bool false "状态筛选"
// @Success 200 {object} response.Response{data=services.ListAPIKeysResponse}
// @Router /api/v1/apikeys [get]
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	var req services.ListAPIKeysRequest
	c.ShouldBindQuery(&req)
	req.Page, req.PageSize = normalizePagination(req.Page, req.PageSize)

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)

	result, err := h.apiKeyService.ListAPIKeys(userID, isAdmin, &req)
	if err != nil {
		h.logger.Error("获取API密钥列表失败",
			zap.Uint("user_id", userID),
			zap.Error(err),
		)
		response.GinError(c, response.CodeInternalError, "获取列表失败")
		return
	}

	// 隐藏完整密钥，只显示前8位
	items := make([]map[string]interface{}, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, buildAPIKeyResponse(&item))
	}

	response.GinSuccess(c, map[string]interface{}{
		"total": result.Total,
		"items": items,
	})
}

// GetAPIKey 获取API密钥详情
// @Summary 获取API密钥详情
// @Description 获取指定API密钥的详细信息
// @Tags API密钥管理
// @Security Bearer
// @Produce json
// @Param id path int true "密钥ID"
// @Success 200 {object} response.Response{data=models.APIKey}
// @Router /api/v1/apikeys/{id} [get]
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的密钥ID")
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)

	apiKey, err := h.apiKeyService.GetAPIKey(id, userID, isAdmin)
	if err != nil {
		response.GinError(c, response.CodeNotFound, err.Error())
		return
	}

	response.GinSuccess(c, buildAPIKeyResponse(apiKey))
}

// UpdateAPIKey 更新API密钥
// @Summary 更新API密钥
// @Description 更新指定API密钥的配置
// @Tags API密钥管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "密钥ID"
// @Param request body services.UpdateAPIKeyRequest true "更新请求"
// @Success 200 {object} response.Response{data=models.APIKey}
// @Router /api/v1/apikeys/{id} [put]
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的密钥ID")
		return
	}

	var req services.UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)

	oldAPIKey, apiKey, err := h.apiKeyService.UpdateAPIKey(id, userID, isAdmin, &req)
	if err != nil {
		h.logger.Warn("更新API密钥失败",
			zap.Uint("user_id", userID),
			zap.Uint("key_id", id),
			zap.Error(err),
		)
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	resourceID := apiKey.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionUpdate,
		Module:     models.ModuleAPIKey,
		Resource:   fmt.Sprintf("apikey:%d", apiKey.ID),
		ResourceID: &resourceID,
		OldData:    oldAPIKey,
		NewData:    apiKey,
	}); err != nil {
		h.logger.Warn("Failed to record apikey update change", zap.Error(err), zap.Uint("key_id", id))
	}

	h.logger.Info("API密钥更新成功",
		zap.Uint("user_id", userID),
		zap.Uint("key_id", id),
	)

	response.GinSuccess(c, buildAPIKeyResponse(apiKey))
}

// DeleteAPIKey 删除API密钥
// @Summary 删除API密钥
// @Description 删除指定的API密钥
// @Tags API密钥管理
// @Security Bearer
// @Produce json
// @Param id path int true "密钥ID"
// @Success 200 {object} response.Response
// @Router /api/v1/apikeys/{id} [delete]
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的密钥ID")
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)

	oldAPIKey, err := h.apiKeyService.DeleteAPIKey(id, userID, isAdmin)
	if err != nil {
		h.logger.Warn("删除API密钥失败",
			zap.Uint("user_id", userID),
			zap.Uint("key_id", id),
			zap.Error(err),
		)
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	resourceID := oldAPIKey.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionDelete,
		Module:     models.ModuleAPIKey,
		Resource:   fmt.Sprintf("apikey:%d", oldAPIKey.ID),
		ResourceID: &resourceID,
		OldData:    oldAPIKey,
		NewData:    nil,
	}); err != nil {
		h.logger.Warn("Failed to record apikey delete change", zap.Error(err), zap.Uint("key_id", id))
	}

	h.logger.Info("API密钥删除成功",
		zap.Uint("user_id", userID),
		zap.Uint("key_id", id),
	)

	response.GinSuccess(c, gin.H{
		"message": "删除成功",
	})
}

// ToggleAPIKeyStatus 切换API密钥状态
// @Summary 切换API密钥状态
// @Description 启用或禁用指定的API密钥
// @Tags API密钥管理
// @Security Bearer
// @Produce json
// @Param id path int true "密钥ID"
// @Success 200 {object} response.Response{data=models.APIKey}
// @Router /api/v1/apikeys/{id}/toggle [post]
func (h *APIKeyHandler) ToggleAPIKeyStatus(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的密钥ID")
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)

	oldAPIKey, apiKey, err := h.apiKeyService.ToggleAPIKeyStatus(id, userID, isAdmin)
	if err != nil {
		h.logger.Warn("切换API密钥状态失败",
			zap.Uint("user_id", userID),
			zap.Uint("key_id", id),
			zap.Error(err),
		)
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	resourceID := apiKey.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     "toggle",
		Module:     models.ModuleAPIKey,
		Resource:   fmt.Sprintf("apikey:%d", apiKey.ID),
		ResourceID: &resourceID,
		OldData:    oldAPIKey,
		NewData:    apiKey,
	}); err != nil {
		h.logger.Warn("Failed to record apikey toggle change", zap.Error(err), zap.Uint("key_id", id))
	}

	h.logger.Info("API密钥状态切换成功",
		zap.Uint("user_id", userID),
		zap.Uint("key_id", id),
		zap.Bool("is_active", apiKey.IsActive),
	)

	response.GinSuccess(c, buildAPIKeyResponse(apiKey))
}

// ListUsageLogs 获取 API Key 使用日志
// @Summary 获取 API Key 使用日志
// @Description 获取指定密钥或当前用户所有密钥的使用日志
// @Tags API密钥管理
// @Security Bearer
// @Produce json
// @Param id path int true "密钥ID（可选，0表示查询所有密钥）"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param success query bool false "筛选成功/失败"
// @Param start_time query string false "开始时间 ISO 8601"
// @Param end_time query string false "结束时间 ISO 8601"
// @Param method query string false "HTTP 方法"
// @Success 200 {object} response.Response
// @Router /api/v1/apikeys/{id}/logs [get]
func (h *APIKeyHandler) ListUsageLogs(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的密钥ID")
		return
	}

	var req services.ListUsageLogsRequest
	c.ShouldBindQuery(&req)
	req.Page, req.PageSize = normalizePagination(req.Page, req.PageSize)

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)

	logs, total, err := h.apiKeyService.ListUsageLogs(userID, isAdmin, id, &req)
	if err != nil {
		h.logger.Error("获取使用日志失败",
			zap.Uint("user_id", userID),
			zap.Uint("key_id", id),
			zap.Error(err),
		)
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	response.GinSuccess(c, map[string]interface{}{
		"total": total,
		"items": logs,
	})
}

// GetUsageLogSummary 获取 API Key 使用统计
// @Summary 获取 API Key 使用统计
// @Description 获取指定密钥或当前用户所有密钥的使用统计
// @Tags API密钥管理
// @Security Bearer
// @Produce json
// @Param id path int true "密钥ID（可选，0表示统计所有密钥）"
// @Success 200 {object} response.Response{data=services.UsageLogSummary}
// @Router /api/v1/apikeys/{id}/summary [get]
func (h *APIKeyHandler) GetUsageLogSummary(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的密钥ID")
		return
	}

	userID, ok := middleware.GetUserID(c)

	if !ok {

		c.AbortWithStatusJSON(401, gin.H{"error": "user not in context"})

		return

	}
	isAdmin := middleware.GetIsAdmin(c)

	summary, err := h.apiKeyService.GetUsageLogSummary(userID, isAdmin, id)
	if err != nil {
		h.logger.Error("获取使用统计失败",
			zap.Uint("user_id", userID),
			zap.Uint("key_id", id),
			zap.Error(err),
		)
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	response.GinSuccess(c, summary)
}
