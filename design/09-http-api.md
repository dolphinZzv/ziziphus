# HTTP REST API 设计

## 设计原则

- **HTTP API 负责数据查询与操作**，实时消息走 WebSocket
- 认证方式：Header `Authorization: Bearer {jwt_token}`
- 统一响应格式
- 分页统一：`page` + `size` 参数

---

## 1. 统一响应格式

### 成功响应

```json
{
  "code": 0,
  "msg": "ok",
  "data": {}
}
```

### 错误响应

```json
{
  "code": 4001,
  "msg": "消息内容不能为空",
  "data": null
}
```

### 分页响应

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "items": [],
    "total": 100,
    "page": 1,
    "size": 20
  }
}
```

### 通用错误码

| code | 说明 |
|------|------|
| 0 | 成功 |
| 4001 | 参数错误 |
| 4002 | 未授权（token 无效） |
| 4003 | 权限不足 |
| 4004 | 资源不存在 |
| 4005 | 频率限制 |
| 5001 | 服务端内部错误 |

---

## 2. 用户 API

### 2.1 注册

```
POST /api/v1/users/register
```

**Request:**
```json
{
  "name": "张三",
  "password": "abc123",
  "avatar": ""
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "user_id": "user_xxxxx",
    "name": "张三",
    "token": "jwt_token_string"
  }
}
```

### 2.2 登录

```
POST /api/v1/users/login
```

**Request:**
```json
{
  "user_id": "user_xxxxx",
  "password": "abc123"
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "user_id": "user_xxxxx",
    "name": "张三",
    "token": "jwt_token_string",
    "expires_at": 1700000000
  }
}
```

### 2.3 获取用户信息

```
GET /api/v1/users/:user_id
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "user_id": "user_xxxxx",
    "name": "张三",
    "avatar": "https://...",
    "type": 0,
    "status": 1,
    "online_devices": [
      {"device": 1, "device_name": "MacBook Pro", "last_active": 1700000000}
    ]
  }
}
```

> `type`: 0=Human, 1=Agent
> `status`: 0=Offline, 1=Online, 2=Busy
> `online_devices` 仅返回当前在线设备列表

### 2.4 批量获取用户信息

```
POST /api/v1/users/batch
```

**Request:**
```json
{
  "user_ids": ["user_a", "user_b", "user_c"]
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "users": {
      "user_a": {"user_id": "user_a", "name": "Alice", "avatar": "...", "type": 0, "status": 1},
      "user_b": {"user_id": "user_b", "name": "Bob", "avatar": "...", "type": 0, "status": 0}
    }
  }
}
```

> 不存在的 user_id 不会出现在返回 map 中
> 客户端打开会话列表时批量拉取所有相关用户信息

### 2.5 更新个人信息

```
PUT /api/v1/users/me
```

**Request:**
```json
{
  "name": "张三的新名字",
  "avatar": "https://..."
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "user_id": "user_xxxxx",
    "name": "张三的新名字",
    "avatar": "https://..."
  }
}
```

### 2.6 搜索用户

```
GET /api/v1/users/search?q=张三&page=1&size=20
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "items": [
      {"user_id": "user_xxx", "name": "张三", "avatar": "...", "type": 0, "status": 1}
    ],
    "total": 1,
    "page": 1,
    "size": 20
  }
}
```

---

## 3. 会话 API

### 3.1 获取会话列表

```
GET /api/v1/conversations?page=1&size=20
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "items": [
      {
        "conv_id": "user_a:user_b",
        "type": 1,
        "name": "Bob",
        "avatar": "https://...",
        "unread_count": 3,
        "last_message": {
          "msg_id": 2024000100,
          "sender_id": "user_b",
          "body": "好的，明天见",
          "content_type": 0,
          "timestamp": 1700000000000,
          "status": 2
        },
        "last_msg_at": 1700000000000,
        "mention_me": false
      },
      {
        "conv_id": "group_gKx3mNpQ8",
        "type": 2,
        "name": "项目讨论组",
        "avatar": "https://...",
        "unread_count": 5,
        "last_message": {
          "msg_id": 2024000200,
          "sender_id": "user_c",
          "body": "@张三 看一下这个 PR",
          "content_type": 0,
          "timestamp": 1700000000100,
          "status": 1
        },
        "last_msg_at": 1700000000100,
        "mention_me": true
      }
    ],
    "total": 10,
    "page": 1,
    "size": 20
  }
}
```

> `type`: 1=P2P, 2=Group
> `unread_count`: 该会话未读消息数（按用户维度聚合，同一用户多终端共享）
> `mention_me`: 是否有 @我的消息

### 3.2 获取会话详情

```
GET /api/v1/conversations/:conv_id
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "conv_id": "group_gKx3mNpQ8",
    "type": 2,
    "name": "项目讨论组",
    "avatar": "https://...",
    "owner_id": "user_a",
    "member_count": 5,
    "created_at": 1700000000000,
    "members": [
      {"user_id": "user_a", "role": 2, "nickname": "", "online": true},
      {"user_id": "user_b", "role": 0, "nickname": "", "online": true},
      {"user_id": "user_c", "role": 1, "nickname": "小C", "online": false}
    ]
  }
}
```

> `role`: 0=Member, 1=Admin, 2=Owner
> `online`: 当前是否至少有一端在线

### 3.3 创建群聊

```
POST /api/v1/conversations/group
```

**Request:**
```json
{
  "name": "项目讨论组",
  "avatar": "https://...",
  "member_ids": ["user_b", "user_c", "user_d"]
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "conv_id": "group_gKx3mNpQ8",
    "name": "项目讨论组",
    "member_count": 4,
    "created_at": 1700000000000
  }
}
```

> 创建者自动成为群主 (Owner)

### 3.4 添加群成员

```
POST /api/v1/conversations/:conv_id/members
```

**Request:**
```json
{
  "user_ids": ["user_e", "user_f"]
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "added": ["user_e", "user_f"],
    "failed": []
  }
}
```

### 3.5 移除群成员

```
DELETE /api/v1/conversations/:conv_id/members/:user_id
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": null
}
```

### 3.6 退出群聊

```
POST /api/v1/conversations/:conv_id/leave
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": null
}
```

### 3.7 更新群信息

```
PUT /api/v1/conversations/:conv_id
```

**Request:**
```json
{
  "name": "新群名",
  "avatar": "https://..."
}
```

---

## 4. 消息 API

### 4.1 获取消息历史

```
GET /api/v1/conversations/:conv_id/messages?before_msg_id=2024001000&limit=50
```

| 参数 | 说明 |
|------|------|
| `before_msg_id` | 拉取此 msg_id 之前的更早消息（不包含该条）。为空时拉取最新消息 |
| `limit` | 数量，默认 50，最大 100 |

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "messages": [
      {
        "msg_id": 2024000999,
        "conv_id": "user_a:user_b",
        "sender_id": "user_b",
        "content_type": 0,
        "body": "在吗？",
        "reply_to": 0,
        "timestamp": 1700000000000,
        "seq": 151,
        "status": 2
      }
    ],
    "has_more": true
  }
}
```

> 增量同步走 WebSocket (type=21)，这里是用于"加载更多历史消息"的 HTTP API

### 4.2 搜索消息

```
GET /api/v1/messages/search?q=关键词&conv_id=group_gKx3mNpQ8&sender_id=user_b&start_time=1700000000&end_time=1700000100&page=1&size=20
```

| 参数 | 说明 |
|------|------|
| `q` | 搜索关键词 |
| `conv_id` | 可选，限定会话 |
| `sender_id` | 可选，限定发送者 |
| `start_time` | 可选，起始时间戳 |
| `end_time` | 可选，结束时间戳 |

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "items": [
      {
        "msg_id": 2024000500,
        "conv_id": "group_gKx3mNpQ8",
        "sender_id": "user_b",
        "content_type": 0,
        "body": "这个关键词匹配的结果",
        "timestamp": 1700000000500,
        "seq": 300
      }
    ],
    "total": 1,
    "page": 1,
    "size": 20
  }
}
```

> 需要 Elasticsearch 索引支持

### 4.3 撤回消息

撤回本质是在会话中插入一条 `ContentType=MsgRecall` 的系统消息，客户端据此展示"某某撤回了一条消息"。

```
POST /api/v1/conversations/:conv_id/messages/:msg_id/recall
```

**Request:**
```json
{
  "reason": "发错了"
}
```

服务端行为：
1. 将原消息标记为 `deleted=true`
2. 在会话中插入一条系统消息（type=MsgRecall, body=`{"msg_id":原ID}`）
3. 通过 WebSocket 推送给所有在线成员

### 4.4 编辑消息

编辑本质是在会话中插入一条 `ContentType=MsgEdit` 的系统消息。

```
PUT /api/v1/conversations/:conv_id/messages/:msg_id
```

**Request:**
```json
{
  "body": "修改后的内容"
}
```

服务端行为：
1. 更新原消息 body
2. 在会话中插入一条系统消息（type=MsgEdit, body=`{"msg_id":原ID,"body":"新内容"}`）
3. 通过 WebSocket 推送给所有在线成员

> 撤回和编辑不涉及复杂的版本管理、编辑窗口等逻辑，与普通消息共享同一通道。

## 5. 联系人 API

### 5.1 获取联系人列表

```
GET /api/v1/contacts?page=1&size=50
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "items": [
      {
        "user_id": "user_b",
        "name": "Bob",
        "avatar": "https://...",
        "status": 1,
        "nickname": "Bobo",
        "added_at": 1700000000000
      }
    ],
    "total": 30,
    "page": 1,
    "size": 50
  }
}
```

### 5.2 添加联系人

```
POST /api/v1/contacts
```

**Request:**
```json
{
  "user_id": "user_b"
}
```

### 5.3 删除联系人

```
DELETE /api/v1/contacts/:user_id
```

### 5.4 修改联系人备注

```
PUT /api/v1/contacts/:user_id
```

**Request:**
```json
{
  "nickname": "新的备注名"
}
```

---

## 6. 文件 API

> Phase 2 实现，Phase 1 不涉及文件上传/下载。

### 6.1 上传文件

```
POST /api/v1/files/upload
```

**Request:** `multipart/form-data`

| 字段 | 说明 |
|------|------|
| `file` | 文件内容 |
| `file_type` | 0=image, 1=file, 2=audio, 3=video |

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "file_id": "file_xxxxx",
    "url": "/files/xxx.png",
    "thumbnail_url": "/files/thumb_xxx.png",
    "size": 102400,
    "name": "photo.png",
    "width": 1920,
    "height": 1080
  }
}
```

### 6.2 上传文件 → 发消息流程

上传文件不等于发送消息。客户端上传文件后还需发一条消息将文件分享到会话：

```
1. 客户端 POST /api/v1/files/upload  →  拿到 file_id + url
2. 客户端通过 WebSocket 发 message.send:
   {
     "type": 1,
     "payload": {
       "conv_id": "user_a:user_b",
       "content_type": 1,        ← Image (或 2=File)
       "body": "{\"file_id\":\"file_xxxxx\",\"url\":\"/files/xxx.png\",\"name\":\"photo.png\"}",
       "client_seq": 43
     }
   }
3. 服务端收到后存储消息，按正常流程推送给接收方
4. 接收方客户端根据 content_type=1 展示图片，url 指向文件服务
```

Phase 1 文件存储在本机磁盘，文件服务由 IM Server 的 HTTP 静态路由提供。

### 6.3 获取文件信息

```
GET /api/v1/files/:file_id
```

---

## 7. 会话未读数 API

### 7.1 获取总未读数

```
GET /api/v1/conversations/unread/total
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "total_unread": 15,
    "mention_count": 2
  }
}
```

> `mention_count`: 包含 @我的未读会话数

### 7.2 标记会话已读

```
POST /api/v1/conversations/:conv_id/read
```

**Request:**
```json
{
  "msg_id": 2024000010
}
```

> 标记该会话中截至 `msg_id` 的所有消息为已读

---

## 8. API 路由一览

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/users/register | 注册 |
| POST | /api/v1/users/login | 登录 |
| GET | /api/v1/users/me | 获取自己的信息 |
| GET | /api/v1/users/:user_id | 获取用户信息 |
| POST | /api/v1/users/batch | 批量获取用户信息 |
| PUT | /api/v1/users/me | 更新个人信息 |
| GET | /api/v1/users/search | 搜索用户 |
| GET | /api/v1/conversations | 获取会话列表 |
| GET | /api/v1/conversations/:conv_id | 获取会话详情 |
| POST | /api/v1/conversations/group | 创建群聊 |
| PUT | /api/v1/conversations/:conv_id | 更新群信息 |
| POST | /api/v1/conversations/:conv_id/members | 添加群成员 |
| DELETE | /api/v1/conversations/:conv_id/members/:user_id | 移除群成员 |
| POST | /api/v1/conversations/:conv_id/leave | 退出群聊 |
| GET | /api/v1/conversations/:conv_id/messages | 获取消息历史 |
| POST | /api/v1/conversations/:conv_id/messages/:msg_id/recall | 撤回消息 |
| PUT | /api/v1/conversations/:conv_id/messages/:msg_id | 编辑消息 |
| GET | /api/v1/messages/search | 搜索消息 |
| GET | /api/v1/contacts | 获取联系人列表 |
| POST | /api/v1/contacts | 添加联系人 |
| DELETE | /api/v1/contacts/:user_id | 删除联系人 |
| PUT | /api/v1/contacts/:user_id | 修改备注 |
| POST | /api/v1/files/upload | 上传文件 |
| GET | /api/v1/files/:file_id | 获取文件信息 |
| GET | /api/v1/conversations/unread/total | 总未读数 |
| POST | /api/v1/conversations/:conv_id/read | 标记已读 |

---

## 9. 客户端调用策略

### 启动时调用顺序

```
App 启动
  │
  1. GET /api/v1/users/me              ← 获取自己的信息 + 验证 token
  2. GET /api/v1/contacts              ← 加载联系人列表
  3. POST /api/v1/users/batch          ← 批量拉取联系人的详细信息
  4. GET /api/v1/conversations         ← 加载会话列表（含未读数）
  5. WebSocket 连接                    ← 建立长连接，开始实时收发
```

### 打开聊天页面时

```
打开会话
  │
  1. GET /api/v1/conversations/:conv_id        ← 会话详情 + 成员列表
  2. POST /api/v1/users/batch                  ← 批量拉取成员信息
  3. GET /api/v1/conversations/:conv_id/messages ← 加载最近消息
  4. WebSocket 开始接收实时推送                  ← 新消息实时流入
```
