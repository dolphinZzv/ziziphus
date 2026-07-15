export interface ConvWebhook {
  id: number
  conv_id: string
  name: string
  api_key?: string
  callback_url?: string
  headers?: { key: string; value: string }[]
  cidr_whitelist?: string[]
  created_by: string
  created_at: number
}
