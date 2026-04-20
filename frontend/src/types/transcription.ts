// 本地转录 API 类型定义

// 转录触发请求
export interface TranscriptionTriggerRequest {
  sampling_rate: number  // 1.0 (1s), 0.5 (2s), 0.2 (5s)
}

// 转录触发响应
export interface TranscriptionTriggerResponse {
  video_file_id: number
  status: string
}

// 转录进度阶段
export type TranscriptionStage = 'extracting' | 'detecting' | 'generating'

// 转录任务状态
export type TranscriptionTaskStatus = 'pending' | 'processing' | 'completed' | 'failed'

// 转录状态响应
export interface TranscriptionStatusResponse {
  status: TranscriptionTaskStatus
  current_stage: TranscriptionStage | ''
  frames_processed: number
  total_frames: number
  percentage: number
  error_message: string
  result_ppt_file_id: number | null
}

// 采样率选项
export interface SamplingRateOption {
  label: string
  value: number  // fps value: 1.0, 0.5, 0.2
  secondsPerFrame: number  // display: 1, 2, 5
  description: string
}

// === Cloud transcription additions ===

// Transcription mode (per D-01, D-02)
export type TranscriptionMode = 'local' | 'cloud'

// Cloud transcription stages (per D-06)
export type CloudTranscriptionStage = 'uploading' | 'queued' | 'cloud_processing' | 'downloading'

// Combined stage type (local OR cloud)
export type AnyTranscriptionStage = TranscriptionStage | CloudTranscriptionStage

// Extended status response that includes cloud fields
export interface TranscriptionStatusResponseExtended extends Omit<TranscriptionStatusResponse, 'current_stage'> {
  mode?: TranscriptionMode
  current_stage: AnyTranscriptionStage | ''
  error_message: string
  result_ppt_file_id: number | null
}

// Text segment with timestamps (per D-10, TRAN-05)
export interface TextSegment {
  text: string
  begin_time: number  // milliseconds
  end_time: number    // milliseconds
  segment_index: number
}

// Text content response
export interface TranscriptionTextResponse {
  segments: TextSegment[]
  total_count: number
}

// Extended trigger request with mode parameter (per D-01, D-03)
export interface TranscriptionTriggerRequestExtended {
  sampling_rate?: number  // Only for local mode -- NOT sent for cloud per D-03
  mode?: TranscriptionMode  // 'local' or 'cloud'
}

// Extended trigger response with mode
export interface TranscriptionTriggerResponseExtended {
  video_file_id: number
  status: string
  mode: TranscriptionMode
}
