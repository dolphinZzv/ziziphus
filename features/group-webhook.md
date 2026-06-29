# Group Webhook Feature

## 概述

群组会话增加 webhook 功能，让外部服务可以通过 API 向群组发消息（需 API key + CIDR 双重验证），群组内消息通过 callback URL 推送给外部服务。每个群组支持多个 webhook，@webhookname 触发指定转发。消息审核：开启后外部消息需管理员审批才推给全员。所有操作记录审计日志。

---

## 数据库

### Migration `020_conv_webhooks.sql`

```sql
-- webhook 配置
CREATE TABLE IF NOT EXISTS conv_webhooks (
    id              BIGSERIAL PRIMARY KEY,
    conv_id         VARCHAR(64) NOT NULL REFERENCES conversations(conv_id) ON DELETE CASCADE,
    name            VARCHAR(128) NOT NULL,        -- webhook 名称，用于 @name
    token           VARCHAR(64) NOT NULL UNIQUE,  -- wh_xxxxx，公开 URL 里的身份
    api_key_hash    VARCHAR(128) NOT NULL DEFAULT '',  -- bcrypt hash
    api_key_salt    VARCHAR(32) NOT NULL DEFAULT '',
    callback_url    VARCHAR(512) NOT NULL DEFAULT '',  -- 空 = 只接收不转发
    headers         JSONB DEFAULT '[]'::jsonb,         -- [{"key":"X-Custom","value":"val"}]
    cidr_whitelist  JSONB DEFAULT '[]'::jsonb,          -- ["10.0.0.0/8"]
    require_audit   BOOLEAN NOT NULL DEFAULT false,
    created_by      VARCHAR(32) NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(conv_id, name)
);

-- 审核日志（也作为转发/重试记录）
CREATE TABLE IF NOT EXISTS webhook_audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    webhook_id  BIGINT NOT NULL REFERENCES conv_webhooks(id) ON DELETE CASCADE,
    conv_id     VARCHAR(64) NOT NULL,
    msg_id      BIGINT NOT NULL,
    action      VARCHAR(16) NOT NULL,               -- 'send','approve','reject','forward','forward_fail','retry'
    actor_id    VARCHAR(32) NOT NULL DEFAULT '',
    reason      VARCHAR(256) DEFAULT '',
    caller_ip   VARCHAR(45) DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);

-- webhook 发出的消息独立关联表
CREATE TABLE IF NOT EXISTS webhook_messages (
    msg_id      BIGINT PRIMARY KEY REFERENCES messages(msg_id) ON DELETE CASCADE,
    webhook_id  BIGINT NOT NULL REFERENCES conv_webhooks(id) ON DELETE CASCADE,
    conv_id     VARCHAR(64) NOT NULL,
    audit_status VARCHAR(16) NOT NULL DEFAULT '',  -- ''=不需审核 / pending / approved / rejected
    source_ip   VARCHAR(45) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_webhook_messages_audit ON webhook_messages(audit_status, conv_id);
CREATE INDEX idx_webhook_messages_webhook ON webhook_messages(webhook_id);
```

### 消息审核状态机

```
webhook 发消息
     │
     ├─ require_audit=false ──→ audit_status='' → 直接推全员
     │
     └─ require_audit=true ───→ audit_status='pending'
               │                     → 仅推管理员
               │
               ├─ approve ───→ status='approved' → 推全员
               │
               └─ reject  ───→ status='rejected' → 不推送
```

---

## Model (`server/pkg/model/webhook.go`)

```go
type ConvWebhook struct {
    ID            int64           `json:"id"`
    ConvID        string          `json:"conv_id"`
    Name          string          `json:"name"`
    Token         string          `json:"token,omitempty"`           // 创建时一次性返回
    APIKey        string          `json:"api_key,omitempty"`         // 创建时一次性返回
    CallbackURL   string          `json:"callback_url,omitempty"`
    Headers       []WebhookHeader `json:"headers,omitempty"`
    CIDRWhitelist []string        `json:"cidr_whitelist,omitempty"`
    RequireAudit  bool            `json:"require_audit"`
    CreatedBy     string          `json:"created_by"`
    CreatedAt     int64           `json:"created_at"`
}

type WebhookHeader struct {
    Key   string `json:"key"`
    Value string `json:"value"`
}

type WebhookAuditLog struct {
    ID        int64  `json:"id"`
    WebhookID int64  `json:"webhook_id"`
    ConvID    string `json:"conv_id"`
    MsgID     int64  `json:"msg_id"`
    Action    string `json:"action"`
    ActorID   string `json:"actor_id"`
    Reason    string `json:"reason,omitempty"`
    CallerIP  string `json:"caller_ip,omitempty"`
    CreatedAt int64  `json:"created_at"`
}

type WebhookMessage struct {
    MsgID       int64  `json:"msg_id"`
    WebhookID   int64  `json:"webhook_id"`
    ConvID      string `json:"conv_id"`
    AuditStatus string `json:"audit_status"`
    SourceIP    string `json:"source_ip"`
    CreatedAt   int64  `json:"created_at"`
}
```

Token: `wh_` + 16 bytes hex. API Key: 16 bytes hex → bcrypt 哈希存储，创建时返回明文一次。

---

## DB Repo (`server/internal/storage/db/webhooks.go`)

接口 `convWebhookDB`：

```go
// webhook CRUD
Create(ctx, wh) (*ConvWebhook, error)
GetByID(ctx, id) (*ConvWebhook, error)
GetByToken(ctx, token) (*ConvWebhook, error)
ListByConvID(ctx, convID) ([]*ConvWebhook, error)
Update(ctx, wh) error
Delete(ctx, id) error
GetByConvIDAndName(ctx, convID, name) (*ConvWebhook, error)  // @mention

// API key
GetAPIKeyHash(ctx, id) (hash string, error)
UpdateAPIKeyHash(ctx, id, hash) error

// webhook_messages
InsertWebhookMessage(ctx, wm) error
GetWebhookMessage(ctx, msgID) (*WebhookMessage, error)
ListPendingAudit(ctx, convID) ([]*WebhookMessage, error)
UpdateAuditStatus(ctx, msgID, status, actorID) error
ListByWebhook(ctx, whID, page, size) ([]*WebhookMessage, int, error)

// audit logs
InsertAuditLog(ctx, log) error
ListAuditLogs(ctx, convID, page, size) ([]*WebhookAuditLog, int, error)
```

---

## API 端点

### 管理端 (authenticated, admin+ 权限)

| Method | Path | 权限 | 功能 |
|--------|------|------|------|
| GET | `/api/v1/conversations/{conv_id}/webhooks` | admin+ | 列表 |
| POST | `/api/v1/conversations/{conv_id}/webhooks` | admin+ | 创建 |
| PUT | `/api/v1/conversations/{conv_id}/webhooks/{id}` | 创建者/owner | 更新 |
| DELETE | `/api/v1/conversations/{conv_id}/webhooks/{id}` | 创建者/owner | 删除 |
| POST | `.../{id}/regenerate-key` | 创建者/owner | 重置 key |
| GET | `.../{id}/logs` | admin+ | 审计日志 |
| GET | `.../webhooks/pending` | admin+ | 待审核消息 |

权限规则: admin+ = role∈(Admin,Owner); 修改/删除 = created_by==当前用户 || role==Owner

### 公开端 (no auth, 有独立 rate limit)

| Method | Path | 功能 |
|--------|------|------|
| POST | `/api/v1/webhooks/{token}` | 外部发消息到群组 |

### 审核端 (authenticated, admin+)

| Method | Path | 功能 |
|--------|------|------|
| POST | `/api/v1/webhooks/messages/{msg_id}/approve` | 通过 |
| POST | `/api/v1/webhooks/messages/{msg_id}/reject` | 拒绝 |

---

## 公开接口处理流程 (POST /api/v1/webhooks/{token})

```
请求 → rate limit (per IP, 10/s)
  │ 超过 → 429
  │
  ├─ token → GetByToken
  │   ├─ 不存在 → 404
  │   └─ 存在 → 继续
  │
  ├─ API key: Authorization: Bearer <key> → bcrypt 比对
  │   ├─ key 为空且无 header → 跳过
  │   ├─ 匹配 → 继续
  │   └─ 不匹配 → 401
  │
  ├─ CIDR: X-Forwarded-For → RemoteAddr
  │   ├─ whitelist 为空 → 放行
  │   ├─ 任一 CIDR 包含 IP → 放行
  │   └─ 都不匹配 → 403
  │
  ├─ body: { content_type, body, reply_to }
  │
  ├─ sender_id = "webhook:{wh.ID}", senderName = wh.Name
  ├─ 写入 messages + webhook_messages (含 source_ip)
  │
  ├─ require_audit=false → 直接推全员
  ├─ require_audit=true  → audit_status='pending', 仅推管理员
  │
  └─ 返回 { msg_id, audit_status, timestamp }
```

---

## Webhook Outgoing — 消息转发 + HMAC + 重试

### @webhookname 行为

群成员发 `@监控系统 检查服务器状态` → 消息正常显示在群里 + **异步**转发到匹配 webhook 的 callback_url。

### 转发时机

在 `Ingest.Ingest()` 的 persist + route+push 之后（goroutine 异步）：

```go
go in.forwardToWebhooks(ctx, msg)
```

只处理群组消息，根据 `@name` 过滤匹配的 webhook。

### HMAC 签名

每次转发带 `X-Signature: sha256=<hmac_hex(api_key, body)>` header，外部服务可验证来源。

### 重试策略

最多 3 次，指数退避 (1s, 3s, 7s)。全部失败后写审计日志 `forward_fail`。

### 转发 payload

```json
{
  "event": "message.created",
  "webhook_id": 123,
  "conv_id": "group_xxx",
  "message": {
    "msg_id": 12345, "sender_id": "user_xxx", "sender_name": "小明",
    "sender_type": 0, "content_type": 1,
    "body": "@监控系统 检查服务器状态",
    "reply_to": 0, "timestamp": 1700000000
  }
}
```

---

## 接收端点限流

公开端点 `POST /api/v1/webhooks/{token}` 使用独立 rate limiter:
- per IP, 10/s, burst 20
- 超过返回 429

---

## Web 前端 UI (group-detail.tsx)

在会话设置区域新增 "Webhooks" 区块（仅 admin+ 可见）：

- 列表每个 webhook：name、callback URL、状态标记、操作按钮
- 添加/编辑弹窗：name、callback_url、headers key/value 列表、cidr_whitelist、require_audit toggle
- 创建后弹窗显示 token + api_key（一次性，可复制）
- 待审核消息列表 + approve/reject 按钮

---

## 文件变更清单

| 操作 | 文件 |
|------|------|
| **新增** | `migrations/020_conv_webhooks.sql` |
| **新增** | `pkg/model/webhook.go` |
| **新增** | `storage/db/webhooks.go` |
| **新增** | `api/webhook.go` |
| **新增** | `web/e2e/webhook.spec.ts` |
| **新增** | `web/src/services/webhook-service.ts` |
| **修改** | `api/router.go` — 注册路由 |
| **修改** | `cmd/panda_ai/main.go` — 接线 |
| **修改** | `internal/message/ingest.go` — 转发逻辑 |
| **修改** | `web/src/features/group/group-detail.tsx` — Webhooks UI |
| **修改** | `web/src/i18n/zh.json`, `en.json` |
| **修改** | `api/handlers_test.go` — NewFileHandler 兼容 |

---

## 测试策略

**单元测试 (mock convWebhookDB)：**
- webhook CRUD、GetByToken、token 唯一约束
- webhook_messages insert/status 流转
- CIDR 检查（匹配/不匹配/空 whitelist/非法 CIDR）
- HMAC 签名生成验证
- @name 解析（边界 case：无 @、多个 @、特殊字符）
- 接收端 rate limit 超过返回 429
- 审核流程 pending→approve→push / pending→reject
- 认证错误 key→401、无 key→401、无 token→404
- 转发流程 + @mention 过滤
- 重试 3 次失败记录 forward_fail

**E2E 测试（使用真实 API，连接测试数据库）：**
> 所有 e2e 测试使用真实的 API 调用，不做 mock。测试数据库使用独立的测试库，测试前后自动清理数据。

- 创建 webhook → POST 真实请求 → 验证返回 token + key
- 用返回的 token + key 发消息 → 验证群内出现消息
- 错误 key → 验证返回 401
- 非白名单 IP (模拟) → 验证返回 403
- 设置 require_audit=true 发消息 → admin approve → 全员收到
- admin reject → 消息不推送
- 群内发 @webhookname 消息 → 验证 callback_url 收到真实 POST
- 群内发普通消息 → 验证 callback_url 不收到

---

## 实现顺序

1. 数据库迁移 SQL
2. Model 结构体
3. DB Repo + 单元测试
4. Webhook Handler (CRUD + Receive + Audit)
5. Handler 单元测试
6. Ingest 转发改造 (forwardToWebhooks + 重试)
7. 路由注册
8. Main.go 接线
9. Web UI (group-detail.tsx)
10. Web service (webhook-service.ts)
11. i18n 键值
12. 构建 + 部署
