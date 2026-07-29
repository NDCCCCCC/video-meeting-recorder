/**
 * Keyboard shortcuts hook for video player controls
 * Implements industry-standard shortcuts (YouTube/VLC patterns)
 */

import { useEffect, useCallback } from 'react'
import { message } from 'antd'

// ==================== Constants ====================
/**
 * Available playback speeds for video player
 */
const PLAYBACK_SPEEDS = [0.5, 1, 1.25, 1.5, 2]

// ==================== Hook Interface ====================

interface UseKeyboardShortcutsProps {
  videoRef: React.RefObject<HTMLVideoElement | null>
  isPlaying: boolean
  playbackRate: number
  volume: number
  enabled: boolean
  onPlayPause: () => void
  onSeek: (seconds: number) => void
  onVolumeChange: (volume: number) => void
  onPlaybackRateChange: (rate: number) => void
  onFullscreen: () => void
  onMuteToggle: () => void
}

// ==================== Hook Implementation ====================

export function useKeyboardShortcuts({
  videoRef,
  isPlaying,
  playbackRate,
  volume,
  enabled,
  onPlayPause,
  onSeek,
  onVolumeChange,
  onPlaybackRateChange,
  onFullscreen,
  onMuteToggle,
}: UseKeyboardShortcutsProps) {
  const handleKeyDown = useCallback(
    (event: KeyboardEvent) => {
      if (!enabled) return

      // Ignore if typing in input, textarea, or select elements
      const target = event.target as HTMLElement
      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      ) {
        return
      }

      const key = event.key.toLowerCase()
      const shift = event.shiftKey

      switch (true) {
        case key === ' ':
          event.preventDefault()
          onPlayPause()
          message.info(isPlaying ? '暂停' : '播放')
          break

        case key === 'arrowleft':
          event.preventDefault()
          onSeek(shift ? -1 : -10)
          message.info(shift ? '快退 1 秒' : '快退 10 秒')
          break

        case key === 'arrowright':
          event.preventDefault()
          onSeek(shift ? 1 : 10)
          message.info(shift ? '快进 1 秒' : '快进 10 秒')
          break

        case key === 'arrowup': {
          event.preventDefault()
          const newVolumeUp = Math.min(1, volume + 0.1)
          onVolumeChange(newVolumeUp)
          message.info(`音量: ${Math.round(newVolumeUp * 100)}%`)
          break
        }

        case key === 'arrowdown': {
          event.preventDefault()
          const newVolumeDown = Math.max(0, volume - 0.1)
          onVolumeChange(newVolumeDown)
          message.info(`音量: ${Math.round(newVolumeDown * 100)}%`)
          break
        }

        case key === 'j':
          event.preventDefault()
          onSeek(-10)
          message.info('快退 10 秒')
          break

        case key === 'l':
          event.preventDefault()
          onSeek(10)
          message.info('快进 10 秒')
          break

        case key === 'k':
          event.preventDefault()
          onPlayPause()
          message.info(isPlaying ? '暂停' : '播放')
          break

        case key === 'm':
          event.preventDefault()
          onMuteToggle()
          message.info(volume > 0 ? '静音' : '取消静音')
          break

        case key === 'f':
          event.preventDefault()
          onFullscreen()
          message.info('全屏')
          break

        case key === '>':
        case key === '.':
          if (shift) {
            event.preventDefault()
            const currentIndex = PLAYBACK_SPEEDS.indexOf(playbackRate)
            const nextSpeed = PLAYBACK_SPEEDS[(currentIndex + 1) % PLAYBACK_SPEEDS.length]
            onPlaybackRateChange(nextSpeed)
            message.info(`播放速度: ${nextSpeed}x`)
          }
          break

        case key === '<':
        case key === ',':
          if (shift) {
            event.preventDefault()
            const currentIndex = PLAYBACK_SPEEDS.indexOf(playbackRate)
            const prevSpeed =
              PLAYBACK_SPEEDS[(currentIndex - 1 + PLAYBACK_SPEEDS.length) % PLAYBACK_SPEEDS.length]
            onPlaybackRateChange(prevSpeed)
            message.info(`播放速度: ${prevSpeed}x`)
          }
          break

        case key === 'home': {
          event.preventDefault()
          const video = videoRef.current
          if (video) {
            video.currentTime = 0
            message.info('跳转到开始')
          }
          break
        }

        case key === 'end': {
          event.preventDefault()
          const videoEnd = videoRef.current
          if (videoEnd) {
            videoEnd.currentTime = videoEnd.duration
            message.info('跳转到结束')
          }
          break
        }

        case key >= '0' && key <= '9': {
          event.preventDefault()
          const percentage = parseInt(key) / 10
          const videoSeek = videoRef.current
          if (videoSeek) {
            videoSeek.currentTime = videoSeek.duration * percentage
            message.info(`跳转到 ${Math.round(percentage * 100)}%`)
          }
          break
        }
      }
    },
    [
      enabled,
      isPlaying,
      playbackRate,
      volume,
      onPlayPause,
      onSeek,
      onVolumeChange,
      onPlaybackRateChange,
      onFullscreen,
      onMuteToggle,
      videoRef,
    ]
  )

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])
}
