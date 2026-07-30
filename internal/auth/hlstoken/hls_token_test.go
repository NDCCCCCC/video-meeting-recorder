package hlstoken

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// testSecret 返回长度 ≥ 32 的测试密钥。
func testSecret() string { return strings.Repeat("k", 32) }

// makeLegacyToken 手工构造一个"旧编码"token：数据用 URLEncoding，签名用指定编码，
// 模拟本修复前签发的 token（无 jti）。用于验证 Verify 的向后兼容承诺（D-03.3）。
func makeLegacyToken(secret string, sigEncoding *base64.Encoding, jti string) string {
	claims := HLSTokenClaims{
		TaskID:    1,
		UserID:    2,
		ExpiresAt: time.Now().Add(time.Minute).Unix(),
		IssuedAt:  time.Now().Unix(),
		Jti:       jti,
	}
	data, _ := json.Marshal(claims)
	encodedData := base64.URLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(encodedData))
	sig := sigEncoding.EncodeToString(mac.Sum(nil))
	return encodedData + "." + sig
}

// TestNewHLSToken_ShortSecretPanics 验证 SEC-004：构造期密钥 < 32 字符 → panic（防御性兜底）。
func TestNewHLSToken_ShortSecretPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewHLSToken("too-short", time.Minute)
	})
}

// TestNewHLSToken_ValidSecretOK 合法长度（≥32）不应 panic。
func TestNewHLSToken_ValidSecretOK(t *testing.T) {
	h := NewHLSToken(testSecret(), time.Minute)
	assert.NotNil(t, h)
	assert.NotNil(t, h.usedJTIs, "usedJTIs 防重放集合应初始化")
}

// TestHLSVerify_BackwardCompat 验证 SEC-004/D-03.3：新代码同时接受新旧三种 base64 编码签名。
func TestHLSVerify_BackwardCompat(t *testing.T) {
	secret := testSecret()
	h := NewHLSToken(secret, time.Minute)

	cases := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "旧 URLEncoding 签名（无 jti）→ 验证通过",
			token:   makeLegacyToken(secret, base64.URLEncoding, ""),
			wantErr: false,
		},
		{
			name:    "旧 StdEncoding 签名 → 验证通过（D-03.3 兼容承诺）",
			token:   makeLegacyToken(secret, base64.StdEncoding, ""),
			wantErr: false,
		},
		{
			name:    "新 RawURLEncoding 签名（Generate）→ 验证通过",
			token:   h.Generate(1, 2),
			wantErr: false,
		},
		{
			name:    "篡改签名 → 拒绝",
			token:   h.Generate(1, 2) + "x",
			wantErr: true,
		},
	}

	// 注意：每个用例使用独立的 jti，避免相互触发防重放；旧 token 无 jti 不受影响。
	for i, tc := range cases {
		// 为新 token 用例生成独立 token，旧 token 用例已固定。
		if tc.name == "新 RawURLEncoding 签名（Generate）→ 验证通过" {
			tc.token = h.Generate(100+uint(i), 200+uint(i))
		}
		t.Run(tc.name, func(t *testing.T) {
			_, err := h.Verify(tc.token)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestHLSVerify_JtiReplayRejection 验证 SEC-004：同一 jti 的 token 在进程内只能验证一次，
// 第二次验证返回 ErrTokenReplayed。
func TestHLSVerify_JtiReplayRejection(t *testing.T) {
	h := NewHLSToken(testSecret(), time.Minute)
	tok := h.Generate(5, 6)

	claims1, err1 := h.Verify(tok)
	assert.NoError(t, err1, "首次验证应成功")
	assert.NotEmpty(t, claims1.Jti, "新签发 token 应包含 jti")

	_, err2 := h.Verify(tok)
	assert.True(t, errors.Is(err2, ErrTokenReplayed), "第二次验证应被防重放拒绝，got: %v", err2)
}

// TestHLSVerify_Expired 验证过期 token 被拒绝（回归路径）。
func TestHLSVerify_Expired(t *testing.T) {
	h := NewHLSToken(testSecret(), -time.Minute) // 负 duration → 立即过期
	tok := h.Generate(7, 8)
	_, err := h.Verify(tok)
	assert.Error(t, err)
}
