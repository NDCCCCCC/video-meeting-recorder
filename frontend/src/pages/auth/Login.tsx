// 登录页面 — Phase 16 D-02 重做
import { App, Form, Input, Button } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'
import { useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../../stores/authStore'
import type { LoginRequest } from '../../types/auth'
import LoginBackground from './components/LoginBackground'
import './Login.css'

export default function Login() {
  const { message } = App.useApp()
  const navigate = useNavigate()
  const location = useLocation()
  const login = useAuthStore((state) => state.login)

  // 获取登录前的跳转路径
  const from = (location.state as { from?: { pathname?: string } } | null)?.from?.pathname || '/'

  const onFinish = async (values: LoginRequest) => {
    try {
      await login(values)
      message.success('登录成功')
      navigate(from, { replace: true })
    } catch (error) {
      message.error(error instanceof Error ? error.message : '登录失败')
    }
  }

  return (
    <LoginBackground>
      <div className="login-card">
        {/* 品牌色 logo — SVG 而非图标 */}
        <div className="login-logo">
          <svg viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg">
            <rect x="8" y="8" width="48" height="48" rx="10" fill="#0F766E" />
            <path
              d="M22 24 L22 40 M22 32 L34 32 M34 24 L34 40"
              stroke="white"
              strokeWidth="3"
              strokeLinecap="round"
            />
            <circle cx="44" cy="24" r="4" fill="#5EEAD4" />
          </svg>
        </div>

        <div className="login-header">
          <h1>录播服务系统</h1>
          <p>视频会议录制管理平台</p>
        </div>

        <Form
          name="login"
          onFinish={onFinish}
          autoComplete="off"
          size="large"
          initialValues={
            import.meta.env.DEV ? { username: 'admin', password: 'admin123' } : undefined
          }
        >
          <Form.Item name="username" rules={[{ required: true, message: '请输入用户名' }]}>
            <Input prefix={<UserOutlined />} placeholder="用户名" />
          </Form.Item>

          <Form.Item name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password prefix={<LockOutlined />} placeholder="密码" />
          </Form.Item>

          <Form.Item>
            <Button type="primary" htmlType="submit" block>
              登录
            </Button>
          </Form.Item>
        </Form>

        {import.meta.env.DEV && (
          <div className="login-footer">
            <p>默认账号: admin / admin123</p>
          </div>
        )}
      </div>
    </LoginBackground>
  )
}