// Dashboard statistics cards component

import { Card, Row, Col, Statistic } from 'antd'
import {
  VideoCameraOutlined,
  HddOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons'
import { m } from 'framer-motion'
import type { TaskStats, FileStats, SystemStats } from '../../../types/dashboard'
import { designTokens } from '../../../styles/theme'
import { staggerContainer, staggerItem } from '../../../motion/motionConfig'

interface StatCardsProps {
  taskStats: TaskStats
  fileStats: FileStats
  systemStats: SystemStats
  loading?: boolean
}

export function StatCards({ taskStats, fileStats, systemStats, loading }: StatCardsProps) {
  // D-07.4: when all stat categories are zero, show honest empty-state copy
  // instead of 13 zero cards (which look like a failure, not "no data yet").
  const isAllZero =
    taskStats.total === 0 && fileStats.total_videos === 0 && systemStats.error_count === 0
  if (isAllZero && !loading) {
    return (
      <div
        style={{
          padding: '32px 24px',
          textAlign: 'center',
          color: designTokens.colors.muted,
          background: designTokens.colors.surface,
          borderRadius: designTokens.borderRadius.md,
          border: `1px solid ${designTokens.colors.border}`,
        }}
      >
        <div style={{ fontSize: 14, color: designTokens.colors.text.secondary }}>暂无数据</div>
        <div style={{ fontSize: 13, marginTop: 4 }}>开始录制或上传文件后，这里会显示实时统计</div>
      </div>
    )
  }

  return (
    <m.div
      variants={staggerContainer}
      initial="hidden"
      animate="visible"
      style={{ display: 'contents' }}
    >
      <Row gutter={[16, 16]}>
        {/* Task Stats (D-04) */}
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="录制任务总数"
                value={taskStats.total}
                prefix={<VideoCameraOutlined />}
                suffix="个"
                loading={loading}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="进行中任务"
                value={taskStats.in_progress}
                prefix={<LoadingOutlined />}
                suffix="个"
                loading={loading}
                valueStyle={{ color: designTokens.colors.primary }}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="成功任务"
                value={taskStats.success}
                prefix={<CheckCircleOutlined />}
                suffix="个"
                loading={loading}
                valueStyle={{ color: designTokens.colors.success }}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="失败任务"
                value={taskStats.fail}
                prefix={<CloseCircleOutlined />}
                suffix="个"
                loading={loading}
                valueStyle={{ color: designTokens.colors.error }}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="平均处理时间"
                value={taskStats.avg_time}
                prefix={<ClockCircleOutlined />}
                suffix="秒"
                loading={loading}
                precision={1}
              />
            </Card>
          </m.div>
        </Col>

        {/* File Stats (D-05) */}
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="视频文件总数"
                value={fileStats.total_videos}
                prefix={<VideoCameraOutlined />}
                suffix="个"
                loading={loading}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="总存储大小"
                value={fileStats.storage_mb}
                prefix={<HddOutlined />}
                suffix="MB"
                loading={loading}
                precision={1}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="转录文件数"
                value={fileStats.transcripts}
                suffix="个"
                loading={loading}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic title="PPT 文件数" value={fileStats.ppts} suffix="个" loading={loading} />
            </Card>
          </m.div>
        </Col>

        {/* System Stats (D-06) */}
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="磁盘使用率"
                value={systemStats.disk_usage_percent}
                suffix="%"
                loading={loading}
                precision={1}
                valueStyle={{
                  color:
                    systemStats.disk_usage_percent > 80
                      ? designTokens.colors.error
                      : systemStats.disk_usage_percent > 60
                        ? designTokens.colors.warning
                        : designTokens.colors.success,
                }}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="内存使用率"
                value={systemStats.memory_usage_percent}
                suffix="%"
                loading={loading}
                precision={1}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic
                title="最近错误数"
                value={systemStats.error_count}
                loading={loading}
                valueStyle={{
                  color: systemStats.error_count > 10 ? designTokens.colors.error : 'inherit',
                }}
              />
            </Card>
          </m.div>
        </Col>
        <Col xs={24} sm={12} md={8} lg={6}>
          <m.div variants={staggerItem} style={{ height: '100%' }}>
            <Card>
              <Statistic title="API 调用数" value={systemStats.api_calls} loading={loading} />
            </Card>
          </m.div>
        </Col>
      </Row>
    </m.div>
  )
}
