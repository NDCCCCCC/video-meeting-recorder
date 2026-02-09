import { ApiResponse } from './auth'

// 用户列表请求参数
export interface UserListParams {
  page?: number
  page_size?: number
  keyword?: string
  role_id?: number
  is_active?: boolean
}

// 用户列表响应
export interface UserListData {
  total: number
  items: UserInfo[]
}

// 用户信息（详细，用于管理）
export interface UserInfo {
  id: number
  created_at: string
  updated_at: string
  username: string
  email: string
  full_name: string
  role_id: number
  role?: Role
  is_active: boolean
  last_login_at: string | null
}

// 角色信息
export interface Role {
  id: number
  name: string
  description: string
}

// 创建用户请求
export interface CreateUserRequest {
  username: string
  password: string
  email?: string
  full_name?: string
  role_id: number
  is_active: boolean
}

// 更新用户请求
export interface UpdateUserRequest {
  email?: string
  full_name?: string
  role_id?: number
  is_active?: boolean
}

// 重置密码请求
export interface ResetPasswordRequest {
  password: string
}

// 更新当前用户资料请求
export interface UpdateProfileRequest {
  email?: string
  full_name?: string
}

// API 响应类型扩展
export type UserListApiResponse = ApiResponse<UserListData>
export type UserApiResponse = ApiResponse<UserInfo>
