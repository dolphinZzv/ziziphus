import { test, expect } from './fixtures/coverage'

const API = 'http://localhost:8080'
const TS = Date.now()

let aTok = '', aName = ''
let bTok = '', bId = '', bName = ''
let groupId = ''
let whId = 0
let groupName = ''

function auth(userId: string, name: string, token: string): string {
  return `sessionStorage.setItem('ziziphus_token', JSON.stringify('${token}'));
sessionStorage.setItem('ziziphus_user', JSON.stringify({user_id:'${userId}',account:'',name:'${name}',avatar:'',type:0,status:1,uid:'',primary_color:'#0F172A',secondary_color:'#64748B',wake_mode:0,api_key:'',created_at:1700000000}));
localStorage.setItem('ziziphus_theme', JSON.stringify('light'));
localStorage.setItem('ziziphus_language', JSON.stringify('zh'));`
}

function sleep(ms: number) { return new Promise(r => setTimeout(r, ms)) }

test.describe('Unread Count Clearing', () => {
  test.beforeAll(async ({ request }) => {
    const rA = await request.post(`${API}/api/v1/users/register`, {
      data: { account: `una_${TS}`, name: 'UnreadA', password: 'test1234' },
      headers: { 'Content-Type': 'application/json' },
    })
    const ja = await rA.json()
    expect(ja.code).toBe(0)
    aTok = ja.data.token
    aName = ja.data.name

    const rB = await request.post(`${API}/api/v1/users/register`, {
      data: { account: `unb_${TS}`, name: 'UnreadB', password: 'test1234' },
      headers: { 'Content-Type': 'application/json' },
    })
    const jb = await rB.json()
    expect(jb.code).toBe(0)
    bId = jb.data.user_id
    bTok = jb.data.token
    bName = jb.data.name

    groupName = `UnreadTest_${TS}`
    const g = await request.post(`${API}/api/v1/conversations/group`, {
      headers: { Authorization: `Bearer ${aTok}`, 'Content-Type': 'application/json' },
      data: { name: groupName, member_ids: [bId] },
    })
    const gR = await g.json()
    expect(gR.code).toBe(0)
    groupId = gR.data.conv_id

    const wh = await request.post(`${API}/api/v1/conversations/${groupId}/webhooks`, {
      headers: { Authorization: `Bearer ${aTok}`, 'Content-Type': 'application/json' },
      data: { name: 'test-hook' },
    })
    const whR = await wh.json()
    expect(whR.code).toBe(0)
    whId = whR.data?.webhook_id || whR.data?.id || 0
    expect(whId).toBeGreaterThan(0)
  })

  test('01 - group conversation unread clears after opening', async ({ page }) => {
    if (!groupId || !whId) { test.skip(); return }

    // Send webhook test message (creates unread for User B)
    const testResp = await page.request.post(
      `${API}/api/v1/conversations/${groupId}/webhooks/${whId}/test`,
      { headers: { Authorization: `Bearer ${aTok}` } },
    )
    expect(testResp.status()).toBe(200)
    await sleep(2000)

    // User B logs in
    await page.addInitScript(auth(bId, bName, bTok))
    await page.goto('/conversations', { waitUntil: 'networkidle' })
    await page.waitForTimeout(3000)

    // ✅ Unread badge visible on the group
    const badge = page.locator('span').filter({ hasText: /^\d+$/ }).first()
    await expect(badge).toBeVisible({ timeout: 5000 })

    const badgeText = await badge.textContent()
    expect(Number(badgeText)).toBeGreaterThan(0)

    // Click the group conversation
    await page.getByText(groupName).first().click()
    await page.waitForTimeout(3000)

    // Go back to conversation list
    await page.goto('/conversations', { waitUntil: 'networkidle' })
    await page.waitForTimeout(3000)

    // ✅ Unread count should be 0 or the badge should not exist for this conversation
    // It's possible there are other number badges (device indicator), so check specifically
    // that the conversation row no longer has a badge next to it
    const row = page.getByText(groupName).first()
    const rowParent = row.locator('..')
    const badgeInRow = rowParent.locator('span').filter({ hasText: /^\d+$/ })
    expect(await badgeInRow.count()).toBe(0)
  })

  test('02 - system conversation unread clears after opening', async ({ page }) => {
    // User A sends contact request → system message for User B
    const cr = await page.request.post(`${API}/api/v1/contact-requests`, {
      headers: { Authorization: `Bearer ${aTok}`, 'Content-Type': 'application/json' },
      data: { user_id: bId },
    })
    expect(cr.status()).toBe(200)
    await sleep(2000)

    await page.addInitScript(auth(bId, bName, bTok))
    await page.goto('/conversations', { waitUntil: 'networkidle' })
    await page.waitForTimeout(3000)
    await sleep(1000)

    // ✅ System conversation visible
    const sysConv = page.getByText('系统消息')
    await expect(sysConv.first()).toBeVisible({ timeout: 5000 })

    // ✅ Unread badge visible
    const badge = page.locator('span').filter({ hasText: /^\d+$/ }).first()
    await expect(badge).toBeVisible({ timeout: 5000 })

    // Click system conversation
    await sysConv.first().click()
    await page.waitForTimeout(3000)

    // Go back
    await page.goto('/conversations', { waitUntil: 'networkidle' })
    await page.waitForTimeout(3000)

    // ✅ Verify unread cleared for this conversation
    const row = page.getByText('系统消息').first()
    const rowParent = row.locator('..')
    const badgeInRow = rowParent.locator('span').filter({ hasText: /^\d+$/ })
    expect(await badgeInRow.count()).toBe(0)
  })
})
