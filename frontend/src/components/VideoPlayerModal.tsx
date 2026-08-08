// 视频播放器模态框组件

import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { Modal, Button, Space, Alert, message, Slider } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  StepForwardOutlined,
  StepBackwardOutlined,
  DownloadOutlined,
  SoundOutlined,
  FullscreenOutlined,
} from '@ant-design/icons'
import type { VideoFile } from '../types/video-file'
import { getVideoPlaybackUrl, downloadVideoFile } from '../api/video-file'
import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts'
import { FrameNavigation } from './FrameNavigation'

// ==================== 常量 ====================
const PLAYBACK_RATES: readonly number[] = [0.5, 1, 1.25, 1.5, 2]

/**
 * Number of seconds to skip when using skip buttons
 * Also used for keyboard shortcuts (J/L keys and Arrow keys without Shift)
 */
const SKIP_SECONDS = 10

// 样式常量
const STYLES = {
  container: {
    position: 'relative',
    backgroundColor: '#000',
    borderRadius: '8px',
    overflow: 'hidden',
  } as const,
  loadingOverlay: {
    position: 'absolute',
    inset: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: '#000',
    zIndex: 1,
  } as const,
  errorOverlay: {
    position: 'absolute',
    inset: 0,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1,
  } as const,
  video: {
    width: '100%',
    maxHeight: '500px',
    display: 'block',
  } as const,
  controlBar: {
    position: 'absolute',
    bottom: 0,
    left: 0,
    right: 0,
    background: 'linear-gradient(transparent, rgba(0,0,0,0.8))',
    padding: '12px 16px',
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  } as const,
  fileInfo: {
    marginTop: 16,
    padding: '12px',
    backgroundColor: '#f5f5f5',
    borderRadius: '4px',
  } as const,
  controlBtn: {
    color: '#fff',
  } as const,
} as const

// ==================== 工具函数 ====================

// 格式化时间（秒 -> MM:SS 或 HH:MM:SS）
type TimeValue = number | undefined | null

function formatTime(seconds: TimeValue): string {
  if (!seconds || !Number.isFinite(seconds)) return '0:00'

  const s = Math.floor(seconds)
  const hours = Math.floor(s / 3600)
  const minutes = Math.floor((s % 3600) / 60)
  const secs = s % 60

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

// 控制按钮组件
function ControlButton({
  icon,
  onClick,
  title,
  disabled = false,
}: {
  icon: React.ReactNode
  onClick: () => void
  title: string
  disabled?: boolean
}) {
  return (
    <Button
      type="text"
      icon={icon}
      onClick={onClick}
      title={title}
      disabled={disabled}
      style={STYLES.controlBtn}
    />
  )
}

// ==================== 主组件 ====================

interface VideoPlayerModalProps {
  file: VideoFile
  visible: boolean
  onClose: () => void
}

export function VideoPlayerModal({ file, visible, onClose }: VideoPlayerModalProps) {
  // ==================== 状态 ====================
  const videoRef = useRef<HTMLVideoElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [volume, setVolume] = useState(1) // Store pre-mute volume
  const [actualVolume, setActualVolume] = useState(1) // Actual volume applied to video
  const [muted, setMuted] = useState(false)
  const [playbackRate, setPlaybackRate] = useState(1)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(true)
  // video_playback_token：5min TTL，需异步获取而非直接拼接 URL
  const [videoUrl, setVideoUrl] = useState<string | undefined>(undefined)

  const isNativelySupported = useMemo(() => file.format === 'mp4', [file.format])

  // 异步加载播放 URL；visibility/file.id 变化时重新获取
  useEffect(() => {
    if (!visible) {
      setVideoUrl(undefined)
      return
    }
    let cancelled = false
    setError(undefined)
    getVideoPlaybackUrl(file.id)
      .then((res) => {
        if (cancelled) return
        if (res.data?.playback_url) {
          setVideoUrl(res.data.playback_url)
        } else {
          setError('无法获取播放链接')
        }
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof Error ? err.message : '获取播放链接失败')
      })
    return () => {
      cancelled = true
    }
  }, [file.id, visible])

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

  const handleSkip = useCallback(
    (seconds: number) => {
      const video = videoRef.current
      if (!video || !duration) return
      video.currentTime = Math.max(0, Math.min(duration, video.currentTime + seconds))
    },
    [duration]
  )

  const handleSeek = useCallback((value: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = value
  }, [])

  const handlePlaybackRate = useCallback(() => {
    const currentIndex = PLAYBACK_RATES.indexOf(playbackRate)
    const nextIndex = (currentIndex + 1) % PLAYBACK_RATES.length
    const nextRate = PLAYBACK_RATES[nextIndex]

    setPlaybackRate(nextRate)
    const video = videoRef.current
    if (video) {
      video.playbackRate = nextRate
      message.info(`播放速度: ${nextRate}x`)
    }
  }, [playbackRate])

  const handleVolumeChange = useCallback(
    (value: number) => {
      const video = videoRef.current
      if (!video) return

      // Batch state updates together (React 18 automatic batching)
      const newMutedState = value > 0 ? false : muted
      setVolume(value)
      setActualVolume(value)
      if (newMutedState !== muted) {
        setMuted(newMutedState)
      }

      video.volume = value
    },
    [muted]
  )

  const handleMuteToggle = useCallback(() => {
    const video = videoRef.current
    if (!video) return

    if (muted) {
      // Unmute: restore previous volume
      const restoreVolume = volume > 0 ? volume : 0.5
      setMuted(false)
      setActualVolume(restoreVolume)
      video.volume = restoreVolume
      video.muted = false
    } else {
      // Mute: store current volume and set to 0
      setMuted(true)
      setVolume(video.volume)
      setActualVolume(0)
      video.muted = true
    }
  }, [muted, volume])

  const handleSeekWithInfinity = useCallback(
    (seconds: number) => {
      const video = videoRef.current
      if (!video || !duration || !Number.isFinite(video.duration)) return

      if (seconds === Infinity) {
        video.currentTime = video.duration
      } else if (seconds === -Infinity) {
        video.currentTime = 0
      } else {
        video.currentTime = Math.max(0, Math.min(duration, video.currentTime + seconds))
      }
    },
    [duration]
  )

  // ==================== 全屏控制 ====================
  const handleFullscreen = useCallback(() => {
    const container = containerRef.current
    if (!container) return

    if (!document.fullscreenElement) {
      container.requestFullscreen().catch((err: Error) => {
        if (err.name === 'NotAllowedError') {
          message.error('请在浏览器设置中允许全屏权限')
        } else {
          message.error(`进入全屏失败: ${err.message}`)
        }
      })
    } else {
      document.exitFullscreen().catch((err: Error) => {
        message.error(`退出全屏失败: ${err.message}`)
      })
    }
  }, [])

  // ==================== 下载 ====================
  // 下载走 Bearer header（downloadVideoFile），不用 playback URL；避免依赖 5min TTL
  const handleDownload = useCallback(() => {
    downloadVideoFile(file.id, file.file_name)
    message.success('开始下载')
  }, [file.id, file.file_name])

  // ==================== 关闭模态框 ====================
  const handleClose = useCallback(() => {
    const video = videoRef.current
    if (video) {
      video.pause()
      video.currentTime = 0
    }
    setIsPlaying(false)
    setCurrentTime(0)
    setError(undefined)
    onClose()
  }, [onClose])

  // ==================== 状态重置 ====================
  useEffect(() => {
    if (visible) {
      setIsPlaying(false)
      setCurrentTime(0)
      setDuration(0)
      setError(undefined)
      setLoading(true)
    }
  }, [visible])

  // ==================== 处理浏览器缓存的视频 ====================
  useEffect(() => {
    if (!visible) return

    const video = videoRef.current
    if (!video) return

    // 检查视频是否已经加载（处理浏览器缓存）
    const checkVideoLoaded = () => {
      if (video.readyState >= 1) {
        // HAVE_METADATA
        setDuration(video.duration)
        setLoading(false)
      }
    }

    // 延迟检查，给视频元素一些时间加载
    const timer = setTimeout(checkVideoLoaded, 100)

    // 如果视频已经加载，立即检查
    if (video.readyState >= 1) {
      clearTimeout(timer)
      checkVideoLoaded()
    }

    return () => clearTimeout(timer)
  }, [visible])

  // ==================== 键盘快捷键 ====================
  useKeyboardShortcuts({
    videoRef,
    isPlaying,
    playbackRate,
    volume: actualVolume,
    enabled: visible,
    onPlayPause: handlePlayPause,
    onSeek: handleSeekWithInfinity,
    onVolumeChange: handleVolumeChange,
    onPlaybackRateChange: setPlaybackRate,
    onFullscreen: handleFullscreen,
    onMuteToggle: handleMuteToggle,
  })

  // ==================== 不支持格式的内容 ====================
  const unsupportedContent = (
    <div
      style={{ padding: '60px 20px', textAlign: 'center', backgroundColor: '#000', color: '#fff' }}
    >
      <Alert
        type="warning"
        title="浏览器不支持直接播放此格式"
        description={
          <div>
            <p>
              <strong>{file.format?.toUpperCase()}</strong> 格式在浏览器中无法直接播放
            </p>
            <p style={{ marginTop: 16 }}>请下载后使用本地播放器（如 VLC、PotPlayer）观看</p>
          </div>
        }
        showIcon
        style={{ marginBottom: 24, textAlign: 'left' }}
      />
      <Button type="primary" icon={<DownloadOutlined />} onClick={handleDownload} size="large">
        下载文件
      </Button>
    </div>
  )

  // ==================== 渲染 ====================
  return (
    <Modal
      title={`${file.file_name} - 视频预览`}
      open={visible}
      onCancel={handleClose}
      footer={null}
      width={900}
      centered
    >
      <div ref={containerRef} style={STYLES.container}>
        {!isNativelySupported ? (
          unsupportedContent
        ) : (
          <>
            {loading && (
              <div style={STYLES.loadingOverlay}>
                <div style={{ color: '#fff' }}>加载中...</div>
              </div>
            )}

            {error && (
              <div style={STYLES.errorOverlay}>
                <Alert type="error" title={error} showIcon style={{ margin: 20 }} />
              </div>
            )}

            {videoUrl && (
              <video
                key={`${file.id}-${visible}-${videoUrl}`}
                ref={videoRef as React.RefObject<HTMLVideoElement>}
                src={videoUrl}
                style={STYLES.video}
                preload="metadata"
                muted={muted}
                {...({ volume: actualVolume } as unknown as React.VideoHTMLAttributes<HTMLVideoElement>)}
                onLoadedMetadata={() => {
                  const video = videoRef.current
                  if (video) {
                    setDuration(video.duration)
                    setLoading(false)
                  }
                }}
                onError={() => {
                  // token 过期（5min）或服务端 403 时尝试重新获取一次
                  const video = videoRef.current
                  const restoreTime = video ? video.currentTime : 0
                  setError('视频加载失败，正在重试...')
                  setLoading(true)
                  getVideoPlaybackUrl(file.id)
                    .then((res) => {
                      if (res.data?.playback_url && video) {
                        video.src = res.data.playback_url
                        video.currentTime = restoreTime
                        video.load()
                        setVideoUrl(res.data.playback_url)
                      }
                    })
                    .catch((err) => {
                      setError(err instanceof Error ? err.message : '视频加载失败')
                    })
                    .finally(() => {
                      setLoading(false)
                    })
                }}
              onTimeUpdate={() => {
                const video = videoRef.current
                if (video) {
                  setCurrentTime(video.currentTime)
                  // Sync playback rate from video element (in case external controls changed it)
                  setPlaybackRate(video.playbackRate)
                }
              }}
              onRateChange={() => {
                const video = videoRef.current
                if (video) setPlaybackRate(video.playbackRate)
              }}
            />
            )}

            {/* 自定义控制条 */}

            {/* 自定义控制条 */}
            <div style={STYLES.controlBar}>
              {/* 进度条 */}
              <Slider
                min={0}
                max={duration || 100}
                value={currentTime}
                onChange={handleSeek}
                trackStyle={{ backgroundColor: '#1890ff' }}
                handleStyle={{ borderColor: '#1890ff', backgroundColor: '#1890ff' }}
                disabled={!duration}
              />

              {/* 控制按钮行 */}
              <div
                style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}
              >
                <Space>
                  <ControlButton
                    icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                    onClick={handlePlayPause}
                    title={isPlaying ? '暂停' : '播放'}
                  />
                  <ControlButton
                    icon={<StepBackwardOutlined />}
                    onClick={() => handleSkip(-SKIP_SECONDS)}
                    title={`快退${SKIP_SECONDS}秒`}
                  />
                  <ControlButton
                    icon={<StepForwardOutlined />}
                    onClick={() => handleSkip(SKIP_SECONDS)}
                    title={`快进${SKIP_SECONDS}秒`}
                  />
                  <FrameNavigation videoRef={videoRef} disabled={!duration || loading} />
                  <Button
                    type="text"
                    onClick={handlePlaybackRate}
                    title="播放速度"
                    style={STYLES.controlBtn}
                  >
                    {playbackRate}x
                  </Button>
                  <span style={STYLES.controlBtn}>
                    {formatTime(currentTime)} / {formatTime(duration)}
                  </span>
                </Space>

                <Space>
                  <ControlButton
                    icon={<SoundOutlined />}
                    onClick={handleMuteToggle}
                    title={muted ? '取消静音 (M)' : '静音 (M)'}
                    disabled={!duration || loading}
                  />
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <Slider
                      min={0}
                      max={1}
                      step={0.1}
                      value={actualVolume}
                      onChange={handleVolumeChange}
                      style={{ width: '80px' }}
                    />
                  </div>
                  <ControlButton
                    icon={<FullscreenOutlined />}
                    onClick={handleFullscreen}
                    title="全屏 (F)"
                    disabled={!duration || loading}
                  />
                  <ControlButton
                    icon={<DownloadOutlined />}
                    onClick={handleDownload}
                    title="下载"
                  />
                </Space>
              </div>
            </div>
          </>
        )}
      </div>

      {/* 文件信息 */}
      <div style={STYLES.fileInfo}>
        <Space size="large">
          <span>
            <strong>格式:</strong> {file.format?.toUpperCase()}
          </span>
          <span>
            <strong>大小:</strong> {(file.file_size / 1024 / 1024).toFixed(2)} MB
          </span>
          <span>
            <strong>时长:</strong> {formatTime(file.duration)}
          </span>
          <span>
            <strong>分辨率:</strong> {file.resolution || '-'}
          </span>
          <span>
            <strong>码率:</strong> {file.bitrate ? `${file.bitrate} kbps` : '-'}
          </span>
        </Space>
      </div>
    </Modal>
  )
}
