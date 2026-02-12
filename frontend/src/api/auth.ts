// 认证 API 客户端

import {
  ApiResponse,
  LoginRequest,
  LoginResponse,
  ChangePasswordRequest,
  ValidationResult,
  User
} from '../types/auth'
import { apiRequest, clearToken as apiClearToken } from './apiClient'

// API 基础 URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// 保存 Token 到 localStorage（用于登录时）
const saveToken = (accessToken: string, refreshToken: string): void => {
  localStorage.setItem('access_token', accessToken)
  localStorage.setItem('refresh_token', refreshToken)
}

// 登录（不需要认证）
export async function login(req: LoginRequest): Promise<ApiResponse<LoginResponse>> {
  const url = `${API_BASE_URL}/api/v1/auth/login`
  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  })

  const data: ApiResponse<LoginResponse> = await response.json()

  if (!response.ok) {
    throw new Error(data.message || 'Login failed')
  }

  if (data.data) {
    saveToken(data.data.access_token, data.data.refresh_token)
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
