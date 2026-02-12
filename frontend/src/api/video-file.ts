// 视频文件管理 API 客户端

import type {
  VideoFile,
  VideoFileListParams,
  VideoFileListResponse,
  VideoFileStats,
} from '../types/video-file'
import type { ApiResponse } from '../types/auth'
import { apiRequest, getToken } from './apiClient'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// 获取文件列表
export async function getVideoFileList(
  params: VideoFileListParams
): Promise<ApiResponse<VideoFileListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  // 按任务ID筛选
  if (params.task_id) queryParams.append('task_id', params.task_id.toString())
  if (params.status) queryParams.append('status', params.status)
  if (params.format) queryParams.append('format', params.format)
  if (params.start_date) queryParams.append('start_date', params.start_date)
  if (params.end_date) queryParams.append('end_date', params.end_date)

  const query = queryParams.toString()
  return apiRequest<VideoFileListResponse>(
    `/api/v1/files${query ? `?${query}` : ''}`
  )
}

// 获取文件详情
export async function getVideoFile(id: number): Promise<ApiResponse<VideoFile>> {
  return apiRequest<VideoFile>(`/api/v1/files/${id}`)
}

// 下载文件
export async function downloadVideoFile(id: number): Promise<void> {
  const url = `${API_BASE_URL}/api/v1/files/${id}/download`
  const token = getToken()

  const response = await fetch(url, {
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })

  if (!response.ok) {
    const data = await response.json()
    throw new Error(data.message || 'Download failed')
  }

  // 获取文件名
  const contentDisposition = response.headers.get('Content-Disposition')
  let fileName = `video_${id}.mp4`
  if (contentDisposition) {
    const match = contentDisposition.match(/filename="(.+)"/)
    if (match) fileName = match[1]
  }

  // 下载文件
  const blob = await response.blob()
  const blobUrl = window.URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = blobUrl
  link.download = fileName
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.URL.revokeObjectURL(blobUrl)
}

// 删除文件
export async function deleteVideoFile(id: number): Promise<ApiResponse<void>> {
  return apiRequest<void>(`/api/v1/files/${id}`, {
    method: 'DELETE',
  })
}

// 获取文件统计
export async function getVideoFileStats(): Promise<ApiResponse<VideoFileStats>> {
  return apiRequest<VideoFileStats>('/api/v1/files/stats')
}

// 扫描并导入文件
export async function scanVideoFiles(): Promise<ApiResponse<ScanResult>> {
  return apiRequest<ScanResult>('/api/v1/files/scan', {
    method: 'POST',
  })
}

// 扫描结果
export interface ScanResult {
  scanned: number    // 扫描到的文件数
  created: number    // 新创建的记录数
  skipped: number    // 跳过的文件数（已存在）
  errors: string[]   // 错误信息列表
}
