export interface ConvWebhook {
  id: number
  conv_id: string
  name: string
  callback_url?: string
  headers?: { key: string; value: string }[]
  cidr_whitelist?: string[]
  require_audit: boolean
  created_by: string
  created_at: number
}

export interface WebhookAuditLog {
  id: number
  webhook_id: number
  conv_id: string
  msg_id: number
  action: string
  actor_id: string
  reason: string
  caller_ip: string
  created_at: number
}

export interface WebhookMessage {
  msg_id: number
  webhook_id: number
  conv_id: string
  audit_status: string
  source_ip: string
  created_at: number
}
