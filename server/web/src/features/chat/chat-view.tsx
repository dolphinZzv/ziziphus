import { useEffect, useRef, useState, useSyncExternalStore, useCallback } from 'react'
import { useParams, useLocation } from 'react-router-dom'
import { chatStore } from '@/stores/chat-store'
import { authStore } from '@/stores/auth-store'
import { uiStore } from '@/stores/ui-store'
import { conversationStore } from '@/stores/conversation-store'
import { conversationService } from '@/services/conversation-service'
import { fileService } from '@/services/file-service'
import { wsClient } from '@/services/websocket-client'
import { MessageType } from '@/types/ws'
import type { MsgPushPayload } from '@/types/ws'
import { ConvType, ConvRole } from '@/types/conversation'
import { ContentType } from '@/types/message'
import { avatarUrl } from '@/lib/file'
import MessageList from './message-list'
import InputBar from './input-bar'
import P2PDetail from './p2p-detail'
import GroupBasicInfo from '@/features/group/group-basic-info'
import GroupSettings from '@/features/group/group-settings'
import WebhookPanel from '@/features/group/webhook-panel'
import MemberListView from '@/features/group/member-list-view'
import AddMemberView from '@/features/group/add-member-view'
import HistoryView from '@/features/history/history-view'
import { MoreVertical, Clock, Copy, Check, Info, Users, LogOut, Folder, Search, ChevronUp, ChevronDown, X, Trash2, UserPlus, ArrowLeft } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useIsMobile } from '@/hooks/use-breakpoint'
import FilePanel from './file-panel'

export default function ChatView() {
  const { convId } = useParams<{ convId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
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
  const isMobile = useIsMobile()
  // Chat panel sub-route — derived from URL
  const activePanel = (() => {
    const m = location.pathname.match(/^\/(?:chat|conversations)\/([^/]+)\/(info|settings|webhooks|add-member|members|detail|history)$/)
    return m && m[1] === convId ? m[2] : null
  })()

  // Navigate to a chat panel sub-route
  const openPanel = (panel: string) => navigate(`/conversations/${convId}/${panel}`)
  // Close panel and return to chat
  const closePanel = () => { navigate(-1) }
  const [copied, setCopied] = useState(false)
  const [showFiles, setShowFiles] = useState(false)
  const [showMenu, setShowMenu] = useState(false)
  const [isOwner, setIsOwner] = useState(false)
  const [filePanelWidth, setFilePanelWidth] = useState(260)
  const [dragging, setDragging] = useState(false)
  const [groupNotice, setGroupNotice] = useState('')
  const [convColor, setConvColor] = useState('')
  const markedReadRef = useRef<Map<string, number>>(new Map())

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

  // Set browser chrome/tab color to conversation's primary color, fallback to user's
  useEffect(() => {
    const color = convColor || user?.primary_color || '#0F172A'
    let meta = document.querySelector('meta[name="theme-color"]') as HTMLMetaElement | null
    if (meta) meta.content = color
    else {
      meta = document.createElement('meta')
      meta.name = 'theme-color'
      meta.content = color
      document.head.appendChild(meta)
    }
  }, [convColor, user?.primary_color])

  const handleClone = async () => {
    if (!convId || !isGroup) return
    if (!confirm(t('group.cloneConfirm'))) return
    try {
      const r = await conversationService.clone(convId)
      setShowMenu(false)
      navigate(`/conversations/${r.conv_id}`)
    } catch {}
  }

  const handleDisband = async () => {
    if (!convId) return
    if (!confirm(t('group.disbandConfirm'))) return
    try {
      await conversationService.disband(convId)
      conversationStore.removeConversation(convId)
      setShowMenu(false)
      navigate('/conversations')
    } catch {}
  }

  const handleLeave = async () => {
    if (!convId) return
    if (!confirm(isGroup ? t('group.leaveConfirm') : t('conversation.leaveConfirm'))) return
    try {
      await conversationService.leave(convId)
      conversationStore.removeConversation(convId)
      setShowMenu(false)
      navigate('/conversations')
    } catch {}
  }

  useEffect(() => {
    if (!convId) return
    chatStore.loadHistory(convId)
    // Fetch notice for this specific group
    conversationService.getDetail(convId).then(d => {
      setGroupNotice(d.type === ConvType.Group && d.notice ? d.notice : '')
      setConvColor(d.primary_color || '')
      const me = d.members?.find(m => m.user_id === user?.user_id)
      setIsOwner(me?.role === ConvRole.Owner)
    }).catch(() => { setGroupNotice(''); setConvColor(''); setIsOwner(false) })
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

  // Mark as read whenever messages change (handles new push messages too)
  useEffect(() => {
    if (!convId) return
    const msgs = chatStore.getMessages(convId)
    if (msgs.length === 0) return
    const maxMsgId = Math.max(...msgs.map(m => m.msg_id))
    if (maxMsgId <= 0) return
    // Only call server if this is a new maxId
    const prev = markedReadRef.current.get(convId) || 0
    if (maxMsgId > prev) {
      markedReadRef.current.set(convId, maxMsgId)
      conversationService.markRead(convId, maxMsgId).catch(() => {})
      conversationStore.markRead(convId)
    }
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
      <div className="h-12 flex items-center px-4 flex-shrink-0 gap-3">
        {/* Avatar */}
        {!showSearch ? (
          <>
          {/* Mobile back button — goes to conversation list */}
          {isMobile && (
            <button
              onClick={() => navigate('/conversations')}
              className="p-1.5 -ml-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors md:hidden"
              aria-label={t('common.back', '返回')}
            >
              <ArrowLeft size={20} />
            </button>
          )}
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
              className="text-[10px] text-[var(--color-muted-soft)] hover:text-[var(--color-ink)] font-mono truncate hidden sm:flex items-center gap-1 transition-colors cursor-pointer"
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
            className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors hidden sm:inline-flex"
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
                    {isGroup ? (
                      <>
                        <button onClick={() => { openPanel('info'); setShowMenu(false) }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                          <Info size={14} /> {t('group.basicInfo')}
                        </button>
                        <button onClick={() => { openPanel('settings'); setShowMenu(false) }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
                          {t('group.settingsMenu')}
                        </button>
                        <button onClick={() => { openPanel('webhooks'); setShowMenu(false) }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
                          {t('group.webhookMenu')}
                        </button>
                        <div className="border-t border-[var(--color-hairline)] my-1" />
                        <button onClick={() => { openPanel('add-member'); setShowMenu(false) }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                          <UserPlus size={14} /> {t('group.addMember')}
                        </button>
                        <button onClick={() => { openPanel('members'); setShowMenu(false) }}
                          className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                          <Users size={14} /> {t('group.members')}
                        </button>
                      </>
                    ) : (
                      <button onClick={() => { openPanel('detail'); setShowMenu(false) }}
                        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                        <Info size={14} /> {t('chat.detail')}
                      </button>
                    )}
                    <button onClick={() => { openPanel('history'); setShowMenu(false) }}
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
                    {!isOwner && (
                    <button onClick={handleLeave}
                      className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-ink)]">
                      <LogOut size={14} /> {t('group.leave')}
                    </button>
                    )}
                    {isGroup && isOwner && (
                      <button onClick={handleDisband}
                        className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--destructive)]/10 text-[var(--destructive)]">
                        <Trash2 size={14} /> {t('group.disband')}
                      </button>
                    )}
                  </>
                )}
                {isSystem && (
                  <button onClick={() => { openPanel('history'); setShowMenu(false) }}
                    className="w-full flex items-center gap-2 px-3 py-2 text-sm hover:bg-[var(--color-surface-soft)] text-[var(--color-body)]">
                    <Clock size={14} /> {t('chat.history')}
                  </button>
                )}
              </div>
            </>
          )}
        </div>
      </div>

      {activePanel === 'info' && (
        <GroupBasicInfo convId={convId} onClose={closePanel} />
      )}
      {activePanel === 'settings' && (
        <GroupSettings convId={convId} onClose={closePanel} />
      )}
      {activePanel === 'webhooks' && (
        <WebhookPanel convId={convId} onClose={closePanel} />
      )}
      {activePanel === 'members' && (
        <MemberListView convId={convId} onClose={closePanel} />
      )}
      {activePanel === 'add-member' && (
        <AddMemberView
          convId={convId}
          onClose={closePanel}
          onAdded={() => {}}
          excludeIds={new Set()}
        />
      )}
      {activePanel === 'detail' && (
        <P2PDetail convId={convId} onClose={closePanel} />
      )}

      {/* Group notice banner */}
      {isGroup && groupNotice && (
        <div className="px-4 py-2 bg-[var(--color-warning)]/5 text-xs text-[var(--color-body)] leading-relaxed">
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

      {!isSystem && <InputBar convId={convId} isP2p={conv?.type === ConvType.P2P} />}

      {/* History modal */}
      {activePanel === 'history' && <HistoryView convId={convId} onClose={closePanel} />}
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

