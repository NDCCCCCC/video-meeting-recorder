package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/tjfoc/gmsm/sm4"
)

const ENCRYPTION_PREFIX = "SM4:" // 加密前缀，与前端保持一致

// DeriveSM4Key 从密钥字符串派生16字节SM4密钥
// 与 auth.deriveSM4Key 相同的实现，用于密码解密
func DeriveSM4Key(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:16]
}

// DecryptPasswordECB 使用 SM4-ECB 模式解密密码
// 密文格式: SM4: 前缀 + Base64 编码的 SM4-ECB 加密数据
func DecryptPasswordECB(ciphertext string, sm4Secret string) (string, error) {
	// 1. 移除前缀标记
	if !strings.HasPrefix(ciphertext, ENCRYPTION_PREFIX) {
		return "", errors.New("密码格式错误: 缺少加密前缀")
	}
	ciphertext = strings.TrimPrefix(ciphertext, ENCRYPTION_PREFIX)

	// 2. 派生密钥
	key := DeriveSM4Key(sm4Secret)

	// 3. Base64 解码
	cipherData, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", errors.New("密码格式错误")
	}

	// 4. 验证密文长度
	if len(cipherData)%sm4.BlockSize != 0 {
		return "", errors.New("密码格式错误")
	}

	// 5. 创建 SM4 加密器
	block, err := sm4.NewCipher(key)
	if err != nil {
		return "", errors.New("密码格式错误")
	}

	// 6. ECB 模式解密
	plaintext := make([]byte, len(cipherData))
	for i := 0; i < len(cipherData); i += sm4.BlockSize {
		block.Decrypt(plaintext[i:i+sm4.BlockSize], cipherData[i:i+sm4.BlockSize])
	}

	// 7. 移除 PKCS7 填充
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
