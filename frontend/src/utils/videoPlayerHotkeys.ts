/**
 * Video player keyboard shortcut definitions
 * Following industry standards (YouTube/VLC patterns)
 */

/**
 * Keyboard shortcut definition with optional modifier keys
 */
export interface KeyboardShortcut {
  readonly key: string
  readonly description: string
  readonly shiftKey?: boolean
  readonly ctrlKey?: boolean
}

export const KEYBOARD_SHORTCUTS: { readonly [key: string]: KeyboardShortcut } = {
  PLAY_PAUSE: { key: ' ', description: '播放/暂停' },
  SEEK_BACK_10: { key: 'ArrowLeft', description: '快退10秒' },
  SEEK_FORWARD_10: { key: 'ArrowRight', description: '快进10秒' },
  VOLUME_UP: { key: 'ArrowUp', description: '音量+10%' },
  VOLUME_DOWN: { key: 'ArrowDown', description: '音量-10%' },
  SEEK_BACK_1: { key: 'j', description: '快退10秒' },
  SEEK_FORWARD_1: { key: 'l', description: '快进10秒' },
  PLAY_PAUSE_ALT: { key: 'k', description: '播放/暂停' },
  MUTE: { key: 'm', description: '静音/取消静音' },
  FULLSCREEN: { key: 'f', description: '全屏' },
  SPEED_UP: { key: '>', shiftKey: true, description: '播放速度+' },
  SPEED_DOWN: { key: '<', shiftKey: true, description: '播放速度-' },
  SEEK_TO_START: { key: 'Home', description: '跳转到开始' },
  SEEK_TO_END: { key: 'End', description: '跳转到结束' },
}

/**
 * Helper function to check if event matches shortcut
 * @param event - Keyboard event to check
 * @param shortcut - Shortcut definition to match against
 * @returns true if event matches shortcut
 */
export function matchesShortcut(event: KeyboardEvent, shortcut: KeyboardShortcut): boolean {
  const keyMatches = event.key.toLowerCase() === shortcut.key.toLowerCase()
  const shiftMatches = shortcut.shiftKey === undefined || shortcut.shiftKey === event.shiftKey
  const ctrlMatches = shortcut.ctrlKey === undefined || shortcut.ctrlKey === event.ctrlKey
  return keyMatches && shiftMatches && ctrlMatches
}
