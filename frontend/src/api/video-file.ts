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
  const url = `${API_BASE_URL}/api/v1/files/${id}/download`

  fetch(url, {
    headers: {
      ...(token ? { 'Authorization': `Bearer ${token}` } : {})
    }
  })
  .then(response => response.blob())
  .then(blob => {
    const blobUrl = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = fileName || `video_${id}.mp4`
    link.click()
    URL.revokeObjectURL(blobUrl)
  })
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

// 文件上传结果
export interface FileUploadResult {
  file_id: number
  file_name: string
  file_path: string
  file_size: number
  mime_type: string
  access_url: string
}

// 上传视频文件（带进度回调）
export function uploadVideoFile(
  file: File,
  onProgress?: (percent: number) => void
): Promise<ApiResponse<FileUploadResult>> {
  return new Promise((resolve, reject) => {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('folder', 'videos')

    const xhr = new XMLHttpRequest()

    // Track upload progress
    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable && onProgress) {
        const percent = (event.loaded / event.total) * 100
        onProgress(percent)
      }
    })

    // Handle completion
    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        try {
          const response = JSON.parse(xhr.responseText)
          resolve(response)
        } catch (error) {
          reject(new Error('解析响应失败'))
        }
      } else {
        reject(new Error(`上传失败: ${xhr.status} ${xhr.statusText}`))
      }
    })

    // Handle errors
    xhr.addEventListener('error', () => {
      reject(new Error('网络错误，上传失败'))
    })

    xhr.addEventListener('abort', () => {
      reject(new Error('上传已取消'))
    })

    // Open and send request
    const token = getToken()
    xhr.open('POST', `${API_BASE_URL}/api/v1/storage/upload`)
    if (token) {
      xhr.setRequestHeader('Authorization', `Bearer ${token}`)
    }
    xhr.send(formData)
  })
}
