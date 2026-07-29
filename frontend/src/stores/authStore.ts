// 认证状态管理

import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { AuthState, LoginRequest, ChangePasswordRequest } from '../types/auth'
import * as authApi from '../api/auth'

interface AuthStore extends AuthState {
  // Actions
  login: (req: LoginRequest) => Promise<void>
  logout: () => Promise<void>
  logoutAll: () => Promise<void>
  refreshUser: () => Promise<void>
  changePassword: (req: ChangePasswordRequest) => Promise<void>
  setToken: (token: string, refreshToken: string) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set) => ({
      // Initial state
      user: null,
      token: null,
      refreshToken: null,
      isAuthenticated: false,

      // 登录
      login: async (req: LoginRequest) => {
        const response = await authApi.login(req)
        if (response.data) {
          set({
            user: response.data.user,
            token: response.data.access_token,
            refreshToken: response.data.refresh_token,
            isAuthenticated: true,
          })
        }
      },

      // 登出
      logout: async () => {
        await authApi.logout()
        set({
          user: null,
          token: null,
          refreshToken: null,
          isAuthenticated: false,
        })
      },

      // 登出所有设备
      logoutAll: async () => {
        await authApi.logoutAll()
        set({
          user: null,
          token: null,
          refreshToken: null,
          isAuthenticated: false,
        })
      },

      // 刷新用户信息
      refreshUser: async () => {
        const response = await authApi.getCurrentUser()
        if (response.data) {
          set({ user: response.data })
        }
      },

      // 修改密码
      changePassword: async (req: ChangePasswordRequest) => {
        await authApi.changePassword(req)
      },

      // 设置 Token
      setToken: (token: string, refreshToken: string) => {
        set({ token, refreshToken, isAuthenticated: true })
      },

      // 清除认证信息
      clearAuth: () => {
        set({
          user: null,
          token: null,
          refreshToken: null,
          isAuthenticated: false,
        })
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        refreshToken: state.refreshToken,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)
