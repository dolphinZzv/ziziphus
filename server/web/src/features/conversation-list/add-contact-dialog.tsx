import { useState } from 'react'
import { contactRequestService } from '@/services/contact-request-service'
import { userService } from '@/services/user-service'
import { uiStore } from '@/stores/ui-store'
import type { User } from '@/types/user'
import { X, Search, UserPlus, Check } from 'lucide-react'

interface Props { onClose: () => void }

export default function AddContactDialog({ onClose }: Props) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<User[]>([])
  const [searching, setSearching] = useState(false)
  const [sentRequests, setSentRequests] = useState<Set<string>>(new Set())
  const [errors, setErrors] = useState<Record<string, string>>({})

  const handleSearch = async () => {
    if (!query.trim()) return
    setSearching(true)
    try { setResults(await userService.search(query.trim())) } catch {}
    setSearching(false)
  }

  const handleAdd = async (userId: string) => {
    setErrors(prev => { const n = { ...prev }; delete n[userId]; return n })
    try {
      await contactRequestService.send(userId)
      setSentRequests(prev => new Set(prev).add(userId))
    } catch (err: any) {
      setErrors(prev => ({ ...prev, [userId]: err?.message || '发送失败' }))
    }
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4" onClick={onClose}>
      <div className="w-full sm:w-[400px] max-h-[480px] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl p-6 flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">添加联系人</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="flex gap-2 mb-4">
          <input type="text" value={query} onChange={e => setQuery(e.target.value)}
            onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleSearch() } }}
            placeholder="搜索用户..." className={`${inputClass} flex-1`} />
          <button onClick={handleSearch} disabled={searching}
            className="h-[42px] px-4 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors disabled:opacity-40">
            <Search size={16} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto space-y-0.5">
          {results.map(user => {
            const sent = sentRequests.has(user.user_id)
            return (
            <div key={user.user_id} className="flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-[var(--color-surface-soft)] transition-colors">
              <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-sm font-semibold flex-shrink-0"
                style={{ background: user.primary_color ? `linear-gradient(135deg, ${user.primary_color}, ${user.secondary_color || user.primary_color})` : 'var(--color-primary)' }}>
                {user.name?.charAt(0)?.toUpperCase() || '?'}
              </div>
              <div className="text-left min-w-0 flex-1">
                <div className="text-sm font-medium text-[var(--color-ink)]">{user.name}</div>
                <div className="text-xs text-[var(--color-muted)]">{user.account}</div>
                {errors[user.user_id] && <div className="text-[10px] text-[var(--destructive)] mt-0.5">{errors[user.user_id]}</div>}
              </div>
              {sent ? (
                <span className="flex items-center gap-1 px-2 py-1 rounded-lg text-[10px] text-[var(--success)] bg-[var(--success)]/5">
                  <Check size={11} /> 已发送
                </span>
              ) : (
                <button onClick={() => handleAdd(user.user_id)}
                  className="px-3 py-1.5 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white text-xs font-medium transition-colors">
                  添加
                </button>
              )}
            </div>
          )})}
          {query && !searching && results.length === 0 && (
            <p className="text-sm text-[var(--color-muted)] text-center py-8">未找到用户</p>
          )}
        </div>
      </div>
    </div>
  )
}
