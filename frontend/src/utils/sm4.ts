// SM4-ECB 加密模式（与后端兼容）
import { sm4 } from 'sm-crypto'

const SM4_KEY_SIZE = 16 // SM4 密钥 16 字节
export const ENCRYPTION_PREFIX = 'SM4:' // 加密前缀标记，用于可靠检测加密密码

/**
 * 从字符串派生 SM4 密钥（与后端 deriveSM4Key 兼容）
 * 使用 SHA256 哈希的前 16 字节
 */
export async function deriveSM4Key(secret: string): Promise<string> {
  // 使用 crypto subtle API 进行 SHA256
  const encoder = new TextEncoder()
  const data = encoder.encode(secret)

  // 在浏览器环境使用 SubtleCrypto
  const hashBuffer = await crypto.subtle.digest('SHA-256', data)
  const hashArray = Array.from(new Uint8Array(hashBuffer))

  // 取前 16 字节并转换为十六进制字符串（sm-crypto expects 32-char hex key）
  const keyBytes = hashArray.slice(0, SM4_KEY_SIZE)
  return keyBytes.map((b) => b.toString(16).padStart(2, '0')).join('')
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

    // 将十六进制转换为 Base64（与后端兼容）
    const encrypted = hexToBase64(hexEncrypted)

    // 添加前缀标记，确保解密检测不会被绕过
    return `${ENCRYPTION_PREFIX}${encrypted}`
  } catch (error) {
    throw new Error(`Failed to encrypt password: ${error}`)
  }
}

/**
 * 十六进制字符串转 Base64
 */
function hexToBase64(hexString: string): string {
  const hexBytes = new Uint8Array(hexString.match(/.{1,2}/g)!.map((byte) => parseInt(byte, 16)))
  return btoa(String.fromCharCode(...hexBytes))
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
