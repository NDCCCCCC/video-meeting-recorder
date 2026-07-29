import { memo } from 'react'
import { useRoutes } from 'react-router-dom'
import { Spin } from 'antd'
import routes from './router'
import './styles/global.css'

// 提取加载组件为静态常量 (rendering-hoist-jsx)
const LoadingFallback = (
  <div className="app-loading">
    <Spin size="large" />
    <div className="app-loading-tip">加载中...</div>
  </div>
)

function App() {
  const element = useRoutes(routes)
  // 注意：页面切换的淡入淡出动画放在 BasicLayout 的内容区（包裹 <Outlet/>），
  // 而不是这里。若在这里用 AnimatePresence + key={pathname} 包整个 element，
  // 切换路由会让整棵路由树（含 header/sidebar）卸载重挂 —— 整页「闪一下」。
  return element || LoadingFallback
}

// 使用 memo 包裹根组件以减少不必要的重渲染
export default memo(App)