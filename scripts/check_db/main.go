package main

import (
	"fmt"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("../../data/record.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法打开数据库: %v", err)
	}

	// 检查 users 表结构
	fmt.Println("=== 检查 users 表结构 ===")
	var sql string
	db.Raw("SELECT sql FROM sqlite_master WHERE type='table' AND name='users'").Scan(&sql)
	fmt.Println(sql)

	// 检查 users_new 表是否存在
	var usersNewCount int64
	db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='users_new'").Scan(&usersNewCount)
	if usersNewCount > 0 {
		fmt.Println("\n⚠ users_new 表仍然存在（迁移失败残留）")
		fmt.Println("正在删除...")
		db.Exec("PRAGMA foreign_keys = OFF")
		db.Exec("DROP TABLE IF EXISTS users_new")
		db.Exec("PRAGMA foreign_keys = ON")
		fmt.Println("✓ 已删除 users_new 表")
	}

	// 检查是否有 password_hash 为 NULL 的用户
	fmt.Println("\n=== 检查用户数据 ===")
	var nullPasswordCount int64
	db.Raw("SELECT COUNT(*) FROM users WHERE password_hash IS NULL").Scan(&nullPasswordCount)
	if nullPasswordCount > 0 {
		fmt.Printf("⚠ 发现 %d 个用户的 password_hash 为 NULL\n", nullPasswordCount)
	} else {
		fmt.Println("✓ 所有用户都有 password_hash")
	}

	// 检查用户数量
	var userCount int64
	db.Raw("SELECT COUNT(*) FROM users").Scan(&userCount)
	fmt.Printf("用户总数: %d\n", userCount)
}
