package models

import (
	"encoding/json"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User 用户模型
type User struct {
	Base
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash *string    `gorm:"type:varchar(255)" json:"-"` // nullable for AD users
	Email        string     `gorm:"type:varchar(100)" json:"email"`
	FullName     string     `gorm:"type:varchar(100)" json:"full_name"`
	Roles        []Role     `gorm:"many2many:users_roles;" json:"roles,omitempty"`
	AllowedIPs   string     `gorm:"type:text" json:"allowed_ips"` // IP限制列表 (JSON数组)
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	APIKeys      []APIKey   `gorm:"foreignKey:UserID" json:"api_keys,omitempty"`

	// AD fields (nullable for local users, per D-21, D-22, D-23)
	ADUsername   string     `gorm:"type:varchar(100)" json:"ad_username"`
	ADDN         string     `gorm:"type:varchar(255)" json:"ad_dn"`
	ADGUID       string     `gorm:"type:char(36);index" json:"ad_guid"`
	ADDepartment string     `gorm:"type:varchar(100)" json:"ad_department"`
	ADUPN        string     `gorm:"type:varchar(200)" json:"ad_upn"`
	LastADLogin  *time.Time `json:"last_ad_login"`
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

	hashStr := string(hash)
	u.PasswordHash = &hashStr
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	if u.PasswordHash == nil {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password))
	return err == nil
}

// HasPermission 检查用户权限
func (u *User) HasPermission(resource, action string) bool {
	if len(u.Roles) == 0 {
		return false
	}

	// 检查所有角色 (OR logic per D-07)
	for _, role := range u.Roles {
		// 管理员拥有所有权限
		if role.Name == RoleAdmin {
			return true
		}

		// 检查该角色的权限
		for _, perm := range role.Permissions {
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
	}

	return false
}

// HasRole 检查用户是否拥有特定角色
func (u *User) HasRole(roleName string) bool {
	for _, role := range u.Roles {
		if role.Name == roleName {
			return true
		}
	}
	return false
}

// GetAllowedIPs 获取IP限制列表
func (u *User) GetAllowedIPs() []string {
	if u.AllowedIPs == "" {
		return []string{}
	}
	var ips []string
	_ = json.Unmarshal([]byte(u.AllowedIPs), &ips)
	return ips
}

// SetAllowedIPs 设置IP限制列表
func (u *User) SetAllowedIPs(ips []string) error {
	data, err := json.Marshal(ips)
	if err != nil {
		return err
	}
	u.AllowedIPs = string(data)
	return nil
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}
