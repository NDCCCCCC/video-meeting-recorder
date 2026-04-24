// Dashboard main page

import { PageHeader } from '@ant-design/pro-layout'
import { StatCards } from './components/StatCards'
import { ChartsSection } from './components/ChartsSection'
import { QuickActions } from './components/QuickActions'
import { useDashboardStats } from '../../hooks/useDashboardStats'

export default function DashboardPage() {
  const { stats, loading, error, refresh } = useDashboardStats()

  if (error) {
    return (
      <div style={{ padding: 24 }}>
        <PageHeader title="管理员仪表板" />
        <div>加载失败: {error.message}</div>
      </div>
    )
  }

  if (!stats) {
    return (
      <div style={{ padding: 24 }}>
        <PageHeader title="管理员仪表板" />
        <div>加载中...</div>
      </div>
    )
  }

  // Mock chart data (will be replaced with real API in future phase)
  const taskTrendData = [
    { date: '2024-04-20', count: 12 },
    { date: '2024-04-21', count: 18 },
    { date: '2024-04-22', count: 15 },
    { date: '2024-04-23', count: 22 },
    { date: '2024-04-24', count: 25 },
  ]

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
    <div style={{ padding: 24, background: '#f0f2f5', minHeight: '100vh' }}>
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
        taskTrendData={taskTrendData}
        taskStatusData={taskStatusData}
        fileTypeData={fileTypeData}
        loading={loading}
      />
    </div>
  )
}
