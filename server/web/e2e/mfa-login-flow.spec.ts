import { test, expect } from './fixtures/coverage'

const API = 'http://localhost:8080'

test.describe('MFA Login Flow E2E', () => {
  let token = ''
  let secret = ''
  let mfaUserId = ''
  const ts = Date.now()

  test('register, setup TOTP, enable MFA', async ({ request }) => {
    const r = await request.post(`${API}/api/v1/users/register`, {
      data: { account: `mfaflow_${ts}`, name: 'MFAFlow', password: 'test123456' },
    })
    const d = await r.json()
    expect(d.code).toBe(0)
    token = d.data.token

    const r2 = await request.post(`${API}/api/v1/users/me/mfa/setup`, {
      data: { mfa_type: 1 },
      headers: { Authorization: `Bearer ${token}` },
    })
    secret = (await r2.json()).data.secret
    expect(secret).toBeTruthy()
  })

  test('verify MFA with TOTP code via page evaluate', async ({ page }) => {
    // Use page context to generate TOTP
    const code = await page.evaluate((s) => {
      const base32 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
      const upper = s.toUpperCase()
      let bits = ''
      for (const ch of upper) {
        const idx = base32.indexOf(ch)
        if (idx >= 0) bits += idx.toString(2).padStart(5, '0')
      }
      const key = new Uint8Array(Math.floor(bits.length / 8))
      for (let i = 0; i < key.length; i++) key[i] = parseInt(bits.slice(i * 8, i * 8 + 8), 2)

      const counter = BigInt(Math.floor(Date.now() / 30000))
      const msg = new Uint8Array(8)
      for (let i = 7; i >= 0; i--) msg[i] = Number((counter >> BigInt((7 - i) * 8)) & BigInt(0xff))

      return crypto.subtle.importKey('raw', key, { name: 'HMAC', hash: 'SHA-1' }, false, ['sign'])
        .then(k => crypto.subtle.sign('HMAC', k, msg))
        .then(sig => {
          const hash = new Uint8Array(sig)
          const offset = hash[hash.length - 1] & 0x0f
          const binary = ((hash[offset] & 0x7f) << 24) | (hash[offset + 1] << 16) | (hash[offset + 2] << 8) | hash[offset + 3]
          return (binary % 1000000).toString().padStart(6, '0')
        })
    }, secret)
    expect(code).toMatch(/^\d{6}$/)

    // Verify to enable MFA
    const r = await request.post(`${API}/api/v1/users/me/mfa/verify`, {
      data: { code },
      headers: { Authorization: `Bearer ${token}` },
    })
    expect((await r.json()).code).toBe(0)
  })

  test('login shows MFA code input', async ({ page }) => {
    await page.addInitScript(`localStorage.setItem('ziziphus_language', JSON.stringify('zh'));`)
    await page.goto('/login')
    await page.waitForTimeout(500)

    // Login with credentials
    await page.getByPlaceholder('账号').fill(`mfaflow_${ts}`)
    await page.getByPlaceholder('密码').fill('test123456')
    await page.getByRole('button', { name: '登录' }).click()
    await page.waitForTimeout(1000)

    // Should now show MFA code input
    const mfaInput = page.locator('input[placeholder*="验证码"]')
    await expect(mfaInput).toBeVisible({ timeout: 5000 })

    // Generate TOTP code and fill
    const code = await page.evaluate((s) => {
      const base32 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
      const upper = s.toUpperCase()
      let bits = ''
      for (const ch of upper) {
        const idx = base32.indexOf(ch)
        if (idx >= 0) bits += idx.toString(2).padStart(5, '0')
      }
      const key = new Uint8Array(Math.floor(bits.length / 8))
      for (let i = 0; i < key.length; i++) key[i] = parseInt(bits.slice(i * 8, i * 8 + 8), 2)

      const counter = BigInt(Math.floor(Date.now() / 30000))
      const msg = new Uint8Array(8)
      for (let i = 7; i >= 0; i--) msg[i] = Number((counter >> BigInt((7 - i) * 8)) & BigInt(0xff))

      return crypto.subtle.importKey('raw', key, { name: 'HMAC', hash: 'SHA-1' }, false, ['sign'])
        .then(k => crypto.subtle.sign('HMAC', k, msg))
        .then(sig => {
          const hash = new Uint8Array(sig)
          const offset = hash[hash.length - 1] & 0x0f
          const binary = ((hash[offset] & 0x7f) << 24) | (hash[offset + 1] << 16) | (hash[offset + 2] << 8) | hash[offset + 3]
          return (binary % 1000000).toString().padStart(6, '0')
        })
    }, secret)
    expect(code).toMatch(/^\d{6}$/)

    await mfaInput.fill(code)
    await page.getByRole('button', { name: '验证' }).click()

    // After verification, should redirect to chat
    await page.waitForTimeout(2000)
    expect(page.url()).toContain('/chat')
  })
})
