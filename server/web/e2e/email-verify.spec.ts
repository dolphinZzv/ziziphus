import { test, expect } from './fixtures/coverage'

const ts = Date.now()

test.describe('Email Verify (browser)', () => {
  test('register and view login page', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(`ev_${ts}`)
    await inputs.nth(1).fill('EmailUser')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill('test123456')
    await inputs.nth(4).fill('test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=EmailUser')).toBeVisible({ timeout: 15000 })
  })
})
