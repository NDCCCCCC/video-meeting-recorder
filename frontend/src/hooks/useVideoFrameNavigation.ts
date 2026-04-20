import { useCallback } from 'react'

// ==================== Types ====================

export interface UseVideoFrameNavigationResult {
  nextFrame: () => void
  prevFrame: () => void
  supportsFrameCallback: boolean
}

// ==================== Constants ====================

/** Default frame rate for frame-level navigation (frames per second) */
const DEFAULT_FRAME_RATE = 30

// ==================== Hook ====================

/**
 * Custom hook for frame-level video navigation
 *
 * Provides functions to navigate video by single frames.
 * Frame time is calculated from the frame rate (default 30fps).
 *
 * @param videoRef - React ref to the HTMLVideoElement
 * @returns Object with navigation functions and browser support status
 */
export function useVideoFrameNavigation(videoRef: React.RefObject<HTMLVideoElement>) {
  const supportsFrameCallback = useCallback(() => {
    const video = videoRef.current
    if (!video) return false

    // More defensive check with explicit property access
    return 'requestVideoFrameCallback' in video &&
           typeof video.requestVideoFrameCallback === 'function'
  }, [videoRef])

  const nextFrame = useCallback(() => {
    const video = videoRef.current
    if (!video || !Number.isFinite(video.duration)) return

    // Calculate frame time from frame rate (default 30fps)
    const frameTime = 1 / DEFAULT_FRAME_RATE
    const newTime = Math.min(video.duration, video.currentTime + frameTime)
    video.currentTime = newTime
  }, [videoRef])

  const prevFrame = useCallback(() => {
    const video = videoRef.current
    if (!video || !Number.isFinite(video.duration)) return

    // Calculate frame time from frame rate (default 30fps)
    const frameTime = 1 / DEFAULT_FRAME_RATE
    const newTime = Math.max(0, video.currentTime - frameTime)
    video.currentTime = newTime
  }, [videoRef])

  return {
    nextFrame,
    prevFrame,
    supportsFrameCallback: supportsFrameCallback(),
  }
}
