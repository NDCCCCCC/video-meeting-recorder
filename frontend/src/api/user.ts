// 用户管理 API 客户端

import type {
  UserListParams,
  CreateUserRequest,
  UpdateUserRequest,
  ResetPasswordRequest,
  UpdateProfileRequest,
  UserListApiResponse,
  UserApiResponse,
} from '../types/user'
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

// 获取用户列表
export async function getUserList(params: UserListParams): Promise<UserListApiResponse> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.role_id) queryParams.append('role_id', params.role_id.toString())
  if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString())

  const query = queryParams.toString()
  return request<UserListApiResponse>(`/api/v1/users${query ? `?${query}` : ''}`)
}

// 获取用户详情
export async function getUser(id: number): Promise<UserApiResponse> {
  return request<UserApiResponse>(`/api/v1/users/${id}`)
}

// 创建用户
export async function createUser(req: CreateUserRequest): Promise<UserApiResponse> {
  return request<UserApiResponse>('/api/v1/users', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新用户
export async function updateUser(id: number, req: UpdateUserRequest): Promise<UserApiResponse> {
  return request<UserApiResponse>(`/api/v1/users/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除用户
export async function deleteUser(id: number): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/users/${id}`, {
    method: 'DELETE',
  })
}

// 重置用户密码
export async function resetUserPassword(id: number, req: ResetPasswordRequest): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/users/${id}/reset-password`, {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 切换用户状态
export async function toggleUserStatus(id: number): Promise<UserApiResponse> {
  return request<UserApiResponse>(`/api/v1/users/${id}/toggle-status`, {
    method: 'POST',
  })
}

// 获取当前用户资料
export async function getCurrentProfile(): Promise<UserApiResponse> {
  return request<UserApiResponse>('/api/v1/users/profile')
}

// 更新当前用户资料
export async function updateCurrentProfile(req: UpdateProfileRequest): Promise<UserApiResponse> {
  return request<UserApiResponse>('/api/v1/users/profile', {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}
