// SM4-ECB 加密模式（与后端兼容）
import { sm4 } from 'sm-crypto'

const SM4_KEY_SIZE = 16 // SM4 密钥 16 字节

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

  // 取前 16 字节并转换为 Base64
  const keyBytes = hashArray.slice(0, SM4_KEY_SIZE)
  return btoa(String.fromCharCode(...keyBytes))
}

/**
 * SM4-ECB 加密密码
 * @param password 明文密码
 * @param key Base64 编码的 SM4 密钥
 * @returns Base64 编码的密文
 */
export function encryptPassword(password: string, key: string): string {
  try {
    // sm-crypto 的 sm4.encrypt 使用 ECB 模式
    const encrypted = sm4.encrypt(password, key)
    return encrypted
  } catch (error) {
    throw new Error(`SM4 加密失败: ${error}`)
  }
}

/**
 * SM4-ECB 解密密码（用于测试验证）
 * @param encrypted Base64 编码的密文
 * @param key Base64 编码的 SM4 密钥
 * @returns 明文密码
 */
export function decryptPassword(encrypted: string, key: string): string {
  try {
    const decrypted = sm4.decrypt(encrypted, key)
    return decrypted
  } catch (error) {
    throw new Error(`SM4 解密失败: ${error}`)
  }
}

/**
 * 检测字符串是否为 SM4 加密格式
 * SM4-ECB 加密后的 Base64 长度必须是 4 的倍数
 * 且密码长度通常 > 32 字符
 */
export function isEncryptedPassword(password: string): boolean {
  // Base64 字符集检查
  const base64Regex = /^[A-Za-z0-9+/=]+$/
  if (!base64Regex.test(password)) return false

  // 长度检查（SM4-ECB 加密后长度为 4 的倍数）
  if (password.length < 32 || password.length % 4 !== 0) return false

  return true
}

/**
 * 获取加密密钥（从环境变量）
 * @returns SM4 加密密钥，如果未配置则返回空字符串
 */
export function getEncryptionKey(): string {
  return import.meta.env.VITE_SM4_SECRET || ''
}
