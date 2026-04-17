// PPT 管理 API 客户端

import type { ApiResponse } from '../types/auth'
import type { SlidesResponse, PPTListResponse, MergeRequest, MergeResponse } from '../types/ppt'
import { apiRequest } from './apiClient'

// 获取 PPT 幻灯片列表
export async function getSlides(pptFileId: number): Promise<ApiResponse<SlidesResponse>> {
  return apiRequest<SlidesResponse>(`/api/v1/ppts/${pptFileId}/slides`)
}

// 获取视频的所有 PPT 结果
export async function getPptsByVideo(videoFileId: number): Promise<ApiResponse<PPTListResponse>> {
  return apiRequest<PPTListResponse>(`/api/v1/videos/${videoFileId}/ppts`)
}

// 合并幻灯片
export async function mergeSlides(req: MergeRequest): Promise<ApiResponse<MergeResponse>> {
  return apiRequest<MergeResponse>('/api/v1/ppts/merge', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

// 删除 PPT 文件
export async function deletePpt(pptFileId: number): Promise<ApiResponse<{ message: string }>> {
  return apiRequest<{ message: string }>(`/api/v1/ppts/${pptFileId}`, {
    method: 'DELETE',
  })
}

// 下载 PPT 文件（使用 window.open 因为返回二进制文件）
export function getPptDownloadUrl(pptFileId: number): string {
  const token = localStorage.getItem('token') || ''
  return `/api/v1/ppts/${pptFileId}/download?token=${token}`
}

// 获取幻灯片图片 URL（GET 图片通过 c.File 提供，不需要额外认证）
export function getSlideImageUrl(
  pptFileId: number,
  resolution: 'thumbnails' | 'fullsize',
  filename: string
): string {
  return `/api/v1/ppts/${pptFileId}/slides/${resolution}/${filename}`
}
