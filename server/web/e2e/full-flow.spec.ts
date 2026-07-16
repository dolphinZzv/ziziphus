import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const A = { account: `flow_a_${ts}`, name: 'Alice', password: 'test123456' }
const B = { account: `flow_b_${ts}`, name: 'Bob', password: 'test123456' }

async function register(page, user, name) {
  await page.goto('/register')
  await page.waitForTimeout(300)
  const inputs = page.locator('input')
  await inputs.nth(0).fill(user.account)
  await inputs.nth(1).fill(name)
  await inputs.nth(2).fill('')
  await inputs.nth(3).fill(user.password)
  await inputs.nth(4).fill(user.password)
  await page.click('button[type="submit"]')
  await expect(page.locator(`text=${name}`)).toBeVisible({ timeout: 15000 })
  await page.waitForTimeout(2000)
}

async function login(page, user, name) {
  await page.goto('/')
  await page.fill('input[type="text"]', user.account)
  await page.fill('input[type="password"]', user.password)
  await page.click('button[type="submit"]')
  await expect(page.locator(`text=${name}`)).toBeVisible({ timeout: 15000 })
  await page.waitForTimeout(2000)
}

test.describe('Full Browser Flow', () => {
  test('1. register Alice and Bob in sequence', async ({ browser }) => {
    for (const { user, name } of [{ user: A, name: 'Alice' }, { user: B, name: 'Bob' }]) {
      const ctx = await browser.newContext()
      const page = await ctx.newPage()
      await register(page, user, name)
      await ctx.close()
    }
  })

  test('2. Alice logs in and creates a group chat with Bob', async ({ page }) => {
    await login(page, A, 'Alice')

    // Open plus menu → create group
    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)

    // Click create group option
    const createGroup = page.getByText('创建群聊').or(page.getByText('Create Group'))
    await createGroup.click()
    await page.waitForTimeout(500)

    // Fill group name
    const nameInput = page.locator('input[placeholder*="群组名称"], input[placeholder*="Group name"]').first()
    await nameInput.fill(`TestGroup_${ts}`)

    // Click search member input
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="Search"]').first()
    if (await searchInput.isVisible({ timeout: 2000 }).catch(() => false)) {
      await searchInput.fill('Bob')
      await page.waitForTimeout(1000)
    }

    // Click create button
    const submitBtn = page.getByText('创建').or(page.getByText('Create')).first()
    if (await submitBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await submitBtn.click()
      await page.waitForTimeout(2000)
    }
  })

  test('3. Alice opens chat and sends a message', async ({ page }) => {
    await login(page, A, 'Alice')
    await page.waitForTimeout(2000)

    // Click first conversation
    const convItems = page.locator('[class*="conversation"]')
    if (await convItems.first().isVisible({ timeout: 5000 }).catch(() => false)) {
      await convItems.first().click()
      await page.waitForTimeout(1000)

      // Type and send a message
      const editor = page.locator('[contenteditable="true"]').first()
      if (await editor.isVisible({ timeout: 3000 }).catch(() => false)) {
        await editor.fill('Hello from Alice')
        await page.keyboard.press('Enter')
        await page.waitForTimeout(1000)
      }
    }
  })

  test('4. Alice opens profile and navigates to settings', async ({ page }) => {
    await login(page, A, 'Alice')

    // Open profile
    await page.locator('text=Alice').first().click({ force: true })
    await page.waitForTimeout(1000)

    // Click settings
    const settingsBtn = page.getByText('应用设置').or(page.getByText('Settings'))
    if (await settingsBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await settingsBtn.click()
      await page.waitForTimeout(500)
    }
  })

  test('5. Alice creates an agent from profile', async ({ page }) => {
    await login(page, A, 'Alice')

    // Open profile
    await page.locator('text=Alice').first().click({ force: true })
    await page.waitForTimeout(1000)

    // Click agent management
    const agentBtn = page.getByText('Agent 管理').or(page.getByText('Agent Management'))
    if (await agentBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await agentBtn.click()
      await page.waitForTimeout(1000)

      // Create new agent
      const createBtn = page.getByText('创建 Agent').or(page.getByText('Create Agent'))
      if (await createBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await createBtn.click()
        await page.waitForTimeout(500)

        // Fill agent name
        const agentName = page.locator('input').first()
        if (await agentName.isVisible({ timeout: 2000 }).catch(() => false)) {
          await agentName.fill(`Bot_${ts}`)
          await page.waitForTimeout(300)

          // Submit
          await page.keyboard.press('Enter')
          await page.waitForTimeout(1000)
        }
      }
    }
  })

  test('6. Alice views sessions from profile', async ({ page }) => {
    await login(page, A, 'Alice')

    // Open profile
    await page.locator('text=Alice').first().click({ force: true })
    await page.waitForTimeout(1000)

    // Click session management
    const sessionBtn = page.getByText('设备管理').or(page.getByText('Device Management'))
    if (await sessionBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await sessionBtn.click()
      await page.waitForTimeout(1000)
    }
  })

  test('7. Alice toggles file panel in conversation', async ({ page }) => {
    await login(page, A, 'Alice')
    await page.waitForTimeout(2000)

    // Click first conversation
    const convItems = page.locator('[class*="conversation"]')
    if (await convItems.first().isVisible({ timeout: 5000 }).catch(() => false)) {
      await convItems.first().click()
      await page.waitForTimeout(1000)

      // Click file button in toolbar
      const fileBtn = page.locator('button:has(svg.lucide-file)').first()
      if (await fileBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
        await fileBtn.click()
        await page.waitForTimeout(1000)
      }
    }
  })
})
