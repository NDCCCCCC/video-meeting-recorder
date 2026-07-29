// PPT 结果切换条 - 水平画廊
// 用于在多个转录结果之间切换

import { Tooltip } from 'antd'
import type { PPTResult } from '../types/ppt'
import dayjs from 'dayjs'

interface PPTGalleryStripProps {
  ppts: PPTResult[]
  currentPptId: number
  onSelect: (pptId: number) => void
}

export default function PPTGalleryStrip({ ppts, currentPptId, onSelect }: PPTGalleryStripProps) {
  if (ppts.length === 0) {
    return <div style={{ textAlign: 'center', color: '#8c8c8c' }}>暂无 PPT 结果</div>
  }

  return (
    <div
      style={{
        display: 'flex',
        gap: 8,
        overflowX: 'auto',
        padding: '12px 0',
      }}
    >
      {ppts.map((ppt) => (
        <Tooltip
          key={ppt.id}
          title={`${dayjs(ppt.created_at).format('HH:mm')} 转录 · ${ppt.page_count} 页`}
        >
          <div
            onClick={() => onSelect(ppt.id)}
            role="button"
            aria-label={`切换到${dayjs(ppt.created_at).format('HH:mm')}的转录结果，${ppt.page_count}页`}
            tabIndex={0}
            style={{
              cursor: 'pointer',
              width: 100,
              height: 80,
              border: currentPptId === ppt.id ? '2px solid #1890ff' : '1px solid #f0f0f0',
              borderRadius: 4,
              padding: 8,
              background: currentPptId === ppt.id ? '#e6f7ff' : '#ffffff',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              transition: 'all 0.2s',
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                onSelect(ppt.id)
              }
            }}
          >
            <div
              style={{
                fontSize: 12,
                color: '#8c8c8c',
                marginBottom: 4,
              }}
            >
              {dayjs(ppt.created_at).format('HH:mm')}
            </div>
            <div
              style={{
                fontSize: 14,
                fontWeight: 500,
                color: '#262626',
              }}
            >
              {ppt.page_count}页
            </div>
            {ppt.source_type === 'merge' && (
              <div
                style={{
                  fontSize: 10,
                  color: '#1890ff',
                  marginTop: 2,
                  fontWeight: 500,
                }}
              >
                合并
              </div>
            )}
          </div>
        </Tooltip>
      ))}
    </div>
  )
}
