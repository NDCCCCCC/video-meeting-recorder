// Dashboard types

export interface DashboardStatsResponse {
  task_stats: TaskStats
  file_stats: FileStats
  system_stats: SystemStats
}

export interface TaskStats {
  total: number
  in_progress: number
  success: number
  fail: number
  avg_time: number // seconds
}

export interface FileStats {
  total_videos: number
  storage_mb: number
  transcripts: number
  ppts: number
}

export interface SystemStats {
  disk_usage_percent: number
  memory_usage_percent: number
  error_count: number
  api_calls: number
}
