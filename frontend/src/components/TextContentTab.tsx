// Text Content Tab - Displays transcription text with clickable timestamps and copy functionality
// Per D-09, D-10, D-11

import { useState, useEffect } from 'react'
import { Button, Space, Spin, Empty, message, Tooltip } from 'antd'
import { CopyOutlined, CheckOutlined } from '@ant-design/icons'
import { getTranscriptionText } from '../api/transcription'
import type { TextSegment } from '../types/transcription'

interface TextContentTabProps {
  videoFileId: number
  onTimestampClick?: (timestampMs: number) => void
}

export default function TextContentTab({ videoFileId, onTimestampClick }: TextContentTabProps) {
  const [segments, setSegments] = useState<TextSegment[]>([])
  const [loading, setLoading] = useState(true)
  const [copiedIndex, setCopiedIndex] = useState<number | null>(null)
  const [copiedAll, setCopiedAll] = useState(false)

  useEffect(() => {
    const fetchText = async () => {
      setLoading(true)
      try {
        const response = await getTranscriptionText(videoFileId)
        if (response.data && response.data.segments) {
          setSegments(response.data.segments)
        }
      } catch {
        // Silently fail -- text content may not exist for local transcription
        setSegments([])
      } finally {
        setLoading(false)
      }
    }

    fetchText()
  }, [videoFileId])

  const formatTimestamp = (milliseconds: number): string => {
    const totalSeconds = Math.floor(milliseconds / 1000)
    const hours = Math.floor(totalSeconds / 3600)
    const minutes = Math.floor((totalSeconds % 3600) / 60)
    const seconds = totalSeconds % 60
    return `[${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}]`
  }

  const handleCopyAll = async () => {
    const fullText = segments
      .map((s) => `${formatTimestamp(s.begin_time)} ${s.text}`)
      .join('\n')
    try {
      await navigator.clipboard.writeText(fullText)
      setCopiedAll(true)
      message.success('已复制全部文字')
      setTimeout(() => setCopiedAll(false), 2000)
    } catch {
      message.error('复制失败')
    }
  }

  const handleCopySegment = async (index: number) => {
    const segment = segments[index]
    const text = `${formatTimestamp(segment.begin_time)} ${segment.text}`
    try {
      await navigator.clipboard.writeText(text)
      setCopiedIndex(index)
      setTimeout(() => setCopiedIndex(null), 2000)
    } catch {
      message.error('复制失败')
    }
  }

  const handleTimestampClick = (timestampMs: number) => {
    if (onTimestampClick) {
      onTimestampClick(timestampMs / 1000) // Convert to seconds
    }
  }

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '40px 0' }}>
        <Spin />
      </div>
    )
  }

  if (segments.length === 0) {
    return (
      <Empty
        description="暂无文字内容"
        image={Empty.PRESENTED_IMAGE_SIMPLE}
      />
    )
  }

  return (
    <div>
      {/* Copy all button per D-11 */}
      <div style={{ marginBottom: 16, textAlign: 'right' }}>
        <Button
          icon={copiedAll ? <CheckOutlined /> : <CopyOutlined />}
          onClick={handleCopyAll}
          size="small"
        >
          {copiedAll ? '已复制' : '复制全部文字'}
        </Button>
      </div>

      {/* Scrollable text content */}
      <div style={{ maxHeight: '500px', overflowY: 'auto' }}>
        {segments.map((segment, index) => (
          <div
            key={segment.segment_index ?? index}
            style={{
              marginBottom: 12,
              padding: 12,
              background: '#f5f5f5',
              borderRadius: 4,
            }}
          >
            <Space size={8} align="start">
              {/* Per-segment copy icon per D-11 */}
              <Tooltip title={copiedIndex === index ? '已复制' : '复制'}>
                <Button
                  type="text"
                  size="small"
                  icon={copiedIndex === index ? <CheckOutlined /> : <CopyOutlined />}
                  onClick={() => handleCopySegment(index)}
                  style={{ minWidth: 24, padding: 0 }}
                />
              </Tooltip>

              {/* Clickable timestamp per D-10 */}
              <span
                role="button"
                tabIndex={0}
                onClick={() => handleTimestampClick(segment.begin_time)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    handleTimestampClick(segment.begin_time)
                  }
                }}
                style={{
                  fontFamily: 'monospace',
                  color: '#1890ff',
                  cursor: onTimestampClick ? 'pointer' : 'default',
                  whiteSpace: 'nowrap',
                  userSelect: 'none',
                }}
              >
                {formatTimestamp(segment.begin_time)}
              </span>

              {/* Text content */}
              <span style={{ lineHeight: 1.6 }}>{segment.text}</span>
            </Space>
          </div>
        ))}
      </div>
    </div>
  )
}
