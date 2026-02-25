package models

// Permission 权限模型
type Permission struct {
	Base
	Resource    string `gorm:"type:varchar(50);not null;index" json:"resource"`
	Action      string `gorm:"type:varchar(50);not null;index" json:"action"`
	Description string `gorm:"type:varchar(100)" json:"description"`
	Roles       []Role `gorm:"many2many:role_permissions;" json:"roles,omitempty"`
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}
