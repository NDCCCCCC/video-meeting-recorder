// 任务管理页面常量配置

import type { VideoRecordingTaskStatus } from '../../types/task'

// 状态配置
export const STATUS_CONFIG: Record<VideoRecordingTaskStatus, { label: string; color: string }> = {
  pending: { label: '待执行', color: 'default' },
  connecting: { label: '连接中', color: 'processing' },
  recording: { label: '录制中', color: 'blue' },
  completed: { label: '已完成', color: 'success' },
  failed: { label: '失败', color: 'error' },
  cancelled: { label: '已取消', color: 'default' },
}

// 常量定义
export const DEFAULT_PAGE_SIZE = 20
export const DEFAULT_PRE_JOIN_MINUTES = 30
export const DEFAULT_RECORD_DELAY_MINUTES = 0
export const POLL_INTERVAL = 5000 // 5秒轮询间隔
export const DELETABLE_STATUSES: VideoRecordingTaskStatus[] = ['pending', 'completed', 'failed', 'cancelled']

// 状态选项（用于筛选器）
export const STATUS_OPTIONS = Object.entries(STATUS_CONFIG).map(([value, { label }]) => ({
  value,
  label,
}))

// 进行中的状态（用于触发轮询）
export const ACTIVE_STATUSES: VideoRecordingTaskStatus[] = ['pending', 'connecting', 'recording']

// 可编辑的状态
export const EDITABLE_STATUS: VideoRecordingTaskStatus = 'pending'

// 可删除的状态
const DELETABLE_STATUS_SET = new Set(DELETABLE_STATUSES)
export function canDeleteTask(status: VideoRecordingTaskStatus): boolean {
  return DELETABLE_STATUS_SET.has(status)
}

// 可启动的状态
export function canStartTask(status: VideoRecordingTaskStatus): boolean {
  return status === 'pending'
}

// 可停止的状态
export function canStopTask(status: VideoRecordingTaskStatus): boolean {
  return status === 'recording' || status === 'connecting'
}

// 可取消的状态
export function canCancelTask(status: VideoRecordingTaskStatus): boolean {
  return status === 'pending' || status === 'connecting'
}

// 可重试的状态
export function canRetryTask(status: VideoRecordingTaskStatus): boolean {
  return status === 'failed'
}

// 可预览的状态
export function canPreviewTask(status: VideoRecordingTaskStatus): boolean {
  return status === 'recording'
}
