import { ApiResponse } from './auth'
import type { InputConfig } from './input-config'

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
  | 'pending' // 待执行
  | 'connecting' // 连接会议中
  | 'recording' // 录制中
  | 'converting' // 转换中（MKV转MP4）
  | 'completed' // 已完成
  | 'failed' // 执行失败
  | 'cancelled' // 已取消

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
  // 输入配置（旧字段 input_config_id 兼容保留；Phase 13 后多配置请使用 task_input_configs）
  input_config_id?: number | null
  input_config?: InputConfig
  task_input_configs?: TaskInputConfig[]
  status: VideoRecordingTaskStatus
  recording_file: string
  recording_duration: number // 秒
  error_msg?: string
  created_by: number
  creator?: TaskCreator
  conference_record?: Record<string, unknown>
  conference_record_id?: number
}

// 任务-输入配置关联（来自 task_input_configs 表）
export interface TaskInputConfig {
  id: number
  task_id: number
  input_config_id: number
  config_type: string
  created_at: string
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
  end_time: string // RFC3339
  pre_join_minutes?: number
  record_delay_minutes?: number
  conference_number: string
  input_config_ids?: number[] // 输入配置ID列表
}

// 更新任务请求
export interface UpdateTaskRequest {
  name?: string
  description?: string
  start_time?: string // RFC3339
  end_time?: string // RFC3339
  pre_join_minutes?: number
  record_delay_minutes?: number
  // 输入配置 ID 列表；后端 nil = 不动、非 nil = 同步为该列表（包含空数组 = 清空）
  input_config_ids?: number[]
}

// API 响应类型扩展
export type TaskListApiResponse = ApiResponse<TaskListData>
export type TaskApiResponse = ApiResponse<VideoRecordingTask>

// 任务状态配置
export const TaskStatusConfig: Record<
  VideoRecordingTaskStatus,
  { label: string; color: string; icon?: string }
> = {
  pending: { label: '待执行', color: 'default' },
  connecting: { label: '连接中', color: 'processing' },
  recording: { label: '录制中', color: 'blue' },
  converting: { label: '转换中', color: 'warning' },
  completed: { label: '已完成', color: 'success' },
  failed: { label: '失败', color: 'error' },
  cancelled: { label: '已取消', color: 'default' },
}
