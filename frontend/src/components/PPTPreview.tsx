// PPT 预览组件 - 主视图 + 侧边栏缩略图
// 支持键盘导航、全屏模式、单页下载和复制

import { Button, Image, InputNumber, Spin, message } from 'antd'
import {
  DownloadOutlined,
  CopyOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
} from '@ant-design/icons'
import type { SlideImage, SelectedSlide } from '../types/ppt'
import { useState, useEffect, useCallback } from 'react'
import SlideThumbnail from './SlideThumbnail'

interface PPTPreviewProps {
  slides: SlideImage[]
  currentSlide: number
  onSlideChange: (index: number) => void
  isMergeMode: boolean
  selectedSlides: SelectedSlide[]
  onToggleSelect: (slide: SlideImage, index: number) => void
  isLoading: boolean
  currentPptId?: number
}

export default function PPTPreview({
  slides,
  currentSlide,
  onSlideChange,
  isMergeMode,
  selectedSlides,
  onToggleSelect,
  isLoading,
  currentPptId = 0,
}: PPTPreviewProps) {
  const [isFullscreen, setIsFullscreen] = useState(false)

  // 键盘导航
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (isFullscreen && e.key === 'Escape') {
        setIsFullscreen(false)
        return
      }

      // 合并模式下禁用键盘导航
      if (isMergeMode) return

      if (e.key === 'ArrowLeft' && currentSlide > 0) {
        onSlideChange(currentSlide - 1)
      } else if (e.key === 'ArrowRight' && currentSlide < slides.length - 1) {
        onSlideChange(currentSlide + 1)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [currentSlide, slides.length, isMergeMode, isFullscreen, onSlideChange])

  // 下载当前幻灯片
  const handleDownloadCurrentSlide = useCallback(() => {
    if (!slides[currentSlide]) return

    const link = document.createElement('a')
    link.href = slides[currentSlide].fullsize_url
    link.download = `slide_${currentSlide + 1}.jpg`
    link.target = '_blank'
    link.click()
  }, [currentSlide, slides])

  // 复制当前幻灯片到剪贴板
  const handleCopyCurrentSlide = useCallback(async () => {
    if (!slides[currentSlide]) return

    try {
      const response = await fetch(slides[currentSlide].fullsize_url)
      const blob = await response.blob()

      await navigator.clipboard.write([
        new ClipboardItem({
          [blob.type]: blob,
        }),
      ])

      message.success('已复制到剪贴板')
    } catch (error) {
      message.error('复制失败，请重试')
    }
  }, [currentSlide, slides])

  return (
    <div style={{ display: 'flex', height: '100%', minHeight: 400 }}>
      {/* 侧边栏缩略图 */}
      <div
        style={{
          width: 200,
          overflowY: 'auto',
          borderRight: '1px solid #f0f0f0',
          padding: 8,
          background: '#fafafa',
          display: isFullscreen ? 'none' : 'block',
        }}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {slides.map((slide, idx) => (
            <SlideThumbnail
              key={slide.slide_number}
              slide={slide}
              slideNumber={slide.slide_number}
              totalSlides={slides.length}
              isCurrent={idx === currentSlide}
              isSelected={
                isMergeMode &&
                selectedSlides.some((s) => s.id === `${currentPptId}_${slide.slide_number}`)
              }
              isSelectable={isMergeMode}
              onClick={() =>
                isMergeMode ? onToggleSelect(slide, idx) : onSlideChange(idx)
              }
            />
          ))}
        </div>
      </div>

      {/* 主视图 */}
      <div
        style={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          padding: 24,
          background: '#ffffff',
          position: 'relative',
        }}
      >
        {/* 退出全屏按钮（仅全屏模式显示） */}
        {isFullscreen && (
          <Button
            icon={<FullscreenExitOutlined />}
            onClick={() => setIsFullscreen(false)}
            style={{
              position: 'absolute',
              top: 16,
              right: 16,
              zIndex: 10,
            }}
            aria-label="退出全屏"
          >
            退出全屏
          </Button>
        )}

        {/* 幻灯片图片容器 */}
        <div
          style={{
            flex: 1,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
          }}
        >
          {isLoading ? (
            <Spin tip="正在生成预览..." />
          ) : slides[currentSlide] ? (
            <Image
              src={slides[currentSlide].fullsize_url}
              alt={`幻灯片 ${currentSlide + 1}`}
              preview={false}
              style={{
                maxWidth: '100%',
                maxHeight: '100%',
                objectFit: 'contain',
              }}
            />
          ) : (
            <div style={{ color: '#8c8c8c' }}>暂无幻灯片</div>
          )}
        </div>

        {/* 页码指示器（非合并模式、非全屏模式显示） */}
        {!isMergeMode && !isFullscreen && slides.length > 0 && (
          <div
            style={{
              borderTop: '1px solid #f0f0f0',
              padding: '16px 24px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              gap: 12,
            }}
          >
            <span>第</span>
            <InputNumber
              min={1}
              max={slides.length}
              value={currentSlide + 1}
              onChange={(v) => v && onSlideChange(v - 1)}
              style={{ width: 80 }}
              aria-label={`跳转到页码，共${slides.length}页`}
            />
            <span>/{slides.length} 页</span>
            <Button
              icon={<DownloadOutlined />}
              aria-label="下载此页幻灯片"
              onClick={handleDownloadCurrentSlide}
            >
              下载此页
            </Button>
            <Button
              icon={<CopyOutlined />}
              aria-label="复制幻灯片到剪贴板"
              onClick={handleCopyCurrentSlide}
            >
              复制图片
            </Button>
            <Button
              icon={<FullscreenOutlined />}
              aria-label="全屏演示"
              onClick={() => setIsFullscreen(true)}
            >
              全屏演示
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
