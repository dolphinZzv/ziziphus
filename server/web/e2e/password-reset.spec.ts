import { test, expect } from './fixtures/coverage'

const ZH_INIT = `
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

const ts = Date.now()
const ACCOUNT = `pr_${ts}`
const PASSWORD = 'oldpassword123'
const NEW_PASSWORD = 'newpassword456'
const EMAIL = `pr_${ts}@example.com`

test.describe('Password Reset', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(ZH_INIT)
  })

  test('register user then reset password via forgot-password flow', async ({ page }) => {
    // ===== Register a user first =====
    await page.goto('/register')
    await page.waitForTimeout(300)
    const regInputs = page.locator('input')
    await regInputs.nth(0).fill(ACCOUNT)
    await regInputs.nth(1).fill('PWResetUser')
    await regInputs.nth(2).fill(EMAIL)
    await regInputs.nth(3).fill(PASSWORD)
    await regInputs.nth(4).fill(PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=PWResetUser').first()).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/conversations/, { timeout: 5000 })

    // ===== Logout by clearing session =====
    await page.evaluate(() => {
      sessionStorage.clear()
      localStorage.clear()
    })
    await page.goto('/login')
    await page.waitForTimeout(500)

    // ===== Navigate to forgot password =====
    await page.getByText('忘记密码？').click()
    await page.waitForTimeout(500)
    await expect(page).toHaveURL('/forgot-password')

    // ===== Enter account for password reset =====
    await expect(page.getByText('找回密码')).toBeVisible()
    const accountInput = page.getByPlaceholder('账号或邮箱')
    await accountInput.fill(ACCOUNT)

    // Intercept the API response to get the reset code
    const responsePromise = page.waitForResponse(
      resp => resp.url().includes('/api/v1/users/password-reset/send-code') && resp.status() === 200
    )

    // Click send code button
    await page.getByText('发送验证码').click()
    await page.waitForTimeout(1000)

    // Wait for the API response
    const response = await responsePromise
    const body = await response.json()
    expect(body.data).toBeDefined()
    expect(body.data.user_id).toBeDefined()

    // Reset code is only returned with ?dev=1 query param.
    // Make a dev API call to get the reset code.
    const devResponse = await page.evaluate(async (uid) => {
      const resp = await fetch('/api/v1/users/password-reset/send-code?dev=1', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ account_or_email: uid }),
      })
      const data = await resp.json()
      return data.data?.code || ''
    }, ACCOUNT)

    expect(devResponse).toBeDefined()
    expect(devResponse.length).toBeGreaterThan(0)
    const resetCode = devResponse

    // ===== Enter reset code and new password =====
    const codeInput = page.getByPlaceholder('验证码')
    await expect(codeInput).toBeVisible({ timeout: 3000 })
    await codeInput.fill(resetCode)

    const newPassInput = page.getByPlaceholder('新密码')
    await newPassInput.fill(NEW_PASSWORD)

    const confirmInput = page.getByPlaceholder('确认密码')
    await confirmInput.fill(NEW_PASSWORD)

    // Submit reset
    await page.getByText('重置密码').click()
    await page.waitForTimeout(1000)

    // ===== Verify success =====
    await expect(page.getByText('密码重置成功')).toBeVisible({ timeout: 5000 })

    // ===== Navigate back to login =====
    await page.getByText('返回登录').click()
    await page.waitForTimeout(500)
    await expect(page).toHaveURL('/login')

    // ===== Login with new password =====
    await page.getByPlaceholder('账号').fill(ACCOUNT)
    await page.getByPlaceholder('密码').fill(NEW_PASSWORD)
    await page.getByText('登录').click()
    await expect(page.locator('text=PWResetUser').first()).toBeVisible({ timeout: 15000 })
    await expect(page).toHaveURL(/\/chat/, { timeout: 5000 })
  })
})
