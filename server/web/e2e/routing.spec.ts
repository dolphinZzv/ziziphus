import { test, expect } from '@playwright/test'

const AUTH_INIT = `
  localStorage.setItem('panda_ai_token', JSON.stringify('test-mock-token'));
  localStorage.setItem('panda_ai_user', JSON.stringify({
    user_id: 'user_001', account: 'testuser', name: 'Test', avatar: '',
    type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
    wake_mode: 0, api_key: '', created_at: 1700000000,
  }));
`

test.describe('Routing & Auth Guard', () => {
  test('redirects to /login when not authenticated', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/login/)
  })

  test('unknown routes redirect to /login', async ({ page }) => {
    await page.goto('/nonexistent')
    await expect(page).toHaveURL(/\/login/)
  })

  test('login route is accessible without auth', async ({ page }) => {
    await page.goto('/login')
    await expect(page).toHaveURL('/login')
    await expect(page.locator('h1')).toHaveText('Panda AI')
  })

  test('register route is accessible without auth', async ({ page }) => {
    await page.goto('/register')
    await expect(page).toHaveURL('/register')
    await expect(page.getByPlaceholder('昵称')).toBeVisible()
  })

  test('redirects /chat to /login when no auth', async ({ page }) => {
    await page.goto('/chat')
    await expect(page).toHaveURL('/login')
  })

  test('/login redirects to /chat when cached auth exists', async ({ page }) => {
    await page.addInitScript(AUTH_INIT)
    await page.goto('/login')
    await expect(page).toHaveURL('/chat')
  })
})
