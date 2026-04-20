// 本地转录 API 客户端

import type { ApiResponse } from '../types/auth'
import type {
  TranscriptionTriggerResponse,
  TranscriptionStatusResponse,
  TranscriptionTriggerResponseExtended,
  TranscriptionTextResponse,
  TranscriptionMode,
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

// === Cloud transcription additions ===

// Submit transcription with mode selection (per D-01, D-03)
// IMPORTANT: When mode='cloud', sampling_rate is NOT sent per D-03
export async function submitTranscriptionWithMode(
  videoFileId: number,
  mode: TranscriptionMode,
  samplingRate?: number
): Promise<ApiResponse<TranscriptionTriggerResponseExtended>> {
  const body: Record<string, unknown> = { mode }
  // Per D-03: only include sampling_rate for local mode
  if (mode === 'local' && samplingRate) {
    body.sampling_rate = samplingRate
  }
  // When mode === 'cloud': body is { mode: 'cloud' } with NO sampling_rate key
  return apiRequest<TranscriptionTriggerResponseExtended>(
    `/api/v1/videos/${videoFileId}/transcribe`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    }
  )
}

// Get transcription text content (per TRAN-05, D-09)
export async function getTranscriptionText(
  videoFileId: number
): Promise<ApiResponse<TranscriptionTextResponse>> {
  return apiRequest<TranscriptionTextResponse>(
    `/api/v1/videos/${videoFileId}/transcription-text`
  )
}
