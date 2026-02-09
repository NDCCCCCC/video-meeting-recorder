package models

import (
	"fmt"
	"time"
)

// HuaweiConfig 华为配置模型
type HuaweiConfig struct {
	Base
	Name             string `gorm:"type:varchar(100);not null" json:"name"`
	Description      string `gorm:"type:text" json:"description"`
	Server           string `gorm:"type:varchar(100);not null" json:"server"`
	Port             int    `gorm:"default:80" json:"port"`
	Username         string `gorm:"type:varchar(50);not null" json:"username"`
	Password         string `gorm:"type:varchar(100);not null" json:"-"`
	TerminalNumber   string `gorm:"type:varchar(50);not null" json:"terminal_number"`
	ConferenceNumber string `gorm:"type:varchar(50)" json:"conference_number"`

	// USB摄像头配置
	USBCameraName       string `gorm:"type:varchar(100)" json:"usb_camera_name"`
	USBCameraDevice     string `gorm:"type:varchar(100)" json:"usb_camera_device"`
	USBCameraPath       string `gorm:"type:varchar(200)" json:"usb_camera_path"`
	CameraBindingStatus string `gorm:"type:varchar(20);default:'unbound'" json:"camera_binding_status"` // unbound | binding | bound | error

	// USB音频设备配置
	USBAudioDevice     string `gorm:"type:varchar(100)" json:"usb_audio_device"`
	USBAudioPath       string `gorm:"type:varchar(200)" json:"usb_audio_path"`
	AudioBindingStatus  string `gorm:"type:varchar(20);default:'unbound'" json:"audio_binding_status"`

	// 录制配置
	RecordDirectory string `gorm:"type:varchar(200)" json:"record_directory"`
	OutputFormat    string `gorm:"type:varchar(20);default:'mp4'" json:"output_format"` // mp4, mkv, avi

	IsActive    bool       `gorm:"default:true" json:"is_active"`
	IsLocked    bool       `gorm:"default:false" json:"is_locked"` // 终端锁定标志
	LockedBy    *uint      `json:"locked_by,omitempty"`            // 锁定者任务ID
	LockedAt    *time.Time `json:"locked_at,omitempty"`

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

// GetRTSPURL 获取RTSP URL
func (h *HuaweiConfig) GetRTSPURL() string {
	scheme := "rtsp"
	if h.Port == 443 {
		scheme = "rtsps"
	}
	return fmt.Sprintf("%s://%s:%d@%s:%d/stream", scheme, h.Username, h.Password, h.Server, h.Port)
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
