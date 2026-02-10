// 视频文件管理 API 客户端

import type {
  VideoFile,
  VideoFileListParams,
  VideoFileListResponse,
  VideoFileStats,
} from '../types/video-file'
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

// 获取文件列表
export async function getVideoFileList(
  params: VideoFileListParams
): Promise<ApiResponse<VideoFileListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
  if (params.conference_record_id) queryParams.append('conference_record_id', params.conference_record_id.toString())
  if (params.status) queryParams.append('status', params.status)
  if (params.format) queryParams.append('format', params.format)
  if (params.start_date) queryParams.append('start_date', params.start_date)
  if (params.end_date) queryParams.append('end_date', params.end_date)

  const query = queryParams.toString()
  return request<ApiResponse<VideoFileListResponse>>(
    `/api/v1/files${query ? `?${query}` : ''}`
  )
}

// 获取文件详情
export async function getVideoFile(id: number): Promise<ApiResponse<VideoFile>> {
  return request<ApiResponse<VideoFile>>(`/api/v1/files/${id}`)
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
  return request<ApiResponse<void>>(`/api/v1/files/${id}`, {
    method: 'DELETE',
  })
}

// 获取文件统计
export async function getVideoFileStats(): Promise<ApiResponse<VideoFileStats>> {
  return request<ApiResponse<VideoFileStats>>('/api/v1/files/stats')
}
