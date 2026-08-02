// USB 设备配置 Tab（自动检测 + 摄像头/音频字段）
// 从 input-configs 表单 Modal 抽出。扫描/选择能力由 useUSBDeviceScan hook 通过 props 注入。

import { Form, Input, Select, Button, Divider, Space, Tag } from 'antd'
import { ScanOutlined, VideoCameraOutlined, AudioOutlined } from '@ant-design/icons'
import type { USBDeviceInfo, ConfigType } from '../../../../types/input-config'

export interface USBConfigTabProps {
  configType: ConfigType
  scanningDevices: boolean
  detectedCameras: USBDeviceInfo[]
  detectedAudios: USBDeviceInfo[]
  onScan: () => void
  onSelectCamera: (device: USBDeviceInfo) => void
  onSelectAudio: (device: USBDeviceInfo) => void
}

export function USBConfigTab({
  configType,
  scanningDevices,
  detectedCameras,
  detectedAudios,
  onScan,
  onSelectCamera,
  onSelectAudio,
}: USBConfigTabProps) {
  if (configType !== 'usb') {
    return (
      <div style={{ color: '#999', textAlign: 'center', padding: '40px' }}>
        请先选择"USB设备直录"类型
      </div>
    )
  }

  return (
    <>
      <div style={{ marginBottom: 16 }}>
        <Button type="primary" icon={<ScanOutlined />} onClick={onScan} loading={scanningDevices}>
          自动检测USB设备
        </Button>
      </div>

      <Divider>摄像头设备</Divider>

      {detectedCameras.length > 0 && (
        <Form.Item label="检测到的摄像头">
          <Select
            placeholder="选择检测到的摄像头设备"
            onChange={(value) => {
              const device = detectedCameras.find((d) => d.device_id === value)
              if (device) onSelectCamera(device)
            }}
            options={detectedCameras.map((device) => ({
              label: (
                <Space>
                  <VideoCameraOutlined />
                  {device.name}
                  <Tag color={device.status === 'available' ? 'green' : 'orange'}>{device.status}</Tag>
                  <span style={{ color: '#999', fontSize: '12px' }}>({device.backend})</span>
                </Space>
              ),
              value: device.device_id,
            }))}
          />
        </Form.Item>
      )}

      <Form.Item name="usb_camera_name" label="摄像头名称">
        <Input placeholder="请输入USB摄像头名称" addonBefore={<VideoCameraOutlined />} />
      </Form.Item>

      <Form.Item
        name="usb_camera_device"
        label="摄像头设备"
        rules={[{ required: configType === 'usb', message: '请选择USB摄像头设备' }]}
      >
        <Input placeholder="例如: /dev/video0 或 video0" />
      </Form.Item>

      <Form.Item name="camera_backend" label="摄像头后端">
        <Select
          placeholder="选择后端类型"
          options={[
            { label: 'V4L2 (Linux)', value: 'v4l2' },
            { label: 'DirectShow (Windows)', value: 'dshow' },
            { label: 'AVFoundation (macOS)', value: 'avfoundation' },
          ]}
        />
      </Form.Item>

      <Divider>音频设备</Divider>

      {detectedAudios.length > 0 && (
        <Form.Item label="检测到的音频设备">
          <Select
            placeholder="选择检测到的音频设备"
            onChange={(value) => {
              const device = detectedAudios.find((d) => d.device_id === value)
              if (device) onSelectAudio(device)
            }}
            options={detectedAudios.map((device) => ({
              label: (
                <Space>
                  <AudioOutlined />
                  {device.name}
                  <Tag color={device.status === 'available' ? 'green' : 'orange'}>{device.status}</Tag>
                  <span style={{ color: '#999', fontSize: '12px' }}>({device.backend})</span>
                </Space>
              ),
              value: device.device_id,
            }))}
          />
        </Form.Item>
      )}

      <Form.Item name="usb_audio_name" label="音频设备名称">
        <Input placeholder="请输入USB音频设备名称" addonBefore={<AudioOutlined />} />
      </Form.Item>

      <Form.Item name="usb_audio_device" label="音频设备">
        <Input placeholder="例如: hw:1,0 或 audio=0" />
      </Form.Item>

      <Form.Item name="audio_backend" label="音频后端">
        <Select
          placeholder="选择后端类型"
          options={[
            { label: 'ALSA (Linux)', value: 'alsa' },
            { label: 'DirectShow (Windows)', value: 'dshow' },
            { label: 'CoreAudio (macOS)', value: 'coreaudio' },
          ]}
        />
      </Form.Item>
    </>
  )
}
