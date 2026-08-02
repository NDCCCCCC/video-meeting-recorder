// 输入配置详情对话框（纯展示）
// 从 pages/system/input-configs/index.tsx 抽出。configTypeMap 与父级 columns 共享，通过 prop 传入。

import type { ReactNode } from 'react'
import { Modal, Button, Space, Descriptions, Tag } from 'antd'
import { SettingOutlined, LockOutlined } from '@ant-design/icons'
import type { InputConfig } from '../../../../types/input-config'

export interface ConfigTypeMeta {
  text: string
  color: string
  icon: ReactNode
}

export interface InputConfigDetailModalProps {
  open: boolean
  config: InputConfig | null
  configTypeMap: Record<string, ConfigTypeMeta>
  onClose: () => void
}

export function InputConfigDetailModal({
  open,
  config,
  configTypeMap,
  onClose,
}: InputConfigDetailModalProps) {
  return (
    <Modal
      title={
        <Space>
          <SettingOutlined />
          配置详情 - {config?.name}
        </Space>
      }
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
      ]}
      width={900}
    >
      {config && (
        <Descriptions column={2} size="small">
          <Descriptions.Item label="配置ID">{config.id}</Descriptions.Item>
          <Descriptions.Item label="配置名称">{config.name}</Descriptions.Item>
          <Descriptions.Item label="配置类型" span={2}>
            <Tag color={configTypeMap[config.config_type]?.color} icon={configTypeMap[config.config_type]?.icon}>
              {configTypeMap[config.config_type]?.text || config.config_type}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="华为终端控制" span={2}>
            {config.config_type === 'usb' ? (
              <Tag color={config.huawei_enabled ? 'green' : 'orange'}>
                {config.huawei_enabled ? '启用自动控制' : '手动模式'}
              </Tag>
            ) : (
              '-'
            )}
          </Descriptions.Item>
          <Descriptions.Item label="状态" span={2}>
            <Space>
              <Tag color={config.is_active ? 'green' : 'red'}>{config.is_active ? '激活' : '禁用'}</Tag>
              {config.is_locked && (
                <Tag color="orange" icon={<LockOutlined />}>
                  已锁定
                </Tag>
              )}
            </Space>
          </Descriptions.Item>
          <Descriptions.Item label="描述" span={2}>
            {config.description || '-'}
          </Descriptions.Item>

          {/* 华为终端字段 */}
          {config.config_type === 'usb' && config.huawei_enabled && (
            <>
              <Descriptions.Item label="服务器">{config.server || '-'}</Descriptions.Item>
              <Descriptions.Item label="端口">{config.port || '-'}</Descriptions.Item>
              <Descriptions.Item label="用户名">{config.username || '-'}</Descriptions.Item>
              <Descriptions.Item label="终端号">{config.terminal_number || '-'}</Descriptions.Item>
              <Descriptions.Item label="会议号" span={2}>
                {config.conference_number || '-'}
              </Descriptions.Item>
            </>
          )}

          {/* USB设备字段 */}
          {config.config_type === 'usb' && (
            <>
              <Descriptions.Item label="摄像头名称" span={2}>
                {config.usb_camera_name || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="摄像头设备" span={2}>
                {config.usb_camera_device || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="摄像头后端" span={2}>
                {config.camera_backend || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="音频设备名称" span={2}>
                {config.usb_audio_name || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="音频设备" span={2}>
                {config.usb_audio_device || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="音频后端" span={2}>
                {config.audio_backend || '-'}
              </Descriptions.Item>
            </>
          )}

          {/* 流媒体字段 */}
          {config.config_type === 'stream' && (
            <>
              <Descriptions.Item label="流媒体协议" span={2}>
                {config.stream_protocol ? (
                  <Tag color="blue">{config.stream_protocol.toUpperCase()}</Tag>
                ) : (
                  '-'
                )}
              </Descriptions.Item>
              <Descriptions.Item label="流媒体URL" span={2}>
                {config.stream_url || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="流媒体状态" span={2}>
                <Tag color={config.stream_enabled ? 'green' : 'red'}>
                  {config.stream_enabled ? '启用' : '禁用'}
                </Tag>
              </Descriptions.Item>
            </>
          )}

          <Descriptions.Item label="输出格式" span={2}>
            {config.output_format || '-'}
          </Descriptions.Item>
          <Descriptions.Item label="关联任务数" span={2}>
            {config.video_recording_tasks?.length || 0}
          </Descriptions.Item>
          <Descriptions.Item label="创建时间">{config.created_at}</Descriptions.Item>
          <Descriptions.Item label="更新时间">{config.updated_at}</Descriptions.Item>
        </Descriptions>
      )}
    </Modal>
  )
}
