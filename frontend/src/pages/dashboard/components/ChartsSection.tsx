// Dashboard charts section component

import { Card, Row, Col, Skeleton } from 'antd'
import { Line, Column, Pie } from '@ant-design/charts'
import type { StatisticalResult } from '@ant-design/charts'

interface ChartsSectionProps {
  taskTrendData: Array<{ date: string; count: number }>
  taskStatusData: Array<{ status: string; count: number }>
  fileTypeData: Array<{ type: string; count: number }>
  loading?: boolean
}

export function ChartsSection({ taskTrendData, taskStatusData, fileTypeData, loading }: ChartsSectionProps) {
  // Line chart config (D-07: trends)
  const lineConfig: StatisticalResult.Line = {
    data: taskTrendData,
    xField: 'date',
    yField: 'count',
    smooth: true,
    animation: true,
    point: { size: 3 },
    tooltip: {
      formatter: (datum) => ({ name: '任务数', value: datum.count }),
    },
  }

  // Column chart config (D-08: comparisons)
  const columnConfig: StatisticalResult.Column = {
    data: taskStatusData,
    xField: 'status',
    yField: 'count',
    label: { position: 'top' },
    meta: {
      status: { alias: '状态' },
      count: { alias: '数量' },
    },
  }

  // Pie chart config (D-09: distributions)
  const pieConfig: StatisticalResult.Pie = {
    data: fileTypeData,
    angleField: 'count',
    colorField: 'type',
    radius: 0.8,
    innerRadius: 0.6,
    label: {
      type: 'inner',
      offset: '-50%',
      content: '{value}',
      style: { fontSize: 14, textAlign: 'center' },
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
          <Card title="任务趋势">
            <Skeleton.Image active style={{ width: '100%', height: 300 }} />
          </Card>
        </Col>
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
        <Card title="任务趋势">
          <Line {...lineConfig} height={300} />
        </Card>
      </Col>
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
