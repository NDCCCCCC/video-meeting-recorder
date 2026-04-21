// 用户管理页面

import { useState, useEffect } from 'react'
import {
  Table,
  Button,
  Space,
  Input,
  Select,
  Modal,
  Form,
  message,
  Popconfirm,
  Switch,
  Tag
} from 'antd'
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  LockOutlined,
  ReloadOutlined
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import * as userApi from '../../../api/user'
import type { UserInfo, UserListParams, CreateUserRequest, UpdateUserRequest } from '../../../types/user'
import { useAuthStore } from '../../../stores/authStore'

export default function UserManagement() {
  const { user: currentUser } = useAuthStore()
  const [users, setUsers] = useState<UserInfo[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [passwordModalVisible, setPasswordModalVisible] = useState(false)
  const [editingUser, setEditingUser] = useState<UserInfo | null>(null)
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null)
  const [form] = Form.useForm()
  const [passwordForm] = Form.useForm()

  // 查询参数
  const [params, setParams] = useState<UserListParams>({
    page: 1,
    page_size: 20,
  })

  // 加载用户列表
  const loadUsers = async () => {
    setLoading(true)
    try {
      const response = await userApi.getUserList(params)
      if (response.data) {
        setUsers(response.data.items)
        setTotal(response.data.total)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载用户列表失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadUsers()
  }, [params])

  // 搜索
  const handleSearch = (value: string) => {
    setParams({ ...params, keyword: value, page: 1 })
  }

  // 角色筛选
  const handleRoleFilter = (value: number | undefined) => {
    setParams({ ...params, role_id: value, page: 1 })
  }

  // 状态筛选
  const handleStatusFilter = (value: boolean | undefined) => {
    setParams({ ...params, is_active: value, page: 1 })
  }

  // 分页变化
  const handleTableChange = (pagination: { current?: number; pageSize?: number }) => {
    setParams({
      ...params,
      page: pagination.current ?? 1,
      page_size: pagination.pageSize ?? 20,
    })
  }

  // 打开新建/编辑对话框
  const openModal = (user: UserInfo | null = null) => {
    setEditingUser(user)
    if (user) {
      form.setFieldsValue({
        username: user.username,
        email: user.email,
        full_name: user.full_name,
        role_ids: user.roles?.map(r => r.id) || [],
        is_active: user.is_active,
      })
    } else {
      form.resetFields()
      form.setFieldsValue({
        is_active: true,
        role_ids: [],
      })
    }
    setModalVisible(true)
  }

  // 关闭对话框
  const closeModal = () => {
    setModalVisible(false)
    setEditingUser(null)
    form.resetFields()
  }

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()

      // Admin check for shared_viewer assignment (D-13)
      if (values.role_ids?.includes(5)) {
        if (!currentUser?.is_admin) {
          message.error('仅管理员可分配"共享查看者"角色')
          return
        }
      }

      if (editingUser) {
        // 更新用户
        const req: UpdateUserRequest = {
          email: values.email,
          full_name: values.full_name,
          role_ids: values.role_ids,
          is_active: values.is_active,
        }
        await userApi.updateUser(editingUser.id, req)
        message.success('更新成功')
      } else {
        // 创建用户
        const req: CreateUserRequest = {
          username: values.username,
          password: values.password,
          email: values.email,
          full_name: values.full_name,
          role_ids: values.role_ids || [],
          is_active: values.is_active ?? true,
        }
        await userApi.createUser(req)
        message.success('创建成功')
      }

      closeModal()
      loadUsers()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败')
    }
  }

  // 删除用户
  const handleDelete = async (id: number) => {
    try {
      await userApi.deleteUser(id)
      message.success('删除成功')
      loadUsers()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }

  // 切换用户状态
  const handleToggleStatus = async (id: number) => {
    try {
      await userApi.toggleUserStatus(id)
      message.success('状态更新成功')
      loadUsers()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败')
    }
  }

  // 打开重置密码对话框
  const openPasswordModal = (id: number) => {
    setSelectedUserId(id)
    setPasswordModalVisible(true)
  }

  // 关闭重置密码对话框
  const closePasswordModal = () => {
    setPasswordModalVisible(false)
    setSelectedUserId(null)
    passwordForm.resetFields()
  }

  // 提交重置密码
  const handleResetPassword = async () => {
    try {
      const values = await passwordForm.validateFields()
      if (selectedUserId) {
        await userApi.resetUserPassword(selectedUserId, { password: values.password })
        message.success('密码重置成功')
        closePasswordModal()
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败')
    }
  }

  // 表格列定义
  const columns: ColumnsType<UserInfo> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '用户名',
      dataIndex: 'username',
      width: 150,
    },
    {
      title: '姓名',
      dataIndex: 'full_name',
      width: 150,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      width: 200,
    },
    {
      title: '角色',
      dataIndex: 'roles',
      width: 200,
      render: (roles) => (
        <>
          {roles?.map((role) => (
            <Tag
              key={role.id}
              color={role.name === 'shared_viewer' ? 'purple' : 'blue'}
              style={{ marginBottom: 4 }}
            >
              {role.description || role.name}
            </Tag>
          ))}
        </>
      ),
    },
    {
      title: '状态',
      dataIndex: 'is_active',
      width: 100,
      render: (active, record) => (
        <Switch
          checked={active}
          onChange={() => handleToggleStatus(record.id)}
          disabled={record.id === 1}
          checkedChildren="启用"
          unCheckedChildren="禁用"
        />
      ),
    },
    {
      title: '最后登录',
      dataIndex: 'last_login_at',
      width: 180,
      render: (time) => time ? new Date(time).toLocaleString('zh-CN') : '-',
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
            icon={<EditOutlined />}
            onClick={() => openModal(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            size="small"
            icon={<LockOutlined />}
            onClick={() => openPasswordModal(record.id)}
          >
            重置密码
          </Button>
          <Popconfirm
            title="确定要删除这个用户吗？"
            onConfirm={() => handleDelete(record.id)}
            disabled={record.id === 1}
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={record.id === 1}
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
        <h2 style={{ margin: 0 }}>用户管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
          新建用户
        </Button>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <Space size="middle">
          <Input.Search
            placeholder="搜索用户名、邮箱或姓名"
            allowClear
            style={{ width: 300 }}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Select
            placeholder="选择角色"
            allowClear
            style={{ width: 150 }}
            onChange={handleRoleFilter}
            options={[
              { label: '管理员', value: 1 },
              { label: '操作员', value: 2 },
              { label: '查看者', value: 3 },
              { label: 'API客户端', value: 4 },
            ]}
          />
          <Select
            placeholder="选择状态"
            allowClear
            style={{ width: 120 }}
            onChange={handleStatusFilter}
            options={[
              { label: '启用', value: true },
              { label: '禁用', value: false },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={loadUsers}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={users}
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

      {/* 新建/编辑用户对话框 */}
      <Modal
        title={editingUser ? '编辑用户' : '新建用户'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={closeModal}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="username"
            label="用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, max: 50, message: '用户名长度为3-50个字符' },
            ]}
          >
            <Input placeholder="请输入用户名" disabled={!!editingUser} />
          </Form.Item>

          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[
                { required: true, message: '请输入密码' },
                { min: 8, message: '密码至少8个字符' },
              ]}
            >
              <Input.Password placeholder="请输入密码" />
            </Form.Item>
          )}

          <Form.Item
            name="email"
            label="邮箱"
            rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          <Form.Item
            name="full_name"
            label="姓名"
            rules={[{ max: 100, message: '姓名最多100个字符' }]}
          >
            <Input placeholder="请输入姓名" />
          </Form.Item>

          <Form.Item
            name="role_ids"
            label="角色"
            rules={[{ required: true, message: '请选择至少一个角色' }]}
          >
            <Select
              mode="multiple"
              placeholder="请选择角色（可多选）"
              options={[
                { label: '管理员', value: 1 },
                { label: '操作员', value: 2 },
                { label: '查看者', value: 3 },
                { label: 'API客户端', value: 4 },
                { label: '共享查看者', value: 5 },
              ]}
              tagRender={(props) => {
                const { label, value, ...restProps } = props
                // Special badge for shared_viewer
                if (value === 5) {
                  return <Tag {...restProps} color="purple">{label}</Tag>
                }
                return <Tag {...restProps}>{label}</Tag>
              }}
            />
          </Form.Item>

          <Form.Item
            name="is_active"
            label="状态"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码对话框 */}
      <Modal
        title="重置密码"
        open={passwordModalVisible}
        onOk={handleResetPassword}
        onCancel={closePasswordModal}
        destroyOnClose
      >
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            name="password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 8, message: '密码至少8个字符' },
            ]}
          >
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>

          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={['password']}
            rules={[
              { required: true, message: '请确认密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
