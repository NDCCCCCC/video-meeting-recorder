package services

import (
	"errors"
	"fmt"

	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// HuaweiConfigService 华为配置服务
type HuaweiConfigService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewHuaweiConfigService 创建华为配置服务
func NewHuaweiConfigService(db *gorm.DB, logger *zap.Logger) *HuaweiConfigService {
	return &HuaweiConfigService{
		db:     db,
		logger: logger,
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
	Total int64              `json:"total"`
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
	CameraBackend    string `json:"camera_backend" binding:"omitempty,max=20"`    // dshow | v4l2 | avfoundation
	AudioBackend     string `json:"audio_backend" binding:"omitempty,max=20"`     // dshow | alsa | coreaudio | wasapi
	USBCameraName    string `json:"usb_camera_name" binding:"omitempty,max=100"`
	USBCameraDevice  string `json:"usb_camera_device" binding:"omitempty,max=100"`
	USBAudioName     string `json:"usb_audio_name" binding:"omitempty,max=100"`
	USBAudioDevice   string `json:"usb_audio_device" binding:"omitempty,max=100"`
	OutputFormat     string `json:"output_format" binding:"omitempty,max=20"`
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
		USBAudioDevice:   req.USBAudioDevice,
		OutputFormat:     req.OutputFormat,
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
		"dshow":      true,
		"alsa":       true,
		"coreaudio":  true,
		"wasapi":     true,
	}
	validOutputFormats := map[string]bool{
		"mp4":  true,
		"mkv":  true,
		"avi":  true,
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
