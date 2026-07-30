import { useState } from 'react'
import { PageHeader } from '@ant-design/pro-layout'
import { Space, Empty, Button } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { AuditTable } from './components/AuditTable'
import { FilterBar } from './components/FilterBar'
import { DiffModal } from './components/DiffModal'
import { ExportButton } from './components/ExportButton'
import { useAuditLogs } from '../../hooks/useAuditLogs'
import EmptyAudit from '@/assets/illustrations/EmptyAudit'
import ErrorNetwork from '@/assets/illustrations/ErrorNetwork'
import { designTokens } from '@/styles/theme'
import type { AuditLog, AuditLogListParams } from '../../types/audit'

export default function AuditLogsPage() {
  const [params, setParams] = useState<AuditLogListParams>({
    page: 1,
    page_size: 20,
  })
  const [selectedLog, setSelectedLog] = useState<AuditLog | null>(null)
  const [diffModalOpen, setDiffModalOpen] = useState(false)

  const { logs, total, loading, error, fetchLogs } = useAuditLogs(params)

  const handleFilter = (newParams: AuditLogListParams) => {
    setParams(newParams)
    fetchLogs(newParams)
  }

  const handleReset = () => {
    const resetParams = { page: 1, page_size: 20 }
    setParams(resetParams)
    fetchLogs(resetParams)
  }

  const handlePageChange = (page: number, pageSize: number) => {
    const newParams = { ...params, page, page_size: pageSize }
    setParams(newParams)
    fetchLogs(newParams)
  }

  const handleViewDetail = (log: AuditLog) => {
    setSelectedLog(log)
    setDiffModalOpen(true)
  }

  const handleDiffModalClose = () => {
    setDiffModalOpen(false)
    setSelectedLog(null)
  }

  // D-05.1 / D-05.2 — 空态与错误态分支
  const showErrorState = !loading && Boolean(error)
  const showEmptyState = !loading && !error && logs.length === 0

  return (
    <div style={{ padding: 24, background: '#f0f2f5', minHeight: '100vh' }}>
      <PageHeader title="审计日志" />
      <div style={{ marginBottom: 16 }}>
        <Space size="middle">
          <FilterBar onFilter={handleFilter} onReset={handleReset} loading={loading} />
          <ExportButton
            params={{
              format: 'csv',
              username: params.username,
              action: params.action,
              module: params.module,
              start_time: params.start_time,
              end_time: params.end_time,
            }}
          />
        </Space>
      </div>
      {showErrorState ? (
        <Empty
          style={{ padding: '48px 24px' }}
          image={
            <ErrorNetwork style={{ width: 180, height: 126, color: designTokens.colors.error }} />
          }
          styles={{ image: { height: 126 } }}
          description={
            <div>
              <div style={{ color: designTokens.colors.text.primary }}>
                加载失败：{error?.message}
              </div>
              <div style={{ color: designTokens.colors.muted, fontSize: 13, marginTop: 4 }}>
                检查网络或稍后再试
              </div>
            </div>
          }
        >
          <Button type="primary" icon={<ReloadOutlined />} onClick={() => fetchLogs(params)}>
            重试
          </Button>
        </Empty>
      ) : showEmptyState ? (
        <Empty
          style={{ padding: '48px 24px' }}
          image={
            <EmptyAudit style={{ width: 180, height: 126, color: designTokens.colors.muted }} />
          }
          styles={{ image: { height: 126 } }}
          description={
            <div>
              <div style={{ color: designTokens.colors.text.primary }}>还没有日志</div>
              <div style={{ color: designTokens.colors.muted, fontSize: 13, marginTop: 4 }}>
                操作发生后会显示在这里
              </div>
            </div>
          }
        />
      ) : (
        <AuditTable
          logs={logs}
          total={total}
          loading={loading}
          onPageChange={handlePageChange}
          onViewDetail={handleViewDetail}
        />
      )}
      <DiffModal log={selectedLog} open={diffModalOpen} onClose={handleDiffModalClose} />
    </div>
  )
}
