package models

import (
	"time"
)

// APIKeyUsageLog API密钥使用日志
type APIKeyUsageLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	APIKeyID  uint           `gorm:"not null;index" json:"api_key_id"`
	APIKey    *APIKey        `gorm:"foreignKey:APIKeyID" json:"api_key,omitempty"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	Method    string         `gorm:"type:varchar(10);not null" json:"method"`          // GET, POST, PUT, DELETE
	Path      string         `gorm:"type:varchar(500);not null" json:"path"`           // 请求路径
	StatusCode int           `gorm:"not null" json:"status_code"`                      // 响应状态码
	ClientIP  string         `gorm:"type:varchar(50)" json:"client_ip"`
	UserAgent string         `gorm:"type:varchar(500)" json:"user_agent"`
	Duration  int            `gorm:"default:0" json:"duration"`                        // 请求耗时(ms)
	Success   bool           `gorm:"default:true" json:"success"`                      // 是否成功
	CreatedAt time.Time      `json:"created_at"`
}

// TableName 指定表名
func (APIKeyUsageLog) TableName() string {
	return "api_key_usage_logs"
}
