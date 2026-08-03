package auth

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
)

// IsADUserNotRegistered reports whether err originates from the
// "AD user not registered, auto-create disabled" branch.
//
// The sentinel now lives in internal/errors (Phase 20 R-3 migration) so the
// mapping.go pipeline can recognize the error in 403-handling without
// importing the auth package. This wrapper remains for any callers that
// still need a direct predicate; the Login handler now uses
// response.HandleError (Phase 20 20-02) and no longer references this helper.
func IsADUserNotRegistered(err error) bool {
	return errors.Is(err, apperrors.ErrADUserNotRegistered)
}

// ADAuthenticator AD域控认证器
type ADAuthenticator struct {
	adConfig     *config.ADAuthConfig
	db           *gorm.DB
	tokenService *SM4TokenService
	sm4Secret    string
	logger       *zap.Logger
	liveConfig   *config.AuthConfig // Live config reference for dynamic settings
}

// NewADAuthenticator creates a new AD authenticator
func NewADAuthenticator(cfg *config.ADAuthConfig, db *gorm.DB, tokenService *SM4TokenService, logger *zap.Logger, sm4Secret string) *ADAuthenticator {
	return &ADAuthenticator{
		adConfig:     cfg,
		db:           db,
		tokenService: tokenService,
		sm4Secret:    sm4Secret,
		logger:       logger,
	}
}

// SetLiveConfig sets the live config reference for dynamic settings
func (a *ADAuthenticator) SetLiveConfig(authConfig *config.AuthConfig) {
	a.liveConfig = authConfig
}

// Name returns the authenticator name
func (a *ADAuthenticator) Name() string {
	return "ad"
}

// Login authenticates a user using AD authentication
func (a *ADAuthenticator) Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Step 1: Connect to AD server
	conn, err := a.connectAD()
	if err != nil {
		a.logger.Error("AD connection failed", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("无法连接到域控服务器，请检查网络和配置: %w", apperrors.ErrADUnreachable) // per D-18, D6
	}
	// STYLE-008: nil 防御——connectAD 失败时 conn 为 nil，defer Close 会 panic
	if conn != nil {
		defer func() { _ = conn.Close() }()
	}

	// Step 2: Bind as admin to search for user
	err = conn.Bind(a.adConfig.BindDN, a.adConfig.Password)
	if err != nil {
		a.logger.Error("AD admin bind failed", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("域控管理员认证失败: %w", apperrors.ErrADConfigError)
	}

	// Step 3: Search for user DN (prevent LDAP injection - use EscapeFilter)
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
		a.logger.Error("AD user search failed", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("搜索用户失败: %w", apperrors.ErrADUnreachable)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("域控账号不存在，请联系管理员确认: %w", apperrors.ErrADAccountNotFound) // per D-20, D6
	}

	userDN := sr.Entries[0].DN
	adUser := a.parseLDAPEntry(sr.Entries[0])

	// Step 4: Check if account is disabled
	if adUser.IsDisabled() {
		return nil, fmt.Errorf("域控账号已禁用: %w", apperrors.ErrUserDisabled)
	}

	// Step 4.5: AD authentication requires SM4-encrypted password only (no plaintext support)
	if !utils.IsEncryptedPassword(req.Password) {
		a.logger.Warn("Plaintext password rejected for AD authentication", zap.String("username", req.Username))
		return nil, fmt.Errorf("域控密码错误: %w", apperrors.ErrUnauthorized)
	}

	// SM4 secret must be configured for AD authentication
	if a.sm4Secret == "" {
		a.logger.Error("SM4_SECRET not configured for AD authentication")
		return nil, fmt.Errorf("系统配置错误: %w", apperrors.ErrADConfigError)
	}

	// Validate SM4 secret configuration
	if err := utils.ValidateSM4Secret(a.sm4Secret); err != nil {
		a.logger.Error("Invalid SM4 secret configuration", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("系统配置错误: %w", apperrors.ErrADConfigError)
	}

	// Decrypt SM4 password
	passwordForBind, err := utils.DecryptPasswordECB(req.Password, a.sm4Secret)
	if err != nil {
		a.logger.Warn("Failed to decrypt SM4 password", zap.String("username", req.Username))
		return nil, fmt.Errorf("域控密码错误: %w", apperrors.ErrUnauthorized)
	}

	a.logger.Debug("SM4 password decrypted for AD login", zap.String("username", req.Username))

	// Step 5: Bind as user to authenticate (verify password)
	err = conn.Bind(userDN, passwordForBind)
	if err != nil {
		a.logger.Warn("AD user bind failed", zap.String("username", req.Username))
		return nil, fmt.Errorf("域控密码错误: %w", apperrors.ErrUnauthorized) // per D-20, D6
	}

	// Step 6: Find or create local user (transparent mapping per D-06, D-08)
	localUser, err := a.findOrCreateLocalUser(ctx, adUser)
	if err != nil {
		a.logger.Error("AD user mapping failed", zap.Error(err), response.SentinelField(err))
		// Preserve sentinel error so handler can map to HTTP 403 instead of 500
		if IsADUserNotRegistered(err) {
			return nil, err
		}
		// Phase 19 D6: 保留底层 err 上下文（errors.Is 链上仍可分类），不再用
		// errors.New("用户映射失败") 把它丢掉。
		return nil, fmt.Errorf("用户映射失败: %w", err)
	}

	// Step 7: Generate token using existing token service
	tokenPair, err := a.tokenService.GenerateTokenPair(localUser)
	if err != nil {
		a.logger.Error("AD token generation failed", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("登录失败，请稍后重试: %w", apperrors.ErrInternal)
	}

	// Step 8: Update last login times
	now := time.Now()
	localUser.LastLoginAt = &now
	localUser.LastADLogin = &now
	a.db.WithContext(ctx).Save(localUser)

	// Step 9: Create session
	_ = a.tokenService.CreateSession(localUser.ID, tokenPair.AccessToken, ipAddress, userAgent, tokenPair.ExpiresAt)

	a.logger.Info("AD user logged in", zap.String("username", req.Username), zap.Uint("user_id", localUser.ID))

	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(a.tokenService.expireHours * 3600),
		User:         a.toUserDTO(localUser),
	}, nil
}

// connectAD connects to the AD server using LDAP or LDAPS
func (a *ADAuthenticator) connectAD() (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if a.adConfig.UseTLS {
		// LDAPS mode (port 636) - recommended for production
		tlsConfig := &tls.Config{
			ServerName:         extractHostname(a.adConfig.Server),
			InsecureSkipVerify: a.adConfig.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}
		conn, err = ldap.DialURL("ldaps://"+a.adConfig.Server, ldap.DialWithTLSConfig(tlsConfig))
	} else {
		// Plain LDAP mode (port 389) - NO TLS, NO StartTLS
		// Warning: credentials will be sent in plain text
		conn, err = ldap.DialURL("ldap://" + a.adConfig.Server)
	}

	if err != nil {
		return nil, fmt.Errorf("AD连接失败: %w", err)
	}

	return conn, nil
}

// parseLDAPEntry parses an LDAP entry to ADUser struct
func (a *ADAuthenticator) parseLDAPEntry(entry *ldap.Entry) *ADUser {
	userAccountControl := uint32(0)
	if attr := entry.GetAttributeValue("userAccountControl"); attr != "" {
		_, _ = fmt.Sscanf(attr, "%d", &userAccountControl)
	}

	return &ADUser{
		Username:           entry.GetAttributeValue("sAMAccountName"),
		DN:                 entry.DN,
		ObjectGUID:         entry.GetAttributeValue("objectGUID"),
		Email:              entry.GetAttributeValue("mail"),
		DisplayName:        entry.GetAttributeValue("displayName"),
		Department:         entry.GetAttributeValue("department"),
		UserPrincipalName:  entry.GetAttributeValue("userPrincipalName"),
		UserAccountControl: userAccountControl,
	}
}

// findOrCreateLocalUser finds an existing local user or creates a new one (if allowed)
func (a *ADAuthenticator) findOrCreateLocalUser(ctx context.Context, adUser *ADUser) (*models.User, error) {
	// First, try to find existing user by ad_guid or username
	var user models.User
	err := a.db.WithContext(ctx).Where("ad_guid = ? OR username = ?", adUser.ObjectGUID, adUser.Username).First(&user).Error

	if err == nil {
		// Found existing user, update AD information
		a.updateADInfo(ctx, &user, adUser)
		// Reload with Roles.Permissions for toUserDTO
		a.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, user.ID)
		return &user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// User not found - check if auto-create is allowed
	// Default to true for backward compatibility if config is not set
	allowAutoCreate := true
	if a.liveConfig != nil {
		allowAutoCreate = a.liveConfig.AD.AllowAutoCreate
	} else if a.adConfig != nil {
		allowAutoCreate = a.adConfig.AllowAutoCreate
	}

	if !allowAutoCreate {
		a.logger.Warn("AD user not found in system and auto-create is disabled",
			zap.String("username", adUser.Username))
		// Return sentinel so handler can map to HTTP 403 (whitelist policy)
		return nil, apperrors.ErrADUserNotRegistered
	}

	// Auto-create is allowed, proceed with user creation
	// Not found, create new user with random password (per D-07)
	user = models.User{
		Username:     adUser.Username,
		Email:        adUser.Email,
		FullName:     adUser.DisplayName,
		ADUsername:   adUser.Username,
		ADDN:         adUser.DN,
		ADGUID:       adUser.ObjectGUID,
		ADDepartment: adUser.Department,
		ADUPN:        adUser.UserPrincipalName,
		IsActive:     true,
	}

	// Generate random password (AD users won't use it)
	randomPassword := utils.GenerateRandomPassword(32)
	if err := user.SetPassword(randomPassword); err != nil {
		return nil, err
	}

	// Assign default viewer role using models.RoleViewer constant
	var defaultRole models.Role
	err = a.db.WithContext(ctx).Where("name = ?", models.RoleViewer).First(&defaultRole).Error
	if err != nil {
		a.logger.Error("Default viewer role not found", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("默认角色不存在: %w", apperrors.ErrADConfigError)
	}

	// Create user first
	if err := a.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, err
	}

	// Then associate the role via users_roles junction table (many2many)
	if err := a.db.WithContext(ctx).Model(&user).Association("Roles").Append(&defaultRole); err != nil {
		a.logger.Error("Failed to assign default role to AD user", zap.Error(err), response.SentinelField(err))
		// Cleanup: delete the user if role assignment fails
		a.db.WithContext(ctx).Delete(&user)
		return nil, fmt.Errorf("分配默认角色失败: %w", err)
	}

	a.logger.Info("Created local user for AD user", zap.String("username", adUser.Username))
	// Reload with Roles.Permissions for toUserDTO
	a.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, user.ID)
	return &user, nil
}

// updateADInfo updates AD information for an existing user
func (a *ADAuthenticator) updateADInfo(ctx context.Context, user *models.User, adUser *ADUser) {
	user.ADUsername = adUser.Username
	user.ADDN = adUser.DN
	user.ADGUID = adUser.ObjectGUID
	user.ADDepartment = adUser.Department
	user.ADUPN = adUser.UserPrincipalName
	a.db.WithContext(ctx).Save(user)
}

// Logout logs out a user by revoking their token
func (a *ADAuthenticator) Logout(token string) error {
	return a.tokenService.RevokeSession(token)
}

// ValidateToken validates a token and returns the associated user
func (a *ADAuthenticator) ValidateToken(ctx context.Context, token string) (*UserDTO, error) {
	// Use existing token validation logic (same as local)
	// Token validation doesn't depend on auth source
	claims, err := a.tokenService.ValidateToken(ctx, token)
	if err != nil {
		return nil, err
	}

	// Load user from database
	var user models.User
	if err := a.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, apperrors.ErrUserDisabled
	}

	return a.toUserDTO(&user), nil
}

// toUserDTO converts a User model to UserDTO
func (a *ADAuthenticator) toUserDTO(user *models.User) *UserDTO {
	// Extract role IDs
	roleIDs := make([]uint, 0, len(user.Roles))
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
	}

	dto := &UserDTO{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
		RoleIDs:  roleIDs,
		IsActive: user.IsActive,
	}

	// Check if user has any roles
	if len(user.Roles) > 0 {
		// Use first role for RoleName (backward compatibility)
		dto.RoleName = user.Roles[0].Name
		dto.IsAdmin = user.HasRole(models.RoleAdmin)

		// Collect permissions from all roles (OR logic)
		permMap := make(map[string]bool)
		for _, role := range user.Roles {
			for _, perm := range role.Permissions {
				permStr := perm.Resource + ":" + perm.Action
				permMap[permStr] = true
			}
		}
		for permStr := range permMap {
			dto.Permissions = append(dto.Permissions, permStr)
		}
	}
	return dto
}

// LookupUser looks up a user in Active Directory by username
// Returns ADUserLookupResult with user information if found
func (a *ADAuthenticator) LookupUser(username string) (*ADUserLookupResult, error) {
	// Connect to AD server
	conn, err := a.connectAD()
	if err != nil {
		a.logger.Error("AD connection failed during lookup", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("无法连接到域控服务器: %w", apperrors.ErrADUnreachable)
	}
	// STYLE-008: nil 防御——connectAD 失败时 conn 为 nil，defer Close 会 panic
	if conn != nil {
		defer func() { _ = conn.Close() }()
	}

	// Bind as admin to search for user
	err = conn.Bind(a.adConfig.BindDN, a.adConfig.Password)
	if err != nil {
		a.logger.Error("AD admin bind failed during lookup", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("域控管理员认证失败: %w", apperrors.ErrADConfigError)
	}

	// Search for user DN (prevent LDAP injection - use EscapeFilter)
	searchRequest := ldap.NewSearchRequest(
		a.adConfig.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		1, 0, false,
		fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", ldap.EscapeFilter(username)),
		[]string{"dn", "sAMAccountName", "mail", "displayName", "userAccountControl", "objectGUID", "department", "userPrincipalName"},
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		a.logger.Error("AD user search failed during lookup", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("搜索用户失败: %w", apperrors.ErrADUnreachable)
	}

	// Check if user was found
	if len(sr.Entries) == 0 {
		return &ADUserLookupResult{
			Found:    false,
			Username: username,
			Message:  "未在域控中找到该用户",
		}, nil
	}

	// Parse the LDAP entry
	adUser := a.parseLDAPEntry(sr.Entries[0])

	// Return the lookup result
	return &ADUserLookupResult{
		Found:      true,
		Username:   adUser.Username,
		Email:      adUser.Email,
		FullName:   adUser.DisplayName,
		Department: adUser.Department,
		UPN:        adUser.UserPrincipalName,
		DN:         adUser.DN,
		Disabled:   adUser.IsDisabled(),
	}, nil
}
