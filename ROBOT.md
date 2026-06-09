# IM Bot 接入文档

## 架构概述

Bot 以 **WebSocket 客户端** 的形式接入系统，与普通用户没有区别。

```
Bot (Go/Python/Node.js)                    Server
       │                                      │
       │  POST /api/v1/users/register         │
       │  (注册 Bot 账号, 获取 JWT token)      │
       │                                      │
       │  POST /api/v1/users/login            │
       │  (获取 JWT token)                    │
       │                                      │
       │  WebSocket /ws?token=<JWT>           │
       │════════════════════════════════════> │
       │                                      │
       │  Frame{Type: MsgSend, Payload: ...}  │
       │════════════════════════════════════> │
       │                                      │
       │  Frame{Type: MsgPush, Payload: ...}  │
       │ <════════════════════════════════════ │
       │                                      │
```

**核心原则：**
- Bot 是一个普通用户，有 `userID`、`name`、`token`
- Bot 通过 WebSocket 接收实时消息推送
- Bot 通过 WebSocket 发送消息，通过 HTTP API 管理群组和联系人
- Bot 可以加入群组或与用户 P2P 通信

---

## 1. 注册 Bot 账号

```bash
curl -X POST http://<server>/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "account": "my_bot",
    "password": "bot_password",
    "name": "MyBot"
  }'
```

响应：
```json
{
  "user_id": "u_xxxxxxxxxxxxx",
  "token": "eyJhbGciOiJSUzI1NiIs..."
}
```

> Bot 只需注册一次，保存 `user_id` 和 `token` 用于后续连接。

---

## 2. 登录获取 Token

如果已有账号需要重新登录：

```bash
curl -X POST http://<server>/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "account": "my_bot",
    "password": "bot_password"
  }'
```

响应：
```json
{
  "user_id": "u_xxxxxxxxxxxxx",
  "token": "eyJhbGciOiJSUzI1NiIs..."
}
```

---

## 3. WebSocket 连接

### 3.1 建立连接

```
ws://<server>/ws?token=<JWT_TOKEN>
```

连接成功后，Server 会推送 `SessionOnline` 事件。

### 3.2 心跳

客户端每隔 **60 秒** 发送一次 Ping，Server 回复 Pong。

```json
// Send:
{"type": 61, "id": "ping_001", "payload": {}}

// Receive:
{"type": 62, "id": "ping_001", "payload": {}}
```

超过 120 秒未收到心跳，Server 会断开连接。

### 3.3 消息协议

所有消息使用 JSON 格式的 Frame 封装：

```json
{
  "type": <int>,        // 消息类型
  "id": "<string>",     // 消息唯一 ID（去重用，每次发送不同）
  "payload": { ... }    // 消息体
}
```

**消息类型常量：**

| 类型 | 值 | 方向 | 说明 |
|------|-----|------|------|
| MsgSend | 1 | Bot → Server | 发送消息 |
| MsgSendAck | 2 | Server → Bot | 发送确认 |
| MsgPush | 11 | Server → Bot | 收到新消息 |
| MsgReadNotify | 32 | Server → Bot | 对方已读 |
| SessionOnline | 41 | Server → Bot | 用户上线 |
| SessionOffline | 42 | Server → Bot | 用户下线 |
| SyncReq | 21 | Bot → Server | 请求历史消息 |
| SyncRes | 22 | Server → Bot | 历史消息响应 |
| Typing | 51 | Bot → Server | 正在输入 |
| Ping | 61 | Bot → Server | 心跳请求 |
| Pong | 62 | Server → Bot | 心跳响应 |
| Error | 71 | Server → Bot | 错误信息 |

---

## 4. 发送消息

Bot 向 WebSocket 写入 Frame：

```json
{
  "type": 1,
  "id": "<client_unique_id>",
  "payload": {
    "conv_id": "<conv_id>",
    "content_type": 1,
    "body": "Hello from Bot!",
    "client_seq": <递增整数>,
    "reply_to": 0,
    "mention": []
  }
}
```

参数说明：

| 字段 | 类型 | 说明 |
|------|------|------|
| conv_id | string | 会话 ID（P2P 或群组） |
| content_type | int | 1=文本, 2=图片, 3=文件 |
| body | string | 消息内容 |
| client_seq | int64 | 客户端序列号（单调递增，用于去重和排序） |
| reply_to | int64 | 回复的消息 ID，0 表示不回复 |
| mention | []string | @ 提及的用户 ID 列表 |

**响应（MsgSendAck）：**

```json
{
  "type": 2,
  "id": "<same_as_request_id>",
  "payload": {
    "msg_id": 123456789,
    "timestamp": 1712345678000,
    "client_seq": 1,
    "status": 0
  }
}
```

### 关于 conv_id

- **P2P 会话**：`conv_id` 格式为 `sort(userA_id,userB_id)` 用 `:` 拼接。例如 Bot 和用户 `u_abc` 的 P2P 会话 ID 为 `u_abc:u_bot`（两个 ID 字典序排序后拼接）。
- **群组会话**：`conv_id` 格式为 `group_<snowflake_id>`，需要通过 HTTP API 创建或获取。

**获取或创建 P2P 会话：**

```bash
curl -X POST http://<server>/api/v1/conversations/p2p \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "u_xxxxxxxxxxxxx"}'
```

---

## 5. 接收消息

Bot 会从 WebSocket 收到 `MsgPush`（type=11）：

```json
{
  "type": 11,
  "id": "",
  "payload": {
    "msg_id": 123456789,
    "conv_id": "conv_xxx",
    "sender_id": "u_xxxxxxxx",
    "content_type": 1,
    "body": "Hello Bot!",
    "reply_to": 0,
    "mention": [],
    "timestamp": 1712345678000,
    "conv_seq": 42
  }
}
```

---

## 6. HTTP API

Bot 可以使用 HTTP API 执行管理操作（所有请求需要 `Authorization: Bearer <token>` 头）。

### 6.1 群组管理

**创建群组：**
```bash
curl -X POST http://<server>/api/v1/conversations/group \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Bot Group",
    "member_ids": ["u_user1", "u_user2"]
  }'
```

**添加成员：**
```bash
curl -X POST http://<server>/api/v1/conversations/<conv_id>/members \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"user_ids": ["u_user3"]}'
```

**移除成员：**
```bash
curl -X DELETE http://<server>/api/v1/conversations/<conv_id>/members/<user_id> \
  -H "Authorization: Bearer <token>"
```

**退出群组：**
```bash
curl -X POST http://<server>/api/v1/conversations/<conv_id>/leave \
  -H "Authorization: Bearer <token>"
```

### 6.2 获取历史消息

```bash
curl -X GET "http://<server>/api/v1/conversations/<conv_id>/messages?limit=50&before_seq=<seq>" \
  -H "Authorization: Bearer <token>"
```

### 6.3 获取会话列表

```bash
curl -X GET http://<server>/api/v1/conversations \
  -H "Authorization: Bearer <token>"
```

### 6.4 用户搜索

```bash
curl -X GET "http://<server>/api/v1/users/search?q=<keyword>" \
  -H "Authorization: Bearer <token>"
```

---

## 7. 示例 Bot

### Go

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var (
	serverURL = "http://localhost:8080"
	wsURL     = "ws://localhost:8080"
	account   = "my_bot"
	password  = "bot_password"
)

type Frame struct {
	Type    int             `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type LoginRes struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}

func main() {
	// 1. 注册（首次）
	// 2. 登录获取 token
	resp, err := http.PostForm(serverURL+"/api/v1/users/login",
		nil, // simplified — use JSON body
	)
	if err != nil {
		log.Fatal("login failed:", err)
	}
	var loginRes LoginRes
	json.NewDecoder(resp.Body).Decode(&loginRes)
	resp.Body.Close()

	token := loginRes.Token

	// 3. WebSocket 连接
	c, _, err := websocket.DefaultDialer.Dial(wsURL+"/ws?token="+token, nil)
	if err != nil {
		log.Fatal("ws dial failed:", err)
	}
	defer c.Close()

	// 4. 心跳 goroutine
	go func() {
		for {
			time.Sleep(55 * time.Second)
			c.WriteJSON(Frame{Type: 61, ID: "ping", Payload: json.RawMessage("{}")})
		}
	}()

	// 5. 消息循环
	var clientSeq int64 = 0
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Fatal("read error:", err)
		}
		var frame Frame
		json.Unmarshal(msg, &frame)

		switch frame.Type {
		case 11: // MsgPush — 收到消息
			var push struct {
				ConvID   string `json:"conv_id"`
				SenderID string `json:"sender_id"`
				Body     string `json:"body"`
				MsgID    int64  `json:"msg_id"`
			}
			json.Unmarshal(frame.Payload, &push)

			log.Printf("收到消息 from %s: %s", push.SenderID, push.Body)

			// 自动回复
			clientSeq++
			reply, _ := json.Marshal(map[string]interface{}{
				"conv_id":     push.ConvID,
				"content_type": 1,
				"body":        "I received: " + push.Body,
				"client_seq":  clientSeq,
				"reply_to":    push.MsgID,
				"mention":     []string{},
			})
			c.WriteJSON(Frame{
				Type:    1,
				ID:      "reply_" + push.ConvID,
				Payload: reply,
			})
		}
	}
}
```

### Python

```python
import asyncio
import json
import websockets

async def bot():
    # 1. 登录获取 token（用 httpx 或 aiohttp）
    # ...

    token = "eyJ..."
    async with websockets.connect(f"ws://localhost:8080/ws?token={token}") as ws:
        # 心跳
        async def heartbeat():
            while True:
                await asyncio.sleep(55)
                await ws.send(json.dumps({"type": 61, "id": "ping", "payload": {}}))
        asyncio.create_task(heartbeat())

        # 消息循环
        client_seq = 0
        async for raw in ws:
            frame = json.loads(raw)
            if frame["type"] == 11:  # MsgPush
                push = frame["payload"]
                print(f"收到: {push['body']}")
                client_seq += 1
                await ws.send(json.dumps({
                    "type": 1,
                    "id": f"reply_{push['conv_id']}",
                    "payload": {
                        "conv_id": push["conv_id"],
                        "content_type": 1,
                        "body": f"Echo: {push['body']}",
                        "client_seq": client_seq,
                        "reply_to": 0,
                        "mention": [],
                    }
                }))

asyncio.run(bot())
```

---

## 8. 部署建议

- Bot 建议使用独立的用户账号，不与真实用户共用
- Bot 的 `account` 命名建议以 `bot_` 前缀区分，例如 `bot_weather`、`bot_alert`
- 生产环境 Bot 应实现自动重连（检测 WebSocket 断开后等待 1-5 秒重新连接）
- 断线重连后需要调用 `SyncReq` 拉取离线期间的消息

### 离线同步

Bot 重连后，拉取未读消息：

```json
{
  "type": 21,
  "id": "sync_001",
  "payload": {
    "conv_id": "<conv_id>",
    "last_conv_seq": <上次收到的 conv_seq>,
    "limit": 50
  }
}
```

响应：
```json
{
  "type": 22,
  "id": "sync_001",
  "payload": {
    "conv_id": "<conv_id>",
    "messages": [...],
    "has_more": false
  }
}
```

---

## 9. 协议参考

| Frame Type | 值 | Payload 结构 |
|-----------|-----|-------------|
| MsgSend | 1 | `MsgSendPayload{conv_id, content_type, body, reply_to, client_seq, mention}` |
| MsgSendAck | 2 | `MsgSendAckPayload{msg_id, timestamp, client_seq, status}` |
| MsgPush | 11 | `MsgPushPayload{msg_id, conv_id, sender_id, content_type, body, reply_to, mention, timestamp, conv_seq}` |
| SyncReq | 21 | `SyncReqPayload{conv_id, last_conv_seq, limit}` |
| SyncRes | 22 | `SyncResPayload{conv_id, messages[], has_more}` |
| MsgReadNotify | 32 | `MsgReadNotifyPayload{conv_id, user_id, msg_id, timestamp}` |
| SessionOnline | 41 | `SessionEventPayload{user_id, session_id}` |
| SessionOffline | 42 | `SessionEventPayload{user_id, session_id}` |
| Typing | 51 | `TypingPayload{conv_id, user_id, session_id}` |
| Ping | 61 | `{}` |
| Pong | 62 | `{}` |
| Error | 71 | `ErrorPayload{code, message}` |
