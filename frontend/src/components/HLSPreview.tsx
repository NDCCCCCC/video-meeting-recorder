// HLS 实时预览组件

import { useState, useEffect, useRef } from 'react'
import { Button, Modal, Alert, Space } from 'antd'
import { PlayCircleOutlined, ReloadOutlined, EyeOutlined } from '@ant-design/icons'
import { getTaskPreview, getHLSStreamUrl } from '../api/task'
import type { VideoRecordingTask } from '../types/task'

interface HLSPreviewProps {
  taskId: number
  taskName: string
  status: string
}

// 简单的 HLS 播放器组件（不依赖外部库）
function HLSPlayer({ src, onError }: { src: string; onError: () => void }) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    // 加载 HLS
    const loadHLS = async () => {
      try {
        // 使用原生的 HLS 支持（Safari）或者 fetch MSE
        if (video.canPlayType('application/vnd.apple.mpegurl')) {
          // Safari 原生支持
          video.src = src
        } else {
          // 其他浏览器，使用 Media Source Extensions
          const response = await fetch(src)
          const m3u8 = await response.text()

          // 解析 m3u8 获取 segment 列表
          const segments = m3u8.split('\n')
            .filter(line => line.trim() && !line.startsWith('#'))
            .filter(line => line.endsWith('.ts'))

          if (segments.length === 0) {
            onError()
            return
          }

          // 创建 MediaSource
          const mediaSource = new MediaSource()
          video.src = URL.createObjectURL(mediaSource)

          mediaSource.addEventListener('sourceopen', async () => {
            const sourceBuffer = mediaSource.addSourceBuffer('video/mp2t')

            // 依次加载并添加 segments
            for (const segment of segments) {
              const segmentUrl = new URL(segment, src).href
              const segmentResponse = await fetch(segmentUrl)
              const segmentData = await segmentResponse.arrayBuffer()

              await new Promise<void>((resolve) => {
                sourceBuffer.addEventListener('updateend', () => resolve(), { once: true })
                sourceBuffer.appendBuffer(segmentData)
              })
            }

            mediaSource.endOfStream()
          })
        }
      } catch (error) {
        console.error('HLS load error:', error)
        onError()
      }
    }

    loadHLS()
  }, [src, onError])

  return (
    <video
      ref={videoRef}
      controls
      autoPlay
      style={{ width: '100%', maxHeight: '450px' }}
      onError={onError}
    >
      您的浏览器不支持 HLS 播放
    </video>
  )
}

export function HLSPreview({ taskId, taskName, status }: HLSPreviewProps) {
  const [visible, setVisible] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string>()
  const [hlsUrl, setHlsUrl] = useState<string>()
  const [currentStatus, setCurrentStatus] = useState(status)

  const openPreview = async () => {
    setVisible(true)
    setLoading(true)
    setError(undefined)

    try {
      const response = await getTaskPreview(taskId)
      if (response.data?.playback_url) {
        // 构建完整的 HLS URL
        const baseUrl = response.data.playback_url
        const fullUrl = `${getHLSStreamUrl(taskId, 'index.m3u8').replace('/stream/', '/preview/stream/')}`
        setHlsUrl(fullUrl)
        setCurrentStatus(response.data.status)
      }
    } catch (err: any) {
      setError(err.message || '加载预览失败')
    } finally {
      setLoading(false)
    }
  }

  const handlePlayerError = () => {
    setError('直播连接中断，请刷新重试')
  }

  const handleRefresh = () => {
    openPreview()
  }

  const handleClose = () => {
    setVisible(false)
    setHlsUrl(undefined)
    setError(undefined)
  }

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
        destroyOnClose
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
            type="error"
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
                <span>预览延迟约 3-10 秒</span>
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
