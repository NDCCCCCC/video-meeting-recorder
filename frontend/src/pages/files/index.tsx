// 文件管理页面

import { useState, useEffect, useCallback, useMemo, lazy, Suspense } from 'react'
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
  Spin,
  Alert,
  Empty,
} from 'antd'
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  FileOutlined,
  FolderOpenOutlined,
  VideoCameraOutlined,
  ScanOutlined,
  UploadOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import { useNavigate } from 'react-router-dom'
import * as videoFileApi from '../../api/video-file'
import {
  submitTranscriptionWithMode,
  getActiveTranscriptionTasks,
  submitBatchTranscription,
} from '../../api/transcription'
import { PermissionGuard } from '../../components/PermissionGuard'
import { PERMISSIONS } from '../../utils/permissions'
const TranscriptionProgressModal = lazy(() => import('../../components/TranscriptionProgressModal'))
import VideoUploadModal from '../../components/VideoUploadModal'
import EmptyFiles from '@/assets/illustrations/EmptyFiles'
import ErrorNetwork from '@/assets/illustrations/ErrorNetwork'
import { designTokens } from '@/styles/theme'
import type {
  VideoFile,
  VideoFileListParams,
  VideoFileStats,
  VideoFileStatus,
} from '../../types/video-file'
import type { TranscriptionMode, BatchTranscriptionRequest } from '../../types/transcription'
import {
  STATUS_CONFIG,
  STATUS_OPTIONS,
  DEFAULT_PAGE_SIZE,
  DEFAULT_FORMAT,
  samplingRateOptions,
} from './constants'
import { formatFileSize, formatDuration } from './utils'
import FileRowActions from './components/FileRowActions'

// 渲染状态标签（纯函数，提升到模块层以保持引用稳定，供 columns useMemo 复用）
function renderStatus(status: VideoFileStatus) {
  const config = STATUS_CONFIG[status]
  return <Tag color={config.color}>{config.label}</Tag>
}

export default function FileManagement() {
  const navigate = useNavigate()
  const [files, setFiles] = useState<VideoFile[]>([])
  const [stats, setStats] = useState<VideoFileStats | null>(null)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  // D-05.2 — 保留加载失败的具体原因，用于错误态展示
  const [loadError, setLoadError] = useState<string | null>(null)
  const [scanning, setScanning] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [viewingFile, setViewingFile] = useState<VideoFile | null>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [batchDeleting, setBatchDeleting] = useState(false)
  const [batchDownloading, setBatchDownloading] = useState(false)

  // Batch transcription state
  const [batchTranscribeModalOpen, setBatchTranscribeModalOpen] = useState(false)
  const [batchTranscribing, setBatchTranscribing] = useState(false)
  const [batchSamplingRate, setBatchSamplingRate] = useState<number>(0.5)
  const [batchTranscribeMode, setBatchTranscribeMode] = useState<TranscriptionMode>('local')

  // Transcription state
  const [transcriptionModalOpen, setTranscriptionModalOpen] = useState(false)
  const [transcriptionVideoFile, setTranscriptionVideoFile] = useState<VideoFile | null>(null)
  const [triggerModalOpen, setTriggerModalOpen] = useState(false)
  const [selectedSamplingRate, setSelectedSamplingRate] = useState<number>(0.5) // default 2s per D-02
  const [triggerLoading, setTriggerLoading] = useState(false)
  const [cloudTranscriptionMode, setCloudTranscriptionMode] = useState<TranscriptionMode>('local')

  // Active transcription tasks cache - track which videos have active tasks
  const [activeTranscriptions, setActiveTranscriptions] = useState<
    Map<number, { mode: string; samplingRate: number }>
  >(new Map())

  // Rename modal state
  const [renameModalVisible, setRenameModalVisible] = useState(false)
  const [renamingFile, setRenamingFile] = useState<VideoFile | null>(null)
  const [newFileName, setNewFileName] = useState('')
  const [renameLoading, setRenameLoading] = useState(false)

  // Upload modal state
  const [uploadModalVisible, setUploadModalVisible] = useState(false)

  // 搜索框受控值（用于"清空筛选"时同步清空输入）
  const [searchInput, setSearchInput] = useState('')

  const [params, setParams] = useState<VideoFileListParams>({
    page: 1,
    page_size: DEFAULT_PAGE_SIZE,
    format: DEFAULT_FORMAT,
  })

  // 加载文件列表
  const loadFiles = useCallback(
    async (showLoading = true) => {
      if (showLoading) setLoading(true)
      try {
        const response = await videoFileApi.getVideoFileList(params)
        if (response.data) {
          setFiles(response.data.items)
          setTotal(response.data.total)
        }
        setLoadError(null)
      } catch (error) {
        const reason = error instanceof Error ? error.message : '网络连接中断'
        setLoadError(reason)
        if (showLoading) {
          message.error(`加载失败：${reason}`)
        }
      } finally {
        if (showLoading) setLoading(false)
      }
    },
    [params]
  )

  // 加载统计信息
  const loadStats = useCallback(async () => {
    try {
      const response = await videoFileApi.getVideoFileStats()
      if (response.data) {
        setStats(response.data)
      }
    } catch (error) {
      console.warn('Failed to load stats:', error)
      // Only show user-facing message for critical errors
      if (error instanceof Error && !error.message.includes('404')) {
        message.warning('无法加载统计信息')
      }
    }
  }, [])

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
    } catch {
      // 静默失败
    }
  }, [])

  // 初始加载 - 并行执行 loadFiles、loadStats 和 loadActiveTranscriptions
  useEffect(() => {
    Promise.all([loadFiles(), loadStats(), loadActiveTranscriptions()])
  }, [loadFiles, loadStats, loadActiveTranscriptions])

  // 自动刷新文件列表 (SCAN-02)
  useEffect(() => {
    const interval = setInterval(() => {
      loadFiles(false) // silent refresh
    }, 5000) // 5 seconds
    return () => clearInterval(interval)
  }, [loadFiles])

  // 搜索处理
  const handleSearch = useCallback((value: string) => {
    setParams((prev) => ({ ...prev, keyword: value, page: 1 }))
  }, [])

  // D-05.1 — 清空筛选条件，回到完整列表
  const handleClearFilters = useCallback(() => {
    setSearchInput('')
    setParams({ page: 1, page_size: DEFAULT_PAGE_SIZE, format: DEFAULT_FORMAT })
  }, [])

  // 状态筛选
  const handleStatusFilter = useCallback((status: VideoFileStatus | undefined) => {
    setParams((prev) => ({ ...prev, status, page: 1 }))
  }, [])

  // 分页变化
  const handleTableChange = useCallback((pagination: { current?: number; pageSize?: number }) => {
    setParams((prev) => ({
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
  const handleDelete = useCallback(
    async (id: number) => {
      try {
        await videoFileApi.deleteVideoFile(id)
        message.success('删除成功')
        loadFiles()
        loadStats()
      } catch (error) {
        message.error(error instanceof Error ? error.message : '删除失败')
      }
    },
    [loadFiles, loadStats]
  )

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
          const errorSummary =
            errors.length > 0
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

  // 批量下载文件
  const handleBatchDownload = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要下载的文件')
      return
    }

    // 计算文件总大小
    const selectedFiles = files.filter((f) => selectedRowKeys.includes(f.id))
    const totalSize = selectedFiles.reduce((sum, f) => sum + f.file_size, 0)
    const totalSizeGB = totalSize / (1024 * 1024 * 1024)

    // 如果文件很大，显示警告
    if (totalSizeGB > 1 || selectedRowKeys.length > 100) {
      Modal.confirm({
        title: '批量下载警告',
        content: `您即将下载 ${selectedRowKeys.length} 个文件，总大小约为 ${totalSizeGB.toFixed(2)} GB。这可能需要一些时间，确定要继续吗？`,
        okText: '继续下载',
        cancelText: '取消',
        onOk: async () => {
          await performBatchDownload()
        },
      })
    } else {
      await performBatchDownload()
    }
  }, [selectedRowKeys, files])

  const performBatchDownload = useCallback(async () => {
    setBatchDownloading(true)
    try {
      videoFileApi.batchDownloadFiles(selectedRowKeys as number[])
      setSelectedRowKeys([])
    } catch (error) {
      message.error(error instanceof Error ? error.message : '批量下载失败')
    } finally {
      setBatchDownloading(false)
    }
  }, [selectedRowKeys])

  // 打开批量转录对话框
  const handleBatchTranscribeClick = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要转录的文件')
      return
    }

    // 检查是否都是视频文件
    const selectedFiles = files.filter((f) => selectedRowKeys.includes(f.id))
    const nonVideoFiles = selectedFiles.filter((f) => f.format !== 'mp4' && f.format !== 'mkv')

    if (nonVideoFiles.length > 0) {
      message.warning(`只能转录视频文件，已忽略 ${nonVideoFiles.length} 个非视频文件`)
    }

    setBatchSamplingRate(0.5) // 重置为默认值
    setBatchTranscribeMode('local') // 重置为默认模式
    setBatchTranscribeModalOpen(true)
  }, [selectedRowKeys, files])

  // 确认批量转录
  const confirmBatchTranscription = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      return
    }

    setBatchTranscribing(true)
    try {
      const request: BatchTranscriptionRequest = {
        video_file_ids: selectedRowKeys as number[],
        sampling_rate: batchTranscribeMode === 'local' ? batchSamplingRate : undefined,
        mode: batchTranscribeMode,
      }

      const response = await submitBatchTranscription(request)

      if (response.data) {
        const { submitted_count, failed_count, errors } = response.data

        if (failed_count > 0) {
          const errorMsg =
            errors.length > 0
              ? `。错误详情：${errors.slice(0, 3).join('; ')}${errors.length > 3 ? '...' : ''}`
              : ''
          message.warning(
            `转录任务创建完成：成功 ${submitted_count} 个，失败 ${failed_count} 个${errorMsg}`
          )
        } else {
          message.success(`成功创建 ${submitted_count} 个转录任务`)
        }

        setBatchTranscribeModalOpen(false)
        setSelectedRowKeys([])

        // 刷新活跃转录任务列表
        loadActiveTranscriptions()
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '批量转录失败')
    } finally {
      setBatchTranscribing(false)
    }
  }, [selectedRowKeys, batchSamplingRate, batchTranscribeMode, loadActiveTranscriptions])

  // 查看详情
  const viewDetail = useCallback((file: VideoFile) => {
    setViewingFile(file)
    setDetailVisible(true)
  }, [])

  // 重命名文件
  const handleRename = useCallback((file: VideoFile) => {
    setRenamingFile(file)
    // Find the last dot and split from there
    const lastDotIndex = file.file_name.lastIndexOf('.')
    const nameWithoutExt =
      lastDotIndex > 0 ? file.file_name.substring(0, lastDotIndex) : file.file_name
    setNewFileName(nameWithoutExt)
    setRenameModalVisible(true)
  }, [])

  const confirmRename = useCallback(async () => {
    if (!renamingFile || !newFileName.trim()) {
      message.warning('请输入文件名')
      return
    }

    // Check if name hasn't changed
    const lastDotIndex = renamingFile.file_name.lastIndexOf('.')
    const nameWithoutExt =
      lastDotIndex > 0 ? renamingFile.file_name.substring(0, lastDotIndex) : renamingFile.file_name
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
          message.success(
            `扫描完成！发现 ${scanned} 个文件，新增 ${created} 条记录，跳过 ${skipped} 个`
          )
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

  // 查看转录进度：从 activeTranscriptions 取任务信息并打开进度弹窗
  const handleViewTranscriptionProgress = useCallback(
    (record: VideoFile) => {
      const taskInfo = activeTranscriptions.get(record.id)
      if (!taskInfo) {
        message.error('转录任务信息未找到')
        return
      }
      setTranscriptionVideoFile(record)
      setCloudTranscriptionMode(taskInfo.mode as TranscriptionMode)
      setSelectedSamplingRate(taskInfo.samplingRate)
      setTranscriptionModalOpen(true)
    },
    [activeTranscriptions]
  )

  const handlePreviewPpt = useCallback(
    (record: VideoFile) => navigate(`/results/${record.id}`),
    [navigate]
  )
  const handleSplit = useCallback((record: VideoFile) => navigate(`/split/${record.id}`), [navigate])

  // 渲染操作按钮：memo 化的 FileRowActions，handler 全部以稳定 useCallback 下传
  const renderActions = useCallback(
    (record: VideoFile) => (
      <FileRowActions
        record={record}
        isActiveTranscription={activeTranscriptions.has(record.id)}
        onDownload={handleDownload}
        onViewTranscriptionProgress={handleViewTranscriptionProgress}
        onTranscribe={handleTranscribeClick}
        onCloudTranscribe={handleCloudTranscribe}
        onPreviewPpt={handlePreviewPpt}
        onSplit={handleSplit}
        onRename={handleRename}
        onDelete={handleDelete}
        onViewDetail={viewDetail}
      />
    ),
    [
      activeTranscriptions,
      handleDownload,
      handleViewTranscriptionProgress,
      handleTranscribeClick,
      handleCloudTranscribe,
      handlePreviewPpt,
      handleSplit,
      handleRename,
      handleDelete,
      viewDetail,
    ]
  )

  // 表格列定义 - 用 useMemo 固化引用（renderActions 为 useCallback，renderStatus 已上提为模块级）
  const columns = useMemo<ColumnsType<VideoFile>>(() => [
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
          upload: { label: '上传', color: 'purple' },
          snapshot: { label: '快照', color: 'green' },
          split: { label: '分割', color: 'orange' },
        }
        const config = SOURCE_CONFIG[sourceType] || SOURCE_CONFIG.recording
        return (
          <Space orientation="vertical" size={2}>
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
  ], [renderActions, navigate])

  const isReady = viewingFile?.status === 'ready'

  // D-05.1 / D-05.2 — 空态与错误态分支
  const isFiltered = Boolean(params.keyword || params.status)
  const showErrorState = !loading && Boolean(loadError)
  const showEmptyState = !loading && !loadError && files.length === 0

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
      <Button
        type="primary"
        icon={<ReloadOutlined />}
        onClick={() => {
          loadFiles()
          loadStats()
        }}
      >
        重试
      </Button>
    </Empty>
  )

  // D-05.1 — 空态：一句话文案 + 一个主操作
  const emptyState = isFiltered ? (
    <Empty
      style={{ padding: '48px 24px' }}
      image={<EmptyFiles style={{ width: 180, height: 126, color: designTokens.colors.muted }} />}
      styles={{ image: { height: 126 } }}
      description={
        <div>
          <div style={{ color: designTokens.colors.text.primary }}>没有匹配的文件</div>
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
      image={<EmptyFiles style={{ width: 180, height: 126, color: designTokens.colors.muted }} />}
      styles={{ image: { height: 126 } }}
      description={
        <div>
          <div style={{ color: designTokens.colors.text.primary }}>还没有文件</div>
          <div style={{ color: designTokens.colors.muted, fontSize: 13, marginTop: 4 }}>
            上传视频，或扫描录制目录导入
          </div>
        </div>
      }
    >
      <Button type="primary" icon={<UploadOutlined />} onClick={() => setUploadModalVisible(true)}>
        上传视频
      </Button>
    </Empty>
  )

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
                <Statistic title="总大小" value={stats.total_size_gb.toFixed(2)} suffix="GB" />
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
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Select
            placeholder="筛选状态"
            allowClear
            style={{ width: 120 }}
            value={params.status}
            onChange={handleStatusFilter}
            options={STATUS_OPTIONS}
          />
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              loadFiles()
              loadStats()
            }}
          >
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
                  disabled={batchDeleting || batchDownloading || batchTranscribing}
                >
                  批量删除 ({selectedRowKeys.length})
                </Button>
              </Popconfirm>
            </PermissionGuard>
          )}
          {selectedRowKeys.length > 0 && (
            <Button
              type="primary"
              icon={<DownloadOutlined />}
              loading={batchDownloading}
              disabled={batchDeleting || batchDownloading || batchTranscribing}
              onClick={handleBatchDownload}
            >
              批量下载 ({selectedRowKeys.length})
            </Button>
          )}
          {selectedRowKeys.length > 0 && (
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              loading={batchTranscribing}
              disabled={batchDeleting || batchDownloading || batchTranscribing}
              onClick={handleBatchTranscribeClick}
            >
              批量转录 ({selectedRowKeys.length})
            </Button>
          )}
          <PermissionGuard permission={PERMISSIONS.FILE_SCAN}>
            <Button type="primary" icon={<ScanOutlined />} onClick={handleScan} loading={scanning}>
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

      {showErrorState ? (
        errorState
      ) : showEmptyState ? (
        emptyState
      ) : (
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
      )}

      {/* 文件详情对话框 */}
      <Modal
        title={
          <Space>
            <FileOutlined />
            文件详情 - {viewingFile?.file_name}
          </Space>
        }
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
                <Col span={8}>
                  <strong>文件ID:</strong>
                </Col>
                <Col span={16}>{viewingFile.id}</Col>

                <Col span={8}>
                  <strong>文件名:</strong>
                </Col>
                <Col span={16}>{viewingFile.file_name}</Col>

                <Col span={8}>
                  <strong>文件路径:</strong>
                </Col>
                <Col span={16} style={{ wordBreak: 'break-all' }}>
                  {viewingFile.file_path}
                </Col>

                <Col span={8}>
                  <strong>格式:</strong>
                </Col>
                <Col span={16}>{viewingFile.format}</Col>

                <Col span={8}>
                  <strong>分辨率:</strong>
                </Col>
                <Col span={16}>{viewingFile.resolution}</Col>

                <Col span={8}>
                  <strong>码率:</strong>
                </Col>
                <Col span={16}>{viewingFile.bitrate} kbps</Col>

                <Col span={8}>
                  <strong>编码:</strong>
                </Col>
                <Col span={16}>{viewingFile.codec}</Col>

                <Col span={8}>
                  <strong>状态:</strong>
                </Col>
                <Col span={16}>{renderStatus(viewingFile.status)}</Col>

                <Col span={8}>
                  <strong>创建时间:</strong>
                </Col>
                <Col span={16}>{dayjs(viewingFile.created_at).format('YYYY-MM-DD HH:mm:ss')}</Col>
              </Row>
            </Card>

            {viewingFile.task && (
              <Card size="small" title="关联任务" style={{ marginTop: 16 }}>
                <Row gutter={[16, 8]}>
                  <Col span={8}>
                    <strong>任务ID:</strong>
                  </Col>
                  <Col span={16}>{viewingFile.task.id}</Col>

                  <Col span={8}>
                    <strong>任务名称:</strong>
                  </Col>
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
            {samplingRateOptions.map((opt) => (
              <Radio
                key={opt.value}
                value={opt.value}
                style={{ display: 'block', marginBottom: 8 }}
              >
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
      <Suspense fallback={<Spin />}>
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
        okButtonProps={{
          disabled:
            !newFileName.trim() ||
            newFileName.trim() === renamingFile?.file_name.replace(/\.[^/.]+$/, ''),
        }}
      >
        <Space orientation="vertical" style={{ width: '100%' }}>
          <Input
            value={newFileName}
            onChange={(e) => setNewFileName(e.target.value)}
            placeholder="新文件名（不含扩展名）"
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

      {/* 批量转录配置对话框 */}
      <Modal
        title="批量转录配置"
        open={batchTranscribeModalOpen}
        onOk={confirmBatchTranscription}
        onCancel={() => setBatchTranscribeModalOpen(false)}
        okText="提交转录任务"
        cancelText="取消"
        confirmLoading={batchTranscribing}
        width={600}
      >
        <div style={{ marginBottom: 16 }}>
          <span>已选择 {selectedRowKeys.length} 个文件进行批量转录</span>
        </div>

        <div style={{ marginBottom: 16 }}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>转录模式</div>
          <Radio.Group
            value={batchTranscribeMode}
            onChange={(e) => setBatchTranscribeMode(e.target.value)}
            disabled={batchTranscribing}
          >
            <Radio value="local">本地转录（快速，免费）</Radio>
            <Radio value="cloud">云端转录（阿里通义听悟，更准确）</Radio>
          </Radio.Group>
        </div>

        {batchTranscribeMode === 'local' && (
          <div style={{ marginBottom: 16 }}>
            <div style={{ marginBottom: 8, fontWeight: 500 }}>采样率</div>
            <div style={{ color: '#888', fontSize: 12, marginBottom: 8 }}>
              采样率决定每秒提取的帧数，数值越小精度越高但文件越大
            </div>
            <Radio.Group
              value={batchSamplingRate}
              onChange={(e) => setBatchSamplingRate(e.target.value)}
              disabled={batchTranscribing}
            >
              {samplingRateOptions.map((opt) => (
                <Radio
                  key={opt.value}
                  value={opt.value}
                  style={{ display: 'block', marginBottom: 8 }}
                >
                  {opt.label} ({opt.description})
                </Radio>
              ))}
            </Radio.Group>
          </div>
        )}

        {batchTranscribeMode === 'cloud' && (
          <div style={{ marginBottom: 16 }}>
            <Alert
              title="云端转录使用阿里通义听悟服务，支持更准确的语音识别和PPT提取，但需要消耗API配额"
              type="info"
              showIcon
            />
          </div>
        )}
      </Modal>
    </div>
  )
}
