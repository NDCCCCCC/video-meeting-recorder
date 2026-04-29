// 输入配置相关类型定义

// USB设备信息
export interface USBDeviceInfo {
  type: string // "camera" | "audio"
  name: string // 设备名称
  device_id: string // 设备ID (/dev/video0, hw:1,0)
  status: string // "available" | "in_use" | "error"
  backend: string // "v4l2" | "alsa" | "dshow" | "avfoundation" | "coreaudio"
}

// 配置类型
export type ConfigType = 'huawei_auto' | 'usb' | 'stream'

// 输入配置
export interface InputConfig {
  id: number
  name: string
  description: string
  config_type: ConfigType
  huawei_enabled: boolean

  // 华为终端字段
  server?: string
  port?: number
  username?: string
  terminal_number?: string
  conference_number?: string

  // USB设备字段
  camera_backend?: string
  usb_camera_name?: string
  usb_camera_device?: string
  camera_binding_status?: string
  audio_backend?: string
  usb_audio_name?: string
  usb_audio_device?: string
  audio_binding_status?: string

  // 流媒体字段
  stream_protocol?: 'rtmp' | 'rtsp' | 'srt' | 'hls'
  stream_url?: string
  stream_username?: string
  stream_enabled?: boolean

  // 录制配置
  output_format?: string

  // 状态字段
  is_active: boolean
  is_locked: boolean
  locked_by?: number
  locked_at?: string
  created_at: string
  updated_at: string

  // 关联数据
  video_recording_tasks?: Array<{
    id: number
    name: string
    status: string
  }>
}

// 列表查询参数
export interface InputConfigListParams {
  page?: number
  page_size?: number
  keyword?: string
  is_active?: boolean
}

// 列表响应
export interface InputConfigListResponse {
  total: number
  items: InputConfig[]
}

// 创建输入配置请求
export interface CreateInputConfigRequest {
  name: string
  description?: string
  config_type: ConfigType
  huawei_enabled?: boolean

  // 华为字段
  server?: string
  port?: number
  username?: string
  password?: string
  terminal_number?: string
  conference_number?: string

  // USB字段
  camera_backend?: string
  usb_camera_name?: string
  usb_camera_device?: string
  audio_backend?: string
  usb_audio_name?: string
  usb_audio_device?: string

  // 流媒体字段
  stream_protocol?: string
  stream_url?: string
  stream_username?: string
  stream_password?: string
  stream_enabled?: boolean

  // 录制配置
  output_format?: string
}

// 更新输入配置请求
export interface UpdateInputConfigRequest {
  name?: string
  description?: string
  huawei_enabled?: boolean

  // 华为字段
  server?: string
  port?: number
  username?: string
  password?: string
  terminal_number?: string
  conference_number?: string

  // USB字段
  camera_backend?: string
  usb_camera_name?: string
  usb_camera_device?: string
  audio_backend?: string
  usb_audio_name?: string
  usb_audio_device?: string

  // 流媒体字段
  stream_protocol?: string
  stream_url?: string
  stream_username?: string
  stream_password?: string
  stream_enabled?: boolean

  // 录制配置
  output_format?: string
  is_active?: boolean
}

// 测试连接请求
export interface TestConnectionRequest {
  config_type: ConfigType
  server?: string
  port?: number
  username?: string
  password?: string
  terminal_number?: string
  usb_camera_device?: string
  stream_protocol?: string
  stream_url?: string
  stream_username?: string
  stream_password?: string
}

// USB设备扫描结果
export interface USBDevicesScanResult {
  cameras: USBDeviceInfo[]
  audios: USBDeviceInfo[]
}
