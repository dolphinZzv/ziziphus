import { test, expect } from './fixtures/coverage'

const ZH_INIT = `
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

const ts = Date.now()
const ACCOUNT = `jwt_${ts}`
const PASSWORD = 'test123456'

test.describe('JWT Auto-Refresh', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(ZH_INIT)
  })

  test('register, corrupt access token, verify auto-refresh keeps session alive', async ({ page }) => {
    // ===== Register a user =====
    await page.goto('/register')
    await page.waitForTimeout(300)
    const regInputs = page.locator('input')
    await regInputs.nth(0).fill(ACCOUNT)
    await regInputs.nth(1).fill('JWTUser')
    await regInputs.nth(2).fill('')
    await regInputs.nth(3).fill(PASSWORD)
    await regInputs.nth(4).fill(PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=JWTUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/conversations/, { timeout: 5000 })

    // ===== Verify we have a real token stored =====
    const realToken = await page.evaluate(() => sessionStorage.getItem('ziziphus_token'))
    expect(realToken).toBeTruthy()
    const realRefresh = await page.evaluate(() => sessionStorage.getItem('ziziphus_refresh_token'))
    expect(realRefresh).toBeTruthy()

    // ===== Corrupt the access token to force 401 =====
    await page.evaluate(() => sessionStorage.setItem('ziziphus_token', 'bad_token_to_trigger_401'))

    // ===== Navigate — triggers API call → 401 → refresh_token → new token → retry =====
    await page.goto('/conversations')
    await page.waitForTimeout(3000)

    // ===== Verify token was refreshed (new token != the garbage we set) =====
    const newToken = await page.evaluate(() => sessionStorage.getItem('ziziphus_token'))
    expect(newToken).toBeTruthy()
    expect(newToken).not.toBe('bad_token_to_trigger_401')

    // ===== Verify still logged in (not redirected to login) =====
    const currentUrl = page.url()
    expect(currentUrl).not.toContain('/login')
    await expect(page.locator('text=JWTUser').first()).toBeVisible({ timeout: 5000 })
  })

  test('clear all tokens, verify redirect to login', async ({ page }) => {
    // ===== Register =====
    const acc2 = `jwt2_${ts}`
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(acc2)
    await inputs.nth(1).fill('JWTUser2')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(PASSWORD)
    await inputs.nth(4).fill(PASSWORD)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=JWTUser2')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)
    await expect(page).toHaveURL(/\/conversations/, { timeout: 5000 })

    // ===== Clear ALL tokens (both access and refresh) =====
    await page.evaluate(() => {
      sessionStorage.clear()
      localStorage.clear()
    })

    // ===== Navigate — should redirect to login since no refresh_token =====
    await page.goto('/conversations')
    await page.waitForTimeout(3000)
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 })
  })
})
