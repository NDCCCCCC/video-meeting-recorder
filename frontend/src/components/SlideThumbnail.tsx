// 幻灯片缩略图组件 - 用于侧边栏和合并选择
// 支持导航模式和选择模式

import { Image } from 'antd'
import type { SlideImage } from '../types/ppt'

interface SlideThumbnailProps {
  slide: SlideImage
  slideNumber: number
  totalSlides: number
  isSelected: boolean
  isSelectable: boolean
  isCurrent: boolean
  onClick: () => void
}

export default function SlideThumbnail({
  slide,
  slideNumber,
  totalSlides,
  isSelected,
  isSelectable,
  isCurrent,
  onClick,
}: SlideThumbnailProps) {
  return (
    <div
      onClick={onClick}
      role="button"
      aria-label={`幻灯片${slideNumber}，共${totalSlides}页${isCurrent ? '，当前幻灯片' : ''}`}
      tabIndex={0}
      style={{
        position: 'relative',
        cursor: isSelectable ? 'pointer' : 'default',
        border: isCurrent || isSelected ? '2px solid #1890ff' : '2px solid transparent',
        borderRadius: 4,
        overflow: 'hidden',
        opacity: isCurrent ? 1 : 0.6,
        transition: 'opacity 0.2s',
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
    >
      <Image
        src={slide.thumbnail_url}
        alt={`幻灯片 ${slideNumber}`}
        width={160}
        height={88}
        preview={false}
        style={{
          objectFit: 'cover',
          display: 'block',
        }}
      />

      {/* 选中标记（合并模式） */}
      {isSelected && (
        <div
          style={{
            position: 'absolute',
            top: 4,
            right: 4,
            background: '#1890ff',
            color: 'white',
            borderRadius: '50%',
            width: 20,
            height: 20,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 12,
            fontWeight: 'bold',
          }}
        >
          ✓
        </div>
      )}

      {/* 幻灯片编号标签 */}
      <div
        style={{
          textAlign: 'center',
          fontSize: 12,
          marginTop: 4,
          color: isCurrent ? '#1890ff' : '#8c8c8c',
        }}
      >
        {slideNumber}
      </div>
    </div>
  )
}
