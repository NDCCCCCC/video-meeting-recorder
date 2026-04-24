import { useState } from 'react'
import { Card, Input, AutoComplete, Checkbox, Button, Space, DatePicker } from 'antd'
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { AuditLogListParams } from '../../../types/audit'

interface FilterBarProps {
  onFilter: (params: AuditLogListParams) => void
  onReset: () => void
  loading?: boolean
}

export function FilterBar({ onFilter, onReset, loading }: FilterBarProps) {
  const [username, setUsername] = useState('')
  const [actions, setActions] = useState<string[]>([])
  const [modules, setModules] = useState<string[]>([])
  const [timeRange, setTimeRange] = useState<[dayjs.Dayjs, dayjs.Dayjs] | null>(null)

  const actionOptions = [
    { label: '登录', value: 'login' },
    { label: '创建', value: 'create' },
    { label: '更新', value: 'update' },
    { label: '删除', value: 'delete' },
    { label: '导出', value: 'export' },
  ]

  const moduleOptions = [
    { label: '用户', value: 'user' },
    { label: '角色', value: 'role' },
    { label: '任务', value: 'task' },
    { label: '文件', value: 'file' },
    { label: '系统', value: 'system' },
  ]

  const handleFilter = () => {
    const params: AuditLogListParams = {
      page: 1,
      page_size: 20,
    }

    if (username) params.username = username
    if (actions.length > 0) params.action = actions.join(',')
    if (modules.length > 0) params.module = modules.join(',')
    if (timeRange) {
      params.start_time = timeRange[0].format('YYYY-MM-DD HH:mm:ss')
      params.end_time = timeRange[1].format('YYYY-MM-DD HH:mm:ss')
    }

    onFilter(params)
  }

  const handleReset = () => {
    setUsername('')
    setActions([])
    setModules([])
    setTimeRange(null)
    onReset()
  }

  return (
    <Card style={{ marginBottom: 16 }}>
      <Space size="middle" wrap>
        <AutoComplete
          value={username}
          onChange={setUsername}
          placeholder="搜索用户名或ID"
          style={{ width: 200 }}
          options={[]} // Could be populated with API call
          filterOption={(inputValue, option) =>
            option?.value.toUpperCase().indexOf(inputValue.toUpperCase()) !== -1
          }
        />
        <Checkbox.Group
          options={actionOptions}
          value={actions}
          onChange={(values) => setActions(values as string[])}
        />
        <Checkbox.Group
          options={moduleOptions}
          value={modules}
          onChange={(values) => setModules(values as string[])}
        />
        <DatePicker.RangePicker
          showTime
          value={timeRange}
          onChange={(dates) => setTimeRange(dates as [dayjs.Dayjs, dayjs.Dayjs] | null)}
          format="YYYY-MM-DD HH:mm:ss"
        />
        <Button type="primary" icon={<SearchOutlined />} onClick={handleFilter} loading={loading}>
          应用过滤
        </Button>
        <Button icon={<ReloadOutlined />} onClick={handleReset}>
          重置
        </Button>
      </Space>
    </Card>
  )
}
