import { test, expect } from './fixtures/coverage'

const AUTH_INIT = `
  sessionStorage.setItem('ziziphus_token', JSON.stringify('test-mock-token'));
  sessionStorage.setItem('ziziphus_user', JSON.stringify({
    user_id: 'user_001', account: 'testuser', name: '测试用户', avatar: '',
    type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
    wake_mode: 0, api_key: '', created_at: 1700000000,
  }));
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

test.describe('App Layout (authenticated)', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/chat')
    await page.waitForTimeout(1500)
  })

  test('renders sidebar with user name', async ({ page }) => {
    await expect(page.getByText('测试用户').first()).toBeVisible({ timeout: 5000 })
  })

  test('empty chat state shown', async ({ page }) => {
    await expect(page.getByText('选择一个会话开始聊天')).toBeVisible({ timeout: 5000 })
  })

  test('connection status shows', async ({ page }) => {
    await page.waitForSelector('text=/已连接|连接中|连接已断开|恢复中/', { timeout: 5000 })
  })
})
