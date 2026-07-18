import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ALICE = { account: `gs_${ts}`, pass: 'test123456' }
const GROUP = `ShareGroup_${ts}`

test.describe('Group Sharecard', () => {

  test('not-logged-in user sees login button on sharecard', async ({ browser }) => {
    // Create a group and get share token
    const ctx = await browser.newContext()
    const page = await ctx.newPage()

    // Register a helper user so we can add a member to the group
    const helperCtx = await browser.newContext()
    const helperPage = await helperCtx.newPage()
    await helperPage.goto('/register')
    await helperPage.waitForTimeout(300)
    let hInputs = helperPage.locator('input')
    await hInputs.nth(0).fill(`gs_helper_${ts}`)
    await hInputs.nth(1).fill('Helper')
    await hInputs.nth(2).fill('')
    await hInputs.nth(3).fill(ALICE.pass)
    await hInputs.nth(4).fill(ALICE.pass)
    await helperPage.click('button[type="submit"]')
    await expect(helperPage.locator('text=Helper').first()).toBeVisible({ timeout: 15000 })
    await helperPage.waitForTimeout(1000)
    await helperPage.close()
    await helperCtx.close()

    // Register Alice
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

    // Create group with Helper as member
    const plusBtn = page.locator('button').filter({ has: page.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await page.waitForTimeout(500)
    await page.getByText('Create Group', { exact: true }).or(page.getByText('创建群组', { exact: true })).click()
    await page.waitForTimeout(500)
    await page.locator('input[placeholder*="Group name"], input[placeholder*="群组名称"]').first().fill(GROUP)
    await page.waitForTimeout(300)

    // Search and select Helper
    const searchInput = page.locator('input[placeholder*="搜索成员"]').first()
    await expect(searchInput).toBeVisible({ timeout: 2000 })
    await searchInput.fill('Helper')
    await page.keyboard.press('Enter')
    await page.waitForTimeout(1500)
    await expect(page.getByText('Helper').first()).toBeVisible({ timeout: 3000 })
    await page.getByText('Helper').first().click()
    await page.waitForTimeout(500)
    await expect(page.getByText('已选').first()).toBeVisible({ timeout: 2000 })

    const submitBtn = page.locator('button').filter({ hasText: /Create Group|创建群组/ }).last()
    await expect(submitBtn).toBeVisible({ timeout: 2000 })
    await submitBtn.click()
    await page.waitForFunction(() => {
      const path = window.location.pathname
      return path.startsWith('/conversations/') && path.length > 6
    }, { timeout: 10000 })
    await page.waitForTimeout(1000)

    // Open group detail
    const moreBtn = page.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    await expect(moreBtn).toBeVisible({ timeout: 2000 })
    await moreBtn.click()
    await page.waitForTimeout(500)
    await page.getByText('Basic Info', { exact: true }).or(page.getByText('基本信息', { exact: true })).click()
    await page.waitForTimeout(500)

    // Generate share link
    await page.getByText('Generate Share Link', { exact: true }).or(page.getByText('生成分享链接', { exact: true })).click()
    await page.waitForTimeout(1000)

    // Extract share token from the visible link
    const shareLinkEl = page.locator('.select-all').first()
    await expect(shareLinkEl).toBeVisible({ timeout: 3000 })
    const shareUrl = await shareLinkEl.textContent()
    expect(shareUrl).toBeTruthy()
    const shareToken = shareUrl!.split('/').pop()
    expect(shareToken).toBeTruthy()

    // Now log Alice out and clear auth state
    await page.evaluate(() => {
      localStorage.clear()
      sessionStorage.clear()
    })
    await page.close()

    // Open sharecard in a fresh context (not logged in)
    const guestCtx = await browser.newContext()
    const guestPage = await guestCtx.newPage()
    await guestPage.goto(`/group-card/${shareToken}`)
    await guestPage.waitForTimeout(1000)

    // ✅ Verify: login button is shown
    await expect(
      guestPage.getByText('Log in to join this group').or(guestPage.getByText('登录后加入群组'))
    ).toBeVisible({ timeout: 10000 })

    // ✅ Verify: register button is shown
    await expect(
      guestPage.getByText('Create Account').or(guestPage.getByText('注册账号'))
    ).toBeVisible({ timeout: 3000 })

    // ✅ Verify: join group button is NOT shown (user not logged in)
    await expect(
      guestPage.getByText('Join Group', { exact: true }).or(guestPage.getByText('加入群组', { exact: true }))
    ).not.toBeVisible({ timeout: 2000 })

    await guestCtx.close()
  })

  test('logged-in user can join group via sharecard', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()

    // Register Bob
    await page.goto('/register')
    await page.waitForTimeout(300)
    let inputs = page.locator('input')
    await inputs.nth(0).fill(`gs_bob_${ts}`)
    await inputs.nth(1).fill('Bob')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=Bob').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/conversations/, { timeout: 5000 })

    // Register Alice in a separate context and create group
    const aliceCtx = await browser.newContext()
    const alicePage = await aliceCtx.newPage()
    await alicePage.goto('/register')
    await alicePage.waitForTimeout(300)
    inputs = alicePage.locator('input')
    await inputs.nth(0).fill(`gs_alice_${ts}`)
    await inputs.nth(1).fill('AliceGS')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await alicePage.click('button[type="submit"]')
    await expect(alicePage.locator('text=AliceGS').first()).toBeVisible({ timeout: 15000 })
    await alicePage.waitForTimeout(2000)

    // Create group with Bob as member
    const plusBtn = alicePage.locator('button').filter({ has: alicePage.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await alicePage.waitForTimeout(500)
    await alicePage.getByText('Create Group', { exact: true }).or(alicePage.getByText('创建群组', { exact: true })).click()
    await alicePage.waitForTimeout(500)
    await alicePage.locator('input[placeholder*="Group name"], input[placeholder*="群组名称"]').first().fill(`JoinGroup_${ts}`)
    await alicePage.waitForTimeout(300)

    // Search and select Bob
    const searchInput = alicePage.locator('input[placeholder*="搜索成员"]').first()
    await expect(searchInput).toBeVisible({ timeout: 2000 })
    await searchInput.fill('Bob')
    await alicePage.keyboard.press('Enter')
    await alicePage.waitForTimeout(1500)
    await expect(alicePage.getByText('Bob').first()).toBeVisible({ timeout: 3000 })
    await alicePage.getByText('Bob').first().click()
    await alicePage.waitForTimeout(500)
    await expect(alicePage.getByText('已选').first()).toBeVisible({ timeout: 2000 })

    let submitBtn = alicePage.locator('button').filter({ hasText: /Create Group|创建群组/ }).last()
    await expect(submitBtn).toBeVisible({ timeout: 2000 })
    await submitBtn.click()
    await alicePage.waitForFunction(() => {
      const path = window.location.pathname
      return path.startsWith('/conversations/') && path.length > 6
    }, { timeout: 10000 })
    await alicePage.waitForTimeout(1000)

    // Open group detail and generate share link
    const moreBtn = alicePage.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    await expect(moreBtn).toBeVisible({ timeout: 2000 })
    await moreBtn.click()
    await alicePage.waitForTimeout(500)
    await alicePage.getByText('Basic Info', { exact: true }).or(alicePage.getByText('基本信息', { exact: true })).click()
    await alicePage.waitForTimeout(500)
    await alicePage.getByText('Generate Share Link', { exact: true }).or(alicePage.getByText('生成分享链接', { exact: true })).click()
    await alicePage.waitForTimeout(1000)

    // Get share token
    const shareLinkEl = alicePage.locator('.select-all').first()
    await expect(shareLinkEl).toBeVisible({ timeout: 3000 })
    const shareUrl = await shareLinkEl.textContent()
    const shareToken = shareUrl!.split('/').pop()
    expect(shareToken).toBeTruthy()
    await alicePage.close()

    // Bob navigates to sharecard (already logged in)
    await page.goto(`/group-card/${shareToken}`)
    await page.waitForTimeout(1500)

    // ✅ Verify: "Join Group" button is shown (not "log in to join")
    await expect(
      page.getByText('Join Group', { exact: true }).or(page.getByText('加入群组', { exact: true }))
    ).toBeVisible({ timeout: 10000 })

    // ✅ Verify: login + register buttons are NOT shown
    await expect(
      page.getByText('Log in to join').or(page.getByText('登录后加入'))
    ).not.toBeVisible({ timeout: 2000 })

    // Click "Join Group" → confirmation dialog
    await page.getByText('Join Group', { exact: true }).or(page.getByText('加入群组', { exact: true })).click()
    await page.waitForTimeout(500)

    // ✅ Verify: confirmation dialog shows
    await expect(
      page.getByText('Join this group?').or(page.getByText('确认加入该群组？'))
    ).toBeVisible({ timeout: 3000 })

    // Confirm join
    await page.getByText('Confirm', { exact: true }).or(page.getByText('确定', { exact: true })).click()
    await page.waitForTimeout(2000)

    // ✅ Verify: either navigated to chat (direct join) or shows "sent" message
    const isInChat = await page.evaluate(() => window.location.pathname.startsWith('/conversations/'))
      .catch(() => false)
    const sentVisible = await page.getByText('Join request sent').or(page.getByText('已发送加入请求'))
      .isVisible({ timeout: 3000 }).catch(() => false)

    expect(isInChat || sentVisible).toBeTruthy()

    await ctx.close()
  })

  test('already a member redirects to chat', async ({ browser }) => {
    const ctx = await browser.newContext()
    const page = await ctx.newPage()

    // Register Carol
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(`gs_carol_${ts}`)
    await inputs.nth(1).fill('Carol')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=Carol').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // Alice creates group with Carol as member
    const aliceCtx = await browser.newContext()
    const alicePage = await aliceCtx.newPage()
    await alicePage.goto('/register')
    await alicePage.waitForTimeout(300)
    let aliceInputs = alicePage.locator('input')
    await aliceInputs.nth(0).fill(`gs_alice2_${ts}`)
    await aliceInputs.nth(1).fill('Alice2')
    await aliceInputs.nth(2).fill('')
    await aliceInputs.nth(3).fill(ALICE.pass)
    await aliceInputs.nth(4).fill(ALICE.pass)
    await alicePage.click('button[type="submit"]')
    await expect(alicePage.locator('text=Alice2').first()).toBeVisible({ timeout: 15000 })
    await alicePage.waitForTimeout(2000)

    // Create group with Carol
    const plusBtn = alicePage.locator('button').filter({ has: alicePage.locator('svg.lucide-plus') })
    await plusBtn.first().click({ force: true })
    await alicePage.waitForTimeout(500)
    await alicePage.getByText('Create Group', { exact: true }).or(alicePage.getByText('创建群组', { exact: true })).click()
    await alicePage.waitForTimeout(500)
    const groupName = `AlreadyMember_${ts}`
    await alicePage.locator('input[placeholder*="Group name"], input[placeholder*="群组名称"]').first().fill(groupName)
    await alicePage.waitForTimeout(300)

    // Search and select Carol
    const searchInput = alicePage.locator('input[placeholder*="搜索成员"]').first()
    await expect(searchInput).toBeVisible({ timeout: 2000 })
    await searchInput.fill('Carol')
    await alicePage.keyboard.press('Enter')
    await alicePage.waitForTimeout(1500)
    await expect(alicePage.getByText('Carol').first()).toBeVisible({ timeout: 3000 })
    await alicePage.getByText('Carol').first().click()
    await alicePage.waitForTimeout(500)
    await expect(alicePage.getByText('已选').first()).toBeVisible({ timeout: 2000 })

    let createBtn = alicePage.locator('button').filter({ hasText: /Create Group|创建群组/ }).last()
    await expect(createBtn).toBeVisible({ timeout: 2000 })
    await createBtn.click()
    await alicePage.waitForFunction(() => {
      const path = window.location.pathname
      return path.startsWith('/conversations/') && path.length > 6
    }, { timeout: 10000 })
    await alicePage.waitForTimeout(1000)

    // Generate share link
    const moreBtn = alicePage.locator('button:has(svg.lucide-ellipsis-vertical)').first()
    await expect(moreBtn).toBeVisible({ timeout: 2000 })
    await moreBtn.click()
    await alicePage.waitForTimeout(500)
    await alicePage.getByText('Basic Info', { exact: true }).or(alicePage.getByText('基本信息', { exact: true })).click()
    await alicePage.waitForTimeout(500)
    await alicePage.getByText('Generate Share Link', { exact: true }).or(alicePage.getByText('生成分享链接', { exact: true })).click()
    await alicePage.waitForTimeout(1000)

    const shareLinkEl = alicePage.locator('.select-all').first()
    await expect(shareLinkEl).toBeVisible({ timeout: 3000 })
    const shareUrl = await shareLinkEl.textContent()
    const shareToken = shareUrl!.split('/').pop()
    expect(shareToken).toBeTruthy()
    await alicePage.close()

    // Carol (already a member) navigates to sharecard
    await page.goto(`/group-card/${shareToken}`)
    await page.waitForTimeout(2000)

    // ✅ Verify: "Join Group" button is shown (Carol is logged in)
    await expect(
      page.getByText('Join Group', { exact: true }).or(page.getByText('加入群组', { exact: true }))
    ).toBeVisible({ timeout: 10000 })

    // Click "Join Group" → confirmation dialog
    await page.getByText('Join Group', { exact: true }).or(page.getByText('加入群组', { exact: true })).click()
    await page.waitForTimeout(500)

    // ✅ Verify: confirmation dialog shows
    await expect(
      page.getByText('Join this group?').or(page.getByText('确认加入该群组？'))
    ).toBeVisible({ timeout: 3000 })

    // Confirm join
    await page.getByText('Confirm', { exact: true }).or(page.getByText('确定', { exact: true })).click()
    await page.waitForTimeout(3000)

    // ✅ Verify: Carol is redirected to the conversations page since she's already a member
    await expect(page).toHaveURL(/\/conversations\//, { timeout: 10000 })

    await ctx.close()
  })
})
