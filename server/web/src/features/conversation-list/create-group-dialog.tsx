import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import { userService } from '@/services/user-service'
import { uiStore } from '@/stores/ui-store'
import type { User } from '@/types/user'
import { X, Search, Users } from 'lucide-react'
import { cn } from '@/lib/cn'

interface Props { onClose: () => void }

export default function CreateGroupDialog({ onClose }: Props) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [groupName, setGroupName] = useState('')
  const [headline, setHeadline] = useState('')
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [selected, setSelected] = useState<User[]>([])
  const [creating, setCreating] = useState(false)

  const handleSearch = async () => {
    if (!query.trim()) return
    try { setResults(await userService.search(query.trim())) } catch {}
  }

  const toggleUser = (user: User) => {
    setSelected(prev => prev.find(u => u.user_id === user.user_id) ? prev.filter(u => u.user_id !== user.user_id) : [...prev, user])
  }

  const handleCreate = async () => {
    if (!groupName.trim() || selected.length === 0) return
    setCreating(true)
    try {
      const r = await conversationService.createGroup(groupName.trim(), headline.trim(), selected.map(u => u.user_id));
      // Navigate first — SheetRouteSync's URL change handler will close the sheet
      // with syncing.current=true, preventing navigate(-1) from firing
      navigate(`/conversations/${r.conv_id}`);
    } catch {}
    setCreating(false)
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[420px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6 flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">创建群组</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <input type="text" value={groupName} onChange={e => setGroupName(e.target.value)} placeholder="群组名称" className={cn(inputClass, 'mb-2')} />
        <input type="text" value={headline} onChange={e => setHeadline(e.target.value)} placeholder="群组简介（选填）" maxLength={120} className={cn(inputClass, 'mb-3')} />

        {/* Selected members — chips */}
        {selected.length > 0 && (
          <div className="flex flex-wrap gap-1.5 mb-3">
            {selected.map(u => (
              <span key={u.user_id} className="inline-flex items-center gap-1 px-2 py-1 rounded bg-[var(--color-primary)] text-white text-xs font-medium">
                {u.name} <X size={12} className="cursor-pointer opacity-70 hover:opacity-100" onClick={() => toggleUser(u)} />
              </span>
            ))}
          </div>
        )}

        <div className="flex gap-2 mb-3">
          <input type="text" value={query} onChange={e => setQuery(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleSearch() } }}
            placeholder="搜索成员..." className={cn(inputClass, 'flex-1')} />
          <button onClick={handleSearch} className="h-[42px] px-4 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors"><Search size={16} /></button>
        </div>

        <div className="flex-1 overflow-y-auto space-y-0.5 mb-4">
          {results.map(user => (
            <button key={user.user_id} onClick={() => toggleUser(user)}
              className="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-[var(--color-surface-soft)] transition-colors">
              <div className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-semibold flex-shrink-0"
                style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                {user.name?.charAt(0)?.toUpperCase() || '?'}
              </div>
              <div className="text-left flex-1 min-w-0">
                <div className="text-sm text-[var(--color-ink)]">{user.name}</div>
                <div className="text-xs text-[var(--color-muted)]">{user.account}</div>
              </div>
              {selected.find(u => u.user_id === user.user_id) && (
                <span className="text-xs text-[var(--color-accent)] font-medium">已选</span>
              )}
            </button>
          ))}
        </div>

        <button onClick={handleCreate} disabled={!groupName.trim() || selected.length === 0 || creating}
          className="w-full h-[42px] rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed">
          {creating ? t('common.loading') : `${t('conversation.createGroup')} (${selected.length})`}
        </button>
      </div>
    </div>
  )
}
