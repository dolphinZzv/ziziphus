import { useSyncExternalStore } from 'react'
import { useNavigate } from 'react-router-dom'
import { conversationStore } from '@/stores/conversation-store'
import { uiStore } from '@/stores/ui-store'
import ConversationRow from './conversation-row'
import { SkeletonRow } from '@/components/skeleton'

export default function ConversationList() {
  const navigate = useNavigate()
  const conversations = useSyncExternalStore(conversationStore.subscribe, () => conversationStore.state.conversations)
  const isLoading = useSyncExternalStore(conversationStore.subscribe, () => conversationStore.state.isLoading)
  const selectedConvId = useSyncExternalStore(uiStore.subscribe, () => uiStore.state.selectedConvId)

  const handleSelect = (convId: string) => {
    uiStore.selectConversation(convId)
    navigate(`/chat/${convId}`)
  }

  if (isLoading && conversations.length === 0) {
    return (
      <div className="py-2">
        {Array.from({ length: 6 }).map((_, i) => <SkeletonRow key={i} />)}
      </div>
    )
  }

  if (conversations.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-sm text-[var(--color-muted)] gap-2">
        <span className="text-3xl">💬</span>
        <p>暂无会话</p>
        <p className="text-xs">点击 + 创建新聊天</p>
      </div>
    )
  }

  return (
    <div className="h-full overflow-y-auto">
      {conversations.map(conv => (
        <ConversationRow
          key={conv.conv_id}
          conversation={conv}
          isSelected={conv.conv_id === selectedConvId}
          onClick={() => handleSelect(conv.conv_id)}
        />
      ))}
    </div>
  )
}
