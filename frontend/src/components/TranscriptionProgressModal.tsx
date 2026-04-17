// 本地转录进度模态框

import { useState, useEffect, useCallback } from 'react'
import { Modal, Progress, message, Button, Space, Alert } from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  DownloadOutlined,
  ReloadOutlined,
  CloudUploadOutlined,
  ClockCircleOutlined,
  CloudDownloadOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons'
import type { TranscriptionTaskStatus, TranscriptionStage, TranscriptionMode, CloudTranscriptionStage } from '../types/transcription'
import { getTranscriptionStatus } from '../api/transcription'

interface TranscriptionProgressModalProps {
  open: boolean
  onClose: () => void
  videoFileId: number
  fileName: string
  samplingRate?: number       // Optional -- only used for local mode
  mode?: TranscriptionMode    // NEW: 'local' | 'cloud', defaults to 'local'
  onCompleted: (pptFileId: number) => void
}

// 阶段显示配置
const STAGE_CONFIG: Record<TranscriptionStage, { label: string; icon: React.ReactNode }> = {
  extracting: { label: '帧提取中', icon: <LoadingOutlined spin /> },
  detecting: { label: '画面检测中', icon: <LoadingOutlined spin /> },
  generating: { label: '生成PPT', icon: <LoadingOutlined spin /> },
}

// Cloud transcription stage config (per D-06)
const CLOUD_STAGE_CONFIG: Record<CloudTranscriptionStage, { label: string; icon: React.ReactNode }> = {
  uploading: { label: '上传中', icon: <CloudUploadOutlined /> },
  queued: { label: '排队中', icon: <ClockCircleOutlined /> },
  cloud_processing: { label: '处理中', icon: <LoadingOutlined spin /> },
  downloading: { label: '下载结果', icon: <CloudDownloadOutlined /> },
}

export default function TranscriptionProgressModal({
  open,
  onClose,
  videoFileId,
  fileName,
  samplingRate,
  mode = 'local',
  onCompleted,
}: TranscriptionProgressModalProps) {
  // 转录状态
  const [status, setStatus] = useState<TranscriptionTaskStatus>('pending')
  const [stage, setStage] = useState<TranscriptionStage | CloudTranscriptionStage | ''>('')
  const [framesProcessed, setFramesProcessed] = useState(0)
  const [totalFrames, setTotalFrames] = useState(0)
  const [percentage, setPercentage] = useState(0)
  const [errorMessage, setErrorMessage] = useState('')
  const [pptFileId, setPptFileId] = useState<number | null>(null)
  const [fallbackToLocal, setFallbackToLocal] = useState(false)

  // Polling interval: 10s for cloud per D-05, 5s for local
  const pollInterval = mode === 'cloud' ? 10000 : 5000

  // 轮询获取转录状态 (per D-16, 5-second interval for local, 10s for cloud per D-05)
  useEffect(() => {
    if (!open) return

    // 立即获取一次状态
    const fetchStatus = async () => {
      try {
        const response = await getTranscriptionStatus(videoFileId)
        if (response.data) {
          const data = response.data
          setStatus(data.status)
          setStage(data.current_stage)
          setFramesProcessed(data.frames_processed)
          setTotalFrames(data.totalFrames)
          setPercentage(data.percentage)
          setErrorMessage(data.error_message)
          setPptFileId(data.result_ppt_file_id)

          // Detect fallback: cloud mode switched to local (per D-08)
          if ('mode' in data && data.mode === 'local' && mode === 'cloud') {
            setFallbackToLocal(true)
          }

          // 转录完成
          if (data.status === 'completed' && data.result_ppt_file_id) {
            onCompleted(data.result_ppt_file_id)
          }
        }
      } catch (error) {
        console.error('获取转录状态失败:', error)
        // 不显示错误消息，静默失败继续轮询
      }
    }

    fetchStatus()

    // 设置轮询间隔 (use pollInterval variable)
    const interval = setInterval(fetchStatus, pollInterval)

    return () => clearInterval(interval) // 清理定时器
  }, [open, videoFileId, onCompleted, pollInterval, mode])

  // 下载 PPT
  const handleDownloadPpt = useCallback(() => {
    if (!pptFileId) return
    window.location.href = `/api/v1/files/${pptFileId}/download`
  }, [pptFileId])

  // 重新转录 (关闭进度模态框，让用户重新触发)
  const handleRetry = useCallback(() => {
    onClose()
  }, [onClose])

  // 渲染本地阶段列表 (per D-14, UI-SPEC)
  const renderLocalStages = useCallback(() => {
    const stages: TranscriptionStage[] = ['extracting', 'detecting', 'generating']
    const stageIndex = stages.indexOf(stage as TranscriptionStage)

    return (
      <div style={{ marginTop: 16 }}>
        {stages.map((s, index) => {
          const config = STAGE_CONFIG[s]
          const isCompleted = index < stageIndex
          const isActive = index === stageIndex && status === 'processing'
          const isPending = index > stageIndex

          let icon: React.ReactNode
          let text: string
          let textStyle: React.CSSObject = {}

          if (isCompleted) {
            icon = <CheckCircleOutlined style={{ color: '#52c41a' }} />
            text = `✓ ${config.label}...`
            textStyle = { color: '#52c41a' }
          } else if (isActive) {
            icon = config.icon
            if (s === 'detecting' && framesProcessed > 0 && totalFrames > 0) {
              text = `● ${config.label} (${framesProcessed}/${totalFrames})...`
            } else {
              text = `● ${config.label}...`
            }
            textStyle = { color: '#1890ff' }
          } else {
            icon = <span style={{ color: '#d9d9d9' }}>○</span>
            text = `○ ${config.label}...`
            textStyle = { color: '#d9d9d9' }
          }

          return (
            <div key={s} style={{ marginBottom: 8, fontSize: 14, ...textStyle }}>
              <Space size={8}>
                {icon}
                <span>{text}</span>
              </Space>
            </div>
          )
        })}
      </div>
    )
  }, [stage, status, framesProcessed, totalFrames])

  // 渲染云端阶段列表 (per D-06)
  const renderCloudStages = useCallback(() => {
    const stages: CloudTranscriptionStage[] = ['uploading', 'queued', 'cloud_processing', 'downloading']
    const stageIndex = stages.indexOf(stage as CloudTranscriptionStage)

    return (
      <div style={{ marginTop: 16 }}>
        {stages.map((s, index) => {
          const config = CLOUD_STAGE_CONFIG[s]
          const isCompleted = index < stageIndex
          const isActive = index === stageIndex && status === 'processing'
          const isPending = index > stageIndex

          let icon: React.ReactNode
          let text: string
          let textStyle: React.CSSObject = {}

          if (isCompleted) {
            icon = <CheckCircleOutlined style={{ color: '#52c41a' }} />
            text = `✓ ${config.label}...`
            textStyle = { color: '#52c41a' }
          } else if (isActive) {
            icon = config.icon
            text = `● ${config.label}...`
            textStyle = { color: '#1890ff' }
          } else {
            icon = <span style={{ color: '#d9d9d9' }}>○</span>
            text = `○ ${config.label}...`
            textStyle = { color: '#d9d9d9' }
          }

          return (
            <div key={s} style={{ marginBottom: 8, fontSize: 14, ...textStyle }}>
              <Space size={8}>
                {icon}
                <span>{text}</span>
              </Space>
            </div>
          )
        })}
      </div>
    )
  }, [stage, status])

  // 渲染阶段列表 - 路由到本地或云端 (per D-06)
  const renderStages = useCallback(() => {
    if (mode === 'cloud' && !fallbackToLocal) {
      return renderCloudStages()
    }
    return renderLocalStages()
  }, [mode, fallbackToLocal, renderCloudStages, renderLocalStages])

  // 渲染完成状态 (per D-15, UI-SPEC)
  if (status === 'completed' && pptFileId) {
    // Determine title based on mode and fallback
    const modalTitle = fallbackToLocal
      ? `本地转录进度（自动切换） - ${fileName}`
      : mode === 'cloud'
      ? `云端转录进度 - ${fileName}`
      : `本地转录进度 - ${fileName}`

    return (
      <Modal
        title={modalTitle}
        open={open}
        onCancel={onClose}
        footer={[
          <Button key="retry" icon={<ReloadOutlined />} onClick={handleRetry}>
            重新转录
          </Button>,
          <Button key="download" type="primary" icon={<DownloadOutlined />} onClick={handleDownloadPpt}>
            下载PPT
          </Button>,
          <Button key="close" onClick={onClose}>
            关闭
          </Button>,
        ]}
        width={600}
      >
        <div style={{ textAlign: 'center', padding: '20px 0' }}>
          <CheckCircleOutlined style={{ fontSize: 48, color: '#52c41a', marginBottom: 16 }} />
          <div style={{ fontSize: 16, marginBottom: 16 }}>✓ 转录完成</div>
          <div style={{ color: '#8c8c8c', fontSize: 14 }}>
            共检测到 {totalFrames} 个画面变化
            <br />
            已生成 PPT 文件: {totalFrames} 页
          </div>
        </div>
      </Modal>
    )
  }

  // 渲染失败状态
  if (status === 'failed') {
    // Determine title based on mode and fallback
    const modalTitle = fallbackToLocal
      ? `本地转录进度（自动切换） - ${fileName}`
      : mode === 'cloud'
      ? `云端转录进度 - ${fileName}`
      : `本地转录进度 - ${fileName}`

    return (
      <Modal
        title={modalTitle}
        open={open}
        onCancel={onClose}
        footer={[
          <Button key="retry" type="primary" icon={<ReloadOutlined />} onClick={handleRetry}>
            重试
          </Button>,
          <Button key="close" onClick={onClose}>
            关闭
          </Button>,
        ]}
        width={600}
      >
        <div style={{ textAlign: 'center', padding: '20px 0' }}>
          <CloseCircleOutlined style={{ fontSize: 48, color: '#ff4d4f', marginBottom: 16 }} />
          <div style={{ fontSize: 16, marginBottom: 16, color: '#ff4d4f' }}>转录失败</div>
          <div style={{ color: '#8c8c8c', fontSize: 14 }}>
            {errorMessage || '转录过程中发生错误，请重试'}
          </div>
        </div>
      </Modal>
    )
  }

  // Determine title based on mode and fallback
  const modalTitle = fallbackToLocal
    ? `本地转录进度（自动切换） - ${fileName}`
    : mode === 'cloud'
    ? `云端转录进度 - ${fileName}`
    : `本地转录进度 - ${fileName}`

  // 渲染进度状态 (per UI-SPEC)
  return (
    <Modal
      title={modalTitle}
      open={open}
      onCancel={onClose} // 允许关闭，非阻塞 per D-14
      footer={[
        <Button key="close" onClick={onClose}>
          关闭
        </Button>,
      ]}
      width={600}
    >
      {/* Fallback alert (per D-08) */}
      {fallbackToLocal && (
        <Alert
          message="云端转录失败，已自动切换到本地转录"
          type="info"
          showIcon
          icon={<InfoCircleOutlined />}
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 当前阶段显示 */}
      <div style={{ marginBottom: 16, fontSize: 14, fontWeight: 500 }}>
        当前阶段:{' '}
        {stage && totalFrames > 0 && stage === 'detecting'
          ? ` 画面检测中 (${framesProcessed}/${totalFrames})`
          : stage
          ? ` ${STAGE_CONFIG[stage]?.label || CLOUD_STAGE_CONFIG[stage as CloudTranscriptionStage]?.label || stage}...`
          : ' 排队中...'}
      </div>

      {/* 进度条 */}
      {status === 'processing' && (
        <div style={{ marginBottom: 16 }}>
          <Progress percent={percentage} status="active" />
        </div>
      )}

      {/* 阶段列表 */}
      {renderStages()}

      {/* 提示信息 */}
      <div style={{ marginTop: 16, padding: '12px', background: '#f5f5f5', borderRadius: 4 }}>
        <div style={{ fontSize: 13, color: '#8c8c8c', marginBottom: 8 }}>
          {mode === 'cloud' && !fallbackToLocal
            ? '云端转录预计需要 5-10 分钟，期间您可以关闭此窗口继续使用系统。完成后将显示通知。'
            : '转录预计需要 2-3 分钟，期间您可以关闭此窗口继续使用系统。完成后将显示通知。'}
        </div>
        {samplingRate && (
          <div style={{ fontSize: 13, color: '#8c8c8c' }}>
            提示: 采样间隔 {samplingRate}s/帧
          </div>
        )}
      </div>
    </Modal>
  )
}
