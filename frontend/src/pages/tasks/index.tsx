// 录制任务管理页面

import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
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
import {
  STATUS_CONFIG,
  DEFAULT_PAGE_SIZE,
  DEFAULT_PRE_JOIN_MINUTES,
  DEFAULT_RECORD_DELAY_MINUTES,
  POLL_INTERVAL,
  DELETABLE_STATUSES,
  ACTIVE_STATUSES,
  STATUS_OPTIONS,
  canStartTask,
  canStopTask,
  canCancelTask,
  canRetryTask,
  canPreviewTask,
  canEditEndTime,
  canEditAllFields,
} from './constants'
import { formatDuration, hasActiveTasks } from './utils'
import type {
  VideoRecordingTask,
  VideoRecordingTaskStatus,
  TaskListParams,
  CreateTaskRequest,
  UpdateTaskRequest,
} from '../../types/task'
import type { HuaweiConfig } from '../../types/huawei-config'

const { RangePicker } = DatePicker

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

  // 使用 ref 存储 loadTasks 函数，避免依赖循环 (rerender-functional-setstate)
  const loadTasksRef = useRef<((showLoading?: boolean) => Promise<void>) | null>(null)

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

  // 存储最新的 loadTasks 引用
  loadTasksRef.current = loadTasks

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

  // 定时轮询：自动刷新进行中的任务状态 (使用 ref 避免依赖)
  useEffect(() => {
    if (!hasActiveTasks(tasks, ACTIVE_STATUSES)) return

    const interval = setInterval(() => {
      loadTasksRef.current?.(false)
    }, POLL_INTERVAL)

    return () => clearInterval(interval)
  }, [tasks])

  // 搜索
  const handleSearch = useCallback((value: string) => {
    setParams(prev => ({ ...prev, keyword: value, page: 1 }))
  }, [])

  // 状态筛选
  const handleStatusFilter = useCallback((value: VideoRecordingTaskStatus | undefined) => {
    setParams(prev => ({ ...prev, status: value, page: 1 }))
  }, [])

  // 日期范围筛选
  const handleDateRangeChange = useCallback((dates: unknown) => {
    if (dates && Array.isArray(dates) && dates.length === 2) {
      setParams(prev => ({
        ...prev,
        start_date: (dates[0] as { format: (fmt: string) => string }).format('YYYY-MM-DD'),
        end_date: (dates[1] as { format: (fmt: string) => string }).format('YYYY-MM-DD'),
        page: 1,
      }))
    } else {
      setParams(prev => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { start_date, end_date, ...rest } = prev
        return rest
      })
    }
  }, [])

  // 分页变化
  const handleTableChange = useCallback((pagination: { current?: number; pageSize?: number }) => {
    setParams(prev => ({
      ...prev,
      page: pagination.current ?? 1,
      page_size: pagination.pageSize ?? DEFAULT_PAGE_SIZE,
    }))
  }, [DEFAULT_PAGE_SIZE])

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
        // 录制中状态只更新结束时间
        const isRecording = editingTask.status === 'recording'
        const req: UpdateTaskRequest = isRecording
          ? {
              end_time: requestData.end_time,
            }
          : {
              name: requestData.name,
              description: requestData.description,
              start_time: requestData.start_time,
              end_time: requestData.end_time,
              pre_join_minutes: requestData.pre_join_minutes,
              record_delay_minutes: requestData.record_delay_minutes,
            }

        await taskApi.updateTask(editingTask.id, req)
        message.success(isRecording ? '结束时间已更新' : '更新成功')
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

  // 获取可删除的任务 ID 列表 (rerender-derived-state)
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
  const renderStatus = useCallback((status: string) => {
    const config = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG]
    return config ? <Tag color={config.color}>{config.label}</Tag> : null
  }, [])

  // 渲染操作按钮
  const renderActions = useCallback((record: VideoRecordingTask) => {
    return (
      <Space size="small">
        {canPreviewTask(record.status) && <HLSPreview taskId={record.id} taskName={record.name} status={record.status} />}
        <PermissionGuard permission={PERMISSIONS.TASK_START}>
          {canStartTask(record.status) && (
            <Tooltip title="启动任务">
              <Button type="link" size="small" icon={<PlayCircleOutlined />} onClick={() => handleStart(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_STOP}>
          {canStopTask(record.status) && (
            <Tooltip title="停止任务">
              <Button type="link" size="small" danger icon={<StopOutlined />} onClick={() => handleStop(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_STOP}>
          {canCancelTask(record.status) && (
            <Tooltip title="取消任务">
              <Button type="link" size="small" icon={<CloseCircleOutlined />} onClick={() => handleCancel(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_START}>
          {canRetryTask(record.status) && (
            <Tooltip title="重试任务">
              <Button type="link" size="small" icon={<ReloadOutlined />} onClick={() => handleRetry(record.id)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_EDIT}>
          {canEditEndTime(record.status) && (
            <Tooltip title={record.status === 'recording' ? '修改结束时间' : '编辑任务'}>
              <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)} />
            </Tooltip>
          )}
        </PermissionGuard>
        <PermissionGuard permission={PERMISSIONS.TASK_DELETE}>
          {DELETABLE_STATUSES.includes(record.status) && (
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

  return (
    <div className="page-container">
      <div className="page-header">
        <h2>录制任务管理</h2>
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

      <div className="toolbar">
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
            options={STATUS_OPTIONS}
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
        title={
          editingTask
            ? editingTask.status === 'recording'
              ? '修改结束时间'
              : '编辑录制任务'
            : '新建录制任务'
        }
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={closeModal}
        width={700}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          {/* 录制中状态只能编辑结束时间，其他字段被禁用并显示提示 */}
          {editingTask && editingTask.status === 'recording' && (
            <div style={{ marginBottom: 16, padding: 12, background: '#e6f7ff', borderRadius: 4 }}>
              任务正在录制中，仅可修改结束时间
            </div>
          )}

          <Form.Item
            name="name"
            label="任务名称"
            rules={[
              { required: !editingTask || canEditAllFields(editingTask.status), message: '请输入任务名称' },
              { max: 200, message: '任务名称最多200个字符' },
            ]}
          >
            <Input
              placeholder="请输入任务名称"
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 500, message: '描述最多500个字符' }]}
          >
            <Input.TextArea
              placeholder="请输入任务描述"
              rows={3}
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="conference_number"
            label="会议号"
            rules={[
              { required: !editingTask || canEditAllFields(editingTask.status), message: '请输入会议号' },
              { max: 50, message: '会议号最多50个字符' },
            ]}
          >
            <Input
              placeholder="请输入华为会议号"
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="huawei_config_id"
            label="华为配置"
            rules={[{ required: !editingTask || canEditAllFields(editingTask.status), message: '请选择华为配置' }]}
          >
            <Select
              placeholder="请选择华为配置"
              loading={configsLoading}
              showSearch
              optionFilterProp="label"
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
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
              rules={[{ required: !editingTask || canEditAllFields(editingTask.status), message: '请选择开始时间' }]}
            >
              <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" disabled={!!editingTask && !canEditAllFields(editingTask.status)} />
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
              <Input type="number" style={{ width: 120 }} disabled={!!editingTask && !canEditAllFields(editingTask.status)} />
            </Form.Item>

            <Form.Item
              name="record_delay_minutes"
              label="录制延迟(分钟)"
              rules={[
                { type: 'number', min: 0, max: 60, message: '录制延迟必须在0-60分钟之间' },
              ]}
              initialValue={DEFAULT_RECORD_DELAY_MINUTES}
            >
              <Input type="number" style={{ width: 120 }} disabled={!!editingTask && !canEditAllFields(editingTask.status)} />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  )
}
