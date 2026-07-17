import { useState, useEffect, useRef, useSyncExternalStore } from 'react'
import { createPortal } from 'react-dom'
import type { Message } from '@/types/message'
import { ContentType, MsgStatus } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
import { authStore } from '@/stores/auth-store'
import { userService } from '@/services/user-service'
import { formatTime } from '@/lib/time'
import { avatarUrl } from '@/lib/file'
import { cn } from '@/lib/cn'
import { Check, CheckCheck, Clock, AlertCircle, Copy, Reply, Cpu, PenLine, Trash2, MoreHorizontal } from 'lucide-react'
import TextBubble from './text-bubble'
import ImageBubble from './image-bubble'
import FileBubble from './file-bubble'
import AgentTimelineBubble from './agent-timeline-bubble'
import FormBubble from './form-bubble'
import FormResponseBubble from './form-response-bubble'
import ReplyPreview from './reply-preview'
import UserCard from '@/components/user-card'
import { useTranslation } from 'react-i18next'

const CACHE_TTL = 5 * 60 * 1000 // 5 minutes
const senderCache = new Map<string, { avatar: string; type: number; ts: number }>()

function useSenderInfo(userId: string, isOwn: boolean): { avatar?: string; isAgent: boolean } {
  // For own messages, use local user info directly — no API call needed
  const me = isOwn ? authStore.state.user : null

  const cached = senderCache.get(userId)
  const initial = cached && (Date.now() - cached.ts) < CACHE_TTL
    ? { avatar: cached.avatar, isAgent: cached.type === 1 }
    : {}
  const [info, setInfo] = useState<{ avatar?: string; isAgent: boolean }>(initial)
  useEffect(() => {
    // Skip sender lookup if no valid userId
    if (!userId || userId.startsWith('webhook:')) return
    const cached = senderCache.get(userId)
    if (cached && (Date.now() - cached.ts) < CACHE_TTL) {
      if (!info.avatar && cached.avatar) setInfo({ avatar: cached.avatar, isAgent: cached.type === 1 })
      return
    }
    userService.getUser(userId).then(u => {
      const entry = { avatar: u?.avatar || '', type: u?.type || 0, ts: Date.now() }
      senderCache.set(userId, entry)
      setInfo({ avatar: entry.avatar, isAgent: entry.type === 1 })
    }).catch(() => {})
  }, [userId])

  if (me) return { avatar: me.avatar || undefined, isAgent: me.type === 1 }
  return info
}

interface Props {
  message: Message
  isOwn: boolean
  isGrouped: boolean
  highlight?: string
  isSearchMatch?: boolean
  isCurrentSearchMatch?: boolean
}

export default function MessageBubble({ message, isOwn, isGrouped, highlight, isSearchMatch, isCurrentSearchMatch }: Props) {
  const { t } = useTranslation()
  // Always call hooks at top level, even for centered messages
  const _senderInfo = useSenderInfo(message.sender_id, isOwn)
  const [showMenu, setShowMenu] = useState(false)
  const [menuPos, setMenuPos] = useState<{ x: number; y: number } | null>(null)
  const [hoverUser, setHoverUser] = useState(false)
  const [editing, setEditing] = useState(false)
  const [editText, setEditText] = useState('')
  const isComposingRef = useRef(false)
  const hoverTimer = useRef<ReturnType<typeof setTimeout>>()
  const menuBtnRef = useRef<HTMLButtonElement>(null)
  const avatarRef = useRef<HTMLButtonElement>(null)
  const [userCardPos, setUserCardPos] = useState<{ x: number; y: number } | null>(null)
  const me = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)

  // --- Feature 3: Read receipts ---
  const [readReceipts, setReadReceipts] = useState<Array<{ user_id: string; name?: string; avatar?: string }> | null>(null)
  const [showReadBy, setShowReadBy] = useState(false)
  const [tooltipPos, setTooltipPos] = useState({ top: 0, left: 0 })
  const readTimer = useRef<ReturnType<typeof setTimeout>>()
  const statusRef = useRef<HTMLDivElement>(null)

  const fetchReadReceipts = async () => {
    if (readReceipts) return
    try {
      const { api } = await import('@/services/api-client')
      const data = await api.request<Array<{ user_id: string }>>(`/api/v1/messages/${message.msg_id}/receipts`)
      const ids = data.filter(r => r.user_id !== message.sender_id).map(r => r.user_id)
      if (ids.length === 0) { setReadReceipts([]); return }
      const { userService } = await import('@/services/user-service')
      const users = await userService.batchGet(ids)
      setReadReceipts(ids.map(id => ({
        user_id: id,
        name: users[id]?.name || id,
        avatar: users[id]?.avatar,
      })))
    } catch { setReadReceipts([]) }
  }

  const handleStatusHover = () => {
    if (!isOwn || message.status !== MsgStatus.Read || message.msg_id <= 0) return
    // Position tooltip above the status icon
    if (statusRef.current) {
      const rect = statusRef.current.getBoundingClientRect()
      setTooltipPos({ top: rect.top - 8, left: rect.left + rect.width / 2 })
    }
    readTimer.current = setTimeout(() => {
      fetchReadReceipts()
      setShowReadBy(true)
    }, 400)
  }

  const handleStatusLeave = () => {
    if (readTimer.current) clearTimeout(readTimer.current)
    setTimeout(() => setShowReadBy(false), 200)
  }

  useEffect(() => {
    return () => { if (readTimer.current) clearTimeout(readTimer.current) }
  }, [])
  // --- End read receipts ---

  const msgTimestampMs = message.timestamp > 1e12 ? message.timestamp : message.timestamp * 1000
  const canEdit = isOwn && (message.content_type === ContentType.Text || message.content_type === ContentType.Edit) && (Date.now() - msgTimestampMs) < 300000 // 5 min
  const canRecall = isOwn && (Date.now() - msgTimestampMs) < 120000 // 2 min

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

  const handleEdit = () => { setEditText(message.body); setEditing(true); setShowMenu(false) }
  const handleSaveEdit = () => {
    if (!editText.trim()) return
    chatStore.editMessage(message.conv_id, message.msg_id, editText.trim())
    setEditing(false)
  }
  const handleCancelEdit = () => { setEditing(false) }

  const handleRecall = () => {
    if (!confirm(t('chat.recallConfirm'))) return
    chatStore.recallMessage(message.conv_id, message.msg_id)
    setShowMenu(false)
  }

  const renderContent = () => {
    switch (message.content_type) {
      case ContentType.Recall: return <span className="italic opacity-50 text-xs">{t('chat.recalled')}</span>
      case ContentType.Edit:
      case ContentType.Text: case ContentType.System: return <TextBubble text={message.body} highlight={highlight} />
      case ContentType.Image: return <ImageBubble body={message.body} msgId={message.msg_id} />
      case ContentType.File: return <FileBubble body={message.body} />
      case ContentType.AgentTimeline: return <AgentTimelineBubble body={message.body} convId={message.conv_id} />
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

  const AvatarDot = ({ name, avatar, userId: _userId, clickable, isAgent }: { name: string; avatar?: string; userId: string; clickable: boolean; isAgent?: boolean }) => (
    clickable ? (
      <button
        ref={avatarRef}
        onMouseEnter={() => { const r = avatarRef.current?.getBoundingClientRect(); if (r) setUserCardPos({ x: r.left, y: r.bottom + 4 }); handleMouseEnter() }}
        onMouseLeave={handleMouseLeave} className="flex-shrink-0 self-start mt-1">
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

  // System messages and recalled messages — centered, no avatar, distinct style
  const isCentered = message.content_type === ContentType.System || message.content_type === ContentType.Recall

  if (isCentered) {
    return (
      <div className="flex justify-center my-2">
        <span className="inline-block px-3 py-1 rounded-full bg-[var(--color-surface-soft)] text-[11px] text-[var(--color-muted)] max-w-[85%] text-center">
          {message.content_type === ContentType.Recall ? String(t('chat.recalled')) : message.body}
        </span>
      </div>
    )
  }

  return (
    <div className={cn(
      'flex gap-2 group m-1',
      isOwn ? 'justify-end' : 'justify-start',
      isCurrentSearchMatch && 'rounded-xl ring-2 ring-[var(--color-primary)] ring-offset-2 ring-offset-[var(--color-surface-card)]'
    )}>
      {/* Avatar */}
      {!isGrouped && !isOwn && (
        <AvatarDot name={senderInitials} avatar={_senderInfo.avatar} userId={message.sender_id} clickable={message.content_type !== ContentType.System && !message.sender_id.startsWith('webhook:')} isAgent={_senderInfo.isAgent} />
      )}
      {isGrouped && !isOwn && <div className="w-8 flex-shrink-0 self-start" />}

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

        <div className="relative group/msg" onContextMenu={e => { e.preventDefault(); setShowMenu(!showMenu) }}>
          {message.reply_to > 0 && <ReplyPreview msgId={message.reply_to} convId={message.conv_id} />}

          {/* ... button — shown on hover */}
          {!isFailed && !editing && (
            <button ref={menuBtnRef} onClick={() => {
              const rect = menuBtnRef.current?.getBoundingClientRect()
              if (rect) setMenuPos({ x: rect.left, y: rect.bottom + 4 })
              setShowMenu(true)
            }}
              className={`absolute z-10 opacity-0 group-hover/msg:opacity-100 transition-opacity p-0.5 rounded hover:bg-black/10 ${isOwn ? '-left-6 bottom-0' : '-right-6 bottom-0'}`}>
              <MoreHorizontal size={14} className="text-[var(--color-muted)]" />
            </button>
          )}

          {/* Menu rendered at body level via portal */}
          {showMenu && menuPos && typeof document !== 'undefined' && createPortal(
            <>
              <div className="fixed inset-0 z-[998]" onClick={() => { setShowMenu(false); setMenuPos(null) }} />
              <div className="fixed z-[999] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl py-1 min-w-[120px]"
                style={{ boxShadow: 'var(--shadow-lg)', left: menuPos.x, top: menuPos.y }}>
                <button onClick={handleCopy} className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]"><Copy size={14} /> {t('chat.copy')}</button>
                <button onClick={handleReply} className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]"><Reply size={14} /> {t('chat.reply')}</button>
                {canEdit && <button onClick={handleEdit} className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]"><PenLine size={14} /> {t('chat.edit')}</button>}
                {canRecall && <button onClick={handleRecall} className="w-full flex items-center gap-2 px-4 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--destructive)]"><Trash2 size={14} /> {t('chat.recall')}</button>}
              </div>
            </>,
            document.body
          )}

          {editing ? (
            <div className="flex flex-col gap-1">
              <textarea value={editText} onChange={e => setEditText(e.target.value)}
                className="w-full min-w-[200px] h-[60px] px-3 py-1.5 rounded-xl text-sm bg-[var(--color-surface-card)] border border-[var(--color-primary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)] text-[var(--color-ink)]"
                onCompositionStart={() => { isComposingRef.current = true }}
                onCompositionEnd={() => { isComposingRef.current = false }}
                onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey && !isComposingRef.current && !e.nativeEvent.isComposing) { e.preventDefault(); handleSaveEdit() } if (e.key === 'Escape') handleCancelEdit() }}
                autoFocus />
              <div className="flex items-center gap-1 text-[10px] text-[var(--color-muted)]">
                <span>{t('chat.editHint')}</span>
                <button onClick={handleSaveEdit} className="text-[var(--color-accent)] font-medium">{t('common.save')}</button>
                <button onClick={handleCancelEdit} className="text-[var(--color-muted)]">{t('common.cancel')}</button>
              </div>
            </div>
          ) : (
          <div className={cn(
            'relative rounded-xl px-3 py-1.5 text-sm',
            isOwn ? 'bg-[var(--bubble-self)] text-[var(--bubble-self-text)]' : 'bg-[var(--bubble-other)] text-[var(--bubble-other-text)] border border-[var(--bubble-other-border)]',
            isFailed && 'border-[var(--destructive)]',
            isSearchMatch && !isCurrentSearchMatch && 'ring-1 ring-[var(--color-primary)]/40'
          )}
            onClick={() => { if (isFailed) handleRetry() }}>
            {renderContent()}
            {message.content_type === 7 && <span className="text-[9px] opacity-40 ml-1">{t('chat.edited')}</span>}
            <div className={cn('flex items-center gap-1 mt-1', !isOwn ? 'justify-end' : 'justify-start')}>
              <span className="text-[10px] opacity-60">{formatTime(message.timestamp)}</span>
              <div
                ref={statusRef}
                className="relative"
                onMouseEnter={handleStatusHover}
                onMouseLeave={handleStatusLeave}
              >
                {statusIcon()}
                {/* Read receipts tooltip */}
                {showReadBy && readReceipts && isOwn && message.status === MsgStatus.Read && createPortal(
                  <div className="fixed z-[999] bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl py-2 px-3 shadow-lg"
                    style={{ boxShadow: 'var(--shadow-lg)', top: tooltipPos.top, left: tooltipPos.left, transform: 'translate(-50%, -100%)' }}
                    onMouseEnter={() => { if (readTimer.current) clearTimeout(readTimer.current); setShowReadBy(true) }}
                    onMouseLeave={handleStatusLeave}>
                    <div className="text-[11px] font-medium text-[var(--color-muted)] mb-1.5">{t('chat.readBy')}</div>
                    {readReceipts.length > 0 ? (
                      <div className="space-y-1.5">
                        {readReceipts.map(r => (
                          <div key={r.user_id} className="flex items-center gap-2">
                            {r.avatar ? (
                              <img src={avatarUrl(r.avatar)} alt="" className="w-5 h-5 rounded-full object-cover" />
                            ) : (
                              <div className="w-5 h-5 rounded-full bg-[var(--color-muted)]/20 flex items-center justify-center text-[9px] font-semibold text-[var(--color-ink)]">
                                {r.name?.charAt(0)?.toUpperCase() || '?'}
                              </div>
                            )}
                            <span className="text-xs text-[var(--color-ink)] truncate max-w-[120px]">{r.name}</span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="text-xs text-[var(--color-muted)]">{t('chat.noReadReceipts')}</div>
                    )}
                  </div>,
                  document.body
                )}
              </div>
            </div>
          </div>
          )}

          {isFailed && (
            <button onClick={handleRetry} className="absolute -bottom-6 right-0 text-[11px] text-[var(--destructive)] flex items-center gap-1">
              <AlertCircle size={12} /> {t('chat.retry')}
            </button>
          )}

          {hoverUser && !isOwn && userCardPos && typeof document !== 'undefined' && !message.sender_id.startsWith('webhook:') && createPortal(
            <div
              className="fixed z-[999]"
              style={{ left: userCardPos.x, top: userCardPos.y }}
              onMouseEnter={handleMouseEnter}
              onMouseLeave={handleMouseLeave}
            >
              <UserCard userId={message.sender_id} onClose={() => { setHoverUser(false); setUserCardPos(null) }} />
            </div>,
            document.body
          )}

        </div>
      </div>
      {/* Own message avatar/spacer on the right */}
      {isOwn && !isGrouped && (
        <AvatarDot name={myInitials} avatar={me?.avatar} userId={me?.user_id || ''} clickable={false} isAgent={me?.type === 1} />
      )}
      {isOwn && isGrouped && <div className="w-8 flex-shrink-0 self-start" />}
    </div>
  )
}
