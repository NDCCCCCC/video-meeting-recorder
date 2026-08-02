// 基本配置 Tab（配置名称/描述/类型/华为开关/输出格式）
// 从 input-configs 表单 Modal 抽出。Form.Items 通过 antd Form context 绑定父级 form。

import type { FormInstance } from 'antd'
import { Form, Input, Select, Switch, Space } from 'antd'
import type { ConfigType } from '../../../../types/input-config'

export interface BasicConfigTabProps {
  form: FormInstance
  configType: ConfigType
  setConfigType: (value: ConfigType) => void
  huaweiEnabled: boolean
  setHuaweiEnabled: (value: boolean) => void
}

export function BasicConfigTab({
  form,
  configType,
  setConfigType,
  huaweiEnabled,
  setHuaweiEnabled,
}: BasicConfigTabProps) {
  return (
    <>
      <Form.Item
        name="name"
        label="配置名称"
        rules={[
          { required: true, message: '请输入配置名称' },
          { max: 100, message: '配置名称最多100个字符' },
        ]}
      >
        <Input placeholder="请输入配置名称" />
      </Form.Item>

      <Form.Item name="description" label="描述">
        <Input.TextArea placeholder="请输入配置描述" rows={2} />
      </Form.Item>

      {/* 配置类型选择器 - 这是关键的新字段 */}
      <Form.Item label="配置类型" required>
        <Select
          value={configType}
          onChange={(value) => {
            setConfigType(value)
            // 重置依赖字段
            if (value !== 'usb') {
              form.setFieldsValue({
                usb_camera_device: undefined,
                usb_camera_name: undefined,
                usb_audio_device: undefined,
                usb_audio_name: undefined,
                camera_backend: undefined,
                audio_backend: undefined,
              })
            }
            if (value !== 'stream') {
              form.setFieldsValue({
                stream_url: undefined,
                stream_protocol: undefined,
                stream_username: undefined,
                stream_password: undefined,
                stream_enabled: false,
              })
            }
          }}
          options={[
            { label: 'USB设备直录', value: 'usb' },
            { label: '流媒体录制', value: 'stream' },
          ]}
        />
      </Form.Item>

      {/* 华为终端控制开关 - 所有配置类型都可以启用 */}
      <Form.Item label="启用华为终端控制">
        <Space>
          <Switch
            checked={huaweiEnabled}
            onChange={(checked) => {
              setHuaweiEnabled(checked)
              form.setFieldsValue({ huawei_enabled: checked })
              if (!checked) {
                // 禁用时清空华为字段
                form.setFieldsValue({
                  server: undefined,
                  port: undefined,
                  username: undefined,
                  password: undefined,
                  terminal_number: undefined,
                  conference_number: undefined,
                })
              }
            }}
            checkedChildren="启用"
            unCheckedChildren="禁用"
          />
          <span style={{ color: '#666', fontSize: '12px' }}>
            {huaweiEnabled ? '启用后将自动控制华为终端（可选功能）' : '不使用华为终端控制'}
          </span>
        </Space>
      </Form.Item>

      {/* 输出格式 - 所有类型通用 */}
      <Form.Item name="output_format" label="输出格式" initialValue="mp4">
        <Select
          options={[
            { label: 'MP4', value: 'mp4' },
            { label: 'MKV', value: 'mkv' },
            { label: 'AVI', value: 'avi' },
          ]}
        />
      </Form.Item>
    </>
  )
}
