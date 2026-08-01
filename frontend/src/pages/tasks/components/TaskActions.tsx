// 任务操作按钮组件（行内操作区）
// 从 pages/tasks/index.tsx 抽出，遵循项目 memo 子组件范式（参照 TaskActions 原实现）

import { memo, lazy } from 'react'
import { Space, Button, Tooltip, Popconfirm } from 'antd'
import {
  CameraOutlined,
  PlayCircleOutlined,
  StopOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
  EditOutlined,
  DeleteOutlined,
} from '@ant-design/icons'
import { PermissionGuard } from '../../../components/PermissionGuard'
import { PERMISSIONS } from '../../../utils/permissions'
import {
  canStartTask,
  canStopTask,
  canCancelTask,
  canRetryTask,
  canPreviewTask,
  canEditEndTime,
  DELETABLE_STATUSES,
} from '../constants'
import type { VideoRecordingTask } from '../../../types/task'

// 动态导入 HLSPreview 组件（命名导出需要包装为默认导出）
const HLSPreview = lazy(() =>
  import('../../../components/HLSPreview').then((module) => ({ default: module.HLSPreview }))
)

export interface TaskActionsProps {
  record: VideoRecordingTask
  onStart: (id: number) => void
  onStop: (id: number) => void
  onCancel: (id: number) => void
  onRetry: (id: number) => void
  onDelete: (id: number) => void
  onEdit: (task: VideoRecordingTask) => void
  onGenerateSnapshot: (id: number) => void
  snapshotGenerating: Set<number>
}

// Memo 化的任务操作按钮组件 (rerender-memo)
export const TaskActions = memo(function TaskActions({
  record,
  onStart,
  onStop,
  onCancel,
  onRetry,
  onDelete,
  onEdit,
  onGenerateSnapshot,
  snapshotGenerating,
}: TaskActionsProps) {
  return (
    <Space size="small">
      {canPreviewTask(record.status) && (
        <HLSPreview taskId={record.id} taskName={record.name} status={record.status} />
      )}
      {record.status === 'recording' && (
        <PermissionGuard permission={PERMISSIONS.RECORDING_SNAPSHOT}>
          <Button
            type="link"
            size="small"
            icon={<CameraOutlined />}
            loading={snapshotGenerating.has(record.id)}
            onClick={() => onGenerateSnapshot(record.id)}
          >
            {snapshotGenerating.has(record.id) ? '生成中...' : '生成MP4快照'}
          </Button>
        </PermissionGuard>
      )}
      <PermissionGuard permission={PERMISSIONS.TASK_START}>
        {canStartTask(record.status) && (
          <Tooltip title="启动任务">
            <Button
              type="link"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => onStart(record.id)}
            />
          </Tooltip>
        )}
      </PermissionGuard>
      <PermissionGuard permission={PERMISSIONS.TASK_STOP}>
        {canStopTask(record.status) && (
          <Tooltip title="停止任务">
            <Button
              type="link"
              size="small"
              danger
              icon={<StopOutlined />}
              onClick={() => onStop(record.id)}
            />
          </Tooltip>
        )}
      </PermissionGuard>
      <PermissionGuard permission={PERMISSIONS.TASK_STOP}>
        {canCancelTask(record.status) && (
          <Tooltip title="取消任务">
            <Button
              type="link"
              size="small"
              icon={<CloseCircleOutlined />}
              onClick={() => onCancel(record.id)}
            />
          </Tooltip>
        )}
      </PermissionGuard>
      <PermissionGuard permission={PERMISSIONS.TASK_START}>
        {canRetryTask(record.status) && (
          <Tooltip title="重试任务">
            <Button
              type="link"
              size="small"
              icon={<ReloadOutlined />}
              onClick={() => onRetry(record.id)}
            />
          </Tooltip>
        )}
      </PermissionGuard>
      <PermissionGuard permission={PERMISSIONS.TASK_EDIT}>
        {canEditEndTime(record.status) && (
          <Tooltip title={record.status === 'recording' ? '修改结束时间' : '编辑任务'}>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => onEdit(record)}
            />
          </Tooltip>
        )}
      </PermissionGuard>
      <PermissionGuard permission={PERMISSIONS.TASK_DELETE}>
        {DELETABLE_STATUSES.includes(record.status) && (
          <Popconfirm title="确定要删除这个任务吗？" onConfirm={() => onDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        )}
      </PermissionGuard>
    </Space>
  )
})
