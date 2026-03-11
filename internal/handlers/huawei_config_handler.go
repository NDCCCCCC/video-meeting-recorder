package handlers

import (
	"fmt"
	"strings"

	"github.com/cpic/record_v2/internal/services"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HuaweiConfigHandler 华为配置处理器
type HuaweiConfigHandler struct {
	configService *services.HuaweiConfigService
	logger        *zap.Logger
	usbScanner    *services.USBDeviceScanner
}

// NewHuaweiConfigHandler 创建华为配置处理器
func NewHuaweiConfigHandler(
	configService *services.HuaweiConfigService,
	logger *zap.Logger,
	usbScanner *services.USBDeviceScanner,
) *HuaweiConfigHandler {
	return &HuaweiConfigHandler{
		configService: configService,
		logger:        logger,
		usbScanner:    usbScanner,
	}
}

// ListConfigs 获取配置列表
// @Summary 获取华为配置列表
// @Description 分页获取华为配置列表，支持筛选
// @Tags 华为配置
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param keyword query string false "搜索关键词"
// @Param is_active query bool false "是否激活"
// @Success 200 {object} response.Response{data=services.ListConfigsResponse}
// @Router /api/v1/huawei-configs [get]
func (h *HuaweiConfigHandler) ListConfigs(c *gin.Context) {
	var req services.ListConfigsRequest
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

	result, err := h.configService.ListConfigs(&req)
	if err != nil {
		h.logger.Error("Failed to list huawei configs", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取配置列表失败")
		return
	}

	response.GinSuccess(c, result)
}

// GetConfig 获取配置详情
// @Summary 获取华为配置详情
// @Description 根据ID获取华为配置详细信息
// @Tags 华为配置
// @Security Bearer
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response{data=models.HuaweiConfig}
// @Router /api/v1/huawei-configs/{id} [get]
func (h *HuaweiConfigHandler) GetConfig(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的配置ID")
		return
	}

	config, err := h.configService.GetConfigByID(id)
	if err != nil {
		response.GinError(c, response.CodeNotFound, "配置不存在")
		return
	}

	response.GinSuccess(c, config)
}

// CreateConfig 创建华为配置
// @Summary 创建华为配置
// @Description 创建新的华为配置
// @Tags 华为配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.CreateConfigRequest true "创建配置请求"
// @Success 200 {object} response.Response{data=models.HuaweiConfig}
// @Router /api/v1/huawei-configs [post]
func (h *HuaweiConfigHandler) CreateConfig(c *gin.Context) {
	var req services.CreateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	config, err := h.configService.CreateConfig(&req)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("Huawei config created", zap.Uint("config_id", config.ID), zap.String("name", config.Name))
	response.GinSuccess(c, config)
}

// UpdateConfig 更新华为配置
// @Summary 更新华为配置
// @Description 更新华为配置信息
// @Tags 华为配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param id path int true "配置ID"
// @Param request body services.UpdateConfigRequest true "更新配置请求"
// @Success 200 {object} response.Response{data=models.HuaweiConfig}
// @Router /api/v1/huawei-configs/{id} [put]
func (h *HuaweiConfigHandler) UpdateConfig(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的配置ID")
		return
	}

	var req services.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	config, err := h.configService.UpdateConfig(id, &req)
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("Huawei config updated", zap.Uint("config_id", id))
	response.GinSuccess(c, config)
}

// DeleteConfig 删除华为配置
// @Summary 删除华为配置
// @Description 删除指定的华为配置
// @Tags 华为配置
// @Security Bearer
// @Param id path int true "配置ID"
// @Success 200 {object} response.Response
// @Router /api/v1/huawei-configs/{id} [delete]
func (h *HuaweiConfigHandler) DeleteConfig(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		response.GinError(c, response.CodeInvalidRequest, "无效的配置ID")
		return
	}

	if err := h.configService.DeleteConfig(id); err != nil {
		response.GinError(c, response.CodeInvalidRequest, err.Error())
		return
	}

	h.logger.Info("Huawei config deleted", zap.Uint("config_id", id))
	response.GinSuccess(c, gin.H{"message": "删除成功"})
}

// GetActiveConfigs 获取可用配置列表
// @Summary 获取可用华为配置列表
// @Description 获取所有激活状态的华为配置，用于下拉选择
// @Tags 华为配置
// @Security Bearer
// @Success 200 {object} response.Response{data=[]models.HuaweiConfig}
// @Router /api/v1/huawei-configs/active [get]
func (h *HuaweiConfigHandler) GetActiveConfigs(c *gin.Context) {
	configs, err := h.configService.GetActiveConfigs()
	if err != nil {
		h.logger.Error("Failed to get active huawei configs", zap.Error(err))
		response.GinError(c, response.CodeInternalError, "获取可用配置失败")
		return
	}

	response.GinSuccess(c, configs)
}

// ScanUSBDevices 扫描USB设备
// @Summary 扫描系统USB设备
// @Description 自动识别系统中的摄像头和音频设备
// @Tags 华为配置
// @Security Bearer
// @Success 200 {object} response.Response{data=map[string][]services.USBDeviceInfo}
// @Router /api/v1/huawei-configs/scan-devices [get]
func (h *HuaweiConfigHandler) ScanUSBDevices(c *gin.Context) {
	devices := h.usbScanner.ScanAllUSBDevices()
	response.GinSuccess(c, devices)
}

// GetRecommendedDevice 获取推荐设备
// @Summary 获取推荐USB设备
// @Description 获取指定类型的第一个可用设备
// @Tags 华为配置
// @Security Bearer
// @Param type query string true "设备类型" Enums(camera, audio)
// @Success 200 {object} response.Response{data=services.USBDeviceInfo}
// @Router /api/v1/huawei-configs/recommended-device [get]
func (h *HuaweiConfigHandler) GetRecommendedDevice(c *gin.Context) {
	deviceType := c.Query("type")
	if deviceType == "" {
		response.GinError(c, response.CodeInvalidRequest, "请指定设备类型 (camera 或 audio)")
		return
	}

	if deviceType != "camera" && deviceType != "audio" {
		response.GinError(c, response.CodeInvalidRequest, "设备类型必须是 camera 或 audio")
		return
	}

	device := h.usbScanner.GetRecommendedDevice(deviceType)
	if device == nil {
		response.GinSuccess(c, gin.H{"message": "未找到可用设备", "device": nil})
		return
	}

	response.GinSuccess(c, device)
}

// TestStream 测试流媒体连接
// @Summary 测试流媒体连接
// @Description 测试指定流媒体地址的连接是否可用
// @Tags 华为配置
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body services.TestStreamRequest true "测试流媒体连接请求"
// @Success 200 {object} response.Response
// @Router /api/v1/huawei-configs/test-stream [post]
func (h *HuaweiConfigHandler) TestStream(c *gin.Context) {
	var req services.TestStreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	if err := h.configService.TestStreamConnection(&req); err != nil {
		response.GinError(c, response.CodeInternalError, err.Error())
		return
	}

	response.GinSuccess(c, gin.H{"message": "连接测试成功"})
}

// PreviewStream 预览流媒体
// @Summary 预览流媒体
// @Description 将流媒体转换为 HLS 格式供前端播放
// @Tags 华为配置
// @Security Bearer
// @Param protocol query string true "协议类型" Enums(rtmp, rtsp, srt, hls)
// @Param url query string true "流媒体URL"
// @Param username query string false "用户名"
// @Param password query string false "密码"
// @Success 200 {object} response.Response
// @Router /api/v1/stream/preview [get]
func (h *HuaweiConfigHandler) PreviewStream(c *gin.Context) {
	protocol := c.Query("protocol")
	url := c.Query("url")
	_, _ = c.Query("username"), c.Query("password") // 预留认证参数

	if protocol == "" || url == "" {
		response.GinError(c, response.CodeInvalidRequest, "缺少必要参数")
		return
	}

	// 对于 HLS，直接返回 URL
	if protocol == "hls" {
		response.GinSuccess(c, gin.H{"preview_url": url, "type": "hls"})
		return
	}

	// 对于其他协议，返回提示信息
	// 说明：RTMP/RTSP/SRT 转换为 HLS 需要持续运行的服务
	// 建议用户使用录制的 HLS 预览功能或直接使用 VLC 等播放器预览
	h.logger.Info("预览请求",
		zap.String("protocol", protocol),
		zap.String("url", url),
	)

	response.GinSuccess(c, gin.H{
		"message":           fmt.Sprintf("%s 协议需要先启动录制才能预览。请创建临时录制任务来预览画面。", strings.ToUpper(protocol)),
		"protocol":          protocol,
		"preview_available": false,
	})
}
