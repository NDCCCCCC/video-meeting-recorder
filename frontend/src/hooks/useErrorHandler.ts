import { useCallback } from 'react'
import { message } from 'antd'

interface ErrorHandlerOptions {
  showMessage?: boolean
  duration?: number
}

export function useErrorHandler() {
  const handleError = useCallback((error: unknown, options: ErrorHandlerOptions = {}) => {
    const { showMessage = true, duration = 5 } = options

    let errorMessage = '操作失败'

    if (error instanceof Error) {
      errorMessage = error.message
    } else if (typeof error === 'string') {
      errorMessage = error
    } else if (error && typeof error === 'object' && 'response' in error) {
      // Axios-like error response
      const err = error as { response?: { data?: { message?: string } } }
      errorMessage = err.response?.data?.message || errorMessage
    }

    if (showMessage) {
      message.error(errorMessage, duration)
    }

    return errorMessage
  }, [])

  return { handleError }
}
