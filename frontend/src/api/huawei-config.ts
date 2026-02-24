// 华为配置管理 API 客户端

import type {
  HuaweiConfig,
  HuaweiConfigListParams,
  HuaweiConfigListResponse,
  CreateHuaweiConfigRequest,
  UpdateHuaweiConfigRequest,
  USBDevicesScanResult,
  USBDeviceInfo,
  TestStreamRequest,
} from '../types/huawei-config'
import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'

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
  return apiRequest(`/api/v1/huawei-configs${query ? `?${query}` : ''}`)
}

// 获取配置详情
export async function getHuaweiConfig(
  id: number
): Promise<ApiResponse<HuaweiConfig>> {
  return apiRequest(`/api/v1/huawei-configs/${id}`)
}

// 创建配置
export async function createHuaweiConfig(
  req: CreateHuaweiConfigRequest
): Promise<ApiResponse<HuaweiConfig>> {
  return apiRequest('/api/v1/huawei-configs', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新配置
export async function updateHuaweiConfig(
  id: number,
  req: UpdateHuaweiConfigRequest
): Promise<ApiResponse<HuaweiConfig>> {
  return apiRequest(`/api/v1/huawei-configs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除配置
export async function deleteHuaweiConfig(
  id: number
): Promise<ApiResponse<void>> {
  return apiRequest(`/api/v1/huawei-configs/${id}`, {
    method: 'DELETE',
  })
}

// 获取可用配置列表
export async function getActiveHuaweiConfigs(): Promise<ApiResponse<HuaweiConfig[]>> {
  return apiRequest('/api/v1/huawei-configs/active')
}

// 扫描USB设备
export async function scanUSBDevices(): Promise<ApiResponse<USBDevicesScanResult>> {
  return apiRequest('/api/v1/huawei-configs/scan-devices')
}

// 获取推荐设备
export async function getRecommendedDevice(type: 'camera' | 'audio'): Promise<ApiResponse<USBDeviceInfo | null>> {
  return apiRequest(`/api/v1/huawei-configs/recommended-device?type=${type}`)
}

// 测试流媒体连接
export async function testStream(
  req: TestStreamRequest
): Promise<ApiResponse<void>> {
  return apiRequest('/api/v1/huawei-configs/test-stream', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}
