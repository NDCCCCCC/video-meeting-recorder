// 视频分割 API 客户端

import type { ApiResponse } from '../types/auth'
import type { VideoFile } from '../types/video-file'
import { apiRequest } from './apiClient'

export interface SplitRequest {
  markers: number[]
  re_encode?: boolean
}

export interface SplitResponse {
  video_file_id: number
  status: string
  segment_count: number
}

export interface SplitStatusResponse {
  status: string
  segments?: VideoFile[]
}

// 提交分割任务
export async function submitSplit(
  videoFileId: number,
  markers: number[],
  reEncode: boolean = false
): Promise<ApiResponse<SplitResponse>> {
  return apiRequest<SplitResponse>(`/api/v1/videos/${videoFileId}/split`, {
    method: 'POST',
    body: JSON.stringify({ markers, re_encode: reEncode }),
  })
}

// 获取分割状态
export async function getSplitStatus(
  videoFileId: number
): Promise<ApiResponse<SplitStatusResponse>> {
  return apiRequest<SplitStatusResponse>(`/api/v1/videos/${videoFileId}/split-status`)
}

// 获取分割段落列表
export async function getSegments(videoFileId: number): Promise<ApiResponse<VideoFile[]>> {
  return apiRequest<VideoFile[]>(`/api/v1/videos/${videoFileId}/segments`)
}
