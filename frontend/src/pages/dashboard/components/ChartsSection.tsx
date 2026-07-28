// Dashboard charts section component

import { Card, Row, Col, Skeleton } from 'antd'
import { Column, Pie } from '@ant-design/charts'

interface ChartsSectionProps {
  taskStatusData: Array<{ status: string; count: number }>
  fileTypeData: Array<{ type: string; count: number }>
  loading?: boolean
}

export function ChartsSection({ taskStatusData, fileTypeData, loading }: ChartsSectionProps) {
  // Column chart config (D-07.2: 任务状态 distribution from real stats)
  const columnConfig = {
    data: taskStatusData,
    xField: 'status' as const,
    yField: 'count' as const,
    label: { position: 'top' as const },
    meta: {
      status: { alias: '状态' },
      count: { alias: '数量' },
    },
  }

  // Pie chart config (D-07.2: 文件类型 distribution from real stats)
  const pieConfig = {
    data: fileTypeData,
    angleField: 'count' as const,
    colorField: 'type' as const,
    radius: 0.8,
    innerRadius: 0.6,
    label: {
      type: 'inner' as const,
      offset: '-50%',
      content: '{value}',
      style: { fontSize: 14, textAlign: 'center' as const },
    },
    statistic: {
      title: { offsetY: -8, content: '文件总数' },
      content: { offsetY: 4, style: { fontSize: '24px' } },
    },
  }

  if (loading) {
    return (
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="任务状态">
            <Skeleton.Image active style={{ width: '100%', height: 300 }} />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="文件类型">
            <Skeleton.Image active style={{ width: '100%', height: 300 }} />
          </Card>
        </Col>
      </Row>
    )
  }

  return (
    <Row gutter={[16, 16]}>
      <Col xs={24} lg={12}>
        <Card title="任务状态">
          <Column {...columnConfig} height={300} />
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <Card title="文件类型">
          <Pie {...pieConfig} height={300} />
        </Card>
      </Col>
    </Row>
  )
}
