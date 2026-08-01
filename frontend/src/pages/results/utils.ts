// PPT 结果详情页工具函数

/**
 * 格式化文件大小（字节 -> MB）
 * @param bytes 字节数
 * @returns 形如 "12.34 MB"；0 或空值返回 "0 MB"
 */
export function formatFileSize(bytes: number): string {
  if (!bytes || bytes === 0) return '0 MB'
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}
