import { ApiResponse } from './auth'

// 审计日志
export interface AuditLog {
  id: number
  user_id?: number
  username: string
  role_id?: number
  role_name?: string
  action: string
  module: string
  resource?: string
  resource_id?: number
  request_id?: string
  trace_id?: string
  method?: string
  path?: string
  old_data?: string
  new_data?: string
  diff_data?: string
  status: string
  error_msg?: string
  error_code?: string
  ip_address?: string
  user_agent?: string
  duration: number
  created_at: string
}

// 审计日志列表请求参数
export interface AuditLogListParams {
  page?: number
  page_size?: number
  username?: string
  action?: string
  module?: string
  start_time?: string
  end_time?: string
}

// 审计日志导出请求参数
export interface AuditLogExportParams {
  format: 'csv' | 'json'
  username?: string
  action?: string
  module?: string
  start_time?: string
  end_time?: string
}

// 审计日志列表数据
export interface AuditLogListData {
  total: number
  items: AuditLog[]
}

// API 响应类型扩展
export type AuditLogListApiResponse = ApiResponse<AuditLogListData>
export type AuditLogApiResponse = ApiResponse<AuditLog>
