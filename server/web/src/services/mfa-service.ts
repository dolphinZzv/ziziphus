import { api } from './api-client'

export interface MFAStatus {
  enabled: boolean
  mfa_type: number // 0=none, 1=totp, 2=email
}

export interface MFASetupResponse {
  mfa_type: number
  secret?: string
  uri?: string
}

export const mfaService = {
  getStatus() {
    return api.request<MFAStatus>('/api/v1/users/me/mfa')
  },

  setup(mfaType: number) {
    return api.request<MFASetupResponse>('/api/v1/users/me/mfa/setup', {
      method: 'POST',
      body: { mfa_type: mfaType },
    })
  },

  verify(code: string) {
    return api.request<{ enabled: boolean }>('/api/v1/users/me/mfa/verify', {
      method: 'POST',
      body: { code },
    })
  },

  disable() {
    return api.request<{ enabled: boolean }>('/api/v1/users/me/mfa/disable', {
      method: 'POST',
    })
  },

  sendEmailCode(email: string) {
    return api.request<{ code: string; expires_in: number }>('/api/v1/users/me/email/send-code', {
      method: 'POST',
      body: { email },
    })
  },

  confirmEmail(code: string) {
    return api.request<{ email: string; verified: boolean }>('/api/v1/users/me/email/confirm', {
      method: 'POST',
      body: { code },
    })
  },
}
