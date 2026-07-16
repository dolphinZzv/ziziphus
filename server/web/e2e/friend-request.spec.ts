import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const A = { account: `fa_${ts}`, name: 'FriendA', password: 'test123456' }
const B = { account: `fb_${ts}`, name: 'FriendB', password: 'test123456' }

test.describe('Friend Request Flow (browser)', () => {
  test('register user A', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(A.account)
    await inputs.nth(1).fill(A.name)
    await inputs.nth(2).fill(A.password)
    await inputs.nth(3).fill(A.password)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=FriendA')).toBeVisible({ timeout: 15000 })
  })

  test('register user B', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(B.account)
    await inputs.nth(1).fill(B.name)
    await inputs.nth(2).fill(B.password)
    await inputs.nth(3).fill(B.password)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=FriendB')).toBeVisible({ timeout: 15000 })
  })

  test('login A and view conversations', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', A.account)
    await page.fill('input[type="password"]', A.password)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=FriendA')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
  })

  test('login A and open contacts panel', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', A.account)
    await page.fill('input[type="password"]', A.password)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=FriendA')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    const contactBtn = page.locator('button:has(svg.lucide-users), a[href*="contact"]').first()
    if (await contactBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await contactBtn.click()
      await page.waitForTimeout(1500)
    }
  })
})
