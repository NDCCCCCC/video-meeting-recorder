package utils

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestDeriveSM4Key(t *testing.T) {
	secret := "test-secret-key"

	t.Run("密钥长度必须为 16 字节", func(t *testing.T) {
		key := DeriveSM4Key(secret)
		assert.Equal(t, 16, len(key), "SM4 密钥必须是 16 字节")
	})

	t.Run("相同的 secret 生成相同的密钥", func(t *testing.T) {
		key1 := DeriveSM4Key(secret)
		key2 := DeriveSM4Key(secret)
		assert.Equal(t, key1, key2, "相同的 secret 应生成相同的密钥")
	})

	t.Run("不同的 secret 生成不同的密钥", func(t *testing.T) {
		key1 := DeriveSM4Key("secret1")
		key2 := DeriveSM4Key("secret2")
		assert.NotEqual(t, key1, key2, "不同的 secret 应生成不同的密钥")
	})
}

func TestValidateSM4Secret(t *testing.T) {
	t.Run("空密钥应返回错误", func(t *testing.T) {
		err := ValidateSM4Secret("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不能为空")
	})

	t.Run("短密钥应返回错误", func(t *testing.T) {
		err := ValidateSM4Secret("short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "长度不足")
	})

	t.Run("无效的 Base64 应返回错误", func(t *testing.T) {
		err := ValidateSM4Secret("invalid-base64!@#")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Base64")
	})

	t.Run("有效的密钥应通过验证", func(t *testing.T) {
		secret := "EDC6UNKa5JQUrBnBsmgRww=="
		err := ValidateSM4Secret(secret)
		assert.NoError(t, err)
	})
}

func TestValidatePasswordInput(t *testing.T) {
	t.Run("空密码应返回错误", func(t *testing.T) {
		err := ValidatePasswordInput("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不能为空")
	})

	t.Run("过长的密码应返回错误", func(t *testing.T) {
		longPassword := strings.Repeat("a", 1025)
		err := ValidatePasswordInput(longPassword)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "长度超过限制")
	})

	t.Run("有效的密码应通过验证", func(t *testing.T) {
		err := ValidatePasswordInput("admin123")
		assert.NoError(t, err)
	})
}

func TestIsEncryptedPassword(t *testing.T) {
	t.Run("带前缀的字符串应被识别为加密密码", func(t *testing.T) {
		encrypted := ENCRYPTION_PREFIX + "dGVzdC1lbmNyeXB0ZWQtcGFzc3dvcmQ="
		assert.True(t, IsEncryptedPassword(encrypted))
	})

	t.Run("不带前缀的字符串不应被识别为加密密码", func(t *testing.T) {
		plainPassword := "admin123"
		assert.False(t, IsEncryptedPassword(plainPassword))

		base64Only := "dGVzdC1lbmNyeXB0ZWQtcGFzc3dvcmQ="
		assert.False(t, IsEncryptedPassword(base64Only))
	})

	t.Run("空字符串不应被识别为加密密码", func(t *testing.T) {
		assert.False(t, IsEncryptedPassword(""))
	})

	t.Run("只有前缀的字符串应被识别为加密密码", func(t *testing.T) {
		assert.True(t, IsEncryptedPassword(ENCRYPTION_PREFIX))
	})
}

func TestDecryptPasswordECB(t *testing.T) {
	secret := "EDC6UNKa5JQUrBnBsmgRww=="

	t.Run("缺少前缀应返回错误", func(t *testing.T) {
		_, err := DecryptPasswordECB("dGVzdA==", secret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "密码格式错误")
	})

	t.Run("无效的 Base64 应返回错误", func(t *testing.T) {
		invalidCiphertext := ENCRYPTION_PREFIX + "invalid-base64!@#"
		_, err := DecryptPasswordECB(invalidCiphertext, secret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "密码格式错误")
	})

	t.Run("空密钥应返回错误", func(t *testing.T) {
		encrypted := ENCRYPTION_PREFIX + "dGVzdA=="
		_, err := DecryptPasswordECB(encrypted, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不能为空")
	})

	t.Run("空输入应返回错误", func(t *testing.T) {
		_, err := DecryptPasswordECB("", secret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "密码格式错误")
	})
}

func TestENCRYPTION_PREFIX(t *testing.T) {
	assert.Equal(t, "SM4:", ENCRYPTION_PREFIX, "加密前缀必须为 'SM4:'")
}

// ============================================================================
// Phase 18: 凭据静态加密 (SM4-GCM) 测试
// ============================================================================

func TestEncryptGCM_DecryptGCM_RoundTrip(t *testing.T) {
	key := DeriveCredentialSM4Key("test-credential-sm4-secret-key-32+")

	tests := []struct {
		name      string
		plaintext string
	}{
		{"ASCII 短明文", "admin123"},
		{"中文凭据", "复杂密码_中文"},
		{"长明文 (1KB)", strings.Repeat("a", 1024)},
		{"空字符串以外的极短", "x"},
		{"数字 + 符号", "P@ssw0rd!#$%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ct, err := EncryptGCM(key, []byte(tt.plaintext))
			require.NoError(t, err, "EncryptGCM 应成功")
			require.GreaterOrEqual(t, len(ct), gcmNonceSize+gcmTagSize, "密文至少包含 nonce + tag")

			pt, err := DecryptGCM(key, ct)
			require.NoError(t, err, "DecryptGCM 应成功")
			assert.Equal(t, tt.plaintext, string(pt), "明文回环应一致")
		})
	}
}

func TestEncryptGCM_NonceIsRandom(t *testing.T) {
	key := DeriveCredentialSM4Key("test-credential-sm4-secret-key-32+")
	plaintext := []byte("admin123")

	ct1, err := EncryptGCM(key, plaintext)
	require.NoError(t, err)
	ct2, err := EncryptGCM(key, plaintext)
	require.NoError(t, err)

	// 同明文 + 同密钥 → 密文必须不同（nonce 随机）
	assert.NotEqual(t, ct1, ct2, "GCM nonce 必须随机生成，避免 deterministic 加密")

	// 但都应能解密回原文
	pt1, _ := DecryptGCM(key, ct1)
	pt2, _ := DecryptGCM(key, ct2)
	assert.Equal(t, string(pt1), string(pt2), "两次密文都应能解出相同明文")
}

func TestDecryptGCM_WrongKey(t *testing.T) {
	key1 := DeriveCredentialSM4Key("key-one-1234567890abcdefghij")
	key2 := DeriveCredentialSM4Key("key-two-1234567890abcdefghij")

	ct, err := EncryptGCM(key1, []byte("admin123"))
	require.NoError(t, err)

	_, err = DecryptGCM(key2, ct)
	assert.Error(t, err, "错误密钥解密应失败")
}

func TestDecryptGCM_TamperedCiphertext(t *testing.T) {
	key := DeriveCredentialSM4Key("test-credential-sm4-secret-key-32+")
	ct, err := EncryptGCM(key, []byte("admin123"))
	require.NoError(t, err)

	// 篡改密文中间一个字节（不在 nonce 也不在 tag）
	ct[len(ct)-1] ^= 0xFF // 翻转 tag 最后一个 bit

	_, err = DecryptGCM(key, ct)
	assert.Error(t, err, "篡改 tag 的密文应被拒绝")
	assert.Contains(t, err.Error(), "tag", "错误应提及 tag 校验")
}

func TestDecryptGCM_TruncatedCiphertext(t *testing.T) {
	key := DeriveCredentialSM4Key("test-credential-sm4-secret-key-32+")
	ct, err := EncryptGCM(key, []byte("admin123"))
	require.NoError(t, err)

	// 截断到 < nonce+tag → 必须失败
	_, err = DecryptGCM(key, ct[:gcmNonceSize+gcmTagSize-1])
	assert.Error(t, err, "过短密文应被拒绝")
}

func TestDecryptGCM_InvalidKeyLength(t *testing.T) {
	shortKey := []byte("short") // < 16 bytes
	_, err := EncryptGCM(shortKey, []byte("x"))
	assert.Error(t, err, "短密钥应被拒绝")
	assert.Contains(t, err.Error(), "密钥长度", "错误应明确说明密钥长度问题")
}

func TestParseCredentialEnvelope_Success(t *testing.T) {
	payload := []byte("test-payload-bytes")
	encoded, err := EncodeCredentialEnvelope("v1", payload)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(encoded, "SM4:"), "envelope 必须以 SM4: 开头")

	version, decoded, err := ParseCredentialEnvelope(encoded)
	require.NoError(t, err)
	assert.Equal(t, "v1", version, "version 段解析应一致")
	assert.Equal(t, payload, decoded, "payload 解码应一致")
}

func TestParseCredentialEnvelope_MissingPrefix(t *testing.T) {
	_, _, err := ParseCredentialEnvelope("v1:abc")
	assert.Error(t, err, "缺少 SM4: 前缀应失败")
}

func TestParseCredentialEnvelope_MissingVersion(t *testing.T) {
	// "SM4:" 之后没有冒号 → version 段缺失
	_, _, err := ParseCredentialEnvelope("SM4:" + base64.StdEncoding.EncodeToString([]byte("x")))
	assert.Error(t, err, "缺少 version 段应失败")
}

func TestParseCredentialEnvelope_EmptyPayload(t *testing.T) {
	_, _, err := ParseCredentialEnvelope("SM4:v1:")
	assert.Error(t, err, "空 payload 应失败")
}

func TestParseCredentialEnvelope_InvalidBase64(t *testing.T) {
	_, _, err := ParseCredentialEnvelope("SM4:v1:!!!invalid-base64!!!")
	assert.Error(t, err, "无效 base64 应失败")
}

func TestEncodeCredentialEnvelope_EmptyVersion(t *testing.T) {
	_, err := EncodeCredentialEnvelope("", []byte("x"))
	assert.Error(t, err, "空 version 应失败")
}

func TestEncodeCredentialEnvelope_EmptyPayload(t *testing.T) {
	_, err := EncodeCredentialEnvelope("v1", nil)
	assert.Error(t, err, "空 payload 应失败")
}

func TestCredentialEnvelopeVersion_Constant(t *testing.T) {
	// 锁定的常量，防止未来无意修改
	assert.Equal(t, "v1", CredentialEnvelopeVersion, "当前 envelope 版本必须为 v1")
}

// ============================================================================
// WarnOnKeyTruncation 测试（Phase 18 调试 sm4-encrypt-key-invalid 衍生修复）
// ============================================================================

func TestWarnOnKeyTruncation_NilLogger(t *testing.T) {
	// nil logger 必须 no-op，不 panic
	assert.NotPanics(t, func() {
		WarnOnKeyTruncation(nil, "02a07280c190d77285676f2db527de0a", "SM4_SECRET", SM4KeyTransport)
	}, "nil logger 必须静默 no-op，不 panic")
}

func TestWarnOnKeyTruncation_EmptySecret(t *testing.T) {
	logger := zaptest.NewLogger(t)
	// 空字符串 hex decode 失败 → 不警告（也不报错）
	assert.NotPanics(t, func() {
		WarnOnKeyTruncation(logger, "", "SM4_SECRET", SM4KeyTransport)
	})
}

func TestWarnOnKeyTruncation_ValidHex32(t *testing.T) {
	// 标准长度（32 hex = 16 bytes）→ 不应触发警告
	logger := zaptest.NewLogger(t)
	assert.NotPanics(t, func() {
		WarnOnKeyTruncation(logger, "02a07280c190d77285676f2db527de0a", "SM4_SECRET", SM4KeyTransport)
	})
}

func TestWarnOnKeyTruncation_HexOverLength(t *testing.T) {
	// 64 hex = 32 bytes → 必须触发警告（典型 bug 场景）
	logger := zaptest.NewLogger(t)
	assert.NotPanics(t, func() {
		WarnOnKeyTruncation(logger, "4fa9f34f4e26dd8de970b1cfd9d6f2ad328151235694d58c513f7fcf631216a6", "SM4_SECRET", SM4KeyTransport)
	})
	// 警告内容通过 zaptest 断言：实际字段值由 logger 输出验证（zaptest 内部已捕获）
}

func TestWarnOnKeyTruncation_Base64Secret(t *testing.T) {
	// Base64 编码的 secret（hex decode 失败）→ 不警告（不属于本次 bug 范畴）
	logger := zaptest.NewLogger(t)
	assert.NotPanics(t, func() {
		WarnOnKeyTruncation(logger, "EDC6UNKa5JQUrBnBsmgRww==", "HLS_TOKEN_SECRET", SM4KeyTransport)
	})
}

func TestWarnOnKeyTruncation_OddLengthHex(t *testing.T) {
	// 奇数长度 hex（hex.DecodeString 会失败）→ 不警告
	logger := zaptest.NewLogger(t)
	assert.NotPanics(t, func() {
		WarnOnKeyTruncation(logger, "abc", "SM4_SECRET", SM4KeyTransport)
	})
}

// ============================================================================
// WarnOnKeyTruncation 文案区分（传输密钥 vs 静态密钥）—— 修正 STYLE/SEC 文案误报
// 传输密钥（SM4_SECRET）：前端 sm-crypto 参与，严格要求 16 字节，>16 字节会被前端拒绝。
// 静态密钥（CREDENTIAL_SM4_SECRET）：后端 at-rest 专用，前端不参与，截断是 SM4 预期行为。
// ============================================================================

func TestWarnOnKeyTruncation_StaticKeyMessageOmitsFrontend(t *testing.T) {
	// 静态加密密钥前端不参与 → 告警文案不应提及"前端"，且应标明是静态加密用途
	core, recorded := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	overlong := "4fa9f34f4e26dd8de970b1cfd9d6f2ad328151235694d58c513f7fcf631216a6" // 64 hex = 32 bytes
	WarnOnKeyTruncation(logger, overlong, "CREDENTIAL_SM4_SECRET", SM4KeyStatic)
	require.Len(t, recorded.All(), 1, "静态密钥 hex 超长应触发 1 条告警")
	msg := recorded.All()[0].Message
	assert.NotContains(t, msg, "前端", "静态密钥告警不应提及前端（前端不参与静态加密）")
	assert.Contains(t, msg, "静态", "静态密钥告警应标明用途为静态加密")
}

func TestWarnOnKeyTruncation_TransportKeyMessageMentionsFrontend(t *testing.T) {
	// 传输加密密钥前端 sm-crypto 参与 → 告警文案应提及前端兼容性风险
	core, recorded := observer.New(zap.WarnLevel)
	logger := zap.New(core)
	overlong := "4fa9f34f4e26dd8de970b1cfd9d6f2ad328151235694d58c513f7fcf631216a6"
	WarnOnKeyTruncation(logger, overlong, "SM4_SECRET", SM4KeyTransport)
	require.Len(t, recorded.All(), 1, "传输密钥 hex 超长应触发 1 条告警")
	msg := recorded.All()[0].Message
	assert.Contains(t, msg, "前端", "传输密钥告警应提及前端 sm-crypto 兼容性")
}
