// 带标记的时间线组件

import { Slider, Input, Tag, Space, Tooltip } from 'antd'
import { CloseOutlined } from '@ant-design/icons'
import { useMemo, useCallback } from 'react'
import type { SliderSingleProps } from 'antd'

// ==================== 类型定义 ====================

export interface TimelineWithMarkersProps {
  duration: number           // 视频总时长（秒）
  markers: number[]          // 标记时间戳数组（秒）
  currentTime: number        // 当前播放位置
  onMarkerAdd: (time: number) => void
  onMarkerRemove: (time: number) => void
  onSeek: (time: number) => void
}

// ==================== 工具函数 ====================

// 格式化时间（秒 -> MM:SS）
function formatTime(seconds: number): string {
  if (!seconds || !Number.isFinite(seconds)) return '0:00'
  const s = Math.floor(seconds)
  const minutes = Math.floor(s / 60)
  const secs = s % 60
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

// 解析时间字符串（MM:SS 或秒数）为秒数
function parseTimeInput(input: string): number | null {
  if (!input || !input.trim()) return null

  // 尝试解析为 MM:SS 格式
  const mmssMatch = input.match(/^(\d+):(\d+)$/)
  if (mmssMatch) {
    const minutes = parseInt(mmssMatch[1], 10)
    const seconds = parseInt(mmssMatch[2], 10)
    if (minutes >= 0 && seconds >= 0 && seconds < 60) {
      return minutes * 60 + seconds
    }
    return null
  }

  // 尝试解析为纯秒数
  const seconds = parseFloat(input)
  if (!isNaN(seconds) && seconds >= 0) {
    return seconds
  }

  return null
}

// ==================== 主组件 ====================

export function TimelineWithMarkers({
  duration,
  markers,
  currentTime,
  onMarkerAdd,
  onMarkerRemove,
  onSeek,
}: TimelineWithMarkersProps) {
  // ==================== 计算标记点 ====================

  const marks = useMemo(() => {
    const result: Record<number, { style: React.CSSProperties; label: string }> = {}
    markers.forEach(marker => {
      result[marker] = {
        style: {
          color: '#1890ff',
          fontWeight: 'bold',
        },
        label: formatTime(marker),
      }
    })
    return result
  }, [markers])

  // ==================== 验证逻辑 ====================

  const validateMarker = useCallback((time: number): string | null => {
    if (time <= 0 || time >= duration) {
      return `时间必须在 0 到 ${formatTime(duration)} 之间`
    }
    if (markers.some(m => Math.abs(m - time) < 2)) {
      return '标记点间距必须至少 2 秒'
    }
    if (markers.length >= 20) {
      return '最多只能添加 20 个标记点'
    }
    return null
  }, [duration, markers])

  // ==================== 事件处理 ====================

  const handleSliderChange: SliderSingleProps['onChange'] = useCallback((value: number) => {
    onSeek(value)
  }, [onSeek])

  const handleSliderClick = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
    // 只处理轨道区域的点击（不处理标记点点击）
    const target = e.target as HTMLElement
    if (target.closest('.ant-slider-mark-text') || target.closest('.ant-slider-handle')) {
      return
    }

    const sliderTrack = target.closest('.ant-slider')
    if (!sliderTrack) return

    const rect = sliderTrack.getBoundingClientRect()
    const clickX = e.clientX - rect.left
    const percentage = clickX / rect.width
    const time = percentage * duration

    const error = validateMarker(time)
    if (error) {
      // 静默失败，不显示错误（用户可能是误点击）
      return
    }

    onMarkerAdd(Math.round(time * 10) / 10) // 保留一位小数
  }, [duration, validateMarker, onMarkerAdd])

  const handleManualAdd = useCallback((value: string) => {
    const time = parseTimeInput(value)
    if (time === null) {
      // 静默失败，输入会在下次聚焦时清除
      return
    }

    const error = validateMarker(time)
    if (error) {
      // TODO: 显示错误提示（需要在父组件处理）
      return
    }

    onMarkerAdd(Math.round(time * 10) / 10) // 保留一位小数
  }, [validateMarker, onMarkerAdd])

  const handleMarkerRemove = useCallback((marker: number) => {
    onMarkerRemove(marker)
  }, [onMarkerRemove])

  // ==================== 渲染 ====================

  return (
    <div style={{ width: '100%' }}>
      {/* 时间线滑块 */}
      <div onClickCapture={handleSliderClick} style={{ marginBottom: 16 }}>
        <Slider
          min={0}
          max={duration}
          value={currentTime}
          onChange={handleSliderChange}
          marks={marks}
          tooltip={{ formatter: (val) => formatTime(val as number) }}
          trackStyle={{ backgroundColor: '#1890ff' }}
          handleStyle={{ borderColor: '#1890ff', backgroundColor: '#1890ff' }}
          disabled={!duration}
        />
      </div>

      {/* 手动输入和标记列表 */}
      <Space direction="vertical" style={{ width: '100%' }} size="small">
        {/* 手动输入框 */}
        <Input.Search
          placeholder="输入时间点 (MM:SS 或秒数)"
          enterButton="添加标记"
          onSearch={handleManualAdd}
          style={{ width: '100%' }}
          disabled={!duration}
        />

        {/* 标记列表 */}
        {markers.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
            {markers.map((marker, index) => (
              <Tooltip key={marker} title={`标记 ${index + 1}: ${formatTime(marker)}`}>
                <Tag
                  closable
                  onClose={() => handleMarkerRemove(marker)}
                  closeIcon={<CloseOutlined style={{ fontSize: 10 }} />}
                  color="blue"
                  style={{ marginBottom: 4 }}
                >
                  段落 {index + 1}: {formatTime(marker)}
                </Tag>
              </Tooltip>
            ))}
          </div>
        )}

        {/* 空状态提示 */}
        {markers.length === 0 && (
          <div style={{ color: '#999', fontSize: 14, textAlign: 'center', padding: '8px 0' }}>
            暂无标记点，点击时间线或输入时间添加
          </div>
        )}
      </Space>
    </div>
  )
}
