export enum SessionStatus {
  Active = 0,
  Inactive = 1,
  Expired = 2,
}

export enum DeviceType {
  Phone = 0,
  Desktop = 1,
  Web = 2,
  Tablet = 3,
}

export interface Session {
  session_id: string
  user_id: string
  device: DeviceType
  device_name: string
  device_id?: string
  client_ip?: string
  conn_id?: string
  status: SessionStatus
  login_at: number
  last_active: number
  metadata?: Record<string, unknown>
}
