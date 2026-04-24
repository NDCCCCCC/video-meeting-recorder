import { useState } from 'react'
import { PageHeader } from '@ant-design/pro-layout'
import { Space } from 'antd'
import { AuditTable } from './components/AuditTable'
import { FilterBar } from './components/FilterBar'
import { DiffModal } from './components/DiffModal'
import { ExportButton } from './components/ExportButton'
import { useAuditLogs } from '../../hooks/useAuditLogs'
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

  return (
    <div style={{ padding: 24, background: '#f0f2f5', minHeight: '100vh' }}>
      <PageHeader title="审计日志" />
      <div style={{ marginBottom: 16 }}>
        <Space size="middle">
          <FilterBar
            onFilter={handleFilter}
            onReset={handleReset}
            loading={loading}
          />
          <ExportButton params={{
            format: 'csv',
            username: params.username,
            action: params.action,
            module: params.module,
            start_time: params.start_time,
            end_time: params.end_time,
          }} />
        </Space>
      </div>
      <AuditTable
        logs={logs}
        total={total}
        loading={loading}
        onPageChange={handlePageChange}
        onViewDetail={handleViewDetail}
      />
      <DiffModal
        log={selectedLog}
        open={diffModalOpen}
        onClose={handleDiffModalClose}
      />
    </div>
  )
}
