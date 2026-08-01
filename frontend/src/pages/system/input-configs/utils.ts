// 输入配置管理页面工具函数

import type {
  USBDeviceInfo,
  ConfigType,
  CreateInputConfigRequest,
  UpdateInputConfigRequest,
} from '../../../types/input-config'

/**
 * 解析摄像头设备 ID：把后端原始 device_id 转成兼容的索引值
 * - dshow: "video=0" -> "0"
 * - v4l2:  "/dev/video0" -> "video0"
 * - 其他后端：保持原值
 */
export function resolveCameraDeviceId(device: USBDeviceInfo): string {
  if (device.backend === 'dshow' && device.device_id.startsWith('video=')) {
    return device.device_id.replace('video=', '')
  }
  if (device.backend === 'v4l2' && device.device_id.startsWith('/dev/video')) {
    return device.device_id.replace('/dev/', '')
  }
  return device.device_id
}

/**
 * 解析音频设备 ID
 * - dshow + "audio=" 前缀：去掉前缀
 * - wasapi / dshow（无 audio= 前缀）：使用设备名称
 * - alsa（hw:0,0 等）及其他：保持原值
 */
export function resolveAudioDeviceId(device: USBDeviceInfo): string {
  if (device.backend === 'dshow' && device.device_id.startsWith('audio=')) {
    return device.device_id.replace('audio=', '')
  }
  if (device.backend === 'wasapi' || device.backend === 'dshow') {
    return device.name
  }
  return device.device_id
}

/**
 * 根据表单值组装"新建输入配置"请求体
 * config_type / huawei_enabled 来自页面 state（驱动 Tab 显隐），其余取表单值
 */
export function buildCreatePayload(
  values: CreateInputConfigRequest,
  configType: ConfigType,
  huaweiEnabled: boolean
): CreateInputConfigRequest {
  return {
    name: values.name,
    description: values.description,
    config_type: configType,
    huawei_enabled: huaweiEnabled,
    // 华为字段
    server: values.server,
    port: values.port,
    username: values.username,
    password: values.password,
    terminal_number: values.terminal_number,
    conference_number: values.conference_number,
    // USB字段
    camera_backend: values.camera_backend,
    usb_camera_name: values.usb_camera_name,
    usb_camera_device: values.usb_camera_device,
    audio_backend: values.audio_backend,
    usb_audio_name: values.usb_audio_name,
    usb_audio_device: values.usb_audio_device,
    // 流媒体字段
    stream_protocol: values.stream_protocol,
    stream_url: values.stream_url,
    stream_username: values.stream_username,
    stream_password: values.stream_password,
    stream_enabled: values.stream_enabled,
    // 录制配置
    output_format: values.output_format,
  }
}

/**
 * 根据表单值组装"更新输入配置"请求体
 * 仅当密码字段有值时才携带 password（空表示不修改）
 */
export function buildUpdatePayload(values: CreateInputConfigRequest): UpdateInputConfigRequest {
  const payload: UpdateInputConfigRequest = {
    name: values.name,
    description: values.description,
    huawei_enabled: values.huawei_enabled,
    // 华为字段
    server: values.server,
    port: values.port,
    username: values.username,
    terminal_number: values.terminal_number,
    conference_number: values.conference_number,
    // USB字段
    camera_backend: values.camera_backend,
    usb_camera_name: values.usb_camera_name,
    usb_camera_device: values.usb_camera_device,
    audio_backend: values.audio_backend,
    usb_audio_name: values.usb_audio_name,
    usb_audio_device: values.usb_audio_device,
    // 流媒体字段
    stream_protocol: values.stream_protocol,
    stream_url: values.stream_url,
    stream_username: values.stream_username,
    stream_password: values.stream_password,
    stream_enabled: values.stream_enabled,
    // 录制配置
    output_format: values.output_format,
  }
  // 只有在密码字段有值时才更新密码
  if (values.password && values.password.trim() !== '') {
    payload.password = values.password
  }
  return payload
}
