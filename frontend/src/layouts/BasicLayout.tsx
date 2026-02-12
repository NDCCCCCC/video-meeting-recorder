// 基础布局

import { Outlet, useNavigate } from 'react-router-dom'
import { Layout, Menu, Dropdown, Avatar } from 'antd'
import {
  VideoCameraOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  FolderOutlined,
  CloudServerOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'
import type { MenuProps } from 'antd'
import './BasicLayout.css'

const { Header, Sider, Content } = Layout

export default function BasicLayout() {
  const navigate = useNavigate()
  const logout = useAuthStore((state) => state.logout)
  const user = useAuthStore((state) => state.user)

  const handleLogout = async () => {
    await logout()
    navigate('/auth/login')
  }

  // 检查菜单权限的辅助函数
  const canAccessPath = (path: string | undefined): boolean => {
    if (!path) return true
    const MENU_PERMISSIONS: Record<string, string> = {
      '/tasks': 'tasks:view',
      '/files': 'files:view',
      '/system/users': 'users:view',
      '/system/roles': 'roles:view',
      '/system/huawei-configs': 'configs:view',
    }
    const required = MENU_PERMISSIONS[path]
    if (!required) return true
    if (!user) return false
    if (user.is_admin) return true
    return user.permissions?.includes(required) ?? false
  }

  // 构建过滤后的菜单项
  const menuItems: MenuProps['items'] = [
    // { key: '/', icon: <HomeOutlined />, label: '仪表盘' },
    canAccessPath('/tasks') ? { key: '/tasks', icon: <VideoCameraOutlined />, label: '录制任务' } : null,
    canAccessPath('/files') ? { key: '/files', icon: <FolderOutlined />, label: '文件管理' } : null,
    // { key: '/audit', icon: <HistoryOutlined />, label: '审计日志' },
    canAccessPath('/system/users') || canAccessPath('/system/roles') || canAccessPath('/system/huawei-configs') ? {
      key: 'system',
      icon: <SettingOutlined />,
      label: '系统管理',
      children: [
        canAccessPath('/system/users') ? { key: '/system/users', label: '用户管理' } : null,
        canAccessPath('/system/roles') ? { key: '/system/roles', label: '角色管理' } : null,
        canAccessPath('/system/huawei-configs') ? { key: '/system/huawei-configs', icon: <CloudServerOutlined />, label: '华为配置' } : null,
        // { key: '/system/settings', label: '系统设置' },
      ].filter((item): item is NonNullable<typeof item> => item !== null),
    } : null,
  ].filter((item): item is NonNullable<typeof item> => item !== null)

  const userMenuItems: MenuProps['items'] = [
    { key: 'profile', icon: <UserOutlined />, label: '个人信息' },
    { type: 'divider' },
    { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: handleLogout },
  ]

  return (
    <Layout className="basic-layout">
      <Sider width={240} theme="light">
        <div className="logo">
          <h2>录制管理系统</h2>
        </div>
        <Menu
          mode="inline"
          items={menuItems}
          selectedKeys={[location.pathname]}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header className="layout-header">
          <div className="header-right">
            <Dropdown menu={{ items: userMenuItems }} placement="bottomRight">
              <Avatar icon={<UserOutlined />} style={{ cursor: 'pointer' }} />
            </Dropdown>
            <span className="user-name">{user?.full_name || user?.username}</span>
          </div>
        </Header>
        <Content className="layout-content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}
