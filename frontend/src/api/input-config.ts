// 输入配置管理 API 客户端

import type {
  InputConfig,
  InputConfigListParams,
  InputConfigListResponse,
  CreateInputConfigRequest,
  UpdateInputConfigRequest,
  TestConnectionRequest,
  USBDevicesScanResult,
} from '../types/input-config'
import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'

// 获取输入配置列表
export async function getInputConfigList(
  params: InputConfigListParams
): Promise<ApiResponse<InputConfigListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.is_active !== undefined) queryParams.append('is_active', params.is_active.toString())

  const query = queryParams.toString()
  return apiRequest(`/api/v1/input-configs${query ? `?${query}` : ''}`)
}

// 获取单个输入配置
export async function getInputConfig(
  id: number
): Promise<ApiResponse<InputConfig>> {
  return apiRequest(`/api/v1/input-configs/${id}`)
}

// 创建输入配置
export async function createInputConfig(
  req: CreateInputConfigRequest
): Promise<ApiResponse<InputConfig>> {
  return apiRequest('/api/v1/input-configs', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 更新输入配置
export async function updateInputConfig(
  id: number,
  req: UpdateInputConfigRequest
): Promise<ApiResponse<InputConfig>> {
  return apiRequest(`/api/v1/input-configs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(req),
  })
}

// 删除输入配置
export async function deleteInputConfig(
  id: number
): Promise<ApiResponse<void>> {
  return apiRequest(`/api/v1/input-configs/${id}`, {
    method: 'DELETE',
  })
}

// 测试连接
export async function testConnection(
  id: number,
  req: TestConnectionRequest
): Promise<ApiResponse<void>> {
  return apiRequest(`/api/v1/input-configs/${id}/test`, {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 扫描USB设备
export async function scanUSBDevices(): Promise<ApiResponse<USBDevicesScanResult>> {
  return apiRequest('/api/v1/input-configs/usb-devices')
}

// 获取激活的输入配置
export async function getActiveInputConfigs(): Promise<ApiResponse<InputConfig[]>> {
  return apiRequest('/api/v1/input-configs/active')
}
