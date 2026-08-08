// Package videoplaybacktoken 提供已录制视频文件播放专用的 HMAC token 服务。
//
// 设计要点:
//   - 设计范式与 internal/auth/hlstoken 同源(HMAC-SHA256 + base64url(claims).base64url(sig)),
//     但只读端点 + 5min TTL 不需要 jti / DB persistence / sweeper——避免重引入非必要
//     复杂度(范围越小风险越低)。
//   - 用途:让 <video> 元素可在不带 Authorization 头的 HTTP 请求里获得 5min 短效播放凭据。
//     路径模式为 /api/v1/files/playback/:token(token 在 path 中,而非 query string,
//     避免 URL 出现在日志/浏览器历史/Referer 等泄露面)。
//   - 校验失败统一映射到 apperrors.ErrTokenInvalid / ErrTokenExpired,与 SM4 / HLS token
//     路径保持一致,便于 handler 走 response.HandleError 统一响应。
package videoplaybacktoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
)

// Claims 绑死 (fileID, userID, exp, iat);Verify 失败视作 token 不可信,
// handler 仍需在拿到 claims 后比对 file.CreatedBy == claims.UserID 做 defense-in-depth。
type Claims struct {
	FileID    uint  `json:"file_id"`
	UserID    uint  `json:"user_id"`
	ExpiresAt int64 `json:"expires_at"`
	IssuedAt  int64 `json:"issued_at"`
}

// VideoPlaybackToken 是 video playback token 的服务实例。启动期构造,运行期无状态。
// secret 在构造时拷贝为 []byte 避免外部后续修改原 secret 字符串影响校验。
type VideoPlaybackToken struct {
	secretCopy []byte
	duration   time.Duration
	logger     *zap.Logger
}

// NewVideoPlaybackToken 构造 video playback token 服务。
//
// 防御性兜底:secret < 32 字符直接 panic(fail-fast)。启动期 cfg.ValidateVideoPlaybackTokenConfig
// 应已先行 Fatal,这里只是双保险——若绕过配置校验走到这里,立即崩溃而非默默签发弱密钥 token。
// duration 校验由 cfg.ValidateVideoPlaybackTokenConfig 负责(不接受 ≤ 0);这里容忍
// 任意 duration 以便测试构造过期 token 场景。
func NewVideoPlaybackToken(secret string, duration time.Duration, logger *zap.Logger) *VideoPlaybackToken {
	if len(secret) < 32 {
		panic(fmt.Sprintf("videoplaybacktoken: secret 必须 ≥ 32 字符(当前 %d),拒绝初始化", len(secret)))
	}
	if logger != nil {
		logger.Info("video_playback_token 服务初始化",
			zap.Duration("duration", duration),
			zap.Int("secret_len", len(secret)),
		)
	}
	return &VideoPlaybackToken{
		secretCopy: append([]byte(nil), []byte(secret)...),
		duration:   duration,
		logger:     logger,
	}
}

// Generate 签发 token。返回格式: base64url(json_claims).base64url(hmac_sha256_sig)。
//
// claims 在签名前 JSON marshal,signature 走 HMAC-SHA256,使用 RawURLEncoding 去除
// 尾部 padding 让 URL 更紧凑。
func (t *VideoPlaybackToken) Generate(fileID, userID uint) string {
	now := time.Now()
	claims := Claims{
		FileID:    fileID,
		UserID:    userID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(t.duration).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		// json.Marshal 只会因自定义类型失败;Claims 全是基础类型不可能出错。
		// 真出错时不让调用方拿到半成品 token,直接返回空串让调用方处理。
		if t.logger != nil {
			t.logger.Error("video_playback_token claims 序列化失败",
				zap.Uint("file_id", fileID), zap.Uint("user_id", userID), zap.Error(err))
		}
		return ""
	}
	encoded := base64.URLEncoding.EncodeToString(payload)
	sig := t.sign(encoded)
	return encoded + "." + sig
}

// Verify 校验 token,返回 claims。失败原因映射到 apperrors sentinel。
//
// 校验顺序:拆分 → HMAC 比对(constant-time)→ base64 解码 + JSON 反序列化 → 时间窗。
// 时间窗检查放在签名校验之后,避免攻击者用合法 base64 字符串探测系统时钟。
//
// 签名比对尝试多种 base64 编码(RawURL/URL/Std)——与 HLS token 一致,保持服务滚动
// 重启后旧 token 仍可校验一次,降低上线抖动。
func (t *VideoPlaybackToken) Verify(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, apperrors.ErrTokenInvalid
	}
	encoded, sig := parts[0], parts[1]

	mac := hmac.New(sha256.New, t.secretCopy)
	mac.Write([]byte(encoded))
	expectedMAC := mac.Sum(nil)

	providedMAC, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		providedMAC, err = base64.URLEncoding.DecodeString(sig)
		if err != nil {
			providedMAC, err = base64.StdEncoding.DecodeString(sig)
			if err != nil {
				return nil, apperrors.ErrTokenInvalid
			}
		}
	}
	if !hmac.Equal(expectedMAC, providedMAC) {
		return nil, apperrors.ErrTokenInvalid
	}

	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("video_playback_token 解码失败: %w", apperrors.ErrTokenInvalid)
	}
	var claims Claims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("video_playback_token 解析失败: %w", apperrors.ErrTokenInvalid)
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return nil, apperrors.ErrTokenExpired
	}

	return &claims, nil
}

// sign 内部 HMAC-SHA256 签名。签发使用 RawURLEncoding 去除尾部 padding,
// Verify 多编码兼容保证平滑过渡。
func (t *VideoPlaybackToken) sign(payload string) string {
	mac := hmac.New(sha256.New, t.secretCopy)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// String 返回 claims 的简短描述,用于日志与调试。
func (c *Claims) String() string {
	return fmt.Sprintf("file_id=%d,user_id=%d,expires_at=%d",
		c.FileID, c.UserID, c.ExpiresAt)
}

// IsExpired 检查 token 是否已过期(用于 handler 内可选的二次确认)。
func (c *Claims) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// TimeRemaining 返回 token 剩余有效时间(秒)。
func (c *Claims) TimeRemaining() int64 {
	remaining := c.ExpiresAt - time.Now().Unix()
	if remaining < 0 {
		return 0
	}
	return remaining
}
