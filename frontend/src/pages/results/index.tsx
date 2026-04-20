// PPT 结果详情页

import { useState, useEffect, useCallback, useRef } from 'react'
import {
  Button,
  Card,
  Descriptions,
  Space,
  message,
  Popconfirm,
  Divider,
  Dropdown,
} from 'antd'
import {
  ArrowLeftOutlined,
  DownloadOutlined,
  RedoOutlined,
  MergeCellsOutlined,
  DeleteOutlined,
  CloudOutlined,
  LaptopOutlined,
  ScanOutlined,
  CameraOutlined,
  VideoCameraOutlined,
  DragOutlined,
} from '@ant-design/icons'
import { useNavigate, useParams } from 'react-router-dom'
import dayjs from 'dayjs'
import PPTPreview from '../../components/PPTPreview'
import PPTGalleryStrip from '../../components/PPTGalleryStrip'
import MergeSelectionBar from '../../components/MergeSelectionBar'
import TextContentTab from '../../components/TextContentTab'
import DuplicateDetectionPanel from '../../components/DuplicateDetectionPanel'
import SlideCapturePanel, { DirectCaptureButton } from '../../components/SlideCapturePanel'
import { VideoPreviewPanel } from '../../components/VideoPreviewPanel'
import SlideThumbnail from '../../components/SlideThumbnail'
import { PPTResultsDropdown } from '../../components/PPTResultsDropdown'
import {
  getPptsByVideo,
  getSlides,
  mergeSlides,
  deletePpt,
  getPptDownloadUrl,
  reorderSlides,
} from '../../api/ppt'
import {
  submitTranscriptionWithMode,
} from '../../api/transcription'
import type {
  PPTResult,
  SlideImage,
  SelectedSlide,
  MergeSlideItem,
} from '../../types/ppt'
import type { TranscriptionMode } from '../../types/transcription'
import TranscriptionProgressModal from '../../components/TranscriptionProgressModal'

// 格式化文件大小
const formatFileSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return '0 MB'
  return `${(bytes / 1024 / 1024).toFixed(2)} MB`
}

// Side-by-side preview layout styles
const previewAreaStyle: React.CSSProperties = {
  // CSS class 'ppt-preview-grid' handles grid layout and responsive breakpoint
}

const previewBoxStyle: React.CSSProperties = {
  position: 'relative',
  width: '100%',
  aspectRatio: '16 / 9',  // Maintain 16:9 aspect ratio
  backgroundColor: '#000',
  borderRadius: '8px',
  overflow: 'hidden',
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

  // Ref to track cleanup function for polling
  const slidesPollCleanupRef = useRef<(() => void) | null>(null)

  // Ref for thumbnail container (for auto-scroll to current slide)
  const thumbnailContainerRef = useRef<HTMLDivElement>(null)

  // Video ref for direct capture access
  const videoRef = useRef<HTMLVideoElement>(null)

  // Re-transcribe modal state
  const [retranscribeModalOpen, setRetranscribeModalOpen] = useState(false)
  const [retranscribeMode, setRetranscribeMode] = useState<TranscriptionMode>('local')

  // Duplicate detection panel state
  const [duplicateDetectionOpen, setDuplicateDetectionOpen] = useState(false)

  // Slide capture panel state
  const [isCapturePanelOpen, setIsCapturePanelOpen] = useState(false)

  // Video preview panel state
  const [isVideoPanelVisible, setIsVideoPanelVisible] = useState(true)

  // Drag reorder mode state
  const [isDragMode, setIsDragMode] = useState(false)
  const [draggedSlide, setDraggedSlide] = useState<number | null>(null)

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
          (a: PPTResult, b: PPTResult) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
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
            // 轮询等待提取完成 - 使用取消标志
            let cancelled = false
            let intervalId: NodeJS.Timeout | null = null

            const poll = async () => {
              if (cancelled) return

              try {
                const pollResponse = await getSlides(pptId)
                if (cancelled) return

                if (pollResponse.data && pollResponse.data.status === 'ready') {
                  if (intervalId) clearInterval(intervalId)
                  if (!cancelled) {
                    setSlides(pollResponse.data.slides)
                    setIsLoadingSlides(false)
                  }
                }
              } catch (error) {
                if (!cancelled) {
                  console.error('Polling error:', error)
                }
              }
            }

            intervalId = setInterval(poll, 2000)

            // 存储清理函数到 ref
            return () => {
              cancelled = true
              if (intervalId) clearInterval(intervalId)
            }
          } else {
            setSlides(response.data.slides)
            setIsLoadingSlides(false)
          }
        }
      } catch (error) {
        message.error(error instanceof Error ? error.message : '加载幻灯片失败')
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
    // 清理之前的轮询
    if (slidesPollCleanupRef.current) {
      slidesPollCleanupRef.current()
      slidesPollCleanupRef.current = null
    }

    if (currentPptId <= 0) return

    setCurrentSlide(0)
    let cancelled = false
    let intervalId: NodeJS.Timeout | null = null

    const poll = async () => {
      if (cancelled) return
      try {
        const response = await getSlides(currentPptId)
        if (cancelled) return

        if (response.data?.status === 'ready') {
          if (intervalId) clearInterval(intervalId)
          if (!cancelled) {
            setSlides(response.data.slides)
            setIsLoadingSlides(false)
          }
        }
      } catch (error) {
        if (!cancelled) {
          console.error('Polling error:', error)
        }
      }
    }

    // Initial check
    poll().then(() => {
      // If still extracting, start polling
      if (!cancelled && isLoadingSlides) {
        intervalId = setInterval(poll, 2000)
      }
    })

    return () => {
      cancelled = true
      if (intervalId) clearInterval(intervalId)
    }
  }, [currentPptId])

  // 切换幻灯片
  const handleSlideChange = useCallback((index: number) => {
    setCurrentSlide(index)
  }, [])

  // 合并模式：切换选择
  const handleToggleSelect = useCallback(
    (slide: SlideImage, _index: number) => {
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
  const handleRetranscribeWithMode = useCallback(async (mode: TranscriptionMode) => {
    setRetranscribeMode(mode)
    try {
      if (mode === 'cloud') {
        // Per D-03: cloud mode starts immediately, no sampling_rate
        await submitTranscriptionWithMode(videoFileIdNum, 'cloud')
      } else {
        await submitTranscriptionWithMode(videoFileIdNum, 'local', 0.5)
      }
      setRetranscribeModalOpen(true)
    } catch (err) {
      message.error(err instanceof Error ? err.message : '提交转录任务失败')
    }
  }, [videoFileIdNum])

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

  // 处理幻灯片删除后的回调
  const handleSlidesDeleted = useCallback(async () => {
    // 刷新 PPT 列表以获取更新的页数
    await loadPpts()
    // 重新加载当前 PPT 的幻灯片
    if (currentPptId > 0) {
      await loadSlides(currentPptId)
    }
  }, [loadPpts, loadSlides, currentPptId])

  // 处理幻灯片插入后的回调
  const handleSlideInserted = useCallback(async (newSlideNumber: number) => {
    message.success(`幻灯片已插入到位置 ${newSlideNumber}`)
    // 刷新 PPT 列表以获取更新的页数
    await loadPpts()
    // 重新加载当前 PPT 的幻灯片
    if (currentPptId > 0) {
      await loadSlides(currentPptId)
    }
    // 更新当前幻灯片到新插入的幻灯片
    setCurrentSlide(newSlideNumber - 1) // Convert to 0-based index
  }, [loadPpts, loadSlides, currentPptId])

  // Handle direct slide capture (no modal)
  const handleDirectCapture = useCallback(async (newSlideNumber: number) => {
    // Refresh PPT list to get updated page count
    await loadPpts()
    // Reload current PPT slides
    if (currentPptId > 0) {
      await loadSlides(currentPptId)
    }
    // Update current slide to newly inserted slide
    setCurrentSlide(newSlideNumber - 1) // Convert to 0-based index
  }, [loadPpts, loadSlides, currentPptId])

  // 处理视频->幻灯片同步回调（反向同步）
  const handleVideoSlideChange = useCallback((slideNumber: number) => {
    // VideoPreviewPanel uses 1-based slide numbers
    // Convert to 0-based index for PPTPreview
    const index = slideNumber - 1
    if (index >= 0 && index < slides.length) {
      setCurrentSlide(index)
    }
  }, [slides.length])

  // 处理 PPT 结果切换
  const handlePptChange = useCallback(async (pptId: number) => {
    setCurrentPptId(pptId)
    setCurrentSlide(0)
    await loadSlides(pptId)
  }, [loadSlides])

  // 拖拽排序处理函数
  const handleDragStart = useCallback((slideNumber: number) => {
    setDraggedSlide(slideNumber)
  }, [])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault() // 允许放置
  }, [])

  const handleDrop = useCallback(async (e: React.DragEvent, targetSlideNumber: number) => {
    e.preventDefault()
    if (draggedSlide === null || draggedSlide === targetSlideNumber) {
      setDraggedSlide(null)
      return
    }

    // 创建新的幻灯片顺序
    const newSlideOrder = slides.map(s => s.slide_number)
    const draggedIndex = newSlideOrder.indexOf(draggedSlide)
    const targetIndex = newSlideOrder.indexOf(targetSlideNumber)

    // 移除被拖拽的幻灯片
    newSlideOrder.splice(draggedIndex, 1)
    // 在目标位置插入
    newSlideOrder.splice(targetIndex, 0, draggedSlide)

    try {
      const response = await reorderSlides(currentPptId, newSlideOrder)
      if (response.data?.success) {
        message.success('幻灯片顺序已更新')
        // 重新加载幻灯片
        await loadSlides(currentPptId)
      } else {
        message.error('更新幻灯片顺序失败')
      }
    } catch (error) {
      message.error('更新幻灯片顺序失败: ' + (error as Error).message)
    } finally {
      setDraggedSlide(null)
    }
  }, [draggedSlide, slides, currentPptId, loadSlides])

  const handleDragEnd = useCallback(() => {
    setDraggedSlide(null)
  }, [])

  // 自动滚动到当前幻灯片缩略图
  useEffect(() => {
    if (thumbnailContainerRef.current && currentSlide >= 0) {
      const container = thumbnailContainerRef.current
      const thumbnailList = container.children[0] as HTMLElement
      if (thumbnailList && thumbnailList.children[currentSlide]) {
        const currentThumbnail = thumbnailList.children[currentSlide] as HTMLElement
        currentThumbnail.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
      }
    }
  }, [currentSlide])

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
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <Button
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate('/files')}
          >
            返回
          </Button>
          <h2 style={{ margin: 0 }}>PPT结果 - {videoName}</h2>
        </div>
        {ppts.length > 1 && (
          <PPTResultsDropdown
            ppts={ppts}
            currentPptId={currentPptId}
            onPptChange={handlePptChange}
          />
        )}
      </div>

      {/* Preview Area with Side-by-Side Layout */}
      <div className="ppt-preview-grid" style={previewAreaStyle}>
        {/* Left: Thumbnail Sidebar (160px) */}
        <div
          ref={thumbnailContainerRef}
          className="thumbnail-sidebar"
          style={{
            overflowY: 'auto',
            borderRight: '1px solid #f0f0f0',
            padding: 8,
            background: '#fafafa',
            maxHeight: 'calc(100vh - 200px)',
            scrollBehavior: 'smooth',
          }}
        >
          <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
            {slides.map((slide, idx) => (
              <SlideThumbnail
                key={slide.slide_number}
                slide={slide}
                slideNumber={slide.slide_number}
                totalSlides={slides.length}
                isCurrent={idx === currentSlide}
                isSelected={
                  isMergeMode &&
                  selectedSlides.some((s) => s.id === `${currentPptId}_${slide.slide_number}`)
                }
                isSelectable={isMergeMode}
                isDraggable={isDragMode}
                isDragging={draggedSlide === slide.slide_number}
                onDragStart={isDragMode ? () => handleDragStart(slide.slide_number) : undefined}
                onDragOver={isDragMode ? handleDragOver : undefined}
                onDrop={isDragMode ? (e, num) => handleDrop(e, num) : undefined}
                onDragEnd={isDragMode ? handleDragEnd : undefined}
                onClick={() =>
                  isMergeMode ? handleToggleSelect(slide, idx) : handleSlideChange(idx)
                }
              />
            ))}
          </div>
        </div>

        {/* Center: PPT Preview (16:9) */}
        <div style={previewBoxStyle}>
          <PPTPreview
            slides={slides}
            currentSlide={currentSlide}
            onSlideChange={handleSlideChange}
            isMergeMode={isMergeMode}
            selectedSlides={selectedSlides}
            onToggleSelect={handleToggleSelect}
            isLoading={isLoadingSlides}
            currentPptId={currentPptId}
            containerStyle={{ display: 'flex', flexDirection: 'column', height: '100%' }}
            hideThumbnailSidebar={true}
          />
        </div>

        {/* Right: Video Preview (16:9) */}
        {isVideoPanelVisible && (
          <div style={previewBoxStyle}>
            <VideoPreviewPanel
              videoRef={videoRef}
              videoFileId={videoFileIdNum}
              currentSlide={currentSlide + 1}
              onSlideClick={handleVideoSlideChange}
              style={{ height: '100%', border: 'none', boxShadow: 'none' }}
              autoPlay={false}
              showControls={true}
            />
          </div>
        )}
      </div>

      {/* Info & Operations Bar - Inline layout without tabs */}
      <Card size="small" style={{ marginBottom: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          {/* Basic Info - displayed inline without tabs */}
          <Descriptions column={2} size="small">
            <Descriptions.Item label="视频名称">{videoName}</Descriptions.Item>
            <Descriptions.Item label="转录时间">
              {currentPpt ? dayjs(currentPpt.created_at).format('YYYY-MM-DD HH:mm') : '—'}
            </Descriptions.Item>
            <Descriptions.Item label="页数">{currentPpt?.page_count || 0} 页</Descriptions.Item>
            <Descriptions.Item label="文件大小">{formatFileSize(currentPpt?.file_size || 0)}</Descriptions.Item>
            <Descriptions.Item label="类型">
              {currentPpt?.source_type === 'merge' ? '合并' : '转录'}
            </Descriptions.Item>
          </Descriptions>

          <Divider style={{ margin: '8px 0' }} />

          {/* Operation buttons - horizontal layout with wrapping */}
          <Space wrap>
            <Button
              icon={<DownloadOutlined />}
              onClick={handleDownloadPpt}
            >
              下载PPT
            </Button>
            <Dropdown
              menu={{
                items: [
                  {
                    key: 'local',
                    icon: <LaptopOutlined />,
                    label: '本地转录',
                    onClick: () => handleRetranscribeWithMode('local'),
                  },
                  {
                    key: 'cloud',
                    icon: <CloudOutlined />,
                    label: '云端转录（通义听悟）',
                    onClick: () => handleRetranscribeWithMode('cloud'),
                  },
                ],
              }}
              trigger={['click']}
            >
              <Button icon={<RedoOutlined />}>
                重新转录
              </Button>
            </Dropdown>
            <Button
              type="primary"
              icon={<MergeCellsOutlined />}
              onClick={() => setIsMergeMode(!isMergeMode)}
            >
              {isMergeMode ? '取消合并' : '合并幻灯片'}
            </Button>
            <Button
              icon={<VideoCameraOutlined />}
              onClick={() => setIsVideoPanelVisible(!isVideoPanelVisible)}
            >
              {isVideoPanelVisible ? '隐藏视频预览' : '显示视频预览'}
            </Button>
            <Button
              icon={<DragOutlined />}
              onClick={() => setIsDragMode(!isDragMode)}
              type={isDragMode ? 'primary' : 'default'}
            >
              {isDragMode ? '完成排序' : '拖拽排序'}
            </Button>
            <Button
              icon={<ScanOutlined />}
              onClick={() => setDuplicateDetectionOpen(true)}
            >
              检测重复幻灯片
            </Button>

            {/* Direct capture button - replaces old modal-only capture */}
            <DirectCaptureButton
              pptFileId={currentPptId}
              currentSlide={currentSlide}
              onCaptureComplete={handleDirectCapture}
              videoRef={videoRef}
              disabled={!isVideoPanelVisible || slides.length === 0}
            />

            {/* Advanced capture with preview modal */}
            <Button
              icon={<CameraOutlined />}
              onClick={() => setIsCapturePanelOpen(true)}
            >
              高级捕获（带预览）
            </Button>
            <Popconfirm
              title="确定要删除此PPT文件吗？删除后无法恢复。"
              onConfirm={handleDeletePpt}
              okText="确定"
              cancelText="取消"
            >
              <Button danger icon={<DeleteOutlined />}>
                删除PPT
              </Button>
            </Popconfirm>
          </Space>

          <Divider style={{ margin: '8px 0' }} />

          {/* Text Content - displayed inline without tabs */}
          <TextContentTab videoFileId={videoFileIdNum} />
        </Space>
      </Card>

      {/* Merge Selection Bar - Below Info Bar */}
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

      {/* Multiple PPT Results Gallery */}
      {ppts.length > 1 && (
        <Card title="多个转录结果" size="small">
          <PPTGalleryStrip
            ppts={ppts}
            currentPptId={currentPptId}
            onSelect={setCurrentPptId}
          />
        </Card>
      )}

      {/* 重新转录模态框 */}
      <TranscriptionProgressModal
        open={retranscribeModalOpen}
        onClose={() => setRetranscribeModalOpen(false)}
        videoFileId={videoFileIdNum}
        fileName={videoName}
        samplingRate={0.5}
        mode={retranscribeMode}
        onCompleted={handleTranscriptionCompleted}
      />

      {/* 重复幻灯片检测面板 */}
      <DuplicateDetectionPanel
        pptFileId={currentPptId}
        visible={duplicateDetectionOpen}
        onClose={() => setDuplicateDetectionOpen(false)}
        onSlidesDeleted={handleSlidesDeleted}
      />

      {/* 捕获幻灯片面板 */}
      <SlideCapturePanel
        pptFileId={currentPptId}
        videoFileId={videoFileIdNum}
        currentSlide={currentSlide + 1} // Convert to 1-based
        totalSlides={slides.length}
        onSlideInserted={handleSlideInserted}
        onCancel={() => setIsCapturePanelOpen(false)}
        open={isCapturePanelOpen}
      />
    </div>
  )
}
