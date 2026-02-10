package models

import "time"

// UserNotificationSetting 用户通知配置
type UserNotificationSetting struct {
	ID     uint `gorm:"primarykey" json:"id"`
	UserID uint `gorm:"uniqueIndex:idx_user_notification_setting;not null" json:"user_id"`
	User   *User `gorm:"foreignKey:UserID" json:"user,omitempty"`

	// 渠道开关
	EmailEnabled    bool `gorm:"default:true" json:"email_enabled"`
	SMSEnabled      bool `gorm:"default:false" json:"sms_enabled"`
	DingTalkEnabled bool `gorm:"default:false" json:"dingtalk_enabled"`
	WeChatEnabled   bool `gorm:"default:false" json:"wechat_enabled"`
	FeiShuEnabled   bool `gorm:"default:false" json:"feishu_enabled"`

	// 通知类型偏好
	TaskEnabled     bool `gorm:"default:true" json:"task_enabled"`
	SystemEnabled   bool `gorm:"default:true" json:"system_enabled"`
	WarningEnabled  bool `gorm:"default:true" json:"warning_enabled"`
	ReminderEnabled bool `gorm:"default:true" json:"reminder_enabled"`
	ConferenceEnabled bool `gorm:"default:true" json:"conference_enabled"`

	// 免打扰时段
	EnableQuietHours bool   `gorm:"default:false" json:"enable_quiet_hours"`
	QuietHoursStart  string `gorm:"type:time;default:22:00:00" json:"quiet_hours_start"`
	QuietHoursEnd    string `gorm:"type:time;default:08:00:00" json:"quiet_hours_end"`

	// 频率限制（每小时最大通知数）
	MaxNotificationsPerHour int `gorm:"default:10" json:"max_notifications_per_hour"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Base
}

// TableName 指定表名
func (UserNotificationSetting) TableName() string {
	return "user_notification_settings"
}
