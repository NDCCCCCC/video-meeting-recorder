import { lazy } from 'react'
import ProtectedLayout from '../components/ProtectedLayout'

// 路由配置
export default [
  // 公开路由
  {
    path: '/auth/login',
    Component: lazy(() => import('../pages/auth/Login')),
  },
  // 主应用路由（需要认证）
  {
    path: '/',
    Component: ProtectedLayout,
    children: [
      {
        index: true,
        Component: lazy(() => import('../pages/dashboard')),
      },
      {
        path: 'tasks',
        Component: lazy(() => import('../pages/tasks')),
      },
      {
        path: 'conferences',
        Component: lazy(() => import('../pages/conferences')),
      },
      {
        path: 'files',
        Component: lazy(() => import('../pages/files')),
      },
      {
        path: 'audit',
        Component: lazy(() => import('../pages/audit')),
      },
      {
        path: 'system',
        children: [
          { path: 'users', Component: lazy(() => import('../pages/system/users')) },
          { path: 'roles', Component: lazy(() => import('../pages/system/roles')) },
          { path: 'huawei-configs', Component: lazy(() => import('../pages/system/huawei-configs')) },
          { path: 'settings', Component: lazy(() => import('../pages/system/settings')) },
        ],
      },
    ],
  },
  // 404页面
  {
    path: '*',
    Component: lazy(() => import('../pages/error/NotFound')),
  },
]
