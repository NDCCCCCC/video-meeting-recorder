// VideoPreviewPanel - Video player with timestamp synchronization for PPT slides

import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Card, Alert, Space, Button, message } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  StepForwardOutlined,
  StepBackwardOutlined,
  FullscreenOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import { getTimestampMap } from '../api/transcription'
import type { SlideTimestamp } from '../types/transcription'
import { getToken } from '../api/apiClient'
import { PlaybackSpeedControl, usePlaybackSpeed } from './PlaybackSpeedControl'
import EditableProgressBar from './EditableProgressBar'

// ==================== 常量 ====================
const SKIP_SECONDS = 10
const TIME_UPDATE_DEBOUNCE_MS = 1000  // Debounce timeupdate events to avoid excessive updates
const SYNC_THRESHOLD_SECONDS = 5  // Maximum time difference for video-slide sync

// ==================== 工具函数 ====================

// 格式化时间（秒 -> MM:SS 或 HH:MM:SS）
function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00'

  const s = Math.floor(seconds)
  const hours = Math.floor(s / 3600)
  const minutes = Math.floor((s % 3600) / 60)
  const secs = s % 60

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

// ==================== 组件接口 ====================

interface VideoPreviewPanelProps {
  videoFileId: number
  currentSlide?: number  // 1-based slide number
  onSlideClick?: (slideNumber: number) => void  // Callback for video -> slide sync
  style?: React.CSSProperties
  autoPlay?: boolean  // Auto-play video when seeking to slide
  showControls?: boolean  // Show custom playback controls
  videoRef?: React.RefObject<HTMLVideoElement | null>  // Optional external videoRef
}

// ==================== 主组件 ====================

export function VideoPreviewPanel({
  videoFileId,
  currentSlide,
  onSlideClick,
  style,
  autoPlay = false,
  showControls = true,
  videoRef: externalVideoRef,
}: VideoPreviewPanelProps) {
  // ==================== 状态 ====================
  // Use external videoRef if provided, otherwise create internal ref
  const internalVideoRef = useRef<HTMLVideoElement>(null)
  const videoRef = externalVideoRef || internalVideoRef
  const containerRef = useRef<HTMLDivElement>(null)

  const [isLoading, setIsLoading] = useState(true)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [error, setError] = useState<string>()
  const [timestampMap, setTimestampMap] = useState<Map<number, number>>(new Map())
  const [timestampError, setTimestampError] = useState<string>()
  const [playbackRate, setPlaybackRate] = useState(1.0)

  // Initialize playback speed hook
  const { changeSpeed } = usePlaybackSpeed(videoRef as React.RefObject<HTMLVideoElement>)

  // ==================== 计算值 ====================
  const videoUrl = useMemo(() => {
    const API_BASE_URL = import.meta.env.VITE_API_URL || ''
    const token = getToken()
    return token
      ? `${API_BASE_URL}/api/v1/files/${videoFileId}/download?token=${token}`
      : `${API_BASE_URL}/api/v1/files/${videoFileId}/download`
  }, [videoFileId])

  // ==================== 加载时间戳映射 ====================
  useEffect(() => {
    let mounted = true

    const loadTimestamps = async () => {
      try {
        setIsLoading(true)
        setTimestampError(undefined)

        const response = await getTimestampMap(videoFileId)

        if (!mounted) return

        if (response.data?.success && Array.isArray(response.data?.slide_timestamps)) {
          const map = new Map<number, number>()
          response.data.slide_timestamps.forEach((ts: SlideTimestamp) => {
            // WR-05: Add validation for positive slide_number and valid timestamp
            if (ts.slide_number && ts.slide_number > 0 && typeof ts.timestamp === 'number' && ts.timestamp >= 0) {
              map.set(ts.slide_number, ts.timestamp)
            }
          })
          setTimestampMap(map)

          if (map.size === 0) {
            setTimestampError('暂无时间戳数据，视频预览同步功能不可用')
          }
        } else {
          setTimestampError('无法加载时间戳数据')
        }
      } catch (err) {
        if (mounted) {
          console.error('Failed to load timestamp map:', err)
          setTimestampError('加载时间戳数据失败')
        }
      } finally {
        if (mounted) {
          setIsLoading(false)
        }
      }
    }

    loadTimestamps()

    return () => {
      mounted = false
    }
  }, [videoFileId])

  // ==================== 幻灯片 -> 视频同步 ====================
  useEffect(() => {
    if (currentSlide === undefined || currentSlide <= 0) return

    const timestamp = timestampMap.get(currentSlide)
    if (timestamp === undefined) return

    const video = videoRef.current
    if (!video || video.duration === 0) return

    // Seek to timestamp
    const wasPlaying = !video.paused
    video.currentTime = timestamp

    // Auto-play if enabled
    if (autoPlay && !wasPlaying) {
      video.play().catch(() => {
        // Ignore play errors (user may have paused)
      })
    }
  }, [currentSlide, timestampMap, autoPlay])

  // Reset playback speed when videoFileId changes (new video)
  useEffect(() => {
    const video = videoRef.current
    if (video) {
      video.playbackRate = 1.0
      setPlaybackRate(1.0)
    }
  }, [videoFileId])

  // ==================== 视频 -> 幻灯片同步（反向同步）====================
  useEffect(() => {
    if (!onSlideClick) return

    const video = videoRef.current
    if (!video) return

    let lastUpdateTime = 0

    const handleTimeUpdate = () => {
      const now = Date.now()
      if (now - lastUpdateTime < TIME_UPDATE_DEBOUNCE_MS) {
        return  // Debounce
      }
      lastUpdateTime = now

      const currentTime = video.currentTime

      // WR-06: Use constant for sync threshold and handle empty timestamp map
      if (timestampMap.size === 0) return

      // Find closest slide based on current timestamp
      let closestSlide: number | undefined
      let minDiff = Infinity

      for (const [slideNumber, timestamp] of timestampMap.entries()) {
        const diff = Math.abs(timestamp - currentTime)
        if (diff < minDiff) {
          minDiff = diff
          closestSlide = slideNumber
        }
      }

      if (closestSlide !== undefined && minDiff < SYNC_THRESHOLD_SECONDS) {
        onSlideClick(closestSlide)
      }
    }

    video.addEventListener('timeupdate', handleTimeUpdate)

    return () => {
      video.removeEventListener('timeupdate', handleTimeUpdate)
    }
  }, [timestampMap, onSlideClick])

  // ==================== 播放控制 ====================
  const handlePlayPause = useCallback(() => {
    const video = videoRef.current
    if (!video) return

    if (isPlaying) {
      video.pause()
    } else {
      video.play().catch(() => {
        message.error('播放失败，请稍后重试')
      })
    }
  }, [isPlaying])

  const handleSkip = useCallback((seconds: number) => {
    const video = videoRef.current
    if (!video || !duration) return
    video.currentTime = Math.max(0, Math.min(duration, video.currentTime + seconds))
  }, [duration])

  const handleSeek = useCallback((value: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = value
    video.playbackRate = playbackRate  // Restore speed after seek
  }, [playbackRate])

  // ==================== 全屏控制 ====================
  const handleFullscreen = useCallback(() => {
    const video = videoRef.current
    if (!video) return

    if (!document.fullscreenElement) {
      video.parentElement?.requestFullscreen().catch(() => {
        message.error('进入全屏失败')
      })
    } else {
      document.exitFullscreen()
    }
  }, [videoRef])

  // ==================== 视频事件处理 ====================
  const handleLoadedMetadata = useCallback(() => {
    const video = videoRef.current
    if (video) {
      setDuration(video.duration)
      setIsLoading(false)
    }
  }, [])

  const handleError = useCallback(() => {
    setError('视频加载失败，请检查文件是否存在或稍后重试')
    setIsLoading(false)
  }, [])

  const handleTimeUpdate = useCallback(() => {
    const video = videoRef.current
    if (video) {
      setCurrentTime(video.currentTime)
    }
  }, [])

  const handlePlay = useCallback(() => {
    setIsPlaying(true)
  }, [])

  const handlePause = useCallback(() => {
    setIsPlaying(false)
  }, [])

  // ==================== 渲染 ====================
  return (
    <div ref={containerRef} style={{ position: 'relative', ...style }}>
      <Card
        title="视频预览"
        size="small"
        extra={
          timestampMap.size > 0 && (
            <Space>
              <Button
                type="text"
                icon={<SyncOutlined />}
                size="small"
                onClick={() => {
                  // WR-08: Use explicit undefined check and has() for better defensive programming
                  if (currentSlide !== undefined && timestampMap.has(currentSlide)) {
                    const timestamp = timestampMap.get(currentSlide)
                    if (timestamp !== undefined && videoRef.current) {
                      videoRef.current.currentTime = timestamp
                    }
                  }
                }}
              >
                跳转到当前幻灯片
              </Button>
            </Space>
          )
        }
      >
        {isLoading && (
          <div style={{ padding: '40px', textAlign: 'center' }}>
            <SyncOutlined spin /> 加载中...
          </div>
        )}

        {timestampError && (
          <Alert
            type="warning"
            message={timestampError}
            showIcon
            style={{ marginBottom: 12 }}
          />
        )}

        {error && (
          <Alert
            type="error"
            message={error}
            showIcon
            style={{ marginBottom: 12 }}
          />
        )}

        <div
          style={{
            position: 'relative',
            backgroundColor: '#000',
            borderRadius: '8px',
            overflow: 'hidden',
            width: '100%',
            aspectRatio: '16 / 9',
          }}
        >
          <video
            ref={videoRef}
            src={videoUrl}
            style={{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: '100%',
              objectFit: 'contain',
              display: 'block',
            }}
            preload="metadata"
            onLoadedMetadata={handleLoadedMetadata}
            onError={handleError}
            onTimeUpdate={handleTimeUpdate}
            onPlay={handlePlay}
            onPause={handlePause}
          />

          {showControls && duration > 0 && (
            <div
              style={{
                position: 'absolute',
                bottom: 0,
                left: 0,
                right: 0,
                background: 'linear-gradient(transparent, rgba(0,0,0,0.8))',
                padding: '12px 16px',
                display: 'flex',
                flexDirection: 'column',
                gap: '8px',
                boxSizing: 'border-box',
              }}
            >
              {/* Editable progress bar with time input */}
              <EditableProgressBar
                currentTime={currentTime}
                duration={duration}
                onSeek={handleSeek}
              />

              {/* 控制按钮行 */}
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <Space>
                  <Button
                    type="text"
                    icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                    onClick={handlePlayPause}
                    style={{ color: '#fff' }}
                  />
                  <Button
                    type="text"
                    icon={<StepBackwardOutlined />}
                    onClick={() => handleSkip(-SKIP_SECONDS)}
                    style={{ color: '#fff' }}
                  />
                  <Button
                    type="text"
                    icon={<StepForwardOutlined />}
                    onClick={() => handleSkip(SKIP_SECONDS)}
                    style={{ color: '#fff' }}
                  />
                  <PlaybackSpeedControl
                    currentSpeed={playbackRate}
                    onSpeedChange={(speed) => {
                      changeSpeed(speed)
                      setPlaybackRate(speed)
                    }}
                    style={{ marginLeft: 8 }}
                  />
                </Space>

                <Button
                  type="text"
                  icon={<FullscreenOutlined />}
                  onClick={handleFullscreen}
                  style={{ color: '#fff' }}
                />
              </div>
            </div>
          )}
        </div>

        {/* WR-07: Only show timestamp if it exists for current slide */}
        {timestampMap.size > 0 && currentSlide && timestampMap.has(currentSlide) && (
          <div style={{ marginTop: 12, fontSize: '12px', color: '#666' }}>
            当前幻灯片: {currentSlide} | 时间戳: {formatTime(timestampMap.get(currentSlide) || 0)}
          </div>
        )}
      </Card>
    </div>
  )
}
