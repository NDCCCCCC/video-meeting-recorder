package auth

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/tjfoc/gmsm/sm4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SM4TokenService 基于SM4-GCM的Token服务，替代JWT。
// Token格式: base64url(nonce[12] + SM4-GCM(claims_json) + authTag[16])
// SM4-GCM同时提供加密（payload不可读）和认证（防篡改）。
type SM4TokenService struct {
	gcm            cipher.AEAD // 预初始化的SM4-GCM实例，避免每次加解密重复创建
	issuer         string
	expireHours    int
	refreshExpire  int
	maxSession     int
	db             *gorm.DB
	logger         *zap.Logger
}

// Claims Token声明
type Claims struct {
	UserID      uint     `json:"uid"`
	Username    string   `json:"sub"`
	RoleID      uint     `json:"rid"`
	Permissions []string `json:"perms"`
	IsAdmin     bool     `json:"adm"`
	TokenType   string   `json:"tt"` // "access" | "refresh"
	IssuedAt    int64    `json:"iat"`
	ExpiresAt   int64    `json:"exp"`
	NotBefore   int64    `json:"nbf"`
	Issuer      string   `json:"iss"`
}

// TokenPair Token对
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// NewSM4TokenService 创建SM4 Token服务。
// 使用配置中的SM4Secret作为SM4密钥源，通过SHA256派生16字节密钥。
func NewSM4TokenService(cfg *config.Config, db *gorm.DB, logger *zap.Logger) *SM4TokenService {
	key := deriveSM4Key(cfg.Auth.SM4Secret)

	block, err := sm4.NewCipher(key)
	if err != nil {
		logger.Fatal("创建SM4加密器失败", zap.Error(err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		logger.Fatal("创建GCM模式失败", zap.Error(err))
	}

	return &SM4TokenService{
		gcm:           gcm,
		issuer:        "record_v2",
		expireHours:   int(cfg.Auth.AccessTokenDuration.Hours()),
		refreshExpire: int(cfg.Auth.RefreshTokenDuration.Hours()),
		maxSession:    int(cfg.Auth.MaxSessionDuration.Hours()),
		db:            db,
		logger:        logger,
	}
}

// deriveSM4Key 使用SHA256从密钥字符串派生16字节SM4密钥。
// 使用哈希函数确保任意长度输入都能产生高质量密钥。
func deriveSM4Key(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:16]
}

// GenerateTokenPair 生成Access Token和Refresh Token对
func (s *SM4TokenService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.expireHours) * time.Hour)

	accessToken, err := s.generateToken(user, "access", time.Duration(s.expireHours)*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("生成access token失败: %w", err)
	}

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

// generateToken 生成单个Token
func (s *SM4TokenService) generateToken(user *models.User, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()

	var permissions []string
	isAdmin := false
	if user.Role != nil {
		isAdmin = user.Role.Name == models.RoleAdmin
		for _, perm := range user.Role.Permissions {
			permissions = append(permissions, perm.Resource+":"+perm.Action)
		}
	}

	claims := &Claims{
		UserID:      user.ID,
		Username:    user.Username,
		RoleID:      user.RoleID,
		Permissions: permissions,
		IsAdmin:     isAdmin,
		TokenType:   tokenType,
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(duration).Unix(),
		NotBefore:   now.Unix(),
		Issuer:      s.issuer,
	}

	return s.encryptToken(claims)
}

// encryptToken 使用SM4-GCM加密Claims。
// 输出格式: base64url(nonce || ciphertext || tag)
func (s *SM4TokenService) encryptToken(claims *Claims) (string, error) {
	plaintext, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("序列化claims失败: %w", err)
	}

	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成nonce失败: %w", err)
	}

	// Seal将nonce附加到ciphertext前面，末尾附加auth tag
	ciphertext := s.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// decryptToken 解密并解析Token
func (s *SM4TokenService) decryptToken(tokenString string) (*Claims, error) {
	data, err := base64.RawURLEncoding.DecodeString(tokenString)
	if err != nil {
		return nil, errors.New("invalid token")
	}

	nonceSize := s.gcm.NonceSize()
	if len(data) < nonceSize+s.gcm.Overhead() {
		return nil, errors.New("invalid token")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("invalid token")
	}

	var claims Claims
	if err := json.Unmarshal(plaintext, &claims); err != nil {
		return nil, errors.New("invalid token")
	}

	return &claims, nil
}

// ValidateToken 验证Access Token，返回解析后的Claims
func (s *SM4TokenService) ValidateToken(tokenString string) (*Claims, error) {
	claims, err := s.decryptToken(tokenString)
	if err != nil {
		return nil, err
	}

	if err := s.validateClaims(claims, "access"); err != nil {
		return nil, err
	}

	return claims, nil
}

// ValidateRefreshToken 验证Refresh Token
func (s *SM4TokenService) ValidateRefreshToken(tokenString string) (*Claims, error) {
	claims, err := s.decryptToken(tokenString)
	if err != nil {
		return nil, err
	}

	if err := s.validateClaims(claims, "refresh"); err != nil {
		return nil, err
	}

	return claims, nil
}

// validateClaims 校验claims的通用逻辑
func (s *SM4TokenService) validateClaims(claims *Claims, expectedType string) error {
	if claims.TokenType != expectedType {
		return errors.New("invalid token type")
	}

	now := time.Now().Unix()
	if now > claims.ExpiresAt {
		return errors.New("token已过期")
	}
	if now < claims.NotBefore {
		return errors.New("token未生效")
	}
	return nil
}

// RefreshAccessToken 使用Refresh Token刷新，生成新的Token对。
// 实现Token轮换：验证后立即撤销旧token，防止重放攻击。
// 如果检测到已撤销的token被重放，撤销该用户所有会话。
func (s *SM4TokenService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// Token轮换：检查该refresh token是否已被使用过
	var session models.Session
	result := s.db.Where("token = ? AND is_active = ?", refreshToken, false).First(&session)
	if result.Error == nil {
		// 该token已被撤销（使用过），说明发生重放攻击
		// 安全措施：撤销该用户所有会话
		s.logger.Warn("检测到Refresh Token重放攻击",
			zap.Uint("user_id", claims.UserID),
			zap.String("token_prefix", refreshToken[:8]),
		)
		_ = s.RevokeUserSessions(claims.UserID)
		return nil, errors.New("token reuse detected")
	}

	// 立即撤销当前refresh token（轮换）
	if err := s.RevokeSession(refreshToken); err != nil {
		s.logger.Warn("撤销旧token失败", zap.Error(err))
	}

	var user models.User
	if err := s.db.Preload("Role.Permissions").First(&user, claims.UserID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("user is inactive")
	}

	return s.GenerateTokenPair(&user)
}

// CreateSession 创建会话记录
func (s *SM4TokenService) CreateSession(userID uint, token string, ipAddress, userAgent string, expiresAt time.Time) error {
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
func (s *SM4TokenService) RevokeSession(token string) error {
	return s.db.Model(&models.Session{}).
		Where("token = ?", token).
		Update("is_active", false).Error
}

// RevokeUserSessions 撤销用户所有会话
func (s *SM4TokenService) RevokeUserSessions(userID uint) error {
	return s.db.Model(&models.Session{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", false).Error
}

// CleanExpiredSessions 清理过期会话
func (s *SM4TokenService) CleanExpiredSessions() error {
	return s.db.Model(&models.Session{}).
		Where("expires_at < ?", time.Now()).
		Delete(&models.Session{}).Error
}

// GenerateRandomSecret 生成随机SM4密钥（Base64编码）
func GenerateRandomSecret() (string, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
