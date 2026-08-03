package handlers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// InputConfigHandler 输入配置处理器
type InputConfigHandler struct {
	configService *services.InputConfigService
	auditService  *audit.AuditLogService
	logger        *zap.Logger
	usbScanner    *services.USBDeviceScanner
}

// NewInputConfigHandler 创建输入配置处理器
func NewInputConfigHandler(
	configService *services.InputConfigService,
	auditService *audit.AuditLogService,
	logger *zap.Logger,
	usbScanner *services.USBDeviceScanner,
) *InputConfigHandler {
	return &InputConfigHandler{
		configService: configService,
		auditService:  auditService,
		logger:        logger,
		usbScanner:    usbScanner,
	}
}

// ListConfigs 获取输入配置列表
// @Summary 获取输入配置列表
// @Description 分页获取输入配置列表，支持筛选
// @Tags 输入配置
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Param is_active query bool false "是否激活"
// @Success 200 {object} response.Response{data=services.InputConfigListResponse}
// @Router /api/v1/input-configs [get]
func (h *InputConfigHandler) ListConfigs(c *gin.Context) {
	var req services.ListConfigsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误")
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	result, err := h.configService.ListConfigs(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to list input configs", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, result)
}

// GetConfig 获取输入配置详情
// @Summary 获取输入配置详情
// @Description 根据ID获取输入配置详情
// @Tags 输入配置
// @Security Bearer
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response{data=models.InputConfig}
// @Router /api/v1/input-configs/{id} [get]
func (h *InputConfigHandler) GetConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的配置ID")
		return
	}

	config, err := h.configService.GetConfigByID(c.Request.Context(), uint(id))
	if err != nil {
		h.logger.Error("Failed to get input config", zap.Error(err), response.SentinelField(err))
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, config)
}

// CreateConfig 创建输入配置
// @Summary 创建输入配置
// @Description 创建新的输入配置（华为终端/USB设备/流媒体）
// @Tags 输入配置
// @Security Bearer
// @Accept json
// @Param request body services.CreateInputConfigRequest true "创建配置请求"
// @Success 200 {object} response.Response{data=models.InputConfig}
// @Router /api/v1/input-configs [post]
func (h *InputConfigHandler) CreateConfig(c *gin.Context) {
	var req services.CreateInputConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	config, err := h.configService.CreateConfig(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 审计：OldData=nil (insert), NewData=新 config
	resourceID := config.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionCreate,
		Module:     models.ModuleInputConfig,
		Resource:   fmt.Sprintf("input_config:%d", config.ID),
		ResourceID: &resourceID,
		OldData:    nil,
		NewData:    config,
	}); err != nil {
		h.logger.Warn("Failed to record input config create change", zap.Error(err), response.SentinelField(err), zap.Uint("config_id", config.ID))
	}

	h.logger.Info("Input config created",
		zap.Uint("config_id", config.ID),
		zap.String("config_type", config.ConfigType),
	)
	response.GinSuccess(c, config)
}

// UpdateConfig 更新输入配置
// @Summary 更新输入配置
// @Description 更新输入配置信息
// @Tags 输入配置
// @Security Bearer
// @Accept json
// @Param id path int true "配置ID"
// @Param request body services.UpdateInputConfigRequest true "更新配置请求"
// @Success 200 {object} response.Response{data=models.InputConfig}
// @Router /api/v1/input-configs/{id} [put]
func (h *InputConfigHandler) UpdateConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的配置ID")
		return
	}

	var req services.UpdateInputConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	oldConfig, config, err := h.configService.UpdateConfig(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 审计：OldData=oldConfig (pre-mutation), NewData=config (post-mutation)
	resourceID := config.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionUpdate,
		Module:     models.ModuleInputConfig,
		Resource:   fmt.Sprintf("input_config:%d", config.ID),
		ResourceID: &resourceID,
		OldData:    oldConfig,
		NewData:    config,
	}); err != nil {
		h.logger.Warn("Failed to record input config update change", zap.Error(err), response.SentinelField(err), zap.Uint("config_id", config.ID))
	}

	response.GinSuccess(c, config)
}

// DeleteConfig 删除输入配置
// @Summary 删除输入配置
// @Description 删除指定的输入配置
// @Tags 输入配置
// @Security Bearer
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response
// @Router /api/v1/input-configs/{id} [delete]
func (h *InputConfigHandler) DeleteConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的配置ID")
		return
	}

	oldConfig, err := h.configService.DeleteConfig(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleError(c, err)
		return
	}

	// 审计：OldData=oldConfig (pre-delete), NewData=nil
	resourceID := oldConfig.ID
	if err := h.auditService.RecordChange(c.Request.Context(), audit.RecordChangeOpts{
		Action:     models.ActionDelete,
		Module:     models.ModuleInputConfig,
		Resource:   fmt.Sprintf("input_config:%d", oldConfig.ID),
		ResourceID: &resourceID,
		OldData:    oldConfig,
		NewData:    nil,
	}); err != nil {
		h.logger.Warn("Failed to record input config delete change", zap.Error(err), response.SentinelField(err), zap.Uint("config_id", uint(id)))
	}

	response.GinSuccess(c, gin.H{"message": "配置已删除"})
}

// TestConnection 测试输入配置连接
// @Summary 测试输入配置连接
// @Description 测试USB设备/流媒体/华为终端的连接性
// @Tags 输入配置
// @Security Bearer
// @Accept json
// @Param id path int true "配置ID"
// @Param request body services.TestConnectionRequest true "测试连接请求"
// @Success 200 {object} response.Response
// @Router /api/v1/input-configs/{id}/test [post]
func (h *InputConfigHandler) TestConnection(c *gin.Context) {
	var req services.TestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.configService.TestConnection(c.Request.Context(), &req); err != nil {
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, gin.H{"message": "连接测试成功"})
}

// ScanUSBDevices 扫描系统USB设备
// @Summary 扫描系统USB设备
// @Description 扫描系统中的摄像头和音频设备
// @Tags 输入配置
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string][]services.USBDeviceInfo}
// @Router /api/v1/input-configs/usb-devices [get]
func (h *InputConfigHandler) ScanUSBDevices(c *gin.Context) {
	devices := h.usbScanner.ScanAllUSBDevices()
	response.GinSuccess(c, devices)
}

// GetActiveConfigs 获取激活的输入配置
// @Summary 获取激活的输入配置
// @Description 获取所有激活的输入配置列表
// @Tags 输入配置
// @Security Bearer
// @Success 200 {object} response.Response{data=[]models.InputConfig}
// @Router /api/v1/input-configs/active [get]
func (h *InputConfigHandler) GetActiveConfigs(c *gin.Context) {
	var req services.ListConfigsRequest
	isActive := true
	req.IsActive = &isActive
	req.Page = 1
	req.PageSize = 1000

	result, err := h.configService.ListConfigs(c.Request.Context(), &req)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.GinSuccess(c, result.Items)
}
