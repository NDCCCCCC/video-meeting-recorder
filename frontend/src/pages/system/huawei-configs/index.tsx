// 华为配置管理页面

import { useState, useEffect } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Modal,
  Form,
  message,
  Popconfirm,
  Tag,
  InputNumber,
  Tabs,
  Descriptions,
  Select,
  Divider,
} from 'antd'
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  CloudServerOutlined,
  LockOutlined,
  ScanOutlined,
  VideoCameraOutlined,
  AudioOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as huaweiConfigApi from '../../../api/huawei-config'
import type {
  HuaweiConfig,
  HuaweiConfigListParams,
  CreateHuaweiConfigRequest,
  UpdateHuaweiConfigRequest,
  USBDeviceInfo,
  TestStreamRequest,
} from '../../../types/huawei-config'

export default function HuaweiConfigManagement() {
  const [configs, setConfigs] = useState<HuaweiConfig[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [editingConfig, setEditingConfig] = useState<HuaweiConfig | null>(null)
  const [viewingConfig, setViewingConfig] = useState<HuaweiConfig | null>(null)
  const [form] = Form.useForm()

  // USB设备扫描相关状态
  const [scanningDevices, setScanningDevices] = useState(false)
  const [detectedCameras, setDetectedCameras] = useState<USBDeviceInfo[]>([])
  const [detectedAudios, setDetectedAudios] = useState<USBDeviceInfo[]>([])

  const [params, setParams] = useState<HuaweiConfigListParams>({
    page: 1,
    page_size: 20,
  })

  const loadConfigs = async () => {
    setLoading(true)
    try {
      const response = await huaweiConfigApi.getHuaweiConfigList(params)
      if (response.data) {
        setConfigs(response.data.items)
        setTotal(response.data.total)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载配置列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadConfigs()
  }, [params])

  const handleSearch = (value: string) => {
    setParams({ ...params, keyword: value, page: 1 })
  }

  const handleTableChange = (pagination: { current?: number; pageSize?: number }) => {
    setParams({
      ...params,
      page: pagination.current ?? 1,
      page_size: pagination.pageSize ?? 20,
    })
  }

  const openModal = (config: HuaweiConfig | null = null) => {
    setEditingConfig(config)
    if (config) {
      form.setFieldsValue({
        name: config.name,
        description: config.description,
        server: config.server,
        port: config.port,
        username: config.username,
        password: '', // 编辑模式下密码为空，表示不修改
        terminal_number: config.terminal_number,
        conference_number: config.conference_number,
        usb_camera_name: config.usb_camera_name,
        usb_camera_device: config.usb_camera_device,
        usb_audio_name: config.usb_audio_name,
        usb_audio_device: config.usb_audio_device,
        record_directory: config.record_directory,
        output_format: config.output_format,
        camera_backend: config.camera_backend,
        audio_backend: config.audio_backend,
        // 流媒体配置
        stream_protocol: config.stream_protocol,
        stream_url: config.stream_url,
        stream_username: config.stream_username,
        stream_enabled: config.stream_enabled || false,
      })
    } else {
      form.resetFields()
      form.setFieldsValue({ port: 22, output_format: 'mp4', stream_enabled: false })
    }
    setModalVisible(true)
    // 自动扫描 USB 设备，以便用户选择
    handleScanDevices()
  }

  const closeModal = () => {
    setModalVisible(false)
    setEditingConfig(null)
    form.resetFields()
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()

      if (editingConfig) {
        // 编辑模式：密码为空则不更新密码
        const req: UpdateHuaweiConfigRequest = {
          name: values.name,
          description: values.description,
          server: values.server,
          port: values.port,
          username: values.username,
          terminal_number: values.terminal_number,
          conference_number: values.conference_number,
          usb_camera_name: values.usb_camera_name,
          usb_camera_device: values.usb_camera_device,
          usb_audio_name: values.usb_audio_name || '',
          usb_audio_device: values.usb_audio_device,
          record_directory: values.record_directory,
          output_format: values.output_format,
          camera_backend: values.camera_backend,
          audio_backend: values.audio_backend,
          // 流媒体配置
          stream_protocol: values.stream_protocol,
          stream_url: values.stream_url,
          stream_username: values.stream_username,
          stream_password: values.stream_password,
          stream_enabled: values.stream_enabled,
        }
        // 只有在密码字段有值时才更新密码
        if (values.password && values.password.trim() !== '') {
          req.password = values.password
        }
        await huaweiConfigApi.updateHuaweiConfig(editingConfig.id, req)
        message.success('更新成功')
      } else {
        // 新建模式：密码必填
        const req: CreateHuaweiConfigRequest = {
          name: values.name,
          description: values.description,
          server: values.server,
          port: values.port,
          username: values.username,
          password: values.password,
          terminal_number: values.terminal_number,
          conference_number: values.conference_number,
          usb_camera_name: values.usb_camera_name,
          usb_camera_device: values.usb_camera_device,
          usb_audio_name: values.usb_audio_name || '',
          usb_audio_device: values.usb_audio_device,
          record_directory: values.record_directory,
          output_format: values.output_format,
          camera_backend: values.camera_backend,
          audio_backend: values.audio_backend,
          // 流媒体配置
          stream_protocol: values.stream_protocol,
          stream_url: values.stream_url,
          stream_username: values.stream_username,
          stream_password: values.stream_password,
          stream_enabled: values.stream_enabled,
        }
        await huaweiConfigApi.createHuaweiConfig(req)
        message.success('创建成功')
      }

      closeModal()
      loadConfigs()
    } catch (error) {
      // 改进错误提示，显示具体的验证错误信息
      const err = error as Error & { errorFields?: Array<{ name?: string[]; errors?: string[] }> }
      if (err.errorFields) {
        // Ant Design 表单验证错误
        const firstError = err.errorFields[0]
        const fieldName = firstError?.name?.[0] || '字段'
        const errorMessage = firstError?.errors?.[0] || '验证失败'
        message.error(`${fieldName}: ${errorMessage}`)
      } else if (err.message) {
        message.error(err.message)
      } else {
        message.error('操作失败，请检查表单填写是否正确')
      }
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await huaweiConfigApi.deleteHuaweiConfig(id)
      message.success('删除成功')
      loadConfigs()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }

  const viewDetail = (config: HuaweiConfig) => {
    setViewingConfig(config)
    setDetailVisible(true)
  }

  // 扫描USB设备
  const handleScanDevices = async () => {
    setScanningDevices(true)
    try {
      const response = await huaweiConfigApi.scanUSBDevices()
      if (response.data) {
        setDetectedCameras(response.data.cameras || [])
        setDetectedAudios(response.data.audios || [])
        message.success(`检测到 ${(response.data.cameras?.length || 0)} 个摄像头，${(response.data.audios?.length || 0)} 个音频设备`)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '扫描USB设备失败')
    } finally {
      setScanningDevices(false)
    }
  }

  // 选择摄像头
  const handleSelectCamera = (device: USBDeviceInfo) => {
    // 提取设备索引（用于兼容）
    let deviceIndex = device.device_id
    if (device.backend === 'dshow' && device.device_id.startsWith('video=')) {
      // video=0 -> 0
      deviceIndex = device.device_id.replace('video=', '')
    } else if (device.backend === 'v4l2' && device.device_id.startsWith('/dev/video')) {
      // /dev/video0 -> video0
      deviceIndex = device.device_id.replace('/dev/', '')
    }

    form.setFieldsValue({
      usb_camera_name: device.name,
      usb_camera_device: deviceIndex,  // 只存储索引，不包含前缀
    })
    message.info(`已选择摄像头: ${device.name}`)
  }

  // 选择音频设备
  const handleSelectAudio = (device: USBDeviceInfo) => {
    // 提取设备索引（用于兼容）
    let deviceIndex = device.device_id
    if (device.backend === 'dshow' && device.device_id.startsWith('audio=')) {
      // audio=0 -> 0
      deviceIndex = device.device_id.replace('audio=', '')
    } else if (device.backend === 'wasapi' || device.backend === 'dshow') {
      // Windows 音频设备使用完整名称
      deviceIndex = device.name
    } else if (device.backend === 'alsa' && device.device_id.startsWith('hw:')) {
      // hw:0,0 -> hw:0,0 (保持不变)
      deviceIndex = device.device_id
    }

    form.setFieldsValue({
      usb_audio_name: device.name,      // 实际设备名称
      usb_audio_device: deviceIndex,     // 设备索引
    })
    message.info(`已选择音频设备: ${device.name}`)
  }

  // 测试流媒体连接
  const handleTestStream = async () => {
    try {
      const values = await form.validateFields(['stream_protocol', 'stream_url'])
      const req: TestStreamRequest = {
        protocol: values.stream_protocol,
        url: values.stream_url,
        username: values.stream_username,
        password: values.stream_password,
      }
      await huaweiConfigApi.testStream(req)
      message.success('流媒体连接测试成功')
    } catch (error: any) {
      if (error?.errorFields) {
        message.error('请先填写协议和URL')
      } else {
        message.error(error instanceof Error ? error.message : '连接测试失败')
      }
    }
  }

  const columns: ColumnsType<HuaweiConfig> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '配置名称',
      dataIndex: 'name',
      width: 150,
      render: (name) => <Tag color="blue">{name}</Tag>,
    },
    {
      title: '服务器',
      dataIndex: 'server',
      width: 150,
    },
    {
      title: '端口',
      dataIndex: 'port',
      width: 80,
    },
    {
      title: '终端号',
      dataIndex: 'terminal_number',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      width: 80,
      render: (active, record) => (
        <Space direction="vertical" size={0}>
          <Tag color={active ? 'green' : 'red'}>{active ? '激活' : '禁用'}</Tag>
          {record.is_locked && <Tag color="orange" icon={<LockOutlined />}>已锁定</Tag>}
        </Space>
      ),
    },
    {
      title: '关联任务',
      dataIndex: 'video_recording_tasks',
      width: 100,
      render: (tasks) => tasks?.length || 0,
    },
    {
      title: '操作',
      key: 'action',
      width: 220,
      fixed: 'right' as const,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={() => viewDetail(record)}
          >
            详情
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openModal(record)}
            disabled={record.is_locked}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个配置吗？"
            onConfirm={() => handleDelete(record.id)}
            disabled={record.is_locked}
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={record.is_locked}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: '20px' }}>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>
          <CloudServerOutlined /> 华为配置管理
        </h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
          新建配置
        </Button>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <Space size="middle">
          <Input.Search
            placeholder="搜索配置名称、描述或服务器"
            allowClear
            style={{ width: 300 }}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Button icon={<ReloadOutlined />} onClick={loadConfigs}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={configs}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1000 }}
        pagination={{
          current: params.page,
          pageSize: params.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        onChange={handleTableChange}
      />

      {/* 新建/编辑配置对话框 */}
      <Modal
        title={editingConfig ? '编辑华为配置' : '新建华为配置'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={closeModal}
        width={800}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Tabs
            items={[
              {
                key: 'basic',
                label: '基本配置',
                children: (
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
                      <Input.TextArea placeholder="请输入配置描述" rows={3} />
                    </Form.Item>

                    <Space size="large" style={{ width: '100%' }}>
                      <Form.Item
                        name="server"
                        label="服务器地址"
                        rules={[{ required: true, message: '请输入服务器地址' }]}
                      >
                        <Input placeholder="例如: 192.168.1.100" />
                      </Form.Item>

                      <Form.Item
                        name="port"
                        label="端口"
                        rules={[{ required: true, message: '请输入端口' }]}
                      >
                        <InputNumber min={1} max={65535} style={{ width: 150 }} />
                      </Form.Item>
                    </Space>

                    <Space size="large" style={{ width: '100%' }}>
                      <Form.Item
                        name="username"
                        label="用户名"
                        rules={[{ required: true, message: '请输入用户名' }]}
                      >
                        <Input placeholder="请输入用户名" />
                      </Form.Item>

                      <Form.Item
                        name="password"
                        label={editingConfig ? "密码（留空不修改）" : "密码"}
                        rules={editingConfig ? [] : [{ required: true, message: '请输入密码' }]}
                      >
                        <Input.Password placeholder={editingConfig ? "留空则不修改密码" : "请输入密码"} />
                      </Form.Item>
                    </Space>

                    <Form.Item
                      name="terminal_number"
                      label="终端号"
                      rules={[{ required: true, message: '请输入终端号' }]}
                    >
                      <Input placeholder="请输入终端号" />
                    </Form.Item>
                  </>
                ),
              },
              {
                key: 'conference',
                label: '会议配置',
                children: (
                  <>
                    <Form.Item name="conference_number" label="会议号">
                      <Input placeholder="请输入会议号（可选）" />
                    </Form.Item>

                    <Form.Item name="output_format" label="输出格式">
                      <Input placeholder="例如: mp4" />
                    </Form.Item>

                    <Form.Item name="record_directory" label="录制目录">
                      <Input placeholder="请输入录制文件保存目录" />
                    </Form.Item>
                  </>
                ),
              },
              {
                key: 'usb',
                label: 'USB设备',
                children: (
                  <>
                    <div style={{ marginBottom: 16 }}>
                      <Button
                        type="primary"
                        icon={<ScanOutlined />}
                        onClick={handleScanDevices}
                        loading={scanningDevices}
                      >
                        自动检测USB设备
                      </Button>
                    </div>

                    <Divider>摄像头设备</Divider>

                    {detectedCameras.length > 0 && (
                      <Form.Item label="检测到的摄像头">
                        <Select
                          placeholder="选择检测到的摄像头设备"
                          onChange={(value) => {
                            const device = detectedCameras.find(d => d.device_id === value)
                            if (device) handleSelectCamera(device)
                          }}
                          options={detectedCameras.map(device => ({
                            label: (
                              <Space>
                                <VideoCameraOutlined />
                                {device.name}
                                <Tag color={device.status === 'available' ? 'green' : 'orange'}>{device.status}</Tag>
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

                    <Form.Item name="usb_camera_device" label="摄像头设备">
                      <Input placeholder="例如: /dev/video0" />
                    </Form.Item>

                    <Divider>音频设备</Divider>

                    {detectedAudios.length > 0 && (
                      <Form.Item label="检测到的音频设备">
                        <Select
                          placeholder="选择检测到的音频设备"
                          onChange={(value) => {
                            const device = detectedAudios.find(d => d.device_id === value)
                            if (device) handleSelectAudio(device)
                          }}
                          options={detectedAudios.map(device => ({
                            label: (
                              <Space>
                                <AudioOutlined />
                                {device.name}
                                <Tag color={device.status === 'available' ? 'green' : 'orange'}>{device.status}</Tag>
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
                      <Input placeholder="例如: hw:1,0" addonBefore={<AudioOutlined />} />
                    </Form.Item>
                  </>
                ),
              },
              {
                key: 'stream',
                label: '流媒体配置',
                children: (
                  <>
                    <Form.Item
                      name="stream_enabled"
                      label="启用流媒体录制"
                      valuePropName="checked"
                    >
                      <Select
                        options={[
                          { label: '禁用', value: false },
                          { label: '启用', value: true },
                        ]}
                      />
                    </Form.Item>

                    <Form.Item noStyle shouldUpdate={(prev, curr) => prev.stream_enabled !== curr.stream_enabled}>
                      {({ getFieldValue }) =>
                        getFieldValue('stream_enabled') ? (
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

                            <Form.Item label="连接测试">
                              <Button
                                type="default"
                                icon={<PlayCircleOutlined />}
                                onClick={handleTestStream}
                              >
                                测试连接
                              </Button>
                            </Form.Item>
                          </>
                        ) : null
                      }
                    </Form.Item>
                  </>
                ),
              },
            ]}
          />
        </Form>
      </Modal>

      {/* 配置详情对话框 */}
      <Modal
        title={<Space><CloudServerOutlined />配置详情 - {viewingConfig?.name}</Space>}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailVisible(false)}>
            关闭
          </Button>,
        ]}
        width={800}
      >
        {viewingConfig && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="配置ID">{viewingConfig.id}</Descriptions.Item>
            <Descriptions.Item label="配置名称">{viewingConfig.name}</Descriptions.Item>
            <Descriptions.Item label="服务器">{viewingConfig.server}</Descriptions.Item>
            <Descriptions.Item label="端口">{viewingConfig.port}</Descriptions.Item>
            <Descriptions.Item label="用户名">{viewingConfig.username}</Descriptions.Item>
            <Descriptions.Item label="终端号">{viewingConfig.terminal_number}</Descriptions.Item>
            <Descriptions.Item label="会议号">{viewingConfig.conference_number || '-'}</Descriptions.Item>
            <Descriptions.Item label="输出格式">{viewingConfig.output_format}</Descriptions.Item>
            <Descriptions.Item label="状态" span={2}>
              <Space>
                <Tag color={viewingConfig.is_active ? 'green' : 'red'}>
                  {viewingConfig.is_active ? '激活' : '禁用'}
                </Tag>
                {viewingConfig.is_locked && (
                  <Tag color="orange" icon={<LockOutlined />}>
                    已锁定 (任务ID: {viewingConfig.locked_by_task_id})
                  </Tag>
                )}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {viewingConfig.description || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="摄像头名称" span={2}>
              {viewingConfig.usb_camera_name || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="摄像头设备" span={2}>
              {viewingConfig.usb_camera_device || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="音频设备" span={2}>
              {viewingConfig.usb_audio_device || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="录制目录" span={2}>
              {viewingConfig.record_directory || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="关联任务数" span={2}>
              {viewingConfig.video_recording_tasks?.length || 0}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  )
}
