import { api } from './api-client'
import type { Contact } from '@/types/contact'
import type { User } from '@/types/user'
import type { PaginatedData } from '@/types/api'

export const contactService = {
  list(page = 1, size = 100) {
    return api.request<PaginatedData<Contact>>('/api/v1/contacts', {
      query: { page, size },
    })
  },

  add(userId: string) {
    return api.request<null>('/api/v1/contacts', {
      method: 'POST',
      body: { user_id: userId },
    })
  },

  remove(userId: string) {
    return api.request<null>(`/api/v1/contacts/${userId}`, {
      method: 'DELETE',
    })
  },

  updateNickname(userId: string, nickname: string) {
    return api.request<null>(`/api/v1/contacts/${userId}`, {
      method: 'PUT',
      body: { nickname },
    })
  },

  // Alias for userService.batchGet for convenience
  batchGetUsers(userIds: string[]) {
    return api.request<Record<string, User>>('/api/v1/users/batch', {
      method: 'POST',
      body: { user_ids: userIds },
    })
  },
}
