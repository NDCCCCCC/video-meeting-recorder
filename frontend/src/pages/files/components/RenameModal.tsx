// 重命名文件对话框（展示型）
// 从 pages/files/index.tsx 抽出。state（newFileName/renameLoading）与提交逻辑
// 留在父级，本组件只接收 props。

import { Modal, Input, Space } from 'antd'
import type { VideoFile } from '../../../types/video-file'

export interface RenameModalProps {
  open: boolean
  file: VideoFile | null
  newFileName: string
  onNewFileNameChange: (value: string) => void
  loading: boolean
  onClose: () => void
  onOk: () => void
}

export function RenameModal({
  open,
  file,
  newFileName,
  onNewFileNameChange,
  loading,
  onClose,
  onOk,
}: RenameModalProps) {
  return (
    <Modal
      title="重命名文件"
      open={open}
      onOk={onOk}
      onCancel={onClose}
      confirmLoading={loading}
      okButtonProps={{
        disabled:
          !newFileName.trim() || newFileName.trim() === file?.file_name.replace(/\.[^/.]+$/, ''),
      }}
    >
      <Space orientation="vertical" style={{ width: '100%' }}>
        <Input
          value={newFileName}
          onChange={(e) => onNewFileNameChange(e.target.value)}
          placeholder="新文件名（不含扩展名）"
          maxLength={200}
          autoFocus
          onPressEnter={onOk}
        />
        {file && (
          <div style={{ color: '#888', fontSize: 12 }}>
            文件扩展名.{file.file_name.split('.').pop()} 将自动添加
          </div>
        )}
      </Space>
    </Modal>
  )
}
