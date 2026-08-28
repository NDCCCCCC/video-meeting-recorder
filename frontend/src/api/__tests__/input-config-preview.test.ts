/**
 * getInputConfigPreview 回归测试（quick 260828-krh）
 *
 * 覆盖 3 类行为：
 *  1. API 成功路径 → 返回 Blob
 *  2. API 错误路径 → reject 且 error.message 含 JSON body message
 *  3. 多 id 路由 → 两次请求 URL 分别以 /1/preview 与 /2/preview 结尾
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// 屏蔽 antd UI 副作用
vi.mock('antd', () => ({
  message: { error: vi.fn(), loading: vi.fn(), success: vi.fn() },
}))

type FetchCall = { url: string; init?: RequestInit }
let calls: FetchCall[] = []
let mockBlob: Blob
let fetchSpy: ReturnType<typeof vi.fn>

function reset() {
  calls = []
  mockBlob = new Blob(['fake-jpeg'], { type: 'image/jpeg' })
  fetchSpy = vi.fn()
}

function okResponse(url: string, _init?: RequestInit): Promise<Response> {
  calls.push({ url, init: _init })
  const id = parseInt(url.match(/\/api\/v1\/input-configs\/(\d+)\/preview/)?.[1] ?? '0')
  if (id === 0) {
    return Promise.resolve({
      ok: false, status: 500,
      json: async () => ({ message: '预览抓帧失败' }),
    } as unknown as Response)
  }
  return Promise.resolve({
    ok: true, blob: async () => mockBlob,
  } as unknown as Response)
}

function errResponse(url: string, _init?: RequestInit): Promise<Response> {
  calls.push({ url, init: _init })
  return Promise.resolve({
    ok: false, status: 500,
    json: async () => ({ message: '预览抓帧失败' }),
  } as unknown as Response)
}

vi.mock('../apiClient', () => ({
  authedFetch: (url: string, init?: RequestInit) => fetchSpy(url, init),
}))

import { getInputConfigPreview } from '../input-config'

describe('getInputConfigPreview', () => {
  beforeEach(() => { reset() })
  afterEach(() => { vi.restoreAllMocks() })

  it('成功路径：返回 Blob', async () => {
    fetchSpy.mockImplementationOnce(okResponse)
    const blob = await getInputConfigPreview(7)
    expect(blob).toBeInstanceOf(Blob)
    expect(calls[0].url).toContain('/api/v1/input-configs/7/preview')
  })

  it('错误路径：reject 且 message 含 JSON body message', async () => {
    fetchSpy.mockImplementationOnce(errResponse)
    await expect(getInputConfigPreview(1)).rejects.toThrow('预览抓帧失败')
  })

  it('多 id 路由：两次请求 URL 分别以 /1/preview 与 /2/preview 结尾', async () => {
    fetchSpy.mockImplementation(okResponse)
    await getInputConfigPreview(1)
    await getInputConfigPreview(2)
    expect(calls[0].url).toMatch(/\/1\/preview$/)
    expect(calls[1].url).toMatch(/\/2\/preview$/)
  })
})
