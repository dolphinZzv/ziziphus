import { test, expect } from './fixtures/coverage'

const ts = Date.now()
const ALICE = { account: `alice_${ts}`, name: 'Alice', pass: 'test123456' }

test.describe('Full Browser Flow', () => {
  test('register, login, profile, settings, agent, sessions', async ({ page }) => {
    // ===== Register =====
    await page.goto('/register')
    await page.waitForTimeout(300)
    let inputs = page.locator('input')
    await inputs.nth(0).fill(ALICE.account)
    await inputs.nth(1).fill(ALICE.name)
    await inputs.nth(2).fill('')
    await inputs.nth(3).fill(ALICE.pass)
    await inputs.nth(4).fill(ALICE.pass)
    await page.click('button[type="submit"]')
    await expect(page.locator(`text=${ALICE.name}`)).toBeVisible({ timeout: 15000 })
    await page.waitForTimeout(2000)

    // ✅ Verify: logged in and on chat page
    await expect(page).toHaveURL(/\/chat/, { timeout: 5000 })

    // ===== Open profile from sidebar =====
    await page.waitForTimeout(2000)
    await page.locator(`text=${ALICE.name}`).first().click({ force: true })
    await page.waitForTimeout(1500)

    // ✅ Verify: profile dialog shows agent management link
    await expect(page.getByText('Agent 管理').or(page.getByText('Agent Management'))).toBeVisible({ timeout: 3000 })

    // ===== Open settings from profile =====
    const settingsBtn = page.getByRole('button', { name: 'App Settings' })
    await expect(settingsBtn).toBeVisible({ timeout: 2000 })
    await settingsBtn.click()
    await page.waitForTimeout(1000)

    // ✅ Verify: settings page shows theme option
    await expect(page.getByText('主题').or(page.getByText('Theme'))).toBeVisible({ timeout: 3000 })
    await page.keyboard.press('Escape')
    await page.waitForTimeout(300)

    // ===== Create agent =====
    // Profile is already open (re-opened by route listener after Escape above).
    // No need to click the user name again — doing so with force:true while the
    // profile overlay covers the sidebar would hit the overlay's onClose instead.
    const agentBtn = page.getByText('Agent 管理').or(page.getByText('Agent Management'))
    await expect(agentBtn).toBeVisible({ timeout: 2000 })
    await agentBtn.click()
    await page.waitForTimeout(1000)

    // Click the + button to create a new agent
    const addBtn = page.locator('button:has(svg.lucide-plus)').last()
    await expect(addBtn).toBeVisible({ timeout: 2000 })
    await addBtn.click()
    await page.waitForTimeout(500)

    const agentNameInput = page.getByPlaceholder('Agent 名称')
    await expect(agentNameInput).toBeVisible({ timeout: 2000 })
    const agentName = `Bot_${ts}`
    await agentNameInput.fill(agentName)
    await page.waitForTimeout(300)
    // Click save button
    await page.getByText('保存').click()
    await page.waitForTimeout(1500)

    // ✅ Verify: agent appears in list
    await page.waitForTimeout(1500)
    await expect(page.getByText(agentName)).toBeVisible({ timeout: 5000 })

    // Close agent management dialog
    await page.keyboard.press('Escape')
    await page.waitForTimeout(500)

    // Close profile dialog
    await page.keyboard.press('Escape')
    await page.waitForTimeout(500)

    // ===== Sessions =====
    await page.locator(`text=${ALICE.name}`).first().click({ force: true })
    await page.waitForTimeout(1000)

    const sessionBtn = page.getByText('设备管理').or(page.getByText('Device Management'))
    await expect(sessionBtn).toBeVisible({ timeout: 3000 })
    await sessionBtn.click()
    await page.waitForTimeout(1000)
  })
})
