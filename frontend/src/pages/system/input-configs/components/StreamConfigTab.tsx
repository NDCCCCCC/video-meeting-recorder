// 流媒体配置 Tab
// 从 input-configs 表单 Modal 抽出。stream_enabled 的联动用 antd Form shouldUpdate（form context 内部完成）。

import { Form, Input, Select, Switch, Space } from 'antd'
import type { ConfigType } from '../../../../types/input-config'

export interface StreamConfigTabProps {
  configType: ConfigType
}

export function StreamConfigTab({ configType }: StreamConfigTabProps) {
  if (configType !== 'stream') {
    return (
      <div style={{ color: '#999', textAlign: 'center', padding: '40px' }}>
        请先选择"流媒体录制"类型
      </div>
    )
  }

  return (
    <>
      <Form.Item name="stream_enabled" label="启用流媒体录制" valuePropName="checked">
        <Switch checkedChildren="启用" unCheckedChildren="禁用" />
      </Form.Item>

      <Form.Item
        noStyle
        shouldUpdate={(prev, curr) => prev.stream_enabled !== curr.stream_enabled}
      >
        {({ getFieldValue }) =>
          getFieldValue('stream_enabled') !== false ? (
            <>
              <Form.Item
                name="stream_protocol"
                label="流媒体协议"
                rules={[{ required: true, message: '请选择流媒体协议' }]}
              >
                <Select
                  placeholder="请选择协议类型"
                  options={[
                    { label: 'RTMP', value: 'rtmp' },
                    { label: 'RTSP', value: 'rtsp' },
                    { label: 'SRT', value: 'srt' },
                    { label: 'HLS', value: 'hls' },
                  ]}
                />
              </Form.Item>

              <Form.Item
                name="stream_url"
                label="流媒体URL"
                rules={[
                  { required: true, message: '请输入流媒体URL' },
                  { type: 'url', message: '请输入有效的URL' },
                ]}
              >
                <Input placeholder="例如: rtmp://example.com/live/stream" />
              </Form.Item>

              <Space size="large" style={{ width: '100%' }}>
                <Form.Item name="stream_username" label="用户名（可选）">
                  <Input placeholder="请输入用户名" />
                </Form.Item>

                <Form.Item name="stream_password" label="密码（可选）">
                  <Input.Password placeholder="请输入密码" />
                </Form.Item>
              </Space>
            </>
          ) : null
        }
      </Form.Item>
    </>
  )
}
