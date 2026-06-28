import { test, expect } from './fixtures/coverage'

const AUTH_INIT = `
  localStorage.setItem('panda_ai_token', JSON.stringify('test-mock-token'));
  localStorage.setItem('panda_ai_user', JSON.stringify({
    user_id: 'user_001', account: 'testuser', name: '测试用户', avatar: '',
    type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
    wake_mode: 0, api_key: '', created_at: 1700000000,
  }));
  localStorage.setItem('panda_ai_theme', JSON.stringify('light'));
  localStorage.setItem('panda_ai_language', JSON.stringify('zh'));
`

test.describe('Sheets & Dialogs (authenticated)', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/chat')
    await page.waitForTimeout(1500)
  })

  test('settings dialog opens and shows content', async ({ page }) => {
    const settingsBtn = page.locator('button[title="设置"]')
    await expect(settingsBtn).toBeVisible({ timeout: 5000 })
    await settingsBtn.click({ force: true })
    await page.waitForTimeout(400)
    await expect(page.getByText('主题')).toBeVisible({ timeout: 3000 })
    await expect(page.getByText('语言')).toBeVisible()
    await expect(page.getByText('气泡颜色')).toBeVisible()
  })

  test('profile dialog opens from sidebar name click', async ({ page }) => {
    await expect(page.getByText('测试用户').first()).toBeVisible({ timeout: 5000 })
    await page.getByText('测试用户').first().click({ force: true })
    await page.waitForTimeout(400)
    await expect(page.getByText('Agent 管理')).toBeVisible({ timeout: 3000 })
  })

  test('agent management opens from sidebar bot button', async ({ page }) => {
    const btn = page.locator('button[title="Agent"]')
    await expect(btn).toBeVisible({ timeout: 5000 })
    await btn.click({ force: true })
    await page.waitForTimeout(400)
    await expect(page.getByText('Agent 管理')).toBeVisible({ timeout: 3000 })
  })

  test('session management opens from sidebar smartphone button', async ({ page }) => {
    const btn = page.locator('button[title="设备管理"]')
    await expect(btn).toBeVisible({ timeout: 5000 })
    await btn.click({ force: true })
    await page.waitForTimeout(400)
    await expect(page.getByText('设备管理')).toBeVisible({ timeout: 3000 })
  })
})
