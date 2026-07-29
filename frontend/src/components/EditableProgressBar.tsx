import { useState, useEffect } from 'react'
import { Input, Button, Space } from 'antd'

interface EditableProgressBarProps {
  currentTime: number
  duration: number
  onSeek: (time: number) => void
}

// Format seconds to HH:MM:SS (always show hours)
function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '00:00:00'

  const s = Math.floor(seconds)
  const hours = Math.floor(s / 3600)
  const minutes = Math.floor((s % 3600) / 60)
  const secs = s % 60

  return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
}

// Parse HH:MM:SS to seconds
function parseTimeToSeconds(timeStr: string): number {
  const parts = timeStr.split(':').map(Number)
  if (parts.length === 3) {
    return parts[0] * 3600 + parts[1] * 60 + parts[2]
  } else if (parts.length === 2) {
    return parts[0] * 60 + parts[1]
  }
  return parts[0] || 0
}

export default function EditableProgressBar({
  currentTime,
  duration,
  onSeek,
}: EditableProgressBarProps) {
  const [inputValue, setInputValue] = useState(formatTime(currentTime))

  // Update input when currentTime changes (from video playback)
  useEffect(() => {
    setInputValue(formatTime(currentTime))
  }, [currentTime])

  const handleJump = () => {
    const seconds = parseTimeToSeconds(inputValue)
    if (seconds >= 0 && seconds <= duration) {
      onSeek(seconds)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleJump()
    }
  }

  return (
    <Space.Compact style={{ width: '100%', display: 'flex' }}>
      {/* Current time input - always show HH:MM:SS */}
      <Input
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="00:00:00"
        style={{ width: 100, fontFamily: 'monospace', textAlign: 'center' }}
        size="small"
      />

      {/* Jump button */}
      <Button type="primary" size="small" onClick={handleJump}>
        跳转
      </Button>

      {/* Progress bar - wider */}
      <input
        type="range"
        min={0}
        max={duration}
        value={currentTime}
        onChange={(e) => onSeek(Number(e.target.value))}
        style={{ flex: 1, cursor: 'pointer', margin: '0 8px' }}
      />

      {/* Duration display - always show HH:MM:SS */}
      <span
        style={{
          color: '#fff',
          fontSize: '13px',
          minWidth: 80,
          textAlign: 'right',
          fontFamily: 'monospace',
        }}
      >
        {formatTime(duration)}
      </span>
    </Space.Compact>
  )
}
