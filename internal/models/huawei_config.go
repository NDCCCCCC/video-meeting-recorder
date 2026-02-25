package models

import (
	"fmt"
	"time"
)

// HuaweiConfig 华为配置模型
// 包含华为终端连接配置和USB设备配置
// RTSP流作为独立输入源，配置在VideoRecordingTask中
type HuaweiConfig struct {
	Base
	Name             string `gorm:"type:varchar(100);not null" json:"name"`
	Description      string `gorm:"type:text" json:"description"`
	Server           string `gorm:"type:varchar(100);not null" json:"server"`         // 华为终端服务器地址
	Port             int    `gorm:"default:80" json:"port"`                           // 华为终端端口
	Username         string `gorm:"type:varchar(50);not null" json:"username"`        // 登录用户名
	Password         string `gorm:"type:varchar(100);not null" json:"-"`              // 登录密码（不输出到JSON）
	TerminalNumber   string `gorm:"type:varchar(50);not null" json:"terminal_number"` // 终端号码
	ConferenceNumber string `gorm:"type:varchar(50)" json:"conference_number"`        // 会议号

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

	IsActive bool       `gorm:"default:true" json:"is_active"`
	IsLocked bool       `gorm:"default:false" json:"is_locked"` // 终端锁定标志
	LockedBy *uint      `json:"locked_by,omitempty"`            // 锁定者任务ID
	LockedAt *time.Time `json:"locked_at,omitempty"`

	VideoRecordingTasks []VideoRecordingTask `gorm:"foreignKey:HuaweiConfigID" json:"video_recording_tasks,omitempty"`
}

// 设备绑定状态常量
const (
	DeviceStatusUnbound = "unbound"
	DeviceStatusBinding = "binding"
	DeviceStatusBound   = "bound"
	DeviceStatusError   = "error"
)

// IsCameraBound 检查摄像头是否已绑定
func (h *HuaweiConfig) IsCameraBound() bool {
	return h.CameraBindingStatus == DeviceStatusBound
}

// IsAudioBound 检查音频设备是否已绑定
func (h *HuaweiConfig) IsAudioBound() bool {
	return h.AudioBindingStatus == DeviceStatusBound
}

// Lock 锁定华为配置
func (h *HuaweiConfig) Lock(taskID uint) error {
	if h.IsLocked && h.LockedBy != nil && *h.LockedBy != taskID {
		return fmt.Errorf("配置已被其他任务锁定")
	}
	h.IsLocked = true
	now := time.Now()
	h.LockedBy = &taskID
	h.LockedAt = &now
	return nil
}

// Unlock 解锁华为配置
func (h *HuaweiConfig) Unlock() error {
	h.IsLocked = false
	h.LockedBy = nil
	h.LockedAt = nil
	return nil
}

// IsLockedByTask 检查是否被指定任务锁定
func (h *HuaweiConfig) IsLockedByTask(taskID uint) bool {
	return h.IsLocked && h.LockedBy != nil && *h.LockedBy == taskID
}

// Validate 验证华为配置
func (h *HuaweiConfig) Validate() error {
	var errs []string

	if h.Name == "" {
		errs = append(errs, "配置名称不能为空")
	}
	if h.Server == "" {
		errs = append(errs, "服务器地址不能为空")
	}
	if h.Username == "" {
		errs = append(errs, "用户名不能为空")
	}
	if h.Password == "" {
		errs = append(errs, "密码不能为空")
	}
	if h.TerminalNumber == "" {
		errs = append(errs, "终端号码不能为空")
	}

	if len(errs) > 0 {
		return fmt.Errorf("验证失败: %s", errs)
	}
	return nil
}

// TableName 指定表名
func (HuaweiConfig) TableName() string {
	return "huawei_configs"
}
