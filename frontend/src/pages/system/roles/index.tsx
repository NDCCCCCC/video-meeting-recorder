// 角色管理页面

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
  Transfer,
  Tag,
  Descriptions
} from 'antd'
import {
  PlusOutlined,
  SearchOutlined,
  EditOutlined,
  DeleteOutlined,
  KeyOutlined,
  ReloadOutlined,
  SafetyOutlined
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { TransferProps } from 'antd/es/transfer'
import * as roleApi from '../../../api/role'
import type { RoleInfo, RoleListParams, CreateRoleRequest, UpdateRoleRequest, Permission } from '../../../types/role'

// 解析 allowed_ips 字段（可能是 JSON 字符串或数组）
const parseAllowedIPs = (ips: any): string[] => {
  if (!ips) return []
  if (Array.isArray(ips)) return ips
  if (typeof ips === 'string') {
    try {
      return JSON.parse(ips)
    } catch {
      return []
    }
  }
  return []
}

export default function RoleManagement() {
  const [roles, setRoles] = useState<RoleInfo[]>([])
  const [allPermissions, setAllPermissions] = useState<Permission[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [permissionModalVisible, setPermissionModalVisible] = useState(false)
  const [editingRole, setEditingRole] = useState<RoleInfo | null>(null)
  const [selectedRoleId, setSelectedRoleId] = useState<number | null>(null)
  const [selectedPermissionIds, setSelectedPermissionIds] = useState<number[]>([])
  const [form] = Form.useForm()

  // 查询参数
  const [params, setParams] = useState<RoleListParams>({
    page: 1,
    page_size: 20,
  })

  // 加载角色列表
  const loadRoles = async () => {
    setLoading(true)
    try {
      const response = await roleApi.getRoleList(params)
      if (response.data) {
        setRoles(response.data.items)
        setTotal(response.data.total)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载角色列表失败')
    } finally {
      setLoading(false)
    }
  }

  // 加载所有权限
  const loadAllPermissions = async () => {
    try {
      const response = await roleApi.getAllPermissions()
      if (response.data) {
        setAllPermissions(response.data)
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载权限列表失败')
    }
  }

  useEffect(() => {
    loadRoles()
  }, [params])

  // 搜索
  const handleSearch = (value: string) => {
    setParams({ ...params, keyword: value, page: 1 })
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
  const openModal = (role: RoleInfo | null = null) => {
    setEditingRole(role)
    if (role) {
      form.setFieldsValue({
        name: role.name,
        description: role.description,
        allowed_ips: parseAllowedIPs(role.allowed_ips),
      })
    } else {
      form.resetFields()
    }
    setModalVisible(true)
  }

  // 关闭对话框
  const closeModal = () => {
    setModalVisible(false)
    setEditingRole(null)
    form.resetFields()
  }

  // 提交表单
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()

      if (editingRole) {
        // 更新角色
        const req: UpdateRoleRequest = {
          description: values.description,
          allowed_ips: values.allowed_ips,
        }
        await roleApi.updateRole(editingRole.id, req)
        message.success('更新成功')
      } else {
        // 创建角色
        const req: CreateRoleRequest = {
          name: values.name,
          description: values.description,
          allowed_ips: values.allowed_ips || [],
        }
        await roleApi.createRole(req)
        message.success('创建成功')
      }

      closeModal()
      loadRoles()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败')
    }
  }

  // 删除角色
  const handleDelete = async (id: number) => {
    try {
      await roleApi.deleteRole(id)
      message.success('删除成功')
      loadRoles()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '删除失败')
    }
  }

  // 打开权限管理对话框
  const openPermissionModal = async (role: RoleInfo) => {
    setSelectedRoleId(role.id)
    setEditingRole(role)

    // 加载角色当前权限
    try {
      const response = await roleApi.getRolePermissions(role.id)
      if (response.data) {
        setSelectedPermissionIds(response.data.map(p => p.id))
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '加载角色权限失败')
    }

    // 确保所有权限已加载
    if (allPermissions.length === 0) {
      await loadAllPermissions()
    }

    setPermissionModalVisible(true)
  }

  // 关闭权限管理对话框
  const closePermissionModal = () => {
    setPermissionModalVisible(false)
    setSelectedRoleId(null)
    setEditingRole(null)
    setSelectedPermissionIds([])
  }

  // 权限穿梭框变化
  const handlePermissionChange: TransferProps['onChange'] = (newTargetKeys) => {
    setSelectedPermissionIds(newTargetKeys.map(id => Number(id)))
  }

  // 保存权限分配
  const handleSavePermissions = async () => {
    if (!selectedRoleId) return

    try {
      await roleApi.assignPermissions(selectedRoleId, {
        permission_ids: selectedPermissionIds,
      })
      message.success('权限分配成功')
      closePermissionModal()
      loadRoles()
    } catch (error) {
      message.error(error instanceof Error ? error.message : '操作失败')
    }
  }

  // 表格列定义
  const columns: ColumnsType<RoleInfo> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 80,
    },
    {
      title: '角色名称',
      dataIndex: 'name',
      width: 150,
      render: (name) => <Tag color="blue">{name}</Tag>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      width: 250,
      render: (desc) => desc || '-',
    },
    {
      title: '权限数量',
      dataIndex: 'permissions',
      width: 120,
      render: (permissions) => permissions?.length || 0,
    },
    {
      title: '权限列表',
      dataIndex: 'permissions',
      width: 400,
      render: (permissions: Permission[] | undefined) => {
        if (!permissions || permissions.length === 0) return '-'
        return (
          <Space size={[4, 4]} wrap>
            {permissions.map((p: Permission) => (
              <Tag key={p.id} color="geekblue" style={{ margin: 0 }}>
                {p.resource}:{p.action}
              </Tag>
            ))}
          </Space>
        )
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 250,
      fixed: 'right' as const,
      render: (_, record) => (
        <Space size="small">
          <Button
            type="link"
            size="small"
            icon={<KeyOutlined />}
            onClick={() => openPermissionModal(record)}
          >
            权限
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
            title="确定要删除这个角色吗？"
            onConfirm={() => handleDelete(record.id)}
            disabled={record.id <= 4}
          >
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={record.id <= 4}
            >
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  // 权限穿梭框数据源
  const transferDataSource = allPermissions.map(p => ({
    key: p.id.toString(),
    title: `${p.resource}:${p.action} - ${p.description}`,
    description: p.description,
  }))

  return (
    <div style={{ padding: '20px' }}>
      <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2 style={{ margin: 0 }}>角色管理</h2>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
          新建角色
        </Button>
      </div>

      <div style={{ marginBottom: '16px' }}>
        <Space size="middle">
          <Input.Search
            placeholder="搜索角色名称或描述"
            allowClear
            style={{ width: 300 }}
            onSearch={handleSearch}
            enterButton={<SearchOutlined />}
          />
          <Button icon={<ReloadOutlined />} onClick={loadRoles}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={roles}
        rowKey="id"
        loading={loading}
        scroll={{ x: 1400 }}
        pagination={{
          current: params.page,
          pageSize: params.page_size,
          total,
          showSizeChanger: true,
          showTotal: (t) => `共 ${t} 条`,
        }}
        onChange={handleTableChange}
      />

      {/* 新建/编辑角色对话框 */}
      <Modal
        title={editingRole ? '编辑角色' : '新建角色'}
        open={modalVisible}
        onOk={handleSubmit}
        onCancel={closeModal}
        width={600}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="角色名称"
            rules={[
              { required: true, message: '请输入角色名称' },
              { min: 2, max: 50, message: '角色名称长度为2-50个字符' },
            ]}
          >
            <Input placeholder="请输入角色名称" disabled={!!editingRole} />
          </Form.Item>

          <Form.Item
            name="description"
            label="描述"
            rules={[{ max: 200, message: '描述最多200个字符' }]}
          >
            <Input.TextArea placeholder="请输入角色描述" rows={4} />
          </Form.Item>

          <Form.Item
            name="allowed_ips"
            label="IP地址限制"
            extra="每行一个IP地址，支持格式：192.168.1.100 或 192.168.1.0/24 或 192.168.1.100-192.168.1.200"
          >
            <Input.TextArea
              placeholder="例如：&#10;192.168.1.100&#10;192.168.1.0/24&#10;192.168.1.100-192.168.1.200"
              rows={4}
              onChange={(e) => {
                // Convert textarea lines to array
                const lines = e.target.value.split('\n')
                  .map(line => line.trim())
                  .filter(line => line.length > 0)
                form.setFieldsValue({ allowed_ips: lines })
              }}
            />
          </Form.Item>

          {editingRole && (
            <Form.Item label="当前权限">
              <div style={{ maxHeight: '150px', overflow: 'auto', padding: '8px', background: '#f5f5f5', borderRadius: '4px' }}>
                {editingRole.permissions && editingRole.permissions.length > 0 ? (
                  <Space size={[4, 4]} wrap>
                    {editingRole.permissions.map(p => (
                      <Tag key={p.id} color="geekblue">
                        {p.resource}:{p.action}
                      </Tag>
                    ))}
                  </Space>
                ) : (
                  <span style={{ color: '#999' }}>暂无权限</span>
                )}
              </div>
            </Form.Item>
          )}
        </Form>
      </Modal>

      {/* 权限管理对话框 */}
      <Modal
        title={<Space><SafetyOutlined />分配权限 - {editingRole?.name}</Space>}
        open={permissionModalVisible}
        onOk={handleSavePermissions}
        onCancel={closePermissionModal}
        width={800}
        destroyOnClose
      >
        <div style={{ marginBottom: '16px' }}>
          <Descriptions size="small" column={2}>
            <Descriptions.Item label="角色ID">{selectedRoleId}</Descriptions.Item>
            <Descriptions.Item label="角色名称">{editingRole?.name}</Descriptions.Item>
            <Descriptions.Item label="当前权限数">
              {selectedPermissionIds.length}
            </Descriptions.Item>
            <Descriptions.Item label="可用权限数">
              {allPermissions.length}
            </Descriptions.Item>
          </Descriptions>
        </div>

        <Transfer
          dataSource={transferDataSource}
          titles={['可用权限', '已分配权限']}
          targetKeys={selectedPermissionIds.map(id => id.toString())}
          onChange={handlePermissionChange}
          render={item => item.title}
          listStyle={{
            width: 320,
            height: 400,
          }}
          showSearch
          filterOption={(inputValue, item) =>
            item.title?.toLowerCase().includes(inputValue.toLowerCase()) ||
            item.description?.toLowerCase().includes(inputValue.toLowerCase())
          }
        />
      </Modal>
    </div>
  )
}
