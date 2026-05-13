import { Select } from 'antd'
import { DashboardOutlined } from '@ant-design/icons'
import { memo, useState, useCallback, useEffect } from 'react'

// ==================== 常量 ====================

export const SPEED_OPTIONS = [
  { label: '0.5x', value: 0.5 },
  { label: '1x', value: 1.0 },
  { label: '1.25x', value: 1.25 },
  { label: '1.5x', value: 1.5 },
  { label: '2x', value: 2.0 },
  { label: '3x', value: 3.0 },
  { label: '5x', value: 5.0 },
]

// ==================== Hook ====================

export function usePlaybackSpeed(videoRef: React.RefObject<HTMLVideoElement | null>) {
  const [playbackRate, setPlaybackRate] = useState(1.0)

  const changeSpeed = useCallback((rate: number) => {
    const video = videoRef.current
    if (video) {
      video.playbackRate = rate
      setPlaybackRate(rate)
    }
  }, [videoRef])

  // Re-apply speed after video events (seek, load, etc.)
  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    const handleRateChange = () => setPlaybackRate(video.playbackRate)

    video.addEventListener('ratechange', handleRateChange)
    return () => video.removeEventListener('ratechange', handleRateChange)
  }, [videoRef])

  return { playbackRate, changeSpeed }
}

// ==================== 组件接口 ====================

interface PlaybackSpeedControlProps {
  currentSpeed: number
  onSpeedChange: (speed: number) => void
  style?: React.CSSProperties
}

// ==================== 组件 ====================

// 使用 memo 包裹组件避免不必要的重渲染 (rerender-memo)
export const PlaybackSpeedControl = memo(function PlaybackSpeedControl({
  currentSpeed,
  onSpeedChange,
  style
}: PlaybackSpeedControlProps) {
  return (
    <Select
      value={currentSpeed}
      onChange={onSpeedChange}
      options={SPEED_OPTIONS}
      style={{ width: 80, ...style }}
      size="small"
      prefix={<DashboardOutlined />}
    />
  )
})
