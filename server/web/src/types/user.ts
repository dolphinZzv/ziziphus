// User types - match Go model exactly
export enum UserType {
  Human = 0,
  Agent = 1,
}

export enum UserStatus {
  Offline = 0,
  Online = 1,
  Busy = 2,
}

export enum WakeMode {
  All = 0,
  MentionOnly = 1,
}

export interface User {
  user_id: string
  account: string
  type: UserType
  name: string
  avatar: string
  status: UserStatus
  uid: string
  primary_color: string
  secondary_color: string
  ext_meta?: Record<string, unknown>
  wake_mode: WakeMode
  api_key: string
  created_at: number
}

export interface OnlineDevice {
  device: number
  device_name: string
  last_active: number
}
