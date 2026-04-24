package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
