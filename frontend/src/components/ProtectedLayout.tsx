import { lazy, Suspense } from 'react'
import { Spin } from 'antd'
import ProtectedRoute from './ProtectedRoute'

const BasicLayout = lazy(() => import('../layouts/BasicLayout'))

// 加载中组件
const LoadingFallback = <Spin size="large" />

// 受保护的路由包装组件
export default function ProtectedLayout() {
  return (
    <ProtectedRoute>
      <Suspense fallback={LoadingFallback}>
        <BasicLayout />
      </Suspense>
    </ProtectedRoute>
  )
}
