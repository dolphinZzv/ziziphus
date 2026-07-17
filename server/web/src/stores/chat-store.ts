import { getItem, setItem, removeItem } from '@/lib/storage'
import { messageService } from '@/services/message-service'
import { wsClient } from '@/services/websocket-client'
import { notifySound, notifyTitle } from '@/lib/notify'
import { nextClientSeq } from '@/lib/storage'
import { MessageType } from '@/types/ws'
import type { WSFrame, MsgSendPayload, MsgPushPayload, MsgReadNotifyPayload, TypingPayload } from '@/types/ws'
import { ContentType, MsgStatus, type Message } from '@/types/message'
import type { MsgEditPushPayload, MsgRecallPushPayload } from '@/types/ws'
import { conversationStore } from '@/stores/conversation-store'

interface ChatState {
  messagesByConvId: Map<string, Message[]>
  drafts: Map<string, string>
  replyTo: Map<string, Message | null>
  typingUsers: Map<string, string[]>
  allHistoryLoaded: Map<string, boolean>
  isLoading: boolean
  uploadProgress: Map<string, number>
}

function getInitialState(): ChatState {
  const messagesByConvId = new Map<string, Message[]>()
  const drafts = new Map<string, string>()
  const replyTo = new Map<string, Message | null>()
  const typingUsers = new Map<string, string[]>()
  const allHistoryLoaded = new Map<string, boolean>()
  const uploadProgress = new Map<string, number>()
  return { messagesByConvId, drafts, replyTo, typingUsers, allHistoryLoaded, isLoading: false, uploadProgress }
}

let state = getInitialState()
const listeners = new Set<() => void>()
function emit() { listeners.forEach(l => l()) }
// Stable references for useSyncExternalStore
const EMPTY_ARR: Message[] = Object.freeze([]) as unknown as Message[]
const EMPTY_STR_ARR: string[] = Object.freeze([]) as unknown as string[]

export const chatStore = {
  get state() { return state },

  subscribe(fn: () => void) {
    listeners.add(fn)
    return () => { listeners.delete(fn) }
  },

  getMessages(convId: string): Message[] {
    return state.messagesByConvId.get(convId) || EMPTY_ARR
  },

  getDraft(convId: string): string {
    return state.drafts.get(convId) || getItem<string>(`draft_${convId}`, '')
  },

  setDraft(convId: string, text: string) {
    const drafts = new Map(state.drafts)
    if (text.trim()) {
      drafts.set(convId, text)
      setItem(`draft_${convId}`, text)
    } else {
      drafts.delete(convId)
      removeItem(`draft_${convId}`)
    }
    state = { ...state, drafts }; emit()
  },

  getReplyTo(convId: string): Message | null {
    return state.replyTo.get(convId) || null
  },

  setReplyTo(convId: string, msg: Message | null) {
    const replyTo = new Map(state.replyTo)
    if (msg) replyTo.set(convId, msg)
    else replyTo.delete(convId)
    state = { ...state, replyTo }; emit()
  },

  isTyping(convId: string): string[] {
    return state.typingUsers.get(convId) || EMPTY_STR_ARR
  },

  setTyping(convId: string, userId: string, isTyping: boolean) {
    const typingUsers = new Map(state.typingUsers)
    const current = typingUsers.get(convId) || []
    const updated = isTyping
      ? [...current.filter(id => id !== userId), userId]
      : current.filter(id => id !== userId)
    if (updated.length > 0) typingUsers.set(convId, updated)
    else typingUsers.delete(convId)
    state = { ...state, typingUsers }; emit()
  },

  async loadHistory(convId: string) {
    const messagesByConvId = new Map(state.messagesByConvId)
    if (!messagesByConvId.has(convId)) messagesByConvId.set(convId, [])
    state = { ...state, messagesByConvId, isLoading: true }; emit()

    try {
      const data = await messageService.getHistory(convId, { limit: 50 })
      // Server returns plain array, not PaginatedData
      const arr = Array.isArray(data) ? data : (data as any).items || []
      const messages = arr.reverse()
      // Merge append-only agent timeline entries into their parent messages
      const parentIds = new Set(messages.map(m => m.msg_id))
      const appendMap = new Map<number, any[]>()
      const filtered: typeof messages = []
      for (const m of messages) {
        if (m.content_type === ContentType.AgentTimeline) {
          try {
            const t = JSON.parse(m.body)
            if (t.parentMsgID > 0 && parentIds.has(t.parentMsgID)) {
              // Collect entries to merge into parent
              if (!appendMap.has(t.parentMsgID)) appendMap.set(t.parentMsgID, [])
              appendMap.get(t.parentMsgID)!.push(...(t.entries || []))
              continue // Don't show as separate bubble
            }
          } catch {}
        }
        filtered.push(m)
      }
      // Apply merged entries and status to parent messages
      if (appendMap.size > 0) {
        // Collect latest status from appends
        const parentStatus = new Map<number, string>()
        for (const msg of messages) {
          if (msg.content_type === ContentType.AgentTimeline) {
            try { const t = JSON.parse(msg.body); if (t.parentMsgID > 0) parentStatus.set(t.parentMsgID, t.status || 'running') } catch {}
          }
        }
        for (let i = 0; i < filtered.length; i++) {
          const m = filtered[i]
          const appendEntries = appendMap.get(m.msg_id)
          if (appendEntries && m.content_type === ContentType.AgentTimeline) {
            try {
              const parentBody = JSON.parse(m.body)
              const existingIds = new Set((parentBody.entries || []).map((e: any) => e.id))
              const newEntries = appendEntries.filter((e: any) => !existingIds.has(e.id))
              if (newEntries.length > 0) {
                parentBody.entries = [...(parentBody.entries || []), ...newEntries]
              }
              if (parentStatus.has(m.msg_id)) parentBody.status = parentStatus.get(m.msg_id)
              filtered[i] = { ...m, body: JSON.stringify(parentBody) }
            } catch {}
          }
        }
      }
      messagesByConvId.set(convId, filtered)
      const allHistoryLoaded = new Map(state.allHistoryLoaded)
      allHistoryLoaded.set(convId, messages.length < 50)
      state = { ...state, messagesByConvId, allHistoryLoaded, isLoading: false }; emit()
    } catch {
      state = { ...state, isLoading: false }; emit()
    }
  },

  async loadMore(convId: string) {
    const messages = state.messagesByConvId.get(convId) || []
    if (messages.length === 0) return
    const oldest = messages[0]

    try {
      const data = await messageService.getHistory(convId, { before: oldest.msg_id > 0 ? oldest.msg_id : undefined, limit: 50 })
      const arr = Array.isArray(data) ? data : (data as any).items || []
      if (arr.length === 0) {
        const allHistoryLoaded = new Map(state.allHistoryLoaded)
        allHistoryLoaded.set(convId, true)
        state = { ...state, allHistoryLoaded }; emit()
        return
      }
      const newMessages = arr.reverse()
      const messagesByConvId = new Map(state.messagesByConvId)
      // Deduplicate
      const existingIds = new Set(messages.map(m => m.msg_id > 0 ? m.msg_id : m.client_seq))
      const unique = newMessages.filter(m => !existingIds.has(m.msg_id > 0 ? m.msg_id : m.client_seq))
      messagesByConvId.set(convId, [...unique, ...messages])
      if (newMessages.length < 50) {
        const allHistoryLoaded = new Map(state.allHistoryLoaded)
        allHistoryLoaded.set(convId, true)
        state = { ...state, allHistoryLoaded }
      }
      state = { ...state, messagesByConvId }; emit()
    } catch { /* silent */ }
  },

  async sendMessage(convId: string, body: string, contentType: ContentType = ContentType.Text, replyTo: number = 0, mention: string[] = []) {
    const clientSeq = nextClientSeq()
    const me = getItem<any>('user', null)

    // Create local message
    const localMsg: Message = {
      msg_id: 0,
      conv_id: convId,
      sender_id: me?.user_id || '',
      sender_name: me?.name || '',
      sender_session_id: '',
      content_type: contentType,
      body,
      mention,
      reply_to: replyTo,
      timestamp: Math.floor(Date.now() / 1000),
      client_seq: clientSeq,
      conv_seq: 0,
      status: MsgStatus.Sending,
    }

    this.upsertMessage(convId, localMsg)

    const payload: MsgSendPayload = {
      conv_id: convId,
      content_type: contentType,
      body,
      reply_to: replyTo,
      client_seq: clientSeq,
      mention: mention.length > 0 ? mention : undefined,
    }

    try {
      const frame = await wsClient.sendWithAck({
        type: MessageType.MsgSend,
        id: `send-${clientSeq}`,
        payload,
      }, 10000)

      const ackPayload = frame.payload as { msg_id: number; timestamp: number; client_seq: number; status: number }
      this.updateMessageAfterAck(convId, clientSeq, ackPayload.msg_id, ackPayload.timestamp, ackPayload.status)
    } catch (err) {
      this.updateMessageStatus(convId, clientSeq, MsgStatus.Sending, true)
      throw err
    }
  },

  upsertMessage(convId: string, msg: Message) {
    const messagesByConvId = new Map(state.messagesByConvId)
    const messages = [...(messagesByConvId.get(convId) || [])]
    const key = msg.msg_id > 0 ? msg.msg_id : msg.client_seq
    const idx = messages.findIndex(m => (m.msg_id > 0 ? m.msg_id : m.client_seq) === key)
    if (idx >= 0) {
      messages[idx] = msg
    } else {
      messages.push(msg)
      // Sort: unsent at end, then by timestamp, then by conv_seq
      messages.sort((a, b) => {
        if (a.msg_id === 0 && b.msg_id > 0) return 1
        if (a.msg_id > 0 && b.msg_id === 0) return -1
        if (a.timestamp !== b.timestamp) return a.timestamp - b.timestamp
        return a.conv_seq - b.conv_seq
      })
    }
    messagesByConvId.set(convId, messages)
    state = { ...state, messagesByConvId }; emit()
  },

  updateMessageAfterAck(convId: string, clientSeq: number, msgId: number, timestamp: number, status: number) {
    const messagesByConvId = new Map(state.messagesByConvId)
    const messages = (messagesByConvId.get(convId) || []).map(m => {
      if (m.client_seq === clientSeq && m.msg_id === 0) {
        return { ...m, msg_id: msgId, timestamp, status: status as MsgStatus }
      }
      return m
    })
    messagesByConvId.set(convId, messages)
    state = { ...state, messagesByConvId }; emit()
  },

  updateMessageStatus(convId: string, key: number, status: MsgStatus, isError = false) {
    const messagesByConvId = new Map(state.messagesByConvId)
    const messages = (messagesByConvId.get(convId) || []).map(m => {
      if (m.msg_id === key || m.client_seq === key) {
        return { ...m, status }
      }
      return m
    })
    messagesByConvId.set(convId, messages)
    state = { ...state, messagesByConvId }; emit()
  },


  async editMessage(convId: string, msgId: number, newBody: string) {
    // Optimistically update local message first
    const msgs = this.getMessages(convId)
    const msg = msgs.find(m => m.msg_id === msgId)
    if (msg) this.upsertMessage(convId, { ...msg, body: newBody, content_type: 7 })
    // Send to server (fire-and-forget — don't block on ack)
    try {
      wsClient.send({
        type: MessageType.MsgEdit,
        payload: { conv_id: convId, msg_id: msgId, new_body: newBody },
      })
    } catch { /* ignore */ }
  },

  async recallMessage(convId: string, msgId: number) {
    // Optimistically update local message first
    const msgs = this.getMessages(convId)
    const msg = msgs.find(m => m.msg_id === msgId)
    if (msg) this.upsertMessage(convId, { ...msg, body: '', content_type: 6 })
    // Send to server (fire-and-forget — don't block on ack)
    try {
      wsClient.send({
        type: MessageType.MsgRecall,
        payload: { conv_id: convId, msg_id: msgId },
      })
    } catch { /* ignore */ }
  },

  handleEditPush(payload: MsgEditPushPayload) {
    const msgs = this.getMessages(payload.conv_id)
    const msg = msgs.find(m => m.msg_id === payload.msg_id)
    if (msg) {
      this.upsertMessage(payload.conv_id, { ...msg, body: payload.new_body, content_type: 7 })
    }
  },

  handleRecallPush(payload: MsgRecallPushPayload) {
    const msgs = this.getMessages(payload.conv_id)
    const msg = msgs.find(m => m.msg_id === payload.msg_id)
    if (msg) {
      this.upsertMessage(payload.conv_id, { ...msg, body: '', content_type: 6 })
    }
  },

  handlePush(payload: MsgPushPayload) {
    // Skip push if this message already exists locally (from sendMessage).
    // The local message has the correct sender_id; the push may have a stale one.
    const existingMsgs = state.messagesByConvId.get(payload.conv_id)
    if (existingMsgs?.some(m => m.msg_id === payload.msg_id || (m.client_seq === payload.client_seq && payload.client_seq > 0))) {
      return
    }

    const msg: Message = {
      msg_id: payload.msg_id,
      conv_id: payload.conv_id,
      sender_id: payload.sender_id,
      sender_name: payload.sender_name || payload.sender_id,
      sender_session_id: '',
      content_type: payload.content_type as ContentType,
      body: payload.body,
      mention: payload.mention,
      reply_to: payload.reply_to,
      timestamp: payload.timestamp,
      client_seq: 0,
      conv_seq: payload.conv_seq,
      status: MsgStatus.Delivered,
    }

    // Merge incremental agent timeline entries into parent message
    if (msg.content_type === ContentType.AgentTimeline) {
      try {
        const timeline = JSON.parse(msg.body) as { parentMsgID?: number; entries?: unknown[]; status?: string }
        if (timeline.parentMsgID && timeline.parentMsgID > 0) {
          const messages = state.messagesByConvId.get(payload.conv_id) || []
          // Find parent: exact match first, then ±5 range for server Snowflake timing gaps
          let parent = messages.find(m => m.msg_id === timeline.parentMsgID)
          if (!parent && timeline.parentMsgID > 0) {
            parent = messages.find(m => Math.abs(m.msg_id - timeline.parentMsgID) <= 5)
          }
          if (parent) {
            try {
              const parentBody = JSON.parse(parent.body) as { parentMsgID: number; entries: unknown[]; status: string }
              const existingIds = new Set(parentBody.entries.map((e: any) => e.id || ''))
              const newEntries = (timeline.entries || []).filter((e: any) => !existingIds.has(e.id))
              if (newEntries.length > 0) {
                parentBody.entries.push(...newEntries)
                parentBody.status = timeline.status || parentBody.status
                parent.body = JSON.stringify(parentBody)
                // Update parent in place
                const messagesByConvId = new Map(state.messagesByConvId)
                messagesByConvId.set(payload.conv_id, messages.map(m =>
                  m.msg_id === parent.msg_id ? { ...parent } : m
                ))
                state = { ...state, messagesByConvId }; emit()
                return // Don't add a separate bubble for the append
              }
            } catch {}
          }
        }
      } catch {}
    }

    this.upsertMessage(payload.conv_id, msg)

    if (!conversationStore.has(payload.conv_id)) {
      conversationStore.refresh()
    }    // Notification: sound + title badge (if tab is backgrounded)
    notifySound()
    notifyTitle(1)
  },

  handleReadNotify(payload: MsgReadNotifyPayload) {
    // Update read status for messages
    const messagesByConvId = new Map(state.messagesByConvId)
    const messages = (messagesByConvId.get(payload.conv_id) || []).map(m => {
      if (m.msg_id <= payload.msg_id && m.status === MsgStatus.Delivered) {
        return { ...m, status: MsgStatus.Read }
      }
      return m
    })
    messagesByConvId.set(payload.conv_id, messages)
    state = { ...state, messagesByConvId }; emit()
  },

  setUploadProgress(convId: string, clientSeq: number, progress: number) {
    const uploadProgress = new Map(state.uploadProgress)
    uploadProgress.set(`${convId}_${clientSeq}`, progress)
    state = { ...state, uploadProgress }; emit()
  },

  getUploadProgress(convId: string, clientSeq: number): number {
    return state.uploadProgress.get(`${convId}_${clientSeq}`) || 0
  },
}
