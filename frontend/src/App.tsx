import { Suspense } from 'react'
import { useRoutes } from 'react-router-dom'
import { Spin } from 'antd'
import routes from './router'
import './styles/global.css'

function App() {
  const element = useRoutes(routes)

  return (
    <Suspense fallback={
      <div style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '100vh'
      }}>
        <Spin size="large" tip="加载中..." />
      </div>
    }>
      {element}
    </Suspense>
  )
}

export default App
