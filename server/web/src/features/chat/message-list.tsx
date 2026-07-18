import { useEffect, useRef, useLayoutEffect, useCallback, useState, memo } from 'react'
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
  searchKeyword?: string
  matchIndex?: number
  searchMatches?: number[]
}

const MemoBubble = memo(MessageBubble)

export default function MessageList({ convId, messages, currentUserId, searchKeyword, matchIndex, searchMatches }: Props) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const contentRef = useRef<HTMLDivElement>(null)
  const prevLen = useRef(0)
  const prevFirstId = useRef(0)
  const shouldAutoScroll = useRef(true)
  const initialScrollDone = useRef(false)
  const loadMoreState = useRef<'idle' | 'pending' | 'done'>('idle')
  const loadMorePrevHeight = useRef(0)
  const matchRefs = useRef<(HTMLDivElement | null)[]>([])

  // Batch-fetch sender info
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
    const missing = idsArr.filter(id => !senderMap[id])
    if (missing.length === 0) return
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
    const el = scrollRef.current
    if (!el) return
    el.scrollTo({ top: el.scrollHeight, behavior: smooth ? 'smooth' : 'instant' })
  }, [])

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
      // Safety: reset to idle after 5s even if loadMore doesn't change messages
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

  // ResizeObserver — auto-scroll when content grows while user is at bottom
  useEffect(() => {
    const el = contentRef.current
    if (!el) return
    const ro = new ResizeObserver(() => {
      if (shouldAutoScroll.current) scrollToEnd()
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [scrollToEnd])

  // Reset on conversation change
  useLayoutEffect(() => {
    shouldAutoScroll.current = true
    initialScrollDone.current = false
    loadMoreState.current = 'idle'
    scrollToEnd()
    prevLen.current = messages.length
    prevFirstId.current = messages[0]?.msg_id || 0
    requestAnimationFrame(() => {
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
      // Adjust scroll position to keep visual content stable — no flicker.
      const el = scrollRef.current
      if (el && loadMorePrevHeight.current > 0) {
        const heightDiff = el.scrollHeight - loadMorePrevHeight.current
        if (heightDiff > 0) {
          el.scrollTop += heightDiff
        }
      }
      loadMoreState.current = 'idle'
    } else if (messages.length > prevLen.current && shouldAutoScroll.current) {
      // New messages appended at the end
      scrollToEnd()
    }

    prevLen.current = messages.length
    if (firstId > 0) prevFirstId.current = firstId
  })

  // Scroll to current search match
  useEffect(() => {
    if (searchMatches && searchMatches.length > 0 && matchIndex !== undefined) {
      const idx = searchMatches[matchIndex]
      if (idx !== undefined && matchRefs.current[idx]) {
        matchRefs.current[idx]?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }
    }
  }, [matchIndex, searchMatches])

  const rows: React.ReactNode[] = []
  let lastDate = 0
  const lowerKeyword = searchKeyword?.toLowerCase() || ''

  for (let i = 0; i < messages.length; i++) {
    const msg = messages[i]
    if (!isSameDay(msg.timestamp, lastDate)) {
      rows.push(<DateSeparator key={`date-${msg.timestamp}`} timestamp={msg.timestamp} />)
      lastDate = msg.timestamp
    }
    const isMatch = searchMatches?.includes(i) ?? false
    const isCurrentMatch = matchIndex !== undefined && searchMatches?.[matchIndex] === i
    rows.push(
      <div key={msg.msg_id > 0 ? `msg-${msg.msg_id}` : `local-${msg.client_seq}`}
        id={`msg-${msg.msg_id}`}
        ref={el => { matchRefs.current[i] = el }}
        className="animate-msg-in"
      >
        <MemoBubble
          message={msg}
          isOwn={msg.sender_id === currentUserId}
          isGrouped={false}
          senderInfo={senderMap[msg.sender_id]}
          highlight={lowerKeyword}
          isSearchMatch={isMatch}
          isCurrentSearchMatch={isCurrentMatch}
        />
      </div>
    )
  }

  const [showScrollBtn, setShowScrollBtn] = useState(false)

  return (
    <div className="relative h-full">
    <div ref={scrollRef} onScroll={handleScroll} className="h-full overflow-y-auto px-4 py-2">
      <div ref={contentRef}>
        {rows}
      </div>
      <div className="h-5" />
    </div>
    {showScrollBtn && (
      <button onClick={() => { shouldAutoScroll.current = true; scrollToEnd(true) }}
        className="absolute bottom-3 right-6 w-8 h-8 rounded-full bg-[var(--color-surface-card)] border border-[var(--color-hairline)] flex items-center justify-center hover:bg-[var(--color-surface-soft)] transition-all z-10"
        style={{ boxShadow: 'var(--shadow-md)' }}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M6 9l6 6 6-6"/></svg>
      </button>
    )}
    </div>
  )
}
