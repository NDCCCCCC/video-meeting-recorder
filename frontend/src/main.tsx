import { StrictMode, lazy, Suspense } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { App as AntdApp, ConfigProvider } from 'antd'
import { designTokens } from './styles/theme'
import MotionProvider from './motion/MotionProvider'
import App from './App'
import './styles/global.css'
import './styles/performance.css'

// bundle-defer-third-party: 延迟加载 antd locale 以减少初始包大小
const AntdConfigProvider = lazy(async () => {
  const { default: zhCN } = await import('antd/locale/zh_CN')
  return {
    default: ({ children }: { children: React.ReactNode }) => (
      <ConfigProvider
        locale={zhCN}
        theme={{
          token: {
            colorPrimary: designTokens.colors.accent,
            colorSuccess: designTokens.colors.success,
            colorWarning: designTokens.colors.warning,
            colorError: designTokens.colors.error,
            colorText: designTokens.colors.text.primary,
            colorTextSecondary: designTokens.colors.text.secondary,
            colorTextDisabled: designTokens.colors.text.disabled,
            colorBgContainer: '#ffffff',
            colorBorder: designTokens.colors.border,
            // antd 6 flat motion tokens (NOT nested under motion — verified seeds.d.ts:163-210)
            // NOTE: motionUnit 单位是「秒」(seed.js 默认 0.1)，不是 ms！
            // seeds.d.ts 的 @default 100ms 文档注释具误导性 —— 实际计算为
            // motionDurationMid = `${(motionBase + motionUnit * 2).toFixed(1)}s` (genCommonMapToken.js)。
            // 之前误传 100 导致动画时长 = 200s（应为 0.2s），下拉菜单等动画看起来「卡住」。
            motion: true,
            motionUnit: 0.1,
            motionEaseOut: designTokens.motion.easing.standard,
            motionEaseInOut: designTokens.motion.easing.standard,
            boxShadow: designTokens.elevation.sm,
            boxShadowSecondary: designTokens.elevation.md,
            borderRadius: designTokens.borderRadius.base,
            fontSize: designTokens.fontSize.base,
            marginXS: designTokens.spacing.xs,
            marginSM: designTokens.spacing.sm,
            margin: designTokens.spacing.md,
            marginLG: designTokens.spacing.lg,
            marginXL: designTokens.spacing.xl,
          },
        }}
      >
        <AntdApp>{children}</AntdApp>
      </ConfigProvider>
    ),
  }
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <Suspense
        fallback={
          <div
            style={{
              display: 'flex',
              justifyContent: 'center',
              alignItems: 'center',
              height: '100vh',
            }}
          >
            加载中...
          </div>
        }
      >
        <AntdConfigProvider>
          <MotionProvider>
            <App />
          </MotionProvider>
        </AntdConfigProvider>
      </Suspense>
    </BrowserRouter>
  </StrictMode>
)
