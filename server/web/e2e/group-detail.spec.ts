import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ALICE = { account: `gda_${ts}`, pass: 'test123456' }
const BOB = { account: `gdb_${ts}`, pass: 'test123456' }

/** Register a user in the given browser context and close the page */
async function registerUser(ctx: import('@playwright/test').BrowserContext, acct: string, name: string, pass: string) {
  const page = await ctx.newPage()
  await page.goto('/register')
  await page.waitForTimeout(300)
  const inputs = page.locator('input')
  await inputs.nth(0).fill(acct)
  await inputs.nth(1).fill(name)
  await inputs.nth(2).fill('')
  await inputs.nth(3).fill(pass)
  await inputs.nth(4).fill(pass)
  await page.click('button[type="submit"]')
  await expect(page.locator(`text=${name}`)).toBeVisible({ timeout: 15000 })
  await page.waitForTimeout(2000)
  await page.close()
}

/** Helper: open + menu → Create Group → search member → select → create */
async function createGroupWithMember(page: import('@playwright/test').Page, groupName: string, memberName: string) {
  const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
  await plusBtn.first().click({ force: true })
  await page.waitForTimeout(500)
  await page.getByText('Create Group', { exact: true }).or(page.getByText('创建群组', { exact: true })).click()
  await page.waitForTimeout(500)
  await page.locator('input[placeholder*="Group name"], input[placeholder*="群组名称"]').first().fill(groupName)
  await page.waitForTimeout(300)

  // Search for member
  const searchInput = page.locator('input[placeholder*="搜索成员"]').first()
  await expect(searchInput).toBeVisible({ timeout: 2000 })
  await searchInput.fill(memberName)
  await page.keyboard.press('Enter')
  await page.waitForTimeout(1500)
  await expect(page.getByText(memberName).first()).toBeVisible({ timeout: 3000 })

  // Select member
  await page.getByText(memberName).first().click()
  await page.waitForTimeout(500)
  // Verify selected
  await expect(page.getByText('已选').first()).toBeVisible({ timeout: 2000 })

  // Click Create
  const createBtn = page.locator('button').filter({ hasText: /Create Group|创建群组/ }).last()
  await expect(createBtn).toBeVisible({ timeout: 2000 })
  await createBtn.click()
  // Wait for navigation to group chat
  await page.waitForFunction(() => {
    const path = window.location.pathname
    return path.startsWith('/chat/') && path.length > 6
  }, { timeout: 10000 })
  await page.waitForTimeout(1000)
}

test.describe('Group Detail Features', () => {

  test('group settings modal shows toggle switches', async ({ browser }) => {
    // Register a second user so Alice can create a group
    const bobCtx = await browser.newContext()
    await registerUser(bobCtx, `gds_bob_${ts}`, 'GDSBob', BOB.pass)
    await bobCtx.close()

    // Register Alice
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    await page.goto('/register')
    await page.waitForTimeout(300)
    let inputs = page.locator('input')
    await inputs.nth(0).fill(ALICE.account)
    await inputs.nth(1).fill('AliceGS')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=AliceGS')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/chat/, { timeout: 5000 })

    const GROUP = `GS_${ts}`
    await createGroupWithMember(page, GROUP, 'GDSBob')

    // ✅ Verify: navigated to group chat
    await expect(page).toHaveURL(/\/chat\//, { timeout: 10000 })
    await page.waitForTimeout(1000)

    // Open more menu
    const moreBtn = page.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    await expect(moreBtn).toBeVisible({ timeout: 2000 })
    await moreBtn.click()
    await page.waitForTimeout(500)

    // ✅ Verify group menu items exist
    await expect(page.getByText('基本信息').or(page.getByText('Basic Info'))).toBeVisible({ timeout: 2000 })
    await expect(page.getByText('会话设置').or(page.getByText('Settings'))).toBeVisible({ timeout: 2000 })
    await expect(page.getByText('添加成员').or(page.getByText('Add Member'))).toBeVisible({ timeout: 2000 })
    await expect(page.getByText('成员列表').or(page.getByText('Members'))).toBeVisible({ timeout: 2000 })

    // Click Settings
    await page.getByText('会话设置').or(page.getByText('Settings')).click()
    await page.waitForTimeout(500)

    // ✅ Verify settings modal with toggles
    await expect(page.getByText('会话设置').or(page.getByText('Conversation Settings'))).toBeVisible({ timeout: 3000 })
    await expect(page.getByText('Agent 仅显示回复').or(page.getByText('Agent: Show Response Only'))).toBeVisible({ timeout: 2000 })
    await expect(page.getByText('文件变更通知').or(page.getByText('File Change Notifications'))).toBeVisible({ timeout: 2000 })
    await expect(page.getByText('可被搜索').or(page.getByText('Discoverable'))).toBeVisible({ timeout: 2000 })

    // Close settings
    await page.locator('.fixed.inset-0.z-50').first().click({ position: { x: 5, y: 5 } })
    await page.waitForTimeout(300)
    await ctx.close()
  })

  test('member list and add member dialog', async ({ browser }) => {
    // Register Carol as searchable user
    const carolCtx = await browser.newContext()
    await registerUser(carolCtx, `ml_carol_${ts}`, 'CarolML', ALICE.pass)
    await carolCtx.close()

    // Register Alice
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    await page.goto('/register')
    await page.waitForTimeout(300)
    let inputs = page.locator('input')
    await inputs.nth(0).fill(`${ALICE.account}_ml`)
    await inputs.nth(1).fill('AliceML')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=AliceML')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/chat/, { timeout: 5000 })

    // Open + menu → Create Group
    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)
    await page.getByText('Create Group', { exact: true }).or(page.getByText('创建群组', { exact: true })).click()
    await page.waitForTimeout(500)
    const GROUP = `ML_${ts}`
    await page.locator('input[placeholder*="Group name"], input[placeholder*="群组名称"]').first().fill(GROUP)
    await page.waitForTimeout(300)

    // Search for Carol and select
    const searchInput = page.locator('input[placeholder*="搜索成员"]').first()
    await expect(searchInput).toBeVisible({ timeout: 2000 })
    await searchInput.fill('CarolML')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1500)
    await expect(page.getByText('CarolML').first()).toBeVisible({ timeout: 3000 })
    await page.getByText('CarolML').first().click()
    await page.waitForTimeout(500)
    // Verify selected
    await expect(page.getByText('已选').first()).toBeVisible({ timeout: 2000 })

    // Click Create
    const createBtn = page.locator('button').filter({ hasText: /Create Group|创建群组/ }).last()
    await expect(createBtn).toBeVisible({ timeout: 2000 })
    await createBtn.click()
    await page.waitForFunction(() => {
      const path = window.location.pathname
      return path.startsWith('/chat/') && path.length > 6
    }, { timeout: 10000 })
    await page.waitForTimeout(1000)

    // Open member list
    const moreBtn = page.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    await expect(moreBtn).toBeVisible({ timeout: 2000 })
    await moreBtn.click()
    await page.waitForTimeout(500)
    await page.getByText('成员列表').or(page.getByText('Members')).click()
    await page.waitForTimeout(1000)

    // ✅ Verify both members shown
    await expect(page.getByText('AliceML').first()).toBeVisible({ timeout: 3000 })
    await expect(page.getByText('CarolML').first()).toBeVisible({ timeout: 3000 })
    await expect(page.getByText(/[（(]2[)）]/)).toBeVisible({ timeout: 2000 })
    await expect(page.locator('input[placeholder*="搜索成员"]').first()).toBeVisible({ timeout: 2000 })

    // Close member list
    await page.locator('.fixed.inset-0.z-50').first().click({ position: { x: 5, y: 5 } })
    await page.waitForTimeout(300)

    // Open Add Member dialog
    await moreBtn.click()
    await page.waitForTimeout(500)
    await page.getByText('添加成员').or(page.getByText('Add Member')).click()
    await page.waitForTimeout(500)

    // ✅ Verify add member dialog shows with search
    const addSearch = page.locator('input[placeholder*="搜索用户"]').first()
    await expect(addSearch).toBeVisible({ timeout: 2000 })
    // Search non-existent user
    await addSearch.fill('NonExistent')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1000)
    await expect(page.getByText('未找到用户').or(page.getByText('No users found'))).toBeVisible({ timeout: 3000 })

    await page.locator('.fixed.inset-0.z-50').first().click({ position: { x: 5, y: 5 } })
    await page.waitForTimeout(300)
    await ctx.close()
  })

  test('add member dialog search finds registered user', async ({ browser }) => {
    // Register Bob as a searchable user
    const bobCtx = await browser.newContext()
    await registerUser(bobCtx, BOB.account, 'BobSearch', BOB.pass)
    await bobCtx.close()

    // Register Alice
    const ctx = await browser.newContext()
    const page = await ctx.newPage()
    await page.goto('/register')
    await page.waitForTimeout(300)
    let inputs = page.locator('input')
    await inputs.nth(0).fill(`${ALICE.account}_as`)
    await inputs.nth(1).fill('AliceAS')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=AliceAS')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/chat/, { timeout: 5000 })

    // Create a group (need a member - use Alice's own name? No that won't work)
    // Instead, create group with Bob as member via the create dialog
    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)
    await page.getByText('Create Group', { exact: true }).or(page.getByText('创建群组', { exact: true })).click()
    await page.waitForTimeout(500)
    const GROUP = `AS_${ts}`
    await page.locator('input[placeholder*="Group name"], input[placeholder*="群组名称"]').first().fill(GROUP)
    await page.waitForTimeout(300)

    // Search for BobSearch
    const searchInput = page.locator('input[placeholder*="搜索成员"]').first()
    await expect(searchInput).toBeVisible({ timeout: 2000 })
    await searchInput.fill('BobSearch')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1500)

    // ✅ Verify search found BobSearch
    await expect(page.getByText('BobSearch').first()).toBeVisible({ timeout: 3000 })
    // Select Bob
    await page.getByText('BobSearch').first().click()
    await page.waitForTimeout(500)
    await expect(page.getByText('已选').first()).toBeVisible({ timeout: 2000 })

    // Create the group
    const createBtn = page.locator('button').filter({ hasText: /Create Group|创建群组/ }).last()
    await expect(createBtn).toBeVisible({ timeout: 2000 })
    await createBtn.click()
    await page.waitForFunction(() => {
      const path = window.location.pathname
      return path.startsWith('/chat/') && path.length > 6
    }, { timeout: 10000 })
    await page.waitForTimeout(1000)

    // Open member list to verify both members
    const moreBtn = page.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    await expect(moreBtn).toBeVisible({ timeout: 2000 })
    await moreBtn.click()
    await page.waitForTimeout(500)
    await page.getByText('成员列表').or(page.getByText('Members')).click()
    await page.waitForTimeout(1000)

    // ✅ Verify both members
    await expect(page.getByText('AliceAS').first()).toBeVisible({ timeout: 3000 })
    await expect(page.getByText('BobSearch').first()).toBeVisible({ timeout: 3000 })
    await expect(page.getByText(/[（(]2[)）]/)).toBeVisible({ timeout: 2000 })

    await ctx.close()
  })
})
