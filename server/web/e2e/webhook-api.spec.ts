import { test, expect } from './fixtures/coverage'

const API = process.env.E2E_API_URL || 'http://47.95.200.101:10011'

const TS = Date.now()

test.describe('Webhook - Real API', () => {
  let authToken = ''
  let userId = ''
  let userBId = ''
  let authTokenB = ''
  let groupId = ''
  let whToken = ''
  let whApiKey = ''
  let whId = 0

  test.beforeAll(async ({ request }) => {
    // Register user A
    const regA = await request.post(`${API}/api/v1/users/register`, {
      data: { name: `e2e-a-${TS}`, account: `e2e-a-${TS}`, password: 'test12345678' },
    })
    // Register user B
    const regB = await request.post(`${API}/api/v1/users/register`, {
      data: { name: `e2e-b-${TS}`, account: `e2e-b-${TS}`, password: 'test12345678' },
    })

    const bodyA = await regA.json()
    const bodyB = await regB.json()
    expect(bodyA.code).toBe(0)
    expect(bodyB.code).toBe(0)

    authToken = bodyA.data.token
    userId = bodyA.data.user_id
    authTokenB = bodyB.data.token
    userBId = bodyB.data.user_id
    expect(authToken).toBeTruthy()
    expect(userId).toBeTruthy()
    expect(userBId).toBeTruthy()
    console.log(`User A: ${userId}, User B: ${userBId}`)
  })

  test('Create group conversation', async ({ request }) => {
    const res = await request.post(`${API}/api/v1/conversations/group`, {
      headers: { Authorization: `Bearer ${authToken}` },
      data: { name: `e2e-group-${TS}`, member_ids: [userBId] },
    })
    const body = await res.json()
    expect(body.code).toBe(0)
    groupId = body.data.conv_id
    expect(groupId).toBeTruthy()
    console.log(`Group: ${groupId}`)
  })

  test('Create webhook', async ({ request }) => {
    expect(groupId).toBeTruthy()
    const res = await request.post(`${API}/api/v1/conversations/${groupId}/webhooks`, {
      headers: { Authorization: `Bearer ${authToken}` },
      data: { name: 'e2e-hook', callback_url: '', require_audit: false },
    })
    const body = await res.json()
    expect(body.code).toBe(0)
    expect(body.data.token).toBeTruthy()
    expect(body.data.api_key).toBeTruthy()
    whToken = body.data.token
    whApiKey = body.data.api_key
    whId = body.data.id
    console.log(`Webhook: token=${whToken}, key=${whApiKey}`)
  })

  test('Send message via webhook', async ({ request }) => {
    expect(whToken).toBeTruthy()
    const res = await request.post(`${API}/api/v1/webhooks/${whToken}`, {
      headers: { Authorization: `Bearer ${whApiKey}` },
      data: { body: `E2E webhook test ${TS}` },
    })
    const body = await res.json()
    expect(body.code).toBe(0)
    expect(body.data.msg_id).toBeGreaterThan(0)
    expect(body.data.audit_status).toBe('approved')
    console.log(`Message sent: ${body.data.msg_id}`)
  })

  test('Verify message in group', async ({ request }) => {
    const res = await request.get(
      `${API}/api/v1/conversations/${groupId}/messages?limit=10`,
      { headers: { Authorization: `Bearer ${authToken}` } }
    )
    const body = await res.json()
    expect(body.code).toBe(0)
    const messages = Array.isArray(body.data) ? body.data : body.data?.items || []
    const found = messages.find((m: any) => m.body?.includes(`E2E webhook test ${TS}`))
    expect(found).toBeTruthy()
    expect(found.sender_name).toBe('e2e-hook')
  })

  test('Auth error returns 401', async ({ request }) => {
    const res = await request.post(`${API}/api/v1/webhooks/${whToken}`, {
      headers: { Authorization: 'Bearer bad-key' },
      data: { body: 'should fail' },
    })
    const body = await res.json()
    expect(body.code).not.toBe(0)
  })

  test('Webhook list shows token and api_key', async ({ request }) => {
    const res = await request.get(
      `${API}/api/v1/conversations/${groupId}/webhooks`,
      { headers: { Authorization: `Bearer ${authToken}` } }
    )
    const body = await res.json()
    expect(body.code).toBe(0)
    const list = Array.isArray(body.data) ? body.data : []
    const wh = list.find((w: any) => w.name === 'e2e-hook')
    expect(wh).toBeTruthy()
    expect(wh.token).toBeTruthy()
    expect(wh.api_key).toBeTruthy()
  })

  test('Cleanup - delete webhook', async ({ request }) => {
    if (whId) {
      await request.delete(
        `${API}/api/v1/conversations/${groupId}/webhooks/${whId}`,
        { headers: { Authorization: `Bearer ${authToken}` } }
      )
    }
  })
})
