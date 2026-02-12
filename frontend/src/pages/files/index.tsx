// 文件管理页面

import { useState, useEffect } from 'react'
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
} from 'antd'
import {
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  DownloadOutlined,
  FileOutlined,
  FolderOpenOutlined,
  VideoCameraOutlined,
  CloudDownloadOutlined,
  EyeOutlined,
  ScanOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import * as videoFileApi from '../../api/video-file'
import { PermissionGuard } from '../../components/PermissionGuard'
import { PERMISSIONS } from '../../utils/permissions'
import { RenderVideoPreview } from '../../components/VideoPlayerSimple'
import type {
  VideoFile,
  VideoFileListParams,
  VideoFileStats,
  VideoFileStatus,
} from '../../types/video-file'

const statusOptions = [
  { label: '就绪', value: 'ready' },
  { label: '处理中', value: 'processing' },
  { label: '错误', value: 'error' },
  { label: '删除中', value: 'deleting' },
]

const statusColors: Record<VideoFileStatus, string> = {
  ready: 'success',
  processing: 'processing',
  error: 'error',
  deleting: 'default',
}

const statusLabels: Record<VideoFileStatus, string> = {
  ready: '就绪',
  processing: '处理中',
  error: '错误',
  deleting: '删除中',
}

export default function FileManagement() {
  const [files, setFiles] = useState<VideoFile[]>([])
  const [stats, setStats] = useState<VideoFileStats | null>(null)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [viewingFile, setViewingFile] = useState<VideoFile | null>(null)

  const [params, setParams] = useState<VideoFileListParams>({
    page: 1,
    page_size: 20,
    format: 'mp4', // 默认只显示 mp4 文件
  })

  const loadFiles = async () => {
    setLoading(true)
    try {
      const response = await videoFileApi.getVideoFileList(params)
      if (response.data) {
        setFiles(response.data.items)
        setTotal(response.data.total)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载文件列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadStats = async () => {
    try {
      const response = await videoFileApi.getVideoFileStats()
      if (response.data) {
        setStats(response.data)
      }
    } catch (error) {
      console.error('Failed to load file stats:', error)
    }
  }

  useEffect(() => {
    loadFiles()
  }, [params])

  useEffect(() => {
    loadStats()
  }, [])

  const handleSearch = (value: string) => {
    setParams({ ...params, keyword: value, page: 1 })
  }

  const handleStatusFilter = (status: VideoFileStatus | undefined) => {
    setParams({ ...params, status, page: 1 })
  }

  const handleTableChange = (pagination: any) => {
    setParams({
      ...params,
      page: pagination.current,
      page_size: pagination.pageSize,
    })
  }

  const handleDownload = async (id: number, fileName: string) => {
    try {
      await videoFileApi.downloadVideoFile(id)
      message.success(`下载 ${fileName} 成功`)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '下载失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await videoFileApi.deleteVideoFile(id)
      message.success('删除成功')
      loadFiles()
      loadStats()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }

  const viewDetail = (file: VideoFile) => {
    setViewingFile(file)
    setDetailVisible(true)
  }

  const handleScan = async () => {
    setScanning(true)
    try {
      const response = await videoFileApi.scanVideoFiles()
      if (response.data) {
        const result: videoFileApi.ScanResult = response.data
        const { scanned, created, skipped, errors } = result
        if (created > 0) {
          message.success(`扫描完成！发现 ${scanned} 个文件，新增 ${created} 条记录，跳过 ${skipped} 个已存在的文件`)
        } else {
          message.info(`扫描完成！发现 ${scanned} 个文件，但都是已存在的记录`)
        }
        if (errors && errors.length > 0) {
          console.warn('扫描过程中的错误:', errors)
        }
        loadFiles()
        loadStats()
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '扫描失败')
    } finally {
      setScanning(false)
    }
  }

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
      render: (name) => (
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
      render: (size) => `${(size / 1024 / 1024).toFixed(2)} MB`,
    },
    {
      title: '时长',
      dataIndex: 'duration',
      width: 100,
      render: (duration) => {
        const minutes = Math.floor(duration / 60)
        const seconds = duration % 60
        return `${minutes}:${seconds.toString().padStart(2, '0')}`
      },
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
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: VideoFileStatus) => (
        <Tag color={statusColors[status]}>{statusLabels[status]}</Tag>
      ),
    },
    {
      title: '录制时间',
      dataIndex: 'recorded_at',
      width: 160,
      render: (time) => (time ? dayjs(time).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 250,
      fixed: 'right' as const,
      render: (_, record) => (
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
          <Button
            type="link"
            size="small"
            icon={<DownloadOutlined />}
            onClick={() => handleDownload(record.id, record.file_name)}
            disabled={record.status !== 'ready'}
          >
            下载
          </Button>
          <PermissionGuard permission={PERMISSIONS.FILE_DELETE}>
            <Popconfirm
              title="确定要删除这个文件吗？"
              onConfirm={() => handleDelete(record.id)}
              disabled={record.status === 'processing'}
            >
              <Button
                type="link"
                size="small"
                danger
                icon={<DeleteOutlined />}
                disabled={record.status === 'processing'}
              >
                删除
              </Button>
            </Popconfirm>
          </PermissionGuard>
        </Space>
      ),
    },
  ]

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
                <Statistic
                  title="文件总数"
                  value={stats.total}
                  prefix={<FileOutlined />}
                />
              </Card>
            </Col>
            <Col span={6}>
              <Card>
                <Statistic
                  title="总大小"
                  value={stats.total_size_gb.toFixed(2)}
                  suffix="GB"
                  prefix={<CloudDownloadOutlined />}
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
                <Statistic
                  title="当前页"
                  value={files.length}
                  suffix="/ 条"
                />
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
            options={statusOptions}
          />
          <Button icon={<ReloadOutlined />} onClick={() => { loadFiles(); loadStats() }}>
            刷新
          </Button>
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
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={files}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1400 }}
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
          viewingFile && viewingFile.status === 'ready' && (
            <Button
              key="download"
              type="primary"
              icon={<DownloadOutlined />}
              onClick={() => {
                if (viewingFile) {
                  handleDownload(viewingFile.id, viewingFile.file_name)
                }
              }}
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
                  <Statistic title="文件大小" value={(viewingFile.file_size / 1024 / 1024).toFixed(2)} suffix="MB" />
                </Col>
                <Col span={12}>
                  <Statistic
                    title="时长"
                    value={`${Math.floor(viewingFile.duration / 60)}:${(viewingFile.duration % 60).toString().padStart(2, '0')}`}
                  />
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
                <Col span={16}>
                  <Tag color={statusColors[viewingFile.status]}>{statusLabels[viewingFile.status]}</Tag>
                </Col>

                <Col span={8}><strong>录制时间:</strong></Col>
                <Col span={16}>{viewingFile.recorded_at ? dayjs(viewingFile.recorded_at).format('YYYY-MM-DD HH:mm:ss') : '-'}</Col>

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
    </div>
  )
}
