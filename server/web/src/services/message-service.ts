import { api } from './api-client'
import type { Message } from '@/types/message'
import type { PaginatedData } from '@/types/api'

export const messageService = {
  getHistory(
    convId: string,
    params: {
      before?: number
      around?: number
      limit?: number
      keyword?: string
      start_date?: string
      end_date?: string
    } = {}
  ) {
    return api.request<PaginatedData<Message>>(`/api/v1/conversations/${convId}/messages`, {
      query: {
        before: params.before,
        around: params.around,
        limit: params.limit || 50,
        keyword: params.keyword,
        start_date: params.start_date,
        end_date: params.end_date,
      },
    })
  },
}
