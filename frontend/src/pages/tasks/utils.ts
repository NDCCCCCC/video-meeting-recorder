// 任务管理页面工具函数

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
