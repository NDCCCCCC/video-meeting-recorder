import { useCallback } from 'react'

// ==================== Types ====================

export interface UseVideoFrameNavigationResult {
  nextFrame: () => void
  prevFrame: () => void
  supportsFrameCallback: boolean
}

// ==================== Hook ====================

/**
 * Custom hook for frame-level video navigation
 *
 * Provides functions to navigate video by single frames.
 * Frame time is calculated as 1/30 second (standard 30fps video).
 *
 * @param videoRef - React ref to the HTMLVideoElement
 * @returns Object with navigation functions and browser support status
 */
export function useVideoFrameNavigation(videoRef: React.RefObject<HTMLVideoElement>) {
  // Standard frame time for 30fps video (1 frame = 1/30 second ≈ 0.033s)
  const FRAME_TIME = 1 / 30

  const nextFrame = useCallback(() => {
    const video = videoRef.current
    if (!video) return

    // Advance by one frame
    const newTime = Math.min(video.duration, video.currentTime + FRAME_TIME)
    video.currentTime = newTime
  }, [videoRef])

  const prevFrame = useCallback(() => {
    const video = videoRef.current
    if (!video) return

    // Go back by one frame
    const newTime = Math.max(0, video.currentTime - FRAME_TIME)
    video.currentTime = newTime
  }, [videoRef])

  // Detect if browser supports requestVideoFrameCallback API
  const supportsFrameCallback = useCallback(() => {
    const video = videoRef.current
    if (!video) return false

    // Check for requestVideoFrameCallback support (Chrome/Edge only)
    return typeof (video as any).requestVideoFrameCallback === 'function'
  }, [videoRef])

  return {
    nextFrame,
    prevFrame,
    supportsFrameCallback: supportsFrameCallback(),
  }
}
