"""
Dolphin IM Bot — Python 示例

用法:
    python bot.py --account my_bot --password bot_password [--server http://localhost:8080]

支持的命令:
    - /help       显示帮助
    - /ping       回复 pong
    - /time       回复当前时间
    - /echo <msg> 复读消息
"""

import asyncio
import json
import argparse
import time
import uuid
from datetime import datetime

import httpx
import websockets


class Bot:
    def __init__(self, server: str, account: str, password: str):
        self.server = server.rstrip("/")
        self.ws_url = self.server.replace("http://", "ws://").replace("https://", "wss://")
        self.account = account
        self.password = password
        self.token = ""
        self.user_id = ""
        self.client_seq = 0
        self.ws = None
        self._running = True

    # ── HTTP ────────────────────────────────────────────────

    def _http(self) -> httpx.Client:
        headers = {}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        return httpx.Client(base_url=self.server, headers=headers, timeout=10)

    def register(self) -> bool:
        """注册 Bot 账号（首次使用）"""
        try:
            with self._http() as c:
                r = c.post("/api/v1/users/register", json={
                    "account": self.account,
                    "password": self.password,
                    "name": self.account,
                })
                if r.status_code == 200:
                    data = r.json()
                    self.token = data["token"]
                    self.user_id = data["user_id"]
                    print(f"[注册成功] user_id={self.user_id}")
                    return True
                elif r.status_code == 409:
                    print("[注册跳过] 账号已存在")
                    return True
                print(f"[注册失败] {r.text}")
                return False
        except Exception as e:
            print(f"[注册异常] {e}")
            return False

    def login(self) -> bool:
        """登录获取 token"""
        try:
            with self._http() as c:
                r = c.post("/api/v1/users/login", json={
                    "account": self.account,
                    "password": self.password,
                })
                if r.status_code == 200:
                    data = r.json()
                    self.token = data["token"]
                    self.user_id = data["user_id"]
                    print(f"[登录成功] user_id={self.user_id}")
                    return True
                print(f"[登录失败] {r.text}")
                return False
        except Exception as e:
            print(f"[登录异常] {e}")
            return False

    # ── WebSocket ──────────────────────────────────────────

    async def connect(self):
        uri = f"{self.ws_url}/ws?token={self.token}"
        self.ws = await websockets.connect(uri)
        print("[WS 已连接]")

    async def heartbeat(self):
        """每 55 秒发送一次心跳"""
        while self._running:
            await asyncio.sleep(55)
            try:
                await self.ws.send(json.dumps({
                    "type": 61, "id": f"ping_{int(time.time())}", "payload": {}
                }))
            except Exception:
                break

    async def send_message(self, conv_id: str, body: str, reply_to: int = 0):
        """发送消息到指定会话"""
        self.client_seq += 1
        await self.ws.send(json.dumps({
            "type": 1,
            "id": str(uuid.uuid4()),
            "payload": {
                "conv_id": conv_id,
                "content_type": 1,
                "body": body,
                "client_seq": self.client_seq,
                "reply_to": reply_to,
                "mention": [],
            }
        }))

    def handle_command(self, text: str, conv_id: str, reply_to: int) -> str | None:
        """处理命令消息，返回回复内容"""
        text = text.strip()
        if text == "/help":
            return (
                "支持的命令:\n"
                "  /help   - 显示帮助\n"
                "  /ping   - 回复 pong\n"
                "  /time   - 当前时间\n"
                "  /echo   - 复读"
            )
        if text == "/ping":
            return "pong"
        if text == "/time":
            return datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        if text.startswith("/echo "):
            return text[6:]
        return None  # 不回复

    async def handle_push(self, payload: dict):
        """处理收到的消息推送"""
        conv_id = payload["conv_id"]
        sender_id = payload["sender_id"]
        body = payload.get("body", "")
        msg_id = payload.get("msg_id", 0)

        # 不回复自己的消息
        if sender_id == self.user_id:
            return

        print(f"[消息] from={sender_id} body={body[:60]}")

        # 检查是否是命令
        reply = self.handle_command(body, conv_id, msg_id)
        if reply is not None:
            await self.send_message(conv_id, reply, msg_id)

    async def run(self):
        """主循环: 连接 → 心跳 → 处理消息"""
        # 首次运行尝试注册，然后登录
        if not self.login():
            if not self.register():
                print("[错误] 登录和注册都失败")
                return
            if not self.login():
                return

        while self._running:
            try:
                await self.connect()
                # 启动心跳
                hb = asyncio.create_task(self.heartbeat())

                async for raw in self.ws:
                    frame = json.loads(raw)
                    t = frame.get("type")

                    if t == 11:  # MsgPush
                        await self.handle_push(frame.get("payload", {}))
                    elif t == 2:  # MsgSendAck
                        pass  # 发送确认，可以忽略
                    elif t == 41:  # SessionOnline
                        uid = frame.get("payload", {}).get("user_id", "")
                        print(f"[上线] {uid}")
                    elif t == 42:  # SessionOffline
                        uid = frame.get("payload", {}).get("user_id", "")
                        print(f"[下线] {uid}")

                # 连接断开，清理
                hb.cancel()
                print("[WS 断开] 5 秒后重连...")
                await asyncio.sleep(5)

            except websockets.ConnectionClosed:
                print("[WS 断开] 5 秒后重连...")
                await asyncio.sleep(5)
            except Exception as e:
                print(f"[异常] {e} 10 秒后重连...")
                await asyncio.sleep(10)


def main():
    parser = argparse.ArgumentParser(description="Dolphin IM Bot")
    parser.add_argument("--server", default="http://localhost:8080")
    parser.add_argument("--account", default="example_bot")
    parser.add_argument("--password", default="bot123")
    args = parser.parse_args()

    bot = Bot(server=args.server, account=args.account, password=args.password)
    asyncio.run(bot.run())


if __name__ == "__main__":
    main()
