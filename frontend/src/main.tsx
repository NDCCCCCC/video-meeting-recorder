import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { designTokens } from './styles/theme'
import App from './App'
import './styles/global.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <ConfigProvider
        locale={zhCN}
        theme={{
          token: {
            colorPrimary: designTokens.colors.primary,
            colorSuccess: designTokens.colors.success,
            colorWarning: designTokens.colors.warning,
            colorError: designTokens.colors.error,
            colorText: designTokens.colors.text.primary,
            colorTextSecondary: designTokens.colors.text.secondary,
            colorTextDisabled: designTokens.colors.text.disabled,
            borderRadius: designTokens.borderRadius,
            fontSize: designTokens.fontSize.base,
            marginXS: designTokens.spacing.xs,
            marginSM: designTokens.spacing.sm,
            margin: designTokens.spacing.md,
            marginLG: designTokens.spacing.lg,
            marginXL: designTokens.spacing.xl,
          },
        }}
      >
        <App />
      </ConfigProvider>
    </BrowserRouter>
  </StrictMode>,
)
