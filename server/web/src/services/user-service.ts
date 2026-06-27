import { api } from './api-client'
import type { User } from '@/types/user'
import type { PaginatedData } from '@/types/api'

export const userService = {
  async search(query: string): Promise<User[]> {
    const data = await api.request<PaginatedData<User> | User[]>('/api/v1/users/search', { query: { q: query } })
    // Server returns PaginatedData { items: [...] } — unwrap
    if (data && typeof data === 'object' && 'items' in data) return (data as PaginatedData<User>).items
    return (data as User[]) || []
  },

  getUser(userId: string) {
    return api.request<User>(`/api/v1/users/${userId}`)
  },

  async batchGet(userIds: string[]): Promise<Record<string, User>> {
    const data = await api.request<any>('/api/v1/users/batch', {
      method: 'POST',
      body: { user_ids: userIds },
    })
    // Server returns flat {user_id: User} map; unwrap if nested
    return data?.users || data || {}
  },
}
