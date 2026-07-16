import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ACCOUNT = `mfal_${ts}`

test.describe('MFA Login Flow (browser)', () => {
  test('register via browser', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT)
    await inputs.nth(1).fill('MFALoginUser')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill('test123456')
    await inputs.nth(4).fill('test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=MFALoginUser')).toBeVisible({ timeout: 15000 })
  })
})
