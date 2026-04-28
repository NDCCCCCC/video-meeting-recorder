package main

import (
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

func main() {
	server := "21.120.164.44:389"
	bindDN := "ninedrunk@pr.intra.cpic.com.cn"
	password := ")(PO09po"
	baseDN := "dc=pr,dc=intra,dc=cpic,dc=com,dc=cn"

	fmt.Printf("=== LDAP 端口 389 测试 ===\n")
	fmt.Printf("服务器: %s\n", server)
	fmt.Println()

	// 测试普通LDAP连接
	fmt.Println("--- 测试普通LDAP连接 (无TLS) ---")
	conn, err := ldap.Dial("tcp", server)
	if err != nil {
		fmt.Printf("❌ LDAP连接失败: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("✓ LDAP 连接成功")

	// 测试绑定
	err = conn.Bind(bindDN, password)
	if err != nil {
		fmt.Printf("❌ 绑定失败: %v\n", err)
		return
	}
	fmt.Println("✓ 用户绑定成功")

	// 测试搜索
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		10, 0, false,
		"(objectClass=user)",
		[]string{"dn", "cn", "sAMAccountName"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		fmt.Printf("❌ 搜索失败: %v\n", err)
		return
	}

	fmt.Printf("✓ 搜索成功，找到 %d 个条目\n", len(sr.Entries))

	if len(sr.Entries) > 0 {
		fmt.Println("\n示例用户:")
		for i, entry := range sr.Entries {
			if i >= 3 {
				break
			}
			fmt.Printf("  - DN: %s\n", entry.DN)
		}
		fmt.Println("\n=== 测试通过 ===")
	}
}
