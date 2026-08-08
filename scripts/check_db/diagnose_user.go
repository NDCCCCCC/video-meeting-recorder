//go:build diagnose

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "D:/CODE/ClaudeCode/record_V2/data/record.db?mode=rw")
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	fmt.Println("=== AD 用户角色诊断 ===")
	fmt.Println()

	// 1. 检查所有 AD 用户
	fmt.Println("1. 检查所有 AD 用户 (ad_guid IS NOT NULL):")
	fmt.Println("────────────────────────────────────────────────────────────────────────")
	rows, err := db.Query(`
		SELECT id, username, ad_guid, ad_username, is_active, created_at
		FROM users
		WHERE ad_guid IS NOT NULL
		ORDER BY id DESC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = rows.Close() }()

	hasADUsers := false
	for rows.Next() {
		hasADUsers = true
		var id uint
		var username, adGUID, adUsername string
		var isActive int
		var createdAt string
		err = rows.Scan(&id, &username, &adGUID, &adUsername, &isActive, &createdAt)
		if err != nil {
			log.Fatal(err)
		}
		activeStr := "ACTIVE"
		if isActive == 0 {
			activeStr = "INACTIVE"
		}
		fmt.Printf("  ID: %d, Username: %s, AD Username: %s, Status: %s\n", id, username, adUsername, activeStr)
	}
	if !hasADUsers {
		fmt.Println("  ⚠️  未找到 AD 用户")
	}
	fmt.Println()

	// 2. 检查 users_roles 表内容
	fmt.Println("2. 检查 users_roles 关联表:")
	fmt.Println("────────────────────────────────────────────────────────────────────────")
	rows2, err := db.Query(`
		SELECT ur.user_id, ur.role_id, u.username, r.name as role_name
		FROM users_roles ur
		JOIN users u ON ur.user_id = u.id
		JOIN roles r ON ur.role_id = r.id
		ORDER BY ur.user_id DESC
		LIMIT 20
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	hasRelations := false
	for rows2.Next() {
		hasRelations = true
		var userID, roleID uint
		var username, roleName string
		err = rows2.Scan(&userID, &roleID, &username, &roleName)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  User ID: %d (%s) -> Role ID: %d (%s)\n", userID, username, roleID, roleName)
	}
	if !hasRelations {
		fmt.Println("  ⚠️  users_roles 表为空")
	}
	fmt.Println()

	// 3. 检查 AD 用户的角色关联
	fmt.Println("3. 检查 AD 用户的具体角色关联:")
	fmt.Println("────────────────────────────────────────────────────────────────────────")
	rows3, err := db.Query(`
		SELECT u.id, u.username, u.ad_username,
		       COUNT(ur.role_id) as role_count,
		       GROUP_CONCAT(r.name, ', ') as role_names
		FROM users u
		LEFT JOIN users_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE u.ad_guid IS NOT NULL
		GROUP BY u.id
		ORDER BY u.id DESC
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows3.Close()

	hasADWithRoles := false
	for rows3.Next() {
		hasADWithRoles = true
		var id uint
		var username, adUsername string
		var roleCount int
		var roleNames sql.NullString
		err = rows3.Scan(&id, &username, &adUsername, &roleCount, &roleNames)
		if err != nil {
			log.Fatal(err)
		}

		status := "✅"
		if roleCount == 0 {
			status = "❌ 无角色"
		}
		roles := "(无角色)"
		if roleNames.Valid {
			roles = roleNames.String
		}
		fmt.Printf("  %s ID: %d, Username: %s, AD: %s, Roles: %s\n", status, id, username, adUsername, roles)
	}
	if !hasADWithRoles {
		fmt.Println("  ⚠️  未找到 AD 用户")
	}
	fmt.Println()

	// 4. 检查角色表
	fmt.Println("4. 检查可用角色:")
	fmt.Println("────────────────────────────────────────────────────────────────────────")
	rows4, err := db.Query("SELECT id, name, description FROM roles ORDER BY id")
	if err != nil {
		log.Fatal(err)
	}
	defer rows4.Close()

	for rows4.Next() {
		var id uint
		var name, description string
		err = rows4.Scan(&id, &name, &description)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  ID: %d, Name: %s, Description: %s\n", id, name, description)
	}
	fmt.Println()

	// 5. 诊断总结
	fmt.Println("5. 诊断总结:")
	fmt.Println("────────────────────────────────────────────────────────────────────────")

	// 检查是否有 AD 用户没有角色
	var orphanCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM users u
		LEFT JOIN users_roles ur ON u.id = ur.user_id
		WHERE u.ad_guid IS NOT NULL AND ur.role_id IS NULL
	`).Scan(&orphanCount)
	if err != nil {
		log.Fatal(err)
	}

	if orphanCount > 0 {
		fmt.Printf("  ❌ 发现 %d 个 AD 用户没有角色关联！\n", orphanCount)
		fmt.Println("  💡 这就是 sidebar 不显示的原因 - IsAdmin = false, Permissions = []")
	} else {
		var adUserCount int
		err = db.QueryRow("SELECT COUNT(*) FROM users WHERE ad_guid IS NOT NULL").Scan(&adUserCount)
		if err == nil && adUserCount > 0 {
			fmt.Println("  ✅ 所有 AD 用户都有角色关联")
			fmt.Println("  💡 问题可能在其他地方 - 需要检查登录响应中的 user 对象")
		} else {
			fmt.Println("  ⚠️  数据库中没有 AD 用户")
		}
	}
	fmt.Println()
}
