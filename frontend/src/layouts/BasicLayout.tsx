// 基础布局

import { Outlet, useNavigate } from 'react-router-dom'
import { Layout, Menu, Dropdown, Avatar } from 'antd'
import {
  HomeOutlined,
  VideoCameraOutlined,
  UserOutlined,
  SettingOutlined,
  LogoutOutlined,
  FileTextOutlined,
  FolderOutlined,
  HistoryOutlined,
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

  const menuItems: MenuProps['items'] = [
    { key: '/', icon: <HomeOutlined />, label: '仪表盘' },
    { key: '/tasks', icon: <VideoCameraOutlined />, label: '录制任务' },
    { key: '/conferences', icon: <FileTextOutlined />, label: '会议记录' },
    { key: '/files', icon: <FolderOutlined />, label: '文件管理' },
    { key: '/audit', icon: <HistoryOutlined />, label: '审计日志' },
    {
      key: 'system',
      icon: <SettingOutlined />,
      label: '系统管理',
      children: [
        { key: '/system/users', label: '用户管理' },
        { key: '/system/roles', label: '角色管理' },
        { key: '/system/huawei-configs', icon: <CloudServerOutlined />, label: '华为配置' },
        { key: '/system/settings', label: '系统设置' },
      ],
    },
  ]

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
