import { ApiResponse } from './auth'

// 任务列表请求参数
export interface TaskListParams {
  page?: number
  page_size?: number
  keyword?: string
  status?: VideoRecordingTaskStatus
  created_by?: number
  start_date?: string
  end_date?: string
}

// 任务列表响应
export interface TaskListData {
  total: number
  items: VideoRecordingTask[]
}

// 视频录制任务状态
export type VideoRecordingTaskStatus =
  | 'pending'     // 待执行
  | 'connecting'  // 连接会议中
  | 'recording'   // 录制中
  | 'completed'   // 已完成
  | 'failed'      // 执行失败
  | 'cancelled'   // 已取消

// 视频录制任务
export interface VideoRecordingTask {
  id: number
  created_at: string
  updated_at: string
  name: string
  description: string
  start_time: string
  end_time: string
  pre_join_minutes: number
  record_delay_minutes: number
  conference_number: string
  huawei_config_id: number
  huawei_config?: HuaweiConfig
  status: VideoRecordingTaskStatus
  recording_file: string
  recording_duration: number // 秒
  error_msg?: string
  created_by: number
  creator?: TaskCreator
  conference_record?: Record<string, unknown>
  conference_record_id?: number
}

// 华为配置（简要）
export interface HuaweiConfig {
  id: number
  name: string
  site_url: string
}

// 任务创建者
export interface TaskCreator {
  id: number
  username: string
  full_name: string
}

// 创建任务请求
export interface CreateTaskRequest {
  name: string
  description?: string
  start_time: string // RFC3339
  end_time: string   // RFC3339
  pre_join_minutes?: number
  record_delay_minutes?: number
  conference_number: string
  huawei_config_id: number
}

// 更新任务请求
export interface UpdateTaskRequest {
  name?: string
  description?: string
  start_time?: string // RFC3339
  end_time?: string   // RFC3339
  pre_join_minutes?: number
  record_delay_minutes?: number
}

// API 响应类型扩展
export type TaskListApiResponse = ApiResponse<TaskListData>
export type TaskApiResponse = ApiResponse<VideoRecordingTask>

// 任务状态配置
export const TaskStatusConfig: Record<VideoRecordingTaskStatus, { label: string; color: string; icon?: string }> = {
  pending: { label: '待执行', color: 'default' },
  connecting: { label: '连接中', color: 'processing' },
  recording: { label: '录制中', color: 'blue' },
  completed: { label: '已完成', color: 'success' },
  failed: { label: '失败', color: 'error' },
  cancelled: { label: '已取消', color: 'default' },
}
