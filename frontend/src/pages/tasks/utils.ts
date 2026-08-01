// 任务管理页面工具函数

import type { InputConfig } from '../../types/input-config'

/**
 * 格式化时长（秒 -> HH:MM:SS 或 MM:SS）
 * @param seconds 秒数
 * @returns 格式化的时长字符串
 */
export function formatDuration(seconds: number | undefined | null): string {
  if (!seconds) return '-'

  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

/**
 * 检查是否有活跃任务（用于触发轮询）
 * @param tasks 任务列表
 * @param activeStatuses 活跃状态列表
 * @returns 是否有活跃任务
 */
export function hasActiveTasks<T extends { status: string }>(
  tasks: T[],
  activeStatuses: string[]
): boolean {
  return tasks.some((task) => activeStatuses.includes(task.status))
}

/**
 * 检查任务是否可删除
 * @param status 任务状态
 * @param deletableStatuses 可删除的状态列表
 * @returns 是否可删除
 */
export function isTaskDeletable(status: string, deletableStatuses: string[]): boolean {
  return deletableStatuses.includes(status)
}

/**
 * 获取表格行的复选框属性
 * @param deletableTaskIds 可删除的任务ID集合
 * @returns 复选框属性函数
 */
export function getCheckboxProps(deletableTaskIds: Set<number>) {
  return (record: { id: number; name: string }) => ({
    disabled: !deletableTaskIds.has(record.id),
    name: record.name,
  })
}

// ============ 输入配置类型相关 ============

export type ConfigType = 'usb' | 'stream' | 'none'

export const CONFIG_TYPE_TAGS: Record<ConfigType, { color: string; label: string }> = {
  usb: { color: 'blue', label: 'USB' },
  stream: { color: 'green', label: '流媒体' },
  none: { color: 'default', label: '未配置' },
}

/**
 * 判定输入配置的类型（USB / 流媒体 / 未配置）
 */
export function getConfigType(config: InputConfig): ConfigType {
  const hasUSB = config.usb_camera_device || config.usb_audio_device
  const hasStream = config.stream_enabled && config.stream_url

  if (hasUSB) return 'usb'
  if (hasStream) return 'stream'
  return 'none'
}

/**
 * 获取输入配置的类型标签配置（含流媒体协议后缀）
 */
export function getConfigTypeTagConfig(config: InputConfig) {
  const type = getConfigType(config)
  const tagConfig = CONFIG_TYPE_TAGS[type]
  const label =
    type === 'stream'
      ? `${tagConfig.label}(${config.stream_protocol?.toUpperCase()})`
      : tagConfig.label
  return { ...tagConfig, label }
}

/**
 * 校验输入配置选择：最多1路USB + 最多1路流媒体，且不能选"未配置"项。
 * 供 handleSubmit 与表单 validator 复用，消除重复校验。
 * @returns 第一条错误信息；全部合法返回 null
 */
export function validateInputConfigSelection(selectedConfigs: InputConfig[]): string | null {
  const usbCount = selectedConfigs.filter((c) => getConfigType(c) === 'usb').length
  const streamCount = selectedConfigs.filter((c) => getConfigType(c) === 'stream').length

  if (usbCount > 1) return '最多只能选择1个USB配置'
  if (streamCount > 1) return '最多只能选择1个流媒体配置'

  const invalidConfigs = selectedConfigs.filter((c) => getConfigType(c) === 'none')
  if (invalidConfigs.length > 0) {
    return `配置"${invalidConfigs[0].name}"未配置USB设备或流媒体`
  }
  return null
}
