import { useState, useCallback, useRef } from 'react'
import { Modal, Upload, Progress, message, List, Tag } from 'antd'
import {
  InboxOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
} from '@ant-design/icons'
import type { UploadProps } from 'antd'
import { uploadVideoFile, scanVideoFiles, type FileUploadResult } from '../api/video-file'
import * as authApi from '../api/auth'

const { Dragger } = Upload

interface VideoUploadModalProps {
  visible: boolean
  onCancel: () => void
  onUploadSuccess: () => void
}

// 验证上传结果 - 后端返回的是 FileUploadResult 类型
const validateUploadResult = (data: unknown): data is FileUploadResult => {
  const result = data as FileUploadResult
  return !!(result?.file_id && result?.file_name && result?.file_path)
}

// Video file formats allowed for upload
const ACCEPTED_VIDEO_FORMATS = ['.mp4', '.mkv', '.avi', '.mov']
// Maximum file size: 5GB (matches backend defaultMaxFileSize)
const MAX_FILE_SIZE = 5 * 1024 * 1024 * 1024

// 上传状态
type UploadStatus = 'pending' | 'uploading' | 'success' | 'error'

interface UploadTask {
  file: File
  status: UploadStatus
  progress: number
  error?: string
}

export default function VideoUploadModal({
  visible,
  onCancel,
  onUploadSuccess,
}: VideoUploadModalProps) {
  const [uploadTasks, setUploadTasks] = useState<UploadTask[]>([])
  const [isUploading, setIsUploading] = useState(false)
  const abortedRef = useRef(false)

  const handleCancel = useCallback(() => {
    if (isUploading) {
      abortedRef.current = true
    }
    setUploadTasks([])
    setIsUploading(false)
    abortedRef.current = false
    onCancel()
  }, [isUploading, onCancel])

  // 验证文件
  const validateFile = useCallback((file: File): string | null => {
    if (file.size > MAX_FILE_SIZE) {
      return `文件大小超过限制 (最大 5GB)`
    }
    const hasValidExtension = ACCEPTED_VIDEO_FORMATS.some((ext) =>
      file.name.toLowerCase().endsWith(ext)
    )
    if (!hasValidExtension) {
      return `不支持的文件格式，仅支持: ${ACCEPTED_VIDEO_FORMATS.join(', ')}`
    }
    return null
  }, [])

  // 处理文件选择
  const handleFileSelect = useCallback((files: File[]) => {
    const tasks: UploadTask[] = files.map((file) => ({
      file,
      status: 'pending',
      progress: 0,
    }))
    setUploadTasks(tasks)
  }, [])

  // 开始批量上传
  const startUpload = useCallback(async () => {
    if (uploadTasks.length === 0) return

    setIsUploading(true)
    abortedRef.current = false
    let successCount = 0
    let failCount = 0

    for (let i = 0; i < uploadTasks.length; i++) {
      if (abortedRef.current) {
        message.warning('上传已取消')
        break
      }

      const task = uploadTasks[i]
      const validationError = validateFile(task.file)
      if (validationError) {
        setUploadTasks((prev) =>
          prev.map((t, idx) => (idx === i ? { ...t, status: 'error', error: validationError } : t))
        )
        failCount++
        continue
      }

      // 更新状态为上传中
      setUploadTasks((prev) =>
        prev.map((t, idx) => (idx === i ? { ...t, status: 'uploading' } : t))
      )

      try {
        const result = await uploadVideoFile(task.file, (percent) => {
          setUploadTasks((prev) =>
            prev.map((t, idx) => (idx === i ? { ...t, progress: percent } : t))
          )
        })

        if (result.data && validateUploadResult(result.data)) {
          setUploadTasks((prev) =>
            prev.map((t, idx) => (idx === i ? { ...t, status: 'success', progress: 100 } : t))
          )
          successCount++
        } else {
          throw new Error('服务器返回的数据格式无效')
        }
      } catch (error) {
        setUploadTasks((prev) =>
          prev.map((t, idx) =>
            idx === i
              ? {
                  ...t,
                  status: 'error',
                  error: error instanceof Error ? error.message : '上传失败',
                }
              : t
          )
        )
        failCount++
      }
    }

    setIsUploading(false)

    // 上传完成后刷新 token
    try {
      await authApi.getCurrentUser()
    } catch (refreshError) {
      console.warn('Token refresh after upload failed:', refreshError)
    }

    // 扫描导入文件到视频管理系统
    if (successCount > 0) {
      try {
        message.loading('正在导入文件...', 0)
        const scanResult = await scanVideoFiles()
        message.destroy() // 关闭 loading

        if (scanResult.data) {
          const { created, skipped } = scanResult.data
          if (created > 0) {
            message.success(
              `上传完成！已导入 ${created} 个新文件${skipped > 0 ? `，跳过 ${skipped} 个已存在文件` : ''}`
            )
          } else if (skipped > 0) {
            message.info(`上传完成！文件已存在，跳过 ${skipped} 个`)
          } else {
            message.warning('上传完成，但文件未被识别')
          }
        }
      } catch (scanError) {
        message.destroy()
        console.error('扫描文件失败:', scanError)
        message.warning('文件已上传，但导入到列表失败，请点击"扫描导入"手动同步')
      }
    }

    // 显示结果
    if (abortedRef.current) {
      if (successCount > 0) {
        message.info(`上传已取消: 成功 ${successCount} 个, 失败 ${failCount} 个`)
      }
    } else if (successCount === 0) {
      message.error(`全部上传失败: ${failCount} 个文件`)
    } else if (failCount > 0) {
      message.warning(`上传完成: 成功 ${successCount} 个, 失败 ${failCount} 个`)
    }

    // 触发回调刷新列表
    if (successCount > 0) {
      onUploadSuccess()
    }

    // 延迟关闭模态框
    setTimeout(() => {
      handleCancel()
    }, 1500)
  }, [uploadTasks, validateFile, onUploadSuccess, handleCancel])

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: true,
    fileList: [],
    beforeUpload: (_file, fileList) => {
      handleFileSelect(fileList as File[])
      return false // Prevent auto upload
    },
    disabled: isUploading,
    accept: ACCEPTED_VIDEO_FORMATS.join(','),
  }

  // 计算总体进度
  const totalProgress =
    uploadTasks.length > 0
      ? uploadTasks.reduce((sum, task) => sum + task.progress, 0) / uploadTasks.length
      : 0

  const completedCount = uploadTasks.filter(
    (t) => t.status === 'success' || t.status === 'error'
  ).length

  return (
    <Modal
      title="批量上传视频"
      open={visible}
      onCancel={handleCancel}
      okText={uploadTasks.length > 0 ? '开始上传' : '关闭'}
      onOk={uploadTasks.length > 0 && !isUploading ? startUpload : undefined}
      cancelButtonProps={{ disabled: isUploading }}
      okButtonProps={{ disabled: isUploading }}
      width={700}
      closable={!isUploading}
      maskClosable={!isUploading}
    >
      <div style={{ padding: '20px 0' }}>
        {uploadTasks.length === 0 ? (
          <Dragger {...uploadProps}>
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">点击或拖拽视频文件到此区域上传</p>
            <p className="ant-upload-hint">
              支持格式: MP4, MKV, AVI, MOV | 最大文件大小: 5GB | 支持多文件选择
            </p>
          </Dragger>
        ) : (
          <>
            {isUploading && (
              <div style={{ marginBottom: 16 }}>
                <Progress
                  percent={Math.round(totalProgress)}
                  status={abortedRef.current ? 'exception' : 'active'}
                  format={(percent) => `${completedCount}/${uploadTasks.length} 完成 (${percent}%)`}
                />
              </div>
            )}

            <List
              size="small"
              dataSource={uploadTasks}
              renderItem={(task) => {
                const statusIcon = {
                  pending: <LoadingOutlined />,
                  uploading: <LoadingOutlined />,
                  success: <CheckCircleOutlined style={{ color: '#52c41a' }} />,
                  error: <CloseCircleOutlined style={{ color: '#ff4d4f' }} />,
                }[task.status]

                const statusColor = {
                  pending: 'default',
                  uploading: 'processing',
                  success: 'success',
                  error: 'error',
                }[task.status]

                const statusText = {
                  pending: '等待中',
                  uploading: `上传中 ${task.status === 'uploading' ? `(${Math.round(task.progress)}%)` : ''}`,
                  success: '上传成功',
                  error: task.error || '上传失败',
                }[task.status]

                return (
                  <List.Item>
                    <List.Item.Meta
                      avatar={statusIcon}
                      title={
                        <div
                          style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            alignItems: 'center',
                          }}
                        >
                          <span
                            style={{
                              flex: 1,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              whiteSpace: 'nowrap',
                            }}
                          >
                            {task.file.name}
                          </span>
                          <Tag color={statusColor}>{statusText}</Tag>
                        </div>
                      }
                      description={
                        <div>
                          <span>{(task.file.size / 1024 / 1024).toFixed(2)} MB</span>
                          {task.status === 'uploading' && (
                            <Progress
                              percent={Math.round(task.progress)}
                              size="small"
                              status="active"
                              style={{ marginTop: 4, width: 200 }}
                            />
                          )}
                        </div>
                      }
                    />
                  </List.Item>
                )
              }}
            />

            {!isUploading && (
              <div style={{ marginTop: 16, textAlign: 'center' }}>
                <a onClick={() => setUploadTasks([])}>重新选择文件</a>
              </div>
            )}
          </>
        )}
      </div>
    </Modal>
  )
}
