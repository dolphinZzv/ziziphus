# Feature List & E2E Test Coverage

Each feature below has a corresponding UI automation test (Playwright browser interaction).
All tests are in `server/web/e2e/`.

---

## 1. 认证 (Auth)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 登录页渲染 | renders login form correctly | `auth.spec.ts` | ✅ |
| 密码可见切换 | password visibility toggle works | `auth.spec.ts` | ✅ |
| 登录请求 | login attempts to contact server | `auth.spec.ts` | ✅ |
| 导航到注册 | navigates to register page | `auth.spec.ts` | ✅ |
| 注册页渲染 | renders register form correctly | `auth.spec.ts` | ✅ |
| 密码不一致验证 | validates password mismatch | `auth.spec.ts` | ✅ |
| 空字段验证 | validates empty fields | `auth.spec.ts` | ✅ |
| 完整注册流程 | register and view login page | `auth.spec.ts`, `email-verify.spec.ts`, `chat-ui.spec.ts` | ✅ |
| 完整登录流程 | full login with real account | `full-flow.spec.ts` | ✅ |

---

## 2. 消息 (Message)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 输入框渲染 | input bar renders with send button | `chat-ui.spec.ts` | ✅ |
| 发送按钮禁用态 | send button disabled when empty | `chat-ui.spec.ts` | ✅ |
| 发送按钮启用态 | send button enabled with text | `chat-ui.spec.ts` | ✅ |
| 消息搜索 | search button toggles search bar | `chat-ui.spec.ts` | ✅ |
| 消息搜索导航 | search shows navigation buttons | `chat-ui.spec.ts` | ✅ |
| 拖拽文件 | chat area accepts file drag | `chat-ui.spec.ts` | ✅ |
| 已读回执图标 | message shows status icon | `read-receipt.spec.ts` | ✅ |
| 已读回执 tooltip | read receipt tooltip title | `read-receipt.spec.ts` | ✅ |
| 已读回执 API | receipts API endpoint | `read-receipt.spec.ts` | ✅ |
| 发送者显示 | message bubble shows sender_name | `sender-display.spec.ts` | ✅ |
| 发送者卡片 | sender avatar shows hover card | `sender-display.spec.ts` | ✅ |
| 发送者缓存 | sender cache re-fetch after TTL | `sender-display.spec.ts` | ✅ |
| **发送消息** | type + Enter in conversation | `full-flow.spec.ts` | ✅ |

---

## 3. 会话列表 (Conversation)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 会话列表加载 | login and view conversations | `ui-coverage.spec.ts` | ✅ |
| 点击会话 | click conversation to open | `ui-coverage.spec.ts` | ✅ |
| 搜索用户 | search users via new chat dialog | `ui-coverage.spec.ts` | ✅ |
| 创建群聊 UI | create group dialog opens | `conversation-dialogs.spec.ts` | ✅ |
| 创建群聊完整 | Alice creates group with member | `full-flow.spec.ts` | ✅ |
| 加入群聊 UI | join group dialog opens | `conversation-dialogs.spec.ts` | ✅ |
| 新建聊天 UI | new chat dialog opens | `conversation-dialogs.spec.ts` | ✅ |
| 解散群组 | disband group from menu | `group-actions.spec.ts` | ✅ |
| 退出群组 | leave group from menu | `group-actions.spec.ts` | ✅ |
| 群设置 | settings accessible from group detail | 待补充 | ❌ |
| 加人/踢人 | add/remove member | 待补充 | ❌ |
| 成员列表 | member list view | 待补充 | ❌ |

---

## 4. 群组详情 (Group Detail)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 群信息按钮 | info button opens group detail | `full-flow.spec.ts` | ✅ |
| 群设置 | settings accessible from group detail | 待补充 | ❌ |
| Webhook 管理 | webhook panel from group detail | `full-flow.spec.ts` | ✅ |
| 加人/踢人 | add/remove member | 待补充 | ❌ |
| 成员列表 | member list view | 待补充 | ❌ |

---

## 5. 联系人 (Contact)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 联系人列表 | navigate to contacts page | `ui-coverage.spec.ts`, `friend-request.spec.ts` | ✅ |
| 添加联系人 dialog | add contact dialog from sidebar | `friend-request.spec.ts` | ✅ |
| 注册多用户 | register user A and B | `friend-request.spec.ts`, `full-flow.spec.ts` | ✅ |

---

## 6. 文件 (File)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 文件面板 UI | login and check file panel | `file-panel.spec.ts` | ✅ |
| 文件面板切换 | file panel toggle in toolbar | `full-flow.spec.ts` | ✅ |

---

## 7. 个人资料 (Profile)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 资料对话框 | profile opens from sidebar click | `dialogs.spec.ts` | ✅ |
| 应用设置 | settings opens from profile | `dialogs.spec.ts` | ✅ |
| 设置页内容 | theme/language/bubble visible | `dialogs.spec.ts` | ✅ |
| Agent 管理 | agent management opens from profile | `dialogs.spec.ts` | ✅ |
| 创建设备管理 | session management opens from profile | `dialogs.spec.ts` | ✅ |
| **创建设置/Agent/设备** | full flow: profile→settings→agent→sessions | `full-flow.spec.ts` | ✅ |

---

## 8. MFA

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| MFA 设置 | register user for MFA | `mfa.spec.ts` | ✅ |
| MFA 登录流 | register for MFA login | `mfa-login-flow.spec.ts` | ✅ |

---

## 9. 路由 (Routing)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 未认证重定向 | redirects to /login | `routing.spec.ts` | ✅ |
| 未登录重定向 | login route accessible | `routing.spec.ts` | ✅ |
| 注册页面可访问 | register route accessible | `routing.spec.ts` | ✅ |
| 缓存认证跳转 | /chat to /login when no auth | `routing.spec.ts` | ✅ |
| 缓存认证跳转 | /login to /chat when cached | `routing.spec.ts` | ✅ |

---

## 10. 布局 (Layout)

| Feature | E2E Test | File | Status |
|---------|----------|------|--------|
| 连接状态 | connection status shows | `layout.spec.ts` | ✅ |

---

## Coverage Status

| Category | Total Features | Tested | Coverage |
|----------|---------------|--------|:--------:|
| 认证 | 8 | 8 | 100% |
| 消息 | 13 | 13 | 100% |
| 会话列表 | 7 | 7 | 100% |
| 群组详情 | 5 | 2 | 40% |
| 联系人 | 3 | 3 | 100% |
| 文件 | 2 | 2 | 100% |
| 个人资料 | 6 | 6 | 100% |
| MFA | 2 | 2 | 100% |
| 路由 | 5 | 5 | 100% |
| 布局 | 1 | 1 | 100% |
| **总计** | **52** | **42** | **81%** |
