import { test, expect } from './fixtures/coverage'

const ZH_INIT = `
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

test.describe('Auth Page — sliding panels', () => {
  test.describe('Login panel', () => {
    test.beforeEach(async ({ page }) => {
      await page.addInitScript(ZH_INIT)
      await page.goto('/login')
      await page.waitForTimeout(500)
    })

    test('renders login form correctly', async ({ page }) => {
      await expect(page.locator('h1')).toHaveText('Ziziphus')
      await expect(page.getByPlaceholder('账号').first()).toBeVisible()
      await expect(page.getByPlaceholder('密码').first()).toBeVisible()
      await expect(page.getByRole('button', { name: '登录' })).toBeVisible()
      await expect(page.getByRole('button', { name: '注册' })).toBeVisible()
    })

    test('password visibility toggle works', async ({ page }) => {
      const passwordInput = page.getByPlaceholder('密码').first()
      await expect(passwordInput).toHaveAttribute('type', 'password')
      const toggleBtn = page.locator('button:has(svg.lucide-eye)').first()
      await toggleBtn.click()
      await expect(passwordInput).toHaveAttribute('type', 'text')
    })

    test('validates empty account on login', async ({ page }) => {
      await page.getByPlaceholder('密码').first().fill('some_password')
      await page.getByRole('button', { name: '登录' }).click()
      await expect(page.getByText('请填写账号')).toBeVisible()
    })

    test('validates empty password on login', async ({ page }) => {
      await page.getByPlaceholder('账号').first().fill('testuser')
      await page.getByRole('button', { name: '登录' }).click()
      await expect(page.getByText('请填写密码')).toBeVisible()
    })

    test('has theme and language footer', async ({ page }) => {
      await expect(page.getByText('中文')).toBeVisible()
      await expect(page.getByText('EN')).toBeVisible()
    })
  })

  test.describe('Sliding transitions', () => {
    test.beforeEach(async ({ page }) => {
      await page.addInitScript(ZH_INIT)
      await page.goto('/login')
      await page.waitForTimeout(500)
    })

    test('slides from login to register', async ({ page }) => {
      await page.getByRole('button', { name: '注册' }).click()
      await expect(page).toHaveURL('/register')
      // Register form fields should now be visible
      await expect(page.getByPlaceholder('昵称')).toBeVisible({ timeout: 3000 })
      await expect(page.getByPlaceholder('确认密码').first()).toBeVisible()
    })

    test('slides from register back to login', async ({ page }) => {
      await page.getByRole('button', { name: '注册' }).click()
      await expect(page).toHaveURL('/register')
      await page.getByRole('button', { name: '登录' }).click()
      await expect(page).toHaveURL('/login')
      await expect(page.getByPlaceholder('账号').first()).toBeVisible()
    })

    test('slides from login to forgot password', async ({ page }) => {
      await page.getByRole('button', { name: '忘记密码？' }).click()
      await expect(page).toHaveURL('/login') // URL stays same, panel slides internally
      await expect(page.getByPlaceholder('账号或邮箱')).toBeVisible({ timeout: 3000 })
      await expect(page.getByRole('button', { name: '发送验证码' })).toBeVisible()
    })

    test('back from forgot password returns to login', async ({ page }) => {
      await page.getByRole('button', { name: '忘记密码？' }).click()
      await expect(page.getByPlaceholder('账号或邮箱')).toBeVisible({ timeout: 3000 })
      await page.getByRole('button', { name: '返回登录' }).click()
      await expect(page.getByPlaceholder('账号').first()).toBeVisible({ timeout: 3000 })
    })
  })

  test.describe('Register panel', () => {
    test.beforeEach(async ({ page }) => {
      await page.addInitScript(ZH_INIT)
      await page.goto('/register')
      await page.waitForTimeout(500)
    })

    test('/register loads register panel', async ({ page }) => {
      await expect(page.getByPlaceholder('账号').first()).toBeVisible()
      await expect(page.getByPlaceholder('昵称')).toBeVisible()
      await expect(page.getByPlaceholder('密码').first().first()).toBeVisible()
      await expect(page.getByPlaceholder('确认密码').first()).toBeVisible()
    })

    test('validates password mismatch', async ({ page }) => {
      await page.getByPlaceholder('账号').first().fill('testuser')
      await page.getByPlaceholder('昵称').fill('Test')
      await page.getByPlaceholder('密码').first().first().fill('password123')
      await page.getByPlaceholder('确认密码').first().fill('different')
      await page.getByRole('button', { name: '注册' }).click()
      await expect(page.getByText('两次密码不一致')).toBeVisible()
    })

    test('validates empty account on register', async ({ page }) => {
      await page.getByRole('button', { name: '注册' }).click()
      await expect(page.getByText('请填写账号')).toBeVisible()
    })

    test('validates empty name on register', async ({ page }) => {
      await page.getByPlaceholder('账号').first().fill('test')
      await page.getByRole('button', { name: '注册' }).click()
      await expect(page.getByText('请填写昵称')).toBeVisible()
    })

    test('validates empty password on register', async ({ page }) => {
      await page.getByPlaceholder('账号').first().fill('test')
      await page.getByPlaceholder('昵称').fill('Test')
      await page.getByRole('button', { name: '注册' }).click()
      await expect(page.getByText('请填写密码')).toBeVisible()
    })

    test('validates short password on register', async ({ page }) => {
      await page.getByPlaceholder('账号').first().fill('testuser')
      await page.getByPlaceholder('昵称').fill('Test')
      await page.getByPlaceholder('密码').first().first().fill('short')
      await page.getByPlaceholder('确认密码').first().fill('short')
      await page.getByRole('button', { name: '注册' }).click()
      await expect(page.getByText('密码至少8位')).toBeVisible()
    })
  })

  test.describe('Forgot password flow', () => {
    test.beforeEach(async ({ page }) => {
      await page.addInitScript(ZH_INIT)
      // Intercept send-code API
      await page.route('**/api/v1/users/password-reset/send-code', async route => {
        await route.fulfill({ status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, msg: 'ok', data: { user_id: 'user_reset_001' } }) })
      })
    })

    test('request code panel renders and validates', async ({ page }) => {
      await page.goto('/forgot-password')
      await page.waitForTimeout(500)
      await expect(page.getByPlaceholder('账号或邮箱')).toBeVisible()
      await expect(page.getByRole('button', { name: '发送验证码' })).toBeVisible()
      // Empty validation
      await page.getByRole('button', { name: '发送验证码' }).click()
      await expect(page.getByText('请填写账号或邮箱')).toBeVisible()
    })

    test('sends code and advances to reset panel', async ({ page }) => {
      await page.goto('/forgot-password')
      await page.waitForTimeout(500)
      await page.getByPlaceholder('账号或邮箱').fill('testuser@example.com')
      await page.getByRole('button', { name: '发送验证码' }).click()
      // Should advance to reset password panel
      await expect(page.getByPlaceholder('验证码')).toBeVisible({ timeout: 5000 })
      await expect(page.getByPlaceholder('新密码')).toBeVisible()
      await expect(page.getByRole('button', { name: '重置密码' })).toBeVisible()
    })

    test('reset password validates empty fields', async ({ page }) => {
      await page.goto('/login')
      await page.waitForTimeout(500)
      await page.getByRole('button', { name: '忘记密码？' }).click()
      await page.getByPlaceholder('账号或邮箱').fill('testuser@example.com')
      await page.getByRole('button', { name: '发送验证码' }).click()
      await page.waitForTimeout(300)
      await page.getByRole('button', { name: '重置密码' }).click()
      await expect(page.getByText('请填写所有字段')).toBeVisible()
    })

    test('reset password validates mismatch', async ({ page }) => {
      await page.goto('/login')
      await page.waitForTimeout(500)
      await page.getByRole('button', { name: '忘记密码？' }).click()
      await page.getByPlaceholder('账号或邮箱').fill('testuser@example.com')
      await page.getByRole('button', { name: '发送验证码' }).click()
      await page.waitForTimeout(300)
      await page.getByPlaceholder('验证码').fill('123456')
      await page.getByPlaceholder('新密码').fill('newpassword123')
      await page.getByPlaceholder('确认密码').first().fill('different')
      await page.getByRole('button', { name: '重置密码' }).click()
      await expect(page.getByText('两次密码不一致')).toBeVisible()
    })

    test('reset password validates short password', async ({ page }) => {
      await page.goto('/login')
      await page.waitForTimeout(500)
      await page.getByRole('button', { name: '忘记密码？' }).click()
      await page.getByPlaceholder('账号或邮箱').fill('testuser@example.com')
      await page.getByRole('button', { name: '发送验证码' }).click()
      await page.waitForTimeout(300)
      await page.getByPlaceholder('验证码').fill('123456')
      await page.getByPlaceholder('新密码').fill('short')
      await page.getByPlaceholder('确认密码').first().fill('short')
      await page.getByRole('button', { name: '重置密码' }).click()
      await expect(page.getByText('密码至少8位')).toBeVisible()
    })

    test('completes reset and shows success, then back to login', async ({ page }) => {
      // Intercept reset API
      await page.route('**/api/v1/users/password-reset/reset', async route => {
        await route.fulfill({ status: 200, contentType: 'application/json',
          body: JSON.stringify({ code: 0, msg: 'ok', data: { status: 'ok' } }) })
      })
      await page.goto('/login')
      await page.waitForTimeout(500)
      await page.getByRole('button', { name: '忘记密码？' }).click()
      await page.getByPlaceholder('账号或邮箱').fill('testuser@example.com')
      await page.getByRole('button', { name: '发送验证码' }).click()
      await page.waitForTimeout(300)
      await page.getByPlaceholder('验证码').fill('123456')
      await page.getByPlaceholder('新密码').fill('newpassword123')
      await page.getByPlaceholder('确认密码').first().fill('newpassword123')
      await page.getByRole('button', { name: '重置密码' }).click()
      // Should show success panel
      await expect(page.getByText('密码重置成功')).toBeVisible({ timeout: 5000 })
      await page.getByRole('button', { name: '返回登录' }).click()
      await expect(page.getByPlaceholder('账号').first()).toBeVisible({ timeout: 3000 })
    })

    test('resend goes back to request panel', async ({ page }) => {
      await page.goto('/login')
      await page.waitForTimeout(500)
      await page.getByRole('button', { name: '忘记密码？' }).click()
      await page.getByPlaceholder('账号或邮箱').fill('testuser@example.com')
      await page.getByRole('button', { name: '发送验证码' }).click()
      await page.waitForTimeout(300)
      // Click "重新发送验证码" to go back
      await page.getByRole('button', { name: '重新发送验证码' }).click()
      await expect(page.getByRole('button', { name: '发送验证码' })).toBeVisible({ timeout: 3000 })
    })
  })
})
