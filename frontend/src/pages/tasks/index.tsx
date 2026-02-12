// 录制任务管理页面

import { useState, useEffect, useMemo, useCallback } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  message,
  Popconfirm,
  Tag,
  DatePicker,
  Tooltip,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  StopOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import * as taskApi from '../../api/task'
import * as huaweiConfigApi from '../../api/huawei-config'
import { HLSPreview } from '../../components/HLSPreview'
import { PermissionGuard } from '../../components/PermissionGuard'
import { PERMISSIONS } from '../../utils/permissions'
import type {
  VideoRecordingTask,
  TaskListParams,
  CreateTaskRequest,
  UpdateTaskRequest,
  VideoRecordingTaskStatus,
} from '../../types/task'
import type { HuaweiConfig } from '../../types/huawei-config'

const { RangePicker } = DatePicker

// 状态配置
const STATUS_CONFIG: Record<VideoRecordingTaskStatus, { label: string; color: string }> = {
  pending: { label: '待执行', color: 'default' },
  connecting: { label: '连接中', color: 'processing' },
  recording: { label: '录制中', color: 'blue' },
  completed: { label: '已完成', color: 'success' },
  failed: { label: '失败', color: 'error' },
  cancelled: { label: '已取消', color: 'default' },
}

// 常量定义
const DEFAULT_PAGE_SIZE = 20
const DEFAULT_PRE_JOIN_MINUTES = 30
const DEFAULT_RECORD_DELAY_MINUTES = 0
const DELETABLE_STATUSES: VideoRecordingTaskStatus[] = ['pending', 'completed', 'failed', 'cancelled']

// 格式化时长
const formatDuration = (seconds: number): string => {
  if (!seconds) return '-'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

export default function TaskManagement() {
  const [tasks, setTasks] = useState<VideoRecordingTask[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingTask, setEditingTask] = useState<VideoRecordingTask | null>(null)
  const [form] = Form.useForm()

  // 华为配置列表
  const [huaweiConfigs, setHuaweiConfigs] = useState<HuaweiConfig[]>([])
  const [configsLoading, setConfigsLoading] = useState(false)

  // 行选择状态
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [batchDeleteLoading, setBatchDeleteLoading] = useState(false)

  // 查询参数
  const [params, setParams] = useState<TaskListParams>({
    page: 1,
    page_size: DEFAULT_PAGE_SIZE,
  })

  // 加载任务列表
  const loadTasks = useCallback(async (showLoading = false) => {
    if (showLoading) setLoading(true)
    try {
      const response = await taskApi.getTaskList(params)
      if (response.data) {
        setTasks(response.data.items)
        setTotal(response.data.total)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载任务列表失败')
    } finally {
      if (showLoading) setLoading(false)
    }
  }, [params])

  // 加载华为配置列表
  const loadHuaweiConfigs = useCallback(async () => {
    setConfigsLoading(true)
    try {
      const response = await huaweiConfigApi.getActiveHuaweiConfigs()
      if (response.data) {
        setHuaweiConfigs(response.data)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载华为配置失败')
    } finally {
      setConfigsLoading(false)
    }
  }, [])

  // 初始加载
  useEffect(() => {
    loadTasks(true)
    loadHuaweiConfigs()
  }, [loadTasks, loadHuaweiConfigs])

  // 定时轮询：自动刷新进行中的任务状态
  useEffect(() => {
    const hasActiveTasks = tasks.some(task =>
      ['pending', 'connecting', 'recording'].includes(task.status)
    )

    if (!hasActiveTasks) return

    const interval = setInterval(() => {
      loadTasks(false)
    }, 5000)

    return () => clearInterval(interval)
  }, [tasks, loadTasks])

  // 搜索
  const handleSearch = useCallback((value: string) => {
    setParams(prev => ({ ...prev, keyword: value, page: 1 }))
  }, [])

  // 状态筛选
  const handleStatusFilter = useCallback((value: VideoRecordingTaskStatus | undefined) => {
    setParams(prev => ({ ...prev, status: value, page: 1 }))
  }, [])

  // 日期范围筛选
  const handleDateRangeChange = useCallback((dates: any) => {
    if (dates && dates.length === 2) {
      setParams(prev => ({
        ...prev,
        start_date: dates[0].format('YYYY-MM-DD'),
        end_date: dates[1].format('YYYY-MM-DD'),
        page: 1,
      }))
    } else {
      setParams(prev => {
        const { start_date, end_date, ...rest } = prev
        return rest
      })
    }
  }, [])

  // 分页变化
  const handleTableChange = useCallback((pagination: any) => {
    setParams(prev => ({
      ...prev,
      page: pagination.current,
      page_size: pagination.pageSize,
    }))
  }, [])

  // 打开新建/编辑对话框
  const openModal = useCallback((task: VideoRecordingTask | null = null) => {
    setEditingTask(task)
    if (task) {
      form.setFieldsValue({
        name: task.name,
        description: task.description,
        start_time: dayjs(task.start_time),
        end_time: dayjs(task.end_time),
        pre_join_minutes: task.pre_join_minutes,
        record_delay_minutes: task.record_delay_minutes,
        conference_number: task.conference_number,
        huawei_config_id: task.huawei_config_id,
      })
    } else {
      form.resetFields()
    }
    setModalVisible(true)
  }, [form])

  // 关闭对话框
  const closeModal = useCallback(() => {
    setModalVisible(false)
    setEditingTask(null)
    form.resetFields()
  }, [form])

  // 提交表单
  const handleSubmit = useCallback(async () => {
    try {
      const values = await form.validateFields()

      const requestData = {
        ...values,
        start_time: values.start_time.toISOString(),
        end_time: values.end_time.toISOString(),
      }

      if (editingTask) {
        const req: UpdateTaskRequest = {
          name: requestData.name,
          description: requestData.description,
          start_time: requestData.start_time,
          end_time: requestData.end_time,
          pre_join_minutes: requestData.pre_join_minutes,
          record_delay_minutes: requestData.record_delay_minutes,
        }
        await taskApi.updateTask(editingTask.id, req)
        message.success('更新成功')
      } else {
        const req: CreateTaskRequest = requestData
        await taskApi.createTask(req)
        message.success('创建成功')
      }

      closeModal()
      loadTasks()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败')
    }
  }, [editingTask, form, closeModal, loadTasks])

  // 删除任务
  const handleDelete = useCallback(async (id: number) => {
    try {
      await taskApi.deleteTask(id)
      message.success('删除成功')
      loadTasks()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }, [loadTasks])

  // 启动任务
  const handleStart = useCallback(async (id: number) => {
    try {
      await taskApi.startTask(id)
      message.success('任务启动成功')
      loadTasks()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '启动失败')
    }
  }, [loadTasks])

  // 停止任务
  const handleStop = useCallback(async (id: number) => {
    try {
      await taskApi.stopTask(id)
      message.success('任务停止成功')
      loadTasks()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '停止失败')
    }
  }, [loadTasks])

  // 取消任务
  const handleCancel = useCallback(async (id: number) => {
    try {
      await taskApi.cancelTask(id)
      message.success('任务取消成功')
      loadTasks()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '取消失败')
    }
  }, [loadTasks])

  // 重试任务
  const handleRetry = useCallback(async (id: number) => {
    try {
      await taskApi.retryTask(id)
      message.success('任务重试成功')
      loadTasks()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '重试失败')
    }
  }, [loadTasks])

  // 批量删除任务
  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的任务')
      return
    }

    const cannotDeleteTasks = tasks.filter(task =>
      selectedRowKeys.includes(task.id) && !DELETABLE_STATUSES.includes(task.status)
    )

    if (cannotDeleteTasks.length > 0) {
      const cannotDeleteNames = cannotDeleteTasks.map(t => t.name).join('、')
      message.warning(`以下任务无法删除：${cannotDeleteNames}\n只能删除待执行、已完成、失败、已取消状态的任务`)
      return
    }

    Modal.confirm({
      title: '批量删除任务',
      content: `确定要删除选中的 ${selectedRowKeys.length} 个任务吗？`,
      okText: '确定',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        setBatchDeleteLoading(true)
        try {
          await taskApi.batchDeleteTasks(selectedRowKeys as number[])
          message.success(`成功删除 ${selectedRowKeys.length} 个任务`)
          setSelectedRowKeys([])
          loadTasks()
        } catch (error) {
          message.error(error instanceof Error ? error.message : '批量删除失败')
        } finally {
          setBatchDeleteLoading(false)
        }
      },
    })
  }, [selectedRowKeys, tasks, loadTasks])

  // 获取可删除的任务 ID 列表
  const deletableTaskIds = useMemo(() => {
    return new Set(
      tasks
        .filter(task => DELETABLE_STATUSES.includes(task.status))
        .map(task => task.id)
    )
  }, [tasks])

  // 行选择配置
  const rowSelection = useMemo(() => ({
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(newSelectedRowKeys)
    },
    getCheckboxProps: (record: VideoRecordingTask) => ({
      disabled: !deletableTaskIds.has(record.id),
      name: record.name,
    }),
  }), [selectedRowKeys, deletableTaskIds])

  // 渲染状态标签
  const renderStatus = useCallback((status: VideoRecordingTaskStatus) => {
    const config = STATUS_CONFIG[status]
    return <Tag color={config.color}>{config.label}</Tag>
  }, [])

  // 渲染操作按钮
  const renderActions = useCallback((record: VideoRecordingTask) => {
    const canEdit = record.status === 'pending'
    const canDelete = DELETABLE_STATUSES.includes(record.status)
    const canStart = record.status === 'pending'
    const canStop = ['recording', 'connecting'].includes(record.status)
    const canCancel = ['pending', 'connecting'].includes(record.status)
    const canRetry = record.status === 'failed'
    const canPreview = record.status === 'recording'

    return (
      <Space size="small">
        {canPreview && <HLSPreview taskId={record.id} taskName={record.name} status={record.status} />}
        <PermissionGuard permission={PERMISSIONS.TASK_START}>
          {canStart && (
            <Tooltip title="启动任务">
              <Button type="link" size="small" icon={<PlayCircleOutlined />} onClick={() => handleStart(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_STOP}>
          {canStop && (
            <Tooltip title="停止任务">
              <Button type="link" size="small" danger icon={<StopOutlined />} onClick={() => handleStop(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_STOP}>
          {canCancel && (
            <Tooltip title="取消任务">
              <Button type="link" size="small" icon={<CloseCircleOutlined />} onClick={() => handleCancel(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_START}>
          {canRetry && (
            <Tooltip title="重试任务">
              <Button type="link" size="small" icon={<ReloadOutlined />} onClick={() => handleRetry(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_EDIT}>
          {canEdit && (
            <Tooltip title="编辑任务">
              <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_DELETE}>
          {canDelete && (
            <Popconfirm title="确定要删除这个任务吗？" onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
        </PermissionGuard>
      </Space>
    )
  }, [handleStart, handleStop, handleCancel, handleRetry, handleDelete, openModal])

  // 表格列定义
  const columns: ColumnsType<VideoRecordingTask> = useMemo(() => [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '任务名称',
      dataIndex: 'name',
      width: 200,
      ellipsis: true,
    },
    {
      title: '会议号',
      dataIndex: 'conference_number',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: renderStatus,
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      width: 160,
      render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      width: 160,
      render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '提前进入(分钟)',
      dataIndex: 'pre_join_minutes',
      width: 120,
      align: 'center',
    },
    {
      title: '录制时长',
      dataIndex: 'recording_duration',
      width: 120,
      align: 'center',
      render: formatDuration,
    },
    {
      title: '错误信息',
      dataIndex: 'error_msg',
      width: 200,
      ellipsis: true,
      render: (msg) => msg ? <Tooltip title={msg}><span style={{ color: 'red' }}>{msg}</span></Tooltip> : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      fixed: 'right' as const,
      render: (_: unknown, record: VideoRecordingTask) => renderActions(record),
    },
  ], [renderStatus, renderActions])

  const statusOptions = useMemo(() =>
    Object.entries(STATUS_CONFIG).map(([value, { label }]) => ({
      value,
      label,
    }))
  , [])

  return (
    <div style={{ padding: '20px' }}>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>录制任务管理</h2>
        <Space>
          <PermissionGuard permission={PERMISSIONS.TASK_DELETE}>
            {selectedRowKeys.length > 0 && (
              <Button
                danger
                icon={<DeleteOutlined />}
                loading={batchDeleteLoading}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
          </PermissionGuard>
          <PermissionGuard permission={PERMISSIONS.TASK_CREATE}>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
              新建任务
            </Button>
          </PermissionGuard>
        </Space>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <Space size="middle" wrap>
          <Input.Search
            placeholder="搜索任务名称、会议号"
            allowClear
            style={{ width: 250 }}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Select
            placeholder="选择状态"
            allowClear
            style={{ width: 120 }}
            onChange={handleStatusFilter}
            options={statusOptions}
          />
          <RangePicker
            placeholder={['开始日期', '结束日期']}
            onChange={handleDateRangeChange}
          />
          <Button icon={<ReloadOutlined />} onClick={() => loadTasks(true)}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={tasks}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1500 }}
        rowSelection={rowSelection}
        pagination={{
          current: params.page,
          pageSize: params.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        onChange={handleTableChange}
      />

      {/* 新建/编辑任务对话框 */}
      <Modal
        title={editingTask ? '编辑录制任务' : '新建录制任务'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={closeModal}
        width={700}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="任务名称"
            rules={[
              { required: true, message: '请输入任务名称' },
              { max: 200, message: '任务名称最多200个字符' },
            ]}
          >
            <Input placeholder="请输入任务名称" />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 500, message: '描述最多500个字符' }]}
          >
            <Input.TextArea placeholder="请输入任务描述" rows={3} />
          </Form.Item>

          <Form.Item
            name="conference_number"
            label="会议号"
            rules={[
              { required: true, message: '请输入会议号' },
              { max: 50, message: '会议号最多50个字符' },
            ]}
          >
            <Input placeholder="请输入华为会议号" />
          </Form.Item>

          <Form.Item
            name="huawei_config_id"
            label="华为配置"
            rules={[{ required: true, message: '请选择华为配置' }]}
          >
            <Select
              placeholder="请选择华为配置"
              loading={configsLoading}
              showSearch
              optionFilterProp="label"
            >
              {huaweiConfigs.map((config) => (
                <Select.Option key={config.id} value={config.id}>
                  {config.name} ({config.server}:{config.port})
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Space size="large">
            <Form.Item
              name="start_time"
              label="开始时间"
              rules={[{ required: true, message: '请选择开始时间' }]}
            >
              <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
            </Form.Item>

            <Form.Item
              name="end_time"
              label="结束时间"
              rules={[{ required: true, message: '请选择结束时间' }]}
            >
              <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
            </Form.Item>
          </Space>

          <Space size="large">
            <Form.Item
              name="pre_join_minutes"
              label="提前进入(分钟)"
              rules={[
                { type: 'number', min: 0, max: 60, message: '提前进入时间必须在0-60分钟之间' },
              ]}
              initialValue={DEFAULT_PRE_JOIN_MINUTES}
            >
              <Input type="number" style={{ width: 120 }} />
            </Form.Item>

            <Form.Item
              name="record_delay_minutes"
              label="录制延迟(分钟)"
              rules={[
                { type: 'number', min: 0, max: 60, message: '录制延迟必须在0-60分钟之间' },
              ]}
              initialValue={DEFAULT_RECORD_DELAY_MINUTES}
            >
              <Input type="number" style={{ width: 120 }} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  )
}
