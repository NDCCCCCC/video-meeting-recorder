// PPT 结果详情页

import { useState, useEffect, useCallback } from 'react'
import {
  Button,
  Card,
  Row,
  Col,
  Descriptions,
  Space,
  message,
  Popconfirm,
  Modal,
} from 'antd'
import {
  ArrowLeftOutlined,
  DownloadOutlined,
  RedoOutlined,
  MergeCellsOutlined,
  DeleteOutlined,
} from '@ant-design/icons'
import { useNavigate, useParams } from 'react-router-dom'
import dayjs from 'dayjs'
import PPTPreview from '../../components/PPTPreview'
import PPTGalleryStrip from '../../components/PPTGalleryStrip'
import MergeSelectionBar from '../../components/MergeSelectionBar'
import {
  getPptsByVideo,
  getSlides,
  mergeSlides,
  deletePpt,
  getPptDownloadUrl,
} from '../../api/ppt'
import type {
  PPTResult,
  SlideImage,
  SelectedSlide,
  MergeSlideItem,
} from '../../types/ppt'
import TranscriptionProgressModal from '../../components/TranscriptionProgressModal'
import { submitTranscription } from '../../api/transcription'

// 格式化文件大小
const formatFileSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 MB'
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

export default function ResultDetailPage() {
  const navigate = useNavigate()
  const { videoFileId } = useParams<{ videoFileId: string }>()
  const videoFileIdNum = parseInt(videoFileId || '0', 10)

  // State
  const [ppts, setPpts] = useState<PPTResult[]>([])
  const [currentPptId, setCurrentPptId] = useState<number>(0)
  const [slides, setSlides] = useState<SlideImage[]>([])
  const [currentSlide, setCurrentSlide] = useState(0)
  const [isLoadingSlides, setIsLoadingSlides] = useState(false)
  const [isMergeMode, setIsMergeMode] = useState(false)
  const [selectedSlides, setSelectedSlides] = useState<SelectedSlide[]>([])
  const [isMerging, setIsMerging] = useState(false)
  const [videoName, setVideoName] = useState('')
  const [loading, setLoading] = useState(false)

  // Re-transcribe modal state
  const [retranscribeModalOpen, setRetranscribeModalOpen] = useState(false)

  // 当前选中的 PPT
  const currentPpt = ppts.find((p) => p.id === currentPptId)

  // 加载 PPT 列表
  const loadPpts = useCallback(async () => {
    if (!videoFileIdNum) return

    setLoading(true)
    try {
      const response = await getPptsByVideo(videoFileIdNum)
      if (response.data && response.data.ppts) {
        const sortedPpts = response.data.ppts.sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        )
        setPpts(sortedPpts)
        if (sortedPpts.length > 0) {
          setCurrentPptId(sortedPpts[0].id)
        }
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载 PPT 列表失败')
    } finally {
      setLoading(false)
    }
  }, [videoFileIdNum])

  // 加载幻灯片
  const loadSlides = useCallback(
    async (pptId: number) => {
      setIsLoadingSlides(true)
      try {
        const response = await getSlides(pptId)
        if (response.data) {
          if (response.data.status === 'extracting') {
            // 轮询等待提取完成
            const pollInterval = setInterval(async () => {
              const pollResponse = await getSlides(pptId)
              if (pollResponse.data && pollResponse.data.status === 'ready') {
                clearInterval(pollInterval)
                setSlides(pollResponse.data.slides)
                setIsLoadingSlides(false)
              }
            }, 2000)

            // 清理定时器
            return () => clearInterval(pollInterval)
          } else {
            setSlides(response.data.slides)
          }
        }
      } catch (error) {
        message.error(error instanceof Error ? error.message : '加载幻灯片失败')
      } finally {
        setIsLoadingSlides(false)
      }
    },
    []
  )

  // 加载视频名称（从 PPT 文件名推断或使用 ID）
  const loadVideoName = useCallback(async () => {
    // 简单处理：使用 videoFileId 作为名称
    // 实际应用中可能需要调用 API 获取视频文件信息
    setVideoName(`视频 ${videoFileIdNum}`)
  }, [videoFileIdNum])

  // 初始加载
  useEffect(() => {
    loadPpts()
    loadVideoName()
  }, [loadPpts, loadVideoName])

  // 当 currentPptId 变化时加载幻灯片
  useEffect(() => {
    if (currentPptId > 0) {
      loadSlides(currentPptId)
      setCurrentSlide(0)
    }
  }, [currentPptId, loadSlides])

  // 切换幻灯片
  const handleSlideChange = useCallback((index: number) => {
    setCurrentSlide(index)
  }, [])

  // 合并模式：切换选择
  const handleToggleSelect = useCallback(
    (slide: SlideImage, index: number) => {
      const slideId = `${currentPptId}_${slide.slide_number}`
      const existing = selectedSlides.find((s) => s.id === slideId)

      if (existing) {
        // 取消选择
        setSelectedSlides((prev) => prev.filter((s) => s.id !== slideId))
      } else {
        // 检查 200 页限制
        if (selectedSlides.length >= 200) {
          message.warning('最多只能选择 200 页幻灯片进行合并')
          return
        }

        // 添加选择
        const pptName = currentPpt?.file_name || `PPT ${currentPptId}`
        setSelectedSlides((prev) => [
          ...prev,
          {
            id: slideId,
            ppt_file_id: currentPptId,
            slide_number: slide.slide_number,
            thumbnail_url: slide.thumbnail_url,
            source_name: pptName,
          },
        ])
      }
    },
    [currentPptId, selectedSlides, currentPpt]
  )

  // 合并模式：移除幻灯片
  const handleRemoveSlide = useCallback(
    (slideId: string) => {
      setSelectedSlides((prev) => prev.filter((s) => s.id !== slideId))
    },
    []
  )

  // 合并模式：确认合并
  const handleConfirmMerge = useCallback(async () => {
    if (selectedSlides.length === 0) {
      message.warning('请先选择要合并的幻灯片')
      return
    }

    setIsMerging(true)
    try {
      const mergeItems: MergeSlideItem[] = selectedSlides.map((s) => ({
        ppt_file_id: s.ppt_file_id,
        slide_number: s.slide_number,
      }))

      const response = await mergeSlides({
        slides: mergeItems,
        video_file_id: videoFileIdNum,
      })

      if (response.data) {
        message.success(`合并完成！已生成 ${response.data.page_count} 页 PPT。`)
        // 刷新 PPT 列表
        await loadPpts()
        // 退出合并模式
        setIsMergeMode(false)
        setSelectedSlides([])
        // 切换到新生成的 PPT
        setCurrentPptId(response.data.ppt_file_id)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '合并失败')
    } finally {
      setIsMerging(false)
    }
  }, [selectedSlides, videoFileIdNum, loadPpts])

  // 下载 PPT
  const handleDownloadPpt = useCallback(() => {
    if (!currentPptId) return
    window.open(getPptDownloadUrl(currentPptId), '_blank')
  }, [currentPptId])

  // 重新转录
  const handleRetranscribe = useCallback(() => {
    setRetranscribeModalOpen(true)
  }, [])

  // 转录完成回调
  const handleTranscriptionCompleted = useCallback(
    async (pptFileId: number) => {
      setRetranscribeModalOpen(false)
      // 刷新 PPT 列表
      await loadPpts()
      // 切换到新生成的 PPT
      setCurrentPptId(pptFileId)
      message.success('转录完成')
    },
    [loadPpts]
  )

  // 删除 PPT
  const handleDeletePpt = useCallback(async () => {
    if (!currentPptId) return

    try {
      await deletePpt(currentPptId)
      message.success('PPT 已删除')

      // 从列表中移除
      const newPpts = ppts.filter((p) => p.id !== currentPptId)
      setPpts(newPpts)

      // 如果删除的是当前 PPT，切换到下一个
      if (newPpts.length > 0) {
        setCurrentPptId(newPpts[0].id)
      } else {
        // 没有更多 PPT，返回文件列表
        navigate('/files')
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }, [currentPptId, ppts, navigate])

  if (loading) {
    return <div style={{ padding: 24, textAlign: 'center' }}>加载中...</div>
  }

  if (ppts.length === 0) {
    return (
      <div style={{ padding: 24 }}>
        <div style={{ marginBottom: 16 }}>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/files')}>
            返回
          </Button>
        </div>
        <div
          style={{
            textAlign: 'center',
            padding: '100px 0',
            color: '#8c8c8c',
          }}
        >
          该视频尚未进行转录，或者转录任务还在进行中。
          <br />
          请等待转录完成后查看 PPT 预览。
        </div>
      </div>
    )
  }

  return (
    <div style={{ padding: 24 }}>
      {/* 标题栏 */}
      <div
        style={{
          marginBottom: 16,
          display: 'flex',
          alignItems: 'center',
          gap: 8,
        }}
      >
        <Button
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/files')}
        >
          返回
        </Button>
        <h2 style={{ margin: 0 }}>PPT结果 - {videoName}</h2>
      </div>

      <Row gutter={24}>
        {/* 左列 - 预览区域 (70%) */}
        <Col span={17}>
          <Card bodyStyle={{ padding: 0 }}>
            <PPTPreview
              slides={slides}
              currentSlide={currentSlide}
              onSlideChange={handleSlideChange}
              isMergeMode={isMergeMode}
              selectedSlides={selectedSlides}
              onToggleSelect={handleToggleSelect}
              isLoading={isLoadingSlides}
              currentPptId={currentPptId}
            />
          </Card>

          {/* 合并选择栏 - 仅在合并模式显示 */}
          {isMergeMode && (
            <MergeSelectionBar
              selectedSlides={selectedSlides}
              onReorder={setSelectedSlides}
              onRemove={handleRemoveSlide}
              onConfirm={handleConfirmMerge}
              onCancel={() => {
                setIsMergeMode(false)
                setSelectedSlides([])
              }}
              isMerging={isMerging}
            />
          )}
        </Col>

        {/* 右列 - 信息面板 (30%) */}
        <Col span={7}>
          {/* 基本信息卡片 */}
          <Card title="基本信息" size="small" style={{ marginBottom: 16 }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item label="视频名称">
                {videoName}
              </Descriptions.Item>
              <Descriptions.Item label="转录时间">
                {currentPpt ? dayjs(currentPpt.created_at).format('YYYY-MM-DD HH:mm') : '—'}
              </Descriptions.Item>
              <Descriptions.Item label="页数">
                {currentPpt?.page_count || 0} 页
              </Descriptions.Item>
              <Descriptions.Item label="文件大小">
                {formatFileSize(currentPpt?.file_size || 0)}
              </Descriptions.Item>
              <Descriptions.Item label="类型">
                {currentPpt?.source_type === 'merge' ? '合并' : '转录'}
              </Descriptions.Item>
            </Descriptions>
          </Card>

          {/* 操作按钮卡片 */}
          <Card title="操作" size="small" style={{ marginBottom: 16 }}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Button
                block
                icon={<DownloadOutlined />}
                onClick={handleDownloadPpt}
              >
                下载PPT
              </Button>
              <Button
                block
                icon={<RedoOutlined />}
                onClick={handleRetranscribe}
              >
                重新转录
              </Button>
              <Button
                block
                type="primary"
                icon={<MergeCellsOutlined />}
                onClick={() => setIsMergeMode(!isMergeMode)}
              >
                {isMergeMode ? '取消合并' : '合并幻灯片'}
              </Button>
              <Popconfirm
                title="确定要删除此PPT文件吗？删除后无法恢复。"
                onConfirm={handleDeletePpt}
                okText="确定"
                cancelText="取消"
              >
                <Button block danger icon={<DeleteOutlined />}>
                  删除PPT
                </Button>
              </Popconfirm>
            </Space>
          </Card>

          {/* 多个转录结果 */}
          {ppts.length > 1 && (
            <Card title="多个转录结果" size="small">
              <PPTGalleryStrip
                ppts={ppts}
                currentPptId={currentPptId}
                onSelect={setCurrentPptId}
              />
            </Card>
          )}
        </Col>
      </Row>

      {/* 重新转录模态框 */}
      <TranscriptionProgressModal
        open={retranscribeModalOpen}
        onClose={() => setRetranscribeModalOpen(false)}
        videoFileId={videoFileIdNum}
        fileName={videoName}
        samplingRate={0.5}
        onCompleted={handleTranscriptionCompleted}
      />
    </div>
  )
}
