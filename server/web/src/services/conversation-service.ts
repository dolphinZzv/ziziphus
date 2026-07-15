import { api } from './api-client'
import type { ConvListItem, ConversationDetail, JoinRequest } from '@/types/conversation'
import type { PaginatedData } from '@/types/api'

export const conversationService = {
  list(page = 1, size = 50) {
    return api.request<PaginatedData<ConvListItem>>('/api/v1/conversations', {
      query: { page, size },
    })
  },

  getDetail(convId: string) {
    return api.request<ConversationDetail>(`/api/v1/conversations/${convId}`)
  },

  createP2P(userId: string) {
    return api.request<{ conv_id: string }>('/api/v1/conversations/p2p', {
      method: 'POST',
      body: { user_id: userId },
    })
  },

  createGroup(name: string, headline: string, memberIds: string[], avatar?: string) {
    return api.request<{ conv_id: string }>('/api/v1/conversations/group', {
      method: 'POST',
      body: { name, headline, member_ids: memberIds, avatar },
    })
  },

  updateGroup(convId: string, data: { name?: string; avatar?: string; notice?: string; cover?: string; headline?: string }) {
    return api.request<null>(`/api/v1/conversations/${convId}`, { method: 'PUT', body: data })
  },

  pin(convId: string) {
    return api.request<{ conv_id: string; pinned: boolean }>(`/api/v1/conversations/${convId}/pin`, { method: 'POST' })
  },

  unpin(convId: string) {
    return api.request<{ conv_id: string; pinned: boolean }>(`/api/v1/conversations/${convId}/unpin`, { method: 'POST' })
  },

  clone(convId: string, name?: string) {
    return api.request<{ conv_id: string; name: string }>(`/api/v1/conversations/${convId}/clone`, { method: 'POST', body: name ? { name } : {} })
  },

  addMembers(convId: string, userIds: string[]) {
    return api.request<null>(`/api/v1/conversations/${convId}/members`, {
      method: 'POST',
      body: { user_ids: userIds },
    })
  },

  removeMember(convId: string, userId: string) {
    return api.request<null>(`/api/v1/conversations/${convId}/members/${userId}`, {
      method: 'DELETE',
    })
  },

  leave(convId: string) {
    return api.request<null>(`/api/v1/conversations/${convId}/leave`, {
      method: 'POST',
    })
  },

  disband(convId: string) {
    return api.request<{ status: string }>(`/api/v1/conversations/${convId}/disband`, {
      method: 'POST',
    })
  },

  markRead(convId: string, msgId: number) {
    return api.request<null>(`/api/v1/conversations/${convId}/read`, {
      method: 'POST',
      body: { msg_id: msgId },
    })
  },

  requestJoin(convId: string) {
    return api.request<null>(`/api/v1/conversations/${convId}/join-requests`, {
      method: 'POST',
    })
  },

  listJoinRequests(convId: string) {
    return api.request<JoinRequest[]>(`/api/v1/conversations/${convId}/join-requests`)
  },

  approveJoinRequest(convId: string, userId: string) {
    return api.request<null>(`/api/v1/conversations/${convId}/join-requests/${userId}/approve`, {
      method: 'POST',
    })
  },

  rejectJoinRequest(convId: string, userId: string) {
    return api.request<null>(`/api/v1/conversations/${convId}/join-requests/${userId}/reject`, {
      method: 'POST',
    })
  },

  searchGroups(query: string) {
    return api.request<PaginatedData<ConvListItem>>('/api/v1/groups/search', { query: { q: query } }).then(d => d.items)
  },

  getUnreadTotal() {
    return api.request<{ total: number }>('/api/v1/conversations/unread/total')
  },

  getSettings(convId: string) {
    return api.request<{ settings: Record<string, unknown> }>(`/api/v1/conversations/${convId}/settings`)
  },

  updateSettings(convId: string, settings: Record<string, unknown>) {
    return api.request<{ settings: Record<string, unknown> }>(`/api/v1/conversations/${convId}/settings`, {
      method: 'PUT',
      body: { settings },
    })
  },
}
