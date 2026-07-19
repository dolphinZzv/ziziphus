import { useEffect, useState } from 'react'
import { cn } from '@/lib/cn'
import { formatMessageTime } from '@/lib/time'
import { avatarUrl } from '@/lib/file'
import { ConvType, type ConvListItem } from '@/types/conversation'
import { UserType } from '@/types/user'
import { ContentType } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
import { BellOff, Cpu, Pin, Users, MessageCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

/** Auto-updating relative time label. Re-renders every 30s so "刚刚" → "X分钟前" → "HH:mm". */
function RelativeTime({ ts }: { ts: number }) {
  const [, tick] = useState(0)
  useEffect(() => {
    const id = setInterval(() => tick(n => n + 1), 30_000)
    return () => clearInterval(id)
  }, [])
  return <>{formatMessageTime(ts)}</>
}

interface Props { conversation: ConvListItem; isSelected: boolean; onClick: () => void }

export default function ConversationRow({ conversation, isSelected, onClick }: Props) {
  const { t } = useTranslation()
  const { name, avatar, type, last_message, last_msg_at, unread_count, mention_me, mute, partner_type, pinned } = conversation

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

  const draft = chatStore.getDraft(conversation.conv_id)
  const rawPreview = getPreview()
  const hasDraft = !!(draft && draft.trim())

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={e => { if (e.key === 'Enter') onClick() }}
      className={cn(
        'w-full flex items-center gap-2.5 px-4 h-[48px] text-left transition-colors cursor-pointer group relative overflow-visible btn-press',
        isSelected ? 'bg-[var(--color-primary)]/5' : 'hover:bg-[var(--color-surface-soft)]'
      )}
    >
      <div className="relative flex-shrink-0">
        {avatar ? (
          <img loading="lazy" decoding="async" src={avatarUrl(avatar)} alt="" className="w-9 h-9 rounded-full object-cover" />
        ) : isSystem ? (
          <div className="w-9 h-9 rounded-full flex items-center justify-center"
            style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
            <MessageCircle size={18} className="text-white" />
          </div>
        ) : (
          <div className="w-9 h-9 rounded-full flex items-center justify-center text-white text-xs font-semibold"
            style={{ background: isAI ? 'linear-gradient(135deg, #8B5CF6, #A78BFA)' : `linear-gradient(135deg, var(--color-primary), var(--color-muted))` }}>
            {initials}
          </div>
        )}
        {isAI && (
          <div className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-purple-500 flex items-center justify-center">
            <Cpu size={8} className="text-white" />
          </div>
        )}
        {isGroup && (
          <div className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-[var(--color-surface-card)] flex items-center justify-center">
            <Users size={8} className="text-[var(--color-muted)]" />
          </div>
        )}
      </div>

      <div className="flex-1 min-w-0 overflow-hidden">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-1 min-w-0">
            {pinned && <Pin size={10} className="text-[var(--color-accent)] flex-shrink-0" />}
            <span className="text-sm font-medium text-[var(--color-ink)] truncate">{displayName}</span>
          </div>
          <span className="text-[10px] text-[var(--color-muted-soft)] flex-shrink-0"><RelativeTime ts={last_msg_at} /></span>
        </div>
        <div className="flex items-center justify-between gap-2 mt-0.5">
          <span className="text-[12px] text-[var(--color-muted)] truncate leading-snug">
            {mute && <BellOff size={10} className="inline mr-1" />}
            {hasDraft ? (
              <><span className="text-[var(--color-accent)] text-[10px] font-medium mr-1">[{t('chat.draft')}]</span>{draft}</>
            ) : (
              <><span className="text-[var(--color-muted-soft)]">{senderLabel}</span>{rawPreview || ''}</>
            )}
          </span>
          {unread_count > 0 ? (
            <span className="flex-shrink-0 min-w-[16px] h-[16px] rounded-sm bg-[var(--color-primary)] text-white text-[9px] font-semibold flex items-center justify-center px-1 ml-1">
              {unread_count > 99 ? '99+' : unread_count}
            </span>
          ) : mention_me ? (
            <span className="flex-shrink-0 h-[16px] rounded-sm bg-[var(--warning)]/15 text-[var(--warning)] text-[9px] font-semibold flex items-center justify-center px-1 ml-1">
              @你
            </span>
          ) : null}
        </div>
      </div>
    </div>
  )
}
