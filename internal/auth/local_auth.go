package auth

import (
	"context"
	"errors"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/services/audit"
	"github.com/cpic/record_v2/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LocalAuthenticator 本地认证器
type LocalAuthenticator struct {
	db           *gorm.DB
	tokenService *SM4TokenService
	cfg          *config.AuthConfig
	logger       *zap.Logger
	rateLimiter  *RateLimiter
	auditLogger  *audit.AuditLogService
}

// NewLocalAuthenticator creates a new local authenticator
func NewLocalAuthenticator(db *gorm.DB, tokenService *SM4TokenService, cfg *config.AuthConfig, logger *zap.Logger, rateLimiter *RateLimiter) *LocalAuthenticator {
	return &LocalAuthenticator{
		db:           db,
		tokenService: tokenService,
		cfg:          cfg,
		logger:       logger,
		rateLimiter:  rateLimiter,
	}
}

// Name returns the authenticator name
func (a *LocalAuthenticator) Name() string {
	return "local"
}

// SetAuditService sets the audit service
func (a *LocalAuthenticator) SetAuditService(auditLogger *audit.AuditLogService) {
	a.auditLogger = auditLogger
}

// Login authenticates a user using local authentication
func (a *LocalAuthenticator) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// 1. 查找用户（预加载角色和权限）
	var user models.User
	err := a.db.Preload("Roles.Permissions").Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			a.logger.Warn("Login failed: user not found", zap.String("username", req.Username))
			return nil, errors.New("用户名或密码错误")
		}
		a.logger.Error("Login failed: database error", zap.Error(err))
		return nil, errors.New("登录失败，请稍后重试")
	}

	// 2. 检查解密失败速率限制
	if a.rateLimiter != nil && a.rateLimiter.ShouldBlock(req.Username) {
		a.logger.Warn("Decrypt failure rate limit exceeded",
			zap.String("ip", ipAddress),
		)
		return nil, errors.New("登录尝试过于频繁，请稍后再试")
	}

	// 3. 验证 SM4 密钥配置（如果使用加密密码）
	if a.cfg != nil && utils.IsEncryptedPassword(req.Password) {
		if a.cfg.SM4Secret != "" {
			if err := utils.ValidateSM4Secret(a.cfg.SM4Secret); err != nil {
				a.logger.Error("Invalid SM4 secret configuration", zap.Error(err))
				return nil, errors.New("系统配置错误")
			}
		}
	}

	// 4. 尝试解密密码（如果已加密）
	passwordToCheck := req.Password
	if a.cfg != nil && utils.IsEncryptedPassword(req.Password) {
		decrypted, err := utils.DecryptPasswordECB(req.Password, a.cfg.SM4Secret)
		if err != nil {
			// 记录解密失败
			if a.rateLimiter != nil {
				a.rateLimiter.RecordFailure(req.Username)
			}

			a.logger.Warn("Failed to decrypt password")
			return nil, errors.New("用户名或密码错误")
		}
		passwordToCheck = decrypted
		a.logger.Debug("Password decrypted for login")
	}

	// 5. 检查密码（使用解密后的密码）
	if !user.CheckPassword(passwordToCheck) {
		// 记录密码验证失败
		if a.rateLimiter != nil {
			a.rateLimiter.RecordFailure(req.Username)
		}
		a.logger.Warn("Login failed: invalid password", zap.String("username", req.Username))
		return nil, errors.New("用户名或密码错误")
	}

	// 登录成功，清除失败记录
	if a.rateLimiter != nil {
		a.rateLimiter.Clear(req.Username)
	}

	// 6. 检查用户状态
	if !user.IsActive {
		a.logger.Warn("Login failed: user inactive", zap.String("username", req.Username))
		return nil, errors.New("用户已被禁用")
	}

	// 7. 检查IP限制
	if err := a.checkIPRestrictions(&user, ipAddress); err != nil {
		a.logger.Warn("Login failed: IP restriction", zap.String("username", req.Username), zap.String("ip", ipAddress))
		return nil, err
	}

	// 8. 生成Token
	tokenPair, err := a.tokenService.GenerateTokenPair(&user)
	if err != nil {
		a.logger.Error("Login failed: token generation", zap.Error(err))
		return nil, errors.New("登录失败，请稍后重试")
	}

	// 9. 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	if err := a.db.Save(&user).Error; err != nil {
		a.logger.Error("Login: failed to update last login", zap.Error(err))
	}

	// 10. 创建session记录
	if err := a.tokenService.CreateSession(user.ID, tokenPair.AccessToken, ipAddress, userAgent, tokenPair.ExpiresAt); err != nil {
		a.logger.Warn("Failed to create session", zap.Error(err))
	}

	// 11. Audit log
	if a.auditLogger != nil {
		a.auditLogger.LogOperation(context.Background(), &audit.LogOperationRequest{
			UserID:    user.ID,
			Username:  user.Username,
			Action:    "login",
			Module:    models.ModuleUser,
			IPAddress: ipAddress,
			Status:    models.StatusSuccess,
		})
	}

	a.logger.Info("User logged in", zap.String("username", req.Username), zap.Uint("user_id", user.ID))

	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(a.tokenService.expireHours * 3600),
		User:         a.toUserDTO(&user),
	}, nil
}

// Logout logs out a user by revoking their token
func (a *LocalAuthenticator) Logout(token string) error {
	return a.tokenService.RevokeSession(token)
}

// ValidateToken validates a token and returns the associated user
func (a *LocalAuthenticator) ValidateToken(token string) (*UserDTO, error) {
	// Use existing token validation logic from token service
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

// checkIPRestrictions checks IP restrictions for the user
func (a *LocalAuthenticator) checkIPRestrictions(user *models.User, clientIP string) error {
	validator := &IPValidator{}

	// Collect all allowed IPs from user and all roles
	allowedIPs := make([]string, 0)

	// Add user's IP restrictions
	if len(user.GetAllowedIPs()) > 0 {
		allowedIPs = append(allowedIPs, user.GetAllowedIPs()...)
	}

	// Add IP restrictions from all roles (OR logic)
	for _, role := range user.Roles {
		if len(role.GetAllowedIPs()) > 0 {
			allowedIPs = append(allowedIPs, role.GetAllowedIPs()...)
		}
	}

	// If no restrictions, allow all IPs
	if len(allowedIPs) == 0 {
		return nil
	}

	// Check if client IP is allowed
	allowed, err := validator.IsIPAllowed(clientIP, allowedIPs)
	if err != nil {
		if a.logger != nil {
			a.logger.Warn("IP validation failed",
				zap.String("client_ip", clientIP),
				zap.Uint("user_id", user.ID),
				zap.Error(err),
			)
		}
		// Log validation failure to audit trail
		if a.auditLogger != nil {
			a.auditLogger.LogOperation(context.Background(), &audit.LogOperationRequest{
				UserID:    user.ID,
				Username:  user.Username,
				Action:    models.ActionIPRestrictionFailed,
				Module:    models.ModuleUser,
				IPAddress: clientIP,
				Status:    models.StatusFailure,
				ErrorMsg:  "IP地址验证失败: " + err.Error(),
			})
		}
		return errors.New("IP地址验证失败")
	}

	if !allowed {
		// Log IP restriction failure to audit trail
		if a.auditLogger != nil {
			a.auditLogger.LogOperation(context.Background(), &audit.LogOperationRequest{
				UserID:    user.ID,
				Username:  user.Username,
				Action:    models.ActionIPRestrictionFailed,
				Module:    models.ModuleUser,
				IPAddress: clientIP,
				Status:    models.StatusFailure,
				ErrorMsg:  "您的IP地址不在允许列表中",
			})
		}
		return errors.New("您的IP地址不在允许列表中")
	}

	return nil
}

// toUserDTO converts a User model to UserDTO
func (a *LocalAuthenticator) toUserDTO(user *models.User) *UserDTO {
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
