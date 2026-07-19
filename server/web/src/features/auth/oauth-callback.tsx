import { useEffect, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { authStore } from '@/stores/auth-store'
import { wsClient } from '@/services/websocket-client'
import { Loader } from 'lucide-react'

export default function OAuthCallback() {
  const navigateRef = useRef(useNavigate())
  const paramsRef = useRef(useSearchParams()[0])
  const called = useRef(false)

  useEffect(() => {
    if (called.current) return
    called.current = true
    const navigate = navigateRef.current
    const params = paramsRef.current

    const error = params.get('error')
    if (error) {
      const msg = decodeURIComponent(error)
      if (!msg.startsWith('provider_error:')) {
        alert(msg)
      }
      navigate('/login', { replace: true })
      return
    }

    const token = params.get('token')
    const refreshToken = params.get('refresh_token')
    const fileToken = params.get('file_token')
    const userID = params.get('user_id')

    if (token && refreshToken && userID) {
      fetch('/api/v1/users/me', {
        headers: { 'Authorization': `Bearer ${token}` }
      })
        .then(r => r.json())
        .then(data => {
          if (data.user_id) {
            authStore.setAuth(data, token, refreshToken, '', fileToken || '')
            wsClient.connect(token)
            navigate('/conversations', { replace: true })
          } else {
            throw new Error('Invalid user data')
          }
        })
        .catch(() => {
          authStore.setAuth(
            {
              user_id: userID,
              name: params.get('name') || userID,
              account: userID,
              avatar: '',
              cover: '',
              type: 0,
              status: 1,
              uid: '',
              primary_color: '#0F172A',
              secondary_color: '#64748B',
              discoverable: true,
              allow_direct_chat: true,
              created_at: 0,
            },
            token,
            refreshToken,
            '',
            fileToken || ''
          )
          wsClient.connect(token)
          navigate('/conversations', { replace: true })
        })
    } else {
      navigate('/login', { replace: true })
    }
  }, [])

  return (
    <div className="h-full flex items-center justify-center">
      <Loader size={24} className="animate-spin text-[var(--color-muted)]" />
    </div>
  )
}
