// 系统设置 API 客户端

import type { ApiResponse } from '../types/auth'
import { apiRequest } from './apiClient'

// 系统配置
export interface SystemConfig {
  storage: {
    recordings_path: string
    hls_path: string
    temp_path: string
    max_disk_usage: number
  }
  ffmpeg: {
    path: string
    ffprobe_path: string
  }
  logging: {
    level: string
    format: string
    output: string
  }
}

// 更新系统配置请求
export interface UpdateSystemConfigRequest {
  recordings_path?: string
  hls_path?: string
  temp_path?: string
  max_disk_usage?: number
  ffmpeg_path?: string
  ffprobe_path?: string
  log_level?: string
  log_format?: string
  log_output?: string
}

// 获取系统配置
export async function getSystemConfig(): Promise<ApiResponse<SystemConfig>> {
  return apiRequest<SystemConfig>('/api/v1/system/config')
}

// 更新系统配置
export async function updateSystemConfig(
  data: UpdateSystemConfigRequest
): Promise<ApiResponse<{ message: string }>> {
  const response = await fetch('/api/v1/system/config', {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...(localStorage.getItem('token') ? { Authorization: `Bearer ${localStorage.getItem('token')}` } : {}),
    },
    body: JSON.stringify(data),
  })

  if (!response.ok) {
    const err = await response.json()
    throw new Error(err.message || '更新配置失败')
  }

  return response.json()
}

// 清空文件数据库
export async function clearFileDatabase(): Promise<ApiResponse<{ message: string }>> {
  return apiRequest<{ message: string }>('/api/v1/system/clear-files', {
    method: 'POST',
  })
}
