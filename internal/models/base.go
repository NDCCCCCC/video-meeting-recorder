package models

import (
	"time"

	"gorm.io/gorm"
)

// Base 基础模型
type Base struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate GORM hook - 在创建前调用
func (b *Base) BeforeCreate(tx *gorm.DB) error {
	return nil
}
