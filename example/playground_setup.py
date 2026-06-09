"""
Playground 验证脚本

流程:
  1. 注册/登录 python_bot 和 go_bot
  2. 创建 Werewolf 群组, 把两个 Bot 都加进去
  3. 两个 Bot 连接 WebSocket, 各发送 "hello world"
  4. 验证: 每个 Bot 都收到了对方的消息
  5. 验证: 群组成员列表包含两个 Bot

全部通过返回 0, 任一步失败返回 1。
"""

import asyncio
import json
import sys
import uuid

import httpx
import websockets


SERVER = "http://127.0.0.1:8080"
WS_URL = "ws://127.0.0.1:8080"

BOTS = [
    {"account": "python_bot", "password": "bot123", "name": "PythonBot"},
    {"account": "go_bot",     "password": "bot123", "name": "GoBot"},
]

PASS = 0
FAIL = 0


def ok(msg):
    global PASS
    PASS += 1
    print(f"  ✓ {msg}")


def ng(msg):
    global FAIL
    FAIL += 1
    print(f"  ✗ {msg}")


# ── HTTP ──────────────────────────────────────────────────

class BotClient:
    def __init__(self, account, password, name):
        self.account = account
        self.password = password
        self.name = name
        self.token = ""
        self.user_id = ""
        self.http = httpx.Client(base_url=SERVER, timeout=10)

    def _set_auth(self):
        self.http.headers["Authorization"] = f"Bearer {self.token}"

    def _unwrap(self, r):
        """解析 {code, msg, data} 响应，返回 data 或 None"""
        if r.status_code != 200:
            return None
        body = r.json()
        if body.get("code") != 0:
            return None
        return body.get("data")

    def register(self):
        r = self.http.post("/api/v1/users/register", json={
            "account": self.account, "password": self.password, "name": self.name,
        })
        data = self._unwrap(r)
        if data:
            self.token = data["token"]
            self.user_id = data["user_id"]
            self._set_auth()
            ok(f"{self.name} 注册成功  user_id={self.user_id}")
            return True
        # 账号已存在 (code=4001)
        try:
            code = r.json().get("code")
        except Exception:
            code = None
        if code == 4001:
            print(f"  - {self.name} 账号已存在")
            return self.login()
        ng(f"{self.name} 注册失败: {r.text}")
        return False

    def login(self):
        r = self.http.post("/api/v1/users/login", json={
            "account": self.account, "password": self.password,
        })
        data = self._unwrap(r)
        if data:
            self.token = data["token"]
            self.user_id = data["user_id"]
            self._set_auth()
            ok(f"{self.name} 登录成功")
            return True
        ng(f"{self.name} 登录失败: {r.text}")
        return False

    def create_group(self, name, member_ids):
        r = self.http.post("/api/v1/conversations/group", json={
            "name": name, "member_ids": member_ids,
        })
        data = self._unwrap(r)
        if data:
            conv_id = data["conv_id"]
            ok(f"创建群组 {name}  conv_id={conv_id}")
            return conv_id
        # 可能已存在，从会话列表查找
        r2 = self.http.get("/api/v1/conversations")
        data2 = self._unwrap(r2)
        if data2:
            for conv in data2.get("items", []):
                if conv.get("name") == name and conv.get("type") == 2:
                    print(f"  - 群组已存在, 复用 conv_id={conv['conv_id']}")
                    return conv["conv_id"]
        ng(f"创建群组失败: {r.text}")
        return None

    def get_conv_detail(self, conv_id):
        r = self.http.get(f"/api/v1/conversations/{conv_id}")
        data = self._unwrap(r)
        if data:
            return data
        ng(f"获取群组详情失败: {r.text}")
        return None


# ── WebSocket 消息收发 + 验证 ────────────────────────────

async def bot_connect_and_verify(bot_info, conv_id, other_user_id):
    """Bot 连接 WS → 发送 hello world → 验证收到对方消息 → 返回结果"""
    account = bot_info["account"]
    password = bot_info["password"]
    name = bot_info["name"]

    # 登录
    async with httpx.AsyncClient(base_url=SERVER) as client:
        r = await client.post("/api/v1/users/login", json={
            "account": account, "password": password,
        })
        body = r.json()
        if body.get("code") != 0:
            return ng(f"{name} 登录失败")
        token = body["data"]["token"]

    results = {"sent_ack": False, "received_hello": False, "sender_id": ""}

    try:
        async with websockets.connect(f"{WS_URL}/ws?token={token}") as ws:
            ok(f"{name} WebSocket 已连接")
            await asyncio.sleep(0.5)

            # 发消息
            msg = {
                "type": 1, "id": str(uuid.uuid4()),
                "payload": {
                    "conv_id": conv_id, "content_type": 1, "body": "hello world",
                    "client_seq": 1, "reply_to": 0, "mention": [],
                }
            }
            await ws.send(json.dumps(msg))
            ok(f"{name} 发送: hello world")

            # 收消息 (最多等 5 秒)
            deadline = asyncio.get_event_loop().time() + 5
            while asyncio.get_event_loop().time() < deadline:
                try:
                    raw = await asyncio.wait_for(ws.recv(), timeout=1)
                    frame = json.loads(raw)
                    t = frame.get("type")

                    if t == 2:  # MsgSendAck
                        results["sent_ack"] = True

                    elif t == 11:  # MsgPush
                        p = frame["payload"]
                        sid = p.get("sender_id", "")
                        body = p.get("body", "")
                        results["received_hello"] = (body == "hello world")
                        results["sender_id"] = sid

                except asyncio.TimeoutError:
                    break

    except Exception as e:
        ng(f"{name} WebSocket 异常: {e}")
        return results

    # 验证
    if results["sent_ack"]:
        ok(f"{name} 发送确认 (MsgSendAck) ✓")
    else:
        ng(f"{name} 未收到发送确认")

    if results["received_hello"] and results["sender_id"] == other_user_id:
        ok(f"{name} 收到对方 ({results['sender_id']}) 的 hello world ✓")
    elif results["received_hello"]:
        ng(f"{name} 收到消息但发送者不符 (期望={other_user_id}, 实际={results['sender_id']})")
    else:
        ng(f"{name} 未收到对方的 hello world")

    return results


# ── 主流程 ────────────────────────────────────────────────

async def main():
    global PASS, FAIL
    print("\n==================== Dolphin Playground ====================\n")

    # ── 1. 检查 Server 是否运行 ──
    try:
        r = httpx.get(f"{SERVER}/metrics", timeout=5)
        if r.status_code != 200:
            print(f"  ✗ Server 未就绪 (HTTP {r.status_code})")
            print("  请先运行: make server")
            sys.exit(1)
        ok("Server 连接正常")
    except Exception as e:
        print(f"  ✗ Server 连接失败: {e}")
        print("  请先运行: make server")
        sys.exit(1)

    # ── 2. 注册 / 登录 Bot ──
    print("\n--- 步骤 1: Bot 账号注册/登录 ---")
    tokens = {}
    for bot in BOTS:
        c = BotClient(bot["account"], bot["password"], bot["name"])
        if not c.register():
            sys.exit(1)
        tokens[bot["name"]] = {"token": c.token, "user_id": c.user_id}

    # ── 3. 创建 Werewolf 群组 ──
    print("\n--- 步骤 2: 创建 Werewolf 群组 ---")
    admin = BotClient(BOTS[0]["account"], BOTS[0]["password"], BOTS[0]["name"])
    admin.login()
    member_ids = [t["user_id"] for t in tokens.values()]
    conv_id = admin.create_group("Werewolf", member_ids)
    if not conv_id:
        sys.exit(1)

    # ── 4. 验证群组成员 ──
    print("\n--- 步骤 3: 验证群组成员 ---")
    detail = admin.get_conv_detail(conv_id)
    if detail:
        members = detail.get("members", [])
        member_user_ids = {m["user_id"] for m in members}
        for name, info in tokens.items():
            uid = info["user_id"]
            if uid in member_user_ids:
                ok(f"{name} ({uid}) 在群组成员中")
            else:
                ng(f"{name} ({uid}) 不在群组成员中")
    if len(members or []) >= 2:
        ok(f"群组成员数 >= 2 (实际={len(members or [])})")
    else:
        ng(f"群组成员数不足 (实际={len(members or [])})")

    # ── 5. Bot 互发消息并验证 ──
    print("\n--- 步骤 4: Bot 互发 hello world ---")
    user_ids = {bot["name"]: info["user_id"] for bot, info in zip(BOTS, tokens.values())}

    results = await asyncio.gather(
        bot_connect_and_verify(BOTS[0], conv_id, user_ids["GoBot"]),
        bot_connect_and_verify(BOTS[1], conv_id, user_ids["PythonBot"]),
    )

    # ── 6. 汇总 ──
    print(f"\n==================== 结果汇总 ====================")
    total = PASS + FAIL
    if FAIL == 0:
        print(f"  通过: {PASS}/{total}")
        print("  状态: ALL PASS ✓")
        print("==================================================\n")
        sys.exit(0)
    else:
        print(f"  通过: {PASS}/{total}  失败: {FAIL}/{total}")
        print("  状态: SOME FAILED ✗")
        print("==================================================\n")
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
