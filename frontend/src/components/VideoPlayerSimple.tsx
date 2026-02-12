// 视频文件播放器组件（简化版用于表格内嵌套）

import { useState } from 'react'
import { Button } from 'antd'
import { PlayCircleOutlined } from '@ant-design/icons'
import type { VideoFile } from '../types/video-file'
import { VideoPlayerModal } from './VideoPlayerModal'

// 用于在表格中渲染的包装组件
export function RenderVideoPreview(file: VideoFile) {
  const [visible, setVisible] = useState(false)

  return (
    <>
      <Button
        type="link"
        size="small"
        icon={<PlayCircleOutlined />}
        onClick={() => setVisible(true)}
        disabled={file.status !== 'ready'}
        title={file.status === 'ready' ? '播放视频' : '仅就绪状态可播放'}
      >
        播放
      </Button>
      <VideoPlayerModal file={file} visible={visible} onClose={() => setVisible(false)} />
    </>
  )
}
