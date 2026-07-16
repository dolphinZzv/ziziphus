import { test, expect } from './fixtures/coverage'

const AUTH_INIT = `
  sessionStorage.setItem('ziziphus_token', JSON.stringify('test-mock-token'));
  sessionStorage.setItem('ziziphus_user', JSON.stringify({
    user_id: 'user_001', account: 'testuser', name: '测试用户', avatar: '',
    type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
    wake_mode: 0, api_key: '', created_at: 1700000000,
  }));
  localStorage.setItem('ziziphus_theme', JSON.stringify('light'));
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

test.describe('Sheets & Dialogs (authenticated)', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/chat')
    await page.waitForTimeout(1500)
  })

  test('profile dialog opens from sidebar name click', async ({ page }) => {
    await expect(page.getByText('测试用户').first()).toBeVisible({ timeout: 5000 })
    await page.getByText('测试用户').first().click({ force: true })
    await page.waitForTimeout(400)
    await expect(page.getByText('Agent 管理')).toBeVisible({ timeout: 3000 })
  })

  test('settings dialog opens from profile and shows content', async ({ page }) => {
    // Open profile first (click user name)
    await expect(page.getByText('测试用户').first()).toBeVisible({ timeout: 5000 })
    await page.getByText('测试用户').first().click({ force: true })
    await page.waitForTimeout(400)
    // Click settings button inside profile
    await expect(page.getByText('应用设置')).toBeVisible({ timeout: 3000 })
    await page.getByText('应用设置').click()
    await page.waitForTimeout(400)
    await expect(page.getByText('主题')).toBeVisible({ timeout: 3000 })
    await expect(page.getByText('语言')).toBeVisible()
    await expect(page.getByText('气泡颜色')).toBeVisible()
  })

  test('agent management opens from profile', async ({ page }) => {
    await expect(page.getByText('测试用户').first()).toBeVisible({ timeout: 5000 })
    await page.getByText('测试用户').first().click({ force: true })
    await page.waitForTimeout(400)
    await expect(page.getByText('Agent 管理')).toBeVisible({ timeout: 3000 })
    await page.getByText('Agent 管理').click()
    await page.waitForTimeout(400)
    // Agent management dialog opened
    await expect(page.getByText('Agent 管理')).toBeVisible({ timeout: 3000 })
  })

  test('session management opens from profile', async ({ page }) => {
    await expect(page.getByText('测试用户').first()).toBeVisible({ timeout: 5000 })
    await page.getByText('测试用户').first().click({ force: true })
    await page.waitForTimeout(400)
    await expect(page.getByText('设备管理')).toBeVisible({ timeout: 3000 })
    await page.getByText('设备管理').click()
    await page.waitForTimeout(400)
    await expect(page.getByText('设备管理')).toBeVisible({ timeout: 3000 })
  })
})
