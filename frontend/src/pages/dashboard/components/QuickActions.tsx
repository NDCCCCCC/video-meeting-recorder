// Dashboard quick actions component

import { Card, Button, Space } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  ClearOutlined,
  ReloadOutlined,
} from '@ant-design/icons'

interface QuickActionsProps {
  onRefresh: () => void
  onStartTask?: () => void
  onStopTask?: () => void
  onCleanup?: () => void
}

export function QuickActions({ onRefresh, onStartTask, onStopTask, onCleanup }: QuickActionsProps) {
  return (
    <Card title="快速操作" style={{ marginBottom: 16 }}>
      <Space size="middle">
        <Button
          type="primary"
          icon={<PlayCircleOutlined />}
          onClick={onStartTask}
        >
          启动录制任务
        </Button>
        <Button
          icon={<PauseCircleOutlined />}
          onClick={onStopTask}
        >
          停止任务
        </Button>
        <Button
          icon={<ClearOutlined />}
          onClick={onCleanup}
        >
          任务清理
        </Button>
        <Button
          icon={<ReloadOutlined />}
          onClick={onRefresh}
        >
          刷新数据
        </Button>
      </Space>
    </Card>
  )
}
