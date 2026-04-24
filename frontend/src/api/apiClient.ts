// 统一的 API 客户端
// 处理所有 API 请求的认证、401 错误和 token 刷新

import type { ApiResponse } from '../types/auth'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

// 正在刷新 token 的标志，防止并发刷新
let isRefreshing = false
// 等待刷新完成的回调队列
let refreshSubscribers: Array<(token: string) => void> = []

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

// 将回调加入队列
function subscribeTokenRefresh(callback: (token: string) => void) {
  refreshSubscribers.push(callback)
}

// 通知所有订阅者 token 已刷新
function onTokenRefreshed(token: string) {
  refreshSubscribers.forEach((callback) => callback(token))
  refreshSubscribers = []
}

// 获取 Token - 使用缓存
export const getToken = (): string | null => {
  updateTokenCache()
  console.log('[DEBUG] getToken called, cachedToken:', cachedToken ? `${cachedToken.substring(0, 20)}...` : null)
  console.log('[DEBUG] authStorageString:', authStorageString ? `${authStorageString.substring(0, 50)}...` : null)
  return cachedToken
}

// 获取刷新 Token - 使用缓存
const getRefreshToken = (): string | null => {
  updateTokenCache()
  return cachedRefreshToken
}

// 保存 Token（用于刷新后更新）
const saveToken = (accessToken: string, refreshToken: string): void => {
  console.log('[DEBUG] saveToken called, accessToken:', accessToken ? `${accessToken.substring(0, 20)}...` : null)
  // 同时更新 localStorage 和 authStore
  localStorage.setItem('access_token', accessToken)
  localStorage.setItem('refresh_token', refreshToken)

  // 更新 authStore
  const authStorage = localStorage.getItem('auth-storage')
  console.log('[DEBUG] saveToken - authStorage before:', authStorage ? `${authStorage.substring(0, 50)}...` : null)
  if (authStorage) {
    const parsed = JSON.parse(authStorage)
    parsed.state.token = accessToken
    parsed.state.refreshToken = refreshToken
    parsed.state.isAuthenticated = true
    localStorage.setItem('auth-storage', JSON.stringify(parsed))
    // 关键修复：同时更新 authStorageString 缓存，保持一致性
    authStorageString = JSON.stringify(parsed)
    console.log('[DEBUG] saveToken - authStorage after:', localStorage.getItem('auth-storage')?.substring(0, 50) + '...')
  }

  // 立即更新缓存变量，避免下次读取时使用旧值
  cachedToken = accessToken
  cachedRefreshToken = refreshToken
  console.log('[DEBUG] saveToken - cachedToken now:', cachedToken ? `${cachedToken.substring(0, 20)}...` : null)
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

// 刷新 Token
async function refreshAccessToken(refreshToken: string): Promise<string> {
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

  const data: ApiResponse<{ access_token: string; refresh_token: string }> = await response.json()
  if (data.data) {
    saveToken(data.data.access_token, data.data.refresh_token)
    return data.data.access_token
  }

  throw new Error('Invalid refresh response')
}

// 处理 401 未授权 - 清除认证状态并跳转登录页
function handleUnauthorized() {
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

      // 如果正在刷新，等待刷新完成
      if (isRefreshing) {
        return new Promise<ApiResponse<T>>((resolve, reject) => {
          subscribeTokenRefresh((newToken: string) => {
            // 使用新 token 重试请求
            const newHeaders = { ...headers, Authorization: `Bearer ${newToken}` }
            fetch(url, { ...options, headers: newHeaders })
              .then(async (res) => {
                const data = await res.json()
                if (!res.ok) {
                  reject(new Error(data.message || 'Request failed'))
                } else {
                  resolve(data)
                }
              })
              .catch(reject)
          })
        })
      }

      // 开始刷新 token
      isRefreshing = true

      try {
        const newToken = await refreshAccessToken(refreshToken)
        isRefreshing = false
        onTokenRefreshed(newToken)

        // 使用新 token 重试原始请求
        const newHeaders = { ...headers }
        newHeaders['Authorization'] = `Bearer ${newToken}`

        const retryResponse = await fetch(url, {
          ...options,
          headers: newHeaders,
        })

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
        isRefreshing = false
        // 刷新失败，跳转登录
        handleUnauthorized()
        throw error
      }
    }

    const data: ApiResponse<T> = await response.json()

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

// 导出 clearToken 供其他模块使用
export { clearToken as apiClearToken }

// 导出 saveToken 供 auth.ts 使用
export { saveToken }
