// 会议记录相关类型定义

export type ConferenceStatus = 'not_started' | 'in_progress' | 'completed' | 'failed'

export interface ConferenceRecord {
  id: number
  conference_number: string
  title: string
  start_time: string
  end_time: string | null
  status: ConferenceStatus
  attendees: number
  description: string
  huawei_config_id: number | null
  huawei_config?: {
    id: number
    name: string
    server: string
  } | null
  video_files?: Array<{
    id: number
    file_name: string
    file_path: string
    file_size: number
    duration: number
  }>
  video_recording_task?: {
    id: number
    name: string
    status: string
  } | null
  created_at: string
  updated_at: string
}

export interface ConferenceListParams {
  page?: number
  page_size?: number
  keyword?: string
  status?: ConferenceStatus
  conference_number?: string
  start_date?: string
  end_date?: string
}

export interface ConferenceListResponse {
  total: number
  items: ConferenceRecord[]
}

export interface CreateConferenceRequest {
  conference_number: string
  title: string
  start_time: string
  end_time: string
  description?: string
  huawei_config_id?: number
}

export interface UpdateConferenceRequest {
  title?: string
  end_time?: string
  description?: string
  status?: ConferenceStatus
}
