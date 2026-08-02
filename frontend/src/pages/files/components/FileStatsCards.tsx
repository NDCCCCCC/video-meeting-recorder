// 文件管理页统计卡片（纯展示）
// 从 pages/files/index.tsx 抽出。

import { Row, Col, Card, Statistic } from 'antd'
import { FileOutlined } from '@ant-design/icons'
import type { VideoFileStats } from '../../../types/video-file'

export interface FileStatsCardsProps {
  stats: VideoFileStats | null
  currentCount: number
}

export function FileStatsCards({ stats, currentCount }: FileStatsCardsProps) {
  if (!stats) return null
  return (
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
          <Statistic title="当前页" value={currentCount} suffix="/ 条" />
        </Card>
      </Col>
    </Row>
  )
}
