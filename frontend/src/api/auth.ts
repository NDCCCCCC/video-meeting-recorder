// 认证 API 客户端

import {
  ApiResponse,
  LoginRequest,
  LoginResponse,
  RefreshTokenResponse,
  ChangePasswordRequest,
  ValidationResult,
  User
} from '../types/auth'

// API 基础 URL
const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// 获取存储的 Token
const getToken = (): string | null => {
  return localStorage.getItem('access_token')
}

// 获取存储的刷新 Token
const getRefreshToken = (): string | null => {
  return localStorage.getItem('refresh_token')
}

// 保存 Token
const saveToken = (accessToken: string, refreshToken: string): void => {
  localStorage.setItem('access_token', accessToken)
  localStorage.setItem('refresh_token', refreshToken)
}

// 清除 Token
const clearToken = (): void => {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
}

// 通用请求函数
async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = `${API_BASE_URL}${endpoint}`

  // 添加认证头
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }

  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers,
    })

    const data: ApiResponse<T> = await response.json()

    // 处理 401 未授权
    if (response.status === 401) {
      // 尝试刷新 Token
      const refreshToken = getRefreshToken()
      if (refreshToken) {
        try {
          const newTokens = await refreshAccessToken(refreshToken)
          saveToken(newTokens.access_token, newTokens.refresh_token)

          // 重试原始请求
          const newHeaders = { ...headers }
          newHeaders['Authorization'] = `Bearer ${newTokens.access_token}`

          const retryResponse = await fetch(url, {
            ...options,
            headers: newHeaders,
          })
          return await retryResponse.json()
        } catch (error) {
          // 刷新失败，清除 Token 并跳转登录
          clearToken()
          window.location.href = '/auth/login'
          throw new Error('Session expired')
        }
      } else {
        clearToken()
        window.location.href = '/auth/login'
        throw new Error('Unauthorized')
      }
    }

    if (!response.ok) {
      throw new Error(data.message || 'Request failed')
    }

    return data
  } catch (error) {
    if (error instanceof Error) {
      throw error
    }
    throw new Error('Network error')
  }
}

// 登录
export async function login(req: LoginRequest): Promise<ApiResponse<LoginResponse>> {
  const response = await request<LoginResponse>('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify(req),
  })

  if (response.data) {
    saveToken(response.data.access_token, response.data.refresh_token)
  }

  return response
}

// 刷新 Token
async function refreshAccessToken(refreshToken: string): Promise<RefreshTokenResponse> {
  const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ refreshToken }),
  })

  if (!response.ok) {
    throw new Error('Failed to refresh token')
  }

  const data: ApiResponse<RefreshTokenResponse> = await response.json()
  return data.data!
}

// 登出
export async function logout(): Promise<void> {
  try {
    await request('/api/v1/auth/logout', {
      method: 'POST',
    })
  } finally {
    clearToken()
  }
}

// 登出所有设备
export async function logoutAll(): Promise<void> {
  await request('/api/v1/auth/logout-all', {
    method: 'POST',
  })
  clearToken()
}

// 获取当前用户信息
export async function getCurrentUser(): Promise<ApiResponse<User>> {
  return request<User>('/api/v1/auth/me')
}

// 修改密码
export async function changePassword(req: ChangePasswordRequest): Promise<ApiResponse<void>> {
  return request('/api/v1/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 验证密码强度
export async function validatePassword(password: string): Promise<ApiResponse<ValidationResult>> {
  return request<ValidationResult>('/api/v1/auth/validate-password', {
    method: 'POST',
    body: JSON.stringify({ password }),
  })
}

export { clearToken }
