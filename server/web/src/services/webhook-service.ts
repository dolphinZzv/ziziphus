import { api } from './api-client'
import type { ConvWebhook } from '@/types/webhook'

export const webhookService = {
  list(convId: string) {
    return api.request<ConvWebhook[]>(`/api/v1/conversations/${convId}/webhooks`)
  },

  create(convId: string, data: {
    name: string
    callback_url?: string
    headers?: { key: string; value: string }[]
    cidr_whitelist?: string[]
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

  test(convId: string, id: number) {
    return api.request<{ status: string; msg_id: number; body: string }>(
      `/api/v1/conversations/${convId}/webhooks/${id}/test`,
      { method: 'POST' }
    )
  },
}
