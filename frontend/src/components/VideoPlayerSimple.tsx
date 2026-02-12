// 视频文件播放器组件（简化版用于调试）

import { useState, useRef, useEffect } from 'react'
import { Modal, Button, Space, Alert, message } from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  StepForwardOutlined,
  StepBackwardOutlined,
  DownloadOutlined,
  FullscreenOutlined,
} from '@ant-design/icons'
import type { VideoFile } from '../types/video-file'

interface VideoPlayerProps {
  file: VideoFile
  visible: boolean
  onClose: () => void
}

// 格式化时间（秒 -> MM:SS 或 HH:MM:SS）
function formatTime(seconds: number): string {
  const minutes = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}

export function VideoPlayer({ file, visible, onClose }: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [error, setError] = useState<string>()
  const [loading, setLoading] = useState(true)

  // 构建视频 URL（使用查询参数传递 token）
  const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
  const token = localStorage.getItem('access_token')
  const videoUrl = token
    ? `${API_BASE_URL}/api/v1/files/${file.id}/download?token=${token}`
    : `${API_BASE_URL}/api/v1/files/${file.id}/download`

  // 重置状态当 modal 关闭/打开时
  useEffect(() => {
    if (visible) {
      setIsPlaying(false)
      setCurrentTime(0)
      setDuration(0)
      setError(undefined)
      setLoading(true)
    }
  }, [visible])

  // 视频事件处理
  useEffect(() => {
    const video = videoRef.current
    if (!video || !visible) {
      return
    }

    const handleTimeUpdate = () => setCurrentTime(video.currentTime)
    const handleLoadedMetadata = () => {
      setDuration(video.duration)
      setLoading(false)
    }
    const handlePlay = () => setIsPlaying(true)
    const handlePause = () => setIsPlaying(false)
    const handleEnded = () => setIsPlaying(false)
    const handleError = () => {
      setError('视频加载失败')
      setLoading(false)
    }

    // 添加事件监听
    video.addEventListener('timeupdate', handleTimeUpdate)
    video.addEventListener('loadedmetadata', handleLoadedMetadata)
    video.addEventListener('play', handlePlay)
    video.addEventListener('pause', handlePause)
    video.addEventListener('ended', handleEnded)
    video.addEventListener('error', handleError)

    // 确保视频已加载
    if (video.readyState >= 1) {
      setDuration(video.duration)
      setLoading(false)
    }

    // 尝试自动播放
    video.play().catch(() => {
      // Auto-play blocked by browser policy is OK
    })

    return () => {
      video.removeEventListener('timeupdate', handleTimeUpdate)
      video.removeEventListener('loadedmetadata', handleLoadedMetadata)
      video.removeEventListener('play', handlePlay)
      video.removeEventListener('pause', handlePause)
      video.removeEventListener('ended', handleEnded)
      video.removeEventListener('error', handleError)
    }
  }, [visible, file.id])

  const handlePlayPause = () => {
    const video = videoRef.current
    if (!video) return

    if (isPlaying) {
      video.pause()
    } else {
      video.play().catch(err => {
        console.error('[VideoPlayer] Play failed:', err)
        message.error('播放失败')
      })
    }
  }

  const handleSkip = (seconds: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = Math.max(0, Math.min(duration || 0, video.currentTime + seconds))
  }

  const handleSeek = (value: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = value
  }

  const handlePlaybackRate = () => {
    const rates = [1, 1.5, 2]
    const currentIndex = rates.indexOf((window as any).videoPlaybackRate || 1)
    const nextIndex = (currentIndex + 1) % rates.length
    const nextRate = rates[nextIndex]
    ;(window as any).videoPlaybackRate = nextRate
    message.info(`播放速度: ${nextRate}x`)
  }

  const handleFullscreen = () => {
    const container = containerRef.current
    if (!container) return

    if (!document.fullscreenElement) {
      container.requestFullscreen().catch(() => {
        message.error('进入全屏失败')
      })
    } else {
      document.exitFullscreen()
    }
  }

  const handleDownload = () => {
    const token = localStorage.getItem('access_token')
    const url = token
      ? `${API_BASE_URL}/api/v1/files/${file.id}/download?token=${token}`
      : `${API_BASE_URL}/api/v1/files/${file.id}/download`

    const link = document.createElement('a')
    link.href = url
    link.download = file.file_name
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    message.success('开始下载')
  }

  const handleClose = () => {
    const video = videoRef.current
    if (video) {
      video.pause()
      video.currentTime = 0
    }
    setIsPlaying(false)
    setCurrentTime(0)
    setError(undefined)
    setLoading(false)
    onClose()
  }

  // 检查是否支持浏览器原生播放
  const isNativelySupported = file.format === 'mp4'

  return (
    <Modal
      title={`${file.file_name} - 视频预览`}
      open={visible}
      onCancel={handleClose}
      footer={null}
      width={900}
      centered
      // 不使用 destroyOnHidden/destroyOnClose，保持组件挂载以避免 video 元素重新创建
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
                  backgroundColor: '000',
                  zIndex: 1,
                }}
              >
                <Alert
                  type="error"
                  message={error}
                  showIcon
                  style={{ margin: 20, textAlign: 'left' }}
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

            {/* 控制条 */}
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
                max={duration || 100}
                value={currentTime}
                onChange={(e) => handleSeek(parseFloat(e.target.value))}
                style={{ flex: 1, cursor: 'pointer' }}
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
                  <Button
                    type="text"
                    icon={isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                    onClick={handlePlayPause}
                    style={{ color: '#fff' }}
                    title={isPlaying ? '暂停' : '播放'}
                  />

                  {/* 快退/快进 */}
                  <Button
                    type="text"
                    icon={<StepBackwardOutlined />}
                    onClick={() => handleSkip(-10)}
                    style={{ color: '#fff' }}
                    title="快退10秒"
                  />
                  <Button
                    type="text"
                    icon={<StepForwardOutlined />}
                    onClick={() => handleSkip(10)}
                    style={{ color: '#fff' }}
                    title="快进10秒"
                  />

                  {/* 播放速度 */}
                  <Button
                    type="text"
                    onClick={handlePlaybackRate}
                    style={{ color: '#fff', fontSize: '12px', minWidth: 'auto' }}
                    title="播放速度"
                  >
                    {((window as any).videoPlaybackRate || 1)}x)
                  </Button>

                  {/* 时间显示 */}
                  <span style={{ color: '#fff', fontSize: '12px' }}>
                    {formatTime(currentTime)} / {formatTime(duration || 0)}
                  </span>

                  {/* 全屏 */}
                  <Button
                    type="text"
                    icon={<FullscreenOutlined />}
                    onClick={handleFullscreen}
                    style={{ color: '#fff' }}
                    title="全屏"
                  />

                  {/* 下载 */}
                  <Button
                    type="text"
                    icon={<DownloadOutlined />}
                    onClick={handleDownload}
                    style={{ color: '#fff' }}
                    title="下载"
                  />
                </Space>
              </div>
            </div>
          </>)}
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
