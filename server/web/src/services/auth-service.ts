import { api } from './api-client'
import type { User, WakeMode } from '@/types/user'

export const authService = {
  register(account: string, name: string, password: string) {
    return api.request<{ user: User; token: string; refresh_token: string }>('/api/v1/users/register', {
      method: 'POST',
      body: { account, name, password },
    })
  },

  login(account: string, password: string) {
    return api.request<{ user: User; token: string; refresh_token: string; session_id: string }>('/api/v1/users/login', {
      method: 'POST',
      body: { account, password },
    })
  },

  refresh(refreshToken: string) {
    return api.request<{ token: string; refresh_token: string }>('/api/v1/users/refresh', {
      method: 'POST',
      body: { refresh_token: refreshToken },
    })
  },

  getMe() {
    return api.request<User>('/api/v1/users/me')
  },

  updateMe(data: { name?: string; avatar?: string; primary_color?: string; secondary_color?: string }) {
    return api.request<User>('/api/v1/users/me', {
      method: 'PUT',
      body: data,
    })
  },

  // Agent management
  listAgents() {
    return api.request<User[]>('/api/v1/users/me/agents')
  },

  createAgent(data: { name: string; headline?: string; avatar?: string; cover?: string; wake_mode?: WakeMode; primary_color?: string; secondary_color?: string }) {
    return api.request<User>('/api/v1/users/me/agents', {
      method: 'POST',
      body: data,
    })
  },

  updateAgent(agentId: string, data: { name?: string; headline?: string; avatar?: string; cover?: string; wake_mode?: WakeMode; primary_color?: string; secondary_color?: string }) {
    return api.request<User>(`/api/v1/users/me/agents/${agentId}`, {
      method: 'PUT',
      body: data,
    })
  },

  deleteAgent(agentId: string) {
    return api.request<null>(`/api/v1/users/me/agents/${agentId}`, {
      method: 'DELETE',
    })
  },

  regenerateAgentKey(agentId: string) {
    return api.request<{ api_key: string }>(`/api/v1/users/me/agents/${agentId}/regenerate-key`, {
      method: 'PUT',
    })
  },
}
