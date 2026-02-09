package auth

import (
	"errors"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service 认证服务
type Service struct {
	db                *gorm.DB
	jwtService        *JWTService
	passwordValidator *PasswordValidator
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
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	RoleID   uint   `json:"role_id"`
	RoleName string `json:"role_name,omitempty"`
	IsActive bool   `json:"is_active"`
}

// NewService 创建认证服务
func NewService(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *Service {
	jwtService := NewJWTService(cfg, db, logger)
	passwordValidator := NewPasswordValidator(8, true, true, true, false)

	return &Service{
		db:                db,
		jwtService:        jwtService,
		passwordValidator: passwordValidator,
		logger:            logger,
	}
}

// Login 用户登录
func (s *Service) Login(req *LoginRequest, ipAddress, userAgent string) (*LoginResponse, error) {
	// 1. 查找用户
	var user models.User
	err := s.db.Preload("Role").Where("username = ?", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	// 2. 检查密码
	if !user.CheckPassword(req.Password) {
		return nil, errors.New("用户名或密码错误")
	}

	// 3. 检查用户状态
	if !user.IsActive {
		return nil, errors.New("用户已被禁用")
	}

	// 4. 生成JWT token
	tokenPair, err := s.jwtService.GenerateTokenPair(&user)
	if err != nil {
		return nil, err
	}

	// 5. 更新最后登录时间
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Save(&user)

	// 6. 创建session记录（可选，用于token撤销）
	if err := s.jwtService.CreateSession(user.ID, tokenPair.AccessToken, ipAddress, userAgent, tokenPair.ExpiresAt); err != nil {
		s.logger.Warn("Failed to create session", zap.Error(err))
	}

	// 7. 返回响应
	return &LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(s.jwtService.expireHours * 3600),
		User:         s.toUserDTO(&user),
	}, nil
}

// RefreshToken 刷新Token
func (s *Service) RefreshToken(refreshToken string) (*RefreshTokenResponse, error) {
	tokenPair, err := s.jwtService.RefreshAccessToken(refreshToken)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(s.jwtService.expireHours * 3600),
	}, nil
}

// Logout 用户登出
func (s *Service) Logout(token string) error {
	// 撤销token
	if err := s.jwtService.RevokeSession(token); err != nil {
		s.logger.Warn("Failed to revoke session", zap.Error(err))
	}
	return nil
}

// LogoutAll 登出所有设备
func (s *Service) LogoutAll(userID uint) error {
	return s.jwtService.RevokeUserSessions(userID)
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
	if err := s.db.Preload("Role").First(&user, userID).Error; err != nil {
		return nil, err
	}
	return s.toUserDTO(&user), nil
}

// toUserDTO 转换为UserDTO
func (s *Service) toUserDTO(user *models.User) *UserDTO {
	dto := &UserDTO{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
		RoleID:   user.RoleID,
		IsActive: user.IsActive,
	}
	if user.Role != nil {
		dto.RoleName = user.Role.Name
	}
	return dto
}
