import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import { contactStore } from '@/stores/contact-store'
import { uiStore } from '@/stores/ui-store'
import { useSyncExternalStore } from 'react'
import { X, Search, MessageCircle } from 'lucide-react'
import { avatarUrl } from '@/lib/file'

interface Props { onClose: () => void }

export default function NewChatDialog({ onClose }: Props) {
  const navigate = useNavigate()
  const contacts = useSyncExternalStore(contactStore.subscribe, () => contactStore.state.contacts)
  const [filter, setFilter] = useState('')

  const filtered = contacts.filter(c => {
    if (!filter.trim()) return true
    const name = (c.name || c.nickname || c.user_id || '').toLowerCase()
    return name.includes(filter.trim().toLowerCase())
  })

  const handleCreate = async (userId: string) => {
    try { const r = await conversationService.createP2P(userId); uiStore.closeSheet(); navigate(`/conversations/${r.conv_id}`) } catch {}
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[400px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6 flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">新建聊天</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="mb-4">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)]" />
            <input type="text" value={filter} onChange={e => setFilter(e.target.value)}
              placeholder="搜索联系人..." className={`${inputClass} pl-8`} />
          </div>
        </div>

        <div className="flex-1 overflow-y-auto space-y-0.5">
          {filtered.map(c => {
            const displayName = c.name || c.nickname || c.user_id
            return (
              <button key={c.user_id} type="button" onClick={() => handleCreate(c.user_id)}
                className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-[var(--color-surface-soft)] transition-colors">
                {c.avatar ? (
                  <img src={avatarUrl(c.avatar)} alt="" className="w-9 h-9 rounded-full object-cover flex-shrink-0" />
                ) : (
                  <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-semibold flex-shrink-0"
                    style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                    {displayName.charAt(0).toUpperCase()}
                  </div>
                )}
                <div className="text-left min-w-0 flex-1">
                  <div className="text-sm font-medium text-[var(--color-ink)]">{displayName}</div>
                  {c.headline && <div className="text-xs text-[var(--color-muted)] truncate">{c.headline}</div>}
                </div>
                <MessageCircle size={16} className="text-[var(--color-muted-soft)]" />
              </button>
            )
          })}
          {contacts.length === 0 && (
            <p className="text-sm text-[var(--color-muted)] text-center py-8">暂无联系人，请先添加好友</p>
          )}
        </div>
      </div>
    </div>
  )
}
