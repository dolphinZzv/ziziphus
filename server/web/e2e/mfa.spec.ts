import { test, expect } from './fixtures/coverage'

test('MFA TOTP section visible in user settings', async ({ page }) => {
  const ts = Date.now()
  const r = await page.request.post('http://localhost:8080/api/v1/users/register', {
    data: { account: `mfa_ui_${ts}`, name: 'MFATest', password: 'test123456' },
  })
  const d = await r.json()
  const { token, user_id: uid } = d.data

  await page.addInitScript(`(() => {
    localStorage.setItem('panda_ai_token', JSON.stringify('${token}'));
    localStorage.setItem('panda_ai_user', JSON.stringify({
      user_id: '${uid}', account: 'mfa_ui_${ts}', name: 'MFATest', avatar: '',
      type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
      wake_mode: 0, api_key: '', discoverable: true, allow_direct_chat: true, created_at: Date.now(),
    }));
    localStorage.setItem('panda_ai_language', JSON.stringify('zh'));
  })()`)

  await page.goto('/chat')
  await page.waitForTimeout(2000)

  // Open profile
  await page.getByText('MFATest').first().click()
  await page.waitForTimeout(800)

  // Click user settings
  await page.getByText('用户设置').first().click()
  await page.waitForTimeout(1500)

  // MFA section visible
  await expect(page.getByText('多因素认证')).toBeVisible()
  await expect(page.getByText('开启 TOTP 认证')).toBeVisible()
  await expect(page.getByText('开启邮箱验证码认证')).toBeVisible()
})
