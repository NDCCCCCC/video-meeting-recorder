// Dashboard API client

import type { DashboardStatsResponse } from '../types/dashboard'
import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'

export async function getDashboardStats(): Promise<ApiResponse<DashboardStatsResponse>> {
  return apiRequest('/api/v1/dashboard/stats')
}
