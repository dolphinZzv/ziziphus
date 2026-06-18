"""
Agent Timeline 集成测试

读取 .env 中的配置, 向指定会话发送 agent timeline 消息流:
  1. 创建消息气泡 (thinking entry, status=running)
  2. 追加 toolCall entry
  3. 追加 toolResult entry
  4. 追加 response entry, status=completed

验证: 每步都收到 MsgSendAck, 且 msgID 正确传递。
"""

import asyncio
import json
import os
import sys
import time
import uuid

import httpx
import websockets


def load_env(path=None):
    """Minimal .env loader, no external dependency."""
    if path is None:
        path = os.path.join(os.path.dirname(__file__), ".env")
    if not os.path.exists(path):
        return
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, val = line.partition("=")
            key = key.strip()
            val = val.strip().strip('"').strip("'")
            if key not in os.environ:
                os.environ[key] = val


load_env()

SERVER = os.environ.get("SERVER", "http://127.0.0.1:8080")
WS_URL = os.environ.get("WS_URL", "ws://127.0.0.1:8080")
ACCOUNT = os.environ["ACCOUNT"]
PASSWORD = os.environ["PASSWORD"]
CONV_NAME = os.environ.get("CONV_NAME", "Dolphin")

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


def make_entry(entry_type, content, tool_name=None, tool_input=None, status=None):
    return {
        "id": str(uuid.uuid4()),
        "type": entry_type,
        "content": content,
        "toolName": tool_name,
        "toolInput": tool_input,
        "status": status,
        "timestamp": int(time.time() * 1000),
    }


def make_agent_body(title, entries, status, parent_msg_id=0):
    body = {
        "title": title,
        "entries": entries,
        "status": status,
        "parentMsgID": parent_msg_id,
    }
    return body


async def main():
    print(f"Agent Timeline Integration Test")
    print(f"  Server: {SERVER}")
    print(f"  Account: {ACCOUNT}")
    print(f"  Target: {CONV_NAME}\n")

    # ── 1. Login ──────────────────────────────────────────────
    print("[1] Login")
    async with httpx.AsyncClient(base_url=SERVER, timeout=30) as http:
        resp = await http.post("/api/v1/users/login", json={
            "account": ACCOUNT,
            "password": PASSWORD,
        })
        data = resp.json()
        if data.get("code") != 0:
            ng(f"Login failed: {data.get('message', data)}")
            return 1
        token = data["data"]["token"]
        user_id = data["data"]["user_id"]
        ok(f"Logged in as {ACCOUNT} (user_id={user_id})")

        # ── 2. Find Dolphin conversation ──────────────────────
        print("\n[2] Find Dolphin conversation")
        http.headers["Authorization"] = f"Bearer {token}"

        resp = await http.get("/api/v1/conversations")
        conv_data = resp.json()
        if conv_data.get("code") != 0:
            ng(f"List conversations failed: {conv_data}")
            return 1

        items = conv_data.get("data", {}).get("items", [])
        dolphin_conv = None
        for c in items:
            if c.get("name") == CONV_NAME:
                dolphin_conv = c
                break

        if not dolphin_conv:
            ng(f"Conversation '{CONV_NAME}' not found. Available: {[c['name'] for c in items]}")
            return 1

        conv_id = dolphin_conv["conv_id"]
        ok(f"Found: {conv_id} ({dolphin_conv.get('name')})")

    # ── 3. Connect WebSocket ─────────────────────────────────
    print("\n[3] Connect WebSocket")
    ws_url = f"{WS_URL}/ws?token={token}"
    client_seq = 0

    async def next_seq():
        nonlocal client_seq
        client_seq += 1
        return client_seq

    async def send_frame(ws, payload, frame_id=None):
        seq = await next_seq()
        frame = {
            "type": 1,  # MsgSend
            "id": frame_id or f"msg_{seq}",
            "payload": {
                "conv_id": conv_id,
                "content_type": 9,
                "body": json.dumps(payload, ensure_ascii=False),
                "client_seq": seq,
                "reply_to": 0,
                "mention": [],
            },
        }
        await ws.send(json.dumps(frame, ensure_ascii=False))

    async def recv_ack(ws, timeout=10):
        deadline = time.time() + timeout
        while time.time() < deadline:
            try:
                raw = await asyncio.wait_for(ws.recv(), timeout=min(2, max(0.1, deadline - time.time())))
                frame = json.loads(raw)
                if frame.get("type") == 2:  # MsgSendAck
                    payload = frame["payload"]
                    if isinstance(payload, str):
                        payload = json.loads(payload)
                    return payload.get("msg_id")
            except asyncio.TimeoutError:
                continue
            except Exception:
                continue
        return None

    async with websockets.connect(ws_url) as ws:
        # Drain welcome frames (session online, etc.)
        await asyncio.sleep(0.5)
        while True:
            try:
                raw = await asyncio.wait_for(ws.recv(), timeout=1)
            except asyncio.TimeoutError:
                break
        ok("WebSocket connected, ready")

        # ── 4. Send agent timeline messages ──────────────────

        print("\n[4] Send agent timeline messages")

        # Step 1: Create bubble with thinking entry
        print("\n  Step 1: Create bubble (thinking)")
        body1 = make_agent_body(
            title="Researching user query...",
            entries=[make_entry("thinking", "Let me analyze this question and search for relevant information.")],
            status="running",
        )
        await send_frame(ws, body1)
        msg_id_1 = await recv_ack(ws)
        if msg_id_1:
            ok(f"Created bubble, msgID={msg_id_1}")
        else:
            ng("Failed to get ack for step 1")
            return 1

        await asyncio.sleep(0.5)

        # Step 2: Append toolCall entry
        print("\n  Step 2: Append toolCall")
        body2 = make_agent_body(
            title=None,
            entries=[make_entry(
                "toolCall",
                "Calling search_web to find latest information",
                tool_name="search_web",
                tool_input=json.dumps({"query": "Swift 6 concurrency", "max_results": 5}),
                status="running",
            )],
            status="running",
            parent_msg_id=msg_id_1,
        )
        await send_frame(ws, body2)
        msg_id_2 = await recv_ack(ws)
        if msg_id_2:
            ok(f"Appended toolCall, msgID={msg_id_2}")
        else:
            ng("Failed to get ack for step 2")

        await asyncio.sleep(0.5)

        # Step 3: Append toolResult entry
        print("\n  Step 3: Append toolResult")
        body3 = make_agent_body(
            title=None,
            entries=[make_entry(
                "toolResult",
                "Found 5 results. Top result: Swift 6 introduces full data-race safety with Strict Concurrency Checking enabled by default.",
                tool_name="search_web",
                status="success",
            )],
            status="running",
            parent_msg_id=msg_id_1,
        )
        await send_frame(ws, body3)
        msg_id_3 = await recv_ack(ws)
        if msg_id_3:
            ok(f"Appended toolResult, msgID={msg_id_3}")
        else:
            ng("Failed to get ack for step 3")

        await asyncio.sleep(0.5)

        # Step 4: Append response + complete
        print("\n  Step 4: Append response + complete")
        body4 = make_agent_body(
            title=None,
            entries=[make_entry(
                "response",
                "Based on my research, Swift 6 introduces full data-race safety. "
                "The compiler now performs strict concurrency checking by default, "
                "which helps prevent data races at compile time. Key features include:\n\n"
                "1. **Strict Concurrency Checking** - Enabled by default\n"
                "2. **Sendable Protocol** - Marks types as safe to share across concurrency domains\n"
                "3. **Actor Isolation** - Ensures safe access to actor state\n\n"
                "This is a major step forward for safe concurrent programming in Swift.",
            )],
            status="completed",
            parent_msg_id=msg_id_1,
        )
        await send_frame(ws, body4)
        msg_id_4 = await recv_ack(ws)
        if msg_id_4:
            ok(f"Appended response, msgID={msg_id_4}")
        else:
            ng("Failed to get ack for step 4")

    # ── 5. Summary ───────────────────────────────────────────
    print(f"\n{'='*50}")
    print(f"Results: {PASS} passed, {FAIL} failed")
    if FAIL == 0:
        print(f"Root msgID for follow-up appends: {msg_id_1}")
        print("ALL CHECKS PASSED")
        return 0
    else:
        print("SOME CHECKS FAILED")
        return 1


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
