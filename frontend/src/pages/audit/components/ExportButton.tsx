import { useState } from 'react'
import { Button, Dropdown, message } from 'antd'
import { DownloadOutlined } from '@ant-design/icons'
import type { MenuProps } from 'antd'
import type { AuditLogExportParams } from '../../../types/audit'
import * as auditApi from '../../../api/audit'

interface ExportButtonProps {
  params: AuditLogExportParams
}

export function ExportButton({ params }: ExportButtonProps) {
  const [loading, setLoading] = useState(false)

  const handleExport = async (format: 'csv' | 'json') => {
    setLoading(true)
    try {
      const blob = await auditApi.exportAuditLogs({ ...params, format })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `audit_logs_${new Date().getTime()}.${format}`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
      message.success(`导出${format.toUpperCase()}成功`)
    } catch (error) {
      message.error(error instanceof Error ? error.message : '导出失败')
    } finally {
      setLoading(false)
    }
  }

  const menuItems: MenuProps['items'] = [
    {
      key: 'csv',
      label: '导出为 CSV',
      onClick: () => handleExport('csv'),
    },
    {
      key: 'json',
      label: '导出为 JSON',
      onClick: () => handleExport('json'),
    },
  ]

  return (
    <Dropdown.Button
      menu={{ items: menuItems }}
      loading={loading}
      icon={<DownloadOutlined />}
    >
      导出日志
    </Dropdown.Button>
  )
}
