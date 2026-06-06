# 消息路由设计（P2P 与群聊）

## 1. 消息路由总流程

```mermaid
flowchart TD
    A[WebSocket 消息入站] --> B[协议解析 + 消息校验]
    B --> C[查询 Conv + 校验权限]
    C --> D{Conv 类型}
    D -->|P2P| E[P2P 路由]
    D -->|Group| F[群聊路由]
    E --> G[查询各接收者在线 Session]
    F --> G
    G --> H[并行推送所有在线终端]
    G --> I[离线消息写入 MQ]
```

---

## 2. P2P 消息路由

### ConvID 说明

ConvID 不编码类型信息（无 `p2p:` 前缀）。服务端收到消息时通过 ConvID 查 conversations 表获取会话类型（P2P/Group）。

### 路由流程

```mermaid
sequenceDiagram
    participant A as UserA
    participant S as 服务端
    participant B as UserB

    A->>S: 发送消息 (conv, body)
    S->>S: 解析 ConvID → P2P
    S->>S: 接收者 = UserB
    S->>S: 查找 UserB 在线 Session
    S->>B: 并行推送
    S->>S: 记录送达回执
    S-->>A: Ack
```

### P2P 会话隐式创建

UserA 第一次给 UserB 发消息：
1. 检查 ConvID 是否存在
2. 不存在 → 创建会话 + 添加双方成员
3. 写入消息
4. 按常规路由推送

---

## 3. 群聊消息路由

### 群聊数据结构

| 字段 | 类型 | 说明 |
|------|------|------|
| ConvID | string | 会话 ID |
| Name | string | 群名称 |
| OwnerID | string | 群主 |
| MemberCount | int | 成员数量 |
| MaxMembers | int | 人数上限 |
| JoinPermission | enum | 自由 / 审核 / 禁止 |

### 群成员

| 字段 | 类型 | 说明 |
|------|------|------|
| UserID | string | 用户 ID |
| Role | enum | Member / Admin / Owner |
| Nickname | string | 群内昵称 |
| Mute | bool | 是否禁言 |
| JoinedAt | int64 | 加入时间 |

### 路由流程

```mermaid
sequenceDiagram
    participant A as UserA
    participant S as 服务端
    participant B as UserB
    participant C as UserC
    participant D as UserD

    A->>S: 发送群消息
    S->>S: 校验 A 是群成员
    S->>S: 查询所有成员（排除 A）
    S->>B: 推送
    S->>C: 推送
    Note over D: 离线
    S->>S: 写入离线消息队列
    S-->>A: Ack
```

### 写扩散 vs 读扩散

| 特性 | 写扩散 | 读扩散 |
|------|--------|--------|
| 写入方式 | 每人一条 | 只写一条 |
| 读取方式 | 读收件箱 | 读群消息 + 过滤 |
| 适用规模 | ≤500 人 | >500 人 |
| 写入开销 | O(N) | O(1) |
| 读取开销 | O(1) | O(logN) |

推荐策略：混合模式。小群写扩散，大群读扩散。

> **Phase 1 实现：统一使用读扩散。** 所有消息写入 `messages` 表（按 conv_id 存储），各终端通过 conv_seq 增量同步。写扩散作为 Phase 2 优化项，在群人数超过阈值时切换。

### 群聊功能

```mermaid
flowchart LR
    A[加群] --> B[搜索群 → 申请 → 审核/直接加入]
    C[退群] --> D[移除成员 → 系统消息通知]
    E[踢人] --> F[仅管理员 → 强制移除 → 系统通知]
    G[@提及] --> H[@某人 / @all → 特殊通知]
```

### 系统消息类型

| 类型 | 说明 |
|------|------|
| 成员加入 | 新人入群通知 |
| 成员退出 | 主动退群通知 |
| 成员被踢 | 管理员踢人通知 |
| 群创建 | 群创建成功通知 |
| 群名变更 | 群名称修改通知 |
| 管理员变更 | 添加/移除管理员通知 |

系统消息与普通消息共用同一 channel，`ContentType = System`，客户端做特殊渲染。

---

## 4. 消息推送实现

```mermaid
flowchart LR
    A[Push Queue] --> B[Worker 1]
    A --> C[Worker 2]
    A --> D[Worker 3]
    B --> E[Gateway → WebSocket → 客户端]
    C --> E
    D --> E
```

- 每个推送任务包含 `(SessionID, Message)`
- 多个 Worker 并发消费
- Worker 通过 SessionID 查找对应 WebSocket 连接并写入
