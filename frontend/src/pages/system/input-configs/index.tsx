// 输入配置管理页面

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
  Tabs,
  Descriptions,
  Select,
  Divider,
  Switch,
  InputNumber,
} from 'antd'
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SettingOutlined,
  LockOutlined,
  ScanOutlined,
  VideoCameraOutlined,
  AudioOutlined,
  PlayCircleOutlined,
  CloudServerOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as inputConfigApi from '../../../api/input-config'
import type {
  InputConfig,
  InputConfigListParams,
  CreateInputConfigRequest,
  UpdateInputConfigRequest,
  USBDeviceInfo,
  ConfigType,
} from '../../../types/input-config'

export default function InputConfigManagement() {
  const [configs, setConfigs] = useState<InputConfig[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [editingConfig, setEditingConfig] = useState<InputConfig | null>(null)
  const [viewingConfig, setViewingConfig] = useState<InputConfig | null>(null)
  const [form] = Form.useForm()

  // 表单状态管理
  const [configType, setConfigType] = useState<ConfigType>('usb')
  const [huaweiEnabled, setHuaweiEnabled] = useState(false)

  // USB设备扫描相关状态
  const [scanningDevices, setScanningDevices] = useState(false)
  const [detectedCameras, setDetectedCameras] = useState<USBDeviceInfo[]>([])
  const [detectedAudios, setDetectedAudios] = useState<USBDeviceInfo[]>([])

  const [params, setParams] = useState<InputConfigListParams>({
    page: 1,
    page_size: 20,
  })

  const loadConfigs = async () => {
    setLoading(true)
    try {
      const response = await inputConfigApi.getInputConfigList(params)
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

  const openModal = (config: InputConfig | null = null) => {
    setEditingConfig(config)
    if (config) {
      setConfigType(config.config_type)
      setHuaweiEnabled(config.huawei_enabled)
      form.setFieldsValue({
        name: config.name,
        description: config.description,
        config_type: config.config_type,
        huawei_enabled: config.huawei_enabled,
        server: config.server,
        port: config.port,
        username: config.username,
        password: '', // 编辑模式下密码为空，表示不修改
        terminal_number: config.terminal_number,
        conference_number: config.conference_number,
        camera_backend: config.camera_backend,
        usb_camera_name: config.usb_camera_name,
        usb_camera_device: config.usb_camera_device,
        audio_backend: config.audio_backend,
        usb_audio_name: config.usb_audio_name,
        usb_audio_device: config.usb_audio_device,
        stream_protocol: config.stream_protocol,
        stream_url: config.stream_url,
        stream_username: config.stream_username,
        stream_enabled: config.stream_enabled || false,
        output_format: config.output_format || 'mp4',
      })
    } else {
      setConfigType('usb')
      setHuaweiEnabled(false)
      form.resetFields()
      form.setFieldsValue({
        config_type: 'usb',
        huawei_enabled: false,
        output_format: 'mp4',
        stream_enabled: false,
      })
    }
    setModalVisible(true)
    // 自动扫描 USB 设备，以便用户选择
    handleScanDevices()
  }

  const closeModal = () => {
    setModalVisible(false)
    setEditingConfig(null)
    form.resetFields()
    setDetectedCameras([])
    setDetectedAudios([])
    setConfigType('usb')
    setHuaweiEnabled(false)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()

      if (editingConfig) {
        // 编辑模式：密码为空则不更新密码
        const req: UpdateInputConfigRequest = {
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
          req.password = values.password
        }
        await inputConfigApi.updateInputConfig(editingConfig.id, req)
        message.success('更新成功')
      } else {
        // 新建模式：根据配置类型决定必填字段
        const req: CreateInputConfigRequest = {
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
        await inputConfigApi.createInputConfig(req)
        message.success('创建成功')
      }

      closeModal()
      loadConfigs()
    } catch (error) {
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
      await inputConfigApi.deleteInputConfig(id)
      message.success('删除成功')
      loadConfigs()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }

  const viewDetail = (config: InputConfig) => {
    setViewingConfig(config)
    setDetailVisible(true)
  }

  // 扫描USB设备
  const handleScanDevices = async () => {
    setScanningDevices(true)
    try {
      const response = await inputConfigApi.scanUSBDevices()
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
      usb_camera_device: deviceIndex,
      camera_backend: device.backend,
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
      usb_audio_name: device.name,
      usb_audio_device: deviceIndex,
      audio_backend: device.backend,
    })
    message.info(`已选择音频设备: ${device.name}`)
  }

  // 配置类型显示映射
  const configTypeMap = {
    usb: { text: 'USB直录', color: 'green', icon: <VideoCameraOutlined /> },
    stream: { text: '流媒体', color: 'orange', icon: <PlayCircleOutlined /> },
  }

  const columns: ColumnsType<InputConfig> = [
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
      title: '配置类型',
      dataIndex: 'config_type',
      width: 120,
      render: (type: ConfigType) => {
        const { text, color, icon } = configTypeMap[type] || { text: type, color: 'default', icon: null }
        return (
          <Tag color={color} icon={icon}>
            {text}
          </Tag>
        )
      },
    },
    {
      title: '描述',
      dataIndex: 'description',
      ellipsis: true,
      width: 200,
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      width: 100,
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
      title: '创建时间',
      dataIndex: 'created_at',
      width: 180,
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
          <SettingOutlined /> 输入配置管理
        </h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
          新建配置
        </Button>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <Space size="middle">
          <Input.Search
            placeholder="搜索配置名称、描述"
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
        scroll={{ x: 1200 }}
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
        title={editingConfig ? '编辑输入配置' : '新建输入配置'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={closeModal}
        width={900}
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
                      <Input.TextArea placeholder="请输入配置描述" rows={2} />
                    </Form.Item>

                    {/* 配置类型选择器 - 这是关键的新字段 */}
                    <Form.Item
                      label="配置类型"
                      required
                    >
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
                    <Form.Item
                      label="启用华为终端控制"
                    >
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
                    <Form.Item
                      name="output_format"
                      label="输出格式"
                      initialValue="mp4"
                    >
                      <Select
                        options={[
                          { label: 'MP4', value: 'mp4' },
                          { label: 'MKV', value: 'mkv' },
                          { label: 'AVI', value: 'avi' },
                        ]}
                      />
                    </Form.Item>
                  </>
                ),
              },
              {
                key: 'huawei',
                label: '华为终端配置',
                disabled: !huaweiEnabled,
                children: (
                  <>
                    {huaweiEnabled ? (
                      <>
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

                        <Space size="large" style={{ width: '100%' }}>
                          <Form.Item
                            name="terminal_number"
                            label="终端号"
                            rules={[{ required: true, message: '请输入终端号' }]}
                          >
                            <Input placeholder="请输入终端号" />
                          </Form.Item>

                          <Form.Item name="conference_number" label="会议号">
                            <Input placeholder="请输入会议号（可选）" />
                          </Form.Item>
                        </Space>
                      </>
                    ) : (
                      <div style={{ color: '#999', textAlign: 'center', padding: '40px' }}>
                        请先启用华为终端控制
                      </div>
                    )}
                  </>
                ),
              },
              {
                key: 'usb',
                label: 'USB设备配置',
                disabled: configType !== 'usb',
                children: (
                  <>
                    {configType === 'usb' ? (
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
                                    <Tag color={device.status === 'available' ? 'green' : 'orange'}>
                                      {device.status}
                                    </Tag>
                                    <span style={{ color: '#999', fontSize: '12px' }}>
                                      ({device.backend})
                                    </span>
                                  </Space>
                                ),
                                value: device.device_id,
                              }))}
                            />
                          </Form.Item>
                        )}

                        <Form.Item
                          name="usb_camera_name"
                          label="摄像头名称"
                        >
                          <Input placeholder="请输入USB摄像头名称" addonBefore={<VideoCameraOutlined />} />
                        </Form.Item>

                        <Form.Item
                          name="usb_camera_device"
                          label="摄像头设备"
                          rules={[{ required: configType === 'usb', message: '请选择USB摄像头设备' }]}
                        >
                          <Input placeholder="例如: /dev/video0 或 video0" />
                        </Form.Item>

                        <Form.Item
                          name="camera_backend"
                          label="摄像头后端"
                        >
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
                                const device = detectedAudios.find(d => d.device_id === value)
                                if (device) handleSelectAudio(device)
                              }}
                              options={detectedAudios.map(device => ({
                                label: (
                                  <Space>
                                    <AudioOutlined />
                                    {device.name}
                                    <Tag color={device.status === 'available' ? 'green' : 'orange'}>
                                      {device.status}
                                    </Tag>
                                    <span style={{ color: '#999', fontSize: '12px' }}>
                                      ({device.backend})
                                    </span>
                                  </Space>
                                ),
                                value: device.device_id,
                              }))}
                            />
                          </Form.Item>
                        )}

                        <Form.Item
                          name="usb_audio_name"
                          label="音频设备名称"
                        >
                          <Input placeholder="请输入USB音频设备名称" addonBefore={<AudioOutlined />} />
                        </Form.Item>

                        <Form.Item
                          name="usb_audio_device"
                          label="音频设备"
                        >
                          <Input placeholder="例如: hw:1,0 或 audio=0" />
                        </Form.Item>

                        <Form.Item
                          name="audio_backend"
                          label="音频后端"
                        >
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
                    ) : (
                      <div style={{ color: '#999', textAlign: 'center', padding: '40px' }}>
                        请先选择"USB设备直录"类型
                      </div>
                    )}
                  </>
                ),
              },
              {
                key: 'stream',
                label: '流媒体配置',
                disabled: configType !== 'stream',
                children: (
                  <>
                    {configType === 'stream' ? (
                      <>
                        <Form.Item
                          name="stream_enabled"
                          label="启用流媒体录制"
                          valuePropName="checked"
                        >
                          <Switch
                            checkedChildren="启用"
                            unCheckedChildren="禁用"
                          />
                        </Form.Item>

                        <Form.Item noStyle shouldUpdate={(prev, curr) => prev.stream_enabled !== curr.stream_enabled}>
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
                    ) : (
                      <div style={{ color: '#999', textAlign: 'center', padding: '40px' }}>
                        请先选择"流媒体录制"类型
                      </div>
                    )}
                  </>
                ),
              },
            ]}
          />
        </Form>
      </Modal>

      {/* 配置详情对话框 */}
      <Modal
        title={<Space><SettingOutlined />配置详情 - {viewingConfig?.name}</Space>}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailVisible(false)}>
            关闭
          </Button>,
        ]}
        width={900}
      >
        {viewingConfig && (
          <Descriptions column={2} bordered size="small">
            <Descriptions.Item label="配置ID">{viewingConfig.id}</Descriptions.Item>
            <Descriptions.Item label="配置名称">{viewingConfig.name}</Descriptions.Item>
            <Descriptions.Item label="配置类型" span={2}>
              <Tag color={configTypeMap[viewingConfig.config_type]?.color} icon={configTypeMap[viewingConfig.config_type]?.icon}>
                {configTypeMap[viewingConfig.config_type]?.text || viewingConfig.config_type}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="华为终端控制" span={2}>
              {viewingConfig.config_type === 'usb' ? (
                <Tag color={viewingConfig.huawei_enabled ? 'green' : 'orange'}>
                  {viewingConfig.huawei_enabled ? '启用自动控制' : '手动模式'}
                </Tag>
              ) : (
                '-'
              )}
            </Descriptions.Item>
            <Descriptions.Item label="状态" span={2}>
              <Space>
                <Tag color={viewingConfig.is_active ? 'green' : 'red'}>
                  {viewingConfig.is_active ? '激活' : '禁用'}
                </Tag>
                {viewingConfig.is_locked && (
                  <Tag color="orange" icon={<LockOutlined />}>
                    已锁定
                  </Tag>
                )}
              </Space>
            </Descriptions.Item>
            <Descriptions.Item label="描述" span={2}>
              {viewingConfig.description || '-'}
            </Descriptions.Item>

            {/* 华为终端字段 */}
            {viewingConfig.config_type === 'usb' && viewingConfig.huawei_enabled && (
              <>
                <Descriptions.Item label="服务器">{viewingConfig.server || '-'}</Descriptions.Item>
                <Descriptions.Item label="端口">{viewingConfig.port || '-'}</Descriptions.Item>
                <Descriptions.Item label="用户名">{viewingConfig.username || '-'}</Descriptions.Item>
                <Descriptions.Item label="终端号">{viewingConfig.terminal_number || '-'}</Descriptions.Item>
                <Descriptions.Item label="会议号" span={2}>
                  {viewingConfig.conference_number || '-'}
                </Descriptions.Item>
              </>
            )}

            {/* USB设备字段 */}
            {(viewingConfig.config_type === 'usb' || (viewingConfig.config_type === 'usb' && !viewingConfig.huawei_enabled)) && (
              <>
                <Descriptions.Item label="摄像头名称" span={2}>
                  {viewingConfig.usb_camera_name || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="摄像头设备" span={2}>
                  {viewingConfig.usb_camera_device || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="摄像头后端" span={2}>
                  {viewingConfig.camera_backend || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="音频设备名称" span={2}>
                  {viewingConfig.usb_audio_name || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="音频设备" span={2}>
                  {viewingConfig.usb_audio_device || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="音频后端" span={2}>
                  {viewingConfig.audio_backend || '-'}
                </Descriptions.Item>
              </>
            )}

            {/* 流媒体字段 */}
            {viewingConfig.config_type === 'stream' && (
              <>
                <Descriptions.Item label="流媒体协议" span={2}>
                  {viewingConfig.stream_protocol ? (
                    <Tag color="blue">{viewingConfig.stream_protocol.toUpperCase()}</Tag>
                  ) : '-'}
                </Descriptions.Item>
                <Descriptions.Item label="流媒体URL" span={2}>
                  {viewingConfig.stream_url || '-'}
                </Descriptions.Item>
                <Descriptions.Item label="流媒体状态" span={2}>
                  <Tag color={viewingConfig.stream_enabled ? 'green' : 'red'}>
                    {viewingConfig.stream_enabled ? '启用' : '禁用'}
                  </Tag>
                </Descriptions.Item>
              </>
            )}

            <Descriptions.Item label="输出格式" span={2}>
              {viewingConfig.output_format || '-'}
            </Descriptions.Item>
            <Descriptions.Item label="关联任务数" span={2}>
              {viewingConfig.video_recording_tasks?.length || 0}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{viewingConfig.created_at}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{viewingConfig.updated_at}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  )
}
