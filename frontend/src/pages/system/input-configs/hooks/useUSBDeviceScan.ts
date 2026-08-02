// USB 设备扫描 hook
// 封装 input-configs 表单中"自动检测 USB 设备 + 选择摄像头/音频"的状态与逻辑。
// selectCamera/selectAudio 通过传入的 form 实例写回字段值（device_id 解析复用 ../utils）。

import { useState, useCallback } from 'react'
import { message, type FormInstance } from 'antd'
import { scanUSBDevices } from '../../../../api/input-config'
import { resolveCameraDeviceId, resolveAudioDeviceId } from '../utils'
import type { USBDeviceInfo } from '../../../../types/input-config'

export function useUSBDeviceScan(form: FormInstance) {
  const [scanningDevices, setScanningDevices] = useState(false)
  const [detectedCameras, setDetectedCameras] = useState<USBDeviceInfo[]>([])
  const [detectedAudios, setDetectedAudios] = useState<USBDeviceInfo[]>([])

  const scanDevices = useCallback(async () => {
    setScanningDevices(true)
    try {
      const response = await scanUSBDevices()
      if (response.data) {
        setDetectedCameras(response.data.cameras || [])
        setDetectedAudios(response.data.audios || [])
        message.success(
          `检测到 ${response.data.cameras?.length || 0} 个摄像头，${response.data.audios?.length || 0} 个音频设备`
        )
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '扫描USB设备失败')
    } finally {
      setScanningDevices(false)
    }
  }, [])

  const selectCamera = useCallback(
    (device: USBDeviceInfo) => {
      form.setFieldsValue({
        usb_camera_name: device.name,
        usb_camera_device: resolveCameraDeviceId(device),
        camera_backend: device.backend,
      })
      message.info(`已选择摄像头: ${device.name}`)
    },
    [form]
  )

  const selectAudio = useCallback(
    (device: USBDeviceInfo) => {
      form.setFieldsValue({
        usb_audio_name: device.name,
        usb_audio_device: resolveAudioDeviceId(device),
        audio_backend: device.backend,
      })
      message.info(`已选择音频设备: ${device.name}`)
    },
    [form]
  )

  // 关闭弹窗时清空已检测设备
  const clearDevices = useCallback(() => {
    setDetectedCameras([])
    setDetectedAudios([])
  }, [])

  return {
    scanningDevices,
    detectedCameras,
    detectedAudios,
    scanDevices,
    selectCamera,
    selectAudio,
    clearDevices,
  }
}
