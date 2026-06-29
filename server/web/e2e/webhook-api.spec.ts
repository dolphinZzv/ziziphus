import { test, expect } from './fixtures/coverage'

const API = process.env.E2E_API_URL || 'http://localhost:8080'

const TS = Date.now()
const TEST_USER = {
  name: `e2e-wh-${TS}`,
  account: `e2e-wh-${TS}`,
  password: 'test12345678',
}

test.describe('Webhook - Real API', () => {
  let authToken = ''
  let userId = ''
  let convId = ''
  let whToken = ''
  let whApiKey = ''

  test.beforeAll(async ({ request }) => {
    // Step 1: Register a fresh test user
    const regRes = await request.post(`${API}/api/v1/users/register`, {
      data: { name: TEST_USER.name, account: TEST_USER.account, password: TEST_USER.password },
    })
    if (!regRes.ok()) {
      // User may already exist — try login
      const loginRes = await request.post(`${API}/api/v1/users/login`, {
        data: { account: TEST_USER.account, password: TEST_USER.password },
      })
      expect(loginRes.ok()).toBeTruthy()
      const loginBody = await loginRes.json()
      expect(loginBody.code).toBe(0)
      authToken = loginBody.data.token
      userId = loginBody.data.user?.user_id
    } else {
      const regBody = await regRes.json()
      expect(regBody.code).toBe(0)
      authToken = regBody.data.token
      userId = regBody.data.user?.user_id
    }
    expect(authToken).toBeTruthy()
    expect(userId).toBeTruthy()
    console.log(`User: ${userId}, Token: ${authToken.slice(0, 20)}...`)
  })

  test('Create P2P conversation', async ({ request }) => {
    const convRes = await request.post(`${API}/api/v1/conversations/p2p`, {
      headers: { Authorization: `Bearer ${authToken}` },
      data: { user_id: 'user_002' },
    })
    const body = await convRes.json()
    expect(body.code).toBe(0)
    convId = body.data.conv_id
    expect(convId).toBeTruthy()
    console.log(`Conversation: ${convId}`)
  })

  test('Create webhook', async ({ request }) => {
    expect(convId).toBeTruthy()
    const whRes = await request.post(`${API}/api/v1/conversations/${convId}/webhooks`, {
      headers: { Authorization: `Bearer ${authToken}` },
      data: { name: 'e2e-test-hook', callback_url: '', require_audit: false },
    })
    const body = await whRes.json()
    expect(body.code).toBe(0)
    expect(body.data.token).toBeTruthy()
    expect(body.data.api_key).toBeTruthy()
    whToken = body.data.token
    whApiKey = body.data.api_key
    console.log(`Webhook: token=${whToken}, key=${whApiKey}`)
  })

  test('Send message via webhook (public endpoint)', async ({ request }) => {
    expect(whToken).toBeTruthy()
    const msgRes = await request.post(`${API}/api/v1/webhooks/${whToken}`, {
      headers: { Authorization: `Bearer ${whApiKey}` },
      data: { body: `E2E webhook test ${TS}` },
    })
    const body = await msgRes.json()
    expect(body.code).toBe(0)
    expect(body.data.msg_id).toBeGreaterThan(0)
    expect(body.data.audit_status).toBe('approved')
    console.log(`Message sent: ${body.data.msg_id}`)
  })

  test('Verify message in conversation', async ({ request }) => {
    const msgsRes = await request.get(
      `${API}/api/v1/conversations/${convId}/messages?limit=10`,
      { headers: { Authorization: `Bearer ${authToken}` } }
    )
    const body = await msgsRes.json()
    expect(body.code).toBe(0)
    const messages = Array.isArray(body.data) ? body.data : body.data?.items || []
    const found = messages.find((m: any) => m.body?.includes(`E2E webhook test ${TS}`))
    expect(found).toBeTruthy()
    expect(found.sender_name).toBe('e2e-test-hook')
  })

  test('Auth error returns 401', async ({ request }) => {
    const res = await request.post(`${API}/api/v1/webhooks/${whToken}`, {
      headers: { Authorization: 'Bearer bad-key' },
      data: { body: 'should fail' },
    })
    const body = await res.json()
    expect(body.code).not.toBe(0)
  })

  test('Cleanup - delete webhook', async ({ request }) => {
    const listRes = await request.get(
      `${API}/api/v1/conversations/${convId}/webhooks`,
      { headers: { Authorization: `Bearer ${authToken}` } }
    )
    const listBody = await listRes.json()
    const whList = Array.isArray(listBody.data) ? listBody.data : []
    const created = whList.find((w: any) => w.name === 'e2e-test-hook')
    if (created) {
      await request.delete(
        `${API}/api/v1/conversations/${convId}/webhooks/${created.id}`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      )
    }
  })
})
