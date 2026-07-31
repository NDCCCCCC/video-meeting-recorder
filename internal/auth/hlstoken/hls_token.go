package hlstoken

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ErrTokenReplayed 表示同一 jti 的 token 已被使用过（防重放）。
// SEC-004 (Phase 19): Verify 不再一次性拒绝已见过的 jti——多分片 HLS 共用同一 token
// 需要多次 Verify 全部通过。此 error 保留用于未来按需启用严格一次性语义的调用点，
// 当前 Verify 不会返回它。
var ErrTokenReplayed = errors.New("token 已被使用（防重放）")

// maxUsedJTIs 是 usedJTIs map 的硬上限。超过时强制驱逐全部过期项 + 最早过期项，
// 防止 sweeper goroutine 死亡时 map 无限增长（SEC-004 决策 2）。
const maxUsedJTIs = 100000

// HLSTokenClaims HLS Token 声明
type HLSTokenClaims struct {
	TaskID    uint   `json:"task_id"`
	UserID    uint   `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
	IssuedAt  int64  `json:"issued_at"`
	Jti       string `json:"jti"` // SEC-004: 唯一标识，用于一次性防重放
}

// HLSToken HLS Token 管理器
type HLSToken struct {
	secret        string
	tokenDuration time.Duration
	// SEC-004 (Phase 19): jti → ExpiresAt 索引。Verify 幂等记录/覆盖（同 token 重验
	// 只刷新过期时间）；真正阻止 post-TTL 重放的是 Verify 内 time.Now() > ExpiresAt
	// 检查。map 角色从"一次性重放拒绝"转为"过期索引追踪"（启用驱逐 + 未来吊销）。
	// 局限：进程重启后记录清零；服务端持久化 used_jtis（Redis/DB）留作下个独立 phase。
	usedJTIs map[string]int64
	mu       sync.Mutex

	// 生命周期管理：sweepLoop goroutine + Start/Stop（镜像 PERF-006 Huawei client Stop 模式）。
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewHLSToken 创建 HLS Token 管理器
func NewHLSToken(secret string, duration time.Duration) *HLSToken {
	// SEC-004: 密钥长度校验（防御性兜底；启动期 config.ValidateProductionSecrets 已先行 Fatal）。
	if len(secret) < 32 {
		panic(fmt.Sprintf("HLS token secret 必须 ≥ 32 字符（当前 %d），拒绝初始化", len(secret)))
	}
	return &HLSToken{
		secret:        secret,
		tokenDuration: duration,
		usedJTIs:      make(map[string]int64),
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
		Jti:       generateJti(), // SEC-004: 一次性防重放标识
	}

	// 序列化为 JSON
	data, err := json.Marshal(claims)
	if err != nil {
		return ""
	}

	// Base64 编码
	encodedData := base64.URLEncoding.EncodeToString(data)

	// 生成签名（SEC-004/D-03.3: 签发用 RawURLEncoding）
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

	// SEC-004/D-03.3: 计算期望 HMAC 字节，并尝试多种 base64 编码解码 provided 签名。
	// 新 token 用 RawURLEncoding 签发；旧 token（重启前）用 URLEncoding/StdEncoding 签发，
	// 这里同时接受三种编码以保持向后兼容（旧 token 重启后仍可验证一次）。
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write([]byte(encodedData))
	expectedMAC := mac.Sum(nil)

	providedMAC, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		providedMAC, err = base64.URLEncoding.DecodeString(signature)
		if err != nil {
			providedMAC, err = base64.StdEncoding.DecodeString(signature)
			if err != nil {
				return nil, fmt.Errorf("token 签名无效")
			}
		}
	}
	if !hmac.Equal(expectedMAC, providedMAC) {
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

	// SEC-004 (Phase 19): jti 索引追踪（幂等）。不再一次性拒绝——同一 token 在其
	// TTL 窗口内可多次 Verify（HLS 多分片共用同一 token，见 rewriteM3U8WithToken）。
	// 真正阻止 post-TTL 重放的是上面的 time.Now() > ExpiresAt 检查（不变）。
	// map 用于过期索引追踪（启用 evictExpired 驱逐 + 未来吊销扩展）。
	if claims.Jti != "" {
		h.mu.Lock()
		h.usedJTIs[claims.Jti] = claims.ExpiresAt
		h.enforceCapLocked()
		h.mu.Unlock()
	}

	return &claims, nil
}

// evictExpired 删除已过期（expiresAt < now）的 jti 索引项。
func (h *HLSToken) evictExpired() {
	now := time.Now().Unix()
	h.mu.Lock()
	for jti, exp := range h.usedJTIs {
		if exp < now {
			delete(h.usedJTIs, jti)
		}
	}
	h.mu.Unlock()
}

// enforceCapLocked 在 usedJTIs 超过 maxUsedJTIs 时强制回收。调用方必须持有 h.mu。
// 先驱逐全部过期项；若仍超限（极端：全部是未过期长 TTL 项），按最早 ExpiresAt
// 逐项删除直至回到上限以下，避免 sweeper goroutine 死亡时 map 无限增长。
func (h *HLSToken) enforceCapLocked() {
	if len(h.usedJTIs) <= maxUsedJTIs {
		return
	}
	now := time.Now().Unix()
	for jti, exp := range h.usedJTIs {
		if exp < now {
			delete(h.usedJTIs, jti)
		}
	}
	for len(h.usedJTIs) > maxUsedJTIs {
		var oldestJti string
		oldestExp := int64(1<<63 - 1)
		for jti, exp := range h.usedJTIs {
			if exp < oldestExp {
				oldestExp = exp
				oldestJti = jti
			}
		}
		delete(h.usedJTIs, oldestJti)
	}
}

// sweepLoop 周期性驱逐过期 jti 索引项。遵循 BUG-006 NewTicker+select 约定
// （非裸 time.Sleep，可被 ctx 取消）。
func (h *HLSToken) sweepLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.evictExpired()
		}
	}
}

// Start 启动后台 sweeper goroutine，周期性驱逐过期 jti 索引项。
// ctx 作为父生命周期；Stop 会取消派生的子 ctx 并等待 goroutine 退出。
// 重复调用 Start 是 no-op。
func (h *HLSToken) Start(ctx context.Context) {
	h.mu.Lock()
	if h.cancel != nil {
		h.mu.Unlock()
		return
	}
	childCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	h.mu.Unlock()

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.sweepLoop(childCtx)
	}()
}

// Stop 取消 sweeper goroutine 并等待其退出。重复调用是 no-op。
func (h *HLSToken) Stop() {
	h.mu.Lock()
	cancel := h.cancel
	h.cancel = nil
	h.mu.Unlock()
	if cancel != nil {
		cancel()
		h.wg.Wait()
	}
}

// sign 生成签名（SEC-004/D-03.3: 新签发统一用 RawURLEncoding，去除尾部 padding）
func (h *HLSToken) sign(data string) string {
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// generateJti 生成 16 字节随机 hex 编码的唯一标识（crypto/rand）。
func generateJti() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand 失败极罕见；回退到时间戳保证唯一性，不影响安全（HMAC 仍是主防线）。
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
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
