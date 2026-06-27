import { ContentType } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
import { useSyncExternalStore } from 'react'

interface Props { msgId: number; convId: string }

export default function ReplyPreview({ msgId, convId }: Props) {
  const messages = useSyncExternalStore(chatStore.subscribe, () => chatStore.getMessages(convId))
  const replied = messages.find(m => m.msg_id === msgId)

  if (!replied) {
    return (
      <div className="border-l-2 border-[var(--color-primary)] pl-2 mb-1 text-[11px] opacity-60">
        引用消息
      </div>
    )
  }

  const preview = () => {
    switch (replied.content_type) {
      case ContentType.Image: return '[图片]'
      case ContentType.File: return '[文件]'
      default: return replied.body?.slice(0, 50) || ''
    }
  }

  return (
    <div className="border-l-2 border-[var(--color-primary)] pl-2 mb-1 text-[11px] opacity-70">
      <div className="font-medium text-[var(--color-primary)]">{replied.sender_name}</div>
      <div className="truncate">{preview()}</div>
    </div>
  )
}
