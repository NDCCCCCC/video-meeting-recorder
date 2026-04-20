import React, { useState, useRef, useEffect } from 'react'
import { Modal, Button, Space, InputNumber, Image, message, Spin, Progress, Select } from 'antd'
import { CameraOutlined, PlayCircleOutlined, PauseCircleOutlined, CheckOutlined } from '@ant-design/icons'
import type { SlideCapturePanelProps } from '../types/ppt'
import { captureFrame, insertSlide } from '../api/ppt'

interface SlideCapturePanelProps {
  pptFileId: number
  videoFileId: number
  currentSlide: number
  totalSlides: number
  onSlideInserted?: (newSlideNumber: number) => void
  onCancel?: () => void
  open?: boolean
}

interface VideoRefState {
  currentTime: number
  duration: number
  isPlaying: boolean
}

const SlideCapturePanel: React.FC<SlideCapturePanelProps> = ({
  pptFileId,
  videoFileId,
  currentSlide,
  totalSlides,
  onSlideInserted,
  onCancel,
  open = false,
}) => {
  const [capturedFrame, setCapturedFrame] = useState<string | null>(null)
  const [isCapturing, setIsCapturing] = useState(false)
  const [isInserting, setIsInserting] = useState(false)
  const [insertPosition, setInsertPosition] = useState<number>(currentSlide + 1)
  const [insertPositionOption, setInsertPositionOption] = useState<string>('after')
  const [videoState, setVideoState] = useState<VideoRefState>({
    currentTime: 0,
    duration: 0,
    isPlaying: false,
  })
  const videoRef = useRef<HTMLVideoElement>(null)

  // Update insert position when current slide changes
  useEffect(() => {
    if (insertPositionOption === 'after') {
      setInsertPosition(currentSlide + 1)
    } else if (insertPositionOption === 'before') {
      setInsertPosition(currentSlide)
    } else if (insertPositionOption === 'end') {
      setInsertPosition(totalSlides + 1)
    }
  }, [currentSlide, totalSlides, insertPositionOption])

  // Handle insert position option change
  const handleInsertPositionOptionChange = (value: string) => {
    setInsertPositionOption(value)
    if (value === 'after') {
      setInsertPosition(currentSlide + 1)
    } else if (value === 'before') {
      setInsertPosition(currentSlide)
    } else if (value === 'end') {
      setInsertPosition(totalSlides + 1)
    } else if (value === 'custom') {
      // Keep current insertPosition value
    }
  }

  // Handle video time update
  const handleTimeUpdate = () => {
    if (videoRef.current) {
      setVideoState({
        ...videoState,
        currentTime: videoRef.current.currentTime,
      })
    }
  }

  // Handle video loaded metadata
  const handleLoadedMetadata = () => {
    if (videoRef.current) {
      setVideoState({
        ...videoState,
        duration: videoRef.current.duration,
      })
    }
  }

  // Handle play/pause toggle
  const handleTogglePlay = () => {
    if (videoRef.current) {
      if (videoState.isPlaying) {
        videoRef.current.pause()
      } else {
        videoRef.current.play()
      }
      setVideoState({
        ...videoState,
        isPlaying: !videoState.isPlaying,
      })
    }
  }

  // Handle frame capture
  const handleCaptureFrame = async () => {
    setIsCapturing(true)
    setCapturedFrame(null)

    try {
      const response = await captureFrame(pptFileId, videoState.currentTime)

      if (response.data.success && response.data.frame_data) {
        setCapturedFrame(response.data.frame_data)
        message.success('帧捕获成功')
      } else {
        message.error('帧捕获失败')
      }
    } catch (error) {
      console.error('Failed to capture frame:', error)
      message.error('帧捕获失败: ' + (error as Error).message)
    } finally {
      setIsCapturing(false)
    }
  }

  // Handle slide insertion
  const handleInsertSlide = async () => {
    if (!capturedFrame) {
      message.warning('请先捕获帧')
      return
    }

    setIsInserting(true)

    try {
      const response = await insertSlide(
        pptFileId,
        capturedFrame,
        insertPosition,
        videoState.currentTime
      )

      if (response.data.success) {
        message.success(`幻灯片插入成功，位置: ${response.data.inserted_slide_number}`)

        // Call callback with new slide number
        if (onSlideInserted) {
          onSlideInserted(response.data.inserted_slide_number)
        }

        // Reset captured frame
        setCapturedFrame(null)

        // Close modal after short delay
        setTimeout(() => {
          if (onCancel) {
            onCancel()
          }
        }, 1000)
      } else {
        message.error('幻灯片插入失败')
      }
    } catch (error) {
      console.error('Failed to insert slide:', error)
      message.error('幻灯片插入失败: ' + (error as Error).message)
    } finally {
      setIsInserting(false)
    }
  }

  // Format time as MM:SS
  const formatTime = (seconds: number): string => {
    const mins = Math.floor(seconds / 60)
    const secs = Math.floor(seconds % 60)
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }

  // Generate insert position options
  const insertPositionOptions = [
    { label: `当前幻灯片之后 (位置 ${currentSlide + 1})`, value: 'after' },
    { label: `当前幻灯片之前 (位置 ${currentSlide})`, value: 'before' },
    { label: `最后 (位置 ${totalSlides + 1})`, value: 'end' },
    { label: '自定义位置', value: 'custom' },
  ]

  return (
    <Modal
      title="捕获并插入幻灯片"
      open={open}
      onCancel={onCancel}
      width={900}
      footer={[
        <Button key="cancel" onClick={onCancel} disabled={isInserting}>
          取消
        </Button>,
        <Button
          key="insert"
          type="primary"
          onClick={handleInsertSlide}
          disabled={!capturedFrame || isInserting}
          loading={isInserting}
          icon={<CheckOutlined />}
        >
          插入幻灯片
        </Button>,
      ]}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* Video Player Section */}
        <div>
          <div style={{ marginBottom: 8, fontWeight: 'bold' }}>视频预览</div>
          <div style={{ position: 'relative', backgroundColor: '#000', borderRadius: 8 }}>
            <video
              ref={videoRef}
              src={`/api/v1/videos/${videoFileId}/stream`}
              style={{ width: '100%', maxHeight: 400 }}
              onTimeUpdate={handleTimeUpdate}
              onLoadedMetadata={handleLoadedMetadata}
            />
            <div
              style={{
                position: 'absolute',
                bottom: 10,
                left: 10,
                color: '#fff',
                backgroundColor: 'rgba(0, 0, 0, 0.7)',
                padding: '4px 8px',
                borderRadius: 4,
                fontFamily: 'monospace',
              }}
            >
              {formatTime(videoState.currentTime)} / {formatTime(videoState.duration)}
            </div>
          </div>

          {/* Video Controls */}
          <Space style={{ marginTop: 12 }}>
            <Button
              icon={videoState.isPlaying ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={handleTogglePlay}
            >
              {videoState.isPlaying ? '暂停' : '播放'}
            </Button>
            <Button
              type="primary"
              icon={<CameraOutlined />}
              onClick={handleCaptureFrame}
              loading={isCapturing}
            >
              捕获当前帧
            </Button>
          </Space>

          {/* Progress Bar */}
          {videoState.duration > 0 && (
            <Progress
              percent={(videoState.currentTime / videoState.duration) * 100}
              showInfo={false}
              strokeColor="#1890ff"
              style={{ marginTop: 8 }}
            />
          )}
        </div>

        {/* Captured Frame Preview */}
        {capturedFrame && (
          <div>
            <div style={{ marginBottom: 8, fontWeight: 'bold' }}>捕获的帧预览</div>
            <Image
              src={capturedFrame}
              alt="Captured frame"
              style={{ width: '100%', maxHeight: 300, objectFit: 'contain' }}
            />
          </div>
        )}

        {/* Insert Position Selection */}
        <div>
          <div style={{ marginBottom: 8, fontWeight: 'bold' }}>插入位置</div>
          <Space direction="vertical" style={{ width: '100%' }}>
            <Select
              value={insertPositionOption}
              onChange={handleInsertPositionOptionChange}
              options={insertPositionOptions}
              style={{ width: '100%' }}
            />
            {insertPositionOption === 'custom' && (
              <InputNumber
                value={insertPosition}
                onChange={(value) => setInsertPosition(value || 1)}
                min={1}
                max={totalSlides + 1}
                style={{ width: '100%' }}
                addonBefore="位置"
              />
            )}
            {insertPositionOption !== 'custom' && (
              <div style={{ color: '#666', fontSize: 12 }}>
                将插入到位置: {insertPosition}
              </div>
            )}
          </Space>
        </div>

        {/* Instructions */}
        <div style={{ backgroundColor: '#f0f2f5', padding: 12, borderRadius: 4 }}>
          <div style={{ fontWeight: 'bold', marginBottom: 4 }}>使用说明:</div>
          <ul style={{ margin: 0, paddingLeft: 20 }}>
            <li>播放视频到想要捕获的帧</li>
            <li>点击"捕获当前帧"按钮</li>
            <li>选择插入位置</li>
            <li>点击"插入幻灯片"完成操作</li>
          </ul>
        </div>
      </Space>
    </Modal>
  )
}

export default SlideCapturePanel
