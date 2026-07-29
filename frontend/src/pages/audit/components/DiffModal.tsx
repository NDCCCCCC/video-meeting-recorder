import { Modal } from 'antd'
import { diffJson } from 'diff'
import type { AuditLog } from '../../../types/audit'

interface DiffModalProps {
  log: AuditLog | null
  open: boolean
  onClose: () => void
}

export function DiffModal({ log, open, onClose }: DiffModalProps) {
  if (!log) return null

  // Parse JSON data
  const oldData = log.old_data ? JSON.parse(log.old_data) : {}
  const newData = log.new_data ? JSON.parse(log.new_data) : {}

  // Compute diff
  const oldText = JSON.stringify(oldData, null, 2)
  const newText = JSON.stringify(newData, null, 2)
  const changes = diffJson(oldText, newText)

  return (
    <Modal title="变更详情" open={open} onCancel={onClose} width={1000} footer={null}>
      <div style={{ display: 'flex', gap: '16px' }}>
        {/* Old Data */}
        <div style={{ flex: 1 }}>
          <h4>变更前</h4>
          <pre
            style={{
              background: '#f5f5f5',
              padding: '12px',
              borderRadius: '6px',
              fontSize: '12px',
              maxHeight: 400,
              overflow: 'auto',
            }}
          >
            {changes.map((part, index) => (
              <span
                key={index}
                style={{
                  backgroundColor: part.removed ? '#ffccc7' : 'transparent',
                  color: part.removed ? '#ff4d4f' : 'inherit',
                }}
              >
                {part.value}
              </span>
            ))}
          </pre>
        </div>

        {/* New Data */}
        <div style={{ flex: 1 }}>
          <h4>变更后</h4>
          <pre
            style={{
              background: '#f5f5f5',
              padding: '12px',
              borderRadius: '6px',
              fontSize: '12px',
              maxHeight: 400,
              overflow: 'auto',
            }}
          >
            {changes.map((part, index) => (
              <span
                key={index}
                style={{
                  backgroundColor: part.added ? '#b7eb8f' : 'transparent',
                  color: part.added ? '#52c41a' : 'inherit',
                }}
              >
                {part.value}
              </span>
            ))}
          </pre>
        </div>
      </div>
    </Modal>
  )
}
