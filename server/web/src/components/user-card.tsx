import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
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

interface Props { userId: string; onClose: () => void }

export default function UserCard({ userId, onClose }: Props) {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
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

  const copyId = () => { navigator.clipboard.writeText(userId); setCopied(true); setTimeout(() => setCopied(false), 2000) }
  const handleStartChat = async () => {
    try { const r = await conversationService.createP2P(userId); onClose(); navigate(`/conversations/${r.conv_id}`) } catch {}
  }
  const handleAddContact = async () => {
    try { await contactRequestService.send(userId); setRequestSent(true) } catch {}
  }

  const initials = (user?.name || userId).charAt(0).toUpperCase()

  return (
    <div className="w-[240px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl overflow-hidden"
      style={{ boxShadow: 'var(--shadow-md)' }}>
      {loading ? (
        <div className="flex items-center justify-center py-8"><Loader2 size={16} className="animate-spin text-[var(--color-muted)]" /></div>
      ) : user ? (
        <>
          {/* Cover — full width behind avatar */}
          <div className="h-14 relative"
            style={{ background: user.cover
              ? `url(${user.cover.includes('?') ? user.cover : `${user.cover}?w=480&h=112`}) center/cover`
              : `linear-gradient(135deg, ${user.primary_color || 'var(--color-primary)'}, ${user.secondary_color || user.primary_color || 'var(--color-muted)'})` }}>
            {user.cover && <div className="absolute inset-0 bg-black/10" />}
          </div>

          {/* Avatar — overlaps cover */}
          <div className="flex justify-center -mt-7 mb-2">
            <div className="relative z-10">
              {user.avatar ? (
                <img src={avatarUrl(user.avatar, 128)} alt="" className="w-14 h-14 rounded-full object-cover border-[3px] border-[var(--color-surface-card)] shadow-sm" />
              ) : (
                <div className="w-14 h-14 rounded-full flex items-center justify-center text-white text-lg font-bold border-[3px] border-[var(--color-surface-card)] shadow-sm"
                  style={{ background: user.primary_color
                    ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})`
                    : isAgent ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : 'var(--color-primary)' }}>
                  {initials}
                </div>
              )}
            </div>
          </div>

          {/* Name */}
          <div className="text-center px-4 mb-2">
            <div className="font-headline text-base font-semibold text-[var(--color-ink)] flex items-center justify-center gap-1.5">
              {user.name}
              {isAgent && <Bot size={14} className="text-purple-500" />}
            </div>
            {user.headline && <div className="text-xs text-[var(--color-muted)] mt-0.5">{user.headline}</div>}
            <div className="text-xs text-[var(--color-muted)]">@{user.account}</div>
          </div>

          {/* ID */}
          <div className="flex items-center justify-center gap-1 text-[11px] text-[var(--color-muted-soft)] font-mono select-all mb-3">
            {userId.slice(0, 16)}...
            <button onClick={copyId} className="hover:text-[var(--color-ink)]">
              {copied ? <Check size={12} className="text-[var(--success)]" /> : <Copy size={12} />}
            </button>
          </div>

          {/* Actions */}
          {!isSelf && (
            <div className="flex gap-2 px-4 pb-4 pt-2 border-t border-[var(--color-hairline)]">
              <button onClick={handleStartChat}
                className="flex-1 h-8 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-xs font-medium flex items-center justify-center gap-1 transition-colors">
                <MessageCircle size={13} /> {t('contact.startChat')}
              </button>
              {isContact ? (
                <button disabled className="flex-1 h-8 rounded-xl bg-[var(--success)]/10 text-[var(--success)] text-xs font-medium flex items-center justify-center gap-1">
                  <UserCheck size={13} /> {t('contact.alreadyFriends')}
                </button>
              ) : requestSent ? (
                <button disabled className="flex-1 h-8 rounded-xl bg-[var(--color-muted)]/10 text-[var(--color-muted)] text-xs font-medium flex items-center justify-center gap-1">
                  <Check size={13} /> {t('contact.requestSent')}
                </button>
              ) : (
                <button onClick={handleAddContact}
                  className="flex-1 h-8 rounded-xl border border-[var(--color-hairline)] hover:bg-[var(--color-surface-soft)] text-[var(--color-body)] text-xs font-medium flex items-center justify-center gap-1 transition-colors">
                  <UserPlus size={13} /> {t('contact.addFriend')}
                </button>
              )}
            </div>
          )}
        </>
      ) : (
        <p className="text-xs text-[var(--color-muted)] text-center py-4">{t('error.loadFailed')}</p>
      )}
    </div>
  )
}
