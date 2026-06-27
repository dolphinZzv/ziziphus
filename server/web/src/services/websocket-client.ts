import { getItem, getDeviceId, safeUUID } from '@/lib/storage'
import { MessageType, type WSFrame, type MsgPushPayload } from '@/types/ws'
import type {
  MsgSendPayload, MsgSendAckPayload, SyncReqPayload, SyncResPayload,
  MsgReadNotifyPayload, SessionEventPayload, SessionRecoverPayload,
  SessionRecoverAckPayload, TypingPayload, ErrorPayload,
} from '@/types/ws'

type MessageHandler = (payload: unknown, frame: WSFrame) => void

export type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'recovering'

export class WebSocketClient {
  private ws: WebSocket | null = null
  private handlers = new Map<MessageType, Set<MessageHandler>>()
  private ackContinuations = new Map<string, {
    resolve: (frame: WSFrame) => void
    reject: (err: Error) => void
    timer: ReturnType<typeof setTimeout>
  }>()
  private sendQueue: WSFrame[] = []
  private pingTimer: ReturnType<typeof setInterval> | null = null
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectDelay = 1000
  private maxReconnectDelay = 30000
  private shouldReconnect = false
  private token = ''
  private sessionId: string | null = null
  private statusListeners = new Set<(s: ConnectionStatus) => void>()

  connectionStatus: ConnectionStatus = 'disconnected'

  private setStatus(s: ConnectionStatus) {
    this.connectionStatus = s
    this.statusListeners.forEach(fn => fn(s))
  }

  onStatusChange(fn: (s: ConnectionStatus) => void) {
    this.statusListeners.add(fn)
    return () => { this.statusListeners.delete(fn) }
  }

  connect(token: string) {
    this.token = token
    this.shouldReconnect = true
    this.sessionId = getItem<string>('session_id', null) ?? null
    this.doConnect()
  }

  private doConnect() {
    if (this.ws) {
      this.ws.onclose = null
      this.ws.close()
    }

    this.setStatus('connecting')
    const baseUrl = getItem<string>('server_url', '')
    // 没有自定义地址时走相对路径，由 Vite proxy 转发到远程服务器
    const wsBase = baseUrl
      ? baseUrl.replace('http://', 'ws://').replace('https://', 'wss://')
      : `ws://${window.location.host}`
    const deviceId = getDeviceId()
    const url = `${wsBase}/ws?token=${encodeURIComponent(this.token)}&platform=web&device_id=${encodeURIComponent(deviceId)}`

    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      this.setStatus('connected')
      this.reconnectDelay = 1000
      this.startPing()
      this.flushSendQueue()
    }

    this.ws.onmessage = (event) => {
      try {
        const raw = JSON.parse(event.data)
        // Check for raw type-only frames (without id field on server)
        const frame: WSFrame = {
          type: raw.type,
          id: raw.id || '',
          payload: raw.payload,
        }
        this.handleFrame(frame)
      } catch { /* parse error */ }
    }

    this.ws.onclose = () => {
      this.setStatus('disconnected')
      this.stopPing()
      this.ws = null
      if (this.shouldReconnect) {
        this.scheduleReconnect()
      }
    }

    this.ws.onerror = () => { /* onclose will fire after */ }
  }

  private scheduleReconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.reconnectTimer = setTimeout(() => {
      if (this.shouldReconnect) this.doConnect()
    }, this.reconnectDelay)
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay)
  }

  disconnect() {
    this.shouldReconnect = false
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.stopPing()
    this.ws?.close()
    this.ws = null
    this.setStatus('disconnected')
  }

  private startPing() {
    this.pingTimer = setInterval(() => {
      this.sendRaw({ type: MessageType.Ping, id: safeUUID(), payload: {} })
    }, 30000)
  }

  private stopPing() {
    if (this.pingTimer) { clearInterval(this.pingTimer); this.pingTimer = null }
  }

  private flushSendQueue() {
    while (this.sendQueue.length > 0) {
      const frame = this.sendQueue.shift()!
      this.ws?.send(JSON.stringify({ type: frame.type, id: frame.id, payload: frame.payload }))
    }
  }

  private sendRaw(frame: { type: MessageType; id: string; payload: unknown }) {
    const json = JSON.stringify({ type: frame.type, id: frame.id, payload: frame.payload })
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(json)
    } else {
      this.sendQueue.push({ type: frame.type, id: frame.id, payload: frame.payload } as WSFrame)
    }
  }

  send(frame: WSFrame): string {
    const id = frame.id || safeUUID()
    this.sendRaw({ type: frame.type, id, payload: frame.payload })
    return id
  }

  sendWithAck(frame: WSFrame, timeout = 5000): Promise<WSFrame> {
    return new Promise((resolve, reject) => {
      const id = frame.id || safeUUID()
      const timer = setTimeout(() => {
        this.ackContinuations.delete(id)
        reject(new Error('Request timeout'))
      }, timeout)
      this.ackContinuations.set(id, { resolve, reject, timer })
      this.sendRaw({ type: frame.type, id, payload: frame.payload })
    })
  }

  on(type: MessageType, handler: MessageHandler) {
    if (!this.handlers.has(type)) this.handlers.set(type, new Set())
    this.handlers.get(type)!.add(handler)
    return () => { this.handlers.get(type)?.delete(handler) }
  }

  off(type: MessageType, handler: MessageHandler) {
    this.handlers.get(type)?.delete(handler)
  }

  getSessionId(): string | null {
    return this.sessionId
  }

  private handleFrame(frame: WSFrame) {
    // Resolve ACK
    if (frame.id) {
      const ack = this.ackContinuations.get(frame.id)
      if (ack) {
        clearTimeout(ack.timer)
        this.ackContinuations.delete(frame.id)
        ack.resolve(frame)
      }
    }

    // Route to handlers
    const typeHandlers = this.handlers.get(frame.type)
    if (typeHandlers) {
      typeHandlers.forEach(h => h(frame.payload, frame))
    }

    // Specific type handling
    switch (frame.type) {
      case MessageType.SessionRecoverAck: {
        const payload = frame.payload as SessionRecoverAckPayload
        this.sessionId = payload.session_id
        this.setStatus('connected')
        break
      }
      case MessageType.Pong:
        // Ignore, keep-alive
        break
      case MessageType.Error: {
        const payload = frame.payload as ErrorPayload
        if (payload.code === 4001) {
          // Kicked
          this.disconnect()
        }
        break
      }
    }
  }
}

export const wsClient = new WebSocketClient()
