// 系统设置页面

import { useState, useEffect } from 'react'
import {
  Card,
  Form,
  Input,
  InputNumber,
  Button,
  Space,
  message,
  Divider,
  Alert,
  Popconfirm,
  Select,
} from 'antd'
import {
  SaveOutlined,
  ReloadOutlined,
  DeleteOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import * as systemApi from '../../../api/system'

const LOG_LEVEL_OPTIONS = [
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warn', value: 'warn' },
  { label: 'Error', value: 'error' },
]

const LOG_FORMAT_OPTIONS = [
  { label: 'JSON', value: 'json' },
  { label: 'Text', value: 'text' },
]

const LOG_OUTPUT_OPTIONS = [
  { label: '文件', value: 'file' },
  { label: '控制台', value: 'stdout' },
]

export default function Settings() {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)

  // 加载配置
  const loadConfig = async () => {
    setLoading(true)
    try {
      const response = await systemApi.getSystemConfig()
      if (response.data) {
        // 设置表单值
        form.setFieldsValue({
          recordings_path: response.data.storage.recordings_path,
          hls_path: response.data.storage.hls_path,
          temp_path: response.data.storage.temp_path,
          max_disk_usage: response.data.storage.max_disk_usage,
          ffmpeg_path: response.data.ffmpeg.path,
          ffprobe_path: response.data.ffmpeg.ffprobe_path,
          log_level: response.data.logging.level,
          log_format: response.data.logging.format,
          log_output: response.data.logging.output,
        })
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载配置失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConfig()
  }, [])

  // 保存配置
  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const response = await systemApi.updateSystemConfig(values)
      if (response.data) {
        message.success(response.data.message || '配置已更新')
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存配置失败')
    } finally {
      setSaving(false)
    }
  }

  // 重置配置
  const handleReset = () => {
    form.resetFields()
    message.info('配置已重置')
  }

  // 清空文件数据库
  const handleClearFiles = async () => {
    try {
      const response = await systemApi.clearFileDatabase()
      if (response.data) {
        message.success(response.data.message || '文件数据库已清空')
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '清空失败')
    }
  }

  return (
    <div style={{ padding: '20px' }}>
      <h2 style={{ marginBottom: '24px' }}>系统设置</h2>

      {loading ? (
        <div style={{ textAlign: 'center', padding: '100px' }}>加载中...</div>
      ) : (
        <Form form={form} layout="vertical">
          {/* 存储配置 */}
          <Card title="存储配置" style={{ marginBottom: '16px' }}>
            <Form.Item
              label="录制文件路径"
              name="recordings_path"
              rules={[{ required: true, message: '请输入录制文件路径' }]}
            >
              <Input placeholder="./data/recordings" />
            </Form.Item>

            <Form.Item
              label="HLS文件路径"
              name="hls_path"
              rules={[{ required: true, message: '请输入HLS文件路径' }]}
            >
              <Input placeholder="./data/hls" />
            </Form.Item>

            <Form.Item
              label="临时文件路径"
              name="temp_path"
              rules={[{ required: true, message: '请输入临时文件路径' }]}
            >
              <Input placeholder="./data/temp" />
            </Form.Item>

            <Form.Item
              label="磁盘使用限制 (%)"
              name="max_disk_usage"
              rules={[{ required: true, message: '请输入磁盘使用限制' }]}
            >
              <InputNumber min={1} max={100} style={{ width: '100%' }} />
            </Form.Item>
          </Card>

          {/* FFmpeg 配置 */}
          <Card title="FFmpeg 配置" style={{ marginBottom: '16px' }}>
            <Form.Item
              label="FFmpeg 路径"
              name="ffmpeg_path"
              rules={[{ required: true, message: '请输入FFmpeg路径' }]}
            >
              <Input placeholder="./bin/ffmpeg.exe" />
            </Form.Item>

            <Form.Item
              label="FFprobe 路径"
              name="ffprobe_path"
              rules={[{ required: true, message: '请输入FFprobe路径' }]}
            >
              <Input placeholder="./bin/ffprobe.exe" />
            </Form.Item>

            <Alert
              title="FFmpeg 路径修改后需要重启服务才能生效"
              type="info"
              showIcon
              style={{ marginTop: '12px' }}
            />
          </Card>

          {/* 日志配置 */}
          <Card title="日志配置" style={{ marginBottom: '16px' }}>
            <Form.Item
              label="日志级别"
              name="log_level"
              rules={[{ required: true, message: '请选择日志级别' }]}
            >
              <Select options={LOG_LEVEL_OPTIONS} />
            </Form.Item>

            <Form.Item
              label="日志格式"
              name="log_format"
              rules={[{ required: true, message: '请选择日志格式' }]}
            >
              <Select options={LOG_FORMAT_OPTIONS} />
            </Form.Item>

            <Form.Item
              label="日志输出"
              name="log_output"
              rules={[{ required: true, message: '请选择日志输出' }]}
            >
              <Select options={LOG_OUTPUT_OPTIONS} />
            </Form.Item>
          </Card>

          {/* 危险操作 */}
          <Card title="危险操作" style={{ marginBottom: '24px' }}>
            <Alert
              title="清空文件数据库将删除所有文件记录，此操作不可恢复！"
              type="warning"
              showIcon
              style={{ marginBottom: '16px' }}
            />
            <Popconfirm
              title="确定要清空所有文件记录吗？此操作不可恢复！"
              onConfirm={handleClearFiles}
              okText="确定"
              cancelText="取消"
              okType="danger"
              icon={<ExclamationCircleOutlined style={{ color: 'red' }} />}
            >
              <Button type="primary" danger icon={<DeleteOutlined />}>
                清空所有文件记录
              </Button>
            </Popconfirm>
          </Card>

          {/* 操作按钮 */}
          <Divider />
          <Space>
            <Button type="primary" icon={<SaveOutlined />} onClick={handleSave} loading={saving}>
              保存配置
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
          </Space>
        </Form>
      )}
    </div>
  )
}
