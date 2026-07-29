import { describe, it, expect } from 'vitest'
import {
  deriveSM4Key,
  encryptPassword,
  decryptPassword,
  isEncryptedPassword,
  ENCRYPTION_PREFIX,
} from './sm4'

describe('SM4 Utils', () => {
  const testSecret = 'EDC6UNKa5JQUrBnBsmgRww=='
  const testPassword = 'admin123'

  describe('ENCRYPTION_PREFIX', () => {
    it('should be "SM4:"', () => {
      expect(ENCRYPTION_PREFIX).toBe('SM4:')
    })
  })

  describe('deriveSM4Key', () => {
    it('should derive consistent key from same secret', async () => {
      const key1 = await deriveSM4Key(testSecret)
      const key2 = await deriveSM4Key(testSecret)
      expect(key1).toBe(key2)
    })

    it('should derive different keys from different secrets', async () => {
      const key1 = await deriveSM4Key('secret1')
      const key2 = await deriveSM4Key('secret2')
      expect(key1).not.toBe(key2)
    })

    it('should derive key of correct length', async () => {
      const key = await deriveSM4Key(testSecret)
      expect(key.length).toBeGreaterThan(0)
    })
  })

  describe('encryptPassword and decryptPassword', () => {
    it('should encrypt and decrypt password correctly', async () => {
      const key = await deriveSM4Key(testSecret)
      const encrypted = encryptPassword(testPassword, key)
      const decrypted = decryptPassword(encrypted, key)

      expect(decrypted).toBe(testPassword)
    })

    it('should add prefix to encrypted password', async () => {
      const key = await deriveSM4Key(testSecret)
      const encrypted = encryptPassword(testPassword, key)

      expect(encrypted.startsWith(ENCRYPTION_PREFIX)).toBe(true)
    })

    it('should produce consistent ciphertext for same password (ECB mode)', async () => {
      const key = await deriveSM4Key(testSecret)
      const encrypted1 = encryptPassword(testPassword, key)
      const encrypted2 = encryptPassword(testPassword, key)

      expect(encrypted1).toBe(encrypted2)
    })

    it('should throw error for empty password', async () => {
      const key = await deriveSM4Key(testSecret)

      expect(() => encryptPassword('', key)).toThrow()
    })

    it('should throw error for invalid encrypted data', async () => {
      const key = await deriveSM4Key(testSecret)

      expect(() => decryptPassword('invalid', key)).toThrow()
    })
  })

  describe('isEncryptedPassword', () => {
    it('should detect encrypted password by prefix', async () => {
      const key = await deriveSM4Key(testSecret)
      const encrypted = encryptPassword(testPassword, key)

      expect(isEncryptedPassword(encrypted)).toBe(true)
    })

    it('should return false for plaintext password', () => {
      expect(isEncryptedPassword(testPassword)).toBe(false)
    })

    it('should return false for empty string', () => {
      expect(isEncryptedPassword('')).toBe(false)
    })

    it('should return false for string without prefix', () => {
      expect(isEncryptedPassword('dGVzdA==')).toBe(false)
    })

    it('should return true for string with only prefix', () => {
      expect(isEncryptedPassword(ENCRYPTION_PREFIX)).toBe(true)
    })
  })
})
