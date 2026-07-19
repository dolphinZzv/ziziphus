import { useEffect, useRef, useLayoutEffect, useCallback, useState, memo, useMemo } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import type { Message } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
import { userService } from '@/services/user-service'
import MessageBubble from './message-bubble'
import DateSeparator from '@/components/date-separator'
import { isSameDay } from '@/lib/time'

interface Props {
  convId: string
  messages: Message[]
  currentUserId: string
}

const MemoBubble = memo(MessageBubble)

type ListItem =
  | { type: 'date'; ts: number; key: string }
  | { type: 'msg'; msg: Message; index: number; key: string }

export default function MessageList({ convId, messages, currentUserId }: Props) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const prevLen = useRef(0)
  const prevFirstId = useRef(0)
  const shouldAutoScroll = useRef(true)
  const initialScrollDone = useRef(false)
  const loadMoreState = useRef<'idle' | 'pending' | 'done'>('idle')
  const loadMorePrevHeight = useRef(0)
  const [showScrollBtn, setShowScrollBtn] = useState(false)

  // Build virtual list items
  const items = useMemo<ListItem[]>(() => {
    const result: ListItem[] = []
    let lastDate = 0
    for (let i = 0; i < messages.length; i++) {
      const msg = messages[i]
      if (!isSameDay(msg.timestamp, lastDate)) {
        result.push({ type: 'date', ts: msg.timestamp, key: `date-${msg.timestamp}` })
        lastDate = msg.timestamp
      }
      result.push({
        type: 'msg',
        msg,
        index: i,
        key: msg.msg_id > 0 ? `msg-${msg.msg_id}` : `local-${msg.client_seq}`,
      })
    }
    return result
  }, [messages])

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: (i) => items[i]?.type === 'date' ? 28 : 72,
    overscan: 15,
    paddingEnd: 20,
    measureElement: (el) => el.scrollHeight,
  })
  const virtualizerRef = useRef(virtualizer)
  virtualizerRef.current = virtualizer

  // Batch-fetch sender info
  const fetchedSendersRef = useRef(new Set<string>())
  const [senderMap, setSenderMap] = useState<Record<string, { avatar?: string; isAgent: boolean }>>({})
  useEffect(() => {
    const ids = new Set<string>()
    for (const msg of messages) {
      if (msg.sender_id && msg.sender_id !== currentUserId && !msg.sender_id.startsWith('webhook:') && msg.sender_id !== 'system') {
        ids.add(msg.sender_id)
      }
    }
    const idsArr = Array.from(ids)
    if (idsArr.length === 0) return
    const missing = idsArr.filter(id => !fetchedSendersRef.current.has(id))
    if (missing.length === 0) return
    missing.forEach(id => fetchedSendersRef.current.add(id))
    userService.batchGet(missing).then(users => {
      setSenderMap(prev => {
        const next = { ...prev }
        for (const id of missing) {
          const u = users[id]
          next[id] = { avatar: u?.avatar || '', isAgent: u?.type === 1 }
        }
        return next
      })
    }).catch(() => {})
  }, [messages, currentUserId])

  const scrollToEnd = useCallback((smooth = false) => {
    if (items.length === 0) return
    virtualizer.scrollToIndex(items.length - 1, { align: 'end', behavior: smooth ? 'smooth' : 'auto' })
  }, [virtualizer, items.length])

  const lastLoadMoreTime = useRef(0)
  const scrollBtnVisible = useRef(false)
  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el) return
    if (loadMoreState.current !== 'idle') return

    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight

    // Load more at top (500ms cooldown)
    if (initialScrollDone.current && el.scrollTop < 120 && Date.now() - lastLoadMoreTime.current > 500) {
      lastLoadMoreTime.current = Date.now()
      loadMorePrevHeight.current = el.scrollHeight
      loadMoreState.current = 'pending'
      chatStore.loadMore(convId)
      setTimeout(() => {
        if (loadMoreState.current === 'pending') loadMoreState.current = 'idle'
      }, 5000)
    }

    // Auto-scroll hysteresis: easy to exit, hard to re-enter
    if (shouldAutoScroll.current) {
      if (distFromBottom > 100) shouldAutoScroll.current = false
    } else {
      if (distFromBottom < 5) shouldAutoScroll.current = true
    }

    // Throttled scroll button
    const newBtnVisible = distFromBottom > 300
    if (newBtnVisible !== scrollBtnVisible.current) {
      scrollBtnVisible.current = newBtnVisible
      setShowScrollBtn(newBtnVisible)
    }
  }, [convId])

  // Resize observer — auto-scroll when content grows while user is at bottom
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const ro = new ResizeObserver(() => {
      if (shouldAutoScroll.current) scrollToEnd()
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [scrollToEnd])

  // Scroll to bottom on conversation change
  const messagesRef = useRef(messages)
  messagesRef.current = messages
  const itemsRef = useRef(items)
  itemsRef.current = items
  useLayoutEffect(() => {
    shouldAutoScroll.current = true
    initialScrollDone.current = false
    loadMoreState.current = 'idle'
    const msgs = messagesRef.current
    const itms = itemsRef.current
    prevLen.current = msgs.length
    prevFirstId.current = msgs[0]?.msg_id || 0

    // Force virtualizer to re-measure on conversation switch
    virtualizerRef.current.measure()

    requestAnimationFrame(() => {
      if (itms.length > 0) {
        virtualizerRef.current.scrollToIndex(itms.length - 1, { align: 'end', behavior: 'auto' })
      }
      requestAnimationFrame(() => {
        initialScrollDone.current = true
      })
    })
  }, [convId])

  // Handle message changes: prepend (loadMore) or append (new message)
  useLayoutEffect(() => {
    if (messages.length === prevLen.current) return

    const firstMsg = messages[0]
    const firstId = firstMsg?.msg_id || 0

    if (firstId > 0 && firstId !== prevFirstId.current && loadMoreState.current !== 'idle') {
      // Messages were prepended (loadMore completed).
      const el = scrollRef.current
      if (el && loadMorePrevHeight.current > 0) {
        const heightDiff = el.scrollHeight - loadMorePrevHeight.current
        if (heightDiff > 0) {
          el.scrollTop += heightDiff
        }
      }
      loadMoreState.current = 'idle'
    } else if (messages.length > prevLen.current && shouldAutoScroll.current) {
      scrollToEnd()
    }

    prevLen.current = messages.length
    if (firstId > 0) prevFirstId.current = firstId
  })

  const handleScrollBtnClick = () => {
    shouldAutoScroll.current = true
    scrollToEnd(true)
  }

  return (
    <div className="relative h-full">
      <div ref={scrollRef} onScroll={handleScroll} className="h-full overflow-y-auto px-4">
        <div style={{ height: virtualizer.getTotalSize(), position: 'relative' }}>
          {virtualizer.getVirtualItems().map(virtualRow => {
            const item = items[virtualRow.index]
            if (!item) return null
            if (item.type === 'date') {
              return (
                <div key={item.key}
                  data-index={virtualRow.index}
                  ref={virtualizer.measureElement}
                  style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: virtualRow.size, transform: `translateY(${virtualRow.start}px)`, overflow: 'visible' }}>
                  <DateSeparator timestamp={item.ts} />
                </div>
              )
            }
            const msg = item.msg
            return (
              <div key={item.key}
                id={`msg-${msg.msg_id}`}
                data-index={virtualRow.index}
                ref={virtualizer.measureElement}
                style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: virtualRow.size, transform: `translateY(${virtualRow.start}px)`, overflow: 'visible' }}
                className="animate-msg-in"
              >
                <MemoBubble
                  message={msg}
                  isOwn={msg.sender_id === currentUserId}
                  isGrouped={false}
                  senderInfo={senderMap[msg.sender_id]}
                />
              </div>
            )
          })}
        </div>
      </div>
      {showScrollBtn && (
        <button onClick={handleScrollBtnClick}
          className="absolute bottom-3 right-6 w-8 h-8 rounded-full bg-[var(--color-surface-card)] border border-[var(--color-hairline)] flex items-center justify-center hover:bg-[var(--color-surface-soft)] transition-all z-10"
          style={{ boxShadow: 'var(--shadow-md)' }}>
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M6 9l6 6 6-6"/></svg>
        </button>
      )}
    </div>
  )
}
