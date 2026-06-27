import { api } from '@/services/api-client'
import type { Contact } from '@/types/contact'
import type { User } from '@/types/user'

interface ContactState {
  contacts: Contact[]
  userMap: Map<string, User>
  onlineUsers: Set<string>
  isLoading: boolean
}

let state: ContactState = { contacts: [], userMap: new Map(), onlineUsers: new Set(), isLoading: false }
const listeners = new Set<() => void>()
function emit() { listeners.forEach(l => l()) }

export const contactStore = {
  get state() { return state },

  subscribe(fn: () => void) {
    listeners.add(fn)
    return () => { listeners.delete(fn) }
  },

  async load() {
    state = { ...state, isLoading: true }; emit()
    try {
      const data = await api.request<{ items: Contact[]; total: number; page: number; size: number }>(
        '/api/v1/contacts', { query: { page: 1, size: 200 } }
      )
      const userMap = new Map(state.userMap)
      const userIds = data.items.map(c => c.contact_id)
      if (userIds.length > 0) {
        try {
          const users = await api.request<Record<string, User>>('/api/v1/users/batch', {
            method: 'POST', body: { user_ids: userIds },
          })
          Object.entries(users).forEach(([k, v]) => userMap.set(k, v))
        } catch { /* ignore */ }
      }
      state = { ...state, contacts: data.items, userMap, isLoading: false }; emit()
    } catch {
      state = { ...state, isLoading: false }; emit()
    }
  },

  async add(userId: string) {
    await api.request('/api/v1/contacts', { method: 'POST', body: { user_id: userId } })
    await this.load()
  },

  async remove(userId: string) {
    await api.request(`/api/v1/contacts/${userId}`, { method: 'DELETE' })
    state = { ...state, contacts: state.contacts.filter(c => c.contact_id !== userId) }; emit()
  },

  async updateNickname(userId: string, nickname: string) {
    await api.request(`/api/v1/contacts/${userId}`, { method: 'PUT', body: { nickname } })
    state = { ...state, contacts: state.contacts.map(c => c.contact_id === userId ? { ...c, nickname } : c) }; emit()
  },

  setOnline(userId: string, online: boolean) {
    const onlineUsers = new Set(state.onlineUsers)
    if (online) onlineUsers.add(userId)
    else onlineUsers.delete(userId)
    state = { ...state, onlineUsers }; emit()
  },

  isOnline(userId: string): boolean {
    return state.onlineUsers.has(userId)
  },

  getUser(userId: string): User | undefined {
    return state.userMap.get(userId)
  },
}
