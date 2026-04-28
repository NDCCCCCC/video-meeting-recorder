package main

import (
	"crypto/tls"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

func main() {
	// AD 服务器配置
	server := "21.120.164.44:636"
	bindDN := "ninedrunk@pr.intra.cpic.com.cn"
	password := ")(PO09po"
	baseDN := "dc=pr,dc=intra,dc=cpic,dc=com,dc=cn"

	fmt.Printf("=== LDAP/LDAPS 连接测试 ===\n")
	fmt.Printf("服务器: %s\n", server)
	fmt.Printf("BindDN: %s\n", bindDN)
	fmt.Printf("BaseDN: %s\n", baseDN)
	fmt.Println()

	// 尝试不同的TLS配置
	tlsConfigs := []struct {
		name string
		cfg  *tls.Config
	}{
		{
			name: "TLS 1.2+ with InsecureSkipVerify",
			cfg: &tls.Config{
				ServerName:         "21.120.164.44",
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
			},
		},
		{
			name: "TLS 1.0+ with InsecureSkipVerify",
			cfg: &tls.Config{
				ServerName:         "21.120.164.44",
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS10,
			},
		},
		{
			name: "TLS 1.2+ with Renegotiation",
			cfg: &tls.Config{
				ServerName:         "21.120.164.44",
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS12,
				// 启用重新协商
				Renegotiation: tls.RenegotiateFreelyAsClient,
			},
		},
	}

	for _, config := range tlsConfigs {
		fmt.Printf("\n--- 测试配置: %s ---\n", config.name)

		// 使用 DialURL 替代 DialTLS
		ldapsURL := fmt.Sprintf("ldaps://%s", server)
		conn, err := ldap.DialURL(ldapsURL, ldap.DialWithTLSConfig(config.cfg))
		if err != nil {
			fmt.Printf("❌ 连接失败: %v\n", err)
			continue
		}
		defer conn.Close()

		fmt.Println("✓ LDAPS 连接成功")

		// 测试绑定
		err = conn.Bind(bindDN, password)
		if err != nil {
			fmt.Printf("❌ 绑定失败: %v\n", err)
			continue
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
			continue
		}

		fmt.Printf("✓ 搜索成功，找到 %d 个条目\n", len(sr.Entries))

		if len(sr.Entries) > 0 {
			fmt.Println("\n=== 所有测试通过 ===")
			return
		}
	}

	fmt.Println("\n❌ 所有配置都测试失败")
}
