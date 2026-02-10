// 会议管理页面

import { useState, useEffect } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Modal,
  Form,
  message,
  Popconfirm,
  Tag,
  Select,
  DatePicker,
  Descriptions,
  Card,
} from 'antd'
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  ReloadOutlined,
  CalendarOutlined,
  VideoCameraOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import dayjs from 'dayjs'
import * as conferenceApi from '../../api/conference'
import * as huaweiConfigApi from '../../api/huawei-config'
import type {
  ConferenceRecord,
  ConferenceListParams,
  CreateConferenceRequest,
  UpdateConferenceRequest,
  ConferenceStatus,
} from '../../types/conference'
import type { HuaweiConfig } from '../../types/huawei-config'

const statusOptions = [
  { label: '未开始', value: 'not_started' },
  { label: '进行中', value: 'in_progress' },
  { label: '已完成', value: 'completed' },
  { label: '失败', value: 'failed' },
]

const statusColors: Record<ConferenceStatus, string> = {
  not_started: 'default',
  in_progress: 'processing',
  completed: 'success',
  failed: 'error',
}

const statusLabels: Record<ConferenceStatus, string> = {
  not_started: '未开始',
  in_progress: '进行中',
  completed: '已完成',
  failed: '失败',
}

export default function ConferenceManagement() {
  const [conferences, setConferences] = useState<ConferenceRecord[]>([])
  const [huaweiConfigs, setHuaweiConfigs] = useState<HuaweiConfig[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [detailVisible, setDetailVisible] = useState(false)
  const [editingConference, setEditingConference] = useState<ConferenceRecord | null>(null)
  const [viewingConference, setViewingConference] = useState<ConferenceRecord | null>(null)
  const [form] = Form.useForm()

  const [params, setParams] = useState<ConferenceListParams>({
    page: 1,
    page_size: 20,
  })

  const loadConferences = async () => {
    setLoading(true)
    try {
      const response = await conferenceApi.getConferenceList(params)
      if (response.data) {
        setConferences(response.data.items)
        setTotal(response.data.total)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载会议列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadHuaweiConfigs = async () => {
    try {
      const response = await huaweiConfigApi.getActiveHuaweiConfigs()
      if (response.data) {
        setHuaweiConfigs(response.data)
      }
    } catch (error) {
      console.error('Failed to load Huawei configs:', error)
    }
  }

  useEffect(() => {
    loadConferences()
  }, [params])

  useEffect(() => {
    loadHuaweiConfigs()
  }, [])

  const handleSearch = (value: string) => {
    setParams({ ...params, keyword: value, page: 1 })
  }

  const handleStatusFilter = (status: ConferenceStatus | undefined) => {
    setParams({ ...params, status, page: 1 })
  }

  const handleTableChange = (pagination: any) => {
    setParams({
      ...params,
      page: pagination.current,
      page_size: pagination.pageSize,
    })
  }

  const openModal = (conference: ConferenceRecord | null = null) => {
    setEditingConference(conference)
    if (conference) {
      form.setFieldsValue({
        conference_number: conference.conference_number,
        title: conference.title,
        start_time: dayjs(conference.start_time),
        end_time: conference.end_time ? dayjs(conference.end_time) : null,
        description: conference.description,
        huawei_config_id: conference.huawei_config_id,
        status: conference.status,
      })
    } else {
      form.resetFields()
      form.setFieldsValue({
        start_time: dayjs(),
        end_time: dayjs().add(1, 'hour'),
      })
    }
    setModalVisible(true)
  }

  const closeModal = () => {
    setModalVisible(false)
    setEditingConference(null)
    form.resetFields()
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()

      if (editingConference) {
        const req: UpdateConferenceRequest = {
          title: values.title,
          end_time: values.end_time ? values.end_time.toISOString() : undefined,
          description: values.description,
          status: values.status,
        }
        await conferenceApi.updateConference(editingConference.id, req)
        message.success('更新成功')
      } else {
        const req: CreateConferenceRequest = {
          conference_number: values.conference_number,
          title: values.title,
          start_time: values.start_time.toISOString(),
          end_time: values.end_time.toISOString(),
          description: values.description,
          huawei_config_id: values.huawei_config_id,
        }
        await conferenceApi.createConference(req)
        message.success('创建成功')
      }

      closeModal()
      loadConferences()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败')
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await conferenceApi.deleteConference(id)
      message.success('删除成功')
      loadConferences()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }

  const viewDetail = (conference: ConferenceRecord) => {
    setViewingConference(conference)
    setDetailVisible(true)
  }

  const columns: ColumnsType<ConferenceRecord> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: '会议号',
      dataIndex: 'conference_number',
      width: 120,
    },
    {
      title: '标题',
      dataIndex: 'title',
      width: 200,
      ellipsis: true,
    },
    {
      title: '开始时间',
      dataIndex: 'start_time',
      width: 160,
      render: (time) => dayjs(time).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '结束时间',
      dataIndex: 'end_time',
      width: 160,
      render: (time) => (time ? dayjs(time).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (status: ConferenceStatus) => (
        <Tag color={statusColors[status]}>{statusLabels[status]}</Tag>
      ),
    },
    {
      title: '参会人数',
      dataIndex: 'attendees',
      width: 100,
    },
    {
      title: '视频文件',
      dataIndex: 'video_files',
      width: 100,
      render: (files) => files?.length || 0,
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      fixed: 'right' as const,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            onClick={() => viewDetail(record)}
          >
            详情
          </Button>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => openModal(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这个会议吗？"
            onConfirm={() => handleDelete(record.id)}
            disabled={record.video_recording_task !== null}
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={record.video_recording_task !== null}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div style={{ padding: '20px' }}>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>
          <CalendarOutlined /> 会议管理
        </h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
          新建会议
        </Button>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <Space size="middle" wrap>
          <Input.Search
            placeholder="搜索会议标题、描述或会议号"
            allowClear
            style={{ width: 300 }}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Select
            placeholder="筛选状态"
            allowClear
            style={{ width: 150 }}
            onChange={handleStatusFilter}
            options={statusOptions}
          />
          <Button icon={<ReloadOutlined />} onClick={loadConferences}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={conferences}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1200 }}
        pagination={{
          current: params.page,
          pageSize: params.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        onChange={handleTableChange}
      />

      {/* 新建/编辑会议对话框 */}
      <Modal
        title={editingConference ? '编辑会议' : '新建会议'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={closeModal}
        width={700}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="conference_number"
            label="会议号"
            rules={[
              { required: true, message: '请输入会议号' },
              { max: 50, message: '会议号最多50个字符' },
            ]}
          >
            <Input placeholder="请输入会议号" disabled={!!editingConference} />
          </Form.Item>

          <Form.Item
            name="title"
            label="会议标题"
            rules={[
              { required: true, message: '请输入会议标题' },
              { max: 200, message: '会议标题最多200个字符' },
            ]}
          >
            <Input placeholder="请输入会议标题" />
          </Form.Item>

          <Space size="large" style={{ width: '100%' }}>
            <Form.Item
              name="start_time"
              label="开始时间"
              rules={[{ required: true, message: '请选择开始时间' }]}
            >
              <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" disabled={!!editingConference} />
            </Form.Item>

            <Form.Item
              name="end_time"
              label="结束时间"
              rules={[{ required: true, message: '请选择结束时间' }]}
            >
              <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" />
            </Form.Item>
          </Space>

          <Form.Item name="huawei_config_id" label="华为配置">
            <Select
              placeholder="请选择华为配置（可选）"
              allowClear
              options={huaweiConfigs.map((c) => ({
                label: `${c.name} (${c.server}:${c.port})`,
                value: c.id,
              }))}
            />
          </Form.Item>

          {editingConference && (
            <Form.Item name="status" label="状态">
              <Select options={statusOptions} />
            </Form.Item>
          )}

          <Form.Item name="description" label="描述">
            <Input.TextArea
              placeholder="请输入会议描述"
              rows={4}
              maxLength={1000}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* 会议详情对话框 */}
      <Modal
        title={<Space><CalendarOutlined />会议详情 - {viewingConference?.title}</Space>}
        open={detailVisible}
        onCancel={() => setDetailVisible(false)}
        footer={[
          <Button key="close" onClick={() => setDetailVisible(false)}>
            关闭
          </Button>,
        ]}
        width={900}
      >
        {viewingConference && (
          <>
            <Descriptions column={2} bordered size="small" style={{ marginBottom: 16 }}>
              <Descriptions.Item label="会议ID">{viewingConference.id}</Descriptions.Item>
              <Descriptions.Item label="会议号">{viewingConference.conference_number}</Descriptions.Item>
              <Descriptions.Item label="标题" span={2}>{viewingConference.title}</Descriptions.Item>
              <Descriptions.Item label="开始时间">
                {dayjs(viewingConference.start_time).format('YYYY-MM-DD HH:mm:ss')}
              </Descriptions.Item>
              <Descriptions.Item label="结束时间">
                {viewingConference.end_time
                  ? dayjs(viewingConference.end_time).format('YYYY-MM-DD HH:mm:ss')
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={statusColors[viewingConference.status]}>
                  {statusLabels[viewingConference.status]}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label="参会人数">{viewingConference.attendees}</Descriptions.Item>
              <Descriptions.Item label="华为配置" span={2}>
                {viewingConference.huawei_config
                  ? `${viewingConference.huawei_config.name} (${viewingConference.huawei_config.server})`
                  : '-'}
              </Descriptions.Item>
              <Descriptions.Item label="描述" span={2}>
                {viewingConference.description || '-'}
              </Descriptions.Item>
            </Descriptions>

            {viewingConference.video_files && viewingConference.video_files.length > 0 && (
              <Card
                title={<Space><VideoCameraOutlined />视频文件 ({viewingConference.video_files.length})</Space>}
                size="small"
              >
                <Table
                  size="small"
                  dataSource={viewingConference.video_files}
                  rowKey="id"
                  pagination={false}
                  columns={[
                    { title: '文件名', dataIndex: 'file_name', ellipsis: true },
                    {
                      title: '文件大小',
                      dataIndex: 'file_size',
                      render: (size) => `${(size / 1024 / 1024).toFixed(2)} MB`,
                    },
                    {
                      title: '时长',
                      dataIndex: 'duration',
                      render: (d) => `${Math.floor(d / 60)}:${(d % 60).toString().padStart(2, '0')}`,
                    },
                  ]}
                />
              </Card>
            )}

            {viewingConference.video_recording_task && (
              <Card
                title="关联录制任务"
                size="small"
                style={{ marginTop: 16 }}
              >
                <Descriptions size="small" column={2}>
                  <Descriptions.Item label="任务ID">{viewingConference.video_recording_task.id}</Descriptions.Item>
                  <Descriptions.Item label="任务名称">{viewingConference.video_recording_task.name}</Descriptions.Item>
                  <Descriptions.Item label="状态">
                    <Tag color={viewingConference.video_recording_task.status === 'completed' ? 'green' : 'blue'}>
                      {viewingConference.video_recording_task.status}
                    </Tag>
                  </Descriptions.Item>
                </Descriptions>
              </Card>
            )}
          </>
        )}
      </Modal>
    </div>
  )
}
