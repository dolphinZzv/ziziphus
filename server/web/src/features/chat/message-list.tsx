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
}

// Memoized bubble to avoid re-rendering unchanged messages
const MemoBubble = memo(MessageBubble)

export default function MessageList({ convId, messages, currentUserId }: Props) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const loadingMore = useRef(false)
  const prevLen = useRef(0)

  const scrollToEnd = useCallback(() => {
    const el = scrollRef.current
    if (el) el.scrollTo({ top: el.scrollHeight + 1000, behavior: 'instant' })
  }, [])

  // Scroll to bottom on new conversation
  useLayoutEffect(() => {
    scrollToEnd()
    prevLen.current = messages.length
  }, [convId])

  // Scroll when new messages arrive (new push or own send)
  useLayoutEffect(() => {
    if (loadingMore.current) return
    if (messages.length > prevLen.current) scrollToEnd()
    prevLen.current = messages.length
  }, [messages.length])

  // Load older messages when scrolling up
  const handleScroll = useCallback(() => {
    const el = scrollRef.current
    if (!el || loadingMore.current) return
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
  }, [convId])

  // Build message rows with date separators
  const rows: React.ReactNode[] = []
  let lastDate = 0
  for (const msg of messages) {
    if (!isSameDay(msg.timestamp, lastDate)) {
      rows.push(<DateSeparator key={`date-${msg.timestamp}`} timestamp={msg.timestamp} />)
      lastDate = msg.timestamp
    }
    rows.push(
      <MemoBubble
        key={msg.msg_id > 0 ? `msg-${msg.msg_id}` : `local-${msg.client_seq}`}
        message={msg}
        isOwn={msg.sender_id === currentUserId}
        isGrouped={false}
      />
    )
  }

  // Show "scroll to bottom" button when scrolled up
  const [showScrollBtn, setShowScrollBtn] = useState(false)

  const handleScrollWithBtn = useCallback(() => {
    handleScroll()
    const el = scrollRef.current
    if (el) setShowScrollBtn(el.scrollHeight - el.scrollTop - el.clientHeight > 300)
  }, [handleScroll])

  return (
    <div className="relative h-full">
    <div ref={scrollRef} onScroll={handleScrollWithBtn} className="h-full overflow-y-auto px-4 py-2">
      {rows}
      <div className="h-5" />
    </div>
    {showScrollBtn && (
      <button onClick={scrollToEnd}
        className="absolute bottom-3 right-6 w-8 h-8 rounded-full bg-[var(--color-surface-card)] border border-[var(--color-hairline)] shadow-md flex items-center justify-center hover:bg-[var(--color-surface-soft)] transition-all z-10"
        style={{ boxShadow: 'var(--shadow-md)' }}>
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M6 9l6 6 6-6"/></svg>
      </button>
    )}
    </div>
  )
}
