export enum ContentType {
  Text = 0,
  Image = 1,
  File = 2,
  Audio = 3,
  Video = 4,
  System = 5,
  Recall = 6,
  Edit = 7,
  Custom = 8,
  AgentTimeline = 9,
  Form = 10,
  FormResponse = 11,
}

export enum MsgStatus {
  Sending = 0,
  Sent = 1,
  Delivered = 2,
  Read = 3,
}

export interface Message {
  msg_id: number
  conv_id: string
  sender_id: string
  sender_name: string
  sender_session_id: string
  content_type: ContentType
  body: string
  mention?: string[]
  reply_to: number
  timestamp: number
  client_seq: number
  conv_seq: number
  status: MsgStatus
  deleted?: boolean
}

// For local messages before server assigns msg_id
export interface LocalMessage extends Message {
  stableId: string
}

export function messageStableId(msg: Message): string {
  return msg.msg_id > 0 ? String(msg.msg_id) : `local-${msg.client_seq}`
}
