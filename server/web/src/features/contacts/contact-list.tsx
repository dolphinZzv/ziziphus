import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { contactStore } from '@/stores/contact-store'
import { uiStore } from '@/stores/ui-store'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import { X, MessageCircle, Trash2, Search, ArrowLeft } from 'lucide-react'
import { useIsMobile } from '@/hooks/use-breakpoint'
import { avatarUrl } from '@/lib/file'

interface Props { onClose: () => void; inline?: boolean }

export default function ContactList({ onClose, inline }: Props) { const isMobile=useIsMobile()
  const { t } = useTranslation()
  const contacts = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.contacts)
  const onlineUsers = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.onlineUsers)
  const isLoading = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.isLoading)
  const navigate = useNavigate()
  const [contactFilter, setContactFilter] = useState('')

  useEffect(() => { contactStore.load() }, [])

  const handleChat = async (userId: string) => {
    try { const r = await conversationService.createP2P(userId); uiStore.setSidebarView(null); navigate(`/conversations/${r.conv_id}`) } catch (e) { console.error(e) }
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  const inner = (
    <div className={`${inline ? 'h-full' : 'w-full sm:w-[380px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)]'} bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6 flex flex-col overflow-hidden`}
      style={inline ? {} : { boxShadow: 'var(--shadow-lg)' }} onClick={inline ? undefined : e => e.stopPropagation()}>
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          {inline && (
            <button onClick={() => uiStore.setSidebarView(null)} className="p-1 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">
              <ArrowLeft size={18} />
            </button>
          )}
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">{t('contact.title')}</h3>
        </div>
        {!inline && <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">{isMobile ? <ArrowLeft size={18} /> : <X size={16} />}</button>}
      </div>

      {/* Contact search filter */}
      {contacts.length > 0 && (
        <div className="mb-3">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)]" />
            <input type="text" value={contactFilter} onChange={e => setContactFilter(e.target.value)}
              placeholder={t('conversation.searchPlaceholder')} className={`${inputClass} pl-8`} />
          </div>
        </div>
      )}

      <div className="flex-1 overflow-y-auto space-y-0.5">
        {contacts.filter(contact => {
          if (!contactFilter.trim()) return true
          const name = (contact.name || contact.nickname || contact.user_id || '').toLowerCase()
          return name.includes(contactFilter.trim().toLowerCase())
        }).map(contact => {
          const displayName = contact.name || contact.nickname || contact.user_id
          const isOnline = onlineUsers.has(contact.user_id)
          return (
            <div key={contact.user_id} className="flex items-center gap-3 px-3 h-12 rounded-xl hover:bg-[var(--color-surface-soft)] group">
              <div className="relative">
                {contact.avatar ? (
                  <img src={avatarUrl(contact.avatar)} alt="" className="w-9 h-9 rounded-full object-cover" />
                ) : (
                  <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-semibold"
                    style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                    {displayName.charAt(0).toUpperCase()}
                  </div>
                )}
                <span className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-[var(--color-surface-card)] ${isOnline ? 'bg-[var(--success)]' : 'bg-[var(--color-muted)]'}`} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm text-[var(--color-ink)]">{displayName}</div>
                {contact.headline && <div className="text-[11px] text-[var(--color-muted)] truncate">{contact.headline}</div>}
              </div>
              <button onClick={() => handleChat(contact.user_id)}
                className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] opacity-0 group-hover:opacity-100 text-[var(--color-muted)] transition-all"><MessageCircle size={16} /></button>
              <button onClick={() => { if (confirm(`确定删除联系人 ${displayName}？`)) { contactStore.remove(contact.user_id).catch(() => {}) } }}
                className="p-1.5 rounded-xl hover:bg-[var(--destructive)]/10 opacity-0 group-hover:opacity-100 text-[var(--destructive)] transition-all"><Trash2 size={14} /></button>
            </div>
          )
        })}
        {contacts.length === 0 && !isLoading && (
          <p className="text-sm text-[var(--color-muted)] text-center py-8">{t('contact.noContacts')}</p>
        )}
      </div>
    </div>
  )

  if (inline) return inner
  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      {inner}
    </div>
  )
}
