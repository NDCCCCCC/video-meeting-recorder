import { InputNumber, Space } from 'antd'
import { useState } from 'react'

interface EditableProgressBarProps {
  currentTime: number
  duration: number
  onSeek: (time: number) => void
}

// Format seconds to MM:SS or HH:MM:SS
function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '0:00'

  const s = Math.floor(seconds)
  const hours = Math.floor(s / 3600)
  const minutes = Math.floor((s % 3600) / 60)
  const secs = s % 60

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }
  return `${minutes}:${secs.toString().padStart(2, '0')}`
}

// Parse MM:SS or HH:MM:SS to seconds
function parseTimeToSeconds(timeStr: string): number {
  const parts = timeStr.split(':').map(Number)
  if (parts.length === 3) {
    return parts[0] * 3600 + parts[1] * 60 + parts[2]
  } else if (parts.length === 2) {
    return parts[0] * 60 + parts[1]
  }
  return parts[0] || 0
}

export default function EditableProgressBar({ currentTime, duration, onSeek }: EditableProgressBarProps) {
  const [inputTime, setInputTime] = useState(currentTime)

  const handleTimeChange = (value: number | null) => {
    if (value === null || value < 0 || value > duration) return
    setInputTime(value)
    onSeek(value)
  }

  return (
    <Space style={{ width: '100%', display: 'flex', justifyContent: 'space-between' }}>
      {/* Current time input */}
      <InputNumber
        value={Math.round(currentTime)}
        onChange={handleTimeChange}
        min={0}
        max={duration}
        formatter={(value) => formatTime(value || 0)}
        parser={(value) => parseTimeToSeconds(value || '')}
        style={{ width: 100 }}
        size="small"
      />

      {/* Progress bar */}
      <input
        type="range"
        min={0}
        max={duration}
        value={currentTime}
        onChange={(e) => onSeek(Number(e.target.value))}
        style={{ flex: 1, cursor: 'pointer' }}
      />

      {/* Duration display */}
      <span style={{ color: '#fff', fontSize: '12px', minWidth: 50 }}>
        {formatTime(duration)}
      </span>
    </Space>
  )
}
