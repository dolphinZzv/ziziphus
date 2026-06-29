import { test, expect } from './fixtures/coverage'

const API = process.env.E2E_API_URL || 'http://47.95.200.101:10011'
const TS = Date.now()

test('Webhook full lifecycle - register, create, send, verify', async ({ request }) => {
  // Step 1: Register two users
  const regA = await request.post(`${API}/api/v1/users/register`, {
    data: { name: `e2e-a-${TS}`, account: `e2e-a-${TS}`, password: 'test12345678' },
  })
  const regB = await request.post(`${API}/api/v1/users/register`, {
    data: { name: `e2e-b-${TS}`, account: `e2e-b-${TS}`, password: 'test12345678' },
  })
  const bodyA = await regA.json()
  const bodyB = await regB.json()
  expect(bodyA.code).toBe(0)
  expect(bodyB.code).toBe(0)
  const tokenA = bodyA.data.token
  const userIdB = bodyB.data.user_id
  expect(tokenA).toBeTruthy()
  console.log(`Auth OK: ${bodyA.data.user_id}`)

  // Step 2: Create group
  const groupRes = await request.post(`${API}/api/v1/conversations/group`, {
    headers: { Authorization: `Bearer ${tokenA}` },
    data: { name: `e2e-group-${TS}`, member_ids: [userIdB] },
  })
  const groupBody = await groupRes.json()
  expect(groupBody.code).toBe(0)
  const groupId = groupBody.data.conv_id
  expect(groupId).toBeTruthy()
  console.log(`Group: ${groupId}`)

  // Step 3: Create webhook
  const whRes = await request.post(`${API}/api/v1/conversations/${groupId}/webhooks`, {
    headers: { Authorization: `Bearer ${tokenA}` },
    data: { name: 'e2e-hook', callback_url: '', require_audit: false },
  })
  const whBody = await whRes.json()
  expect(whBody.code).toBe(0)
  expect(whBody.data.token).toBeTruthy()
  expect(whBody.data.api_key).toBeTruthy()
  const whToken = whBody.data.token
  const whKey = whBody.data.api_key
  const whId = whBody.data.id
  console.log(`Webhook created: token=${whToken}`)

  // Step 4: Send message via public webhook endpoint
  const msgRes = await request.post(`${API}/api/v1/webhooks/${whToken}`, {
    headers: { Authorization: `Bearer ${whKey}` },
    data: { body: `E2E webhook test ${TS}` },
  })
  const msgBody = await msgRes.json()
  expect(msgBody.code).toBe(0)
  expect(msgBody.data.msg_id).toBeGreaterThan(0)
  expect(msgBody.data.audit_status).toBe('approved')
  console.log(`Message sent: ${msgBody.data.msg_id}`)

  // Step 5: Verify message in group with correct sender_name
  const msgsRes = await request.get(
    `${API}/api/v1/conversations/${groupId}/messages?limit=10`,
    { headers: { Authorization: `Bearer ${tokenA}` } }
  )
  const msgsBody = await msgsRes.json()
  expect(msgsBody.code).toBe(0)
  const messages = Array.isArray(msgsBody.data) ? msgsBody.data : msgsBody.data?.items || []
  const found = messages.find((m: any) => m.body?.includes(`E2E webhook test ${TS}`))
  expect(found).toBeTruthy()
  expect(found.sender_name).toBe('e2e-hook')
  console.log(`Message verified: sender_name=${found.sender_name}`)

  // Step 6: Auth error with wrong key
  const authRes = await request.post(`${API}/api/v1/webhooks/${whToken}`, {
    headers: { Authorization: 'Bearer bad-key' },
    data: { body: 'should fail' },
  })
  const authBody = await authRes.json()
  expect(authBody.code).not.toBe(0)
  expect(authBody.code).toBe(401)
  console.log(`Auth error: code=${authBody.code}`)

  // Step 7: List webhooks shows token and api_key
  const listRes = await request.get(
    `${API}/api/v1/conversations/${groupId}/webhooks`,
    { headers: { Authorization: `Bearer ${tokenA}` } }
  )
  const listBody = await listRes.json()
  expect(listBody.code).toBe(0)
  const list = Array.isArray(listBody.data) ? listBody.data : []
  const wh = list.find((w: any) => w.name === 'e2e-hook')
  expect(wh).toBeTruthy()
  expect(wh.token).toBeTruthy()
  expect(wh.api_key).toBeTruthy()
  console.log(`Webhook list: token visible=${!!wh.token}, key visible=${!!wh.api_key}`)

  // Step 8: Cleanup
  if (whId) {
    await request.delete(
      `${API}/api/v1/conversations/${groupId}/webhooks/${whId}`,
      { headers: { Authorization: `Bearer ${tokenA}` } }
    )
    console.log('Cleanup done')
  }
})
