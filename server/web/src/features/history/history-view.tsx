import { useEffect, useRef, useState, useCallback } from 'react'
import { messageService } from '@/services/message-service'
import type { Message } from '@/types/message'
import { ContentType } from '@/types/message'
import { X, Search, Loader2, ArrowLeft } from 'lucide-react'
import { format } from 'date-fns'
import { useIsMobile } from '@/hooks/use-breakpoint'

interface Props { convId: string; onClose: () => void }

export default function HistoryView({ convId, onClose }: Props) { const isMobile=useIsMobile()
  const [messages, setMessages] = useState<Message[]>([])
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const scrollRef = useRef<HTMLDivElement>(null)

  const load = async (kw: string, append = false) => {
    if (append) {
      setLoadingMore(true)
    } else {
      setLoading(true)
      setMessages([])
      setHasMore(true)
    }
    try {
      const before = append ? messages[messages.length - 1]?.msg_id : undefined
      const data = await messageService.getHistory(convId, {
        limit: 30,
        keyword: kw || undefined,
        before: before && before > 0 ? before : undefined,
      })
      const arr = Array.isArray(data) ? data : (data as any).items || []
      if (append) {
        setMessages(prev => [...prev, ...arr])
      } else {
        setMessages(arr)
      }
      setHasMore(arr.length >= 30)
    } catch {}
    setLoading(false)
    setLoadingMore(false)
  }

  useEffect(() => { load('') }, [convId])

  const search = () => load(keyword)

  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el || loadingMore || !hasMore || loading) return
    // Near bottom → load more (older messages)
    if (el.scrollHeight - el.scrollTop - el.clientHeight < 60) {
      load(keyword, true)
    }
  }, [keyword, messages, loadingMore, hasMore, loading])

  const inputClass = 'w-full h-[42px] px-3.5 rounded-xl bg-[var(--color-surface-card)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-[var(--color-hairline)] hover:border-[var(--color-primary)] focus:outline-none focus:border-[var(--color-primary)] focus:ring-2 focus:ring-[var(--color-primary)]/10'

  return (
    <div className="fixed inset-0 z-50 flex sm:items-center sm:justify-center bg-black/30" onClick={onClose}>
      <div className="w-full sm:w-[480px] h-full sm:h-auto max-h-[100dvh] sm:max-h-[calc(100vh-80px)] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-none sm:rounded-xl p-6 flex flex-col"
        style={{ boxShadow: 'var(--shadow-lg)' }} onClick={e => e.stopPropagation()}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-headline text-lg font-semibold text-[var(--color-ink)]">消息记录</h3>
          <button onClick={onClose} className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)]">{isMobile ? <ArrowLeft size={18} /> : <X size={16} />}</button>
        </div>

        <div className="flex gap-2 mb-4">
          <input type="text" value={keyword} onChange={e => setKeyword(e.target.value)} onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); search() } }}
            placeholder="搜索关键词..." className={`${inputClass} flex-1`} />
          <button onClick={search} className="h-[42px] px-4 rounded-xl bg-[var(--color-primary)] hover:bg-[var(--color-primary-hover)] text-white transition-colors"><Search size={16} /></button>
        </div>

        <div ref={scrollRef} onScroll={handleScroll} className="flex-1 overflow-y-auto space-y-1">
          {loading && <p className="text-xs text-[var(--color-muted)] text-center py-4"><Loader2 size={14} className="inline animate-spin mr-1" />加载中...</p>}
          {messages.map(msg => (
            <div key={msg.msg_id} className="px-3 py-2.5 rounded-xl hover:bg-[var(--color-surface-soft)]">
              <div className="flex items-center justify-between mb-0.5">
                <span className="text-xs font-medium text-[var(--color-ink)]">{msg.sender_name || msg.sender_id}</span>
                <span className="text-[10px] text-[var(--color-muted)]">{msg.timestamp ? format(new Date(msg.timestamp), 'MM/dd HH:mm') : ''}</span>
              </div>
              <div className="text-xs text-[var(--color-body)] line-clamp-2">
                {msg.content_type === ContentType.Image ? '[图片]' : msg.content_type === ContentType.File ? '[文件]' : msg.content_type === ContentType.AgentTimeline ? '[Agent]' : msg.body?.slice(0, 100)}
              </div>
            </div>
          ))}
          {loadingMore && <p className="text-xs text-[var(--color-muted)] text-center py-2"><Loader2 size={14} className="inline animate-spin mr-1" />加载更多...</p>}
          {!loading && messages.length === 0 && <p className="text-sm text-[var(--color-muted)] text-center py-8">暂无消息</p>}
        </div>
      </div>
    </div>
  )
}
