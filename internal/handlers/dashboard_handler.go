package handlers

import (
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DashboardHandler 仪表板处理器
type DashboardHandler struct {
	dashboardService *services.DashboardService
	logger           *zap.Logger
}

// NewDashboardHandler 创建仪表板处理器
func NewDashboardHandler(dashboardService *services.DashboardService, logger *zap.Logger) *DashboardHandler {
	return &DashboardHandler{
		dashboardService: dashboardService,
		logger:           logger,
	}
}

// GetStats 获取仪表板统计数据
// @Summary 获取仪表板统计数据
// @Description 获取任务、文件、系统统计信息
// @Tags 仪表板
// @Security Bearer
// @Success 200 {object} response.Response{data=services.DashboardStatsResponse}
// @Router /api/v1/dashboard/stats [get]
func (h *DashboardHandler) GetStats(c *gin.Context) {
	// 调用服务获取统计数据
	result, err := h.dashboardService.GetDashboardStats(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get dashboard stats", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取统计数据失败")
		return
	}

	response.GinSuccess(c, result)
}
