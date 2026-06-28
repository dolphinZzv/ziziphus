import { api } from './api-client'

export interface ContactRequestInfo {
  id: number
  from_user_id: string
  to_user_id: string
  form_msg_id: number
  status: number // 0=pending, 1=approved, 2=rejected
  message: string
  created_at: number
  updated_at: number
}

export const contactRequestService = {
  send(userId: string, message?: string) {
    return api.request<{ request_id: number; form_msg_id: number }>(
      '/api/v1/contact-requests',
      { method: 'POST', body: { user_id: userId, message: message || '' } }
    )
  },

  listSent(page = 1, size = 20) {
    return api.request<ContactRequestInfo[]>(
      '/api/v1/contact-requests/sent',
      { query: { page, size } }
    )
  },

  listReceived(status?: number, page = 1, size = 20) {
    return api.request<ContactRequestInfo[]>(
      '/api/v1/contact-requests/received',
      { query: { status, page, size } }
    )
  },

  getByFormMsgId(formMsgId: number) {
    return api.request<ContactRequestInfo>(
      `/api/v1/contact-requests/by-form/${formMsgId}`
    )
  },
}
