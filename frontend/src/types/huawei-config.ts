// 华为配置相关类型定义

// USB设备信息
export interface USBDeviceInfo {
  type: string // "camera" | "audio"
  name: string // 设备名称
  device_id: string // 设备ID (/dev/video0, hw:1,0)
  status: string // "available" | "in_use" | "error"
  backend: string // "v4l2" | "alsa"
}

// USB设备扫描结果
export interface USBDevicesScanResult {
  cameras?: USBDeviceInfo[]
  audios?: USBDeviceInfo[]
}

export interface HuaweiConfig {
  id: number
  name: string
  description: string
  server: string
  port: number
  username: string
  password: string
  terminal_number: string
  conference_number: string | null
  usb_camera_name: string | null
  usb_camera_device: string | null
  usb_audio_name: string | null
  usb_audio_device: string | null
  record_directory: string | null
  output_format: string
  is_active: boolean
  is_locked: boolean
  locked_by_task_id: number | null
  created_at: string
  updated_at: string
  video_recording_tasks?: Array<{
    id: number
    name: string
    status: string
  }>
}

export interface HuaweiConfigListParams {
  page?: number
  page_size?: number
  keyword?: string
  is_active?: boolean
}

export interface HuaweiConfigListResponse {
  total: number
  items: HuaweiConfig[]
}

export interface CreateHuaweiConfigRequest {
  name: string
  description?: string
  server: string
  port: number
  username: string
  password: string
  terminal_number: string
  conference_number?: string
  usb_camera_name?: string
  usb_camera_device?: string
  usb_audio_name?: string
  usb_audio_device?: string
  record_directory?: string
  output_format?: string
}

export interface UpdateHuaweiConfigRequest {
  name?: string
  description?: string
  server?: string
  port?: number
  username?: string
  password?: string
  terminal_number?: string
  conference_number?: string
  usb_camera_name?: string
  usb_camera_device?: string
  usb_audio_name?: string
  usb_audio_device?: string
  record_directory?: string
  output_format?: string
  is_active?: boolean
}
