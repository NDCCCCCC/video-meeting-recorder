import { memo } from 'react'
import { useRoutes, useLocation } from 'react-router-dom'
import { Spin } from 'antd'
import { AnimatePresence, m } from 'framer-motion'
import routes from './router'
import { routeFade } from './motion/motionConfig'
import './styles/global.css'

// 提取加载组件为静态常量 (rendering-hoist-jsx)
const LoadingFallback = (
  <div className="app-loading">
    <Spin size="large" tip="加载中..." />
  </div>
)

function App() {
  const element = useRoutes(routes)
  const location = useLocation()
  return (
    <AnimatePresence mode="wait">
      <m.div
        key={location.pathname}
        variants={routeFade}
        initial="initial"
        animate="animate"
        exit="exit"
      >
        {element || LoadingFallback}
      </m.div>
    </AnimatePresence>
  )
}

// 使用 memo 包裹根组件以减少不必要的重渲染
export default memo(App)
