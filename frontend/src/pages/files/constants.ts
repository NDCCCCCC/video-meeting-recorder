// 文件管理页面常量配置

import type { VideoFileStatus } from '../../types/video-file'
import type { SamplingRateOption } from '../../types/transcription'

// 状态配置
export const STATUS_CONFIG: Record<VideoFileStatus, { label: string; color: string }> = {
  ready: { label: '就绪', color: 'success' },
  processing: { label: '处理中', color: 'processing' },
  error: { label: '错误', color: 'error' },
  deleting: { label: '删除中', color: 'default' },
}

// 状态选项（用于筛选器）
export const STATUS_OPTIONS = Object.entries(STATUS_CONFIG).map(([value, { label }]) => ({
  label,
  value,
}))

// 默认分页大小
export const DEFAULT_PAGE_SIZE = 20
// 默认文件格式筛选
export const DEFAULT_FORMAT = 'mp4'

// 采样率选项 (per D-02, 增加更高精度选项)
export const samplingRateOptions: SamplingRateOption[] = [
  {
    label: '0.05秒/帧',
    value: 0.05,
    secondsPerFrame: 0.05,
    description: '极高精度 (20fps), 文件很大',
  },
  {
    label: '0.1秒/帧',
    value: 0.1,
    secondsPerFrame: 0.1,
    description: '很高精度 (10fps), 文件较大',
  },
  { label: '0.2秒/帧', value: 0.2, secondsPerFrame: 0.2, description: '高精度 (5fps)' },
  { label: '0.5秒/帧', value: 0.5, secondsPerFrame: 0.5, description: '推荐 (2fps)' },
  { label: '1秒/帧', value: 1.0, secondsPerFrame: 1, description: '标准 (1fps), 文件较小' },
]
