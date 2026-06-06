# Phase 1 Server Feature 列表与 E2E 测试

> 服务端功能，E2E 通过 HTTP + WebSocket API 直接验证。

---

## 1. 用户注册 / 登录 / JWT 鉴权

### Feature

- 用户注册：提供 name + password，返回 user_id + JWT token
- 用户登录：提供 user_id + password，返回 JWT token + expires_at
- Token 鉴权：HTTP 接口统一 Bearer token 认证
- 用户信息查询：GET /api/v1/users/me

### E2E

**E2E-S1.1: 注册 → 登录成功**

1. POST /api/v1/users/register {name: "张三", password: "abc123"}
2. 断言：返回 code=0, data.user_id 格式 user_xxxxx, data.token 非空
3. POST /api/v1/users/login {user_id, password: "abc123"}
4. 断言：返回 code=0, data.token 非空, data.expires_at > 当前时间

**E2E-S1.2: Token 鉴权拦截**

1. GET /api/v1/users/me（Authorization: Bearer valid_token）→ 返回用户信息
2. GET /api/v1/users/me（无 Authorization）→ 返回 4002
3. GET /api/v1/users/me（Authorization: Bearer invalid_token）→ 返回 4002

**E2E-S1.3: 错误密码登录**

1. POST /api/v1/users/login {user_id, password: "wrong"}
2. 断言：返回 code=4002

---

## 2. 文本消息收发

### Feature

- ContentType 0=Text 文本消息存储与路由
- ContentType 5=System 系统消息（加群/退群通知）
- conv_seq 每个会话独立递增
- 发送者不收到自己的推送
- 消息持久化到 PostgreSQL

### E2E

**E2E-S2.1: P2P 消息收发**

1. UserA, UserB 在线
2. UserA → WS: MsgSend (conv_id="user_a:user_b", content_type=0, body="hello", client_seq=1)
3. 断言：UserA 收到 MsgSendAck (msg_id>0, status=1, client_seq=1)
4. 断言：UserB 收到 MsgPush (msg_id 与 Ack 一致, conv_seq>=1, sender_id="user_a")
5. 断言：UserA 不收到自己的推送

**E2E-S2.2: 群聊消息收发**

1. 群 conv_group 有 UserA, UserB, UserC
2. UserA → WS: MsgSend (conv_id=conv_group, body="群消息")
3. 断言：UserB 收到 MsgPush
4. 断言：UserC 收到 MsgPush
5. 断言：UserA 不收到自己的推送

**E2E-S2.3: conv_seq 递增**

1. 发送 3 条消息到同一会话
2. 断言：三条消息的 conv_seq 分别为 N, N+1, N+2

**E2E-S2.4: 消息持久化**

1. 发送消息后 GET /api/v1/conversations/:conv_id/messages?limit=50
2. 断言：返回的消息包含刚发送的消息

---

## 3. P2P 会话隐式创建

### Feature

- P2P ConvID = 双方 user_id 字典序拼接
- 首次发消息自动创建会话并添加双方成员
- 目标用户不在线 → 消息存储，上线后同步

### E2E

**E2E-S3.1: 首次发消息自动创建 P2P**

1. UserA, UserB 在线
2. UserA → WS: MsgSend (conv_id="user_a:user_b", body="你好")
3. 断言：UserA 收到 MsgSendAck
4. 断言：UserB 收到 MsgPush
5. UserA GET /api/v1/conversations → 列表包含 "user_a:user_b"
6. UserA GET /api/v1/conversations/user_a:user_b → type=1 (P2P)

---

## 4. 群聊管理

### Feature

- 创建群聊，创建者自动成为 Owner
- 添加 / 移除群成员
- 退出群聊
- 系统消息通知（加人/退群/踢人）
- 权限校验（管理员才能踢人）

### E2E

**E2E-S4.1: 创建群聊**

1. POST /api/v1/conversations/group {name: "测试群", member_ids: ["user_b", "user_c"]}
2. 断言：返回 data.conv_id 以 group_ 开头, data.member_count=3
3. 断言：UserB 收到 MsgPush (content_type=5)
4. 断言：UserC 收到 MsgPush (content_type=5)

**E2E-S4.2: 添加成员**

1. POST /api/v1/conversations/:conv_id/members {user_ids: ["user_d"]}
2. 断言：返回 data.added=["user_d"]
3. 断言：UserD 收到 MsgPush (content_type=5)

**E2E-S4.3: 移除成员**

1. Owner DELETE /api/v1/conversations/:conv_id/members/:user_c
2. 断言：返回 200
3. 断言：UserC 收到 MsgPush (content_type=5)
4. UserC 发消息到该群 → 断言：服务端拒绝

**E2E-S4.4: 退出群聊**

1. UserD POST /api/v1/conversations/:conv_id/leave
2. 断言：返回 200
3. 断言：其他成员收到系统消息

**E2E-S4.5: 非管理员踢人**

1. UserB（普通成员）DELETE /api/v1/conversations/:conv_id/members/:user_d
2. 断言：返回 4003

**E2E-S4.6: 群详情**

1. GET /api/v1/conversations/:conv_id
2. 断言：返回 type=2, owner_id, members 含 role 字段

---

## 5. 多终端扇出

### Feature

- 一个用户多个 WebSocket 连接并行在线
- 消息推送到接收方所有在线终端
- 用户上线/下线通知

### E2E

**E2E-S5.1: 多终端接收**

1. UserB 建立两个 WS 连接 B1, B2
2. UserA 发送 MsgSend
3. 断言：B1 收到 MsgPush
4. 断言：B2 收到 MsgPush（与 B1 完全一致）

**E2E-S5.2: 多终端发送**

1. UserA 建立两个连接 A1, A2
2. UserA 通过 A1 发送消息
3. 断言：A1 收到 MsgSendAck
4. 断言：A2 不收到任何推送

**E2E-S5.3: 上线通知**

1. UserB 在线
2. UserA 建立 WS 连接
3. 断言：UserB 收到 SessionOnline (type=41, user_id="user_a")

**E2E-S5.4: 下线通知**

1. UserB 在线
2. UserA 断开 WS 连接
3. 断言：UserB 收到 SessionOffline (type=42, user_id="user_a")

---

## 6. 增量消息同步

### Feature

- 每个 Session 独立维护 session:seq
- sync.req → sync.res 按会话拉取增量
- 分页（limit + has_more）

### E2E

**E2E-S6.1: sync.req → sync.res**

1. conv 中已有 5 条消息
2. UserA 新建 WS 连接 → SyncReq (conv_id, last_conv_seq=0, limit=50)
3. 断言：收到 SyncRes (messages 5 条, has_more=false)
4. 断言：每条消息有 msg_id, conv_seq, sender_id, body, timestamp

**E2E-S6.2: 增量分页**

1. conv 有 60 条消息
2. SyncReq (last_conv_seq=0, limit=50) → 返回 50 条, has_more=true
3. SyncReq (last_conv_seq=50, limit=50) → 返回 10 条, has_more=false

---

## 7. 消息历史拉取

### Feature

- GET /api/v1/conversations/:conv_id/messages
- before_msg_id 翻页
- 按时间倒序，limit 上限 100

### E2E

**E2E-S7.1: 翻页**

1. conv 有 80 条消息
2. GET ?limit=50 → 50 条, has_more=true
3. GET ?before_msg_id=最早一条的msg_id&limit=50 → 30 条, has_more=false

**E2E-S7.2: 空会话**

1. GET /api/v1/conversations/:new_conv_id/messages?limit=50
2. 断言：messages=[], has_more=false

**E2E-S7.3: limit 上限**

1. GET ?limit=200 → 最多返回 100 条

---

## 8. 已读回执

### Feature

- POST /api/v1/conversations/:conv_id/read 上报已读
- 服务端转发 MsgReadNotify 到发送方
- 未读数 = 最新 conv_seq - user_seq

### E2E

**E2E-S8.1: 标记已读 → 通知发送方**

1. UserA 发送消息 (msg_id=1001)
2. UserB POST /api/v1/conversations/:conv_id/read {msg_id: 1001}
3. 断言：UserB 返回 200
4. 断言：UserA 收到 MsgReadNotify (type=32, user_id="user_b")

**E2E-S8.2: 未读数变化**

1. UserB 在 conv 中有 5 条未读
2. GET /api/v1/conversations → conv 的 unread_count=5
3. UserB 标记已读到最新 msg_id
4. GET /api/v1/conversations → conv 的 unread_count=0

---

## 9. 消息去重

### Feature

- 唯一索引 (SenderID, SessionID, ClientSeq)
- 重复消息返回已有 MsgID

### E2E

**E2E-S9.1: 同 Session 去重**

1. UserA → WS: MsgSend (client_seq=100, body="hello") → Ack msg_id=1001
2. UserA → WS: MsgSend (client_seq=100, body="hello") → Ack msg_id=1001（相同）
3. GET /api/v1/conversations/:conv_id/messages → 只有 1 条 msg_id=1001

**E2E-S9.2: 不同 Session 不冲突**

1. UserA 建立两个连接 A1, A2
2. A1 → WS: MsgSend (client_seq=1, body="from A1") → Ack msg_id=1001
3. A2 → WS: MsgSend (client_seq=1, body="from A2") → Ack msg_id=1002（不同）

---

## 10. 联系人管理

### Feature

- HTTP CRUD：添加 / 删除 / 修改备注
- 联系人列表含在线状态

### E2E

**E2E-S10.1: 联系人 CRUD**

1. POST /api/v1/contacts {user_id: "user_b"} → 200
2. GET /api/v1/contacts → items 包含 user_b
3. PUT /api/v1/contacts/user_b {nickname: "Bobo"}
4. GET /api/v1/contacts → user_b 的 nickname="Bobo"
5. DELETE /api/v1/contacts/user_b
6. GET /api/v1/contacts → items 不包含 user_b

**E2E-S10.2: 在线状态**

1. UserB 在线
2. GET /api/v1/contacts → user_b 的 status=1 (Online)

---

## 11. 会话列表

### Feature

- 按最后消息时间倒序
- 含未读数、最后一条消息预览

### E2E

**E2E-S11.1: 会话列表结构**

1. UserA 参与 conv_A, conv_B
2. GET /api/v1/conversations?page=1&size=20
3. 断言：每个 item 含 conv_id, type, name, unread_count, last_message, last_msg_at
4. 断言：last_message 含 msg_id, sender_id, body, content_type, timestamp

**E2E-S11.2: 未读数随已读变化**

1. UserA 在 conv_A 有 3 条未读 → unread_count=3
2. 标记已读 → unread_count=0

---

## 12. 用户在线状态

### Feature

- 用户信息返回在线设备列表
- 上下线自动切换 status

### E2E

**E2E-S12.1: 在线/离线切换**

1. UserB 登录建立 WS → UserA GET /api/v1/users/user_b → status=1
2. UserB 断开所有 WS → UserA GET /api/v1/users/user_b → status=0

**E2E-S12.2: 批量拉取**

1. POST /api/v1/users/batch {user_ids: ["user_a", "user_b", "不存在"]}
2. 断言：返回 users 包含 user_a, user_b
3. 断言：不存在的 user_id 不在返回 map 中

---

## 13. 心跳 / 连接保活

### Feature

- Ping/Pong 30s 间隔
- 90s 无心跳 → 断开
- SessionRecover 重连恢复

### E2E

**E2E-S13.1: Ping/Pong**

1. 客户端 → WS: Ping (type=61) → 收到 Pong (type=62)

**E2E-S13.2: 超时断开**

1. 客户端建立 WS 后不发送任何消息
2. 等待 100s → 连接被服务端关闭

**E2E-S13.3: SessionRecover**

1. 客户端登录获得 session_id
2. 断开 WS → 新建 WS → SessionRecover (type=43, session_id)
3. 断言：收到 SessionRecoverAck (type=44)
4. 发送 SyncReq → 正常收到 SyncRes

**E2E-S13.4: 过期拒绝**

1. 获得 session_id → 等待过期
2. SessionRecover (session_id=过期ID) → 收到 Error (code=4004)

---

## 14. 基础限流

### Feature

- 消息发送频率限制（按用户）
- 消息体大小上限校验

### E2E

**E2E-S14.1: 频率超限**

1. UserA 连续发送超过上限的消息（如 30条/秒）
2. 断言：第 N+1 条收到 Error (code=4003)

**E2E-S14.2: 消息体超长**

1. UserA → WS: MsgSend (body=超过上限的字符串, 如 10KB)
2. 断言：收到 Error (code=4005)

---

## 15. 监控指标

### Feature

- /metrics 暴露 Prometheus 指标

### E2E

**E2E-S15.1: 指标暴露**

1. GET /metrics
2. 断言：包含 im_connections_total, im_messages_sent_total, im_messages_push_total

**E2E-S15.2: 连接数指标**

1. 建立 2 个 WS 连接 → im_connections_total = 2
2. 断开 1 个 → im_connections_total = 1
