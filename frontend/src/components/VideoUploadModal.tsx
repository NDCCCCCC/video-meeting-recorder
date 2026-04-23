import { useState, useCallback } from 'react'
import { Modal, Upload, Progress, message } from 'antd'
import { InboxOutlined } from '@ant-design/icons'
import type { UploadProps } from 'antd'
import { uploadVideoFile } from '../api/video-file'
import type { VideoFile } from '../types/video-file'

const { Dragger } = Upload

interface VideoUploadModalProps {
  visible: boolean
  onCancel: () => void
  onUploadSuccess: (file: VideoFile) => void
}

// Video file formats allowed for upload
const ACCEPTED_VIDEO_FORMATS = ['.mp4', '.mkv', '.avi', '.mov']
// MIME types for validation
const VIDEO_MIME_TYPES = [
  'video/mp4',
  'video/x-matroska',
  'video/x-msvideo',
  'video/quicktime',
]
// Maximum file size: 5GB (matches backend defaultMaxFileSize)
const MAX_FILE_SIZE = 5 * 1024 * 1024 * 1024

export default function VideoUploadModal({
  visible,
  onCancel,
  onUploadSuccess,
}: VideoUploadModalProps) {
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [fileList, setFileList] = useState<any[]>([])

  const handleUpload = useCallback(
    async (file: File) => {
      // Validate file size
      if (file.size > MAX_FILE_SIZE) {
        message.error(`文件大小超过限制 (最大 5GB)`)
        return false
      }

      // Validate file type by extension
      const hasValidExtension = ACCEPTED_VIDEO_FORMATS.some((ext) =>
        file.name.toLowerCase().endsWith(ext)
      )
      if (!hasValidExtension) {
        message.error(`不支持的文件格式，仅支持: ${ACCEPTED_VIDEO_FORMATS.join(', ')}`)
        return false
      }

      // Validate file type by MIME type
      if (!VIDEO_MIME_TYPES.includes(file.type)) {
        message.error('文件类型验证失败，请上传有效的视频文件')
        return false
      }

      setUploading(true)
      setProgress(0)

      try {
        const result = await uploadVideoFile(file, (percent) => {
          setProgress(percent)
        })

        if (result.data) {
          message.success(`${file.name} 上传成功`)
          setFileList([])
          setProgress(0)
          onUploadSuccess(result.data as VideoFile)
        }
      } catch (error) {
        message.error(error instanceof Error ? error.message : '上传失败')
        setFileList([])
      } finally {
        setUploading(false)
      }

      return false // Prevent auto upload
    },
    [onUploadSuccess]
  )

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: false,
    fileList,
    beforeUpload: handleUpload,
    onChange: (info) => {
      setFileList(info.fileList)
    },
    onRemove: () => {
      setFileList([])
      setProgress(0)
    },
    disabled: uploading,
    maxCount: 1,
    accept: ACCEPTED_VIDEO_FORMATS.join(','),
  }

  return (
    <Modal
      title="上传视频"
      open={visible}
      onCancel={onCancel}
      okButtonProps={{ style: { display: 'none' } }}
      cancelButtonProps={{ disabled: uploading }}
      cancelText="关闭"
      width={600}
    >
      <div style={{ padding: '20px 0' }}>
        <Dragger {...uploadProps}>
          <p className="ant-upload-drag-icon">
            <InboxOutlined />
          </p>
          <p className="ant-upload-text">点击或拖拽视频文件到此区域上传</p>
          <p className="ant-upload-hint">
            支持格式: MP4, MKV, AVI, MOV | 最大文件大小: 5GB
          </p>
        </Dragger>

        {uploading && (
          <div style={{ marginTop: 20 }}>
            <Progress percent={Math.round(progress)} status="active" />
            <p style={{ textAlign: 'center', marginTop: 8, color: '#666' }}>
              正在上传...
            </p>
          </div>
        )}
      </div>
    </Modal>
  )
}
