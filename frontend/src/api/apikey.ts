// API密钥管理 API
import { apiRequest } from './apiClient'
import type {
  APIKey,
  CreateAPIKeyRequest,
  UpdateAPIKeyRequest,
  ListAPIKeysRequest,
  ListAPIKeysResponse,
} from '../types/apikey'

// 获取API密钥列表
export function listAPIKeys(params?: ListAPIKeysRequest) {
  const queryParams = new URLSearchParams()
  if (params?.page) queryParams.append('page', params.page.toString())
  if (params?.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params?.keyword) queryParams.append('keyword', params.keyword)
  if (params?.is_active !== undefined) queryParams.append('is_active', params.is_active.toString())

  const query = queryParams.toString()
  return apiRequest<ListAPIKeysResponse>(`/api/v1/apikeys${query ? `?${query}` : ''}`)
}

// 获取API密钥详情
export function getAPIKey(id: number) {
  return apiRequest<APIKey>(`/api/v1/apikeys/${id}`)
}

// 创建API密钥
export function createAPIKey(data: CreateAPIKeyRequest) {
  return apiRequest<APIKey>('/api/v1/apikeys', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

// 更新API密钥
export function updateAPIKey(id: number, data: UpdateAPIKeyRequest) {
  return apiRequest<APIKey>(`/api/v1/apikeys/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

// 删除API密钥
export function deleteAPIKey(id: number) {
  return apiRequest<void>(`/api/v1/apikeys/${id}`, {
    method: 'DELETE',
  })
}

// 切换API密钥状态
export function toggleAPIKeyStatus(id: number) {
  return apiRequest<APIKey>(`/api/v1/apikeys/${id}/toggle`, {
    method: 'POST',
  })
}
