// 视频文件相关类型定义
//
// 迁移说明（v2.0 重构）:
// - 移除 conference_record_id 和 conference_record 字段
// - 新增 task_id 和 task 字段，直接关联 VideoRecordingTask
// - 文件扫描时从路径（task_XX）推断 task_id
// - 数据库结构从 Task -> ConferenceRecord -> File 简化为 Task -> File

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
  task_id?: number | null      // 关联任务ID
  task?: {
    id: number
    name: string
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
  task_id?: number       // 按任务ID筛选
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
