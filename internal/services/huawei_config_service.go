package services

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HuaweiConfigService 华为配置服务
type HuaweiConfigService struct {
	db     *gorm.DB
	logger *zap.Logger
	config *config.Config
}

// NewHuaweiConfigService 创建华为配置服务
func NewHuaweiConfigService(db *gorm.DB, logger *zap.Logger, cfg *config.Config) *HuaweiConfigService {
	return &HuaweiConfigService{
		db:     db,
		logger: logger,
		config: cfg,
	}
}

// ListConfigsRequest 配置列表请求
type ListConfigsRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size" binding:"max=100"`
	Keyword  string `form:"keyword"`
	IsActive *bool  `form:"is_active"`
}

// ListConfigsResponse 配置列表响应
type ListConfigsResponse struct {
	Total int64                 `json:"total"`
	Items []models.HuaweiConfig `json:"items"`
}

// CreateConfigRequest 创建配置请求
type CreateConfigRequest struct {
	Name             string `json:"name" binding:"required,max=100"`
	Description      string `json:"description" binding:"max=500"`
	Server           string `json:"server" binding:"required,max=100"`
	Port             int    `json:"port" binding:"min=1,max=65535"`
	Username         string `json:"username" binding:"required,max=50"`
	Password         string `json:"password" binding:"required,max=100"`
	TerminalNumber   string `json:"terminal_number" binding:"required,max=50"`
	ConferenceNumber string `json:"conference_number" binding:"omitempty,max=50"`
	CameraBackend    string `json:"camera_backend" binding:"omitempty,max=20"` // dshow | v4l2 | avfoundation
	AudioBackend     string `json:"audio_backend" binding:"omitempty,max=20"`  // dshow | alsa | coreaudio | wasapi
	USBCameraName    string `json:"usb_camera_name" binding:"omitempty,max=100"`
	USBCameraDevice  string `json:"usb_camera_device" binding:"omitempty,max=100"`
	USBAudioName     string `json:"usb_audio_name" binding:"omitempty,max=100"`
	USBAudioDevice   string `json:"usb_audio_device" binding:"omitempty,max=100"`
	OutputFormat     string `json:"output_format" binding:"omitempty,max=20"`
	// 流媒体配置字段
	StreamProtocol string `json:"stream_protocol" binding:"omitempty,oneof=rtmp rtsp srt hls"`
	StreamURL      string `json:"stream_url" binding:"omitempty,max=500,url"`
	StreamUsername string `json:"stream_username" binding:"omitempty,max=100"`
	StreamPassword string `json:"stream_password" binding:"omitempty,max=100"`
	StreamEnabled  bool   `json:"stream_enabled"`
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	Name             *string `json:"name" binding:"omitempty,max=100"`
	Description      *string `json:"description" binding:"omitempty,max=500"`
	Server           *string `json:"server" binding:"omitempty,max=100"`
	Port             *int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username         *string `json:"username" binding:"omitempty,max=50"`
	Password         *string `json:"password" binding:"omitempty,max=100"`
	TerminalNumber   *string `json:"terminal_number" binding:"omitempty,max=50"`
	ConferenceNumber *string `json:"conference_number" binding:"omitempty,max=50"`
	CameraBackend    *string `json:"camera_backend" binding:"omitempty,max=20"`
	AudioBackend     *string `json:"audio_backend" binding:"omitempty,max=20"`
	USBCameraName    *string `json:"usb_camera_name" binding:"omitempty,max=100"`
	USBCameraDevice  *string `json:"usb_camera_device" binding:"omitempty,max=100"`
	USBAudioName     *string `json:"usb_audio_name" binding:"omitempty,max=100"`
	USBAudioDevice   *string `json:"usb_audio_device" binding:"omitempty,max=100"`
	OutputFormat     *string `json:"output_format" binding:"omitempty,max=20"`
	IsActive         *bool   `json:"is_active"`
	// 流媒体配置字段
	StreamProtocol *string `json:"stream_protocol" binding:"omitempty,oneof=rtmp rtsp srt hls"`
	StreamURL      *string `json:"stream_url" binding:"omitempty,max=500,url"`
	StreamUsername *string `json:"stream_username" binding:"omitempty,max=100"`
	StreamPassword *string `json:"stream_password" binding:"omitempty,max=100"`
	StreamEnabled  *bool   `json:"stream_enabled"`
}

// TestStreamRequest 测试流媒体连接请求
type TestStreamRequest struct {
	Protocol string `json:"protocol" binding:"required,oneof=rtmp rtsp srt hls"`
	URL      string `json:"url" binding:"required,max=500,url"`
	Username string `json:"username" binding:"omitempty,max=100"`
	Password string `json:"password" binding:"omitempty,max=100"`
}

// ListConfigs 获取配置列表
func (s *HuaweiConfigService) ListConfigs(req *ListConfigsRequest) (*ListConfigsResponse, error) {
	var configs []models.HuaweiConfig
	var total int64

	query := s.db.Model(&models.HuaweiConfig{}).Preload("VideoRecordingTasks")

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ? OR server LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 状态筛选
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Offset(offset).
		Limit(req.PageSize).
		Order("id ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}

	return &ListConfigsResponse{
		Total: total,
		Items: configs,
	}, nil
}

// GetConfigByID 根据ID获取配置
func (s *HuaweiConfigService) GetConfigByID(id uint) (*models.HuaweiConfig, error) {
	var config models.HuaweiConfig
	if err := s.db.Preload("VideoRecordingTasks").First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// CreateConfig 创建配置
func (s *HuaweiConfigService) CreateConfig(req *CreateConfigRequest) (*models.HuaweiConfig, error) {
	config := &models.HuaweiConfig{
		Name:             req.Name,
		Description:      req.Description,
		Server:           req.Server,
		Port:             req.Port,
		Username:         req.Username,
		Password:         req.Password,
		TerminalNumber:   req.TerminalNumber,
		ConferenceNumber: req.ConferenceNumber,
		CameraBackend:    req.CameraBackend,
		AudioBackend:     req.AudioBackend,
		USBCameraName:    req.USBCameraName,
		USBCameraDevice:  req.USBCameraDevice,
		USBAudioName:     req.USBAudioName,
		USBAudioDevice:   req.USBAudioDevice,
		OutputFormat:     req.OutputFormat,
		StreamProtocol:   req.StreamProtocol,
		StreamURL:        req.StreamURL,
		StreamUsername:   req.StreamUsername,
		StreamPassword:   req.StreamPassword,
		StreamEnabled:    req.StreamEnabled,
		IsActive:         true,
	}

	// 验证后端配置
	if err := s.validateBackendConfig(config); err != nil {
		return nil, err
	}

	// 设置默认值
	if config.CameraBackend == "" {
		config.CameraBackend = "dshow"
	}
	if config.AudioBackend == "" {
		config.AudioBackend = "dshow"
	}
	if config.OutputFormat == "" {
		config.OutputFormat = "mp4"
	}

	if err := s.db.Create(config).Error; err != nil {
		return nil, err
	}

	s.logger.Info("华为配置已创建",
		zap.Uint("config_id", config.ID),
		zap.String("name", config.Name),
	)

	return config, nil
}

// UpdateConfig 更新配置
func (s *HuaweiConfigService) UpdateConfig(id uint, req *UpdateConfigRequest) (*models.HuaweiConfig, error) {
	var config models.HuaweiConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return nil, errors.New("配置不存在")
	}

	// 检查是否被锁定
	if config.IsLocked {
		return nil, errors.New("配置正在使用中，无法修改")
	}

	updates := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Server != nil {
		updates["server"] = *req.Server
	}
	if req.Port != nil {
		updates["port"] = *req.Port
	}
	if req.Username != nil {
		updates["username"] = *req.Username
	}
	if req.Password != nil {
		updates["password"] = *req.Password
	}
	if req.TerminalNumber != nil {
		updates["terminal_number"] = *req.TerminalNumber
	}
	if req.ConferenceNumber != nil {
		updates["conference_number"] = *req.ConferenceNumber
	}
	if req.CameraBackend != nil {
		updates["camera_backend"] = *req.CameraBackend
	}
	if req.AudioBackend != nil {
		updates["audio_backend"] = *req.AudioBackend
	}
	if req.USBCameraName != nil {
		updates["usb_camera_name"] = *req.USBCameraName
	}
	if req.USBCameraDevice != nil {
		updates["usb_camera_device"] = *req.USBCameraDevice
	}
	if req.USBAudioName != nil {
		updates["usb_audio_name"] = *req.USBAudioName
	}
	if req.USBAudioDevice != nil {
		updates["usb_audio_device"] = *req.USBAudioDevice
	}
	if req.OutputFormat != nil {
		updates["output_format"] = *req.OutputFormat
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.StreamProtocol != nil {
		updates["stream_protocol"] = *req.StreamProtocol
	}
	if req.StreamURL != nil {
		updates["stream_url"] = *req.StreamURL
	}
	if req.StreamUsername != nil {
		updates["stream_username"] = *req.StreamUsername
	}
	if req.StreamPassword != nil {
		updates["stream_password"] = *req.StreamPassword
	}
	if req.StreamEnabled != nil {
		updates["stream_enabled"] = *req.StreamEnabled
	}

	if err := s.db.Model(&config).Updates(updates).Error; err != nil {
		return nil, err
	}

	s.db.Preload("VideoRecordingTasks").First(&config, id)

	s.logger.Info("华为配置已更新", zap.Uint("config_id", id))

	return &config, nil
}

// DeleteConfig 删除配置
func (s *HuaweiConfigService) DeleteConfig(id uint) error {
	var config models.HuaweiConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return errors.New("配置不存在")
	}

	// 检查是否被锁定
	if config.IsLocked {
		return errors.New("配置正在使用中，无法删除")
	}

	// 检查是否有关联的录制任务
	var count int64
	if err := s.db.Model(&models.VideoRecordingTask{}).Where("huawei_config_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("配置已被录制任务使用，无法删除")
	}

	result := s.db.Delete(&models.HuaweiConfig{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("配置不存在")
	}

	s.logger.Info("华为配置已删除", zap.Uint("config_id", id))

	return nil
}

// LockConfig 锁定配置
func (s *HuaweiConfigService) LockConfig(id uint, taskID uint) error {
	var config models.HuaweiConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return errors.New("配置不存在")
	}

	if err := config.Lock(taskID); err != nil {
		return err
	}

	if err := s.db.Save(&config).Error; err != nil {
		return err
	}

	s.logger.Info("华为配置已锁定",
		zap.Uint("config_id", id),
		zap.Uint("task_id", taskID),
	)

	return nil
}

// UnlockConfig 解锁配置
func (s *HuaweiConfigService) UnlockConfig(id uint) error {
	var config models.HuaweiConfig
	if err := s.db.First(&config, id).Error; err != nil {
		return errors.New("配置不存在")
	}

	if err := config.Unlock(); err != nil {
		return err
	}

	if err := s.db.Save(&config).Error; err != nil {
		return err
	}

	s.logger.Info("华为配置已解锁", zap.Uint("config_id", id))

	return nil
}

// GetActiveConfigs 获取所有可用配置
func (s *HuaweiConfigService) GetActiveConfigs() ([]models.HuaweiConfig, error) {
	var configs []models.HuaweiConfig
	if err := s.db.Where("is_active = ?", true).Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// validateBackendConfig 验证后端配置
func (s *HuaweiConfigService) validateBackendConfig(config *models.HuaweiConfig) error {
	validCameraBackends := map[string]bool{
		"dshow":        true,
		"v4l2":         true,
		"avfoundation": true,
	}
	validAudioBackends := map[string]bool{
		"dshow":     true,
		"alsa":      true,
		"coreaudio": true,
		"wasapi":    true,
	}
	validOutputFormats := map[string]bool{
		"mp4": true,
		"mkv": true,
		"avi": true,
	}

	if config.CameraBackend != "" && !validCameraBackends[config.CameraBackend] {
		return fmt.Errorf("无效的摄像头后端: %s，支持的后端: dshow, v4l2, avfoundation", config.CameraBackend)
	}
	if config.AudioBackend != "" && !validAudioBackends[config.AudioBackend] {
		return fmt.Errorf("无效的音频后端: %s，支持的后端: dshow, alsa, coreaudio, wasapi", config.AudioBackend)
	}
	if config.OutputFormat != "" && !validOutputFormats[config.OutputFormat] {
		return fmt.Errorf("无效的输出格式: %s，支持的格式: mp4, mkv, avi", config.OutputFormat)
	}

	return nil
}

// TestStreamConnection 测试流媒体连接
// 使用 FFprobe 检测流媒体源是否可用，超时时间 10 秒
func (s *HuaweiConfigService) TestStreamConnection(req *TestStreamRequest) error {
	s.logger.Info("测试流媒体连接",
		zap.String("protocol", req.Protocol),
		zap.String("url", req.URL),
	)

	// 检查 FFprobe 路径
	ffprobePath := s.config.FFmpeg.FFProbePath
	if ffprobePath == "" {
		// 如果未配置，尝试使用系统路径
		ffprobePath = "ffprobe"
	}

	// 构建输入参数
	// 注意：FFprobe 不支持 -t 选项（那是 FFmpeg 的选项）
	// 测试连接时只需要确认流可访问，不需要限制分析时长
	var inputArgs []string
	switch req.Protocol {
	case "rtmp":
		inputArgs = []string{"-i", req.URL}
	case "rtsp":
		inputArgs = []string{"-rtsp_transport", "tcp", "-i", req.URL}
	case "srt":
		inputArgs = []string{"-i", req.URL}
	case "hls":
		inputArgs = []string{"-i", req.URL}
	default:
		return fmt.Errorf("不支持的流媒体协议: %s", req.Protocol)
	}

	// 添加认证信息（如果提供）
	if req.Username != "" {
		// FFprobe 支持在 URL 中嵌入认证信息
		// 格式: protocol://username:password@host/path
		if req.Password != "" {
			// 解析 URL 并插入认证信息
			urlParts := strings.SplitN(req.URL, "://", 2)
			if len(urlParts) == 2 {
				authURL := fmt.Sprintf("%s://%s:%s@%s", urlParts[0], req.Username, req.Password, urlParts[1])
				// 更新最后一个参数（URL）
				inputArgs[len(inputArgs)-1] = authURL
			}
		} else {
			// 只有用户名没有密码
			urlParts := strings.SplitN(req.URL, "://", 2)
			if len(urlParts) == 2 {
				authURL := fmt.Sprintf("%s://%s@%s", urlParts[0], req.Username, urlParts[1])
				inputArgs[len(inputArgs)-1] = authURL
			}
		}
	}

	// 构建完整的 FFprobe 命令
	// -show_streams: 显示流信息
	// -loglevel error: 只显示错误信息
	// 注意：-t 选项已作为输入选项放在 inputArgs 中（必须在 -i 之前）
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-show_streams",
		"-show_format",
		"-print_format", "json",
	}
	args = append(args, inputArgs...)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ffprobePath, args...)

	// 运行命令并捕获输出
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("连接超时（15秒），请检查网络和流媒体地址是否正确")
	}

	if err != nil {
		s.logger.Error("流媒体连接测试失败",
			zap.String("protocol", req.Protocol),
			zap.String("url", req.URL),
			zap.Error(err),
			zap.String("output", string(output)),
		)
		// 解析错误信息，提供更友好的提示
		outputStr := string(output)
		if strings.Contains(outputStr, "Connection refused") {
			return fmt.Errorf("连接被拒绝，请检查流媒体服务器是否运行")
		}
		if strings.Contains(outputStr, "Network is unreachable") {
			return fmt.Errorf("网络不可达，请检查网络连接")
		}
		if strings.Contains(outputStr, "404") || strings.Contains(outputStr, "Not Found") {
			return fmt.Errorf("流媒体地址不存在（404）")
		}
		if strings.Contains(outputStr, "401") || strings.Contains(outputStr, "Unauthorized") {
			return fmt.Errorf("认证失败，请检查用户名和密码")
		}
		if strings.Contains(outputStr, "403") || strings.Contains(outputStr, "Forbidden") {
			return fmt.Errorf("访问被拒绝（403），请检查权限设置")
		}
		if strings.Contains(outputStr, "Timeout") {
			return fmt.Errorf("连接超时，请检查网络或防火墙设置")
		}
		return fmt.Errorf("连接测试失败: %v", err)
	}

	// 验证输出是否包含有效的流信息
	outputStr := string(output)
	if !strings.Contains(outputStr, "streams") && !strings.Contains(outputStr, "format") {
		return fmt.Errorf("未检测到有效的音视频流，请确认流媒体源正常推送")
	}

	s.logger.Info("流媒体连接测试成功",
		zap.String("protocol", req.Protocol),
		zap.String("url", req.URL),
	)

	return nil
}
