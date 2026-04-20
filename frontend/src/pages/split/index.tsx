// 视频分割页面

import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Button,
  Space,
  Table,
  Modal,
  message,
  Alert,
  Spin,
  Tag,
} from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  StepForwardOutlined,
  StepBackwardOutlined,
  ArrowLeftOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as videoFileApi from '../../api/video-file'
import * as splitApi from '../../api/split'
import { TimelineWithMarkers } from '../../components/TimelineWithMarkers'
import { getToken } from '../../api/apiClient'
import type { VideoFile } from '../../types/video-file'

// ==================== 常量 ====================

const SKIP_SECONDS = 10
const POLL_INTERVAL = 2000 // 2秒轮询间隔

// ==================== 工具函数 ====================

function formatTime(seconds: number): string {
  if (!seconds || !Number.isFinite(seconds)) return '0:00'
  const s = Math.floor(seconds)
  const minutes = Math.floor(s / 60)
  const secs = s % 60
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

// ==================== 段落预览类型 ====================

interface SegmentPreview {
  index: number
  start: number
  end: number
  duration: number
}

// ==================== 主组件 ====================

export default function SplitPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  // ==================== 状态 ====================

  const [videoFile, setVideoFile] = useState<VideoFile | null>(null)
  const [markers, setMarkers] = useState<number[]>([])
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [isPlaying, setIsPlaying] = useState(false)
  const [splitting, setSplitting] = useState(false)
  const [splitProgress, setSplitProgress] = useState<string | null>(null)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const videoRef = useRef<HTMLVideoElement>(null)

  // ==================== 计算值 ====================

  const videoUrl = useMemo(() => {
    if (!id) return ''
    const API_BASE_URL = import.meta.env.VITE_API_URL || ''
    const token = getToken()
    return token
      ? `${API_BASE_URL}/api/v1/files/${id}/download?token=${token}`
      : `${API_BASE_URL}/api/v1/files/${id}/download`
  }, [id])

  // 段落预览数据
  const segmentPreviews: SegmentPreview[] = useMemo(() => {
    if (markers.length === 0) return []

    const sortedMarkers = [...markers].sort((a, b) => a - b)
    const segments: SegmentPreview[] = []

    // 第一个段落：0 -> markers[0]
    segments.push({
      index: 1,
      start: 0,
      end: sortedMarkers[0],
      duration: sortedMarkers[0],
    })

    // 中间段落：markers[i] -> markers[i+1]
    for (let i = 0; i < sortedMarkers.length - 1; i++) {
      segments.push({
        index: i + 2,
        start: sortedMarkers[i],
        end: sortedMarkers[i + 1],
        duration: sortedMarkers[i + 1] - sortedMarkers[i],
      })
    }

    // 最后一个段落：markers[last] -> duration
    segments.push({
      index: sortedMarkers.length + 1,
      start: sortedMarkers[sortedMarkers.length - 1],
      end: duration,
      duration: duration - sortedMarkers[sortedMarkers.length - 1],
    })

    return segments
  }, [markers, duration])

  // ==================== 加载视频文件 ====================

  useEffect(() => {
    if (!id) return

    const loadVideoFile = async () => {
      setLoading(true)
      setError(null)
      try {
        const response = await videoFileApi.getVideoFile(parseInt(id, 10))
        if (response.data) {
          setVideoFile(response.data)
          setDuration(response.data.duration)
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : '加载视频文件失败')
      } finally {
        setLoading(false)
      }
    }

    loadVideoFile()
  }, [id])

  // ==================== 视频播放控制 ====================

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
    setIsPlaying(!isPlaying)
  }, [isPlaying])

  const handleSkip = useCallback((seconds: number) => {
    const video = videoRef.current
    if (!video || !duration) return
    video.currentTime = Math.max(0, Math.min(duration, video.currentTime + seconds))
  }, [duration])

  const handleSeek = useCallback((time: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = time
    setCurrentTime(time)
  }, [])

  // ==================== 标记管理 ====================

  const handleMarkerAdd = useCallback((time: number) => {
    setMarkers(prev => {
      const newMarkers = [...prev, time].sort((a, b) => a - b)
      return newMarkers
    })
  }, [])

  const handleMarkerRemove = useCallback((time: number) => {
    setMarkers(prev => prev.filter(m => m !== time))
  }, [])

  // ==================== 分割执行 ====================

  const handleSplit = useCallback(async () => {
    if (!id || markers.length < 2) {
      message.warning('请至少添加 2 个标记点')
      return
    }

    Modal.confirm({
      title: '确认分割',
      content: `确认将视频分割为 ${markers.length + 1} 个段落？`,
      okText: '确认分割',
      cancelText: '取消',
      onOk: async () => {
        setSplitting(true)
        setSplitProgress('正在分割中...')

        try {
          // 提交分割任务
          const splitResponse = await splitApi.submitSplit(parseInt(id, 10), markers, false)

          if (splitResponse.data) {
            // 轮询分割状态
            const pollInterval = setInterval(async () => {
              try {
                const statusResponse = await splitApi.getSplitStatus(parseInt(id, 10))

                if (statusResponse.data) {
                  const { status } = statusResponse.data

                  if (status === 'completed') {
                    clearInterval(pollInterval)
                    setSplitProgress(null)
                    setSplitting(false)
                    message.success(`分割完成！已生成 ${markers.length + 1} 个视频段落`)
                    navigate('/files')
                  } else if (status === 'failed') {
                    clearInterval(pollInterval)
                    setSplitProgress(null)
                    setSplitting(false)
                    message.error('视频分割失败，请检查视频文件是否完整或尝试使用重新编码模式')
                  } else if (status === 'processing') {
                    setSplitProgress(`正在分割中...`)
                  }
                }
              } catch (err) {
                clearInterval(pollInterval)
                setSplitProgress(null)
                setSplitting(false)
                message.error(err instanceof Error ? err.message : '查询分割状态失败')
              }
            }, POLL_INTERVAL)
          }
        } catch (err) {
          setSplitProgress(null)
          setSplitting(false)
          message.error(err instanceof Error ? err.message : '提交分割任务失败')
        }
      },
    })
  }, [id, markers, navigate])

  // ==================== 表格列定义 ====================

  const columns: ColumnsType<SegmentPreview> = [
    {
      title: '段落',
      dataIndex: 'index',
      width: 80,
      align: 'center',
      render: (index: number) => <Tag color="blue">段落 {index}</Tag>,
    },
    {
      title: '开始时间',
      dataIndex: 'start',
      width: 120,
      render: (start: number) => formatTime(start),
    },
    {
      title: '结束时间',
      dataIndex: 'end',
      width: 120,
      render: (end: number) => formatTime(end),
    },
    {
      title: '预计时长',
      dataIndex: 'duration',
      width: 120,
      render: (dur: number) => formatTime(dur),
    },
  ]

  // ==================== 渲染 ====================

  if (loading) {
    return (
      <div style={{ padding: '24px', textAlign: 'center' }}>
        <Spin size="large" tip="加载中..." />
      </div>
    )
  }

  if (error || !videoFile) {
    return (
      <div style={{ padding: '24px' }}>
        <Alert
          type="error"
          message="加载失败"
          description={error || '视频文件不存在'}
          showIcon
          action={
            <Button size="small" danger onClick={() => navigate('/files')}>
              返回
            </Button>
          }
        />
      </div>
    )
  }

  return (
    <div style={{ padding: '24px' }}>
      {/* 页面标题 */}
      <div style={{ marginBottom: '24px' }}>
        <Space>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/files')}
          >
            返回
          </Button>
          <h2 style={{ margin: 0 }}>视频分割 - {videoFile.file_name}</h2>
        </Space>
      </div>

      {/* 视频播放器 */}
      <Card title="视频预览" style={{ marginBottom: '24px' }}>
        <div style={{ position: 'relative', backgroundColor: '#000', borderRadius: '8px', overflow: 'hidden' }}>
          <video
            ref={videoRef}
            src={videoUrl}
            style={{ width: '100%', maxHeight: '500px', display: 'block' }}
            preload="metadata"
            onLoadedMetadata={() => {
              const video = videoRef.current
              if (video) {
                setDuration(video.duration)
              }
            }}
            onTimeUpdate={() => {
              const video = videoRef.current
              if (video) setCurrentTime(video.currentTime)
            }}
            onPlay={() => setIsPlaying(true)}
            onPause={() => setIsPlaying(false)}
          />

          {/* 播放控制条 */}
          <div
            style={{
              position: 'absolute',
              bottom: 0,
              left: 0,
              right: 0,
              background: 'linear-gradient(transparent, rgba(0,0,0,0.8))',
              padding: '12px 16px',
              display: 'flex',
              alignItems: 'center',
              gap: '16px',
            }}
          >
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
                title={`快退${SKIP_SECONDS}秒`}
                style={{ color: '#fff' }}
              />
              <Button
                type="text"
                icon={<StepForwardOutlined />}
                onClick={() => handleSkip(SKIP_SECONDS)}
                title={`快进${SKIP_SECONDS}秒`}
                style={{ color: '#fff' }}
              />
            </Space>

            <span style={{ color: '#fff', marginLeft: 'auto' }}>
              {formatTime(currentTime)} / {formatTime(duration)}
            </span>
          </div>
        </div>
      </Card>

      {/* 时间线与标记 */}
      <Card title="添加分割标记" style={{ marginBottom: '24px' }}>
        <TimelineWithMarkers
          duration={duration}
          markers={markers}
          currentTime={currentTime}
          onMarkerAdd={handleMarkerAdd}
          onMarkerRemove={handleMarkerRemove}
          onSeek={handleSeek}
        />

        {/* 警告提示 */}
        <Alert
          type="warning"
          message="快速分割模式可能有±2秒误差"
          description="FFmpeg 快速分割模式基于关键帧定位，实际分割点可能与标记点略有偏差。如需精确分割，请在分割后使用重新编码模式。"
          showIcon
          style={{ marginTop: 16 }}
        />
      </Card>

      {/* 段落预览 */}
      <Card
        title={
          <Space>
            <span>段落预览</span>
            {markers.length > 0 && (
              <Tag color="blue">将生成 {markers.length + 1} 个段落</Tag>
            )}
          </Space>
        }
        style={{ marginBottom: '24px' }}
      >
        {segmentPreviews.length > 0 ? (
          <Table
            columns={columns}
            dataSource={segmentPreviews}
            rowKey="index"
            pagination={false}
            size="small"
          />
        ) : (
          <Alert
            type="info"
            message="暂无分割标记"
            description={'点击视频时间线添加分割点，或输入时间精确定位。添加标记后点击"确认分割"开始处理。'}
            showIcon
          />
        )}
      </Card>

      {/* 操作按钮 */}
      <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px' }}>
        <Button onClick={() => navigate('/files')}>
          取消
        </Button>
        <Button
          type="primary"
          onClick={handleSplit}
          disabled={markers.length < 2 || splitting}
          loading={splitting}
        >
          {splitting ? splitProgress : '确认分割'}
        </Button>
      </div>
    </div>
  )
}
