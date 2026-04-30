package models

import "time"

// Session 会话模型
type Session struct {
	Base
	Token       string     `gorm:"type:varchar(512);uniqueIndex;not null" json:"token"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	User        *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ExpiresAt   time.Time  `gorm:"index" json:"expires_at"`
	IPAddress   string     `gorm:"type:varchar(50)" json:"ip_address"`
	UserAgent   string     `gorm:"type:varchar(500)" json:"user_agent"`
	IsActive    bool       `gorm:"default:true;index" json:"is_active"`
	LastUsedAt  *time.Time `gorm:"index" json:"last_used_at,omitempty"` // Token 最后使用时间，用于宽限期机制
}

// IsExpired 检查会话是否过期
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Refresh 刷新会话有效期
func (s *Session) Refresh(duration time.Duration) error {
	s.ExpiresAt = time.Now().Add(duration)
	return nil
}

// Revoke 撤销会话
func (s *Session) Revoke() error {
	s.IsActive = false
	return nil
}

// TableName 指定表名
func (Session) TableName() string {
	return "sessions"
}
