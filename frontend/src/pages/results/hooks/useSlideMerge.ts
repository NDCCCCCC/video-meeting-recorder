// 幻灯片合并 hook
// 封装 results 页"合并模式"的选中状态 + 切换/确认逻辑。
// 合并成功后刷新 PPT 列表并切换到新生成的 PPT。

import { useState, useCallback } from 'react'
import { message } from 'antd'
import { mergeSlides } from '../../../api/ppt'
import type { SlideImage, SelectedSlide, PPTResult, MergeSlideItem } from '../../../types/ppt'

interface UseSlideMergeArgs {
  currentPptId: number
  currentPpt?: PPTResult
  videoFileIdNum: number
  loadPpts: () => Promise<unknown>
  setCurrentPptId: (id: number) => void
}

export function useSlideMerge({
  currentPptId,
  currentPpt,
  videoFileIdNum,
  loadPpts,
  setCurrentPptId,
}: UseSlideMergeArgs) {
  const [isMergeMode, setIsMergeMode] = useState(false)
  const [selectedSlides, setSelectedSlides] = useState<SelectedSlide[]>([])
  const [isMerging, setIsMerging] = useState(false)

  const handleToggleSelect = useCallback(
    (slide: SlideImage, _index: number) => {
      const slideId = `${currentPptId}_${slide.slide_number}`
      const existing = selectedSlides.find((s) => s.id === slideId)

      if (existing) {
        // 取消选择
        setSelectedSlides((prev) => prev.filter((s) => s.id !== slideId))
      } else {
        // 检查 200 页限制
        if (selectedSlides.length >= 200) {
          message.warning('最多只能选择 200 页幻灯片进行合并')
          return
        }

        // 添加选择
        const pptName = currentPpt?.file_name || `PPT ${currentPptId}`
        setSelectedSlides((prev) => [
          ...prev,
          {
            id: slideId,
            ppt_file_id: currentPptId,
            slide_number: slide.slide_number,
            thumbnail_url: slide.thumbnail_url,
            source_name: pptName,
          },
        ])
      }
    },
    [currentPptId, selectedSlides, currentPpt]
  )

  const handleConfirmMerge = useCallback(async () => {
    if (selectedSlides.length === 0) {
      message.warning('请先选择要合并的幻灯片')
      return
    }

    setIsMerging(true)
    try {
      const mergeItems: MergeSlideItem[] = selectedSlides.map((s) => ({
        ppt_file_id: s.ppt_file_id,
        slide_number: s.slide_number,
      }))

      const response = await mergeSlides({
        slides: mergeItems,
        video_file_id: videoFileIdNum,
      })

      if (response.data) {
        message.success(`合并完成！已生成 ${response.data.page_count} 页 PPT。`)
        // 刷新 PPT 列表
        await loadPpts()
        // 退出合并模式
        setIsMergeMode(false)
        setSelectedSlides([])
        // 切换到新生成的 PPT
        setCurrentPptId(response.data.ppt_file_id)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '合并失败')
    } finally {
      setIsMerging(false)
    }
  }, [selectedSlides, videoFileIdNum, loadPpts, setCurrentPptId])

  return {
    isMergeMode,
    setIsMergeMode,
    selectedSlides,
    isMerging,
    handleToggleSelect,
    handleConfirmMerge,
  }
}
