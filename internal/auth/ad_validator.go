package auth

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

// ADConfigValidator AD 域控配置验证器：执行四级验证（格式→网络→认证→功能）
// 帮助管理员在部署前确认 AD 配置正确性（per Spike 005 four-layer validation）。
type ADConfigValidator struct {
	logger *zap.Logger
}

func NewADConfigValidator(logger *zap.Logger) *ADConfigValidator {
	return &ADConfigValidator{logger: logger}
}

func (v *ADConfigValidator) Validate(config *ADAuthConfig) *ADConfigValidationResult {
	result := &ADConfigValidationResult{
		Valid:    false,
		Level:    0,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Layer 1: Format validation (no network calls)
	if err := v.validateFormat(config); err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.Level = 1

	// Layer 2: Network validation (TCP connection)
	start := time.Now()
	conn, err := v.testConnection(config)
	if err != nil {
		result.Errors = append(result.Errors, v.formatConnectionError(err))
		return result
	}
	// STYLE-008: nil 防御——testConnection 失败时 conn 为 nil，defer Close 会 panic
	if conn != nil {
		defer conn.Close()
	}
	result.ResponseTime = time.Since(start).Milliseconds()
	result.Level = 2

	// Layer 3: Authentication validation (bind test)
	if err := v.testBind(conn, config); err != nil {
		result.Errors = append(result.Errors, v.formatBindError(err))
		return result
	}
	result.Level = 3

	// Layer 4: Functionality validation (user search)
	if err := v.testFunctionality(conn, config); err != nil {
		result.Warnings = append(result.Warnings, "功能测试警告: "+err.Error())
	}
	result.Level = 4

	// Check for port 389 warning (per D-12, D-14)
	if !config.UseTLS {
		result.Warnings = append(result.Warnings,
			"⚠️ 使用LDAP 389端口时密码将以明文传输，存在安全风险。建议在生产环境使用LDAPS 636端口。")
	}

	result.Valid = true
	return result
}

func (v *ADConfigValidator) validateFormat(config *ADAuthConfig) error {
	var errs []string

	if config.Server == "" {
		errs = append(errs, "服务器地址不能为空")
	}
	if config.BindDN == "" {
		errs = append(errs, "BindDN不能为空")
	}
	if config.Password == "" {
		errs = append(errs, "管理员密码不能为空")
	}
	if config.BaseDN == "" {
		errs = append(errs, "BaseDN不能为空")
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func (v *ADConfigValidator) testConnection(config *ADAuthConfig) (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if config.UseTLS {
		// LDAPS mode (port 636)
		tlsConfig := &tls.Config{
			ServerName:         extractHostname(config.Server),
			InsecureSkipVerify: config.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}
		conn, err = ldap.DialTLS("tcp", config.Server, tlsConfig)
	} else {
		// Plain LDAP mode (port 389) - NO TLS, NO StartTLS
		// Warning: credentials will be sent in plain text
		conn, err = ldap.Dial("tcp", config.Server)
	}

	if err != nil {
		return nil, err
	}

	return conn, nil
}

func (v *ADConfigValidator) testBind(conn *ldap.Conn, config *ADAuthConfig) error {
	err := conn.Bind(config.BindDN, config.Password)
	if err != nil {
		return fmt.Errorf("认证失败: %w", err)
	}
	return nil
}

func (v *ADConfigValidator) testFunctionality(conn *ldap.Conn, config *ADAuthConfig) error {
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		"(objectClass=user)",
		[]string{"dn", "sAMAccountName"},
		nil,
	)

	_, err := conn.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("搜索测试失败: %w", err)
	}

	return nil
}

func (v *ADConfigValidator) formatConnectionError(err error) string {
	errMsg := err.Error()

	// User-friendly Chinese messages (per D-18, D-20)
	switch {
	case strings.Contains(errMsg, "no such host"):
		return fmt.Sprintf("无法解析服务器地址: %v (请检查服务器地址是否正确)", err)
	case strings.Contains(errMsg, "connection refused"):
		return fmt.Sprintf("连接被拒绝: %v (请检查防火墙设置和LDAP服务是否启动)", err)
	case strings.Contains(errMsg, "i/o timeout"):
		return fmt.Sprintf("连接超时: %v (请检查网络连接和服务器状态)", err)
	case strings.Contains(errMsg, "certificate"):
		return fmt.Sprintf("TLS证书错误: %v (请检查证书配置或临时使用测试模式)", err)
	default:
		return fmt.Sprintf("连接失败: %v", err)
	}
}

func (v *ADConfigValidator) formatBindError(err error) string {
	// Log detailed LDAP error to backend (per D-19)
	v.logger.Error("AD bind error details",
		zap.String("ldap_error", err.Error()),
		zap.Any("error_type", fmt.Sprintf("%T", err)),
	)

	// Return sanitized message to user (per D-18, D-19)
	if ldapErr, ok := err.(*ldap.Error); ok {
		switch ldapErr.ResultCode {
		case ldap.LDAPResultInvalidCredentials:
			return "管理员用户名或密码错误"
		case ldap.LDAPResultNoSuchObject:
			return "BindDN指定的对象不存在"
		case ldap.LDAPResultInsufficientAccessRights:
			return "管理员权限不足"
		default:
			return fmt.Sprintf("认证失败: %v", err)
		}
	}

	return fmt.Sprintf("认证失败: %v", err)
}

func extractHostname(serverAddr string) string {
	parts := strings.Split(serverAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return serverAddr
}
