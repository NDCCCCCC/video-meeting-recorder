package utils

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"

	"github.com/tjfoc/gmsm/sm4"
)

// DeriveSM4Key 从密钥字符串派生16字节SM4密钥
// 与 auth.deriveSM4Key 相同的实现，用于密码解密
func DeriveSM4Key(secret string) []byte {
	hash := sha256.Sum256([]byte(secret))
	return hash[:16]
}

// DecryptPasswordECB 使用 SM4-ECB 模式解密密码
// 密文格式: Base64 编码的 SM4-ECB 加密数据
func DecryptPasswordECB(ciphertext string, sm4Secret string) (string, error) {
	// 1. 派生密钥
	key := DeriveSM4Key(sm4Secret)

	// 2. Base64 解码
	cipherData, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", errors.New("密码格式错误: Base64 解码失败")
	}

	// 3. 验证密文长度（SM4 分组大小为 16 字节）
	if len(cipherData)%sm4.BlockSize != 0 {
		return "", errors.New("密码格式错误: 密文长度无效")
	}

	// 4. 创建 SM4 加密器
	block, err := sm4.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 5. ECB 模式解密（逐块解密）
	plaintext := make([]byte, len(cipherData))
	for i := 0; i < len(cipherData); i += sm4.BlockSize {
		block.Decrypt(plaintext[i:i+sm4.BlockSize], cipherData[i:i+sm4.BlockSize])
	}

	// 6. 移除 PKCS7 填充
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > sm4.BlockSize {
		return "", errors.New("密码格式错误: 填充无效")
	}

	plaintext = plaintext[:len(plaintext)-padding]

	return string(plaintext), nil
}

// IsEncryptedPassword 检测密码是否为 SM4 加密格式
// 通过 Base64 格式和长度特征判断
func IsEncryptedPassword(password string) bool {
	// 长度检查（SM4-ECB 加密后通常 > 32 字符）
	if len(password) < 32 {
		return false
	}

	// Base64 格式检查
	_, err := base64.StdEncoding.DecodeString(password)
	return err == nil
}
