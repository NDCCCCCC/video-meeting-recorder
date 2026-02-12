package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// JWTService JWT服务
type JWTService struct {
	secretKey     []byte
	issuer        string
	expireHours   int
	refreshExpire int
	maxSession    int
	db            *gorm.DB
	logger        *zap.Logger
}

// Claims JWT声明
type Claims struct {
	UserID      uint     `json:"user_id"`
	Username    string   `json:"username"`
	RoleID      uint     `json:"role_id"`
	Permissions []string `json:"permissions"` // 用户权限列表
	IsAdmin     bool     `json:"is_admin"`    // 是否是管理员
	TokenType   string   `json:"token_type"` // access | refresh
	jwt.RegisteredClaims
}

// TokenPair Token对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// NewJWTService 创建JWT服务
func NewJWTService(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *JWTService {
	return &JWTService{
		secretKey:     []byte(cfg.Auth.JWTSecret),
		issuer:        "record_v2",
		expireHours:   int(cfg.Auth.AccessTokenDuration.Hours()),
		refreshExpire: int(cfg.Auth.RefreshTokenDuration.Hours()),
		maxSession:    int(cfg.Auth.MaxSessionDuration.Hours()),
		db:            db,
		logger:        logger,
	}
}

// GenerateTokenPair 生成Token对
func (s *JWTService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.expireHours) * time.Hour)

	// 生成Access Token
	accessToken, err := s.generateToken(user, "access", time.Duration(s.expireHours)*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("生成access token失败: %w", err)
	}

	// 生成Refresh Token
	refreshToken, err := s.generateToken(user, "refresh", time.Duration(s.refreshExpire)*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("生成refresh token失败: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// generateToken 生成Token
func (s *JWTService) generateToken(user *models.User, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()

	// 加载用户权限
	var permissions []string
	isAdmin := false

	if user.Role != nil {
		isAdmin = user.Role.Name == models.RoleAdmin
		for _, perm := range user.Role.Permissions {
			// 构造权限字符串: resource:action
			permStr := perm.Resource + ":" + perm.Action
			permissions = append(permissions, permStr)
		}
	}

	claims := &Claims{
		UserID:      user.ID,
		Username:    user.Username,
		RoleID:      user.RoleID,
		Permissions: permissions,
		IsAdmin:     isAdmin,
		TokenType:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   user.Username,
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ValidateToken 验证Token
func (s *JWTService) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// 检查token类型
		if claims.TokenType != "access" {
			return nil, errors.New("invalid token type")
		}
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateRefreshToken 验证Refresh Token
func (s *JWTService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}

	if claims.TokenType != "refresh" {
		return nil, errors.New("invalid token type")
	}

	return claims, nil
}

// RefreshAccessToken 刷新Access Token
func (s *JWTService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	// 验证refresh token
	claims, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 加载用户信息（预加载权限）
	var user models.User
	if err := s.db.Preload("Role.Permissions").First(&user, claims.UserID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	// 检查用户状态
	if !user.IsActive {
		return nil, errors.New("user is inactive")
	}

	// 生成新的token对
	return s.GenerateTokenPair(&user)
}

// CreateSession 创建会话记录
func (s *JWTService) CreateSession(userID uint, token string, ipAddress, userAgent string, expiresAt time.Time) error {
	session := &models.Session{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		IsActive:  true,
	}

	return s.db.Create(session).Error
}

// RevokeSession 撤销会话
func (s *JWTService) RevokeSession(token string) error {
	return s.db.Model(&models.Session{}).
		Where("token = ?", token).
		Update("is_active", false).Error
}

// RevokeUserSessions 撤销用户所有会话
func (s *JWTService) RevokeUserSessions(userID uint) error {
	return s.db.Model(&models.Session{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", false).Error
}

// CleanExpiredSessions 清理过期会话
func (s *JWTService) CleanExpiredSessions() error {
	return s.db.Model(&models.Session{}).
		Where("expires_at < ?", time.Now()).
		Delete(&models.Session{}).Error
}

// GenerateRandomSecret 生成随机密钥
func GenerateRandomSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
