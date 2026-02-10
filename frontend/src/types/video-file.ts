// 视频文件相关类型定义

export type VideoFileStatus = 'ready' | 'processing' | 'error' | 'deleting'

export interface VideoFile {
  id: number
  file_name: string
  file_path: string
  file_size: number
  duration: number
  format: string
  resolution: string
  bitrate: number
  codec: string
  conference_record_id: number | null
  conference_record?: {
    id: number
    title: string
    conference_number: string
  } | null
  status: VideoFileStatus
  thumbnail_path: string | null
  recorded_at: string | null
  created_at: string
  updated_at: string
}

export interface VideoFileListParams {
  page?: number
  page_size?: number
  keyword?: string
  conference_record_id?: number
  status?: VideoFileStatus
  format?: string
  start_date?: string
  end_date?: string
}

export interface VideoFileListResponse {
  total: number
  items: VideoFile[]
  total_size: number
  total_size_gb: number
}

export interface VideoFileStats {
  total: number
  total_size: number
  total_size_gb: number
}
