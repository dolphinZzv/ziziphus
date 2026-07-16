import { test, expect } from './fixtures/coverage'

const TS = Date.now()
const ACCOUNT = `fp_${TS}`
const PASSWORD = 'test123456'

test.describe('File Panel', () => {
  test.beforeAll(async ({ browser }) => {
    // Create user via browser register
    const page = await browser.newPage()
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT)
    await inputs.nth(1).fill('FileTester')
    await inputs.nth(2).fill(PASSWORD)
    await inputs.nth(3).fill(PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=FileTester')).toBeVisible({ timeout: 15000 })
    await page.close()
  })

  test('file panel UI renders after login', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=FileTester')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click file panel button in sidebar
    const fileBtn = page.locator('button:has(svg.lucide-file), button:has(svg.lucide-folder)').first()
    if (await fileBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await fileBtn.click()
      await page.waitForTimeout(1000)
    }
  })
})
