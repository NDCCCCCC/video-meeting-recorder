// 会议管理 API 客户端

import type {
  ConferenceRecord,
  ConferenceListParams,
  ConferenceListResponse,
  CreateConferenceRequest,
  UpdateConferenceRequest,
} from '../types/conference'
import type { ApiResponse } from '../types/auth'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// 获取 Token
const getToken = (): string => {
  return localStorage.getItem('access_token') || ''
}

// 通用请求函数
async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string>),
  }

  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  const data = await response.json()

  if (!response.ok) {
    throw new Error(data.message || 'Request failed')
  }

  return data
}

// 获取会议列表
export async function getConferenceList(
  params: ConferenceListParams
): Promise<ApiResponse<ConferenceListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.status) queryParams.append('status', params.status)
  if (params.conference_number) queryParams.append('conference_number', params.conference_number)
  if (params.start_date) queryParams.append('start_date', params.start_date)
  if (params.end_date) queryParams.append('end_date', params.end_date)

  const query = queryParams.toString()
  return request<ApiResponse<ConferenceListResponse>>(
    `/api/v1/conferences${query ? `?${query}` : ''}`
  )
}

// 获取会议详情
export async function getConference(
  id: number
): Promise<ApiResponse<ConferenceRecord>> {
  return request<ApiResponse<ConferenceRecord>>(`/api/v1/conferences/${id}`)
}

// 创建会议
export async function createConference(
  req: CreateConferenceRequest
): Promise<ApiResponse<ConferenceRecord>> {
  return request<ApiResponse<ConferenceRecord>>('/api/v1/conferences', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新会议
export async function updateConference(
  id: number,
  req: UpdateConferenceRequest
): Promise<ApiResponse<ConferenceRecord>> {
  return request<ApiResponse<ConferenceRecord>>(`/api/v1/conferences/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除会议
export async function deleteConference(
  id: number
): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/conferences/${id}`, {
    method: 'DELETE',
  })
}

// 根据状态获取会议列表
export async function getConferencesByStatus(
  status: string
): Promise<ApiResponse<ConferenceRecord[]>> {
  return request<ApiResponse<ConferenceRecord[]>>(
    `/api/v1/conferences/by-status?status=${status}`
  )
}
