// WebSocket frame types - match Go protocol exactly

export enum MessageType {
  MsgSend = 1,
  MsgSendAck = 2,
  MsgPush = 11,
  MsgReceived = 12,
  MsgEdit = 13,
  MsgRecall = 14,
  SyncReq = 21,
  SyncRes = 22,
  MsgReadNotify = 32,
  SessionOnline = 41,
  SessionOffline = 42,
  SessionRecover = 43,
  SessionRecoverAck = 44,
  Typing = 51,
  Ping = 61,
  Pong = 62,
  Error = 71,
}

export interface MsgEditPayload {
  conv_id: string
  msg_id: number
  new_body: string
}

export interface MsgRecallPayload {
  conv_id: string
  msg_id: number
}

export interface MsgEditPushPayload {
  conv_id: string
  msg_id: number
  sender_id: string
  new_body: string
  edited_at: number
  timestamp: number
}

export interface MsgRecallPushPayload {
  conv_id: string
  msg_id: number
  sender_id: string
  recalled_at: number
  timestamp: number
}

export interface WSFrame {
  type: MessageType
  id: string
  payload: unknown
}

// Payloads
export interface MsgSendPayload {
  conv_id: string
  content_type: number
  body: string
  reply_to: number
  client_seq: number
  mention?: string[]
}

export interface MsgSendAckPayload {
  msg_id: number
  timestamp: number
  client_seq: number
  status: number
}

export interface MsgPushPayload {
  msg_id: number
  conv_id: string
  sender_id: string
  sender_name: string
  content_type: number
  body: string
  reply_to: number
  mention?: string[]
  timestamp: number
  conv_seq: number
}

export interface SyncReqPayload {
  conv_id: string
  last_conv_seq: number
  limit: number
}

export interface SyncMessage {
  msg_id: number
  sender_id: string
  content_type: number
  body: string
  timestamp: number
  conv_seq: number
}

export interface SyncResPayload {
  conv_id: string
  messages: SyncMessage[]
  has_more: boolean
}

export interface MsgReadNotifyPayload {
  conv_id: string
  user_id: string
  session_id: string
  msg_id: number
  timestamp: number
}

export interface SessionEventPayload {
  user_id: string
  session_id: string
  device?: number
  device_name?: string
}

export interface SessionRecoverPayload {
  session_id: string
}

export interface SessionRecoverAckPayload {
  session_id: string
  user_id: string
  timestamp: number
}

export interface TypingPayload {
  conv_id: string
  user_id: string
  session_id: string
}

export interface ErrorPayload {
  code: number
  message: string
}
