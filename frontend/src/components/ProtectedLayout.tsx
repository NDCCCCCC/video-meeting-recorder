import { lazy, Suspense } from 'react'
import { Spin } from 'antd'
import ProtectedRoute from './ProtectedRoute'

const BasicLayout = lazy(() => import('../layouts/BasicLayout'))

// 加载中组件 — 复用 App.tsx 同款 .app-loading 容器，
// 保证 Suspense fallback 渲染在视口中心而非父容器左上角。
const LoadingFallback = (
  <div className="app-loading">
    <Spin size="large" />
    <div className="app-loading-tip">加载中...</div>
  </div>
)

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
