// 幻灯片轮询加载 hook（保守搬家自 results/index.tsx，逻辑未改动）
// 拥有 slides/isLoadingSlides 状态 + loadSlides（手动重载）+ currentPptId 变化时的自动轮询 effect。
// currentPptId/setCurrentSlide 使用广泛，留在父级作为入参。
// 注意：effect 内对 isLoadingSlides 的读取为既有 stale-closure 行为，搬家后原样保留
// （未做合并/去重，避免改动核心加载流程的回归风险）。

import { useState, useCallback, useEffect, useRef } from 'react'
import { message } from 'antd'
import { getSlides } from '../../../api/ppt'
import type { SlideImage } from '../../../types/ppt'

export function useSlidePolling(
  currentPptId: number,
  setCurrentSlide: (index: number) => void
) {
  const [slides, setSlides] = useState<SlideImage[]>([])
  const [isLoadingSlides, setIsLoadingSlides] = useState(false)
  const slidesPollCleanupRef = useRef<(() => void) | null>(null)

  // 加载幻灯片
  const loadSlides = useCallback(async (pptId: number) => {
    setIsLoadingSlides(true)
    try {
      const response = await getSlides(pptId)
      if (response.data) {
        if (response.data.status === 'extracting') {
          // 轮询等待提取完成 - 使用取消标志
          let cancelled = false
          let intervalId: NodeJS.Timeout | null = null

          const poll = async () => {
            if (cancelled) return

            try {
              const pollResponse = await getSlides(pptId)
              if (cancelled) return

              if (pollResponse.data && pollResponse.data.status === 'ready') {
                if (intervalId) clearInterval(intervalId)
                if (!cancelled) {
                  setSlides(pollResponse.data.slides)
                  setIsLoadingSlides(false)
                }
              }
            } catch (error) {
              if (!cancelled) {
                console.error('Polling error:', error)
              }
            }
          }

          intervalId = setInterval(poll, 2000)

          // 存储清理函数到 ref
          return () => {
            cancelled = true
            if (intervalId) clearInterval(intervalId)
          }
        } else {
          setSlides(response.data.slides)
          setIsLoadingSlides(false)
        }
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载幻灯片失败')
      setIsLoadingSlides(false)
    }
  }, [])

  // 当 currentPptId 变化时加载幻灯片
  useEffect(() => {
    // 清理之前的轮询
    if (slidesPollCleanupRef.current) {
      slidesPollCleanupRef.current()
      slidesPollCleanupRef.current = null
    }

    if (currentPptId <= 0) return

    setCurrentSlide(0)
    let cancelled = false
    let intervalId: NodeJS.Timeout | null = null

    const poll = async () => {
      if (cancelled) return
      try {
        const response = await getSlides(currentPptId)
        if (cancelled) return

        if (response.data?.status === 'ready') {
          if (intervalId) clearInterval(intervalId)
          if (!cancelled) {
            setSlides(response.data.slides)
            setIsLoadingSlides(false)
          }
        }
      } catch (error) {
        if (!cancelled) {
          console.error('Polling error:', error)
        }
      }
    }

    // WR-03, WR-04: Add proper error handling for initial poll and prevent race conditions
    poll()
      .catch((error) => {
        if (!cancelled) {
          console.error('Initial poll error:', error)
          message.error('加载幻灯片失败')
          setIsLoadingSlides(false)
        }
      })
      .then(() => {
        // If still extracting, start polling
        if (!cancelled && isLoadingSlides) {
          intervalId = setInterval(poll, 2000)
        }
      })

    return () => {
      cancelled = true
      if (intervalId) clearInterval(intervalId)
    }
  }, [currentPptId])

  return { slides, isLoadingSlides, loadSlides }
}
