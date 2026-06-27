import { test, expect } from '@playwright/test'

const AUTH_INIT = `
  localStorage.setItem('panda_ai_token', JSON.stringify('test-mock-token'));
  localStorage.setItem('panda_ai_user', JSON.stringify({
    user_id: 'user_001', account: 'testuser', name: '测试用户', avatar: '',
    type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
    wake_mode: 0, api_key: '', created_at: 1700000000,
  }));
`

test.describe('Chat UI', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/chat/conv_test_001')
    await page.waitForTimeout(800)
  })

  test('chat toolbar shows conversation id', async ({ page }) => {
    await expect(page.locator('span').filter({ hasText: 'conv_test_001' }).first()).toBeVisible({ timeout: 5000 })
  })

  test('input bar renders with all buttons', async ({ page }) => {
    await expect(page.getByPlaceholder('输入消息...')).toBeVisible({ timeout: 5000 })
    await expect(page.locator('button:has(svg.lucide-paperclip)')).toBeVisible()
    await expect(page.locator('button:has(svg.lucide-at-sign)')).toBeVisible()
    await expect(page.locator('button:has(svg.lucide-send)')).toBeVisible()
  })

  test('send button disabled when empty', async ({ page }) => {
    const sendBtn = page.locator('button:has(svg.lucide-send)')
    await expect(sendBtn).toBeVisible({ timeout: 5000 })
    await expect(sendBtn).toBeDisabled()
  })

  test('send button enabled with text', async ({ page }) => {
    const input = page.getByPlaceholder('输入消息...')
    await input.fill('Hello')
    const sendBtn = page.locator('button:has(svg.lucide-send)')
    await expect(sendBtn).not.toBeDisabled()
  })
})
