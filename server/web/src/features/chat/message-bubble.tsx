import { useState, useEffect, useRef, useSyncExternalStore } from 'react'
import type { Message } from '@/types/message'
import { ContentType, MsgStatus } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
import { authStore } from '@/stores/auth-store'
import { userService } from '@/services/user-service'
import { formatTime } from '@/lib/time'
import { avatarUrl } from '@/lib/file'
import { cn } from '@/lib/cn'
import { Check, CheckCheck, Clock, AlertCircle, Copy, Reply, Cpu } from 'lucide-react'
import TextBubble from './text-bubble'
import ImageBubble from './image-bubble'
import FileBubble from './file-bubble'
import AgentTimelineBubble from './agent-timeline-bubble'
import FormBubble from './form-bubble'
import FormResponseBubble from './form-response-bubble'
import ReplyPreview from './reply-preview'
import UserCard from '@/components/user-card'
import { useTranslation } from 'react-i18next'

const senderCache = new Map<string, { avatar: string; type: number }>()

function useSenderInfo(userId: string): { avatar?: string; isAgent: boolean } {
  const [info, setInfo] = useState<{ avatar?: string; isAgent: boolean }>(
    senderCache.has(userId) ? { avatar: senderCache.get(userId)!.avatar, isAgent: senderCache.get(userId)!.type === 1 } : {}
  )
  useEffect(() => {
    if (senderCache.has(userId)) return
    userService.getUser(userId).then(u => {
      const entry = { avatar: u?.avatar || '', type: u?.type || 0 }
      senderCache.set(userId, entry)
      setInfo({ avatar: entry.avatar, isAgent: entry.type === 1 })
    }).catch(() => {})
  }, [userId])
  return info
}

interface Props { message: Message; isOwn: boolean; isGrouped: boolean }

export default function MessageBubble({ message, isOwn, isGrouped }: Props) {
  const [showMenu, setShowMenu] = useState(false)
  const [hoverUser, setHoverUser] = useState(false)
  const hoverTimer = useRef<ReturnType<typeof setTimeout>>()
  const me = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)

  const handleMouseEnter = () => {
    if (hoverTimer.current) clearTimeout(hoverTimer.current)
    setHoverUser(true)
  }
  const handleMouseLeave = () => {
    hoverTimer.current = setTimeout(() => setHoverUser(false), 200)
  }

  const handleReply = () => { chatStore.setReplyTo(message.conv_id, message); setShowMenu(false) }
  const handleCopy = () => { navigator.clipboard.writeText(message.body); setShowMenu(false) }
  const handleRetry = () => { chatStore.sendMessage(message.conv_id, message.body, message.content_type, message.reply_to, message.mention) }

  const renderContent = () => {
    switch (message.content_type) {
      case ContentType.Text: case ContentType.System: return <TextBubble text={message.body} />
      case ContentType.Image: return <ImageBubble body={message.body} msgId={message.msg_id} />
      case ContentType.File: return <FileBubble body={message.body} />
      case ContentType.AgentTimeline: return <AgentTimelineBubble body={message.body} />
      case ContentType.Form: return <FormBubble body={message.body} msgId={message.msg_id} convId={message.conv_id} senderId={message.sender_id} />
      case ContentType.FormResponse: return <FormResponseBubble body={message.body} />
      default: return <TextBubble text={message.body || '[不支持的消息类型]'} />
    }
  }

  const statusIcon = () => {
    if (!isOwn) return null
    const cls = 'text-[var(--color-muted)]'
    switch (message.status) {
      case MsgStatus.Sending: return <Clock size={12} className={cls} />
      case MsgStatus.Sent: return <Check size={12} className={cls} />
      case MsgStatus.Delivered: return <CheckCheck size={12} className={cls} />
      case MsgStatus.Read: return <CheckCheck size={12} className="text-[var(--color-accent)]" />
      default: return null
    }
  }

  const isFailed = isOwn && message.msg_id === 0 && message.status === MsgStatus.Sending
  const senderInitials = message.sender_name?.charAt(0)?.toUpperCase() || '?'
  const myInitials = me?.name?.charAt(0)?.toUpperCase() || '?'

  const AvatarDot = ({ name, avatar, userId, clickable, isAgent }: { name: string; avatar?: string; userId: string; clickable: boolean; isAgent?: boolean }) => (
    clickable ? (
      <button onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave} className="flex-shrink-0 self-start mt-1">
        <div className="relative">
          {avatar ? (
            <img src={avatarUrl(avatar)} alt="" className="w-8 h-8 rounded-full object-cover hover:ring-2 hover:ring-[var(--color-primary)]/30 transition-all" />
          ) : (
            <div className="w-8 h-8 rounded-full bg-[var(--color-muted)]/20 flex items-center justify-center text-xs text-[var(--color-ink)] font-semibold overflow-hidden hover:ring-2 hover:ring-[var(--color-primary)]/30 transition-all">{name}</div>
          )}
          {isAgent && <span className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-purple-500 flex items-center justify-center border border-[var(--color-surface-card)]"><Cpu size={8} className="text-white" /></span>}
        </div>
      </button>
    ) : (
      <div className="flex-shrink-0 self-start mt-1">
        <div className="relative">
          {avatar ? (
            <img src={avatarUrl(avatar)} alt="" className="w-8 h-8 rounded-full object-cover" />
          ) : (
            <div className="w-8 h-8 rounded-full bg-[var(--color-muted)]/20 flex items-center justify-center text-xs text-[var(--color-ink)] font-semibold overflow-hidden">{name}</div>
          )}
          {isAgent && <span className="absolute -bottom-0.5 -right-0.5 w-3.5 h-3.5 rounded-full bg-purple-500 flex items-center justify-center border border-[var(--color-surface-card)]"><Cpu size={8} className="text-white" /></span>}
        </div>
      </div>
    )
  )

  return (
    <div className={cn('flex gap-2 group m-1', isOwn ? 'flex-row-reverse' : 'flex-row')}>
      {/* Avatar */}
      {!isGrouped && isOwn && (
        <AvatarDot name={myInitials} avatar={me?.avatar} userId={me?.user_id || ''} clickable={false} isAgent={me?.type === 1} />
      )}
      {!isGrouped && !isOwn && (
        <AvatarDot name={senderInitials} userId={message.sender_id} clickable={true} isAgent={useSenderInfo(message.sender_id).isAgent} />
      )}
      {isGrouped && <div className="w-8 flex-shrink-0 self-start" />}

      <div className={cn('max-w-[85%]', isOwn ? 'items-end' : 'items-start')}>
        {!isGrouped && !isOwn && message.sender_name && (
          <button
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            className="text-[11px] text-[var(--color-muted)] hover:text-[var(--color-ink)] mb-1 ml-1 transition-colors cursor-pointer"
          >
            {message.sender_name}
          </button>
        )}

        <div className="relative" onContextMenu={e => { e.preventDefault(); setShowMenu(true) }}>
          {message.reply_to > 0 && <ReplyPreview msgId={message.reply_to} convId={message.conv_id} />}

          <div className={cn(
            'relative rounded-lg px-3 py-1.5 text-sm',
            isOwn ? 'bg-[var(--bubble-self)] text-[var(--bubble-self-text)]' : 'bg-[var(--bubble-other)] text-[var(--bubble-other-text)] border border-[var(--bubble-other-border)]',
            isFailed && 'border-[var(--destructive)]'
          )}
            style={{ boxShadow: 'var(--shadow-sm)' }}
            onClick={() => { if (isFailed) handleRetry() }}>
            {renderContent()}
            <div className={cn('flex items-center gap-1 mt-1', !isOwn ? 'justify-end' : 'justify-start')}>
              <span className="text-[10px] opacity-60">{formatTime(message.timestamp)}</span>
              {statusIcon()}
            </div>
          </div>

          {isFailed && (
            <button onClick={handleRetry} className="absolute -bottom-6 right-0 text-[11px] text-[var(--destructive)] flex items-center gap-1">
              <AlertCircle size={12} /> 重试
            </button>
          )}

          {hoverUser && !isOwn && (
            <div
              className="absolute z-50"
              style={{ bottom: '100%', left: 0, marginBottom: 8 }}
              onMouseEnter={handleMouseEnter}
              onMouseLeave={handleMouseLeave}
            >
              <UserCard userId={message.sender_id} onClose={() => setHoverUser(false)} />
            </div>
          )}

          {showMenu && (
            <>
              <div className="fixed inset-0 z-40" onClick={() => setShowMenu(false)} />
              <div className="absolute z-50 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg py-1 min-w-[120px]"
                style={{ boxShadow: 'var(--shadow-md)', bottom: isOwn ? '100%' : 'auto', top: !isOwn ? '100%' : 'auto', right: isOwn ? 0 : 'auto', left: !isOwn ? 0 : 'auto' }}>
                <button onClick={handleCopy} className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]"><Copy size={14} /> 复制</button>
                <button onClick={handleReply} className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]"><Reply size={14} /> 回复</button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}
