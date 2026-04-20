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
import { PlaybackSpeedControl, usePlaybackSpeed, SPEED_OPTIONS } from './PlaybackSpeedControl'

// ==================== 常量 ====================
const SKIP_SECONDS = 10
const TIME_UPDATE_DEBOUNCE_MS = 1000  // Debounce timeupdate events to avoid excessive updates

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
}

// ==================== 主组件 ====================

export function VideoPreviewPanel({
  videoFileId,
  currentSlide,
  onSlideClick,
  style,
  autoPlay = false,
  showControls = true,
}: VideoPreviewPanelProps) {
  // ==================== 状态 ====================
  const videoRef = useRef<HTMLVideoElement>(null)
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
  const { playbackRate: currentPlaybackRate, changeSpeed } = usePlaybackSpeed(videoRef)

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
            // WR-02: Add validation for slide_number and timestamp
            if (ts.slide_number && typeof ts.timestamp === 'number') {
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

      if (closestSlide !== undefined && minDiff < 5) {  // Within 5 seconds
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
    const container = containerRef.current
    if (!container) return

    if (!document.fullscreenElement) {
      container.requestFullscreen().catch(() => {
        message.error('进入全屏失败')
      })
    } else {
      document.exitFullscreen()
    }
  }, [])

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
                  // WR-08: Use explicit undefined check instead of falsy check
                  if (currentSlide !== undefined) {
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
          }}
        >
          <video
            ref={videoRef}
            src={videoUrl}
            style={{
              width: '100%',
              maxHeight: '400px',
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
              }}
            >
              {/* 进度条 */}
              <input
                type="range"
                min={0}
                max={duration}
                value={currentTime}
                onChange={(e) => handleSeek(Number(e.target.value))}
                style={{
                  width: '100%',
                  cursor: 'pointer',
                }}
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
                  <span style={{ color: '#fff', marginLeft: '8px' }}>
                    {formatTime(currentTime)} / {formatTime(duration)}
                  </span>
                  <PlaybackSpeedControl
                    videoRef={videoRef}
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

        {timestampMap.size > 0 && currentSlide && (
          <div style={{ marginTop: 12, fontSize: '12px', color: '#666' }}>
            当前幻灯片: {currentSlide} | 时间戳: {formatTime(timestampMap.get(currentSlide) || 0)}
          </div>
        )}
      </Card>
    </div>
  )
}
