// 视频文件播放器组件（用于已录制文件的点播）

import { useState, useRef, useEffect, useCallback } from 'react'
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

interface VideoPlayerProps {
  file: VideoFile
  visible: boolean
  onClose: () => void
}

// 格式化时间（秒 -> MM:SS 或 HH:MM:SS）
function formatTime(seconds: number): string {
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

export function VideoPlayer({ file, visible, onClose }: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [volume, setVolume] = useState(1)
  const [playbackRate, setPlaybackRate] = useState(1)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(true)

  // 构建视频 URL（使用查询参数传递 token）
  const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
  const token = getToken()
  const videoUrl = token
    ? `${API_BASE_URL}/api/v1/files/${file.id}/download?token=${token}`
    : `${API_BASE_URL}/api/v1/files/${file.id}/download`

  // 检查是否支持浏览器原生播放
  const isNativelySupported = file.format === 'mp4'

  // 更新播放/暂停状态
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

  // 快进/快退（10秒）
  const handleSkip = useCallback((seconds: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = Math.max(0, Math.min(duration, video.currentTime + seconds))
  }, [duration])

  // 跳转到指定位置
  const handleSeek = useCallback((value: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = value
  }, [])

  // 调整播放速度
  const handlePlaybackRate = useCallback(() => {
    const rates = [0.5, 1, 1.25, 1.5, 2]
    const currentIndex = rates.indexOf(playbackRate)
    const nextIndex = (currentIndex + 1) % rates.length
    const nextRate = rates[nextIndex]
    setPlaybackRate(nextRate)

    const video = videoRef.current
    if (video) {
      video.playbackRate = nextRate
      message.info(`播放速度: ${nextRate}x`)
    }
  }, [playbackRate])

  // 调整音量
  const handleVolumeChange = useCallback((value: number) => {
    setVolume(value)
    const video = videoRef.current
    if (video) {
      video.volume = value
    }
  }, [])

  // 全屏切换
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

  // 下载文件
  const handleDownload = useCallback(() => {
    const token = localStorage.getItem('token')
    const url = `${videoUrl}?token=${token}`
    const link = document.createElement('a')
    link.href = url
    link.download = file.file_name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    message.success('开始下载')
  }, [videoUrl, file.file_name])

  // 视频事件处理
  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    const handleTimeUpdate = () => setCurrentTime(video.currentTime)
    const handleLoadedMetadata = () => {
      setDuration(video.duration)
      setLoading(false)
    }
    const handlePlay = () => setIsPlaying(true)
    const handlePause = () => setIsPlaying(false)
    const handleEnded = () => setIsPlaying(false)
    const handleError = () => {
      setError('视频加载失败，请检查文件是否存在或稍后重试')
      setLoading(false)
    }

    video.addEventListener('timeupdate', handleTimeUpdate)
    video.addEventListener('loadedmetadata', handleLoadedMetadata)
    video.addEventListener('play', handlePlay)
    video.addEventListener('pause', handlePause)
    video.addEventListener('ended', handleEnded)
    video.addEventListener('error', handleError)

    return () => {
      video.removeEventListener('timeupdate', handleTimeUpdate)
      video.removeEventListener('loadedmetadata', handleLoadedMetadata)
      video.removeEventListener('play', handlePlay)
      video.removeEventListener('pause', handlePause)
      video.removeEventListener('ended', handleEnded)
      video.removeEventListener('error', handleError)
    }
  }, [])

  // 当 visible 变化时重置状态并刷新 token
  useEffect(() => {
    if (visible) {
      setIsPlaying(false)
      setCurrentTime(0)
      setDuration(0)
      setError(undefined)
      setLoading(true)
    }
  }, [visible])

  // 关闭时重置状态
  const handleClose = useCallback(() => {
    const video = videoRef.current
    if (video) {
      video.pause()
      video.currentTime = 0
    }
    setIsPlaying(false)
    setCurrentTime(0)
    setError(undefined)
    setLoading(false)  // 修复：关闭时设置 loading=false，不是 true
    onClose()
  }, [onClose])

  // 播放控制按钮组件
  const ControlButton = ({
    icon,
    onClick,
    title,
    disabled = false,
  }: {
    icon: React.ReactNode
    onClick: () => void
    title: string
    disabled?: boolean
  }) => (
    <Button
      type="text"
      icon={icon}
      onClick={onClick}
      title={title}
      disabled={disabled}
      style={{ color: '#fff' }}
    />
  )

  return (
    <Modal
      title={`${file.file_name} - 视频预览`}
      open={visible}
      onCancel={handleClose}
      footer={null}
      width={900}
      centered
      destroyOnClose
    >
      <div
        ref={containerRef}
        style={{
          position: 'relative',
          backgroundColor: '#000',
          borderRadius: '8px',
          overflow: 'hidden',
        }}
      >
        {!isNativelySupported ? (
          <div
            style={{
              padding: '60px 20px',
              textAlign: 'center',
              backgroundColor: '#000',
              color: '#fff',
            }}
          >
            <Alert
              type="warning"
              message="浏览器不支持直接播放此格式"
              description={
                <div>
                  <p>
                    <strong>{file.format.toUpperCase()}</strong> 格式在浏览器中无法直接播放
                  </p>
                  <p style={{ marginTop: 16 }}>
                    请下载后使用本地播放器（如 VLC、PotPlayer）观看
                  </p>
                </div>
              }
              showIcon
              style={{ marginBottom: 24, textAlign: 'left' }}
            />
            <Button
              type="primary"
              icon={<DownloadOutlined />}
              onClick={handleDownload}
              size="large"
            >
              下载文件
            </Button>
          </div>
        ) : (
          <>
            {loading && (
              <div
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  right: 0,
                  bottom: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  backgroundColor: '#000',
                  zIndex: 1,
                }}
              >
                <div style={{ color: '#fff' }}>加载中...</div>
              </div>
            )}

            {error && (
              <div
                style={{
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  right: 0,
                  bottom: 0,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  zIndex: 1,
                }}
              >
                <Alert
                  type="error"
                  message={error}
                  showIcon
                  style={{ margin: 20 }}
                />
              </div>
            )}

            <video
              ref={videoRef}
              src={videoUrl}
              crossOrigin="anonymous"
              style={{
                width: '100%',
                maxHeight: '500px',
                display: 'block',
              }}
              preload="metadata"
            />

            {/* 自定义控制条 */}
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
              <Slider
                min={0}
                max={duration || 100}
                value={currentTime}
                onChange={handleSeek}
                tooltip={{ formatter: (value) => formatTime(value || 0) }}
                trackStyle={{ backgroundColor: '#1890ff' }}
                handleStyle={{ borderColor: '#1890ff', backgroundColor: '#1890ff' }}
                disabled={!duration}
              />

              {/* 控制按钮行 */}
              <div
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                }}
              >
                <Space>
                  {/* 播放/暂停 */}
                  <ControlButton
                    icon={
                      isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />
                    }
                    onClick={handlePlayPause}
                    title={isPlaying ? '暂停' : '播放'}
                  />

                  {/* 快退/快进 */}
                  <ControlButton
                    icon={<StepBackwardOutlined />}
                    onClick={() => handleSkip(-10)}
                    title="快退10秒"
                  />
                  <ControlButton
                    icon={<StepForwardOutlined />}
                    onClick={() => handleSkip(10)}
                    title="快进10秒"
                  />

                  {/* 播放速度 */}
                  <Button
                    type="text"
                    onClick={handlePlaybackRate}
                    title="播放速度"
                    style={{ color: '#fff', fontSize: '12px', minWidth: 'auto' }}
                  >
                    {playbackRate}x
                  </Button>

                  {/* 时间显示 */}
                  <span style={{ color: '#fff', fontSize: '12px' }}>
                    {formatTime(currentTime)} / {formatTime(duration)}
                  </span>
                </Space>

                <Space>
                  {/* 音量控制 */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <SoundOutlined style={{ color: '#fff' }} />
                    <Slider
                      min={0}
                      max={1}
                      step={0.1}
                      value={volume}
                      onChange={handleVolumeChange}
                      style={{ width: '80px' }}
                    />
                  </div>

                  {/* 全屏 */}
                  <ControlButton
                    icon={<FullscreenOutlined />}
                    onClick={handleFullscreen}
                    title="全屏"
                  />

                  {/* 下载 */}
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
      <div style={{ marginTop: 16, padding: '12px', backgroundColor: '#f5f5f5', borderRadius: '4px' }}>
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

// 用于在表格中渲染的包装组件
export function RenderVideoPreview(file: VideoFile) {
  const [visible, setVisible] = useState(false)

  return (
    <>
      <Button
        type="link"
        size="small"
        icon={<PlayCircleOutlined />}
        onClick={() => setVisible(true)}
        disabled={file.status !== 'ready'}
        title={file.status === 'ready' ? '播放视频' : '仅就绪状态可播放'}
      >
        播放
      </Button>
      <VideoPlayer file={file} visible={visible} onClose={() => setVisible(false)} />
    </>
  )
}
