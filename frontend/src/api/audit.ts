// 审计日志 API 客户端

import type {
	AuditLog,
	AuditLogListParams,
	AuditLogListApiResponse,
	AuditLogExportParams,
} from '../types/audit'
import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'

// 获取审计日志列表
export async function getAuditLogs(params: AuditLogListParams): Promise<AuditLogListApiResponse> {
	const queryParams = new URLSearchParams()
	if (params.page) queryParams.append('page', params.page.toString())
	if (params.page_size) queryParams.append('page_size', params.page_size.toString())
	if (params.username) queryParams.append('username', params.username)
	if (params.action) queryParams.append('action', params.action)
	if (params.module) queryParams.append('module', params.module)
	if (params.start_time) queryParams.append('start_time', params.start_time)
	if (params.end_time) queryParams.append('end_time', params.end_time)

	const query = queryParams.toString()
	return apiRequest(`/api/v1/audit/logs${query ? `?${query}` : ''}`)
}

// 获取审计日志详情
export async function getAuditLogById(id: number): Promise<AuditLogApiResponse> {
	return apiRequest(`/api/v1/audit/logs/${id}`)
}

// 导出审计日志
export async function exportAuditLogs(params: AuditLogExportParams): Promise<Blob> {
	const queryParams = new URLSearchParams()
	if (params.username) queryParams.append('username', params.username)
	if (params.action) queryParams.append('action', params.action)
	if (params.module) queryParams.append('module', params.module)
	if (params.start_time) queryParams.append('start_time', params.start_time)
	if (params.end_time) queryParams.append('end_time', params.end_time)
	queryParams.append('format', params.format || 'csv')

	const query = queryParams.toString()
	const response = await fetch(`/api/v1/audit/logs/export${query ? `?${query}` : ''}`, {
		headers: {
			Authorization: `Bearer ${localStorage.getItem('access_token')}`,
		},
	})

	if (!response.ok) {
		throw new Error(`导出失败: ${response.statusText}`)
	}

	return response.blob()
}

// 获取审计日志统计
export async function getAuditStatistics(days?: number): Promise<ApiResponse<any>> {
	const queryParams = new URLSearchParams()
	if (days) queryParams.append('days', days.toString())

	const query = queryParams.toString()
	return apiRequest(`/api/v1/audit/statistics${query ? `?${query}` : ''}`)
}
