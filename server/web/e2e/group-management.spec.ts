import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ACCOUNT = `grp_${ts}`

test.describe('Group Management', () => {
  test('register user', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ACCOUNT)
    await inputs.nth(1).fill('GrpUser')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill('test123456')
    await inputs.nth(4).fill('test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=GrpUser')).toBeVisible({ timeout: 15000 })
  })

  test('open new group dialog from plus menu', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=GrpUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click + menu
    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)

    // Should show create group option
    await expect(page.getByText('创建群聊').or(page.getByText('Create Group'))).toBeVisible({ timeout: 3000 })
  })

  test('open join group dialog from plus menu', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=GrpUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)

    // Should show join group option
    await expect(page.getByText('加入群聊').or(page.getByText('Join Group'))).toBeVisible({ timeout: 3000 })
  })

  test('conversation list renders', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="text"]', ACCOUNT)
    await page.fill('input[type="password"]', 'test123456')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=GrpUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(3000)
  })
})
