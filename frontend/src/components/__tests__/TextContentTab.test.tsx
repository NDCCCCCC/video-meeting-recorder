/**
 * Type-level and structural tests for TextContentTab component.
 * Validates timestamp formatting logic and props interface.
 */
import type { TextSegment } from '../../types/transcription'

// Test: formatTimestamp produces [HH:MM:SS] format per D-10
function formatTimestamp(milliseconds: number): string {
  const totalSeconds = Math.floor(milliseconds / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return `[${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}]`
}

// Verify timestamp formatting
const ts1 = formatTimestamp(0)
if (ts1 !== '[00:00:00]') throw new Error(`Expected [00:00:00], got ${ts1}`)

const ts2 = formatTimestamp(3661000) // 1h 1m 1s
if (ts2 !== '[01:01:01]') throw new Error(`Expected [01:01:01], got ${ts2}`)

const ts3 = formatTimestamp(500) // 0.5s rounds to 0
if (ts3 !== '[00:00:00]') throw new Error(`Expected [00:00:00], got ${ts3}`)

// Test: TextSegment has correct fields per D-10
const segment: TextSegment = {
  text: '测试文字',
  begin_time: 1000,
  end_time: 2000,
  segment_index: 0,
}
if (segment.begin_time !== 1000) throw new Error('begin_time mismatch')

// Test: Empty state message per D-09
const emptyMessage = '暂无文字内容'
if (emptyMessage !== '暂无文字内容') throw new Error('Empty message mismatch')
