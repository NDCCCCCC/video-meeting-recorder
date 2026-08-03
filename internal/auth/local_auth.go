package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/utils"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
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
func (a *LocalAuthenticator) Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// 1. 查找用户（预加载角色和权限）
	var user models.User
	err := a.db.WithContext(ctx).Preload("Roles.Permissions").Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			a.logger.Warn("Login failed: user not found", zap.String("username", req.Username))
			a.logLoginFailure(ctx, req.Username, ipAddress, userAgent, "用户不存在")
			return nil, fmt.Errorf("用户名或密码错误: %w", apperrors.ErrUnauthorized)
		}
		a.logger.Error("Login failed: database error", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("登录失败，请稍后重试: %w", apperrors.ErrInternal)
	}

	// 2. 检查解密失败速率限制
	if a.rateLimiter != nil && a.rateLimiter.ShouldBlock(req.Username) {
		a.logger.Warn("Decrypt failure rate limit exceeded",
			zap.String("ip", ipAddress),
		)
		a.logLoginFailure(ctx, req.Username, ipAddress, userAgent, "登录频率超限")
		return nil, errors.New("登录尝试过于频繁，请稍后再试")
	}

	// 3. 验证 SM4 密钥配置（如果使用加密密码）
	if a.cfg != nil && utils.IsEncryptedPassword(req.Password) {
		if a.cfg.SM4Secret != "" {
			if err := utils.ValidateSM4Secret(a.cfg.SM4Secret); err != nil {
				a.logger.Error("Invalid SM4 secret configuration", zap.Error(err), response.SentinelField(err))
				return nil, fmt.Errorf("系统配置错误: %w", apperrors.ErrADConfigError)
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
			a.logLoginFailure(ctx, req.Username, ipAddress, userAgent, "密码解密失败")
			return nil, fmt.Errorf("用户名或密码错误: %w", apperrors.ErrUnauthorized)
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
		a.logLoginFailure(ctx, req.Username, ipAddress, userAgent, "密码错误")
		return nil, fmt.Errorf("用户名或密码错误: %w", apperrors.ErrUnauthorized)
	}

	// 登录成功，清除失败记录
	if a.rateLimiter != nil {
		a.rateLimiter.Clear(req.Username)
	}

	// 6. 检查用户状态
	if !user.IsActive {
		a.logger.Warn("Login failed: user inactive", zap.String("username", req.Username))
		a.logLoginFailure(ctx, req.Username, ipAddress, userAgent, "用户已被禁用")
		return nil, fmt.Errorf("用户已被禁用: %w", apperrors.ErrUserDisabled)
	}

	// 7. 检查IP限制
	if err := a.checkIPRestrictions(ctx, &user, ipAddress); err != nil {
		a.logger.Warn("Login failed: IP restriction", zap.String("username", req.Username), zap.String("ip", ipAddress))
		return nil, err
	}

	// 8. 生成Token
	tokenPair, err := a.tokenService.GenerateTokenPair(&user)
	if err != nil {
		a.logger.Error("Login failed: token generation", zap.Error(err), response.SentinelField(err))
		return nil, fmt.Errorf("登录失败，请稍后重试: %w", apperrors.ErrInternal)
	}

	// 9. 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	if err := a.db.WithContext(ctx).Save(&user).Error; err != nil {
		a.logger.Error("Login: failed to update last login", zap.Error(err), response.SentinelField(err))
	}

	// 10. 创建session记录（按 RefreshToken 存储，便于 RefreshAccessToken 中的宽限期查找）
	if err := a.tokenService.CreateSession(user.ID, tokenPair.RefreshToken, ipAddress, userAgent, tokenPair.ExpiresAt); err != nil {
		a.logger.Warn("Failed to create session", zap.Error(err), response.SentinelField(err))
	}

	// 11. Audit log
	if a.auditLogger != nil {
		_ = a.auditLogger.LogOperation(ctx, &audit.LogOperationRequest{
			UserID:    user.ID,
			Username:  user.Username,
			Action:    models.ActionLogin,
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
func (a *LocalAuthenticator) ValidateToken(ctx context.Context, token string) (*UserDTO, error) {
	// Use existing token validation logic from token service
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

// checkIPRestrictions checks IP restrictions for the user
func (a *LocalAuthenticator) checkIPRestrictions(ctx context.Context, user *models.User, clientIP string) error {
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
				response.SentinelField(err),
			)
		}
		// Log validation failure to audit trail
		if a.auditLogger != nil {
			_ = a.auditLogger.LogOperation(ctx, &audit.LogOperationRequest{
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
			_ = a.auditLogger.LogOperation(ctx, &audit.LogOperationRequest{
				UserID:    user.ID,
				Username:  user.Username,
				Action:    models.ActionIPRestrictionFailed,
				Module:    models.ModuleUser,
				IPAddress: clientIP,
				Status:    models.StatusFailure,
				ErrorMsg:  "您的IP地址不在允许列表中",
			})
		}
		return fmt.Errorf("您的IP地址不在允许列表中: %w", apperrors.ErrForbidden)
	}

	return nil
}

// logLoginFailure 记录登录失败审计（统一封装，避免重复样板代码）。
// 所有失败分支（用户不存在/密码错/用户禁用/速率限制/解密失败）共享此路径，
// 这样安全分析时不必逐个搜调用点。
func (a *LocalAuthenticator) logLoginFailure(ctx context.Context, username, ipAddress, userAgent, errMsg string) {
	if a.auditLogger == nil {
		return
	}
	_ = a.auditLogger.LogOperation(ctx, &audit.LogOperationRequest{
		Username:  username,
		Action:    models.ActionLogin,
		Module:    models.ModuleUser,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Status:    models.StatusFailure,
		ErrorMsg:  errMsg,
	})
}

// toUserDTO converts a User model to UserDTO
func (a *LocalAuthenticator) toUserDTO(user *models.User) *UserDTO {
	// Extract role IDs
	var roleIDs []uint
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
