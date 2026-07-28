import { test, expect } from '@playwright/test'

const authMe = {
  data: { id: 1, username: 'admin', is_admin: true },
  message: 'ok',
}

test.describe('文件管理', () => {
  test('空数据状态', async ({ page }) => {
    // Mock auth check so we land on /files, not login redirect
    await page.route('/api/v1/auth/me', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(authMe),
      })
    )
    // Mock files list to return empty results (default state per D-06.2)
    await page.route('/api/v1/files**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ data: { items: [], total: 0 }, message: 'ok' }),
      })
    )
    await page.goto('/files')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('files-empty.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    })
  })
})
