# 多终端同时登陆与消息同步

## 1. 多终端设计

核心思想：**每个终端独立持有 Session，按 User 维度聚合。**

```
User "张三"
  ├── Session A: iPhone 15
  ├── Session B: MacBook Pro
  └── Session C: Chrome 网页
```

### 登录流程

```mermaid
sequenceDiagram
    participant A as 终端A
    participant S as 服务端
    participant B as 终端B(已在线)

    A->>S: Login (token, device)
    S->>S: 验证 token
    S->>S: 创建 SessionA
    S->>S: 绑定 ConnID
    S-->>A: LoginResp (session_id)
    S->>B: session.online (user_id, device)
```

### 登出流程

```mermaid
sequenceDiagram
    participant A as 终端A
    participant S as 服务端
    participant B as 终端B(已在线)

    A->>S: Logout
    S->>S: 清理 SessionA
    S->>S: 关闭 WebSocket
    S->>B: session.offline (user_id)
```

### 互踢策略

| 模式 | 行为 | 适用场景 |
|------|------|---------|
| 共存（默认） | N 个终端共存 | 通用 IM |
| 互踢 | 新终端挤掉旧终端 | 安全敏感应用 |

---

## 2. 消息发送与多终端扇出

### 发消息流程

```mermaid
sequenceDiagram
    participant Sender as 发送者
    participant Server as 服务端
    participant R1 as 接收方A
    participant R2 as 接收方B

    Sender->>Server: SendMsg (conv_id, body)
    Server->>Server: 生成 MsgID (Snowflake)
    Server->>Server: 校验发送权限
    Server->>Server: 写入 messages 表
    Server-->>Sender: SendAck (msg_id, timestamp)
    Server->>Server: 查询 Conv 所有成员
    Server->>R1: message.push (并发推送)
    Server->>R2: message.push (并发推送)
    Note over R2: 离线 → 存 MQ
```

### 扇出逻辑

```mermaid
flowchart LR
    A[消息路由] --> B{成员列表}
    B --> C[成员A]
    B --> D[成员B]
    B --> E[成员C]
    C --> F{在线?}
    D --> G{在线?}
    E --> H{在线?}
    F -->|是| I[推送Session A1]
    F -->|是| J[推送Session A2]
    G -->|否| K[离线消息队列]
    H -->|是| L[推送Session C1]
```

---

## 3. 增量消息同步

每个 Session 独立维护 `last_seq`，标记已接收的最新序号。

### 首次登录（拉取历史消息）

首次进入会话时 `last_conv_seq=0`，拉取最近消息。

```mermaid
sequenceDiagram
    participant C as 客户端
    participant S as 服务端

    C->>S: sync.req (conv_id=user_a:user_b, last_conv_seq=0, limit=50)
    S-->>C: sync.res (messages[...50条], has_more=true)
    C->>S: sync.req (conv_id=user_a:user_b, last_conv_seq=50, limit=50)
    S-->>C: sync.res (messages[...], has_more=false)
```

### 增量同步（断线重连）

断线重连后，对每个已知会话并发 sync。

```mermaid
sequenceDiagram
    participant C as 客户端
    participant S as 服务端

    par 会话A
        C->>S: sync.req (conv_id=user_a:user_b, last_conv_seq=50)
        S-->>C: sync.res (messages[51..100])
    and 会话B
        C->>S: sync.req (conv_id=group_gKx3mNpQ8, last_conv_seq=20)
        S-->>C: sync.res (messages[21..30])
    end
```

### 实时推送 + ack

```mermaid
sequenceDiagram
    participant C as 客户端
    participant S as 服务端

    S-->>C: message.push (conv_seq=201)
    C->>S: message.received (type=12, conv_id, conv_seq=201)
    S->>S: session:seq:{session_id}:{conv_id} = 201
```

### 未读数计算

```
某个会话的未读数 = 该会话最新 conv_seq - user_seq

user_seq 取该用户所有 Session 中该会话的 conv_seq 最大值
    Redis: user:seq:{user_id}:{conv_id} = max(session:seq:*)

例：
  UserA 有两个终端
    手机: session:seq:s1:user_a:user_b = 100
    电脑: session:seq:s2:user_a:user_b = 80
    → user:seq:userA:user_a:user_b = max(100, 80) = 100
    会话最新 conv_seq = 105
    → 未读数 = 105 - 100 = 5
```

### 同步关键点

| 组件 | 存储位置 | 说明 |
|------|---------|------|
| session:seq | Redis | 每个 Session 在每个会话的已读位置 |
| user:seq | Redis | 用户维度聚合（取 Session 最大值） |
| 离线消息 | DB 持久化 + Redis Stream | Redis 仅做推送缓冲 |

---

## 4. 心跳与连接保活

```mermaid
sequenceDiagram
    participant C as 客户端
    participant S as 服务端

    loop 每30秒
        C->>S: ping
        S-->>C: pong
    end

    Note over S: 90秒未收到 ping
    S->>S: 标记 Session 断开
    S->>S: 广播 session.offline
```

### Session 恢复

断网重连后：
1. 客户端重建 WebSocket 连接（token 在 URL 中鉴权）
2. 客户端发送 `SessionRecover` (type=43) 携带旧 session_id 绑定新连接
3. 服务端恢复 Session，更新 ConnID
4. 执行增量同步拉取断网期间消息
