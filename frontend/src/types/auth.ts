// 认证相关类型定义

// 统一响应结构
export interface ApiResponse<T = any> {
  code: number
  message: string
  data?: T
}

// 登录请求
export interface LoginRequest {
  username: string
  password: string
}

// 登录响应
export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
}

// 刷新Token响应
export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
  expires_in: number
}

// 用户信息
export interface User {
  id: number
  username: string
  email: string
  full_name: string
  role_id: number
  role_name?: string
  is_active: boolean
}

// 修改密码请求
export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

// 密码验证结果
export interface ValidationResult {
  valid: boolean
  errors: string[]
}

// 认证状态
export interface AuthState {
  user: User | null
  token: string | null
  refreshToken: string | null
  isAuthenticated: boolean
}
