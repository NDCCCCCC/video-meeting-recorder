// 华为配置管理 API 客户端

import type {
  HuaweiConfig,
  HuaweiConfigListParams,
  HuaweiConfigListResponse,
  CreateHuaweiConfigRequest,
  UpdateHuaweiConfigRequest,
  USBDevicesScanResult,
  USBDeviceInfo,
} from '../types/huawei-config'
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

// 获取配置列表
export async function getHuaweiConfigList(
  params: HuaweiConfigListParams
): Promise<ApiResponse<HuaweiConfigListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString())

  const query = queryParams.toString()
  return request<ApiResponse<HuaweiConfigListResponse>>(
    `/api/v1/huawei-configs${query ? `?${query}` : ''}`
  )
}

// 获取配置详情
export async function getHuaweiConfig(
  id: number
): Promise<ApiResponse<HuaweiConfig>> {
  return request<ApiResponse<HuaweiConfig>>(`/api/v1/huawei-configs/${id}`)
}

// 创建配置
export async function createHuaweiConfig(
  req: CreateHuaweiConfigRequest
): Promise<ApiResponse<HuaweiConfig>> {
  return request<ApiResponse<HuaweiConfig>>('/api/v1/huawei-configs', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新配置
export async function updateHuaweiConfig(
  id: number,
  req: UpdateHuaweiConfigRequest
): Promise<ApiResponse<HuaweiConfig>> {
  return request<ApiResponse<HuaweiConfig>>(`/api/v1/huawei-configs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除配置
export async function deleteHuaweiConfig(
  id: number
): Promise<ApiResponse<void>> {
  return request<ApiResponse<void>>(`/api/v1/huawei-configs/${id}`, {
    method: 'DELETE',
  })
}

// 获取可用配置列表
export async function getActiveHuaweiConfigs(): Promise<ApiResponse<HuaweiConfig[]>> {
  return request<ApiResponse<HuaweiConfig[]>>('/api/v1/huawei-configs/active')
}

// 扫描USB设备
export async function scanUSBDevices(): Promise<ApiResponse<USBDevicesScanResult>> {
  return request<ApiResponse<USBDevicesScanResult>>('/api/v1/huawei-configs/scan-devices')
}

// 获取推荐设备
export async function getRecommendedDevice(type: 'camera' | 'audio'): Promise<ApiResponse<USBDeviceInfo | null>> {
  return request<ApiResponse<USBDeviceInfo | null>>(`/api/v1/huawei-configs/recommended-device?type=${type}`)
}
