package models

import (
	"errors"

	"gorm.io/gorm"
)

// Permission 权限模型
type Permission struct {
	Base
	Resource string  `gorm:"type:varchar(50);not null;index" json:"resource"`
	Action   string  `gorm:"type:varchar(50);not null;index" json:"action"`
	Roles    []Role `gorm:"many2many:role_permissions;" json:"roles,omitempty"`
}

// BeforeCreate GORM hook - 确保resource+action组合唯一
func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	var count int64
	if err := tx.Model(&Permission{}).
		Where("resource = ? AND action = ?", p.Resource, p.Action).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("permission with this resource and action already exists")
	}
	return nil
}

// TableName 指定表名
func (Permission) TableName() string {
	return "permissions"
}
