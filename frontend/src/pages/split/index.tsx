// 视频分割页面

import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Button, Space, Table, Modal, message, Alert, Spin, Tag } from 'antd'
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
import type { VideoFile } from '../../types/video-file'
import styles from './SplitPage.module.css'

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
  const [existingSplits, setExistingSplits] = useState<VideoFile[]>([])
  const [checkingSplits, setCheckingSplits] = useState(false)

  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  // video_playback_token：5min TTL，异步获取
  const [videoUrl, setVideoUrl] = useState<string | undefined>(undefined)

  const videoRef = useRef<HTMLVideoElement>(null)

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
        const fileId = parseInt(id, 10)
        const response = await videoFileApi.getVideoFile(fileId)
        if (response.data) {
          setVideoFile(response.data)
          setDuration(response.data.duration)
        }
        // 串行拉取播放 URL（依赖登录态 + 文件存在）
        try {
          const urlRes = await videoFileApi.getVideoPlaybackUrl(fileId)
          if (urlRes.data?.playback_url) {
            setVideoUrl(urlRes.data.playback_url)
          }
        } catch (urlErr) {
          setError(urlErr instanceof Error ? urlErr.message : '获取播放链接失败')
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

  const handleSkip = useCallback(
    (seconds: number) => {
      const video = videoRef.current
      if (!video || !duration) return
      video.currentTime = Math.max(0, Math.min(duration, video.currentTime + seconds))
    },
    [duration]
  )

  const handleSeek = useCallback((time: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = time
    setCurrentTime(time)
  }, [])

  // ==================== 标记管理 ====================

  const handleMarkerAdd = useCallback((time: number) => {
    setMarkers((prev) => {
      const newMarkers = [...prev, time].sort((a, b) => a - b)
      return newMarkers
    })
  }, [])

  const handleMarkerRemove = useCallback((time: number) => {
    setMarkers((prev) => prev.filter((m) => m !== time))
  }, [])

  // ==================== 分割执行 ====================

  const handleSplit = useCallback(async () => {
    if (!id || markers.length < 1) {
      message.warning('请至少添加 1 个标记点')
      return
    }

    // Check for existing splits before showing confirmation
    setCheckingSplits(true)
    try {
      const response = await videoFileApi.getVideoSegments(parseInt(id, 10))
      setExistingSplits(response.data || [])
    } catch (error) {
      console.error('Failed to check existing splits:', error)
      setExistingSplits([])
    } finally {
      setCheckingSplits(false)
    }

    // Show confirmation modal
    Modal.confirm({
      title: existingSplits.length > 0 ? '重新分割视频' : '确认分割',
      content:
        existingSplits.length > 0 ? (
          <div>
            <Alert
              title="检测到现有分割"
              description={`此视频已有 ${existingSplits.length} 个分割段，重新分割将自动删除这些文件。`}
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />
            <p>将被删除的文件：</p>
            <ul style={{ maxHeight: 200, overflow: 'auto', paddingLeft: 20 }}>
              {existingSplits.map((split) => (
                <li key={split.id}>
                  {split.file_name} ({(split.file_size / 1024 / 1024).toFixed(2)} MB)
                </li>
              ))}
            </ul>
            <p style={{ color: '#ff4d4f', fontWeight: 'bold' }}>
              ⚠️ 此操作不可撤销，确认要继续吗？
            </p>
          </div>
        ) : (
          `确认将视频分割为 ${markers.length + 1} 个段落？`
        ),
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
  }, [id, markers, navigate, existingSplits])

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
      <div className={styles.loadingContainer}>
        <Spin size="large" />
        <div style={{ marginTop: 12, color: '#999' }}>加载中...</div>
      </div>
    )
  }

  if (error || !videoFile) {
    return (
      <div className={styles.errorContainer}>
        <Alert
          type="error"
          title="加载失败"
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
    <div className={styles.pageContainer}>
      {/* 页面标题 */}
      <div className={styles.headerSection}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/files')}>
            返回
          </Button>
          <h2 style={{ margin: 0 }}>视频分割 - {videoFile.file_name}</h2>
        </Space>
      </div>

      {/* 左右布局：左侧视频 + 右侧操作 */}
      <div className={styles.mainLayout}>
        {/* 左侧：视频播放器 */}
        <div className={styles.videoSection}>
          <Card
            title="视频预览"
            className={styles.videoCard}
            styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', padding: 0 } }}
          >
            <div className={styles.videoContainer}>
              <video
                ref={videoRef}
                src={videoUrl}
                className={styles.videoElement}
                preload="metadata"
                onLoadedMetadata={() => {
                  const video = videoRef.current
                  if (video) {
                    setDuration(video.duration)
                  }
                }}
                onError={() => {
                  // token 过期或服务端 403 时尝试重新获取一次
                  if (!id) return
                  const video = videoRef.current
                  const restoreTime = video ? video.currentTime : 0
                  setError('视频加载失败，正在重试...')
                  videoFileApi
                    .getVideoPlaybackUrl(parseInt(id, 10))
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
                }}
                onTimeUpdate={() => {
                  const video = videoRef.current
                  if (video) setCurrentTime(video.currentTime)
                }}
                onPlay={() => setIsPlaying(true)}
                onPause={() => setIsPlaying(false)}
              />

              {/* 进度条 */}
              <div
                className={styles.progressBar}
                onClick={(e) => {
                  const rect = e.currentTarget.getBoundingClientRect()
                  const ratio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width))
                  if (duration) handleSeek(ratio * duration)
                }}
              >
                {/* 已播放进度 */}
                <div
                  className={styles.progressBarFill}
                  style={{ width: duration ? `${(currentTime / duration) * 100}%` : '0%' }}
                />
                {/* 标记点 */}
                {markers.map((m) => (
                  <div
                    key={m}
                    className={styles.marker}
                    style={{ left: duration ? `${(m / duration) * 100}%` : 0 }}
                  />
                ))}
              </div>

              {/* 播放控制条 */}
              <div className={styles.controlsBar}>
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

                <span className={styles.timeDisplay}>
                  {formatTime(currentTime)} / {formatTime(duration)}
                </span>
              </div>
            </div>
          </Card>
        </div>

        {/* 右侧：时间线、段落预览、操作 */}
        <div className={styles.sidebar}>
          {/* 时间线与标记 */}
          <Card title="添加分割标记" className={styles.card}>
            <TimelineWithMarkers
              duration={duration}
              markers={markers}
              currentTime={currentTime}
              onMarkerAdd={handleMarkerAdd}
              onMarkerRemove={handleMarkerRemove}
              onSeek={handleSeek}
            />

            <Alert
              type="warning"
              title="快速分割模式可能有±2秒误差"
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
                {markers.length > 0 && <Tag color="blue">将生成 {markers.length + 1} 个段落</Tag>}
              </Space>
            }
            className={styles.segmentCard}
            styles={{ body: { flex: 1, overflow: 'auto' } }}
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
                title="暂无分割标记"
                description={
                  '点击视频时间线添加分割点，或输入时间精确定位。添加标记后点击"确认分割"开始处理。'
                }
                showIcon
              />
            )}
          </Card>

          {/* 操作按钮 */}
          <div className={styles.actionButtons}>
            <Button onClick={() => navigate('/files')}>取消</Button>
            <Button
              type="primary"
              onClick={handleSplit}
              disabled={markers.length < 1 || splitting || checkingSplits}
              loading={splitting || checkingSplits}
            >
              {checkingSplits ? '检查中...' : splitting ? splitProgress : '确认分割'}
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}
