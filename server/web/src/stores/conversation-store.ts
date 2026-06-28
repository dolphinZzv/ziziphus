import { api } from '@/services/api-client'
import { ConvType } from '@/types/conversation'
import type { ConvListItem } from '@/types/conversation'
import { ContentType } from '@/types/message'
import i18n from '@/i18n'

interface ConvState {
  conversations: ConvListItem[]
  unreadTotal: number
  isLoading: boolean
  _initialLoad: boolean
}

let state: ConvState = { conversations: [], unreadTotal: 0, isLoading: false, _initialLoad: true }
const listeners = new Set<() => void>()
function emit() { listeners.forEach(l => l()) }

function sortConversations(items: ConvListItem[]) {
  return [...items].sort((a, b) => {
    // Pinned first
    if ((b.pinned ? 1 : 0) !== (a.pinned ? 1 : 0)) return (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0)
    // Then by last_msg_at DESC
    return (b.last_msg_at || 0) - (a.last_msg_at || 0)
  })
}

export function getConversationPreview(item: ConvListItem): string {
  if (!item.last_message) return ''
  const m = item.last_message

  if (m.content_type === ContentType.Form) {
    try {
      const body = JSON.parse(m.body)
      if (body.type === 'contact_request' && body.from_user_name) {
        return `好友申请 · ${body.from_user_name}`
      }
    } catch { /* fallthrough */ }
  }

  if (m.content_type === ContentType.FormResponse) {
    try {
      const body = JSON.parse(m.body)
      if (body.action === 'approve') return `你已通过${body.responder_name}的好友申请`
      if (body.action === 'reject') return `你已拒绝${body.responder_name}的好友申请`
    } catch { /* fallthrough */ }
  }

  // Default: sender + body text
  const sender = m.sender_name || m.sender_id || ''
  const prefix = sender ? `${sender}: ` : ''
  return prefix + m.body
}

export const conversationStore = {
  get state() { return state },

  subscribe(fn: () => void) {
    listeners.add(fn)
    return () => { listeners.delete(fn) }
  },

  async load() {
    state = { ...state, isLoading: true }; emit()
    try {
      const data = await api.request<{ items: ConvListItem[]; total: number; page: number; size: number }>(
        '/api/v1/conversations', { query: { page: 1, size: 100 } }
      )
      state = { ...state, conversations: sortConversations(data.items), unreadTotal: data.total, isLoading: false, _initialLoad: false }; emit()
    } catch {
      state = { ...state, isLoading: false, _initialLoad: false }; emit()
    }
  },

  async refresh() {
    try {
      const data = await api.request<{ items: ConvListItem[]; total: number; page: number; size: number }>(
        '/api/v1/conversations', { query: { page: 1, size: 100 } }
      )
      state = { ...state, conversations: sortConversations(data.items), unreadTotal: data.total }; emit()
    } catch { /* silent */ }
  },

  upsertConversation(item: ConvListItem) {
    const idx = state.conversations.findIndex(c => c.conv_id === item.conv_id)
    const updated = [...state.conversations]
    if (idx >= 0) {
      updated[idx] = { ...updated[idx], ...item }
    } else {
      updated.unshift(item)
    }
    state = { ...state, conversations: sortConversations(updated) }; emit()
  },

  incrementUnread(convId: string) {
    const updated = state.conversations.map(c =>
      c.conv_id === convId ? { ...c, unread_count: c.unread_count + 1 } : c
    )
    state = { ...state, conversations: updated }; emit()
  },

  markRead(convId: string) {
    const updated = state.conversations.map(c =>
      c.conv_id === convId ? { ...c, unread_count: 0, mention_me: false } : c
    )
    state = { ...state, conversations: updated }; emit()
  },

  async pin(convId: string) {
    try { await api.request(`/api/v1/conversations/${convId}/pin`, { method: 'POST' }) } catch {}
    const updated = state.conversations.map(c => c.conv_id === convId ? { ...c, pinned: true } : c)
    state = { ...state, conversations: sortConversations(updated) }; emit()
  },

  async unpin(convId: string) {
    try { await api.request(`/api/v1/conversations/${convId}/unpin`, { method: 'POST' }) } catch {}
    const updated = state.conversations.map(c => c.conv_id === convId ? { ...c, pinned: false } : c)
    state = { ...state, conversations: sortConversations(updated) }; emit()
  },

  removeConversation(convId: string) {
    state = { ...state, conversations: state.conversations.filter(c => c.conv_id !== convId) }; emit()
  },

  has(convId: string): boolean {
    return state.conversations.some(c => c.conv_id === convId)
  },
}
