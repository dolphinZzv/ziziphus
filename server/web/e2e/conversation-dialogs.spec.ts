import { test, expect } from './fixtures/coverage'

const AUTH_INIT = `
  localStorage.setItem('ziziphus_token', JSON.stringify('test-mock-token'));
  localStorage.setItem('ziziphus_user', JSON.stringify({
    user_id: 'user_001', account: 'testuser', name: '测试用户', avatar: '',
    type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
    wake_mode: 0, api_key: '', discoverable: true, allow_direct_chat: true, created_at: 1700000000,
  }));
  localStorage.setItem('ziziphus_theme', JSON.stringify('light'));
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

test.describe('New Chat Dialog', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/chat')
    await page.waitForTimeout(1500)
  })

  test('opens via plus menu', async ({ page }) => {
    // Click plus button
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await expect(page.getByText('新建聊天')).toBeVisible({ timeout: 3000 })
    await page.getByText('新建聊天').click({ force: true })
    await expect(page.getByPlaceholder('搜索用户...')).toBeVisible({ timeout: 3000 })
  })

  test('has search input and close button', async ({ page }) => {
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await page.getByText('新建聊天').click({ force: true })
    await expect(page.getByPlaceholder('搜索用户...')).toBeVisible({ timeout: 3000 })
    // Close via X button
    await page.locator('.fixed.inset-0.z-50 button:has(svg.lucide-x)').first().click({ force: true })
    await page.waitForTimeout(300)
    await expect(page.getByPlaceholder('搜索用户...')).not.toBeVisible()
  })

  test('can be closed by clicking backdrop', async ({ page }) => {
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await page.getByText('新建聊天').click({ force: true })
    await expect(page.getByPlaceholder('搜索用户...')).toBeVisible({ timeout: 3000 })
    // Click backdrop
    const backdrops = page.locator('.fixed.inset-0.z-50')
    await backdrops.first().click({ force: true, position: { x: 10, y: 10 } })
    await page.waitForTimeout(300)
    await expect(page.getByPlaceholder('搜索用户...')).not.toBeVisible()
  })
})

test.describe('Create Group Dialog', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/chat')
    await page.waitForTimeout(1500)
  })

  test('opens via plus menu with group name input', async ({ page }) => {
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await expect(page.getByText('创建群组')).toBeVisible({ timeout: 3000 })
    await page.getByText('创建群组').click({ force: true })
    await expect(page.getByPlaceholder('群组名称')).toBeVisible({ timeout: 3000 })
  })

  test('has member search input', async ({ page }) => {
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await page.getByText('创建群组').click({ force: true })
    await expect(page.getByPlaceholder('搜索成员...')).toBeVisible({ timeout: 3000 })
  })

  test('create button disabled when no name or members', async ({ page }) => {
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await page.getByText('创建群组').click({ force: true })
    await page.waitForTimeout(300)
    const btn = page.locator('button:has-text("创建群组 (")')
    await expect(btn).toBeDisabled()
  })
})

test.describe('Join Group Dialog', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/chat')
    await page.waitForTimeout(1500)
  })

  test('opens via plus menu with group search', async ({ page }) => {
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await expect(page.getByText('加入群组')).toBeVisible({ timeout: 3000 })
    await page.getByText('加入群组').click({ force: true })
    await expect(page.getByPlaceholder('搜索群组...')).toBeVisible({ timeout: 3000 })
  })

  test('has search button', async ({ page }) => {
    await page.locator('button:has(svg.lucide-plus)').first().click({ force: true })
    await page.getByText('加入群组').click({ force: true })
    await expect(page.locator('button:has(svg.lucide-search)')).toBeVisible({ timeout: 3000 })
  })
})
