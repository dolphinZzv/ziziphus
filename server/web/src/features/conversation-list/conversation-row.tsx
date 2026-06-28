import { cn } from '@/lib/cn'
import { formatMessageTime } from '@/lib/time'
import { avatarUrl } from '@/lib/file'
import { ConvType, type ConvListItem } from '@/types/conversation'
import { UserType } from '@/types/user'
import { ContentType } from '@/types/message'
import { conversationStore } from '@/stores/conversation-store'
import { BellOff, Cpu, Pin, PinOff, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

interface Props { conversation: ConvListItem; isSelected: boolean; onClick: () => void }

export default function ConversationRow({ conversation, isSelected, onClick }: Props) {
  const { t } = useTranslation()
  const { name, avatar, type, last_message, last_msg_at, unread_count, mention_me, mute, partner_type, pinned } = conversation

  const handlePin = (e: React.MouseEvent) => {
    e.stopPropagation()
    pinned ? conversationStore.unpin(conversation.conv_id) : conversationStore.pin(conversation.conv_id)
  }

  const lm = last_message
  const getPreview = (): string => {
    if (!lm?.body && lm?.body !== '') return ''
    if (lm?.body === '') return ''
    switch (lm?.content_type) {
      case ContentType.Text: return lm.body
      case ContentType.Image: return t('chat.image')
      case ContentType.File: return t('chat.file')
      case ContentType.AgentTimeline:
        try { const p = JSON.parse(lm.body); return p.entries?.at(-1)?.content || p.entries?.at(-1)?.title || t('conversation.agentResult') } catch { return t('conversation.agentResult') }
      case ContentType.System: return lm.body
      case ContentType.Form:
        try { const f = JSON.parse(lm.body); if (f.type === 'contact_request' && f.from_user_name) return `${t('conversation.friendRequest')} · ${f.from_user_name}`; return t('conversation.form') } catch { return '[表单]' }
      case ContentType.FormResponse:
        try { const r = JSON.parse(lm.body); const name = r.responder_name || ''; return r.action === 'approve' ? `${t('conversation.approved')}${name ? ' · ' + name : ''}` : `${t('conversation.rejected')}${name ? ' · ' + name : ''}` } catch { return t('conversation.response') }
      case ContentType.Audio: return t('chat.audio')
      case ContentType.Video: return t('chat.video')
      default: return lm?.body || ''
    }
  }

  const isGroup = type === ConvType.Group
  const isSystem = type === ConvType.System
  const displayName = isSystem ? t('conversation.systemMessage') : name
  const senderLabel = lm?.sender_name ? `${lm.sender_name}: ` : ''
  const isAI = partner_type === UserType.Agent
  const initials = isSystem ? '系' : (name ? name.split(' ').map(s => s[0]).join('').slice(0, 2).toUpperCase() || name.charAt(0).toUpperCase() : '?')

  const previewText = getPreview()

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={e => { if (e.key === 'Enter') onClick() }}
      className={cn(
        'w-full flex items-center gap-3 px-4 h-[52px] text-left transition-colors cursor-pointer group relative overflow-visible',
        isSelected ? 'bg-[var(--color-primary)]/5' : 'hover:bg-[var(--color-surface-soft)]'
      )}
    >
      <div className="relative flex-shrink-0">
        {avatar ? (
          <img src={avatarUrl(avatar)} alt="" className="w-10 h-10 rounded-full object-cover" />
        ) : (
          <div className="w-10 h-10 rounded-full flex items-center justify-center text-white text-sm font-semibold"
            style={{ background: isAI ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : `linear-gradient(135deg, var(--color-primary), var(--color-muted))` }}>
            {initials}
          </div>
        )}
        {isAI && (
          <div className="absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full bg-purple-500 flex items-center justify-center">
            <Cpu size={10} className="text-white" />
          </div>
        )}
        {isGroup && (
          <div className="absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full bg-[var(--color-surface-card)] flex items-center justify-center">
            <Users size={9} className="text-[var(--color-muted)]" />
          </div>
        )}
      </div>

      <div className="flex-1 min-w-0 overflow-hidden">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1">
            {pinned && <Pin size={10} className="text-[var(--color-accent)] flex-shrink-0" />}
            <span className="text-[15px] font-semibold text-[var(--color-ink)] truncate">{displayName}</span>
          </div>
          <div className="flex items-center gap-1 flex-shrink-0 ml-1">
            <button onClick={handlePin}
              className="p-0.5 rounded hover:bg-[var(--color-hairline)] opacity-0 group-hover:opacity-100 transition-opacity"
              title={pinned ? t('common.unpin') : t('common.pin')}>
              {pinned ? <PinOff size={11} className="text-[var(--color-muted)]" /> : <Pin size={11} className="text-[var(--color-muted)]" />}
            </button>
            <span className="text-[11px] text-[var(--color-muted)]">{formatMessageTime(last_msg_at)}</span>
          </div>
        </div>
        <div className="flex items-center justify-between gap-2 mt-0.5">
          <span className="text-[13px] text-[var(--color-muted)] truncate leading-snug">
            {mute && <BellOff size={10} className="inline mr-1" />}
            <span className="text-[var(--color-muted-soft)]">{senderLabel}</span>
            {previewText || ''}
          </span>
          {unread_count > 0 ? (
            <span className="flex-shrink-0 min-w-[18px] h-[18px] rounded-sm bg-[var(--color-primary)] text-white text-[10px] font-semibold uppercase tracking-wider flex items-center justify-center px-1 ml-1">
              {unread_count > 99 ? '99+' : unread_count}
            </span>
          ) : mention_me ? (
            <span className="flex-shrink-0 h-[18px] rounded-sm bg-[var(--warning)]/15 text-[var(--warning)] text-[10px] font-semibold uppercase tracking-wider flex items-center justify-center px-1.5 ml-1">
              @你
            </span>
          ) : null}
        </div>
      </div>
    </div>
  )
}
