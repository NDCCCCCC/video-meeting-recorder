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
import { getToken } from '../api/apiClient'

// ==================== 常量 ====================
const PLAYBACK_RATES: readonly number[] = [0.5, 1, 1.25, 1.5, 2]
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
  const [volume, setVolume] = useState(1)
  const [playbackRate, setPlaybackRate] = useState(1)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(true)

  // ==================== 计算值 ====================
  const videoUrl = useMemo(() => {
    const API_BASE_URL = import.meta.env.VITE_API_URL || ''
    const token = getToken()
    return token
      ? `${API_BASE_URL}/api/v1/files/${file.id}/download?token=${token}`
      : `${API_BASE_URL}/api/v1/files/${file.id}/download`
  }, [file.id])

  const isNativelySupported = useMemo(() => file.format === 'mp4', [file.format])

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

  const handleVolumeChange = useCallback((value: number) => {
    setVolume(value)
    const video = videoRef.current
    if (video) video.volume = value
  }, [])

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

  // ==================== 下载 ====================
  const handleDownload = useCallback(() => {
    const link = document.createElement('a')
    link.href = videoUrl
    link.download = file.file_name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    message.success('开始下载')
  }, [videoUrl, file.file_name])

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

  // ==================== 视频事件处理 ====================
  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    const handleTimeUpdate = () => setCurrentTime(video.currentTime)
    const handleLoadedMetadata = () => {
      setDuration(video.duration)
      setLoading(false)
    }
    const handleError = () => {
      setError('视频加载失败，请检查文件是否存在或稍后重试')
      setLoading(false)
    }

    video.addEventListener('timeupdate', handleTimeUpdate)
    video.addEventListener('loadedmetadata', handleLoadedMetadata)
    video.addEventListener('error', handleError)

    return () => {
      video.removeEventListener('timeupdate', handleTimeUpdate)
      video.removeEventListener('loadedmetadata', handleLoadedMetadata)
      video.removeEventListener('error', handleError)
    }
  }, [visible, videoUrl])

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

  // ==================== 不支持格式的内容 ====================
  const unsupportedContent = (
    <div style={{ padding: '60px 20px', textAlign: 'center', backgroundColor: '#000', color: '#fff' }}>
      <Alert
        type="warning"
        message="浏览器不支持直接播放此格式"
        description={
          <div>
            <p><strong>{file.format?.toUpperCase()}</strong> 格式在浏览器中无法直接播放</p>
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
        {!isNativelySupported ? unsupportedContent : (
          <>
            {loading && (
              <div style={STYLES.loadingOverlay}>
                <div style={{ color: '#fff' }}>加载中...</div>
              </div>
            )}

            {error && (
              <div style={STYLES.errorOverlay}>
                <Alert type="error" message={error} showIcon style={{ margin: 20 }} />
              </div>
            )}

            <video
              key={videoUrl}
              ref={videoRef}
              src={videoUrl}
              style={STYLES.video}
              preload="metadata"
            />

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
              <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
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
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <SoundOutlined style={STYLES.controlBtn} />
                    <Slider
                      min={0}
                      max={1}
                      step={0.1}
                      value={volume}
                      onChange={handleVolumeChange}
                      style={{ width: '80px' }}
                    />
                  </div>
                  <ControlButton
                    icon={<FullscreenOutlined />}
                    onClick={handleFullscreen}
                    title="全屏"
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
          <span><strong>格式:</strong> {file.format?.toUpperCase()}</span>
          <span><strong>大小:</strong> {(file.file_size / 1024 / 1024).toFixed(2)} MB</span>
          <span><strong>时长:</strong> {formatTime(file.duration)}</span>
          <span><strong>分辨率:</strong> {file.resolution || '-'}</span>
          <span><strong>码率:</strong> {file.bitrate ? `${file.bitrate} kbps` : '-'}</span>
        </Space>
      </div>
    </Modal>
  )
}
