import { test, expect } from './fixtures/coverage'

test('register page has email input', async ({ page }) => {
  await page.addInitScript(`localStorage.setItem('ziziphus_language', JSON.stringify('zh'));`)
  await page.goto('/register')
  await page.waitForTimeout(1000)
  await expect(page.locator('input[type="email"]')).toBeVisible({ timeout: 5000 })
})

test('profile page buttons visible', async ({ page }) => {
  const ts = Date.now()
  const r = await page.request.post('http://localhost:8080/api/v1/users/register', {
    data: { account: `prof_${ts}`, name: 'ProfTest', password: 'test123456' },
  })
  const d = await r.json()
  const { token, user_id: uid } = d.data

  await page.addInitScript(`(() => {
    sessionStorage.setItem('ziziphus_token', JSON.stringify('${token}'));
    sessionStorage.setItem('ziziphus_user', JSON.stringify({
      user_id: '${uid}', account: 'prof_${ts}', name: 'ProfTest', avatar: '',
      type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
      wake_mode: 0, api_key: '', discoverable: true, allow_direct_chat: true, created_at: Date.now(),
    }));
    localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
  })()`)

  await page.goto('/chat')
  await page.waitForTimeout(2000)

  // Sidebar should show the user name
  await expect(page.getByText('ProfTest')).toBeVisible({ timeout: 5000 })

  // Click on user name in sidebar to open profile
  await page.getByText('ProfTest').first().click()
  await page.waitForTimeout(1000)

  // Profile should show action buttons
  await expect(page.getByText('Agent 管理')).toBeVisible({ timeout: 5000 })
  await expect(page.getByText('用户设置')).toBeVisible()
  await expect(page.getByText('设备管理')).toBeVisible()
  await expect(page.getByText('应用设置')).toBeVisible()
})

test('user settings has email input', async ({ page }) => {
  const ts = Date.now()
  const r = await page.request.post('http://localhost:8080/api/v1/users/register', {
    data: { account: `set_${ts}`, name: 'SetTest', password: 'test123456' },
  })
  const d = await r.json()
  const { token, user_id: uid } = d.data

  await page.addInitScript(`(() => {
    sessionStorage.setItem('ziziphus_token', JSON.stringify('${token}'));
    sessionStorage.setItem('ziziphus_user', JSON.stringify({
      user_id: '${uid}', account: 'set_${ts}', name: 'SetTest', avatar: '',
      type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
      wake_mode: 0, api_key: '', discoverable: true, allow_direct_chat: true, created_at: Date.now(),
    }));
    localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
  })()`)

  await page.goto('/chat')
  await page.waitForTimeout(2000)

  // Open profile
  await page.getByText('SetTest').first().click()
  await page.waitForTimeout(800)

  // Click user settings
  await page.getByText('用户设置').first().click()
  await page.waitForTimeout(1500)

  // Now in user settings — check email input
  const emailInput = page.getByPlaceholder('email@example.com')
  await expect(emailInput).toBeVisible({ timeout: 5000 })
})
