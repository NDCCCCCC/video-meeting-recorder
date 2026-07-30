package auth

import (
	"context"
	"errors"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/services/audit"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Authenticator 定义认证策略接口（STYLE-003：从 ad_config.go 迁移到本消费方包）。
// 保留命名以兼容所有现有调用方（service.go 字段类型 / ad_auth.go / local_auth.go / admin_handler.go）。
// 接口定义放在消费方包符合 Go 惯例（"accept interfaces, return structs"）。
type Authenticator interface {
	// Login authenticates a user and returns a login response.
	// ctx 用于把请求上下文（RequestID/TraceID）传递到审计日志，保证审计与
	// 请求链路可串联；为 nil 时调用方需自行降级处理。
	Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error)

	// Logout logs out a user by revoking their token
	Logout(token string) error

	// ValidateToken validates a token and returns the associated user
	ValidateToken(token string) (*UserDTO, error)

	// Name returns the authenticator name
	Name() string
}

// Service 认证服务
type Service struct {
	db                *gorm.DB
	tokenService      *SM4TokenService
	passwordValidator *PasswordValidator
	cfg               *config.Config
	logger            *zap.Logger
	rateLimiter       *RateLimiter
	auditLogger       *audit.AuditLogService

	// NEW: Strategy pattern - authenticators (per Spike 003)
	localAuth Authenticator
	adAuth   Authenticator
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
	Email    string `json:"email" binding:"omitempty,email"`
	FullName string `json:"full_name" binding:"omitempty,max=100"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
	User         *UserDTO `json:"user"`
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserDTO 用户DTO
type UserDTO struct {
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email"`
	FullName    string   `json:"full_name"`
	RoleIDs     []uint   `json:"role_ids"`
	RoleName    string   `json:"role_name,omitempty"`
	IsAdmin     bool     `json:"is_admin"`
	Permissions []string `json:"permissions"`
	IsActive    bool     `json:"is_active"`
}

// NewService 创建认证服务
func NewService(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Service {
	tokenService := NewSM4TokenService(cfg, db, logger)
	passwordValidator := NewPasswordValidator(8, true, true, true, false)
	rateLimiter := NewRateLimiterFromConfig(cfg)

	// Create authenticators (strategy pattern per Spike 003)
	localAuth := NewLocalAuthenticator(db, tokenService, &cfg.Auth, logger, rateLimiter)
	adAuth := NewADAuthenticator(&cfg.Auth.AD, db, tokenService, logger, cfg.Auth.SM4Secret)
	// Set live config reference for AD authenticator to get dynamic settings
	adAuth.SetLiveConfig(&cfg.Auth)

	return &Service{
		db:                db,
		tokenService:      tokenService,
		passwordValidator: passwordValidator,
		cfg:               cfg,
		logger:            logger,
		rateLimiter:       rateLimiter,
		localAuth:         localAuth,
		adAuth:           adAuth,
	}
}

// SetAuditService 设置审计服务
func (s *Service) SetAuditService(auditService *audit.AuditLogService) {
	s.auditLogger = auditService
	// Update local authenticator with audit service
	if localAuth, ok := s.localAuth.(*LocalAuthenticator); ok {
		localAuth.SetAuditService(auditService)
	}
}

// GetADAuthenticator returns the AD authenticator (nil if not available)
func (s *Service) GetADAuthenticator() *ADAuthenticator {
	if adAuth, ok := s.adAuth.(*ADAuthenticator); ok {
		return adAuth
	}
	return nil
}

// Login 用户登录
func (s *Service) Login(ctx context.Context, req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// Select authenticator based on config mode (per D-01, D-03)
	// Uses strategy pattern from Spike 003
	var authenticator Authenticator

	switch s.cfg.Auth.Mode {
	case "local":
		authenticator = s.localAuth
	case "ad":
		authenticator = s.adAuth
	default:
		s.logger.Error("Invalid authentication mode", zap.String("mode", s.cfg.Auth.Mode))
		return nil, errors.New("无效的认证模式")
	}

	// Route to selected authenticator (no fallback per D-04)
	return authenticator.Login(ctx, req, ipAddress, userAgent)
}

// CheckIPRestriction 检查用户IP限制
func (s *Service) CheckIPRestriction(ctx context.Context, user *models.User, clientIP string) error {
	validator := &IPValidator{}

	// Collect all allowed IPs from user and all roles
	allowedIPs := make([]string, 0)

	// Add user's IP restrictions
	if len(user.GetAllowedIPs()) > 0 {
		allowedIPs = append(allowedIPs, user.GetAllowedIPs()...)
	}

	// Add IP restrictions from all roles (OR logic per D-02)
	for _, role := range user.Roles {
		if len(role.GetAllowedIPs()) > 0 {
			allowedIPs = append(allowedIPs, role.GetAllowedIPs()...)
		}
	}

	// If no restrictions, allow all IPs
	if len(allowedIPs) == 0 {
		return nil
	}

	// Debug logging for IP restriction check
	if s.logger != nil {
		s.logger.Info("IP restriction check",
			zap.String("client_ip", clientIP),
			zap.Uint("user_id", user.ID),
			zap.Strings("allowed_ips", allowedIPs),
		)
	}

	// Check if client IP is allowed
	allowed, err := validator.IsIPAllowed(clientIP, allowedIPs)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("IP validation failed",
				zap.String("client_ip", clientIP),
				zap.Uint("user_id", user.ID),
				zap.Error(err),
			)
		}
		// Log validation failure to audit trail per D-14
		if s.auditLogger != nil {
			s.auditLogger.LogOperation(context.Background(), &audit.LogOperationRequest{
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
		// Log IP restriction failure to audit trail per D-14
		if s.auditLogger != nil {
			s.auditLogger.LogOperation(context.Background(), &audit.LogOperationRequest{
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

// RefreshToken 刷新Token
func (s *Service) RefreshToken(refreshToken string) (*RefreshTokenResponse, error) {
	tokenPair, err := s.tokenService.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(s.tokenService.expireHours * 3600),
	}, nil
}

// Logout 用户登出
func (s *Service) Logout(token string) error {
	// Use local authenticator for logout (both modes use same token service)
	return s.localAuth.Logout(token)
}

// LogoutAll 登出所有设备
func (s *Service) LogoutAll(userID uint) error {
	return s.tokenService.RevokeUserSessions(userID)
}

// ChangePassword 修改密码
func (s *Service) ChangePassword(userID uint, req *ChangePasswordRequest) error {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	// 验证旧密码
	if !user.CheckPassword(req.OldPassword) {
		return errors.New("原密码错误")
	}

	// 验证新密码强度
	result := s.passwordValidator.Validate(req.NewPassword)
	if !result.Valid {
		return errors.New(result.Errors[0])
	}

	// 设置新密码
	if err := user.SetPassword(req.NewPassword); err != nil {
		return err
	}

	// 保存
	return s.db.Save(&user).Error
}

// ValidatePassword 验证密码强度
func (s *Service) ValidatePassword(password string) *ValidationResult {
	return s.passwordValidator.Validate(password)
}

// GetUserByID 根据ID获取用户
func (s *Service) GetUserByID(userID uint) (*UserDTO, error) {
	var user models.User
	if err := s.db.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return s.toUserDTO(&user), nil
}

// toUserDTO 转换为UserDTO
func (s *Service) toUserDTO(user *models.User) *UserDTO {
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

		// Collect permissions from all roles (OR logic per D-07)
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
