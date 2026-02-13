// 基础布局

import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu, Dropdown, Avatar } from 'antd'
import {
  DashboardOutlined,
  VideoCameraOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  FolderOutlined,
  CloudServerOutlined,
  TeamOutlined,
  SafetyOutlined,
  AuditOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'
import type { MenuProps } from 'antd'
import './BasicLayout.css'

const { Header, Sider, Content } = Layout

// 菜单权限映射
const MENU_PERMISSIONS: Record<string, string> = {
  '/dashboard': 'dashboard:view',
  '/tasks': 'tasks:view',
  '/files': 'files:view',
  '/audit': 'audit:view',
  '/system/users': 'users:view',
  '/system/roles': 'roles:view',
  '/system/huawei-configs': 'configs:view',
  '/system/settings': 'system:settings',
}

// 检查菜单权限
function canAccessPath(path: string | undefined, user: any): boolean {
  if (!path) return true
  const required = MENU_PERMISSIONS[path]
  if (!required) return true
  if (!user) return false
  if (user.is_admin) return true
  return user.permissions?.includes(required) ?? false
}

export default function BasicLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const logout = useAuthStore((state) => state.logout)
  const user = useAuthStore((state) => state.user)

  const handleLogout = async () => {
    await logout()
    navigate('/auth/login')
  }

  // 构建过滤后的菜单项
  const menuItems: MenuProps['items'] = [
    canAccessPath('/dashboard', user) ? { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' } : null,
    canAccessPath('/tasks', user) ? { key: '/tasks', icon: <VideoCameraOutlined />, label: '录制任务' } : null,
    canAccessPath('/files', user) ? { key: '/files', icon: <FolderOutlined />, label: '文件管理' } : null,
    canAccessPath('/audit', user) ? { key: '/audit', icon: <AuditOutlined />, label: '审计日志' } : null,
    canAccessPath('/system/users', user) || canAccessPath('/system/roles', user) || canAccessPath('/system/huawei-configs', user) || canAccessPath('/system/settings', user) ? {
      key: 'system',
      icon: <SettingOutlined />,
      label: '系统管理',
      children: [
        canAccessPath('/system/users', user) ? { key: '/system/users', icon: <TeamOutlined />, label: '用户管理' } : null,
        canAccessPath('/system/roles', user) ? { key: '/system/roles', icon: <SafetyOutlined />, label: '角色管理' } : null,
        canAccessPath('/system/huawei-configs', user) ? { key: '/system/huawei-configs', icon: <CloudServerOutlined />, label: '华为配置' } : null,
        canAccessPath('/system/settings', user) ? { key: '/system/settings', icon: <SettingOutlined />, label: '系统设置' } : null,
      ].filter((item): item is NonNullable<typeof item> => item !== null),
    } : null,
  ].filter((item): item is NonNullable<typeof item> => item !== null)

  const userMenuItems: MenuProps['items'] = [
    { key: 'profile', icon: <UserOutlined />, label: '个人信息' },
    { type: 'divider' },
    { key: 'settings', icon: <SettingOutlined />, label: '系统设置' },
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
          onClick={({ key }) => navigate(key as string)}
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
