import { Button, Space, Tooltip } from 'antd'
import { StepForwardOutlined, UndoOutlined } from '@ant-design/icons'
import { memo } from 'react'
import { useVideoFrameNavigation } from '../hooks/useVideoFrameNavigation'

// ==================== Types ====================

interface FrameNavigationProps {
  videoRef: React.RefObject<HTMLVideoElement | null>
  disabled?: boolean
}

// ==================== Component ====================

/**
 * Frame navigation component for video player
 *
 * Provides +/-1 frame buttons for precise video navigation.
 * Only renders in browsers that support requestVideoFrameCallback API (Chrome/Edge).
 *
 * @param videoRef - React ref to the HTMLVideoElement
 * @param disabled - Whether the controls are disabled
 */
// 使用 memo 包裹组件避免不必要的重渲染 (rerender-memo)
export const FrameNavigation = memo(function FrameNavigation({ videoRef, disabled = false }: FrameNavigationProps) {
  const { nextFrame, prevFrame, supportsFrameCallback } = useVideoFrameNavigation(videoRef)

  // Don't render if browser doesn't support frame-level navigation
  if (!supportsFrameCallback) {
    return null
  }

  return (
    <Space size="small">
      <Tooltip title="上一帧 (Shift+←)">
        <Button
          type="text"
          icon={<UndoOutlined />}
          onClick={prevFrame}
          disabled={disabled}
          size="small"
          style={{ color: '#fff' }}
        >
          -1帧
        </Button>
      </Tooltip>
      <Tooltip title="下一帧 (Shift+→)">
        <Button
          type="text"
          icon={<StepForwardOutlined />}
          onClick={nextFrame}
          disabled={disabled}
          size="small"
          style={{ color: '#fff' }}
        >
          +1帧
        </Button>
      </Tooltip>
    </Space>
  )
})
