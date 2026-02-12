// 视频文件播放器组件（已废弃，请使用 VideoPlayerModal）

import { VideoPlayerModal } from './VideoPlayerModal'
import type { VideoFile } from '../types/video-file'

interface VideoPlayerProps {
  file: VideoFile
  visible: boolean
  onClose: () => void
}

// 导出统一的播放器组件以保持向后兼容
export function VideoPlayer(props: VideoPlayerProps) {
  return <VideoPlayerModal {...props} />
}

// 导出渲染组件
export { RenderVideoPreview } from './VideoPlayerSimple'
