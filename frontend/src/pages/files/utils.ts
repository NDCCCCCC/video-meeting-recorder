// 文件管理页面工具函数

/**
 * 格式化文件大小（字节 -> MB）
 * @param bytes 字节数
 * @returns 形如 "12.34 MB" 的字符串
 */
export function formatFileSize(bytes: number): string {
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

/**
 * 格式化时长（秒 -> MM:SS）
 * @param seconds 秒数
 * @returns 形如 "M:SS" 的字符串
 */
export function formatDuration(seconds: number): string {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}
