// 批量转录配置对话框（展示型）
// 从 pages/files/index.tsx 抽出。state（mode/samplingRate/loading）与提交逻辑
// 留在父级（batchTranscribing 与工具栏批量按钮共享 disabled 态）。

import { Modal, Radio, Alert } from 'antd'
import { samplingRateOptions } from '../constants'
import type { TranscriptionMode } from '../../../types/transcription'

export interface BatchTranscribeConfigModalProps {
  open: boolean
  selectedCount: number
  mode: TranscriptionMode
  onModeChange: (mode: TranscriptionMode) => void
  samplingRate: number
  onSamplingRateChange: (rate: number) => void
  loading: boolean
  onClose: () => void
  onOk: () => void
}

export function BatchTranscribeConfigModal({
  open,
  selectedCount,
  mode,
  onModeChange,
  samplingRate,
  onSamplingRateChange,
  loading,
  onClose,
  onOk,
}: BatchTranscribeConfigModalProps) {
  return (
    <Modal
      title="批量转录配置"
      open={open}
      onOk={onOk}
      onCancel={onClose}
      okText="提交转录任务"
      cancelText="取消"
      confirmLoading={loading}
      width={600}
    >
      <div style={{ marginBottom: 16 }}>
        <span>已选择 {selectedCount} 个文件进行批量转录</span>
      </div>

      <div style={{ marginBottom: 16 }}>
        <div style={{ marginBottom: 8, fontWeight: 500 }}>转录模式</div>
        <Radio.Group
          value={mode}
          onChange={(e) => onModeChange(e.target.value)}
          disabled={loading}
        >
          <Radio value="local">本地转录（快速，免费）</Radio>
          <Radio value="cloud">云端转录（阿里通义听悟，更准确）</Radio>
        </Radio.Group>
      </div>

      {mode === 'local' && (
        <div style={{ marginBottom: 16 }}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>采样率</div>
          <div style={{ color: '#888', fontSize: 12, marginBottom: 8 }}>
            采样率决定每秒提取的帧数，数值越小精度越高但文件越大
          </div>
          <Radio.Group
            value={samplingRate}
            onChange={(e) => onSamplingRateChange(e.target.value)}
            disabled={loading}
          >
            {samplingRateOptions.map((opt) => (
              <Radio key={opt.value} value={opt.value} style={{ display: 'block', marginBottom: 8 }}>
                {opt.label} ({opt.description})
              </Radio>
            ))}
          </Radio.Group>
        </div>
      )}

      {mode === 'cloud' && (
        <div style={{ marginBottom: 16 }}>
          <Alert
            title="云端转录使用阿里通义听悟服务，支持更准确的语音识别和PPT提取，但需要消耗API配额"
            type="info"
            showIcon
          />
        </div>
      )}
    </Modal>
  )
}
