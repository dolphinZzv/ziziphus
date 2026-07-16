import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ACCOUNT = `ui_${ts}`
const PASSWORD = 'test123456'

test.describe('UI Coverage — real browser interactions', () => {
  test('1. register and auto-login via browser', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)

    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT)
    await inputs.nth(1).fill('UiUser')
    await inputs.nth(2).fill(PASSWORD)
    await inputs.nth(3).fill(PASSWORD)

    await page.click('button[type="submit"]')
    // Triggers: POST /api/v1/users/register → POST /api/v1/users/login
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
  })

  test('2. login and view conversations', async ({ page }) => {
    await page.goto('/')
    await page.waitForTimeout(300)
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })

    // Wait for conversation list to load
    await page.waitForTimeout(2000)
  })

  test('3. open new chat dialog and search users', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click the + button (new chat) — triggers user search
    const plusBtns = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    if (await plusBtns.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await plusBtns.first().click({ force: true })
      await page.waitForTimeout(1000)

      // Type in search — triggers GET /api/v1/users/search
      const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="Search"]').first()
      if (await searchInput.isVisible({ timeout: 2000 }).catch(() => false)) {
        await searchInput.fill('test')
        await page.waitForTimeout(1500)
      }
    }
  })

  test('4. click first conversation and view chat', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)

    // Click on first conversation — triggers GET /api/v1/conversations/{id}
    // and GET /api/v1/conversations/{id}/messages
    const convRows = page.locator('[class*="conversation"], [class*="chat-item"], [class*="conv-item"]')
    if (await convRows.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      // Click the first visible conversation
      await convRows.first().click()
      await page.waitForTimeout(1500)

      // Try to type a message — triggers send message
      const editor = page.locator('[contenteditable="true"]').first()
      if (await editor.isVisible({ timeout: 2000 }).catch(() => false)) {
        await editor.fill('Hello from coverage test')
        // Try pressing Enter to send
        await page.keyboard.press('Enter')
        await page.waitForTimeout(500)
      }
    }
  })

  test('5. view settings page', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Look for settings button — triggers various API calls
    const settingsBtns = page.locator('a[href*="settings"], button:has(svg.lucide-settings), button:has(svg.lucide-cog)')
    if (await settingsBtns.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await settingsBtns.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('6. view profile page', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click user name/avatar in sidebar — triggers GET /api/v1/users/me
    const userBtns = page.locator('text=UiUser')
    if (await userBtns.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await userBtns.first().click()
      await page.waitForTimeout(1500)

      // Look for a profile edit button
      const editBtn = page.locator('button:has-text("编辑"), button:has-text("Edit")').first()
      if (await editBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await editBtn.click()
        await page.waitForTimeout(1000)
      }
    }
  })

  test('7. view contacts', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Find and click contacts — triggers GET /api/v1/contacts
    const contactBtns = page.locator('a[href*="contact"], button:has(svg.lucide-users)')
    if (await contactBtns.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await contactBtns.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('8. view agents list', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Find and click agents/bots — triggers GET /api/v1/users/me/agents
    const agentBtns = page.locator('button:has(svg.lucide-bot)')
    if (await agentBtns.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await agentBtns.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('9. view sessions list', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Find and click sessions — triggers GET /api/v1/sessions
    const sessionBtns = page.locator('button:has(svg.lucide-monitor), button:has(svg.lucide-smartphone)')
    if (await sessionBtns.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await sessionBtns.first().click()
      await page.waitForTimeout(1500)
    }
  })

  test('10. view group detail if available', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=UiUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)

    // Click first conversation — might be a group
    const convRows = page.locator('[class*="conversation"], [class*="chat-item"]')
    if (await convRows.first().isVisible({ timeout: 3000 }).catch(() => false)) {
      await convRows.first().click()
      await page.waitForTimeout(1500)
    }
  })
})
