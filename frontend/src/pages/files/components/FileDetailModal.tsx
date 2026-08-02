// 文件详情对话框（纯展示）
// 从 pages/files/index.tsx 抽出。状态标签用 STATUS_CONFIG 内联渲染。

import { Modal, Button, Space, Card, Statistic, Row, Col, Tag } from 'antd'
import { FileOutlined, DownloadOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { STATUS_CONFIG } from '../constants'
import { formatFileSize, formatDuration } from '../utils'
import type { VideoFile } from '../../../types/video-file'

export interface FileDetailModalProps {
  open: boolean
  file: VideoFile | null
  onClose: () => void
  onDownload: (id: number, fileName: string) => void
}

export function FileDetailModal({ open, file, onClose, onDownload }: FileDetailModalProps) {
  const isReady = file?.status === 'ready'
  return (
    <Modal
      title={
        <Space>
          <FileOutlined />
          文件详情 - {file?.file_name}
        </Space>
      }
      open={open}
      onCancel={onClose}
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
        isReady && file ? (
          <Button
            key="download"
            type="primary"
            icon={<DownloadOutlined />}
            onClick={() => onDownload(file.id, file.file_name)}
          >
            下载
          </Button>
        ) : null,
      ]}
      width={700}
    >
      {file && (
        <>
          <Card size="small" style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Statistic title="文件大小" value={formatFileSize(file.file_size)} />
              </Col>
              <Col span={12}>
                <Statistic title="时长" value={formatDuration(file.duration)} />
              </Col>
            </Row>
          </Card>

          <Card size="small" title="基本信息">
            <Row gutter={[16, 8]}>
              <Col span={8}>
                <strong>文件ID:</strong>
              </Col>
              <Col span={16}>{file.id}</Col>

              <Col span={8}>
                <strong>文件名:</strong>
              </Col>
              <Col span={16}>{file.file_name}</Col>

              <Col span={8}>
                <strong>文件路径:</strong>
              </Col>
              <Col span={16} style={{ wordBreak: 'break-all' }}>
                {file.file_path}
              </Col>

              <Col span={8}>
                <strong>格式:</strong>
              </Col>
              <Col span={16}>{file.format}</Col>

              <Col span={8}>
                <strong>分辨率:</strong>
              </Col>
              <Col span={16}>{file.resolution}</Col>

              <Col span={8}>
                <strong>码率:</strong>
              </Col>
              <Col span={16}>{file.bitrate} kbps</Col>

              <Col span={8}>
                <strong>编码:</strong>
              </Col>
              <Col span={16}>{file.codec}</Col>

              <Col span={8}>
                <strong>状态:</strong>
              </Col>
              <Col span={16}>
                <Tag color={STATUS_CONFIG[file.status].color}>
                  {STATUS_CONFIG[file.status].label}
                </Tag>
              </Col>

              <Col span={8}>
                <strong>创建时间:</strong>
              </Col>
              <Col span={16}>{dayjs(file.created_at).format('YYYY-MM-DD HH:mm:ss')}</Col>
            </Row>
          </Card>

          {file.task && (
            <Card size="small" title="关联任务" style={{ marginTop: 16 }}>
              <Row gutter={[16, 8]}>
                <Col span={8}>
                  <strong>任务ID:</strong>
                </Col>
                <Col span={16}>{file.task.id}</Col>

                <Col span={8}>
                  <strong>任务名称:</strong>
                </Col>
                <Col span={16}>{file.task.name}</Col>
              </Row>
            </Card>
          )}
        </>
      )}
    </Modal>
  )
}
