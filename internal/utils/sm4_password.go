package utils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	apperrors "github.com/NDCCCCCC/video-meeting-recorder/internal/errors"
	"github.com/tjfoc/gmsm/sm4"
	"go.uber.org/zap"
)

const (
	ENCRYPTION_PREFIX = "SM4:" // 加密前缀，与前端保持一致
	// CredentialEnvelopeVersion 是当前凭据静态加密的 envelope 版本号。
	// ciphertext envelope 格式: SM4:<version>:<base64(nonce_12B | ciphertext | tag_16B)>
	// 历史上 (Phase 18 之前) 所有凭据都直接走 ENCRYPTION_PREFIX + base64(ECB) 模式。
	// Phase 18 引入版本化 envelope，把"凭据静态加密"从"传输加密"中分离出来。
	CredentialEnvelopeVersion = "v1"
	// gcmNonceSize 是 SM4-GCM nonce（IV）长度，按 NIST SP 800-38D 推荐 96 位 = 12 字节。
	gcmNonceSize = 12
	// gcmTagSize 是 SM4-GCM 认证 tag 长度，按 GCM 规范为 128 位 = 16 字节。
	gcmTagSize = 16
)

// ValidateSM4Secret 验证 SM4 密钥的有效性
// Phase 19 D18: 3 散点统一 -> apperrors.ErrInvalidInput (400 输入参数错)。
func ValidateSM4Secret(secret string) error {
	if secret == "" {
		return fmt.Errorf("SM4 密钥不能为空: %w", apperrors.ErrInvalidInput)
	}

	// 验证密钥长度（建议至少 16 字符）
	if len(secret) < 16 {
		return fmt.Errorf("SM4 密钥长度不足，至少需要 16 字符: %w", apperrors.ErrInvalidInput)
	}

	// 验证密钥必须是可解码的 Base64（避免「无错误即视为通过」导致的 panic）
	if _, err := base64.StdEncoding.DecodeString(secret); err != nil {
		return fmt.Errorf("SM4 密钥不是有效的 Base64: %w: %w", apperrors.ErrInvalidInput, err)
	}

	return nil
}

// ValidatePasswordInput 验证密码输入的有效性
// Phase 19 D18: 2 散点统一 -> apperrors.ErrInvalidInput。
func ValidatePasswordInput(password string) error {
	if password == "" {
		return fmt.Errorf("密码不能为空: %w", apperrors.ErrInvalidInput)
	}

	// 防止过长的密码导致 DoS
	if len(password) > 1024 {
		return fmt.Errorf("密码长度超过限制: %w", apperrors.ErrInvalidInput)
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
// Phase 19 D18: 6 散点统一 -> apperrors.ErrInvalidInput (格式错 -> 400 BadRequest)。
func DecryptPasswordECB(ciphertext string, sm4Secret string) (string, error) {
	// 1. 验证输入
	if err := ValidatePasswordInput(ciphertext); err != nil {
		return "", fmt.Errorf("密码格式错误: %w", apperrors.ErrInvalidInput)
	}

	if err := ValidateSM4Secret(sm4Secret); err != nil {
		return "", err
	}

	// 2. 移除前缀标记
	if !strings.HasPrefix(ciphertext, ENCRYPTION_PREFIX) {
		return "", fmt.Errorf("密码格式错误: 缺少 SM4: 前缀: %w", apperrors.ErrInvalidInput)
	}
	ciphertext = strings.TrimPrefix(ciphertext, ENCRYPTION_PREFIX)

	// 3. 派生密钥
	key := DeriveSM4Key(sm4Secret)

	// 4. Base64 解码
	cipherData, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("密码格式错误: base64 解码失败: %w: %w", apperrors.ErrInvalidInput, err)
	}

	// 5. 验证密文长度
	if len(cipherData)%sm4.BlockSize != 0 {
		return "", fmt.Errorf("密码格式错误: 密文长度非块对齐: %w", apperrors.ErrInvalidInput)
	}

	// 6. 创建 SM4 加密器
	block, err := sm4.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("密码格式错误: 创建 SM4 cipher 失败: %w: %w", apperrors.ErrInvalidInput, err)
	}

	// 7. ECB 模式解密
	plaintext := make([]byte, len(cipherData))
	for i := 0; i < len(cipherData); i += sm4.BlockSize {
		block.Decrypt(plaintext[i:i+sm4.BlockSize], cipherData[i:i+sm4.BlockSize])
	}

	// 8. 移除 PKCS7 填充
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > sm4.BlockSize {
		return "", fmt.Errorf("密码格式错误: padding 非法: %w", apperrors.ErrInvalidInput)
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

// WarnOnKeyTruncation 检测 SM4 密钥 secret 在 hex-decode 后是否会被 DeriveSM4Key 静默截断，
// 若会被截断则用 logger.Warn 发出配置警告。**不阻塞启动**——只是让运维知道 secret 长度
// 不符合预期（期望 32 hex chars = 16 bytes），避免再次出现
// `openssl rand -hex 32` 生成 64-char secret 时后端静默吞后 32 字节、与前端 sm-crypto
// 严格要求 16 字节 key 不匹配的隐形 bug（Phase 18 调试会话 sm4-encrypt-key-invalid）。
//
// 调用约定：
//   - 仅在启动期校验函数（ValidateProductionSecrets / ValidateCredentialSM4Config）
//     中各调用一次，避免热路径日志噪音
//   - logger 为 nil 时静默 no-op（不输出）
//   - secret 非 hex（如 Base64 或裸字符串）时不警告（DeriveSM4Key 对非 hex 走 fallback
//     路径，截断行为依赖于原始字节长度 + 是否 ≥ 16，此处只针对 hex 路径最常见 bug）
//   - secretName 用于日志字段（如 "SM4_SECRET"、"CREDENTIAL_SM4_SECRET"）
func WarnOnKeyTruncation(logger *zap.Logger, secret, secretName string) {
	if logger == nil {
		return
	}
	keyBytes, err := hex.DecodeString(secret)
	if err != nil {
		// 非 hex 编码（Base64 或裸字符串）— 不属于本次静默截断 bug 的范畴，不警告。
		return
	}
	const sm4KeyBytes = 16
	if len(keyBytes) > sm4KeyBytes {
		logger.Warn("SM4 密钥长度超过 16 字节，DeriveSM4Key 将静默截断（前端 sm-crypto 会拒绝）",
			zap.String("secret_name", secretName),
			zap.Int("hex_length", len(secret)),
			zap.Int("decoded_bytes", len(keyBytes)),
			zap.Int("dropped_bytes", len(keyBytes)-sm4KeyBytes),
			zap.String("remediation", "重新生成 SM4_SECRET：openssl rand -hex 16（输出 32 hex chars = 16 bytes）"),
		)
	}
}

// ============================================================================
// Phase 18: 凭据静态加密（SM4-GCM，envelope: SM4:<version>:<base64>）
// ============================================================================
//
// 与 DecryptPasswordECB (SM4-ECB 传输加密) 严格分离：
//   - DecryptPasswordECB：解密前端浏览器用 SM4-ECB 加密后通过 HTTP body 传过来的明文
//     —— 属于"传输加密"。prefix 仍是 ENCRYPTION_PREFIX 但没有 version 段。
//   - EncryptGCM / DecryptGCM：把凭据明文落到 SQLite 之前用 SM4-GCM 加密，
//     envelope 是 "SM4:<version>:<base64(nonce | ct | tag)>"。
//     —— 属于"静态加密（at-rest）"。version 段用于密钥轮换（rotation）。
//
// 共享密钥不同：传输用 Auth.SM4Secret（CREDENTIAL_SM4_SECRET 与之解耦）。
// 静态加密密钥 = CREDENTIAL_SM4_SECRET，env 名为 CREDENTIAL_SM4_SECRET / CREDENTIAL_SM4_VERSION。

// DeriveCredentialSM4Key 把 CREDENTIAL_SM4_SECRET（任意 ≥ 32 字符高熵字符串）
// 归一化为 SM4 16 字节密钥。Phase 18 内部：取 hex.DecodeString 优先，失败回退到
// 取前 16 字节；与 DeriveSM4Key 行为对齐（避免两套派生规则产生不兼容 ciphertext）。
func DeriveCredentialSM4Key(secret string) []byte {
	return DeriveSM4Key(secret)
}

// EncryptGCM 用 SM4-GCM 加密明文，输出 "nonce_12B | ciphertext | tag_16B" 的拼接。
// 失败时返回的 error 不暴露密钥。
//
// 由于 gmsm/sm4 v1.4.1 的 Sm4GCM 解密路径要求密文长度 = BlockSize*n（即不处理非块对齐的最后一段），
// EncryptGCM 内部先对明文做 PKCS#7 风格 padding（补到 16 字节边界），DecryptGCM 内部移除。
// 这样做：(1) 任意长度明文都可加密；(2) 解密出"padded plaintext"后能用最后字节推回原长；
// (3) tag 仍然覆盖原始明文（padding 字节也参与 GHASH），tamper detection 不受影响。
//
// **重要**：gmsm 的 GetY0 在 96-bit IV 时直接 `IV = append(IV, 0001)`，
// 会**就地修改**调用方传入的 IV slice 的 backing array。当 IV 来自更大的 buffer
// （如我们 envelope 的 `nonce | ct | tag`），会污染 ct 段。本函数和 DecryptGCM
// 都传入**拷贝**后的 nonce，避免污染。
//
// Phase 19 D18: 4 散点统一 sentinel 化——密钥长度/nonce 生成/加密失败 -> ErrInternal;
//   tag 长度错 -> ErrInternal (gmsm 库 bug, 非用户输入)。
func EncryptGCM(key []byte, plaintext []byte) ([]byte, error) {
	if len(key) != sm4.BlockSize {
		return nil, fmt.Errorf("SM4-GCM 密钥长度错误: %d (期望 %d): %w",
			len(key), sm4.BlockSize, apperrors.ErrInternal)
	}
	padded := pkcs7Pad(plaintext, sm4.BlockSize)
	nonceSrc := make([]byte, gcmNonceSize)
	if _, err := io.ReadFull(rand.Reader, nonceSrc); err != nil {
		return nil, fmt.Errorf("SM4-GCM nonce 生成失败: %w: %w", apperrors.ErrInternal, err)
	}
	// 拷贝 nonce —— gmsm 会通过 append 把它就地扩展到 16 字节
	nonceForGCM := make([]byte, len(nonceSrc))
	copy(nonceForGCM, nonceSrc)
	ct, tag, err := sm4.Sm4GCM(key, nonceForGCM, padded, nil, true)
	if err != nil {
		return nil, fmt.Errorf("SM4-GCM 加密失败: %w: %w", apperrors.ErrInternal, err)
	}
	if len(tag) != gcmTagSize {
		return nil, fmt.Errorf("SM4-GCM tag 长度错误: %d (期望 %d): %w",
			len(tag), gcmTagSize, apperrors.ErrInternal)
	}
	out := make([]byte, 0, len(nonceSrc)+len(ct)+len(tag))
	out = append(out, nonceSrc...)
	out = append(out, ct...)
	out = append(out, tag...)
	return out, nil
}

// pkcs7Pad 把 data 填充到 blockSize 的整数倍，返回 (data + padBytes)，
// 其中最后 N 个字节都等于 N（N ∈ [1, blockSize]）。
// 这是 PKCS#7 在固定 blockSize 下的标准定义。
func pkcs7Pad(data []byte, blockSize int) []byte {
	padLen := blockSize - (len(data) % blockSize)
	out := make([]byte, len(data)+padLen)
	copy(out, data)
	for i := len(data); i < len(out); i++ {
		out[i] = byte(padLen)
	}
	return out
}

// pkcs7Unpad 还原 PKCS#7 padding。padding 非法时返回 error。
// Phase 19 D18: 3 散点统一 -> apperrors.ErrInvalidInput (padding 非法视作输入错)。
func pkcs7Unpad(padded []byte, blockSize int) ([]byte, error) {
	if len(padded) == 0 || len(padded)%blockSize != 0 {
		return nil, fmt.Errorf("PKCS#7 padding 长度非法: %d (blockSize=%d): %w",
			len(padded), blockSize, apperrors.ErrInvalidInput)
	}
	padLen := int(padded[len(padded)-1])
	if padLen < 1 || padLen > blockSize {
		return nil, fmt.Errorf("PKCS#7 padding 数值非法: %d (blockSize=%d): %w",
			padLen, blockSize, apperrors.ErrInvalidInput)
	}
	for i := len(padded) - padLen; i < len(padded); i++ {
		if padded[i] != byte(padLen) {
			return nil, fmt.Errorf("PKCS#7 padding 字节不一致（密文可能损坏）: %w", apperrors.ErrInvalidInput)
		}
	}
	return padded[:len(padded)-padLen], nil
}

// DecryptGCM 解密 SM4-GCM ciphertext。失败时返回的 error 不暴露密钥。
// 输入必须是 EncryptGCM 输出的原始字节（nonce | ct | tag）。
// Phase 19 D18: 4 散点统一 sentinel——密钥错/cipher 长度 -> ErrInvalidInput
//   (输入错); 解密失败 -> ErrInternal; tag 校验失败 -> ErrInvalidInput (密文被篡改)。
func DecryptGCM(key []byte, data []byte) ([]byte, error) {
	if len(key) != sm4.BlockSize {
		return nil, fmt.Errorf("SM4-GCM 密钥长度错误: %d (期望 %d): %w",
			len(key), sm4.BlockSize, apperrors.ErrInternal)
	}
	if len(data) < gcmNonceSize+gcmTagSize {
		return nil, fmt.Errorf("SM4-GCM ciphertext 过短: %d 字节: %w",
			len(data), apperrors.ErrInvalidInput)
	}
	nonce := data[:gcmNonceSize]
	ct := data[gcmNonceSize : len(data)-gcmTagSize]
	tag := data[len(data)-gcmTagSize:]
	// **拷贝 nonce**：gmsm 的 GetY0 通过 `append` 就地扩展 nonce 到 16 字节，
	// 如果直接传入 `nonce` 切片（其 backing array 来自 `data`），会污染 ct 段。
	nonceForGCM := make([]byte, len(nonce))
	copy(nonceForGCM, nonce)
	// gmsm Sm4GCM 解密分支的语义：
	//   - 不主动比对 tag（避免双重实现 GCM spec 的 tag 校验逻辑）
	//   - 第二个返回值 _T 是"基于提供的 ciphertext 用 GCM 模式复算出的 expected tag"
	//   - 我们必须自己把 expected tag 跟密文尾部的 tag 做常量时间比对
	pt, expectedTag, err := sm4.Sm4GCM(key, nonceForGCM, ct, nil, false)
	if err != nil {
		return nil, fmt.Errorf("SM4-GCM 解密失败: %w: %w", apperrors.ErrInternal, err)
	}
	if !constantTimeEqual(tag, expectedTag) {
		return nil, fmt.Errorf("SM4-GCM tag 校验失败：密文被篡改或密钥错误: %w", apperrors.ErrInvalidInput)
	}
	return pkcs7Unpad(pt, sm4.BlockSize)
}

// constantTimeEqual 做常量时间比较，避免 tag 比对被 timing attack 推断。
func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// ParseCredentialEnvelope 解析 "SM4:<version>:<base64>" envelope。
// 返回值：version 段（如 "v1"）+ 已 base64 解码后的 payload。
// envelope 格式错误时返回 error —— 永远不静默跳过。
// Phase 19 D18: 4 散点统一 -> apperrors.ErrInvalidInput (envelope 格式错 400)。
func ParseCredentialEnvelope(envelope string) (version string, payload []byte, err error) {
	if !strings.HasPrefix(envelope, ENCRYPTION_PREFIX) {
		return "", nil, fmt.Errorf("envelope 缺少 SM4: 前缀: %w", apperrors.ErrInvalidInput)
	}
	rest := strings.TrimPrefix(envelope, ENCRYPTION_PREFIX)
	// 期望格式: <version>:<base64> —— 至少一个冒号
	idx := strings.Index(rest, ":")
	if idx <= 0 {
		return "", nil, fmt.Errorf("envelope 缺少 version 段: %w", apperrors.ErrInvalidInput)
	}
	version = rest[:idx]
	encoded := rest[idx+1:]
	if encoded == "" {
		return "", nil, fmt.Errorf("envelope payload 为空: %w", apperrors.ErrInvalidInput)
	}
	payload, err = base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("envelope base64 解码失败: %w: %w", apperrors.ErrInvalidInput, err)
	}
	return version, payload, nil
}

// EncodeCredentialEnvelope 把 version + raw payload 编码成 "SM4:<version>:<base64>" envelope。
// Phase 19 D18: 2 散点统一 -> apperrors.ErrInvalidInput。
func EncodeCredentialEnvelope(version string, payload []byte) (string, error) {
	if version == "" {
		return "", fmt.Errorf("version 不能为空: %w", apperrors.ErrInvalidInput)
	}
	if len(payload) == 0 {
		return "", fmt.Errorf("payload 不能为空: %w", apperrors.ErrInvalidInput)
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	return ENCRYPTION_PREFIX + version + ":" + encoded, nil
}
