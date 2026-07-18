import { test, expect } from './fixtures/coverage'

const API = 'http://localhost:8080'
const TS = Date.now()

let aTok = '', aId = '', aName = ''
let bTok = '', bId = '', bName = ''
let groupId = ''
let webhookApiKey = ''

/** Helper: store auth tokens in sessionStorage + basic user info in localStorage. */
function auth(userId: string, name: string, token: string): string {
  return `sessionStorage.setItem('ziziphus_token', JSON.stringify('${token}'));
sessionStorage.setItem('ziziphus_user', JSON.stringify({user_id:'${userId}',account:'',name:'${name}',avatar:'',type:0,status:1,uid:'',primary_color:'#0F172A',secondary_color:'#64748B',wake_mode:0,api_key:'',created_at:1700000000}));
localStorage.setItem('ziziphus_theme', JSON.stringify('light'));
localStorage.setItem('ziziphus_language', JSON.stringify('zh'));`
}

/** sleep */
function sleep(ms: number) { return new Promise(r => setTimeout(r, ms)) }

test.describe('Unread Count Clearing', () => {
  test.beforeAll(async ({ request }) => {
    // Register user A
    const rA = await request.post(`${API}/api/v1/users/register`, {
      data: { account: `una_${TS}`, name: 'UnreadA', password: 'test1234' },
      headers: { 'Content-Type': 'application/json' },
    })
    const ja = await rA.json()
    expect(ja.code).toBe(0)
    aId = ja.data.user_id
    aTok = ja.data.token
    aName = ja.data.name

    // Register user B
    const rB = await request.post(`${API}/api/v1/users/register`, {
      data: { account: `unb_${TS}`, name: 'UnreadB', password: 'test1234' },
      headers: { 'Content-Type': 'application/json' },
    })
    const jb = await rB.json()
    expect(jb.code).toBe(0)
    bId = jb.data.user_id
    bTok = jb.data.token
    bName = jb.data.name

    // User A creates a group with User B
    const createG = await request.post(`${API}/api/v1/conversations/group`, {
      headers: { Authorization: `Bearer ${aTok}`, 'Content-Type': 'application/json' },
      data: { name: `UnreadTest_${TS}`, member_ids: [bId] },
    })
    const gResp = await createG.json()
    expect(gResp.code).toBe(0)
    groupId = gResp.data.conv_id
    expect(groupId).toBeTruthy()

    // Create a webhook for the group (so we can send messages via API)
    const whCreate = await request.post(`${API}/api/v1/conversations/${groupId}/webhooks`, {
      headers: { Authorization: `Bearer ${aTok}`, 'Content-Type': 'application/json' },
      data: { name: 'test-hook' },
    })
    const whResp = await whCreate.json()
    expect(whResp.code).toBe(0)
    webhookApiKey = whResp.data?.api_key_plain || whResp.data?.api_key || ''
    // If API key not returned, try fetching webhook list
    if (!webhookApiKey) {
      const whList = await request.get(`${API}/api/v1/conversations/${groupId}/webhooks`, {
        headers: { Authorization: `Bearer ${aTok}` },
      })
      const whListResp = await whList.json()
      webhookApiKey = whListResp.data?.items?.[0]?.api_key_plain || ''
    }
    expect(webhookApiKey).toBeTruthy()
  })

  test('01 - conversation unread clears after clicking conversation', async ({ page }) => {
    if (!groupId || !webhookApiKey) { test.skip(); return }

    // Send a message via webhook API (creates unread for User B)
    const sendMsg = await page.request.post(`${API}/api/v1/webhooks/receive`, {
      headers: { 'Content-Type': 'application/json', 'X-API-Key': webhookApiKey },
      data: { body: 'unread test message - should disappear', conv_id: groupId },
    })
    expect(sendMsg.status()).toBe(200)
    await sleep(1500)

    // User B logs in
    await page.addInitScript(auth(bId, bName, bTok))
    await page.goto('/conversations', { waitUntil: 'networkidle' })
    await page.waitForTimeout(2000)

    // ✅ Verify: conversation list shows the group
    await expect(page.getByText(`UnreadTest_${TS}`).first()).toBeVisible({ timeout: 5000 })

    // ✅ Verify: unread badge is visible (the number "1")
    const unreadBadge = page.locator('span').filter({ hasText: /^1$/ }).first()
    await expect(unreadBadge).toBeVisible({ timeout: 5000 })

    // Click the conversation to open it — this should trigger markRead
    await page.getByText(`UnreadTest_${TS}`).first().click()
    await page.waitForTimeout(3000)

    // Go back to conversation list
    const backBtn = page.getByRole('button', { name: /返回|back/i }).or(page.locator('svg').first())
    if (await backBtn.isVisible().catch(() => false)) {
      await backBtn.click()
      await page.waitForTimeout(1500)
    } else {
      await page.goto('/conversations', { waitUntil: 'networkidle' })
      await page.waitForTimeout(2000)
    }

    // ✅ Verify: unread badge is gone
    const unreadAfter = page.locator('span').filter({ hasText: /^1$/ })
    const count = await unreadAfter.count()
    expect(count).toBe(0)
  })

  test('02 - system conversation unread clears after clicking', async ({ page }) => {
    // Trigger a system message: User A sends a contact request to User B
    const contactReq = await page.request.post(`${API}/api/v1/contact-requests`, {
      headers: { Authorization: `Bearer ${aTok}`, 'Content-Type': 'application/json' },
      data: { user_id: bId },
    })
    expect(contactReq.status()).toBe(200)
    await sleep(2000)

    // User B logs in
    await page.addInitScript(auth(bId, bName, bTok))
    await page.goto('/conversations', { waitUntil: 'networkidle' })
    await page.waitForTimeout(2000)

    // ✅ Verify: system conversation is visible
    const sysConv = page.getByText('系统消息').or(page.getByText('System Messages'))
    await expect(sysConv.first()).toBeVisible({ timeout: 5000 })

    // ✅ Verify: unread badge is visible on system conversation
    const unreadBadge = page.locator('span').filter({ hasText: /^1$/ }).first()
    await expect(unreadBadge).toBeVisible({ timeout: 5000 })

    // Click the system conversation
    await sysConv.first().click()
    await page.waitForTimeout(3000)

    // Go back to conversation list
    const backBtn = page.getByRole('button', { name: /返回|back/i }).or(page.locator('svg').first())
    if (await backBtn.isVisible().catch(() => false)) {
      await backBtn.click()
      await page.waitForTimeout(1500)
    } else {
      await page.goto('/conversations', { waitUntil: 'networkidle' })
      await page.waitForTimeout(2000)
    }

    // ✅ Verify: unread badge is gone
    const unreadAfter = page.locator('span').filter({ hasText: /^1$/ })
    const count = await unreadAfter.count()
    expect(count).toBe(0)
  })
})
