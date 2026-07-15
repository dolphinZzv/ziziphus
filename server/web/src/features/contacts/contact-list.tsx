import { useEffect, useState, useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'
import { contactStore } from '@/stores/contact-store'
import { authStore } from '@/stores/auth-store'
import { uiStore } from '@/stores/ui-store'
import { userService } from '@/services/user-service'
import { contactRequestService } from '@/services/contact-request-service'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import type { User } from '@/types/user'
import { X, UserPlus, MessageCircle, Trash2, Search, Check, ArrowRight, ArrowLeft } from 'lucide-react'

interface Props { onClose: () => void; inline?: boolean }

export default function ContactList({ onClose, inline }: Props) {
  const { t } = useTranslation()
  const contacts = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.contacts)
  const onlineUsers = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.onlineUsers)
  const isLoading = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.isLoading)
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [searching, setSearching] = useState(false)
  const [showSearch, setShowSearch] = useState(false)
  const [sentRequests, setSentRequests] = useState<Set<string>>(new Set())

  const [requestError, setRequestError] = useState('')
  const [contactFilter, setContactFilter] = useState('')
  const [requestMessage, setRequestMessage] = useState('')

  useEffect(() => { contactStore.load() }, [])

  const handleChat = async (userId: string) => {
    try { const r = await conversationService.createP2P(userId); uiStore.setSidebarView(null); navigate(`/chat/${r.conv_id}`) } catch {}
  }

  const handleSearch = async () => {
    if (!query.trim()) return
    setSearching(true)
    try { setResults(await userService.search(query.trim())) } catch {}
    setSearching(false)
  }

  const [successMsg, setSuccessMsg] = useState('')
  const me = authStore.state.user

  const handleSendRequest = async (userId: string) => {
    setRequestError('')
    setSuccessMsg('')
    try {
      await contactRequestService.send(userId, requestMessage)
      setSentRequests(prev => new Set(prev).add(userId))
      setRequestMessage('')
      setSuccessMsg(t('contact.requestSentHint'))
    } catch (err: any) {
      setRequestError(err?.message || t('contact.sendFailed'))
    }
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  const inner = (
    <div className={`${inline ? 'h-full' : 'w-[380px] max-h-[500px]'} bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl p-6 flex flex-col overflow-hidden`}
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
        <div className="flex items-center gap-1">
          <button onClick={() => setShowSearch(!showSearch)} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><UserPlus size={16} /></button>
          {!inline && <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>}
        </div>
      </div>

      {/* Add contact search */}
      {showSearch && (
        <>
          <div className="flex gap-2 mb-3">
            <input type="text" value={query} onChange={e => setQuery(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleSearch() } }}
              placeholder="搜索用户..." className={`${inputClass} flex-1`} />
            <button onClick={handleSearch} disabled={searching}
              className="h-[42px] px-4 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors disabled:opacity-40">
              <Search size={16} />
            </button>
          </div>
          {requestError && (
            <div className="mb-3 text-xs text-[var(--destructive)]">{requestError}</div>
          )}
          {successMsg && (
            <div className="mb-3 flex items-center justify-between px-3 py-2 rounded-xl bg-green-500/10 text-xs text-green-600">
              <span className="flex items-center gap-1"><Check size={12} /> {successMsg}</span>
              <button onClick={() => { navigate(`/chat/sys:${me?.user_id}`); inline ? uiStore.setSidebarView(null) : onClose() }}
                className="flex items-center gap-1 px-2 py-1 rounded text-[10px] bg-green-500/20 hover:bg-green-500/30 transition-colors">
                {t('common.more')} <ArrowRight size={10} />
              </button>
            </div>
          )}
        </>
      )}

      {showSearch && results.length > 0 && (
        <>
          <div className="mb-2">
            <input type="text" value={requestMessage} onChange={e => setRequestMessage(e.target.value)}
              placeholder="附言（选填）..." className={`${inputClass} text-xs`} maxLength={200} />
          </div>
          <div className="mb-4 max-h-[160px] overflow-y-auto space-y-0.5 border-b border-[var(--color-hairline)] pb-3">
          {results.map(user => {
            const sent = sentRequests.has(user.user_id)
            return (
            <div key={user.user_id} className="flex items-center gap-3 px-3 py-2 rounded-xl hover:bg-[var(--color-surface-soft)]">
              <div className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-semibold flex-shrink-0"
                style={{ background: user.primary_color ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})` : 'var(--color-primary)' }}>
                {(user?.name || user?.account || '?').charAt(0).toUpperCase()}
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm text-[var(--color-ink)]">{user.name}</div>
                <div className="text-[11px] text-[var(--color-muted)]">@{user.account}</div>
              </div>
              {sent ? (
                <span className="flex items-center gap-1 px-2 py-1.5 text-xs text-[var(--color-muted)]">
                  <Check size={12} /> {t('contact.requestSent')}
                </span>
              ) : (
                <button onClick={() => handleSendRequest(user.user_id)}
                  className="px-3 py-1.5 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-xs font-medium transition-colors">
                  {t('contact.addFriend')}
                </button>
              )}
            </div>
          )})}
        </div>
        </>
      )}

      {/* Contact search filter */}
      {!showSearch && contacts.length > 0 && (
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
                <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-semibold"
                  style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                  {displayName.charAt(0).toUpperCase()}
                </div>
                <span className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-[var(--color-surface-card)] ${isOnline ? 'bg-[var(--success)]' : 'bg-[var(--color-muted)]'}`} />
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm text-[var(--color-ink)]">{displayName}</div>
                {contact.headline && <div className="text-[11px] text-[var(--color-muted)] truncate">{contact.headline}</div>}
                <div className="text-[11px] text-[var(--color-muted)]">{isOnline ? t('contact.online') : t('contact.offline')}</div>
              </div>
              <button onClick={() => handleChat(contact.user_id)}
                className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] opacity-0 group-hover:opacity-100 text-[var(--color-muted)] transition-all"><MessageCircle size={16} /></button>
              <button onClick={() => { if (confirm(`确定删除联系人 ${displayName}？`)) { contactStore.remove(contact.user_id).catch(() => {}) } }}
                className="p-1.5 rounded-xl hover:bg-[var(--destructive)]/10 opacity-0 group-hover:opacity-100 text-[var(--destructive)] transition-all"><Trash2 size={14} /></button>
            </div>
          )
        })}
        {contacts.length === 0 && !isLoading && !showSearch && (
          <p className="text-sm text-[var(--color-muted)] text-center py-8">{t('contact.noContacts')}</p>
        )}
      </div>
    </div>
  )

  if (inline) return inner
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      {inner}
    </div>
  )
}
