// SM4-ECB 加密模式（与后端兼容）
import { sm4 } from 'sm-crypto'

export const ENCRYPTION_PREFIX = 'SM4:' // 加密前缀标记，用于可靠检测加密密码

/**
 * 从字符串派生 SM4 密钥（与后端 deriveSM4Key 兼容）
 *
 * ⚠️ 契约：secret 必须是 **恰好 32 字符的十六进制字符串**（hex-decode 后正好 16 字节）。
 * 这是 SM4 块密码的强制要求（sm-crypto strict-check：sm4/index.js:255-257 throw "key is invalid"
 * if key.length !== 16）。后端 DeriveSM4Key 在 hex-decode 后会**静默截断** > 16 字节的 key（这
 * 一历史行为掩盖了配置错误，见 internal/utils/sm4_password.go:DeriveSM4Key）。
 *
 * 正确生成方式：`openssl rand -hex 16` 输出 32 hex chars。`openssl rand -hex 32`
 * 输出的 64-char secret 在本函数中会被 sm-crypto 拒绝（前端登录报错 "key is invalid"），
 * 且后端会静默截断丢弃后 32 字节——这是 Phase 18 调试会话 sm4-encrypt-key-invalid 的根因。
 */
export async function deriveSM4Key(secret: string): Promise<string> {
  // 直接返回原始 hex 字符串。sm-crypto 的 sm4.encrypt/decrypt 内部 `hexToArray(key)`
  // 会按 1:2 转换（每 2 个 hex 字符 → 1 字节），所以 32-char 输入得到 16 字节 key。
  // 与后端 internal/utils/sm4_password.go DeriveSM4Key 的 `hex.DecodeString(secret)`
  // 行为完全一致——只要输入恰好 32 字符。
  return secret
}

/**
 * SM4-ECB 加密密码
 * @param password 明文密码
 * @param key SM4 密钥（32字符十六进制字符串）
 * @returns Base64 编码的密文（带前缀标记）
 */
export function encryptPassword(password: string, key: string): string {
  if (!password) {
    throw new Error('Password must not be empty')
  }
  try {
    // sm-crypto 的 sm4.encrypt 返回十六进制字符串
    const hexEncrypted = sm4.encrypt(password, key)

    // 把 hex 中间去掉，让存储的密文是 Base64(raw ECB bytes)。
    // 这样与后端 internal/utils/sm4_password.go DecryptPasswordECB 的格式
    // 完全一致：它做 `base64.StdEncoding.DecodeString(ciphertext) → raw bytes
    // → sm4 ECB decrypt`，前端也按 raw bytes Base64 喂回去。
    const rawBytes = hexToBytes(hexEncrypted)
    const encrypted = btoa(String.fromCharCode(...rawBytes))

    // 添加前缀标记，确保解密检测不会被绕过
    return `${ENCRYPTION_PREFIX}${encrypted}`
  } catch (error) {
    throw new Error(`Failed to encrypt password: ${error}`)
  }
}

/**
 * 十六进制字符串转字节数组
 */
function hexToBytes(hexString: string): Uint8Array {
  return new Uint8Array(hexString.match(/.{1,2}/g)!.map((byte) => parseInt(byte, 16)))
}

/**
 * Base64 字符串转十六进制
 */
function base64ToHex(b64String: string): string {
  const bytes = Uint8Array.from(atob(b64String), (c) => c.charCodeAt(0))
  return Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('')
}

/**
 * SM4-ECB 解密密码（用于测试验证）
 * @param encrypted Base64 编码的密文（带前缀标记）
 * @param key Base64 编码的 SM4 密钥
 * @returns 明文密码
 */
export function decryptPassword(encrypted: string, key: string): string {
  const ciphertext = encrypted.replace(ENCRYPTION_PREFIX, '')
  if (!ciphertext) {
    throw new Error('Encrypted password must not be empty after prefix removal')
  }
  try {
    // encryptPassword 存的是 Base64 密文；sm-crypto 需要 hex
    const hexCiphertext = base64ToHex(ciphertext)

    // sm-crypto 的 sm4.decrypt 使用 ECB 模式
    const decrypted = sm4.decrypt(hexCiphertext, key)
    return decrypted
  } catch (error) {
    throw new Error(`Failed to decrypt password: ${error}`)
  }
}

/**
 * 检测字符串是否为 SM4 加密格式（使用前缀标记）
 */
export function isEncryptedPassword(password: string): boolean {
  // 使用前缀标记进行可靠检测
  return password.startsWith(ENCRYPTION_PREFIX)
}

/**
 * 获取加密密钥（从环境变量）
 * @returns SM4 加密密钥，如果未配置则返回空字符串
 */
export function getEncryptionKey(): string {
  return import.meta.env.VITE_SM4_SECRET || ''
}
