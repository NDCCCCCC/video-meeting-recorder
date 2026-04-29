package models

import "time"

// TaskInputConfig 任务与输入配置的关联表
// 支持一个任务关联多个输入配置（如：USB设备 + 流媒体）
type TaskInputConfig struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	TaskID        uint      `gorm:"not null;index:idx_task_input_config;uniqueIndex:idx_task_input_config_unique" json:"task_id"`
	InputConfigID uint      `gorm:"not null;index:idx_input_config;uniqueIndex:idx_task_input_config_unique" json:"input_config_id"`
	ConfigType    string    `gorm:"type:varchar(20);not null" json:"config_type"` // huawei_auto | usb | stream
	CreatedAt     time.Time `json:"created_at"`

	Task         *VideoRecordingTask `gorm:"foreignKey:TaskID;constraint:OnDelete:CASCADE" json:"-"`
	InputConfig  *InputConfig        `gorm:"foreignKey:InputConfigID;constraint:OnDelete:CASCADE" json:"-"`
}

// TableName 指定表名
func (TaskInputConfig) TableName() string {
	return "task_input_configs"
}
