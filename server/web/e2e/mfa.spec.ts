import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ALICE = { account: `mfa_${ts}`, pass: 'test123456' }

test.describe('MFA Settings UI', () => {
  test('register, open privacy, enable TOTP MFA setup, cancel', async ({ page }) => {
    // Register
    await page.goto('/register')
    await page.waitForTimeout(300)
    const inputs = page.locator('input')
    await inputs.nth(0).fill(ALICE.account)
    await inputs.nth(1).fill('MFAUser')
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator('text=MFAUser')).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // ✅ Verify: logged into chat
    await expect(page).toHaveURL(/\/chat/, { timeout: 5000 })

    // Open profile
    await page.locator('text=MFAUser').first().click({ force: true })
    await page.waitForTimeout(1000)

    // Click user settings (Shield icon) in profile → opens privacy view with MFA
    const userSettingsBtn = page.getByText('用户设置').or(page.getByText('User Settings'))
    await expect(userSettingsBtn).toBeVisible({ timeout: 3000 })
    await userSettingsBtn.click()
    await page.waitForTimeout(1000)

    // ✅ Verify: privacy view with MFA section visible
    await expect(page.getByText('多因素认证 (MFA)')).toBeVisible({ timeout: 3000 })

    // Click TOTP setup button
    const totpBtn = page.locator('button:has-text("TOTP")')
    await expect(totpBtn).toBeVisible({ timeout: 2000 })
    await totpBtn.click()
    await page.waitForTimeout(1000)

    // ✅ Verify: TOTP setup UI with secret key and verify code input
    await expect(page.getByPlaceholder('输入 6 位验证码')).toBeVisible({ timeout: 3000 })
    const secretCode = page.locator('code')
    await expect(secretCode).toBeVisible({ timeout: 2000 })
    const secretText = await secretCode.textContent()
    expect(secretText?.length).toBeGreaterThan(0)

    // Close setup
    await page.getByText('取消').click()
    await page.waitForTimeout(500)

    // ✅ Verify: back to setup buttons
    const totpBtnAgain = page.locator('button:has-text("TOTP")')
    await expect(totpBtnAgain).toBeVisible({ timeout: 3000 })
  })
})
