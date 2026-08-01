package services

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// InputConfigService 输入配置服务
type InputConfigService struct {
	db         *gorm.DB
	logger     *zap.Logger
	config     *config.Config
	usbScanner *USBDeviceScanner
	encryptor  *CredentialEncryptor // Phase 18: Phase 18 时为 nil（注入时机由 cmd/server/app.go 决定）
}

// NewInputConfigService 创建输入配置服务。
// encryptor 可为 nil（用于早期启动或测试场景）；nil 时不会加密凭据，
// 写入端会记录 Warn 但不阻断（保留 Phase 17 行为，便于单元测试）。
func NewInputConfigService(db *gorm.DB, logger *zap.Logger, cfg *config.Config, usbScanner *USBDeviceScanner, encryptor *CredentialEncryptor) *InputConfigService {
	return &InputConfigService{
		db:         db,
		logger:     logger,
		config:     cfg,
		usbScanner: usbScanner,
		encryptor:  encryptor,
	}
}

// encryptPasswordField 在 encryptor 不为 nil 时把明文包成 envelope；否则原样返回。
// 空字符串始终原样返回（Phase 17 行为：空凭据 = 无凭据）。
func (s *InputConfigService) encryptPasswordField(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	if s.encryptor == nil {
		return plaintext, nil
	}
	return s.encryptor.Encrypt(plaintext)
}

// decryptPasswordField 是 encryptPasswordField 的反向操作。
func (s *InputConfigService) decryptPasswordField(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if s.encryptor == nil {
		return value, nil
	}
	return s.encryptor.Decrypt(value)
}

// InputConfigListResponse 输入配置列表响应
type InputConfigListResponse struct {
	Total int64                `json:"total"`
	Items []models.InputConfig `json:"items"`
}

// ListConfigsRequest 查询输入配置列表请求
type ListConfigsRequest struct {
	Page       int    `form:"page" binding:"required,min=1"`
	PageSize   int    `form:"page_size" binding:"required,min=1,max=100"`
	Keyword    string `form:"keyword"`
	IsActive   *bool  `form:"is_active"`
	ConfigType string `form:"config_type" binding:"omitempty,oneof=usb stream huawei_auto"`
}

// CreateInputConfigRequest 创建输入配置请求
type CreateInputConfigRequest struct {
	Name          string `json:"name" binding:"required,max=100"`
	Description   string `json:"description" binding:"max=500"`
	ConfigType    string `json:"config_type" binding:"required,oneof=usb stream"`
	HuaweiEnabled bool   `json:"huawei_enabled"`
	// 华为终端字段
	Server           string `json:"server" binding:"omitempty,max=100"`
	Port             int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username         string `json:"username" binding:"omitempty,max=50"`
	Password         string `json:"password" binding:"omitempty,max=100"`
	TerminalNumber   string `json:"terminal_number" binding:"omitempty,max=50"`
	ConferenceNumber string `json:"conference_number" binding:"omitempty,max=50"`
	// USB字段
	CameraBackend   string `json:"camera_backend" binding:"omitempty,max=20"`
	USBCameraName   string `json:"usb_camera_name" binding:"omitempty,max=100"`
	USBCameraDevice string `json:"usb_camera_device" binding:"omitempty,max=100"`
	AudioBackend    string `json:"audio_backend" binding:"omitempty,max=20"`
	USBAudioName    string `json:"usb_audio_name" binding:"omitempty,max=100"`
	USBAudioDevice  string `json:"usb_audio_device" binding:"omitempty,max=100"`
	// 流媒体字段
	StreamProtocol string `json:"stream_protocol" binding:"omitempty,oneof=rtmp rtsp srt hls"`
	StreamURL      string `json:"stream_url" binding:"omitempty,max=500"`
	StreamUsername string `json:"stream_username" binding:"omitempty,max=100"`
	StreamPassword string `json:"stream_password" binding:"omitempty,max=100"`
	StreamEnabled  bool   `json:"stream_enabled"`
	// 录制配置
	OutputFormat string `json:"output_format" binding:"omitempty,max=20"`
}

// UpdateInputConfigRequest 更新输入配置请求
type UpdateInputConfigRequest struct {
	Name          *string `json:"name" binding:"omitempty,max=100"`
	Description   *string `json:"description" binding:"omitempty,max=500"`
	HuaweiEnabled *bool   `json:"huawei_enabled"`
	// 华为终端字段
	Server           *string `json:"server" binding:"omitempty,max=100"`
	Port             *int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username         *string `json:"username" binding:"omitempty,max=50"`
	Password         *string `json:"password" binding:"omitempty,max=100"`
	TerminalNumber   *string `json:"terminal_number" binding:"omitempty,max=50"`
	ConferenceNumber *string `json:"conference_number" binding:"omitempty,max=50"`
	// USB字段
	CameraBackend   *string `json:"camera_backend" binding:"omitempty,max=20"`
	USBCameraName   *string `json:"usb_camera_name" binding:"omitempty,max=100"`
	USBCameraDevice *string `json:"usb_camera_device" binding:"omitempty,max=100"`
	AudioBackend    *string `json:"audio_backend" binding:"omitempty,max=20"`
	USBAudioName    *string `json:"usb_audio_name" binding:"omitempty,max=100"`
	USBAudioDevice  *string `json:"usb_audio_device" binding:"omitempty,max=100"`
	// 流媒体字段
	StreamProtocol *string `json:"stream_protocol" binding:"omitempty,oneof=rtmp rtsp srt hls"`
	StreamURL      *string `json:"stream_url" binding:"omitempty,max=500"`
	StreamUsername *string `json:"stream_username" binding:"omitempty,max=100"`
	StreamPassword *string `json:"stream_password" binding:"omitempty,max=100"`
	StreamEnabled  *bool   `json:"stream_enabled"`
	// 录制配置
	OutputFormat *string `json:"output_format" binding:"omitempty,max=20"`
	IsActive     *bool   `json:"is_active"`
}

// TestConnectionRequest 测试连接请求
type TestConnectionRequest struct {
	ConfigType string `json:"config_type" binding:"required,oneof=usb stream"`
	// 华为字段
	Server         string `json:"server" binding:"omitempty,max=100"`
	Port           int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Username       string `json:"username" binding:"omitempty,max=50"`
	Password       string `json:"password" binding:"omitempty,max=100"`
	TerminalNumber string `json:"terminal_number" binding:"omitempty,max=50"`
	// USB字段
	USBCameraDevice string `json:"usb_camera_device" binding:"omitempty,max=100"`
	// 流媒体字段
	StreamProtocol string `json:"stream_protocol" binding:"required,oneof=rtmp rtsp srt hls"`
	StreamURL      string `json:"stream_url" binding:"required,max=500"`
	StreamUsername string `json:"stream_username" binding:"omitempty,max=100"`
	StreamPassword string `json:"stream_password" binding:"omitempty,max=100"`
}

// ListConfigs 获取输入配置列表
func (s *InputConfigService) ListConfigs(ctx context.Context, req *ListConfigsRequest) (*InputConfigListResponse, error) {
	var configs []models.InputConfig
	var total int64

	query := s.db.WithContext(ctx).Model(&models.InputConfig{}).Preload("VideoRecordingTasks")

	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ? OR config_type LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (req.Page - 1) * req.PageSize
	if err := query.
		Offset(offset).
		Limit(req.PageSize).
		Order("id ASC").
		Find(&configs).Error; err != nil {
		return nil, err
	}

	return &InputConfigListResponse{
		Total: total,
		Items: configs,
	}, nil
}

// GetConfigByID 根据ID获取输入配置
func (s *InputConfigService) GetConfigByID(ctx context.Context, id uint) (*models.InputConfig, error) {
	var config models.InputConfig
	if err := s.db.WithContext(ctx).Preload("VideoRecordingTasks").First(&config, id).Error; err != nil {
		return nil, err
	}
	// Phase 18: 解密凭据列后再返回（解密失败 → 阻断调用方；上层须知此 row 异常）
	if s.encryptor != nil {
		if pwd, err := s.decryptPasswordField(config.Password); err != nil {
			s.logger.Error("GetConfigByID 解密 password 失败",
				zap.Uint("config_id", config.ID), zap.Error(err), response.SentinelField(err))
			return nil, fmt.Errorf("解密 password 失败 (id=%d): %w", config.ID, err)
		} else {
			config.Password = pwd
		}
		if spwd, err := s.decryptPasswordField(config.StreamPassword); err != nil {
			s.logger.Error("GetConfigByID 解密 stream_password 失败",
				zap.Uint("config_id", config.ID), zap.Error(err), response.SentinelField(err))
			return nil, fmt.Errorf("解密 stream_password 失败 (id=%d): %w", config.ID, err)
		} else {
			config.StreamPassword = spwd
		}
	}
	return &config, nil
}

// CreateConfig 创建输入配置
func (s *InputConfigService) CreateConfig(ctx context.Context, req *CreateInputConfigRequest) (*models.InputConfig, error) {
	// Phase 18: 加密凭据列后再写库（仅非空值）
	password, err := s.encryptPasswordField(req.Password)
	if err != nil {
		return nil, fmt.Errorf("加密 password 失败: %w", err)
	}
	streamPassword, err := s.encryptPasswordField(req.StreamPassword)
	if err != nil {
		return nil, fmt.Errorf("加密 stream_password 失败: %w", err)
	}

	config := &models.InputConfig{
		Name:             req.Name,
		Description:      req.Description,
		ConfigType:       req.ConfigType,
		HuaweiEnabled:    req.HuaweiEnabled,
		Server:           req.Server,
		Port:             req.Port,
		Username:         req.Username,
		Password:         password,
		TerminalNumber:   req.TerminalNumber,
		ConferenceNumber: req.ConferenceNumber,
		CameraBackend:    req.CameraBackend,
		USBCameraName:    req.USBCameraName,
		USBCameraDevice:  req.USBCameraDevice,
		AudioBackend:     req.AudioBackend,
		USBAudioName:     req.USBAudioName,
		USBAudioDevice:   req.USBAudioDevice,
		StreamProtocol:   req.StreamProtocol,
		StreamURL:        req.StreamURL,
		StreamUsername:   req.StreamUsername,
		StreamPassword:   streamPassword,
		StreamEnabled:    req.StreamEnabled,
		OutputFormat:     req.OutputFormat,
		IsActive:         true,
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

	// 验证配置
	if err := s.ValidateConfig(config); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(config).Error; err != nil {
		return nil, err
	}

	s.logger.Info("输入配置已创建",
		zap.Uint("config_id", config.ID),
		zap.String("config_type", config.ConfigType),
	)

	return config, nil
}

// UpdateConfig 更新输入配置
// 返回 (oldConfig, newConfig, error): oldConfig 是 mutate 前的快照，newConfig 是 mutate 后的记录
// 调用方（handler）负责把这两个对象交给 audit.RecordChange 写审计表
func (s *InputConfigService) UpdateConfig(ctx context.Context, id uint, req *UpdateInputConfigRequest) (*models.InputConfig, *models.InputConfig, error) {
	var config models.InputConfig
	if err := s.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, nil, err
	}

	// ★ Snapshot OldData BEFORE mutation（用于审计 OldData 捕获）
	// 注意：oldConfig 仍存 ciphertext（因为审计场景下要让 OldData 展示密文形态）。
	// 若想给审计展示解密后的明文，需由调用方在 audit middleware 内 sanitize 后再序列化；
	// 这里保持纯明文 → ciphertext 的语义边界，audit Sanitizer 已按字段名脱敏 Password/StreamPassword。
	oldConfig := config

	// 更新非空字段
	if req.Name != nil {
		config.Name = *req.Name
	}
	if req.Description != nil {
		config.Description = *req.Description
	}
	if req.HuaweiEnabled != nil {
		config.HuaweiEnabled = *req.HuaweiEnabled
	}
	if req.Server != nil {
		config.Server = *req.Server
	}
	if req.Port != nil {
		config.Port = *req.Port
	}
	if req.Username != nil {
		config.Username = *req.Username
	}
	if req.Password != nil {
		// Phase 18: 加密（nil 表示"不改"；空字符串表示"清空"——也按清空处理）
		enc, eerr := s.encryptPasswordField(*req.Password)
		if eerr != nil {
			return nil, nil, fmt.Errorf("加密 password 失败: %w", eerr)
		}
		config.Password = enc
	}
	if req.TerminalNumber != nil {
		config.TerminalNumber = *req.TerminalNumber
	}
	if req.ConferenceNumber != nil {
		config.ConferenceNumber = *req.ConferenceNumber
	}
	if req.CameraBackend != nil {
		config.CameraBackend = *req.CameraBackend
	}
	if req.USBCameraName != nil {
		config.USBCameraName = *req.USBCameraName
	}
	if req.USBCameraDevice != nil {
		config.USBCameraDevice = *req.USBCameraDevice
	}
	if req.AudioBackend != nil {
		config.AudioBackend = *req.AudioBackend
	}
	if req.USBAudioName != nil {
		config.USBAudioName = *req.USBAudioName
	}
	if req.USBAudioDevice != nil {
		config.USBAudioDevice = *req.USBAudioDevice
	}
	if req.StreamProtocol != nil {
		config.StreamProtocol = *req.StreamProtocol
	}
	if req.StreamURL != nil {
		config.StreamURL = *req.StreamURL
	}
	if req.StreamUsername != nil {
		config.StreamUsername = *req.StreamUsername
	}
	if req.StreamPassword != nil {
		// Phase 18: 加密
		enc, eerr := s.encryptPasswordField(*req.StreamPassword)
		if eerr != nil {
			return nil, nil, fmt.Errorf("加密 stream_password 失败: %w: %w", apperrors.ErrInternal, eerr)
		}
		config.StreamPassword = enc
	}
	if req.StreamEnabled != nil {
		config.StreamEnabled = *req.StreamEnabled
	}
	if req.OutputFormat != nil {
		config.OutputFormat = *req.OutputFormat
	}
	if req.IsActive != nil {
		config.IsActive = *req.IsActive
	}

	// 验证配置
	if err := s.ValidateConfig(&config); err != nil {
		return nil, nil, err
	}

	if err := s.db.WithContext(ctx).Save(&config).Error; err != nil {
		return nil, nil, err
	}

	s.logger.Info("输入配置已更新",
		zap.Uint("config_id", config.ID),
		zap.String("config_type", config.ConfigType),
	)

	// 返回的 newConfig 是 DB 内的密文形态；调用方如需明文，自行调 GetConfigByID。
	return &oldConfig, &config, nil
}

// DeleteConfig 删除输入配置
// 返回 (oldConfig, error): oldConfig 是删除前的快照，handler 负责把 OldData 交给 audit.RecordChange
func (s *InputConfigService) DeleteConfig(ctx context.Context, id uint) (*models.InputConfig, error) {
	var config models.InputConfig
	if err := s.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, err
	}

	// ★ Snapshot OldData BEFORE delete（用于审计 OldData 捕获）
	oldConfig := config

	// 检查是否有任务正在使用
	var count int64
	s.db.WithContext(ctx).Table("task_input_configs").Where("input_config_id = ?", id).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("配置正在被任务使用，无法删除: %w", apperrors.ErrForbidden)
	}

	if err := s.db.WithContext(ctx).Delete(&models.InputConfig{}, id).Error; err != nil {
		return nil, err
	}

	s.logger.Info("输入配置已删除", zap.Uint("config_id", id))
	return &oldConfig, nil
}

// ValidateConfig 验证输入配置
func (s *InputConfigService) ValidateConfig(config *models.InputConfig) error {
	// 首先运行模型级验证
	if err := config.Validate(); err != nil {
		return err
	}

	// 额外的服务级验证
	switch config.ConfigType {
	case models.ConfigTypeUSB:
		if config.USBCameraDevice == "" {
			return fmt.Errorf("USB配置必须指定摄像头设备: %w", apperrors.ErrInvalidInput)
		}

	case models.ConfigTypeStream:
		if config.StreamURL == "" {
			return fmt.Errorf("流媒体配置必须指定流地址: %w", apperrors.ErrInvalidInput)
		}

	default:
		return fmt.Errorf("无效的配置类型: %s: %w", config.ConfigType, apperrors.ErrInvalidInput)
	}

	return nil
}

// validateHuaweiFields 验证华为终端必填字段
func (s *InputConfigService) validateHuaweiFields(config *models.InputConfig) error {
	if config.Server == "" {
		return fmt.Errorf("华为服务器地址不能为空: %w", apperrors.ErrInvalidInput)
	}
	if config.Username == "" {
		return fmt.Errorf("用户名不能为空: %w", apperrors.ErrInvalidInput)
	}
	if config.TerminalNumber == "" {
		return fmt.Errorf("终端号码不能为空: %w", apperrors.ErrInvalidInput)
	}
	return nil
}

// TestConnection 测试输入配置连接
func (s *InputConfigService) TestConnection(ctx context.Context, req *TestConnectionRequest) error {
	s.logger.Info("测试输入配置连接",
		zap.String("config_type", req.ConfigType),
	)

	switch req.ConfigType {
	case models.ConfigTypeUSB:
		return s.testUSBDevice(req)
	case models.ConfigTypeStream:
		return s.testStreamConnection(ctx, req)
	default:
		return fmt.Errorf("不支持的配置类型: %w", apperrors.ErrInvalidInput)
	}
}

// testUSBDevice 测试USB设备连接
func (s *InputConfigService) testUSBDevice(req *TestConnectionRequest) error {
	devices := s.usbScanner.ScanAllUSBDevices()

	deviceFound := false
	for _, camera := range devices["cameras"] {
		if camera.DeviceID == req.USBCameraDevice {
			deviceFound = true
			if camera.Status != "available" {
				return fmt.Errorf("USB设备不可用: %s: %w", camera.Status, apperrors.ErrInvalidInput)
			}
			break
		}
	}

	if !deviceFound {
		return fmt.Errorf("未找到指定的USB设备，请先扫描设备: %w", apperrors.ErrNotFound)
	}

	s.logger.Info("USB设备连接测试成功", zap.String("device", req.USBCameraDevice))
	return nil
}

// testStreamConnection 测试流媒体连接
func (s *InputConfigService) testStreamConnection(ctx context.Context, req *TestConnectionRequest) error {
	s.logger.Info("测试流媒体连接",
		zap.String("protocol", req.StreamProtocol),
		zap.String("url", req.StreamURL),
	)

	ffprobePath := s.config.FFmpeg.FFProbePath
	if ffprobePath == "" {
		ffprobePath = "ffprobe"
	}

	var inputArgs []string
	switch req.StreamProtocol {
	case "rtmp":
		inputArgs = []string{"-i", req.StreamURL}
	case "rtsp":
		inputArgs = []string{"-rtsp_transport", "tcp", "-i", req.StreamURL}
	case "srt":
		inputArgs = []string{"-i", fmt.Sprintf("srt://%s", req.StreamURL)}
	case "hls":
		inputArgs = []string{"-i", req.StreamURL}
	default:
		return fmt.Errorf("不支持的流媒体协议: %w", apperrors.ErrInvalidInput)
	}

	// PERF-003/BUG-005: ffprobe 超时 ctx 从请求 ctx 派生，使请求取消能中断探测。
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	args := []string{"-v", "error"}
	args = append(args, inputArgs...)
	args = append(args, "-f", "null", "-")

	cmd := exec.CommandContext(ctx, ffprobePath, args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("连接超时（15秒），请检查网络和流媒体地址是否正确: %w", apperrors.ErrServiceUnavailable)
	}

	if err != nil {
		s.logger.Error("FFprobe failed",
			zap.Error(err),
			zap.String("output", string(output)),
			response.SentinelField(err),
		)
		return fmt.Errorf("流媒体连接测试失败: %w: %w", apperrors.ErrInternal, err)
	}

	s.logger.Info("流媒体连接测试成功")
	return nil
}

// testHuaweiConnection 测试华为终端连接
func (s *InputConfigService) testHuaweiConnection(req *TestConnectionRequest) error {
	s.logger.Info("测试华为终端连接",
		zap.String("server", req.Server),
		zap.String("terminal", req.TerminalNumber),
	)

	// TODO: 实现实际的华为API连接测试
	// 这应该与现有的 HuaweiConferenceConnector 集成

	return fmt.Errorf("华为终端连接测试尚未实现: %w", apperrors.ErrServiceUnavailable)
}
