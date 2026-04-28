package migrations

import (
	"log"

	"gorm.io/gorm"
)

// DropLegacyRoleIDMigration 清理多角色迁移后遗留的 role_id 字段约束
type DropLegacyRoleIDMigration struct{}

func (m *DropLegacyRoleIDMigration) Name() string {
	return "012_drop_legacy_role_id"
}

func (m *DropLegacyRoleIDMigration) Up(db *gorm.DB) error {
	// 方案 A：保留遗留的 role_id 列
	// 此 migration 变为 no-op，仅验证 schema 状态
	//
	// 原因：User 模型使用 gorm:"-" 忽略 RoleID 字段
	// GORM AutoMigrate 不会尝试删除或修改被忽略的列
	//
	hasColumn, err := columnExists(db, "users", "role_id")
	if err != nil {
		return err
	}

	if !hasColumn {
		log.Println("INFO: No role_id column found in users table")
		return nil
	}

	log.Println("INFO: role_id column exists in users table (kept as legacy field)")
	log.Println("INFO: Migration 012 is now a no-op (legacy column preserved)")

	return nil
}

func (m *DropLegacyRoleIDMigration) Down(db *gorm.DB) error {
	// 回滚：无需操作
	log.Println("INFO: Rollback not needed - migration skipped column removal")
	return nil
}
