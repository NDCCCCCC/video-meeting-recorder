package utils

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/tjfoc/gmsm/sm4"
)

const ENCRYPTION_PREFIX = "SM4:" // 加密前缀，与前端保持一致

// ValidateSM4Secret 验证 SM4 密钥的有效性
func ValidateSM4Secret(secret string) error {
	if secret == "" {
		return errors.New("SM4 密钥不能为空")
	}

	// 验证密钥长度（建议至少 16 字符）
	if len(secret) < 16 {
		return errors.New("SM4 密钥长度不足，至少需要 16 字符")
	}

	// 验证密钥必须是可解码的 Base64（避免「无错误即视为通过」导致的 panic）
	if _, err := base64.StdEncoding.DecodeString(secret); err != nil {
		return fmt.Errorf("SM4 密钥不是有效的 Base64: %w", err)
	}

	return nil
}

// ValidatePasswordInput 验证密码输入的有效性
func ValidatePasswordInput(password string) error {
	if password == "" {
		return errors.New("密码不能为空")
	}

	// 防止过长的密码导致 DoS
	if len(password) > 1024 {
		return errors.New("密码长度超过限制")
	}

	return nil
}

// DeriveSM4Key 从密钥字符串获取SM4密钥
// 将十六进制字符串转换为字节（与前端 sm-crypto 兼容）
func DeriveSM4Key(secret string) []byte {
	// 将十六进制字符串解码为字节
	// 32个十六进制字符 = 16字节
	keyBytes, err := hex.DecodeString(secret)
	if err != nil {
		// 如果不是有效的十六进制，回退到原始方式
		keyBytes = []byte(secret)
		if len(keyBytes) > 16 {
			return keyBytes[:16]
		}
		if len(keyBytes) < 16 {
			padded := make([]byte, 16)
			copy(padded, keyBytes)
			return padded
		}
		return keyBytes
	}
	// 确保密钥至少16字节
	if len(keyBytes) < 16 {
		padded := make([]byte, 16)
		copy(padded, keyBytes)
		return padded
	}
	return keyBytes[:16]
}

// DecryptPasswordECB 使用 SM4-ECB 模式解密密码
// 密文格式: SM4: 前缀 + Base64 编码的 SM4-ECB 加密数据
func DecryptPasswordECB(ciphertext string, sm4Secret string) (string, error) {
	// 1. 验证输入
	if err := ValidatePasswordInput(ciphertext); err != nil {
		return "", errors.New("密码格式错误")
	}

	if err := ValidateSM4Secret(sm4Secret); err != nil {
		return "", err
	}

	// 2. 移除前缀标记
	if !strings.HasPrefix(ciphertext, ENCRYPTION_PREFIX) {
		return "", errors.New("密码格式错误")
	}
	ciphertext = strings.TrimPrefix(ciphertext, ENCRYPTION_PREFIX)

	// 3. 派生密钥
	key := DeriveSM4Key(sm4Secret)

	// 4. Base64 解码
	cipherData, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", errors.New("密码格式错误")
	}

	// 5. 验证密文长度
	if len(cipherData)%sm4.BlockSize != 0 {
		return "", errors.New("密码格式错误")
	}

	// 6. 创建 SM4 加密器
	block, err := sm4.NewCipher(key)
	if err != nil {
		return "", errors.New("密码格式错误")
	}

	// 7. ECB 模式解密
	plaintext := make([]byte, len(cipherData))
	for i := 0; i < len(cipherData); i += sm4.BlockSize {
		block.Decrypt(plaintext[i:i+sm4.BlockSize], cipherData[i:i+sm4.BlockSize])
	}

	// 8. 移除 PKCS7 填充
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > sm4.BlockSize {
		return "", errors.New("密码格式错误")
	}

	plaintext = plaintext[:len(plaintext)-padding]

	return string(plaintext), nil
}

// IsEncryptedPassword 检测密码是否为 SM4 加密格式
// 使用前缀标记进行可靠检测
func IsEncryptedPassword(password string) bool {
	// 使用前缀标记进行可靠检测
	return strings.HasPrefix(password, ENCRYPTION_PREFIX)
}
