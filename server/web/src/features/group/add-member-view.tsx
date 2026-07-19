import { useState } from 'react'
import { userService } from '@/services/user-service'
import { avatarUrl } from '@/lib/file'
import type { User } from '@/types/user'
import { X, Search, Plus, ArrowLeft } from 'lucide-react'
import { useIsMobile } from '@/hooks/use-breakpoint'

interface Props { convId: string; onClose: () => void; onAdded: (user: User) => void; excludeIds: Set<string> }

export default function AddMemberView({ convId: _convId, onClose, onAdded, excludeIds }: Props) { const isMobile=useIsMobile()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [searching, setSearching] = useState(false)
  const [addedIds, setAddedIds] = useState<Set<string>>(new Set())

  const handleSearch = async () => {
    if (!query.trim()) return
    setSearching(true)
    try {
      const users = await userService.search(query.trim())
      setResults(users.filter(x => !excludeIds.has(x.user_id)))
    } catch (e) { console.error(e) }
    setSearching(false)
  }

  const handleAdd = (user: User) => {
    onAdded(user)
    setAddedIds(prev => new Set(prev).add(user.user_id))
  }

  const inputClass = 'w-full h-10 px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[380px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] rounded-none sm:rounded-xl overflow-hidden flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>

        <div className="flex items-center justify-between px-5 py-4">
          <h3 className="font-headline text-base font-semibold text-[var(--color-ink)]">添加成员</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">{isMobile ? <ArrowLeft size={18} /> : <X size={16} />}</button>
        </div>

        <div className="p-4">
          <div className="flex gap-2 mb-3">
            <input type="text" value={query} onChange={e => setQuery(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleSearch() } }}
              placeholder="搜索用户..." className={`${inputClass} flex-1`} />
            <button onClick={handleSearch} disabled={searching}
              className="h-10 px-4 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors disabled:opacity-40">
              <Search size={16} />
            </button>
          </div>

          <div className="max-h-[100dvh] sm:max-h-[calc(100vh-80px)] overflow-y-auto space-y-0.5">
            {results.map(user => {
              const added = addedIds.has(user.user_id)
              return (
                <div key={user.user_id} className="flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-[var(--color-surface-soft)] transition-colors">
                  {user.avatar ? (
                    <img loading="lazy" decoding="async" src={avatarUrl(user.avatar, 64)} alt="" className="w-9 h-9 rounded-full object-cover flex-shrink-0" />
                  ) : (
                    <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-semibold flex-shrink-0"
                      style={{ background: user.primary_color ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})` : 'var(--color-primary)' }}>
                      {user.name?.charAt(0)?.toUpperCase() || '?'}
                    </div>
                  )}
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-[var(--color-ink)]">{user.name}</div>
                    <div className="text-[11px] text-[var(--color-muted)]">@{user.account || user.user_id}</div>
                  </div>
                  {added ? (
                    <span className="text-[11px] text-[var(--success)] font-medium px-2">已添加</span>
                  ) : (
                    <button onClick={() => handleAdd(user)}
                      className="p-1.5 rounded-xl bg-[var(--color-primary)]/10 hover:bg-[var(--color-primary)]/20 text-[var(--color-primary)] transition-colors">
                      <Plus size={16} />
                    </button>
                  )}
                </div>
              )
            })}
            {query && !searching && results.length === 0 && (
              <p className="text-sm text-[var(--color-muted)] text-center py-8">未找到用户</p>
            )}
            {!query && (
              <p className="text-sm text-[var(--color-muted)] text-center py-8">搜索用户以添加</p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
