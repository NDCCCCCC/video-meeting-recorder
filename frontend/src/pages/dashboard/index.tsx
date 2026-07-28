// Dashboard main page

import { PageHeader } from '@ant-design/pro-layout'
import { m } from 'framer-motion'
import { StatCards } from './components/StatCards'
import { ChartsSection } from './components/ChartsSection'
import { QuickActions } from './components/QuickActions'
import { useDashboardStats } from '../../hooks/useDashboardStats'
import { fadeIn } from '../../motion/motionConfig'
import { designTokens } from '../../styles/theme'

export default function DashboardPage() {
  const { stats, loading, error, refresh } = useDashboardStats()

  if (error) {
    return (
      <m.div variants={fadeIn} initial="hidden" animate="visible" style={{ padding: 24 }}>
        <PageHeader title="管理员仪表板" />
        <div>加载失败: {error.message}</div>
      </m.div>
    )
  }

  if (!stats) {
    return (
      <m.div variants={fadeIn} initial="hidden" animate="visible" style={{ padding: 24 }}>
        <PageHeader title="管理员仪表板" />
        <div>加载中...</div>
      </m.div>
    )
  }

  // Real stats from /api/v1/dashboard/stats (D-07.1, D-07.2 — no mock data, no trend chart)
  const taskStatusData = [
    { status: '成功', count: stats.task_stats.success },
    { status: '失败', count: stats.task_stats.fail },
    { status: '进行中', count: stats.task_stats.in_progress },
  ]

  const fileTypeData = [
    { type: '视频', count: stats.file_stats.total_videos },
    { type: '转录', count: stats.file_stats.transcripts },
    { type: 'PPT', count: stats.file_stats.ppts },
  ]

  return (
    <m.div
      variants={fadeIn}
      initial="hidden"
      animate="visible"
      style={{ padding: 24, background: designTokens.colors.surface, minHeight: '100vh' }}
    >
      <PageHeader title="管理员仪表板" />
      <QuickActions onRefresh={refresh} />
      <div style={{ marginBottom: 24 }}>
        <StatCards
          taskStats={stats.task_stats}
          fileStats={stats.file_stats}
          systemStats={stats.system_stats}
          loading={loading}
        />
      </div>
      <ChartsSection
        taskStatusData={taskStatusData}
        fileTypeData={fileTypeData}
        loading={loading}
      />
    </m.div>
  )
}
