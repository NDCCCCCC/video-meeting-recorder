// API密钥相关类型定义

export interface APIKey {
  id: number
  name: string
  key: string
  user_id: number
  expires_at: string | null
  is_active: boolean
  scopes: string[]
  ip_whitelist: string[]
  description: string
  inherit_perms: boolean
  last_used_at: string | null
  created_at: string
  updated_at: string
}

export interface CreateAPIKeyRequest {
  name: string
  expires_at: string | null
  scopes: string[]
  inherit_perms: boolean
  ip_whitelist: string[]
  description: string
}

export interface UpdateAPIKeyRequest {
  name?: string
  is_active?: boolean
  scopes?: string[]
  ip_whitelist?: string[]
  description?: string
}

export interface ListAPIKeysRequest {
  page?: number
  page_size?: number
  keyword?: string
  is_active?: boolean
}

export interface ListAPIKeysResponse {
  total: number
  items: APIKey[]
}

// API密钥作用域选项
export const API_KEY_SCOPES = [
  { label: '读取', value: 'read' },
  { label: '写入', value: 'write' },
  { label: '管理员', value: 'admin' },
]

export type APIKeyScope = 'read' | 'write' | 'admin'
