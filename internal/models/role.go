package models

import "gorm.io/gorm"

// Role 角色模型
type Role struct {
	Base
	Name        string       `gorm:"type:varchar(50);uniqueIndex;not null" json:"name"`
	Description string       `gorm:"type:text" json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions,omitempty"`
}

// 预定义角色
const (
	RoleAdmin        = "admin"         // 系统管理员
	RoleOperator     = "operator"      // 操作员
	RoleViewer       = "viewer"        // 查看者
	RoleAPIClient    = "api_client"    // API客户端
	RoleSharedViewer = "shared_viewer" // 共享查看者 (D-04)
)

// BeforeCreate GORM hook - 在创建前调用
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	// 确保角色名称唯一
	return nil
}

// TableName 指定表名
func (Role) TableName() string {
	return "roles"
}
