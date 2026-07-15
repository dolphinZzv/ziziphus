import { useEffect, useRef, useLayoutEffect, useCallback, useState, memo } from 'react'
import type { Message } from '@/types/message'
import { chatStore } from '@/stores/chat-store'
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
  const loadingMore = useRef(false)
  const prevLen = useRef(0)
  const shouldAutoScroll = useRef(false) // true = user is at bottom, auto-follow
  const matchRefs = useRef<(HTMLDivElement | null)[]>([])

  const scrollToEnd = useCallback((smooth = false) => {
    const el = scrollRef.current
    if (!el) return
    el.scrollTo({ top: el.scrollHeight, behavior: smooth ? 'smooth' : 'instant' })
  }, [])

  // Detect when user manually scrolls up — stop auto-follow
  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el || loadingMore.current) return
    // Load more when near top
    if (el.scrollTop < 120) {
      loadingMore.current = true
      const prevHeight = el.scrollHeight
      chatStore.loadMore(convId)
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          const el2 = scrollRef.current
          if (el2) el2.scrollTop = el2.scrollHeight - prevHeight
          loadingMore.current = false
        })
      })
    }
    // If user is within 50px of bottom → auto-follow; otherwise stop
    const distFromBottom = el.scrollHeight - el.scrollTop - el.clientHeight
    shouldAutoScroll.current = distFromBottom < 50
    setShowScrollBtn(distFromBottom > 300)
  }, [convId])

  // ResizeObserver: keep scroll pinned to bottom when auto-follow is on.
  // This handles images loading, agent timeline expanding, and initial render.
  useEffect(() => {
    const el = scrollRef.current
    if (!el) return
    const ro = new ResizeObserver(() => {
      if (shouldAutoScroll.current) scrollToEnd()
    })
    ro.observe(el)
    return () => ro.disconnect()
  }, [scrollToEnd])

  // Reset auto-scroll and scroll to end on conversation change
  useLayoutEffect(() => {
    shouldAutoScroll.current = true
    scrollToEnd()
    prevLen.current = messages.length
  }, [convId])

  // New messages arrived — auto-scroll if user was already at bottom
  useLayoutEffect(() => {
    if (loadingMore.current) return
    if (messages.length > prevLen.current) {
      if (shouldAutoScroll.current) scrollToEnd()
    }
    prevLen.current = messages.length
  }, [messages.length])

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
      >
        <MemoBubble
          message={msg}
          isOwn={msg.sender_id === currentUserId}
          isGrouped={false}
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
      {rows}
      <div className="h-5" />
    </div>
    {showScrollBtn && (
      <button onClick={() => { shouldAutoScroll.current = true; scrollToEnd(true) }}
        className="absolute bottom-3 right-6 w-8 h-8 rounded-full bg-[var(--color-surface-card)] border border-[var(--color-hairline)] shadow-md flex items-center justify-center hover:bg-[var(--color-surface-soft)] transition-all z-10"
        style={{ boxShadow: 'var(--shadow-md)' }}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M6 9l6 6 6-6"/></svg>
      </button>
    )}
    </div>
  )
}
