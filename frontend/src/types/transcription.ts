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
