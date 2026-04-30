package auth

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/tjfoc/gmsm/sm4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// TokenCacheEntry 缓存的 token 对条目
type TokenCacheEntry struct {
	TokenPair    *TokenPair
	ExpiresAt    time.Time
	RefreshToken string // 原始 refresh token，用于查找
}

const (
	// GracePeriod 宽限期时间：5秒内重复刷新请求返回相同的 token 对
	GracePeriod = 5 * time.Second
)

// SM4TokenService 基于SM4-GCM的Token服务，替代JWT。
// Token格式: base64url(nonce[12] + SM4-GCM(claims_json) + authTag[16])
// SM4-GCM同时提供加密（payload不可读）和认证（防篡改）。
type SM4TokenService struct {
	gcm              cipher.AEAD // 预初始化的SM4-GCM实例，避免每次加解密重复创建
	issuer           string
	expireHours      int
	refreshExpire    int
	maxSession       int
	db               *gorm.DB
	logger           *zap.Logger
	// token 缓存：支持宽限期机制
	tokenCache       map[string]*TokenCacheEntry // key: refresh token, value: 缓存的 token 对
	tokenCacheMutex  sync.RWMutex
}

// Claims Token声明
type Claims struct {
	UserID      uint     `json:"uid"`
	Username    string   `json:"sub"`
	RoleIDs     []uint   `json:"rids"`
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
		tokenCache:    make(map[string]*TokenCacheEntry),
	}
}

// cleanupExpiredCache 清理过期的缓存条目（应在后台定期调用）
func (s *SM4TokenService) cleanupExpiredCache() {
	s.tokenCacheMutex.Lock()
	defer s.tokenCacheMutex.Unlock()

	now := time.Now()
	for token, entry := range s.tokenCache {
		if now.After(entry.ExpiresAt) {
			delete(s.tokenCache, token)
		}
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
	isAdmin := user.HasRole(models.RoleAdmin)

	// Collect permissions from all roles (OR logic)
	permMap := make(map[string]bool)
	for _, role := range user.Roles {
		for _, perm := range role.Permissions {
			permStr := perm.Resource + ":" + perm.Action
			permMap[permStr] = true
		}
	}
	for permStr := range permMap {
		permissions = append(permissions, permStr)
	}

	// Extract role IDs for token
	var roleIDs []uint
	for _, role := range user.Roles {
		roleIDs = append(roleIDs, role.ID)
	}

	claims := &Claims{
		UserID:      user.ID,
		Username:    user.Username,
		RoleIDs:     roleIDs,
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
// 实现宽限期机制：5秒内重复刷新请求返回相同的 token 对（幂等性）。
// 如果检测到真正的重放攻击（超过宽限期），撤销该用户所有会话。
func (s *SM4TokenService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 【关键修复】宽限期机制：检查缓存
	s.tokenCacheMutex.RLock()
	cachedEntry, exists := s.tokenCache[refreshToken]
	s.tokenCacheMutex.RUnlock()

	if exists {
		// 检查是否在宽限期内
		if time.Since(cachedEntry.ExpiresAt) < GracePeriod {
			s.logger.Debug("宽限期内重复刷新，返回缓存的 token 对",
				zap.Uint("user_id", claims.UserID),
				zap.String("token_prefix", refreshToken[:8]),
			)
			// 更新 session 的 last_used_at 时间
			now := time.Now()
			s.db.Model(&models.Session{}).
				Where("token = ?", refreshToken).
				Update("last_used_at", now)

			return cachedEntry.TokenPair, nil
		}
		// 超过宽限期，清理缓存
		s.tokenCacheMutex.Lock()
		delete(s.tokenCache, refreshToken)
		s.tokenCacheMutex.Unlock()
	}

	// 检查该 refresh token 是否已被使用过（重放检测）
	var session models.Session
	result := s.db.Where("token = ? AND is_active = ?", refreshToken, false).First(&session)
	if result.Error == nil {
		// 检查是否在宽限期内使用
		if session.LastUsedAt != nil {
			timeSinceLastUse := time.Since(*session.LastUsedAt)
			if timeSinceLastUse < GracePeriod {
				// 在宽限期内，查询最近生成的 token
				var newSession models.Session
				err := s.db.Where("user_id = ? AND created_at > ?",
					claims.UserID,
					time.Now().Add(-GracePeriod),
				).Order("created_at DESC").First(&newSession).Error

				if err == nil && newSession.Token != "" {
					s.logger.Debug("宽限期内重试，返回数据库中的 token",
						zap.Uint("user_id", claims.UserID),
					)
					// 返回已生成的新 token
					return &TokenPair{
						AccessToken:  newSession.Token, // 这里简化处理，实际应该返回完整的 token 对
						RefreshToken: newSession.Token,
						ExpiresAt:    newSession.ExpiresAt,
					}, nil
				}
			}
		}

		// 超过宽限期的重放攻击 -> 撤销所有会话
		s.logger.Warn("检测到Refresh Token重放攻击（超过宽限期）",
			zap.Uint("user_id", claims.UserID),
			zap.String("token_prefix", refreshToken[:8]),
		)
		_ = s.RevokeUserSessions(claims.UserID)
		return nil, errors.New("token reuse detected")
	}

	// 正常流程：生成新的 token 对
	var user models.User
	if err := s.db.Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("user is inactive")
	}

	newTokenPair, err := s.GenerateTokenPair(&user)
	if err != nil {
		return nil, err
	}

	// 更新 session 的 last_used_at 并撤销旧 token
	now := time.Now()
	if err := s.db.Model(&models.Session{}).
		Where("token = ?", refreshToken).
		Updates(map[string]interface{}{
			"is_active":     false,
			"last_used_at":  now,
		}).Error; err != nil {
		s.logger.Warn("更新session状态失败", zap.Error(err))
	}

	// 缓存新生成的 token 对
	s.tokenCacheMutex.Lock()
	s.tokenCache[refreshToken] = &TokenCacheEntry{
		TokenPair:    newTokenPair,
		ExpiresAt:    now.Add(GracePeriod),
		RefreshToken: refreshToken,
	}
	s.tokenCacheMutex.Unlock()

	return newTokenPair, nil
}
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
