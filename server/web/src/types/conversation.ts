import type { UserType, WakeMode } from './user'

export enum ConvType {
  P2P = 1,
  Group = 2,
  System = 3,
}

export enum ConvRole {
  Member = 0,
  Admin = 1,
  Owner = 2,
}

export enum JoinRequestStatus {
  Pending = 0,
  Approved = 1,
  Rejected = 2,
}

export interface Conversation {
  conv_id: string
  type: ConvType
  name: string
  owner_id: string
  avatar?: string
  cover?: string
  primary_color?: string
  max_members?: number
  last_msg_id?: number
  last_msg_at?: number
  created_at: number
}

export interface ConvMember {
  conv_id: string
  user_id: string
  role: ConvRole
  nickname?: string
  mute: boolean
  joined_at: number
  user_type: UserType
  wake_mode: WakeMode
}

export interface JoinRequest {
  conv_id: string
  user_id: string
  status: JoinRequestStatus
  created_at: number
  updated_at: number
}

// Client-side conversation list item (matches server's ConvListItem)
export interface LastMessageInfo {
  msg_id: number
  sender_id: string
  sender_name: string
  body: string
  content_type: number
  timestamp: number
  status: number
}

export interface ConvListItem {
  conv_id: string
  type: ConvType
  name: string
  avatar: string
  unread_count: number
  last_message?: LastMessageInfo
  last_msg_at: number
  role: number
  mute: boolean
  mention_me: boolean
  partner_type: number
  pinned: boolean
}

export interface ConversationDetail extends Conversation {
  members: ConvMember[]
  notice?: string
  share_token?: string
}
