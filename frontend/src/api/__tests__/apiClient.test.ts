/**
 * apiClient token 状态机回归测试（quick 260828-j2a）。
 *
 * 根因背景：并发 401 时 refresh 单飞窗口过窄 + retry 仍 401 无二次恢复 +
 * 冷却期误登出 + 无主动刷新，导致"一批 401 之后持续 401，刷新页面才恢复"。
 *
 * 覆盖 5 类行为：
 *  1. 并发 401 → 单飞 refresh（refresh 端点恰好被调用 1 次），全部请求用新 token 重放成功
 *  2. refresh 完成后迟到的 401 → 命中 recentRefresh 缓存重放，不再发第二次 refresh
 *  2b. 超过缓存窗口（REFRESH_GRACE_MS 30s）后的迟到 401 → 允许发起新 refresh
 *  3. 缓存 token 重放仍 401 → 升级为新 refresh 再重放（而不是把"未授权"抛给页面）
 *  3b. 新 refresh 的重放仍 401 → 抛错但不触发登出
 *  4. refresh 连续失败 → 单飞一次 + 整个登出流程只执行一次 + 冷却期抛可重试错误
 *  5. access token 临期（剩余寿命 < 60s）→ 请求前主动单飞刷新
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// 屏蔽 antd UI 副作用（apiClient 在集中错误处理路径调 message.error）
vi.mock('antd', () => ({
  message: { error: vi.fn(), loading: vi.fn(), success: vi.fn() },
}))

const REFRESH_ENDPOINT = '/api/v1/auth/refresh'

interface FetchCall {
  url: string
  init?: RequestInit
}

type HandlerResult = Response | Promise<Response>
type Handler = (url: string, init?: RequestInit) => HandlerResult

let calls: FetchCall[] = []
let handler: Handler = () => okData()
let refreshCalls = 0
let redirectCount = 0
let currentHref = ''
let locationStubbed = false

// ---------------------------------------------------------------------------
// fetch mock 基建
// ---------------------------------------------------------------------------

function authOf(init?: RequestInit): string {
  const headers = init?.headers as Record<string, string> | undefined
  return headers?.Authorization ?? ''
}

function jsonRes(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  } as unknown as Response
}

function okData(data: unknown = { ok: true }): Response {
  return jsonRes({ code: 0, message: 'ok', data })
}

function unauthorized(): Response {
  return jsonRes({ code: 401, message: '未授权：token 无效' }, 401)
}

function refreshResponse(accessToken: string, refreshToken: string, expiresIn = 7200): Response {
  return jsonRes({
    code: 0,
    message: 'ok',
    data: { access_token: accessToken, refresh_token: refreshToken, expires_in: expiresIn },
  })
}

interface Deferred<T> {
  promise: Promise<T>
  resolve: (value: T) => void
}

function deferred<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

function isRefresh(url: string): boolean {
  return url.includes(REFRESH_ENDPOINT)
}

/** 直接写 localStorage（绕过 apiClient 模块状态，用于构造/重置认证态） */
function seedTokens(accessToken: string, refreshToken: string): void {
  localStorage.setItem('access_token', accessToken)
  localStorage.setItem('refresh_token', refreshToken)
  localStorage.setItem(
    'auth-storage',
    JSON.stringify({
      state: { token: accessToken, refreshToken, isAuthenticated: true },
      version: 0,
    })
  )
}

/** 每个 case 重新加载模块，重置模块级单飞/缓存/登出状态 */
const loadClient = () => import('../apiClient')

beforeEach(() => {
  vi.resetModules()
  vi.useFakeTimers()
  localStorage.clear()
  sessionStorage.clear()
  calls = []
  refreshCalls = 0
  redirectCount = 0
  currentHref = ''

  // 用可计数 href setter 的假 location 替换 happy-dom 的真实 location，
  // 避免 handleUnauthorized 触发真实导航
  const stub: Record<string, unknown> = { pathname: '/files', assign: vi.fn(), replace: vi.fn() }
  Object.defineProperty(stub, 'href', {
    configurable: true,
    get: () => currentHref,
    set: (value: string) => {
      redirectCount++
      currentHref = value
    },
  })
  try {
    Object.defineProperty(globalThis, 'location', {
      value: stub,
      configurable: true,
      writable: true,
    })
    locationStubbed = true
  } catch {
    locationStubbed = false
  }

  handler = () => okData()
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit): Promise<Response> => {
      const url = typeof input === 'string' ? input : String(input)
      calls.push({ url, init })
      return handler(url, init)
    })
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.useRealTimers()
})

describe('apiClient token 状态机（quick 260828-j2a）', () => {
  it('Test 1: 8 个并发 401 只触发一次 refresh，全部请求用新 token 重放成功', async () => {
    expect.assertions(11)
    seedTokens('AT1', 'RT1')

    handler = (url, init) => {
      if (isRefresh(url)) {
        refreshCalls++
        return refreshResponse('AT2', 'RT2')
      }
      return authOf(init) === 'Bearer AT2' ? okData() : unauthorized()
    }

    const client = await loadClient()
    const endpoints = [
      '/api/v1/files/stats',
      '/api/v1/files',
      '/api/v1/input-configs/active',
      '/api/v1/transcriptions/active',
      '/api/v1/recordings',
      '/api/v1/tasks',
      '/api/v1/system/config',
      '/api/v1/auth/me',
    ]

    const results = await Promise.all(endpoints.map((endpoint) => client.apiRequest(endpoint)))
    expect(results).toHaveLength(8)
    for (const result of results) {
      expect(result.code).toBe(0)
    }
    // 单飞：8 个并发 401 只刷新一次
    expect(refreshCalls).toBe(1)
    // 每个请求都用新 token 重放过一次
    const replays = calls.filter(
      (c) => !isRefresh(c.url) && authOf(c.init) === 'Bearer AT2'
    )
    expect(replays).toHaveLength(8)
  })

  it('Test 2: refresh 完成后迟到的 401 命中缓存重放，不再发第二次 refresh', async () => {
    seedTokens('AT1', 'RT1')

    // /files/stats 的响应在 refresh 完成之后才到达（迟到的 401）
    const lateStats = deferred<Response>()
    let lateServed = false
    handler = (url, init) => {
      if (isRefresh(url)) {
        refreshCalls++
        return refreshResponse('AT2', 'RT2')
      }
      if (url.endsWith('/files/stats') && !lateServed) {
        lateServed = true
        return lateStats.promise
      }
      return authOf(init) === 'Bearer AT1' ? unauthorized() : okData()
    }

    const client = await loadClient()
    const lateReq = client.apiRequest('/api/v1/files/stats')

    // 另一个请求触发正常的 401 → refresh → 重放
    const first = await client.apiRequest('/api/v1/files/scan')
    expect(first.code).toBe(0)
    expect(refreshCalls).toBe(1)

    // refresh 已完成，之后迟到的 401 才被处理
    vi.advanceTimersByTime(100)
    lateStats.resolve(unauthorized())

    const late = await lateReq
    expect(late.code).toBe(0)
    // 关键断言：不再发第二次 refresh
    expect(refreshCalls).toBe(1)
    // 该请求用缓存的新 token 重放成功
    const replay = calls.filter(
      (c) => c.url.endsWith('/files/stats') && authOf(c.init) === 'Bearer AT2'
    )
    expect(replay).toHaveLength(1)
  })

  it('Test 2b: 超过缓存窗口（30s）后的迟到 401 触发新的 refresh', async () => {
    seedTokens('AT1', 'RT1')

    const lateStats = deferred<Response>()
    let lateServed = false
    handler = (url, init) => {
      if (isRefresh(url)) {
        refreshCalls++
        return refreshResponse('AT2', 'RT2')
      }
      if (url.endsWith('/files/stats') && !lateServed) {
        lateServed = true
        return lateStats.promise
      }
      return authOf(init) === 'Bearer AT1' ? unauthorized() : okData()
    }

    const client = await loadClient()
    const lateReq = client.apiRequest('/api/v1/files/stats')
    const first = await client.apiRequest('/api/v1/files/scan')
    expect(first.code).toBe(0)
    expect(refreshCalls).toBe(1)

    // 超过前端缓存窗口（REFRESH_GRACE_MS = 30s）
    vi.advanceTimersByTime(31_000)
    lateStats.resolve(unauthorized())

    const late = await lateReq
    expect(late.code).toBe(0)
    // 缓存过期 → 重新刷新
    expect(refreshCalls).toBe(2)
  })

  it('Test 3: 缓存 token 重放仍 401 → 升级为新 refresh 再重放成功', async () => {
    seedTokens('AT1', 'RT1')

    // Phase 1：正常刷新成功，缓存 AT2
    handler = (url, init) => {
      if (isRefresh(url)) {
        refreshCalls++
        return refreshResponse('AT2', 'RT2')
      }
      return authOf(init) === 'Bearer AT1' ? unauthorized() : okData()
    }

    const client = await loadClient()
    const first = await client.apiRequest('/api/v1/files/scan')
    expect(first.code).toBe(0)
    expect(refreshCalls).toBe(1)

    // Phase 2：服务端开始拒绝 AT2，只接受 AT3
    handler = (url, init) => {
      if (isRefresh(url)) {
        refreshCalls++
        return refreshResponse('AT3', 'RT3')
      }
      return authOf(init) === 'Bearer AT2' ? unauthorized() : okData()
    }

    vi.advanceTimersByTime(100)
    const second = await client.apiRequest('/api/v1/files/stats')
    expect(second.code).toBe(0)
    // 缓存重放失败 → 升级为一次新的 refresh
    expect(refreshCalls).toBe(2)
    // 最终重放使用第二次刷新得到的 token
    const finalReplay = calls.find(
      (c) => c.url.endsWith('/files/stats') && authOf(c.init) === 'Bearer AT3'
    )
    expect(finalReplay).toBeDefined()
  })

  it('Test 3b: 新 refresh 后的重放仍 401 → 抛错给调用方且不触发登出', async () => {
    expect.assertions(4)
    seedTokens('AT1', 'RT1')

    handler = (url, init) => {
      if (isRefresh(url)) {
        refreshCalls++
        return refreshResponse('AT2', 'RT2')
      }
      return unauthorized()
    }

    const client = await loadClient()
    await expect(client.apiRequest('/api/v1/files/stats')).rejects.toThrow('未授权')
    expect(refreshCalls).toBe(1)
    // 未登出：token 保留（refresh 已成功保存新 token，登出会清空它）
    expect(localStorage.getItem('access_token')).not.toBeNull()
    expect(redirectCount).toBe(0)
  })

  it('Test 4: refresh 连续失败 → 单飞一次 + 登出只执行一次 + 冷却期抛可重试错误', async () => {
    expect(locationStubbed).toBe(true)
    seedTokens('AT1', 'RT1')

    handler = (url) => {
      if (isRefresh(url)) {
        refreshCalls++
        return jsonRes({ code: 401, message: '无效的刷新令牌' }, 401)
      }
      return unauthorized()
    }

    const client = await loadClient()
    const endpoints = [
      '/api/v1/files/stats',
      '/api/v1/files',
      '/api/v1/input-configs/active',
      '/api/v1/transcriptions/active',
      '/api/v1/recordings',
      '/api/v1/tasks',
      '/api/v1/system/config',
      '/api/v1/auth/me',
    ]

    const settled = await Promise.allSettled(endpoints.map((e) => client.apiRequest(e)))
    for (const result of settled) {
      expect(result.status).toBe('rejected')
    }
    // 单飞：只刷新一次
    expect(refreshCalls).toBe(1)
    // 登出流程只执行一次：token 清空 + 一次跳转
    expect(localStorage.getItem('access_token')).toBeNull()
    expect(redirectCount).toBe(1)
    expect(currentHref).toBe('/auth/login')

    // 冷却期内再次 401：抛可重试错误，不刷新、不再次登出
    seedTokens('AT9', 'RT9')
    const redirectsBefore = redirectCount
    let caught: (Error & { retryable?: boolean }) | null = null
    try {
      await client.apiRequest('/api/v1/files/stats')
    } catch (error) {
      caught = error as Error & { retryable?: boolean }
    }
    expect(caught).not.toBeNull()
    expect(caught?.retryable).toBe(true)
    expect(refreshCalls).toBe(1)
    expect(redirectCount).toBe(redirectsBefore)
  })

  it('Test 5: token 临期（剩余寿命 < 60s）时请求前主动单飞刷新', async () => {
    seedTokens('AT1', 'RT1')

    // 服务端接受任何 token：区分"主动刷新"与"401 后重放"靠 /files/stats
    // 的调用次数（主动刷新 = 1 次请求；被动路径 = 401 + 重放共 2 次）
    handler = (url) => {
      if (isRefresh(url)) {
        refreshCalls++
        return refreshResponse('AT2', 'RT2', 120)
      }
      return okData()
    }

    const client = await loadClient()
    client.saveToken('AT1', 'RT1', 120) // 记录 expires_at = now + 120s

    // 寿命充足：不主动刷新
    const early = await client.apiRequest('/api/v1/files/scan')
    expect(early.code).toBe(0)
    expect(refreshCalls).toBe(0)

    // 推进至剩余寿命 59s（< PROACTIVE_MARGIN 60s）
    vi.advanceTimersByTime(61_000)
    const late = await client.apiRequest('/api/v1/files/stats')
    expect(late.code).toBe(0)
    expect(refreshCalls).toBe(1)
    // 请求本身携带刷新后的 token（主动刷新，而非 401 后重放）
    const proactive = calls.find(
      (c) => c.url.endsWith('/files/stats') && authOf(c.init) === 'Bearer AT2'
    )
    expect(proactive).toBeDefined()
    // 该请求没有先经历 401
    const statsCalls = calls.filter((c) => c.url.endsWith('/files/stats'))
    expect(statsCalls).toHaveLength(1)
    expect(authOf(statsCalls[0]?.init)).toBe('Bearer AT2')
  })
})
