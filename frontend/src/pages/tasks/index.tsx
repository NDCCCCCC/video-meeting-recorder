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
  Tag,
  DatePicker,
  Tooltip,
  Empty,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  PlusOutlined,
  SearchOutlined,
  DeleteOutlined,
  ReloadOutlined,
  ClearOutlined,
} from '@ant-design/icons'
import dayjs from 'dayjs'
import * as taskApi from '../../api/task'
import * as inputConfigApi from '../../api/input-config'
import { apiRequest } from '../../api/apiClient'
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
  canEditAllFields,
} from './constants'
import { formatDuration, hasActiveTasks } from './utils'
import { TaskActions } from './components/TaskActions'
import EmptyTasks from '@/assets/illustrations/EmptyTasks'
import ErrorNetwork from '@/assets/illustrations/ErrorNetwork'
import { designTokens } from '@/styles/theme'
import type {
  VideoRecordingTask,
  VideoRecordingTaskStatus,
  TaskListParams,
  CreateTaskRequest,
  UpdateTaskRequest,
} from '../../types/task'
import type { InputConfig } from '../../types/input-config'

const { RangePicker } = DatePicker

type ConfigType = 'usb' | 'stream' | 'none'

const CONFIG_TYPE_TAGS: Record<ConfigType, { color: string; label: string }> = {
  usb: { color: 'blue', label: 'USB' },
  stream: { color: 'green', label: '流媒体' },
  none: { color: 'default', label: '未配置' },
}

const getConfigType = (config: InputConfig): ConfigType => {
  const hasUSB = config.usb_camera_device || config.usb_audio_device
  const hasStream = config.stream_enabled && config.stream_url

  if (hasUSB) return 'usb'
  if (hasStream) return 'stream'
  return 'none'
}

const getConfigTypeTagConfig = (config: InputConfig) => {
  const type = getConfigType(config)
  const tagConfig = CONFIG_TYPE_TAGS[type]
  const label =
    type === 'stream'
      ? `${tagConfig.label}(${config.stream_protocol?.toUpperCase()})`
      : tagConfig.label
  return { ...tagConfig, label }
}

export default function TaskManagement() {
  const [tasks, setTasks] = useState<VideoRecordingTask[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  // D-05.2 — 保留加载失败的具体原因，用于错误态展示
  const [loadError, setLoadError] = useState<string | null>(null)
  // 搜索框受控值（用于"清空筛选"时同步清空输入）
  const [searchInput, setSearchInput] = useState('')
  // 日期范围受控值
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingTask, setEditingTask] = useState<VideoRecordingTask | null>(null)
  const [form] = Form.useForm()

  // 输入配置列表
  const [huaweiConfigs, setHuaweiConfigs] = useState<InputConfig[]>([])
  const [configsLoading, setConfigsLoading] = useState(false)

  // 行选择状态
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [batchDeleteLoading, setBatchDeleteLoading] = useState(false)

  // 快照生成状态
  const [snapshotGenerating, setSnapshotGenerating] = useState<Set<number>>(new Set())

  // 查询参数
  const [params, setParams] = useState<TaskListParams>({
    page: 1,
    page_size: DEFAULT_PAGE_SIZE,
  })

  // 使用 ref 存储 loadTasks 函数，避免依赖循环 (rerender-functional-setstate)
  const loadTasksRef = useRef<((showLoading?: boolean) => Promise<void>) | null>(null)

  // 加载任务列表
  const loadTasks = useCallback(
    async (showLoading = false) => {
      if (showLoading) setLoading(true)
      try {
        const response = await taskApi.getTaskList(params)
        if (response.data) {
          setTasks(response.data.items)
          setTotal(response.data.total)
        }
        setLoadError(null)
      } catch (error) {
        const reason = error instanceof Error ? error.message : '网络连接中断'
        setLoadError(reason)
        message.error(`加载失败：${reason}`)
      } finally {
        if (showLoading) setLoading(false)
      }
    },
    [params]
  )

  // 存储最新的 loadTasks 引用
  loadTasksRef.current = loadTasks

  // 加载输入配置列表（使用 input_configs 表）
  const loadHuaweiConfigs = useCallback(async () => {
    setConfigsLoading(true)
    try {
      const response = await inputConfigApi.getActiveInputConfigs()
      if (response.data) {
        setHuaweiConfigs(response.data)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载输入配置失败')
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
    setParams((prev) => ({ ...prev, keyword: value, page: 1 }))
  }, [])

  // 状态筛选
  const handleStatusFilter = useCallback((value: VideoRecordingTaskStatus | undefined) => {
    setParams((prev) => ({ ...prev, status: value, page: 1 }))
  }, [])

  // D-05.1 — 清空筛选条件，回到完整列表
  const handleClearFilters = useCallback(() => {
    setSearchInput('')
    setDateRange(null)
    setParams({ page: 1, page_size: DEFAULT_PAGE_SIZE })
  }, [])

  // 日期范围筛选
  const handleDateRangeChange = useCallback((dates: unknown) => {
    setDateRange(dates as [dayjs.Dayjs, dayjs.Dayjs] | null)
    if (dates && Array.isArray(dates) && dates.length === 2) {
      setParams((prev) => ({
        ...prev,
        start_date: (dates[0] as { format: (fmt: string) => string }).format('YYYY-MM-DD'),
        end_date: (dates[1] as { format: (fmt: string) => string }).format('YYYY-MM-DD'),
        page: 1,
      }))
    } else {
      setParams((prev) => {
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        const { start_date, end_date, ...rest } = prev
        return rest
      })
    }
  }, [])

  // 分页变化
  const handleTableChange = useCallback((pagination: { current?: number; pageSize?: number }) => {
    setParams((prev) => ({
      ...prev,
      page: pagination.current ?? 1,
      page_size: pagination.pageSize ?? DEFAULT_PAGE_SIZE,
    }))
  }, [])

  // 打开新建/编辑对话框
  const openModal = useCallback(
    (task: VideoRecordingTask | null = null) => {
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
    },
    [form]
  )

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

      // 处理华为配置：支持单选和多选
      let configIds: number[] = []
      if (Array.isArray(values.huawei_config_id)) {
        configIds = values.huawei_config_id
      } else if (values.huawei_config_id) {
        configIds = [values.huawei_config_id]
      }

      // 验证配置类型限制
      const selectedConfigs = huaweiConfigs.filter((c) => configIds.includes(c.id))
      const usbCount = selectedConfigs.filter((c) => getConfigType(c) === 'usb').length
      const streamCount = selectedConfigs.filter((c) => getConfigType(c) === 'stream').length

      if (usbCount > 1) {
        message.error('最多只能选择1个USB配置')
        return
      }
      if (streamCount > 1) {
        message.error('最多只能选择1个流媒体配置')
        return
      }

      const requestData = {
        ...values,
        huawei_config_id: Array.isArray(values.huawei_config_id)
          ? values.huawei_config_id[0]
          : values.huawei_config_id,
        input_config_ids: Array.isArray(values.huawei_config_id)
          ? values.huawei_config_id
          : values.huawei_config_id
            ? [values.huawei_config_id]
            : [],
        start_time: values.start_time.toISOString(),
        end_time: values.end_time.toISOString(),
      }

      if (editingTask) {
        const isRecording = editingTask.status === 'recording'
        const req: UpdateTaskRequest = isRecording
          ? { end_time: requestData.end_time }
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
    } catch (error: any) {
      if (error?.errorFields) {
        const firstError = error.errorFields[0]
        const fieldName = firstError?.name?.[0] || '字段'
        const errorMessage = firstError?.errors?.[0] || '验证失败'
        message.error(`${fieldName}: ${errorMessage}`)
      } else {
        message.error(error instanceof Error ? error.message : '操作失败')
      }
    }
  }, [editingTask, form, closeModal, loadTasks, huaweiConfigs])

  // 删除任务
  const handleDelete = useCallback(
    async (id: number) => {
      try {
        await taskApi.deleteTask(id)
        message.success('删除成功')
        loadTasks()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '删除失败')
      }
    },
    [loadTasks]
  )

  // 启动任务
  const handleStart = useCallback(
    async (id: number) => {
      try {
        await taskApi.startTask(id)
        message.success('任务启动成功')
        loadTasks()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '启动失败')
      }
    },
    [loadTasks]
  )

  // 停止任务
  const handleStop = useCallback(
    async (id: number) => {
      try {
        await taskApi.stopTask(id)
        message.success('任务停止成功')
        loadTasks()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '停止失败')
      }
    },
    [loadTasks]
  )

  // 取消任务
  const handleCancel = useCallback(
    async (id: number) => {
      try {
        await taskApi.cancelTask(id)
        message.success('任务取消成功')
        loadTasks()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '取消失败')
      }
    },
    [loadTasks]
  )

  // 重试任务
  const handleRetry = useCallback(
    async (id: number) => {
      try {
        await taskApi.retryTask(id)
        message.success('任务重试成功')
        loadTasks()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '重试失败')
      }
    },
    [loadTasks]
  )

  // 生成MP4快照
  const handleGenerateSnapshot = useCallback(async (id: number) => {
    setSnapshotGenerating((prev) => new Set(prev).add(id))
    try {
      const result = await apiRequest<{ snapshot_file_id: number; file_name: string }>(
        `/api/v1/tasks/${id}/snapshot`,
        { method: 'POST' }
      )
      if (result.data) {
        message.success(`快照已生成：${result.data.file_name}`)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '生成快照失败')
    } finally {
      setSnapshotGenerating((prev) => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }, [])

  // 批量删除任务
  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的任务')
      return
    }

    const cannotDeleteTasks = tasks.filter(
      (task) => selectedRowKeys.includes(task.id) && !DELETABLE_STATUSES.includes(task.status)
    )

    if (cannotDeleteTasks.length > 0) {
      const cannotDeleteNames = cannotDeleteTasks.map((t) => t.name).join('、')
      message.warning(
        `以下任务无法删除：${cannotDeleteNames}\n只能删除待执行、已完成、失败、已取消状态的任务`
      )
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

  // 清理卡住的任务
  const handleClearStuckTasks = useCallback(async () => {
    Modal.confirm({
      title: '清理卡住的任务',
      content:
        '此操作将所有"转换中"状态超过30分钟的任务标记为失败，并释放相关的华为终端锁。是否继续？',
      okText: '确定清理',
      cancelText: '取消',
      onOk: async () => {
        try {
          const response = await taskApi.clearStuckTasks(30)
          if (response.data) {
            const result = response.data
            if (result.total_cleared > 0) {
              message.success(
                `已清理 ${result.total_cleared} 个卡住的任务，释放了 ${result.total_unlocked} 个终端锁`
              )
            } else {
              message.info('没有发现卡住的任务')
            }
            loadTasks()
          }
        } catch (error) {
          message.error(error instanceof Error ? error.message : '清理失败')
        }
      },
    })
  }, [loadTasks])

  // 获取可删除的任务 ID 列表 (rerender-derived-state)
  const deletableTaskIds = useMemo(() => {
    return new Set(
      tasks.filter((task) => DELETABLE_STATUSES.includes(task.status)).map((task) => task.id)
    )
  }, [tasks])

  // 行选择配置
  const rowSelection = useMemo(
    () => ({
      selectedRowKeys,
      onChange: (newSelectedRowKeys: React.Key[]) => {
        setSelectedRowKeys(newSelectedRowKeys)
      },
      getCheckboxProps: (record: VideoRecordingTask) => ({
        disabled: !deletableTaskIds.has(record.id),
        name: record.name,
      }),
    }),
    [selectedRowKeys, deletableTaskIds]
  )

  // 渲染状态标签
  const renderStatus = useCallback((status: string) => {
    const config = STATUS_CONFIG[status as keyof typeof STATUS_CONFIG]
    return config ? <Tag color={config.color}>{config.label}</Tag> : null
  }, [])

  // 渲染操作按钮 - 使用稳定的回调引用
  const renderActions = useCallback(
    (record: VideoRecordingTask) => {
      return (
        <TaskActions
          record={record}
          onStart={handleStart}
          onStop={handleStop}
          onCancel={handleCancel}
          onRetry={handleRetry}
          onDelete={handleDelete}
          onEdit={openModal}
          onGenerateSnapshot={handleGenerateSnapshot}
          snapshotGenerating={snapshotGenerating}
        />
      )
    },
    [
      handleStart,
      handleStop,
      handleCancel,
      handleRetry,
      handleDelete,
      openModal,
      handleGenerateSnapshot,
      snapshotGenerating,
    ]
  )

  // 渲染配置类型标签
  const renderConfigTypeTag = useCallback((config: InputConfig): React.ReactElement => {
    const tagConfig = getConfigTypeTagConfig(config)
    return <Tag color={tagConfig.color}>{tagConfig.label}</Tag>
  }, [])

  // 表格列定义
  const columns: ColumnsType<VideoRecordingTask> = useMemo(
    () => [
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
        render: (msg) =>
          msg ? (
            <Tooltip title={msg}>
              <span style={{ color: 'red' }}>{msg}</span>
            </Tooltip>
          ) : (
            '-'
          ),
      },
      {
        title: '操作',
        key: 'action',
        width: 200,
        fixed: 'right' as const,
        render: (_: unknown, record: VideoRecordingTask) => renderActions(record),
      },
    ],
    [renderStatus, renderActions]
  )

  // 录制中状态提示内容 (rendering-hoist-jsx)
  const RECORDING_TIP = (
    <div style={{ marginBottom: 16, padding: 12, background: '#e6f7ff', borderRadius: 4 }}>
      任务正在录制中，仅可修改结束时间
    </div>
  )

  // D-05.1 / D-05.2 — 空态与错误态分支
  const isFiltered = Boolean(
    params.keyword || params.status || params.start_date || params.end_date
  )
  const showErrorState = !loading && Boolean(loadError)
  const showEmptyState = !loading && !loadError && tasks.length === 0

  // D-05.2 — 加载失败：带回具体原因 + 重试
  const errorState = (
    <Empty
      style={{ padding: '48px 24px' }}
      image={<ErrorNetwork style={{ width: 180, height: 126, color: designTokens.colors.error }} />}
      styles={{ image: { height: 126 } }}
      description={
        <div>
          <div style={{ color: designTokens.colors.text.primary }}>加载失败：{loadError}</div>
          <div style={{ color: designTokens.colors.muted, fontSize: 13, marginTop: 4 }}>
            检查网络或稍后再试
          </div>
        </div>
      }
    >
      <Button type="primary" icon={<ReloadOutlined />} onClick={() => loadTasks(true)}>
        重试
      </Button>
    </Empty>
  )

  // D-05.1 — 空态：一句话文案 + 一个主操作
  const emptyState = isFiltered ? (
    <Empty
      style={{ padding: '48px 24px' }}
      image={<EmptyTasks style={{ width: 180, height: 126, color: designTokens.colors.muted }} />}
      styles={{ image: { height: 126 } }}
      description={
        <div>
          <div style={{ color: designTokens.colors.text.primary }}>没有匹配的任务</div>
          <div style={{ color: designTokens.colors.muted, fontSize: 13, marginTop: 4 }}>
            换个关键词，或清空筛选条件
          </div>
        </div>
      }
    >
      <Button type="primary" onClick={handleClearFilters}>
        清空筛选
      </Button>
    </Empty>
  ) : (
    <Empty
      style={{ padding: '48px 24px' }}
      image={<EmptyTasks style={{ width: 180, height: 126, color: designTokens.colors.muted }} />}
      styles={{ image: { height: 126 } }}
      description={
        <div>
          <div style={{ color: designTokens.colors.text.primary }}>还没有录制任务</div>
          <div style={{ color: designTokens.colors.muted, fontSize: 13, marginTop: 4 }}>
            填会议号和时间，到点自动开录
          </div>
        </div>
      }
    >
      <PermissionGuard permission={PERMISSIONS.TASK_CREATE}>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
          开始录制
        </Button>
      </PermissionGuard>
    </Empty>
  )

  return (
    <div className="page-container">
      <div className="page-header">
        <h2>录制任务管理</h2>
        <Space>
          <PermissionGuard permission={PERMISSIONS.TASK_DELETE}>
            {selectedRowKeys.length > 0 ? (
              <Button
                danger
                icon={<DeleteOutlined />}
                loading={batchDeleteLoading}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            ) : null}
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
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Select
            placeholder="选择状态"
            allowClear
            style={{ width: 120 }}
            value={params.status}
            onChange={handleStatusFilter}
            options={STATUS_OPTIONS}
          />
          <RangePicker
            placeholder={['开始日期', '结束日期']}
            value={dateRange}
            onChange={handleDateRangeChange}
          />
          <Button icon={<ReloadOutlined />} onClick={() => loadTasks(true)}>
            刷新
          </Button>
          <PermissionGuard permission={PERMISSIONS.TASK_DELETE}>
            <Button icon={<ClearOutlined />} onClick={handleClearStuckTasks}>
              清理卡住任务
            </Button>
          </PermissionGuard>
        </Space>
      </div>

      {showErrorState ? (
        errorState
      ) : showEmptyState ? (
        emptyState
      ) : (
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
      )}

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
        destroyOnHidden
      >
        <Form form={form} layout="vertical">
          {/* 录制中状态只能编辑结束时间，其他字段被禁用并显示提示 */}
          {editingTask && editingTask.status === 'recording' ? RECORDING_TIP : null}

          <Form.Item
            name="name"
            label="任务名称"
            rules={[
              {
                required: !editingTask || canEditAllFields(editingTask.status),
                message: '请输入任务名称',
              },
              { max: 200, message: '任务名称最多200个字符' },
            ]}
          >
            <Input
              placeholder="例：周例会（2026-07-28）"
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 500, message: '描述最多500个字符' }]}
          >
            <Input.TextArea
              placeholder="会议主题、参会人或备注"
              rows={3}
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="conference_number"
            label="会议号"
            rules={[
              {
                required: !editingTask || canEditAllFields(editingTask.status),
                message: '请输入会议号',
              },
              { max: 50, message: '会议号最多50个字符' },
            ]}
          >
            <Input
              placeholder="华为会议号，如 987654321"
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="huawei_config_id"
            label="输入配置（可选，最多选一路USB和一路流媒体）"
            rules={[
              {
                validator: async (_, value) => {
                  // 输入配置是可选的，如果用户选择了配置则验证
                  if (value && (Array.isArray(value) ? value.length > 0 : value)) {
                    const ids = Array.isArray(value) ? value : [value]
                    const selectedConfigs = huaweiConfigs.filter((c) => ids.includes(c.id))
                    const usbCount = selectedConfigs.filter(
                      (c) => getConfigType(c) === 'usb'
                    ).length
                    const streamCount = selectedConfigs.filter(
                      (c) => getConfigType(c) === 'stream'
                    ).length

                    if (usbCount > 1) {
                      throw new Error('最多只能选择1个USB配置')
                    }
                    if (streamCount > 1) {
                      throw new Error('最多只能选择1个流媒体配置')
                    }

                    const invalidConfigs = selectedConfigs.filter(
                      (c) => getConfigType(c) === 'none'
                    )
                    if (invalidConfigs.length > 0) {
                      throw new Error(`配置"${invalidConfigs[0].name}"未配置USB设备或流媒体`)
                    }
                  }
                },
              },
            ]}
          >
            <Select
              mode="multiple"
              placeholder="最多一路 USB + 一路流媒体"
              loading={configsLoading}
              showSearch
              optionFilterProp="label"
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
              tagRender={(props) => {
                const { label, value, onClose } = props
                const config = huaweiConfigs.find((c) => c.id === value)
                const tagConfig = config ? getConfigTypeTagConfig(config) : CONFIG_TYPE_TAGS.none

                return (
                  <Tag
                    color={tagConfig.color}
                    closable
                    onClose={onClose}
                    style={{ marginRight: 3 }}
                  >
                    {label}
                  </Tag>
                )
              }}
            >
              {huaweiConfigs.map((config) => {
                const configType = getConfigType(config)
                // 根据配置类型显示不同信息
                const detailInfo =
                  configType === 'usb'
                    ? `${config.usb_camera_device || '无摄像头'}`
                    : configType === 'stream'
                      ? `${config.stream_protocol || 'RTMP'}:${config.stream_url || '无地址'}`
                      : `${config.server || '无服务器'}:${config.port || 80}`

                return (
                  <Select.Option key={config.id} value={config.id}>
                    <Space>
                      {config.name} ({detailInfo}){renderConfigTypeTag(config)}
                    </Space>
                  </Select.Option>
                )
              })}
            </Select>
          </Form.Item>

          <Space size="large">
            <Form.Item
              name="start_time"
              label="开始时间"
              rules={[
                {
                  required: !editingTask || canEditAllFields(editingTask.status),
                  message: '请选择开始时间',
                },
              ]}
            >
              <DatePicker
                showTime
                format="YYYY-MM-DD HH:mm:ss"
                disabled={!!editingTask && !canEditAllFields(editingTask.status)}
              />
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
              <Input
                type="number"
                style={{ width: 120 }}
                disabled={!!editingTask && !canEditAllFields(editingTask.status)}
              />
            </Form.Item>

            <Form.Item
              name="record_delay_minutes"
              label="录制延迟(分钟)"
              rules={[{ type: 'number', min: 0, max: 60, message: '录制延迟必须在0-60分钟之间' }]}
              initialValue={DEFAULT_RECORD_DELAY_MINUTES}
            >
              <Input
                type="number"
                style={{ width: 120 }}
                disabled={!!editingTask && !canEditAllFields(editingTask.status)}
              />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  )
}
