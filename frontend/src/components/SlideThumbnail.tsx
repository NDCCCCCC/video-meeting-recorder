// 幻灯片缩略图组件 - 用于侧边栏和合并选择
// 支持导航模式和选择模式

import React, { memo } from 'react'
import { Image, Skeleton } from 'antd'
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

export default memo(function SlideThumbnail({
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
        transition: 'opacity 0.2s, transform 0.2s',
      }}
      onMouseEnter={(e) => {
        if (!isCurrent) {
          e.currentTarget.style.opacity = '0.8'
          e.currentTarget.style.transform = 'scale(1.02)'
        }
      }}
      onMouseLeave={(e) => {
        if (!isCurrent) {
          e.currentTarget.style.opacity = '0.6'
          e.currentTarget.style.transform = 'scale(1)'
        }
      }}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
    >
      {slide.thumbnail_url ? (
        <Image
          src={slide.thumbnail_url}
          alt={`幻灯片 ${slideNumber}`}
          width={160}
          height={90}
          preview={false}
          loading="lazy"
          onError={(e) => {
            // Fallback to placeholder on error
            const target = e.target as HTMLImageElement
            target.src = 'data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iMTYwIiBoZWlnaHQ9IjkwIiB4bWxucz0iaHR0cDovL3d3dy53My5vcmcvMjAwMC9zdmciPjxyZWN0IHdpZHRoPSIxNjAiIGhlaWdodD0iOTAiIGZpbGw9IiNmMGYwZjAiLz48dGV4dCB4PSI4MCIgeT0iNDUiIGZvbnQtc2l6ZT0iMTIiIGZpbGw9IiM4YzhjOGMiIHRleHQtYW5jaG9yPSJtaWRkbGUiPk5vIFByZXZpZXc8L3RleHQ+PC9zdmc+'
          }}
          style={{
            objectFit: 'cover',
            display: 'block',
          }}
        />
      ) : (
        <Skeleton.Image
          active
          style={{ width: 160, height: 90 }}
        />
      )}

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
