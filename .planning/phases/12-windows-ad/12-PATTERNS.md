# Phase 12: Windows AD域控认证 - Pattern Map

**Mapped:** 2026-04-28
**Files analyzed:** 10
**Analogs found:** 8 / 10

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/migrations/013_add_ad_fields.go` | migration | schema-change | `internal/migrations/011_add_ip_restrictions.go` | exact |
| `internal/models/user.go` | model | CRUD | `internal/models/user.go` (existing) | modify |
| `internal/config/config.go` | config | read-only | `internal/config/config.go` (existing) | modify |
| `internal/auth/authenticator.go` | interface | request-response | `internal/auth/service.go` (existing pattern) | role-match |
| `internal/auth/local_auth.go` | service | request-response | `internal/auth/service.go` (extract from) | extract |
| `internal/auth/ad_auth.go` | service | request-response | `internal/auth/service.go` | role-match |
| `internal/auth/ad_config.go` | config | read-only | `internal/config/config.go` | role-match |
| `internal/auth/ad_validator.go` | service | request-response | Spike 005 pattern | no-analog |
| `internal/handlers/auth_handler.go` | handler | request-response | `internal/handlers/auth_handler.go` (existing) | modify |
| `internal/handlers/admin_handler.go` | handler | request-response | `internal/handlers/user_handler.go` | role-match |
| `frontend/src/pages/system/auth-config/index.tsx` | component | request-response | `frontend/src/pages/system/users/index.tsx` | role-match |

## Pattern Assignments

### `internal/migrations/013_add_ad_fields.go` (migration, schema-change)

**Analog:** `internal/migrations/011_add_ip_restrictions.go`

**Migration structure pattern** (lines 10-54):
```go
package migrations

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// AddADFieldsMigration 添加AD相关字段到users表
type AddADFieldsMigration struct{}

func (m *AddADFieldsMigration) Name() string {
	return "013_add_ad_fields"
}

func (m *AddADFieldsMigration) Up(db *gorm.DB) error {
	// Add AD fields (nullable for local users)
	fields := []struct {
		column string
		typ    string
	}{
		{"ad_username", "VARCHAR(100)"},
		{"ad_dn", "VARCHAR(255)"},
		{"ad_guid", "CHAR(36)"},
		{"ad_department", "VARCHAR(100)"},
		{"ad_upn", "VARCHAR(200)"},
		{"last_ad_login", "DATETIME"},
	}

	for _, field := range fields {
		exists, err := columnExists(db, "users", field.column)
		if err != nil {
			return fmt.Errorf("failed to check %s column in users: %w", field.column, err)
		}
		if !exists {
			if err := db.Exec("ALTER TABLE users ADD COLUMN " + field.column + " " + field.typ).Error; err != nil {
				return fmt.Errorf("failed to add %s column to users: %w", field.column, err)
			}
			log.Println("INFO: Added column " + field.column + " to users table")
		} else {
			log.Println("INFO: " + field.column + " column already exists in users table, skipping")
		}
	}

	// Create index on ad_guid for faster AD user lookups
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_ad_guid ON users(ad_guid)")

	log.Println("INFO: AD fields migration completed")
	return nil
}

func (m *AddADFieldsMigration) Down(db *gorm.DB) error {
	// SQLite doesn't support DROP COLUMN, leave deprecated per multi-role pattern
	log.Println("WARN: Rolling back AD fields migration: columns will remain deprecated")
	return nil
}
```

**Helper usage pattern** (from `internal/migrations/helpers.go` lines 18-27):
```go
// columnExists 检查列是否已存在
func columnExists(db *gorm.DB, tableName, columnName string) (bool, error) {
	var count int
	err := db.Raw(`
		SELECT COUNT(*)
		FROM pragma_table_info(?)
		WHERE name = ?
	`, tableName, columnName).Scan(&count).Error
	return count > 0, err
}
```

---

### `internal/models/user.go` (model, CRUD - MODIFY)

**Analog:** `internal/models/user.go` (existing, lines 12-23)

**Add AD fields to existing User struct:**
```go
// User 用户模型
type User struct {
	Base
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	PasswordHash string     `gorm:"type:varchar(255);not null" json:"-"`
	Email        string     `gorm:"type:varchar(100)" json:"email"`
	FullName     string     `gorm:"type:varchar(100)" json:"full_name"`
	Roles        []Role     `gorm:"many2many:users_roles;" json:"roles,omitempty"`
	AllowedIPs   string     `gorm:"type:text" json:"allowed_ips"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	APIKeys      []APIKey   `gorm:"foreignKey:UserID" json:"api_keys,omitempty"`

	// NEW: AD fields (nullable for local users)
	ADUsername   string     `gorm:"type:varchar(100)" json:"ad_username"`
	ADDN         string     `gorm:"type:varchar(255)" json:"ad_dn"`
	ADGUID       string     `gorm:"type:char(36);index" json:"ad_guid"`
	ADDepartment string     `gorm:"type:varchar(100)" json:"ad_department"`
	ADUPN        string     `gorm:"type:varchar(200)" json:"ad_upn"`
	LastADLogin  *time.Time `json:"last_ad_login"`
}
```

---

### `internal/config/config.go` (config, read-only - MODIFY)

**Analog:** `internal/config/config.go` (existing AuthConfig lines 58-68)

**Extend AuthConfig pattern:**
```go
// AuthConfig 认证配置
type AuthConfig struct {
	SM4Secret              string        `mapstructure:"sm4_secret" json:"sm4_secret" yaml:"sm4_secret"`
	AccessTokenDuration    time.Duration `mapstructure:"access_token_duration" json:"access_token_duration" yaml:"access_token_duration"`
	RefreshTokenDuration   time.Duration `mapstructure:"refresh_token_duration" json:"refresh_token_duration" yaml:"refresh_token_duration"`
	MaxSessionDuration     time.Duration `mapstructure:"max_session_duration" json:"max_session_duration" yaml:"max_session_duration"`
	HLSTokenSecret          string        `mapstructure:"hls_token_secret" json:"hls_token_secret" yaml:"hls_token_secret"`
	HLSTokenDuration        time.Duration `mapstructure:"hls_token_duration" json:"hls_token_duration" yaml:"hls_token_duration"`
	MaxDecryptFailures      int           `mapstructure:"max_decrypt_failures" json:"max_decrypt_failures" yaml:"max_decrypt_failures"`
	DecryptFailureWindow    int           `mapstructure:"decrypt_failure_window" json:"decrypt_failure_window" yaml:"decrypt_failure_window"`

	// NEW: Authentication mode (local, ad)
	Mode string `mapstructure:"mode" json:"mode" yaml:"mode"`

	// NEW: AD configuration
	AD ADAuthConfig `mapstructure:"ad" json:"ad" yaml:"ad"`
}

// ADAuthConfig AD域控配置
type ADAuthConfig struct {
	Server   string `mapstructure:"server" json:"server" yaml:"server"`
	BindDN   string `mapstructure:"bind_dn" json:"bind_dn" yaml:"bind_dn"`
	Password string `mapstructure:"password" json:"-" yaml:"password"`
	BaseDN   string `mapstructure:"base_dn" json:"base_dn" yaml:"base_dn"`
	UseTLS   bool   `mapstructure:"use_tls" json:"use_tls" yaml:"use_tls"`

	// Connection pool settings
	PoolSize int `mapstructure:"pool_size" json:"pool_size" yaml:"pool_size"`

	// Timeout settings
	DialTimeout    int `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`
	RequestTimeout int `mapstructure:"request_timeout" json:"request_timeout" yaml:"request_timeout"`

	// Test mode (for development only)
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
}
```

**Config defaults pattern** (from lines 280-356):
```go
// Add to setDefaults function
if cfg.Auth.Mode == "" {
	cfg.Auth.Mode = "local" // Default to local mode (safest)
}
if cfg.Auth.AD.PoolSize == 0 {
	cfg.Auth.AD.PoolSize = 10
}
if cfg.Auth.AD.DialTimeout == 0 {
	cfg.Auth.AD.DialTimeout = 10 // seconds
}
if cfg.Auth.AD.RequestTimeout == 0 {
	cfg.Auth.AD.RequestTimeout = 30 // seconds
}
```

---

### `internal/auth/authenticator.go` (interface, request-response)

**Analog:** Spike 003 pattern + `internal/auth/service.go` structure

**Authenticator interface pattern** (from Spike 003):
```go
package auth

// Authenticator 认证器接口
type Authenticator interface {
	// Login 用户登录
	Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error)

	// Logout 用户登出
	Logout(token string) error

	// ValidateToken 验证token
	ValidateToken(token string) (*UserDTO, error)

	// Name 认证器名称
	Name() string
}

// LoginRequest 登录请求 (reuse from service.go)
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应 (reuse from service.go)
type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
	User         *UserDTO `json:"user"`
}
```

---

### `internal/auth/local_auth.go` (service, request-response)

**Analog:** `internal/auth/service.go` (extract existing Login logic)

**Extract from existing service pattern** (lines 96-190):
```go
package auth

import (
	"context"
	"errors"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services/audit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LocalAuthenticator 本地认证器
type LocalAuthenticator struct {
	db          *gorm.DB
	tokenService *SM4TokenService
	cfg         *config.AuthConfig
	logger      *zap.Logger
	auditLogger *audit.AuditLogService
}

func NewLocalAuthenticator(db *gorm.DB, tokenService *SM4TokenService, cfg *config.AuthConfig, logger *zap.Logger, auditLogger *audit.AuditLogService) *LocalAuthenticator {
	return &LocalAuthenticator{
		db:           db,
		tokenService: tokenService,
		cfg:          cfg,
		logger:       logger,
		auditLogger:  auditLogger,
	}
}

func (a *LocalAuthenticator) Name() string {
	return "local"
}

func (a *LocalAuthenticator) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Copy existing login logic from service.go Login() method
	// Lines 99-189 contain the local auth logic to extract
	// ...
}

func (a *LocalAuthenticator) Logout(token string) error {
	return a.tokenService.RevokeSession(token)
}

func (a *LocalAuthenticator) ValidateToken(token string) (*UserDTO, error) {
	// Token validation logic
	// ...
}
```

---

### `internal/auth/ad_auth.go` (service, request-response)

**Analog:** Spike 001 + `internal/auth/service.go` pattern

**AD authenticator structure pattern** (from Spike 001):
```go
package auth

import (
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ADAuthenticator AD域控认证器
type ADAuthenticator struct {
	adConfig    *config.ADAuthConfig
	db          *gorm.DB
	tokenService *SM4TokenService
	logger      *zap.Logger
}

func NewADAuthenticator(cfg *config.ADAuthConfig, db *gorm.DB, tokenService *SM4TokenService, logger *zap.Logger) *ADAuthenticator {
	return &ADAuthenticator{
		adConfig:     cfg,
		db:           db,
		tokenService: tokenService,
		logger:       logger,
	}
}

func (a *ADAuthenticator) Name() string {
	return "ad"
}

func (a *ADAuthenticator) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Step 1: Connect to AD server
	conn, err := a.connectAD()
	if err != nil {
		return nil, fmt.Errorf("连接AD服务器失败: %w", err)
	}
	defer conn.Close()

	// Step 2: Bind as admin to search for user
	err = conn.Bind(a.adConfig.BindDN, a.adConfig.Password)
	if err != nil {
		return nil, fmt.Errorf("管理员绑定失败: %w", err)
	}

	// Step 3: Search for user DN (prevent LDAP injection)
	searchRequest := ldap.NewSearchRequest(
		a.adConfig.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", ldap.EscapeFilter(req.Username)),
		[]string{"dn", "sAMAccountName", "mail", "displayName", "userAccountControl", "objectGUID", "department", "userPrincipalName"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, errors.New("域控账号不存在，请联系管理员确认")
	}

	userDN := sr.Entries[0].DN
	adUser := a.parseLDAPEntry(sr.Entries[0])

	// Step 4: Check if account is disabled
	if adUser.IsDisabled() {
		return nil, errors.New("域控账号已禁用")
	}

	// Step 5: Bind as user to authenticate
	err = conn.Bind(userDN, req.Password)
	if err != nil {
		return nil, fmt.Errorf("域控密码错误")
	}

	// Step 6: Find or create local user
	localUser, err := a.findOrCreateLocalUser(adUser)
	if err != nil {
		return nil, err
	}

	// Step 7: Generate token using existing token service
	tokenPair, err := a.tokenService.GenerateTokenPair(localUser)
	if err != nil {
		return nil, err
	}

	// Step 8: Update last login time
	now := time.Now()
	localUser.LastLoginAt = &now
	localUser.LastADLogin = &now
	a.db.Save(localUser)

	// Step 9: Create session
	a.tokenService.CreateSession(localUser.ID, tokenPair.AccessToken, ipAddress, userAgent, tokenPair.ExpiresAt)

	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(a.tokenService.expireHours * 3600),
		User:         a.toUserDTO(localUser),
	}, nil
}

func (a *ADAuthenticator) connectAD() (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if a.adConfig.UseTLS {
		// LDAPS mode (port 636)
		tlsConfig := &tls.Config{
			ServerName:         extractHostname(a.adConfig.Server),
			InsecureSkipVerify: a.adConfig.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}
		conn, err = ldap.DialTLS("tcp", a.adConfig.Server, tlsConfig)
	} else {
		// LDAP mode (port 389) with StartTLS
		conn, err = ldap.Dial("tcp", a.adConfig.Server)
		if err == nil {
			err = conn.StartTLS(&tls.Config{
				ServerName: extractHostname(a.adConfig.Server),
				MinVersion: tls.VersionTLS12,
			})
		}
	}

	if err != nil {
		return nil, fmt.Errorf("AD连接失败: %w", err)
	}

	return conn, nil
}

func (a *ADAuthenticator) Logout(token string) error {
	return a.tokenService.RevokeSession(token)
}

func (a *ADAuthenticator) ValidateToken(token string) (*UserDTO, error) {
	// Use existing token validation logic
	// ...
}
```

---

### `internal/auth/ad_config.go` (config, read-only)

**Analog:** `internal/config/config.go` pattern

**AD config structures pattern** (move from config.go for separation of concerns):
```go
package auth

// ADUser AD用户信息
type ADUser struct {
	Username        string
	DN              string
	ObjectGUID      string
	Email           string
	DisplayName     string
	Department      string
	UserPrincipalName string
	UserAccountControl uint32
}

func (u *ADUser) IsDisabled() bool {
	// Check ACCOUNTDISABLE bit (0x0002)
	return u.UserAccountControl & 0x0002 != 0
}

// ADConfigValidationResult AD配置验证结果
type ADConfigValidationResult struct {
	Valid        bool     `json:"valid"`
	Level        int      `json:"level"`
	Errors       []string `json:"errors,omitempty"`
	Warnings     []string `json:"warnings,omitempty"`
	ServerInfo   string   `json:"server_info,omitempty"`
	ResponseTime int64    `json:"response_time_ms,omitempty"`
}
```

---

### `internal/auth/ad_validator.go` (service, request-response)

**Analog:** Spike 005 pattern (no existing analog in codebase)

**Four-layer validation pattern** (from Spike 005):
```go
package auth

import (
	"crypto/tls"
	"fmt"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

// ADConfigValidator AD配置验证器
type ADConfigValidator struct {
	logger *zap.Logger
}

func NewADConfigValidator(logger *zap.Logger) *ADConfigValidator {
	return &ADConfigValidator{logger: logger}
}

func (v *ADConfigValidator) Validate(config *ADAuthConfig) *ADConfigValidationResult {
	result := &ADConfigValidationResult{
		Valid:  false,
		Level:  0,
		Errors: []string{},
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
	defer conn.Close()
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
		return fmt.Errorf(strings.Join(errs, "; "))
	}
	return nil
}

func (v *ADConfigValidator) testConnection(config *ADAuthConfig) (*ldap.Conn, error) {
	// Same connection logic as ad_auth.go
	// ...
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
```

---

### `internal/handlers/auth_handler.go` (handler, request-response - MODIFY)

**Analog:** `internal/handlers/auth_handler.go` (existing)

**Handler pattern** (lines 34-62):
```go
// Add new AD test connection endpoint to existing auth_handler.go

// TestADConnection 测试AD连接
// @Summary 测试AD域控连接
// @Description 验证AD配置是否正确
// @Tags 认证
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body auth.ADAuthConfig true "AD配置"
// @Success 200 {object} response.Response{data=auth.ADConfigValidationResult}
// @Router /api/v1/auth/ad/test-connection [post]
func (h *AuthHandler) TestADConnection(c *gin.Context) {
	var req auth.ADAuthConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	validator := auth.NewADConfigValidator(h.logger)
	result := validator.Validate(&req)

	if result.Valid {
		response.GinSuccess(c, result)
	} else {
		response.GinError(c, response.CodeInvalidRequest, "AD配置验证失败")
		c.JSON(200, gin.H{"valid": false, "errors": result.Errors})
	}
}
```

---

### `internal/handlers/admin_handler.go` (handler, request-response)

**Analog:** `internal/handlers/user_handler.go`

**Admin handler pattern** (lines 96-111):
```go
package handlers

import (
	"github.com/cpic/record_v2/internal/auth"
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/middleware"
	"github.com/cpic/record_v2/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	cfg    *config.Config
	logger *zap.Logger
}

func NewAdminHandler(cfg *config.Config, logger *zap.Logger) *AdminHandler {
	return &AdminHandler{
		cfg:    cfg,
		logger: logger,
	}
}

// GetAuthConfig 获取认证配置
// @Summary 获取认证配置
// @Description 获取当前认证配置
// @Tags 系统管理
// @Security Bearer
// @Success 200 {object} response.Response{data=config.AuthConfig}
// @Router /api/v1/admin/auth/config [get]
func (h *AdminHandler) GetAuthConfig(c *gin.Context) {
	// Return sanitized config (hide password)
	sanitized := map[string]interface{}{
		"mode": h.cfg.Auth.Mode,
		"ad": map[string]interface{}{
			"server": h.cfg.Auth.AD.Server,
			"bind_dn": h.cfg.Auth.AD.BindDN,
			"base_dn": h.cfg.Auth.AD.BaseDN,
			"use_tls": h.cfg.Auth.AD.UseTLS,
			"pool_size": h.cfg.Auth.AD.PoolSize,
			// Password excluded
		},
	}
	response.GinSuccess(c, sanitized)
}

// UpdateAuthConfig 更新认证配置
// @Summary 更新认证配置
// @Description 更新认证配置并验证
// @Tags 系统管理
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body object{mode=string,ad=auth.ADAuthConfig} true "认证配置"
// @Success 200 {object} response.Response
// @Router /api/v1/admin/auth/config [put]
func (h *AdminHandler) UpdateAuthConfig(c *gin.Context) {
	var req struct {
		Mode string                `json:"mode" binding:"required,oneof=local ad"`
		AD   auth.ADAuthConfig     `json:"ad"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
		return
	}

	currentUserID := middleware.GetUserID(c)

	// If switching to AD mode, validate AD config first
	if req.Mode == "ad" {
		validator := auth.NewADConfigValidator(h.logger)
		result := validator.Validate(&req.AD)

		if !result.Valid {
			response.GinError(c, response.CodeInvalidRequest, "AD配置验证失败: "+strings.Join(result.Errors, "; "))
			return
		}
	}

	// Log the configuration change
	h.logger.Info("Authentication mode changed",
		zap.Uint("user_id", currentUserID),
		zap.String("old_mode", h.cfg.Auth.Mode),
		zap.String("new_mode", req.Mode),
	)

	// Update configuration
	h.cfg.Auth.Mode = req.Mode
	if req.Mode == "ad" {
		h.cfg.Auth.AD = req.AD
	}

	response.GinSuccess(c, gin.H{"message": "认证配置已更新"})
}
```

---

### `frontend/src/pages/system/auth-config/index.tsx` (component, request-response)

**Analog:** `frontend/src/pages/system/users/index.tsx`

**Component structure pattern** (from existing user management):
```tsx
import React, { useState, useEffect } from 'react';
import { Form, Input, Switch, Button, message, Modal, Alert } from 'antd';
import { getAuthConfig, updateAuthConfig, testADConnection } from '@/api/auth';

const AuthConfigPage: React.FC = () => {
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [testing, setTesting] = useState(false);
  const [config, setConfig] = useState<any>(null);

  useEffect(() => {
    fetchConfig();
  }, []);

  const fetchConfig = async () => {
    setLoading(true);
    try {
      const response = await getAuthConfig();
      setConfig(response.data);
      form.setFieldsValue(response.data);
    } catch (error) {
      message.error('获取配置失败');
    } finally {
      setLoading(false);
    }
  };

  const handleTestConnection = async () => {
    setTesting(true);
    try {
      const values = await form.validateFields();
      const response = await testADConnection(values.ad);

      if (response.data.valid) {
        message.success('AD连接测试成功');
      } else {
        message.error('AD连接测试失败: ' + response.data.errors.join(', '));
      }
    } catch (error) {
      message.error('连接测试失败');
    } finally {
      setTesting(false);
    }
  };

  const handleSave = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields();

      // Warn if using port 389
      if (values.mode === 'ad' && !values.ad.use_tls) {
        Modal.confirm({
          title: '安全警告',
          content: '使用LDAP 389端口时密码将以明文传输，存在安全风险。建议在生产环境使用LDAPS 636端口。',
          onOk: async () => {
            await saveConfig(values);
          },
        });
      } else {
        await saveConfig(values);
      }
    } catch (error) {
      message.error('保存配置失败');
    } finally {
      setLoading(false);
    }
  };

  const saveConfig = async (values: any) => {
    try {
      await updateAuthConfig(values);
      message.success('配置已更新');
      fetchConfig();
    } catch (error: any) {
      message.error(error.response?.data?.error || '保存失败');
    }
  };

  return (
    <div className="auth-config-page">
      <h1>认证配置</h1>

      <Form form={form} layout="vertical">
        <Form.Item
          label="认证模式"
          name="mode"
          rules={[{ required: true }]}
        >
          <Select>
            <Option value="local">本地认证</Option>
            <Option value="ad">AD域控认证</Option>
          </Select>
        </Form.Item>

        <Form.Item noStyle shouldUpdate={(prev, curr) => prev.mode !== curr.mode}>
          {({ getFieldValue }) =>
            getFieldValue('mode') === 'ad' ? (
              <>
                <Alert
                  message="AD域控配置"
                  description="配置AD服务器连接信息"
                  type="info"
                  showIcon
                  style={{ marginBottom: 16 }}
                />

                <Form.Item label="AD服务器地址" name={['ad', 'server']} rules={[{ required: true }]}>
                  <Input placeholder="ad.example.com:636" />
                </Form.Item>

                <Form.Item label="BindDN" name={['ad', 'bind_dn']} rules={[{ required: true }]}>
                  <Input placeholder="cn=admin,cn=users,dc=example,dc=com" />
                </Form.Item>

                <Form.Item label="管理员密码" name={['ad', 'password']} rules={[{ required: true }]}>
                  <Input.Password />
                </Form.Item>

                <Form.Item label="BaseDN" name={['ad', 'base_dn']} rules={[{ required: true }]}>
                  <Input placeholder="dc=example,dc=com" />
                </Form.Item>

                <Form.Item label="使用LDAPS" name={['ad', 'use_tls']} valuePropName="checked">
                  <Switch />
                </Form.Item>

                {!getFieldValue(['ad', 'use_tls']) && (
                  <Alert
                    message="安全警告"
                    description="⚠️ 使用LDAP 389端口时密码将以明文传输，存在安全风险。建议在生产环境使用LDAPS 636端口。"
                    type="warning"
                    showIcon
                    style={{ marginBottom: 16 }}
                  />
                )}

                <Button onClick={handleTestConnection} loading={testing}>
                  测试连接
                </Button>
              </>
            ) : null
          }
        </Form.Item>

        <Form.Item>
          <Button type="primary" onClick={handleSave} loading={loading}>
            保存配置
          </Button>
        </Form.Item>
      </Form>
    </div>
  );
};

export default AuthConfigPage;
```

---

## Shared Patterns

### Error Handling
**Source:** `internal/handlers/auth_handler.go` (lines 47-54) and `pkg/response`
**Apply to:** All handler files

```go
// Standard error response pattern
if err != nil {
    h.logger.Warn("Operation failed", zap.Error(err))
    response.GinError(c, response.CodeInvalidCredential, err.Error())
    return
}
```

### Validation Pattern
**Source:** `internal/handlers/auth_handler.go` (lines 35-39)
**Apply to:** All handler POST/PUT endpoints

```go
var req auth.LoginRequest
if err := c.ShouldBindJSON(&req); err != nil {
    response.GinError(c, response.CodeInvalidRequest, "请求参数错误: "+err.Error())
    return
}
```

### Logging Pattern
**Source:** `internal/handlers/auth_handler.go` (lines 48-49, 56-59)
**Apply to:** All service and handler files

```go
// Warning log for failures
h.logger.Warn("Login failed",
    zap.String("username", req.Username),
    zap.Error(err),
)

// Info log for successful operations
h.logger.Info("User logged in",
    zap.String("username", req.Username),
    zap.Uint("user_id", result.User.ID),
)
```

### Database Migration Pattern
**Source:** `internal/migrations/011_add_ip_restrictions.go`
**Apply to:** All migration files

- Check column exists before adding
- Use descriptive log messages
- Handle SQLite limitations (no DROP COLUMN)
- Return clear error messages with context

### Configuration Environment Variable Pattern
**Source:** `internal/config/config.go` (lines 172-197)
**Apply to:** All sensitive configuration (AD password)

```go
// Use environment variable expansion for sensitive data
Password string `mapstructure:"password" json:"-" yaml:"password"`

// Expand ${VAR:default} format
func expandEnvWithDefault(s string) string {
    // See lines 173-197 in config.go
}
```

### Middleware Authentication Pattern
**Source:** `internal/handlers/user_handler.go` (lines 138, 235)
**Apply to:** All protected endpoints

```go
// Get current user ID from JWT token
currentUserID := middleware.GetUserID(c)

// Use for audit logging
h.logger.Info("User updated", zap.Uint("user_id", id))
```

---

## No Analog Found

Files with no close match in the codebase (use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/auth/ad_validator.go` | service | request-response | No validation service pattern exists; use Spike 005 four-layer validation |
| `frontend/src/pages/system/auth-config/index.tsx` | component | request-response | No exact config page analog; use user management page structure |

**Use Spike 005 patterns for AD validation:**
- Four-layer validation (format → network → auth → functionality)
- LDAP-specific error formatting
- Connection testing with timeout handling

---

## Metadata

**Analog search scope:**
- `internal/migrations/` - Migration file patterns
- `internal/models/` - Model definitions
- `internal/config/` - Configuration structures
- `internal/auth/` - Existing authentication service
- `internal/handlers/` - Handler patterns
- `.planning/spikes/001-005/` - Validated spike patterns

**Files scanned:** 15
**Pattern extraction date:** 2026-04-28

**Key patterns identified:**
1. All migrations use `columnExists()` helper before adding columns
2. Handlers use `response.GinError()` and `response.GinSuccess()` for consistent responses
3. Services use structured logging with `zap.Logger`
4. Configuration uses mapstructure tags with JSON/YAML support
5. Sensitive data (passwords) uses `json:"-"` to hide from API responses
6. Frontend uses antd components with Form validation
7. Middleware provides `GetUserID()` helper for authenticated requests

---

*Phase: 12-windows-ad*
*Pattern mapping completed: 2026-04-28*
