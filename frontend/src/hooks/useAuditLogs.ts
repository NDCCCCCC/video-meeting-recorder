import { useState, useEffect } from 'react'
import { message } from 'antd'
import * as auditApi from '../api/audit'
import type { AuditLog, AuditLogListParams } from '../types/audit'

interface UseAuditLogsResult {
  logs: AuditLog[]
  total: number
  loading: boolean
  error: Error | null
  fetchLogs: (params: AuditLogListParams) => Promise<void>
}

export function useAuditLogs(initialParams: AuditLogListParams = {}): UseAuditLogsResult {
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const fetchLogs = async (params: AuditLogListParams) => {
    setLoading(true)
    setError(null)
    try {
      const response = await auditApi.getAuditLogs(params)
      if (response.data) {
        setLogs(response.data.items)
        setTotal(response.data.total)
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to load audit logs')
      setError(error)
      message.error(error.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs(initialParams)
  }, [])

  return { logs, total, loading, error, fetchLogs }
}
