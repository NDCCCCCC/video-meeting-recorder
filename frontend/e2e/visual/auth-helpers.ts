import type { Page } from '@playwright/test'

// 测试用伪造用户（匹配 frontend/src/types/auth.ts 的 User 形状）
const AUTH_USER = {
  id: 1,
  username: 'admin',
  email: 'admin@e2e.test',
  full_name: '管理员',
  role_id: 1,
  role_name: '系统管理员',
  is_admin: true,
  permissions: ['*'],
  is_active: true,
}

/**
 * 注入 zustand persist 的 `auth-storage` localStorage，让 ProtectedRoute 同步守卫放行。
 *
 * 为什么 page.route mock 不够：ProtectedRoute.tsx 同步读 useAuthStore.isAuthenticated，
 * 浏览器初始 localStorage 为空 → persist 恢复出 isAuthenticated=false → 渲染 dashboard 前
 * 就 <Navigate to="/auth/login">，dashboard 内容从未渲染，page.route 的 API mock
 * 拦不到同步重定向。addInitScript 在 app bundle 之前注入 localStorage，persist 同步恢复
 * 出已登录态，守卫放行。
 */
export async function mockAuth(page: Page) {
  await page.addInitScript((user) => {
    window.localStorage.setItem(
      'auth-storage',
      JSON.stringify({
        state: {
          user,
          token: 'e2e-fake-access-token',
          refreshToken: 'e2e-fake-refresh-token',
          isAuthenticated: true,
        },
        version: 0,
      })
    )
  }, AUTH_USER)
}
