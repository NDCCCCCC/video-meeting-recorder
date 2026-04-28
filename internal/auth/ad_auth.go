package auth

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/utils"
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

// NewADAuthenticator creates a new AD authenticator
func NewADAuthenticator(cfg *config.ADAuthConfig, db *gorm.DB, tokenService *SM4TokenService, logger *zap.Logger) *ADAuthenticator {
	return &ADAuthenticator{
		adConfig:     cfg,
		db:           db,
		tokenService: tokenService,
		logger:       logger,
	}
}

// Name returns the authenticator name
func (a *ADAuthenticator) Name() string {
	return "ad"
}

// Login authenticates a user using AD authentication
func (a *ADAuthenticator) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Step 1: Connect to AD server
	conn, err := a.connectAD()
	if err != nil {
		a.logger.Error("AD connection failed", zap.Error(err))
		return nil, fmt.Errorf("无法连接到域控服务器，请检查网络和配置") // per D-18
	}
	defer conn.Close()

	// Step 2: Bind as admin to search for user
	err = conn.Bind(a.adConfig.BindDN, a.adConfig.Password)
	if err != nil {
		a.logger.Error("AD admin bind failed", zap.Error(err))
		return nil, fmt.Errorf("域控管理员认证失败")
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
		a.logger.Error("AD user search failed", zap.Error(err))
		return nil, fmt.Errorf("搜索用户失败")
	}

	if len(sr.Entries) == 0 {
		return nil, errors.New("域控账号不存在，请联系管理员确认") // per D-20
	}

	userDN := sr.Entries[0].DN
	adUser := a.parseLDAPEntry(sr.Entries[0])

	// Step 4: Check if account is disabled
	if adUser.IsDisabled() {
		return nil, errors.New("域控账号已禁用")
	}

	// Step 5: Bind as user to authenticate (verify password)
	err = conn.Bind(userDN, req.Password)
	if err != nil {
		a.logger.Warn("AD user bind failed", zap.String("username", req.Username))
		return nil, errors.New("域控密码错误") // per D-20
	}

	// Step 6: Find or create local user (transparent mapping per D-06, D-08)
	localUser, err := a.findOrCreateLocalUser(adUser)
	if err != nil {
		a.logger.Error("AD user mapping failed", zap.Error(err))
		return nil, errors.New("用户映射失败")
	}

	// Step 7: Generate token using existing token service
	tokenPair, err := a.tokenService.GenerateTokenPair(localUser)
	if err != nil {
		a.logger.Error("AD token generation failed", zap.Error(err))
		return nil, errors.New("登录失败，请稍后重试")
	}

	// Step 8: Update last login times
	now := time.Now()
	localUser.LastLoginAt = &now
	localUser.LastADLogin = &now
	a.db.Save(localUser)

	// Step 9: Create session
	a.tokenService.CreateSession(localUser.ID, tokenPair.AccessToken, ipAddress, userAgent, tokenPair.ExpiresAt)

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

// parseLDAPEntry parses an LDAP entry to ADUser struct
func (a *ADAuthenticator) parseLDAPEntry(entry *ldap.Entry) *ADUser {
	userAccountControl := uint32(0)
	if attr := entry.GetAttributeValue("userAccountControl"); attr != "" {
		fmt.Sscanf(attr, "%d", &userAccountControl)
	}

	return &ADUser{
		Username:          entry.GetAttributeValue("sAMAccountName"),
		DN:                entry.DN,
		ObjectGUID:        entry.GetAttributeValue("objectGUID"),
		Email:             entry.GetAttributeValue("mail"),
		DisplayName:       entry.GetAttributeValue("displayName"),
		Department:        entry.GetAttributeValue("department"),
		UserPrincipalName: entry.GetAttributeValue("userPrincipalName"),
		UserAccountControl: userAccountControl,
	}
}

// findOrCreateLocalUser finds an existing local user or creates a new one
func (a *ADAuthenticator) findOrCreateLocalUser(adUser *ADUser) (*models.User, error) {
	// First, try to find existing user by ad_guid or username
	var user models.User
	err := a.db.Where("ad_guid = ? OR username = ?", adUser.ObjectGUID, adUser.Username).First(&user).Error

	if err == nil {
		// Found existing user, update AD information
		a.updateADInfo(&user, adUser)
		return &user, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

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
	err = a.db.Where("name = ?", models.RoleViewer).First(&defaultRole).Error
	if err != nil {
		a.logger.Error("Default viewer role not found", zap.Error(err))
		return nil, fmt.Errorf("系统配置错误：默认角色不存在")
	}
	user.Roles = []models.Role{defaultRole}

	if err := a.db.Create(&user).Error; err != nil {
		return nil, err
	}

	a.logger.Info("Created local user for AD user", zap.String("username", adUser.Username))
	return &user, nil
}

// updateADInfo updates AD information for an existing user
func (a *ADAuthenticator) updateADInfo(user *models.User, adUser *ADUser) {
	user.ADUsername = adUser.Username
	user.ADDN = adUser.DN
	user.ADGUID = adUser.ObjectGUID
	user.ADDepartment = adUser.Department
	user.ADUPN = adUser.UserPrincipalName
	a.db.Save(user)
}

// Logout logs out a user by revoking their token
func (a *ADAuthenticator) Logout(token string) error {
	return a.tokenService.RevokeSession(token)
}

// ValidateToken validates a token and returns the associated user
func (a *ADAuthenticator) ValidateToken(token string) (*UserDTO, error) {
	// Use existing token validation logic (same as local)
	// Token validation doesn't depend on auth source
	claims, err := a.tokenService.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// Load user from database
	var user models.User
	if err := a.db.Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, errors.New("用户已被禁用")
	}

	return a.toUserDTO(&user), nil
}

// toUserDTO converts a User model to UserDTO
func (a *ADAuthenticator) toUserDTO(user *models.User) *UserDTO {
	// Extract role IDs
	var roleIDs []uint
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
	}

	dto := &UserDTO{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		FullName:  user.FullName,
		RoleIDs:   roleIDs,
		IsActive:  user.IsActive,
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
