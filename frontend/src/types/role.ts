import { ApiResponse } from './auth'

// 角色列表请求参数
export interface RoleListParams {
  page?: number
  page_size?: number
  keyword?: string
}

// 角色列表响应
export interface RoleListData {
  total: number
  items: RoleInfo[]
}

// 角色信息（详细）
export interface RoleInfo {
  id: number
  created_at: string
  updated_at: string
  name: string
  description: string
  allowed_ips?: string[]
  permissions?: Permission[]
}

// 权限信息
export interface Permission {
  id: number
  created_at: string
  updated_at: string
  resource: string
  action: string
  description: string
}

// 创建角色请求
export interface CreateRoleRequest {
  name: string
  description?: string
  allowed_ips?: string[]
}

// 更新角色请求
export interface UpdateRoleRequest {
  description?: string
  allowed_ips?: string[]
}

// 分配权限请求
export interface AssignPermissionsRequest {
  permission_ids: number[]
}

// API 响应类型扩展
export type RoleListApiResponse = ApiResponse<RoleListData>
export type RoleApiResponse = ApiResponse<RoleInfo>
export type PermissionListApiResponse = ApiResponse<Permission[]>
