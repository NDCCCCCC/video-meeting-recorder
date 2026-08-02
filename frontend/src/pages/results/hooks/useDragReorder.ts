// 幻灯片拖拽排序 hook
// 封装 results 页"拖拽排序"模式的本地状态 + 4 个 DnD handler。
// 排序成功后调用传入的 loadSlides 刷新。

import { useState, useCallback } from 'react'
import { message } from 'antd'
import { reorderSlides } from '../../../api/ppt'
import type { SlideImage } from '../../../types/ppt'

interface UseDragReorderArgs {
  slides: SlideImage[]
  currentPptId: number
  loadSlides: (pptId: number) => Promise<unknown>
}

export function useDragReorder({ slides, currentPptId, loadSlides }: UseDragReorderArgs) {
  const [isDragMode, setIsDragMode] = useState(false)
  const [draggedSlide, setDraggedSlide] = useState<number | null>(null)

  const handleDragStart = useCallback((slideNumber: number) => {
    setDraggedSlide(slideNumber)
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault() // 允许放置
  }, [])

  const handleDrop = useCallback(
    async (e: React.DragEvent, targetSlideNumber: number) => {
      e.preventDefault()
      if (draggedSlide === null || draggedSlide === targetSlideNumber) {
        setDraggedSlide(null)
        return
      }

      // 创建新的幻灯片顺序
      const newSlideOrder = slides.map((s) => s.slide_number)
      const draggedIndex = newSlideOrder.indexOf(draggedSlide)
      const targetIndex = newSlideOrder.indexOf(targetSlideNumber)

      // 移除被拖拽的幻灯片
      newSlideOrder.splice(draggedIndex, 1)
      // 在目标位置插入
      newSlideOrder.splice(targetIndex, 0, draggedSlide)

      try {
        const response = await reorderSlides(currentPptId, newSlideOrder)
        if (response.data?.success) {
          message.success('幻灯片顺序已更新')
          // 重新加载幻灯片
          await loadSlides(currentPptId)
        } else {
          message.error('更新幻灯片顺序失败')
        }
      } catch (error) {
        message.error('更新幻灯片顺序失败: ' + (error as Error).message)
      } finally {
        setDraggedSlide(null)
      }
    },
    [draggedSlide, slides, currentPptId, loadSlides]
  )

  const handleDragEnd = useCallback(() => {
    setDraggedSlide(null)
  }, [])

  return {
    isDragMode,
    setIsDragMode,
    draggedSlide,
    handleDragStart,
    handleDragOver,
    handleDrop,
    handleDragEnd,
  }
}
