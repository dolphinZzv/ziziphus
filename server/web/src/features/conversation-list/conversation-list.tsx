import { useMemo, useState } from 'react'
import { useSyncExternalStore } from 'react'
import { useNavigate } from 'react-router-dom'
import { useIsMobile } from '@/hooks/use-breakpoint'
import { conversationStore } from '@/stores/conversation-store'
import { uiStore } from '@/stores/ui-store'
import ConversationRow from './conversation-row'
import { SkeletonRow } from '@/components/skeleton'
import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export default function ConversationList() {
  const navigate = useNavigate()
  const conversations = useSyncExternalStore(conversationStore.subscribe, () => conversationStore.state.conversations)
  const isLoading = useSyncExternalStore(conversationStore.subscribe, () => conversationStore.state.isLoading)
  const selectedConvId = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.selectedConvId)
  const { t } = useTranslation()
  const [query, setQuery] = useState('')

  const filtered = useMemo(() => {
    if (!query.trim()) return conversations
    const q = query.trim().toLowerCase()
    return conversations.filter(c => c.name.toLowerCase().includes(q))
  }, [conversations, query])

  const handleSelect = (convId: string) => {
    uiStore.selectConversation(convId)
    navigate(`/conversations/${convId}`)
  }

  return (
    <div className="h-full flex flex-col">
      {conversations.length >= 10 && (
        <div className="px-3 pt-3 pb-2">
          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-muted)]" />
            <input
              type="text"
              value={query}
              onChange={e => setQuery(e.target.value)}
              placeholder={t('conversation.searchPlaceholder')}
              className="w-full h-9 pl-8 pr-3 rounded-xl bg-[var(--color-surface-soft)] text-sm text-[var(--color-ink)] placeholder:text-[var(--color-muted-soft)] border border-transparent focus:border-[var(--color-primary)]/30 focus:outline-none"
            />
          </div>
        </div>
      )}

      <div className="flex-1 overflow-y-auto">
        {isLoading && conversations.length === 0 ? (
          <div className="py-2">
            {Array.from({ length: 6 }).map((_, i) => <SkeletonRow key={i} />)}
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-sm text-[var(--color-muted)] gap-2">
            <span className="text-3xl">{query ? '🔍' : '💬'}</span>
            <p>{query ? t('conversation.noMatch') : t('conversation.noConversations')}</p>
            {!query && <p className="text-xs">{t('conversation.startChat')}</p>}
          </div>
        ) : (
          filtered.map(conv => (
            <ConversationRow
              key={conv.conv_id}
              conversation={conv}
              isSelected={conv.conv_id === selectedConvId}
              onClick={() => handleSelect(conv.conv_id)}
            />
          ))
        )}
      </div>
    </div>
  )
}
