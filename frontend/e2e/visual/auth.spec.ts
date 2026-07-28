import { test, expect } from '@playwright/test'

test.describe('登录页', () => {
  test('默认状态', async ({ page }) => {
    await page.goto('/auth/login')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('login-default.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    })
  })

  test('失败状态', async ({ page }) => {
    await page.route('/api/v1/auth/login', (route) =>
      route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ message: '用户名或密码错误' }),
      })
    )
    await page.goto('/auth/login')
    await page.waitForLoadState('networkidle')
    await page.fill('input[placeholder="用户名"]', 'admin')
    await page.fill('input[placeholder="密码"]', 'wrongpass')
    await page.click('button[type="submit"]')
    await page.waitForTimeout(800)
    await expect(page).toHaveScreenshot('login-failed.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    })
  })
})
