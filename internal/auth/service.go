package auth

import (
	"errors"
	"time"

	"github.com/cpic/record_v2/internal/config"
	"github.com/cpic/record_v2/internal/models"
	"github.com/cpic/record_v2/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 认证服务
type Service struct {
	db                *gorm.DB
	tokenService      *SM4TokenService
	passwordValidator *PasswordValidator
	cfg               *config.Config
	logger            *zap.Logger
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

	return &Service{
		db:                db,
		tokenService:      tokenService,
		passwordValidator: passwordValidator,
		cfg:               cfg,
		logger:            logger,
	}
}

// Login 用户登录
func (s *Service) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// 1. 查找用户（预加载角色和权限）
	var user models.User
	err := s.db.Preload("Roles.Permissions").Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	// 2. 尝试解密密码（如果已加密）
	passwordToCheck := req.Password
	if utils.IsEncryptedPassword(req.Password) {
		decrypted, err := utils.DecryptPasswordECB(req.Password, s.cfg.Auth.SM4Secret)
		if err != nil {
			s.logger.Warn("Failed to decrypt password",
				zap.String("username", req.Username),
				zap.Error(err),
			)
			return nil, errors.New("密码格式错误")
		}
		passwordToCheck = decrypted
		s.logger.Debug("Password decrypted for login",
			zap.String("username", req.Username),
		)
	}

	// 3. 检查密码（使用解密后的密码）
	if !user.CheckPassword(passwordToCheck) {
		return nil, errors.New("用户名或密码错误")
	}

	// 4. 检查用户状态
	if !user.IsActive {
		return nil, errors.New("用户已被禁用")
	}

	// 5. 生成Token
	tokenPair, err := s.tokenService.GenerateTokenPair(&user)
	if err != nil {
		return nil, err
	}

	// 6. 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Save(&user)

	// 7. 创建session记录（可选，用于token撤销）
	if err := s.tokenService.CreateSession(user.ID, tokenPair.AccessToken, ipAddress, userAgent, tokenPair.ExpiresAt); err != nil {
		s.logger.Warn("Failed to create session", zap.Error(err))
	}

	// 8. 返回响应
	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(s.tokenService.expireHours * 3600),
		User:         s.toUserDTO(&user),
	}, nil
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
	// 撤销token
	if err := s.tokenService.RevokeSession(token); err != nil {
		s.logger.Warn("Failed to revoke session", zap.Error(err))
	}
	return nil
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
