import { test, expect } from './fixtures/coverage'

const API = 'http://localhost:8080'

test.describe('Email MFA Flow', () => {
  let secondCode = ''
  const ts = Date.now()

  test('full email MFA flow via API', async ({ request }) => {
    // Register
    const r = await request.post(`${API}/api/v1/users/register`, {
      data: { account: `emf_${ts}`, name: 'EmailMFA', password: 'test123456', email: 'cafebabe_2019@qq.com' },
    })
    const d = await r.json()
    expect(d.code).toBe(0)
    const token = d.data.token
    const userId = d.data.user_id

    // Setup email MFA (code returned in dev mode)
    const r2 = await request.post(`${API}/api/v1/users/me/mfa/setup`, {
      data: { mfa_type: 2 },
      headers: { Authorization: `Bearer ${token}` },
    })
    const d2 = await r2.json()
    expect(d2.code).toBe(0)
    const setupCode = d2.data.code
    expect(setupCode).toMatch(/^\d{6}$/)

    // Verify to enable
    const r3 = await request.post(`${API}/api/v1/users/me/mfa/verify`, {
      data: { code: setupCode },
      headers: { Authorization: `Bearer ${token}` },
    })
    expect((await r3.json()).code).toBe(0)

    // Login - should trigger MFA challenge WITH a new code
    const r4 = await request.post(`${API}/api/v1/users/login`, {
      data: { account: `emf_${ts}`, password: 'test123456' },
    })
    const d4 = await r4.json()
    expect(d4.code).toBe(0)
    expect(d4.data?.mfa_required).toBe(true)
    expect(d4.data?.masked_email).toBe('c***9@qq.com')

    secondCode = d4.data?.code || ''
    expect(secondCode).toMatch(/^\d{6}$/)

    // Verify MFA login with the new code
    const r5 = await request.post(`${API}/api/v1/auth/mfa/verify`, {
      data: { user_id: userId, mfa_token: d4.data.mfa_token, code: secondCode },
    })
    const d5 = await r5.json()
    expect(d5.code).toBe(0)
    expect(d5.data?.token).toBeTruthy()

    // Token works
    const r6 = await request.get(`${API}/api/v1/users/me`, {
      headers: { Authorization: `Bearer ${d5.data.token}` },
    })
    expect((await r6.json()).data?.name).toBe('EmailMFA')
  })
})
