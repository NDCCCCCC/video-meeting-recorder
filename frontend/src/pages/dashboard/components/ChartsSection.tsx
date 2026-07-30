// Dashboard charts section component

import { Card, Row, Col, Skeleton, Empty } from 'antd'
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

  // Pie chart config (D-07.2 / D-09: 文件类型 distribution from real stats)
  //
  // 历史踩坑记录：
  //   1. v4 的 `label.type: 'inner'` 在 G2 v5 不再注册，触发
  //      `Unknown Component: shape.inner`。
  //   2. G2 v5 的 `text` annotation 在零数据 / 退化半径下，会产生
  //      `Invalid SVG Path definition: M0,0LNaN,NaN`。
  // 现在改为：
  //   - `label.position: 'inside'`（v5 标准命名）
  //   - 文件总数用 React 计算并放进 Card 标题区，不走 G2 annotation
  //   - `sum === 0` 时显示 <Empty/> 占位，避免让 G2 在退化几何上求解
  const fileTotal = fileTypeData.reduce((sum, datum) => sum + datum.count, 0)
  const hasFileData = fileTypeData.length > 0 && fileTotal > 0

  const pieConfig = hasFileData
    ? {
        data: fileTypeData,
        angleField: 'count' as const,
        colorField: 'type' as const,
        radius: 0.8,
        innerRadius: 0.6,
        label: {
          position: 'inside' as const,
          // G2 v5 invokes `formatter(text, datum, index, abstractData)`.
          // First arg is text resolved from `label.text` — for inside labels
          // no static text is configured, so it arrives as undefined. We
          // return the datum's `type` from the second positional arg.
          formatter: (_text: string | undefined, datum?: { type: string; count: number }) =>
            datum?.type ?? '',
          style: { fontSize: 14, textAlign: 'center' as const },
        },
      }
    : null

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
        <Card
          title="文件类型"
          // 把文件总数从 G2 annotation 搬到这里：v5 的 text annotation
          // 在零数据 / 退化几何下会产生 NaN path 报错，迁出渲染区即可。
          extra={
            hasFileData ? (
              <span style={{ color: 'rgba(0,0,0,0.65)' }}>
                文件总数 <strong>{fileTotal}</strong>
              </span>
            ) : null
          }
        >
          {hasFileData && pieConfig ? (
            <Pie {...pieConfig} height={300} />
          ) : (
            <div style={{ height: 300, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <Empty description="暂无文件类型数据" />
            </div>
          )}
        </Card>
      </Col>
    </Row>
  )
}
