// 基础布局

import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useCallback, useMemo, memo } from 'react'
import { Layout, Menu, Dropdown, Avatar } from 'antd'
import type { MenuProps } from 'antd'
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
  SafetyCertificateOutlined,
  AuditOutlined,
  KeyOutlined,
} from '@ant-design/icons'
import { useAuthStore } from '../stores/authStore'
import type { User } from '../types/auth'
import { MENU_PERMISSIONS } from '../utils/permissions'
import './BasicLayout.css'

const { Header, Sider, Content } = Layout

// 检查菜单权限 - 提取为纯函数便于测试
function canAccessPath(path: string | undefined, user: User | null): boolean {
  if (!path) return true
  const required = MENU_PERMISSIONS[path]
  if (!required) return true
  if (!user) return false
  if (user.is_admin) return true
  return user.permissions?.includes(required) ?? false
}

function BasicLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const logout = useAuthStore((state) => state.logout)
  const user = useAuthStore((state) => state.user)

  // 使用 useCallback 缓存事件处理函数
  // try/finally 兜底：无论 logout() 是否抛错（后端登出接口失败等），
  // 都必须跳转到登录页，避免"点了没反应"。
  const handleLogout = useCallback(async () => {
    try {
      await logout()
    } finally {
      navigate('/auth/login')
    }
  }, [logout, navigate])

  // 使用 useCallback 缓存菜单点击处理
  const handleMenuClick = useCallback(
    ({ key }: { key: string }) => {
      navigate(key)
    },
    [navigate]
  )

  // 用户下拉菜单点击处理
  // NOTE: 不能依赖 items 数组里单项的 onClick（antd Dropdown menu 上下文下不可靠触发），
  // 统一在菜单级别 onClick 里按 key 分发。见 Phase 16 修复。
  const handleUserMenuClick = useCallback(
    ({ key }: { key: string }) => {
      console.log('[AUTHDBG] menu onClick fired, key=', key)
      if (key === 'logout') {
        // 登出走菜单级别处理，而非 item onClick
        void handleLogout()
        return
      }
      navigate(key)
    },
    [handleLogout, navigate]
  )

  // 使用 useMemo 缓存菜单项计算
  const menuItems: MenuProps['items'] = useMemo(() => {
    const hasSystemAccess =
      canAccessPath('/system/users', user) ||
      canAccessPath('/system/roles', user) ||
      canAccessPath('/system/apikeys', user) ||
      canAccessPath('/system/input-configs', user) ||
      canAccessPath('/system/auth-config', user) ||
      canAccessPath('/system/settings', user)

    const items: MenuProps['items'] = [
      canAccessPath('/dashboard', user)
        ? { key: '/dashboard', icon: <DashboardOutlined />, label: '仪表盘' }
        : null,
      canAccessPath('/tasks', user)
        ? { key: '/tasks', icon: <VideoCameraOutlined />, label: '录制任务' }
        : null,
      canAccessPath('/files', user)
        ? { key: '/files', icon: <FolderOutlined />, label: '文件管理' }
        : null,
      canAccessPath('/audit', user)
        ? { key: '/audit', icon: <AuditOutlined />, label: '审计日志' }
        : null,
      hasSystemAccess
        ? {
            key: 'system',
            icon: <SettingOutlined />,
            label: '系统管理',
            children: [
              canAccessPath('/system/users', user)
                ? { key: '/system/users', icon: <TeamOutlined />, label: '用户管理' }
                : null,
              canAccessPath('/system/roles', user)
                ? { key: '/system/roles', icon: <SafetyOutlined />, label: '角色管理' }
                : null,
              canAccessPath('/system/apikeys', user)
                ? { key: '/system/apikeys', icon: <KeyOutlined />, label: 'API密钥' }
                : null,
              canAccessPath('/system/input-configs', user)
                ? { key: '/system/input-configs', icon: <CloudServerOutlined />, label: '输入配置' }
                : null,
              canAccessPath('/system/auth-config', user)
                ? {
                    key: '/system/auth-config',
                    icon: <SafetyCertificateOutlined />,
                    label: '认证管理',
                  }
                : null,
              canAccessPath('/system/settings', user)
                ? { key: '/system/settings', icon: <SettingOutlined />, label: '系统设置' }
                : null,
            ].filter((item): item is NonNullable<typeof item> => item !== null),
          }
        : null,
    ].filter((item): item is NonNullable<typeof item> => item !== null)

    return items
  }, [user])

  // 用户菜单项 - 使用 useMemo 避免每次渲染重新创建
  const userMenuItems: MenuProps['items'] = useMemo(
    () => [
      { key: '/system/settings', icon: <SettingOutlined />, label: '系统设置' },
      { type: 'divider' as const },
      { key: 'logout', icon: <LogoutOutlined />, label: '退出登录' },
    ],
    []
  )

  // 用户名显示 - 使用 useMemo 缓存计算
  const displayName = useMemo(() => {
    return user?.full_name || user?.username || '用户'
  }, [user?.full_name, user?.username])

  // 默认展开的菜单项 - 当访问系统相关页面时自动展开系统管理菜单
  const defaultOpenKeys = useMemo(() => {
    if (location.pathname.startsWith('/system')) return ['system']
    return []
  }, [location.pathname])

  return (
    <Layout className="basic-layout">
      <Sider width={240} theme="light">
        <div className="logo">
          <h2>录播服务系统</h2>
        </div>
        <Menu
          mode="inline"
          items={menuItems}
          selectedKeys={[location.pathname]}
          defaultOpenKeys={defaultOpenKeys}
          onClick={handleMenuClick}
        />
      </Sider>
      <Layout>
        <Header className="layout-header">
          <div className="header-right">
            <Dropdown
              menu={{ items: userMenuItems, onClick: handleUserMenuClick }}
              trigger={['click']}
              placement="bottomRight"
            >
              <Avatar icon={<UserOutlined />} style={{ cursor: 'pointer' }} />
            </Dropdown>
            <span className="user-name">{displayName}</span>
          </div>
        </Header>
        <Content className="layout-content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

// 使用 memo 优化，仅在 props 或 context 变化时重渲染
export default memo(BasicLayout)
