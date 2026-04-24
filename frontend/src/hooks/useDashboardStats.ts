// Dashboard stats custom hook

import { useState, useEffect } from 'react'
import { message } from 'antd'
import * as dashboardApi from '../api/dashboard'
import type { DashboardStatsResponse } from '../types/dashboard'

interface UseDashboardStatsResult {
  stats: DashboardStatsResponse | null
  loading: boolean
  error: Error | null
  refresh: () => Promise<void>
}

export function useDashboardStats(): UseDashboardStatsResult {
  const [stats, setStats] = useState<DashboardStatsResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const fetchStats = async () => {
    setLoading(true)
    setError(null)
    try {
      const response = await dashboardApi.getDashboardStats()
      if (response.data) {
        setStats(response.data)
      }
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to load dashboard stats')
      setError(error)
      message.error(error.message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStats()
  }, [])

  return { stats, loading, error, refresh: fetchStats }
}
