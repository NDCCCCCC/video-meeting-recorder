// 文件行操作区（预览 + 详情 + "更多"下拉菜单）
// 从 pages/files/index.tsx 的 renderActions 抽出，套用 TaskActions 的 memo 子组件范式。
// 父级以稳定 useCallback 传入所有 handler，保证 memo 生效。

import { memo } from 'react'
import { Space, Button, Dropdown, Modal } from 'antd'
import type { MenuProps } from 'antd'
import {
  DownloadOutlined,
  CloudOutlined,
  LaptopOutlined,
  FilePptOutlined,
  ScissorOutlined,
  EditOutlined,
  DeleteOutlined,
  EyeOutlined,
} from '@ant-design/icons'
import { RenderVideoPreview } from '../../../components/VideoPlayerSimple'
import type { VideoFile } from '../../../types/video-file'

export interface FileRowActionsProps {
  record: VideoFile
  isActiveTranscription: boolean
  onDownload: (id: number, fileName: string) => void
  onViewTranscriptionProgress: (record: VideoFile) => void
  onTranscribe: (record: VideoFile) => void
  onCloudTranscribe: (record: VideoFile) => void
  onPreviewPpt: (record: VideoFile) => void
  onSplit: (record: VideoFile) => void
  onRename: (record: VideoFile) => void
  onDelete: (id: number) => void
  onViewDetail: (record: VideoFile) => void
}

function FileRowActions({
  record,
  isActiveTranscription,
  onDownload,
  onViewTranscriptionProgress,
  onTranscribe,
  onCloudTranscribe,
  onPreviewPpt,
  onSplit,
  onRename,
  onDelete,
  onViewDetail,
}: FileRowActionsProps) {
  // 构建"更多"操作菜单
  const moreMenuItems: MenuProps['items'] = []

  // 下载
  moreMenuItems.push({
    key: 'download',
    icon: <DownloadOutlined />,
    label: '下载文件',
    disabled: record.status !== 'ready',
    onClick: () => onDownload(record.id, record.file_name),
  })

  // 转录相关（仅 mp4 格式）
  if (record.format === 'mp4') {
    if (isActiveTranscription) {
      // 有活跃转录任务
      moreMenuItems.push({
        key: 'view-transcription',
        icon: <CloudOutlined />,
        label: '查看转录进度',
        onClick: () => onViewTranscriptionProgress(record),
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
            onClick: () => onTranscribe(record),
          },
          {
            key: 'cloud',
            icon: <CloudOutlined />,
            label: '云端转录（通义听悟）',
            onClick: () => onCloudTranscribe(record),
          },
        ],
      })
    }
  }

  // 预览PPT（使用后端返回的 has_ppt 字段）
  if (record.has_ppt) {
    moreMenuItems.push({
      key: 'preview-ppt',
      icon: <FilePptOutlined />,
      label: '预览PPT',
      onClick: () => onPreviewPpt(record),
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
      onClick: () => onSplit(record),
    })
  }

  // 重命名（非原始录制文件）
  if (record.source_type !== 'recording' || record.parent_id) {
    moreMenuItems.push({
      key: 'rename',
      icon: <EditOutlined />,
      label: '重命名',
      disabled: record.status !== 'ready',
      onClick: () => onRename(record),
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
        onOk: () => onDelete(record.id),
      })
    },
  })

  return (
    <Space size="small">
      <RenderVideoPreview {...record} />
      <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => onViewDetail(record)}>
        详情
      </Button>
      <Dropdown menu={{ items: moreMenuItems }} trigger={['click']}>
        <Button size="small">更多</Button>
      </Dropdown>
    </Space>
  )
}

export default memo(FileRowActions)
