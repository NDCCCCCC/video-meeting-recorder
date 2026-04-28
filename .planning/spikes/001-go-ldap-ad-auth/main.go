package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// Spike 001: 验证Go LDAP库对Windows Active Directory认证的支持

// ADConfig AD域控配置
type ADConfig struct {
	Server   string // LDAP服务器地址，如: ad.example.com:636
	BindDN   string // 管理员绑定DN，如: cn=admin,dc=example,dc=com
	Password string // 管理员密码
	BaseDN   string // 搜索基础DN，如: dc=example,dc=com
	UseTLS   bool   // 是否使用LDAPS
}

// ADUser AD用户信息
type ADUser struct {
	DN           string
	Username     string
	Email        string
	DisplayName  string
	Department   string
	LastLogon    time.Time
	IsEnabled    bool
}

// ADAuthenticator AD认证器
type ADAuthenticator struct {
	config *ADConfig
}

// NewADAuthenticator 创建AD认证器
func NewADAuthenticator(config *ADConfig) *ADAuthenticator {
	return &ADAuthenticator{config: config}
}

// Connect 连接到AD服务器
func (a *ADAuthenticator) Connect() (*ldap.Conn, error) {
	var l *ldap.Conn
	var err error

	if a.config.UseTLS {
		// 使用LDAPS (端口636)
		l, err = ldap.DialTLS("tcp", a.config.Server, &tls.Config{
			ServerName:         a.config.Server,
			InsecureSkipVerify: false, // 生产环境应设置为true并配置证书
			MinVersion:         tls.VersionTLS12,
		})
		if err != nil {
			return nil, fmt.Errorf("LDAPS连接失败: %w", err)
		}
	} else {
		// 使用LDAP (端口389)
		l, err = ldap.Dial("tcp", a.config.Server)
		if err != nil {
			return nil, fmt.Errorf("LDAP连接失败: %w", err)
		}

		// 升级到StartTLS
		err = l.StartTLS(&tls.Config{
			ServerName:         a.config.Server,
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		})
		if err != nil {
			l.Close()
			return nil, fmt.Errorf("StartTLS升级失败: %w", err)
		}
	}

	return l, nil
}

// AuthenticateUser 认证用户
func (a *ADAuthenticator) AuthenticateUser(username, password string) (*ADUser, error) {
	// 1. 连接到AD服务器
	conn, err := a.Connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 2. 使用管理员账户绑定
	err = conn.Bind(a.config.BindDN, a.config.Password)
	if err != nil {
		return nil, fmt.Errorf("管理员绑定失败: %w", err)
	}

	// 3. 搜索用户DN
	userDN, err := a.findUserDN(conn, username)
	if err != nil {
		return nil, err
	}

	// 4. 使用用户凭据重新绑定进行认证
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("用户认证失败: %w", err)
	}

	// 5. 获取用户属性
	user, err := a.getUserAttributes(conn, userDN)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// findUserDN 查找用户DN
func (a *ADAuthenticator) findUserDN(conn *ldap.Conn, username string) (string, error) {
	// AD常用属性: sAMAccountName (登录名), userPrincipalName (UPN)
	searchRequest := ldap.NewSearchRequest(
		a.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", ldap.EscapeFilter(username)),
		[]string{"dn"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return "", fmt.Errorf("搜索用户失败: %w", err)
	}

	if len(sr.Entries) == 0 {
		return "", fmt.Errorf("用户不存在: %s", username)
	}

	if len(sr.Entries) > 1 {
		return "", fmt.Errorf("找到多个用户: %s", username)
	}

	return sr.Entries[0].DN, nil
}

// getUserAttributes 获取用户属性
func (a *ADAuthenticator) getUserAttributes(conn *ldap.Conn, userDN string) (*ADUser, error) {
	searchRequest := ldap.NewSearchRequest(
		userDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(objectClass=user)",
		[]string{
			"dn",
			"sAMAccountName",
			"userPrincipalName",
			"mail",
			"displayName",
			"department",
			"lastLogon",
			"userAccountControl",
		},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("获取用户属性失败: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("未找到用户: %s", userDN)
	}

	entry := sr.Entries[0]

	// 解析userAccountControl判断账户是否启用
	// 常见值: 512 (正常启用), 514 (禁用), 66048 (启用, 密码永不过期)
	uac := entry.GetAttributeValue("userAccountControl")
	isEnabled := uac != "" && uac[0] != '5' // 简单判断

	user := &ADUser{
		DN:          entry.DN,
		Username:    entry.GetAttributeValue("sAMAccountName"),
		Email:       entry.GetAttributeValue("mail"),
		DisplayName: entry.GetAttributeValue("displayName"),
		Department:  entry.GetAttributeValue("department"),
		IsEnabled:   isEnabled,
	}

	return user, nil
}

// TestConnection 测试AD连接
func (a *ADAuthenticator) TestConnection() error {
	conn, err := a.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	// 尝试绑定
	err = conn.Bind(a.config.BindDN, a.config.Password)
	if err != nil {
		return fmt.Errorf("绑定失败: %w", err)
	}

	// 获取当前用户信息
	whoami, err := conn.WhoAmI(nil)
	if err != nil {
		return fmt.Errorf("获取当前用户失败: %w", err)
	}

	log.Printf("✓ 连接成功! 当前用户: %s", whoami.AuthzID)
	return nil
}

// ListUsers 列出所有用户 (用于测试)
func (a *ADAuthenticator) ListUsers(maxResults int) ([]ADUser, error) {
	conn, err := a.Connect()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	err = conn.Bind(a.config.BindDN, a.config.Password)
	if err != nil {
		return nil, err
	}

	searchRequest := ldap.NewSearchRequest(
		a.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		"(&(objectClass=user)(!(objectClass=computer)))", // 排除计算机账户
		[]string{
			"dn",
			"sAMAccountName",
			"mail",
			"displayName",
			"department",
			"userAccountControl",
		},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	users := make([]ADUser, 0, len(sr.Entries))
	for i, entry := range sr.Entries {
		if maxResults > 0 && i >= maxResults {
			break
		}

		uac := entry.GetAttributeValue("userAccountControl")
		isEnabled := uac != "" && uac[0] != '5'

		users = append(users, ADUser{
			DN:          entry.DN,
			Username:    entry.GetAttributeValue("sAMAccountName"),
			Email:       entry.GetAttributeValue("mail"),
			DisplayName: entry.GetAttributeValue("displayName"),
			Department:  entry.GetAttributeValue("department"),
			IsEnabled:   isEnabled,
		})
	}

	return users, nil
}

func main() {
	// 示例配置 (实际使用时需要从环境变量或配置文件读取)
	config := &ADConfig{
		Server:   "ad.example.com:636",
		BindDN:   "cn=admin,dc=example,dc=com",
		Password: "admin_password",
		BaseDN:   "dc=example,dc=com",
		UseTLS:   true,
	}

	authenticator := NewADAuthenticator(config)

	// 测试连接
	fmt.Println("=== 测试AD连接 ===")
	if err := authenticator.TestConnection(); err != nil {
		log.Fatalf("连接测试失败: %v", err)
	}

	// 列出用户
	fmt.Println("\n=== 列出前5个用户 ===")
	users, err := authenticator.ListUsers(5)
	if err != nil {
		log.Printf("列出用户失败: %v", err)
	} else {
		for _, user := range users {
			fmt.Printf("- %s (%s) - %s - 启用:%v\n",
				user.Username, user.Email, user.DisplayName, user.IsEnabled)
		}
	}

	// 测试认证 (需要有效的用户名和密码)
	fmt.Println("\n=== 测试用户认证 ===")
	// user, err := authenticator.AuthenticateUser("testuser", "testpassword")
	// if err != nil {
	//     log.Printf("认证失败: %v", err)
	// } else {
	//     fmt.Printf("✓ 认证成功! 用户: %s (%s)\n", user.DisplayName, user.Email)
	// }

	fmt.Println("提示: 取消注释上面代码并填入有效的AD凭据来测试认证功能")
}
