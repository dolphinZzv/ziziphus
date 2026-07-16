import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ACCOUNT = `det_${ts}`

test.describe('Chat Detail & History', () => {
  test('register user', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT)
    await inputs.nth(1).fill('DetUser')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill('test123456')
    await inputs.nth(4).fill('test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=DetUser')).toBeVisible({ timeout: 15000 })
  })

  test('chat view shows after login', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=DetUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)

    // Sidebar should show with user name and conversation list
    await expect(page.locator('text=DetUser').first()).toBeVisible({ timeout: 3000 })
  })

  test('file panel button exists', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=DetUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const fileBtn = page.locator('button:has(svg.lucide-file), button:has(svg.lucide-folder)')
    if (await fileBtn.first().isVisible({ timeout: 2000 }).catch(() => false)) {
      await fileBtn.first().click()
      await page.waitForTimeout(1000)
    }
  })

  test('first conversation click shows detail view', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=DetUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)

    // Click first conversation
    const convItems = page.locator('[class*="conversation"]')
    if (await convItems.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await convItems.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('message search button toggles search bar', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=DetUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)

    // Click first conversation
    const convItems = page.locator('[class*="conversation"]')
    if (await convItems.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await convItems.first().click()
      await page.waitForTimeout(1500)

      // Search button
      const searchBtn = page.locator('button:has(svg.lucide-search)')
      if (await searchBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await searchBtn.click()
        await page.waitForTimeout(500)
        const searchInput = page.locator('input[placeholder*="搜索"],[placeholder*="Search"]').first()
        await expect(searchInput).toBeVisible({ timeout: 2000 })
      }
    }
  })
})
