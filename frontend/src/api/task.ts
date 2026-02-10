// 录制任务管理 API 客户端

import type {
  TaskListParams,
  CreateTaskRequest,
  UpdateTaskRequest,
  TaskListApiResponse,
  TaskApiResponse,
} from '../types/task'
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

// 获取任务列表
export async function getTaskList(params: TaskListParams): Promise<TaskListApiResponse> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.status) queryParams.append('status', params.status)
  if (params.created_by) queryParams.append('created_by', params.created_by.toString())
  if (params.start_date) queryParams.append('start_date', params.start_date)
  if (params.end_date) queryParams.append('end_date', params.end_date)

  const query = queryParams.toString()
  return request<TaskListApiResponse>(`/api/v1/recordings${query ? `?${query}` : ''}`)
}

// 获取任务详情
export async function getTask(id: number): Promise<TaskApiResponse> {
  return request<TaskApiResponse>(`/api/v1/recordings/${id}`)
}

// 创建任务
export async function createTask(req: CreateTaskRequest): Promise<TaskApiResponse> {
  return request<TaskApiResponse>('/api/v1/recordings', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新任务
export async function updateTask(id: number, req: UpdateTaskRequest): Promise<TaskApiResponse> {
  return request<TaskApiResponse>(`/api/v1/recordings/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除任务
export async function deleteTask(id: number): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/recordings/${id}`, {
    method: 'DELETE',
  })
}

// 启动任务
export async function startTask(id: number): Promise<TaskApiResponse> {
  return request<TaskApiResponse>(`/api/v1/recordings/${id}/start`, {
    method: 'POST',
  })
}

// 停止任务
export async function stopTask(id: number): Promise<TaskApiResponse> {
  return request<TaskApiResponse>(`/api/v1/recordings/${id}/stop`, {
    method: 'POST',
  })
}

// 取消任务
export async function cancelTask(id: number): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/recordings/${id}/cancel`, {
    method: 'POST',
  })
}

// 重试任务
export async function retryTask(id: number): Promise<TaskApiResponse> {
  return request<TaskApiResponse>(`/api/v1/recordings/${id}/retry`, {
    method: 'POST',
  })
}

// 获取HLS预览信息
export async function getTaskPreview(id: number): Promise<{ data: { task_id: number; playback_url: string; status: string } }> {
  return request(`/api/v1/recordings/${id}/preview`)
}

// 获取HLS流文件URL
export function getHLSStreamUrl(id: number, file: string): string {
  return `${API_BASE_URL}/api/v1/recordings/${id}/preview/stream/${file}`
}
