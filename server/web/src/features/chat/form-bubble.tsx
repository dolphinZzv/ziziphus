import { useState, useEffect } from 'react'
import type { FormDefinitionBody } from '@/types/form'
import { ContentType } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
import { authStore } from '@/stores/auth-store'
import { api } from '@/services/api-client'
import { avatarUrl } from '@/lib/file'
import { cn } from '@/lib/cn'
import { Check, X, Loader2 } from 'lucide-react'
interface Props {
  body: string
  msgId: number
  convId: string
  senderId: string
}

interface ContactRequestInfo {
  id: number
  status: number // 0=pending, 1=approved, 2=rejected
}

type ActionState = 'idle' | 'loading' | 'approved' | 'rejected' | 'error'

export default function FormBubble({ body, msgId, convId, senderId }: Props) {
  const [form] = useState<FormDefinitionBody>(() => {
    try { return JSON.parse(body) }
    catch { return null }
  })
  const [actionState, setActionState] = useState<ActionState>('idle')
  const [errorMsg, setErrorMsg] = useState('')

  const me = authStore.state.user
  // For form messages, the sender_id is empty (system-sent).
  // Use form.from_user_id to determine if this is the current user's own form.
  const isMine = me?.user_id === senderId || me?.user_id === form?.from_user_id

  // Fetch authoritative status from server on mount.
  useEffect(() => {
    if (!form || form.type !== 'contact_request') return

    api.request<ContactRequestInfo>(`/api/v1/contact-requests/by-form/${msgId}`)
      .then(req => {
        if (req.status === 1) setActionState('approved')
        else if (req.status === 2) setActionState('rejected')
        // else 0 = pending, leave idle
      })
      .catch(() => {
        // Network error — leave body.status as initial placeholder.
        // The user can still try to act; the server will validate.
      })
  }, [form, msgId])

  if (!form) {
    return <div className="text-xs text-[var(--color-muted)]">[表单数据异常]</div>
  }

  // Generic form rendering (not a contact request)
  if (form.type !== 'contact_request') {
    return (
      <div className="space-y-2">
        <div className="font-medium text-sm">{form.title}</div>
        {form.description && <div className="text-xs opacity-70">{form.description}</div>}
        {form.actions.map(a => (
          <button key={a.action} disabled className="px-3 py-1 rounded text-xs bg-[var(--color-muted)]/20">
            {a.label}
          </button>
        ))}
      </div>
    )
  }

  // Contact request card
  const handleAction = (action: string) => {
    setActionState('loading')
    setErrorMsg('')

    const respBody = JSON.stringify({
      form_msg_id: msgId,
      request_id: form.request_id,
      action,
      responder_id: me?.user_id || '',
      responder_name: me?.name || '',
      submitted_at: Date.now(),
    })

    const timeoutId = setTimeout(() => {
      setActionState('idle')
      setErrorMsg(t('error.timeoutRetry'))
    }, 5000)

    chatStore.sendMessage(convId, respBody, ContentType.FormResponse, msgId)
      .then(() => {
        clearTimeout(timeoutId)
        setActionState(action === 'approve' ? 'approved' : 'rejected')
      })
      .catch(err => {
        clearTimeout(timeoutId)
        setActionState('idle')
        setErrorMsg(err?.message || t('error.retry'))
      })
  }

  const isResolved = actionState === 'approved' || actionState === 'rejected'
  const isLoading = actionState === 'loading'

  return (
    <div className="space-y-2 max-w-[260px]">
      {/* Sender info */}
      {!isMine && (
        <div className="flex items-center gap-2">
          {form.from_user_avatar ? (
            <img loading="lazy" decoding="async" src={avatarUrl(form.from_user_avatar)} alt="" className="w-8 h-8 rounded-full object-cover" />
          ) : (
            <div className="w-8 h-8 rounded-full bg-[var(--color-muted)]/20 flex items-center justify-center text-xs font-semibold">
              {form.from_user_name?.charAt(0) || '?'}
            </div>
          )}
          <div>
            <div className="text-sm font-medium">{form.from_user_name}</div>
            <div className="text-xs text-[var(--color-muted)]">{form.title}</div>
          </div>
        </div>
      )}

      {/* Message */}
      {form.message && (
        <div className="text-xs bg-[var(--color-surface-soft)] rounded px-2 py-1 text-[var(--color-body)]">
          {form.message}
        </div>
      )}

      {/* Result badge */}
      {isResolved && (
        <div className={cn(
          'flex items-center gap-1 text-xs font-medium px-2 py-1 rounded',
          actionState === 'approved'
            ? 'bg-green-500/10 text-green-600'
            : 'bg-red-500/10 text-red-600'
        )}>
          {actionState === 'approved' ? <Check size={12} /> : <X size={12} />}
          {actionState === 'approved' ? (form.actions.find(a => a.action === 'approve')?.label || t('friendRequest.approved')) : (form.actions.find(a => a.action === 'reject')?.label || t('friendRequest.rejected'))}
        </div>
      )}

      {/* Error message */}
      {errorMsg && (
        <div className="text-xs text-[var(--destructive)]">{errorMsg}</div>
      )}

      {/* Action buttons (only for recipient, when not resolved) */}
      {!isMine && !isResolved && (
        <div className="flex gap-2">
          {form.actions.map(a => (
            <button
              key={a.action}
              disabled={isLoading}
              onClick={() => handleAction(a.action)}
              className={cn(
                'px-3 py-1 rounded text-xs font-medium transition-colors disabled:opacity-50',
                a.style === 'primary'
                  ? 'bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)]'
                  : a.style === 'danger'
                    ? 'bg-red-500/10 text-red-600 hover:bg-red-500/20 border border-red-500/20'
                    : 'bg-[var(--color-muted)]/20 text-[var(--color-body)] hover:bg-[var(--color-muted)]/30'
              )}
            >
              {isLoading && <Loader2 size={10} className="inline mr-1 animate-spin" />}
              {a.label}
            </button>
          ))}
        </div>
      )}

      {/* For sender: show pending status */}
      {isMine && !isResolved && (
        <div className="text-xs text-[var(--color-muted)]">等待对方回复</div>
      )}
    </div>
  )
}
