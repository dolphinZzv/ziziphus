import { test, expect } from './fixtures/coverage'

const TS = Date.now()
const ACCOUNT = `ui_${TS}`
const PASSWORD = 'test123456'

test.describe('UI Coverage — full browser interaction', () => {
  test('1. register via browser', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT) 
    await inputs.nth(1).fill('UiUser')
    await inputs.nth(2).fill('')      
    await inputs.nth(3).fill(PASSWORD)
    await inputs.nth(4).fill(PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
  })

  test('2. login and view conversations', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)
  })

  test('3. search users via new chat dialog', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    if (await plusBtn.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await plusBtn.first().click({ force: true })
      await page.waitForTimeout(1000)
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(1500)
      }
    }
  })

  test('4. conversations and settings', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)

    // Click first conversation in sidebar
    const convItems = page.locator('[class*="conversation"]')
    if (await convItems.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await convItems.first().click()
      await page.waitForTimeout(1500)
    }

    // Look for file panel button
    const fileBtn = page.locator('button:has(svg.lucide-file), button:has(svg.lucide-folder)')
    if (await fileBtn.first().isVisible({ timeout: 2000 }).catch(() => false)) {
      await fileBtn.first().click()
      await page.waitForTimeout(1000)
    }
  })

  test('5. settings page', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const settingsBtn = page.locator('a[href*="settings"], button:has(svg.lucide-settings)')
    if (await settingsBtn.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await settingsBtn.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('6. profile page', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click user name/avatar for profile
    const userLabel = page.locator('text=UiUser')
    if (await userLabel.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await userLabel.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('7. contacts list', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const contactBtn = page.locator('a[href*="contact"], button:has(svg.lucide-users)')
    if (await contactBtn.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await contactBtn.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('8. agents list', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const agentBtn = page.locator('button:has(svg.lucide-bot)')
    if (await agentBtn.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await agentBtn.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('9. sessions list', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const sessionBtn = page.locator('button:has(svg.lucide-monitor), button:has(svg.lucide-smartphone)')
    if (await sessionBtn.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await sessionBtn.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('10. conversation interaction', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)

    const convItems = page.locator('[class*="conversation"]')
    if (await convItems.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await convItems.first().click()
      await page.waitForTimeout(1500)

      const editor = page.locator('[contenteditable="true"]').first()
      if (await editor.isVisible({ timeout: 2000 }).catch(() => false)) {
        await editor.fill('Coverage test message')
        await page.keyboard.press('Enter')
        await page.waitForTimeout(500)
      }
    }
  })
})
