package auth

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tjfoc/gmsm/sm4"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/NDCCCCCC/video-meeting-recorder/internal/config"
	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/NDCCCCCC/video-meeting-recorder/internal/models"
	"github.com/NDCCCCCC/video-meeting-recorder/pkg/response"
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
	gcm                     cipher.AEAD // 预初始化的SM4-GCM实例，避免每次加解密重复创建
	issuer                  string
	expireHours             int
	refreshExpire           int
	maxSession              int
	db                      *gorm.DB
	logger                  *zap.Logger
	allowedTokenURLPrefixes []string
	// token 缓存：支持宽限期机制
	tokenCache      map[string]*TokenCacheEntry // key: refresh token, value: 缓存的 token 对
	tokenCacheMutex sync.RWMutex
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
		logger.Fatal("创建SM4加密器失败", zap.Error(err), response.SentinelField(err))
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		logger.Fatal("创建GCM模式失败", zap.Error(err), response.SentinelField(err))
	}

	return &SM4TokenService{
		gcm:                     gcm,
		issuer:                  "record_v2",
		expireHours:             int(cfg.Auth.AccessTokenDuration.Hours()),
		refreshExpire:           int(cfg.Auth.RefreshTokenDuration.Hours()),
		maxSession:              int(cfg.Auth.MaxSessionDuration.Hours()),
		db:                      db,
		logger:                  logger,
		allowedTokenURLPrefixes: append([]string(nil), cfg.Security.AllowedTokenURLPrefixes...),
		tokenCache:              make(map[string]*TokenCacheEntry),
	}
}

// deriveSM4Key 将 32 字符 hex 格式的 SM4 secret 直接解码为 16 字节原始密钥。
// SEC-011: 不再使用 SHA256 截断派生（entropy 损失）；16 字节等价 32 hex 字符，
// 对齐前端 sm4.ts 的密钥格式约定。调用方应保证 secret 为 32 hex 字符（SEC-001
// 启动校验 len(secret) >= 32 强制）。若 hex decode 失败则 Fatal 退出。
func deriveSM4Key(secret string) []byte {
	// SEC-011: 取 SM4 secret 的前 32 hex 字符并直接解码为 16 字节原始密钥。
	// SEC-001 启动校验已保证 secret >= 32 字符，故 len(hexStr) >= 32。
	hexStr := secret
	if len(hexStr) > 32 {
		hexStr = hexStr[:32]
	}
	key, err := hex.DecodeString(hexStr)
	if err != nil || len(key) != 16 {
		// 无法解码：保留 SHA256 截断作为最终回退，避免启动崩溃——SEC-001
		// 启动校验已记录该 secret 质量问题，此处仅给出可用密钥。
		hash := sha256.Sum256([]byte(secret))
		return hash[:16]
	}
	return key
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

	var permissions []string //nolint:prealloc // 声明在 permMap 之前，无法用 len(permMap) 预分配
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
	roleIDs := make([]uint, 0, len(user.Roles))
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
		return nil, apperrors.ErrTokenInvalid
	}

	nonceSize := s.gcm.NonceSize()
	if len(data) < nonceSize+s.gcm.Overhead() {
		return nil, apperrors.ErrTokenInvalid
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := s.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, apperrors.ErrTokenInvalid
	}

	var claims Claims
	if err := json.Unmarshal(plaintext, &claims); err != nil {
		return nil, apperrors.ErrTokenInvalid
	}

	return &claims, nil
}

// ValidateToken 验证Access Token，返回解析后的Claims
func (s *SM4TokenService) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
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
		return apperrors.ErrTokenInvalid
	}

	now := time.Now().Unix()
	if now > claims.ExpiresAt {
		return apperrors.ErrTokenExpired
	}
	if now < claims.NotBefore {
		return apperrors.ErrTokenNotYetValid
	}
	return nil
}

// RefreshAccessToken 使用Refresh Token刷新，生成新的Token对。
// 实现宽限期机制：5秒内重复刷新请求返回相同的 token 对（幂等性）。
// 关键：在宽限期内不撤销旧 token，只有在超过宽限期后才撤销。
func (s *SM4TokenService) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	return s.RefreshAccessTokenWithContext(context.Background(), refreshToken)
}

// RefreshAccessTokenWithContext 刷新令牌并允许调用方取消宽限期撤销任务。
func (s *SM4TokenService) RefreshAccessTokenWithContext(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	// 步骤1: 检查缓存 - 快速路径
	s.tokenCacheMutex.RLock()
	cachedEntry, exists := s.tokenCache[refreshToken]
	s.tokenCacheMutex.RUnlock()

	if exists {
		// 检查缓存是否过期
		if now.Before(cachedEntry.ExpiresAt) {
			s.logger.Debug("宽限期内重复刷新（缓存），返回缓存的 token 对",
				zap.Uint("user_id", claims.UserID),
				zap.String("token_prefix", refreshToken[:8]),
			)
			return cachedEntry.TokenPair, nil
		}
		// 缓存过期，清理
		s.tokenCacheMutex.Lock()
		delete(s.tokenCache, refreshToken)
		s.tokenCacheMutex.Unlock()
	}

	// 步骤2: 查找当前 token 的 session
	var session models.Session
	result := s.db.WithContext(ctx).Where("token = ?", refreshToken).First(&session)

	if result.Error != nil {
		// session 不存在，可能是其他地方创建的 token
		// 继续正常流程
	} else {
		// session 存在，检查状态
		if !session.IsActive {
			// token 已被使用过
			if session.LastUsedAt != nil {
				timeSinceLastUse := now.Sub(*session.LastUsedAt)
				if timeSinceLastUse < GracePeriod {
					// 在宽限期内，查找最近生成的新 token
					var newSession models.Session
					err := s.db.WithContext(ctx).Where("user_id = ? AND created_at > ?",
						claims.UserID,
						now.Add(-GracePeriod),
					).Order("created_at DESC").First(&newSession).Error

					if err == nil {
						// 修复 Bug #2：原代码直接把 newSession.Token（旧 refresh token）同时作为 AT 和 RT 返回，
						// 导致 AT == RT 且用户下一次 refresh 时 token type 校验失败被踢。
						// 正确做法：在宽限期内重新生成新的 token 对并缓存。
						s.logger.Debug("宽限期内重复刷新（数据库），生成新 token 对",
							zap.Uint("user_id", claims.UserID),
						)

						var user models.User
						if err := s.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
							return nil, apperrors.ErrUserNotFound
						}
						newTokenPair, err := s.GenerateTokenPair(&user)
						if err != nil {
							return nil, err
						}

						// 缓存新生成的 token 对，避免再次进入数据库查询路径
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
			}

			// 超过宽限期的重放攻击
			s.logger.Warn("检测到Refresh Token重放攻击（超过宽限期）",
				zap.Uint("user_id", claims.UserID),
				zap.String("token_prefix", refreshToken[:8]),
			)
			if err := s.RevokeUserSessions(claims.UserID); err != nil {
				s.logger.Warn("撤销用户会话失败", zap.Uint("user_id", claims.UserID), zap.Error(err), response.SentinelField(err))
			}
			return nil, apperrors.ErrTokenReplayed
		}
	}

	// 步骤3: 正常流程 - 第一次使用此 refresh token
	var user models.User
	if err := s.db.WithContext(ctx).Preload("Roles.Permissions").First(&user, claims.UserID).Error; err != nil {
		return nil, apperrors.ErrUserNotFound
	}

	if !user.IsActive {
		return nil, apperrors.ErrUserDisabled
	}

	newTokenPair, err := s.GenerateTokenPair(&user)
	if err != nil {
		return nil, err
	}

	// 关键修复：只更新 last_used_at，不立即撤销 token
	// 这样在宽限期内重复使用时，session 仍然是 active 的
	if err := s.db.WithContext(ctx).Model(&models.Session{}).
		Where("token = ?", refreshToken).
		Update("last_used_at", now).Error; err != nil {
		s.logger.Warn("更新session last_used_at失败", zap.Error(err), response.SentinelField(err))
	}

	// 缓存新生成的 token 对，设置宽限期过期时间
	s.tokenCacheMutex.Lock()
	s.tokenCache[refreshToken] = &TokenCacheEntry{
		TokenPair:    newTokenPair,
		ExpiresAt:    now.Add(GracePeriod),
		RefreshToken: refreshToken,
	}
	s.tokenCacheMutex.Unlock()

	// 启动后台任务：在宽限期过后撤销旧 token
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("GracePeriod revoke goroutine panicked",
					zap.Any("recover", r), zap.Stack("stack"))
			}
		}()
		timer := time.NewTimer(GracePeriod)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		s.db.Model(&models.Session{}).
			Where("token = ? AND is_active = ?", refreshToken, true).
			Update("is_active", false)
	}()

	return newTokenPair, nil
}

// CreateSession 创建会话记录
func (s *SM4TokenService) CreateSession(userID uint, token, ipAddress, userAgent string, expiresAt time.Time) error {
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

// AllowedTokenURLPrefixes returns a defensive copy of configured query-token routes.
func (s *SM4TokenService) AllowedTokenURLPrefixes() []string {
	return append([]string(nil), s.allowedTokenURLPrefixes...)
}
