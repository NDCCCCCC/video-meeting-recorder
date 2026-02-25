// HLS 实时预览组件

import { useState, useEffect, useRef, useCallback } from 'react'
import { Button, Modal, Alert, Space } from 'antd'
import { ReloadOutlined, EyeOutlined } from '@ant-design/icons'
import { getTaskPreview } from '../api/task'
import type { VideoRecordingTask } from '../types/task'
import Hls from 'hls.js'

interface HLSPreviewProps {
  taskId: number
  taskName: string
  status: string
}

// HLS 播放器组件（使用 hls.js 库）
function HLSPlayer({ src, onError }: { src: string; onError: () => void }) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  // 使用 ref 存储稳定的 onError 回调，避免 useEffect 重复触发
  const onErrorRef = useRef(onError)
  onErrorRef.current = onError

  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    // 清理函数
    const cleanup = () => {
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }
    }

    // 加载 HLS
    const loadHLS = () => {
      cleanup()

      try {
        // 检查是否支持 HLS.js
        if (Hls.isSupported()) {
          // 使用 hls.js - 直播流优化配置
          const hls = new Hls({
            debug: false,
            enableWorker: true,
            lowLatencyMode: true,
            backBufferLength: 30, // 减少后缓冲，降低内存占用
            maxBufferLength: 30,   // 最大缓冲30秒
            maxMaxBufferLength: 60,
            liveSyncDuration: 3,   // 直播同步延迟，尽量接近直播边缘
            liveMaxLatencyDuration: 10, // 最大延迟10秒
            liveDurationInfinity: true,  // 直播流模式
          })

          hlsRef.current = hls

          hls.loadSource(src)
          hls.attachMedia(video)

          hls.on(Hls.Events.MANIFEST_PARSED, () => {
            video.play().catch(err => {
              console.warn('Auto-play prevented:', err)
            })
          })

          hls.on(Hls.Events.ERROR, (_event, data) => {
            if (data.fatal) {
              switch (data.type) {
                case Hls.ErrorTypes.NETWORK_ERROR:
                  console.error('HLS network error:', data)
                  hls.startLoad()
                  break
                case Hls.ErrorTypes.MEDIA_ERROR:
                  console.error('HLS media error:', data)
                  hls.recoverMediaError()
                  break
                default:
                  console.error('HLS fatal error:', data)
                  cleanup()
                  onErrorRef.current()
                  break
              }
            }
          })
        } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
          // Safari 原生支持
          video.src = src
          video.play().catch(err => {
            console.warn('Auto-play prevented:', err)
          })
        } else {
          onErrorRef.current()
        }
      } catch (error) {
        console.error('HLS load error:', error)
        onErrorRef.current()
      }
    }

    loadHLS()

    return cleanup
  }, [src]) // 只依赖 src，避免 onError 变化导致重建

  return (
    <video
      ref={videoRef}
      // 直播模式不显示控制条（进度条、时间等）
      muted    // 静音以提高自动播放成功率
      playsInline // 移动端防止全屏
      autoPlay  // 自动播放
      style={{ width: '100%', maxHeight: '450px', backgroundColor: '#000' }}
    />
  )
}

export function HLSPreview({ taskId, taskName, status }: HLSPreviewProps) {
  const [visible, setVisible] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>()
  const [hlsUrl, setHlsUrl] = useState<string>()
  const [currentStatus, setCurrentStatus] = useState(status)
  const [isPreparing, setIsPreparing] = useState(false)
  const [retryCount, setRetryCount] = useState(0)
  const retryTimerRef = useRef<NodeJS.Timeout | null>(null)

  const MAX_RETRY_COUNT = 10  // 最大重试次数

  // 清理定时器
  useEffect(() => {
    return () => {
      if (retryTimerRef.current) {
        clearTimeout(retryTimerRef.current)
      }
    }
  }, [])

  // 使用 useCallback 稳定回调函数，避免子组件重建
  const handlePlayerError = useCallback(() => {
    setError('直播连接中断，请刷新重试')
  }, [])

  const openPreview = async () => {
    setVisible(true)
    setLoading(true)
    setError(undefined)
    setHlsUrl(undefined)
    setIsPreparing(false)
    setRetryCount(0)  // 重置重试计数

    // 清理之前的定时器
    if (retryTimerRef.current) {
      clearTimeout(retryTimerRef.current)
      retryTimerRef.current = null
    }

    try {
      const response = await getTaskPreview(taskId)
      if (response.data) {
        if (response.data.ready === false) {
          // HLS 正在准备中
          setIsPreparing(true)
          setError(response.data.message || 'HLS预览正在准备中，请稍后刷新')
          // 自动重试：3秒后自动刷新（最多10次）
          if (retryCount < MAX_RETRY_COUNT) {
            retryTimerRef.current = setTimeout(() => {
              setRetryCount(prev => prev + 1)
              handleRefresh()
            }, 3000)
          } else {
            setError('HLS预览初始化超时，请关闭后重试')
            setIsPreparing(false)
          }
        } else if (response.data.playback_url) {
          // HLS 已就绪，使用 API 返回的带 token 的 URL
          setHlsUrl(response.data.playback_url)
          setCurrentStatus(response.data.status)
          setIsPreparing(false)
          setError(undefined)
          setRetryCount(0)  // 成功后重置计数
        }
      }
    } catch (err) {
      const error = err as Error & { response?: { data?: { message?: string } } }
      setError(error.response?.data?.message || error.message || '加载预览失败')
      setIsPreparing(false)
    } finally {
      setLoading(false)
    }
  }

  const handleRefresh = useCallback(() => {
    openPreview()
  }, [taskId]) // 只依赖 taskId

  const handleClose = useCallback(() => {
    // 清理定时器
    if (retryTimerRef.current) {
      clearTimeout(retryTimerRef.current)
      retryTimerRef.current = null
    }
    setVisible(false)
    setHlsUrl(undefined)
    setError(undefined)
  }, [])

  return (
    <>
      <Button
        icon={<EyeOutlined />}
        onClick={openPreview}
        size="small"
        disabled={status !== 'recording'}
        title={status === 'recording' ? '实时预览' : '仅录制中可预览'}
      >
        预览
      </Button>

      <Modal
        title={`${taskName} - 实时预览`}
        open={visible}
        onCancel={handleClose}
        footer={null}
        width={800}
        // 移除 destroyOnClose，避免重新打开时重建组件
      >
        {loading && (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <Space direction="vertical">
              <div>加载中...</div>
            </Space>
          </div>
        )}

        {error && (
          <Alert
            type={isPreparing ? "warning" : "error"}
            message={error}
            action={
              <Button size="small" onClick={handleRefresh}>
                <ReloadOutlined /> 刷新
              </Button>
            }
            style={{ marginBottom: 16 }}
          />
        )}

        {hlsUrl && !error && (
          <>
            <HLSPlayer src={hlsUrl} onError={handlePlayerError} />
            <div style={{ marginTop: 16, color: '#999', fontSize: 12 }}>
              <Space>
                <span>状态: {currentStatus === 'recording' ? '录制中' : currentStatus}</span>
                <span>•</span>
                <span>预览延迟约 3-5 秒</span>
              </Space>
            </div>
          </>
        )}

        {!loading && !error && !hlsUrl && (
          <Alert
            type="warning"
            message="暂无预览可用"
            description="该任务暂无可用的 HLS 预览流"
          />
        )}
      </Modal>
    </>
  )
}

// 用于在表格中渲染的包装组件
export function RenderTaskPreview(task: VideoRecordingTask) {
  return <HLSPreview taskId={task.id} taskName={task.name} status={task.status} />
}
