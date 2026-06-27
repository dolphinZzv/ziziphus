import { useEffect, useState, useSyncExternalStore } from 'react'
import { contactStore } from '@/stores/contact-store'
import { uiStore } from '@/stores/ui-store'
import { userService } from '@/services/user-service'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import type { User } from '@/types/user'
import { X, UserPlus, MessageCircle, Trash2, Search } from 'lucide-react'

interface Props { onClose: () => void }

export default function ContactList({ onClose }: Props) {
  const contacts = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.contacts)
  const userMap = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.userMap)
  const onlineUsers = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.onlineUsers)
  const isLoading = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.isLoading)
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [searching, setSearching] = useState(false)
  const [showSearch, setShowSearch] = useState(false)

  useEffect(() => { contactStore.load() }, [])

  const handleChat = async (userId: string) => {
    try { const r = await conversationService.createP2P(userId); uiStore.closeSheet(); navigate(`/chat/${r.conv_id}`) } catch {}
  }

  const handleSearch = async () => {
    if (!query.trim()) return
    setSearching(true)
    try { setResults(await userService.search(query.trim())) } catch {}
    setSearching(false)
  }

  const handleAdd = async (userId: string) => {
    await contactStore.add(userId)
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-lg bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-[380px] max-h-[500px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg p-6 flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">联系人</h3>
          <div className="flex items-center gap-1">
            <button onClick={() => setShowSearch(!showSearch)} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><UserPlus size={16} /></button>
            <button onClick={onClose} className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
          </div>
        </div>

        {/* Add contact search */}
        {showSearch && (
          <div className="flex gap-2 mb-4">
            <input type="text" value={query} onChange={e => setQuery(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleSearch() } }}
              placeholder="搜索用户..." className={`${inputClass} flex-1`} />
            <button onClick={handleSearch} disabled={searching}
              className="h-[42px] px-4 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors disabled:opacity-40">
              <Search size={16} />
            </button>
          </div>
        )}

        {showSearch && results.length > 0 && (
          <div className="mb-4 max-h-[160px] overflow-y-auto space-y-0.5 border-b border-[var(--color-hairline)] pb-3">
            {results.map(user => (
              <div key={user.user_id} className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-[var(--color-surface-soft)]">
                <div className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-semibold flex-shrink-0"
                  style={{ background: user.primary_color ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})` : 'var(--color-primary)' }}>
                  {(user.name || user.account).charAt(0).toUpperCase()}
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm text-[var(--color-ink)]">{user.name}</div>
                  <div className="text-[11px] text-[var(--color-muted)]">@{user.account}</div>
                </div>
                <button onClick={() => handleAdd(user.user_id)}
                  className="px-3 py-1.5 rounded-lg bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-xs font-medium transition-colors">
                  添加
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="flex-1 overflow-y-auto space-y-0.5">
          {contacts.map(contact => {
            const user = userMap.get(contact.contact_id)
            const isOnline = onlineUsers.has(contact.contact_id)
            return (
              <div key={contact.contact_id} className="flex items-center gap-3 px-3 h-12 rounded-lg hover:bg-[var(--color-surface-soft)] group">
                <div className="relative">
                  <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-semibold"
                    style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                    {(user?.name || contact.nickname || contact.contact_id).charAt(0).toUpperCase()}
                  </div>
                  <span className={`absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-[var(--color-surface-card)] ${isOnline ? 'bg-[var(--success)]' : 'bg-[var(--color-muted)]'}`} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm text-[var(--color-ink)]">{user?.name || contact.nickname || contact.contact_id}</div>
                  <div className="text-[11px] text-[var(--color-muted)]">{isOnline ? '在线' : '离线'}</div>
                </div>
                <button onClick={() => handleChat(contact.contact_id)}
                  className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] opacity-0 group-hover:opacity-100 text-[var(--color-muted)] transition-all"><MessageCircle size={16} /></button>
                <button onClick={() => contactStore.remove(contact.contact_id)}
                  className="p-1.5 rounded-lg hover:bg-[var(--destructive)]/10 opacity-0 group-hover:opacity-100 text-[var(--destructive)] transition-all"><Trash2 size={14} /></button>
              </div>
            )
          })}
          {contacts.length === 0 && !isLoading && !showSearch && (
            <p className="text-sm text-[var(--color-muted)] text-center py-8">暂无联系人</p>
          )}
        </div>
      </div>
    </div>
  )
}
