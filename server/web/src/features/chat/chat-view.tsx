import { useEffect, useRef, useState, useSyncExternalStore, useMemo } from 'react'
import { useParams, useLocation } from 'react-router-dom'
import { chatStore } from '@/stores/chat-store'
import { authStore } from '@/stores/auth-store'
import { conversationStore } from '@/stores/conversation-store'
import { conversationService } from '@/services/conversation-service'
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
import UserCard from '@/components/user-card'
import HistoryView from '@/features/history/history-view'
import { MoreVertical, Clock, Copy, Check, Info, Users, LogOut, Folder, Search, Trash2, UserPlus, ArrowLeft, Pin, PinOff, MessageCircle } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useIsMobile } from '@/hooks/use-breakpoint'
import FilePanel from './file-panel'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

export default function ChatView() {
  const { convId } = useParams<{ convId: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const { t } = useTranslation()
  const rawMessages = useSyncExternalStore(chatStore.subscribe, () => chatStore.getMessages(convId || ''))
  // Filter: remove agent timeline append-only messages that were merged into parents
  const messages = useMemo(() => {
    const parentMsgIds = new Set<number>()
    return rawMessages.filter(m => {
      if (m.msg_id > 0) parentMsgIds.add(m.msg_id)
      if (m.content_type === ContentType.AgentTimeline) {
        try {
          const tm = JSON.parse(m.body)
          if (tm.parentMsgID > 0 && parentMsgIds.has(tm.parentMsgID)) return false
        } catch (e) { console.error(e) }
      }
      return true
    })
  }, [rawMessages])
  const user = useSyncExternalStore(authStore.subscribe, () => authStore.state.user)
  const conversations = useSyncExternalStore(conversationStore.subscribe, () => conversationStore.state.conversations)
  const isMobile = useIsMobile()
  // Chat panel sub-route — derived from URL
  const activePanel = (() => {
    const m = location.pathname.match(/^\/(?:chat|conversations)\/([^/]+)\/(info|settings|webhooks|add-member|members|detail|history)$/)
    const userM = location.pathname.match(/^\/(?:chat|conversations)\/([^/]+)\/user\/([^/]+)$/)
    if (userM && userM[1] === convId) return 'user'
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
  const [bgImage, setBgImage] = useState('')
  const [filePanelWidth, setFilePanelWidth] = useState(260)
  const [dragging, setDragging] = useState(false)
  const [groupNotice, setGroupNotice] = useState('')
  const [convColor, setConvColor] = useState('')
  const [detailName, setDetailName] = useState('')
  const [detailAvatar, setDetailAvatar] = useState('')
  const lastMarkedRef = useRef<Map<string, number>>(new Map())

  // --- Drag-drop handled by InputBar ---

  // --- End drag-drop ---

  const conv = conversations.find(c => c.conv_id === convId)
  const isGroup = conv?.type === ConvType.Group

  // Set browser chrome/tab color to conversation's primary color, fallback to user's
  useEffect(() => {
    const color = convColor || conv?.primary_color || user?.primary_color || '#0F172A'
    let meta = document.querySelector('meta[name="theme-color"]') as HTMLMetaElement | null
    if (meta) meta.content = color
    else {
      meta = document.createElement('meta')
      meta.name = 'theme-color'
      meta.content = color
      document.head.appendChild(meta)
    }
  }, [convColor, conv?.primary_color, user?.primary_color])

  const handleClone = async () => {
    if (!convId || !isGroup) return
    if (!confirm(t('group.cloneConfirm'))) return
    try {
      const r = await conversationService.clone(convId)
      setShowMenu(false)
      navigate(`/conversations/${r.conv_id}`)
    } catch (e) { console.error(e) }
  }

  const handleDisband = async () => {
    if (!convId) return
    if (!confirm(t('group.disbandConfirm'))) return
    try {
      await conversationService.disband(convId)
      conversationStore.removeConversation(convId)
      setShowMenu(false)
      navigate('/conversations')
    } catch (e) { console.error(e) }
  }

  const handleLeave = async () => {
    if (!convId) return
    if (!confirm(isGroup ? t('group.leaveConfirm') : t('conversation.leaveConfirm'))) return
    try {
      await conversationService.leave(convId)
      conversationStore.removeConversation(convId)
      setShowMenu(false)
      navigate('/conversations')
    } catch (e) { console.error(e) }
  }

  useEffect(() => {
    if (!convId) return
    chatStore.loadHistory(convId)
    // Fetch notice for this specific group
    conversationService.getDetail(convId).then(d => {
      setGroupNotice(d.type === ConvType.Group && d.notice ? d.notice : '')
      setConvColor(d.primary_color || '')
      // For P2P conversations, resolve peer name/avatar from members
      if (d.type === ConvType.P2P) {
        const peer = d.members?.find(m => m.user_id !== user?.user_id)
        setDetailName(peer?.name || d.name || '')
        setDetailAvatar(peer?.avatar || d.avatar || '')
        setConvColor(peer?.primary_color || d.primary_color || '')
      } else {
        setDetailName(d.name || '')
        setDetailAvatar(d.avatar || '')
      }
      const me = d.members?.find(m => m.user_id === user?.user_id)
      setIsOwner(me?.role === ConvRole.Owner)
    }).catch(() => { setGroupNotice(''); setConvColor(''); setDetailName(''); setDetailAvatar(''); setIsOwner(false) })
    // Fetch conversation background image
    conversationService.getSettings(convId).then(res => {
      setBgImage((res.settings as any)?.background_image || '')
    }).catch(() => {})
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
  }, [convId, user?.user_id])

  // Mark as read whenever messages change (handles new push messages too)
  useEffect(() => {
    if (!convId) return
    const msgs = chatStore.getMessages(convId)
    if (msgs.length === 0) return
    const maxMsgId = Math.max(...msgs.map(m => m.msg_id))
    if (maxMsgId <= 0) return
    const prev = lastMarkedRef.current.get(convId) || 0
    if (maxMsgId <= prev) return
    lastMarkedRef.current.set(convId, maxMsgId)
    conversationService.markRead(convId, maxMsgId)
      .then(() => conversationStore.markRead(convId))
      .catch((e) => { console.error('[markRead] failed:', e) })
  }, [convId, messages])

  if (!convId) return null

  const isSystem = conv?.type === ConvType.System || convId.startsWith('sys:')
  const displayName = isSystem ? t('conversation.systemMessage') : (conv?.name || detailName || convId)
  const displayAvatar = conv?.avatar || detailAvatar || ''
  const initials = displayName.charAt(0).toUpperCase()

  return (
    <div
      className="flex h-full"
    >
      <div className={'flex-1 flex flex-col min-w-0' + (bgImage ? ' relative' : '')}
        style={bgImage ? { backgroundImage: 'url(' + bgImage + ')', backgroundSize: 'cover', backgroundPosition: 'center', backgroundRepeat: 'no-repeat' } as React.CSSProperties : undefined}>
      {/* Chat toolbar */}
      <div className="h-12 flex items-center px-4 flex-shrink-0 gap-3">
        {/* Avatar */}
        
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
          <button onClick={() => isGroup ? openPanel('info') : !isSystem ? openPanel('detail') : undefined}
            className="relative flex-shrink-0">
            {displayAvatar ? (
              <img loading="lazy" decoding="async" src={avatarUrl(displayAvatar)} alt="" className="w-7 h-7 rounded-full object-cover" />
            ) : isSystem ? (
              <div className="w-7 h-7 rounded-full flex items-center justify-center"
                style={{ background: 'linear-gradient(135deg, var(--color-primary), var(--color-muted))' }}>
                <MessageCircle size={15} className="text-white" />
              </div>
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
          </button>
          <div className="flex-1 min-w-0">
            <button onClick={() => isGroup ? openPanel('info') : !isSystem ? openPanel('detail') : undefined}
              className="text-left w-full">
              <span className="font-headline text-sm font-semibold text-[var(--color-ink)] truncate block">
                {displayName}
              </span>
            </button>
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

        {!isSystem && (
          <>
          <button onClick={() => openPanel('history')}
            className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
            title={t('chat.search')}>
            <Search size={17} />
          </button>
          <button onClick={() => conv?.pinned ? conversationStore.unpin(convId) : conversationStore.pin(convId)}
            className="p-1.5 rounded-xl hover:bg-[var(--color-surface-soft)] text-[var(--color-muted)] hover:text-[var(--color-ink)] transition-colors"
            title={conv?.pinned ? t('common.unpin') : t('common.pin')}>
            {conv?.pinned ? <PinOff size={16} /> : <Pin size={16} />}
          </button>
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
          <GroupNotice text={groupNotice} />
        </div>
      )}

      {/* Messages */}
      <div className="flex-1 overflow-hidden">
        <MessageList
          convId={convId}
          messages={messages}
          currentUserId={user?.user_id || ''}
        />
      </div>

      {!isSystem && <InputBar convId={convId} isP2p={conv?.type === ConvType.P2P} />}

      {/* History modal */}
      {activePanel === 'history' && <HistoryView convId={convId} onClose={closePanel} />}
      {activePanel === 'user' && (
        <div className="fixed inset-0 z-50 flex sm:hidden items-center justify-center bg-black/30" onClick={closePanel}>
          <div onClick={e => e.stopPropagation()}>
            <UserCard userId={location.pathname.split('/').pop() || ''} onClose={closePanel} />
          </div>
        </div>
      )}
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

function GroupNotice({ text }: { text: string }) {
  return (
    <div className="prose prose-xs dark:prose-invert max-w-none text-current [&_p]:my-0 [&_code]:text-[11px] [&_a]:text-current [&_a]:underline">
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{text}</ReactMarkdown>
    </div>
  )
}

