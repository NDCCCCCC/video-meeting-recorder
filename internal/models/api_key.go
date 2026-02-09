package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/common"
	"gorm.io/gorm"
)

// APIKey API密钥模型
type APIKey struct {
	Base
	Name       string     `gorm:"type:varchar(100);not null" json:"name"`
	Key        string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"key"`
	UserID     uint       `gorm:"not null;index" json:"user_id"`
	User       *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	IsActive   bool       `gorm:"default:true" json:"is_active"`
	Scopes     string     `gorm:"type:text" json:"scopes"` // JSON数组字符串
}

// 作用域定义
const (
	ScopeRead    = "read"
	ScopeWrite   = "write"
	ScopeExecute = "execute"
	ScopeAdmin   = "admin"
)

// BeforeCreate GORM hook - 生成API密钥
func (a *APIKey) BeforeCreate(tx *gorm.DB) error {
	if a.Key == "" {
		a.Key = a.GenerateKey()
	}
	return nil
}

// IsExpired 检查是否过期
func (a *APIKey) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// HasScope 检查是否有指定作用域
func (a *APIKey) HasScope(scope string) bool {
	// TODO: 解析Scopes JSON字符串并检查
	return true
}

// GenerateKey 生成新的API密钥
func (a *APIKey) GenerateKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// 如果加密随机数生成失败，使用时间戳作为后备
		return hex.EncodeToString([]byte(time.Now().String())) + hex.EncodeToString(b)
	}
	return "rec_" + hex.EncodeToString(b)
}

// Validate 验证API密钥
func (a *APIKey) Validate() error {
	if a.Name == "" {
		return common.ErrInvalidInput
	}
	if a.UserID == 0 {
		return common.ErrInvalidInput
	}
	if a.IsExpired() {
		return common.ErrInvalidInput
	}
	if !a.IsActive {
		return common.ErrUnauthorized
	}
	return nil
}

// TableName 指定表名
func (APIKey) TableName() string {
	return "api_keys"
}
