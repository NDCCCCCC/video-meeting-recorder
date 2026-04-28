package main

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

func main() {
	// AD 服务器配置
	server := "[REMOVED-INTERNAL-HOST]:636"
	bindDN := "[REMOVED-ACCOUNT]@[REMOVED-INTERNAL-DOMAIN]"
	password := "[REMOVED-CREDENTIAL]"
	baseDN := "[REMOVED-INTERNAL-BASE-DN]"

	fmt.Printf("=== LDAP/LDAPS 连接测试 ===\n")
	fmt.Printf("服务器: %s\n", server)
	fmt.Printf("BindDN: %s\n", bindDN)
	fmt.Printf("BaseDN: %s\n", baseDN)
	fmt.Println()

	// 测试 LDAPS (端口 636)
	fmt.Println("测试 LDAPS 连接...")
	tlsConfig := &tls.Config{
		ServerName:         "[REMOVED-INTERNAL-HOST]",
		InsecureSkipVerify: true, // 跳过证书验证用于测试
		MinVersion:         tls.VersionTLS12,
	}

	conn, err := ldap.DialTLS("tcp", server, tlsConfig)
	if err != nil {
		log.Fatalf("❌ LDAPS 连接失败: %v", err)
	}
	defer conn.Close()

	fmt.Println("✓ LDAPS 连接成功")

	// 测试绑定
	fmt.Println("\n测试用户绑定...")
	err = conn.Bind(bindDN, password)
	if err != nil {
		log.Fatalf("❌ 绑定失败: %v", err)
	}
	fmt.Println("✓ 用户绑定成功")

	// 测试搜索
	fmt.Println("\n测试搜索操作...")
	searchRequest := ldap.NewSearchRequest(
		baseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		10, 0, false,
		"(objectClass=user)",
		[]string{"dn", "cn", "sAMAccountName", "mail", "userPrincipalName"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		log.Fatalf("❌ 搜索失败: %v", err)
	}

	fmt.Printf("✓ 搜索成功，找到 %d 个条目\n\n", len(sr.Entries))

	// 显示前几个结果
	if len(sr.Entries) > 0 {
		fmt.Println("示例用户条目:")
		for i, entry := range sr.Entries {
			if i >= 3 {
				break
			}
			fmt.Printf("  - DN: %s\n", entry.DN)
			if len(entry.Attributes) > 0 {
				for _, attr := range entry.Attributes {
					if len(attr.Values) > 0 {
						fmt.Printf("      %s: %s\n", attr.Name, attr.Values[0])
					}
				}
			}
			fmt.Println()
		}
	}

	fmt.Println("=== 所有测试通过 ===")
}
