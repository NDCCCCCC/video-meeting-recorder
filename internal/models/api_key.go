package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"go.uber.org/zap"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"

	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/common"
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
		// STYLE-006: GenerateKey 返回 error 替代 panic；modern OS 几乎不会失败
		// 但模型层保留错误路径，避免进程级 panic
		key, err := generateAPIKey()
		if err != nil {
			return fmt.Errorf("生成API密钥失败: %w: %w", apperrors.ErrInternal, err)
		}
		a.Key = key
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
// Deprecated: 此方法保留仅为向后兼容；推荐使用 generateAPIKey() 配合 BeforeCreate hook。
// 新代码应在 BeforeCreate hook 中调用 generateAPIKey 并处理 error。
func (a *APIKey) GenerateKey() string {
	key, err := generateAPIKey()
	if err != nil {
		// 旧路径保持 panic 行为以避免静默退化（与 gin.Recovery 兜底配合）
		// 新代码应使用 BeforeCreate 路径并将错误返回 GORM
		panic("failed to generate random bytes for API key: " + err.Error())
	}
	return key
}

// generateAPIKey 包级纯函数，返回随机生成的 API key 字符串与错误。
// 分离该函数便于 BeforeCreate hook 与需要错误返回的调用方使用。
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("crypto/rand.Read 失败: %w: %w", apperrors.ErrInternal, err)
	}
	return "rec_" + hex.EncodeToString(b), nil
}

// GetScopeList 获取作用域列表
func (a *APIKey) GetScopeList() []string {
	if a.Scopes == "" {
		return []string{}
	}
	var scopes []string
	if err := json.Unmarshal([]byte(a.Scopes), &scopes); err != nil {
		zap.L().Warn("JSON 字段解析失败", zap.String("field", "Scopes"), zap.Error(err))
		return nil
	}
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
	if err := json.Unmarshal([]byte(a.IPWhitelist), &whitelist); err != nil {
		zap.L().Warn("JSON 字段解析失败", zap.String("field", "IPWhitelist"), zap.Error(err))
		return nil
	}
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
	if whitelist == nil && a.IPWhitelist != "" {
		return false
	}
	if len(whitelist) == 0 {
		return true
	}
	// BUG-016: 归一化 IPv6-mapped IPv4（如 ::ffff:192.0.2.1 等价 192.0.2.1）
	parsed := net.ParseIP(clientIP)
	ip := normalizeIP(parsed)
	for _, allowed := range whitelist {
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

// normalizeIP 将 IPv6-mapped IPv4 转换为 IPv4（如 ::ffff:192.0.2.1 → 192.0.2.1），
// 使 net.IP.To4() 兼容 RFC 4291 客户端双栈字符串（BUG-016）。
func normalizeIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
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
