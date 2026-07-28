import { useCallback } from 'react'
import { message } from 'antd'

interface ErrorHandlerOptions {
  showMessage?: boolean
  duration?: number
}

// D-05.4 — 最后兜底文案：诚实、简短，不用没有信息量的机器腔
const FALLBACK_MESSAGE = '操作未完成，请稍后重试'

// D-05.2 — 服务端没给出具体原因时，至少按状态码说清楚"是什么问题"
const STATUS_MESSAGES: Record<number, string> = {
  400: '请求参数有误',
  401: '登录已过期，请重新登录',
  403: '没有访问权限',
  404: '请求的资源不存在',
  408: '请求超时，请重试',
  409: '资源冲突，请刷新后重试',
  413: '文件超过大小限制',
  429: '请求过于频繁，请稍后重试',
  500: '服务端出错了',
  502: '服务端网关异常',
  503: '服务暂时不可用',
  504: '服务端响应超时',
}

interface AxiosLikeError {
  response?: {
    status?: number
    data?: { message?: string; error?: string; detail?: string }
  }
  code?: string
  message?: string
}

// 从 axios 风格的错误里挖出后端返回的真实原因
function extractResponseMessage(error: unknown): string | undefined {
  if (!error || typeof error !== 'object' || !('response' in error)) return undefined
  const { response } = error as AxiosLikeError
  const data = response?.data
  const serverMessage = data?.message || data?.error || data?.detail
  if (serverMessage) return serverMessage
  if (response?.status && STATUS_MESSAGES[response.status]) {
    return STATUS_MESSAGES[response.status]
  }
  return undefined
}

export function useErrorHandler() {
  const handleError = useCallback((error: unknown, options: ErrorHandlerOptions = {}) => {
    const { showMessage = true, duration = 5 } = options

    let errorMessage = FALLBACK_MESSAGE

    // D-05.2 — 优先级：后端返回的具体原因 > 状态码语义 > Error.message > 字符串错误
    const responseMessage = extractResponseMessage(error)

    if (responseMessage) {
      errorMessage = responseMessage
    } else if (typeof error === 'string' && error.trim()) {
      errorMessage = error
    } else if (error instanceof Error && error.message) {
      errorMessage = error.message
    }

    // 仍是兜底文案时，把原始 Error.message 带回来，别把真实原因吞掉
    if (errorMessage === FALLBACK_MESSAGE && error instanceof Error && error.message) {
      errorMessage = error.message
    }

    // 网络层错误没有 response，浏览器给的 message 很难懂，换成人话
    if (error instanceof Error && (error.name === 'TypeError' || error.message === 'Network Error')) {
      errorMessage = '网络连接中断'
    }

    if (showMessage) {
      message.error(errorMessage, duration)
    }

    return errorMessage
  }, [])

  return { handleError }
}
