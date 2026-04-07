package models

import (
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
	Base
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	Email        string     `gorm:"type:varchar(100)" json:"email"`
	FullName     string     `gorm:"type:varchar(100)" json:"full_name"`
	RoleID       uint       `gorm:"not null" json:"role_id"`
	Role         *Role      `gorm:"foreignKey:RoleID" json:"role,omitempty"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	APIKeys      []APIKey   `gorm:"foreignKey:UserID" json:"api_keys,omitempty"`
}

// SetPassword 设置密码
func (u *User) SetPassword(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度至少为8位")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.PasswordHash = string(hash)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// HasPermission 检查用户权限
func (u *User) HasPermission(resource, action string) bool {
	if u.Role == nil {
		return false
	}

	// 管理员拥有所有权限
	if u.Role.Name == RoleAdmin {
		return true
	}

	// 检查角色权限
	for _, perm := range u.Role.Permissions {
		if perm.Resource == resource && (perm.Action == action || perm.Action == "*") {
			return true
		}
		// 检查通配符权限 (e.g., "conferences:*")
		if perm.Resource == resource || perm.Resource == "*" {
			if perm.Action == action || perm.Action == "*" {
				return true
			}
		}
	}

	return false
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
