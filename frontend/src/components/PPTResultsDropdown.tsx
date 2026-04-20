import { Dropdown, Button, Tag, Space } from 'antd'
import { CheckCircleOutlined, DownOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { PPTResult } from '../types/ppt'
import type { MenuProps } from 'antd'

interface PPTResultsDropdownProps {
  ppts: PPTResult[]
  currentPptId: number
  onPptChange: (pptId: number) => void
}

export function PPTResultsDropdown({
  ppts,
  currentPptId,
  onPptChange,
}: PPTResultsDropdownProps) {
  const currentPpt = ppts.find((p) => p.id === currentPptId)

  const menuItems: MenuProps['items'] = ppts.map((ppt) => ({
    key: ppt.id,
    label: (
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', minWidth: 200 }}>
        <Space>
          <span>{ppt.file_name}</span>
          {ppt.id === currentPptId && (
            <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 14 }} />
          )}
        </Space>
        <Tag color={ppt.source_type === 'merge' ? 'blue' : 'green'} style={{ margin: 0 }}>
          {ppt.page_count} 页
        </Tag>
      </div>
    ),
    onClick: () => onPptChange(ppt.id),
  }))

  return (
    <Dropdown menu={{ items: menuItems }} trigger={['click']} placement="bottomRight">
      <Button icon={<DownOutlined />} size="small">
        {currentPpt?.file_name || '选择转录结果'}
      </Button>
    </Dropdown>
  )
}
