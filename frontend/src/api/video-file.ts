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
  return apiRequest<VideoFileListResponse>(`/api/v1/files${query ? `?${query}` : ''}`)
}

// 获取文件详情
export async function getVideoFile(id: number): Promise<ApiResponse<VideoFile>> {
  return apiRequest<VideoFile>(`/api/v1/files/${id}`)
}

// 获取视频播放 URL（video_playback_token）
//
// 返回的 playback_url 是路径形式 /api/v1/files/playback/<token>，浏览器 <video>
// 元素可直接使用；token 5min TTL，过期后由前端重新调用本函数获取新 URL。
export async function getVideoPlaybackUrl(
  id: number
): Promise<ApiResponse<{ playback_url: string; expires_at: number }>> {
  return apiRequest<{ playback_url: string; expires_at: number }>(
    `/api/v1/files/${id}/playback_url`
  )
}

// 下载文件（触发浏览器原生下载，显示进度）
export function downloadVideoFile(id: number, fileName?: string): void {
  const token = getToken()
  const url = `${API_BASE_URL}/api/v1/files/${id}/download`

  fetch(url, {
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
    .then((response) => response.blob())
    .then((blob) => {
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
export async function getVideoFileStats(
  format: string = 'mp4'
): Promise<ApiResponse<VideoFileStats>> {
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
  onProgress?: (percent: number) => void,
  onXhrCreated?: (xhr: XMLHttpRequest) => void
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

          // Validate response structure
          if (!response) {
            reject(new Error('服务器返回的响应为空'))
            return
          }
          if (!response.data) {
            reject(new Error('服务器返回的响应缺少 data 字段'))
            return
          }
          if (response.data.file_id === undefined || response.data.file_id === null) {
            reject(
              new Error(`服务器返回的响应缺少 file_id 字段. data: ${JSON.stringify(response.data)}`)
            )
            return
          }
          resolve(response)
        } catch (error) {
          console.error('[Upload Debug] Parse error:', error)
          reject(new Error('解析响应失败'))
        }
      } else {
        // Try to parse error message from server
        let errorMsg = `上传失败: ${xhr.status} ${xhr.statusText}`
        try {
          const errorResponse = JSON.parse(xhr.responseText)
          console.error('[Upload Debug] Error response:', errorResponse)
          if (errorResponse.message) {
            errorMsg = errorResponse.message
          }
        } catch {
          // Use default error message
        }
        reject(new Error(errorMsg))
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
    // Expose xhr to caller for cancellation support
    if (onXhrCreated) {
      onXhrCreated(xhr)
    }
    xhr.send(formData)
  })
}

// --- Phase 14 Batch Download API ---

import { message } from 'antd'
import dayjs from 'dayjs'

// 批量下载文件（打包为ZIP）
export function batchDownloadFiles(ids: number[]): void {
  const token = getToken()
  const url = `${API_BASE_URL}/api/v1/files/batch/download`

  // 显示加载提示
  const hide = message.loading('正在打包文件，请稍候...', 0)

  fetch(url, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({ ids }),
  })
    .then((response) => {
      if (!response.ok) {
        throw new Error(`下载失败: ${response.status}`)
      }
      return response.blob()
    })
    .then((blob) => {
      // 从响应头获取文件名
      const filename = `files_batch_${dayjs().format('YYYYMMDD_HHmmss')}.zip`

      // 创建下载链接
      const blobUrl = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = blobUrl
      link.download = filename
      link.click()
      URL.revokeObjectURL(blobUrl)

      message.success(`成功下载 ${ids.length} 个文件`)
    })
    .catch((error) => {
      console.error('批量下载失败:', error)
      message.error(error instanceof Error ? error.message : '批量下载失败')
    })
    .finally(() => {
      hide()
    })
}
