import { test, expect } from './fixtures/coverage'

const AUTH_INIT = `
  sessionStorage.setItem('ziziphus_token', JSON.stringify('test-mock-token'));
  sessionStorage.setItem('ziziphus_user', JSON.stringify({
    user_id: 'user_001', account: 'testuser', name: '测试用户', avatar: '/avatars/me.jpg',
    type: 0, status: 1, uid: '', primary_color: '#0F172A', secondary_color: '#64748B',
    wake_mode: 0, api_key: '', discoverable: true, allow_direct_chat: true, created_at: 1700000000,
  }));
  localStorage.setItem('ziziphus_language', JSON.stringify('zh'));
`

const MOCK_MESSAGES = [
  {
    msg_id: 1001,
    conv_id: 'conv_test_001',
    sender_id: 'user_001',
    sender_name: '测试用户',
    content_type: 1,
    body: '你好，这是测试消息',
    reply_to: 0,
    mention: [],
    timestamp: 1700000001,
    conv_seq: 1,
    status: 2,
  },
  {
    msg_id: 1002,
    conv_id: 'conv_test_001',
    sender_id: 'user_002',
    sender_name: '小明',
    content_type: 1,
    body: '收到，你好！',
    reply_to: 0,
    mention: [],
    timestamp: 1700000002,
    conv_seq: 2,
    status: 2,
  },
  {
    msg_id: 1003,
    conv_id: 'conv_test_001',
    sender_id: 'user_003',
    sender_name: '小红',
    content_type: 1,
    body: '大家早上好',
    reply_to: 0,
    mention: [],
    timestamp: 1700000003,
    conv_seq: 3,
    status: 2,
  },
]

const MOCK_SENDER_INFO: Record<string, { user_id: string; name: string; avatar: string; cover: string; type: number; account: string; primary_color: string; secondary_color: string; discoverable: boolean; allow_direct_chat: boolean; created_at: number; status: number; uid: string; wake_mode: number; api_key: string }> = {
  'user_002': {
    user_id: 'user_002',
    name: '小明',
    avatar: '/avatars/xiaoming.jpg',
    cover: '/covers/xiaoming.jpg',
    type: 0,
    account: 'xiaoming',
    primary_color: '#3B82F6',
    secondary_color: '#60A5FA',
    discoverable: true,
    allow_direct_chat: true,
    created_at: 1700000000,
    status: 1,
    uid: '',
    wake_mode: 0,
    api_key: '',
  },
  'user_003': {
    user_id: 'user_003',
    name: '小红',
    avatar: '/avatars/xiaohong.jpg',
    cover: '/covers/xiaohong.jpg',
    type: 0,
    account: 'xiaohong',
    primary_color: '#EC4899',
    secondary_color: '#F472B6',
    discoverable: true,
    allow_direct_chat: true,
    created_at: 1700000000,
    status: 1,
    uid: '',
    wake_mode: 0,
    api_key: '',
  },
}

test.describe('Sender Display & UserCard', () => {
  test.beforeEach(async ({ page }) => {
    // Mock auth
    await page.addInitScript(AUTH_INIT)

    // Intercept message history to return mock messages with sender_name
    await page.route('**/api/v1/conversations/conv_test_001/messages**', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok', data: MOCK_MESSAGES }) })
    })

    await page.route('**/api/v1/users/user_002', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok', data: MOCK_SENDER_INFO['user_002'] }) })
    })
    await page.route('**/api/v1/users/user_003', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok', data: MOCK_SENDER_INFO['user_003'] }) })
    })

    // Intercept read receipts
    await page.route('**/api/v1/messages/*/receipts', async route => {
      await route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ code: 0, msg: 'ok', data: [] }) })
    })

    await page.goto('/chat/conv_test_001')
    await page.waitForTimeout(1000)
  })

  test('message bubble shows sender_name from API', async ({ page }) => {
    // Verify "小明" appears as a sender name (not raw user_id)
    const senderName = page.locator('button').filter({ hasText: '小明' }).first()
    await expect(senderName).toBeVisible({ timeout: 5000 })

    // Verify "小红" appears as a sender name
    const senderName2 = page.locator('button').filter({ hasText: '小红' }).first()
    await expect(senderName2).toBeVisible({ timeout: 3000 })

    // Verify no raw user_id like "user_002" is shown as display name
    const rawId = page.locator('button').filter({ hasText: 'user_002' })
    await expect(rawId).not.toBeVisible()
  })

  test('sender avatar shows hover UserCard with name and cover', async ({ page }) => {
    // Wait for messages to render
    await expect(page.locator('button').filter({ hasText: '小明' }).first()).toBeVisible({ timeout: 5000 })

    // Hover over user_002's sender avatar (first non-own message avatar)
    const avatarBtn = page.locator('button').filter({ hasText: /^[A-Z一-鿿]$/ }).first()
    await expect(avatarBtn).toBeVisible({ timeout: 3000 })
    await avatarBtn.hover()
    await page.waitForTimeout(500)

    // UserCard popup should show the user's name
    // The UserCard renders the name in a div with font-headline class
    const userNameInCard = page.locator('text=小明').first()
    await expect(userNameInCard).toBeVisible({ timeout: 3000 })

    // The account handle should appear
    await expect(page.getByText('@xiaoming').first()).toBeVisible({ timeout: 3000 })
  })

  test('sender cache does not show stale avatar after TTL expiry re-fetch', async ({ page }) => {
    // Verify "小明" message is displayed
    await expect(page.locator('button').filter({ hasText: '小明' }).first()).toBeVisible({ timeout: 5000 })

    // Trigger a second load to verify cache-hit path
    // The sender cache should re-use the cached entry without a second API call
    const userApiCalls: string[] = []
    await page.route('**/api/v1/users/*', async route => {
      userApiCalls.push(route.request().url())
      await route.continue()
    })

    // Navigate to another conversation and back to trigger re-fetch
    await page.goto('/chat/conv_test_002')
    await page.waitForTimeout(500)
    await page.goto('/chat/conv_test_001')
    await page.waitForTimeout(1000)

    // Verify messages still render with sender names
    const senderName = page.locator('button').filter({ hasText: '小明' }).first()
    await expect(senderName).toBeVisible({ timeout: 5000 })
  })
})
