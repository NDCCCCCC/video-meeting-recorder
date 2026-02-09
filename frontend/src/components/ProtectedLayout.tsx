import { lazy } from 'react'
import ProtectedRoute from './ProtectedRoute'
const BasicLayout = lazy(() => import('../layouts/BasicLayout'))

// 受保护的路由包装组件
export default function ProtectedLayout() {
  return (
    <ProtectedRoute>
      <BasicLayout />
    </ProtectedRoute>
  )
}
