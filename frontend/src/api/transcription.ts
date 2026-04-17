// 本地转录 API 客户端

import type { ApiResponse } from '../types/auth'
import type {
  TranscriptionTriggerRequest,
  TranscriptionTriggerResponse,
  TranscriptionStatusResponse
} from '../types/transcription'
import { apiRequest } from './apiClient'

// 提交转录任务
export async function submitTranscription(
  videoFileId: number,
  samplingRate: number = 0.5  // default 2s per D-02
): Promise<ApiResponse<TranscriptionTriggerResponse>> {
  return apiRequest<TranscriptionTriggerResponse>(
    `/api/v1/videos/${videoFileId}/transcribe`,
    {
      method: 'POST',
      body: JSON.stringify({ sampling_rate: samplingRate }),
    }
  )
}

// 获取转录状态
export async function getTranscriptionStatus(
  videoFileId: number
): Promise<ApiResponse<TranscriptionStatusResponse>> {
  return apiRequest<TranscriptionStatusResponse>(
    `/api/v1/videos/${videoFileId}/transcription-status`
  )
}
