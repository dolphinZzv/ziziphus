import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { conversationService } from '@/services/conversation-service'
import { uiStore } from '@/stores/ui-store'
import type { ConvListItem } from '@/types/conversation'
import { X, Search, UserPlus } from 'lucide-react'
import { cn } from '@/lib/cn'

interface Props { onClose: () => void }

export default function JoinGroupDialog({ onClose }: Props) {
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<ConvListItem[]>([])
  const [searching, setSearching] = useState(false)

  const handleSearch = async () => {
    if (!query.trim()) return
    setSearching(true)
    try { setResults(await conversationService.searchGroups(query.trim())) } catch {}
    setSearching(false)
  }

  const handleJoin = async (convId: string) => {
    try { await conversationService.requestJoin(convId); uiStore.closeSheet(); navigate(`/chat/${convId}`) } catch {}
  }

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[400px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6 flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">加入群组</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]"><X size={16} /></button>
        </div>

        <div className="flex gap-2 mb-4">
          <input type="text" value={query} onChange={e => setQuery(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); handleSearch() } }}
            placeholder="搜索群组..." className={cn(inputClass, 'flex-1')} />
          <button onClick={handleSearch} disabled={searching}
            className="h-[42px] px-4 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors disabled:opacity-40">
            <Search size={16} />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto space-y-0.5">
          {results.map(group => (
            <div key={group.conv_id} className="flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-[var(--color-surface-soft)]">
              <div className="w-9 h-9 rounded-xl flex items-center justify-center text-white text-sm font-semibold flex-shrink-0"
                style={{ background: 'linear-gradient(135deg, var(--color-accent), #34D399)' }}>
                {group.name?.charAt(0)?.toUpperCase() || 'G'}
              </div>
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium text-[var(--color-ink)]">{group.name}</div>
                {group.headline && <div className="text-[11px] text-[var(--color-muted)] truncate">{group.headline}</div>}
              </div>
              {/* Secondary button */}
              <button onClick={() => handleJoin(group.conv_id)}
                className="px-3 py-1.5 rounded-xl border border-[var(--color-primary)] text-[var(--color-primary)] text-xs font-medium hover:bg-[var(--color-primary)]/5 transition-colors">
                加入
              </button>
            </div>
          ))}
          {query && !searching && results.length === 0 && (
            <p className="text-sm text-[var(--color-muted)] text-center py-8">未找到群组</p>
          )}
        </div>
      </div>
    </div>
  )
}
