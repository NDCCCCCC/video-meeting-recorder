// 文件管理页面

import { useState, useEffect, useCallback, lazy, Suspense } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Modal,
  message,
  Popconfirm,
  Tag,
  Select,
  Card,
  Statistic,
  Row,
  Col,
  Tooltip,
  Radio,
  Dropdown,
  Spin,
} from 'antd'
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  FileOutlined,
  FolderOpenOutlined,
  VideoCameraOutlined,
  EyeOutlined,
  ScanOutlined,
  ScissorOutlined,
  FilePptOutlined,
  CloudOutlined,
  LaptopOutlined,
  EditOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { useNavigate } from 'react-router-dom'
import * as videoFileApi from '../../api/video-file'
import { submitTranscriptionWithMode, getActiveTranscriptionTasks } from '../../api/transcription'
import { getPptsByVideo } from '../../api/ppt'
import { PermissionGuard } from '../../components/PermissionGuard'
import { PERMISSIONS } from '../../utils/permissions'
import { RenderVideoPreview } from '../../components/VideoPlayerSimple'
const TranscriptionProgressModal = lazy(() => import('../../components/TranscriptionProgressModal'))
import VideoUploadModal from '../../components/VideoUploadModal'
import type {
  VideoFile,
  VideoFileListParams,
  VideoFileStats,
  VideoFileStatus,
} from '../../types/video-file'
import type { SamplingRateOption, TranscriptionMode } from '../../types/transcription'

// 状态配置
const STATUS_CONFIG: Record<VideoFileStatus, { label: string; color: string }> = {
  ready: { label: '就绪', color: 'success' },
  processing: { label: '处理中', color: 'processing' },
  error: { label: '错误', color: 'error' },
  deleting: { label: '删除中', color: 'default' },
}

const STATUS_OPTIONS = Object.entries(STATUS_CONFIG).map(([value, { label }]) => ({
  label,
  value,
}))

// 默认分页大小
const DEFAULT_PAGE_SIZE = 20
// 默认文件格式筛选
const DEFAULT_FORMAT = 'mp4'

// 采样率选项 (per D-02, 增加更高精度选项)
const samplingRateOptions: SamplingRateOption[] = [
  { label: '0.05秒/帧', value: 0.05, secondsPerFrame: 0.05, description: '极高精度 (20fps), 文件很大' },
  { label: '0.1秒/帧', value: 0.1, secondsPerFrame: 0.1, description: '很高精度 (10fps), 文件较大' },
  { label: '0.2秒/帧', value: 0.2, secondsPerFrame: 0.2, description: '高精度 (5fps)' },
  { label: '0.5秒/帧', value: 0.5, secondsPerFrame: 0.5, description: '推荐 (2fps)' },
  { label: '1秒/帧', value: 1.0, secondsPerFrame: 1, description: '标准 (1fps), 文件较小' },
]

// 格式化文件大小
const formatFileSize = (bytes: number): string => `${(bytes / 1024 / 1024).toFixed(2)} MB`

// 格式化时长
const formatDuration = (seconds: number): string => {
  const mins = Math.floor(seconds / 60)
  const secs = seconds % 60
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

export default function FileManagement() {
  const navigate = useNavigate()
  const [files, setFiles] = useState<VideoFile[]>([])
  const [stats, setStats] = useState<VideoFileStats | null>(null)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [viewingFile, setViewingFile] = useState<VideoFile | null>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [batchDeleting, setBatchDeleting] = useState(false)

  // Transcription state
  const [transcriptionModalOpen, setTranscriptionModalOpen] = useState(false)
  const [transcriptionVideoFile, setTranscriptionVideoFile] = useState<VideoFile | null>(null)
  const [triggerModalOpen, setTriggerModalOpen] = useState(false)
  const [selectedSamplingRate, setSelectedSamplingRate] = useState<number>(0.5) // default 2s per D-02
  const [triggerLoading, setTriggerLoading] = useState(false)
  const [cloudTranscriptionMode, setCloudTranscriptionMode] = useState<TranscriptionMode>('local')

  // PPT results cache - track which videos have PPT results
  const [videosWithPpt, setVideosWithPpt] = useState<Set<number>>(new Set())

  // Active transcription tasks cache - track which videos have active tasks
  const [activeTranscriptions, setActiveTranscriptions] = useState<Map<number, { mode: string; samplingRate: number }>>(new Map())

  // Rename modal state
  const [renameModalVisible, setRenameModalVisible] = useState(false)
  const [renamingFile, setRenamingFile] = useState<VideoFile | null>(null)
  const [newFileName, setNewFileName] = useState('')
  const [renameLoading, setRenameLoading] = useState(false)

  // Upload modal state
  const [uploadModalVisible, setUploadModalVisible] = useState(false)

  const [params, setParams] = useState<VideoFileListParams>({
    page: 1,
    page_size: DEFAULT_PAGE_SIZE,
    format: DEFAULT_FORMAT,
  })

  // 加载文件列表
  const loadFiles = useCallback(async (showLoading = true) => {
    if (showLoading) setLoading(true)
    try {
      const response = await videoFileApi.getVideoFileList(params)
      if (response.data) {
        setFiles(response.data.items)
        setTotal(response.data.total)
      }
    } catch (error) {
      if (showLoading) {
        message.error(error instanceof Error ? error.message : '加载文件列表失败')
      }
    } finally {
      if (showLoading) setLoading(false)
    }
  }, [params])

  // 加载统计信息
  const loadStats = useCallback(async () => {
    try {
      const response = await videoFileApi.getVideoFileStats()
      if (response.data) {
        setStats(response.data)
      }
    } catch (error) {
      // Stats loading failure is non-critical, silently ignore
    }
  }, [])

  // 检查视频是否有 PPT 结果
  const checkHasPpt = useCallback(async (videoFileId: number) => {
    // 先检查缓存
    if (videosWithPpt.has(videoFileId)) {
      return true
    }

    try {
      const response = await getPptsByVideo(videoFileId)
      if (response.data && response.data.ppts && response.data.ppts.length > 0) {
        // 添加到缓存
        setVideosWithPpt(prev => new Set(prev).add(videoFileId))
        return true
      }
      return false
    } catch (error) {
      // PPT check failure is non-critical, silently ignore
      return false
    }
  }, [videosWithPpt])

  // 加载活跃的转录任务
  const loadActiveTranscriptions = useCallback(async () => {
    try {
      const response = await getActiveTranscriptionTasks()
      if (response.data && response.data.tasks) {
        const activeMap = new Map<number, { mode: string; samplingRate: number }>()
        for (const task of response.data.tasks) {
          activeMap.set(task.video_file_id, {
            mode: task.mode,
            samplingRate: task.sampling_rate,
          })
        }
        setActiveTranscriptions(activeMap)
      }
    } catch (error) {
      // 静默失败
    }
  }, [])

  // 初始加载 - 并行执行 loadFiles、loadStats 和 loadActiveTranscriptions
  useEffect(() => {
    Promise.all([loadFiles(), loadStats(), loadActiveTranscriptions()])
  }, [loadFiles, loadStats, loadActiveTranscriptions])

  // 检查当前页面的视频是否有 PPT 结果
  useEffect(() => {
    const checkPptResults = async () => {
      const checks = files.map(async (file) => {
        if (file.format === 'mp4' && file.status === 'ready') {
          await checkHasPpt(file.id)
        }
      })
      await Promise.all(checks)
    }

    if (files.length > 0) {
      checkPptResults()
    }
  }, [files, checkHasPpt])

  // 自动刷新文件列表 (SCAN-02)
  useEffect(() => {
    const interval = setInterval(() => {
      loadFiles(false) // silent refresh
    }, 5000) // 5 seconds
    return () => clearInterval(interval)
  }, [loadFiles])

  // 搜索处理
  const handleSearch = useCallback((value: string) => {
    setParams(prev => ({ ...prev, keyword: value, page: 1 }))
  }, [])

  // 状态筛选
  const handleStatusFilter = useCallback((status: VideoFileStatus | undefined) => {
    setParams(prev => ({ ...prev, status, page: 1 }))
  }, [])

  // 分页变化
  const handleTableChange = useCallback((pagination: { current?: number; pageSize?: number }) => {
    setParams(prev => ({
      ...prev,
      page: pagination.current ?? 1,
      page_size: pagination.pageSize ?? 20,
    }))
  }, [])

  // 下载文件
  const handleDownload = useCallback(async (id: number, fileName: string) => {
    try {
      await videoFileApi.downloadVideoFile(id, fileName)
      message.success(`开始下载 ${fileName}`)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '下载失败')
    }
  }, [])

  // 删除文件
  const handleDelete = useCallback(async (id: number) => {
    try {
      await videoFileApi.deleteVideoFile(id)
      message.success('删除成功')
      loadFiles()
      loadStats()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }, [loadFiles, loadStats])

  // 批量删除文件
  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的文件')
      return
    }

    setBatchDeleting(true)
    try {
      const response = await videoFileApi.batchDeleteFiles(selectedRowKeys as number[])
      if (response.data) {
        const { success, failed, errors } = response.data
        if (failed > 0) {
          // 合并错误消息，避免弹出多个 message
          const errorSummary = errors.length > 0
            ? `。错误详情：${errors.slice(0, 3).join('; ')}${errors.length > 3 ? '...' : ''}`
            : ''
          message.warning(`删除完成：成功 ${success} 个，失败 ${failed} 个${errorSummary}`)
        } else {
          message.success(`成功删除 ${success} 个文件`)
        }
        setSelectedRowKeys([])
        loadFiles()
        loadStats()
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '批量删除失败')
    } finally {
      setBatchDeleting(false)
    }
  }, [selectedRowKeys, loadFiles, loadStats])

  // 查看详情
  const viewDetail = useCallback((file: VideoFile) => {
    setViewingFile(file)
    setDetailVisible(true)
  }, [])

  // 重命名文件
  const handleRename = useCallback((file: VideoFile) => {
    setRenamingFile(file)
    // Strip extension from filename for editing
    const nameWithoutExt = file.file_name.replace(/\.[^/.]+$/, '')
    setNewFileName(nameWithoutExt)
    setRenameModalVisible(true)
  }, [])

  const confirmRename = useCallback(async () => {
    if (!renamingFile || !newFileName.trim()) {
      message.warning('请输入文件名')
      return
    }

    // Check if name hasn't changed
    const nameWithoutExt = renamingFile.file_name.replace(/\.[^/.]+$/, '')
    if (newFileName.trim() === nameWithoutExt) {
      message.info('文件名未改变')
      setRenameModalVisible(false)
      return
    }

    setRenameLoading(true)
    try {
      await videoFileApi.renameVideoFile(renamingFile.id, newFileName.trim())
      message.success('重命名成功')
      setRenameModalVisible(false)
      loadFiles(false) // Refresh list without loading indicator
    } catch (error) {
      message.error(error instanceof Error ? error.message : '重命名失败')
    } finally {
      setRenameLoading(false)
    }
  }, [renamingFile, newFileName, loadFiles])


  // 扫描导入
  const handleScan = useCallback(async () => {
    setScanning(true)
    try {
      const response = await videoFileApi.scanVideoFiles()
      if (response.data) {
        const { scanned, created, skipped } = response.data
        if (created > 0) {
          message.success(`扫描完成！发现 ${scanned} 个文件，新增 ${created} 条记录，跳过 ${skipped} 个`)
        } else {
          message.info(`扫描完成！发现 ${scanned} 个文件，但都是已存在的记录`)
        }
        loadFiles()
        loadStats()
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '扫描失败')
    } finally {
      setScanning(false)
    }
  }, [loadFiles, loadStats])

  // 转录相关处理函数
  // 打开触发模态框 (采样率选择)
  const handleTranscribeClick = useCallback((record: VideoFile) => {
    setTranscriptionVideoFile(record)
    setSelectedSamplingRate(0.5) // reset to default 2s per D-02
    setCloudTranscriptionMode('local') // Set mode to local
    setTriggerModalOpen(true)
  }, [])

  // 提交转录任务
  const handleTranscriptionSubmit = useCallback(async () => {
    if (!transcriptionVideoFile) return
    setTriggerLoading(true)
    try {
      await submitTranscriptionWithMode(transcriptionVideoFile.id, 'local', selectedSamplingRate)
      setTriggerModalOpen(false)
      setTranscriptionModalOpen(true) // open progress modal
    } catch (err) {
      message.error(err instanceof Error ? err.message : '提交转录任务失败')
    } finally {
      setTriggerLoading(false)
    }
  }, [transcriptionVideoFile, selectedSamplingRate])

  // 云端转录处理 (per D-01, D-03)
  // Per D-03: cloud mode starts immediately -- no sampling rate step
  // submitTranscriptionWithMode with mode='cloud' does NOT send sampling_rate
  const handleCloudTranscribe = useCallback(async (record: VideoFile) => {
    setTranscriptionVideoFile(record)
    setCloudTranscriptionMode('cloud')
    try {
      await submitTranscriptionWithMode(record.id, 'cloud')
      setTranscriptionModalOpen(true) // Open progress modal directly (no sampling rate step per D-03)
    } catch (err) {
      message.error(err instanceof Error ? err.message : '提交云端转录任务失败')
    }
  }, [])

  // 转录完成回调
  const handleTranscriptionCompleted = useCallback(() => {
    loadFiles()
    loadStats()
  }, [loadFiles, loadStats])

  // 渲染状态标签 - 简单渲染函数不需要 memoization
  function renderStatus(status: VideoFileStatus) {
    const config = STATUS_CONFIG[status]
    return <Tag color={config.color}>{config.label}</Tag>
  }

  // 渲染操作按钮 - 作为 Table column 的 render prop 不需要 useCallback
  function renderActions(record: VideoFile) {
    // 构建"更多"操作菜单
    const moreMenuItems: any[] = []

    // 下载
    moreMenuItems.push({
      key: 'download',
      icon: <DownloadOutlined />,
      label: '下载文件',
      disabled: record.status !== 'ready',
      onClick: () => handleDownload(record.id, record.file_name),
    })

    // 转录相关（仅 mp4 格式）
    if (record.format === 'mp4') {
      if (activeTranscriptions.has(record.id)) {
        // 有活跃转录任务
        moreMenuItems.push({
          key: 'view-transcription',
          icon: <CloudOutlined />,
          label: '查看转录进度',
          onClick: () => {
            const taskInfo = activeTranscriptions.get(record.id)!
            setTranscriptionVideoFile(record)
            setCloudTranscriptionMode(taskInfo.mode as TranscriptionMode)
            setSelectedSamplingRate(taskInfo.samplingRate)
            setTranscriptionModalOpen(true)
          },
        })
      } else if (record.status === 'ready') {
        // 可以开始转录
        moreMenuItems.push({
          key: 'transcribe-submenu',
          icon: <CloudOutlined />,
          label: '开始转录',
          children: [
            {
              key: 'local',
              icon: <LaptopOutlined />,
              label: '本地转录',
              onClick: () => handleTranscribeClick(record),
            },
            {
              key: 'cloud',
              icon: <CloudOutlined />,
              label: '云端转录（通义听悟）',
              onClick: () => handleCloudTranscribe(record),
            },
          ],
        })
      }
    }

    // 预览PPT
    if (videosWithPpt.has(record.id)) {
      moreMenuItems.push({
        key: 'preview-ppt',
        icon: <FilePptOutlined />,
        label: '预览PPT',
        onClick: () => navigate(`/results/${record.id}`),
      })
    }

    // 分隔线
    moreMenuItems.push({ type: 'divider' })

    // 视频分割（仅 mp4 格式）
    if (record.format === 'mp4') {
      moreMenuItems.push({
        key: 'split',
        icon: <ScissorOutlined />,
        label: '视频分割',
        onClick: () => navigate(`/split/${record.id}`),
      })
    }

    // 重命名（非原始录制文件）
    if (record.source_type !== 'recording' || record.parent_id) {
      moreMenuItems.push({
        key: 'rename',
        icon: <EditOutlined />,
        label: '重命名',
        disabled: record.status !== 'ready',
        onClick: () => handleRename(record),
      })
    }

    // 删除
    moreMenuItems.push({
      key: 'delete',
      icon: <DeleteOutlined />,
      label: '删除',
      danger: true,
      disabled: record.status === 'processing',
      onClick: () => {
        Modal.confirm({
          title: '确定要删除这个文件吗？',
          onOk: () => handleDelete(record.id),
        })
      },
    })

    return (
      <Space size="small">
        <RenderVideoPreview {...record} />
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => viewDetail(record)}
        >
          详情
        </Button>
        <Dropdown
          menu={{ items: moreMenuItems }}
          trigger={['click']}
        >
          <Button size="small">
            更多
          </Button>
        </Dropdown>
      </Space>
    )
  }

  // 表格列定义 - 移除 useMemo，因为 renderActions 依赖的值频繁变化
  const columns: ColumnsType<VideoFile> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '文件名',
      dataIndex: 'file_name',
      width: 250,
      ellipsis: true,
      render: (name: string) => (
        <Space>
          <VideoCameraOutlined />
          <Tooltip title={name}>{name}</Tooltip>
        </Space>
      ),
    },
    {
      title: '大小',
      dataIndex: 'file_size',
      width: 100,
      render: formatFileSize,
    },
    {
      title: '时长',
      dataIndex: 'duration',
      width: 100,
      render: formatDuration,
    },
    {
      title: '格式',
      dataIndex: 'format',
      width: 80,
    },
    {
      title: '分辨率',
      dataIndex: 'resolution',
      width: 100,
    },
    {
      title: '来源',
      dataIndex: 'source_type',
      width: 120,
      render: (sourceType: string, record: VideoFile) => {
        const SOURCE_CONFIG: Record<string, { label: string; color: string }> = {
          recording: { label: '录制', color: 'blue' },
          snapshot: { label: '快照', color: 'green' },
          split: { label: '分割', color: 'orange' },
        }
        const config = SOURCE_CONFIG[sourceType] || SOURCE_CONFIG.recording
        return (
          <Space direction="vertical" size={2}>
            <Tag color={config.color}>{config.label}</Tag>
            {record.parent_id && (
              <Button
                type="link"
                size="small"
                style={{ padding: 0, fontSize: 12 }}
                onClick={() => navigate(`/files`)}
              >
                查看原视频
              </Button>
            )}
          </Space>
        )
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: renderStatus,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      width: 160,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 250,
      fixed: 'right' as const,
      render: renderActions,
    },
  ]

  const isReady = viewingFile?.status === 'ready'

  return (
    <div style={{ padding: '20px' }}>
      <div style={{ marginBottom: '16px' }}>
        <h2 style={{ margin: 0, marginBottom: '16px' }}>
          <FolderOpenOutlined /> 文件管理
        </h2>

        {/* 统计卡片 */}
        {stats && (
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={6}>
              <Card>
                <Statistic title="文件总数" value={stats.total} prefix={<FileOutlined />} />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总大小"
                  value={stats.total_size_gb.toFixed(2)}
                  suffix="GB"
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="平均大小"
                  value={stats.total > 0 ? (stats.total_size_gb / stats.total).toFixed(2) : 0}
                  suffix="GB"
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic title="当前页" value={files.length} suffix="/ 条" />
              </Card>
            </Col>
          </Row>
        )}
      </div>

      <div style={{ marginBottom: '16px' }}>
        <Space size="middle" wrap>
          <Input.Search
            placeholder="搜索文件名或路径"
            allowClear
            style={{ width: 300 }}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Select
            placeholder="筛选状态"
            allowClear
            style={{ width: 120 }}
            onChange={handleStatusFilter}
            options={STATUS_OPTIONS}
          />
          <Button icon={<ReloadOutlined />} onClick={() => { loadFiles(); loadStats() }}>
            刷新
          </Button>
          {selectedRowKeys.length > 0 && (
            <PermissionGuard permission={PERMISSIONS.FILE_DELETE}>
              <Popconfirm
                title={`确定要删除选中的 ${selectedRowKeys.length} 个文件吗？`}
                onConfirm={handleBatchDelete}
                okText="确定"
                cancelText="取消"
                okType="danger"
              >
                <Button
                  danger
                  icon={<DeleteOutlined />}
                  loading={batchDeleting}
                >
                  批量删除 ({selectedRowKeys.length})
                </Button>
              </Popconfirm>
            </PermissionGuard>
          )}
          <PermissionGuard permission={PERMISSIONS.FILE_SCAN}>
            <Button
              type="primary"
              icon={<ScanOutlined />}
              onClick={handleScan}
              loading={scanning}
            >
              扫描导入
            </Button>
          </PermissionGuard>
          <Button
            type="primary"
            icon={<UploadOutlined />}
            onClick={() => setUploadModalVisible(true)}
          >
            上传视频
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={files}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1400 }}
        rowSelection={{
          selectedRowKeys,
          onChange: (selectedKeys) => setSelectedRowKeys(selectedKeys),
          getCheckboxProps: (record: VideoFile) => ({
            disabled: record.status === 'processing',
          }),
        }}
        pagination={{
          current: params.page,
          pageSize: params.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        onChange={handleTableChange}
      />

      {/* 文件详情对话框 */}
      <Modal
        title={<Space><FileOutlined />文件详情 - {viewingFile?.file_name}</Space>}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailVisible(false)}>
            关闭
          </Button>,
          isReady && (
            <Button
              key="download"
              type="primary"
              icon={<DownloadOutlined />}
              onClick={() => viewingFile && handleDownload(viewingFile.id, viewingFile.file_name)}
            >
              下载
            </Button>
          ),
        ]}
        width={700}
      >
        {viewingFile && (
          <>
            <Card size="small" style={{ marginBottom: 16 }}>
              <Row gutter={16}>
                <Col span={12}>
                  <Statistic title="文件大小" value={formatFileSize(viewingFile.file_size)} />
                </Col>
                <Col span={12}>
                  <Statistic title="时长" value={formatDuration(viewingFile.duration)} />
                </Col>
              </Row>
            </Card>

            <Card size="small" title="基本信息">
              <Row gutter={[16, 8]}>
                <Col span={8}><strong>文件ID:</strong></Col>
                <Col span={16}>{viewingFile.id}</Col>

                <Col span={8}><strong>文件名:</strong></Col>
                <Col span={16}>{viewingFile.file_name}</Col>

                <Col span={8}><strong>文件路径:</strong></Col>
                <Col span={16} style={{ wordBreak: 'break-all' }}>{viewingFile.file_path}</Col>

                <Col span={8}><strong>格式:</strong></Col>
                <Col span={16}>{viewingFile.format}</Col>

                <Col span={8}><strong>分辨率:</strong></Col>
                <Col span={16}>{viewingFile.resolution}</Col>

                <Col span={8}><strong>码率:</strong></Col>
                <Col span={16}>{viewingFile.bitrate} kbps</Col>

                <Col span={8}><strong>编码:</strong></Col>
                <Col span={16}>{viewingFile.codec}</Col>

                <Col span={8}><strong>状态:</strong></Col>
                <Col span={16}>{renderStatus(viewingFile.status)}</Col>

                <Col span={8}><strong>创建时间:</strong></Col>
                <Col span={16}>{dayjs(viewingFile.created_at).format('YYYY-MM-DD HH:mm:ss')}</Col>
              </Row>
            </Card>

            {viewingFile.task && (
              <Card size="small" title="关联任务" style={{ marginTop: 16 }}>
                <Row gutter={[16, 8]}>
                  <Col span={8}><strong>任务ID:</strong></Col>
                  <Col span={16}>{viewingFile.task.id}</Col>

                  <Col span={8}><strong>任务名称:</strong></Col>
                  <Col span={16}>{viewingFile.task.name}</Col>
                </Row>
              </Card>
            )}
          </>
        )}
      </Modal>

      {/* 转录触发模态框 (采样率选择) */}
      <Modal
        title={`本地转录 - ${transcriptionVideoFile?.file_name || ''}`}
        open={triggerModalOpen}
        onCancel={() => setTriggerModalOpen(false)}
        onOk={handleTranscriptionSubmit}
        okText="开始转录"
        cancelText="取消"
        confirmLoading={triggerLoading}
      >
        <div style={{ marginBottom: 16 }}>
          <div style={{ marginBottom: 8 }}>选择采样间隔:</div>
          <Radio.Group
            value={selectedSamplingRate}
            onChange={(e) => setSelectedSamplingRate(e.target.value)}
          >
            {samplingRateOptions.map(opt => (
              <Radio key={opt.value} value={opt.value} style={{ display: 'block', marginBottom: 8 }}>
                {opt.label} ({opt.description})
              </Radio>
            ))}
          </Radio.Group>
        </div>
        <div style={{ color: '#faad14', fontSize: 13 }}>
          提示: 转录过程可能需要几分钟，期间可以关闭此窗口继续使用系统。
        </div>
      </Modal>

      {/* 转录进度模态框 - 使用 Suspense 包裹动态导入 */}
      <Suspense fallback={<Spin tip="加载中..." />}>
        <TranscriptionProgressModal
          open={transcriptionModalOpen}
          onClose={() => setTranscriptionModalOpen(false)}
          videoFileId={transcriptionVideoFile?.id || 0}
          fileName={transcriptionVideoFile?.file_name || ''}
          samplingRate={selectedSamplingRate}
          mode={cloudTranscriptionMode}
          onCompleted={handleTranscriptionCompleted}
        />
      </Suspense>

      {/* 重命名文件对话框 */}
      <Modal
        title="重命名文件"
        open={renameModalVisible}
        onOk={confirmRename}
        onCancel={() => setRenameModalVisible(false)}
        confirmLoading={renameLoading}
        okButtonProps={{ disabled: !newFileName.trim() || newFileName.trim() === renamingFile?.file_name.replace(/\.[^/.]+$/, '') }}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Input
            value={newFileName}
            onChange={(e) => setNewFileName(e.target.value)}
            placeholder="请输入新文件名"
            maxLength={200}
            autoFocus
            onPressEnter={confirmRename}
          />
          {renamingFile && (
            <div style={{ color: '#888', fontSize: 12 }}>
              文件扩展名.{renamingFile.file_name.split('.').pop()} 将自动添加
            </div>
          )}
        </Space>
      </Modal>

      {/* 上传视频模态框 */}
      <VideoUploadModal
        visible={uploadModalVisible}
        onCancel={() => setUploadModalVisible(false)}
        onUploadSuccess={() => {
          loadFiles()
          loadStats()
        }}
      />
    </div>
  )
}
