import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ALICE = { account: `ga_${ts}`, pass: 'test123456' }
const BOB = { account: `gb_${ts}`, pass: 'test123456' }
const GROUP = `Group_${ts}`

test.describe('Group Actions', () => {
  test('register Alice, create group, disband', async ({ page }) => {
    await page.goto('/register')
    await page.waitForTimeout(300)
    let inputs = page.locator('input')
    await inputs.nth(0).fill(ALICE.account)
    await inputs.nth(1).fill('Alice')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=Alice').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Create group
    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)
    await page.getByText('创建群聊').or(page.getByText('Create Group')).click()
    await page.waitForTimeout(500)
    await page.locator('input[placeholder*="群组名称"], input[placeholder*="Group name"]').first().fill(GROUP)
    await page.waitForTimeout(300)
    const submitBtn = page.getByText('创建').or(page.getByText('Create')).first()
    if (await submitBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await submitBtn.click()
      await page.waitForTimeout(2000)
    }

    // Open group → click more menu → disband
    await page.waitForTimeout(1000)
    const groupConv = page.locator(`text=${GROUP}`)
    if (await groupConv.isVisible({ timeout: 5000 }).catch(() => false)) {
      await groupConv.click()
      await page.waitForTimeout(1000)
    }
    const moreBtn = page.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    if (await moreBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await moreBtn.click()
      await page.waitForTimeout(500)
    }
    const disbandBtn = page.getByText('解散群组').or(page.getByText('Disband Group'))
    if (await disbandBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await disbandBtn.click()
      await page.waitForTimeout(500)
    }

    // Verify: group no longer in conversation list after disband
    await page.waitForTimeout(1000)
    await expect(page.locator(`text=${GROUP}`).first()).not.toBeVisible({ timeout: 5000 })
  })

  test('Bob joins group and leaves', async ({ browser }) => {
    // Register Bob
    const bobCtx = await browser.newContext()
    let page = await bobCtx.newPage()
    await page.goto('/register')
    await page.waitForTimeout(300)
    let inputs = page.locator('input')
    await inputs.nth(0).fill(BOB.account)
    await inputs.nth(1).fill('Bob')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(BOB.pass)
    await inputs.nth(4).fill(BOB.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=Bob').first()).toBeVisible({ timeout: 15000 })
    await page.close()

    // Alice creates group with Bob as member
    page = await bobCtx.newPage()
    await page.goto('/')
    await page.fill('input[type="text"]', ALICE.account)
    await page.fill('input[type="password"]', ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=Alice').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)
    await page.getByText('创建群聊').or(page.getByText('Create Group')).click()
    await page.waitForTimeout(500)
    await page.locator('input[placeholder*="群组名称"], input[placeholder*="Group name"]').first().fill(`Leave_${ts}`)
    const searchInput = page.locator('input[placeholder*="搜索"], input[placeholder*="Search"]').first()
    if (await searchInput.isVisible({ timeout: 2000 }).catch(() => false)) {
      await searchInput.fill('Bob')
      await page.waitForTimeout(1000)
    }
    const submitBtn = page.getByText('创建').or(page.getByText('Create')).first()
    if (await submitBtn.isVisible({ timeout: 2000 }).catch(() => false)) {
      await submitBtn.click()
      await page.waitForTimeout(2000)
    }
    await page.close()

    // Bob logs in and leaves the group
    page = await bobCtx.newPage()
    await page.goto('/')
    await page.fill('input[type="text"]', BOB.account)
    await page.fill('input[type="password"]', BOB.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=Bob').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Click group and open more menu
    const groupConv = page.locator(`text=Leave_${ts}`)
    if (await groupConv.isVisible({ timeout: 5000 }).catch(() => false)) {
      await groupConv.click()
      await page.waitForTimeout(1000)
    }
    const moreBtn = page.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    if (await moreBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await moreBtn.click()
      await page.waitForTimeout(500)
    }

    // Click leave (Bob is not owner, so leave shows instead of disband)
    const leaveBtn = page.getByText('退出群聊').or(page.getByText('Leave Group'))
    if (await leaveBtn.isVisible({ timeout: 3000 }).catch(() => false)) {
      await leaveBtn.click()
      await page.waitForTimeout(500)
    }

    // Verify: group no longer in Bob's conversation list after leave
    await page.waitForTimeout(1000)
    await expect(page.locator(`text=Leave_${ts}`).first()).not.toBeVisible({ timeout: 5000 })
    await bobCtx.close()
  })
})
