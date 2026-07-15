import { test, expect } from './fixtures/coverage'

const ZH_INIT = `
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(ZH_INIT)
    await page.goto('/login')
    await page.waitForTimeout(500)
  })

  test('renders login form correctly', async ({ page }) => {
    await expect(page.locator('h1')).toHaveText('Ziziphus')
    await expect(page.getByPlaceholder('账号')).toBeVisible()
    await expect(page.getByPlaceholder('密码')).toBeVisible()
    await expect(page.getByText('记住账号')).toBeVisible()
    await expect(page.getByRole('button', { name: '登录' })).toBeVisible()
    await expect(page.getByText('创建新账号')).toBeVisible()
  })

  test('password visibility toggle works', async ({ page }) => {
    const passwordInput = page.getByPlaceholder('密码')
    await expect(passwordInput).toHaveAttribute('type', 'password')
    const toggleBtn = page.locator('button:has(svg.lucide-eye)').first()
    await toggleBtn.click()
    await expect(passwordInput).toHaveAttribute('type', 'text')
  })

  test('login attempts to contact server', async ({ page }) => {
    await page.getByPlaceholder('账号').fill('nonexistent_user')
    await page.getByPlaceholder('密码').fill('wrong_password')
    await page.getByRole('button', { name: '登录' }).click()
    await expect(page.getByRole('button', { name: '登录' })).toBeVisible({ timeout: 10000 })
  })

  test('navigates to register page', async ({ page }) => {
    await page.getByText('创建新账号').click()
    await expect(page).toHaveURL('/register')
    await expect(page.getByPlaceholder('昵称')).toBeVisible()
    await expect(page.getByPlaceholder('确认密码')).toBeVisible()
  })

  test('remember account checkbox toggles', async ({ page }) => {
    const checkbox = page.getByRole('checkbox')
    await expect(checkbox).toBeChecked()
    await checkbox.uncheck()
    await expect(checkbox).not.toBeChecked()
    await checkbox.check()
    await expect(checkbox).toBeChecked()
  })

  test('has theme and language footer', async ({ page }) => {
    await expect(page.getByText('中文')).toBeVisible()
    await expect(page.getByText('EN')).toBeVisible()
  })
})

test.describe('Register Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(ZH_INIT)
    await page.goto('/register')
    await page.waitForTimeout(500)
  })

  test('renders register form correctly', async ({ page }) => {
    await expect(page.getByPlaceholder('账号')).toBeVisible()
    await expect(page.getByPlaceholder('昵称')).toBeVisible()
    await expect(page.getByPlaceholder('密码').first()).toBeVisible()
    await expect(page.getByPlaceholder('确认密码')).toBeVisible()
    await expect(page.getByRole('button', { name: '注册' })).toBeVisible()
    await expect(page.getByText('已有账号？去登录')).toBeVisible()
  })

  test('validates password mismatch', async ({ page }) => {
    await page.getByPlaceholder('账号').fill('testuser')
    await page.getByPlaceholder('昵称').fill('Test')
    await page.getByPlaceholder('密码').first().fill('password123')
    await page.getByPlaceholder('确认密码').fill('different')
    await page.getByRole('button', { name: '注册' }).click()
    await expect(page.getByText('两次密码不一致')).toBeVisible()
  })

  test('validates empty fields', async ({ page }) => {
    await page.getByRole('button', { name: '注册' }).click()
    await expect(page.getByText('请填写所有字段')).toBeVisible()
  })

  test('navigates back to login', async ({ page }) => {
    await page.getByText('已有账号？去登录').click()
    await expect(page).toHaveURL('/login')
  })
})
