// PPT 管理 API 客户端

import type { ApiResponse } from '../types/auth'
import type { SlidesResponse, PPTListResponse, MergeRequest, MergeResponse } from '../types/ppt'
import { apiRequest, getToken } from './apiClient'

// Frame capture and slide insertion types
export interface CaptureFrameRequest {
  timestamp: number
}

export interface CaptureFrameResponse {
  success: boolean
  frame_data: string // base64 data URL
  timestamp: number
  preview_url: string
}

export interface InsertSlideRequest {
  frame_data: string // base64 data URL
  insert_position: number
  timestamp: number
}

export interface InsertSlideResponse {
  success: boolean
  page_count: number
  inserted_slide_number: number
  new_slide_url: string
  backup_path: string
}

// Duplicate detection types
export interface DuplicateGroup {
  slides: number[]
  similarity: number
  ssim_score: number
  phash_distance: number
  edge_change_rate: number
}

export interface DuplicateDetectionResponse {
  groups: DuplicateGroup[]
  total_scanned: number
  duplicate_count: number
}

export interface DeleteSlidesRequest {
  slides: number[]
}

export interface DeleteSlidesResponse {
  message: string
  page_count: number
  deleted_slides: number[]
  backup_path: string
}

export interface RollbackResponse {
  message: string
  restored: boolean
  page_count: number
}

// 获取 PPT 幻灯片列表
export async function getSlides(pptFileId: number): Promise<ApiResponse<SlidesResponse>> {
  return apiRequest<SlidesResponse>(`/api/v1/ppts/${pptFileId}/slides`)
}

// 获取视频的所有 PPT 结果
export async function getPptsByVideo(videoFileId: number): Promise<ApiResponse<PPTListResponse>> {
  return apiRequest<PPTListResponse>(`/api/v1/videos/${videoFileId}/ppts`)
}

// 批量检查多个视频的 PPT 结果
export interface BatchCheckRequest {
  video_ids: number[]
}

export interface BatchCheckResult {
  has_ppt: boolean
  count?: number
  error?: string
}

export interface BatchCheckResponse {
  results: Record<number, BatchCheckResult>
}

export async function batchCheckPpts(videoIds: number[]): Promise<ApiResponse<BatchCheckResponse>> {
  return apiRequest<BatchCheckResponse>('/api/v1/ppts/batch-check', {
    method: 'POST',
    body: JSON.stringify({ video_ids: videoIds }),
  })
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
  const token = getToken() || ''
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

// PPT 文件类型
export interface PPTFile {
  id: number
  file_name: string
  file_path: string
  file_size: number
  page_count: number
  source_type: string
  created_at: string
}

// 重命名请求
export interface RenameRequest {
  new_name: string
}

// 重命名 PPT 文件
export async function renamePptFile(
  id: number,
  newName: string
): Promise<ApiResponse<{ message: string; data: PPTFile }>> {
  return apiRequest<{ message: string; data: PPTFile }>(`/api/v1/ppts/${id}/rename`, {
    method: 'POST',
    body: JSON.stringify({ new_name: newName }),
  })
}

// 检测重复幻灯片
export async function detectDuplicates(
  pptFileId: number,
  threshold?: number
): Promise<ApiResponse<DuplicateDetectionResponse>> {
  const url = threshold
    ? `/api/v1/ppts/${pptFileId}/duplicates?threshold=${threshold}`
    : `/api/v1/ppts/${pptFileId}/duplicates`
  return apiRequest<DuplicateDetectionResponse>(url)
}

// 删除幻灯片
export async function deleteSlides(
  pptFileId: number,
  slides: number[]
): Promise<ApiResponse<DeleteSlidesResponse>> {
  return apiRequest<DeleteSlidesResponse>(`/api/v1/ppts/${pptFileId}/slides`, {
    method: 'DELETE',
    body: JSON.stringify({ slides }),
  })
}

// 回滚 PPT 到备份版本
export async function rollbackPPT(
  pptFileId: number
): Promise<ApiResponse<RollbackResponse>> {
  return apiRequest<RollbackResponse>(`/api/v1/ppts/${pptFileId}/rollback`, {
    method: 'POST',
  })
}

// 捕获视频帧
export async function captureFrame(
  pptFileId: number,
  timestamp: number
): Promise<ApiResponse<CaptureFrameResponse>> {
  return apiRequest<CaptureFrameResponse>(`/api/v1/ppts/${pptFileId}/capture`, {
    method: 'POST',
    body: JSON.stringify({ timestamp }),
  })
}

// 插入捕获的帧作为新幻灯片
export async function insertSlide(
  pptFileId: number,
  frameData: string,
  insertPosition: number,
  timestamp: number
): Promise<ApiResponse<InsertSlideResponse>> {
  return apiRequest<InsertSlideResponse>(`/api/v1/ppts/${pptFileId}/slides`, {
    method: 'POST',
    body: JSON.stringify({
      frame_data: frameData,
      insert_position: insertPosition,
      timestamp,
    }),
  })
}

// 获取捕获帧预览 URL
export function getCapturedPreviewUrl(pptFileId: number, timestamp: number): string {
  const token = getToken() || ''
  return `/api/v1/ppts/${pptFileId}/captured-preview?ts=${timestamp}&token=${token}`
}

// 重排序幻灯片请求
export interface ReorderSlidesRequest {
  slide_order: number[] // 新的幻灯片顺序 [slide_number, ...]
}

export interface ReorderSlidesResponse {
  success: boolean
  message: string
  new_order: number[]
}

// 重排序幻灯片
export async function reorderSlides(
  pptFileId: number,
  slideOrder: number[]
): Promise<ApiResponse<ReorderSlidesResponse>> {
  return apiRequest<ReorderSlidesResponse>(`/api/v1/ppts/${pptFileId}/reorder`, {
    method: 'POST',
    body: JSON.stringify({ slide_order: slideOrder }),
  })
}
