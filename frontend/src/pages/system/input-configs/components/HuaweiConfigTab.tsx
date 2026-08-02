// 华为终端配置 Tab
// 从 input-configs 表单 Modal 抽出。未启用华为控制时显示占位提示。

import { Form, Input, InputNumber, Space } from 'antd'
import type { InputConfig } from '../../../../types/input-config'

export interface HuaweiConfigTabProps {
  huaweiEnabled: boolean
  editingConfig: InputConfig | null
}

export function HuaweiConfigTab({ huaweiEnabled, editingConfig }: HuaweiConfigTabProps) {
  if (!huaweiEnabled) {
    return <div style={{ color: '#999', textAlign: 'center', padding: '40px' }}>请先启用华为终端控制</div>
  }

  return (
    <>
      <Space size="large" style={{ width: '100%' }}>
        <Form.Item name="server" label="服务器地址" rules={[{ required: true, message: '请输入服务器地址' }]}>
          <Input placeholder="例如: 192.168.1.100" />
        </Form.Item>

        <Form.Item name="port" label="端口" rules={[{ required: true, message: '请输入端口' }]}>
          <InputNumber min={1} max={65535} style={{ width: 150 }} />
        </Form.Item>
      </Space>

      <Space size="large" style={{ width: '100%' }}>
        <Form.Item name="username" label="用户名" rules={[{ required: true, message: '请输入用户名' }]}>
          <Input placeholder="请输入用户名" />
        </Form.Item>

        <Form.Item
          name="password"
          label={editingConfig ? '密码（留空不修改）' : '密码'}
          rules={editingConfig ? [] : [{ required: true, message: '请输入密码' }]}
        >
          <Input.Password placeholder={editingConfig ? '留空则不修改密码' : '请输入密码'} />
        </Form.Item>
      </Space>

      <Space size="large" style={{ width: '100%' }}>
        <Form.Item name="terminal_number" label="终端号" rules={[{ required: true, message: '请输入终端号' }]}>
          <Input placeholder="请输入终端号" />
        </Form.Item>

        <Form.Item name="conference_number" label="会议号">
          <Input placeholder="请输入会议号（可选）" />
        </Form.Item>
      </Space>
    </>
  )
}
