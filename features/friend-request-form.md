# 好友申请 + 系统消息会话 + 表单消息 设计文档

> 创建: 2026-06-28 | 状态: 待实现

## Context

**问题**：当前联系人添加是单向的，A 加 B 无需 B 同意。且不存在独立的系统消息通道——好友申请无法在双方不是好友时送达。

**方案**：
1. 新增 `ConvSystem (type=3)` 系统消息会话——每个用户一个独立会话（`sys:<user_id>`），仅自己可见
2. 新增 `ContentType=10 (Form)` + `ContentType=11 (FormResponse)` 表单消息
3. A 发起申请 → B 的系统会话收到表单卡片 → B 点击"通过"/"拒绝"
4. B 处理后通知 A，A↔B 的 P2P 会话创建

## 整体流程

### 发起（事务内完成）

```
A → POST /api/v1/contact-requests  { user_id: "B", message: "我是xxx" }

  校验（事务开始前）:
    - 不给自己发: sender_id != target_id
    - 目标存在: userRepo.GetByID(target_id) != nil
    - 双方均未在对方联系人列表中:
        contactRepo.Exists(A, B) == false && contactRepo.Exists(B, A) == false
    - 已有 pending 请求: contactRequestRepo.GetPending(A, B) == nil
      （仅查 pending，已 rejected 的允许重新发起——删好友场景）

  DB 事务 BEGIN:
    1. INSERT contact_requests (status=pending, form_msg_id=0)
       → 获取 request_id
    2. 确保 B 系统会话存在: GetOrCreateSystemConvTx(ctx, B)
    3. 构造 FormDefinitionBody
    4. SendFormMessage(B系统会话) → 获得 form_msg_id
    5. UPDATE contact_requests SET form_msg_id = form_msg_id
    6. 确保 A 系统会话存在: GetOrCreateSystemConvTx(ctx, A)
    7. SendSystemMessage(A系统会话, "你已向 {B_name} 发送了好友申请")
  COMMIT ← 任何步骤失败则全部回滚，无中间状态

  返回 { request_id, form_msg_id }
```

### B 收到

```
MsgPush(convID=sys:B, content_type=10, body=FormDefinitionBody)
  → 会话列表未读+1，预览 "好友申请 · 张三"
  → B 打开系统会话 → form-bubble 渲染

  form-bubble 加载时的状态判定流程：
    1. 从 body 解析出 request_id 和 body.status（"active"）
    2. [初始占位] 先用 body.status 渲染 UI（按钮可点击）
    3. [异步确认] 调用 GET /by-form/:msg_id
       - 如果 status != 0 (pending) → 覆盖为 "closed"
       - 请求失败/超时 → 保持 body.status，按钮仍可用（允许 B 重试操作）
    4. [关键] body.status 仅作初始占位，API 返回值覆盖之
```

### B 通过

```
B 点击"通过" → 乐观更新:
  1. setFormStatus('closed')
  2. setActionResult('approved')
  3. 按钮置灰，显示 "已通过" + ✔
  4. 设置 5s 超时计时器

  → MsgSend(convID=sys:B, content_type=11, body=FormResponseBody{...})

  成功路径: 收到 MsgSendAck → clearTimeout → 状态确认
  失败路径:
    收到 MsgSendAck 但 status!=MsgSent（server error）
    OR 5s 超时未收到 ack
    → 回退乐观更新:
      setFormStatus('active')     // 恢复按钮
      setActionResult(null)      // 清除已处理的标记
      setErrorMessage('操作失败，请重试')

  服务端拒绝路径（并发竞争）:
    收到 MsgPush (ContentType=5, System Message):
      "该申请已被处理"  ← 说明另一个会话已处理
    → 回退乐观更新
    → 调用 /by-form/:msg_id 获取最新状态并展示

  ingest 处理 FormResponse（服务端）:
    1. 解析 body → FormResponseBody
    2. 查 contact_requests WHERE id = request_id
       - 不存在 → 返回错误 "申请不存在"
    3. 校验 sender_id == to_user_id → 否则拒绝
    4. 校验 msg.conv_id == "sys:" + to_user_id → 否则拒绝
    5. 校验 status == pending → 否则返回 "该申请已被处理"（幂等提示）
    6. UPDATE contact_requests SET status=approved, updated_at=NOW()
         WHERE id = request_id AND status = pending  ← SELECT FOR UPDATE 行锁保证并发安全
    7. INSERT INTO contacts (A→B), (B→A)
    8. GetOrCreateP2P(A, B)
    9. SendSystemMessage(P2P, "你们已成为好友，可以开始聊天了")
   10. SendSystemMessage(A系统会话, "{B_name} 已通过你的好友申请")
   11. SendSystemMessage(B系统会话, "你已通过 {A_name} 的好友申请")
   12. MsgSendAck → B 确认成功
   13. MsgPush(P2P) → A 和 B 在线客户端检测到新会话 → 自动刷新会话列表
```

### B 拒绝

```
流程同"通过"，但:
  - UPDATE status = rejected
  - 不创建 contacts
  - 不创建 P2P
  - 发送系统消息:
    SendSystemMessage(A系统会话, "{B_name} 已拒绝你的好友申请")
    SendSystemMessage(B系统会话, "你已拒绝 {A_name} 的好友申请")
```

### 删好友后重新申请

```
前提：contacts 表是双向独立的（A→B 和 B→A 各一条记录）

删除好友：DELETE FROM contacts WHERE user_id = A AND contact_id = B
  → A 不再有 B，B 可能仍有 A（取决于是否也单向删除）
  → 前端 "删除好友" 操作为双向删除: DELETE A→B + DELETE B→A

重新申请校验:
  - contacts 任一方向存在 → "你们已经是好友"
  - 双方都不存在 + 无 pending 请求 → 允许发起
  - 存在 rejected 旧请求 → 允许（先 DELETE 旧 rejected 记录，再 INSERT 新 pending）
```

## 数据模型

### conversations.type 新增 ConvSystem = 3

```go
const (
    ConvP2P    ConvType = 1
    ConvGroup  ConvType = 2
    ConvSystem ConvType = 3  // 新增
)
```

convID 格式：`sys:<user_id>`。每个用户一个系统会话，创建时仅有自己一个 member（role=owner）。不可删除、不可退出、不可发普通文本。

### contact_requests 表（新迁移 013）

```sql
CREATE TABLE contact_requests (
    id            BIGSERIAL PRIMARY KEY,
    from_user_id  VARCHAR(32) NOT NULL REFERENCES users(id),
    to_user_id    VARCHAR(32) NOT NULL REFERENCES users(id),
    form_msg_id   BIGINT NOT NULL DEFAULT 0,
    status        SMALLINT NOT NULL DEFAULT 0,  -- 0=pending, 1=approved, 2=rejected
    message       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(from_user_id, to_user_id)
);
CREATE INDEX IF NOT EXISTS idx_contact_requests_to_user ON contact_requests(to_user_id, status);
CREATE INDEX IF NOT EXISTS idx_contact_requests_from_user ON contact_requests(from_user_id);
```

### ContentType 新增

```go
ContentForm         ContentType = 10
ContentFormResponse ContentType = 11
```

### FormDefinitionBody（ContentType=10，body JSON）

所有用户可见文本由服务端在发送时根据 Accept-Language 翻译为最终文本后写入 body。

```json
{
  "form_id": "uuid-v4",
  "type": "contact_request",
  "title": "好友申请",
  "from_user_id": "user_a",
  "from_user_name": "张三",
  "from_user_avatar": "...",
  "request_id": 1,
  "message": "我是xxx",
  "actions": [
    {"action": "approve", "label": "通过", "style": "primary"},
    {"action": "reject", "label": "拒绝", "style": "danger"}
  ],
  "status": "active"
}
```

- `status` 字段：消息创建时写为 `"active"`。消息不可变，此值不随申请处理而更新。前端以 API 查询结果为准，body.status 仅作 API 返回前的初始占位。

### FormResponseBody（ContentType=11）

```json
{
  "form_msg_id": 123456789,
  "request_id": 1,
  "action": "approve",
  "responder_id": "user_b",
  "responder_name": "李四",
  "submitted_at": 1719532900000
}
```

`responder_id` 由客户端填写，服务端必须校验（== to_user_id）。

## 并发安全

### 发起申请的并发（A 双客户端同时发送）

- 两个客户端同时通过 pre-check → BEGIN → INSERT 时 UNIQUE(from_user_id, to_user_id) 保证只有一个成功
- 失败方捕获 PostgreSQL unique_violation → 返回 ErrContactRequestDuplicate → 事务回滚

### 处理 FormResponse 的并发（B 双设备同时点击）

- 设备1：SELECT FOR UPDATE row lock → UPDATE → COMMIT
- 设备2：等待锁释放 → 读到 status 已是 approved → 返回 "该申请已被处理"
- 用 `SELECT ... FOR UPDATE` 行锁序列化

## 表单状态判定机制（三级优先级）

| 优先级 | 来源 | 说明 |
|-------|------|------|
| 1（最高） | 用户本地操作 | 乐观 UI 本地内存 `localStatus` |
| 2 | API 查询 | `GET /by-form/:msg_id` 服务端权威状态 |
| 3（最低） | body.status | FormDefinitionBody 中写死的 `"active"`，仅 API 未返回前的初始占位 |

## 乐观更新失败回退

| 失败类型 | 触发条件 | 处理 |
|---------|---------|------|
| 网络超时 | 5s 内未收到 MsgSendAck | 回退 UI + "操作失败，请重试" |
| Ack 错误 | MsgSendAck.status != MsgSent | 同上 |
| 服务端业务拒绝 | MsgSendAck.ErrorCode != 0 | 显示具体原因 |
| 并发已处理 | 另一个设备先处理 | 收到 MsgPush(System Msg) → 回退 UI |
| 服务端崩了 | 连接断开 | 回退 UI + 重连后 API 确认状态 |

## 未读 Badge 跨设备一致性

- 利用已有 MarkRead + MsgReadNotify 机制
- 处理完表单后自动 mark read：`SetUserSeq(ctx, userID, sysConvID, formResponseMsg.convSeq)`

## 会话列表

- **查询**：`GET /api/v1/conversations` 返回所有 member 会话。系统会话懒创建，存在后始终返回
- **排序**：系统会话(type=3)固定置顶，其余按 last_msg_at DESC
- **预览**：Form→"好友申请 · {name}"，FormResponse→"你已通过/拒绝..."，System→body原文
- **新会话感知**：MsgPush 到未知 P2P/系统会话时客户端自动 refresh()

## 消息去重

| 消息来源 | sender_id | sender_session_id | 去重 |
|---------|-----------|-------------------|------|
| Form（服务端） | `""` | `strconv.FormatInt(msgID, 10)` | snowflake ID 唯一 |
| FormResponse（用户） | 用户 ID | 用户 session_id | 已有 client_seq 去重 |
| SystemMessage（服务端） | `""` | `strconv.FormatInt(msgID, 10)` | snowflake ID 唯一 |

## API 端点

| 方法 | 路径 | 处理函数 | 响应 |
|------|------|---------|------|
| `POST` | `/api/v1/contact-requests` | `ContactHandler.RequestContact` | `{ request_id, form_msg_id }` |
| `GET` | `/api/v1/contact-requests/sent` | `ContactHandler.ListSentRequests` | `ContactRequest[]` (分页) |
| `GET` | `/api/v1/contact-requests/received` | `ContactHandler.ListReceivedRequests` | `ContactRequest[]` (分页，`?status=`) |
| `GET` | `/api/v1/contact-requests/by-form/{msg_id}` | `ContactHandler.GetRequestByFormMsgID` | `ContactRequest` |

FormResponse 不走独立 API——通过 WebSocket MsgSend(content_type=11) 发送，ingest 拦截处理。

## 错误码

| 错误码 | 消息 | HTTP Status |
|-------|------|------------|
| `ErrContactRequestSelf` | 不能给自己发好友申请 | 400 |
| `ErrContactRequestDuplicate` | 已有待处理的好友申请 | 409 |
| `ErrAlreadyFriends` | 你们已经是好友 | 409 |
| `ErrContactRequestNotFound` | 好友申请不存在 | 404 |
| `ErrContactRequestAlreadyHandled` | 该申请已被处理 | 409 |

## i18n Key 表

| Key | 中文 | 英文 |
|-----|------|------|
| `contact_request.title` | 好友申请 | Friend Request |
| `contact_request.approve` | 通过 | Approve |
| `contact_request.reject` | 拒绝 | Reject |
| `contact_request.approved` | 已通过 | Approved |
| `contact_request.rejected` | 已拒绝 | Rejected |
| `contact_request.sent` | 你已向 {name} 发送了好友申请 | You sent a friend request to {name} |
| `contact_request.approved_by` | {name} 已通过你的好友申请 | {name} approved your friend request |
| `contact_request.rejected_by` | {name} 已拒绝你的好友申请 | {name} rejected your friend request |
| `contact_request.you_approved` | 你已通过 {name} 的好友申请 | You approved {name}'s friend request |
| `contact_request.you_rejected` | 你已拒绝 {name} 的好友申请 | You rejected {name}'s friend request |
| `contact_request.friend_established` | 你们已成为好友，可以开始聊天了 | You are now friends. Start chatting! |
| `err.contact_request_duplicate` | 已有待处理的好友申请 | A pending friend request already exists |
| `err.contact_request_already_friends` | 你们已经是好友 | You are already friends |
| `err.contact_request_not_found` | 好友申请不存在 | Friend request not found |
| `err.contact_request_self` | 不能给自己发好友申请 | Cannot send friend request to yourself |
| `err.contact_request_already_handled` | 该申请已被处理 | This request has already been handled |

策略：服务端在 SendFormMessage/SendSystemMessage 时根据 Accept-Language 翻译为最终文本写入 body，前端直接渲染。

## 实现步骤

### Phase 1: 数据模型（Go 服务端）
- `server/pkg/model/conversation.go` — `ConvSystem = 3`
- `server/pkg/model/message.go` — `ContentForm = 10, ContentFormResponse = 11`
- `server/pkg/model/form.go` — FormDefinitionBody, FormResponseBody, FormAction
- `server/pkg/model/contact_request.go` — ContactRequest 结构体 + 状态常量
- `server/pkg/model/errors.go` — 5 个新错误码
- `server/pkg/model/form_test.go` — JSON 往返测试
- `013_contact_requests.sql` — 建表
- `contact_requests.go` — 仓库（Insert, UpdateStatusTx, GetByID, GetByFormMsgID, GetByPair, ListSent, ListReceived, Delete）

### Phase 2: 系统会话 + 表单消息基础设施
- `conversation/manager.go` — GetOrCreateSystemConv, GetOrCreateSystemConvTx
- `storage/db/conversations.go` — CreateTx
- `message/ingest.go` — SendFormMessage + FormResponse 拦截处理
- `pkg/i18n/messages.go` — 15 个 key

### Phase 3: 好友申请 API
- `api/contact.go` — RequestContact, ListSentRequests, ListReceivedRequests, GetRequestByFormMsgID
- `api/contact.go` Remove — 双向删除
- `api/router.go` — 4 条路由
- `cmd/panda_ai/main.go` — 依赖注入

### Phase 4: Web 前端
- `types/message.ts` + `types/form.ts` — 类型
- `services/contact-request-service.ts` — API 封装
- `form-bubble.tsx` — 好友申请卡片 + 三级状态判定 + 乐观更新回退
- `form-response-bubble.tsx` — 回复气泡
- `message-bubble.tsx` — 分发 Form/FormResponse
- `chat-view.tsx` — 系统会话隐藏 InputBar
- `conversation-store.ts` — 置顶 + 预览 + 自动 refresh

### Phase 5: Swift 原生客户端
- `Conversation.swift`, `Message.swift`, `FormBody.swift` — 模型
- `ContactService.swift` — +4 API
- `ChatViewModel.swift` — sendFormResponse + 乐观更新回退
- `ConversationListViewModel.swift` — 置顶 + 预览 + refresh
- macOS/iOS: FormBubbleView, FormResponseBubbleView, MessageBubble 分发, 隐藏 input bar

### Phase 6: 测试
- Go: form_test, contact_requests_test, ingest_test, manager_test, handlers_test
- Web e2e: friend-request.spec.ts（完整流程 + 拒绝 + 并发 + 超时回退 + 删好友重新申请）
- Swift: FormBodyTests

## 验证清单

| # | 场景 | 预期 |
|---|------|------|
| 1 | 存量用户首次收到系统消息 | 系统会话懒创建 |
| 2 | A → B 好友申请 | B 系统会话收到 Form 卡片（WebSocket 实时） |
| 3 | B 通过 | contacts 双向创建 + P2P 创建 + 双方通知 + 会话列表刷新 |
| 4 | B 拒绝 | 不创建 contacts/P2P，双方收到拒绝通知 |
| 5 | 双设备并发通过 | 一个成功一个返回 "该申请已被处理" |
| 6 | 乐观更新超时 | 按钮回退可点击 + 错误提示 |
| 7 | C 冒充 B 发送 | 服务端拒绝 |
| 8 | 重复处理 | 服务端拒绝 |
| 9 | B 重新打开会话 | `/by-form/:msg_id` 确认已处理状态 |
| 10 | 不能给自己发 | 返回错误 |
| 11 | 已是好友 | 返回错误（双向检查） |
| 12 | 已有 pending | 返回错误 |
| 13 | 删好友后重新申请 | 允许 |
| 14 | 未读 badge 跨设备 | MarkRead + MsgReadNotify 同步消除 |
| 15 | P2P 新会话感知 | MsgPush 触发自动刷新 |
| 16 | 会话预览 | 系统会话显示正确文案 |
| 17 | 系统会话不可输入 | InputBar 隐藏 |
| 18 | 排序 | 系统会话置顶 |
| 19 | i18n | 双语正确 |
| 20 | 旧客户端 | 回退 JSON 文本（不崩溃） |
| 21 | 所有测试 | `go test` / Playwright / XCTest 通过 |
