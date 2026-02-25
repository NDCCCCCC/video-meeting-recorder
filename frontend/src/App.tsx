import { memo } from 'react'
import { useRoutes } from 'react-router-dom'
import { Spin } from 'antd'
import routes from './router'
import './styles/global.css'

// 提取加载组件为静态常量 (rendering-hoist-jsx)
const LoadingFallback = (
  <div className="app-loading">
    <Spin size="large" tip="加载中..." />
  </div>
)

function App() {
  const element = useRoutes(routes)
  return (
    <div>
      {element || LoadingFallback}
    </div>
  )
}

// 使用 memo 包裹根组件以减少不必要的重渲染
export default memo(App)
