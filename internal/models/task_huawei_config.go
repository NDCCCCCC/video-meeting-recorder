package models

import "time"

// TaskHuaweiConfig 任务与华为配置的关联表
// 支持一个任务关联多个华为配置（如：USB设备 + 流媒体）
type TaskHuaweiConfig struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	TaskID         uint      `gorm:"not null;index:idx_task_config;index:idx_task_huawei_config" json:"task_id"`
	HuaweiConfigID uint      `gorm:"not null;index:idx_task_config;index:idx_huawei_config" json:"huawei_config_id"`
	ConfigType     string    `gorm:"type:varchar(20);not null" json:"config_type"` // usb | stream
	CreatedAt      time.Time `json:"created_at"`

	Task         *VideoRecordingTask `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"-"`
	HuaweiConfig *HuaweiConfig       `gorm:"foreignKey:HuaweiConfigID;constraint:OnDelete:CASCADE" json:"-"`
}

func (TaskHuaweiConfig) TableName() string {
	return "task_huawei_configs"
}
