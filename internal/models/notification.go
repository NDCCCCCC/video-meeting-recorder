package models

import (
	"encoding/json"
	"time"

	"go.uber.org/zap"
)

// NotificationMessage 通知消息
type NotificationMessage struct {
	ID uint `gorm:"primarykey" json:"id"`

	// 接收者信息
	UserID   uint   `gorm:"index;not null" json:"user_id"`
	Username string `gorm:"index" json:"username,omitempty"`

	// 消息内容
	Type    NotificationType `gorm:"type:varchar(20);index;not null" json:"type"`
	Title   string           `gorm:"type:varchar(200);not null" json:"title"`
	Content string           `gorm:"type:text" json:"content"`
	Data    string           `gorm:"type:text" json:"data,omitempty"`

	// 关联信息
	RelatedID   *uint  `json:"related_id,omitempty"`
	RelatedType string `gorm:"type:varchar(50)" json:"related_type,omitempty"`
	RelatedURL  string `gorm:"type:varchar(500)" json:"related_url,omitempty"`

	// 状态
	IsRead bool       `gorm:"index;default:false" json:"is_read"`
	ReadAt *time.Time `json:"read_at,omitempty"`

	// 渠道状态
	ChannelStatus string `gorm:"type:text" json:"channel_status,omitempty"` // JSON: {"email":"sent","sms":"failed"}

	// 时间
	CreatedAt time.Time  `gorm:"index" json:"created_at"`
	ExpiredAt *time.Time `json:"expired_at,omitempty"`

	Base
}

// TableName 指定表名
func (NotificationMessage) TableName() string {
	return "notification_messages"
}

// NotificationType 通知类型
type NotificationType string

const (
	TypeSystem     NotificationType = "system"     // 系统通知
	TypeTask       NotificationType = "task"       // 任务通知
	TypeConference NotificationType = "conference" // 会议通知
	TypeWarning    NotificationType = "warning"    // 警告通知
	TypeReminder   NotificationType = "reminder"   // 提醒通知
)

// NotificationChannel 通知渠道
type NotificationChannel string

const (
	ChannelSystem   NotificationChannel = "system"   // 站内消息
	ChannelEmail    NotificationChannel = "email"    // 邮件
	ChannelSMS      NotificationChannel = "sms"      // 短信
	ChannelWebhook  NotificationChannel = "webhook"  // Webhook
	ChannelDingTalk NotificationChannel = "dingtalk" // 钉钉
	ChannelWeChat   NotificationChannel = "wechat"   // 企业微信
	ChannelFeiShu   NotificationChannel = "feishu"   // 飞书
)

// ChannelStatus 渠道状态
type ChannelStatus string

const (
	ChannelStatusPending ChannelStatus = "pending"
	ChannelStatusSent    ChannelStatus = "sent"
	ChannelStatusFailed  ChannelStatus = "failed"
)

// GetChannelStatusMap 获取渠道状态映射
func (n *NotificationMessage) GetChannelStatusMap() map[string]ChannelStatus {
	if n.ChannelStatus == "" {
		return make(map[string]ChannelStatus)
	}
	var statusMap map[string]ChannelStatus
	if err := json.Unmarshal([]byte(n.ChannelStatus), &statusMap); err != nil {
		zap.L().Warn("通知渠道状态解析失败", zap.Error(err))
		return make(map[string]ChannelStatus)
	}
	return statusMap
}

// SetChannelStatusMap 设置渠道状态
func (n *NotificationMessage) SetChannelStatusMap(statusMap map[string]ChannelStatus) {
	data, _ := json.Marshal(statusMap)
	n.ChannelStatus = string(data)
}

// NotificationData 通知数据
type NotificationData struct {
	// 扩展数据，用于模板渲染
	Params map[string]interface{} `json:"params,omitempty"`
}

// GetData 获取扩展数据
func (n *NotificationMessage) GetData() *NotificationData {
	if n.Data == "" {
		return &NotificationData{}
	}
	var data NotificationData
	if err := json.Unmarshal([]byte(n.Data), &data); err != nil {
		zap.L().Warn("通知数据解析失败", zap.Error(err))
		return &NotificationData{}
	}
	return &data
}

// SetData 设置扩展数据
func (n *NotificationMessage) SetData(data *NotificationData) {
	jsonData, _ := json.Marshal(data)
	n.Data = string(jsonData)
}
