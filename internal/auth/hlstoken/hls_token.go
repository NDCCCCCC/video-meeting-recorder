package hlstoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HLSTokenClaims HLS Token 声明
type HLSTokenClaims struct {
	TaskID    uint      `json:"task_id"`
	UserID    uint      `json:"user_id"`
	ExpiresAt int64     `json:"expires_at"`
	IssuedAt  int64     `json:"issued_at"`
}

// HLSToken HLS Token 管理器
type HLSToken struct {
	secret     string
	tokenDuration time.Duration
}

// NewHLSToken 创建 HLS Token 管理器
func NewHLSToken(secret string, duration time.Duration) *HLSToken {
	return &HLSToken{
		secret:     secret,
		tokenDuration: duration,
	}
}

// Generate 生成访问 Token
func (h *HLSToken) Generate(taskID, userID uint) string {
	now := time.Now()
	claims := HLSTokenClaims{
		TaskID:    taskID,
		UserID:    userID,
		ExpiresAt: now.Add(h.tokenDuration).Unix(),
		IssuedAt:  now.Unix(),
	}

	// 序列化为 JSON
	data, err := json.Marshal(claims)
	if err != nil {
		return ""
	}

	// Base64 编码
	encodedData := base64.URLEncoding.EncodeToString(data)

	// 生成签名
	signature := h.sign(encodedData)

	// 返回格式: data.signature
	return fmt.Sprintf("%s.%s", encodedData, signature)
}

// Verify 验证 Token
func (h *HLSToken) Verify(token string) (*HLSTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("无效的 token 格式")
	}

	encodedData := parts[0]
	signature := parts[1]

	// 验证签名
	expectedSignature := h.sign(encodedData)
	if signature != expectedSignature {
		return nil, fmt.Errorf("token 签名无效")
	}

	// 解码数据
	data, err := base64.URLEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("token 解码失败: %w", err)
	}

	// 解析 JSON
	var claims HLSTokenClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("token 解析失败: %w", err)
	}

	// 检查过期时间
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token 已过期")
	}

	return &claims, nil
}

// sign 生成签名
func (h *HLSToken) sign(data string) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

// GetTokenFromURL 从 URL 查询参数中提取 token
func GetTokenFromURL(queryParams map[string]string) string {
	// 支持两种参数名: token 和 access_token
	if token, ok := queryParams["token"]; ok {
		return token
	}
	if token, ok := queryParams["access_token"]; ok {
		return token
	}
	return ""
}

// ParseTaskIDFromToken 从 token 中解析任务 ID（用于日志记录）
func ParseTaskIDFromToken(token string) uint {
	if parts := strings.Split(token, "."); len(parts) == 2 {
		if data, err := base64.URLEncoding.DecodeString(parts[0]); err == nil {
			var claims HLSTokenClaims
			if json.Unmarshal(data, &claims) == nil {
				return claims.TaskID
			}
		}
	}
	return 0
}

// String 返回 token 的字符串表示（用于调试）
func (c *HLSTokenClaims) String() string {
	return fmt.Sprintf("task_id=%d,user_id=%d,expires_at=%d",
		c.TaskID, c.UserID, c.ExpiresAt)
}

// IsExpired 检查 token 是否已过期
func (c *HLSTokenClaims) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// TimeRemaining 返回 token 剩余有效时间（秒）
func (c *HLSTokenClaims) TimeRemaining() int64 {
	remaining := c.ExpiresAt - time.Now().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}

// FormatExpiresAt 格式化过期时间
func (c *HLSTokenClaims) FormatExpiresAt() string {
	return time.Unix(c.ExpiresAt, 0).Format("2006-01-02 15:04:05")
}

// FormatIssuedAt 格式化签发时间
func (c *HLSTokenClaims) FormatIssuedAt() string {
	return time.Unix(c.IssuedAt, 0).Format("2006-01-02 15:04:05")
}

// ValidateUser 验证用户是否有权限访问此任务
func (c *HLSTokenClaims) ValidateUser(userID uint) bool {
	return c.UserID == userID
}

// ParseUserIDFromToken 从 token 中解析用户 ID
func ParseUserIDFromToken(token string) uint {
	if parts := strings.Split(token, "."); len(parts) == 2 {
		if data, err := base64.URLEncoding.DecodeString(parts[0]); err == nil {
			var claims HLSTokenClaims
			if json.Unmarshal(data, &claims) == nil {
				return claims.UserID
			}
		}
	}
	return 0
}

// GenerateTokenURL 生成带 token 的完整 URL
func GenerateTokenURL(baseURL string, taskID, userID uint, secret string, duration time.Duration) string {
	tokenGen := NewHLSToken(secret, duration)
	token := tokenGen.Generate(taskID, userID)
	if strings.Contains(baseURL, "?") {
		return fmt.Sprintf("%s&token=%s", baseURL, token)
	}
	return fmt.Sprintf("%s?token=%s", baseURL, token)
}

// ParseTaskIDFromURLString 从 URL 字符串中解析任务 ID（兼容性函数）
func ParseTaskIDFromURLString(urlStr string) (uint, error) {
	// 从 URL 中提取任务 ID
	// 格式: /api/v1/recordings/{task_id}/preview/stream/{file}
	parts := strings.Split(urlStr, "/")
	for i, part := range parts {
		if part == "recordings" && i+1 < len(parts) {
			taskID, err := strconv.ParseUint(parts[i+1], 10, 32)
			if err != nil {
				return 0, err
			}
			return uint(taskID), nil
		}
	}
	return 0, fmt.Errorf("无法从 URL 中解析任务 ID")
}
