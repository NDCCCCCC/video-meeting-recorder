package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"strings"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/common"
	"gorm.io/gorm"
)

// APIKey API密钥模型
type APIKey struct {
	Base
	Name         string     `gorm:"type:varchar(100);not null" json:"name"`
	Key          string     `gorm:"type:varchar(68);uniqueIndex;not null" json:"key"` // rec_前缀 + 64位hex
	UserID       uint       `gorm:"not null;index" json:"user_id"`
	User         *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	Scopes       string     `gorm:"type:text" json:"scopes"`              // JSON数组字符串
	IPWhitelist  string     `gorm:"type:text" json:"ip_whitelist"`        // JSON数组字符串
	Description  string     `gorm:"type:varchar(500)" json:"description"` // 描述
	InheritPerms bool       `gorm:"default:true" json:"inherit_perms"`    // 是否继承用户权限
}

// 作用域定义
const (
	ScopeRead  = "read"
	ScopeWrite = "write"
	ScopeAdmin = "admin"
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
func (a *APIKey) HasScope(requiredScope string) bool {
	// 如果继承用户权限，则检查通过
	if a.InheritPerms {
		return true
	}
	scopes := a.GetScopeList()
	for _, scope := range scopes {
		if scope == ScopeAdmin {
			return true
		}
		if scope == ScopeWrite && (requiredScope == ScopeRead || requiredScope == ScopeWrite) {
			return true
		}
		if scope == requiredScope {
			return true
		}
	}
	return false
}

// GenerateKey 生成新的API密钥
func (a *APIKey) GenerateKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// 如果加密随机数生成失败，记录错误并panic
		// 不使用不安全的时间戳后备方案
		panic("failed to generate random bytes for API key: " + err.Error())
	}
	return "rec_" + hex.EncodeToString(b)
}

// GetScopeList 获取作用域列表
func (a *APIKey) GetScopeList() []string {
	if a.Scopes == "" {
		return []string{}
	}
	var scopes []string
	_ = json.Unmarshal([]byte(a.Scopes), &scopes)
	return scopes
}

// SetScopes 设置作用域列表
func (a *APIKey) SetScopes(scopes []string) error {
	data, err := json.Marshal(scopes)
	if err != nil {
		return err
	}
	a.Scopes = string(data)
	return nil
}

// GetIPWhitelist 获取IP白名单列表
func (a *APIKey) GetIPWhitelist() []string {
	if a.IPWhitelist == "" {
		return []string{}
	}
	var whitelist []string
	_ = json.Unmarshal([]byte(a.IPWhitelist), &whitelist)
	return whitelist
}

// SetIPWhitelist 设置IP白名单列表
func (a *APIKey) SetIPWhitelist(whitelist []string) error {
	data, err := json.Marshal(whitelist)
	if err != nil {
		return err
	}
	a.IPWhitelist = string(data)
	return nil
}

// IsIPAllowed 检查IP是否在白名单中
func (a *APIKey) IsIPAllowed(clientIP string) bool {
	whitelist := a.GetIPWhitelist()
	if len(whitelist) == 0 {
		return true
	}
	ip := net.ParseIP(clientIP)
	for _, allowed := range whitelist {
		// 磁化显示密钥，仅保留前8位
		if allowed == clientIP {
			return true
		}
		// CIDR匹配
		if strings.Contains(allowed, "/") {
			_, ipNet, err := net.ParseCIDR(allowed)
			if err != nil {
				continue
			}
			if ip != nil && ipNet.Contains(ip) {
				return true
			}
		}
	}
	return false
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
