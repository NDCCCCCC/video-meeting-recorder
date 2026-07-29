import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Button,
  Table,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  message,
  Switch,
  Popconfirm,
  Tooltip,
  Card,
  DatePicker,
} from 'antd'
import { PlusOutlined, ReloadOutlined, CopyOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import type { Dayjs } from 'dayjs'
import type { ColumnsType } from 'antd/es/table'
import * as apikeyAPI from '../../../api/apikey'
import type { APIKey, CreateAPIKeyRequest, UpdateAPIKeyRequest } from '../../../types/apikey'
import { API_KEY_SCOPES } from '../../../types/apikey'

// 纯函数，不依赖组件状态，提取到组件外部
const maskKey = (key: string) => {
  if (!key) return ''
  if (key.length <= 12) return '********'
  return `${key.substring(0, 12)}${'*'.repeat(16)}`
}

const formatTime = (timeStr: string | null) => {
  if (!timeStr) return '永久有效'
  return new Date(timeStr).toLocaleString('zh-CN')
}

const processIPWhitelist = (value: string): string[] => {
  if (!value || typeof value !== 'string') return []
  return value
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0)
}

const ipWhitelistArrayToString = (arr: string[]): string => {
  if (!arr || arr.length === 0) return ''
  return arr.join('\n')
}

const APIKeysPage: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [keyword, setKeyword] = useState('')

  // 创建/编辑模态框
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [editModalVisible, setEditModalVisible] = useState(false)
  const [currentKey, setCurrentKey] = useState<APIKey | null>(null)
  const [form] = Form.useForm()

  // 显示完整密钥的模态框
  const [fullKeyModalVisible, setFullKeyModalVisible] = useState(false)
  const [fullKeyValue, setFullKeyValue] = useState('')

  // 加载API密钥列表
  const loadAPIKeys = useCallback(async () => {
    setLoading(true)
    try {
      const res = await apikeyAPI.listAPIKeys({
        page,
        page_size: pageSize,
        keyword: keyword || undefined,
      })
      setApiKeys(res.data?.items || [])
      setTotal(res.data?.total || 0)
    } catch {
      message.error('加载API密钥列表失败')
    } finally {
      setLoading(false)
    }
  }, [page, pageSize, keyword])

  useEffect(() => {
    loadAPIKeys()
  }, [loadAPIKeys])

  // 创建API密钥
  const handleCreate = async (values: CreateAPIKeyRequest & { expires_at?: Dayjs }) => {
    try {
      const processedValues = {
        name: values.name,
        scopes: values.scopes,
        inherit_perms: values.inherit_perms,
        ip_whitelist: processIPWhitelist(values.ip_whitelist as unknown as string),
        expires_at: values.expires_at ? values.expires_at.toISOString() : null,
        description: values.description,
      }
      const res = await apikeyAPI.createAPIKey(processedValues)
      message.success('API密钥创建成功')
      setCreateModalVisible(false)
      form.resetFields()

      // 显示完整密钥
      setFullKeyValue(res.data?.key || '')
      setFullKeyModalVisible(true)

      loadAPIKeys()
    } catch (error: any) {
      message.error(error.message || '创建失败')
    }
  }

  // 更新API密钥
  const handleUpdate = async (values: UpdateAPIKeyRequest) => {
    if (!currentKey) return
    try {
      const processedValues = {
        ...values,
        ip_whitelist: processIPWhitelist(values.ip_whitelist as any),
      }
      await apikeyAPI.updateAPIKey(currentKey.id, processedValues)
      message.success('API密钥更新成功')
      setEditModalVisible(false)
      setCurrentKey(null)
      form.resetFields()
      loadAPIKeys()
    } catch (error: any) {
      message.error(error.message || '更新失败')
    }
  }

  // 删除API密钥
  const handleDelete = useCallback(
    async (id: number) => {
      try {
        await apikeyAPI.deleteAPIKey(id)
        message.success('API密钥删除成功')
        loadAPIKeys()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    },
    [loadAPIKeys]
  )

  // 切换状态
  const handleToggle = useCallback(
    async (id: number) => {
      try {
        await apikeyAPI.toggleAPIKeyStatus(id)
        message.success('状态切换成功')
        loadAPIKeys()
      } catch (error: any) {
        message.error(error.message || '操作失败')
      }
    },
    [loadAPIKeys]
  )

  // 打开编辑模态框
  const openEditModal = useCallback(
    (record: APIKey) => {
      setCurrentKey(record)
      form.setFieldsValue({
        name: record.name,
        is_active: record.is_active,
        scopes: record.scopes,
        ip_whitelist: ipWhitelistArrayToString(record.ip_whitelist),
        description: record.description,
      })
      setEditModalVisible(true)
    },
    [form]
  )

  // 复制密钥
  const copyKey = useCallback((key: string) => {
    navigator.clipboard
      .writeText(key)
      .then(() => {
        message.success('密钥已复制到剪贴板')
      })
      .catch(() => {
        const textarea = document.createElement('textarea')
        textarea.value = key
        textarea.style.position = 'fixed'
        textarea.style.opacity = '0'
        document.body.appendChild(textarea)
        textarea.select()
        try {
          document.execCommand('copy')
          message.success('密钥已复制到剪贴板')
        } catch {
          message.error('复制失败，请手动复制')
        }
        document.body.removeChild(textarea)
      })
  }, [])

  const columns: ColumnsType<APIKey> = useMemo(
    () => [
      {
        title: 'ID',
        dataIndex: 'id',
        width: 60,
      },
      {
        title: '名称',
        dataIndex: 'name',
        width: 150,
        render: (text, record) => (
          <div>
            <div>{text}</div>
            {record.description && (
              <div style={{ fontSize: '12px', color: '#9ca3af' }}>{record.description}</div>
            )}
          </div>
        ),
      },
      {
        title: '密钥',
        dataIndex: 'key',
        width: 200,
        render: (text) => (
          <Space>
            <code
              style={{
                fontSize: '12px',
                backgroundColor: '#f3f4f6',
                padding: '4px 8px',
                borderRadius: '4px',
              }}
            >
              {maskKey(text)}
            </code>
            {text && (
              <Tooltip title="复制完整密钥">
                <Button
                  type="text"
                  size="small"
                  icon={<CopyOutlined />}
                  onClick={() => copyKey(text)}
                />
              </Tooltip>
            )}
          </Space>
        ),
      },
      {
        title: '作用域',
        dataIndex: 'scopes',
        width: 120,
        render: (scopes: string[]) => (
          <>
            {scopes?.map((scope) => (
              <Tag
                key={scope}
                color={scope === 'admin' ? 'red' : scope === 'write' ? 'blue' : 'green'}
              >
                {scope}
              </Tag>
            ))}
          </>
        ),
      },
      {
        title: 'IP白名单',
        dataIndex: 'ip_whitelist',
        width: 150,
        render: (whitelist: string[]) => {
          if (!whitelist || whitelist.length === 0)
            return <span style={{ color: '#9ca3af' }}>未设置</span>
          return (
            <Tooltip title={whitelist.join('\n')}>
              <span>{whitelist.length} 条规则</span>
            </Tooltip>
          )
        },
      },
      {
        title: '有效期',
        dataIndex: 'expires_at',
        width: 120,
        render: formatTime,
      },
      {
        title: '最后使用',
        dataIndex: 'last_used_at',
        width: 120,
        render: (timeStr: string | null) => {
          if (!timeStr) return <span style={{ color: '#9ca3af' }}>未使用</span>
          return formatTime(timeStr)
        },
      },
      {
        title: '状态',
        dataIndex: 'is_active',
        width: 80,
        render: (active: boolean, record) => (
          <Switch
            checked={active}
            onChange={() => handleToggle(record.id)}
            checkedChildren="启用"
            unCheckedChildren="禁用"
          />
        ),
      },
      {
        title: '操作',
        key: 'action',
        width: 120,
        fixed: 'right' as const,
        render: (_, record) => (
          <Space>
            <Button type="link" size="small" onClick={() => openEditModal(record)}>
              编辑
            </Button>
            <Popconfirm
              title="确定要删除这个API密钥吗？"
              onConfirm={() => handleDelete(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button type="link" size="small" danger>
                删除
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [handleDelete, handleToggle, openEditModal, copyKey]
  )

  return (
    <div style={{ padding: '20px' }}>
      <Card>
        <div
          style={{
            marginBottom: '16px',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Space>
            <Input.Search
              placeholder="搜索名称"
              allowClear
              style={{ width: 200 }}
              onSearch={(value) => {
                setKeyword(value)
                setPage(1)
              }}
            />
            <Button icon={<ReloadOutlined />} onClick={loadAPIKeys}>
              刷新
            </Button>
          </Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateModalVisible(true)}
          >
            新建密钥
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={apiKeys}
          rowKey="id"
          loading={loading}
          scroll={{ x: 1000 }}
          pagination={{
            current: page,
            pageSize: pageSize,
            total: total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (p, ps) => {
              setPage(p)
              setPageSize(ps || 20)
            },
          }}
        />
      </Card>

      {/* 创建API密钥模态框 */}
      <Modal
        title="创建API密钥"
        open={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false)
          form.resetFields()
        }}
        onOk={() => form.submit()}
        width={600}
        destroyOnClose
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreate}
          initialValues={{
            scopes: ['read'],
            inherit_perms: true,
            ip_whitelist: [],
          }}
        >
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="例如：生产环境只读密钥" />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input.TextArea placeholder="可选描述信息" rows={2} />
          </Form.Item>

          <Form.Item
            label="作用域"
            name="scopes"
            rules={[{ required: true, message: '请选择作用域' }]}
          >
            <Select mode="multiple" options={API_KEY_SCOPES} placeholder="选择作用域" />
          </Form.Item>

          <Form.Item label="有效期" name="expires_at">
            <DatePicker
              showTime
              style={{ width: '100%' }}
              placeholder="留空表示永久有效"
              disabledDate={(current) => current && current < dayjs().startOf('day')}
            />
          </Form.Item>

          <Form.Item
            label="权限继承"
            name="inherit_perms"
            valuePropName="checked"
            tooltip="启用后，密钥将继承所属用户的完整角色权限"
          >
            <Switch checkedChildren="继承权限" unCheckedChildren="自定义权限" />
          </Form.Item>

          <Form.Item
            label="IP白名单"
            name="ip_whitelist"
            tooltip="每行一个IP地址或CIDR范围，留空表示不限制"
          >
            <Input.TextArea
              placeholder="例如：&#10;192.168.1.100&#10;192.168.1.0/24"
              rows={4}
            />
          </Form.Item>

          <div style={{ color: '#9ca3af', fontSize: '14px' }}>
            <p>
              <strong>注意：</strong>
            </p>
            <ul style={{ listStyleType: 'disc', listStylePosition: 'inside' }}>
              <li>密钥创建后将显示完整的密钥值，请妥善保存</li>
              <li>出于安全考虑，完整密钥只会显示一次</li>
              <li>有效期留空表示永久有效</li>
            </ul>
          </div>
        </Form>
      </Modal>

      {/* 编辑API密钥模态框 */}
      <Modal
        title="编辑API密钥"
        open={editModalVisible}
        onCancel={() => {
          setEditModalVisible(false)
          setCurrentKey(null)
          form.resetFields()
        }}
        onOk={() => form.submit()}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical" onFinish={handleUpdate}>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>

          <Form.Item label="描述" name="description">
            <Input.TextArea rows={2} />
          </Form.Item>

          <Form.Item
            label="作用域"
            name="scopes"
            rules={[{ required: true, message: '请选择作用域' }]}
          >
            <Select mode="multiple" options={API_KEY_SCOPES} />
          </Form.Item>

          <Form.Item label="IP白名单" name="ip_whitelist">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 完整密钥显示模态框 */}
      <Modal
        title="API密钥创建成功"
        open={fullKeyModalVisible}
        onCancel={() => setFullKeyModalVisible(false)}
        footer={[
          <Button
            key="copy"
            type="primary"
            onClick={() => {
              copyKey(fullKeyValue)
              setFullKeyModalVisible(false)
            }}
          >
            复制并关闭
          </Button>,
        ]}
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
          <div
            style={{
              backgroundColor: '#fefce8',
              border: '1px solid #fde047',
              borderRadius: '4px',
              padding: '16px',
            }}
          >
            <p style={{ color: '#92400e', fontWeight: 500 }}>⚠️ 重要提示</p>
            <p style={{ color: '#a16207', fontSize: '14px', marginTop: '8px' }}>
              出于安全考虑，完整密钥只会显示这一次。请立即复制并妥善保存。
            </p>
          </div>

          <div>
            <label
              style={{ display: 'block', fontSize: '14px', fontWeight: 500, marginBottom: '8px' }}
            >
              您的API密钥：
            </label>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <code
                style={{
                  flex: 1,
                  fontFamily: 'monospace',
                  fontSize: '14px',
                  backgroundColor: '#f3f4f6',
                  padding: '8px 12px',
                  borderRadius: '4px',
                  wordBreak: 'break-all',
                  userSelect: 'all',
                }}
              >
                {fullKeyValue}
              </code>
              <Button icon={<CopyOutlined />} onClick={() => copyKey(fullKeyValue)}>
                复制
              </Button>
            </div>
          </div>

          <div style={{ fontSize: '14px', color: '#4b5563' }}>
            <p style={{ fontWeight: 500, marginBottom: '8px' }}>使用方式：</p>
            <code
              style={{
                display: 'block',
                backgroundColor: '#f3f4f6',
                padding: '8px',
                borderRadius: '4px',
              }}
            >
              X-API-Key: {fullKeyValue}
            </code>
          </div>
        </div>
      </Modal>
    </div>
  )
}

export default APIKeysPage
