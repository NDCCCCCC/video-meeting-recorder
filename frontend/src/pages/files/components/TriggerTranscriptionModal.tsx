// 本地转录触发对话框（采样率选择，展示型）
// 从 pages/files/index.tsx 抽出。selectedSamplingRate/transcriptionVideoFile 等
// 与转录进度弹窗共享的 state 仍由父级持有。

import { Modal, Radio } from 'antd'
import { samplingRateOptions } from '../constants'
import type { VideoFile } from '../../../types/video-file'

export interface TriggerTranscriptionModalProps {
  open: boolean
  file: VideoFile | null
  samplingRate: number
  onSamplingRateChange: (rate: number) => void
  loading: boolean
  onClose: () => void
  onOk: () => void
}

export function TriggerTranscriptionModal({
  open,
  file,
  samplingRate,
  onSamplingRateChange,
  loading,
  onClose,
  onOk,
}: TriggerTranscriptionModalProps) {
  return (
    <Modal
      title={`本地转录 - ${file?.file_name || ''}`}
      open={open}
      onCancel={onClose}
      onOk={onOk}
      okText="开始转录"
      cancelText="取消"
      confirmLoading={loading}
    >
      <div style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 8 }}>选择采样间隔:</div>
        <Radio.Group value={samplingRate} onChange={(e) => onSamplingRateChange(e.target.value)}>
          {samplingRateOptions.map((opt) => (
            <Radio key={opt.value} value={opt.value} style={{ display: 'block', marginBottom: 8 }}>
              {opt.label} ({opt.description})
            </Radio>
          ))}
        </Radio.Group>
      </div>
      <div style={{ color: '#faad14', fontSize: 13 }}>
        提示: 转录过程可能需要几分钟，期间可以关闭此窗口继续使用系统。
      </div>
    </Modal>
  )
}
