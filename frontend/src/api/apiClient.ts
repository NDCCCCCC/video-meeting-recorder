// 统一的 API 客户端
// 处理所有 API 请求的认证、401 错误和 token 刷新
//
// token 状态机（quick 260828-j2a）：
//   1. 单飞 refresh：并发 401 共享同一次 refresh（refreshingPromise）
//   2. 缓存重放：refresh 完成后迟到的 401 直接用最近一次刷新结果重放，
//      不再触发第二次 refresh（REFRESH_GRACE_MS 须 >= 后端 GracePeriod 30s，
//      见 internal/auth/sm4_token.go），否则后端会把良性并发刷新判为重放攻击
//      并撤销该用户全部会话
//   3. 二次恢复：缓存 token 重放仍 401 → 升级为一次真实 refresh 再重放；
//      已用新 refresh 的 token 仍 401 → 抛错给调用方（不强制登出）
//   4. 登出收敛：整个登出流程只执行一次（hasHandledUnauthorized）
//   5. 主动刷新：基于后端 expires_in，在 access token 临期前主动单飞刷新，
//      被动 401 批次从根源减少

import type { ApiResponse } from '../types/auth'
import { message } from 'antd'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

// 正在刷新 token 的 Promise（单飞）：并发 401 / 并发主动刷新共享同一次刷新
let refreshingPromise: Promise<string> | null = null
// 最近一次成功刷新结果（token + 时间戳）：供"迟到 401"直接重放
let recentRefresh: { token: string; at: number } | null = null
// 上次刷新失败的时间戳，用于防止频繁重试
let lastRefreshFailureTime = 0
// access token 预计过期时间戳（saveToken 传入 expires_in 时启用主动刷新）
let tokenExpiresAt: number | null = null
// 登出流程只执行一次的标志
let hasHandledUnauthorized = false

// 刷新失败后的冷却时间（毫秒）
const REFRESH_COOLDOWN = 5000
// 迟到 401 的缓存重放窗口（毫秒）。必须 >= 后端 GracePeriod（internal/auth/sm4_token.go）
const REFRESH_GRACE_MS = 30000
// access token 剩余寿命低于该值（毫秒）时，请求前主动刷新
const PROACTIVE_MARGIN_MS = 60000

// refresh 端点明确失败（非 2xx / 响应无效）：视为会话不可恢复，触发登出
class TokenRefreshError extends Error {
  constructor(msg = 'Failed to refresh token') {
    super(msg)
    this.name = 'TokenRefreshError'
  }
}

// 冷却期内的 401：不强制登出，抛出可重试错误由调用方决定重试时机
class TokenRefreshCooldownError extends Error {
  readonly retryable = true

  constructor() {
    super('Token refresh failed recently, please retry later')
    this.name = 'TokenRefreshCooldownError'
  }
}

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

// 保存 Token（用于登录 / 刷新后更新）
// expiresInSec：后端返回的 access token 有效期（秒），传入后启用临期主动刷新
const saveToken = (accessToken: string, refreshToken: string, expiresInSec?: number): void => {
  // 新的有效 token = 会话恢复：重置登出 / 冷却状态
  hasHandledUnauthorized = false
  lastRefreshFailureTime = 0
  tokenExpiresAt =
    typeof expiresInSec === 'number' && expiresInSec > 0
      ? Date.now() + expiresInSec * 1000
      : null

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
  tokenExpiresAt = null
  recentRefresh = null
}

// 刷新 Token (async-defer-await: 将 await 移到实际使用的分支)
async function refreshAccessToken(refreshToken: string): Promise<string> {
  const response = await fetch(`${API_BASE_URL}/api/v1/auth/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ refreshToken }),
  })

  // 先检查响应状态，再等待 JSON 解析
  if (!response.ok) {
    throw new TokenRefreshError()
  }

  const data: ApiResponse<{
    access_token: string
    refresh_token: string
    expires_in?: number
  }> = await response.json()

  if (!data.data) {
    throw new TokenRefreshError('Invalid refresh response')
  }

  saveToken(data.data.access_token, data.data.refresh_token, data.data.expires_in)
  // 记录最近一次刷新结果，供迟到的 401 直接重放（不发第二次 refresh）
  recentRefresh = { token: data.data.access_token, at: Date.now() }
  return data.data.access_token
}

// 发起（或复用）一次单飞刷新
function startRefresh(): Promise<string> {
  if (refreshingPromise) {
    return refreshingPromise
  }

  const refreshToken = getRefreshToken()
  if (!refreshToken) {
    handleUnauthorized()
    throw new Error('Unauthorized - no refresh token')
  }

  // 冷却期内不再发起刷新，避免打爆 refresh 端点
  if (lastRefreshFailureTime > 0 && Date.now() - lastRefreshFailureTime < REFRESH_COOLDOWN) {
    throw new TokenRefreshCooldownError()
  }

  refreshingPromise = refreshAccessToken(refreshToken)
    .catch((error: unknown) => {
      // 刷新失败：记录冷却时间并清掉缓存结果。
      // 仅在 refresh 端点明确返回 401/无效（TokenRefreshError）时登出；
      // 网络抖动等瞬时错误只进入冷却期，不把用户整体登出
      lastRefreshFailureTime = Date.now()
      recentRefresh = null
      if (error instanceof TokenRefreshError) {
        handleUnauthorized()
      }
      throw error
    })
    .finally(() => {
      // 释放单飞窗口（失败后的重试受冷却期约束）
      refreshingPromise = null
    })

  return refreshingPromise
}

// 处理 401 未授权 - 清除认证状态并跳转登录页
// 使用 hasHandledUnauthorized 标志保证整个登出流程只执行一次
function handleUnauthorized() {
  if (hasHandledUnauthorized) {
    return
  }

  hasHandledUnauthorized = true
  clearToken()

  // 使用 window.location.href 确保在所有情况下都能跳转
  // 保存当前路径以便登录后返回
  const currentPath = window.location.pathname
  if (currentPath !== '/auth/login') {
    sessionStorage.setItem('redirectAfterLogin', currentPath)
  }

  window.location.href = '/auth/login'
}

// 用指定 token 发起请求；非 2xx 时抛出带 status 的错误
async function requestWithToken<T>(
  url: string,
  options: RequestInit,
  headers: Record<string, string>,
  token: string
): Promise<ApiResponse<T>> {
  const response = await fetch(url, {
    ...options,
    headers: { ...headers, Authorization: `Bearer ${token}` },
  })

  const data = (await response.json()) as ApiResponse<T>
  if (!response.ok) {
    const error = new Error(data.message || 'Request failed after token refresh') as Error & {
      status?: number
    }
    error.status = response.status
    throw error
  }
  return data
}

// 401 恢复路径：单飞 / 缓存重放 / 新刷新 三级 token 解析 + 重放
async function recoverAndRetry<T>(
  url: string,
  options: RequestInit,
  headers: Record<string, string>
): Promise<ApiResponse<T>> {
  let token: string
  let fromCache = false

  if (refreshingPromise) {
    // (a) 已有 in-flight 单飞刷新：直接等待同一次刷新结果
    token = await refreshingPromise
  } else if (recentRefresh && Date.now() - recentRefresh.at < REFRESH_GRACE_MS) {
    // (b) 最近一次刷新结果仍在窗口内：用缓存 token 重放，不发 refresh
    token = recentRefresh.token
    fromCache = true
  } else {
    // (c) 发起新的单飞刷新
    token = await startRefresh()
  }

  try {
    return await requestWithToken<T>(url, options, headers, token)
  } catch (error) {
    const status = (error as { status?: number })?.status

    // 非 401 的失败（5xx / 网络错误）原样抛出
    if (status !== 401) {
      throw error
    }

    // 缓存 token 重放仍 401 → 升级为一次真实 refresh 再重放一次
    if (fromCache) {
      const freshToken = await startRefresh()
      // 新 refresh 的重放仍 401 → 抛错给调用方（不强制登出）
      return requestWithToken<T>(url, options, headers, freshToken)
    }

    // 已用新 refresh 的 token 仍 401：refresh 服务端已确认新 token 有效，
    // retry 401 通常是瞬时错误 → 抛错给调用方处理，不强制登出
    throw error
  }
}

// 临期主动刷新：剩余寿命不足时先走同一单飞状态机刷新。
// 失败时仍返回旧 token 发请求，由 401 兜底路径处理
async function ensureFreshToken(): Promise<string | null> {
  const token = getToken()
  if (!token) {
    return null
  }

  const refreshToken = getRefreshToken()
  if (!refreshToken) {
    return token
  }

  const needsRefresh =
    tokenExpiresAt !== null && Date.now() >= tokenExpiresAt - PROACTIVE_MARGIN_MS
  if (!needsRefresh) {
    return token
  }

  if (!refreshingPromise) {
    // 缓存结果仍在窗口内：直接复用，避免多余 refresh
    if (recentRefresh && Date.now() - recentRefresh.at < REFRESH_GRACE_MS) {
      return recentRefresh.token
    }
    try {
      return await startRefresh()
    } catch {
      return token
    }
  }

  try {
    return await refreshingPromise
  } catch {
    return token
  }
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

  // 临期主动刷新：把被动 401 批次从根源消掉
  const proactiveToken = await ensureFreshToken()
  if (proactiveToken) {
    headers['Authorization'] = `Bearer ${proactiveToken}`
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

      // 冷却期内（上次刷新失败后 5 秒内）：抛可重试错误，不强制登出
      if (lastRefreshFailureTime > 0 && Date.now() - lastRefreshFailureTime < REFRESH_COOLDOWN) {
        throw new TokenRefreshCooldownError()
      }

      return await recoverAndRetry<T>(url, options, headers)
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
