import { useState, useCallback } from 'react'
import { message } from 'antd'

interface UseLoadingStateResult {
  loading: boolean
  error: Error | null
  execute: <T>(asyncFn: () => Promise<T>) => Promise<T | null>
  reset: () => void
}

export function useLoadingState(): UseLoadingStateResult {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const execute = useCallback(async <T>(asyncFn: () => Promise<T>): Promise<T | null> => {
    setLoading(true)
    setError(null)
    try {
      const result = await asyncFn()
      return result
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Unknown error')
      setError(error)
      message.error(error.message)
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const reset = useCallback(() => {
    setLoading(false)
    setError(null)
  }, [])

  return { loading, error, execute, reset }
}
