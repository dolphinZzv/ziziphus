import { useEffect, useRef, useState, useSyncExternalStore } from 'react'
import { useParams } from 'react-router-dom'
import { chatStore } from '@/stores/chat-store'
import { authStore } from '@/stores/auth-store'
import { conversationStore } from '@/stores/conversation-store'
import { conversationService } from '@/services/conversation-service'
import { wsClient } from '@/services/websocket-client'
import { MessageType } from '@/types/ws'
import type { MsgPushPayload } from '@/types/ws'
import { ConvType } from '@/types/conversation'
import { ContentType } from '@/types/message'
import { avatarUrl } from '@/lib/file'
import MessageList from './message-list'
import InputBar from './input-bar'
import P2PDetail from './p2p-detail'
import GroupDetail from '@/features/group/group-detail'
import HistoryView from '@/features/history/history-view'
import { MoreVertical, Clock, Copy, Check, Info, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export default function ChatView() {
  const { convId } = useParams<{ convId: string }>()
  const { t } = useTranslation()
  const rawMessages = useSyncExternalStore(chatStore.subscribe, () => chatStore.getMessages(convId || ''))
  // Filter: remove agent timeline append-only messages that were merged into parents
  const parentMsgIds = new Set<number>()
  const messages = rawMessages.filter(m => {
    if (m.msg_id > 0) parentMsgIds.add(m.msg_id)
    if (m.content_type === ContentType.AgentTimeline) {
      try {
        const tm = JSON.parse(m.body)
        if (tm.parentMsgID > 0 && parentMsgIds.has(tm.parentMsgID)) return false
      } catch {}
    }
    return true
  })
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const conversations = useSyncExternalStore(conversationStore.subscribe, () => conversationStore.state.conversations)
  const [showDetail, setShowDetail] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [groupNotice, setGroupNotice] = useState('')
  const markedReadRef = useRef<Set<string>>(new Set())

  useEffect(() => {
    if (!convId) return
    chatStore.loadHistory(convId)
    // Fetch notice for this specific group
    conversationService.getDetail(convId).then(d => {
      setGroupNotice(d.type === ConvType.Group && d.notice ? d.notice : '')
    }).catch(() => { setGroupNotice('') })
    // Listen for push messages
    const unsub = wsClient.on(MessageType.MsgPush, (payload: unknown) => {
      const push = payload as MsgPushPayload
      if (push.conv_id === convId) chatStore.handlePush(push)
    })
    return () => { unsub?.() }
  }, [convId])

  // Mark as read once messages are loaded (use max msg_id from loaded messages)
  useEffect(() => {
    if (!convId || markedReadRef.current.has(convId)) return
    const msgs = chatStore.getMessages(convId)
    if (msgs.length === 0) return
    const maxMsgId = Math.max(...msgs.map(m => m.msg_id))
    if (maxMsgId <= 0) return
    markedReadRef.current.add(convId)
    conversationService.markRead(convId, maxMsgId).catch(() => {})
    conversationStore.markRead(convId)
  }, [convId, messages])

  if (!convId) return null

  const conv = conversations.find(c => c.conv_id === convId)
  const isGroup = conv?.type === ConvType.Group
  const isSystem = conv?.type === ConvType.System
  const displayName = isSystem ? t('conversation.systemMessage') : (conv?.name || convId)
  const displayAvatar = conv?.avatar || ''
  const initials = displayName.charAt(0).toUpperCase()

  return (
    <div className="flex flex-col h-full">
      {/* Chat toolbar */}
      <div className="h-12 flex items-center px-4 border-b border-[var(--color-hairline)] flex-shrink-0 bg-[var(--color-surface-card)] gap-3">
        {/* Avatar */}
        <div className="relative flex-shrink-0">
          {displayAvatar ? (
            <img src={avatarUrl(displayAvatar)} alt="" className="w-7 h-7 rounded-full object-cover" />
          ) : (
            <div className="w-7 h-7 rounded-full flex items-center justify-center text-white text-xs font-semibold"
              style={{ background: isGroup
                ? 'linear-gradient(135deg, var(--color-accent), #34D399)'
                : 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
              {initials}
            </div>
          )}
          {isGroup && (
            <div className="absolute -bottom-1 -right-1 w-[14px] h-[14px] rounded-full bg-[var(--color-surface-card)] flex items-center justify-center">
              <Users size={8} className="text-[var(--color-muted)]" />
            </div>
          )}
        </div>
        <div className="flex-1 min-w-0">
          <span className="font-headline text-sm font-semibold text-[var(--color-ink)] truncate block">
            {displayName}
          </span>
          {isSystem ? (
            <div className="h-[15px]" />
          ) : (
          <button
            onClick={() => { navigator.clipboard.writeText(convId); setCopied(true); setTimeout(() => setCopied(false), 2000) }}
            className="text-[10px] text-[var(--color-muted-soft)] hover:text-[var(--color-ink)] font-mono truncate flex items-center gap-1 transition-colors cursor-pointer"
            title={t('chat.clickCopyId')}
          >
            {convId}
            {copied ? <Check size={10} className="text-[var(--success)]" /> : <Copy size={10} />}
          </button>
          )}
        </div>
        <div className="relative">
          <button onClick={() => setShowMenu(!showMenu)}
            className="p-1.5 rounded-lg hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors">
            <MoreVertical size={18} />
          </button>
          {showMenu && (
            <>
              <div className="fixed inset-0 z-10" onClick={() => setShowMenu(false)} />
              <div className="absolute right-0 top-full mt-1 w-40 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-lg z-20 py-1"
                style={{ boxShadow: 'var(--shadow-md)' }}>
                {!isSystem && (
                  <button onClick={() => { setShowDetail(true); setShowMenu(false) }}
                    className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                    <Info size={14} /> {t('chat.detail')}
                  </button>
                )}
                <button onClick={() => { setShowHistory(true); setShowMenu(false) }}
                  className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                  <Clock size={14} /> {t('chat.history')}
                </button>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Detail dialog */}
      {showDetail && isGroup && (
        <GroupDetail convId={convId} onClose={() => setShowDetail(false)} />
      )}
      {showDetail && !isGroup && (
        <P2PDetail convId={convId} onClose={() => setShowDetail(false)} />
      )}

      {/* Group notice banner */}
      {isGroup && groupNotice && (
        <div className="px-4 py-2 bg-[var(--color-warning)]/5 border-b border-[var(--color-hairline)] text-xs text-[var(--color-body)] leading-relaxed">
          <span className="text-[var(--color-muted)] mr-1">📢</span>
          {groupNotice}
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-hidden">
        <MessageList convId={convId} messages={messages} currentUserId={user?.user_id || ''} />
      </div>

      {!isSystem && <InputBar convId={convId} />}

      {/* History modal */}
      {showHistory && <HistoryView convId={convId} onClose={() => setShowHistory(false)} />}
    </div>
  )
}
