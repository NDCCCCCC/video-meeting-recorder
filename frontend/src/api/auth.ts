// 认证 API 客户端

import {
  ApiResponse,
  LoginRequest,
  LoginResponse,
  ChangePasswordRequest,
  ValidationResult,
  User,
  ADUserLookupResult,
} from '../types/auth'
import { apiRequest, clearToken as apiClearToken, saveToken } from './apiClient'
import { encryptPassword, getEncryptionKey } from '../utils/sm4'

// API 基础 URL
const API_BASE_URL = import.meta.env.VITE_API_URL || ''

// 登录（不需要认证）
export async function login(req: LoginRequest): Promise<ApiResponse<LoginResponse>> {
  // 输入验证
  if (!req.username || req.username.trim() === '') {
    throw new Error('用户名不能为空')
  }

  if (!req.password || req.password.trim() === '') {
    throw new Error('密码不能为空')
  }

  // 获取加密密钥
  const encryptionKey = getEncryptionKey()

  // 加密密码
  const encryptedPassword = encryptionKey
    ? encryptPassword(req.password, encryptionKey)
    : req.password // 如果没有密钥则使用明文（向后兼容）

  // 构建请求体（使用加密后的密码）
  const loginRequest = {
    username: req.username,
    password: encryptedPassword,
  }

  const url = `${API_BASE_URL}/api/v1/auth/login`
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(loginRequest),
  })

  const data: ApiResponse<LoginResponse> = await response.json()

  if (!response.ok) {
    throw new Error(data.message || 'Login failed')
  }

  if (data.data) {
    saveToken(data.data.access_token, data.data.refresh_token)
    // 注意：不再手动更新 auth-storage，避免与 zustand persist 竞态
    // zustand persist 会在 authStore.login() 的 set() 调用时自动处理持久化
  }

  return data
}

// 登出
export async function logout(): Promise<void> {
  try {
    await apiRequest('/api/v1/auth/logout', {
      method: 'POST',
    })
  } finally {
    apiClearToken()
  }
}

// 登出所有设备
export async function logoutAll(): Promise<void> {
  await apiRequest('/api/v1/auth/logout-all', {
    method: 'POST',
  })
  apiClearToken()
}

// 获取当前用户信息
export async function getCurrentUser(): Promise<ApiResponse<User>> {
  return apiRequest<User>('/api/v1/auth/me')
}

// 修改密码
export async function changePassword(req: ChangePasswordRequest): Promise<ApiResponse<void>> {
  return apiRequest('/api/v1/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 验证密码强度
export async function validatePassword(password: string): Promise<ApiResponse<ValidationResult>> {
  return apiRequest<ValidationResult>('/api/v1/auth/validate-password', {
    method: 'POST',
    body: JSON.stringify({ password }),
  })
}

// 导出 clearToken
export { apiClearToken as clearToken }

// ============= AD 认证配置 API =============

import type {
  AuthConfigResponse,
  UpdateAuthConfigRequest,
  ADValidationResult,
  ADAuthConfig,
} from '../types/auth'

/**
 * Get current authentication configuration
 */
export async function getAuthConfig(): Promise<ApiResponse<AuthConfigResponse>> {
  return apiRequest<AuthConfigResponse>('/api/v1/admin/auth/config')
}

/**
 * Update authentication configuration
 */
export async function updateAuthConfig(
  data: UpdateAuthConfigRequest
): Promise<ApiResponse<{ message: string }>> {
  return apiRequest<{ message: string }>('/api/v1/admin/auth/config', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

/**
 * Test AD connection
 */
export async function testADConnection(
  config: ADAuthConfig
): Promise<ApiResponse<ADValidationResult>> {
  return apiRequest<ADValidationResult>('/api/v1/auth/ad/test-connection', {
    method: 'POST',
    body: JSON.stringify(config),
  })
}

/**
 * Lookup AD user by username
 */
export async function lookupADUser(username: string): Promise<ApiResponse<ADUserLookupResult>> {
  return apiRequest<ADUserLookupResult>('/api/v1/admin/auth/lookup-ad-user', {
    method: 'POST',
    body: JSON.stringify({ username }),
  })
}
