import { useEffect, useState, useSyncExternalStore } from 'react'
import { useNavigate } from 'react-router-dom'
import { userService } from '@/services/user-service'
import { conversationService } from '@/services/conversation-service'
import { contactRequestService } from '@/services/contact-request-service'
import { authStore } from '@/stores/auth-store'
import { contactStore } from '@/stores/contact-store'
import { avatarUrl } from '@/lib/file'
import type { User } from '@/types/user'
import { UserType } from '@/types/user'
import { Copy, Check, Loader2, Bot, MessageCircle, UserPlus, UserCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface Props {
  userId: string
  onClose: () => void
}

export default function UserCard({ userId, onClose }: Props) {
  const navigate = useNavigate()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const { t } = useTranslation()
  const [copied, setCopied] = useState(false)
  const [requestSent, setRequestSent] = useState(false)
  const currentUser = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const contacts = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.contacts)
  const isSelf = currentUser?.user_id === userId
  const isContact = contacts.some(c => c.user_id === userId)
  const isAgent = user?.type === UserType.Agent

  useEffect(() => {
    userService.getUser(userId).then(u => { setUser(u); setLoading(false) }).catch(() => setLoading(false))
  }, [userId])

  const copyId = () => {
    navigator.clipboard.writeText(userId)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const handleStartChat = async () => {
    try {
      const r = await conversationService.createP2P(userId)
      onClose()
      navigate(`/chat/${r.conv_id}`)
    } catch {}
  }

  const handleAddContact = async () => {
    try {
      await contactRequestService.send(userId)
      setRequestSent(true)
    } catch {}
  }

  const initials = (user?.name || userId).charAt(0).toUpperCase()

  return (
    <div className="w-[240px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg p-4"
      style={{ boxShadow: 'var(--shadow-md)' }}>
      {loading ? (
          <div className="flex items-center justify-center py-4"><Loader2 size={16} className="animate-spin text-[var(--color-muted)]" /></div>
        ) : user ? (
          <div className="text-center space-y-3">
            {/* Avatar */}
            <div className="flex justify-center">
              {user.avatar ? (
                <img src={avatarUrl(user.avatar)} alt="" className="w-14 h-14 rounded-full object-cover" />
              ) : (
                <div className="w-14 h-14 rounded-full flex items-center justify-center text-white text-lg font-bold"
                  style={{ background: user.primary_color
                    ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})`
                    : isAgent ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : 'var(--color-primary)' }}>
                  {initials}
                </div>
              )}
            </div>

            {/* Name + type badge */}
            <div>
              <div className="font-headline text-base font-semibold text-[var(--color-ink)] flex items-center justify-center gap-1.5">
                {user.name}
                {isAgent && <Bot size={14} className="text-purple-500" />}
              </div>
              <div className="text-xs text-[var(--color-muted)] mt-0.5">@{user.account}</div>
            </div>

            {/* ID */}
            <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted-soft)] font-mono select-all">
              {userId.slice(0, 16)}...
              <button onClick={copyId} className="hover:text-[var(--color-ink)]">
                {copied ? <Check size={12} className="text-[var(--success)]" /> : <Copy size={12} />}
              </button>
            </div>

            {/* Actions — only for non-self users */}
            {!isSelf && (
              <div className="flex gap-2 pt-1 border-t border-[var(--color-hairline)]">
                <button onClick={handleStartChat}
                  className="flex-1 h-8 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-xs font-medium flex items-center justify-center gap-1 transition-colors">
                  <MessageCircle size={13} /> 发起会话
                </button>
                {isContact ? (
                  <button disabled
                    className="flex-1 h-8 rounded-lg bg-[var(--success)]/10 text-[var(--success)] text-xs font-medium flex items-center justify-center gap-1">
                    <UserCheck size={13} /> 已是好友
                  </button>
                ) : requestSent ? (
                  <button disabled
                    className="flex-1 h-8 rounded-lg bg-[var(--color-muted)]/10 text-[var(--color-muted)] text-xs font-medium flex items-center justify-center gap-1">
                    <Check size={13} /> 已发送
                  </button>
                ) : (
                  <button onClick={handleAddContact}
                    className="flex-1 h-8 rounded-lg border border-[var(--color-hairline)] hover:bg-[var(--color-surface-soft)] text-[var(--color-body)] text-xs font-medium flex items-center justify-center gap-1 transition-colors">
                    <UserPlus size={13} /> 添加好友
                  </button>
                )}
              </div>
            )}
          </div>
        ) : (
          <p className="text-xs text-[var(--color-muted)] text-center py-2">{t('error.loadFailed')}</p>
        )}
    </div>
  )
}
