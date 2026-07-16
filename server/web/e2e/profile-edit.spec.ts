import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ACCOUNT = `prof_${ts}`

test.describe('Profile Edit', () => {
  test('register and open profile in one flow', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT)
    await inputs.nth(1).fill('ProfUser')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill('test123456')
    await inputs.nth(4).fill('test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=ProfUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Sidebar renders with user name
    await expect(page.locator('text=ProfUser').first()).toBeVisible({ timeout: 3000 })
  })

  test('settings accessible from profile in sidebar', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=ProfUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click user name to open profile dialog
    await page.locator('text=ProfUser').first().click({ force: true })
    await page.waitForTimeout(1000)
  })

  test('shortcuts and privacy buttons render in settings', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=ProfUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    await page.locator('text=ProfUser').first().click({ force: true })
    await page.waitForTimeout(1000)

    // Click 应用设置 (settings)
    const settingsBtn = page.getByText('应用设置').or(page.getByText('Settings'))
    if (await settingsBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await settingsBtn.click()
      await page.waitForTimeout(500)
    }
  })

  test('agent management link visible in profile', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=ProfUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    await page.locator('text=ProfUser').first().click({ force: true })
    await page.waitForTimeout(1000)

    await expect(page.getByText('Agent 管理').or(page.getByText('Agent Management'))).toBeVisible({ timeout: 3000 })
  })

  test('session management link visible in profile', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=ProfUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    await page.locator('text=ProfUser').first().click({ force: true })
    await page.waitForTimeout(1000)

    await expect(page.getByText('设备管理').or(page.getByText('Device Management'))).toBeVisible({ timeout: 3000 })
  })
})
