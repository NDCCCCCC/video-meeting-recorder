import { Table, Tag, Button, Space } from 'antd'
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table'
import dayjs from 'dayjs'
import type { AuditLog } from '../../../types/audit'

interface AuditTableProps {
  logs: AuditLog[]
  total: number
  loading?: boolean
  onPageChange: (page: number, pageSize: number) => void
  onViewDetail: (log: AuditLog) => void
}

export function AuditTable({ logs, total, loading, onPageChange, onViewDetail }: AuditTableProps) {
  const columns: ColumnsType<AuditLog> = [
    {
      title: '时间',
      dataIndex: 'created_at',
      width: 180,
      sorter: true,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: '用户',
      dataIndex: 'username',
      width: 120,
    },
    {
      title: '操作',
      dataIndex: 'action',
      width: 100,
      render: (action: string) => {
        const colorMap: Record<string, string> = {
          login: 'green',
          logout: 'default',
          create: 'blue',
          update: 'orange',
          delete: 'red',
          export: 'purple',
        }
        return <Tag color={colorMap[action] || 'default'}>{action}</Tag>
      },
    },
    {
      title: '模块',
      dataIndex: 'module',
      width: 100,
    },
    {
      title: '资源',
      dataIndex: 'resource',
      width: 200,
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'success' ? 'success' : 'error'}>{status}</Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      fixed: 'right' as const,
      render: (_, record) => (
        <Button type="link" size="small" onClick={() => onViewDetail(record)}>
          查看详情
        </Button>
      ),
    },
  ]

  const handleTableChange = (
    pagination: TablePaginationConfig,
    _filters: unknown,
    _sorter: unknown
  ) => {
    onPageChange(pagination.current || 1, pagination.pageSize || 20)
  }

  return (
    <Table
      columns={columns}
      dataSource={logs}
      rowKey="id"
      loading={loading}
      scroll={{ x: 1000 }}
      pagination={{
        current: 1,
        pageSize: 20,
        total,
        showSizeChanger: true,
        showTotal: (t) => `共 ${t} 条`,
      }}
      onChange={handleTableChange}
    />
  )
}
