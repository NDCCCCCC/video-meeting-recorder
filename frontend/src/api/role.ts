// 角色管理 API 客户端

import type {
  RoleListParams,
  CreateRoleRequest,
  UpdateRoleRequest,
  AssignPermissionsRequest,
  RoleListApiResponse,
  RoleApiResponse,
  PermissionListApiResponse,
} from '../types/role'
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

// 获取角色列表
export async function getRoleList(params: RoleListParams): Promise<RoleListApiResponse> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)

  const query = queryParams.toString()
  return request<RoleListApiResponse>(`/api/v1/roles${query ? `?${query}` : ''}`)
}

// 获取角色详情
export async function getRole(id: number): Promise<RoleApiResponse> {
  return request<RoleApiResponse>(`/api/v1/roles/${id}`)
}

// 创建角色
export async function createRole(req: CreateRoleRequest): Promise<RoleApiResponse> {
  return request<RoleApiResponse>('/api/v1/roles', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新角色
export async function updateRole(id: number, req: UpdateRoleRequest): Promise<RoleApiResponse> {
  return request<RoleApiResponse>(`/api/v1/roles/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除角色
export async function deleteRole(id: number): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/roles/${id}`, {
    method: 'DELETE',
  })
}

// 获取角色权限
export async function getRolePermissions(id: number): Promise<PermissionListApiResponse> {
  return request<PermissionListApiResponse>(`/api/v1/roles/${id}/permissions`)
}

// 分配权限
export async function assignPermissions(id: number, req: AssignPermissionsRequest): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/roles/${id}/permissions`, {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 获取所有权限
export async function getAllPermissions(): Promise<PermissionListApiResponse> {
  return request<PermissionListApiResponse>('/api/v1/permissions')
}
