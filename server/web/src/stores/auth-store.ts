import { getItem, setItem, removeItem } from '@/lib/secure-storage'
import { api, __setLogoutHandler } from '@/services/api-client'
import { wsClient } from '@/services/websocket-client'
import type { User } from '@/types/user'

export interface MFAChallenge {
  userId: string
  mfaToken: string
  mfaType: number
  maskedEmail?: string
  code?: string
}

export interface AuthState {
  user: User | null
  token: string
  refreshToken: string
  sessionId: string
  isLoggedIn: boolean
  isLoading: boolean
  error: string | null
  _initialized: boolean
  mfaChallenge: MFAChallenge | null
}

function getInitialState(): AuthState {
  let token = ''
  let user: User | null = null
  let sessionId = ''
  let refreshToken = ''
  try {
    token = getItem<string>('token', '')!
    user = getItem<User>('user', null)!
    sessionId = getItem<string>('session_id', '')!
    refreshToken = getItem<string>('refresh_token', '')!
    // Validate cached user has essential fields
    if (user && !user.name && !user.account) {
      user = null
      removeItem('user')
    }
  } catch {
    // Corrupted localStorage — clear it
    removeItem('token')
    removeItem('user')
    removeItem('session_id')
    removeItem('refresh_token')
  }
  return {
    user,
    token,
    refreshToken,
    sessionId,
    isLoggedIn: !!token && !!user,
    isLoading: false,
    mfaChallenge: null,
    error: null,
    _initialized: !token || !user, // 已登录则先显示页面，异步验证
  }
}

let state = getInitialState()
const listeners = new Set<() => void>()

// Wire up auto-logout when API returns 401
__setLogoutHandler(() => authStore.logout())

function emit() {
  listeners.forEach(l => l())
}

export const authStore = {
  get state() { return state },

  subscribe(fn: () => void) {
    listeners.add(fn)
    return () => { listeners.delete(fn) }
  },

  async login(account: string, password: string) {
    state = { ...state, isLoading: true, error: null, mfaChallenge: null }; emit()
    try {
      const result = await api.request<Record<string, unknown>>(
        '/api/v1/users/login', { method: 'POST', body: { account, password } }
      )
      if (result.mfa_required) {
        state = {
          ...state,
          isLoading: false,
          mfaChallenge: {
            userId: result.user_id as string,
            mfaToken: result.mfa_token as string,
            mfaType: result.mfa_type as number,
            maskedEmail: result.masked_email as string | undefined,
            code: result.code as string | undefined,
          },
        }; emit()
        return
      }
      const user: User = {
        user_id: result.user_id as string,
        name: (result.name as string) || account,
        account: (result.account as string) || account,
        avatar: (result.avatar as string) || '',
        cover: (result.cover as string) || '',
        email: (result.email as string) || '',
        type: (result.type as number) || 0,
        status: (result.status as number) || 1,
        uid: (result.uid as string) || '',
        primary_color: (result.primary_color as string) || '#0F172A',
        secondary_color: (result.secondary_color as string) || '#64748B',
        wake_mode: (result.wake_mode as number) || 0,
        api_key: (result.api_key as string) || '',
        discoverable: (result.discoverable as boolean) ?? true,
        allow_direct_chat: (result.allow_direct_chat as boolean) ?? true,
        created_at: (result.created_at as number) || 0,
      }
      this.setAuth(user, result.token as string, result.refresh_token as string, (result.session_id as string) || '')
      wsClient.connect(result.token as string)
      this.refreshUserProfile()
    } catch (e: unknown) {
      state = { ...state, isLoading: false, error: e instanceof Error ? e.message : 'Login failed' }; emit()
      throw e
    }
  },

  async mfaVerify(code: string) {
    const challenge = state.mfaChallenge
    if (!challenge) throw new Error('No MFA challenge')
    state = { ...state, isLoading: true, error: null }; emit()
    try {
      const result = await api.request<Record<string, unknown>>('/api/v1/auth/mfa/verify', {
        method: 'POST',
        body: { user_id: challenge.userId, mfa_token: challenge.mfaToken, code },
      })
      const user: User = {
        user_id: result.user_id as string,
        name: (result.name as string) || '',
        account: (result.account as string) || '',
        avatar: (result.avatar as string) || '',
        cover: (result.cover as string) || '',
        email: (result.email as string) || '',
        type: (result.type as number) || 0,
        status: (result.status as number) || 1,
        uid: (result.uid as string) || '',
        primary_color: (result.primary_color as string) || '#0F172A',
        secondary_color: (result.secondary_color as string) || '#64748B',
        wake_mode: (result.wake_mode as number) || 0,
        api_key: (result.api_key as string) || '',
        discoverable: (result.discoverable as boolean) ?? true,
        allow_direct_chat: (result.allow_direct_chat as boolean) ?? true,
        created_at: (result.created_at as number) || 0,
      }
      this.setAuth(user, result.token as string, result.refresh_token as string, (result.session_id as string) || '')
      wsClient.connect(result.token as string)
      this.refreshUserProfile()
      state = { ...state, mfaChallenge: null }; emit()
    } catch (e: unknown) {
      state = { ...state, isLoading: false, error: e instanceof Error ? e.message : '验证失败' }; emit()
      throw e
    }
  },

  async register(account: string, name: string, password: string, email?: string) {
    state = { ...state, isLoading: true, error: null }; emit()
    try {
      const result = await api.request<Record<string, unknown>>(
        '/api/v1/users/register', { method: 'POST', body: { account, name, password, email: email || '' } }
      )
      const user: User = {
        user_id: result.user_id as string,
        name: (result.name as string) || name,
        account: (result.account as string) || account,
        avatar: (result.avatar as string) || '',
        cover: (result.cover as string) || '',
        email: (result.email as string) || (email || ''),
        type: (result.type as number) || 0,
        status: (result.status as number) || 1,
        uid: (result.uid as string) || '',
        primary_color: (result.primary_color as string) || '#0F172A',
        secondary_color: (result.secondary_color as string) || '#64748B',
        wake_mode: (result.wake_mode as number) || 0,
        api_key: (result.api_key as string) || '',
        discoverable: (result.discoverable as boolean) ?? true,
        allow_direct_chat: (result.allow_direct_chat as boolean) ?? true,
        created_at: (result.created_at as number) || 0,
      }
      this.setAuth(user, result.token as string, result.refresh_token as string, (result.session_id as string) || '')
      wsClient.connect(result.token as string)
      this.refreshUserProfile()
    } catch (e: unknown) {
      state = { ...state, isLoading: false, error: e instanceof Error ? e.message : 'Registration failed' }; emit()
      throw e
    }
  },

  setAuth(user: User, token: string, refreshToken: string, sessionId: string) {
    setItem('user', user)
    setItem('token', token)
    setItem('refresh_token', refreshToken)
    setItem('session_id', sessionId)
    state = { ...state, user, token, refreshToken, sessionId, isLoggedIn: true, isLoading: false, error: null, _initialized: true }
    emit()
  },

  async refreshUserProfile() {
    try {
      const me = await api.request<User>('/api/v1/users/me')
      setItem('user', me)
      state = { ...state, user: me }; emit()
    } catch { /* keep cached user */ }
  },

  async checkExistingSession() {
    const token = getItem<string>('token', '')
    const user = getItem<User>('user', null)
    if (!token || !user) {
      state = { ...state, isLoading: false, _initialized: true }; emit()
      return
    }
    // Already showing main layout from initial state — just connect WS and verify in background
    wsClient.connect(token)
    state = { ...state, isLoading: true }; emit()
    try {
      const me = await api.request<User>('/api/v1/users/me')
      setItem('user', me)
      state = { ...state, user: me, isLoading: false, _initialized: true }; emit()
    } catch {
      // Token may be expired but keep showing cached UI
      state = { ...state, isLoading: false, _initialized: true }; emit()
    }
  },

  async updateProfile(data: { name?: string; avatar?: string; cover?: string; email?: string; primary_color?: string; secondary_color?: string; headline?: string; discoverable?: boolean; allow_direct_chat?: boolean }) {
    const cur = state.user
    // Always send all fields so server-side UPDATE doesn't wipe unchanged columns
    const body = {
      name: data.name ?? cur?.name ?? '',
      avatar: data.avatar ?? cur?.avatar ?? '',
      cover: data.cover ?? cur?.cover ?? '',
      email: data.email ?? cur?.email ?? '',
      primary_color: data.primary_color ?? cur?.primary_color ?? '',
      secondary_color: data.secondary_color ?? cur?.secondary_color ?? '',
      headline: data.headline ?? cur?.headline ?? '',
      discoverable: data.discoverable ?? cur?.discoverable ?? true,
      allow_direct_chat: data.allow_direct_chat ?? cur?.allow_direct_chat ?? true,
    }
    await api.request<Record<string, unknown>>('/api/v1/users/me', { method: 'PUT', body })
    let user = await api.request<User>('/api/v1/users/me')
    if (!user.avatar && body.avatar) user = { ...user, avatar: body.avatar }
    setItem('user', user)
    state = { ...state, user }; emit()
    return user
  },

  logout() {
    wsClient.disconnect()
    removeItem('user')
    removeItem('token')
    removeItem('refresh_token')
    removeItem('session_id')
    for (const key of Object.keys(localStorage)) {
      if (key.startsWith('ziziphus_msg_') || key.startsWith('ziziphus_conv_')) {
        localStorage.removeItem(key)
      }
    }
    state = { ...state, user: null, token: '', refreshToken: '', sessionId: '', isLoggedIn: false, isLoading: false, error: null, _initialized: true }
    emit()
  },
}
