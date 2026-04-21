package models

import "time"

// UserRole 用户角色关联表（多对多）
type UserRole struct {
	UserID    uint      `gorm:"primaryKey;autoIncrement:false" json:"user_id"`
	RoleID    uint      `gorm:"primaryKey;autoIncrement:false" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (UserRole) TableName() string {
	return "users_roles"
}
