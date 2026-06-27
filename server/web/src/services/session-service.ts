import { api } from './api-client'
import type { Session } from '@/types/session'

export const sessionService = {
  list() {
    return api.request<Session[]>('/api/v1/sessions')
  },

  delete(sessionId: string) {
    return api.request<null>(`/api/v1/sessions/${sessionId}`, {
      method: 'DELETE',
    })
  },
}
