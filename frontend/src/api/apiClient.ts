// 统一的 API 客户端
// 处理所有 API 请求的认证、401 错误和 token 刷新

import type { ApiResponse } from '../types/auth'
import { message } from 'antd'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

// 正在刷新 token 的 Promise，防止并发刷新
let refreshingPromise: Promise<string> | null = null
// 正在登出的标志，防止并发请求重复登出
let isLoggingOut = false
// 上次刷新失败的时间戳，用于防止频繁重试
let lastRefreshFailureTime = 0
// 刷新失败后的冷却时间（毫秒）
const REFRESH_COOLDOWN = 5000

// 缓存 localStorage 读取以避免频繁解析 JSON (client-localstorage-schema)
let cachedToken: string | null = null
let cachedRefreshToken: string | null = null
let authStorageString: string | null = null

// 更新缓存
const updateTokenCache = () => {
  const currentAuthStorage = localStorage.getItem('auth-storage')
  if (currentAuthStorage !== authStorageString) {
    authStorageString = currentAuthStorage
    if (currentAuthStorage) {
      try {
        const parsed = JSON.parse(currentAuthStorage)
        cachedToken = parsed.state?.token || null
        cachedRefreshToken = parsed.state?.refreshToken || null
      } catch {
        cachedToken = null
        cachedRefreshToken = null
      }
    } else {
      cachedToken = localStorage.getItem('access_token')
      cachedRefreshToken = localStorage.getItem('refresh_token')
    }
  }
}

// 获取 Token - 使用缓存
export const getToken = (): string | null => {
  updateTokenCache()
  return cachedToken
}

// 获取刷新 Token - 使用缓存
const getRefreshToken = (): string | null => {
  updateTokenCache()
  return cachedRefreshToken
}

// 保存 Token（用于刷新后更新）
const saveToken = (accessToken: string, refreshToken: string): void => {
  // 重置登出标志，允许后续的登出操作
  isLoggingOut = false

  // 同时更新 localStorage 和 authStore
  localStorage.setItem('access_token', accessToken)
  localStorage.setItem('refresh_token', refreshToken)

  // 更新 authStore（通过直接修改 localStorage，zustand persist 会自动同步）
  const authStorage = localStorage.getItem('auth-storage')
  if (authStorage) {
    try {
      const parsed = JSON.parse(authStorage)
      parsed.state.token = accessToken
      parsed.state.refreshToken = refreshToken
      parsed.state.isAuthenticated = true
      const newAuthStorage = JSON.stringify(parsed)
      localStorage.setItem('auth-storage', newAuthStorage)
      // 关键修复：同时更新 authStorageString 缓存，保持一致性
      authStorageString = newAuthStorage
    } catch (e) {
      console.warn('Failed to update auth-storage:', e)
    }
  }

  // 立即更新缓存变量，避免下次读取时使用旧值
  cachedToken = accessToken
  cachedRefreshToken = refreshToken
}

// 清除 Token
export const clearToken = (): void => {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  localStorage.removeItem('auth-storage')
  // 清除缓存变量，确保下次读取时不会使用旧值
  cachedToken = null
  cachedRefreshToken = null
  authStorageString = null
}

// 刷新 Token (async-defer-await: 将 await 移到实际使用的分支)
async function refreshAccessToken(refreshToken: string): Promise<string> {
  const response = fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ refreshToken }),
  })

  // 先检查响应状态，再等待 JSON 解析
  const res = await response
  if (!res.ok) {
    throw new Error('Failed to refresh token')
  }

  const data: ApiResponse<{ access_token: string; refresh_token: string }> = await res.json()
  if (data.data) {
    saveToken(data.data.access_token, data.data.refresh_token)
    return data.data.access_token
  }

  throw new Error('Invalid refresh response')
}

// 处理 401 未授权 - 清除认证状态并跳转登录页
// 使用 isLoggingOut 标志防止并发请求重复登出
function handleUnauthorized() {
  // 防止并发请求重复触发登出
  if (isLoggingOut) {
    return
  }

  isLoggingOut = true
  clearToken()

  // 使用 window.location.href 确保在所有情况下都能跳转
  // 保存当前路径以便登录后返回
  const currentPath = window.location.pathname
  if (currentPath !== '/auth/login') {
    sessionStorage.setItem('redirectAfterLogin', currentPath)
  }

  window.location.href = '/auth/login'
}

// 通用请求函数
export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const url = `${API_BASE_URL}${endpoint}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }

  let token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  try {
    const response = await fetch(url, {
      ...options,
      headers,
    })

    // 处理 401 未授权
    if (response.status === 401) {
      const refreshToken = getRefreshToken()

      // 没有刷新 token，直接跳转登录
      if (!refreshToken) {
        handleUnauthorized()
        throw new Error('Unauthorized - no refresh token')
      }

      // 检查是否在冷却期内（上次刷新失败后 5 秒内）
      const now = Date.now()
      if (lastRefreshFailureTime > 0 && now - lastRefreshFailureTime < REFRESH_COOLDOWN) {
        handleUnauthorized()
        throw new Error('Token refresh failed recently, please login again')
      }

      // 关键修复：使用 Promise 缓存防止并发刷新
      // 如果正在刷新，直接返回现有的 Promise，而不是创建新的刷新请求
      if (refreshingPromise) {
        try {
          const newToken = await refreshingPromise
          // 使用新 token 重试原始请求
          const newHeaders = { ...headers, Authorization: `Bearer ${newToken}` }
          const retryResponse = await fetch(url, { ...options, headers: newHeaders })
          if (retryResponse.status === 401) {
            handleUnauthorized()
            throw new Error('Session expired')
          }
          const data = await retryResponse.json()
          if (!retryResponse.ok) {
            throw new Error(data.message || 'Request failed')
          }
          return data
        } catch (error) {
          handleUnauthorized()
          throw error
        }
      }

      // 开始刷新 token - 创建 Promise 并缓存
      refreshingPromise = refreshAccessToken(refreshToken)

      try {
        const newToken = await refreshingPromise

        // 刷新成功，清除失败时间戳
        lastRefreshFailureTime = 0

        // 使用新 token 重试原始请求
        const newHeaders = { ...headers, Authorization: `Bearer ${newToken}` }
        const retryResponse = await fetch(url, { ...options, headers: newHeaders })

        if (retryResponse.status === 401) {
          // 刷新后仍然 401，token 已彻底失效
          handleUnauthorized()
          throw new Error('Session expired')
        }

        const data = await retryResponse.json()
        if (!retryResponse.ok) {
          throw new Error(data.message || 'Request failed')
        }

        return data
      } catch (error) {
        // 刷新失败，记录失败时间并跳转登录
        lastRefreshFailureTime = Date.now()
        handleUnauthorized()
        throw error
      } finally {
        // 清除 Promise 缓存，允许下次刷新（但在冷却期内）
        refreshingPromise = null
      }
    }

    const data: ApiResponse<T> = await response.json()

    if (!response.ok) {
      throw new Error(data.message || 'Request failed')
    }

    return data
  } catch (error) {
    // Handle HTTP errors with centralized error handling per D-39
    if (error && typeof error === 'object' && 'response' in error) {
      const err = error as { response?: { status?: number; data?: { message?: string } } }

      if (err.response) {
        const { status, data } = err.response

        // Map error codes to user-friendly messages per D-38
        const errorMessages: Record<number, string> = {
          400: '请求参数错误',
          401: '登录已过期，请重新登录',
          403: '权限不足，无法访问此资源',
          404: '请求的资源不存在',
          500: '服务器错误，请稍后重试',
          502: '网关错误，请稍后重试',
          503: '服务不可用，请稍后重试',
        }

        const errorMessage = data?.message || errorMessages[status || 500] || '请求失败，请稍后重试'
        message.error(errorMessage, 5) // 5 seconds duration per D-38

        // Redirect to login on 401
        if (status === 401) {
          handleUnauthorized()
        }
      }
    } else if (error instanceof Error && error.message === 'Network error') {
      // Network error (no response received)
      message.error('网络连接失败，请检查网络设置', 5)
    }

    if (error instanceof Error) {
      throw error
    }
    throw new Error('Network error')
  }
}

// 导出 clearToken 供其他模块使用
export { clearToken as apiClearToken }

// 导出 saveToken 供 auth.ts 使用
export { saveToken }
