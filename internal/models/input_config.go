package models

import (
	"errors"
	"fmt"
	"time"
)

// ConfigType 配置类型常量
const (
	ConfigTypeHuaweiAuto = "huawei_auto"
	ConfigTypeUSB        = "usb"
	ConfigTypeStream     = "stream"
)

// InputConfig 输入配置模型
// 重构自华为配置，支持多种录制输入源：华为终端控制、USB设备直录、流媒体录制
// 一个配置可以同时包含华为控制信息和录制源信息
type InputConfig struct {
	Base
	Name             string `gorm:"type:varchar(100);not null" json:"name"`
	Description      string `gorm:"type:text" json:"description"`
	ConfigType       string `gorm:"type:varchar(20);not null;index" json:"config_type"` // usb | stream
	HuaweiEnabled    bool   `gorm:"default:false" json:"huawei_enabled"`                  // 华为控制开关

	// 华为终端连接配置（当 huawei_enabled=true 时必填）
	Server           string `gorm:"type:varchar(100)" json:"server"`                      // 华为终端服务器地址
	Port             int    `gorm:"default:80" json:"port"`                               // 华为终端端口
	Username         string `gorm:"type:varchar(50)" json:"username"`                     // 登录用户名
	Password         string `gorm:"type:varchar(100)" json:"-"`                            // 登录密码（不输出到JSON）
	TerminalNumber   string `gorm:"type:varchar(50)" json:"terminal_number"`              // 终端号码
	ConferenceNumber string `gorm:"type:varchar(50)" json:"conference_number"`            // 会议号

	// USB摄像头后端配置
	CameraBackend string `gorm:"type:varchar(20);default:'dshow'" json:"camera_backend"` // dshow (Windows) | v4l2 (Linux) | avfoundation (macOS)

	// USB摄像头配置
	USBCameraName       string `gorm:"type:varchar(100)" json:"usb_camera_name"`
	USBCameraDevice     string `gorm:"type:varchar(100)" json:"usb_camera_device"`
	CameraBindingStatus string `gorm:"type:varchar(20);default:'unbound'" json:"camera_binding_status"` // unbound | binding | bound | error

	// USB音频后端配置
	AudioBackend string `gorm:"type:varchar(20);default:'dshow'" json:"audio_backend"` // dshow (Windows) | alsa (Linux) | coreaudio (macOS)

	// USB音频设备配置
	USBAudioName       string `gorm:"type:varchar(100)" json:"usb_audio_name"`
	USBAudioDevice     string `gorm:"type:varchar(100)" json:"usb_audio_device"`
	AudioBindingStatus string `gorm:"type:varchar(20);default:'unbound'" json:"audio_binding_status"`

	// 录制配置
	OutputFormat string `gorm:"type:varchar(20);default:'mp4'" json:"output_format"` // mp4, mkv, avi

	// 流媒体配置
	StreamProtocol string `gorm:"type:varchar(20)" json:"stream_protocol"` // rtmp, rtsp, srt, hls
	StreamURL      string `gorm:"type:varchar(500)" json:"stream_url"`
	StreamUsername string `gorm:"type:varchar(100)" json:"stream_username"`
	StreamPassword string `gorm:"type:varchar(100)" json:"-"` // 不输出到JSON
	StreamEnabled  bool   `gorm:"default:false" json:"stream_enabled"`

	IsActive bool       `gorm:"default:true" json:"is_active"`
	IsLocked bool       `gorm:"default:false" json:"is_locked"` // 终端锁定标志
	LockedBy *uint      `json:"locked_by,omitempty"`            // 锁定者任务ID
	LockedAt *time.Time `json:"locked_at,omitempty"`

	VideoRecordingTasks []VideoRecordingTask `gorm:"foreignKey:InputConfigID" json:"video_recording_tasks,omitempty"`
}

// IsCameraBound 检查摄像头是否已绑定
func (c *InputConfig) IsCameraBound() bool {
	return c.CameraBindingStatus == DeviceStatusBound
}

// IsAudioBound 检查音频设备是否已绑定
func (c *InputConfig) IsAudioBound() bool {
	return c.AudioBindingStatus == DeviceStatusBound
}

// Lock 锁定输入配置
func (c *InputConfig) Lock(taskID uint) error {
	if c.IsLocked && c.LockedBy != nil && *c.LockedBy != taskID {
		return fmt.Errorf("配置已被其他任务锁定")
	}
	c.IsLocked = true
	now := time.Now()
	c.LockedBy = &taskID
	c.LockedAt = &now
	return nil
}

// Unlock 解锁输入配置
func (c *InputConfig) Unlock() error {
	c.IsLocked = false
	c.LockedBy = nil
	c.LockedAt = nil
	return nil
}

// IsLockedByTask 检查是否被指定任务锁定
func (c *InputConfig) IsLockedByTask(taskID uint) bool {
	return c.IsLocked && c.LockedBy != nil && *c.LockedBy == taskID
}

// Validate 验证输入配置
func (c *InputConfig) Validate() error {
	if c.Name == "" {
		return errors.New("配置名称不能为空")
	}

	// 如果启用华为控制，验证华为字段
	if c.HuaweiEnabled {
		if c.Server == "" {
			return errors.New("华为服务器地址不能为空")
		}
		if c.Username == "" {
			return errors.New("用户名不能为空")
		}
		if c.TerminalNumber == "" {
			return errors.New("终端号码不能为空")
		}
	}

	// 根据配置类型验证录制源字段
	switch c.ConfigType {
	case ConfigTypeHuaweiAuto:
		// 华为自动模式：如果未启用华为控制，必须有USB或流媒体录制源
		if !c.HuaweiEnabled {
			hasUSB := c.USBCameraDevice != ""
			hasStream := c.StreamURL != ""
			if !hasUSB && !hasStream {
				return errors.New("华为控制关闭时，必须选择USB或流媒体录制源")
			}
		}

	case ConfigTypeUSB:
		if c.USBCameraDevice == "" {
			return errors.New("USB配置必须指定摄像头设备")
		}

	case ConfigTypeStream:
		if c.StreamURL == "" {
			return errors.New("流媒体配置必须指定流地址")
		}

	default:
		return fmt.Errorf("无效的配置类型: %s", c.ConfigType)
	}

	return nil
}

// TableName 指定表名
func (InputConfig) TableName() string {
	return "input_configs"
}
