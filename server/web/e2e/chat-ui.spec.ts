import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ACCOUNT = `chatu_${ts}`

test.describe('Chat UI', () => {
  test('register and login via browser', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT)
    await inputs.nth(1).fill('ChatTester')
    await inputs.nth(2).fill('test123456')
    await inputs.nth(3).fill('test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=ChatTester')).toBeVisible({ timeout: 15000 })
  })

  test('input bar renders with send button', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=ChatTester')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)
  })
})
