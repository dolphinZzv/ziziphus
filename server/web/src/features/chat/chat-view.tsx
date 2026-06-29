import { useEffect, useRef, useState, useSyncExternalStore, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { chatStore } from '@/stores/chat-store'
import { authStore } from '@/stores/auth-store'
import { conversationStore } from '@/stores/conversation-store'
import { conversationService } from '@/services/conversation-service'
import { fileService } from '@/services/file-service'
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
import MemberListView from '@/features/group/member-list-view'
import HistoryView from '@/features/history/history-view'
import { MoreVertical, Clock, Copy, Check, Info, Users, LogOut, Folder, Search, ChevronUp, ChevronDown, X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import FilePanel from './file-panel'

export default function ChatView() {
  const { convId } = useParams<{ convId: string }>()
  const navigate = useNavigate()
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
  const [showMembers, setShowMembers] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showFiles, setShowFiles] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [filePanelWidth, setFilePanelWidth] = useState(260)
  const [dragging, setDragging] = useState(false)
  const [groupNotice, setGroupNotice] = useState('')
  const markedReadRef = useRef<Set<string>>(new Set())

  // --- Feature 1: In-chat search ---
  const [showSearch, setShowSearch] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [currentMatchIndex, setCurrentMatchIndex] = useState(0)
  const containerRef = useRef<HTMLDivElement>(null)

  // Compute matches from text-based messages
  const searchMatches = searchKeyword.trim()
    ? messages.reduce<number[]>((acc, m, i) => {
        if ((m.content_type === ContentType.Text || m.content_type === ContentType.Edit) &&
            m.body.toLowerCase().includes(searchKeyword.toLowerCase())) {
          acc.push(i)
        }
        return acc
      }, [])
    : []

  const handleSearchPrev = () => {
    setCurrentMatchIndex(i => (i > 0 ? i - 1 : searchMatches.length - 1))
  }
  const handleSearchNext = () => {
    setCurrentMatchIndex(i => (i < searchMatches.length - 1 ? i + 1 : 0))
  }
  const handleSearchClose = () => {
    setShowSearch(false)
    setSearchKeyword('')
    setCurrentMatchIndex(0)
  }

  // Scroll to current match
  useEffect(() => {
    if (searchMatches.length === 0) return
    const targetId = `msg-${messages[searchMatches[currentMatchIndex]]?.msg_id}`
    const el = document.getElementById(targetId)
    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }, [currentMatchIndex, searchMatches])

  // --- Feature 2: Drag-drop on chat area ---
  const handleDropOnChat = useCallback(async (e: React.DragEvent) => {
    e.preventDefault()
    if (!convId) return
    const files = Array.from(e.dataTransfer?.files || [])
    if (files.length === 0) return
    for (const file of files) {
      const type = file.type.startsWith('image/') ? ('image' as const) : ('file' as const)
      try {
        const result = await fileService.upload(file, file.name, type === 'image' ? 0 : 1)
        const body = JSON.stringify({ url: result.url, name: file.name, size: file.size, file_id: result.file_id })
        chatStore.sendMessage(convId, body, type === 'image' ? ContentType.Image : ContentType.File)
      } catch { /* ignore */ }
    }
  }, [convId])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    if (e.dataTransfer?.types?.some(t => t === 'Files')) {
      e.preventDefault()
      e.dataTransfer.dropEffect = 'copy'
    }
  }, [])

  // --- End drag-drop ---

  const conv = conversations.find(c => c.conv_id === convId)
  const isGroup = conv?.type === ConvType.Group

  const handleClone = async () => {
    if (!convId || !isGroup) return
    if (!confirm(t('group.cloneConfirm'))) return
    try {
      const r = await conversationService.clone(convId)
      setShowMenu(false)
      navigate(`/chat/${r.conv_id}`)
    } catch {}
  }

  const handleLeave = async () => {
    if (!convId) return
    if (!confirm(isGroup ? t('group.leaveConfirm') : t('conversation.leaveConfirm'))) return
    try {
      await conversationService.leave(convId)
      conversationStore.removeConversation(convId)
      setShowMenu(false)
      navigate('/chat')
    } catch {}
  }

  useEffect(() => {
    if (!convId) return
    chatStore.loadHistory(convId)
    // Fetch notice for this specific group
    conversationService.getDetail(convId).then(d => {
      setGroupNotice(d.type === ConvType.Group && d.notice ? d.notice : '')
    }).catch(() => { setGroupNotice('') })
    // Listen for push messages
    const u1 = wsClient.on(MessageType.MsgPush, (payload: unknown) => {
      const push = payload as MsgPushPayload
      if (push.conv_id === convId) chatStore.handlePush(push)
    })
    const u2 = wsClient.on(MessageType.MsgEdit, (payload: unknown) => {
      const edit = payload as import('@/types/ws').MsgEditPushPayload
      if (edit.conv_id === convId) chatStore.handleEditPush(edit)
    })
    const u3 = wsClient.on(MessageType.MsgRecall, (payload: unknown) => {
      const recall = payload as import('@/types/ws').MsgRecallPushPayload
      if (recall.conv_id === convId) chatStore.handleRecallPush(recall)
    })
    const u4 = wsClient.on(MessageType.MsgReadNotify, (payload: unknown) => {
      const rn = payload as import('@/types/ws').MsgReadNotifyPayload
      if (rn.conv_id === convId) chatStore.handleReadNotify(rn)
    })
    return () => { u1?.(); u2?.(); u3?.(); u4?.() }
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

  const isSystem = conv?.type === ConvType.System
  const displayName = isSystem ? t('conversation.systemMessage') : (conv?.name || convId)
  const displayAvatar = conv?.avatar || ''
  const initials = displayName.charAt(0).toUpperCase()

  return (
    <div
      className="flex h-full"
      onDragOver={handleDragOver}
      onDrop={handleDropOnChat}
    >
      <div ref={containerRef} className="flex-1 flex flex-col min-w-0 relative">
      {/* Chat toolbar */}
      <div className="h-12 flex items-center px-4 border-b border-[var(--color-hairline)] flex-shrink-0 bg-[var(--color-surface-card)] gap-3">
        {/* Avatar */}
        {!showSearch ? (
          <>
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
          </>
        ) : (
          <>
          <div className="flex-1 flex items-center gap-2">
            <div className="relative flex-1">
              <input
                type="text"
                value={searchKeyword}
                onChange={e => { setSearchKeyword(e.target.value); setCurrentMatchIndex(0) }}
                placeholder={t('chat.searchPlaceholder')}
                className="w-full h-8 pl-3 pr-8 rounded-xl bg-[var(--color-surface-soft)] text-sm border border-[var(--color-hairline)] focus:outline-none focus:border-[var(--color-primary)] text-[var(--color-ink)]"
                autoFocus
                onKeyDown={e => { if (e.key === 'Enter' && !e.nativeEvent.isComposing) handleSearchNext(); if (e.key === 'Escape') handleSearchClose() }}
              />
              {searchKeyword.trim() && (
                <div className="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-0.5 text-[11px] text-[var(--color-muted)]">
                  <span>{searchMatches.length > 0 ? `${currentMatchIndex + 1}/${searchMatches.length}` : '0'}</span>
                </div>
              )}
            </div>
            <button
              onClick={handleSearchPrev}
              disabled={searchMatches.length === 0}
              className="p-1 rounded hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] disabled:opacity-30 transition-colors"
            >
              <ChevronUp size={16} />
            </button>
            <button
              onClick={handleSearchNext}
              disabled={searchMatches.length === 0}
              className="p-1 rounded hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] disabled:opacity-30 transition-colors"
            >
              <ChevronDown size={16} />
            </button>
            <button
              onClick={handleSearchClose}
              className="p-1 rounded hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
            >
              <X size={16} />
            </button>
          </div>
          <div className="flex items-center gap-1">
            {searchMatches.length > 0 && (
              <span className="text-[11px] text-[var(--color-muted)]">
                {searchMatches.length} 条结果
              </span>
            )}
          </div>
          </>
        )}

        {!isSystem && (
          <>
          {!showSearch && (
            <button onClick={() => setShowSearch(true)}
              className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
              title={t('chat.search')}>
              <Search size={17} />
            </button>
          )}
          <button onClick={() => setShowFiles(!showFiles)}
            className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
            title={t('conversation.files')}>
            <Folder size={17} />
          </button>
          </>
        )}
        <div className="relative">
          <button onClick={() => setShowMenu(!showMenu)}
            className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors">
            <MoreVertical size={18} />
          </button>
          {showMenu && (
            <>
              <div className="fixed inset-0 z-10" onClick={() => setShowMenu(false)} />
              <div className="absolute right-0 top-full mt-1 w-40 bg-[var(--color-surface-card)] border border-[var(--color-hairline)] rounded-xl z-20 py-1"
                style={{ boxShadow: 'var(--shadow-md)' }}>
                {!isSystem && (
                  <>
                    <button onClick={() => { setShowDetail(true); setShowMenu(false) }}
                      className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                      <Info size={14} /> {t('chat.detail')}
                    </button>
                    {isGroup && (
                      <button onClick={() => { setShowMembers(true); setShowMenu(false) }}
                        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                        <Users size={14} /> {t('group.members')}
                      </button>
                    )}
                    <button onClick={() => { setShowHistory(true); setShowMenu(false) }}
                      className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                      <Clock size={14} /> {t('chat.history')}
                    </button>
                    {isGroup && (
                      <button onClick={handleClone}
                        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                        <Copy size={14} /> {t('group.clone')}
                      </button>
                    )}
                    <div className="border-t border-[var(--color-hairline)] my-1" />
                    <button onClick={handleLeave}
                      className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-ink)]">
                      <LogOut size={14} /> {t('group.leave')}
                    </button>
                  </>
                )}
                {isSystem && (
                  <button onClick={() => { setShowHistory(true); setShowMenu(false) }}
                    className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                    <Clock size={14} /> {t('chat.history')}
                  </button>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {/* Detail dialog */}
      {showDetail && isGroup && (
        <GroupDetail convId={convId} onClose={() => setShowDetail(false)} />
      )}
      {showMembers && (
        <MemberListView convId={convId} onClose={() => setShowMembers(false)} />
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
        <MessageList
          convId={convId}
          messages={messages}
          currentUserId={user?.user_id || ''}
          searchKeyword={searchKeyword}
          matchIndex={currentMatchIndex}
          searchMatches={searchMatches}
        />
      </div>

      {!isSystem && <InputBar convId={convId} />}

      {/* History modal */}
      {showHistory && <HistoryView convId={convId} onClose={() => setShowHistory(false)} />}
      </div>
      {/* Zero-width drag handle wrapper — sits between chat area and file panel */}
      <div className="relative flex-shrink-0" style={{ width: 0 }}>
        {showFiles && (
          <div className="absolute -left-1 top-0 bottom-0 w-2 cursor-col-resize group z-10"
            onMouseDown={e => {
              e.preventDefault(); e.stopPropagation()
              const sx = e.clientX; const sw = filePanelWidth
              setDragging(true); document.body.style.userSelect = 'none'
              const mv = (ev: MouseEvent) => { ev.preventDefault(); setFilePanelWidth(Math.max(180, Math.min(500, sw + sx - ev.clientX))) }
              const up = () => { setDragging(false); document.body.style.userSelect = ''; document.removeEventListener('mousemove', mv); document.removeEventListener('mouseup', up) }
              document.addEventListener('mousemove', mv); document.addEventListener('mouseup', up)
            }}>
            <div className="absolute left-1/2 -translate-x-1/2 w-px h-full bg-transparent group-hover:w-1 group-hover:bg-[var(--color-primary)] transition-all" />
          </div>
        )}
      </div>
      {/* File panel sidebar */}
      {showFiles && <FilePanel convId={convId} onClose={() => setShowFiles(false)} width={filePanelWidth} />}
      {dragging && <div className="fixed inset-0 z-50 cursor-col-resize" style={{ userSelect: 'none' } as React.CSSProperties} />}
    </div>
  )
}

