import { test, expect } from '@playwright/test'
import { mockAuth } from './auth-helpers'

const mockStats = {
  data: {
    task_stats: { total: 12, in_progress: 3, success: 7, fail: 2, avg_time: 24.5 },
    file_stats: { total_videos: 48, storage_mb: 12288, transcripts: 32, ppts: 18 },
    system_stats: {
      disk_usage_percent: 42.0,
      memory_usage_percent: 58.5,
      error_count: 4,
      api_calls: 1240,
    },
  },
  message: 'ok',
}

const emptyStats = {
  data: {
    task_stats: { total: 0, in_progress: 0, success: 0, fail: 0, avg_time: 0 },
    file_stats: { total_videos: 0, storage_mb: 0, transcripts: 0, ppts: 0 },
    system_stats: {
      disk_usage_percent: 0,
      memory_usage_percent: 0,
      error_count: 0,
      api_calls: 0,
    },
  },
  message: 'ok',
}

const authMe = {
  data: { id: 1, username: 'admin', is_admin: true },
  message: 'ok',
}

test.describe('仪表板', () => {
  test('有数据状态', async ({ page }) => {
    await page.route('/api/v1/dashboard/stats', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockStats),
      })
    )
    // Mock auth check so we land on dashboard, not login redirect (R-1 mitigation)
    await page.route('/api/v1/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(authMe),
      })
    )
    await mockAuth(page)
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('dashboard-loaded.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    })
  })

  test('空数据状态', async ({ page }) => {
    await page.route('/api/v1/dashboard/stats', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(emptyStats),
      })
    )
    await page.route('/api/v1/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(authMe),
      })
    )
    await mockAuth(page)
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('dashboard-empty.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    })
  })

  test('加载失败状态', async ({ page }) => {
    await page.route('/api/v1/dashboard/stats', (route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ message: '服务器错误，请稍后重试' }),
      })
    )
    await page.route('/api/v1/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(authMe),
      })
    )
    await mockAuth(page)
    await page.goto('/dashboard')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('dashboard-error.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    })
  })
})
