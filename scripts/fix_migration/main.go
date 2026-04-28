package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	// 打开数据库（使用相对于项目根目录的路径）
	dbPath := "../../data/record.db"
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法打开数据库: %v", err)
	}

	fmt.Println("=== 清理 Migration 012 失败状态 ===")

	// Step 1: 禁用外键约束
	db.Exec("PRAGMA foreign_keys = OFF")
	fmt.Println("✓ 已禁用外键约束")

	// Step 2: 检查并删除 users_new 表（如果存在）
	var usersNewCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users_new'").Scan(&usersNewCount)
	if usersNewCount > 0 {
		db.Exec("DROP TABLE IF EXISTS users_new")
		fmt.Println("✓ 已删除部分创建的 users_new 表")
	} else {
		fmt.Println("  没有找到 users_new 表")
	}

	// Step 3: 检查 users 表是否仍然存在
	var usersCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users'").Scan(&usersCount)
	if usersCount == 0 {
		fmt.Println("⚠ 警告: users 表不存在！数据库可能已损坏")
	} else {
		fmt.Println("✓ users 表存在")
	}

	// Step 4: 重新启用外键约束
	db.Exec("PRAGMA foreign_keys = ON")
	fmt.Println("✓ 已恢复外键约束")

	fmt.Println("\n=== 清理完成 ===")
	fmt.Println("现在可以重新启动服务运行 migration 012")
	fmt.Println("\n运行命令:")
	fmt.Println("  cd D:\\\\CODE\\\\ClaudeCode\\\\record_V2")
	fmt.Println("  go run ./cmd/server")
}
