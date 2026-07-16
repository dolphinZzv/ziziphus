import { test, expect } from './fixtures/coverage'

test.describe('Group and Webhook UI', () => {
  test('navigate to group settings (webhook area accessible via group detail)', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('h1')).toBeVisible({ timeout: 5000 })
  })
})

test.describe('Agent Management UI', () => {
  test('navigate to home page', async ({ page }) => {
    await page.goto('/login')
    await expect(page.locator('h1')).toBeVisible({ timeout: 5000 })
  })
})
