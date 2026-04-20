// 视频文件管理 API 客户端

import type {
  VideoFile,
  VideoFileListParams,
  VideoFileListResponse,
  VideoFileStats,
} from '../types/video-file'
import type { ApiResponse } from '../types/auth'
import { apiRequest, getToken } from './apiClient'

const API_BASE_URL = import.meta.env.VITE_API_URL || ''

// 获取文件列表
export async function getVideoFileList(
  params: VideoFileListParams
): Promise<ApiResponse<VideoFileListResponse>> {
  const queryParams = new URLSearchParams()
  if (params.page) queryParams.append('page', params.page.toString())
  if (params.page_size) queryParams.append('page_size', params.page_size.toString())
  if (params.keyword) queryParams.append('keyword', params.keyword)
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

// 下载文件（触发浏览器原生下载，显示进度）
export function downloadVideoFile(id: number, fileName?: string): void {
  const token = getToken()
  const url = token
    ? `${API_BASE_URL}/api/v1/files/${id}/download?token=${token}`
    : `${API_BASE_URL}/api/v1/files/${id}/download`

  const link = document.createElement('a')
  link.href = url
  link.download = fileName || `video_${id}.mp4`
  link.style.display = 'none'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

// 删除文件
export async function deleteVideoFile(id: number): Promise<ApiResponse<void>> {
  return apiRequest<void>(`/api/v1/files/${id}`, {
    method: 'DELETE',
  })
}

// 获取文件统计（可指定格式，默认只统计 mp4）
export async function getVideoFileStats(format: string = 'mp4'): Promise<ApiResponse<VideoFileStats>> {
  const params = format ? `?format=${format}` : ''
  return apiRequest<VideoFileStats>(`/api/v1/files/stats${params}`)
}

// 扫描并导入文件
export async function scanVideoFiles(): Promise<ApiResponse<ScanResult>> {
  return apiRequest<ScanResult>('/api/v1/files/scan', {
    method: 'POST',
  })
}

// 扫描结果
export interface ScanResult {
  scanned: number
  created: number
  skipped: number
  errors: string[]
}

// 批量删除请求
export interface BatchDeleteFilesRequest {
  ids: number[]
}

// 批量删除结果
export interface BatchDeleteFilesResult {
  success: number
  failed: number
  errors: string[]
}

// 批量删除文件
export async function batchDeleteFiles(
  ids: number[]
): Promise<ApiResponse<BatchDeleteFilesResult>> {
  return apiRequest<BatchDeleteFilesResult>('/api/v1/files/batch', {
    method: 'DELETE',
    body: JSON.stringify({ ids }),
  })
}

// 重命名请求
export interface RenameRequest {
  new_name: string
}

// 重命名视频文件
export async function renameVideoFile(
  id: number,
  newName: string
): Promise<ApiResponse<VideoFile>> {
  return apiRequest<VideoFile>(`/api/v1/videos/${id}/rename`, {
    method: 'POST',
    body: JSON.stringify({ new_name: newName }),
  })
}

// 获取视频的所有分割段
export async function getVideoSegments(parentId: number): Promise<ApiResponse<VideoFile[]>> {
  return apiRequest<VideoFile[]>(`/api/v1/videos/${parentId}/segments`)
}
