import { test, expect } from '@playwright/test'

test.describe('404 页面', () => {
  test('默认状态', async ({ page }) => {
    // No API mocking needed — the 404 page is pure static UI
    await page.goto('/this-path-does-not-exist')
    await page.waitForLoadState('networkidle')
    await expect(page).toHaveScreenshot('notfound.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.01,
    })
  })
})
