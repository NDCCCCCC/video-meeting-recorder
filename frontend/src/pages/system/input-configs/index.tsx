// 输入配置管理页面

import { useState, useEffect, useCallback, useMemo } from 'react'
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
} from 'antd'
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SettingOutlined,
  LockOutlined,
  VideoCameraOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as inputConfigApi from '../../../api/input-config'
import type { InputConfig, InputConfigListParams, ConfigType } from '../../../types/input-config'
import { buildCreatePayload, buildUpdatePayload } from './utils'
import { InputConfigDetailModal } from './components/InputConfigDetailModal'
import { useUSBDeviceScan } from './hooks/useUSBDeviceScan'
import { BasicConfigTab } from './components/BasicConfigTab'
import { HuaweiConfigTab } from './components/HuaweiConfigTab'
import { USBConfigTab } from './components/USBConfigTab'
import { StreamConfigTab } from './components/StreamConfigTab'

// 配置类型显示映射（模块级，保持引用稳定，供 columns useMemo 与详情 Modal 共用）
const configTypeMap = {
  usb: { text: 'USB直录', color: 'green', icon: <VideoCameraOutlined /> },
  stream: { text: '流媒体', color: 'orange', icon: <PlayCircleOutlined /> },
}

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

  // USB 设备扫描（封装为 hook）
  const {
    scanningDevices,
    detectedCameras,
    detectedAudios,
    scanDevices,
    selectCamera,
    selectAudio,
    clearDevices,
  } = useUSBDeviceScan(form)

  const [params, setParams] = useState<InputConfigListParams>({
    page: 1,
    page_size: 20,
  })

  const loadConfigs = useCallback(async () => {
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
  }, [params])

  useEffect(() => {
    loadConfigs()
  }, [loadConfigs])

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

  const openModal = useCallback((config: InputConfig | null = null) => {
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
    scanDevices()
  }, [form, scanDevices])

  const closeModal = () => {
    setModalVisible(false)
    setEditingConfig(null)
    form.resetFields()
    clearDevices()
    setConfigType('usb')
    setHuaweiEnabled(false)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()

      if (editingConfig) {
        // 编辑模式：密码为空则不更新密码
        const req = buildUpdatePayload(values)
        await inputConfigApi.updateInputConfig(editingConfig.id, req)
        message.success('更新成功')
      } else {
        // 新建模式：根据配置类型决定必填字段
        const req = buildCreatePayload(values, configType, huaweiEnabled)
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

  const handleDelete = useCallback(
    async (id: number) => {
      try {
        await inputConfigApi.deleteInputConfig(id)
        message.success('删除成功')
        loadConfigs()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '删除失败')
      }
    },
    [loadConfigs]
  )

  const viewDetail = useCallback((config: InputConfig) => {
    setViewingConfig(config)
    setDetailVisible(true)
  }, [])

  const columns = useMemo<ColumnsType<InputConfig>>(() => [
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
        const { text, color, icon } = configTypeMap[type] || {
          text: type,
          color: 'default',
          icon: null,
        }
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
        <Space orientation="vertical" size={0}>
          <Tag color={active ? 'green' : 'red'}>{active ? '激活' : '禁用'}</Tag>
          {record.is_locked && (
            <Tag color="orange" icon={<LockOutlined />}>
              已锁定
            </Tag>
          )}
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
          <Button type="link" size="small" onClick={() => viewDetail(record)}>
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
  ], [viewDetail, openModal, handleDelete])

  return (
    <div style={{ padding: '20px' }}>
      <div
        style={{
          marginBottom: '16px',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
      >
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
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          <Tabs
            items={[
              {
                key: 'basic',
                label: '基本配置',
                children: (
                  <BasicConfigTab
                    form={form}
                    configType={configType}
                    setConfigType={setConfigType}
                    huaweiEnabled={huaweiEnabled}
                    setHuaweiEnabled={setHuaweiEnabled}
                  />
                ),
              },
              {
                key: 'huawei',
                label: '华为终端配置',
                disabled: !huaweiEnabled,
                children: (
                  <HuaweiConfigTab huaweiEnabled={huaweiEnabled} editingConfig={editingConfig} />
                ),
              },
              {
                key: 'usb',
                label: 'USB设备配置',
                disabled: configType !== 'usb',
                children: (
                  <USBConfigTab
                    configType={configType}
                    scanningDevices={scanningDevices}
                    detectedCameras={detectedCameras}
                    detectedAudios={detectedAudios}
                    onScan={scanDevices}
                    onSelectCamera={selectCamera}
                    onSelectAudio={selectAudio}
                  />
                ),
              },
              {
                key: 'stream',
                label: '流媒体配置',
                disabled: configType !== 'stream',
                children: <StreamConfigTab configType={configType} />,
              },
            ]}
          />
        </Form>
      </Modal>

      {/* 配置详情对话框 */}
      <InputConfigDetailModal
        open={detailVisible}
        config={viewingConfig}
        configTypeMap={configTypeMap}
        onClose={() => setDetailVisible(false)}
      />
    </div>
  )
}
