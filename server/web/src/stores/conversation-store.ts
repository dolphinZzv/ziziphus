import { api } from '@/services/api-client'
import type { ConvListItem } from '@/types/conversation'

interface ConvState {
  conversations: ConvListItem[]
  unreadTotal: number
  isLoading: boolean
  _initialLoad: boolean
}

let state: ConvState = { conversations: [], unreadTotal: 0, isLoading: false, _initialLoad: true }
const listeners = new Set<() => void>()
function emit() { listeners.forEach(l => l()) }

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
      state = { ...state, conversations: data.items, unreadTotal: data.total, isLoading: false, _initialLoad: false }; emit()
    } catch {
      // On initial load failure, keep empty state and retry later
      state = { ...state, isLoading: false, _initialLoad: false }; emit()
    }
  },

  async refresh() {
    try {
      const data = await api.request<{ items: ConvListItem[]; total: number; page: number; size: number }>(
        '/api/v1/conversations', { query: { page: 1, size: 100 } }
      )
      state = { ...state, conversations: data.items, unreadTotal: data.total }; emit()
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
    updated.sort((a, b) =>
      (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0) || (b.last_msg_at || 0) - (a.last_msg_at || 0)
    )
    state = { ...state, conversations: updated }; emit()
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
    updated.sort((a, b) => (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0) || (b.last_msg_at || 0) - (a.last_msg_at || 0))
    state = { ...state, conversations: updated }; emit()
  },

  async unpin(convId: string) {
    try { await api.request(`/api/v1/conversations/${convId}/unpin`, { method: 'POST' }) } catch {}
    const updated = state.conversations.map(c => c.conv_id === convId ? { ...c, pinned: false } : c)
    updated.sort((a, b) => (b.pinned ? 1 : 0) - (a.pinned ? 1 : 0) || (b.last_msg_at || 0) - (a.last_msg_at || 0))
    state = { ...state, conversations: updated }; emit()
  },

  removeConversation(convId: string) {
    state = { ...state, conversations: state.conversations.filter(c => c.conv_id !== convId) }; emit()
  },
}
