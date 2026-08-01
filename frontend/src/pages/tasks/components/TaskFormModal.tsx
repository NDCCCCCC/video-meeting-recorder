// 新建/编辑录制任务对话框
// 从 pages/tasks/index.tsx 抽出。form 实例由父级传入（父级继续持有 openModal/closeModal/handleSubmit）。

import { Modal, Form, Input, Select, DatePicker, Space, Tag } from 'antd'
import type { FormInstance } from 'antd'
import {
  canEditAllFields,
  DEFAULT_PRE_JOIN_MINUTES,
  DEFAULT_RECORD_DELAY_MINUTES,
} from '../constants'
import {
  getConfigType,
  getConfigTypeTagConfig,
  CONFIG_TYPE_TAGS,
  validateInputConfigSelection,
} from '../utils'
import type { VideoRecordingTask } from '../../../types/task'
import type { InputConfig } from '../../../types/input-config'

// 录制中状态提示内容 (rendering-hoist-jsx)
const RECORDING_TIP = (
  <div style={{ marginBottom: 16, padding: 12, background: '#e6f7ff', borderRadius: 4 }}>
    任务正在录制中，仅可修改结束时间
  </div>
)

function renderConfigTypeTag(config: InputConfig) {
  const tagConfig = getConfigTypeTagConfig(config)
  return <Tag color={tagConfig.color}>{tagConfig.label}</Tag>
}

export interface TaskFormModalProps {
  open: boolean
  form: FormInstance
  editingTask: VideoRecordingTask | null
  huaweiConfigs: InputConfig[]
  configsLoading: boolean
  onOk: () => void
  onCancel: () => void
}

export function TaskFormModal({
  open,
  form,
  editingTask,
  huaweiConfigs,
  configsLoading,
  onOk,
  onCancel,
}: TaskFormModalProps) {
  return (
    <Modal
      title={
        editingTask
          ? editingTask.status === 'recording'
            ? '修改结束时间'
            : '编辑录制任务'
          : '新建录制任务'
      }
      open={open}
      onOk={onOk}
      onCancel={onCancel}
      width={700}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        {/* 录制中状态只能编辑结束时间，其他字段被禁用并显示提示 */}
        {editingTask && editingTask.status === 'recording' ? RECORDING_TIP : null}

        <Form.Item
          name="name"
          label="任务名称"
          rules={[
            {
              required: !editingTask || canEditAllFields(editingTask.status),
              message: '请输入任务名称',
            },
            { max: 200, message: '任务名称最多200个字符' },
          ]}
        >
          <Input
            placeholder="例：周例会（2026-07-28）"
            disabled={!!editingTask && !canEditAllFields(editingTask.status)}
          />
        </Form.Item>

        <Form.Item
          name="description"
          label="描述"
          rules={[{ max: 500, message: '描述最多500个字符' }]}
        >
          <Input.TextArea
            placeholder="会议主题、参会人或备注"
            rows={3}
            disabled={!!editingTask && !canEditAllFields(editingTask.status)}
          />
        </Form.Item>

        <Form.Item
          name="conference_number"
          label="会议号"
          rules={[
            {
              required: !editingTask || canEditAllFields(editingTask.status),
              message: '请输入会议号',
            },
            { max: 50, message: '会议号最多50个字符' },
          ]}
        >
          <Input
            placeholder="华为会议号，如 987654321"
            disabled={!!editingTask && !canEditAllFields(editingTask.status)}
          />
        </Form.Item>

        <Form.Item
          name="huawei_config_id"
          label="输入配置（可选，最多选一路USB和一路流媒体）"
          rules={[
            {
              validator: async (_, value) => {
                // 输入配置是可选的，如果用户选择了配置则验证
                if (value && (Array.isArray(value) ? value.length > 0 : value)) {
                  const ids = Array.isArray(value) ? value : [value]
                  const selectedConfigs = huaweiConfigs.filter((c) => ids.includes(c.id))
                  const error = validateInputConfigSelection(selectedConfigs)
                  if (error) throw new Error(error)
                }
              },
            },
          ]}
        >
          <Select
            mode="multiple"
            placeholder="最多一路 USB + 一路流媒体"
            loading={configsLoading}
            showSearch
            optionFilterProp="label"
            disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            tagRender={(props) => {
              const { label, value, onClose } = props
              const config = huaweiConfigs.find((c) => c.id === value)
              const tagConfig = config ? getConfigTypeTagConfig(config) : CONFIG_TYPE_TAGS.none

              return (
                <Tag color={tagConfig.color} closable onClose={onClose} style={{ marginRight: 3 }}>
                  {label}
                </Tag>
              )
            }}
          >
            {huaweiConfigs.map((config) => {
              const configType = getConfigType(config)
              // 根据配置类型显示不同信息
              const detailInfo =
                configType === 'usb'
                  ? `${config.usb_camera_device || '无摄像头'}`
                  : configType === 'stream'
                    ? `${config.stream_protocol || 'RTMP'}:${config.stream_url || '无地址'}`
                    : `${config.server || '无服务器'}:${config.port || 80}`

              return (
                <Select.Option key={config.id} value={config.id}>
                  <Space>
                    {config.name} ({detailInfo}){renderConfigTypeTag(config)}
                  </Space>
                </Select.Option>
              )
            })}
          </Select>
        </Form.Item>

        <Space size="large">
          <Form.Item
            name="start_time"
            label="开始时间"
            rules={[
              {
                required: !editingTask || canEditAllFields(editingTask.status),
                message: '请选择开始时间',
              },
            ]}
          >
            <DatePicker
              showTime
              format="YYYY-MM-DD HH:mm:ss"
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="end_time"
            label="结束时间"
            rules={[{ required: true, message: '请选择结束时间' }]}
          >
            <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
          </Form.Item>
        </Space>

        <Space size="large">
          <Form.Item
            name="pre_join_minutes"
            label="提前进入(分钟)"
            rules={[
              { type: 'number', min: 0, max: 60, message: '提前进入时间必须在0-60分钟之间' },
            ]}
            initialValue={DEFAULT_PRE_JOIN_MINUTES}
          >
            <Input
              type="number"
              style={{ width: 120 }}
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>

          <Form.Item
            name="record_delay_minutes"
            label="录制延迟(分钟)"
            rules={[{ type: 'number', min: 0, max: 60, message: '录制延迟必须在0-60分钟之间' }]}
            initialValue={DEFAULT_RECORD_DELAY_MINUTES}
          >
            <Input
              type="number"
              style={{ width: 120 }}
              disabled={!!editingTask && !canEditAllFields(editingTask.status)}
            />
          </Form.Item>
        </Space>
      </Form>
    </Modal>
  )
}
