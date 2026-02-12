package handlers

import (
	"strconv"
	"time"

	"github.com/cpic/record_v2/internal/services/audit"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AuditHandler 审计日志处理器
type AuditHandler struct {
	auditService *audit.AuditLogService
	logger       *zap.Logger
}

// NewAuditHandler 创建审计日志处理器
func NewAuditHandler(auditService *audit.AuditLogService) *AuditHandler {
	return &AuditHandler{
		auditService: auditService,
	}
}

// SetLogger 设置日志记录器
func (h *AuditHandler) SetLogger(logger *zap.Logger) {
	h.logger = logger
}

// Stop 停止处理器
func (h *AuditHandler) Stop() {
	if h.auditService != nil {
		h.auditService.Stop()
	}
}

// getUserID 从上下文获取用户ID
func (h *AuditHandler) getUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		if id, ok := userID.(uint); ok {
			return id
		}
	}
	return 0
}

// getDataScope 从上下文获取数据范围
func (h *AuditHandler) getDataScope(c *gin.Context) string {
	if scope, exists := c.Get("data_scope"); exists {
		if s, ok := scope.(string); ok {
			return s
		}
	}
	return "all"
}

// Query 查询审计日志
// @Summary 查询审计日志
// @Tags 审计日志
// @Produce json
// @Param module query string false "模块"
// @Param action query string false "操作"
// @Param status query string false "状态"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "页大小" default(20)
// @Success 200 {object} response.Response
// @Router /api/v1/audit/logs [get]
func (h *AuditHandler) Query(c *gin.Context) {
	var req audit.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	// 解析时间
	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			req.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			req.EndTime = t
		}
	}

	result, err := h.auditService.Query(c.Request.Context(), &req, h.getUserID(c), h.getDataScope(c))
	if err != nil {
		h.logger.Warn("查询审计日志失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "查询失败")
		return
	}

	response.GinSuccess(c, result)
}

// GetByID 获取日志详情
// @Summary 获取审计日志详情
// @Tags 审计日志
// @Produce json
// @Param id path int true "日志ID"
// @Success 200 {object} response.Response
// @Router /api/v1/audit/logs/:id [get]
func (h *AuditHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的ID")
		return
	}

	log, err := h.auditService.GetByID(c.Request.Context(), uint(id), h.getUserID(c), h.getDataScope(c))
	if err != nil {
		response.GinError(c, response.CodeNotFound, "日志不存在")
		return
	}

	response.GinSuccess(c, log)
}

// Statistics 获取操作统计
// @Summary 获取操作统计
// @Tags 审计日志
// @Produce json
// @Param days query int false "统计天数" default(7)
// @Success 200 {object} response.Response
// @Router /api/v1/audit/statistics [get]
func (h *AuditHandler) Statistics(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 {
		days = 7
	}

	stats, err := h.auditService.GetStatistics(c.Request.Context(), days, h.getUserID(c), h.getDataScope(c))
	if err != nil {
		h.logger.Warn("获取统计失败", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取统计失败")
		return
	}

	response.GinSuccess(c, stats)
}
