import { api } from './api-client'
import type { ConvWebhook, WebhookAuditLog, WebhookMessage } from '@/types/webhook'

export const webhookService = {
  list(convId: string) {
    return api.request<ConvWebhook[]>(`/api/v1/conversations/${convId}/webhooks`)
  },

  create(convId: string, data: {
    name: string
    callback_url?: string
    headers?: { key: string; value: string }[]
    cidr_whitelist?: string[]
    require_audit?: boolean
  }) {
    return api.request<ConvWebhook & { token: string; api_key: string }>(
      `/api/v1/conversations/${convId}/webhooks`,
      { method: 'POST', body: data }
    )
  },

  update(convId: string, id: number, data: {
    name?: string
    callback_url?: string
    headers?: { key: string; value: string }[]
    cidr_whitelist?: string[]
    require_audit?: boolean
  }) {
    return api.request<{ status: string }>(
      `/api/v1/conversations/${convId}/webhooks/${id}`,
      { method: 'PUT', body: data }
    )
  },

  delete(convId: string, id: number) {
    return api.request<{ status: string }>(
      `/api/v1/conversations/${convId}/webhooks/${id}`,
      { method: 'DELETE' }
    )
  },

  regenerateKey(convId: string, id: number) {
    return api.request<{ api_key: string }>(
      `/api/v1/conversations/${convId}/webhooks/${id}/regenerate-key`,
      { method: 'POST' }
    )
  },

  logs(convId: string, id: number, page = 1, size = 20) {
    return api.request<{ items: WebhookAuditLog[]; total: number }>(
      `/api/v1/conversations/${convId}/webhooks/${id}/logs`,
      { query: { page: String(page), size: String(size) } }
    )
  },

  pendingMessages(convId: string) {
    return api.request<WebhookMessage[]>(
      `/api/v1/conversations/${convId}/webhooks/pending`
    )
  },

  approveMessage(msgId: number, reason?: string) {
    return api.request<{ status: string }>(
      `/api/v1/webhooks/messages/${msgId}/approve`,
      { method: 'POST', body: reason ? { reason } : {} }
    )
  },

  rejectMessage(msgId: number, reason?: string) {
    return api.request<{ status: string }>(
      `/api/v1/webhooks/messages/${msgId}/reject`,
      { method: 'POST', body: reason ? { reason } : {} }
    )
  },
}
