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
import { apiRequest } from './apiClient'

// 获取角色列表
export async function getRoleList(params: RoleListParams): Promise<RoleListApiResponse> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)

  const query = queryParams.toString()
  return apiRequest(`/api/v1/roles${query ? `?${query}` : ''}`)
}

// 获取角色详情
export async function getRole(id: number): Promise<RoleApiResponse> {
  return apiRequest(`/api/v1/roles/${id}`)
}

// 创建角色
export async function createRole(req: CreateRoleRequest): Promise<RoleApiResponse> {
  return apiRequest('/api/v1/roles', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新角色
export async function updateRole(id: number, req: UpdateRoleRequest): Promise<RoleApiResponse> {
  return apiRequest(`/api/v1/roles/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除角色
export async function deleteRole(id: number): Promise<ApiResponse<void>> {
  return apiRequest(`/api/v1/roles/${id}`, {
    method: 'DELETE',
  })
}

// 获取角色权限
export async function getRolePermissions(id: number): Promise<PermissionListApiResponse> {
  return apiRequest(`/api/v1/roles/${id}/permissions`)
}

// 分配权限
export async function assignPermissions(
  id: number,
  req: AssignPermissionsRequest
): Promise<ApiResponse<void>> {
  return apiRequest(`/api/v1/roles/${id}/permissions`, {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 获取所有权限
export async function getAllPermissions(): Promise<PermissionListApiResponse> {
  return apiRequest('/api/v1/permissions')
}
