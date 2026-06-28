import { ContentType } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
import { useSyncExternalStore } from 'react'
import { useTranslation } from 'react-i18next'

interface Props { msgId: number; convId: string }

export default function ReplyPreview({ msgId, convId }: Props) {
  const messages = useSyncExternalStore(chatStore.subscribe, () => chatStore.getMessages(convId))
  const replied = messages.find(m => m.msg_id === msgId)

  const { t } = useTranslation()
  if (!replied) {
    return (
      <div className="border-l-2 border-[var(--color-primary)] pl-2 mb-1 text-[11px] opacity-60">
        {t('chat.quoteMessage')}
      </div>
    )
  }

  const preview = () => {
    switch (replied.content_type) {
      case ContentType.Image: return t('chat.image')
      case ContentType.File: return t('chat.file')
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
